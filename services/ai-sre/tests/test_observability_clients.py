"""Tests for AI-SRE loki_client and prometheus_client pure helpers."""
from datetime import datetime, timezone

from app.services.loki_client import _timestamp_to_ns, _parse_ts


def test_timestamp_to_ns_valid_iso():
    """ISO timestamp with Z is converted to nanoseconds."""
    out = _timestamp_to_ns("2026-01-01T00:00:00Z")
    assert out > 0
    # 2026-01-01 in ns is roughly 1.76e18
    assert 1.5e18 < out < 2.0e18


def test_timestamp_to_ns_valid_with_offset():
    """ISO timestamp with timezone offset."""
    out = _timestamp_to_ns("2026-06-15T12:00:00+00:00")
    assert out > 0


def test_timestamp_to_ns_invalid_string():
    """Invalid timestamp returns 0."""
    out = _timestamp_to_ns("not-a-date")
    assert out == 0


def test_timestamp_to_ns_empty():
    """Empty string returns 0."""
    out = _timestamp_to_ns("")
    assert out == 0


def test_parse_ts_from_int_nanoseconds():
    """Numeric nanoseconds become ISO string."""
    # 2026-01-01T00:00:00Z nanoseconds
    ns = 1_767_225_600_000_000_000
    out = _parse_ts(ns)
    assert "T" in out
    assert "2026" in out


def test_parse_ts_from_string_iso():
    """ISO string passes through."""
    s = "2026-01-01T00:00:00Z"
    out = _parse_ts(s)
    assert "2026" in out
    assert "T" in out


def test_parse_ts_from_short_string():
    """Short string is returned as-is."""
    s = "abc"
    out = _parse_ts(s)
    assert out == "abc"


def test_parse_ts_from_other_types():
    """Other types become string."""
    assert isinstance(_parse_ts(None), str)
    assert isinstance(_parse_ts([1, 2]), str)
