"""Prometheus metrics for lead-scoring."""
from __future__ import annotations

from prometheus_client import (
    CONTENT_TYPE_LATEST,
    CollectorRegistry,
    Counter,
    Gauge,
    Histogram,
    generate_latest,
)

# Per-service registry so tests + the main app share a single source of
# truth.
registry = CollectorRegistry()

REQUESTS_TOTAL = Counter(
    "lead_scoring_requests_total",
    "Total scoring requests",
    ["tenant_id", "result"],
    registry=registry,
)

INFERENCE_DURATION = Histogram(
    "lead_scoring_inference_duration_seconds",
    "Inference latency",
    ["model_version"],
    buckets=[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5],
    registry=registry,
)

TRAINING_RUNS_TOTAL = Counter(
    "lead_scoring_training_runs_total",
    "Training runs",
    ["tenant_id", "status"],
    registry=registry,
)

ERRORS_TOTAL = Counter(
    "lead_scoring_errors_total",
    "Scoring errors",
    ["tenant_id", "error_type"],
    registry=registry,
)

ENSEMBLE_SCORE = Gauge(
    "lead_scoring_ensemble_score",
    "Last ensemble score per tenant",
    ["tenant_id"],
    registry=registry,
)

MODEL_VERSION = Gauge(
    "lead_scoring_model_version_info",
    "Currently active model version",
    ["tenant_id", "version"],
    registry=registry,
)


def render() -> tuple[bytes, str]:
    """Return Prometheus exposition format body + content type."""
    return generate_latest(registry), CONTENT_TYPE_LATEST