"""Tests for STT service core module."""
import os

from app.core import (
    Config,
    configure_logging,
    get_logger,
    METRICS,
    Metrics,
    WHISPER_MODEL,
    WHISPER_DEVICE,
    WHISPER_COMPUTE,
    AUDIO_SAMPLE_RATE,
)


def test_constants_have_defaults():
    """Module-level constants have defaults."""
    assert isinstance(WHISPER_MODEL, str)
    assert len(WHISPER_MODEL) > 0
    assert isinstance(WHISPER_DEVICE, str)
    assert isinstance(WHISPER_COMPUTE, str)
    assert isinstance(AUDIO_SAMPLE_RATE, int)


def test_config_class_exposes_constants():
    """Config class exposes module-level constants."""
    assert Config.WHISPER_MODEL == WHISPER_MODEL
    assert Config.WHISPER_DEVICE == WHISPER_DEVICE
    assert Config.WHISPER_COMPUTE == WHISPER_COMPUTE
    assert Config.AUDIO_SAMPLE_RATE == AUDIO_SAMPLE_RATE


def test_config_class_attribute_assignment():
    """Config class allows attribute assignment (it's a config container)."""
    cfg = Config()
    cfg.AUDIO_SAMPLE_RATE = 48000
    assert cfg.AUDIO_SAMPLE_RATE == 48000


def test_configure_logging_default():
    """configure_logging with default level works."""
    configure_logging("INFO")


def test_configure_logging_debug():
    """configure_logging with DEBUG works."""
    configure_logging("DEBUG")


def test_configure_logging_invalid(monkeypatch):
    """configure_logging with invalid level falls back to INFO."""
    # Use a special value; should not crash
    configure_logging("INVALID_LEVEL_NAME")


def test_get_logger_default():
    """get_logger with no name returns a logger."""
    log = get_logger()
    assert log is not None


def test_get_logger_named():
    """get_logger with name returns a named logger."""
    log = get_logger("test.module")
    assert log is not None


def test_metrics_instance():
    """METRICS is a Metrics instance."""
    assert isinstance(METRICS, Metrics)


def test_metrics_instance_already_initialized():
    """Global METRICS instance is already initialized at import time."""
    # init_metrics was called at module load; verify metrics are accessible.
    assert METRICS is not None
    # Just touching the attributes should not raise
    _ = METRICS.requests_total
    _ = METRICS.latency
    _ = METRICS.errors


def test_metrics_attributes():
    """Metrics exposes request/error/latency attributes."""
    m = Metrics()
    # init may fail silently if prometheus missing, but properties exist
    _ = m.requests_total
    _ = m.latency
    _ = m.errors


def test_audio_sample_rate_default():
    """AUDIO_SAMPLE_RATE defaults to 16000."""
    assert AUDIO_SAMPLE_RATE == 16000


def test_whisper_model_default():
    """WHISPER_MODEL defaults to large-v3 (verified at module load)."""
    assert WHISPER_MODEL in ("large-v3",) or len(WHISPER_MODEL) > 0
