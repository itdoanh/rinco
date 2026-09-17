"""Seed data for recording service."""
from __future__ import annotations
import time
from typing import Dict, List, Optional

# Sample recording metadata
SAMPLE_RECORDINGS: List[dict] = [
    {
        "id": "rec_001",
        "room_id": "room-sales-weekly",
        "user_id": "user_john",
        "tenant_id": "tenant_acme",
        "status": "ready",
        "started_at": time.time() - 7200,
        "ended_at": time.time() - 3600,
        "duration_seconds": 3600,
        "size_bytes": 524288000,
        "layout": "speaker",
        "quality": "hd",
        "participants": ["john", "sarah", "mike"],
    },
    {
        "id": "rec_002",
        "room_id": "room-product-review",
        "user_id": "user_sarah",
        "tenant_id": "tenant_acme",
        "status": "ready",
        "started_at": time.time() - 86400,
        "ended_at": time.time() - 82800,
        "duration_seconds": 3600,
        "size_bytes": 480000000,
        "layout": "grid",
        "quality": "hd",
        "participants": ["sarah", "alex", "emma"],
    },
    {
        "id": "rec_003",
        "room_id": "room-training-session",
        "user_id": "user_mike",
        "tenant_id": "tenant_globex",
        "status": "ready",
        "started_at": time.time() - 172800,
        "ended_at": time.time() - 169200,
        "duration_seconds": 3600,
        "size_bytes": 620000000,
        "layout": "presentation",
        "quality": "fhd",
        "participants": ["mike", "team_a", "team_b", "team_c"],
    },
    {
        "id": "rec_004",
        "room_id": "room-client-demo",
        "user_id": "user_john",
        "tenant_id": "tenant_acme",
        "status": "processing",
        "started_at": time.time() - 1800,
        "layout": "grid",
        "quality": "hd",
        "participants": ["john", "client_rep"],
    },
    {
        "id": "rec_005",
        "room_id": "room-standup",
        "user_id": "user_alex",
        "tenant_id": "tenant_globex",
        "status": "ready",
        "started_at": time.time() - 43200,
        "ended_at": time.time() - 41400,
        "duration_seconds": 1800,
        "size_bytes": 260000000,
        "layout": "grid",
        "quality": "sd",
        "participants": ["alex", "team"],
    },
]

# Sample transcripts
SAMPLE_TRANSCRIPTS: Dict[str, dict] = {
    "rec_001": {
        "language": "en",
        "segments": [
            {"start": 0.0, "end": 5.0, "text": "Let's start the weekly sales meeting."},
            {"start": 5.0, "end": 12.0, "text": "Today we'll discuss Q3 targets and pipeline updates."},
            {"start": 12.0, "end": 20.0, "text": "John, can you share the current pipeline status?"},
            {"start": 20.0, "end": 45.0, "text": "We have three major deals in negotiation. Total value is approximately 500k."},
            {"start": 45.0, "end": 60.0, "text": "Great progress. Sarah, any updates on the enterprise deal?"},
        ],
    },
    "rec_002": {
        "language": "en",
        "segments": [
            {"start": 0.0, "end": 8.0, "text": "Welcome to the product review session."},
            {"start": 8.0, "end": 25.0, "text": "We're going to demo the new CRM features including AI lead scoring."},
            {"start": 25.0, "end": 40.0, "text": "The AI model can now predict lead quality with 85% accuracy."},
        ],
    },
}

# Room configurations
ROOM_CONFIGS: Dict[str, dict] = {
    "room-sales-weekly": {
        "default_layout": "speaker",
        "default_quality": "hd",
        "auto_record": True,
        "max_participants": 10,
    },
    "room-product-review": {
        "default_layout": "grid",
        "default_quality": "hd",
        "auto_record": True,
        "max_participants": 20,
    },
    "room-training-session": {
        "default_layout": "presentation",
        "default_quality": "fhd",
        "auto_record": True,
        "max_participants": 50,
    },
    "room-standup": {
        "default_layout": "grid",
        "default_quality": "sd",
        "auto_record": False,
        "max_participants": 15,
    },
}


def get_sample_recordings() -> List[dict]:
    """Return sample recordings metadata."""
    return SAMPLE_RECORDINGS


def get_sample_transcripts() -> Dict[str, dict]:
    """Return sample transcripts."""
    return SAMPLE_TRANSCRIPTS


def get_room_configs() -> Dict[str, dict]:
    """Return room configurations."""
    return ROOM_CONFIGS
