"""Tests for ai-sre loki_client helpers."""
from app.services.loki_client import _timestamp_to_ns, _parse_ts


def test_timestamp_to_ns_valid_iso():
    ns = _timestamp_to_ns("2026-01-15T12:00:00Z")
    assert ns > 0


def test_timestamp_to_ns_invalid():
    ns = _timestamp_to_ns("not-a-date")
    assert ns == 0


def test_timestamp_to_ns_empty():
    ns = _timestamp_to_ns("")
    assert ns == 0


def test_timestamp_to_ns_with_timezone():
    ns = _timestamp_to_ns("2026-01-15T12:00:00+00:00")
    assert ns > 0


def test_parse_ts_int():
    # 1 billion nanoseconds = 1 second
    result = _parse_ts(1_000_000_000)
    assert "1970" in result or "T" in result


def test_parse_ts_float():
    result = _parse_ts(1_000_000_000.0)
    assert "T" in result


def test_parse_ts_string_iso():
    result = _parse_ts("2026-01-15T12:00:00Z")
    assert "2026" in result


def test_parse_ts_string_invalid():
    result = _parse_ts("not-a-date")
    assert result == "not-a-date"


def test_parse_ts_short_string():
    # short string returns as-is
    result = _parse_ts("x")
    assert result == "x"


def test_parse_ts_other():
    result = _parse_ts(None)
    assert result == "None"
