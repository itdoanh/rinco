"""Tests for ai-sre anomaly detector and performance predictor."""
from __future__ import annotations

import math
import random

import pytest

from app.services import anomaly_detector, performance_predictor


# ---------------------------------------------------------------------------
# Anomaly detector
# ---------------------------------------------------------------------------
def test_zscore_no_anomaly_within_band():
    samples = [10.0, 10.2, 9.8, 10.1, 10.0, 10.3]
    out = anomaly_detector.zscore_detect("cpu", samples, threshold=3.0)
    assert out == []


def test_zscore_detects_outlier_upward():
    samples = [10.0 + i * 0.01 for i in range(30)] + [95.0]
    out = anomaly_detector.zscore_detect("cpu", samples, threshold=2.5)
    assert len(out) == 1
    a = out[0]
    assert a.metric == "cpu"
    assert a.value == 95.0
    assert a.detector == "zscore"
    assert a.deviation > 2.5


def test_zscore_detects_outlier_downward():
    samples = [50.0 + i * 0.1 for i in range(25)] + [-100.0]
    out = anomaly_detector.zscore_detect("metric", samples, threshold=2.0)
    assert len(out) == 1
    assert out[0].deviation < -2.0


def test_iqr_no_anomaly_when_within_fence():
    samples = [1, 2, 3, 4, 5, 6]
    out = anomaly_detector.iqr_detect("m", samples, k=1.5)
    assert out == []


def test_iqr_detects_far_outlier():
    # Slightly varied history so Q1/Q3/IQR are non-zero, then a huge outlier.
    history = [10.0, 11.0, 9.5, 10.5, 9.8, 10.2, 10.1]
    samples = history + [1000.0]
    out = anomaly_detector.iqr_detect("m", samples, k=1.5)
    assert len(out) == 1
    assert out[0].detector == "iqr"


def test_ewma_detects_regime_change():
    samples = [10.0] * 12 + [50.0]
    out = anomaly_detector.ewma_detect("m", samples, alpha=0.3, threshold=2.5)
    assert len(out) == 1
    assert out[0].detector == "ewma"


def test_severity_buckets_are_monotonic():
    # ratio = |dev| / threshold
    # ratio < 1.5 -> low, 1.5..2 -> medium, 2..4 -> high, >=4 -> critical
    assert anomaly_detector._severity_for(0.5, 1.0) == "low"
    assert anomaly_detector._severity_for(1.6, 1.0) == "medium"
    assert anomaly_detector._severity_for(2.5, 1.0) == "high"
    assert anomaly_detector._severity_for(4.5, 1.0) == "critical"


def test_streaming_detector_observe_buffers_samples():
    d = anomaly_detector.StreamingAnomalyDetector(window_size=10)
    for i in range(8):
        d.observe("metric", 5.0 + 0.01 * i)
    assert len(d._windows["metric"]) == 8
    out = d.observe("metric", 5.05)
    assert isinstance(out, list)


def test_streaming_detector_detect_all_lists_metrics():
    d = anomaly_detector.StreamingAnomalyDetector()
    for v in [1, 2, 3, 4, 5, 1000]:
        d.observe("cpu", v)
    for v in [10, 11, 12, 13, 14, 15]:
        d.observe("mem", v)
    snap = d.detect_all()
    assert set(snap.keys()) == {"cpu", "mem"}
    assert any(a.detector for a in snap["cpu"])


def test_summarise_buckets_by_severity():
    items = [
        anomaly_detector.Anomaly("a", 0, 1, 1, 1, "high", "zscore", "..."),
        anomaly_detector.Anomaly("b", 0, 2, 2, 2, "high", "zscore", "..."),
        anomaly_detector.Anomaly("c", 0, 3, 3, 3, "low", "iqr", "..."),
    ]
    s = anomaly_detector.summarise(items)
    assert s["total"] == 3
    assert s["by_severity"]["high"] == 2
    assert s["by_metric"]["a"] == 1


def test_percentile_basic():
    assert anomaly_detector._percentile([1, 2, 3, 4, 5], 0.5) == 3
    assert anomaly_detector._percentile([1, 2, 3, 4, 5], 0.0) == 1
    assert anomaly_detector._percentile([1, 2, 3, 4, 5], 1.0) == 5


def test_to_dict_round_trip():
    a = anomaly_detector.Anomaly("m", 0, 1, 1, 1, "low", "zscore", "x")
    d = a.to_dict()
    assert d["metric"] == "m"
    assert d["detector"] == "zscore"


# ---------------------------------------------------------------------------
# Performance predictor
# ---------------------------------------------------------------------------
def test_linear_forecast_flat_history():
    samples = [10.0] * 30
    f = performance_predictor.linear_forecast(samples, horizon_minutes=10)
    assert len(f.predicted) == 10
    for v in f.predicted:
        assert math.isclose(v, 10.0, abs_tol=1e-6)
    assert math.isclose(f.trend_per_minute, 0.0, abs_tol=1e-9)


def test_linear_forecast_upward_trend():
    samples = [i * 1.0 for i in range(30)]
    f = performance_predictor.linear_forecast(samples, horizon_minutes=5)
    assert f.trend_per_minute > 0.5  # ~60s per minute
    assert f.predicted[-1] > f.current


