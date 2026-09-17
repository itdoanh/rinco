"""FastAPI entry point for the recording-service."""

from __future__ import annotations

import logging
import os

from fastapi import FastAPI

from .routers import build_router
from .services import RecordingService
from .storage import Storage


logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"))
logger = logging.getLogger("recording-service")


def create_app() -> FastAPI:
    storage = Storage()
    service = RecordingService(storage=storage)
    # Inject mock recordings for dev/offline mode.
    from .seed import seed_recordings

    seed_recordings(service)

    app = FastAPI(
        title="RINCO Recording Service",
        description="Capture, store and transcribe WebRTC meeting recordings.",
        version="0.3.0",
    )
    app.include_router(build_router(service))
    return app


# ``uvicorn app.main:app`` works because the symbol is module-level.
app = create_app()
