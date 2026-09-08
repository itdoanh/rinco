"""Baseline sklearn model used as fallback when XGBoost is unavailable."""
from __future__ import annotations

import json
import os
import pickle
from typing import Any, List

import numpy as np

try:
    from sklearn.linear_model import LogisticRegression  # type: ignore

    HAS_SKLEARN = True
except Exception:  # pragma: no cover
    HAS_SKLEARN = False


FEATURE_NAMES: List[str] = [
    "has_phone",
    "has_email",
    "company_length",
    "name_length",
    "country_vn",
    "page_views",
    "email_opens",
    "email_clicks",
    "is_b2b",
    "company_revenue_log",
    "source_paid",
    "channel_conv_rate",
]


def train_baseline(features: np.ndarray, labels: np.ndarray) -> Any:
    """Train a tiny logistic regression model.  Returns the fitted model or None."""
    if not HAS_SKLEARN:
        return None
    model = LogisticRegression(max_iter=200)
    model.fit(features, labels)
    return model


def save(model: Any, path: str) -> str:
    """Persist a fitted model to disk."""
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as f:
        pickle.dump({"model": model, "features": FEATURE_NAMES}, f)
    sidecar = path + ".json"
    with open(sidecar, "w") as f:
        json.dump(
            {
                "version": "1.0.0",
                "features": FEATURE_NAMES,
                "kind": "logreg",
            },
            f,
        )
    return path


def load(path: str) -> Any:
    """Load a fitted model from disk."""
    if not os.path.exists(path):
        return None
    with open(path, "rb") as f:
        return pickle.load(f)


def predict(model: Any, X: np.ndarray) -> np.ndarray:
    """Predict probabilities."""
    if model is None:
        return np.zeros(len(X))
    return model.predict_proba(X)[:, 1]


__all__ = [
    "FEATURE_NAMES",
    "HAS_SKLEARN",
    "train_baseline",
    "save",
    "load",
    "predict",
]