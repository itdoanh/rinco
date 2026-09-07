"""FastAPI application for the STT service."""
from __future__ import annotations

from contextlib import asynccontextmanager
from typing import Any, AsyncIterator

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import PlainTextResponse

from app import __version__
from app.api import align_router, language_router, transcribe_router
from app.core import METRICS, configure_logging, get_logger

configure_logging()
logger = get_logger(__name__)


# ---------------------------------------------------------------------------
# Lifespan
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    """Application lifespan manager."""
    logger.info("stt_service_starting", version=__version__)
    yield
    logger.info("stt_service_shutting_down")


# ---------------------------------------------------------------------------
# Factory
# ---------------------------------------------------------------------------


def create_app() -> FastAPI:
    """Create and configure the FastAPI application."""
    application = FastAPI(
        title="STT Service",
        description="Speech-to-text transcription service using Whisper",
        version=__version__,
        lifespan=lifespan,
        docs_url="/docs",
        redoc_url="/redoc",
        openapi_url="/openapi.json",
    )

    # CORS
    application.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # Routers
    application.include_router(transcribe_router)
    application.include_router(language_router)
    application.include_router(align_router)

    # Health & metrics
    @application.get("/v1/health", tags=["health"])
    def health() -> dict[str, str]:
        return {"status": "healthy", "service": "stt"}

    @application.get("/v1/metrics", tags=["health"])
    def metrics() -> PlainTextResponse:
        if METRICS._registry is not None:
            from prometheus_client import generate_latest

            return PlainTextResponse(
                generate_latest(METRICS._registry),
                media_type="text/plain",
            )
        return PlainTextResponse("No metrics available", media_type="text/plain")

    @application.get("/", tags=["info"])
    def root() -> dict[str, Any]:
        return {
            "service": "STT Service",
            "version": __version__,
            "description": "Speech-to-text using faster-whisper (large-v3)",
            "docs": "/docs",
        }

    return application


# Default app instance
app = create_app()
