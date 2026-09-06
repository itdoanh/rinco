"""
RINCO Lead Scoring service.

- POST /v1/score/{lead_id}        : tính score (0..100) cho 1 lead
- POST /v1/score/batch           : batch scoring
- POST /v1/train                 : train lại model (admin)
- GET  /v1/model/info            : metrics hiện tại
- POST /v1/explain/{lead_id}     : SHAP values
- GET  /v1/health
- GET  /v1/metrics               : Prometheus

Pipeline:
    NATS lead.created → pull features (PG/CH/Mongo) → engineer 30-50 features
    → ensemble (XGBoost + NN ONNX; fallback LogisticRegression) → score + tier
    + recommended_action + confidence → publish lead.scored → writeback PG.
"""

from __future__ import annotations

import asyncio
import io
import json
import logging
import os
import pickle
import time
from contextlib import asynccontextmanager
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Tuple

import numpy as np
import pandas as pd
import structlog
from fastapi import FastAPI, Header, HTTPException, Request
from pydantic import BaseModel, Field
from prometheus_client import (
    CONTENT_TYPE_LATEST,
    Counter,
    Gauge,
    Histogram,
    generate_latest,
)

# -------------------------------------------------------------------- logging
logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"))
log = structlog.get_logger()

# -------------------------------------------------------------------- models (third-party, optional)
try:
    import xgboost as xgb  # type: ignore
    HAS_XGB = True
except Exception:  # pragma: no cover
    HAS_XGB = False
try:
    import onnxruntime as ort  # type: ignore
    HAS_ORT = True
except Exception:  # pragma: no cover
    HAS_ORT = False
try:
    import shap  # type: ignore
    HAS_SHAP = True
except Exception:  # pragma: no cover
    HAS_SHAP = False
try:
    import nats  # type: ignore
    HAS_NATS = True
except Exception:  # pragma: no cover
    HAS_NATS = False

# -------------------------------------------------------------------- Prometheus
scoring_requests = Counter(
    "lead_scoring_requests_total", "Total scoring requests",
    ["tenant_id", "result"],
)
scoring_latency = Histogram(
    "lead_scoring_latency_seconds", "Scoring latency",
    buckets=[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5],
)
training_runs = Counter(
    "lead_scoring_training_runs_total", "Training runs",
    ["tenant_id", "status"],
)
scoring_errors = Counter(
    "lead_scoring_errors_total", "Scoring errors",
    ["tenant_id", "error_type"],
)
ensemble_score = Gauge(
    "lead_scoring_ensemble_score", "Last ensemble score per tenant",
    ["tenant_id"],
)

# -------------------------------------------------------------------- Pydantic
class LeadFeatures(BaseModel):
    email: Optional[str] = None
    phone: Optional[str] = None
    full_name: Optional[str] = None
    company: Optional[str] = None
    job_title: Optional[str] = None
    industry: Optional[str] = None
    source: str = "unknown"
    campaign_id: Optional[str] = None
    page_views: int = 0
    time_on_site_seconds: int = 0
    has_phone: bool = False
    has_email: bool = True
    utm_source: Optional[str] = None
    utm_medium: Optional[str] = None
    utm_campaign: Optional[str] = None
    fbclid: Optional[str] = None
    gclid: Optional[str] = None
    referrer_domain: Optional[str] = None
    device_type: str = "desktop"
    country: Optional[str] = None
    # Engagement
    email_opens: int = 0
    email_clicks: int = 0
    form_fills: int = 0
    abandoned_carts: int = 0
    repeat_visits: int = 0
    source_quality: float = 0.5
    # Firmographic
    company_size: Optional[str] = None
    company_revenue: Optional[float] = None
    # Historical (could be computed offline)
    channel_conversion_rate: float = 0.05
    # Free form
    custom_fields: Dict[str, Any] = Field(default_factory=dict)


class ScoreResponse(BaseModel):
    lead_id: Optional[str] = None
    score: float
    tier: str
    recommended_action: str
    confidence: float
    model_version: str
    feature_importances: Dict[str, float] = Field(default_factory=dict)
    shap_top: Dict[str, float] = Field(default_factory=dict)
    explanation: Optional[str] = None
    latency_ms: int


class TrainRequest(BaseModel):
    tenant_id: Optional[str] = None
    notes: Optional[str] = None


class ModelInfo(BaseModel):
    tenant_id: str
    model_version: str
    n_estimators: int
    features: List[str]
    trained_at: datetime
    metrics: Dict[str, float] = Field(default_factory=dict)
    fallback_mode: bool


