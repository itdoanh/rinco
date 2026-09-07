"""Chat endpoint with SSE streaming."""
from __future__ import annotations

import json
import re
import time
import uuid
from typing import Any, AsyncGenerator, Dict, List, Optional

from fastapi import APIRouter, Header, HTTPException
from fastapi.responses import StreamingResponse

from app.core import LLM_MODEL, METRICS, get_logger
from app.schemas import ChatRequest, Citation
from app.services.chunker import chunk_text
from app.services.embeddings import embed_texts
from app.services.llm import call_vllm_stream, estimate_tokens
from app.services.qdrant_client import build_filter, search_points, user_collection
from app.services.reranker import rerank

log = get_logger("rag-chatbot.chat")

router = APIRouter()


def sanitize_input(text: str) -> str:
    text = re.sub(r"[^@]+@[^@]+\.[^@]+", "[REDACTED_EMAIL]", text)
    text = re.sub(r"(\+84|0)\d{9,10}", "[REDACTED_PHONE]", text)
    text = re.sub(r"\d{12}", "[REDACTED_CCCD]", text)
    bad = ["ignore previous", "disregard instructions", "system:", "assistant:"]
    for b in bad:
        if b in text.lower():
            text = text.replace(b, "[REDACTED_OVERRIDE]")
    return text


def build_rag_messages(
    query: str,
    history: List[Dict[str, str]],
    context: List[Dict[str, Any]],
    system: Optional[str],
) -> List[Dict[str, str]]:
    sys = system or (
        "Ban la tro ly RINCO. Tra loi ngan gon, chinh xac, trich dan nguon bang [1][2]... "
        "Khong bia thong tin ngoai context."
    )
    msgs: List[Dict[str, str]] = [{"role": "system", "content": sys}]
    for h in history[-6:]:
        if isinstance(h, dict) and "role" in h and "content" in h:
            msgs.append({"role": h["role"], "content": h["content"]})
    ctx_lines = []
    for i, c in enumerate(context):
        src = c.get("payload", {}).get("source_url", "-")
        txt = c.get("payload", {}).get("text", "")[:600]
        ctx_lines.append(f"[{i+1}] {src}: {txt}")
    msgs.append({
        "role": "user",
        "content": f"CONTEXT:\n" + "\n\n".join(ctx_lines) + f"\n\nQUESTION: {query}",
    })
    return msgs


@router.post("/v1/chat")
async def chat(
    req: ChatRequest,
    x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
    x_user_id: str = Header("anonymous", alias="X-User-ID"),
):
    start = time.perf_counter()
    chat_requests = METRICS["chat_requests"]
    if chat_requests:
        chat_requests.labels(tenant_id=x_tenant_id, result="started").inc()
    safe_query = sanitize_input(req.query)
    collection = req.collection or user_collection(x_tenant_id)

    # Retrieve context.
    q_vecs = await embed_texts([safe_query])
    q_vec = q_vecs[0]
    flt = build_filter(x_tenant_id, req.metadata_filter)
    raw = await search_points(collection, q_vec, min(6, req.top_k), flt=flt)
    rr = rerank(safe_query, raw, top_n=min(6, req.top_k))
    citations = [
        Citation(
            chunk_id=h["id"],
            score=float(h.get("rerank_score", h.get("score", 0))),
            source_url=h.get("payload", {}).get("source_url"),
            doc_type=h.get("payload", {}).get("doc_type"),
            text=h.get("payload", {}).get("text", ""),
        )
        for h in rr
    ]
    msgs = build_rag_messages(safe_query, req.history, rr, req.system_prompt)

    if req.stream:
        async def stream_gen() -> AsyncGenerator[str, None]:
            buffer: List[str] = []
            try:
                async for token_json in call_vllm_stream(
                    msgs, req.system_prompt and LLM_MODEL or LLM_MODEL, req.temperature,
                ):
                    try:
                        obj = json.loads(token_json)
                        delta = obj.get("choices", [{}])[0].get("delta", {}).get("content", "") or ""
                    except Exception:
                        delta = ""
                    if not delta:
                        continue
                    buffer.append(delta)
                    st_tokens = METRICS["streaming_tokens"]
                    if st_tokens:
                        st_tokens.labels(tenant_id=x_tenant_id, model=LLM_MODEL).inc()
                    yield json.dumps({"event": "token", "data": delta}, ensure_ascii=False) + "\n"
                answer = "".join(buffer)
                yield json.dumps({
                    "event": "done",
                    "data": {
                        "answer": answer,
                        "citations": [c.model_dump() for c in citations],
                        "collection": collection,
                        "model": LLM_MODEL,
                    },
                }, ensure_ascii=False) + "\n"
                if chat_requests:
                    chat_requests.labels(tenant_id=x_tenant_id, result="ok").inc()
            except Exception as exc:
                if chat_requests:
                    chat_requests.labels(tenant_id=x_tenant_id, result="failed").inc()
                yield json.dumps({"event": "error", "data": {"error": str(exc)}}, ensure_ascii=False) + "\n"
            latency = METRICS["chat_latency"]
            if latency:
                latency.observe(time.perf_counter() - start)
        return StreamingResponse(stream_gen(), media_type="text/event-stream")

    answer = "RAG context assembled; streaming unavailable in this mode."
    if chat_requests:
        chat_requests.labels(tenant_id=x_tenant_id, result="ok").inc()
    latency = METRICS["chat_latency"]
    if latency:
        latency.observe(time.perf_counter() - start)
    from app.schemas import ChatResponse
    return ChatResponse(
        answer=answer, citations=citations, model=LLM_MODEL,
        tokens_used=estimate_tokens(answer),
        latency_ms=int((time.perf_counter() - start) * 1000),
        collection=collection,
    )