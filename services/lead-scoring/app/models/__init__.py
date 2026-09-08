"""Model artifacts (baseline fallback)."""
from app.models.baseline import (
    FEATURE_NAMES,
    HAS_SKLEARN,
    load,
    predict,
    save,
    train_baseline,
)

__all__ = ["FEATURE_NAMES", "HAS_SKLEARN", "train_baseline", "save", "load", "predict"]