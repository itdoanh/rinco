"""Tests for lead-scoring NATS consumer."""
import os
import sys
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# Ensure we can import the app modules
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from app.services.nats_consumer import warm_global_model


@pytest.mark.asyncio
async def test_nats_consumer_no_nats_lib(monkeypatch):
    """When nats lib isn't available, consumer should no-op."""
    import builtins
    real_import = builtins.__import__

    def fake_import(name, *args, **kwargs):
        if name == "nats" or name.startswith("nats."):
            raise ImportError("no nats")
        return real_import(name, *args, **kwargs)

    monkeypatch.setattr(builtins, "__import__", fake_import)

    from app.services.nats_consumer import nats_consumer

    # Should not raise
    await nats_consumer(None)


@pytest.mark.asyncio
async def test_warm_global_model_idempotent():
    """warm_global_model should not crash."""
    warm_global_model()
    warm_global_model()


def test_nats_consumer_signature():
    """Test function signature."""
    import inspect
    from app.services.nats_consumer import nats_consumer

    sig = inspect.signature(nats_consumer)
    params = list(sig.parameters.keys())
    assert params[0] == "app"
    assert "subject" in params
    assert "url" in params


def test_nats_consumer_default_subject():
    """Default subject is lead.created."""
    import inspect
    from app.services.nats_consumer import nats_consumer

    sig = inspect.signature(nats_consumer)
    assert sig.parameters["subject"].default == "lead.created"


def test_nats_consumer_url_default_none():
    """URL defaults to None (reads from env)."""
    import inspect
    from app.services.nats_consumer import nats_consumer

    sig = inspect.signature(nats_consumer)
    assert sig.parameters["url"].default is None


def test_warm_global_model_callable():
    """warm_global_model is callable."""
    assert callable(warm_global_model)
