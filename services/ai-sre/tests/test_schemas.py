"""Tests for ai-sre schemas."""
from __future__ import annotations

import pytest
from pydantic import ValidationError

from app.schemas import (
    IncidentCreate,
    Incident,
    RCAResult,
    AnalyzeRequest,
    AnalyzeResponse,
    HotfixGenerateRequest,
    HotfixGenerateResponse,
    RunbookCreateRequest,
    RunbookCreateResponse,
    ChatRequest,
    ChatResponse,
    CorrelationResponse,
)


# -- Original (Loop 51) tests --

def test_extra_incident_create_minimal():
    """IncidentCreate with required fields."""
    inc = IncidentCreate(
        service="my-svc",
        error="Database connection failed",
    )
    assert inc.service == "my-svc"
    assert inc.error == "Database connection failed"
    assert inc.severity == "P2"
    assert inc.trace_id is None
    assert inc.stack is None
    assert inc.sentry_url is None
    assert inc.files == []


def test_extra_incident_create_full():
    """IncidentCreate with all fields."""
    inc = IncidentCreate(
        service="my-svc",
        error="error",
        stack="traceback",
        severity="P1",
        sentry_url="https://sentry.io/event/123",
        trace_id="abc123",
        files=[{"path": "main.py", "line": 42}],
    )
    assert inc.severity == "P1"
    assert inc.sentry_url == "https://sentry.io/event/123"
    assert inc.trace_id == "abc123"
    assert len(inc.files) == 1


def test_extra_incident_extends():
    """Incident extends IncidentCreate with id and timestamps."""
    inc = Incident(
        service="svc",
        error="err",
        incident_id="INC-001",
        status="open",
    )
    assert inc.incident_id == "INC-001"
    assert inc.status == "open"


def test_extra_rca_result():
    """RCAResult should accept all fields."""
    rca = RCAResult(
        incident_id="INC-001",
        root_cause="DB pool exhausted",
        why="Too many concurrent queries",
        suggested_fix="Increase pool size",
        prevention="Monitor pool usage",
        confidence=0.85,
        file="db.go",
        line=42,
    )
    assert rca.confidence == 0.85
    assert rca.file == "db.go"
    assert rca.line == 42


def test_extra_rca_result_minimal():
    """RCAResult with only required fields."""
    rca = RCAResult(
        incident_id="INC-1",
        root_cause="x",
        why="y",
        suggested_fix="z",
        prevention="p",
        confidence=0.5,
    )
    assert rca.file is None
    assert rca.pr_url is None


def test_extra_analyze_request_defaults():
    """AnalyzeRequest with defaults."""
    req = AnalyzeRequest(
        service="svc",
        error="err",
    )
    assert req.time_window == "1h"
    assert req.auto_create_pr is True
    assert req.auto_create_runbook is True
    assert req.trace_id is None


def test_extra_analyze_request_custom():
    """AnalyzeRequest with custom values."""
    req = AnalyzeRequest(
        service="svc",
        error="err",
        trace_id="t-1",
        stack="traceback",
        time_window="30m",
        auto_create_pr=False,
    )
    assert req.time_window == "30m"
    assert req.auto_create_pr is False


def test_extra_analyze_response():
    """AnalyzeResponse should construct correctly."""
    rca = RCAResult(
        incident_id="INC-1",
        root_cause="x",
        why="y",
        suggested_fix="z",
        prevention="p",
        confidence=0.9,
    )
    resp = AnalyzeResponse(
        incident_id="INC-1",
        rca=rca,
    )
    assert resp.incident_id == "INC-1"
    assert resp.rca.confidence == 0.9
    assert resp.correlation == {}


def test_extra_hotfix_request():
    """HotfixGenerateRequest should accept fields."""
    req = HotfixGenerateRequest(
        error="null pointer",
        stack="traceback",
        source="auth-service",
        context="during login",
    )
    assert req.error == "null pointer"
    assert req.source == "auth-service"


def test_extra_hotfix_response():
    """HotfixGenerateResponse should accept fields."""
    resp = HotfixGenerateResponse(
        diff="--- a\n+++ b",
        file="auth.go",
        line=42,
        confidence=0.88,
        root_cause="null pointer in token parsing",
    )
    assert resp.file == "auth.go"
    assert resp.confidence == 0.88


def test_extra_runbook_create_request():
    """RunbookCreateRequest should accept fields."""
    req = RunbookCreateRequest(
        title="Runbook: DB failures",
        content_html="<h1>...</h1>",
        space="RINCO",
    )
    assert req.title == "Runbook: DB failures"
    assert req.space == "RINCO"


def test_extra_runbook_create_response():
    """RunbookCreateResponse should accept fields."""
    resp = RunbookCreateResponse(
        url="https://confluence/page/123",
        page_id="123",
    )
    assert resp.url == "https://confluence/page/123"
    assert resp.page_id == "123"


def test_extra_chat_request():
    """ChatRequest with optional incident context."""
    req = ChatRequest(
        message="What happened?",
        incident_id="INC-1",
        service="auth-service",
    )
    assert req.message == "What happened?"
    assert req.incident_id == "INC-1"


def test_extra_chat_response():
    """ChatResponse with sources."""
    resp = ChatResponse(
        reply="Database connection failed",
        sources=["log:1", "log:2"],
    )
    assert "Database" in resp.reply
    assert len(resp.sources) == 2


