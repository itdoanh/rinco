"""Tests for rag-chatbot core (config, logging, metrics)."""
from __future__ import annotations

import os

from app.core import (
    CHUNK_SIZE,
    CHUNK_OVERLAP,
    DEFAULT_TOP_K,
    VLLM_URL,
    EMBEDDING_MODEL,
    RERANKER_MODEL,
    LLM_MODEL,
    QDRANT_URL,
    configure_logging,
    get_logger,
    make_metrics,
    METRICS,
)


def test_extra_config_defaults():
    """Config defaults should be sane values."""
    # Just verify they exist as strings/ints
    assert isinstance(EMBEDDING_MODEL, str)
    assert len(EMBEDDING_MODEL) > 0
    assert isinstance(RERANKER_MODEL, str)
    assert isinstance(LLM_MODEL, str)
    assert isinstance(VLLM_URL, str)
    assert isinstance(QDRANT_URL, str)
    assert isinstance(CHUNK_SIZE, int)
    assert isinstance(CHUNK_OVERLAP, int)
    assert isinstance(DEFAULT_TOP_K, int)
    assert CHUNK_SIZE > 0
    assert CHUNK_OVERLAP >= 0
    assert DEFAULT_TOP_K > 0


def test_extra_configure_logging():
    """configure_logging can be called."""
    configure_logging("DEBUG", "test-svc")
    configure_logging("INFO", "test-svc")


def test_extra_get_logger():
    """get_logger returns a logger."""
    log = get_logger("test-module")
    assert log is not None


def test_extra_get_logger_default():
    """get_logger without name returns default."""
    log = get_logger()
    assert log is not None


def test_extra_make_metrics():
    """make_metrics returns a metrics dict."""
    m = make_metrics()
    assert isinstance(m, dict)
    assert "registry" in m
    assert "chat_requests" in m
    assert "chat_latency" in m
    assert "ingest_docs" in m
    assert "search_requests" in m
    assert "streaming_tokens" in m
    assert "CONTENT_TYPE" in m
    assert "generate_latest" in m


def test_extra_make_metrics_content_type():
    """CONTENT_TYPE should be valid Prometheus format or fallback."""
    m = make_metrics()
    ct = m["CONTENT_TYPE"]
    assert isinstance(ct, str)
    assert len(ct) > 0


def test_extra_make_metrics_generate_latest():
    """generate_latest should be callable."""
    m = make_metrics()
    result = m["generate_latest"]()
    assert isinstance(result, (bytes, str))


def test_extra_metrics_global():
    """METRICS should be defined."""
    assert METRICS is not None
    assert isinstance(METRICS, dict)


def test_extra_metrics_counters_incrementable():
    """Counters should be incrementable (if available)."""
    m = make_metrics()
    if m["chat_requests"] is not None:
        m["chat_requests"].labels(tenant_id="t1", result="ok").inc()
    if m["ingest_docs"] is not None:
        m["ingest_docs"].labels(tenant_id="t1", kind="text").inc()


def test_extra_metrics_histogram_observable():
    """Latency histogram should be observable (if available)."""
    m = make_metrics()
    if m["chat_latency"] is not None:
        m["chat_latency"].observe(0.5)


def test_extra_config_env_override(monkeypatch):
    """Environment variables should override defaults."""
    monkeypatch.setenv("VLLM_URL", "http://custom:8000/v1")
    monkeypatch.setenv("EMBEDDING_MODEL", "custom-model")
    monkeypatch.setenv("CHUNK_SIZE", "1024")
    # Reimport to pick up new env
    import importlib
    import app.core
    importlib.reload(app.core)
    # After reload, new values should be present
    assert app.core.VLLM_URL == "http://custom:8000/v1"
    assert app.core.EMBEDDING_MODEL == "custom-model"
    assert app.core.CHUNK_SIZE == 1024
    # Cleanup
    monkeypatch.delenv("VLLM_URL", raising=False)
    monkeypatch.delenv("EMBEDDING_MODEL", raising=False)
    monkeypatch.delenv("CHUNK_SIZE", raising=False)
