"""Tests for ai-sre incident correlator helper functions."""
from __future__ import annotations

import pytest

from app.services.incident_correlator import (
    _build_metric_queries,
    _is_error_log,
    _build_summary,
)


def test_extra_build_metric_queries_returns_expected_labels():
    """Should return standard metric labels."""
    queries = _build_metric_queries("my-service")
    assert "request_rate" in queries
    assert "error_rate" in queries
    assert "p99_latency" in queries
    assert "cpu_usage" in queries
    assert "memory_usage" in queries


def test_extra_build_metric_queries_includes_service():
    """Service name should be embedded in queries."""
    queries = _build_metric_queries("my-service")
    for label, query in queries.items():
        assert 'service="my-service"' in query


def test_extra_build_metric_queries_request_rate_uses_rate():
    """Request rate uses rate() function."""
    queries = _build_metric_queries("svc")
    assert "rate(" in queries["request_rate"]


def test_extra_build_metric_queries_error_rate_filters_5xx():
    """Error rate should filter for 5xx status."""
    queries = _build_metric_queries("svc")
    assert 'status=~"5.."' in queries["error_rate"]


def test_extra_build_metric_queries_p99_uses_quantile():
    """p99 uses histogram_quantile."""
    queries = _build_metric_queries("svc")
    assert "histogram_quantile(0.99" in queries["p99_latency"]


def test_extra_is_error_log_error_level():
    """Error level labels should be detected."""
    log = {"labels": {"level": "error"}, "message": "x"}
    assert _is_error_log(log) is True


def test_extra_is_error_log_fatal_level():
    """Fatal level should be detected."""
    log = {"labels": {"level": "fatal"}, "message": "x"}
    assert _is_error_log(log) is True


def test_extra_is_error_log_err_level():
    """'err' should be detected."""
    log = {"labels": {"level": "err"}, "message": "x"}
    assert _is_error_log(log) is True


def test_extra_is_error_log_critical_level():
    """Critical level should be detected."""
    log = {"labels": {"level": "critical"}, "message": "x"}
    assert _is_error_log(log) is True


def test_extra_is_error_log_message_contains_error():
    """Logs containing 'error' in message should be detected."""
    log = {"labels": {"level": "info"}, "message": "An error occurred"}
    assert _is_error_log(log) is True


def test_extra_is_error_log_message_contains_exception():
    """Logs containing 'exception' in message should be detected."""
    log = {"labels": {"level": "info"}, "message": "NullPointerException thrown"}
    assert _is_error_log(log) is True


def test_extra_is_error_log_normal_log():
    """Normal info logs should NOT be errors."""
    log = {"labels": {"level": "info"}, "message": "User logged in"}
    assert _is_error_log(log) is False


def test_extra_is_error_log_no_labels():
    """Logs without labels should still work."""
    log = {"message": "Some message"}
    assert _is_error_log(log) is False


def test_extra_build_summary_basic():
    """Summary should include service name and counts."""
    summary = _build_summary("my-svc", [], [], {}, [])
    assert "Service: my-svc" in summary
    assert "Total logs in window: 0" in summary
    assert "Error logs: 0" in summary
    assert "Traces found: 0" in summary


def test_extra_build_summary_with_logs():
    """Summary should include log count."""
    logs = [{"message": "log 1"}, {"message": "log 2"}]
    summary = _build_summary("svc", logs, [], {}, [])
    assert "Total logs in window: 2" in summary


def test_extra_build_summary_with_metrics():
    """Summary should include metrics."""
    metrics = {"request_rate": 100.5, "error_rate": 0.01}
    summary = _build_summary("svc", [], [], metrics, [])
    assert "request_rate: 100.5" in summary
    assert "error_rate: 0.01" in summary


def test_extra_build_summary_metric_none_skipped():
    """Metrics with None value should be skipped."""
    metrics = {"request_rate": None, "error_rate": 0.01}
    summary = _build_summary("svc", [], [], metrics, [])
    assert "request_rate" not in summary
    assert "error_rate: 0.01" in summary


def test_extra_build_summary_error_log_sample():
    """First error log message should be included."""
    error_logs = [{"message": "Database connection failed"}]
    summary = _build_summary("svc", [], [], {}, error_logs)
    assert "Database connection failed" in summary


def test_extra_build_summary_traces_with_spans():
    """Total spans should be summed across traces."""
    traces = [
        {"span_count": 5},
        {"span_count": 3},
        {"span_count": 2},
    ]
    summary = _build_summary("svc", [], traces, {}, [])
    assert "Total spans across traces: 10" in summary


def test_extra_build_summary_trace_no_span_count():
    """Trace dict without span_count should be 0."""
    traces = [{}]
    summary = _build_summary("svc", [], traces, {}, [])
    assert "Total spans across traces: 0" in summary


def test_extra_build_summary_truncates_long_messages():
    """Long error messages should be truncated to 200 chars."""
    long_msg = "x" * 500
    error_logs = [{"message": long_msg}]
    summary = _build_summary("svc", [], [], {}, error_logs)
    # The summary line is "First error log: <msg>" where msg is truncated
    assert "x" * 200 in summary
    assert "x" * 201 not in summary


def test_extra_build_metric_queries_different_services():
    """Different services should produce different queries."""
    q1 = _build_metric_queries("svc-a")
    q2 = _build_metric_queries("svc-b")
    assert q1["request_rate"] != q2["request_rate"]
    assert 'svc-a' in q1["request_rate"]
    assert 'svc-b' in q2["request_rate"]
