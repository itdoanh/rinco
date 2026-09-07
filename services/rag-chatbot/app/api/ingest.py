"""Ingest endpoint."""
from __future__ import annotations

import time
import uuid

from fastapi import APIRouter, Header, HTTPException

from app.core import METRICS, get_logger
from app.schemas import IngestRequest, IngestResponse
from app.services.chunker import chunk_text
from app.services.embeddings import embed_texts
from app.services.ingestion import ingest_content
from app.services.qdrant_client import create_collection, upsert_points, user_collection

log = get_logger("rag-chatbot.ingest")

router = APIRouter()


def redact_pii(text: str) -> str:
    import re
    text = re.sub(r"[^@]+@[^@]+\.[^@]+", "[REDACTED_EMAIL]", text)
    text = re.sub(r"(\+84|0)\d{9,10}", "[REDACTED_PHONE]", text)
    text = re.sub(r"\d{12}", "[REDACTED_CCCD]", text)
    return text


@router.post("/v1/ingest", response_model=IngestResponse)
async def ingest(
    req: IngestRequest,
    x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
):
    start = time.perf_counter()
    collection = req.collection or user_collection(x_tenant_id)

    # Ensure the collection exists.
    await create_collection(collection)

    # Fetch / parse content.
    try:
        text = ingest_content(req.source, req.kind)
    except Exception as exc:
        raise HTTPException(400, f"ingest failed: {exc}")

    if not text:
        raise HTTPException(400, "no text extracted from source")

    text = redact_pii(text)
    chunks = chunk_text(text)
    if not chunks:
        raise HTTPException(400, "chunking produced 0 chunks")

    # Embed.
    vecs = await embed_texts(chunks)

    # Build points.
    doc_id = str(uuid.uuid4())
    now = int(time.time())
    base_meta = {
        "tenant_id": x_tenant_id,
        "source_url": req.source if req.kind == "url" else None,
        "doc_type": req.kind,
        "ingestion_date": now,
        "doc_id": doc_id,
        **req.metadata,
    }
    points = []
    for i, (chunk, vec) in enumerate(zip(chunks, vecs)):
        points.append({
            "id": str(uuid.uuid4()),
            "vector": vec,
            "payload": {**base_meta, "text": chunk, "chunk_index": i},
        })

    # Upsert.
    await upsert_points(collection, points)

    # Metric.
    ingest_docs = METRICS["ingest_docs"]
    if ingest_docs:
        ingest_docs.labels(tenant_id=x_tenant_id, kind=req.kind).inc()

    return IngestResponse(
        doc_id=doc_id,
        chunks_created=len(chunks),
        collection=collection,
        bytes_ingested=len(text),
        duration_ms=int((time.perf_counter() - start) * 1000),
    )