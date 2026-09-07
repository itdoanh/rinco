"""Schemas package."""
from __future__ import annotations

from pydantic import BaseModel, Field
from typing import Any


class IncidentCreate(BaseModel):
    """Schema for creating a new incident."""
    service: str = Field(..., description="Affected service name")
    trace_id: str | None = Field(None, description="Optional trace ID from Jaeger")
    error: str = Field(..., description="Error message or exception text")
    stack: str | None = Field(None, description="Stack trace string")
    severity: str = Field("P2", description="Severity level (P0-P4)")
    sentry_url: str | None = Field(None, description="Link to Sentry event")
    files: list[dict[str, Any]] = Field(default_factory=list, description="File locations [{path, line}]")


class Incident(IncidentCreate):
    """Full incident with generated ID and timestamps."""
    incident_id: str
    status: str = "open"
    created_at: str = ""


class RCAResult(BaseModel):
    """Root Cause Analysis result."""
    incident_id: str
    root_cause: str
    why: str
    suggested_fix: str
    prevention: str
    confidence: float
    file: str | None = None
    line: int | None = None
    pr_url: str | None = None
    runbook_url: str | None = None
    notify_sent: bool = False


class AnalyzeRequest(BaseModel):
    """Request body for the /v1/analyze endpoint."""
    service: str
    trace_id: str | None = None
    error: str
    stack: str | None = None
    severity: str = "P2"
    sentry_url: str | None = None
    files: list[dict[str, Any]] = Field(default_factory=list)
    time_window: str = Field("1h", description="Time window for log/trace queries")
    auto_create_pr: bool = Field(True, description="Whether to auto-create a GitHub PR")
    auto_create_runbook: bool = Field(True, description="Whether to auto-create a Confluence runbook")


class AnalyzeResponse(BaseModel):
    """Response from /v1/analyze."""
    incident_id: str
    rca: RCAResult
    correlation: dict[str, Any] = Field(default_factory=dict, description="Raw correlation data")
    pr_url: str | None = None
    runbook_url: str | None = None


class HotfixGenerateRequest(BaseModel):
    """Request for hotfix generation."""
    error: str
    stack: str | None = None
    source: str = ""
    context: str = ""


class HotfixGenerateResponse(BaseModel):
    """Response from hotfix generation."""
    diff: str
    file: str
    line: int
    confidence: float
    root_cause: str = ""


class RunbookCreateRequest(BaseModel):
    """Request to create a runbook."""
    title: str
    content_html: str
    space: str = "RINCO"


class RunbookCreateResponse(BaseModel):
    """Response from runbook creation."""
    url: str
    page_id: str | None = None


class ChatRequest(BaseModel):
    """Simple chat request with optional incident context."""
    message: str
    incident_id: str | None = None
    service: str | None = None


class ChatResponse(BaseModel):
    """Chat response."""
    reply: str
    sources: list[str] = Field(default_factory=list)


class CorrelationResponse(BaseModel):
    """Response from the correlation engine."""
    logs: list[dict[str, Any]] = Field(default_factory=list)
    traces: list[dict[str, Any]] = Field(default_factory=list)
    metrics: dict[str, Any] = Field(default_factory=dict)
    summary: str = ""
