"""Language detection API endpoint."""
from __future__ import annotations

from typing import Annotated, Any

from fastapi import APIRouter, File, HTTPException, UploadFile

from app.core import get_logger
from app.services import WhisperService, get_whisper_service, preprocess_audio

logger = get_logger(__name__)
router = APIRouter(prefix="/v1", tags=["language"])

# ---------------------------------------------------------------------------
# POST /v1/detect-language
# ---------------------------------------------------------------------------


@router.post("/detect-language")
async def detect_language(
    file: Annotated[UploadFile, File(description="Audio file to analyze")],
) -> dict[str, Any]:
    """Detect the spoken language in an audio file.

    Runs a short Whisper inference pass to determine the language
    and its probability without full transcription.

    Args:
        file: Audio file upload.

    Returns:
        Dict with detected language code and probability.
    """
    try:
        audio_bytes = await file.read()
    except Exception as exc:
        raise HTTPException(status_code=400, detail="Failed to read audio file") from exc

    if not audio_bytes:
        raise HTTPException(status_code=400, detail="Empty audio file")

    # Preprocess to standardize format
    audio_bytes = preprocess_audio(audio_bytes, target_sample_rate=16000)

    try:
        service: WhisperService = get_whisper_service()
        # Run transcription with language=None to let Whisper detect
        result = service.transcribe(audio_bytes, language=None, beam_size=1)

        logger.info(
            "language_detected",
            language=result.get("language"),
            probability=result.get("language_probability"),
        )

        return {
            "language": result.get("language"),
            "language_probability": result.get("language_probability"),
            "duration": result.get("duration"),
        }

    except Exception as exc:
        logger.error("language_detection_failed", error=str(exc))
        raise HTTPException(
            status_code=500,
            detail=f"Language detection failed: {exc}",
        ) from exc
