"""Pydantic schemas for the recording-service API."""

from __future__ import annotations

from typing import List, Optional

from pydantic import BaseModel, Field


class StartRecordingRequest(BaseModel):
    """Payload of ``POST /v1/recordings/start``."""

    room_id: str = Field(..., min_length=1, description="WebRTC room identifier")
    user_id: str = Field(..., min_length=1, description="Initiating user")
    layout: Optional[str] = Field("grid", description="grid | speaker | presentation")
    quality: Optional[str] = Field("hd", description="sd | hd | fhd")
    tenant_id: Optional[str] = None


class RecordingMetadata(BaseModel):
    """Public representation of a recording."""

    id: str
    room_id: str
    user_id: str
    tenant_id: Optional[str] = None
    status: str  # started | processing | ready | failed
    started_at: float
    ended_at: Optional[float] = None
    duration_seconds: Optional[float] = None
    size_bytes: Optional[int] = None
    storage_path: Optional[str] = None
    layout: str = "grid"
    quality: str = "hd"
    participants: List[str] = []
    download_url: Optional[str] = None
    transcript_url: Optional[str] = None


class TranscriptPayload(BaseModel):
    """Body of ``POST /v1/recordings/{id}/transcript``."""

    segments: List[dict]
    language: Optional[str] = "en"


class HealthResponse(BaseModel):
    status: str
    bucket: str
    active_recordings: int
    version: str = "0.3.0"
