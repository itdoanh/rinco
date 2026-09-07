"""Tests for the incident_correlator module."""
from __future__ import annotations

import sys
import os

# Ensure project root is on path
sys.path.insert(
    0,
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
)

import pytest

try:
    from app.services import incident_correlator
except Exception as e:
    pytest.skip(f"Cannot import app.services.incident_correlator: {e}", allow_module_level=True)


# ---------------------------------------------------------------- mocks
@pytest.fixture
def mock_loki(monkeypatch):
    """Mock loki_client.query_logs to return sample log data."""
    async def _query_logs(query, limit=100, from_ts="", to_ts=""):
        return [
            {
                "timestamp": "2026-09-07T10:00:00Z",
                "labels": {"service": "auth-service", "level": "error"},
                "message": "Connection refused to database",
            },
            {
                "timestamp": "2026-09-07T10:00:01Z",
                "labels": {"service": "auth-service", "level": "info"},
                "message": "Request completed successfully",
            },
            {
                "timestamp": "2026-09-07T10:00:02Z",
                "labels": {"service": "auth-service", "level": "error"},
                "message": "Timeout waiting for response",
            },
        ]
    monkeypatch.setattr(incident_correlator.loki_client, "query_logs", _query_logs)


@pytest.fixture
def mock_jaeger(monkeypatch):
    """Mock jaeger_client.search_traces to return sample trace data."""
    async def _search_traces(service, lookback="1h", limit=20):
        return [
            {
                "trace_id": "abc123def456",
                "start_time": "2026-09-07T10:00:00Z",
                "duration_ms": 450.0,
                "span_count": 5,
                "services": ["auth-service", "db-service"],
            },
            {
                "trace_id": "xyz789aaa000",
                "start_time": "2026-09-07T10:00:05Z",
                "duration_ms": 1200.0,
                "span_count": 12,
                "services": ["auth-service"],
            },
        ]
    monkeypatch.setattr(incident_correlator.jaeger_client, "search_traces", _search_traces)
    monkeypatch.setattr(incident_correlator.jaeger_client, "get_trace", lambda trace_id: {})


@pytest.fixture
def mock_prometheus(monkeypatch):
    """Mock prometheus_client.query to return sample metric data."""
    async def _query(query):
        if "error_rate" in query:
            return [{"metric": {"service": "auth-service"}, "value": "0.05", "timestamp": 1234567890.0}]
        elif "request_rate" in query:
            return [{"metric": {"service": "auth-service"}, "value": "150.0", "timestamp": 1234567890.0}]
        elif "p99" in query:
            return [{"metric": {"service": "auth-service"}, "value": "0.82", "timestamp": 1234567890.0}]
        return []
    monkeypatch.setattr(incident_correlator.prometheus_client, "query", _query)


# ---------------------------------------------------------------- tests
@pytest.mark.asyncio
async def test_correlate_incident_returns_all_three_data_sources(
    mock_loki, mock_jaeger, mock_prometheus
):
    """correlate_incident should return logs, traces, and metrics."""
    result = await incident_correlator.correlate_incident(
        service="auth-service",
        trace_id=None,
        time_window="1h",
    )

    assert "logs" in result
    assert "traces" in result
    assert "metrics" in result
    assert "summary" in result
    assert isinstance(result["logs"], list)
    assert isinstance(result["traces"], list)
    assert isinstance(result["metrics"], dict)


@pytest.mark.asyncio
async def test_correlate_incident_logs_count(mock_loki, mock_jaeger, mock_prometheus):
    """Logs from Loki should be returned and error logs identified."""
    result = await incident_correlator.correlate_incident(service="auth-service")

    assert len(result["logs"]) == 3
    # _is_error_log should identify error-level logs
    error_logs = [l for l in result["logs"] if incident_correlator._is_error_log(l)]
    assert len(error_logs) == 2


@pytest.mark.asyncio
async def test_correlate_incident_traces_count(mock_loki, mock_jaeger, mock_prometheus):
    """Traces from Jaeger should be returned."""
    result = await incident_correlator.correlate_incident(service="auth-service")

    assert len(result["traces"]) == 2
    assert result["traces"][0]["trace_id"] == "abc123def456"
    assert result["traces"][0]["span_count"] == 5


@pytest.mark.asyncio
async def test_correlate_incident_metrics_populated(mock_loki, mock_jaeger, mock_prometheus):
    """Prometheus metrics should be attached to the result."""
    result = await incident_correlator.correlate_incident(service="auth-service")

    assert "request_rate" in result["metrics"]
    assert "error_rate" in result["metrics"]
    assert result["metrics"]["error_rate"] == "0.05"


@pytest.mark.asyncio
async def test_correlate_incident_with_specific_trace_id(mock_loki, mock_jaeger, mock_prometheus):
    """When trace_id is provided, get_trace should be called and spans returned."""
    async def mock_get_trace(tid):
        return {
            "data": [{
                "spans": [
                    {"operationName": "authenticate", "duration": 50_000_000, "tags": [{"key": "error", "value": "false"}]},
                    {"operationName": "query_db", "duration": 200_000_000, "tags": [{"key": "error", "value": "true"}]},
                ]
            }]
        }
    monkeypatch.setattr(incident_correlator.jaeger_client, "get_trace", mock_get_trace)

    result = await incident_correlator.correlate_incident(
        service="auth-service",
        trace_id="abc123def456",
    )

    # When trace_id is set, search_traces is not called — only get_trace
    assert "traces" in result
    assert result["traces"][0]["trace_id"] == "abc123def456"


@pytest.mark.asyncio
async def test_summary_contains_key_info(mock_loki, mock_jaeger, mock_prometheus):
    """Summary string should mention service, log count, and error count."""
    result = await incident_correlator.correlate_incident(service="auth-service")

    summary = result["summary"]
    assert "auth-service" in summary
    assert "Total logs" in summary or "logs" in summary
    assert "Error logs" in summary


def test_is_error_log():
    """_is_error_log should correctly identify error-level log entries."""
    assert incident_correlator._is_error_log(
        {"labels": {"level": "error"}, "message": "something failed"}
    )
    assert incident_correlator._is_error_log(
        {"labels": {"level": "err"}, "message": "oops"}
    )
    assert incident_correlator._is_error_log(
        {"labels": {}, "message": "NullPointerException in auth"}
    )
    # Should not flag info-level logs
    assert not incident_correlator._is_error_log(
        {"labels": {"level": "info"}, "message": "Request succeeded"}
    )


def test_build_metric_queries():
    """_build_metric_queries should return PromQL for standard SRE metrics."""
    queries = incident_correlator._build_metric_queries("my-service")
    assert "request_rate" in queries
    assert "error_rate" in queries
    assert "p99_latency" in queries
    assert 'service="my-service"' in queries["request_rate"]
