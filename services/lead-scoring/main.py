"""
Lead Scoring Service - ML-based lead scoring với LightGBM.

Multi-tenant: mỗi tenant có model riêng, fallback to default.
"""
import asyncio
import json
import os
from datetime import datetime
from typing import Any

import joblib
import numpy as np
import pandas as pd
from fastapi import FastAPI, Header, HTTPException
from pydantic import BaseModel, Field
import structlog
from prometheus_client import Counter, Histogram, generate_latest
from prometheus_client import CONTENT_TYPE_LATEST

logger = structlog.get_logger()
app = FastAPI(title="Lead Scoring Service", version="1.0.0")

scoring_requests = Counter(
    "lead_scoring_requests_total",
    "Total scoring requests",
    ["tenant_id", "result"],
)
scoring_latency = Histogram(
    "lead_scoring_latency_seconds",
    "Scoring latency",
    buckets=[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5],
)
scoring_errors = Counter(
    "lead_scoring_errors_total",
    "Total scoring errors",
    ["tenant_id", "error_type"],
)

MODELS: dict[str, Any] = {}  # tenant_id → (model, features, version)


class LeadFeatures(BaseModel):
    email: str | None = None
    phone: str | None = None
    full_name: str | None = None
    company: str | None = None
    job_title: str | None = None
    industry: str | None = None
    source: str = "unknown"
    campaign_id: str | None = None
    page_views: int = 0
    time_on_site_seconds: int = 0
    has_phone: bool = False
    has_email: bool = True
    utm_source: str | None = None
    utm_medium: str | None = None
    utm_campaign: str | None = None
    fbclid: str | None = None
    gclid: str | None = None
    referrer_domain: str | None = None
    device_type: str = "desktop"
    country: str | None = None
    custom_fields: dict[str, Any] = Field(default_factory=dict)


class ScoreResponse(BaseModel):
    score: float
    score_band: str
    model_version: str
    feature_importances: dict[str, float] | None = None
    explanation: str | None = None
    latency_ms: int


def load_model(tenant_id: str) -> tuple[Any, list[str], str]:
    """Lazy load tenant model."""
    if tenant_id in MODELS:
        return MODELS[tenant_id]

    base = os.getenv("MODEL_DIR", "/models")
    path = f"{base}/{tenant_id}/model.joblib"

    if not os.path.exists(path):
        # Fallback to global default
        path = f"{base}/default/model.joblib"
        if not os.path.exists(path):
            raise FileNotFoundError(f"No model for {tenant_id} and no default")

    model = joblib.load(path)
    meta_path = path.replace("model.joblib", "meta.json")
    with open(meta_path) as f:
        meta = json.load(f)

    result = (model, meta["features"], meta["version"])
    MODELS[tenant_id] = result
    logger.info("model_loaded", tenant_id=tenant_id, version=meta["version"])
    return result


def featurize(features: LeadFeatures, feature_names: list[str]) -> pd.DataFrame:
    """Build feature vector."""
    is_b2b = any(
        k in (features.job_title or "").lower()
        for k in ["manager", "director", "ceo", "cto", "founder"]
    )
    data = {
        "has_phone": int(features.has_phone),
        "has_email": int(features.has_email),
        "page_views": features.page_views,
        "time_on_site_log": np.log1p(features.time_on_site_seconds),
        "fbclid_present": int(features.fbclid is not None),
        "gclid_present": int(features.gclid is not None),
        "is_mobile": int(features.device_type == "mobile"),
        "is_tablet": int(features.device_type == "tablet"),
        "is_desktop": int(features.device_type == "desktop"),
        "company_length": len(features.company or ""),
        "name_length": len(features.full_name or ""),
        "is_b2b": int(is_b2b),
        "country_vn": int((features.country or "") == "VN"),
        "source_paid": int((features.source or "").lower() in ("facebook_ads", "google_ads", "tiktok_ads")),
        "source_organic": int((features.source or "").lower() in ("organic", "direct", "referral")),
    }
    # Fill missing features
    for f in feature_names:
        if f not in data:
            data[f] = 0

    return pd.DataFrame([data])[feature_names]


