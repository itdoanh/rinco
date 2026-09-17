"""Lead feature extraction utilities.

Wraps :func:`app.services.features.featurize` to expose feature
engineering as a standalone endpoint useful for debugging/explainability
and for use by other services that want to feed the same numeric
vectors into their own models.

Also provides:
- :func:`extract_features_dict` for callers that only want the raw
  feature dict (without producing a model-row DataFrame).
- :func:`feature_schema` returns the canonical ordered list of
  features produced by :func:`app.services.features.feature_names`.
"""
from __future__ import annotations

from typing import Any, Sequence

from app.services.features import (
    DEFAULT_FEATURES,
    _compute_row,
    feature_names as _feature_names,
    featurize as _featurize,
)
from app.schemas.lead import LeadFeatures


def feature_schema() -> list[str]:
    """Return the canonical, deterministic feature name list."""
    return _feature_names()


def extract_features_dict(features: LeadFeatures) -> dict[str, float]:
    """Compute feature values for ``features`` and return as a flat dict.

    Useful when callers want to introspect the engineered features
    without instantiating a model.
    """
    return _compute_row(features)


def vectorize(features: LeadFeatures, names: Sequence[str] | None = None) -> list[float]:
    """Return a 1-D vector aligned with ``names`` (or the default schema)."""
    frame = _featurize(features, list(names) if names else None)
    if hasattr(frame, "to_numpy"):
        arr = frame.to_numpy(dtype=float)
    else:
        # _DictFrame fallback
        cols = frame.columns
        arr = [[row[c] for c in cols] for row in frame.rows]
        import numpy as np  # type: ignore
        arr = np.asarray(arr, dtype=float)
    return [float(v) for v in arr[0]]


__all__ = [
    "DEFAULT_FEATURES",
    "feature_schema",
    "extract_features_dict",
    "vectorize",
]
