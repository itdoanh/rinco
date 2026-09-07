"""Inference pipeline.

The model is an ensemble of (optionally) XGBoost, an ONNX NN and a
sklearn LogisticRegression baseline.  We always keep the baseline
ready so scoring never fails when the heavier models are unavailable.
"""
from __future__ import annotations

import os
import pickle
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional

import numpy as np

try:
    import xgboost as xgb  # type: ignore

    HAS_XGB = True
except Exception:  # pragma: no cover
    HAS_XGB = False
try:
    import onnxruntime as ort  # type: ignore

    HAS_ORT = True
except Exception:  # pragma: no cover
    HAS_ORT = False
try:
    import shap  # type: ignore

    HAS_SHAP = True
except Exception:  # pragma: no cover
    HAS_SHAP = False


@dataclass
class TrainedModel:
    """A trained model + version + metrics + feature names."""

    xgb: Optional[Any] = None
    nn_session: Optional[Any] = None  # onnxruntime InferenceSession
    fallback: Optional[Any] = None  # sklearn LogisticRegression
    scaler: Optional[Any] = None
    features: List[str] = field(default_factory=list)
    metrics: Dict[str, float] = field(default_factory=dict)
    version: str = "1.0.0"
    trained_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    fallback_mode: bool = False


# State
TENANT_MODELS: Dict[str, TrainedModel] = {}
GLOBAL_MODEL: Optional[TrainedModel] = None  # default fallback

MODEL_DIR = os.getenv("MODEL_DIR", "/models")


def _initialise_default_model() -> TrainedModel:
    """Bootstrap a synthetic model so inference always succeeds."""
    m = TrainedModel(features=list(_default_features()), fallback_mode=True)
    if HAS_XGB:
        try:
            m.xgb = xgb.XGBClassifier(
                n_estimators=80,
                max_depth=4,
                learning_rate=0.08,
                random_state=42,
                eval_metric="logloss",
            )
            np.random.seed(0)
            n = 400
            X = np.random.rand(n, len(m.features))
            y = (X[:, 4] + 0.6 * X[:, 0] + 0.3 * X[:, 14] > 0.55).astype(int)
            m.xgb.fit(X, y)
        except Exception:  # pragma: no cover
            m.xgb = None
    try:
        from sklearn.linear_model import LogisticRegression

        np.random.seed(0)
        X = np.random.rand(400, len(m.features))
        y = (X[:, 0] + X[:, 4] > 1.0).astype(int)
        m.fallback = LogisticRegression(max_iter=200).fit(X, y)
    except Exception:  # pragma: no cover
        pass
    return m


def _default_features() -> List[str]:
    from app.services.features import DEFAULT_FEATURES

    return DEFAULT_FEATURES


def _ensure_2d(arr: np.ndarray) -> np.ndarray:
    return arr.reshape(1, -1) if arr.ndim == 1 else arr


def predict_score(model: TrainedModel, X: Any) -> float:
    """Ensemble inference: 0.7 XGBoost + 0.3 NN (or fallback)."""
    if hasattr(X, "to_numpy"):
        X_arr = X.to_numpy(dtype=float)
    else:
        X_arr = np.asarray(X, dtype=float)
    score = 0.0
    weight = 0.0
    if model.xgb is not None:
        try:
            proba = float(model.xgb.predict_proba(X_arr)[0, 1])
            score += 0.7 * proba
            weight += 0.7
        except Exception:  # pragma: no cover
            pass
    if model.nn_session is not None:
        try:
            input_name = model.nn_session.get_inputs()[0].name
            out = model.nn_session.run(
                None, {input_name: X_arr.astype("float32")}
            )[0]
            score += 0.3 * float(np.squeeze(out))
            weight += 0.3
        except Exception:  # pragma: no cover
            pass
    if model.fallback is not None:
        try:
            proba = float(model.fallback.predict_proba(X_arr)[0, 1])
            share = 1.0 - weight
            score += share * proba
            weight += share
        except Exception:  # pragma: no cover
            pass
    if weight == 0:
        # No model available — return a neutral 0.5 instead of raising.
        return 0.5
    proba = score / weight if weight != 1.0 else score
    return max(0.0, min(1.0, proba))


def tier_from_score(score: float) -> str:
    if score >= 0.85:
        return "very-hot"
    if score >= 0.6:
        return "hot"
    if score >= 0.3:
        return "warm"
    return "cold"


def recommended_action(tier: str) -> str:
    return {
        "very-hot": "call-now",
        "hot": "call-soon",
        "warm": "email",
        "cold": "nurture",
    }.get(tier, "nurture")


def confidence_from_features(model: TrainedModel, X: Any) -> float:
    if model.fallback is None:
        return 0.7
    try:
        if hasattr(X, "to_numpy"):
            arr = X.to_numpy(dtype=float)
        else:
            arr = np.asarray(X, dtype=float)
        proba = model.fallback.predict_proba(arr)[0]
        p = float(np.clip(proba[1], 1e-3, 1 - 1e-3))
        ent = -p * np.log2(p) - (1 - p) * np.log2(1 - p)
        return max(0.5, min(0.99, 1.0 - ent))
    except Exception:  # pragma: no cover
        return 0.7


def shap_top_k(model: TrainedModel, X: Any, k: int = 5) -> Dict[str, float]:
    if not HAS_SHAP or model.xgb is None:
        return {}
    try:
        explainer = shap.TreeExplainer(model.xgb)
        sv = explainer.shap_values(X)
        vals = sv[0] if isinstance(sv, list) else sv[0]
        pairs = sorted(
            zip(model.features, [float(v) for v in vals]),
            key=lambda p: abs(p[1]),
            reverse=True,
        )[:k]
        return {name: value for name, value in pairs}
    except Exception:  # pragma: no cover
        return {}


def load_model(tenant_id: str) -> TrainedModel:
    if tenant_id in TENANT_MODELS:
        return TENANT_MODELS[tenant_id]
    pid = os.path.join(MODEL_DIR, tenant_id, "model.pkl")
    if os.path.exists(pid):
        try:
            with open(pid, "rb") as f:
                m = pickle.load(f)
            TENANT_MODELS[tenant_id] = m
            return m
        except Exception:  # pragma: no cover
            pass
    global GLOBAL_MODEL
    if GLOBAL_MODEL is None:
        GLOBAL_MODEL = _initialise_default_model()
    return GLOBAL_MODEL


def persist_model(tenant_id: str, model: TrainedModel) -> str:
    """Persist a trained model to disk."""
    outdir = os.path.join(MODEL_DIR, tenant_id)
    os.makedirs(outdir, exist_ok=True)
    path = os.path.join(outdir, "model.pkl")
    with open(path, "wb") as f:
        pickle.dump(model, f)
    TENANT_MODELS[tenant_id] = model
    return path


__all__ = [
    "TrainedModel",
    "TENANT_MODELS",
    "GLOBAL_MODEL",
    "MODEL_DIR",
    "predict_score",
    "tier_from_score",
    "recommended_action",
    "confidence_from_features",
    "shap_top_k",
    "_initialise_default_model",
    "load_model",
    "persist_model",
]