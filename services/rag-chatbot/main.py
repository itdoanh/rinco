"""
RAG Chatbot Service - Retrieval-Augmented Generation cho customer support.

Multi-tenant: mỗi tenant có Qdrant collection riêng.
"""
import asyncio
import os
from typing import Any

import httpx
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from qdrant_client import QdrantClient
from qdrant_client.models import PointStruct
from sentence_transformers import SentenceTransformer
import structlog

logger = structlog.get_logger()
app = FastAPI(title="RAG Chatbot", version="1.0.0")

EMBED_MODEL = os.getenv("EMBED_MODEL", "intfloat/multilingual-e5-large")
VLLM_URL = os.getenv("VLLM_URL", "http://vllm:8000/v1")
QDRANT_URL = os.getenv("QDRANT_URL", "http://qdrant:6333")
LLM_MODEL = os.getenv("LLM_MODEL", "Qwen2.5-14B-Instruct-AWQ")

embedder = SentenceTransformer(EMBED_MODEL)
qdrant = QdrantClient(url=QDRANT_URL)


class QueryRequest(BaseModel):
    tenant_id: str
    user_id: str
    question: str
    conversation_history: list[dict[str, str]] = []
    top_k: int = 5
    collection: str = "kb"


class QueryResponse(BaseModel):
    answer: str
    sources: list[dict[str, Any]]
    model: str
    latency_ms: int


class IngestRequest(BaseModel):
    tenant_id: str
    doc_id: str
    text: str
    metadata: dict[str, Any] = {}


def embed_sync(text: str) -> list[float]:
    return embedder.encode(text, normalize_embeddings=True).tolist()


def retrieve_sync(tenant_id: str, collection: str, query_vec: list[float], top_k: int) -> list[dict]:
    """Search per-tenant collection."""
    try:
        results = qdrant.search(
            collection_name=f"{tenant_id}_{collection}",
            query_vector=query_vec,
            limit=top_k,
            score_threshold=0.5,
        )
        return [
            {
                "id": str(r.id),
                "score": float(r.score),
                "text": r.payload.get("text", ""),
                "metadata": {k: v for k, v in r.payload.items() if k != "text"},
            }
            for r in results
        ]
    except Exception as e:
        logger.warning("qdrant_search_failed", error=str(e))
        return []


async def generate(prompt: str, history: list[dict]) -> str:
    messages = []
    for h in history[-6:]:
        messages.append({"role": h["role"], "content": h["content"]})
    messages.append({"role": "user", "content": prompt})

    async with httpx.AsyncClient(timeout=60) as client:
        r = await client.post(
            f"{VLLM_URL}/chat/completions",
            json={
                "model": LLM_MODEL,
                "messages": messages,
                "max_tokens": 1024,
                "temperature": 0.3,
            },
        )
        r.raise_for_status()
        return r.json()["choices"][0]["message"]["content"]


def build_prompt(question: str, sources: list[dict]) -> str:
    context_parts = []
    for i, s in enumerate(sources, 1):
        text = s["text"][:1000]
        meta = s["metadata"].get("source", "unknown")
        context_parts.append(f"[{i}] (source: {meta})\n{text}")
    context = "\n\n".join(context_parts)
    return f"""Trả lời câu hỏi dựa trên THÔNG TIN dưới đây. Trích dẫn nguồn bằng ký hiệu [n]. Nếu không chắc chắn, nói "Tôi không chắc chắn".

THÔNG TIN:
{context}

CÂU HỎI: {question}

TRẢ LỜI (tiếng Việt):"""


@app.post("/query", response_model=QueryResponse)
async def query(req: QueryRequest):
    start = asyncio.get_event_loop().time()
    loop = asyncio.get_event_loop()

    q_vec = await loop.run_in_executor(None, embed_sync, req.question)
    sources = await loop.run_in_executor(None, retrieve_sync, req.tenant_id, req.collection, q_vec, req.top_k)

    if not sources:
        raise HTTPException(status_code=404, detail="Không tìm thấy thông tin liên quan")

    prompt = build_prompt(req.question, sources)
    answer = await generate(prompt, req.conversation_history)

    latency_ms = int((asyncio.get_event_loop().time() - start) * 1000)

    return QueryResponse(
        answer=answer,
        sources=[{"score": s["score"], "metadata": s["metadata"]} for s in sources],
        model=LLM_MODEL,
        latency_ms=latency_ms,
    )


@app.post("/ingest")
async def ingest(req: IngestRequest):
    """Ingest document vào KB."""
    vec = embed_sync(req.text)
    qdrant.upsert(
        collection_name=f"{req.tenant_id}_kb",
        points=[PointStruct(id=req.doc_id, vector=vec, payload={"text": req.text, **req.metadata})],
    )
    return {"status": "ok", "doc_id": req.doc_id}


@app.get("/health")
async def health():
    return {"status": "ok", "service": "rag-chatbot"}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8091, workers=1)
