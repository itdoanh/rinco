"""Seed mock data for the recording service.

Pre-populates the in-process recording store with a few demo entries so
that the list/GET endpoints serve non-empty results in dev mode.
"""
from __future__ import annotations

import os
import time
from typing import Any, Dict, List

from .models import RecordingMetadata, TranscriptPayload

# WS-B Loop 9: extended mock data (20+ recordings)
try:
    from app.expansion.seed_extra import (
        EXTENDED_RECORDINGS,
        EXTENDED_STORAGE_STATS,
        EXTENDED_TRANSCRIPTION_JOBS,
    )
except ImportError:  # pragma: no cover
    EXTENDED_RECORDINGS = []
    EXTENDED_STORAGE_STATS = []
    EXTENDED_TRANSCRIPTION_JOBS = []

__all__ = [
    "MOCK_RECORDINGS",
    "EXTENDED_RECORDINGS",
    "EXTENDED_STORAGE_STATS",
    "EXTENDED_TRANSCRIPTION_JOBS",
    "seed_recordings",
]


def _meta(
    rec_id: str,
    room_id: str,
    user_id: str,
    status: str,
    *,
    started_at: float,
    storage_path: str | None = None,
    transcript_url: str | None = None,
) -> RecordingMetadata:
    return RecordingMetadata(
        id=rec_id,
        room_id=room_id,
        user_id=user_id,
        tenant_id="demo-tenant",
        status=status,
        started_at=started_at,
        ended_at=started_at + 1800.0,
        duration_seconds=1800.0,
        size_bytes=12_500_000 if storage_path else None,
        storage_path=storage_path,
        layout="grid",
        quality="hd",
        participants=[user_id, "peer-1", "peer-2"],
        download_url=(
            f"https://stub/rinco-recordings/recordings/{rec_id}.mp4?expires=86400"
            if storage_path
            else None
        ),
        transcript_url=transcript_url,
    )


MOCK_RECORDINGS: List[Dict[str, Any]] = [
    _meta(
        "rec-001",
        "room-demo-001",
        "user-alice",
        "ready",
        started_at=time.time() - 86400,
        storage_path="recordings/rec-001.mp4",
        transcript_url="https://stub/rinco-recordings/transcripts/rec-001.json",
    ).model_dump(),
    _meta(
        "rec-002",
        "room-demo-002",
        "user-bob",
        "processing",
        started_at=time.time() - 3600,
    ).model_dump(),
    _meta(
        "rec-003",
        "room-demo-003",
        "user-carol",
        "ready",
        started_at=time.time() - 7200,
        storage_path="recordings/rec-003.mp4",
    ).model_dump(),
]


def seed_recordings(service) -> None:  # type: ignore[no-untyped-def]
    """Inject mock recordings into a ``RecordingService`` instance."""
    if os.getenv("RECORDING_SEED", "1") not in ("1", "true", "TRUE", "yes"):
        return
    for rec in MOCK_RECORDINGS:
        rid = rec["id"]
        if rid not in service.recordings:
            # Re-hydrate into a RecordingMetadata instance.
            service.recordings[rid] = RecordingMetadata(**rec)


MOCK_TRANSCRIPT = TranscriptPayload(
    segments=[
        {"start": 0.0, "end": 4.2, "speaker": "S1", "text": "Welcome to the RINCO demo."},
        {"start": 4.2, "end": 8.5, "speaker": "S2", "text": "Thanks for joining today."},
    ],
    language="en",
)
