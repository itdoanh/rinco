"""Whisper transcription service using faster-whisper."""
from __future__ import annotations

import tempfile
from typing import Any

from app.core import (
    Config,
    WHISPER_COMPUTE,
    WHISPER_DEVICE,
    WHISPER_MODEL,
    get_logger,
)

logger = get_logger(__name__)

# ---------------------------------------------------------------------------
# faster-whisper availability
# ---------------------------------------------------------------------------

_faster_whisper_available = False
try:
    import faster_whisper

    _faster_whisper_available = True
except ImportError:
    faster_whisper = None  # type: ignore[assignment]


# ---------------------------------------------------------------------------
# Singleton
# ---------------------------------------------------------------------------

_whisper_service_instance: "WhisperService | None" = None


def get_whisper_service() -> "WhisperService":
    """Get or create the singleton WhisperService instance.

    Returns:
        WhisperService singleton.
    """
    global _whisper_service_instance  # noqa: PLW0603
    if _whisper_service_instance is None:
        _whisper_service_instance = WhisperService()
    return _whisper_service_instance


# ---------------------------------------------------------------------------
# Service
# ---------------------------------------------------------------------------


class WhisperService:
    """Speech-to-text service backed by faster-whisper.

    Loads the Whisper model lazily on first transcription request.
    """

    def __init__(
        self,
        model_size: str | None = None,
        device: str | None = None,
        compute_type: str | None = None,
    ) -> None:
        """Initialize the service (model not loaded yet).

        Args:
            model_size: Whisper model size (e.g. "large-v3"). Defaults to config.
            device: Device to run on ("cuda" or "cpu"). Defaults to config.
            compute_type: Compute type ("float16", "int8", etc.). Defaults to config.
        """
        self._model_size = model_size or WHISPER_MODEL
        self._device = device or WHISPER_DEVICE
        self._compute_type = compute_type or WHISPER_COMPUTE
        self._model: Any = None

    @property
    def model(self) -> Any:
        """Lazily load and return the Whisper model."""
        if self._model is None:
            self._load_model()
        return self._model

    def _load_model(self) -> None:
        """Load the faster-whisper model."""
        if not _faster_whisper_available:
            raise RuntimeError(
                "faster-whisper is not installed. "
                "Install it with: pip install faster-whisper"
            )

        logger.info(
            "loading_whisper_model",
            model=self._model_size,
            device=self._device,
            compute_type=self._compute_type,
        )

        try:
            self._model = faster_whisper.WhisperModel(
                self._model_size,
                device=self._device,
                compute_type=self._compute_type,
            )
            logger.info("whisper_model_loaded", model=self._model_size)
        except Exception as exc:  # pragma: no cover
            logger.error("whisper_model_load_failed", error=str(exc))
            raise

    def transcribe(
        self,
        audio_bytes: bytes,
        language: str = "vi",
        beam_size: int = 5,
        vad_filter: bool = True,
        vad_parameters: dict[str, Any] | None = None,
        initial_prompt: str | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """Transcribe audio bytes to text.

        Args:
            audio_bytes: Raw audio data (any format supported by ffmpeg).
            language: Source language code (ISO 639-1, e.g. "vi", "en").
            beam_size: Beam size for decoding (higher = better, slower).
            vad_filter: Enable voice activity detection filter.
            vad_parameters: VAD parameters dict.
            initial_prompt: Optional text prompt to guide the model.
            **kwargs: Additional faster-whisper transcribe options.

        Returns:
            Dict with keys:
                - text: Full transcribed text (str)
                - segments: List of segment dicts with start, end, text, confidence
                - language: Detected language code (str)
                - language_probability: Probability of detected language (float)
                - duration: Audio duration in seconds (float)
        """
        if vad_parameters is None:
            vad_parameters = {"min_silence_duration_ms": 500}

        # Write audio bytes to a temp file (faster-whisper reads from path)
        with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
            tmp.write(audio_bytes)
            tmp_path = tmp.name

        try:
            segments, info = self.model.transcribe(
                tmp_path,
                language=language,
                beam_size=beam_size,
                vad_filter=vad_filter,
                vad_parameters=vad_parameters,
                initial_prompt=initial_prompt,
                **kwargs,
            )

            # Materialize segments
            segment_list: list[dict[str, Any]] = []
            full_text_parts: list[str] = []

            for seg in segments:
                segment_dict: dict[str, Any] = {
                    "start": round(seg.start, 3),
                    "end": round(seg.end, 3),
                    "text": seg.text.strip(),
                    "confidence": round(seg.avg_logprob + 1.0, 3)
                    if seg.avg_logprob is not None
                    else 0.0,
                }
                segment_list.append(segment_dict)
                full_text_parts.append(seg.text)

            return {
                "text": " ".join(full_text_parts).strip(),
                "segments": segment_list,
                "language": info.language if info.language else language,
                "language_probability": round(info.language_probability, 4)
                if info.language_probability is not None
                else 1.0,
                "duration": round(info.duration, 3) if info.duration else 0.0,
            }

        finally:
            # Clean up temp file
            try:
                import os

                os.unlink(tmp_path)
            except OSError:
                pass
