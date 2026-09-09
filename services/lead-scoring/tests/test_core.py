"""Tests for lead-scoring core (config, logging, metrics)."""
from __future__ import annotations

import os
from unittest.mock import patch

import pytest


def test_extra_settings_defaults(monkeypatch):
    """Settings loads with sensible defaults."""
    # Clear env to use defaults
    for k in list(os.environ):
        if k.startswith("LEAD_SCORING_"):
            monkeypatch.delenv(k, raising=False)
    monkeypatch.setenv("LEAD_SCORING_ENV", "test")
    from app.core.config import get_settings, Settings

    # Reset cache
    get_settings.cache_clear()
    s = get_settings()
    assert s.service_name == "lead-scoring"
    assert s.max_batch_size == 1000


def test_extra_settings_env_override(monkeypatch):
    """Settings reads from env with LEAD_SCORING_ prefix."""
    monkeypatch.setenv("LEAD_SCORING_MAX_BATCH_SIZE", "500")
    from app.core.config import get_settings

    get_settings.cache_clear()
    s = get_settings()
    assert s.max_batch_size == 500


def test_extra_settings_env_prefix(monkeypatch):
    """Settings uses LEAD_SCORING_ prefix."""
    # Without LEAD_SCORING_ prefix, env should not affect settings
    monkeypatch.setenv("MAX_BATCH_SIZE", "999")
    from app.core.config import get_settings

    get_settings.cache_clear()
    s = get_settings()
    # The setting should still be 1000 (default), not 999
    assert s.max_batch_size == 1000


def test_extra_settings_cached():
    """get_settings returns the same instance (lru_cache)."""
    from app.core.config import get_settings

    get_settings.cache_clear()
    s1 = get_settings()
    s2 = get_settings()
    assert s1 is s2


def test_extra_env_or():
    """env_or returns default when env var is not set."""
    from app.core.config import env_or

    key = "TEST_NONEXISTENT_VAR_12345"
    if key in os.environ:
        del os.environ[key]
    assert env_or(key, "default") == "default"


def test_extra_env_or_present():
    """env_or returns env value when set."""
    from app.core.config import env_or

    os.environ["TEST_ENV_OR_KEY"] = "value"
    assert env_or("TEST_ENV_OR_KEY", "default") == "value"
    del os.environ["TEST_ENV_OR_KEY"]


def test_extra_settings_log_level_default():
    """Default log level is INFO."""
    from app.core.config import Settings
    s = Settings()
    assert s.log_level == "INFO"


def test_extra_settings_allow_public_train_default():
    """Public train should be disabled by default for security."""
    from app.core.config import Settings
    s = Settings()
    assert s.allow_public_train is False


def test_extra_configure_logging():
    """configure_logging can be called multiple times."""
    from app.core.logging import configure_logging
    configure_logging("DEBUG", "test-svc")
    configure_logging("INFO", "test-svc")


def test_extra_get_logger():
    """get_logger returns a logger."""
    from app.core.logging import get_logger
    log = get_logger("test-module")
    assert log is not None


def test_extra_get_logger_default():
    """get_logger without name uses default."""
    from app.core.logging import get_logger
    log = get_logger()
    assert log is not None


def test_extra_metrics_registry():
    """Metrics module exposes a registry."""
    from app.core.metrics import registry
    assert registry is not None


def test_extra_metrics_requests_total():
    """REQUESTS_TOTAL counter can be incremented."""
    from app.core.metrics import REQUESTS_TOTAL
    REQUESTS_TOTAL.labels(tenant_id="t1", result="ok").inc()
    REQUESTS_TOTAL.labels(tenant_id="t1", result="ok").inc()


def test_extra_metrics_training_runs():
    """TRAINING_RUNS_TOTAL counter works."""
    from app.core.metrics import TRAINING_RUNS_TOTAL
    TRAINING_RUNS_TOTAL.labels(tenant_id="t1", status="success").inc()


def test_extra_metrics_errors():
    """ERRORS_TOTAL counter works."""
    from app.core.metrics import ERRORS_TOTAL
    ERRORS_TOTAL.labels(tenant_id="t1", error_type="validation").inc()


def test_extra_metrics_render():
    """render() returns Prometheus exposition format."""
    from app.core.metrics import render
    body, content_type = render()
    assert isinstance(body, bytes)
    assert "text/plain" in content_type
    assert len(body) > 0


def test_extra_metrics_gauge():
    """Gauge metrics can be set."""
    from app.core.metrics import ENSEMBLE_SCORE
    ENSEMBLE_SCORE.labels(tenant_id="t1").set(0.85)


def test_extra_metrics_model_version():
    """Model version gauge can be set."""
    from app.core.metrics import MODEL_VERSION
    MODEL_VERSION.labels(tenant_id="t1", version="v1").set(1)


def test_extra_metrics_inference_duration():
    """Inference duration histogram can be observed."""
    from app.core.metrics import INFERENCE_DURATION
    INFERENCE_DURATION.labels(model_version="v1").observe(0.05)
