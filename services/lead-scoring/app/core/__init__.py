"""Core utilities: config, logging, metrics, tracing."""
from __future__ import annotations

from app.core.config import Settings, get_settings
from app.core.logging import configure_logging, get_logger
from app.core.metrics import (
    MODEL_VERSION,
    INFERENCE_DURATION,
    REQUESTS_TOTAL,
    TRAINING_RUNS_TOTAL,
    ERRORS_TOTAL,
    ENSEMBLE_SCORE,
    registry,
)
from app.core.tracing import configure_tracing, instrument_fastapi

__all__ = [
    "Settings",
    "get_settings",
    "configure_logging",
    "get_logger",
    "MODEL_VERSION",
    "INFERENCE_DURATION",
    "REQUESTS_TOTAL",
    "TRAINING_RUNS_TOTAL",
    "ERRORS_TOTAL",
    "ENSEMBLE_SCORE",
    "registry",
    "configure_tracing",
    "instrument_fastapi",
]