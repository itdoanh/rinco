"""In-memory incident store with thread-safe locking."""
from __future__ import annotations

import asyncio
import uuid
from datetime import datetime, timezone
from typing import Any

from app.core import get_logger

logger = get_logger("incident_store")

# In-memory incident storage
_incidents: dict[str, dict[str, Any]] = {}
_incident_lock = asyncio.Lock()


async def create_incident(data: dict[str, Any]) -> dict[str, Any]:
    """Create a new incident and store it in memory."""
    incident_id = data.get("incident_id") or str(uuid.uuid4())[:8]
    incident = {
        **data,
        "incident_id": incident_id,
        "status": "open",
        "created_at": datetime.now(timezone.utc).isoformat(),
    }
    async with _incident_lock:
        _incidents[incident_id] = incident
    logger.info("incident_created", incident_id=incident_id)
    return incident


async def get_incident(incident_id: str) -> dict[str, Any] | None:
    """Retrieve an incident by ID."""
    async with _incident_lock:
        return _incidents.get(incident_id)


async def list_incidents(
    service: str | None = None,
    status: str | None = None,
    limit: int = 50,
) -> list[dict[str, Any]]:
    """List incidents with optional filters."""
    async with _incident_lock:
        results = list(_incidents.values())

    if service:
        results = [i for i in results if i.get("service") == service]
    if status:
        results = [i for i in results if i.get("status") == status]

    results.sort(key=lambda i: i.get("created_at", ""), reverse=True)
    return results[:limit]


async def update_incident(incident_id: str, updates: dict[str, Any]) -> dict[str, Any] | None:
    """Update an existing incident."""
    async with _incident_lock:
        if incident_id not in _incidents:
            return None
        _incidents[incident_id].update(updates)
        updated = _incidents[incident_id]
    logger.info("incident_updated", incident_id=incident_id)
    return updated
