"""Tests for AI-SRE /v1/chat endpoint."""
import pytest
from httpx import AsyncClient, ASGITransport
from unittest.mock import patch, AsyncMock

from app.main import app
from app.services import incident_store


@pytest.fixture(autouse=True)
def clear_incidents():
    incident_store._incidents.clear()
    yield
    incident_store._incidents.clear()


@pytest.mark.asyncio
async def test_chat_missing_message():
    """POST /v1/chat without message returns 400."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.post("/v1/chat", json={"message": ""})
    assert resp.status_code == 400


@pytest.mark.asyncio
async def test_chat_simple_success():
    """POST /v1/chat with valid message returns reply."""
    fake_reply = "Yes, the system is healthy."

    class FakeResponse:
        def raise_for_status(self): pass
        def json(self):
            return {"choices": [{"message": {"content": fake_reply}}]}

    class FakeAsyncClient:
        async def __aenter__(self): return self
        async def __aexit__(self, *args): return False
        async def post(self, url, json):
            return FakeResponse()

    with patch("app.api.chat.httpx.AsyncClient", return_value=FakeAsyncClient()):
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.post("/v1/chat", json={"message": "is system healthy?"})
    assert resp.status_code == 200
    data = resp.json()
    assert data["reply"] == fake_reply
    assert data["sources"] == []


@pytest.mark.asyncio
async def test_chat_with_service_context(monkeypatch):
    """POST /v1/chat with service includes service context."""
    fake_reply = "Looking at auth service metrics..."

    class FakeResponse:
        def raise_for_status(self): pass
        def json(self):
            return {"choices": [{"message": {"content": fake_reply}}]}

    class FakeAsyncClient:
        async def __aenter__(self): return self
        async def __aexit__(self, *args): return False
        async def post(self, url, json):
            return FakeResponse()

    # Patch the correlation function used by the chat endpoint
    async def fake_correlate(service, trace_id=None, time_window="1h"):
        return {"summary": "No anomalies", "logs": [], "traces": [], "metrics": {}}

    monkeypatch.setattr(
        "app.api.chat.incident_correlator.correlate_incident",
        fake_correlate,
    )

    with patch("app.api.chat.httpx.AsyncClient", return_value=FakeAsyncClient()):
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.post(
                "/v1/chat",
                json={"message": "what's happening with auth?", "service": "auth"},
            )
    assert resp.status_code == 200
    data = resp.json()
    assert data["reply"] == fake_reply
    # sources should include service tag
    assert "service:auth" in data["sources"]


@pytest.mark.asyncio
async def test_chat_with_incident_context(monkeypatch):
    """POST /v1/chat with incident_id includes incident context."""
    fake_reply = "Investigating incident..."

    class FakeResponse:
        def raise_for_status(self): pass
        def json(self):
            return {"choices": [{"message": {"content": fake_reply}}]}

    class FakeAsyncClient:
        async def __aenter__(self): return self
        async def __aexit__(self, *args): return False
        async def post(self, url, json):
            return FakeResponse()

    async def fake_correlate(service, trace_id=None, time_window="1h"):
        return {"summary": "Spike in errors", "logs": [], "traces": [], "metrics": {}}

    monkeypatch.setattr(
        "app.api.chat.incident_correlator.correlate_incident",
        fake_correlate,
    )

    incident_store._incidents["test-inc"] = {
        "incident_id": "test-inc",
        "service": "billing",
        "error": "Payment declined",
        "status": "open",
        "created_at": "2026-01-01T00:00:00Z",
        "files": [],
        "severity": "P1",
    }

    with patch("app.api.chat.httpx.AsyncClient", return_value=FakeAsyncClient()):
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.post(
                "/v1/chat",
                json={"message": "analyze", "incident_id": "test-inc"},
            )
    assert resp.status_code == 200
    data = resp.json()
    assert data["reply"] == fake_reply
    assert "incident:test-inc" in data["sources"]


@pytest.mark.asyncio
async def test_chat_with_unknown_incident(monkeypatch):
    """POST /v1/chat with non-existent incident_id just skips context."""
    fake_reply = "ok"

    class FakeResponse:
        def raise_for_status(self): pass
        def json(self):
            return {"choices": [{"message": {"content": fake_reply}}]}

    class FakeAsyncClient:
        async def __aenter__(self): return self
        async def __aexit__(self, *args): return False
        async def post(self, url, json):
            return FakeResponse()

    with patch("app.api.chat.httpx.AsyncClient", return_value=FakeAsyncClient()):
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.post(
                "/v1/chat",
                json={"message": "test", "incident_id": "missing-id"},
            )
    assert resp.status_code == 200
    assert resp.json()["sources"] == []


@pytest.mark.asyncio
async def test_chat_llm_503(monkeypatch):
    """POST /v1/chat with HTTP error returns 503."""
    import httpx

    class FakeResponse:
        status_code = 500
        def raise_for_status(self):
            raise httpx.HTTPStatusError("err", request=None, response=self)

    class FakeAsyncClient:
        async def __aenter__(self): return self
        async def __aexit__(self, *args): return False
        async def post(self, url, json):
            return FakeResponse()

    with patch("app.api.chat.httpx.AsyncClient", return_value=FakeAsyncClient()):
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.post("/v1/chat", json={"message": "test"})
    assert resp.status_code == 503


@pytest.mark.asyncio
async def test_chat_llm_general_error():
    """POST /v1/chat with general exception returns 500."""

    class FakeAsyncClient:
        async def __aenter__(self): return self
        async def __aexit__(self, *args): return False
        async def post(self, url, json):
            raise ConnectionError("network fail")

    with patch("app.api.chat.httpx.AsyncClient", return_value=FakeAsyncClient()):
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.post("/v1/chat", json={"message": "test"})
    assert resp.status_code == 500
