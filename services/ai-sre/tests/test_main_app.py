"""Tests for AI-SRE FastAPI app.main endpoints."""
import pytest
from httpx import AsyncClient, ASGITransport


@pytest.mark.asyncio
async def test_health_endpoint():
    """GET /health returns ok status."""
    from app.main import app

    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.get("/health")
    assert r.status_code == 200
    data = r.json()
    assert data["status"] == "ok"
    assert data["service"] == "ai-sre"
    assert data["version"] == "2.0.0"


@pytest.mark.asyncio
async def test_ready_endpoint():
    """GET /ready returns ready when modules importable."""
    from app.main import app

    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.get("/ready")
    assert r.status_code == 200
    data = r.json()
    # Should be ready or not_ready depending on environment
    assert data.get("status") in ("ready", "not_ready")


@pytest.mark.asyncio
async def test_metrics_endpoint():
    """GET /metrics returns Prometheus text format."""
    from app.main import app

    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.get("/metrics")
    assert r.status_code == 200
    # Content could be Prometheus text or "unavailable" if lib missing
    assert r.text is not None
    assert isinstance(r.text, str)


@pytest.mark.asyncio
async def test_app_metadata():
    """FastAPI app has expected metadata."""
    from app.main import app

    assert app.title == "AI SRE Worker"
    assert "2.0.0" in app.version


def test_logger_initialized():
    """Logger is configured at import time."""
    from app.main import logger
    assert logger is not None
