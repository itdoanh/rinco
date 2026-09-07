"""Incident correlator: gather all evidence for a trace/service."""
from __future__ import annotations

import os
import time
from datetime import datetime, timezone
from typing import Any

from app.core import get_logger

from . import jaeger_client
from . import loki_client
from . import prometheus_client

logger = get_logger("incident_correlator")

DEFAULT_LOKI_QUERY_LIMIT = 100
DEFAULT_TRACE_LIMIT = 20


async def correlate_incident(
    service: str,
    trace_id: str | None = None,
    time_window: str = "1h",
) -> dict[str, Any]:
    """
    Gather all observability evidence for an incident.

    Combines logs (Loki), traces (Jaeger), and metrics (Prometheus) into
    a single structured dict for downstream LLM analysis.

    Args:
        service: Target service name.
        trace_id: Optional specific trace ID to focus on.
        time_window: Time window for queries (e.g. "1h", "30m").

    Returns:
        {
            "logs": [...],           # List of log entries
            "traces": [...],         # List of trace summaries
            "metrics": {...},        # Key metric snapshots
            "summary": str,          # Human-readable summary
        }
    """
    t0 = time.monotonic()
    lookback_map = {"1h": 3600, "30m": 1800, "2h": 7200, "15m": 900, "5m": 300}
    seconds = lookback_map.get(time_window, 3600)
    now_ts = datetime.now(timezone.utc)
    from_ts = datetime.fromtimestamp(now_ts.timestamp() - seconds, tz=timezone.utc).isoformat()
    to_ts = now_ts.isoformat()

    logs: list[dict[str, Any]] = []
    traces: list[dict[str, Any]] = []
    metrics: dict[str, Any] = {}

    # 1. Logs from Loki
    try:
        log_query = f'{{service="{service}"}}'
        logs = await loki_client.query_logs(
            query=log_query,
            limit=DEFAULT_LOKI_QUERY_LIMIT,
            from_ts=from_ts,
            to_ts=to_ts,
        )
    except Exception as e:
        logger.warning("correlator_logs_failed", service=service, error=str(e))

    # 2. Traces from Jaeger
    try:
        if trace_id:
            trace_data = await jaeger_client.get_trace(trace_id)
            if trace_data:
                spans = trace_data.get("data", [{}])[0].get("spans", [])
                traces = [{
                    "trace_id": trace_id,
                    "span_count": len(spans),
                    "spans": [
                        {
                            "operation_name": s.get("operationName", ""),
                            "duration_ms": s.get("duration", 0) / 1000.0,
                            "tags": {
                                t["key"]: t.get("value", "")
                                for t in s.get("tags", [])
                            },
                        }
                        for s in spans
                    ],
                }]
        else:
            trace_summaries = await jaeger_client.search_traces(
                service=service,
                lookback=time_window,
                limit=DEFAULT_TRACE_LIMIT,
            )
            traces = trace_summaries
    except Exception as e:
        logger.warning("correlator_traces_failed", service=service, error=str(e))

    # 3. Metrics from Prometheus
    metric_queries = _build_metric_queries(service)
    for metric_label, query in metric_queries.items():
        try:
            result = await prometheus_client.query(query)
            if result:
                metrics[metric_label] = result[0].get("value")
        except Exception as e:
            logger.warning("correlator_metric_failed", label=metric_label, error=str(e))

    # 4. Build summary
    error_logs = [l for l in logs if _is_error_log(l)]
    summary = _build_summary(service, logs, traces, metrics, error_logs)

    elapsed = time.monotonic() - t0
    logger.info(
        "correlation_complete",
        service=service,
        trace_id=trace_id,
        logs_count=len(logs),
        traces_count=len(traces),
        metrics_count=len(metrics),
        elapsed_s=round(elapsed, 2),
    )

    return {
        "logs": logs,
        "traces": traces,
        "metrics": metrics,
        "summary": summary,
    }


def _build_metric_queries(service: str) -> dict[str, str]:
    """Return a map of label -> PromQL query for a given service."""
    return {
        "request_rate": f'rate(http_requests_total{{service="{service}"}}[5m])',
        "error_rate": f'rate(http_requests_total{{service="{service}",status=~"5.."}}[5m])',
        "p99_latency": f'histogram_quantile(0.99, rate(http_request_duration_seconds_bucket{{service="{service}"}}[5m]))',
        "cpu_usage": f'rate(container_cpu_usage_seconds_total{{service="{service}"}}[5m])',
        "memory_usage": f'memory_usage_bytes{{service="{service}"}} / 1024 / 1024',
    }


def _is_error_log(log: dict[str, Any]) -> bool:
    """Check if a Loki log entry looks like an error."""
    level = str(log.get("labels", {}).get("level", "")).lower()
    msg = str(log.get("message", "")).lower()
    return level in ("error", "err", "fatal", "critical") or "error" in msg or "exception" in msg


def _build_summary(
    service: str,
    logs: list[dict[str, Any]],
    traces: list[dict[str, Any]],
    metrics: dict[str, Any],
    error_logs: list[dict[str, Any]],
) -> str:
    """Build a human-readable summary of the incident evidence."""
    lines = [
        f"Service: {service}",
        f"Total logs in window: {len(logs)}",
        f"Error logs: {len(error_logs)}",
        f"Traces found: {len(traces)}",
    ]

    if metrics:
        for label, value in metrics.items():
            if value is not None:
                lines.append(f"  {label}: {round(float(value), 2)}")

    if error_logs:
        sample = error_logs[0].get("message", "")[:200]
        lines.append(f"First error log: {sample}")

    if traces:
        total_spans = sum(t.get("span_count", 0) for t in traces)
        lines.append(f"Total spans across traces: {total_spans}")

    return "\n".join(lines)