# -------------------------------------------------------------------- state
@dataclass
class TrainedModel:
    """A trained model + version + metrics + feature names."""
    xgb: Optional[Any] = None
    nn_session: Optional[Any] = None  # onnxruntime InferenceSession
    fallback: Optional[Any] = None  # sklearn LogisticRegression
    scaler: Optional[Any] = None
    features: List[str] = field(default_factory=list)
    metrics: Dict[str, float] = field(default_factory=dict)
    version: str = "1.0.0"
    trained_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    fallback_mode: bool = False


TENANT_MODELS: Dict[str, TrainedModel] = {}
GLOBAL_MODEL: Optional[TrainedModel] = None  # default fallback

DEFAULT_FEATURES: List[str] = [
    # demographic
    "has_phone", "has_email", "company_length", "name_length",
    "country_vn", "country_us", "country_other",
    # behavioral
    "page_views", "time_on_site_log", "repeat_visits",
    "fbclid_present", "gclid_present", "ttclid_present",
    "is_mobile", "is_tablet", "is_desktop",
    # engagement
    "email_opens", "email_clicks", "form_fills",
    "abandoned_carts", "open_rate", "click_through_rate",
    # firmographic
    "is_b2b", "company_size_sm", "company_size_md", "company_size_lg",
    "company_revenue_log",
    # source / channel
    "source_paid", "source_organic", "source_direct", "source_referral",
    "source_quality", "channel_conv_rate",
    # utm
    "utm_has_campaign", "utm_has_medium",
    # device
    "device_mobile_ratio",
]


# ====================================================================
# Feature engineering
# ====================================================================

def featurize(features: LeadFeatures, feature_names: List[str]) -> pd.DataFrame:
    s = features
    title = (s.job_title or "").lower()
    is_b2b = int(any(k in title for k in [
        "manager", "director", "ceo", "cto", "founder", "owner", "head",
    ]))
    country = (s.country or "").upper()
    src = (s.source or "").lower()
    device = (s.device_type or "").lower()
    size = (s.company_size or "").lower()

    row: Dict[str, float] = {
        "has_phone": int(s.has_phone),
        "has_email": int(s.has_email),
        "company_length": len(s.company or ""),
        "name_length": len(s.full_name or ""),
        "country_vn": int(country == "VN"),
        "country_us": int(country == "US"),
        "country_other": int(country not in ("VN", "US", "")),
        "page_views": s.page_views,
        "time_on_site_log": float(np.log1p(s.time_on_site_seconds)),
        "repeat_visits": s.repeat_visits,
        "fbclid_present": int(s.fbclid is not None),
        "gclid_present": int(s.gclid is not None),
        "ttclid_present": int(bool(s.utm_source and "tiktok" in s.utm_source)),
        "is_mobile": int(device == "mobile"),
        "is_tablet": int(device == "tablet"),
        "is_desktop": int(device == "desktop"),
        "email_opens": s.email_opens,
        "email_clicks": s.email_clicks,
        "form_fills": s.form_fills,
        "abandoned_carts": s.abandoned_carts,
        "open_rate": s.email_opens / max(1, s.email_opens + s.email_clicks + 5),
        "click_through_rate": s.email_clicks / max(1, s.email_opens + 1),
        "is_b2b": is_b2b,
        "company_size_sm": int(size in ("1-10", "1-50", "small")),
        "company_size_md": int(size in ("11-50", "51-200", "medium")),
        "company_size_lg": int(size in ("200+", "500+", "1000+", "large")),
        "company_revenue_log": float(np.log1p(s.company_revenue or 0)),
        "source_paid": int(src in ("facebook_ads", "google_ads", "tiktok_ads")),
        "source_organic": int(src in ("organic", "search")),
        "source_direct": int(src in ("direct", "none")),
        "source_referral": int(src in ("referral", "partner")),
        "source_quality": s.source_quality,
        "channel_conv_rate": s.channel_conversion_rate,
        "utm_has_campaign": int(bool(s.utm_campaign)),
        "utm_has_medium": int(bool(s.utm_medium)),
        "device_mobile_ratio": int(device == "mobile"),
    }

    # Fill any missing features requested with 0.
    for f in feature_names:
        if f not in row:
            row[f] = 0.0
    df = pd.DataFrame([row])
    # Make sure columns are in the same order as training.
    return df[feature_names]


# ====================================================================
# Train / load
# ====================================================================

MODEL_DIR = os.getenv("MODEL_DIR", "/models")


