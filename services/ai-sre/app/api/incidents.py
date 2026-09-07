"""Incidents API: list and retrieve incidents."""
from __future__ import annotations

from fastapi import APIRouter, HTTPException, Query
from pydantic import BaseModel
from typing import Any

from app.core import get_logger, METRICS
from app.schemas import IncidentCreate, Incident
from app.services import incident_store

router = APIRouter(prefix="/v1/incidents", tags=["incidents"])
logger = get_logger("api.incidents")


class IncidentCreateRequest(BaseModel):
    """Request to create a new incident."""
    service: str
    trace_id: str | None = None
    error: str
    stack: str | None = None
    severity: str = "P2"
    sentry_url: str | None = None
    files: list[dict[str, Any]] = []


class IncidentResponse(BaseModel):
    """Full incident response."""
    incident_id: str
    service: str
    trace_id: str | None = None
    error: str
    stack: str | None = None
    severity: str
    sentry_url: str | None = None
    files: list[dict[str, Any]]
    status: str
    created_at: str
    root_cause: str | None = None
    pr_url: str | None = None


@router.post("", response_model=IncidentResponse, status_code=201)
async def create_incident(req: IncidentCreateRequest) -> IncidentResponse:
    """
    Register a new incident in the in-memory store.

    Incidents are stored with a generated ID and timestamp. Use the
    `/v1/analyze` endpoint to trigger RCA analysis.
    """
    data = req.model_dump()
    incident = await incident_store.create_incident(data)
    return IncidentResponse(**incident)


@router.get("", response_model=list[IncidentResponse])
async def list_incidents(
    service: str | None = Query(None, description="Filter by service name"),
    status: str | None = Query(None, description="Filter by status (open/resolved)"),
    limit: int = Query(50, ge=1, le=200, description="Max results to return"),
) -> list[IncidentResponse]:
    """
    List all incidents, optionally filtered by service and status.

    Results are sorted by creation time descending.
    """
    incidents = await incident_store.list_incidents(service=service, status=status, limit=limit)
    return [IncidentResponse(**i) for i in incidents]


@router.get("/{incident_id}", response_model=IncidentResponse)
async def get_incident(incident_id: str) -> IncidentResponse:
    """
    Retrieve a single incident by its ID.
    """
    incident = await incident_store.get_incident(incident_id)
    if not incident:
        raise HTTPException(status_code=404, detail=f"Incident {incident_id} not found")
    return IncidentResponse(**incident)


@router.patch("/{incident_id}")
async def update_incident(
    incident_id: str,
    updates: dict[str, Any],
) -> IncidentResponse:
    """
    Update fields on an existing incident (e.g. mark as resolved).
    """
    updated = await incident_store.update_incident(incident_id, updates)
    if not updated:
        raise HTTPException(status_code=404, detail=f"Incident {incident_id} not found")
    return IncidentResponse(**updated)
