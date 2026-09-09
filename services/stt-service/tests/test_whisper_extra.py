"""Tests for STT service whisper_service.py (without loading the real model)."""
from __future__ import annotations

import sys
from pathlib import Path

# Ensure project root is on path for imports
_root = Path(__file__).resolve().parents[2]
if str(_root) not in sys.path:
    sys.path.insert(0, str(_root))

from unittest.mock import MagicMock, patch

import pytest

from app.services import whisper_service as ws_module
from app.services.whisper_service import WhisperService, get_whisper_service


# =============================================================================
# get_whisper_service (singleton)
# =============================================================================

def test_extra_get_whisper_service_singleton(monkeypatch):
    """get_whisper_service returns the same instance on repeated calls."""
    # Reset the cached singleton so the test is deterministic.
    monkeypatch.setattr(ws_module, "_whisper_service_instance", None)
    s1 = get_whisper_service()
    s2 = get_whisper_service()
    assert s1 is s2
    assert isinstance(s1, WhisperService)


def test_extra_get_whisper_service_constructs_with_defaults(monkeypatch):
    """Singleton uses the module-level Config defaults."""
    monkeypatch.setattr(ws_module, "_whisper_service_instance", None)
    s = get_whisper_service()
    # Defaults from core config.
    assert s._model_size == ws_module.WHISPER_MODEL
    assert s._device == ws_module.WHISPER_DEVICE
    assert s._compute_type == ws_module.WHISPER_COMPUTE


# =============================================================================
# WhisperService init / properties
# =============================================================================

def test_extra_whisper_service_init_defaults():
    """WhisperService() should inherit config defaults."""
    s = WhisperService()
    assert s._model_size == ws_module.WHISPER_MODEL
    assert s._device == ws_module.WHISPER_DEVICE
    assert s._compute_type == ws_module.WHISPER_COMPUTE
    assert s._model is None


def test_extra_whisper_service_init_override():
    """WhisperService accepts model_size/device/compute_type overrides."""
    s = WhisperService(model_size="tiny", device="cpu", compute_type="int8")
    assert s._model_size == "tiny"
    assert s._device == "cpu"
    assert s._compute_type == "int8"


def test_extra_whisper_service_model_lazy_load():
    """Touching .model triggers _load_model exactly once."""
    s = WhisperService()
    assert s._model is None
    fake_model = MagicMock(name="WhisperModel")
    with patch.object(s, "_load_model", autospec=True) as loader:
        loader.side_effect = lambda: setattr(s, "_model", fake_model)
        _ = s.model
        _ = s.model  # call twice
        _ = s.model
    loader.assert_called_once()


def test_extra_whisper_service_load_when_faster_whisper_missing(monkeypatch):
    """If faster_whisper is unavailable, _load_model raises RuntimeError."""
    s = WhisperService()
    monkeypatch.setattr(ws_module, "_faster_whisper_available", False)
    monkeypatch.setattr(ws_module, "faster_whisper", None)
    with pytest.raises(RuntimeError, match="faster-whisper"):
        s._load_model()


# =============================================================================
# transcribe() with mocked model
# =============================================================================

def _fake_segment(start, end, text, logprob=-0.5):
    seg = MagicMock()
    seg.start = start
    seg.end = end
    seg.text = text
    seg.avg_logprob = logprob
    return seg


def _fake_info(language="vi", prob=0.99, duration=10.0):
    info = MagicMock()
    info.language = language
    info.language_probability = prob
    info.duration = duration
    return info


def test_extra_transcribe_returns_full_payload(monkeypatch):
    """transcribe() should return a full payload with text, segments, and metadata."""
    s = WhisperService()
    segments = [
        _fake_segment(0.0, 2.5, "xin chào"),
        _fake_segment(2.5, 5.0, "thế giới"),
    ]
    info = _fake_info(language="vi", prob=0.987, duration=5.0)
    fake_model = MagicMock()
    fake_model.transcribe.return_value = (iter(segments), info)
    s._model = fake_model
    monkeypatch.setattr(ws_module, "_faster_whisper_available", True)

    result = s.transcribe(b"audio-bytes", language="vi", beam_size=5)

    assert result["language"] == "vi"
    assert result["language_probability"] == 0.987
    assert result["duration"] == 5.0
    assert "xin chào" in result["text"] and "thế giới" in result["text"]
    assert len(result["segments"]) == 2
    fake_model.transcribe.assert_called_once()
    _, kwargs = fake_model.transcribe.call_args
    # language + beam_size should be forwarded.
    assert kwargs["language"] == "vi"
    assert kwargs["beam_size"] == 5
    assert kwargs["vad_filter"] is True


