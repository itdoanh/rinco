"""Seed mock data for the RAG chatbot service.

Pre-populates a couple of *stub* collections so the search/chat endpoints
return responses without requiring Qdrant or the embedding model to be
online. The stubs are only used when the embeddings model fails to load
or QDRANT_URL is unreachable.
"""
from __future__ import annotations

import os
import time
import uuid
from typing import Any, Dict, List

__all__ = [
    "MOCK_COLLECTIONS",
    "MOCK_DOCUMENTS",
    "MOCK_ANSWERS",
    "initialise_seed_data",
]


MOCK_COLLECTIONS: List[Dict[str, Any]] = [
    {
        "name": "rinco-handbook",
        "tenant_id": "demo-tenant",
        "documents": 12,
        "created_at": "2026-01-15T00:00:00Z",
        "description": "Internal product handbook and runbooks.",
    },
    {
        "name": "rinco-public",
        "tenant_id": "demo-tenant",
        "documents": 6,
        "created_at": "2026-02-20T00:00:00Z",
        "description": "Public-facing docs and FAQs.",
    },
]


MOCK_DOCUMENTS: List[Dict[str, Any]] = [
    {
        "doc_id": "doc-001",
        "collection": "rinco-handbook",
        "title": "Architecture Overview",
        "snippet": (
            "RINCO is a modular enterprise platform built on a polyglot "
            "microservice mesh..."
        ),
        "score": 0.94,
    },
    {
        "doc_id": "doc-002",
        "collection": "rinco-handbook",
        "title": "Authentication Flow",
        "snippet": "JWT issuance via auth-service, validation by NGINX ingress...",
        "score": 0.88,
    },
    {
        "doc_id": "doc-003",
        "collection": "rinco-public",
        "title": "Pricing FAQ",
        "snippet": "Plans start at $29/seat/month and scale with usage...",
        "score": 0.81,
    },
]


MOCK_ANSWERS: Dict[str, str] = {
    "what is rinco?": (
        "RINCO is an enterprise SaaS platform combining CRM, video meetings, "
        "AI automation, and analytics in a single workspace."
    ),
    "how do we authenticate?": (
        "Authentication is via JWT issued by the auth-service. Each tenant "
        "has a dedicated signing keypair and a 24-hour grace window during "
        "key rotation."
    ),
    "what is the pricing?": (
        "Plans begin at $29 per seat per month; enterprise tier is custom "
        "and includes SOC2 compliance, advanced analytics, and dedicated "
        "support."
    ),
    "how do recordings work?": (
        "Meetings can be recorded when at least one admin enables it. "
        "Recordings are stored in MinIO and a transcript is generated "
        "automatically by the STT service."
    ),
}


def _answer(question: str) -> str:
    """Return the closest canned answer (very small lookup)."""
    q = question.lower().strip()
    for key, answer in MOCK_ANSWERS.items():
        if any(token in q for token in key.split()):
            return answer
    # Fallback: return a generic stub answer.
    return (
        "I'm operating in offline / stub mode because Qdrant and the "
        "embedding model aren't reachable. Please try again once those "
        "services are back online."
    )


def initialise_seed_data() -> None:
    """Lightweight hook; currently a no-op aside from logging."""
    if os.getenv("RAG_CHATBOT_SEED", "1") not in ("1", "true", "TRUE", "yes"):
        return
    # Currently no state to populate eagerly; embeddings & qdrant are
    # lazy-initialised by the lifespan handler.
