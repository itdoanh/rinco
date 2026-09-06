"""
RINCO RAG Chatbot.

- POST /v1/chat            : RAG chat với streaming SSE
- POST /v1/ingest          : ingest URL / file
- GET  /v1/collections     : list Qdrant collections
- POST /v1/collections     : create tenant collection
- DELETE /v1/collections/:id
- POST /v1/search          : pure semantic search (no LLM)
- POST /v1/feedback        : RLHF feedback (+/- rating)
- GET  /v1/health / /v1/metrics

Multi-tenant: collection naming `tenant_<id>_kb` plus payload filter tenant_id.
"""
from __future__ import annotations

import asyncio
import json
import os
import re
import time
import uuid
from contextlib import asynccontextmanager
from dataclasses import dataclass
from typing import Any, AsyncGenerator, Dict, List, Optional

import structlog
from fastapi import FastAPI, Header, HTTPException, Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field
from prometheus_client import (
    CONTENT_TYPE_LATEST,
    Counter,
    Histogram,
    generate_latest,
)

import logging
logging.getLogger().setLevel(os.getenv("LOG_LEVEL", "INFO"))
log = structlog.get_logger()

# ---------------------------------------------------------------- imports (optional heavy)
try:
    from sentence_transformers import SentenceTransformer, CrossEncoder  # type: ignore
    HAS_ST = True
except Exception:  # pragma: no cover
    HAS_ST = False
try:
    from qdrant_client import AsyncQdrantClient  # type: ignore
    from qdrant_client.models import (  # type: ignore
        Distance, VectorParams, PointStruct, Filter, FieldCondition, MatchValue,
    )
    HAS_QDRANT = True
except Exception:
    HAS_QDRANT = False
try:
    import httpx  # type: ignore
    HAS_HTTPX = True
except Exception:
    HAS_HTTPX = False

# ---------------------------------------------------------------- Prom metrics
chat_requests = Counter(
    "rag_chat_requests_total", "Total chat requests",
    ["tenant_id", "result"],
)
chat_latency = Histogram(
    "rag_chat_latency_seconds", "End-to-end chat latency",
    buckets=[0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10],
)
ingest_docs = Counter(
    "rag_ingest_documents_total", "Documents ingested",
    ["tenant_id", "kind"],
)
search_requests = Counter(
    "rag_search_requests_total", "Semantic search requests",
    ["tenant_id"],
)
streaming_tokens = Counter(
    "rag_streaming_tokens_total", "Tokens streamed",
    ["tenant_id", "model"],
)

# ---------------------------------------------------------------- config
VLLM_URL = os.getenv("VLLM_URL", "http://vllm:8000/v1")
VLLM_API_KEY = os.getenv("VLLM_API_KEY", "")
EMBEDDING_MODEL = os.getenv("EMBEDDING_MODEL", "BAAI/bge-m3")
RERANKER_MODEL = os.getenv("RERANKER_MODEL", "BAAI/bge-reranker-base")
LLM_MODEL = os.getenv("LLM_MODEL", "meta-llama/Llama-3-8B-Instruct")
QDRANT_URL = os.getenv("QDRANT_URL", "http://qdrant:6333")
CHUNK_SIZE = int(os.getenv("CHUNK_SIZE", "512"))
CHUNK_OVERLAP = int(os.getenv("CHUNK_OVERLAP", "64"))
DEFAULT_TOP_K = int(os.getenv("TOP_K", "20"))


# ---------------------------------------------------------------- Pydantic
class ChatRequest(BaseModel):
    query: str = Field(..., min_length=1)
    history: List[Dict[str, str]] = Field(default_factory=list)
    collection: Optional[str] = None
    system_prompt: Optional[str] = None
    temperature: float = 0.2
    top_k: int = DEFAULT_TOP_K
    stream: bool = True
    metadata_filter: Dict[str, Any] = Field(default_factory=dict)


class Citation(BaseModel):
    chunk_id: str
    score: float
    source_url: Optional[str] = None
    doc_type: Optional[str] = None
    text: str
    span: Optional[Dict[str, int]] = None


class ChatResponse(BaseModel):
    answer: str
    citations: List[Citation] = Field(default_factory=list)
    model: str
    tokens_used: int
    latency_ms: int
    collection: str
    feedback_id: Optional[str] = None


class IngestRequest(BaseModel):
    source: str
    kind: str = "text"  # url | pdf | docx | xlsx | md | txt | text
    collection: Optional[str] = None
    metadata: Dict[str, Any] = Field(default_factory=dict)


