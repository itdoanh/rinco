"""Transcription API endpoints."""
from __future__ import annotations

import time
from typing import Annotated, Any, AsyncIterator

from fastapi import APIRouter, File, Form, HTTPException, UploadFile
from fastapi.responses import StreamingResponse
from sse_starlette.sse import EventSourceResponse

from app.core import METRICS, get_logger
from app.services import WhisperService, get_whisper_service, preprocess_audio

logger = get_logger(__name__)
router = APIRouter(prefix="/v1", tags=["transcribe"])

# ---------------------------------------------------------------------------
# Supported languages
# ---------------------------------------------------------------------------

SUPPORTED_LANGUAGES = [
    {"code": "vi", "name": "Vietnamese"},
    {"code": "en", "name": "English"},
    {"code": "zh", "name": "Chinese"},
    {"code": "ja", "name": "Japanese"},
    {"code": "ko", "name": "Korean"},
    {"code": "fr", "name": "French"},
    {"code": "de", "name": "German"},
    {"code": "es", "name": "Spanish"},
    {"code": "pt", "name": "Portuguese"},
    {"code": "ru", "name": "Russian"},
    {"code": "ar", "name": "Arabic"},
    {"code": "hi", "name": "Hindi"},
    {"code": "th", "name": "Thai"},
    {"code": "id", "name": "Indonesian"},
    {"code": "ms", "name": "Malay"},
    {"code": "tl", "name": "Tagalog"},
    {"code": "my", "name": "Burmese"},
    {"code": "km", "name": "Khmer"},
    {"code": "lo", "name": "Lao"},
]


@router.get("/languages")
def list_languages() -> list[dict[str, str]]:
    """Return the list of supported language codes and names."""
    return SUPPORTED_LANGUAGES


# ---------------------------------------------------------------------------
# POST /v1/transcribe
# ---------------------------------------------------------------------------

@router.post("/transcribe")
async def transcribe_audio(
    file: Annotated[UploadFile, File(description="Audio file (any ffmpeg-supported format)")],
    language: Annotated[str, Form(description="Source language code (default: vi)")] = "vi",
    beam_size: Annotated[int, Form(description="Beam size for decoding")] = 5,
    task: Annotated[str, Form(description="Task: 'transcribe' or 'translate'")] = "transcribe",
    normalize_volume: Annotated[bool, Form(description="Apply loudness normalization")] = False,
) -> dict[str, Any]:
    """Transcribe an audio file to text.

    Accepts multipart audio upload, preprocesses it to 16kHz mono WAV,
    runs Whisper transcription, and returns text with segment timestamps.

    Args:
        file: Audio file upload.
        language: ISO 639-1 language code.
        beam_size: Beam size for beam search decoding.
        task: "transcribe" or "translate" to English.
        normalize_volume: Whether to apply loudness normalization.

    Returns:
        Dict with text, segments, language, language_probability, duration.
    """
    start_time = time.monotonic()

    # Read file content
    try:
        audio_bytes = await file.read()
    except Exception as exc:
        logger.error("audio_read_failed", error=str(exc))
        raise HTTPException(status_code=400, detail="Failed to read audio file") from exc

    if not audio_bytes:
        raise HTTPException(status_code=400, detail="Empty audio file")

    # Preprocess
    try:
        audio_bytes = preprocess_audio(audio_bytes, target_sample_rate=16000)
    except Exception as exc:
        logger.warning("preprocessing_failed", error=str(exc))
        # Continue anyway

    # Transcribe
    try:
        service: WhisperService = get_whisper_service()
        result = service.transcribe(
            audio_bytes,
            language=language,
            beam_size=beam_size,
            task=task,
        )

        elapsed = time.monotonic() - start_time
        logger.info(
            "transcription_completed",
            language=language,
            duration=result.get("duration", 0),
            elapsed=round(elapsed, 3),
            text_length=len(result.get("text", "")),
        )

        # Record metrics
        metrics = METRICS
        if metrics.requests_total is not None:
            metrics.requests_total.labels(
                status="success",
                language=language,
            ).inc()
        if metrics.latency is not None:
            metrics.latency.labels(language=language).observe(elapsed)

        return result

    except Exception as exc:
        elapsed = time.monotonic() - start_time
        logger.error("transcription_failed", error=str(exc))

        if METRICS.errors is not None:
            METRICS.errors.labels(error_type=type(exc).__name__).inc()
        if METRICS.latency is not None:
            METRICS.latency.labels(language=language).observe(elapsed)

        raise HTTPException(status_code=500, detail=f"Transcription failed: {exc}") from exc


# ---------------------------------------------------------------------------
# POST /v1/transcribe/stream (SSE)
# ---------------------------------------------------------------------------


async def _transcribe_stream_generator(
    audio_bytes: bytes,
    language: str,
    beam_size: int,
) -> AsyncIterator[dict[str, Any]]:
    """Async generator yielding transcription events for SSE."""
    service = get_whisper_service()

    try:
        result = service.transcribe(
            audio_bytes,
            language=language,
            beam_size=beam_size,
        )

        # Yield full result
        yield {"event": "complete", "data": result}

    except Exception as exc:
        yield {"event": "error", "data": {"error": str(exc)}}


@router.post("/transcribe/stream")
async def transcribe_audio_stream(
    file: Annotated[UploadFile, File(description="Audio file")],
    language: Annotated[str, Form(description="Source language code")] = "vi",
    beam_size: Annotated[int, Form(description="Beam size")] = 5,
) -> StreamingResponse:
    """Stream transcription results using Server-Sent Events (SSE).

    Same as /transcribe but streams results as they become available.

    Args:
        file: Audio file upload.
        language: ISO 639-1 language code.
        beam_size: Beam size for decoding.

    Returns:
        SSE stream with transcription segments.
    """
    try:
        audio_bytes = await file.read()
    except Exception as exc:
        raise HTTPException(status_code=400, detail="Failed to read audio file") from exc

    if not audio_bytes:
        raise HTTPException(status_code=400, detail="Empty audio file")

    # Preprocess
    audio_bytes = preprocess_audio(audio_bytes, target_sample_rate=16000)

    async def event_generator() -> AsyncIterator[dict[str, Any]]:
        async for event in _transcribe_stream_generator(audio_bytes, language, beam_size):
            yield event

    return EventSourceResponse(event_generator())
