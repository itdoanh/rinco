"""Tests for ai-sre runbook writer."""
from __future__ import annotations

from app.services.runbook_writer import (
    _escape_html,
    build_runbook_html,
    imported_datetime,
)


def test_extra_escape_html_ampersand():
    """Ampersand should be escaped."""
    assert _escape_html("a & b") == "a &amp; b"


def test_extra_escape_html_less_than():
    """Less-than should be escaped."""
    assert _escape_html("<script>") == "&lt;script&gt;"


def test_extra_escape_html_quote():
    """Quotes should be escaped."""
    assert _escape_html('"hello"') == "&quot;hello&quot;"


def test_extra_escape_html_no_special_chars():
    """Plain text should be unchanged."""
    assert _escape_html("plain text") == "plain text"


def test_extra_escape_html_multiple():
    """Multiple special chars should all be escaped."""
    got = _escape_html("<a href=\"b\">c & d</a>")
    assert "&lt;" in got
    assert "&gt;" in got
    assert "&amp;" in got
    assert "&quot;" in got


def test_extra_imported_datetime_format():
    """imported_datetime should return formatted UTC time."""
    s = imported_datetime()
    # Format: "YYYY-MM-DD HH:MM UTC"
    assert s.endswith(" UTC")
    assert len(s) >= len("2026-01-01 00:00 UTC")


def test_extra_imported_datetime_contains_year():
    """imported_datetime should include current year."""
    s = imported_datetime()
    from datetime import datetime, timezone
    year = datetime.now(timezone.utc).year
    assert str(year) in s


def test_extra_build_runbook_html_basic():
    """build_runbook_html should produce valid HTML."""
    html = build_runbook_html(
        incident_id="INC-001",
        service="my-service",
        root_cause="DB connection pool exhausted",
        suggested_fix="Increase pool size",
        prevention="Monitor pool usage",
        metrics={},
        logs=[],
    )
    assert "<h1>Incident Runbook: my-service</h1>" in html
    assert "INC-001" in html
    assert "DB connection pool exhausted" in html
    assert "Increase pool size" in html
    assert "Monitor pool usage" in html


def test_extra_build_runbook_html_includes_prevention():
    """Prevention section should be included."""
    html = build_runbook_html(
        "INC-1", "svc", "cause", "fix", "prevent this", {}, []
    )
    assert "prevent this" in html
    assert "Prevention" in html


def test_extra_build_runbook_html_escapes_root_cause():
    """Root cause should be HTML-escaped."""
    html = build_runbook_html(
        "INC-1", "svc", "<script>alert('xss')</script>", "fix", "prevent", {}, []
    )
    assert "<script>" not in html
    assert "&lt;script&gt;" in html


def test_extra_build_runbook_html_escapes_suggested_fix():
    """Suggested fix should be HTML-escaped."""
    html = build_runbook_html(
        "INC-1", "svc", "cause", "<bad>", "prevent", {}, []
    )
    assert "&lt;bad&gt;" in html


def test_extra_build_runbook_html_no_metrics():
    """No metrics should show fallback message."""
    html = build_runbook_html(
        "INC-1", "svc", "c", "f", "p", {}, []
    )
    assert "No metrics available" in html


def test_extra_build_runbook_html_with_metrics():
    """Metrics should be rendered as table rows."""
    html = build_runbook_html(
        "INC-1", "svc", "c", "f", "p",
        {"request_rate": 100, "error_rate": 0.01},
        [],
    )
    assert "request_rate" in html
    assert "error_rate" in html
    assert "100" in html
    assert "0.01" in html


def test_extra_build_runbook_html_no_logs():
    """No logs should show fallback message."""
    html = build_runbook_html(
        "INC-1", "svc", "c", "f", "p", {}, []
    )
    assert "No logs available" in html


def test_extra_build_runbook_html_with_logs():
    """Logs should be rendered as table rows."""
    logs = [
        {"timestamp": "2026-01-01T00:00:00Z", "message": "Error 1"},
        {"timestamp": "2026-01-01T00:00:01Z", "message": "Error 2"},
    ]
    html = build_runbook_html(
        "INC-1", "svc", "c", "f", "p", {}, logs
    )
    assert "2026-01-01T00:00:00Z" in html
    assert "Error 1" in html
    assert "Error 2" in html


def test_extra_build_runbook_html_limits_logs_to_20():
    """Should only render first 20 logs."""
    logs = [{"timestamp": f"t{i}", "message": f"m{i}"} for i in range(30)]
    html = build_runbook_html(
        "INC-1", "svc", "c", "f", "p", {}, logs
    )
    # First 20 should be present
    assert "m0" in html
    assert "m19" in html
    # 21+ should not be present
    assert "m20" not in html
    assert "m29" not in html


def test_extra_build_runbook_html_truncates_log_messages():
    """Long log messages should be truncated to 300 chars."""
    long_msg = "x" * 500
    logs = [{"timestamp": "t1", "message": long_msg}]
    html = build_runbook_html(
        "INC-1", "svc", "c", "f", "p", {}, logs
    )
    # Should contain 300 x's
    assert "x" * 300 in html
    # Should not contain 301+ x's in the message context
    # (find the longest run of x's)
    import re
    runs = re.findall(r"x+", html)
    if runs:
        assert max(len(r) for r in runs) <= 300


def test_extra_build_runbook_html_escapes_log_messages():
    """Log messages should be HTML-escaped."""
    logs = [{"timestamp": "t1", "message": "<bad>"}]
    html = build_runbook_html(
        "INC-1", "svc", "c", "f", "p", {}, logs
    )
    assert "&lt;bad&gt;" in html


def test_extra_build_runbook_html_includes_footer():
    """Should include auto-generated footer."""
    html = build_runbook_html(
        "INC-1", "svc", "c", "f", "p", {}, []
    )
    assert "AI SRE Worker" in html


def test_extra_build_runbook_html_uses_storage_format():
    """Output is HTML for Confluence storage format."""
    html = build_runbook_html(
        "INC-1", "svc", "c", "f", "p", {}, []
    )
    assert "<table>" in html
    assert "</table>" in html
    assert "<h1>" in html
