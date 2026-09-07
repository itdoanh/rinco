"""POST /v1/analyze — full RCA pipeline: correlate + analyze + hotfix + PR + runbook."""
from __future__ import annotations

import json
import re
import time
import uuid
from typing import Any

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from app.core import get_logger, METRICS
from app.core import VLLM_URL, LLM_MODEL, GITHUB_TOKEN, GITHUB_REPO, GITHUB_REF
from app.schemas import AnalyzeRequest, AnalyzeResponse, RCAResult

from app.services import incident_correlator
from app.services import hotfix_generator
from app.services import github_client
from app.services import runbook_writer
from app.services import incident_store

router = APIRouter(prefix="/v1", tags=["analyze"])
logger = get_logger("api.analyze")


class AnalyzeRequestIn(BaseModel):
    """Internal request model mirroring AnalyzeRequest."""
    service: str
    trace_id: str | None = None
    error: str
    stack: str | None = None
    severity: str = "P2"
    sentry_url: str | None = None
    files: list[dict[str, Any]] = []
    time_window: str = "1h"
    auto_create_pr: bool = True
    auto_create_runbook: bool = True


@router.post("/analyze", response_model=AnalyzeResponse)
async def analyze(req: AnalyzeRequest) -> AnalyzeResponse:
    """
    Full RCA pipeline for an incident:

    1. Store the incident in memory
    2. Correlate logs (Loki), traces (Jaeger), and metrics (Prometheus)
    3. Run LLM analysis to produce root cause + suggested fix
    4. Optionally create a GitHub hotfix PR
    5. Optionally create a Confluence runbook
    6. Notify via Telegram (if configured)
    """
    t0 = time.monotonic()
    incident_id = str(uuid.uuid4())[:8]

    logger.info(
        "analyze_start",
        incident_id=incident_id,
        service=req.service,
        severity=req.severity,
    )

    # 1. Store incident
    incident_data = {
        "incident_id": incident_id,
        "service": req.service,
        "trace_id": req.trace_id,
        "error": req.error,
        "stack": req.stack,
        "severity": req.severity,
        "sentry_url": req.sentry_url,
        "files": req.files,
    }
    await incident_store.create_incident(incident_data)

    # 2. Correlate observability data
    correlation = await incident_correlator.correlate_incident(
        service=req.service,
        trace_id=req.trace_id,
        time_window=req.time_window,
    )

    # 3. Build LLM prompt and run analysis
    try:
        rca = await _run_llm_analysis(
            incident_id=incident_id,
            service=req.service,
            error=req.error,
            stack=req.stack,
            correlation=correlation,
        )
    except Exception as e:
        logger.error("llm_analysis_failed", error=str(e))
        raise HTTPException(status_code=503, detail="LLM analysis failed")

    pr_url: str | None = None
    runbook_url: str | None = None

    # 4. Auto-create GitHub PR if high confidence and critical severity
    if req.auto_create_pr and rca.get("confidence", 0) >= 0.75 and rca.get("file"):
        pr_url = await _create_hotfix_pr(incident_id, rca, req.service)
        if pr_url:
            logger.info("hotfix_pr_created", pr_url=pr_url)
        await incident_store.update_incident(incident_id, {"pr_url": pr_url})

    # 5. Auto-create Confluence runbook
    if req.auto_create_runbook:
        runbook_url = await _create_runbook(incident_id, req.service, rca, correlation)
        if runbook_url:
            logger.info("runbook_created", url=runbook_url)
        await incident_store.update_incident(incident_id, {"runbook_url": runbook_url})

    # 6. Send Telegram notification
    await _notify_telegram(incident_id, req, rca)

    elapsed = time.monotonic() - t0
    logger.info(
        "analyze_complete",
        incident_id=incident_id,
        confidence=rca.get("confidence", 0),
        pr_url=pr_url,
        runbook_url=runbook_url,
        elapsed_s=round(elapsed, 2),
    )

    # Record metrics
    metrics = METRICS
    if metrics.get("analysis_requests"):
        result_label = "success" if rca.get("confidence", 0) > 0 else "failed"
        metrics["analysis_requests"].labels(tenant_id="default", result=result_label).inc()
    if metrics.get("analysis_latency"):
        metrics["analysis_latency"].observe(elapsed)
    if metrics.get("hotfix_created") and pr_url:
        metrics["hotfix_created"].labels(tenant_id="default", status="created").inc()

    return AnalyzeResponse(
        incident_id=incident_id,
        rca=RCAResult(
            incident_id=incident_id,
            root_cause=rca.get("root_cause", ""),
            why=rca.get("why", ""),
            suggested_fix=rca.get("suggested_fix", ""),
            prevention=rca.get("prevention", ""),
            confidence=rca.get("confidence", 0.0),
            file=rca.get("file"),
            line=rca.get("line"),
            pr_url=pr_url,
            runbook_url=runbook_url,
            notify_sent=True,
        ),
        correlation=correlation,
        pr_url=pr_url,
        runbook_url=runbook_url,
    )


