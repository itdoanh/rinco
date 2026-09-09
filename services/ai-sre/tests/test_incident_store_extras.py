"""Extra tests for incident_store — edge cases, filters, concurrent updates."""
from __future__ import annotations

import asyncio
import uuid

import pytest

from app.services import incident_store


@pytest.fixture(autouse=True)
def _reset_store():
    """Reset in-memory store aggressively before AND after each test."""
    # Clear multiple times in case any prior teardown is pending
    for _ in range(3):
        incident_store._incidents.clear()
    yield
    incident_store._incidents.clear()


@pytest.mark.asyncio
async def test_extra_store_create_incident_autogen_id():
    """When no incident_id provided, one is auto-generated (8 chars)."""
    incident = await incident_store.create_incident({"service": "auth", "severity": "high"})
    assert "incident_id" in incident
    assert len(incident["incident_id"]) == 8
    assert incident["status"] == "open"
    assert "created_at" in incident


@pytest.mark.asyncio
async def test_extra_store_create_incident_explicit_id():
    """When incident_id provided, it is preserved."""
    incident = await incident_store.create_incident({
        "incident_id": "INC-001",
        "service": "auth",
    })
    assert incident["incident_id"] == "INC-001"


@pytest.mark.asyncio
async def test_extra_store_get_incident_existing():
    incident = await incident_store.create_incident({"service": "billing"})
    fetched = await incident_store.get_incident(incident["incident_id"])
    assert fetched is not None
    assert fetched["service"] == "billing"


@pytest.mark.asyncio
async def test_extra_store_get_incident_missing():
    fetched = await incident_store.get_incident("nonexistent")
    assert fetched is None


@pytest.mark.asyncio
async def test_extra_store_list_incidents_no_filters():
    for i in range(5):
        await incident_store.create_incident({"service": f"svc-{i}"})
    all_incidents = await incident_store.list_incidents()
    assert len(all_incidents) == 5


@pytest.mark.asyncio
async def test_extra_store_list_incidents_filter_service():
    await incident_store.create_incident({"service": "auth"})
    await incident_store.create_incident({"service": "billing"})
    await incident_store.create_incident({"service": "auth"})
    auth_incidents = await incident_store.list_incidents(service="auth")
    assert len(auth_incidents) == 2
    assert all(i["service"] == "auth" for i in auth_incidents)


@pytest.mark.asyncio
async def test_extra_store_list_incidents_filter_status():
    inc = await incident_store.create_incident({"service": "auth"})
    await incident_store.create_incident({"service": "auth"})
    open_incidents = await incident_store.list_incidents(status="open")
    assert len(open_incidents) == 2
    # Update one to resolved
    await incident_store.update_incident(inc["incident_id"], {"status": "resolved"})
    resolved_incidents = await incident_store.list_incidents(status="resolved")
    assert len(resolved_incidents) == 1


@pytest.mark.asyncio
async def test_extra_store_list_incidents_limit():
    for i in range(10):
        await incident_store.create_incident({"service": "auth"})
    limited = await incident_store.list_incidents(limit=3)
    assert len(limited) == 3


@pytest.mark.asyncio
async def test_extra_store_list_incidents_sorted_descending():
    """Newer incidents should appear first."""
    inc1 = await incident_store.create_incident({"service": "auth"})
    await asyncio.sleep(0.01)  # ensure different timestamp
    inc2 = await incident_store.create_incident({"service": "auth"})
    sorted_list = await incident_store.list_incidents()
    assert sorted_list[0]["incident_id"] == inc2["incident_id"]
    assert sorted_list[-1]["incident_id"] == inc1["incident_id"]


@pytest.mark.asyncio
async def test_extra_store_update_incident_existing():
    inc = await incident_store.create_incident({"service": "auth"})
    updated = await incident_store.update_incident(inc["incident_id"], {"status": "resolved", "resolved_by": "alice"})
    assert updated is not None
    assert updated["status"] == "resolved"
    assert updated["resolved_by"] == "alice"
    # Original fields preserved
    assert updated["service"] == "auth"


@pytest.mark.asyncio
async def test_extra_store_update_incident_missing():
    result = await incident_store.update_incident("nonexistent", {"status": "resolved"})
    assert result is None


@pytest.mark.asyncio
async def test_extra_store_concurrent_creates():
    """Concurrent creates should all succeed and have unique IDs."""
    async def create(i):
        return await incident_store.create_incident({"service": f"svc-{i}"})

    incidents = await asyncio.gather(*[create(i) for i in range(20)])
    ids = [i["incident_id"] for i in incidents]
    assert len(set(ids)) == 20  # all unique
    assert len(await incident_store.list_incidents(limit=100)) == 20


@pytest.mark.asyncio
async def test_extra_store_uuid_format_in_autogen_id():
    """Auto-generated IDs are 8 chars of a UUID (hex + dash)."""
    incident = await incident_store.create_incident({})
    iid = incident["incident_id"]
    assert len(iid) == 8
    # First 8 chars of UUID are hex (no dashes yet)
    for c in iid:
        assert c == "-" or (c.isalnum() and c.islower()) or c.isdigit()


@pytest.mark.asyncio
async def test_extra_store_data_passthrough():
    """Extra fields in input data are preserved on creation."""
    incident = await incident_store.create_incident({
        "service": "auth",
        "severity": "high",
        "custom_field": "custom_value",
        "tags": ["a", "b"],
    })
    assert incident["severity"] == "high"
    assert incident["custom_field"] == "custom_value"
    assert incident["tags"] == ["a", "b"]
