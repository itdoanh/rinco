"""Tests for incident_store module."""
from __future__ import annotations

import asyncio

import pytest

from app.services import incident_store


@pytest.mark.asyncio
async def test_create_incident():
    """Create an incident with auto-generated ID."""
    incident = await incident_store.create_incident({"service": "api", "title": "Outage"})
    assert incident["service"] == "api"
    assert incident["title"] == "Outage"
    assert incident["status"] == "open"
    assert "incident_id" in incident
    assert "created_at" in incident


@pytest.mark.asyncio
async def test_create_incident_with_custom_id():
    """Create an incident with caller-provided ID."""
    incident = await incident_store.create_incident(
        {"incident_id": "CUSTOM-1", "service": "api"}
    )
    assert incident["incident_id"] == "CUSTOM-1"


@pytest.mark.asyncio
async def test_create_incident_truncates_uuid():
    """Auto-generated IDs are 8 chars (truncated UUID)."""
    incident = await incident_store.create_incident({})
    assert len(incident["incident_id"]) == 8


@pytest.mark.asyncio
async def test_get_incident_exists():
    """Retrieve an existing incident."""
    created = await incident_store.create_incident({"service": "api"})
    got = await incident_store.get_incident(created["incident_id"])
    assert got is not None
    assert got["incident_id"] == created["incident_id"]


@pytest.mark.asyncio
async def test_get_incident_missing():
    """Return None for non-existent incident."""
    got = await incident_store.get_incident("does-not-exist")
    assert got is None


@pytest.mark.asyncio
async def test_list_incidents_empty():
    """List all incidents (filter not applied)."""
    incidents = await incident_store.list_incidents()
    assert isinstance(incidents, list)


@pytest.mark.asyncio
async def test_list_incidents_filter_by_service():
    """Filter incidents by service."""
    await incident_store.create_incident({"service": "api"})
    await incident_store.create_incident({"service": "db"})

    api_incidents = await incident_store.list_incidents(service="api")
    for inc in api_incidents:
        assert inc.get("service") == "api"


@pytest.mark.asyncio
async def test_list_incidents_filter_by_status():
    """Filter incidents by status."""
    inc1 = await incident_store.create_incident({"service": "api"})

    open_incidents = await incident_store.list_incidents(status="open")
    assert any(i["incident_id"] == inc1["incident_id"] for i in open_incidents)


@pytest.mark.asyncio
async def test_list_incidents_limit():
    """Limit number of incidents returned."""
    for i in range(5):
        await incident_store.create_incident({"service": f"svc-{i}"})

    incidents = await incident_store.list_incidents(limit=3)
    assert len(incidents) <= 3


@pytest.mark.asyncio
async def test_list_incidents_sorted_by_created_at():
    """Incidents list is sorted (newest first by created_at)."""
    # Add an explicit small delay so created_at differs.
    await incident_store.create_incident({"service": "svc-1"})
    await asyncio.sleep(0.01)
    await incident_store.create_incident({"service": "svc-2"})

    incidents = await incident_store.list_incidents(limit=10)
    assert len(incidents) >= 2

    # First incident should be svc-2 (newer).
    assert incidents[0]["service"] == "svc-2"


@pytest.mark.asyncio
async def test_update_incident_exists():
    """Update an existing incident."""
    created = await incident_store.create_incident({"service": "api"})
    updated = await incident_store.update_incident(
        created["incident_id"], {"status": "resolved"}
    )
    assert updated is not None
    assert updated["status"] == "resolved"


@pytest.mark.asyncio
async def test_update_incident_missing():
    """Update non-existent incident returns None."""
    updated = await incident_store.update_incident("does-not-exist", {"status": "closed"})
    assert updated is None


@pytest.mark.asyncio
async def test_update_incident_multiple_fields():
    """Update multiple fields at once."""
    created = await incident_store.create_incident({"service": "api"})
    updated = await incident_store.update_incident(
        created["incident_id"],
        {"status": "resolved", "resolution": "fixed"},
    )
    assert updated["status"] == "resolved"
    assert updated["resolution"] == "fixed"


@pytest.mark.asyncio
async def test_concurrent_create():
    """Concurrent creates don't lose data."""
    ids = []
    tasks = [incident_store.create_incident({"service": f"svc-{i}"}) for i in range(20)]
    results = await asyncio.gather(*tasks)
    for r in results:
        ids.append(r["incident_id"])
    assert len(set(ids)) == 20  # All unique


@pytest.mark.asyncio
async def test_concurrent_update():
    """Concurrent updates don't lose data."""
    created = await incident_store.create_incident({"service": "api"})

    async def update(status):
        return await incident_store.update_incident(
            created["incident_id"], {"status": status}
        )

    results = await asyncio.gather(update("s1"), update("s2"), update("s3"))
    final = await incident_store.get_incident(created["incident_id"])
    assert final is not None
    # Last writer wins; but all should return without error
    assert all(r is not None for r in results)


@pytest.mark.asyncio
async def test_incident_status_default_open():
    """New incidents default to status=open."""
    incident = await incident_store.create_incident({})
    assert incident["status"] == "open"


@pytest.mark.asyncio
async def test_list_incidents_no_filters_returns_all():
    """When no filters are given, all incidents are returned."""
    before = len(await incident_store.list_incidents(limit=1000))
    await incident_store.create_incident({"service": "new-svc"})
    after = len(await incident_store.list_incidents(limit=1000))
    assert after == before + 1
