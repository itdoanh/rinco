"""Tests for lead-scoring baseline model helpers."""
import numpy as np
import pytest

from app.models import baseline


def test_feature_names_constant():
    """FEATURE_NAMES is a list of strings."""
    assert isinstance(baseline.FEATURE_NAMES, list)
    assert len(baseline.FEATURE_NAMES) > 0
    for name in baseline.FEATURE_NAMES:
        assert isinstance(name, str)


def test_feature_names_expected_count():
    """12 expected features for lead scoring."""
    assert len(baseline.FEATURE_NAMES) == 12


def test_feature_names_unique():
    """All feature names are unique."""
    assert len(baseline.FEATURE_NAMES) == len(set(baseline.FEATURE_NAMES))


def test_has_sklearn_flag_is_bool():
    """HAS_SKLEARN is a boolean."""
    assert isinstance(baseline.HAS_SKLEARN, bool)


def test_train_baseline_no_sklearn_returns_none(monkeypatch):
    """Without sklearn, train_baseline returns None."""
    monkeypatch.setattr(baseline, "HAS_SKLEARN", False)
    X = np.array([[1, 0, 5, 3]])
    y = np.array([1])
    result = baseline.train_baseline(X, y)
    assert result is None


def test_predict_with_none_model_returns_zeros():
    """predict with None model returns zeros."""
    X = np.array([[1, 0, 5, 3], [0, 1, 2, 7]])
    result = baseline.predict(None, X)
    assert len(result) == 2
    assert all(v == 0 for v in result)


def test_predict_zero_length():
    """predict on empty array."""
    X = np.array([]).reshape(0, 4)
    result = baseline.predict(None, X)
    assert len(result) == 0


def test_load_nonexistent_returns_none(tmp_path):
    """load returns None for nonexistent path."""
    fake = tmp_path / "nonexistent.pkl"
    result = baseline.load(str(fake))
    assert result is None


def test_save_and_load_roundtrip(tmp_path):
    """save then load should restore the model."""
    if not baseline.HAS_SKLEARN:
        pytest.skip("sklearn not available")

    X = np.array([[1, 0, 5, 3, 1, 2, 1, 0, 1, 4.0, 1, 0.5]])
    y = np.array([1])
    model = baseline.train_baseline(X, y)
    assert model is not None

    path = str(tmp_path / "model.pkl")
    saved = baseline.save(model, path)
    assert saved == path

    loaded = baseline.load(path)
    assert loaded is not None


def test_save_creates_sidecar_json(tmp_path):
    """save also creates a .json sidecar with metadata."""
    if not baseline.HAS_SKLEARN:
        pytest.skip("sklearn not available")

    X = np.array([[1, 0, 5, 3, 1, 2, 1, 0, 1, 4.0, 1, 0.5]])
    y = np.array([1])
    model = baseline.train_baseline(X, y)
    path = str(tmp_path / "subdir" / "model.pkl")
    baseline.save(model, path)

    import json
    with open(path + ".json") as f:
        meta = json.load(f)
    assert "version" in meta
    assert "features" in meta
    assert meta["kind"] == "logreg"
    assert meta["features"] == baseline.FEATURE_NAMES


def test_train_baseline_multiclass_labels():
    """Training with multiple classes works."""
    if not baseline.HAS_SKLEARN:
        pytest.skip("sklearn not available")

    X = np.array([
        [1, 0, 5, 3, 1, 2, 1, 0, 1, 4.0, 1, 0.5],
        [0, 1, 2, 7, 0, 1, 0, 1, 0, 3.5, 0, 0.2],
        [1, 1, 8, 5, 1, 3, 2, 1, 1, 5.0, 1, 0.7],
        [0, 0, 1, 1, 0, 0, 0, 0, 0, 1.0, 0, 0.0],
    ])
    y = np.array([1, 0, 1, 0])
    model = baseline.train_baseline(X, y)
    assert model is not None
    pred = baseline.predict(model, X)
    assert len(pred) == 4
    for p in pred:
        assert 0.0 <= p <= 1.0
