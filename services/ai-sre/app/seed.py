"""Seed mock data for the AI-SRE service.

Provides a small in-memory ``incidents`` store and a few synthetic
hotfix / runbook fixtures so the service can respond without external
dependencies (GitHub, Confluence, vLLM) being online.
"""
from __future__ import annotations

import os
import time
import uuid
from typing import Any, Dict, List

# WS-B Loop 9: extended mock data (30+ incidents, 50+ runbooks, 20+ hotfixes)
try:
    from app.expansion.seed_extra import (
        EXTENDED_INCIDENTS,
        EXTENDED_RUNBOOKS,
        EXTENDED_HOTFIXES,
        EXTENDED_DEPLOY_HISTORY,
        EXTENDED_SERVICE_CATALOG,
    )
except ImportError:  # pragma: no cover
    EXTENDED_INCIDENTS = []
    EXTENDED_RUNBOOKS = []
    EXTENDED_HOTFIXES = []
    EXTENDED_DEPLOY_HISTORY = []
    EXTENDED_SERVICE_CATALOG = []

__all__ = [
    "MOCK_INCIDENTS",
    "MOCK_RUNBOOKS",
    "MOCK_HOTFIXES",
    "EXTENDED_INCIDENTS",
    "EXTENDED_RUNBOOKS",
    "EXTENDED_HOTFIXES",
    "EXTENDED_DEPLOY_HISTORY",
    "EXTENDED_SERVICE_CATALOG",
    "initialise_seed_data",
    "create_mock_incident",
]


def _now() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


# ---------------------------------------------------------------------------
# Mock incidents — used when AI_SRE_SEED=1 or when the vLLM backend is
# unreachable.
# ---------------------------------------------------------------------------

MOCK_INCIDENTS: List[Dict[str, Any]] = [
    {
        "incident_id": "inc-001",
        "service": "auth-service",
        "trace_id": "trace-aaa-111",
        "error": "JWT verification failed: signature mismatch",
        "severity": "P1",
        "status": "open",
        "created_at": _now(),
        "files": [
            {"path": "auth-service/internal/handler/jwt.go", "line": 42},
        ],
    },
    {
        "incident_id": "inc-002",
        "service": "video-service",
        "trace_id": "trace-bbb-222",
        "error": "WebRTC ICE connection failed: STUN timeout",
        "severity": "P2",
        "status": "investigating",
        "created_at": _now(),
        "files": [
            {"path": "video-service/internal/turn/client.go", "line": 88},
        ],
    },
    {
        "incident_id": "inc-003",
        "service": "analytics-service",
        "trace_id": "trace-ccc-333",
        "error": "ClickHouse insert query timeout",
        "severity": "P3",
        "status": "resolved",
        "created_at": _now(),
        "files": [],
    },
]

MOCK_RUNBOOKS: List[Dict[str, Any]] = [
    {
        "page_id": "runbook-001",
        "title": "JWT Verification Failures",
        "url": "https://confluence.example.com/runbook/jwt-failures",
        "service": "auth-service",
        "summary": (
            "Check signing key rotation; ensure both old and new keys are "
            "accepted during the 24h grace window."
        ),
    },
    {
        "page_id": "runbook-002",
        "title": "WebRTC ICE / TURN failures",
        "url": "https://confluence.example.com/runbook/ice-turn",
        "service": "video-service",
        "summary": (
            "Verify TURN credentials; check STUN reachability; review "
            "firewall rules for UDP/3478."
        ),
    },
]

MOCK_HOTFIXES: List[Dict[str, Any]] = [
    {
        "hotfix_id": "hf-001",
        "incident_id": "inc-001",
        "diff": (
            "--- a/auth-service/internal/handler/jwt.go\n"
            "+++ b/auth-service/internal/handler/jwt.go\n"
            "@@ -42,1 +42,1 @@\n"
            "-    if err := token.Claims.VerifyString(signingKey); err != nil {\n"
            "+    if err := token.Claims.VerifyString(activeSigningKey()); err != nil {\n"
        ),
        "file": "auth-service/internal/handler/jwt.go",
        "line": 42,
        "confidence": 0.92,
    },
]


def create_mock_incident(service: str, error: str, severity: str = "P2") -> Dict[str, Any]:
    """Generate a synthetic incident record (used in place of GitHub / Sentry)."""
    inc_id = f"inc-{uuid.uuid4().hex[:8]}"
    return {
        "incident_id": inc_id,
        "service": service,
        "error": error,
        "severity": severity,
        "status": "open",
        "created_at": _now(),
        "trace_id": f"trace-{uuid.uuid4().hex[:8]}",
        "files": [],
    }


def initialise_seed_data() -> None:
    """Boot-inject mock incidents into the in-process store."""
    if os.getenv("AI_SRE_SEED", "1") not in ("1", "true", "TRUE", "yes"):
        return
    try:
        # Use the store's create API (sync wrapper uses asyncio.run).
        import asyncio

        from app.services import incident_store

        async def _seed() -> None:
            for inc in MOCK_INCIDENTS:
                if inc["incident_id"] not in incident_store._incidents:
                    await incident_store.create_incident(inc)

        try:
            loop = asyncio.get_event_loop()
            if loop.is_running():
                # We're inside an already-running event loop; skip.
                return
            loop.run_until_complete(_seed())
        except RuntimeError:
            # No loop available; just populate the dict directly.
            for inc in MOCK_INCIDENTS:
                incident_store._incidents.setdefault(
                    inc["incident_id"], inc
                )
    except Exception:  # pragma: no cover
        pass
