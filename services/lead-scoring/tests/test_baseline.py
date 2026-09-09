"""Tests for baseline sklearn model."""
from __future__ import annotations

import os
import tempfile
from unittest.mock import patch

import numpy as np
import pytest

from app.models import baseline


def test_feature_names_count():
    """Feature names list has expected count."""
    assert len(baseline.FEATURE_NAMES) == 12


def test_feature_names_required():
    """All required features present."""
    required = ["has_phone", "has_email", "page_views", "email_opens"]
    for f in required:
        assert f in baseline.FEATURE_NAMES


def test_feature_names_unique():
    """No duplicate feature names."""
    assert len(baseline.FEATURE_NAMES) == len(set(baseline.FEATURE_NAMES))


def test_predict_no_model():
    """predict with None model returns zeros."""
    X = np.array([[1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0]])
    result = baseline.predict(None, X)
    assert len(result) == 1
    assert all(v == 0 for v in result)


def test_predict_empty_X():
    """predict with empty X returns empty array."""
    X = np.array([]).reshape(0, 12)
    result = baseline.predict(None, X)
    assert len(result) == 0


def test_train_baseline_no_sklearn():
    """train_baseline returns None when sklearn unavailable."""
    if not baseline.HAS_SKLEARN:
        result = baseline.train_baseline(
            np.array([[1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0]]),
            np.array([1]),
        )
        assert result is None
    else:
        # If sklearn IS available, train should work
        X = np.array([[1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0]])
        y = np.array([1])
        result = baseline.train_baseline(X, y)
        assert result is not None


def test_load_missing_file():
    """load returns None when file doesn't exist."""
    result = baseline.load("/non/existent/path/model.pkl")
    assert result is None


def test_save_load_roundtrip():
    """save + load should preserve the model."""
    if not baseline.HAS_SKLEARN:
        pytest.skip("sklearn not available")

    X = np.array(
        [[1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0], [0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0]]
    )
    y = np.array([1, 0])
    model = baseline.train_baseline(X, y)
    assert model is not None

    with tempfile.TemporaryDirectory() as tmpdir:
        path = os.path.join(tmpdir, "model.pkl")
        result_path = baseline.save(model, path)
        assert result_path == path
        assert os.path.exists(path)

        # JSON sidecar should also exist
        assert os.path.exists(path + ".json")

        loaded = baseline.load(path)
        assert loaded is not None

        # Predictions should match
        preds1 = baseline.predict(model, X)
        preds2 = baseline.predict(loaded, X)
        np.testing.assert_array_almost_equal(preds1, preds2, decimal=5)


def test_save_creates_directory():
    """save creates parent directories."""
    if not baseline.HAS_SKLEARN:
        pytest.skip("sklearn not available")

    X = np.array([[1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0]])
    y = np.array([1])
    model = baseline.train_baseline(X, y)

    with tempfile.TemporaryDirectory() as tmpdir:
        nested = os.path.join(tmpdir, "a", "b", "c", "model.pkl")
        baseline.save(model, nested)
        assert os.path.exists(nested)


def test_predict_returns_probabilities():
    """predict returns values in [0, 1]."""
    if not baseline.HAS_SKLEARN:
        pytest.skip("sklearn not available")

    X = np.array([[1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0]])
    y = np.array([1])
    model = baseline.train_baseline(X, y)
    preds = baseline.predict(model, X)
    assert all(0 <= p <= 1 for p in preds)


def test_predict_shape():
    """predict returns one prediction per row."""
    if not baseline.HAS_SKLEARN:
        pytest.skip("sklearn not available")

    X = np.array(
        [
            [1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0],
            [0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0],
            [1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0],
        ]
    )
    y = np.array([1, 0, 1])
    model = baseline.train_baseline(X, y)
    preds = baseline.predict(model, X)
    assert len(preds) == 3
