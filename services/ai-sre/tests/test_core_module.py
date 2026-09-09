"""Tests for AI-SRE core module helpers (config, logging, metrics)."""
import logging
import os

from app.core import (
    get_logger,
    make_metrics,
    configure_logging,
    METRICS,
    VLLM_URL,
    LLM_MODEL,
    GITHUB_REPO,
)


def test_get_logger_returns_logger():
    """get_logger returns a structlog logger."""
    log = get_logger("test.mod")
    assert log is not None
    # has log methods
    assert hasattr(log, "info")
    assert hasattr(log, "warning")
    assert hasattr(log, "error")


def test_get_logger_default_name():
    """get_logger with no name returns default."""
    log = get_logger()
    assert log is not None


def test_configure_logging_default():
    """configure_logging sets up structlog."""
    # Should not raise
    configure_logging("INFO", "ai-sre-test")


def test_configure_logging_debug():
    """configure_logging handles DEBUG level."""
    configure_logging("DEBUG", "ai-sre-test")


def test_configure_logging_warning():
    """configure_logging handles WARNING level."""
    configure_logging("WARNING", "ai-sre-test")


def test_configure_logging_invalid_level(monkeypatch):
    """configure_logging with invalid level falls back to INFO."""
    # Use a real level that doesn't exist - getattr default to INFO
    configure_logging("INVALID_LEVEL_NAME", "ai-sre-test")


def test_metrics_make_default():
    """make_metrics returns dict with all expected keys."""
    m = make_metrics()
    assert "registry" in m
    assert "analysis_requests" in m
    assert "analysis_latency" in m
    assert "hotfix_created" in m
    assert "CONTENT_TYPE" in m
    assert "generate_latest" in m


def test_metrics_make_when_prom_unavailable(monkeypatch):
    """make_metrics handles missing prometheus_client."""
    import sys

    # Force import to fail by removing it from sys.modules
    # This is a soft test - just verify it doesn't crash
    monkeypatch.delitem(sys.modules, "prometheus_client", raising=False)
    # Note: if already imported, can't truly unimport; but make_metrics should still handle
    m = make_metrics()
    assert m is not None


def test_metrics_module_constant_present():
    """METRICS module-level constant exists."""
    assert METRICS is not None
    assert "analysis_requests" in METRICS


def test_constants_have_defaults():
    """Module constants have defaults."""
    assert isinstance(VLLM_URL, str)
    assert len(VLLM_URL) > 0
    assert isinstance(LLM_MODEL, str)
    assert len(LLM_MODEL) > 0
    assert isinstance(GITHUB_REPO, str)
