"""Extended mock data for the recording service (WS-B Loop 9).

20+ recordings across the three demo tenants with realistic metadata.
"""
from __future__ import annotations

import os
import time
import uuid
from typing import Any, Dict, List

__all__ = [
    "EXTENDED_RECORDINGS",
    "EXTENDED_STORAGE_STATS",
    "EXTENDED_TRANSCRIPTION_JOBS",
]


_TENANTS = [
    ("demo-tenant-apexfintech", "Apex Fintech"),
    ("demo-tenant-hct-consulting", "HCT Consulting"),
    ("demo-tenant", "Demo Company"),
]

_ROOMS = [
    "hcm-team-sync", "hn-team-sync", "all-hands", "investor-call",
    "product-review", "engineering-sprint", "sales-standup",
    "customer-demo", "1on1-minh-anh", "1on1-hai-yen",
    "townhall-q3", "compliance-training", "demo-prep", "partner-webinar",
    "vietnam-roadshow", "vn-japan-meeting", "weekly-finance",
    "design-review", "security-audit", "board-meeting",
    "product-launch", "all-retrospective",
]

_LANGUAGES = ["vi", "en", "ja"]


def _now() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def _build_extended_recording(i: int, room: str, tenant: str, tenant_name: str) -> Dict[str, Any]:
    duration_min = 15 + (i * 7) % 90
    participants = 2 + (i % 12)
    file_size_mb = duration_min * (3 + (i % 4))
    rec_id = f"rec-ext-{i:04d}"
    return {
        "recording_id": rec_id,
        "room_id": f"room-{room}",
        "tenant_id": tenant,
        "tenant_name": tenant_name,
        "title": f"{tenant_name} - {room.replace('-', ' ').title()}",
        "description": f"Recording of {room} session at {tenant_name}",
        "duration_sec": duration_min * 60,
        "file_size_mb": file_size_mb,
        "participants": [
            {"user_id": f"user-{j:03d}", "name": f"User {j}", "joined_at": _now(), "left_at": _now()}
            for j in range(participants)
        ],
        "start_time": _now(),
        "end_time": _now(),
        "status": (["completed","completed","completed","processing","failed"])[i % 5],
        "language_primary": _LANGUAGES[i % 3],
        "languages_detected": [_LANGUAGES[i % 3], _LANGUAGES[(i + 1) % 3]],
        "quality": (["hd","hd","4k","sd"])[i % 4],
        "format": "mp4",
        "codec": (["h264","h265","vp9"])[i % 3],
        "storage_url": f"s3://rinco-recordings/{tenant}/{rec_id}.mp4",
        "thumbnail_url": f"s3://rinco-recordings/{tenant}/{rec_id}.jpg",
        "view_count": (i * 11) % 200,
        "download_count": i % 15,
        "transcript_status": (["available","available","processing","unavailable"])[i % 4],
        "tags": [
            (["meeting","team-sync","all-hands","sales","demo","training","review","board"])[i % 8],
            _LANGUAGES[i % 3],
            (["internal","customer","investor","partner"])[i % 4],
        ],
        "metadata": {
            "recording_engine": "mediasoup",
            "uploaded_via": (["web","desktop","mobile"])[i % 3],
            "encryption": "AES-256",
            "retention_days": 365,
        },
    }


EXTENDED_RECORDINGS: List[Dict[str, Any]] = [
    _build_extended_recording(i, _ROOMS[i % len(_ROOMS)], _TENANTS[i % 3][0], _TENANTS[i % 3][1])
    for i in range(22)
]


# Storage stats by tenant
EXTENDED_STORAGE_STATS: List[Dict[str, Any]] = [
    {
        "tenant_id": tenant_id,
        "tenant_name": tenant_name,
        "total_recordings": 7 + i,
        "total_size_gb": round(20.0 + (i * 7.5), 2),
        "avg_duration_min": 45,
        "retention_days": 365,
        "encrypted": True,
        "storage_class": (["standard","standard","intelligent-tiering"])[i % 3],
    }
    for i, (tenant_id, tenant_name) in enumerate(_TENANTS)
]


# Transcription jobs (15)
EXTENDED_TRANSCRIPTION_JOBS: List[Dict[str, Any]] = [
    {
        "job_id": f"job-ext-{i:03d}",
        "recording_id": f"rec-ext-{i:04d}",
        "status": (["queued","running","completed","failed"])[i % 4],
        "language": _LANGUAGES[i % 3],
        "progress_pct": min(100, i * 7),
        "started_at": _now(),
        "completed_at": _now() if i % 4 == 2 else None,
        "duration_sec": (i * 9) % 180,
    }
    for i in range(15)
]