async def _run_llm_analysis(
    incident_id: str,
    service: str,
    error: str,
    stack: str | None,
    correlation: dict[str, Any],
) -> dict[str, Any]:
    """Build the analysis prompt, call vLLM, and parse the response."""
    import httpx

    # Format logs for the prompt
    logs = correlation.get("logs", [])
    log_lines = "\n".join(
        f"[{l.get('labels', {}).get('level', '?')}] {l.get('message', '')[:200]}"
        for l in logs[:30]
    )
    metrics_summary = correlation.get("summary", "")
    traces_summary = _summarize_traces(correlation.get("traces", []))

    prompt = f"""You are an expert Site Reliability Engineer AI. Analyze the incident below and produce a Root Cause Analysis (RCA).

## INCIDENT
- Service: {service}
- Error: {error[:500]}
- Stack trace: {(stack or '(no stack)')[:1500]}

## CORRELATED LOGS (Loki, last 30):
{log_lines[:3000] or '(no logs)'}

## CORRELATED TRACES (Jaeger):
{traces_summary or '(no traces)'}

## METRICS SUMMARY (Prometheus):
{metrics_summary or '(no metrics)'}

Respond with ONLY valid JSON (no markdown, no explanation):
{{
  "root_cause": "1-2 sentence summary of the root cause",
  "why": "Detailed explanation of why this happened",
  "suggested_fix": "Concrete code-level fix (the changed lines with context)",
  "prevention": "Long-term preventive measures",
  "file": "exact file path to modify (e.g. pkg/auth/auth.go)",
  "line": integer line number for the change,
  "confidence": float 0.0-1.0 indicating confidence in this analysis
}}
"""

    async with httpx.AsyncClient(timeout=60.0) as client:
        r = await client.post(
            f"{VLLM_URL}/v1/chat/completions",
            json={
                "model": LLM_MODEL,
                "messages": [{"role": "user", "content": prompt}],
                "max_tokens": 2000,
                "temperature": 0.1,
            },
        )
        r.raise_for_status()
        text = r.json()["choices"][0]["message"]["content"]

    return _parse_rca_response(text)


def _parse_rca_response(text: str) -> dict[str, Any]:
    """Parse JSON RCA response with multiple fallback strategies."""
    # Strategy 1: direct JSON
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        pass

    # Strategy 2: JSON in code block
    m = re.search(r"```(?:json)?\s*(\{.*?\})\s*```", text, re.DOTALL)
    if m:
        try:
            return json.loads(m.group(1))
        except json.JSONDecodeError:
            pass

    # Strategy 3: Any {...} block
    m = re.search(r"\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}", text, re.DOTALL)
    if m:
        try:
            return json.loads(m.group(0))
        except json.JSONDecodeError:
            pass

    # Strategy 4: extract fields with regex
    def field(key: str) -> str:
        m = re.search(rf'"{key}"\s*:\s*"([^"]*)"', text)
        return m.group(1) if m else ""

    def num(key: str) -> int:
        m = re.search(rf'"{key}"\s*:\s*(\d+)', text)
        return int(m.group(1)) if m else 0

    def flt(key: str) -> float:
        m = re.search(rf'"{key}"\s*:\s*([0-9.]+)', text)
        return float(m.group(1)) if m else 0.3

    return {
        "root_cause": field("root_cause") or text[:200],
        "why": field("why") or "Could not parse structured response",
        "suggested_fix": field("suggested_fix") or "",
        "prevention": field("prevention") or "",
        "file": field("file") or None,
        "line": num("line") or None,
        "confidence": flt("confidence") or 0.3,
    }


def _summarize_traces(traces: list[dict[str, Any]]) -> str:
    """Build a short text summary from a list of trace summaries."""
    if not traces:
        return ""
    parts = []
    for t in traces[:5]:
        parts.append(
            f"Trace {t.get('trace_id', '?')}: "
            f"{t.get('span_count', 0)} spans, "
            f"duration={t.get('duration_ms', 0):.1f}ms"
        )
    return "\n".join(parts)


