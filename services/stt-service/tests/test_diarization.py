"""Tests for STT service diarization module."""
import pytest

from app.services import diarization


def test_pyannote_unavailable_returns_empty_list():
    """If pyannote is not installed, diarize returns [] (not raises)."""
    result = diarization.diarize(b"fake audio bytes", num_speakers=2)
    assert result == []


def test_diarize_with_bytes_input():
    """diarize accepts raw bytes and returns list."""
    result = diarization.diarize(b"\x00\x01\x02\x03", num_speakers=1)
    # If pyannote unavailable, returns []
    # If available, would attempt pipeline
    assert isinstance(result, list)


def test_diarize_with_string_path():
    """diarize accepts file path string."""
    result = diarization.diarize("/tmp/nonexistent.wav", num_speakers=None)
    assert isinstance(result, list)


def test_diarize_no_speaker_hint():
    """diarize with num_speakers=None falls back to inference."""
    result = diarization.diarize(b"audio")
    assert isinstance(result, list)


def test_pyannote_availability_flag():
    """_pyannote_available flag exists and is a bool."""
    assert isinstance(diarization._pyannote_available, bool)
