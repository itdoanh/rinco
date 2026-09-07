"""Async Loki client for log queries."""
from __future__ import annotations

import os
from datetime import datetime, timezone
from typing import Any

import httpx

from app.core import get_logger

LOKI_URL = os.getenv("LOKI_URL", "http://loki:3100")
TIMEOUT = 10.0

logger = get_logger("loki_client")


def _timestamp_to_ns(ts: str) -> int:
    """Convert ISO timestamp string to Loki-compatible nanoseconds."""
    try:
        dt = datetime.fromisoformat(ts.replace("Z", "+00:00"))
        return int(dt.replace(tzinfo=timezone.utc).timestamp() * 1_000_000_000)
    except Exception:
        return 0


def _parse_ts(value: Any) -> str:
    """Extract timestamp string from Loki result value."""
    if isinstance(value, (int, float)):
        ns = int(value)
        return datetime.fromtimestamp(ns / 1_000_000_000, tz=timezone.utc).isoformat()
    if isinstance(value, str) and len(value) > 13:
        try:
            dt = datetime.fromisoformat(value.replace("Z", "+00:00"))
            return dt.isoformat()
        except Exception:
            return value
    return str(value)


async def query_logs(
    query: str,
    limit: int = 100,
    from_ts: str = "",
    to_ts: str = "",
) -> list[dict[str, Any]]:
    """
    Query Loki for logs matching a LogQL query.

    Returns a list of log entries with timestamp, labels, and message.
    """
    try:
        now_ns = int(datetime.now(timezone.utc).timestamp() * 1_000_000_000)
        start_ns = _timestamp_to_ns(from_ts) if from_ts else now_ns - 3_600_000_000_000
        end_ns = _timestamp_to_ns(to_ts) if to_ts else now_ns

        body: dict[str, Any] = {
            "query": query,
            "limit": min(limit, 500),
            "start": start_ns,
            "end": end_ns,
            "direction": "BACKWARD",
        }

        async with httpx.AsyncClient(timeout=TIMEOUT) as client:
            r = await client.post(
                f"{LOKI_URL}/loki/api/v1/query_range",
                json=body,
            )
            r.raise_for_status()
            data = r.json()

        results: list[dict[str, Any]] = []
        streams = data.get("data", {}).get("result", [])
        for stream in streams:
            labels = stream.get("stream", {})
            entries = stream.get("values", [])
            for ts_nano, line in entries:
                results.append({
                    "timestamp": _parse_ts(ts_nano),
                    "labels": labels,
                    "message": line,
                })
            if len(results) >= limit:
                break

        logger.info("loki_query_success", query=query, count=len(results))
        return results[:limit]

    except httpx.HTTPStatusError as e:
        logger.warning("loki_http_error", status=e.response.status_code, query=query)
        return []
    except Exception as e:
        logger.warning("loki_query_failed", query=query, error=str(e))
        return []


async def query_aggregate(query: str) -> dict[str, Any]:
    """
    Run an aggregate (metric-style) LogQL query that returns stream data.

    Returns {"logs": [...], "count": n}.
    """
    try:
        now_ns = int(datetime.now(timezone.utc).timestamp() * 1_000_000_000)
        start_ns = now_ns - 3_600_000_000_000
        end_ns = now_ns

        body: dict[str, Any] = {
            "query": query,
            "limit": 100,
            "start": start_ns,
            "end": end_ns,
        }

        async with httpx.AsyncClient(timeout=TIMEOUT) as client:
            r = await client.post(
                f"{LOKI_URL}/loki/api/v1/query_range",
                json=body,
            )
            r.raise_for_status()
            data = r.json()

        streams = data.get("data", {}).get("result", [])
        results: list[dict[str, Any]] = []
        for stream in streams:
            labels = stream.get("stream", {})
            for ts_nano, line in stream.get("values", []):
                results.append({
                    "timestamp": _parse_ts(ts_nano),
                    "labels": labels,
                    "message": line,
                })

        return {"logs": results, "count": len(results)}

    except Exception as e:
        logger.warning("loki_aggregate_failed", query=query, error=str(e))
        return {"logs": [], "count": 0}
