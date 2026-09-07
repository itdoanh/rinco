"""Async Qdrant client wrapper."""
from __future__ import annotations

import os
from typing import Any, Dict, List, Optional

try:
    from qdrant_client import AsyncQdrantClient  # type: ignore
    from qdrant_client.models import (  # type: ignore
        Distance, VectorParams, PointStruct, Filter, FieldCondition, MatchValue,
    )
    HAS_QDRANT = True
except Exception:  # pragma: no cover
    AsyncQdrantClient = None  # type: ignore
    Distance = None  # type: ignore
    VectorParams = None  # type: ignore
    PointStruct = None  # type: ignore
    Filter = None  # type: ignore
    FieldCondition = None  # type: ignore
    MatchValue = None  # type: ignore
    HAS_QDRANT = False

from app.core import QDRANT_URL, get_logger

log = get_logger("rag-chatbot.qdrant")

_qdrant: Any = None


def get_client() -> Any:
    global _qdrant
    if _qdrant is not None:
        return _qdrant
    if not HAS_QDRANT:
        return None
    try:
        log.info("qdrant_connecting", url=QDRANT_URL)
        _qdrant = AsyncQdrantClient(url=QDRANT_URL)
        return _qdrant
    except Exception as exc:
        log.warning("qdrant_connect_failed", error=str(exc))
        return None


def build_filter(tenant_id: str, meta: Dict[str, Any] | None = None) -> Any:
    """Build a Qdrant Filter for a tenant + optional metadata."""
    if not HAS_QDRANT or get_client() is None:
        return None
    conditions = [FieldCondition(key="tenant_id", match=MatchValue(value=tenant_id))]
    for k, v in (meta or {}).items():
        conditions.append(
            FieldCondition(key=f"metadata.{k}", match=MatchValue(value=v))
        )
    return Filter(must=conditions)


def user_collection(tenant_id: str) -> str:
    return f"tenant_{tenant_id.replace('-', '_')}_kb"


async def upsert_points(
    collection: str,
    points: List[Dict[str, Any]],
) -> None:
    client = get_client()
    if client is None:
        log.warning("qdrant_not_available_skip_upsert")
        return
    try:
        await client.upsert(
            collection_name=collection,
            points=[
                PointStruct(
                    id=p["id"],
                    vector=p["vector"],
                    payload=p["payload"],
                )
                for p in points
            ],
        )
    except Exception as exc:
        log.warning("qdrant_upsert_failed", error=str(exc))


async def search_points(
    collection: str,
    vector: List[float],
    top_k: int,
    flt: Any = None,
    score_threshold: float = 0.2,
) -> List[Dict[str, Any]]:
    client = get_client()
    if client is None:
        return []
    try:
        res = await client.search(
            collection_name=collection,
            query_vector=vector,
            limit=top_k,
            query_filter=flt,
            score_threshold=score_threshold,
        )
        return [
            {
                "id": str(p.id),
                "score": float(p.score),
                "payload": p.payload,
            }
            for p in res
        ]
    except Exception as exc:
        log.warning("qdrant_search_failed", error=str(exc))
        return []


async def collection_exists(name: str) -> bool:
    client = get_client()
    if client is None:
        return False
    try:
        return await client.collection_exists(name)
    except Exception:
        return False


async def create_collection(
    name: str,
    vector_size: int = 1024,
    distance: str = "Cosine",
) -> None:
    client = get_client()
    if client is None:
        log.warning("qdrant_not_available_skip_create")
        return
    dist_map = {
        "Cosine": Distance.COSINE,
        "Dot": Distance.DOT,
        "Euclid": Distance.EUCLID,
    }
    try:
        await client.create_collection(
            collection_name=name,
            vectors_config=VectorParams(
                size=vector_size,
                distance=dist_map.get(distance, Distance.COSINE),
            ),
        )
    except Exception as exc:
        log.warning("qdrant_create_collection_failed", error=str(exc))


async def delete_collection(name: str) -> None:
    client = get_client()
    if client is None:
        return
    try:
        await client.delete_collection(name)
    except Exception as exc:
        log.warning("qdrant_delete_collection_failed", error=str(exc))