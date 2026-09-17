"""Performance prediction and auto-scaling recommendations.

A simple, dependency-free forecaster: linear regression over a sliding
window plus a safety margin.  The point is to produce *operationally
useful* recommendations (e.g. "scale up to 6 replicas") without
requiring Prophet/LSTM to be present at runtime.

Designed for short-horizon forecasts (15-60 minutes) on metrics like:

- request rate
- p99 latency
- CPU/memory utilisation
- queue depth
"""
from __future__ import annotations

import math
import time
from collections import deque
from dataclasses import dataclass, field
from typing import Any, Deque, Iterable, Sequence


@dataclass
class Forecast:
    """A linear forecast."""

    metric: str
    horizon_minutes: int
    predicted: list[float]
    current: float
    trend_per_minute: float
    confidence: float
    upper_bound: float
    lower_bound: float

    def to_dict(self) -> dict[str, Any]:
        return {
            "metric": self.metric,
            "horizon_minutes": self.horizon_minutes,
            "predicted": [round(v, 4) for v in self.predicted],
            "current": round(self.current, 4),
            "trend_per_minute": round(self.trend_per_minute, 6),
            "confidence": round(self.confidence, 4),
            "upper_bound": round(self.upper_bound, 4),
            "lower_bound": round(self.lower_bound, 4),
        }


@dataclass
class Recommendation:
    """A scaling recommendation."""

    service: str
    metric: str
    action: str  # scale_up | scale_down | hold | add_capacity
    current_replicas: int
    recommended_replicas: int
    rationale: str
    severity: str  # low | medium | high | critical
    forecast_horizon_minutes: int
    predicted_peak: float
    timestamp: float = field(default_factory=time.time)

    def to_dict(self) -> dict[str, Any]:
        return {
            "service": self.service,
            "metric": self.metric,
            "action": self.action,
            "current_replicas": self.current_replicas,
            "recommended_replicas": self.recommended_replicas,
            "rationale": self.rationale,
            "severity": self.severity,
            "forecast_horizon_minutes": self.forecast_horizon_minutes,
            "predicted_peak": round(self.predicted_peak, 4),
            "timestamp": self.timestamp,
        }


def linear_forecast(
    samples: Sequence[float],
    horizon_minutes: int,
    *,
    sample_interval_seconds: float = 60.0,
) -> Forecast:
    """Linear regression forecast over ``samples`` (oldest → newest).

    The output contains ``horizon_minutes`` predicted values starting at
    ``now``; bounds are computed from the residuals of the fit.
    """
    n = len(samples)
    if n < 2:
        return Forecast(
            metric="?",
            horizon_minutes=horizon_minutes,
            predicted=[float(samples[-1])] * horizon_minutes if samples else [0.0] * horizon_minutes,
            current=float(samples[-1]) if samples else 0.0,
            trend_per_minute=0.0,
            confidence=0.0,
            upper_bound=float(samples[-1]) if samples else 0.0,
            lower_bound=float(samples[-1]) if samples else 0.0,
        )

    xs = [i * sample_interval_seconds for i in range(n)]
    ys = [float(v) for v in samples]
    mean_x = sum(xs) / n
    mean_y = sum(ys) / n
    num = sum((xs[i] - mean_x) * (ys[i] - mean_y) for i in range(n))
    den = sum((xs[i] - mean_x) ** 2 for i in range(n)) or 1e-9
    slope = num / den  # value per second
    intercept = mean_y - slope * mean_x
    fitted = [intercept + slope * x for x in xs]
    residuals = [ys[i] - fitted[i] for i in range(n)]
    std = math.sqrt(sum(r * r for r in residuals) / max(1, n - 2)) if n > 2 else 0.0
    ss_tot = sum((y - mean_y) ** 2 for y in ys) or 1e-9
    ss_res = sum(r * r for r in residuals)
    r2 = max(0.0, min(1.0, 1.0 - ss_res / ss_tot))

    horizon_step = sample_interval_seconds
    predictions: list[float] = []
    upper: list[float] = []
    lower: list[float] = []
    for i in range(1, horizon_minutes + 1):
        t = xs[-1] + i * 60.0
        pred = intercept + slope * t
        predictions.append(pred)
        upper.append(pred + 1.96 * std)
        lower.append(pred - 1.96 * std)

    return Forecast(
        metric="?",
        horizon_minutes=horizon_minutes,
        predicted=predictions,
        current=ys[-1],
        trend_per_minute=slope * 60.0,
        confidence=r2,
        upper_bound=max(upper),
        lower_bound=min(lower),
    )


def predict_metric(
    metric: str,
    samples: Sequence[float],
    horizon_minutes: int = 30,
    sample_interval_seconds: float = 60.0,
) -> Forecast:
    """Public wrapper for :func:`linear_forecast` that names the metric."""
    f = linear_forecast(
        list(samples),
        horizon_minutes,
        sample_interval_seconds=sample_interval_seconds,
    )
    f.metric = metric
    return f


# ---------------------------------------------------------------------------
# Auto-scaling
# ---------------------------------------------------------------------------

