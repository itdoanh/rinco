"""Public anomaly and scaling endpoints for the AI SRE service.

Routes:

- ``POST /v1/anomalies/check``         – run detectors over supplied samples
- ``POST /v1/anomalies/observe``       – push a sample, return triggered anomalies
- ``GET  /v1/anomalies``               – list all currently detected anomalies
- ``POST /v1/scaling/recommend``       – derive a replica count from a forecast
- ``POST /v1/scaling/plan``            – end-to-end capacity plan for a service
- ``POST /v1/forecast``                – run a linear forecast over supplied samples

The streaming detectors live in
:mod:`app.services.anomaly_detector` and the forecasting helpers in
:mod:`app.services.performance_predictor`.
"""
from __future__ import annotations

from typing import Any

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

from app.core import get_logger, METRICS
from app.services import anomaly_detector, performance_predictor

router = APIRouter(prefix="/v1", tags=["aiops"])
logger = get_logger("api.aiops")


# ---------------------------------------------------------------------------
# In-memory state (singleton)
# ---------------------------------------------------------------------------
_detector = anomaly_detector.StreamingAnomalyDetector()
_forecaster = performance_predictor.ForecastRegistry()


def _bump(label: str) -> None:
    counter = METRICS.get("analysis_requests")
    if counter:
        counter.labels(tenant_id="default", result=label).inc()


# ---------------------------------------------------------------------------
# Schemas
# ---------------------------------------------------------------------------
class AnomalyCheckRequest(BaseModel):
    metric: str = Field(..., description="Metric name (e.g. 'cpu_usage')")
    samples: list[float] = Field(..., description="Recent samples, oldest first")


class AnomalyObserveRequest(BaseModel):
    metric: str
    value: float


class ScalingRecommendRequest(BaseModel):
    service: str
    metric: str = "cpu"
    current_replicas: int = Field(..., ge=1)
    forecast_horizon_minutes: int = Field(30, ge=1, le=1440)
    predicted_peak: float | None = Field(
        None,
        description="Optional override for the predicted peak; "
                    "if absent a flat forecast is generated.",
    )
    target_utilization: float = 0.7
    min_replicas: int = 1
    max_replicas: int = 20
    safety_margin: float = 1.25


class CapacityPlanRequest(BaseModel):
    service: str
    current_replicas: int = Field(..., ge=1)
    metrics: dict[str, list[float]] = Field(
        default_factory=dict,
        description="Map of metric-name → samples (oldest first)",
    )
    horizon_minutes: int = 30


class ForecastRequest(BaseModel):
    metric: str
    samples: list[float]
    horizon_minutes: int = Field(30, ge=1, le=1440)
    sample_interval_seconds: float = 60.0


# ---------------------------------------------------------------------------
# Anomaly endpoints
# ---------------------------------------------------------------------------
@router.post("/anomalies/check")
async def check_anomalies(req: AnomalyCheckRequest) -> dict[str, Any]:
    """Run all detectors against ``req.samples`` and return anomalies."""
    if len(req.samples) < 5:
        raise HTTPException(400, "need at least 5 samples")

    anomalies: list[dict[str, Any]] = []
    anomalies.extend(a.to_dict() for a in anomaly_detector.zscore_detect(req.metric, req.samples))
    anomalies.extend(a.to_dict() for a in anomaly_detector.iqr_detect(req.metric, req.samples))
    anomalies.extend(a.to_dict() for a in anomaly_detector.ewma_detect(req.metric, req.samples))

    _bump("anomaly_check")
    return {
        "metric": req.metric,
        "summary": anomaly_detector.summarise([anomaly_detector.Anomaly(**a) for a in anomalies]),
        "anomalies": anomalies,
    }


@router.post("/anomalies/observe")
async def observe_anomaly(req: AnomalyObserveRequest) -> dict[str, Any]:
    """Push a sample into the streaming window and return any anomalies."""
    triggered = _detector.observe(req.metric, req.value)
    _bump("anomaly_observe")
    return {
        "metric": req.metric,
        "anomalies": [a.to_dict() for a in triggered],
    }


@router.get("/anomalies")
async def list_anomalies() -> dict[str, Any]:
    """List currently detected anomalies across all metric windows."""
    snapshot = _detector.detect_all()
    flat: list[dict[str, Any]] = []
    for metric, items in snapshot.items():
        for a in items:
            d = a.to_dict()
            d["metric"] = metric
            flat.append(d)
    return {
        "count": len(flat),
        "by_metric": {m: len(items) for m, items in snapshot.items()},
        "anomalies": flat,
    }


# ---------------------------------------------------------------------------
# Scaling endpoints
# ---------------------------------------------------------------------------
@router.post("/scaling/recommend")
async def scaling_recommend(req: ScalingRecommendRequest) -> dict[str, Any]:
    """Produce a scaling recommendation for a single service+metric."""
    forecast = performance_predictor.predict_metric(
        req.metric,
        [req.predicted_peak] if req.predicted_peak is not None else [req.target_utilization],
        horizon_minutes=req.forecast_horizon_minutes,
    )
    if req.predicted_peak is not None:
        # Replace the auto forecast with the user-supplied peak.
        forecast.predicted = [req.predicted_peak] * req.forecast_horizon_minutes
        forecast.upper_bound = req.predicted_peak * 1.2
        forecast.lower_bound = req.predicted_peak * 0.8

    policy = performance_predictor.ScalingPolicy(
        service=req.service,
        metric=req.metric,
        target_utilization=req.target_utilization,
        min_replicas=req.min_replicas,
        max_replicas=req.max_replicas,
        safety_margin=req.safety_margin,
    )
    rec = performance_predictor.recommend_scaling(
        service=req.service,
        current_replicas=req.current_replicas,
        forecast=forecast,
        policy=policy,
    )
    _bump("scaling_recommend")
    return {
        "service": req.service,
        "metric": req.metric,
        "forecast": forecast.to_dict(),
        "recommendation": rec.to_dict(),
    }


@router.post("/scaling/plan")
async def capacity_plan(req: CapacityPlanRequest) -> dict[str, Any]:
    """Capacity plan for a service across multiple metrics."""
    if not req.metrics:
        raise HTTPException(400, "metrics is required")

    recs: list[dict[str, Any]] = []
    for metric, samples in req.metrics.items():
        forecast = performance_predictor.predict_metric(
            metric, samples, horizon_minutes=req.horizon_minutes,
        )
        policy = performance_predictor.ScalingPolicy.default(req.service, metric=metric)
        rec = performance_predictor.recommend_scaling(
            service=req.service,
            current_replicas=req.current_replicas,
            forecast=forecast,
            policy=policy,
        )
        recs.append(rec.to_dict())

    _bump("capacity_plan")
    return {
        "service": req.service,
        "current_replicas": req.current_replicas,
        "horizon_minutes": req.horizon_minutes,
        "recommendations": recs,
        "summary": performance_predictor.summarise_recommendations(
            [performance_predictor.Recommendation(**r) for r in recs]
        ),
    }


# ---------------------------------------------------------------------------
# Forecast endpoint
# ---------------------------------------------------------------------------
@router.post("/forecast")
async def forecast(req: ForecastRequest) -> dict[str, Any]:
    """Run a linear forecast on the supplied samples."""
    f = performance_predictor.predict_metric(
        req.metric,
        req.samples,
        horizon_minutes=req.horizon_minutes,
        sample_interval_seconds=req.sample_interval_seconds,
    )
    _bump("forecast")
    return f.to_dict()


__all__ = ["router"]
