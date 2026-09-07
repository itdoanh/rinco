"""Collections CRUD endpoint."""
from __future__ import annotations

from fastapi import APIRouter, Header, HTTPException

from app.core import get_logger
from app.schemas import CreateCollectionRequest
from app.services.qdrant_client import (
    collection_exists,
    create_collection,
    delete_collection,
    user_collection,
)

log = get_logger("rag-chatbot.collections")

router = APIRouter()


@router.get("/v1/collections")
async def list_collections(
    x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
):
    from app.services.qdrant_client import get_client
    client = get_client()
    if client is None:
        return {"count": 0, "collections": []}
    try:
        result = await client.get_collections()
        return {
            "count": len(result.collections),
            "collections": [
                {"name": c.name} for c in result.collections
                if c.name.startswith(f"tenant_{x_tenant_id.replace('-', '_')}")
            ],
        }
    except Exception as exc:
        log.warning("list_collections_failed", error=str(exc))
        return {"count": 0, "collections": []}


@router.post("/v1/collections")
async def post_collection(
    req: CreateCollectionRequest,
    x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
):
    name = req.name or user_collection(x_tenant_id)
    exists = await collection_exists(name)
    if exists:
        return {"status": "exists", "name": name}
    await create_collection(name, vector_size=req.vector_size, distance=req.distance)
    return {"status": "created", "name": name, "vector_size": req.vector_size}


@router.delete("/v1/collections/{name}")
async def remove_collection(
    name: str,
    x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
):
    expected = f"tenant_{x_tenant_id.replace('-', '_')}"
    if not name.startswith(expected):
        raise HTTPException(403, "forbidden: not your collection")
    await delete_collection(name)
    return {"status": "deleted", "name": name}