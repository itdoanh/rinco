"""Tests for AI-SRE runbook_writer helpers and /v1/runbook endpoint."""
import pytest
from httpx import AsyncClient, ASGITransport

from app.main import app
from app.services import runbook_writer


def test_escape_html_basic():
    """_escape_html escapes &, <, >, \"."""
    out = runbook_writer._escape_html('<a href="x">&y</a>')
    assert "&lt;a" in out
    assert "&gt;" in out
    assert "&amp;" in out
    assert "&quot;" in out


def test_escape_html_no_special():
    """Plain text passes through."""
    assert runbook_writer._escape_html("hello") == "hello"


def test_escape_html_non_string():
    """Non-string gets converted."""
    assert runbook_writer._escape_html(123) == "123"


def test_imported_datetime_format():
    """imported_datetime returns formatted UTC string."""
    out = runbook_writer.imported_datetime()
    assert "UTC" in out
    assert len(out) >= len("2026-01-01 00:00 UTC")


def test_build_runbook_html_minimal():
    """build_runbook_html with minimal args."""
    html = runbook_writer.build_runbook_html(
        incident_id="i-1",
        service="auth",
        root_cause="null pointer",
        suggested_fix="add None check",
        prevention="add tests",
        metrics={},
        logs=[],
    )
    assert "<h1>Incident Runbook: auth</h1>" in html
    assert "i-1" in html
    assert "null pointer" in html
    assert "add None check" in html
    assert "add tests" in html
    assert "No metrics available" in html
    assert "No logs available" in html


def test_build_runbook_html_with_metrics():
    """build_runbook_html with metrics renders rows."""
    metrics = {"p99_latency_ms": 250, "error_rate": 0.05}
    html = runbook_writer.build_runbook_html(
        incident_id="i-2",
        service="billing",
        root_cause="r",
        suggested_fix="f",
        prevention="p",
        metrics=metrics,
        logs=[],
    )
    assert "p99_latency_ms" in html
    assert "250" in html
    assert "error_rate" in html
    assert "0.05" in html


def test_build_runbook_html_with_logs():
    """build_runbook_html renders log table rows."""
    logs = [
        {"timestamp": "2026-01-01T00:00:00Z", "message": "OOM in worker"},
        {"timestamp": "2026-01-01T00:00:01Z", "message": "timeout"},
    ]
    html = runbook_writer.build_runbook_html(
        incident_id="i-3",
        service="ai",
        root_cause="r",
        suggested_fix="f",
        prevention="p",
        metrics={},
        logs=logs,
    )
    assert "OOM in worker" in html
    assert "timeout" in html
    assert "2026-01-01T00:00:00Z" in html


def test_build_runbook_html_truncates_log_message():
    """Log messages are truncated to 300 chars."""
    logs = [{"timestamp": "t", "message": "x" * 1000}]
    html = runbook_writer.build_runbook_html(
        incident_id="i-4",
        service="s",
        root_cause="r",
        suggested_fix="f",
        prevention="p",
        metrics={},
        logs=logs,
    )
    # the long x's should be truncated to 300 chars
    assert "x" * 300 in html
    assert "x" * 301 not in html


def test_build_runbook_html_escapes_html_in_root_cause():
    """HTML special chars in root_cause are escaped."""
    html = runbook_writer.build_runbook_html(
        incident_id="i-5",
        service="s",
        root_cause="<script>alert(1)</script>",
        suggested_fix="<b>fix</b>",
        prevention="<i>prevent</i>",
        metrics={},
        logs=[],
    )
    assert "<script>" not in html
    assert "<b>fix</b>" not in html
    assert "&lt;script&gt;" in html
    assert "&lt;b&gt;fix&lt;/b&gt;" in html


@pytest.mark.asyncio
async def test_build_runbook_endpoint():
    """POST /v1/runbook/build-html returns HTML."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.post(
            "/v1/runbook/build-html",
            params={
                "incident_id": "i-100",
                "service": "auth",
                "root_cause": "DB timeout",
                "suggested_fix": "increase pool",
                "prevention": "add SLA",
            },
        )
    assert resp.status_code == 200
    data = resp.json()
    assert "html" in data
    assert "DB timeout" in data["html"]


@pytest.mark.asyncio
async def test_create_runbook_endpoint_missing_title(monkeypatch):
    """POST /v1/runbook/create without title returns 400."""
    monkeypatch.setattr(runbook_writer, "CONFLUENCE_USER", "user")
    monkeypatch.setattr(runbook_writer, "CONFLUENCE_TOKEN", "tok")
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.post(
            "/v1/runbook/create",
            json={"title": "", "content_html": "<p>x</p>"},
        )
    assert resp.status_code == 400


@pytest.mark.asyncio
async def test_create_runbook_endpoint_missing_content():
    """POST /v1/runbook/create without content returns 400."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.post(
            "/v1/runbook/create",
            json={"title": "t", "content_html": ""},
        )
    assert resp.status_code == 400


@pytest.mark.asyncio
async def test_create_runbook_endpoint_no_credentials(monkeypatch):
    """POST /v1/runbook/create without credentials returns 502."""
    monkeypatch.setattr(runbook_writer, "CONFLUENCE_USER", "")
    monkeypatch.setattr(runbook_writer, "CONFLUENCE_TOKEN", "")
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.post(
            "/v1/runbook/create",
            json={"title": "Runbook", "content_html": "<p>x</p>"},
        )
    assert resp.status_code == 502