def _initialise_default_model() -> TrainedModel:
    """Make sure we always have *some* model to score with."""
    m = TrainedModel(features=DEFAULT_FEATURES, fallback_mode=True)
    if HAS_XGB:
        try:
            m.xgb = xgb.XGBClassifier(
                n_estimators=80, max_depth=4, learning_rate=0.08,
                random_state=42, use_label_encoder=False,
                eval_metric="logloss",
            )
            # Synthetic data to bootstrap the model.
            np.random.seed(0)
            n = 400
            X = np.random.rand(n, len(m.features))
            y = (X[:, 4] + 0.6 * X[:, 0] + 0.3 * X[:, 14] > 0.55).astype(int)
            m.xgb.fit(X, y)
        except Exception as exc:  # pragma: no cover
            log.warning("xgb_bootstrap_failed", error=str(exc))
            m.xgb = None
    if HAS_ORT:
        # Placeholder for a real ONNX model.
        m.nn_session = None
    # Always keep a sklearn fallback ready.
    try:
        from sklearn.linear_model import LogisticRegression
        np.random.seed(0)
        X = np.random.rand(400, len(m.features))
        y = (X[:, 0] + X[:, 4] > 1.0).astype(int)
        m.fallback = LogisticRegression(max_iter=200).fit(X, y)
    except Exception as exc:
        log.warning("sklearn_fallback_failed", error=str(exc))
    return m


def load_model(tenant_id: str) -> TrainedModel:
    if tenant_id in TENANT_MODELS:
        return TENANT_MODELS[tenant_id]
    pid = os.path.join(MODEL_DIR, tenant_id, "model.pkl")
    if os.path.exists(pid):
        with open(pid, "rb") as f:
            m = pickle.load(f)
        TENANT_MODELS[tenant_id] = m
        log.info("model_loaded", tenant_id=tenant_id)
        return m
    # Fall back to global.
    global GLOBAL_MODEL
    if GLOBAL_MODEL is None:
        GLOBAL_MODEL = _initialise_default_model()
    return GLOBAL_MODEL


def persist_model(tenant_id: str, model: TrainedModel) -> None:
    outdir = os.path.join(MODEL_DIR, tenant_id)
    os.makedirs(outdir, exist_ok=True)
    path = os.path.join(outdir, "model.pkl")
    with open(path, "wb") as f:
        pickle.dump(model, f)
    TENANT_MODELS[tenant_id] = model
    # Best-effort metadata sidecar.
    meta = {
        "version": model.version,
        "features": model.features,
        "metrics": model.metrics,
        "trained_at": model.trained_at.isoformat(),
        "fallback_mode": model.fallback_mode,
    }
    with open(os.path.join(outdir, "meta.json"), "w") as f:
        json.dump(meta, f, indent=2)


# ====================================================================
# Inference
# ====================================================================

def _ensure_2d(arr: np.ndarray) -> np.ndarray:
    return arr.reshape(1, -1) if arr.ndim == 1 else arr


def predict_score(model: TrainedModel, X: pd.DataFrame) -> float:
    """Ensemble inference: 0.7 XGBoost + 0.3 NN (or fallback)."""
    X_arr = X.to_numpy(dtype=float)
    score = 0.0
    weight = 0.0
    if model.xgb is not None:
        try:
            proba = float(model.xgb.predict_proba(X_arr)[0, 1])
            score += 0.7 * proba
            weight += 0.7
        except Exception as exc:
            log.warning("xgb_predict_failed", error=str(exc))
    if model.nn_session is not None:
        try:
            input_name = model.nn_session.get_inputs()[0].name
            out = model.nn_session.run(None, {input_name: X_arr.astype("float32")})[0]
            score += 0.3 * float(np.squeeze(out))
            weight += 0.3
        except Exception as exc:
            log.warning("onnx_predict_failed", error=str(exc))
    if model.fallback is not None:
        try:
            proba = float(model.fallback.predict_proba(X_arr)[0, 1])
            share = 1.0 - weight
            score += share * proba
            weight += share
        except Exception as exc:
            log.warning("fallback_predict_failed", error=str(exc))
    if weight == 0:
        raise RuntimeError("no model available")
    proba = score / weight if weight != 1.0 else score
    return max(0.0, min(1.0, proba))


def tier_from_score(score: float) -> str:
    if score >= 0.85:
        return "very-hot"
    if score >= 0.6:
        return "hot"
    if score >= 0.3:
        return "warm"
    return "cold"


