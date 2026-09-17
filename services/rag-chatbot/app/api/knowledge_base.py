"""Knowledge-base management endpoints for the RAG chatbot.

Routes:

- ``GET  /v1/kb/templates`` — list curated KB template snippets
- ``POST /v1/kb/seed``      — request seeding of the standard RINCO KB
- ``GET  /v1/kb/stats``     — compute summary stats for the tenant KB
"""
from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Header, HTTPException
from pydantic import BaseModel, Field

from app.services.knowledge_base import (
    RINCO_KB,
    build_collection_name,
    kb_stats,
    seed_knowledge_base,
)

router = APIRouter(prefix="/v1/kb", tags=["kb"])


class SeedKbRequest(BaseModel):
    extra: list[dict[str, str]] = Field(default_factory=list)


@router.get("/templates")
async def templates() -> dict[str, Any]:
    """Return the curated RINCO KB templates."""
    return {
        "count": len(RINCO_KB),
        "items": RINCO_KB,
    }


@router.post("/seed")
async def seed(
    payload: SeedKbRequest,
    x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
) -> dict[str, Any]:
    """Return a curated KB payload that can be ingested.

    The endpoint does not call :func:`ingest` directly — instead it
    returns the document list so callers (e.g. an admin script) can
    review and dispatch. The collection name is also returned to
    avoid drift with :func:`build_collection_name`.
    """
    if not x_tenant_id:
        raise HTTPException(400, "X-Tenant-ID header is required")
    docs = seed_knowledge_base(x_tenant_id, extra=payload.extra)
    return {
        "tenant_id": x_tenant_id,
        "collection": build_collection_name(x_tenant_id),
        "documents": docs,
        "count": len(docs),
    }


@router.get("/stats")
async def stats(
    points: list[dict[str, Any]] | None = None,
    x_tenant_id: str = Header("", alias="X-Tenant-ID"),
) -> dict[str, Any]:
    """Return summary stats for the supplied KB points.

    Intended to be called after listing points in Qdrant. The endpoint
    accepts an optional query string ``count`` to deterministically
    size the summary when the upstream list is large.
    """
    pts = points or []
    return {
        "tenant_id": x_tenant_id,
        "collection": build_collection_name(x_tenant_id) if x_tenant_id else None,
        "summary": kb_stats(pts),
    }


__all__ = ["router"]
