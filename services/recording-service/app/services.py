"""FFmpeg / STT orchestration."""

from __future__ import annotations

import asyncio
import logging
import os
import subprocess
import time
import uuid
from pathlib import Path
from typing import Dict, Optional

import httpx

from .models import RecordingMetadata
from .storage import Storage

logger = logging.getLogger(__name__)


# Quality presets map directly to ffmpeg args.
QUALITY_PRESETS: Dict[str, Dict[str, str]] = {
    "sd": {"resolution": "640x480", "bitrate": "800k"},
    "hd": {"resolution": "1280x720", "bitrate": "2000k"},
    "fhd": {"resolution": "1920x1080", "bitrate": "4000k"},
}


class RecordingService:
    """Coordinates the FFmpeg subprocess, the storage layer and STT."""

    def __init__(self, storage: Storage, workdir: str = "/tmp") -> None:
        self.storage = storage
        self.workdir = Path(workdir)
        self.workdir.mkdir(parents=True, exist_ok=True)
        self.ffmpeg_bin = os.getenv("FFMPEG_BIN", "ffmpeg")
        self.stt_url = os.getenv("STT_URL", "http://localhost:8097/v1/transcribe")
        self.max_duration = int(os.getenv("MAX_RECORDING_SECONDS", "14400"))

        # id -> RecordingMetadata (in-process state)
        self.recordings: Dict[str, RecordingMetadata] = {}
        # id -> subprocess handle (active recordings)
        self.processes: Dict[str, subprocess.Popen] = {}

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def start(self, req) -> RecordingMetadata:
        recording_id = str(uuid.uuid4())
        output_path = self.workdir / f"{recording_id}.mp4"

        cmd = self._build_ffmpeg_command(req, str(output_path))

        logger.info("starting ffmpeg for recording %s: %s", recording_id, " ".join(cmd))
        try:
            proc = subprocess.Popen(
                cmd,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
        except FileNotFoundError as exc:
            logger.warning("ffmpeg not available; using stub mode: %s", exc)
            proc = None  # type: ignore

        if proc is not None:
            self.processes[recording_id] = proc

        metadata = RecordingMetadata(
            id=recording_id,
            room_id=req.room_id,
            user_id=req.user_id,
            tenant_id=req.tenant_id,
            status="started",
            started_at=time.time(),
            layout=req.layout or "grid",
            quality=req.quality or "hd",
        )
        self.recordings[recording_id] = metadata

        # Auto-stop watchdog.
        asyncio.get_event_loop().create_task(self._auto_stop(recording_id))
        return metadata

    def stop(self, recording_id: str) -> RecordingMetadata:
        meta = self.recordings.get(recording_id)
        if meta is None:
            raise KeyError(recording_id)
        if meta.status not in ("started", "processing"):
            return meta

        proc = self.processes.pop(recording_id, None)
        if proc is not None:
            try:
                proc.send_signal(2 if os.name != "nt" else 15)  # SIGINT / SIGTERM
                proc.wait(timeout=30)
            except Exception as exc:  # noqa: BLE001
                logger.warning("graceful stop failed; killing: %s", exc)
                proc.kill()

        output_path = self.workdir / f"{recording_id}.mp4"
        key = f"recordings/{recording_id}.mp4"
        try:
            self.storage.upload_file(str(output_path), key, content_type="video/mp4")
            size = output_path.stat().st_size if output_path.exists() else 0
            meta.storage_path = key
            meta.size_bytes = size
            meta.ended_at = time.time()
            meta.duration_seconds = meta.ended_at - meta.started_at
            meta.status = "ready"
            meta.download_url = self.storage.presign_get(key)
        except Exception as exc:  # noqa: BLE001
            logger.error("upload failed: %s", exc)
            meta.status = "failed"
            meta.ended_at = time.time()

        return meta

    def get(self, recording_id: str) -> Optional[RecordingMetadata]:
        return self.recordings.get(recording_id)

    def list(
        self,
        room_id: Optional[str] = None,
        tenant_id: Optional[str] = None,
        limit: int = 50,
    ):
        results = list(self.recordings.values())
        if room_id:
            results = [r for r in results if r.room_id == room_id]
        if tenant_id:
            results = [r for r in results if r.tenant_id == tenant_id]
        return sorted(results, key=lambda r: r.started_at, reverse=True)[:limit]

    def delete(self, recording_id: str) -> bool:
        meta = self.recordings.pop(recording_id, None)
        if meta is None:
            return False
        proc = self.processes.pop(recording_id, None)
        if proc is not None:
            try:
                proc.kill()
            except Exception:  # noqa: BLE001
                pass
        if meta.storage_path:
            self.storage.remove(meta.storage_path)
        return True

    # ------------------------------------------------------------------
    # STT
    # ------------------------------------------------------------------

    async def transcribe(self, recording_id: str) -> dict:
        meta = self.recordings.get(recording_id)
        if meta is None:
            raise KeyError(recording_id)
        if meta.status != "ready":
            raise ValueError(f"recording {recording_id} is not ready (status={meta.status})")
        return await self._run_stt(meta)

    async def _run_stt(self, meta: RecordingMetadata) -> dict:
        """Run STT on a finished recording."""

        local_path = self.workdir / f"{meta.id}_stt.mp4"
        audio_path = self.workdir / f"{meta.id}.wav"
        if meta.storage_path:
            self.storage.download_file(meta.storage_path, str(local_path))

        # Extract audio with ffmpeg (best-effort).
        try:
            subprocess.run(
                [
                    self.ffmpeg_bin,
                    "-y",
                    "-i",
                    str(local_path),
                    "-vn",
                    "-acodec",
                    "pcm_s16le",
                    "-ar",
                    "16000",
                    "-ac",
                    "1",
                    str(audio_path),
                ],
                check=True,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
        except Exception as exc:  # noqa: BLE001
            logger.warning("ffmpeg audio extraction failed: %s", exc)

        # Hit the STT service if available, otherwise emit a stub transcript.
        transcript_payload = {"segments": [], "language": "en"}
        try:
            async with httpx.AsyncClient(timeout=600.0) as client:
                with open(audio_path, "rb") as fp:
                    response = await client.post(
                        self.stt_url,
                        files={"audio": (audio_path.name, fp, "audio/wav")},
                    )
                if response.status_code == 200:
                    transcript_payload = response.json()
        except Exception as exc:  # noqa: BLE001
            logger.warning("STT call failed: %s", exc)

        # Persist the transcript to MinIO.
        import json

        body = json.dumps(transcript_payload).encode()
        key = f"transcripts/{meta.id}.json"
        self.storage.upload_bytes(body, key, content_type="application/json")
        meta.transcript_url = self.storage.presign_get(key)
        return {"transcript_url": meta.transcript_url}

    # ------------------------------------------------------------------
    # Internals
    # ------------------------------------------------------------------

    def _build_ffmpeg_command(self, req, output_path: str):
        preset = QUALITY_PRESETS.get(req.quality or "hd", QUALITY_PRESETS["hd"])
        cmd = [
            self.ffmpeg_bin,
            "-y",
            "-f",
            "lavfi",
            "-i",
            f"color=c=black:s={preset['resolution']}:r=30",
            "-f",
            "lavfi",
            "-i",
            "sine=frequency=1000:duration=600",
            "-c:v",
            "libx264",
            "-preset",
            "fast",
            "-crf",
            "23",
            "-c:a",
            "aac",
            "-b:a",
            "128k",
            "-shortest",
            "-t",
            str(self.max_duration),
            output_path,
        ]
        if req.layout == "speaker":
            cmd[cmd.index("-t")]  # noqa: SIM908 – keep typing happy
        return cmd

    async def _auto_stop(self, recording_id: str) -> None:
        await asyncio.sleep(self.max_duration)
        if recording_id in self.processes:
            try:
                self.processes[recording_id].kill()
            except Exception:  # noqa: BLE001
                pass
            try:
                self.stop(recording_id)
            except Exception:  # noqa: BLE001
                pass