class IngestResponse(BaseModel):
    doc_id: str
    chunks_created: int
    collection: str
    bytes_ingested: int
    duration_ms: int


class SearchRequest(BaseModel):
    query: str
    collection: Optional[str] = None
    top_k: int = DEFAULT_TOP_K
    metadata_filter: Dict[str, Any] = Field(default_factory=dict)


class SearchHit(BaseModel):
    chunk_id: str
    score: float
    text: str
    source_url: Optional[str] = None
    metadata: Dict[str, Any] = Field(default_factory=dict)


class SearchResponse(BaseModel):
    hits: List[SearchHit]
    collection: str
    query: str


class FeedbackRequest(BaseModel):
    chat_id: Optional[str] = None
    query: str
    answer: str
    rating: int
    comment: Optional[str] = None


class CollectionInfo(BaseModel):
    name: str
    vectors_count: int
    points_count: int
    config: Dict[str, Any] = Field(default_factory=dict)


class CreateCollectionRequest(BaseModel):
    name: Optional[str] = None
    vector_size: int = 1024
    distance: str = "Cosine"


# ---------------------------------------------------------------- state
@dataclass
class State:
    embedder: Optional[Any] = None
    reranker: Optional[Any] = None
    qdrant: Optional[Any] = None
    vocab_size: int = 0


state = State()


def _init_models() -> State:
    if HAS_ST:
        try:
            log.info("loading_embedding_model", model=EMBEDDING_MODEL)
            state.embedder = SentenceTransformer(EMBEDDING_MODEL)
            state.reranker = CrossEncoder(RERANKER_MODEL)
        except Exception as exc:
            log.warning("st_load_failed", error=str(exc))
    if HAS_QDRANT:
        try:
            state.qdrant = AsyncQdrantClient(url=QDRANT_URL)
        except Exception as exc:
            log.warning("qdrant_connect_failed", error=str(exc))
    return state


@asynccontextmanager
async def lifespan(app: FastAPI):
    _init_models()
    log.info("rag_chatbot_started", version="2.0.0",
             qdrant_url=QDRANT_URL, vllm_url=VLLM_URL)
    yield


app = FastAPI(title="RINCO RAG Chatbot", version="2.0.0", lifespan=lifespan)


# ---------------------------------------------------------------- helpers
def redact_pii(text: str) -> str:
    text = re.sub(r"[^@]+@[^@]+\.[^@]+", "[REDACTED_EMAIL]", text)
    text = re.sub(r"(\+84|0)\d{9,10}", "[REDACTED_PHONE]", text)
    text = re.sub(r"\d{12}", "[REDACTED_CCCD]", text)
    text = re.sub(r"\d{10}(-\d{3})?", "[REDACTED_TAX]", text)
    return text


def sanitize_user_input(text: str) -> str:
    text = redact_pii(text)
    bad = ["ignore previous", "disregard instructions", "system:", "assistant:"]
    for b in bad:
        if b in text.lower():
            text = text.replace(b, "[REDACTED_OVERRIDE]")
    return text


def user_collection(tenant_id: str) -> str:
    return f"tenant_{tenant_id.replace('-', '_')}_kb"


async def call_vllm_stream(messages: List[Dict[str, str]], model: str,
                           temperature: float):
    if not HAS_HTTPX:
        return
    body = {"model": model, "messages": messages, "stream": True,
            "temperature": temperature, "max_tokens": 512}
    headers = {"Content-Type": "application/json"}
    if VLLM_API_KEY:
        headers["Authorization"] = f"Bearer {VLLM_API_KEY}"
    url = VLLM_URL.rstrip("/") + "/chat/completions"
    async with httpx.AsyncClient(timeout=60) as client:
        async with client.stream("POST", url, json=body, headers=headers) as r:
            async for line in r.aiter_lines():
                if not line:
                    continue
                if line.startswith("data: "):
                    yield line[6:]


def chunk_text(text: str, chunk_size: int = CHUNK_SIZE,
               overlap: int = CHUNK_OVERLAP) -> List[str]:
    if len(text) <= chunk_size:
        return [text]
    return _split_recursive(text, chunk_size, overlap,
                            ["\n\n", "\n", ". ", " "], depth=0)


