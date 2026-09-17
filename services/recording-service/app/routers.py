"""FastAPI routers for the recording-service API surface."""

from __future__ import annotations

import logging
from typing import List, Optional

from fastapi import APIRouter, BackgroundTasks, HTTPException, Query

from .models import HealthResponse, RecordingMetadata, StartRecordingRequest, TranscriptPayload
from .services import RecordingService

logger = logging.getLogger(__name__)


def build_router(service: RecordingService) -> APIRouter:
    """Build the APIRouter wired to the supplied service."""

    router = APIRouter()

    @router.get("/health", response_model=HealthResponse)
    def health() -> HealthResponse:
        return HealthResponse(
            status="healthy",
            bucket=service.storage.bucket,
            active_recordings=len(service.processes),
        )

    @router.post("/v1/recordings/start", response_model=RecordingMetadata, status_code=201)
    def start_recording(req: StartRecordingRequest) -> RecordingMetadata:
        return service.start(req)

    @router.post("/v1/recordings/{recording_id}/stop", response_model=RecordingMetadata)
    def stop_recording(recording_id: str) -> RecordingMetadata:
        try:
            return service.stop(recording_id)
        except KeyError:
            raise HTTPException(status_code=404, detail="recording not found")
        except Exception as exc:  # noqa: BLE001
            logger.exception("stop failed")
            raise HTTPException(status_code=500, detail=str(exc))

    @router.get("/v1/recordings/{recording_id}", response_model=RecordingMetadata)
    def get_recording(recording_id: str) -> RecordingMetadata:
        meta = service.get(recording_id)
        if meta is None:
            raise HTTPException(status_code=404, detail="recording not found")
        return meta

    @router.get("/v1/recordings", response_model=List[RecordingMetadata])
    def list_recordings(
        room_id: Optional[str] = None,
        tenant_id: Optional[str] = None,
        limit: int = Query(50, ge=1, le=500),
    ):
        return service.list(room_id=room_id, tenant_id=tenant_id, limit=limit)

    @router.delete("/v1/recordings/{recording_id}")
    def delete_recording(recording_id: str):
        if not service.delete(recording_id):
            raise HTTPException(status_code=404, detail="recording not found")
        return {"deleted": True}

    @router.post("/v1/recordings/{recording_id}/transcribe")
    async def transcribe(recording_id: str, bg: BackgroundTasks):
        try:
            meta = service.get(recording_id)
        except KeyError:
            raise HTTPException(status_code=404, detail="recording not found")
        if meta is None or meta.status != "ready":
            raise HTTPException(status_code=400, detail="recording not ready")
        bg.add_task(service.transcribe, recording_id)
        return {"transcription": "queued"}

    @router.post("/v1/recordings/{recording_id}/transcript")
    def upload_transcript(recording_id: str, payload: TranscriptPayload):
        meta = service.get(recording_id)
        if meta is None:
            raise HTTPException(status_code=404, detail="recording not found")
        import json

        body = json.dumps(payload.dict()).encode()
        key = f"transcripts/{recording_id}.json"
        service.storage.upload_bytes(body, key, content_type="application/json")
        meta.transcript_url = service.storage.presign_get(key)
        return {"transcript_url": meta.transcript_url}

    @router.get("/v1/recordings/{recording_id}/transcript")
    def get_transcript(recording_id: str):
        meta = service.get(recording_id)
        if meta is None:
            raise HTTPException(status_code=404, detail="recording not found")
        if not meta.transcript_url:
            raise HTTPException(status_code=404, detail="transcript not available")
        return {"transcript_url": meta.transcript_url}

    @router.get("/v1/recordings/{recording_id}/download")
    def download_recording(recording_id: str):
        meta = service.get(recording_id)
        if meta is None:
            raise HTTPException(status_code=404, detail="recording not found")
        if not meta.storage_path:
            raise HTTPException(status_code=404, detail="recording not uploaded yet")
        return {"download_url": service.storage.presign_get(meta.storage_path)}

    @router.get("/v1/recordings/{recording_id}/playback")
    def playback(recording_id: str):
        meta = service.get(recording_id)
        if meta is None:
            raise HTTPException(status_code=404, detail="recording not found")
        if not meta.storage_path:
            raise HTTPException(status_code=404, detail="recording not uploaded yet")
        url = service.storage.presign_get(meta.storage_path)
        return {"hls_url": url, "format": "mp4"}

    return router
