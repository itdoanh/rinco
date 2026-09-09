"""Tests for AI-SRE incident_correlator pure helpers."""
from app.services.incident_correlator import (
    _build_metric_queries,
    _is_error_log,
    _build_summary,
)


def test_build_metric_queries_contains_expected_labels():
    """Metric query builder includes all standard SRE metrics."""
    queries = _build_metric_queries("auth")
    assert "request_rate" in queries
    assert "error_rate" in queries
    assert "p99_latency" in queries
    assert "cpu_usage" in queries
    assert "memory_usage" in queries


def test_build_metric_queries_service_substituted():
    """Service name appears in PromQL queries."""
    queries = _build_metric_queries("billing-svc")
    for q in queries.values():
        assert "billing-svc" in q


def test_build_metric_queries_empty_service():
    """Empty service name does not break."""
    queries = _build_metric_queries("")
    # just verify it doesn't crash
    assert "request_rate" in queries


def test_is_error_log_by_level_error():
    """Log marked with level=error is treated as error."""
    log = {"labels": {"level": "error"}, "message": "ok"}
    assert _is_error_log(log) is True


def test_is_error_log_by_level_fatal():
    """Log marked with level=fatal is treated as error."""
    log = {"labels": {"level": "fatal"}, "message": "ok"}
    assert _is_error_log(log) is True


def test_is_error_log_by_level_info():
    """Log marked with level=info is not error."""
    log = {"labels": {"level": "info"}, "message": "all good"}
    assert _is_error_log(log) is False


def test_is_error_log_by_message_keyword():
    """Log containing 'error' in message is treated as error."""
    log = {"labels": {}, "message": "An error occurred"}
    assert _is_error_log(log) is True


def test_is_error_log_by_message_exception():
    """Log containing 'exception' in message is treated as error."""
    log = {"labels": {}, "message": "NullPointerException raised"}
    assert _is_error_log(log) is True


def test_is_error_log_benign():
    """Generic log is not error."""
    log = {"labels": {"level": "debug"}, "message": "user logged in"}
    assert _is_error_log(log) is False


def test_is_error_log_missing_fields():
    """Log with missing labels/message does not crash."""
    log = {}
    assert _is_error_log(log) is False


def test_build_summary_basic():
    """Summary builder produces expected section headers."""
    summary = _build_summary(
        "auth",
        logs=[],
        traces=[],
        metrics={},
        error_logs=[],
    )
    assert "Service: auth" in summary
    assert "Total logs in window: 0" in summary
    assert "Error logs: 0" in summary
    assert "Traces found: 0" in summary


def test_build_summary_with_metrics():
    """Metrics appear in summary."""
    summary = _build_summary(
        "auth",
        logs=[],
        traces=[],
        metrics={"p99_latency": 250.5},
        error_logs=[],
    )
    assert "p99_latency" in summary
    assert "250.5" in summary or "250" in summary


def test_build_summary_with_error_logs_sample():
    """First error log message is sampled in summary."""
    error_logs = [{"message": "OOM in worker thread"}]
    summary = _build_summary(
        "auth",
        logs=error_logs,
        traces=[],
        metrics={},
        error_logs=error_logs,
    )
    assert "First error log" in summary
    assert "OOM" in summary


def test_build_summary_with_traces_total_spans():
    """Summary includes total span count across traces."""
    traces = [
        {"span_count": 5},
        {"span_count": 3},
        {"span_count": 2},
    ]
    summary = _build_summary(
        "auth",
        logs=[],
        traces=traces,
        metrics={},
        error_logs=[],
    )
    assert "Total spans across traces: 10" in summary


def test_build_summary_metric_none_skipped():
    """Metric values of None are not included in summary."""
    summary = _build_summary(
        "auth",
        logs=[],
        traces=[],
        metrics={"unknown": None},
        error_logs=[],
    )
    # Just verify no crash; "unknown" line should not be added
    assert "unknown" not in summary
