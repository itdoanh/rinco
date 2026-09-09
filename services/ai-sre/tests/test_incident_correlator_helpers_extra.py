"""Tests for ai-sre incident_correlator helpers."""
from app.services.incident_correlator import (
    _build_metric_queries,
    _is_error_log,
    _build_summary,
    DEFAULT_LOKI_QUERY_LIMIT,
    DEFAULT_TRACE_LIMIT,
)


def test_build_metric_queries_returns_5():
    queries = _build_metric_queries("test-service")
    assert len(queries) == 5


def test_build_metric_queries_keys():
    queries = _build_metric_queries("test-service")
    assert "request_rate" in queries
    assert "error_rate" in queries
    assert "p99_latency" in queries
    assert "cpu_usage" in queries
    assert "memory_usage" in queries


def test_build_metric_queries_contains_service():
    queries = _build_metric_queries("my-svc")
    for label, query in queries.items():
        assert "my-svc" in query, f"{label} missing service: {query}"


def test_build_metric_queries_request_rate_5m():
    queries = _build_metric_queries("test")
    assert "[5m]" in queries["request_rate"]


def test_is_error_log_level_error():
    log = {"labels": {"level": "error"}, "message": "some error"}
    assert _is_error_log(log) is True


def test_is_error_log_level_err():
    log = {"labels": {"level": "err"}, "message": "x"}
    assert _is_error_log(log) is True


def test_is_error_log_level_fatal():
    log = {"labels": {"level": "fatal"}, "message": "x"}
    assert _is_error_log(log) is True


def test_is_error_log_level_critical():
    log = {"labels": {"level": "critical"}, "message": "x"}
    assert _is_error_log(log) is True


def test_is_error_log_message_keyword():
    log = {"labels": {}, "message": "An error occurred"}
    assert _is_error_log(log) is True


def test_is_error_log_exception_keyword():
    log = {"labels": {}, "message": "NullPointerException raised"}
    assert _is_error_log(log) is True


def test_is_error_log_info():
    log = {"labels": {"level": "info"}, "message": "user logged in"}
    assert _is_error_log(log) is False


def test_is_error_log_debug():
    log = {"labels": {"level": "debug"}, "message": "x"}
    assert _is_error_log(log) is False


def test_is_error_log_empty():
    log = {}
    assert _is_error_log(log) is False


def test_build_summary_basic():
    summary = _build_summary("svc1", [], [], {}, [])
    assert "Service: svc1" in summary
    assert "Total logs in window: 0" in summary
    assert "Error logs: 0" in summary


def test_build_summary_with_logs():
    logs = [
        {"labels": {"level": "error"}, "message": "boom"},
        {"labels": {"level": "info"}, "message": "ok"},
    ]
    error_logs = [logs[0]]
    summary = _build_summary("svc1", logs, [], {}, error_logs)
    assert "Total logs in window: 2" in summary
    assert "Error logs: 1" in summary
    assert "First error log: boom" in summary


def test_build_summary_with_metrics():
    metrics = {"request_rate": 100.5, "error_rate": 0.1}
    summary = _build_summary("svc1", [], [], metrics, [])
    assert "request_rate" in summary
    assert "100.5" in summary


def test_build_summary_with_traces():
    traces = [{"span_count": 5}, {"span_count": 3}]
    summary = _build_summary("svc1", [], traces, {}, [])
    assert "Traces found: 2" in summary
    assert "Total spans across traces: 8" in summary


def test_build_summary_metric_with_none_value():
    metrics = {"request_rate": None, "error_rate": 0.5}
    summary = _build_summary("svc1", [], [], metrics, [])
    assert "error_rate" in summary
    # None metrics should not be in summary
    assert "request_rate" not in summary or "None" not in summary


def test_default_loki_query_limit():
    assert DEFAULT_LOKI_QUERY_LIMIT == 100


def test_default_trace_limit():
    assert DEFAULT_TRACE_LIMIT == 20


def test_build_metric_queries_promql_syntax():
    queries = _build_metric_queries("svc")
    # Check that p99 uses histogram_quantile
    assert "histogram_quantile(0.99" in queries["p99_latency"]
    # Check that error_rate matches 5xx
    assert 'status=~"5.."' in queries["error_rate"]
