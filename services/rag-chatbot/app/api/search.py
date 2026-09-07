"""Semantic search endpoint."""
from __future__ import annotations

from fastapi import APIRouter, Header

from app.core import METRICS, get_logger
from app.schemas import SearchRequest, SearchHit, SearchResponse
from app.services.embeddings import embed_texts
from app.services.qdrant_client import build_filter, search_points, user_collection
from app.services.reranker import rerank

log = get_logger("rag-chatbot.search")

router = APIRouter()


@router.post("/v1/search", response_model=SearchResponse)
async def search(
    req: SearchRequest,
    x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
):
    sr = METRICS["search_requests"]
    if sr:
        sr.labels(tenant_id=x_tenant_id).inc()
    collection = req.collection or user_collection(x_tenant_id)
    q_vecs = await embed_texts([req.query])
    q_vec = q_vecs[0]
    flt = build_filter(x_tenant_id, req.metadata_filter)
    raw = await search_points(collection, q_vec, req.top_k, flt=flt)
    rr = rerank(req.query, raw, top_n=req.top_k)
    hits = [
        SearchHit(
            chunk_id=h["id"],
            score=float(h.get("rerank_score", h.get("score", 0))),
            text=h.get("payload", {}).get("text", "")[:1500],
            source_url=h.get("payload", {}).get("source_url"),
            metadata={
                k: v for k, v in h.get("payload", {}).items()
                if k not in ("text",)
            },
        )
        for h in rr
    ]
    return SearchResponse(hits=hits, collection=collection, query=req.query)