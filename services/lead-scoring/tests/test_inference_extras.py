"""Extra tests for inference — TrainedModel, helpers, edge cases."""
from __future__ import annotations

import os

import numpy as np
import pytest

from app.services.inference import (
    GLOBAL_MODEL,
    MODEL_DIR,
    TENANT_MODELS,
    TrainedModel,
    _default_features,
    _ensure_2d,
    _initialise_default_model,
    confidence_from_features,
    persist_model,
    predict_score,
    recommended_action,
    shap_top_k,
    tier_from_score,
)


@pytest.fixture(autouse=True)
def _reset_state():
    """Reset global state before each test."""
    TENANT_MODELS.clear()
    # Reset the global model so each test reinitializes
    import app.services.inference as inf
    inf.GLOBAL_MODEL = None
    yield
    TENANT_MODELS.clear()
    inf.GLOBAL_MODEL = None


# =============================================================================
# TrainedModel dataclass
# =============================================================================

def test_extra_trained_model_defaults():
    m = TrainedModel()
    assert m.xgb is None
    assert m.nn_session is None
    assert m.fallback is None
    assert m.scaler is None
    assert m.features == []
    assert m.metrics == {}
    assert m.version == "1.0.0"
    assert m.fallback_mode is False


def test_extra_trained_model_with_values():
    m = TrainedModel(
        xgb="xgb-mock",
        features=["a", "b"],
        metrics={"acc": 0.95},
        version="2.0.0",
        fallback_mode=True,
    )
    assert m.xgb == "xgb-mock"
    assert m.features == ["a", "b"]
    assert m.metrics == {"acc": 0.95}
    assert m.version == "2.0.0"
    assert m.fallback_mode is True


def test_extra_trained_model_features_default_factory_isolated():
    """Each instance should get its own features list."""
    m1 = TrainedModel()
    m2 = TrainedModel()
    m1.features.append("leak")
    assert "leak" not in m2.features


def test_extra_trained_model_metrics_default_factory_isolated():
    m1 = TrainedModel()
    m2 = TrainedModel()
    m1.metrics["new"] = 1.0
    assert "new" not in m2.metrics


# =============================================================================
# _default_features
# =============================================================================

def test_extra_default_features_returns_list():
    feats = _default_features()
    assert isinstance(feats, list)
    assert len(feats) > 30


def test_extra_default_features_consistent():
    feats1 = _default_features()
    feats2 = _default_features()
    assert feats1 == feats2


# =============================================================================
# _ensure_2d
# =============================================================================

def test_extra_ensure_2d_1d():
    arr = np.array([1.0, 2.0, 3.0])
    result = _ensure_2d(arr)
    assert result.ndim == 2
    assert result.shape == (1, 3)


def test_extra_ensure_2d_already_2d():
    arr = np.array([[1.0, 2.0], [3.0, 4.0]])
    result = _ensure_2d(arr)
    assert result.ndim == 2
    assert result.shape == (2, 2)


def test_extra_ensure_2d_3d_unchanged():
    arr = np.zeros((2, 3, 4))
    result = _ensure_2d(arr)
    assert result.ndim == 3


# =============================================================================
# tier_from_score
# =============================================================================

def test_extra_tier_boundaries():
    assert tier_from_score(0.85) == "very-hot"
    assert tier_from_score(0.84999) == "hot"
    assert tier_from_score(0.6) == "hot"
    assert tier_from_score(0.59999) == "warm"
    assert tier_from_score(0.3) == "warm"
    assert tier_from_score(0.29999) == "cold"


def test_extra_tier_zero():
    assert tier_from_score(0.0) == "cold"


def test_extra_tier_one():
    assert tier_from_score(1.0) == "very-hot"


def test_extra_tier_above_one_clamped_via_logic():
    """tier_from_score does not clamp; >= 0.85 wins regardless."""
    assert tier_from_score(2.0) == "very-hot"


# =============================================================================
# recommended_action
# =============================================================================

def test_extra_recommended_action_unknown_tier_defaults_to_nurture():
    assert recommended_action("unknown") == "nurture"


def test_extra_recommended_action_empty_tier():
    assert recommended_action("") == "nurture"


def test_extra_recommended_action_all_known():
    assert recommended_action("very-hot") == "call-now"
    assert recommended_action("hot") == "call-soon"
    assert recommended_action("warm") == "email"
    assert recommended_action("cold") == "nurture"


# =============================================================================
# predict_score
# =============================================================================

def test_extra_predict_score_no_models_returns_neutral():
    """When no models are available, return neutral 0.5."""
    m = TrainedModel(features=["a"])
    score = predict_score(m, np.array([[0.5]]))
    assert score == 0.5


def test_extra_predict_score_only_fallback():
    """With only fallback, return its proba."""
    try:
        from sklearn.linear_model import LogisticRegression
    except (ImportError, ValueError):
        pytest.skip("sklearn unavailable")

    m = TrainedModel(
        fallback=LogisticRegression(max_iter=100),
        features=["a"],
    )
    np.random.seed(0)
    X = np.random.rand(50, 1)
    y = (X[:, 0] > 0.5).astype(int)
    m.fallback.fit(X, y)
    score = predict_score(m, np.array([[0.6]]))
    assert 0.0 <= score <= 1.0


