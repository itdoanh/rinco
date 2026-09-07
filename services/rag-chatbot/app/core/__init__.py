"""Core: config, logging, metrics, tracing."""
from __future__ import annotations

import logging
import os
import sys

import structlog

# ---------------------------------------------------------------- config
VLLM_URL = os.getenv("VLLM_URL", "http://vllm:8000/v1")
VLLM_API_KEY = os.getenv("VLLM_API_KEY", "")
EMBEDDING_MODEL = os.getenv("EMBEDDING_MODEL", "BAAI/bge-m3")
RERANKER_MODEL = os.getenv("RERANKER_MODEL", "BAAI/bge-reranker-base")
LLM_MODEL = os.getenv("LLM_MODEL", "meta-llama/Llama-3-8B-Instruct")
QDRANT_URL = os.getenv("QDRANT_URL", "http://qdrant:6333")
CHUNK_SIZE = int(os.getenv("CHUNK_SIZE", "512"))
CHUNK_OVERLAP = int(os.getenv("CHUNK_OVERLAP", "64"))
DEFAULT_TOP_K = int(os.getenv("TOP_K", "20"))

# ---------------------------------------------------------------- logging
def configure_logging(level: str = "INFO", service: str = "rag-chatbot") -> None:
    log_level = getattr(logging, level.upper(), logging.INFO)
    logging.basicConfig(format="%(message)s", stream=sys.stdout, level=log_level, force=True)
    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.processors.add_log_level,
            structlog.processors.TimeStamper(fmt="iso", utc=True),
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(log_level),
        logger_factory=structlog.PrintLoggerFactory(file=sys.stdout),
        cache_logger_on_first_use=True,
    )


def get_logger(name: str | None = None):
    return structlog.get_logger(name or "rag-chatbot")


# ---------------------------------------------------------------- metrics
def make_metrics():
    try:
        from prometheus_client import (  # type: ignore
            CollectorRegistry, Counter, Histogram, CONTENT_TYPE_LATEST, generate_latest,
        )
        registry = CollectorRegistry()
        chat_requests = Counter(
            "rag_chat_requests_total", "Total chat requests",
            ["tenant_id", "result"], registry=registry,
        )
        chat_latency = Histogram(
            "rag_chat_latency_seconds", "End-to-end chat latency",
            buckets=[0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10], registry=registry,
        )
        ingest_docs = Counter(
            "rag_ingest_documents_total", "Documents ingested",
            ["tenant_id", "kind"], registry=registry,
        )
        search_requests = Counter(
            "rag_search_requests_total", "Semantic search requests",
            ["tenant_id"], registry=registry,
        )
        streaming_tokens = Counter(
            "rag_streaming_tokens_total", "Tokens streamed",
            ["tenant_id", "model"], registry=registry,
        )
        return dict(
            registry=registry,
            chat_requests=chat_requests,
            chat_latency=chat_latency,
            ingest_docs=ingest_docs,
            search_requests=search_requests,
            streaming_tokens=streaming_tokens,
            CONTENT_TYPE=CONTENT_TYPE_LATEST,
            generate_latest=lambda: generate_latest(registry),
        )
    except Exception:
        return dict(
            registry=None, chat_requests=None, chat_latency=None,
            ingest_docs=None, search_requests=None, streaming_tokens=None,
            CONTENT_TYPE="text/plain; charset=utf-8",
            generate_latest=lambda: b"unavailable",
        )


METRICS = make_metrics()

configure_logging(os.getenv("LOG_LEVEL", "INFO"), "rag-chatbot")