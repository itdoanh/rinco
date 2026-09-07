"""Scoring endpoints.

The handler is implemented as a module-level singleton of the scoring
function so the NATS consumer (in app/services/nats_consumer.py) can
call the same code path.
"""
from __future__ import annotations

import time
from typing import List, Optional

from fastapi import APIRouter, Header, HTTPException, Request

from app.core.logging import get_logger
from app.core.metrics import (
    ENSEMBLE_SCORE,
    ERRORS_TOTAL,
    INFERENCE_DURATION,
    REQUESTS_TOTAL,
)
from app.schemas.lead import LeadFeatures
from app.schemas.score import HealthResponse, ScoreResponse
from app.services.features import featurize
from app.services.inference import (
    _initialise_default_model,
    confidence_from_features,
    load_model,
    predict_score,
    recommended_action,
    shap_top_k,
    tier_from_score,
)

log = get_logger("lead-scoring.scoring")

router = APIRouter()


async def _run_scoring(features: LeadFeatures, lead_id: Optional[str] = None,
                       tenant_id: str = "default") -> ScoreResponse:
    start = time.perf_counter()
    try:
        model = load_model(tenant_id)
    except Exception as exc:
        ERRORS_TOTAL.labels(tenant_id=tenant_id, error_type="model_load").inc()
        raise HTTPException(503, detail=f"model_load: {exc}")

    X = featurize(features, model.features)
    try:
        with INFERENCE_DURATION.labels(model_version=model.version).time():
            proba = predict_score(model, X)
    except Exception as exc:
        ERRORS_TOTAL.labels(tenant_id=tenant_id, error_type="predict").inc()
        raise HTTPException(500, detail=f"predict: {exc}")

    score_0_100 = round(proba * 100, 2)
    tier = tier_from_score(proba)
    action = recommended_action(tier)
    confidence = round(confidence_from_features(model, X), 3)
    shap_top = shap_top_k(model, X)
    ENSEMBLE_SCORE.labels(tenant_id=tenant_id).set(score_0_100)
    REQUESTS_TOTAL.labels(tenant_id=tenant_id, result=tier).inc()

    explanation = (
        f"Lead scored {score_0_100:.0f} ({tier}). "
        f"Top factors: {', '.join(f'{k}={v:+.2f}' for k, v in list(shap_top.items())[:3])}."
        if shap_top
        else f"Lead scored {score_0_100:.0f} ({tier})."
    )
    latency_ms = int((time.perf_counter() - start) * 1000)
    return ScoreResponse(
        lead_id=lead_id,
        score=score_0_100,
        tier=tier,
        recommended_action=action,
        confidence=confidence,
        model_version=model.version,
        shap_top=shap_top,
        explanation=explanation,
        latency_ms=latency_ms,
    )


@router.post("/v1/score", response_model=ScoreResponse)
async def score_v1(
    features: LeadFeatures,
    x_tenant_id: str = Header("default", alias="X-Tenant-ID"),
):
    return await _run_scoring(features, tenant_id=x_tenant_id)


@router.post("/v1/score/{lead_id}", response_model=ScoreResponse)
async def score_by_lead(
    lead_id: str,
    request: Request,
    x_tenant_id: str = Header("default", alias="X-Tenant-ID"),
):
    body = await request.json()
    feats = LeadFeatures(**body)
    return await _run_scoring(feats, lead_id=lead_id, tenant_id=x_tenant_id)


@router.post("/v1/score/batch")
async def score_batch(
    items: List[LeadFeatures],
    x_tenant_id: str = Header("default", alias="X-Tenant-ID"),
):
    if len(items) > 1000:
        raise HTTPException(400, "Max 1000 leads per batch")
    out: List[ScoreResponse] = []
    for item in items:
        out.append(await _run_scoring(item, tenant_id=x_tenant_id))
    return {"count": len(out), "results": out}


@router.get("/v1/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    from app.services.inference import GLOBAL_MODEL, TENANT_MODELS

    return HealthResponse(
        status="ok",
        service="lead-scoring",
        version="2.0.0",
        models_loaded=list(TENANT_MODELS.keys()),
        fallback_mode=GLOBAL_MODEL is not None and GLOBAL_MODEL.fallback_mode,
    )


@router.get("/v1/metrics")
async def metrics():
    from app.core.metrics import render

    body, content_type = render()
    return body, 200, {"Content-Type": content_type}


__all__ = ["router", "_run_scoring"]