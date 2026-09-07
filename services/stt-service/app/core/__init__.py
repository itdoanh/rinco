"""STT service core package (config, logging, metrics)."""
from __future__ import annotations

__all__ = [
    "Config",
    "configure_logging",
    "get_logger",
    "METRICS",
    "Metrics",
]

import os
import sys
from typing import Any

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

WHISPER_MODEL: str = os.getenv("WHISPER_MODEL", "large-v3")
WHISPER_DEVICE: str = os.getenv("WHISPER_DEVICE", "cuda")
WHISPER_COMPUTE: str = os.getenv("WHISPER_COMPUTE", "float16")
AUDIO_SAMPLE_RATE: int = int(os.getenv("AUDIO_SAMPLE_RATE", "16000"))


class Config:
    """Application configuration."""

    WHISPER_MODEL = WHISPER_MODEL
    WHISPER_DEVICE = WHISPER_DEVICE
    WHISPER_COMPUTE = WHISPER_COMPUTE
    AUDIO_SAMPLE_RATE = AUDIO_SAMPLE_RATE


# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

_structlog_available = False
try:
    import structlog

    _structlog_available = True
except ImportError:
    structlog = None  # type: ignore[assignment]


def configure_logging(level: str | None = None) -> None:
    """Configure structured logging.

    Args:
        level: Log level (DEBUG, INFO, WARNING, ERROR). Defaults to INFO.
    """
    if not _structlog_available:
        import logging

        logging.basicConfig(
            level=getattr(logging, (level or "INFO").upper()),
            format="%(asctime)s %(levelname)s %(name)s %(message)s",
            stream=sys.stdout,
        )
        return

    structlog.configure(
        processors=[
            structlog.stdlib.filter_by_level,
            structlog.stdlib.add_logger_name,
            structlog.stdlib.add_log_level,
            structlog.stdlib.PositionalArgumentsFormatter(),
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.processors.UnicodeDecoder(),
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.stdlib.BoundLogger,
        context_class=dict,
        logger_factory=structlog.stdlib.LoggerFactory(),
        cache_logger_on_first_use=True,
    )


def get_logger(name: str | None = None) -> Any:
    """Get a logger instance.

    Args:
        name: Logger name. Defaults to __name__ of caller.

    Returns:
        Logger instance (structlog if available, stdlib otherwise).
    """
    if _structlog_available:
        return structlog.get_logger(name) if name else structlog.get_logger()
    else:
        import logging

        return logging.getLogger(name)


# ---------------------------------------------------------------------------
# Metrics
# ---------------------------------------------------------------------------

_prometheus_available = False
try:
    from prometheus_client import Counter, Histogram, CollectorRegistry, REGISTRY

    _prometheus_available = True
except ImportError:
    Counter = None  # type: ignore[assignment, misc]
    Histogram = None  # type: ignore[assignment, misc]
    REGISTRY = None  # type: ignore[assignment, misc]


class Metrics:
    """Prometheus metrics for the STT service."""

    def __init__(self) -> None:
        self._registry = REGISTRY
        self._requests_total: Counter | None = None
        self._latency: Histogram | None = None
        self._errors: Counter | None = None

    def init_metrics(self) -> None:
        """Initialize all metrics (call once at startup)."""
        if not _prometheus_available:
            return

        self._requests_total = Counter(
            "transcribe_requests_total",
            "Total number of transcription requests",
            ["status", "language"],
            registry=self._registry,
        )
        self._latency = Histogram(
            "transcribe_latency_seconds",
            "Transcription request latency in seconds",
            ["language"],
            buckets=(0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 120.0),
            registry=self._registry,
        )
        self._errors = Counter(
            "transcribe_errors_total",
            "Total number of transcription errors",
            ["error_type"],
            registry=self._registry,
        )

    @property
    def requests_total(self) -> Counter | None:
        return self._requests_total

    @property
    def latency(self) -> Histogram | None:
        return self._latency

    @property
    def errors(self) -> Counter | None:
        return self._errors


# Global metrics instance
METRICS = Metrics()
METRICS.init_metrics()