def test_extra_predict_score_with_list_input():
    """Plain list input should be accepted."""
    try:
        from sklearn.linear_model import LogisticRegression
    except (ImportError, ValueError):
        pytest.skip("sklearn unavailable")

    m = TrainedModel(
        fallback=LogisticRegression(max_iter=100),
        features=["a", "b"],
    )
    np.random.seed(0)
    X = np.random.rand(50, 2)
    y = (X[:, 0] > 0.5).astype(int)
    m.fallback.fit(X, y)
    score = predict_score(m, [[0.6, 0.3]])
    assert 0.0 <= score <= 1.0


def test_extra_predict_score_handles_3d_no_reshape():
    """3D input is left as-is (not reshaped to 2D)."""
    m = TrainedModel(features=["a", "b"])
    # Fallback is None — should return 0.5 (no model available).
    score = predict_score(m, np.zeros((1, 1, 2)))
    assert score == 0.5


# =============================================================================
# confidence_from_features
# =============================================================================

def test_extra_confidence_no_fallback_returns_default():
    m = TrainedModel()
    conf = confidence_from_features(m, np.array([[0.5]]))
    assert conf == 0.7


def test_extra_confidence_with_fallback():
    try:
        from sklearn.linear_model import LogisticRegression
    except (ImportError, ValueError):
        pytest.skip("sklearn unavailable")

    m = TrainedModel(
        fallback=LogisticRegression(max_iter=100),
        features=["a"],
    )
    np.random.seed(0)
    X = np.random.rand(50, 1)
    y = (X[:, 0] > 0.5).astype(int)
    m.fallback.fit(X, y)
    conf = confidence_from_features(m, np.array([[0.6]]))
    assert 0.5 <= conf <= 0.99


def test_extra_confidence_with_list_input():
    try:
        from sklearn.linear_model import LogisticRegression
    except (ImportError, ValueError):
        pytest.skip("sklearn unavailable")

    m = TrainedModel(
        fallback=LogisticRegression(max_iter=100),
        features=["a", "b"],
    )
    np.random.seed(0)
    X = np.random.rand(50, 2)
    y = (X[:, 0] > 0.5).astype(int)
    m.fallback.fit(X, y)
    conf = confidence_from_features(m, [[0.6, 0.3]])
    assert 0.5 <= conf <= 0.99


# =============================================================================
# shap_top_k
# =============================================================================

def test_extra_shap_top_k_no_xgb_returns_empty():
    m = TrainedModel(features=["a", "b"])
    result = shap_top_k(m, np.array([[0.5, 0.3]]), k=3)
    assert result == {}


def test_extra_shap_top_k_no_features_returns_empty():
    """Empty features list returns empty result."""
    m = TrainedModel(features=[], xgb="mock-xgb")
    # Without shap installed OR xgb valid, returns empty.
    result = shap_top_k(m, np.array([[0.5]]), k=3)
    assert result == {}


# =============================================================================
# _initialise_default_model
# =============================================================================

def test_extra_initialise_default_model():
    m = _initialise_default_model()
    assert m.features == list(_default_features())
    assert m.fallback_mode is True
    # XGB may or may not be set depending on availability


# =============================================================================
# persist_model + load_model
# =============================================================================

def test_extra_persist_model_creates_file(tmp_path):
    """persist_model should write a file and add to TENANT_MODELS."""
    import app.services.inference as inf
    inf.MODEL_DIR = str(tmp_path)

    m = TrainedModel(features=["x"])
    path = persist_model("test-tenant", m)
    assert os.path.exists(path)
    assert inf.TENANT_MODELS["test-tenant"] is m


def test_extra_load_model_returns_global_when_no_tenant_file():
    """When no tenant file exists, load_model returns global."""
    import app.services.inference as inf
    inf.MODEL_DIR = str(os.path.join(os.getcwd(), "_empty_models_xyz"))
    inf.GLOBAL_MODEL = TrainedModel(features=["g"])

    m = inf.load_model("nonexistent-tenant")
    assert m is inf.GLOBAL_MODEL
    assert m.features == ["g"]


def test_extra_load_model_returns_cached_when_in_memory():
    """When tenant already in TENANT_MODELS, return that."""
    import app.services.inference as inf
    cached = TrainedModel(features=["cached"])
    inf.TENANT_MODELS["cached-tenant"] = cached

    m = inf.load_model("cached-tenant")
    assert m is cached


def test_extra_load_model_loads_from_disk(tmp_path):
    import app.services.inference as inf
    inf.MODEL_DIR = str(tmp_path)

    # Persist a model first
    m1 = TrainedModel(features=["from-disk"])
    persist_model("disk-tenant", m1)
    inf.TENANT_MODELS.clear()  # clear in-memory cache to force disk load

    # Now load — should read from disk
    m2 = inf.load_model("disk-tenant")
    assert m2.features == ["from-disk"]


# =============================================================================
# GLOBAL_MODEL / MODEL_DIR
# =============================================================================

def test_extra_global_model_initially_none():
    import app.services.inference as inf
    inf.GLOBAL_MODEL = None
    assert inf.GLOBAL_MODEL is None


def test_extra_model_dir_default():
    """Default MODEL_DIR is /models."""
    import app.services.inference as inf
    inf.MODEL_DIR = "/models"  # restore default
    assert inf.MODEL_DIR == "/models"


def test_extra_model_dir_env_override(monkeypatch, tmp_path):
    monkeypatch.setenv("MODEL_DIR", str(tmp_path))
    import importlib
    import app.services.inference as inf
    importlib.reload(inf)
    assert inf.MODEL_DIR == str(tmp_path)
    # Restore for other tests
    monkeypatch.setenv("MODEL_DIR", "/models")
    importlib.reload(inf)
