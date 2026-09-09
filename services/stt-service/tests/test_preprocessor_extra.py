"""Additional tests for stt-service preprocessor edge cases."""
from __future__ import annotations

import pytest

from app.services.preprocessor import (
    _ffmpeg_available,
    preprocess_audio,
    normalize_volume,
)


def test_extra_ffmpeg_availability_constant():
    """ffmpeg availability flag should be a bool."""
    assert isinstance(_ffmpeg_available, bool)


def test_extra_preprocess_empty_input():
    """Empty input should be returned when ffmpeg is unavailable."""
    result = preprocess_audio(b"")
    if not _ffmpeg_available:
        assert result == b""


def test_extra_normalize_empty_input():
    """Empty input should be returned when ffmpeg is unavailable."""
    result = normalize_volume(b"")
    if not _ffmpeg_available:
        assert result == b""


def test_extra_preprocess_returns_bytes():
    """preprocess_audio should return bytes."""
    result = preprocess_audio(b"some audio data")
    assert isinstance(result, bytes)


def test_extra_normalize_returns_bytes():
    """normalize_volume should return bytes."""
    result = normalize_volume(b"some audio data")
    assert isinstance(result, bytes)


def test_extra_preprocess_custom_sample_rate():
    """preprocess_audio should accept custom sample rate."""
    result = preprocess_audio(b"data", target_sample_rate=8000)
    assert isinstance(result, bytes)


def test_extra_normalize_custom_dbfs():
    """normalize_volume should accept custom dBFS."""
    result = normalize_volume(b"data", target_dbfs=-14.0)
    assert isinstance(result, bytes)


def test_extra_normalize_extreme_dbfs():
    """normalize_volume should handle extreme values."""
    result = normalize_volume(b"data", target_dbfs=-50.0)
    assert isinstance(result, bytes)
    result = normalize_volume(b"data", target_dbfs=0.0)
    assert isinstance(result, bytes)


def test_extra_preprocess_no_panic_on_short_input():
    """preprocess_audio should not panic on short input."""
    for size in [0, 1, 10, 100]:
        result = preprocess_audio(b"x" * size)
        assert isinstance(result, bytes)


def test_extra_normalize_no_panic_on_short_input():
    """normalize_volume should not panic on short input."""
    for size in [0, 1, 10, 100]:
        result = normalize_volume(b"x" * size)
        assert isinstance(result, bytes)