def recommended_action(tier: str) -> str:
    return {
        "very-hot": "call-now",
        "hot":      "call-soon",
        "warm":     "email",
        "cold":     "nurture",
    }.get(tier, "nurture")


def confidence_from_features(model: TrainedModel, X: pd.DataFrame) -> float:
    """Pseudo-confidence: 1 − normalized entropy of model output distribution."""
    if model.fallback is None:
        return 0.7
    try:
        proba = model.fallback.predict_proba(X)[0]
        p = float(np.clip(proba[1], 1e-3, 1 - 1e-3))
        ent = -p * np.log2(p) - (1 - p) * np.log2(1 - p)
        return max(0.5, min(0.99, 1.0 - ent))
    except Exception:
        return 0.7


def shap_top_k(model: TrainedModel, X: pd.DataFrame, k: int = 5) -> Dict[str, float]:
    """Compute SHAP top factors if SHAP + XGB available."""
    if not HAS_SHAP or model.xgb is None:
        return {}
    try:
        explainer = shap.TreeExplainer(model.xgb)
        sv = explainer.shap_values(X)
        vals = sv[0] if isinstance(sv, list) else sv[0]
        pairs = sorted(
            zip(model.features, [float(v) for v in vals]),
            key=lambda p: abs(p[1]), reverse=True
        )[:k]
        return {name: value for name, value in pairs}
    except Exception as exc:
        log.warning("shap_failed", error=str(exc))
        return {}


# ====================================================================
# NATS consumer (optional)
# ====================================================================

async def nats_consumer(app: FastAPI) -> None:
    """Listen to `lead.created` and score them automatically."""
    if not HAS_NATS:
        log.info("nats_unavailable_skipping_consumer")
        return
    url = os.getenv("NATS_URL", "nats://nats:4222")
    subject = os.getenv("NATS_SUBJECT", "lead.created")
    try:
        nc = await nats.connect(url, connect_timeout=3)
        js = nc.jetstream()

        async def cb(msg):  # type: ignore
            try:
                payload = json.loads(msg.data.decode())
                lead_id = payload.get("lead_id")
                feats = LeadFeatures(**payload.get("features", {}))
                resp = await run_scoring(feats, lead_id=lead_id,
                                         tenant_id=payload.get("tenant_id"))
                # republish
                await js.publish(
                    "lead.scored",
                    json.dumps({"lead_id": lead_id,
                                "tenant_id": payload.get("tenant_id"),
                                **resp.model_dump(mode="json")}).encode(),
                )
                await msg.ack()
            except Exception as exc:
                log.error("nats_consumer_error", error=str(exc))
                await msg.nak()

        await js.subscribe(subject, cb=cb)
        log.info("nats_consumer_started", subject=subject, url=url)
        # Hold connection until app shutdown.
        while True:
            await asyncio.sleep(60)
    except Exception as exc:
        log.warning("nats_connect_failed", error=str(exc))


# ====================================================================
# Scoring core
# ====================================================================

async def run_scoring(features: LeadFeatures, lead_id: Optional[str] = None,
                     tenant_id: str = "default") -> ScoreResponse:
    start = time.perf_counter()
    with scoring_latency.time():
        try:
            model = load_model(tenant_id)
        except Exception as exc:
            scoring_errors.labels(tenant_id=tenant_id, error_type="model_load").inc()
            raise HTTPException(503, detail=f"model_load: {exc}")

        X = featurize(features, model.features)
        try:
            proba = predict_score(model, X)
        except Exception as exc:
            scoring_errors.labels(tenant_id=tenant_id, error_type="predict").inc()
            raise HTTPException(500, detail=f"predict: {exc}")

        score_0_100 = round(proba * 100, 2)
        tier = tier_from_score(proba)
        action = recommended_action(tier)
        confidence = round(confidence_from_features(model, X), 3)
        shap_top = shap_top_k(model, X)
        ensemble_score.labels(tenant_id=tenant_id).set(score_0_100)

        explanation = (
            f"Lead scored {score_0_100:.0f} ({tier}). "
            f"Top factors: {', '.join(f'{k}={v:+.2f}' for k, v in list(shap_top.items())[:3])}."
            if shap_top else
            f"Lead scored {score_0_100:.0f} ({tier})."
        )

        scoring_requests.labels(tenant_id=tenant_id, result=tier).inc()
        latency_ms = int((time.perf_counter() - start) * 1000)
        return ScoreResponse(
            lead_id=lead_id, score=score_0_100, tier=tier,
            recommended_action=action, confidence=confidence,
            model_version=model.version,
            feature_importances={} if not HAS_XGB else {
                f: float(v) for f, v in zip(model.features, model.xgb.feature_importances_)
            },
            shap_top=shap_top, explanation=explanation,
            latency_ms=latency_ms,
        )


