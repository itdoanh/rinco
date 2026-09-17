"""Lead-scoring extra endpoints (feature extraction, model versions).

Endpoints:

- ``GET  /v1/features/schema`` — return the canonical feature list
- ``POST /v1/features/extract`` — return the engineered feature dict for a single lead
- ``GET  /v1/model/versions``   — list available model versions per tenant
"""
from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Body, Header, HTTPException

from app.schemas.lead import LeadFeatures
from app.services.feature_extract import (
    extract_features_dict,
    feature_schema,
    vectorize,
)
from app.services.inference import TENANT_MODELS, GLOBAL_MODEL

router = APIRouter()


@router.get("/v1/features/schema")
async def schema() -> dict[str, Any]:
    """Return the canonical feature names."""
    return {
        "features": feature_schema(),
        "count": len(feature_schema()),
    }


@router.post("/v1/features/extract")
async def extract(payload: LeadFeatures) -> dict[str, Any]:
    """Return the engineered feature vector for a lead."""
    feats = extract_features_dict(payload)
    schema = feature_schema()
    vec = vectorize(payload)
    return {
        "features": feats,
        "vector": vec,
        "names": schema,
    }


@router.get("/v1/model/versions")
async def model_versions() -> dict[str, Any]:
    """List all tenant models and the global fallback."""
    versions = {t: m.version for t, m in TENANT_MODELS.items()}
    global_version = GLOBAL_MODEL.version if GLOBAL_MODEL else None
    return {
        "global_model": global_version,
        "tenant_models": versions,
        "total_loaded": len(TENANT_MODELS) + (1 if GLOBAL_MODEL else 0),
    }


__all__ = ["router"]
