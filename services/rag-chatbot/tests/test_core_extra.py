"""Tests for rag-chatbot core module."""
from __future__ import annotations

import importlib
import os

import pytest

from app.core import (
    configure_logging,
    get_logger,
    make_metrics,
    METRICS,
)


def test_default_config_values(monkeypatch):
    """Verify default config values are loaded."""
    for key in ["VLLM_URL", "VLLM_API_KEY", "EMBEDDING_MODEL", "RERANKER_MODEL", "LLM_MODEL", "QDRANT_URL"]:
        monkeypatch.delenv(key, raising=False)
    # Reload core to pick up defaults
    import app.core
    importlib.reload(app.core)
    assert app.core.VLLM_URL == "http://vllm:8000/v1"
    assert app.core.VLLM_API_KEY == ""
    assert app.core.EMBEDDING_MODEL == "BAAI/bge-m3"
    assert app.core.RERANKER_MODEL == "BAAI/bge-reranker-base"
    assert app.core.LLM_MODEL == "meta-llama/Llama-3-8B-Instruct"
    assert app.core.QDRANT_URL == "http://qdrant:6333"


def test_configurable_values(monkeypatch):
    monkeypatch.setenv("VLLM_URL", "http://custom:1234/v1")
    monkeypatch.setenv("VLLM_API_KEY", "secret-key")
    monkeypatch.setenv("EMBEDDING_MODEL", "custom-embed")
    monkeypatch.setenv("RERANKER_MODEL", "custom-rerank")
    monkeypatch.setenv("LLM_MODEL", "custom-llm")
    monkeypatch.setenv("QDRANT_URL", "http://qdrant-custom:9999")
    monkeypatch.setenv("CHUNK_SIZE", "256")
    monkeypatch.setenv("CHUNK_OVERLAP", "32")
    monkeypatch.setenv("TOP_K", "10")
    import app.core
    importlib.reload(app.core)
    assert app.core.VLLM_URL == "http://custom:1234/v1"
    assert app.core.VLLM_API_KEY == "secret-key"
    assert app.core.EMBEDDING_MODEL == "custom-embed"
    assert app.core.RERANKER_MODEL == "custom-rerank"
    assert app.core.LLM_MODEL == "custom-llm"
    assert app.core.QDRANT_URL == "http://qdrant-custom:9999"
    assert app.core.CHUNK_SIZE == 256
    assert app.core.CHUNK_OVERLAP == 32
    assert app.core.DEFAULT_TOP_K == 10


def test_configure_logging_default():
    configure_logging()
    # No exception means success


def test_configure_logging_debug():
    configure_logging("DEBUG")
    import logging
    assert logging.getLogger().level == logging.DEBUG


def test_configure_logging_warning():
    configure_logging("WARNING")
    import logging
    assert logging.getLogger().level == logging.WARNING


def test_configure_logging_error():
    configure_logging("ERROR")
    import logging
    assert logging.getLogger().level == logging.ERROR


def test_configure_logging_invalid_defaults_to_info():
    configure_logging("BOGUS")
    import logging
    assert logging.getLogger().level == logging.INFO


def test_get_logger_default_name():
    log = get_logger()
    assert log is not None


def test_get_logger_with_name():
    log = get_logger("test.module")
    assert log is not None


def test_make_metrics_returns_dict():
    metrics = make_metrics()
    assert isinstance(metrics, dict)


def test_make_metrics_has_keys():
    metrics = make_metrics()
    for key in ["registry", "chat_requests", "chat_latency", "ingest_docs",
                "search_requests", "streaming_tokens", "CONTENT_TYPE", "generate_latest"]:
        assert key in metrics


def test_make_metrics_content_type_default():
    """If prometheus is missing, fallback content-type is text/plain."""
    metrics = make_metrics()
    # Either prometheus or fallback
    assert "CONTENT_TYPE" in metrics


def test_metrics_module_level_singleton():
    """METRICS is created at module load time."""
    assert METRICS is not None
    assert "CONTENT_TYPE" in METRICS


def test_generate_latest_callable():
    metrics = make_metrics()
    fn = metrics["generate_latest"]
    assert callable(fn)
    result = fn()
    assert isinstance(result, (bytes, bytearray))