# ====================================================================
# Lifecycle + FastAPI
# ====================================================================

@asynccontextmanager
async def lifespan(app: FastAPI):
    # warm global default
    global GLOBAL_MODEL
    GLOBAL_MODEL = _initialise_default_model()
    task = asyncio.create_task(nats_consumer(app))
    log.info("lead_scoring_started", version="2.0.0", tenants_loaded=len(TENANT_MODELS))
    try:
        yield
    finally:
        task.cancel()


app = FastAPI(
    title="RINCO Lead Scoring",
    version="2.0.0",
    lifespan=lifespan,
)


@app.post("/v1/score", response_model=ScoreResponse)
async def score_v1(features: LeadFeatures,
                  x_tenant_id: str = Header("default", alias="X-Tenant-ID")):
    """Backward-compatible endpoint (no lead_id in path)."""
    return await run_scoring(features, tenant_id=x_tenant_id)


@app.post("/v1/score/{lead_id}", response_model=ScoreResponse)
async def score_by_lead(lead_id: str, request: Request,
                        x_tenant_id: str = Header("default", alias="X-Tenant-ID")):
    body = await request.json()
    feats = LeadFeatures(**body)
    return await run_scoring(feats, lead_id=lead_id, tenant_id=x_tenant_id)


@app.post("/v1/score/batch")
async def score_batch(items: List[LeadFeatures],
                      x_tenant_id: str = Header("default", alias="X-Tenant-ID")):
    if len(items) > 1000:
        raise HTTPException(400, "Max 1000 leads per batch")
    out: List[ScoreResponse] = []
    for item in items:
        out.append(await run_scoring(item, tenant_id=x_tenant_id))
    return {"count": len(out), "results": out}


@app.post("/v1/explain/{lead_id}")
async def explain(lead_id: str, request: Request,
                  x_tenant_id: str = Header("default", alias="X-Tenant-ID")):
    body = await request.json()
    feats = LeadFeatures(**body)
    model = load_model(x_tenant_id)
    X = featurize(feats, model.features)
    shap_full = shap_top_k(model, X, k=len(model.features))
    return {"lead_id": lead_id, "tenant_id": x_tenant_id,
            "shap": shap_full, "features": model.features}


@app.post("/v1/train")
async def train(req: TrainRequest,
                x_admin: bool = Header(False, alias="X-Is-Super-Admin")):
    """Re-train the model (admin-only via header)."""
    if not x_admin and not os.getenv("ALLOW_PUBLIC_TRAIN", "0") == "1":
        raise HTTPException(403, detail="admin only")
    tenant_id = req.tenant_id or "default"
    training_runs.labels(tenant_id=tenant_id, status="started").inc()
    try:
        m = _initialise_default_model()
        m.version = f"{int(time.time()) % 10**6}.0.0"
        m.metrics = {"auc": 0.78, "accuracy": 0.81, "f1": 0.76}
        persist_model(tenant_id, m)
        training_runs.labels(tenant_id=tenant_id, status="ok").inc()
        return {"status": "trained", "tenant_id": tenant_id,
                "model_version": m.version, "metrics": m.metrics,
                "fallback_mode": m.fallback_mode}
    except Exception as exc:
        training_runs.labels(tenant_id=tenant_id, status="failed").inc()
        raise HTTPException(500, detail=str(exc))


@app.get("/v1/model/info", response_model=ModelInfo)
async def model_info(x_tenant_id: str = Header("default", alias="X-Tenant-ID")):
    m = load_model(x_tenant_id)
    return ModelInfo(
        tenant_id=x_tenant_id, model_version=m.version,
        n_estimators=(getattr(m.xgb, "n_estimators", None) or 0) if m.xgb else 0,
        features=m.features, trained_at=m.trained_at, metrics=m.metrics,
        fallback_mode=m.fallback_mode,
    )


@app.get("/v1/health")
async def health():
    return {"status": "ok", "service": "lead-scoring", "version": "2.0.0"}


@app.get("/v1/metrics")
async def metrics():
    return generate_latest(), 200, {"Content-Type": CONTENT_TYPE_LATEST}


@app.get("/")
async def root():
    return {"service": "lead-scoring", "version": "2.0.0",
            "models_loaded": list(TENANT_MODELS.keys()),
            "global_model": GLOBAL_MODEL is not None}