def test_extra_transcribe_segment_confidence_uses_avg_logprob(monkeypatch):
    """avg_logprob is shifted by +1.0 so values are typically in [0, 1]."""
    s = WhisperService()
    seg = _fake_segment(0.0, 1.0, "hi", logprob=-0.3)
    info = _fake_info()
    s._model = MagicMock()
    s._model.transcribe.return_value = (iter([seg]), info)
    monkeypatch.setattr(ws_module, "_faster_whisper_available", True)

    result = s.transcribe(b"x")
    # -0.3 + 1.0 = 0.7
    assert result["segments"][0]["confidence"] == 0.7


def test_extra_transcribe_segment_without_logprob(monkeypatch):
    """avg_logprob is None → confidence falls back to 0.0."""
    s = WhisperService()
    seg = _fake_segment(0.0, 1.0, "hi", logprob=None)
    info = _fake_info()
    s._model = MagicMock()
    s._model.transcribe.return_value = (iter([seg]), info)
    monkeypatch.setattr(ws_module, "_faster_whisper_available", True)

    result = s.transcribe(b"x")
    assert result["segments"][0]["confidence"] == 0.0


def test_extra_transcribe_text_joins_segments_with_space(monkeypatch):
    """Multiple segments are joined with a space (not concatenated)."""
    s = WhisperService()
    segs = [
        _fake_segment(0.0, 1.0, "hello"),
        _fake_segment(1.0, 2.0, "world"),
    ]
    info = _fake_info()
    s._model = MagicMock()
    s._model.transcribe.return_value = (iter(segs), info)
    monkeypatch.setattr(ws_module, "_faster_whisper_available", True)

    result = s.transcribe(b"x")
    assert result["text"] == "hello world"


def test_extra_transcribe_uses_fallback_language_when_none(monkeypatch):
    """If info.language is None, fallback to the requested language."""
    s = WhisperService()
    info = MagicMock()
    info.language = None
    info.language_probability = None
    info.duration = None
    s._model = MagicMock()
    s._model.transcribe.return_value = (iter([]), info)
    monkeypatch.setattr(ws_module, "_faster_whisper_available", True)

    result = s.transcribe(b"x", language="en")
    assert result["language"] == "en"
    # Falls back to probability 1.0 when None.
    assert result["language_probability"] == 1.0
    # Falls back to 0.0 duration when None.
    assert result["duration"] == 0.0


def test_extra_transcribe_cleans_up_tmp_file(monkeypatch, tmp_path):
    """Temp file written to disk should be unlinked after transcribe."""
    s = WhisperService()
    s._model = MagicMock()
    info = _fake_info()
    s._model.transcribe.return_value = (iter([]), info)
    monkeypatch.setattr(ws_module, "_faster_whisper_available", True)

    # Track unlink calls via os.unlink patch.
    unlinked: list[str] = []
    import os

    original_unlink = os.unlink

    def fake_unlink(path):
        unlinked.append(path)
        return original_unlink(path) if not isinstance(path, str) or not path.startswith(str(tmp_path)) else None

    monkeypatch.setattr("os.unlink", fake_unlink)
    _ = s.transcribe(b"audio")
    # At least one file was deleted (the temp audio file).
    assert len(unlinked) >= 1


def test_extra_transcribe_handles_unlink_oserror(monkeypatch):
    """OSError on unlink is silently swallowed."""
    s = WhisperService()
    s._model = MagicMock()
    info = _fake_info()
    s._model.transcribe.return_value = (iter([]), info)
    monkeypatch.setattr(ws_module, "_faster_whisper_available", True)

    def raise_oserror(_):
        raise OSError("permission denied")

    monkeypatch.setattr("os.unlink", raise_oserror)
    # Should not panic.
    result = s.transcribe(b"x")
    assert "text" in result