async def _create_hotfix_pr(incident_id: str, rca: dict[str, Any], service: str) -> str | None:
    """Create a GitHub PR with the hotfix if a file path is available."""
    file_path = rca.get("file")
    if not file_path:
        return None

    try:
        # Fetch current file content
        current_content = await github_client.get_file_content(
            repo=GITHUB_REPO,
            path=file_path,
            ref=GITHUB_REF,
        )
        if not current_content:
            return None

        # Apply the fix (simple line insertion)
        new_content = _apply_fix(current_content, rca.get("line", 0), rca.get("suggested_fix", ""))

        # Get SHA of current file
        sha = await _get_file_sha(GITHUB_REPO, file_path, GITHUB_REF)
        if not sha:
            return None

        branch = f"hotfix/ai-{incident_id[:8]}"
        title = f"[AI Hotfix] {rca.get('root_cause', 'SRE fix')[:60]}"
        body = _build_pr_body(rca, service)

        return await github_client.create_pr(
            title=title,
            body=body,
            branch=branch,
            base="main",
            file_path=file_path,
            new_content=new_content,
            sha=sha,
            repo=GITHUB_REPO,
        )
    except Exception as e:
        logger.warning("pr_creation_failed", error=str(e))
        return None


async def _get_file_sha(repo: str, path: str, ref: str) -> str | None:
    """Get the SHA of a file at a given ref."""
    import httpx
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            r = await client.get(
                f"https://api.github.com/repos/{repo}/contents/{path}",
                params={"ref": ref},
                headers={"Authorization": f"Bearer {GITHUB_TOKEN}"},
            )
            r.raise_for_status()
            return r.json().get("sha")
    except Exception:
        return None


def _apply_fix(source: str, line: int, fix_code: str) -> str:
    """Apply fix code at the specified line number (1-indexed)."""
    lines = source.split("\n")
    if line < 1 or line > len(lines) + 1:
        return source
    insert_idx = line - 1
    lines.insert(insert_idx, fix_code)
    return "\n".join(lines)


def _build_pr_body(rca: dict[str, Any], service: str) -> str:
    return f"""## AI SRE Auto-Fix

**Service**: {service}
**Root Cause**: {rca.get('root_cause', '?')}
**Confidence**: {rca.get('confidence', 0):.0%}

### Suggested Fix
```
{rca.get('suggested_fix', '?')}
```

### Why
{rca.get('why', '')}

### Prevention
{rca.get('prevention', '')}

---
Auto-generated by AI SRE Worker. Please review carefully before merging.
"""


async def _create_runbook(
    incident_id: str,
    service: str,
    rca: dict[str, Any],
    correlation: dict[str, Any],
) -> str | None:
    """Create a Confluence runbook page for the incident."""
    html = runbook_writer.build_runbook_html(
        incident_id=incident_id,
        service=service,
        root_cause=rca.get("root_cause", ""),
        suggested_fix=rca.get("suggested_fix", ""),
        prevention=rca.get("prevention", ""),
        metrics=correlation.get("metrics", {}),
        logs=correlation.get("logs", []),
    )
    title = f"[AI SRE] Incident {incident_id} — {service}"
    return await runbook_writer.create_runbook(title=title, content_html=html, space="RINCO")


async def _notify_telegram(incident_id: str, req: AnalyzeRequest, rca: dict[str, Any]) -> None:
    """Send a Telegram notification for the incident."""
    import os
    import httpx
    token = os.getenv("TELEGRAM_BOT_TOKEN", "")
    chat_id = os.getenv("TELEGRAM_CHAT_ID", "")
    if not token or not chat_id:
        return

    msg = f"""🚨 *{req.severity}* — `{req.service}`

*Error*: `{req.error[:300]}`

*RCA*: {rca.get('root_cause', '?')[:800]}

*Fix*:
```
{rca.get('suggested_fix', '?')[:1500]}
```

*Confidence*: {rca.get('confidence', 0):.0%}

Incident ID: `{incident_id}`
Trace: `{req.trace_id or '-'}`
[Sentry]({req.sentry_url or '#'})
"""
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            await client.post(
                f"https://api.telegram.org/bot{token}/sendMessage",
                json={"chat_id": chat_id, "text": msg, "parse_mode": "Markdown"},
            )
    except Exception as e:
        logger.warning("telegram_notify_failed", error=str(e))
