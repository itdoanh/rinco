"""Seed mock data for the STT service.

Provides canned transcripts (whisper responses) so the service can be
smoke-tested without a real Whisper model loaded.
"""
from __future__ import annotations

import os
import uuid
from typing import Any, Dict, List

__all__ = [
    "MOCK_LANGUAGES",
    "MOCK_SEGMENTS",
    "MOCK_TRANSCRIPT_RESPONSE",
    "initialise_seed_data",
]


MOCK_LANGUAGES: List[Dict[str, Any]] = [
    {"code": "en", "name": "English", "supported": True},
    {"code": "vi", "name": "Vietnamese", "supported": True},
    {"code": "ja", "name": "Japanese", "supported": True},
    {"code": "ko", "name": "Korean", "supported": True},
    {"code": "zh", "name": "Chinese", "supported": True},
    {"code": "fr", "name": "French", "supported": True},
    {"code": "de", "name": "German", "supported": True},
    {"code": "es", "name": "Spanish", "supported": True},
]


MOCK_SEGMENTS: List[Dict[str, Any]] = [
    {
        "id": 0,
        "start": 0.0,
        "end": 4.5,
        "text": "Welcome to the RINCO platform demo.",
        "speaker": "S1",
        "confidence": 0.97,
        "tokens": [],
        "words": [],
    },
    {
        "id": 1,
        "start": 4.6,
        "end": 9.2,
        "text": "Today we'll show how to use the speech-to-text API.",
        "speaker": "S2",
        "confidence": 0.94,
        "tokens": [],
        "words": [],
    },
    {
        "id": 2,
        "start": 9.4,
        "end": 13.0,
        "text": "We support over 90 languages out of the box.",
        "speaker": "S1",
        "confidence": 0.92,
        "tokens": [],
        "words": [],
    },
]


MOCK_TRANSCRIPT_RESPONSE: Dict[str, Any] = {
    "id": uuid.uuid4().hex,
    "language": "en",
    "duration": 13.0,
    "text": (
        "Welcome to the RINCO platform demo. Today we'll show how to use "
        "the speech-to-text API. We support over 90 languages out of the box."
    ),
    "segments": MOCK_SEGMENTS,
    "model": "large-v3",
    "created_at": "2026-09-17T00:00:00Z",
}


def initialise_seed_data() -> None:
    """No-op for now — Whisper is lazy-loaded. Reserved for future state."""
    if os.getenv("STT_SERVICE_SEED", "1") not in ("1", "true", "TRUE", "yes"):
        return
    # The whisper model and diarization pipeline are lazily initialised
    # by the FastAPI lifespan handler. Nothing to pre-populate here yet.