def test_linear_forecast_short_series_returns_current():
    samples = [42.0]
    f = performance_predictor.linear_forecast(samples, horizon_minutes=5)
    assert all(v == 42.0 for v in f.predicted)
    assert f.confidence == 0.0


def test_predict_metric_sets_metric_name():
    f = performance_predictor.predict_metric("rpc_latency", [1, 2, 3, 4, 5])
    assert f.metric == "rpc_latency"


def test_scaling_policy_default():
    p = performance_predictor.ScalingPolicy.default("crm-core", metric="cpu")
    assert p.service == "crm-core"
    assert p.metric == "cpu"
    assert p.min_replicas == 1
    assert p.max_replicas == 20


def test_recommend_scaling_holds_within_target():
    forecast = performance_predictor.predict_metric(
        "cpu", [0.5] * 20, horizon_minutes=10,
    )
    rec = performance_predictor.recommend_scaling(
        service="svc", current_replicas=3, forecast=forecast,
        policy=performance_predictor.ScalingPolicy(service="svc", metric="cpu", target_utilization=0.8),
    )
    assert rec.action == "hold"
    assert rec.recommended_replicas == 3


def test_recommend_scaling_scale_up_when_predicted_peak_high():
    forecast = performance_predictor.predict_metric(
        "cpu", [0.5] * 20, horizon_minutes=10,
    )
    forecast.predicted = [1.5] * 10
    forecast.upper_bound = 1.6
    rec = performance_predictor.recommend_scaling(
        service="svc", current_replicas=2, forecast=forecast,
        policy=performance_predictor.ScalingPolicy(service="svc", metric="cpu", target_utilization=0.7, safety_margin=1.0),
    )
    assert rec.action == "scale_up"
    assert rec.recommended_replicas >= 3


def test_recommend_scaling_add_capacity_when_clamped():
    forecast = performance_predictor.predict_metric("cpu", [1.0] * 20)
    forecast.predicted = [5.0] * 30
    rec = performance_predictor.recommend_scaling(
        service="svc", current_replicas=1, forecast=forecast,
        policy=performance_predictor.ScalingPolicy(service="svc", metric="cpu", max_replicas=2),
    )
    assert rec.action == "add_capacity"
    assert rec.recommended_replicas == 2
    assert rec.severity == "critical"


def test_recommend_scaling_scale_down_when_declining():
    # Steadily declining history; peak much lower than current capacity.
    samples = [10.0 - i * 0.2 for i in range(60)]  # 10 -> ~-2
    forecast = performance_predictor.predict_metric("cpu", samples, horizon_minutes=10)
    # Peak will be very small relative to a large replica count.
    rec = performance_predictor.recommend_scaling(
        service="svc", current_replicas=20, forecast=forecast,
        policy=performance_predictor.ScalingPolicy(service="svc", metric="cpu", target_utilization=1.0, safety_margin=1.0),
    )
    assert rec.action == "scale_down"
    assert rec.recommended_replicas < 20


def test_forecast_registry_observe_and_forecast():
    reg = performance_predictor.ForecastRegistry(capacity=30, sample_interval_seconds=60.0)
    for i in range(30):
        reg.observe("rpc_latency", float(i))
    f = reg.forecast("rpc_latency", horizon_minutes=5)
    assert f.trend_per_minute > 0


def test_forecast_registry_forecast_all():
    reg = performance_predictor.ForecastRegistry(capacity=10)
    for i in range(10):
        reg.observe("a", 1.0 + i)
        reg.observe("b", 5.0 - i * 0.1)
    snap = reg.forecast_all(horizon_minutes=3)
    assert set(snap.keys()) == {"a", "b"}


def test_capacity_plan_returns_recommendation_per_metric():
    samples = {"cpu": [0.4] * 20, "mem": [0.6] * 20}
    recs = performance_predictor.capacity_plan(
        "svc", samples, current_replicas=2, horizon_minutes=15,
    )
    assert len(recs) == 2
    for r in recs:
        assert r.current_replicas == 2


def test_summarise_recommendations():
    items = [
        performance_predictor.Recommendation(
            service="svc", metric="cpu", action="hold",
            current_replicas=1, recommended_replicas=1,
            rationale="x", severity="low", forecast_horizon_minutes=10, predicted_peak=0.5,
        ),
        performance_predictor.Recommendation(
            service="svc", metric="mem", action="scale_up",
            current_replicas=1, recommended_replicas=3,
            rationale="y", severity="high", forecast_horizon_minutes=10, predicted_peak=0.9,
        ),
    ]
    summary = performance_predictor.summarise_recommendations(items)
    assert summary["count"] == 2
    assert summary["by_action"]["hold"] == 1
    assert summary["by_action"]["scale_up"] == 1


def test_forecast_to_dict_round_trip():
    f = performance_predictor.predict_metric("x", [1, 2, 3, 4, 5])
    d = f.to_dict()
    assert d["metric"] == "x"
    assert len(d["predicted"]) == 30


def test_recommendation_to_dict_round_trip():
    r = performance_predictor.Recommendation(
        service="svc", metric="cpu", action="hold",
        current_replicas=2, recommended_replicas=2,
        rationale="", severity="low", forecast_horizon_minutes=10, predicted_peak=0.5,
    )
    d = r.to_dict()
    assert d["service"] == "svc"
    assert d["action"] == "hold"
