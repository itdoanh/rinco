"""Tests for AI-SRE incident_store and /v1/incidents endpoints."""
import pytest
from httpx import AsyncClient, ASGITransport

from app.main import app
from app.services import incident_store


@pytest.fixture(autouse=True)
def clear_incidents():
    """Reset the in-memory incident store between tests (sync)."""
    incident_store._incidents.clear()
    yield
    incident_store._incidents.clear()


@pytest.mark.asyncio
async def test_create_incident_endpoint_success():
    """POST /v1/incidents creates an incident."""
    payload = {
        "service": "auth",
        "error": "NullPointerException",
        "severity": "P1",
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.post("/v1/incidents", json=payload)
    assert resp.status_code == 201, resp.text
    data = resp.json()
    assert data["service"] == "auth"
    assert data["error"] == "NullPointerException"
    assert data["severity"] == "P1"
    assert data["status"] == "open"
    assert "incident_id" in data
    assert "created_at" in data


@pytest.mark.asyncio
async def test_create_incident_minimal():
    """POST /v1/incidents with minimal payload."""
    payload = {"service": "billing", "error": "Failed"}
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.post("/v1/incidents", json=payload)
    assert resp.status_code == 201
    data = resp.json()
    assert data["severity"] == "P2"  # default
    assert data["status"] == "open"


@pytest.mark.asyncio
async def test_list_incidents_empty():
    """GET /v1/incidents on empty store."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.get("/v1/incidents")
    assert resp.status_code == 200
    assert resp.json() == []


@pytest.mark.asyncio
async def test_list_incidents_filtered_by_service():
    """GET /v1/incidents?service=auth filters correctly."""
    incident_store._incidents["i1"] = {
        "incident_id": "i1",
        "service": "auth",
        "error": "x",
        "status": "open",
        "created_at": "2026-01-01T00:00:00Z",
        "files": [],
        "severity": "P2",
    }
    incident_store._incidents["i2"] = {
        "incident_id": "i2",
        "service": "billing",
        "error": "y",
        "status": "open",
        "created_at": "2026-01-02T00:00:00Z",
        "files": [],
        "severity": "P2",
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.get("/v1/incidents", params={"service": "auth"})
    assert resp.status_code == 200
    data = resp.json()
    assert len(data) == 1
    assert data[0]["service"] == "auth"


@pytest.mark.asyncio
async def test_list_incidents_filtered_by_status():
    """GET /v1/incidents?status=resolved."""
    incident_store._incidents["i1"] = {
        "incident_id": "i1",
        "service": "auth",
        "error": "x",
        "status": "open",
        "created_at": "2026-01-01T00:00:00Z",
        "files": [],
        "severity": "P2",
    }
    incident_store._incidents["i2"] = {
        "incident_id": "i2",
        "service": "auth",
        "error": "y",
        "status": "resolved",
        "created_at": "2026-01-02T00:00:00Z",
        "files": [],
        "severity": "P2",
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.get("/v1/incidents", params={"status": "resolved"})
    assert resp.status_code == 200
    data = resp.json()
    assert len(data) == 1
    assert data[0]["status"] == "resolved"


@pytest.mark.asyncio
async def test_list_incidents_limit_capped():
    """GET /v1/incidents respects limit param."""
    for i in range(5):
        incident_store._incidents[f"i{i}"] = {
            "incident_id": f"i{i}",
            "service": "x",
            "error": "e",
            "status": "open",
            "created_at": f"2026-01-0{i + 1}T00:00:00Z",
            "files": [],
            "severity": "P2",
        }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.get("/v1/incidents", params={"limit": 2})
    assert resp.status_code == 200
    data = resp.json()
    assert len(data) == 2


@pytest.mark.asyncio
async def test_list_incidents_sorted_desc():
    """GET /v1/incidents sorts by created_at desc."""
    incident_store._incidents["old"] = {
        "incident_id": "old",
        "service": "x",
        "error": "e",
        "status": "open",
        "created_at": "2026-01-01T00:00:00Z",
        "files": [],
        "severity": "P2",
    }
    incident_store._incidents["new"] = {
        "incident_id": "new",
        "service": "x",
        "error": "e",
        "status": "open",
        "created_at": "2026-02-01T00:00:00Z",
        "files": [],
        "severity": "P2",
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.get("/v1/incidents")
    data = resp.json()
    assert data[0]["incident_id"] == "new"
    assert data[1]["incident_id"] == "old"


@pytest.mark.asyncio
async def test_get_incident_found():
    """GET /v1/incidents/{id} returns incident."""
    incident_store._incidents["myid"] = {
        "incident_id": "myid",
        "service": "auth",
        "error": "boom",
        "status": "open",
        "created_at": "2026-01-01T00:00:00Z",
        "files": [],
        "severity": "P1",
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.get("/v1/incidents/myid")
    assert resp.status_code == 200
    assert resp.json()["error"] == "boom"


@pytest.mark.asyncio
async def test_get_incident_not_found():
    """GET /v1/incidents/{id} 404 when missing."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.get("/v1/incidents/missing-id")
    assert resp.status_code == 404


@pytest.mark.asyncio
async def test_update_incident_resolve():
    """PATCH /v1/incidents/{id} marks resolved."""
    incident_store._incidents["u1"] = {
        "incident_id": "u1",
        "service": "auth",
        "error": "x",
        "status": "open",
        "created_at": "2026-01-01T00:00:00Z",
        "files": [],
        "severity": "P2",
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.patch("/v1/incidents/u1", json={"status": "resolved"})
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "resolved"


@pytest.mark.asyncio
async def test_update_incident_404():
    """PATCH /v1/incidents/{id} 404 when missing."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.patch("/v1/incidents/nope", json={"status": "resolved"})
    assert resp.status_code == 404


@pytest.mark.asyncio
async def test_incident_store_direct_create():
    """incident_store.create_incident adds generated incident_id."""
    result = await incident_store.create_incident({"service": "svc", "error": "e"})
    assert "incident_id" in result
    assert result["status"] == "open"


@pytest.mark.asyncio
async def test_incident_store_get_missing():
    """incident_store.get_incident returns None for missing."""
    result = await incident_store.get_incident("nope")
    assert result is None


@pytest.mark.asyncio
async def test_incident_store_update_missing():
    """incident_store.update_incident returns None for missing."""
    result = await incident_store.update_incident("nope", {"status": "resolved"})
    assert result is None
