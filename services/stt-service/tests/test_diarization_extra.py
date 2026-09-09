"""Tests for STT service diarization helpers (with mocked pyannote)."""
from __future__ import annotations

import sys
from pathlib import Path

_root = Path(__file__).resolve().parents[2]
if str(_root) not in sys.path:
    sys.path.insert(0, str(_root))

from unittest.mock import MagicMock, patch

import pytest

from app.services import diarization as dia_module
from app.services.diarization import diarize


# =============================================================================
# Pipeline is None (pyannote missing) → empty list
# =============================================================================

def test_extra_diarize_bytes_when_pyannote_missing(monkeypatch):
    """Without pyannote, diarize() on bytes returns an empty list."""
    monkeypatch.setattr(dia_module, "_pyannote_available", False)
    monkeypatch.setattr(dia_module, "Pipeline", None)
    assert diarize(b"fake-audio") == []


def test_extra_diarize_path_when_pyannote_missing(monkeypatch):
    """Same as above but with a path-like string input."""
    monkeypatch.setattr(dia_module, "_pyannote_available", False)
    monkeypatch.setattr(dia_module, "Pipeline", None)
    assert diarize("path/to/audio.wav") == []


# =============================================================================
# Pipeline present but pipeline runs - mocked itertracks
# =============================================================================

def test_extra_diarize_with_mocked_pipeline(monkeypatch):
    """With a mocked pyannote Pipeline, diarize() should produce segments."""

    class FakeTurn:
        def __init__(self, start, end):
            self.start = start
            self.end = end

    class FakeDiarization:
        def itertracks(self, yield_label=True):
            # Yield (turn, _, label) tuples like pyannote does.
            yield (FakeTurn(0.0, 2.0), None, "SPEAKER_00")
            yield (FakeTurn(2.0, 5.0), None, "SPEAKER_01")
            yield (FakeTurn(5.0, 7.5), None, "SPEAKER_00")

    fake_pipeline = MagicMock()
    fake_pipeline.return_value = FakeDiarization()

    monkeypatch.setattr(dia_module, "_pyannote_available", True)
    monkeypatch.setattr(dia_module, "Pipeline", MagicMock())
    monkeypatch.setattr(dia_module.Pipeline, "from_pretrained",
                        MagicMock(return_value=fake_pipeline))

    result = diarize("input.wav")
    assert len(result) == 3
    assert result[0]["speaker"] == "SPEAKER_00"
    assert result[0]["start"] == 0.0
    assert result[0]["end"] == 2.0
    assert result[1]["speaker"] == "SPEAKER_01"
    # Rounding to 3 decimal places.
    assert result[2]["end"] == 7.5


def test_extra_diarize_with_bytes_writes_tempfile(monkeypatch, tmp_path):
    """When input is bytes, diarize() writes a temp file then deletes it."""
    class FakeTurn:
        def __init__(self):
            self.start = 0.0
            self.end = 1.0

    class FakeDiarization:
        def itertracks(self, yield_label=True):
            yield (FakeTurn(), None, "S1")

    fake_pipeline = MagicMock()
    fake_pipeline.return_value = FakeDiarization()

    monkeypatch.setattr(dia_module, "_pyannote_available", True)
    monkeypatch.setattr(dia_module, "Pipeline", MagicMock())
    monkeypatch.setattr(dia_module.Pipeline, "from_pretrained",
                        MagicMock(return_value=fake_pipeline))

    # Patch tempfile.NamedTemporaryFile to point at tmp_path.
    class FakeTmp:
        def __init__(self, suffix="", delete=False):
            self._path = str(tmp_path / "tmp.wav")
            self.name = self._path

        def write(self, _data):
            (tmp_path / "tmp.wav").write_bytes(b"X")

        def __enter__(self):
            return self

        def __exit__(self, *a):
            return False

    import tempfile

    monkeypatch.setattr(tempfile, "NamedTemporaryFile", FakeTmp)

    deleted: list[str] = []
    import os

    original = os.unlink

    def fake_unlink(path):
        deleted.append(path)

    monkeypatch.setattr("os.unlink", fake_unlink)

    result = diarize(b"some-bytes", num_speakers=2)
    assert len(result) == 1
    # Path-based pipeline was called with the temp file path.
    args, kwargs = fake_pipeline.call_args
    assert args[0].endswith(".wav")
    assert kwargs["num_speakers"] == 2
    # Temp file was cleaned up (or attempted to be).
    assert any("tmp.wav" in p for p in deleted)


def test_extra_diarize_pipeline_exception_swallowed(monkeypatch):
    """If the pyannote pipeline raises, diarize() returns empty list."""
    fake_pipeline = MagicMock()
    fake_pipeline.side_effect = RuntimeError("model not loaded")

    monkeypatch.setattr(dia_module, "_pyannote_available", True)
    monkeypatch.setattr(dia_module, "Pipeline", MagicMock())
    monkeypatch.setattr(dia_module.Pipeline, "from_pretrained",
                        MagicMock(return_value=fake_pipeline))

    result = diarize("input.wav")
    assert result == []
