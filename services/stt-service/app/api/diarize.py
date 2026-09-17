"""Diarization endpoint for the STT service.

This endpoint exposes a simple ``POST /v1/diarize`` route that wraps
:func:`app.services.diarization.diarize`.  When pyannote.audio is
unavailable (or pyannote's torch/torchvision dependency is broken)
the endpoint returns an empty segments list rather than failing — the
caller can fall back to single-speaker transcription.
"""
from __future__ import annotations

from typing import Annotated, Any

from fastapi import APIRouter, File, Form, HTTPException, UploadFile

from app.core import get_logger
from app.services.diarization import diarize

logger = get_logger(__name__)
router = APIRouter(prefix="/v1", tags=["diarization"])


@router.post("/diarize")
async def diarize_audio(
    file: Annotated[UploadFile, File(description="Audio file (any ffmpeg format)")],
    num_speakers: Annotated[int, Form(description="Hint for number of speakers")] | None = None,
) -> dict[str, Any]:
    """Run speaker diarization on the supplied audio file.

    Returns a list of dicts: ``{start, end, speaker}``.
    """
    try:
        audio_bytes = await file.read()
    except Exception as exc:
        raise HTTPException(status_code=400, detail="Failed to read audio file") from exc
    if not audio_bytes:
        raise HTTPException(status_code=400, detail="Empty audio file")

    try:
        segments = diarize(audio_bytes, num_speakers=num_speakers)
    except Exception as exc:
        logger.error("diarize_endpoint_failed", error=str(exc))
        # Always degrade gracefully — return empty segments so the
        # caller can decide whether to retry with a different service.
        segments = []

    speakers = sorted({s["speaker"] for s in segments})
    return {
        "segments": segments,
        "num_segments": len(segments),
        "num_speakers": len(speakers),
        "speakers": speakers,
    }


__all__ = ["router"]