def test_extra_correlation_response():
    """CorrelationResponse should accept logs/traces/metrics."""
    resp = CorrelationResponse(
        logs=[{"message": "error 1"}],
        traces=[{"trace_id": "t-1"}],
        metrics={"requests_per_second": 100},
        summary="High error rate",
    )
    assert len(resp.logs) == 1
    assert resp.metrics["requests_per_second"] == 100


# -- Loop 171 additional tests --

def test_incident_create_required_fields():
    inc = IncidentCreate(service="api", error="boom")
    assert inc.service == "api"
    assert inc.error == "boom"
    assert inc.severity == "P2"
    assert inc.trace_id is None
    assert inc.stack is None
    assert inc.sentry_url is None
    assert inc.files == []


def test_incident_create_missing_service():
    with pytest.raises(ValidationError):
        IncidentCreate(error="boom")


def test_incident_create_missing_error():
    with pytest.raises(ValidationError):
        IncidentCreate(service="api")


def test_incident_create_files():
    inc = IncidentCreate(service="api", error="e", files=[{"path": "x.go", "line": 1}])
    assert len(inc.files) == 1
    assert inc.files[0]["path"] == "x.go"


def test_incident_create_default_files():
    inc = IncidentCreate(service="api", error="e")
    assert inc.files == []
    assert isinstance(inc.files, list)


def test_incident_inherits():
    inc = Incident(incident_id="inc-1", service="api", error="e")
    assert inc.incident_id == "inc-1"
    assert inc.status == "open"
    assert inc.created_at == ""
    assert inc.service == "api"


def test_incident_status_default():
    inc = Incident(incident_id="inc-1", service="api", error="e")
    assert inc.status == "open"


def test_rca_result_required():
    rca = RCAResult(
        incident_id="inc-1",
        root_cause="db",
        why="timeout",
        suggested_fix="retry",
        prevention="add timeout",
        confidence=0.9,
    )
    assert rca.incident_id == "inc-1"
    assert rca.confidence == 0.9
    assert rca.file is None
    assert rca.line is None
    assert rca.pr_url is None
    assert rca.runbook_url is None
    assert rca.notify_sent is False


def test_rca_result_missing_field():
    with pytest.raises(ValidationError):
        RCAResult(incident_id="x")


def test_rca_result_with_file():
    rca = RCAResult(
        incident_id="i",
        root_cause="r",
        why="w",
        suggested_fix="f",
        prevention="p",
        confidence=0.5,
        file="x.go",
        line=10,
    )
    assert rca.file == "x.go"
    assert rca.line == 10


def test_analyze_request_required():
    ar = AnalyzeRequest(service="api", error="e")
    assert ar.service == "api"
    assert ar.error == "e"
    assert ar.severity == "P2"
    assert ar.time_window == "1h"
    assert ar.auto_create_pr is True
    assert ar.auto_create_runbook is True
    assert ar.files == []


def test_analyze_request_missing_error():
    with pytest.raises(ValidationError):
        AnalyzeRequest(service="api")


def test_analyze_response_basic():
    rca = RCAResult(
        incident_id="i",
        root_cause="r",
        why="w",
        suggested_fix="f",
        prevention="p",
        confidence=0.7,
    )
    ar = AnalyzeResponse(incident_id="i", rca=rca)
    assert ar.incident_id == "i"
    assert ar.rca.confidence == 0.7
    assert ar.correlation == {}
    assert ar.pr_url is None
    assert ar.runbook_url is None


def test_hotfix_request_defaults():
    h = HotfixGenerateRequest(error="e")
    assert h.error == "e"
    assert h.stack is None
    assert h.source == ""
    assert h.context == ""


def test_hotfix_response():
    h = HotfixGenerateResponse(diff="+x", file="x.go", line=5, confidence=0.8)
    assert h.diff == "+x"
    assert h.file == "x.go"
    assert h.line == 5
    assert h.confidence == 0.8
    assert h.root_cause == ""


def test_hotfix_response_root_cause():
    h = HotfixGenerateResponse(diff="+x", file="x.go", line=5, confidence=0.8, root_cause="rc")
    assert h.root_cause == "rc"


def test_runbook_request():
    rb = RunbookCreateRequest(title="t", content_html="<p>x</p>")
    assert rb.title == "t"
    assert rb.content_html == "<p>x</p>"
    assert rb.space == "RINCO"


def test_runbook_response():
    rb = RunbookCreateResponse(url="https://x", page_id="p1")
    assert rb.url == "https://x"
    assert rb.page_id == "p1"


def test_runbook_response_no_page():
    rb = RunbookCreateResponse(url="https://x")
    assert rb.page_id is None


def test_chat_request_required():
    c = ChatRequest(message="hi")
    assert c.message == "hi"
    assert c.incident_id is None
    assert c.service is None


def test_chat_request_missing_message():
    with pytest.raises(ValidationError):
        ChatRequest()


def test_chat_response_defaults():
    c = ChatResponse(reply="hello")
    assert c.reply == "hello"
    assert c.sources == []


def test_chat_response_with_sources():
    c = ChatResponse(reply="hi", sources=["a", "b"])
    assert c.sources == ["a", "b"]


def test_correlation_defaults():
    c = CorrelationResponse()
    assert c.logs == []
    assert c.traces == []
    assert c.metrics == {}
    assert c.summary == ""


def test_correlation_full():
    c = CorrelationResponse(
        logs=[{"k": "v"}],
        traces=[{"id": "t1"}],
        metrics={"cpu": 80},
        summary="all good",
    )
    assert len(c.logs) == 1
    assert len(c.traces) == 1
    assert c.metrics["cpu"] == 80
    assert c.summary == "all good"
