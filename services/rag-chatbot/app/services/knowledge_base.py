"""Knowledge base helpers for rag-chatbot.

Provides:

- :func:`seed_knowledge_base` — seed an initial KB with the standard
  RINCO documentation snippets (FAQ, getting-started, troubleshooting).
- :func:`build_collection_name` — derive a tenant-scoped Qdrant
  collection name with the standard ``rinco_`` prefix.
- :func:`kb_stats` — return summary statistics for one or more KBs
  (chunk counts, source diversity).
"""
from __future__ import annotations

import hashlib
from typing import Any, Iterable, Sequence

DEFAULT_KB_PREFIX = "rinco_kb_"


def build_collection_name(tenant_id: str, slug: str | None = None) -> str:
    """Return a tenant-scoped collection name.

    Slug is optional; when omitted the result is just the
    ``rinco_kb_<tenant_hash>`` collection, which matches the convention
    in :mod:`app.services.qdrant_client`.
    """
    if not tenant_id:
        raise ValueError("tenant_id is required")
    suffix = tenant_id
    if slug:
        # Hash tenant+salt to keep names short.
        h = hashlib.sha1(f"{tenant_id}:{slug}".encode()).hexdigest()[:8]
        suffix = f"{tenant_id}_{h}"
    return f"{DEFAULT_KB_PREFIX}{suffix}"


# ---------------------------------------------------------------------------
# Curated RINCO knowledge base snippets
# ---------------------------------------------------------------------------
RINCO_KB: list[dict[str, str]] = [
    {
        "title": "Getting started with RINCO CRM",
        "content": (
            "RINCO CRM is a multi-tenant SaaS platform that bundles CRM, "
            "chat, video meeting, AI lead scoring, voice transcription and "
            "AI SRE into a single product. To begin: 1) create a tenant at "
            "https://app.rinco.example/onboard; 2) invite users; 3) import "
            "leads via CSV or the API; 4) enable lead scoring per tenant via "
            "the admin portal."
        ),
    },
    {
        "title": "Lead scoring tiers",
        "content": (
            "Lead scores are returned on a 0–100 scale. The service groups "
            "scores into four tiers: very-hot (>=85, recommend immediate "
            "phone call), hot (>=60, call within an hour), warm (>=30, "
            "nurture via email), and cold (<30, automatic drip campaign)."
        ),
    },
    {
        "title": "Creating a meeting room",
        "content": (
            "Meeting rooms are created on demand by POSTing to "
            "/v1/rooms with a tenant id and the host user id. Each room is a "
            "WebRTC SFU session backed by the meeting-engine Rust service. "
            "Recordings, transcripts (via the STT service), and AI "
            "summaries are persisted on room end."
        ),
    },
    {
        "title": "AI SRE and incident analysis",
        "content": (
            "The AI SRE worker is triggered by Sentry webhooks and manual "
            "calls to /v1/analyze. It correlates logs (Loki), traces "
            "(Jaeger) and metrics (Prometheus) for the incident's "
            "trace_id, sends the evidence to a vLLM-served code-LLM, and "
            "returns a structured RCA plus an optional GitHub PR."
        ),
    },
    {
        "title": "RAG chatbot and knowledge base",
        "content": (
            "The RAG chatbot retrieves per-tenant context from Qdrant. "
            "Embeddings come from sentence-transformers (bge-m3 by default) "
            "and are re-ranked with bge-reranker-base. Ingest documents via "
            "POST /v1/ingest with kind=pdf|docx|xlsx|url|text."
        ),
    },
    {
        "title": "Speech-to-text transcription",
        "content": (
            "The STT service wraps faster-whisper (large-v3). Upload audio "
            "to /v1/transcribe, optionally include language hint (vi/en/...), "
            "and receive text plus segment timestamps. Speaker diarization "
            "is available through pyannote.audio."
        ),
    },
    {
        "title": "Observability stack",
        "content": (
            "RINCO standardises on Loki (logs), Prometheus / VictoriaMetrics "
            "(metrics), Jaeger / Tempo (traces) and ClickHouse (analytics + "
            "audit). The observability-service exposes a unified HTTP + "
            "Connect-RPC API for all three backends plus AlertManager "
            "webhooks."
        ),
    },
    {
        "title": "Privacy and PII handling",
        "content": (
            "All chat, ingestion and analytics pipelines redact phone "
            "numbers, emails and CCCD numbers before any LLM call. "
            "Per-tenant isolation is enforced at every layer: vector "
            "collections, Postgres schemas, NATS subjects and ClickHouse "
            "databases all carry a tenant_id."
        ),
    },
]


def seed_knowledge_base(
    tenant_id: str,
    *,
    extra: Sequence[dict[str, str]] | None = None,
) -> list[dict[str, str]]:
    """Return the curated RINCO KB scoped to ``tenant_id``.

    The result is intended to be passed verbatim to :func:`app.api.ingest`
    or directly fed to :func:`app.services.ingestion.ingest_content`. The
    caller chooses whether to actually embed the documents.

    Args:
        tenant_id: tenant identifier.
        extra: optional caller-supplied snippets to merge with the
            curated list.
    """
    if not tenant_id:
        raise ValueError("tenant_id is required")

    base = [
        {
            "title": item["title"],
            "content": item["content"],
            "tenant_id": tenant_id,
            "source_url": f"https://kb.rinco.example/{tenant_id}/docs",
        }
        for item in RINCO_KB
    ]
    if extra:
        base.extend(
            {"title": e["title"], "content": e["content"], "tenant_id": tenant_id, "source_url": ""}
            for e in extra
        )
    return base


def kb_stats(points: Iterable[dict[str, Any]]) -> dict[str, Any]:
    """Aggregate light-weight stats over a sequence of KB points."""
    docs: list[dict[str, Any]] = list(points)
    sources: set[str] = set()
    chunks = 0
    bytes_ = 0
    for p in docs:
        payload = p.get("payload", {})
        text = payload.get("text", "")
        src = payload.get("source_url") or payload.get("doc_type") or "unknown"
        sources.add(src)
        chunks += 1
        bytes_ += len(text)
    return {
        "chunks": chunks,
        "distinct_sources": len(sources),
        "sources": sorted(sources),
        "approx_chars": bytes_,
    }


__all__ = [
    "DEFAULT_KB_PREFIX",
    "RINCO_KB",
    "build_collection_name",
    "seed_knowledge_base",
    "kb_stats",
]
