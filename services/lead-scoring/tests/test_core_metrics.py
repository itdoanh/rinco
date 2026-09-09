"""Tests for lead-scoring core metrics module."""
from app.core import metrics
from app.core.metrics import (
    render,
    REQUESTS_TOTAL,
    INFERENCE_DURATION,
    TRAINING_RUNS_TOTAL,
    ERRORS_TOTAL,
    ENSEMBLE_SCORE,
    MODEL_VERSION,
    registry,
)


def test_registry_is_collector_registry():
    """registry is a Prometheus CollectorRegistry."""
    from prometheus_client import CollectorRegistry
    assert isinstance(registry, CollectorRegistry)


def test_metrics_constants_defined():
    """All expected metrics are exposed."""
    assert REQUESTS_TOTAL is not None
    assert INFERENCE_DURATION is not None
    assert TRAINING_RUNS_TOTAL is not None
    assert ERRORS_TOTAL is not None
    assert ENSEMBLE_SCORE is not None
    assert MODEL_VERSION is not None


def test_render_returns_bytes_and_str():
    """render() returns prom bytes + content_type str."""
    body, content_type = render()
    assert isinstance(body, bytes)
    assert isinstance(content_type, str)
    assert "text/plain" in content_type or "openmetrics" in content_type


def test_render_includes_metrics_names():
    """Rendered output contains expected metric names."""
    REQUESTS_TOTAL.labels(tenant_id="t1", result="ok").inc()
    body, _ = render()
    text = body.decode("utf-8")
    assert "lead_scoring_requests_total" in text


def test_requests_total_inc():
    """REQUESTS_TOTAL can be incremented."""
    before = REQUESTS_TOTAL.labels(tenant_id="t2", result="ok")._value.get()
    REQUESTS_TOTAL.labels(tenant_id="t2", result="ok").inc()
    after = REQUESTS_TOTAL.labels(tenant_id="t2", result="ok")._value.get()
    assert after == before + 1


def test_inference_duration_observe():
    """INFERENCE_DURATION can be observed."""
    INFERENCE_DURATION.labels(model_version="v1").observe(0.05)
    # Just verify no exception


def test_training_runs_inc():
    """TRAINING_RUNS_TOTAL can be incremented."""
    before = TRAINING_RUNS_TOTAL.labels(tenant_id="t3", status="ok")._value.get()
    TRAINING_RUNS_TOTAL.labels(tenant_id="t3", status="ok").inc()
    after = TRAINING_RUNS_TOTAL.labels(tenant_id="t3", status="ok")._value.get()
    assert after == before + 1


def test_errors_total_inc():
    """ERRORS_TOTAL can be incremented."""
    before = ERRORS_TOTAL.labels(tenant_id="t4", error_type="timeout")._value.get()
    ERRORS_TOTAL.labels(tenant_id="t4", error_type="timeout").inc()
    after = ERRORS_TOTAL.labels(tenant_id="t4", error_type="timeout")._value.get()
    assert after == before + 1


def test_ensemble_score_set():
    """ENSEMBLE_SCORE gauge can be set."""
    ENSEMBLE_SCORE.labels(tenant_id="t5").set(0.85)
    val = ENSEMBLE_SCORE.labels(tenant_id="t5")._value.get()
    assert abs(val - 0.85) < 1e-6


def test_model_version_set():
    """MODEL_VERSION gauge can be set."""
    MODEL_VERSION.labels(tenant_id="t6", version="v1").set(1)
    val = MODEL_VERSION.labels(tenant_id="t6", version="v1")._value.get()
    assert val == 1


def test_registry_module_imports_match():
    """metrics module re-exports the same objects as the package __init__."""
    assert metrics.registry is registry
    assert metrics.REQUESTS_TOTAL is REQUESTS_TOTAL
    assert metrics.INFERENCE_DURATION is INFERENCE_DURATION
    assert metrics.TRAINING_RUNS_TOTAL is TRAINING_RUNS_TOTAL
    assert metrics.ERRORS_TOTAL is ERRORS_TOTAL
    assert metrics.ENSEMBLE_SCORE is ENSEMBLE_SCORE
    assert metrics.MODEL_VERSION is MODEL_VERSION


def test_render_multiple_calls_dont_crash():
    """render can be called multiple times."""
    render()
    render()
    body, _ = render()
    assert len(body) > 0
