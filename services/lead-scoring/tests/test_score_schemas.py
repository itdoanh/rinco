"""Tests for lead-scoring schema models (Pydantic)."""
from app.schemas.score import (
    ScoreResponse, TrainRequest, TrainResponse,
    ModelInfo, HealthResponse, ExplainResponse,
)


def test_score_response_default_fields():
    """ScoreResponse with defaults."""
    r = ScoreResponse(
        score=0.85,
        tier="hot",
        recommended_action="call_now",
        confidence=0.9,
        model_version="v1.0",
        latency_ms=42,
    )
    assert r.score == 0.85
    assert r.tier == "hot"
    assert r.lead_id is None
    assert r.feature_importances == {}
    assert r.shap_top == {}
    assert r.explanation is None


def test_score_response_with_optional():
    """ScoreResponse with all optional fields."""
    r = ScoreResponse(
        lead_id="lead-123",
        score=0.7,
        tier="warm",
        recommended_action="email",
        confidence=0.8,
        model_version="v2.0",
        feature_importances={"feat_a": 0.5},
        shap_top={"a": 0.3},
        explanation="top feature is engagement",
        latency_ms=100,
    )
    assert r.lead_id == "lead-123"
    assert r.explanation.startswith("top feature")


def test_train_request_defaults():
    """TrainRequest with all optional."""
    r = TrainRequest()
    assert r.tenant_id is None
    assert r.notes is None


def test_train_request_with_values():
    r = TrainRequest(tenant_id="t1", notes="re-train")
    assert r.tenant_id == "t1"
    assert r.notes == "re-train"


def test_train_response_required():
    r = TrainResponse(
        status="ok",
        tenant_id="t1",
        model_version="v3.0",
        fallback_mode=False,
    )
    assert r.status == "ok"
    assert r.metrics == {}


def test_model_info_default_metrics():
    r = ModelInfo(
        tenant_id="t1",
        model_version="v1",
        n_estimators=100,
        features=["a", "b"],
        trained_at="2026-01-01",
        fallback_mode=False,
    )
    assert r.metrics == {}
    assert r.fallback_mode is False


def test_health_response_required():
    r = HealthResponse(
        status="ok",
        service="lead-scoring",
        version="1.0",
    )
    assert r.status == "ok"
    assert r.models_loaded == []
    assert r.fallback_mode is False


def test_explain_response_default():
    r = ExplainResponse(lead_id="L1", tenant_id="t1")
    assert r.shap == {}
    assert r.features == []


def test_score_response_serialization():
    """ScoreResponse can serialize and deserialize."""
    r = ScoreResponse(
        lead_id="L1",
        score=0.92,
        tier="hot",
        recommended_action="call",
        confidence=0.95,
        model_version="v1",
        latency_ms=10,
    )
    data = r.model_dump()
    assert data["score"] == 0.92

    r2 = ScoreResponse(**data)
    assert r2.lead_id == "L1"


def test_score_response_score_range():
    """Score should accept 0.0-1.0 floats (no constraint in schema)."""
    r = ScoreResponse(
        score=0.0,
        tier="cold",
        recommended_action="none",
        confidence=0.0,
        model_version="v1",
        latency_ms=0,
    )
    assert r.score == 0.0


def test_health_response_with_models():
    r = HealthResponse(
        status="ok",
        service="lead-scoring",
        version="1.0",
        models_loaded=["v1", "v2"],
        fallback_mode=True,
    )
    assert len(r.models_loaded) == 2
    assert r.fallback_mode is True


def test_model_info_with_metrics():
    r = ModelInfo(
        tenant_id="t1",
        model_version="v1",
        n_estimators=200,
        features=["a", "b", "c"],
        trained_at="2026-01-01",
        metrics={"auc": 0.85, "accuracy": 0.9},
        fallback_mode=True,
    )
    assert r.metrics["auc"] == 0.85
    assert r.fallback_mode is True