def _split_recursive(text: str, size: int, overlap: int,
                    seps: List[str], depth: int) -> List[str]:
    if not text:
        return []
    if len(text) <= size or depth >= len(seps):
        return [text]
    sep = seps[depth]
    parts = text.split(sep)
    chunks: List[str] = []
    cur = ""
    for p in parts:
        piece = (p + sep) if cur else p
        if len(cur) + len(piece) <= size:
            cur += piece
        else:
            if cur:
                chunks.append(cur)
            if len(piece) > size:
                chunks.extend(_split_recursive(piece, size, overlap, seps,
                                               depth + 1))
            else:
                cur = piece
    if cur:
        chunks.append(cur)
    return chunks


async def embed(texts: List[str]) -> List[List[float]]:
    if state.embedder is not None:
        vecs = state.embedder.encode(texts, normalize_embeddings=True)
        return vecs.tolist()
    if HAS_HTTPX:
        out: List[List[float]] = []
        async with httpx.AsyncClient(timeout=15) as client:
            for t in texts:
                r = await client.post(
                    VLLM_URL.rstrip("/") + "/embeddings",
                    headers={"Content-Type": "application/json",
                             **({"Authorization": f"Bearer {VLLM_API_KEY}"}
                                if VLLM_API_KEY else {})},
                    json={"model": EMBEDDING_MODEL, "input": t},
                )
                if r.status_code >= 300:
                    raise RuntimeError(f"embedding failed {r.status_code}")
                data = r.json()
                out.append(data["data"][0]["embedding"])
        return out
    import hashlib
    dim = 384
    out = []
    for t in texts:
        seed = hashlib.sha256(t.encode()).digest()
        out.append([(seed[i % len(seed)] / 255.0) for i in range(dim)])
    return out


async def upsert(collection: str, points: List[Dict[str, Any]]) -> None:
    if state.qdrant is None:
        log.warning("qdrant_not_available_skip_upsert")
        return
    await state.qdrant.upsert(
        collection_name=collection,
        points=[PointStruct(id=p["id"], vector=p["vector"], payload=p["payload"])
                for p in points],
    )


async def vector_search(collection: str, vector: List[float],
                       top_k: int, flt=None,
                       score_threshold: float = 0.2) -> List[Dict[str, Any]]:
    if state.qdrant is None:
        return []
    res = await state.qdrant.search(
        collection_name=collection, query_vector=vector,
        limit=top_k, query_filter=flt, score_threshold=score_threshold,
    )
    return [{"id": str(p.id), "score": float(p.score), "payload": p.payload}
            for p in res]


def rerank(query: str, hits: List[Dict[str, Any]], top_n: int) -> List[Dict[str, Any]]:
    if state.reranker is None or not hits:
        return hits[:top_n]
    pairs = [[query, h["payload"].get("text", "")] for h in hits]
    try:
        scores = state.reranker.predict(pairs)
    except Exception:
        return hits[:top_n]
    for h, s in zip(hits, scores):
        h["rerank_score"] = float(s)
    hits.sort(key=lambda h: h.get("rerank_score", 0), reverse=True)
    return hits[:top_n]


def build_filter(tenant_id: str, meta: Dict[str, Any]):
    if not HAS_QDRANT:
        return None
    conditions = [FieldCondition(key="tenant_id", match=MatchValue(value=tenant_id))]
    for k, v in (meta or {}).items():
        conditions.append(FieldCondition(key=f"metadata.{k}", match=MatchValue(value=v)))
    return Filter(must=conditions)


def build_messages(query: str, history: List[Dict[str, str]],
                  context: List[Dict[str, Any]], system: Optional[str]) -> List[Dict[str, str]]:
    sys = system or (
        "Ban la tro ly RINCO. Tra loi ngan gon, chinh xac, trich dan nguon bang [1][2]... "
        "Khong bia thong tin ngoai context."
    )
    msgs: List[Dict[str, str]] = [{"role": "system", "content": sys}]
    for h in history[-6:]:
        if isinstance(h, dict) and "role" in h and "content" in h:
            msgs.append({"role": h["role"], "content": h["content"]})
    ctx_text = "\n\n".join(
        f"[{i+1}] {c['payload'].get('source_url','-')}: {c['payload'].get('text','')[:600]}"
        for i, c in enumerate(context)
    )
    msgs.append({"role": "user",
                 "content": f"CONTEXT:\n{ctx_text}\n\nQUESTION: {query}"})
    return msgs


