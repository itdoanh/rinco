"""Tests for lead-scoring schemas."""
from __future__ import annotations

from app.schemas.score import (
    ScoreResponse,
    TrainRequest,
    TrainResponse,
    ModelInfo,
    HealthResponse,
    ExplainResponse,
)


def test_score_response_basic():
    """ScoreResponse validates basic input."""
    r = ScoreResponse(
        score=0.85,
        tier="high",
        recommended_action="call",
        confidence=0.92,
        model_version="v1",
        latency_ms=42,
    )
    assert r.score == 0.85
    assert r.tier == "high"
    assert r.latency_ms == 42
    assert r.feature_importances == {}
    assert r.shap_top == {}


def test_score_response_with_features():
    """ScoreResponse with feature importances."""
    r = ScoreResponse(
        score=0.5,
        tier="medium",
        recommended_action="nurture",
        confidence=0.6,
        model_version="v1",
        feature_importances={"page_views": 0.3, "has_email": 0.1},
        shap_top={"page_views": 0.15},
        explanation="High engagement",
        latency_ms=10,
    )
    assert r.feature_importances["page_views"] == 0.3
    assert r.explanation == "High engagement"


def test_train_request_minimal():
    """TrainRequest accepts empty input."""
    req = TrainRequest()
    assert req.tenant_id is None
    assert req.notes is None


def test_train_request_with_values():
    """TrainRequest with full input."""
    req = TrainRequest(tenant_id="t1", notes="drift detected")
    assert req.tenant_id == "t1"
    assert req.notes == "drift detected"


def test_train_response_basic():
    """TrainResponse validates input."""
    r = TrainResponse(
        status="ok",
        tenant_id="t1",
        model_version="v1",
        fallback_mode=False,
    )
    assert r.status == "ok"
    assert r.fallback_mode is False


def test_model_info_basic():
    """ModelInfo validates input."""
    m = ModelInfo(
        tenant_id="t1",
        model_version="v1",
        n_estimators=100,
        features=["a", "b"],
        trained_at="2026-01-01",
        fallback_mode=False,
    )
    assert m.n_estimators == 100
    assert len(m.features) == 2


def test_health_response_basic():
    """HealthResponse validates input."""
    h = HealthResponse(
        status="ok",
        service="lead-scoring",
        version="1.0.0",
    )
    assert h.status == "ok"
    assert h.fallback_mode is False
    assert h.models_loaded == []


def test_health_response_with_models():
    """HealthResponse with models loaded."""
    h = HealthResponse(
        status="ok",
        service="lead-scoring",
        version="1.0.0",
        models_loaded=["v1", "v2"],
        fallback_mode=True,
    )
    assert len(h.models_loaded) == 2
    assert h.fallback_mode is True


def test_explain_response_basic():
    """ExplainResponse validates input."""
    e = ExplainResponse(lead_id="l1", tenant_id="t1")
    assert e.lead_id == "l1"
    assert e.shap == {}
    assert e.features == []


def test_explain_response_with_shap():
    """ExplainResponse with SHAP values."""
    e = ExplainResponse(
        lead_id="l1",
        tenant_id="t1",
        shap={"page_views": 0.3, "email_opens": 0.2},
        features=["page_views", "email_opens"],
    )
    assert e.shap["page_views"] == 0.3
    assert len(e.features) == 2
