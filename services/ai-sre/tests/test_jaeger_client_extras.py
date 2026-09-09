"""Extra tests for AI-SRE Jaeger client helpers (lookback parsing, etc.)."""
from __future__ import annotations

import os
from unittest.mock import AsyncMock, patch

import pytest


@pytest.mark.asyncio
async def test_get_trace_empty_id_returns_empty():
    """An empty trace_id should not make any HTTP call."""
    from app.services.jaeger_client import get_trace
    result = await get_trace("")
    assert result == {}


@pytest.mark.asyncio
async def test_get_trace_success(monkeypatch):
    """Successful trace fetch returns the response data."""
    from app.services import jaeger_client

    fake_response = AsyncMock()
    fake_response.json = lambda: {"data": [{"traceID": "abc"}], "total": 1}
    fake_response.raise_for_status = lambda: None

    fake_client = AsyncMock()
    fake_client.__aenter__ = AsyncMock(return_value=fake_client)
    fake_client.__aexit__ = AsyncMock(return_value=None)
    fake_client.get = AsyncMock(return_value=fake_response)

    monkeypatch.setattr("httpx.AsyncClient", lambda *a, **kw: fake_client)

    result = await jaeger_client.get_trace("abc")
    assert result["total"] == 1


@pytest.mark.asyncio
async def test_get_trace_http_error_returns_empty(monkeypatch):
    """HTTP error status returns empty dict and is caught gracefully."""
    import httpx
    from app.services import jaeger_client

    fake_response = AsyncMock()
    fake_response.raise_for_status = lambda: (_ for _ in ()).throw(
        httpx.HTTPStatusError("not found", request=AsyncMock(), response=AsyncMock())
    )

    fake_client = AsyncMock()
    fake_client.__aenter__ = AsyncMock(return_value=fake_client)
    fake_client.__aexit__ = AsyncMock(return_value=None)
    fake_client.get = AsyncMock(return_value=fake_response)

    monkeypatch.setattr("httpx.AsyncClient", lambda *a, **kw: fake_client)

    result = await jaeger_client.get_trace("missing")
    assert result == {}


@pytest.mark.asyncio
async def test_search_traces_limit_capped(monkeypatch):
    """search_traces must cap limit at 200."""
    from app.services import jaeger_client

    captured_params = {}

    async def fake_get(*args, **kwargs):
        captured_params.update(kwargs.get("params", {}))
        resp = AsyncMock()
        resp.json = lambda: {"data": []}
        resp.raise_for_status = lambda: None
        return resp

    fake_client = AsyncMock()
    fake_client.get = fake_get
    fake_client.__aenter__ = AsyncMock(return_value=fake_client)
    fake_client.__aexit__ = AsyncMock(return_value=None)

    monkeypatch.setattr("httpx.AsyncClient", lambda *a, **kw: fake_client)
    await jaeger_client.search_traces("svc", lookback="1h", limit=1000)
    assert captured_params.get("limit") == 200


@pytest.mark.asyncio
async def test_search_traces_default_lookback(monkeypatch):
    """Unknown lookback string should fall back to 1h (3600s)."""
    from app.services import jaeger_client

    fake_client = AsyncMock()
    fake_response = AsyncMock()
    fake_response.json = lambda: {"data": []}
    fake_response.raise_for_status = lambda: None
    fake_client.get = AsyncMock(return_value=fake_response)
    fake_client.__aenter__ = AsyncMock(return_value=fake_client)
    fake_client.__aexit__ = AsyncMock(return_value=None)

    monkeypatch.setattr("httpx.AsyncClient", lambda *a, **kw: fake_client)
    result = await jaeger_client.search_traces("svc", lookback="bogus")
    assert isinstance(result, list)


def test_jaeger_url_from_env(monkeypatch):
    """JAEGER_URL env var is respected."""
    monkeypatch.setenv("JAEGER_URL", "http://custom:1234")
    # Need to reload the module to pick up the new env var
    import importlib
    from app.services import jaeger_client
    importlib.reload(jaeger_client)
    assert "custom:1234" in jaeger_client.JAEGER_URL
