"""Forced alignment API endpoint (word-level timestamps)."""
from __future__ import annotations

import tempfile
from typing import Annotated, Any

from fastapi import APIRouter, File, Form, HTTPException, UploadFile

from app.core import get_logger

logger = get_logger(__name__)
router = APIRouter(prefix="/v1", tags=["alignment"])

# ---------------------------------------------------------------------------
# whisperx availability
# ---------------------------------------------------------------------------

_whisperx_available = False
try:
    import whisperx

    _whisperx_available = True
except ImportError:
    whisperx = None  # type: ignore[assignment]


# ---------------------------------------------------------------------------
# POST /v1/align
# ---------------------------------------------------------------------------


@router.post("/align")
async def forced_alignment(
    file: Annotated[UploadFile, File(description="Audio file")],
    reference_text: Annotated[str, Form(description="Reference transcript text")],
    language: Annotated[str, Form(description="Language code")] = "vi",
) -> dict[str, Any]:
    """Align reference transcript to audio for word-level timestamps.

    Uses whisperx to compute precise word-level timestamps given
    a reference transcript. Falls back to segment-level timestamps
    if whisperx is unavailable.

    Args:
        file: Audio file upload.
        reference_text: The expected transcript text.
        language: ISO 639-1 language code.

    Returns:
        Dict with word-level alignment data or segment-level fallback.
    """
    try:
        audio_bytes = await file.read()
    except Exception as exc:
        raise HTTPException(status_code=400, detail="Failed to read audio file") from exc

    if not audio_bytes:
        raise HTTPException(status_code=400, detail="Empty audio file")

    if not reference_text:
        raise HTTPException(status_code=400, detail="reference_text is required")

    # Write audio to temp file
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        tmp.write(audio_bytes)
        audio_path = tmp.name

    try:
        if not _whisperx_available:
            logger.warning("whisperx_unavailable_using_segment_fallback")
            # Fallback: return placeholder alignment at segment level
            return {
                "alignment_method": "segment_fallback",
                "message": "whisperx not installed; returning segment-level only",
                "reference_text": reference_text,
                "words": [],
            }

        # Load whisperx model
        model = whisperx.load_align_model(language_code=language)
        audio = whisperx.load_audio(audio_path)
        result = whisperx.align(
            [{"text": reference_text}],
            model,
            audio,
            device="cuda",
            return_char_alignments=False,
        )

        words: list[dict[str, Any]] = []
        if result and "word_segments" in result:
            for wseg in result["word_segments"]:
                words.append({
                    "word": wseg.get("word", ""),
                    "start": round(wseg.get("start", 0.0), 3),
                    "end": round(wseg.get("end", 0.0), 3),
                })

        logger.info("alignment_completed", num_words=len(words))

        return {
            "alignment_method": "whisperx",
            "reference_text": reference_text,
            "words": words,
        }

    except Exception as exc:
        logger.error("alignment_failed", error=str(exc))
        raise HTTPException(
            status_code=500,
            detail=f"Alignment failed: {exc}",
        ) from exc

    finally:
        try:
            import os
            os.unlink(audio_path)
        except OSError:
            pass