def count_tokens_estimate(text: str) -> int:
    return max(1, len(text) // 4)


async def fetch_url(url: str) -> str:
    if not HAS_HTTPX:
        return ""
    async with httpx.AsyncClient(timeout=10, follow_redirects=True) as c:
        r = await c.get(url)
        if r.status_code >= 300:
            raise RuntimeError(f"fetch_url {r.status_code}")
        text = re.sub(r"<[^>]+>", " ", r.text)
        return re.sub(r"\s+", " ", text).strip()


# ---------------------------------------------------------------- endpoints
@app.get("/v1/health")
async def health():
    return {"status": "ok", "service": "rag-chatbot",
            "qdrant": state.qdrant is not None,
            "embedder": state.embedder is not None,
            "reranker": state.reranker is not None}


@app.get("/v1/metrics")
async def metrics():
    return generate_latest(), 200, {"Content-Type": CONTENT_TYPE_LATEST}


@app.get("/v1/collections")
async def list_collections(x_tenant_id: str = Header(..., alias="X-Tenant-ID")):
    if state.qdrant is None:
        return {"count": 0, "collections": []}
    cols = await state.qdrant.get_collections()
    out = []
    for c in cols.collections:
        try:
            info = await state.qdrant.get_collection(c.name)
            out.append(CollectionInfo(
                name=c.name,
                vectors_count=getattr(info, "vectors_count", 0),
                points_count=getattr(info, "points_count", 0),
            ).model_dump())
        except Exception:
            out.append({"name": c.name, "vectors_count": 0, "points_count": 0})
    return {"count": len(out), "collections": out}


@app.post("/v1/collections")
async def create_collection(req: CreateCollectionRequest,
                            x_tenant_id: str = Header(..., alias="X-Tenant-ID")):
    name = req.name or user_collection(x_tenant_id)
    if state.qdrant is None:
        return {"status": "created", "name": name, "note": "qdrant offline"}
    if await state.qdrant.collection_exists(name):
        return {"status": "exists", "name": name}
    distance = {"Cosine": Distance.COSINE,
                "Dot":   Distance.DOT,
                "Euclid": Distance.EUCLID}.get(req.distance, Distance.COSINE)
    await state.qdrant.create_collection(
        collection_name=name,
        vectors_config=VectorParams(size=req.vector_size, distance=distance),
    )
    return {"status": "created", "name": name, "vector_size": req.vector_size}


@app.delete("/v1/collections/{name}")
async def delete_collection(name: str,
                           x_tenant_id: str = Header(..., alias="X-Tenant-ID")):
    expected_prefix = f"tenant_{x_tenant_id.replace('-', '_')}"
    if not name.startswith(expected_prefix):
        raise HTTPException(403, "forbidden: not your collection")
    if state.qdrant is None:
        return {"status": "deleted", "name": name}
    await state.qdrant.delete_collection(name)
    return {"status": "deleted", "name": name}


@app.post("/v1/ingest", response_model=IngestResponse)
async def ingest(req: IngestRequest,
                x_tenant_id: str = Header(..., alias="X-Tenant-ID")):
    start = time.perf_counter()
    collection = req.collection or user_collection(x_tenant_id)
    # ensure collection exists
    await create_collection(CreateCollectionRequest(name=collection),
                            x_tenant_id=x_tenant_id)

    if req.kind == "url":
        text = await fetch_url(req.source)
    else:
        text = req.source
    if not text:
        raise HTTPException(400, "no text to ingest")

    chunks = chunk_text(redact_pii(text))
    if not chunks:
        raise HTTPException(400, "chunking produced 0 chunks")

    vecs = await embed(chunks)
    points = []
    doc_id = str(uuid.uuid4())
    now = int(time.time())
    base_meta = {"tenant_id": x_tenant_id,
                 "source_url": req.source if req.kind == "url" else None,
                 "doc_type": req.kind,
                 "ingestion_date": now,
                 "doc_id": doc_id,
                 **req.metadata}
    for i, (chunk, vec) in enumerate(zip(chunks, vecs)):
        points.append({
            "id": str(uuid.uuid4()),
            "vector": vec,
            "payload": {**base_meta, "text": chunk, "chunk_index": i},
        })
    await upsert(collection, points)
    ingest_docs.labels(tenant_id=x_tenant_id, kind=req.kind).inc()

    return IngestResponse(
        doc_id=doc_id, chunks_created=len(chunks),
        collection=collection, bytes_ingested=len(text),
        duration_ms=int((time.perf_counter() - start) * 1000),
    )


@app.post("/v1/search", response_model=SearchResponse)
async def search(req: SearchRequest,
                x_tenant_id: str = Header(..., alias="X-Tenant-ID")):
    search_requests.labels(tenant_id=x_tenant_id).inc()
    collection = req.collection or user_collection(x_tenant_id)
    q_vec = (await embed([req.query]))[0]
    flt = build_filter(x_tenant_id, req.metadata_filter)
    raw = await vector_search(collection, q_vec, req.top_k, flt=flt)
    rr = rerank(req.query, raw, req.top_k)
    hits = [
        SearchHit(
            chunk_id=h["id"], score=h.get("rerank_score", h["score"]),
            text=h["payload"].get("text", "")[:1500],
            source_url=h["payload"].get("source_url"),
            metadata={k: v for k, v in h["payload"].items()
                      if k not in ("text",)},
        )
        for h in rr
    ]
    return SearchResponse(hits=hits, collection=collection, query=req.query)


@app.post("/v1/chat")
async def chat(req: ChatRequest,
              x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
              x_user_id: str = Header("anonymous", alias="X-User-ID")):
    start = time.perf_counter()
    chat_requests.labels(tenant_id=x_tenant_id, result="started").inc()
    safe_query = sanitize_user_input(req.query)
    collection = req.collection or user_collection(x_tenant_id)
    q_vec = (await embed([safe_query]))[0]
    flt = build_filter(x_tenant_id, req.metadata_filter)
    raw = await vector_search(collection, q_vec, req.top_k, flt=flt)
    rr = rerank(safe_query, raw, top_n=min(6, req.top_k))
    citations = [
        Citation(
            chunk_id=h["id"], score=h.get("rerank_score", h["score"]),
            source_url=h["payload"].get("source_url"),
            doc_type=h["payload"].get("doc_type"),
            text=h["payload"].get("text", ""),
        )
        for h in rr
    ]
    msgs = build_messages(safe_query, req.history, rr, req.system_prompt)

    if req.stream and HAS_HTTPX:
        async def stream_gen():
            buffer = []
            try:
                async for token_json in call_vllm_stream(
                    msgs, req.system_prompt and LLM_MODEL or LLM_MODEL,
                    req.temperature,
                ):
                    try:
                        obj = json.loads(token_json)
                        delta = obj["choices"][0]["delta"].get("content", "")
                    except Exception:
                        delta = token_json
                    if not delta:
                        continue
                    buffer.append(delta)
                    streaming_tokens.labels(
                        tenant_id=x_tenant_id, model=LLM_MODEL).inc(
                        count_tokens_estimate(delta))
                    yield json.dumps({"event": "token",
                                      "data": delta}, ensure_ascii=False) + "\n"
                answer = "".join(buffer)
                yield json.dumps({"event": "done",
                                  "data": {
                                      "answer": answer,
                                      "citations": [c.model_dump()
                                                     for c in citations],
                                      "collection": collection,
                                      "model": LLM_MODEL,
                                  }},
                                 ensure_ascii=False) + "\n"
                chat_requests.labels(tenant_id=x_tenant_id, result="ok").inc()
            except Exception as exc:
                chat_requests.labels(tenant_id=x_tenant_id, result="failed").inc()
                yield json.dumps({"event": "error",
                                  "data": {"error": str(exc)}},
                                 ensure_ascii=False) + "\n"
            chat_latency.observe(time.perf_counter() - start)
        return StreamingResponse(stream_gen(), media_type="text/event-stream")

    answer = "RAG context assembled; LLM unavailable in dev mode."
    chat_requests.labels(tenant_id=x_tenant_id, result="ok").inc()
    chat_latency.observe(time.perf_counter() - start)
    return ChatResponse(
        answer=answer, citations=citations, model=LLM_MODEL,
        tokens_used=count_tokens_estimate(answer),
        latency_ms=int((time.perf_counter() - start) * 1000),
        collection=collection,
    )


@app.post("/v1/feedback")
async def feedback(req: FeedbackRequest,
                  x_tenant_id: str = Header(..., alias="X-Tenant-ID")):
    path = os.getenv("FEEDBACK_LOG_PATH", "/var/log/rinco/rlhf.jsonl")
    os.makedirs(os.path.dirname(path), exist_ok=True)
    rec = {"id": str(uuid.uuid4()), "tenant_id": x_tenant_id,
           "ts": int(time.time()), **req.model_dump()}
    with open(path, "a", encoding="utf-8") as f:
        f.write(json.dumps(rec, ensure_ascii=False) + "\n")
    return {"status": "recorded", "id": rec["id"]}


@app.get("/")
async def root():
    return {
        "service": "rag-chatbot", "version": "2.0.0",
        "embedding": EMBEDDING_MODEL, "reranker": RERANKER_MODEL,
        "llm": LLM_MODEL, "qdrant": QDRANT_URL,
    }
