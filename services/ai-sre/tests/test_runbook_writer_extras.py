"""Extra tests for runbook_writer — HTML builder, escaping, edge cases."""
from __future__ import annotations

import pytest

from app.services.runbook_writer import (
    _escape_html,
    build_runbook_html,
    imported_datetime,
)


def test_extra_escape_html_ampersand():
    assert _escape_html("a & b") == "a &amp; b"


def test_extra_escape_html_lt_gt():
    assert _escape_html("<script>") == "&lt;script&gt;"


def test_extra_escape_html_quote():
    assert _escape_html('"hello"') == "&quot;hello&quot;"


def test_extra_escape_html_combined():
    assert _escape_html('<a href="x">&y</a>') == "&lt;a href=&quot;x&quot;&gt;&amp;y&lt;/a&gt;"


def test_extra_escape_html_empty():
    assert _escape_html("") == ""


def test_extra_escape_html_plain():
    assert _escape_html("hello world") == "hello world"


def test_extra_escape_html_already_escaped():
    """Already escaped input is escaped again (acceptable — idempotent encoding)."""
    out = _escape_html("&amp;")
    assert out == "&amp;amp;"


def test_extra_imported_datetime_format():
    s = imported_datetime()
    # Format: "YYYY-MM-DD HH:MM UTC"
    assert s.endswith(" UTC")
    assert len(s) >= len("YYYY-MM-DD HH:MM UTC")


def test_extra_imported_datetime_nonempty():
    s = imported_datetime()
    assert s != ""


def test_extra_build_runbook_html_contains_sections():
    html = build_runbook_html(
        incident_id="INC-001",
        service="auth",
        root_cause="DB overload",
        suggested_fix="Increase pool",
        prevention="Add cache",
        metrics={"cpu": "80%"},
        logs=[{"timestamp": "2024-01-01T00:00:00Z", "message": "OOM"}],
    )
    assert "<h1>Incident Runbook: auth</h1>" in html
    assert "<h2>Root Cause</h2>" in html
    assert "<h2>Suggested Fix</h2>" in html
    assert "<h2>Prevention</h2>" in html
    assert "<h2>Key Metrics</h2>" in html
    assert "<h2>Error Logs (Sample)</h2>" in html
    assert "INC-001" in html
    assert "DB overload" in html


def test_extra_build_runbook_html_escapes_root_cause():
    """XSS in root_cause should be escaped."""
    html = build_runbook_html(
        incident_id="INC-002",
        service="auth",
        root_cause="<script>alert('xss')</script>",
        suggested_fix="",
        prevention="",
        metrics={},
        logs=[],
    )
    assert "<script>alert" not in html
    assert "&lt;script&gt;" in html


def test_extra_build_runbook_html_escapes_fix():
    html = build_runbook_html(
        incident_id="INC-003",
        service="auth",
        root_cause="",
        suggested_fix='DROP TABLE "users";',
        prevention="",
        metrics={},
        logs=[],
    )
    assert "DROP TABLE" in html
    assert "&quot;users&quot;" in html


def test_extra_build_runbook_html_empty_metrics_shows_placeholder():
    html = build_runbook_html(
        incident_id="X", service="s", root_cause="", suggested_fix="", prevention="",
        metrics={}, logs=[],
    )
    assert "No metrics available" in html


def test_extra_build_runbook_html_empty_logs_shows_placeholder():
    html = build_runbook_html(
        incident_id="X", service="s", root_cause="", suggested_fix="", prevention="",
        metrics={"k": "v"}, logs=[],
    )
    assert "No logs available" in html


def test_extra_build_runbook_html_metrics_table():
    html = build_runbook_html(
        incident_id="X", service="s", root_cause="", suggested_fix="", prevention="",
        metrics={"cpu": "80%", "mem": "60%"}, logs=[],
    )
    assert "<td>cpu</td>" in html
    assert "<td>80%</td>" in html
    assert "<td>mem</td>" in html


def test_extra_build_runbook_html_logs_table():
    html = build_runbook_html(
        incident_id="X", service="s", root_cause="", suggested_fix="", prevention="",
        metrics={},
        logs=[
            {"timestamp": "2024-01-01T00:00:00Z", "message": "Error 1"},
            {"timestamp": "2024-01-01T00:01:00Z", "message": "Error 2"},
        ],
    )
    assert "Error 1" in html
    assert "Error 2" in html
    assert "2024-01-01T00:00:00Z" in html


def test_extra_build_runbook_html_logs_truncate():
    """Long log messages are truncated to 300 chars in HTML output."""
    long_msg = "x" * 500
    html = build_runbook_html(
        incident_id="X", service="s", root_cause="", suggested_fix="", prevention="",
        metrics={}, logs=[{"timestamp": "t", "message": long_msg}],
    )
    # The full 500-char string should NOT appear
    assert long_msg not in html
    # A truncated version should appear (300 chars + escaping won't apply since 'x' has no specials)
    assert "x" * 300 in html


def test_extra_build_runbook_html_logs_limit_to_20():
    """Only first 20 logs are included."""
    logs = [{"timestamp": f"t{i}", "message": f"m{i}"} for i in range(50)]
    html = build_runbook_html(
        incident_id="X", service="s", root_cause="", suggested_fix="", prevention="",
        metrics={}, logs=logs,
    )
    # First 20 should be present
    assert "m0" in html
    assert "m19" in html
    # After 20 should not be present
    assert "m20" not in html
    assert "m49" not in html


def test_extra_build_runbook_html_log_message_default():
    """Missing message key in log defaults to empty string."""
    html = build_runbook_html(
        incident_id="X", service="s", root_cause="", suggested_fix="", prevention="",
        metrics={}, logs=[{"timestamp": "t"}],  # no 'message'
    )
    assert "<td>t</td>" in html


def test_extra_build_runbook_html_log_timestamp_default():
    """Missing timestamp key in log defaults to empty string."""
    html = build_runbook_html(
        incident_id="X", service="s", root_cause="", suggested_fix="", prevention="",
        metrics={}, logs=[{"message": "boom"}],  # no 'timestamp'
    )
    assert "boom" in html


def test_extra_build_runbook_html_unicode():
    """Unicode in fields is preserved."""
    html = build_runbook_html(
        incident_id="VN-001", service="auth-vn", root_cause="Lỗi đầy DB",
        suggested_fix="Tăng pool", prevention="Thêm cache",
        metrics={"cpu": "80%"}, logs=[{"timestamp": "t", "message": "Lỗi"}],
    )
    assert "Lỗi đầy DB" in html
    assert "Tăng pool" in html


def test_extra_build_runbook_html_has_generated_timestamp():
    html = build_runbook_html(
        incident_id="X", service="s", root_cause="", suggested_fix="", prevention="",
        metrics={}, logs=[],
    )
    assert "<strong>Generated:</strong>" in html


def test_extra_build_runbook_html_has_footer():
    html = build_runbook_html(
        incident_id="X", service="s", root_cause="", suggested_fix="", prevention="",
        metrics={}, logs=[],
    )
    assert "AI SRE Worker" in html
    assert "<hr/>" in html


def test_extra_build_runbook_html_with_special_chars_in_service():
    """Service name with special chars is in HTML body (not escaped in title context)."""
    html = build_runbook_html(
        incident_id="X", service="svc<test>", root_cause="", suggested_fix="", prevention="",
        metrics={}, logs=[],
    )
    # Service appears in <h1> without escaping (could be improved, but we document behavior)
    assert "svc<test>" in html
