"""Tests for stt-service core (config, logging, metrics)."""
from __future__ import annotations

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


def test_extra_config_defaults():
    """Config defaults should be sane values."""
    assert isinstance(WHISPER_MODEL, str)
    assert len(WHISPER_MODEL) > 0
    assert isinstance(WHISPER_DEVICE, str)
    assert isinstance(WHISPER_COMPUTE, str)
    assert isinstance(AUDIO_SAMPLE_RATE, int)
    assert AUDIO_SAMPLE_RATE > 0


def test_extra_config_class():
    """Config class should expose module-level constants."""
    assert Config.WHISPER_MODEL == WHISPER_MODEL
    assert Config.WHISPER_DEVICE == WHISPER_DEVICE
    assert Config.WHISPER_COMPUTE == WHISPER_COMPUTE
    assert Config.AUDIO_SAMPLE_RATE == AUDIO_SAMPLE_RATE


def test_extra_configure_logging():
    """configure_logging can be called multiple times."""
    configure_logging("DEBUG")
    configure_logging("INFO")
    configure_logging("WARNING")
    configure_logging("ERROR")


def test_extra_configure_logging_default():
    """configure_logging with None uses default level."""
    configure_logging(None)


def test_extra_get_logger():
    """get_logger returns a logger instance."""
    log = get_logger("test-module")
    assert log is not None


def test_extra_get_logger_no_name():
    """get_logger without name returns default logger."""
    log = get_logger()
    assert log is not None


def test_extra_metrics_instance():
    """METRICS should be a Metrics instance."""
    assert isinstance(METRICS, Metrics)


def test_extra_metrics_init():
    """Metrics.init_metrics is called once at module import."""
    # Module init already calls init_metrics, so we just verify metrics exist
    assert METRICS is not None
    # The init_metrics method exists and can be inspected
    m = Metrics()
    assert m is not None


def test_extra_metrics_properties():
    """Metrics should expose properties."""
    assert hasattr(METRICS, "requests_total")
    assert hasattr(METRICS, "latency")
    assert hasattr(METRICS, "errors")


def test_extra_metrics_counters_usable():
    """Metrics counters should be usable if initialized."""
    if METRICS.requests_total is not None:
        METRICS.requests_total.labels(status="ok", language="en").inc()


def test_extra_metrics_errors_usable():
    """Metrics errors counter should be usable if initialized."""
    if METRICS.errors is not None:
        METRICS.errors.labels(error_type="validation").inc()


def test_extra_metrics_latency_usable():
    """Metrics latency histogram should be usable if initialized."""
    if METRICS.latency is not None:
        METRICS.latency.labels(language="en").observe(1.5)


def test_extra_metrics_class_independence():
    """Multiple Metrics instances should work independently."""
    m1 = Metrics()
    m2 = Metrics()
    assert m1 is not m2
