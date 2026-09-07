"""FastAPI application for AI SRE service."""
from __future__ import annotations

import time
from contextlib import asynccontextmanager
from typing import AsyncIterator

from fastapi import FastAPI, Request, Response
from fastapi.middleware.cors import CORSMiddleware

try:
    from prometheus_client import generate_latest, CONTENT_TYPE_LATEST  # type: ignore
except ImportError:
    generate_latest = None
    CONTENT_TYPE_LATEST = "text/plain; charset=utf-8"

from app.core import get_logger, METRICS, configure_logging
from app.api import (
    analyze_router,
    incidents_router,
    hotfix_router,
    runbook_router,
    chat_router,
)

configure_logging()
logger = get_logger("ai-sre.main")


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    """Application lifespan: startup and shutdown hooks."""
    t0 = time.monotonic()
    logger.info("ai_sre_starting", version="2.0.0")
    # Warm up: check connectivity to dependent services could be added here
    elapsed = time.monotonic() - t0
    logger.info("ai_sre_ready", startup_ms=round(elapsed * 1000))
    yield
    logger.info("ai_sre_shutting_down")


app = FastAPI(
    title="AI SRE Worker",
    description="AI-powered Site Reliability Engine: RCA, hotfix generation, and incident runbooks.",
    version="2.0.0",
    lifespan=lifespan,
)

# CORS for browser-based clients
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Mount routers
app.include_router(analyze_router)
app.include_router(incidents_router)
app.include_router(hotfix_router)
app.include_router(runbook_router)
app.include_router(chat_router)


@app.get("/health")
async def health() -> dict:
    """Basic health check."""
    return {"status": "ok", "service": "ai-sre", "version": "2.0.0"}


@app.get("/ready")
async def ready() -> dict:
    """Readiness probe — checks that core modules are importable."""
    try:
        from app import services  # noqa: F401
        from app.api import analyze  # noqa: F401
        return {"status": "ready"}
    except Exception as e:
        return {"status": "not_ready", "error": str(e)}


@app.get("/metrics")
async def metrics() -> Response:
    """Prometheus metrics endpoint."""
    metrics_data = METRICS
    if metrics_data.get("generate_latest"):
        return Response(
            content=metrics_data["generate_latest"](),
            media_type=metrics_data.get("CONTENT_TYPE", CONTENT_TYPE_LATEST),
        )
    return Response(content="unavailable", media_type=CONTENT_TYPE_LATEST)