def score_band_from_score(score: float) -> str:
    if score >= 0.8:
        return "hot"
    if score >= 0.5:
        return "warm"
    if score >= 0.2:
        return "cold"
    return "low"


def generate_explanation(importances: dict[str, float], band: str) -> str:
    if not importances:
        return f"Lead scored as '{band}'."
    top = sorted(importances.items(), key=lambda x: -x[1])[:3]
    factors = ", ".join(f"{k} ({v:.2f})" for k, v in top)
    return f"Lead scored as '{band}'. Top factors: {factors}."


@app.post("/score", response_model=ScoreResponse)
async def score(
    features: LeadFeatures,
    x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
):
    """Score a lead."""
    start = asyncio.get_event_loop().time()
    with scoring_latency.time():
        try:
            model, feature_names, version = load_model(x_tenant_id)
        except FileNotFoundError as e:
            scoring_errors.labels(tenant_id=x_tenant_id, error_type="model_not_found").inc()
            logger.error("model_load_failed", tenant_id=x_tenant_id, error=str(e))
            raise HTTPException(status_code=503, detail="Model not available")
        except Exception as e:
            scoring_errors.labels(tenant_id=x_tenant_id, error_type="model_load").inc()
            logger.error("model_load_failed", tenant_id=x_tenant_id, error=str(e))
            raise HTTPException(status_code=500, detail="Model load failed")

        df = featurize(features, feature_names)
        try:
            proba = float(model.predict_proba(df)[0, 1])
        except Exception as e:
            scoring_errors.labels(tenant_id=x_tenant_id, error_type="predict").inc()
            logger.error("predict_failed", tenant_id=x_tenant_id, error=str(e))
            raise HTTPException(status_code=500, detail="Prediction failed")

        band = score_band_from_score(proba)

        importances = {}
        if hasattr(model, "feature_importances_"):
            importances = {
                k: float(v) for k, v in zip(feature_names, model.feature_importances_)
            }

        scoring_requests.labels(tenant_id=x_tenant_id, result=band).inc()
        latency_ms = int((asyncio.get_event_loop().time() - start) * 1000)

        return ScoreResponse(
            score=proba,
            score_band=band,
            model_version=version,
            feature_importances=importances,
            explanation=generate_explanation(importances, band),
            latency_ms=latency_ms,
        )


@app.post("/batch-score")
async def batch_score(
    items: list[LeadFeatures],
    x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
):
    """Batch scoring."""
    if len(items) > 1000:
        raise HTTPException(400, "Max 1000 leads per batch")

    try:
        model, feature_names, version = load_model(x_tenant_id)
    except FileNotFoundError:
        raise HTTPException(503, "Model not available")

    df = pd.concat([featurize(item, feature_names) for item in items], ignore_index=True)
    probas = model.predict_proba(df)[:, 1]

    return {
        "model_version": version,
        "scores": [
            {
                "score": float(p),
                "score_band": score_band_from_score(float(p)),
            }
            for p in probas
        ],
    }


@app.post("/retrain")
async def retrain(x_tenant_id: str = Header(..., alias="X-Tenant-ID")):
    """Trigger model retrain (admin only)."""
    # In production: dispatch to ML pipeline
    logger.info("retrain_requested", tenant_id=x_tenant_id)
    return {"status": "queued", "tenant_id": x_tenant_id}


@app.get("/health")
async def health():
    return {"status": "ok", "service": "lead-scoring"}


@app.get("/metrics")
async def metrics():
    return generate_latest(), 200, {"Content-Type": CONTENT_TYPE_LATEST}


@app.on_event("startup")
async def warm_cache():
    """Pre-load common tenant models."""
    tenants = os.getenv("PRELOAD_TENANTS", "default").split(",")
    for t in tenants:
        if t:
            try:
                load_model(t)
                logger.info("preloaded", tenant_id=t)
            except Exception as e:
                logger.warning("preload_failed", tenant_id=t, error=str(e))
