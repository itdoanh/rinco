"""STT sample / debug endpoints.

Routes:

- ``GET  /v1/samples``             — list available canned transcripts
- ``GET  /v1/samples/{name}``      — retrieve a single sample
- ``GET  /v1/status``              — service & dependency status report
"""
from __future__ import annotations

from typing import Any

from fastapi import APIRouter, HTTPException

from app.core import get_logger
from app.services.diarization import _pyannote_available
from app.services.samples import get_sample, list_samples
from app.services.whisper_service import _faster_whisper_available

logger = get_logger(__name__)
router = APIRouter(prefix="/v1", tags=["samples"])


@router.get("/samples")
async def samples() -> dict[str, Any]:
    """List all curated transcript samples."""
    return {
        "count": len(list_samples()),
        "samples": list_samples(),
    }


@router.get("/samples/{name}")
async def sample_by_name(name: str) -> dict[str, Any]:
    """Return a single transcript sample by name."""
    sample = get_sample(name)
    if not sample:
        raise HTTPException(404, f"sample {name} not found")
    return sample


@router.get("/status")
async def status() -> dict[str, Any]:
    """Report which dependencies are wired up."""
    return {
        "whisper_available": _faster_whisper_available,
        "diarization_available": _pyannote_available,
        "samples": len(list_samples()),
        "service": "stt",
        "version": "2.0.0",
    }


__all__ = ["router"]
