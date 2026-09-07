"""Async Jaeger client for trace queries."""
from __future__ import annotations

import os
from typing import Any

import httpx

from app.core import get_logger

JAEGER_URL = os.getenv("JAEGER_URL", "http://jaeger:16686")
TIMEOUT = 10.0

logger = get_logger("jaeger_client")


async def get_trace(trace_id: str) -> dict[str, Any]:
    """
    Fetch a single trace by its ID.

    Returns the full Jaeger trace structure (spans, processes, etc.) or
    an empty dict on failure.
    """
    if not trace_id:
        return {}

    try:
        async with httpx.AsyncClient(timeout=TIMEOUT) as client:
            r = await client.get(
                f"{JAEGER_URL}/api/traces/{trace_id}",
                headers={"Accept": "application/json"},
            )
            r.raise_for_status()
            data = r.json()

        spans = data.get("data", [])
        logger.info("jaeger_trace_fetched", trace_id=trace_id, span_count=len(spans))
        return data

    except httpx.HTTPStatusError as e:
        logger.warning("jaeger_trace_not_found", trace_id=trace_id, status=e.response.status_code)
        return {}
    except Exception as e:
        logger.warning("jaeger_trace_failed", trace_id=trace_id, error=str(e))
        return {}


async def search_traces(
    service: str,
    lookback: str = "1h",
    limit: int = 20,
) -> list[dict[str, Any]]:
    """
    Search for traces belonging to a service.

    Args:
        service: Service name to filter by.
        lookback: Time window (e.g. "1h", "30m", "2h"). Default "1h".
        limit: Maximum number of traces to return (capped at 200).

    Returns a list of trace summaries (traceID + startTime + spanCount).
    """
    try:
        # Parse lookback string to seconds
        lookback_map = {"1h": 3600, "30m": 1800, "2h": 7200, "15m": 900, "5m": 300}
        seconds = lookback_map.get(lookback, 3600)

        params = {
            "service": service,
            "lookback": seconds,
            "limit": min(limit, 200),
            "pretty": "true",
        }

        async with httpx.AsyncClient(timeout=TIMEOUT) as client:
            r = await client.get(
                f"{JAEGER_URL}/api/traces",
                params=params,
                headers={"Accept": "application/json"},
            )
            r.raise_for_status()
            data = r.json()

        traces = data.get("data", [])
        results: list[dict[str, Any]] = []
        for trace in traces:
            results.append({
                "trace_id": trace.get("traceID", ""),
                "start_time": trace.get("startTime", ""),
                "duration_ms": trace.get("duration", 0) / 1000.0,
                "span_count": len(trace.get("spans", [])),
                "services": list({
                    span.get("processName", "")
                    for span in trace.get("spans", [])
                    if span.get("processName")
                }),
            })

        logger.info("jaeger_search_success", service=service, count=len(results))
        return results

    except httpx.HTTPStatusError as e:
        logger.warning("jaeger_search_failed", service=service, status=e.response.status_code)
        return []
    except Exception as e:
        logger.warning("jaeger_search_error", service=service, error=str(e))
        return []
