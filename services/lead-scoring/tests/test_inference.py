"""Tests for lead-scoring inference helpers."""
from __future__ import annotations

import numpy as np
import pytest

from app.services.inference import (
    TrainedModel,
    predict_score,
    tier_from_score,
    recommended_action,
    _ensure_2d,
)


def test_extra_tier_very_hot():
    """Score >= 0.85 should be very-hot."""
    assert tier_from_score(0.85) == "very-hot"
    assert tier_from_score(0.95) == "very-hot"
    assert tier_from_score(1.0) == "very-hot"


def test_extra_tier_hot():
    """Score >= 0.6 and < 0.85 should be hot."""
    assert tier_from_score(0.6) == "hot"
    assert tier_from_score(0.7) == "hot"
    assert tier_from_score(0.84) == "hot"


def test_extra_tier_warm():
    """Score >= 0.3 and < 0.6 should be warm."""
    assert tier_from_score(0.3) == "warm"
    assert tier_from_score(0.4) == "warm"
    assert tier_from_score(0.59) == "warm"


def test_extra_tier_cold():
    """Score < 0.3 should be cold."""
    assert tier_from_score(0.0) == "cold"
    assert tier_from_score(0.1) == "cold"
    assert tier_from_score(0.29) == "cold"


def test_extra_tier_negative_treated_as_cold():
    """Negative scores should be cold."""
    assert tier_from_score(-0.5) == "cold"


def test_extra_recommended_action_very_hot():
    """very-hot tier should recommend call-now."""
    assert recommended_action("very-hot") == "call-now"


def test_extra_recommended_action_hot():
    """hot tier should recommend call-soon."""
    assert recommended_action("hot") == "call-soon"


def test_extra_recommended_action_warm():
    """warm tier should recommend email."""
    assert recommended_action("warm") == "email"


def test_extra_recommended_action_cold():
    """cold tier should recommend nurture."""
    assert recommended_action("cold") == "nurture"


def test_extra_recommended_action_unknown():
    """Unknown tier defaults to nurture."""
    assert recommended_action("unknown") == "nurture"
    assert recommended_action("") == "nurture"


def test_extra_ensure_2d_1d_to_2d():
    """1D array should be reshaped to 2D."""
    arr = np.array([1, 2, 3])
    result = _ensure_2d(arr)
    assert result.ndim == 2
    assert result.shape == (1, 3)


def test_extra_ensure_2d_already_2d():
    """2D array should be unchanged."""
    arr = np.array([[1, 2, 3], [4, 5, 6]])
    result = _ensure_2d(arr)
    assert result.ndim == 2
    assert result.shape == (2, 3)


def test_extra_ensure_2d_3d():
    """3D array should be unchanged (not 1D)."""
    arr = np.zeros((2, 3, 4))
    result = _ensure_2d(arr)
    assert result.ndim == 3
    assert result.shape == (2, 3, 4)


def test_extra_trained_model_defaults():
    """TrainedModel should have sensible defaults."""
    m = TrainedModel()
    assert m.xgb is None
    assert m.nn_session is None
    assert m.fallback is None
    assert m.scaler is None
    assert m.features == []
    assert m.metrics == {}
    assert m.version == "1.0.0"
    assert m.fallback_mode is False


def test_extra_trained_model_trained_at_set():
    """TrainedModel should have a default trained_at timestamp."""
    m = TrainedModel()
    assert m.trained_at is not None


def test_extra_predict_score_no_models():
    """predict_score with no models returns neutral 0.5."""
    m = TrainedModel()
    X = np.array([[1, 2, 3]])
    score = predict_score(m, X)
    assert 0.0 <= score <= 1.0
    # With no models, should be 0.5
    assert abs(score - 0.5) < 0.001


def test_extra_predict_score_with_fallback():
    """predict_score with fallback sklearn uses it."""
    try:
        from sklearn.linear_model import LogisticRegression
    except (ImportError, ValueError):
        pytest.skip("sklearn unavailable")
    m = TrainedModel(
        fallback=LogisticRegression(max_iter=100),
        features=["a", "b", "c"],
    )
    np.random.seed(0)
    X = np.random.rand(100, 3)
    y = (X[:, 0] > 0.5).astype(int)
    m.fallback.fit(X, y)

    test_X = np.array([[0.6, 0.3, 0.2]])
    score = predict_score(m, test_X)
    assert 0.0 <= score <= 1.0


def test_extra_predict_score_handles_1d():
    """predict_score should handle 1D input."""
    try:
        from sklearn.linear_model import LogisticRegression
    except (ImportError, ValueError):
        pytest.skip("sklearn unavailable")
    m = TrainedModel(
        fallback=LogisticRegression(max_iter=100),
        features=["a", "b", "c"],
    )
    np.random.seed(0)
    X = np.random.rand(100, 3)
    y = (X[:, 0] > 0.5).astype(int)
    m.fallback.fit(X, y)

    test_X = np.array([0.6, 0.3, 0.2])  # 1D
    score = predict_score(m, test_X)
    assert 0.0 <= score <= 1.0


def test_extra_predict_score_with_features_attr():
    """predict_score should handle objects with to_numpy."""
    class FakeDF:
        def to_numpy(self, dtype=None):
            return np.array([[0.6, 0.3, 0.2]])

    m = TrainedModel(features=["a", "b", "c"])
    score = predict_score(m, FakeDF())
    assert 0.0 <= score <= 1.0


def test_extra_predict_score_bounded():
    """Score should be bounded between 0 and 1."""
    try:
        from sklearn.linear_model import LogisticRegression
    except (ImportError, ValueError):
        pytest.skip("sklearn unavailable")
    m = TrainedModel(
        fallback=LogisticRegression(max_iter=100),
    )
    np.random.seed(0)
    X = np.random.rand(100, 3)
    y = (X[:, 0] > 0.5).astype(int)
    m.fallback.fit(X, y)

    for _ in range(10):
        test_X = np.random.rand(1, 3)
        score = predict_score(m, test_X)
        assert 0.0 <= score <= 1.0
