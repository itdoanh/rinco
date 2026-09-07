"""Async Prometheus client for metric queries."""
from __future__ import annotations

import os
from typing import Any

import httpx

from app.core import get_logger

PROMETHEUS_URL = os.getenv("PROMETHEUS_URL", "http://prometheus:9090")
TIMEOUT = 10.0

logger = get_logger("prometheus_client")


async def query(query: str) -> list[dict[str, Any]]:
    """
    Execute an instant PromQL query.

    Returns a list of metric result objects with metric labels and value.
    """
    try:
        async with httpx.AsyncClient(timeout=TIMEOUT) as client:
            r = await client.get(
                f"{PROMETHEUS_URL}/api/v1/query",
                params={"query": query},
                headers={"Accept": "application/json"},
            )
            r.raise_for_status()
            data = r.json()

        status = data.get("status", "")
        if status != "success":
            logger.warning("prometheus_query_bad_status", status=status, query=query)
            return []

        results: list[dict[str, Any]] = []
        for item in data.get("data", {}).get("result", []):
            metric = item.get("metric", {})
            value = item.get("value")
            results.append({
                "metric": metric,
                "value": value[1] if value else None,
                "timestamp": value[0] if value else None,
            })

        logger.info("prometheus_query_success", query=query, count=len(results))
        return results

    except httpx.HTTPStatusError as e:
        logger.warning("prometheus_query_failed", status=e.response.status_code, query=query)
        return []
    except Exception as e:
        logger.warning("prometheus_query_error", query=query, error=str(e))
        return []


async def query_range(
    query: str,
    start: float,
    end: float,
    step: str = "15s",
) -> list[dict[str, Any]]:
    """
    Execute a range PromQL query over a time window.

    Args:
        query: PromQL expression.
        start: Unix timestamp for start of range.
        end: Unix timestamp for end of range.
        step: Query resolution step (e.g. "15s", "1m").

    Returns a list of result objects each containing a "values" array.
    """
    try:
        async with httpx.AsyncClient(timeout=TIMEOUT) as client:
            r = await client.get(
                f"{PROMETHEUS_URL}/api/v1/query_range",
                params={
                    "query": query,
                    "start": start,
                    "end": end,
                    "step": step,
                },
                headers={"Accept": "application/json"},
            )
            r.raise_for_status()
            data = r.json()

        status = data.get("status", "")
        if status != "success":
            logger.warning("prometheus_range_bad_status", status=status, query=query)
            return []

        results: list[dict[str, Any]] = []
        for item in data.get("data", {}).get("result", []):
            results.append({
                "metric": item.get("metric", {}),
                "values": item.get("values", []),
            })

        logger.info("prometheus_range_success", query=query, count=len(results))
        return results

    except httpx.HTTPStatusError as e:
        logger.warning("prometheus_range_failed", status=e.response.status_code, query=query)
        return []
    except Exception as e:
        logger.warning("prometheus_range_error", query=query, error=str(e))
        return []


async def get_metric(metric_name: str) -> dict[str, Any]:
    """
    Fetch metadata about a specific metric (type, help, unit).

    Returns a dict with "metric": metric_name, "type", "help" or empty dict on failure.
    """
    try:
        async with httpx.AsyncClient(timeout=TIMEOUT) as client:
            r = await client.get(
                f"{PROMETHEUS_URL}/api/v1/metadata",
                params={"metric": metric_name},
                headers={"Accept": "application/json"},
            )
            r.raise_for_status()
            data = r.json()

        metadata = data.get("data", {}).get(metric_name, [])
        if not metadata:
            return {}

        info = metadata[0]
        return {
            "metric": metric_name,
            "type": info.get("type", "unknown"),
            "help": info.get("help", ""),
            "unit": info.get("unit", ""),
        }

    except Exception as e:
        logger.warning("prometheus_metric_info_failed", metric=metric_name, error=str(e))
        return {}
