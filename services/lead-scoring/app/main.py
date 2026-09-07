"""FastAPI app entrypoint for lead-scoring."""
from __future__ import annotations

import asyncio
import os
from contextlib import asynccontextmanager
from typing import Any

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.api.scoring import router as scoring_router
from app.api.training import router as training_router
from app.api.explain import router as explain_router
from app.core.config import get_settings
from app.core.logging import configure_logging, get_logger
from app.core.tracing import configure_tracing, instrument_fastapi
from app.services.inference import GLOBAL_MODEL, _initialise_default_model
from app.services.nats_consumer import nats_consumer, warm_global_model

log = get_logger("lead-scoring.main")


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = get_settings()
    configure_logging(settings.log_level, settings.service_name)
    warm_global_model()
    configure_tracing(settings.service_name)

    task: asyncio.Task[Any] | None = None
    if settings.nats_url:
        task = asyncio.create_task(nats_consumer(app, subject=settings.nats_subject,
                                                  url=settings.nats_url))
    log.info(
        "lead_scoring_started",
        version=settings.version,
        global_model=GLOBAL_MODEL is not None,
    )
    try:
        yield
    finally:
        if task is not None:
            task.cancel()


def create_app() -> FastAPI:
    settings = get_settings()
    configure_logging(settings.log_level, settings.service_name)
    app = FastAPI(
        title="RINCO Lead Scoring",
        version=settings.version,
        lifespan=lifespan,
    )
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_methods=["*"],
        allow_headers=["*"],
    )
    instrument_fastapi(app)

    app.include_router(scoring_router)
    app.include_router(training_router)
    app.include_router(explain_router)

    @app.get("/")
    async def root():
        from app.services.inference import TENANT_MODELS

        return {
            "service": "lead-scoring",
            "version": settings.version,
            "models_loaded": list(TENANT_MODELS.keys()),
            "global_model": GLOBAL_MODEL is not None,
        }

    @app.exception_handler(Exception)
    async def unhandled(request: Request, exc: Exception):
        log.error("unhandled_exception", error=str(exc), path=str(request.url))
        return JSONResponse(status_code=500, content={"error": "internal server error"})

    return app


app = create_app()