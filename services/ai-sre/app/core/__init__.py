"""Core: config, logging, metrics."""
from __future__ import annotations

import logging
import os
import sys

import structlog

# ---------------------------------------------------------------- config
VLLM_URL = os.getenv("VLLM_URL", "http://vllm:8000")
VLLM_API_KEY = os.getenv("VLLM_API_KEY", "")
LLM_MODEL = os.getenv("LLM_MODEL", "deepseek-coder-v2-lite-instruct")
GITHUB_TOKEN = os.getenv("GITHUB_TOKEN", "")
GITHUB_REPO = os.getenv("GITHUB_REPO", "itdoanh/rinco")
GITHUB_REF = os.getenv("GITHUB_REF", "main")
CLICKHOUSE_URL = os.getenv("CLICKHOUSE_URL", "http://clickhouse:8123")
TELEGRAM_BOT_TOKEN = os.getenv("TELEGRAM_BOT_TOKEN", "")
TELEGRAM_CHAT_ID = os.getenv("TELEGRAM_CHAT_ID", "")
LOKI_URL = os.getenv("LOKI_URL", "http://loki:3100")
JAEGER_URL = os.getenv("JAEGER_URL", "http://jaeger:16686")
PROMETHEUS_URL = os.getenv("PROMETHEUS_URL", "http://prometheus:9090")

# ---------------------------------------------------------------- logging
def configure_logging(level: str = "INFO", service: str = "ai-sre") -> None:
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
    return structlog.get_logger(name or "ai-sre")


# ---------------------------------------------------------------- metrics
def make_metrics():
    try:
        from prometheus_client import (  # type: ignore
            CollectorRegistry, Counter, Histogram, CONTENT_TYPE_LATEST, generate_latest,
        )
        registry = CollectorRegistry()
        analysis_requests = Counter(
            "ai_sre_analysis_requests_total",
            "Total RCA analyses",
            ["tenant_id", "result"], registry=registry,
        )
        analysis_latency = Histogram(
            "ai_sre_analysis_latency_seconds",
            "Analysis latency",
            buckets=[0.5, 1, 2.5, 5, 10, 30, 60],
            registry=registry,
        )
        hotfix_created = Counter(
            "ai_sre_hotfix_created_total",
            "Hotfix PRs created",
            ["tenant_id", "status"], registry=registry,
        )
        return dict(
            registry=registry,
            analysis_requests=analysis_requests,
            analysis_latency=analysis_latency,
            hotfix_created=hotfix_created,
            CONTENT_TYPE=CONTENT_TYPE_LATEST,
            generate_latest=lambda: generate_latest(registry),
        )
    except Exception:
        return dict(
            registry=None, analysis_requests=None, analysis_latency=None,
            hotfix_created=None,
            CONTENT_TYPE="text/plain; charset=utf-8",
            generate_latest=lambda: b"unavailable",
        )


METRICS = make_metrics()
configure_logging(os.getenv("LOG_LEVEL", "INFO"), "ai-sre")