@dataclass
class ScalingPolicy:
    """Per-service scaling thresholds."""

    service: str
    metric: str
    target_utilization: float = 0.7
    min_replicas: int = 1
    max_replicas: int = 20
    safety_margin: float = 1.25  # headroom multiplier

    @classmethod
    def default(cls, service: str, metric: str = "cpu") -> "ScalingPolicy":
        return cls(service=service, metric=metric)


def recommend_scaling(
    service: str,
    current_replicas: int,
    forecast: Forecast,
    *,
    policy: ScalingPolicy | None = None,
) -> Recommendation:
    """Convert a forecast into a concrete replica recommendation."""
    if policy is None:
        policy = ScalingPolicy.default(service)

    predicted_peak = max(forecast.predicted) if forecast.predicted else forecast.current
    current = forecast.current
    target = policy.target_utilization

    # Headroom adjusted peak
    peak = predicted_peak * policy.safety_margin
    ratio = peak / target if target > 0 else 1.0
    recommended = max(policy.min_replicas, math.ceil(current_replicas * ratio))

    if recommended > policy.max_replicas:
        recommended = policy.max_replicas
        action = "add_capacity"
        rationale = (
            f"Predicted peak {peak:.2f} requires {math.ceil(current_replicas * ratio)} "
            f"replicas but max_replicas={policy.max_replicas}; provision extra capacity."
        )
        severity = "critical"
    elif recommended > current_replicas:
        action = "scale_up"
        delta = recommended - current_replicas
        rationale = (
            f"Predicted peak {peak:.2f} exceeds target {target:.2f}; scale up by {delta}."
        )
        severity = "high" if delta >= 3 else "medium"
    elif recommended < current_replicas and forecast.trend_per_minute < -0.01:
        action = "scale_down"
        rationale = (
            f"Forecasted trend {forecast.trend_per_minute:+.4f}/min suggests over-provisioning; "
            f"safe to scale down to {recommended}."
        )
        severity = "low"
    else:
        action = "hold"
        rationale = (
            f"Predicted peak {peak:.2f} within target {target:.2f}; no change needed."
        )
        severity = "low"
        recommended = current_replicas

    return Recommendation(
        service=service,
        metric=policy.metric,
        action=action,
        current_replicas=current_replicas,
        recommended_replicas=recommended,
        rationale=rationale,
        severity=severity,
        forecast_horizon_minutes=forecast.horizon_minutes,
        predicted_peak=predicted_peak,
    )


# ---------------------------------------------------------------------------
# Streaming registry
# ---------------------------------------------------------------------------

class ForecastRegistry:
    """Per-metric sliding windows feeding :func:`predict_metric`."""

    def __init__(self, capacity: int = 120, sample_interval_seconds: float = 60.0) -> None:
        self.capacity = capacity
        self.sample_interval_seconds = sample_interval_seconds
        self._windows: dict[str, Deque[float]] = {}

    def _window_for(self, metric: str) -> Deque[float]:
        if metric not in self._windows:
            self._windows[metric] = deque(maxlen=self.capacity)
        return self._windows[metric]

    def observe(self, metric: str, value: float) -> None:
        self._window_for(metric).append(value)

    def forecast(self, metric: str, horizon_minutes: int = 30) -> Forecast:
        samples = list(self._window_for(metric))
        return predict_metric(
            metric,
            samples,
            horizon_minutes=horizon_minutes,
            sample_interval_seconds=self.sample_interval_seconds,
        )

    def forecast_all(self, horizon_minutes: int = 30) -> dict[str, Forecast]:
        return {m: self.forecast(m, horizon_minutes) for m in list(self._windows.keys())}


def capacity_plan(
    service: str,
    service_metrics: dict[str, Sequence[float]],
    *,
    current_replicas: int,
    horizon_minutes: int = 30,
    policies: dict[str, ScalingPolicy] | None = None,
) -> list[Recommendation]:
    """Compute scaling recommendations for a bundle of metrics of a service.

    ``service_metrics`` maps metric-name → recent samples.
    """
    policies = policies or {}
    out: list[Recommendation] = []
    for metric, samples in service_metrics.items():
        if metric.startswith("__"):
            continue
        f = predict_metric(metric, samples, horizon_minutes=horizon_minutes)
        policy = policies.get(metric) or ScalingPolicy.default(service=service, metric=metric)
        rec = recommend_scaling(
            service=service,
            current_replicas=current_replicas,
            forecast=f,
            policy=policy,
        )
        out.append(rec)
    return out


def summarise_recommendations(recs: Iterable[Recommendation]) -> dict[str, Any]:
    """Aggregate recommendations into a small summary dict."""
    items = list(recs)
    by_action: dict[str, int] = {}
    by_severity: dict[str, int] = {}
    for r in items:
        by_action[r.action] = by_action.get(r.action, 0) + 1
        by_severity[r.severity] = by_severity.get(r.severity, 0) + 1
    return {
        "count": len(items),
        "by_action": by_action,
        "by_severity": by_severity,
    }


__all__ = [
    "Forecast",
    "Recommendation",
    "ScalingPolicy",
    "ForecastRegistry",
    "linear_forecast",
    "predict_metric",
    "recommend_scaling",
    "capacity_plan",
    "summarise_recommendations",
]
