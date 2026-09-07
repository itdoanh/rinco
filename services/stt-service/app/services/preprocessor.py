"""Audio preprocessing utilities (ffmpeg-based normalization and resampling)."""
from __future__ import annotations

import subprocess
import sys
from typing import Any

from app.core import get_logger

logger = get_logger(__name__)

# ---------------------------------------------------------------------------
# ffmpeg availability
# ---------------------------------------------------------------------------

_ffmpeg_available = False
try:
    result = subprocess.run(
        ["ffmpeg", "-version"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    _ffmpeg_available = result.returncode == 0
except Exception:
    _ffmpeg_available = False


# ---------------------------------------------------------------------------
# Preprocessing
# ---------------------------------------------------------------------------


def preprocess_audio(
    audio_bytes: bytes,
    target_sample_rate: int = 16000,
) -> bytes:
    """Normalize and resample audio to WAV PCM 16-bit mono.

    Converts any audio format supported by ffmpeg to a standardized
    16kHz mono WAV suitable for Whisper inference.

    Args:
        audio_bytes: Raw audio data (any ffmpeg-supported format).
        target_sample_rate: Target sample rate in Hz. Defaults to 16000.

    Returns:
        WAV-encoded audio bytes (PCM 16-bit mono).
    """
    if not _ffmpeg_available:
        logger.warning("ffmpeg_unavailable_skipping_preprocessing")
        return audio_bytes

    try:
        proc = subprocess.run(
            [
                sys.executable, "-m", "ffmpeg",
                "-y",
                "-hide_banner",
                "-loglevel", "error",
                "-i", "pipe:0",
                "-ar", str(target_sample_rate),
                "-ac", "1",
                "-f", "wav",
                "pipe:1",
            ],
            input=audio_bytes,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=120,
        )
        if proc.returncode != 0:
            logger.warning(
                "ffmpeg_preprocessing_failed",
                returncode=proc.returncode,
                stderr=proc.stderr.decode(errors="replace"),
            )
            return audio_bytes

        return proc.stdout

    except subprocess.TimeoutExpired:
        logger.warning("ffmpeg_preprocessing_timed_out")
        return audio_bytes
    except Exception as exc:
        logger.warning("ffmpeg_preprocessing_error", error=str(exc))
        return audio_bytes


def normalize_volume(
    audio_bytes: bytes,
    target_dbfs: float = -20.0,
) -> bytes:
    """Normalize audio loudness using the ffmpeg loudnorm filter.

    Applies the ITU-R BS.1770-4 loudness normalization algorithm.

    Args:
        audio_bytes: Raw audio data.
        target_dbfs: Target integrated loudness in dBFS.
            Typical values: -14 (podcast), -20 (broadcast), -23 (streaming).

    Returns:
        Loudness-normalized audio bytes (WAV PCM 16-bit mono).
    """
    if not _ffmpeg_available:
        logger.warning("ffmpeg_unavailable_skipping_volume_normalization")
        return audio_bytes

    try:
        proc = subprocess.run(
            [
                sys.executable, "-m", "ffmpeg",
                "-y",
                "-hide_banner",
                "-loglevel", "error",
                "-i", "pipe:0",
                "-af", f"loudnorm=I={target_dbfs}:TP=-1.5:LRA=11",
                "-ar", "16000",
                "-ac", "1",
                "-f", "wav",
                "pipe:1",
            ],
            input=audio_bytes,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=120,
        )
        if proc.returncode != 0:
            logger.warning(
                "ffmpeg_volume_normalization_failed",
                returncode=proc.returncode,
                stderr=proc.stderr.decode(errors="replace"),
            )
            return audio_bytes

        return proc.stdout

    except subprocess.TimeoutExpired:
        logger.warning("ffmpeg_volume_normalization_timed_out")
        return audio_bytes
    except Exception as exc:
        logger.warning("ffmpeg_volume_normalization_error", error=str(exc))
        return audio_bytes
