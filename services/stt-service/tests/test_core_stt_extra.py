"""Extra tests for stt-service core module."""
import os
import sys
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))) + "/app")

from app.core import Config, METRICS, Metrics, configure_logging, get_logger


def test_config_defaults():
    assert Config.WHISPER_MODEL != ""
    assert Config.WHISPER_DEVICE != ""
    assert Config.AUDIO_SAMPLE_RATE > 0


def test_config_attributes():
    assert hasattr(Config, "WHISPER_MODEL")
    assert hasattr(Config, "WHISPER_DEVICE")
    assert hasattr(Config, "WHISPER_COMPUTE")
    assert hasattr(Config, "AUDIO_SAMPLE_RATE")


def test_metrics_instance():
    assert METRICS is not None
    assert isinstance(METRICS, Metrics)


def test_metrics_class_init():
    m = Metrics()
    assert m is not None


def test_metrics_init_metrics_call():
    # init_metrics called once at module import; calling again would duplicate
    # Just verify the instance has the attributes (set during import)
    assert METRICS._registry is not None


def test_configure_logging_default():
    configure_logging()


def test_configure_logging_with_level():
    configure_logging("DEBUG")
    configure_logging("INFO")
    configure_logging("WARNING")
    configure_logging("ERROR")


def test_configure_logging_none():
    configure_logging(None)


def test_get_logger():
    log = get_logger("test")
    assert log is not None


def test_get_logger_no_name():
    log = get_logger()
    assert log is not None
