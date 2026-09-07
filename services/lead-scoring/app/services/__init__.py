"""Service layer: features, inference, training, clients, NATS consumer."""
from __future__ import annotations

from app.services.features import (
    DEFAULT_FEATURES,
    featurize,
    feature_names,
)
from app.services.inference import (
    TrainedModel,
    predict_score,
    tier_from_score,
    recommended_action,
    confidence_from_features,
    shap_top_k,
    _initialise_default_model,
    GLOBAL_MODEL,
    TENANT_MODELS,
)
from app.services.training import (
    persist_model,
    load_model,
    train,
)

__all__ = [
    "DEFAULT_FEATURES",
    "featurize",
    "feature_names",
    "TrainedModel",
    "predict_score",
    "tier_from_score",
    "recommended_action",
    "confidence_from_features",
    "shap_top_k",
    "_initialise_default_model",
    "GLOBAL_MODEL",
    "TENANT_MODELS",
    "persist_model",
    "load_model",
    "train",
]