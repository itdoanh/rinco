"""Speaker diarization using pyannote.audio."""
from __future__ import annotations

import tempfile
from typing import Any

from app.core import get_logger

logger = get_logger(__name__)

# ---------------------------------------------------------------------------
# pyannote.audio availability
# ---------------------------------------------------------------------------

_pyannote_available = False
try:
    from pyannote.audio import Pipeline

    _pyannote_available = True
except ImportError:
    Pipeline = None  # type: ignore[assignment, misc]


# ---------------------------------------------------------------------------
# Diarization
# ---------------------------------------------------------------------------


def diarize(
    audio_path_or_bytes: str | bytes,
    num_speakers: int | None = None,
) -> list[dict[str, Any]]:
    """Perform speaker diarization on audio.

    Identifies speaker segments in audio without requiring a transcript.

    Args:
        audio_path_or_bytes: Path to an audio file or raw audio bytes.
        num_speakers: Hint for number of speakers. If None, pyannote infers.

    Returns:
        List of dicts, each with keys:
            - start (float): Segment start in seconds.
            - end (float): Segment end in seconds.
            - speaker (str): Speaker label (e.g. "SPEAKER_00").
    """
    if not _pyannote_available:
        logger.warning("pyannote_unavailable_diarization_skipped")
        return []

    try:
        # pyannote expects a file path
        if isinstance(audio_path_or_bytes, bytes):
            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
                tmp.write(audio_path_or_bytes)
                audio_path = tmp.name
        else:
            audio_path = audio_path_or_bytes

        try:
            pipeline = Pipeline.from_pretrained(
                "pyannote/speaker-diarization-3.1",
                use_auth_token=None,  # Set PYANNOTE_TOKEN env var
            )

            diarization = pipeline(
                audio_path,
                num_speakers=num_speakers,
            )

            segments: list[dict[str, Any]] = []
            for turn, _, speaker in diarization.itertracks(yield_label=True):
                segments.append({
                    "start": round(turn.start, 3),
                    "end": round(turn.end, 3),
                    "speaker": speaker,
                })

            logger.info(
                "diarization_completed",
                num_segments=len(segments),
                num_speakers=len({s["speaker"] for s in segments}),
            )
            return segments

        finally:
            if isinstance(audio_path_or_bytes, bytes):
                try:
                    import os
                    os.unlink(audio_path)
                except OSError:
                    pass

    except Exception as exc:
        logger.warning("diarization_failed", error=str(exc))
        return []
