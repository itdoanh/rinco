"""Extra tests for lead-scoring config."""
import os
import sys
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))) + "/app")

from app.core.config import Settings, env_or, get_settings


def test_settings_defaults():
    s = Settings()
    assert s.service_name == "lead-scoring"
    assert s.version != ""
    assert s.http_addr != ""


def test_settings_fields():
    s = Settings()
    assert hasattr(s, "model_dir")
    assert hasattr(s, "default_model_version")
    assert hasattr(s, "max_batch_size")
    assert hasattr(s, "allow_public_train")


def test_get_settings_cached():
    s1 = get_settings()
    s2 = get_settings()
    assert s1 is s2


def test_env_or_set():
    os.environ["LEAD_TEST_VAR"] = "value"
    try:
        assert env_or("LEAD_TEST_VAR", "default") == "value"
    finally:
        del os.environ["LEAD_TEST_VAR"]


def test_env_or_default():
    if "LEAD_TEST_MISSING" in os.environ:
        del os.environ["LEAD_TEST_MISSING"]
    assert env_or("LEAD_TEST_MISSING", "fallback") == "fallback"


def test_settings_custom_url():
    s = Settings(postgres_url="postgres://test")
    assert s.postgres_url == "postgres://test"
