"""STT API routes package."""
from __future__ import annotations

from app.api.transcribe import router as transcribe_router
from app.api.language import router as language_router
from app.api.align import router as align_router

__all__ = [
    "transcribe_router",
    "language_router",
    "align_router",
]
