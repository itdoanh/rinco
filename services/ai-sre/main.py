"""
AI SRE Worker - Root Cause Analysis + Auto Hotfix.

Nhận incident từ Sentry, phân tích bằng LLM, tạo RCA report + auto-PR.
"""
import asyncio
import base64
import json
import os
import re
from datetime import datetime
from typing import Any

import httpx
from fastapi import BackgroundTasks, FastAPI, HTTPException
from pydantic import BaseModel
import structlog

logger = structlog.get_logger()
app = FastAPI(title="AI SRE Worker", version="1.0.0")


class Incident(BaseModel):
    incident_id: str
    service: str
    trace_id: str | None = None
    error: str
    stack: str | None = None
    severity: str = "P2"
    sentry_url: str | None = None
    files: list[dict[str, Any]] = []  # [{path, line}]


class RCAResult(BaseModel):
    incident_id: str
    root_cause: str
    why: str
    suggested_fix: str
    prevention: str
    confidence: float
    file: str | None = None
    line: int | None = None
    pr_url: str | None = None
    notify_sent: bool = False


async def fetch_logs(trace_id: str, clickhouse_url: str) -> list[dict[str, Any]]:
    """ClickHouse query để lấy logs của trace."""
    if not trace_id:
        return []
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            r = await client.get(
                f"{clickhouse_url}/",
                params={
                    "query": f"SELECT timestamp, level, message, caller_file, caller_line FROM rinco_logs.app_logs WHERE trace_id = '{trace_id}' ORDER BY timestamp LIMIT 50 FORMAT JSON",
                },
                headers={"X-ClickHouse-User": "readonly"},
            )
            r.raise_for_status()
            return r.json().get("data", [])
    except Exception as e:
        logger.warning("logs_fetch_failed", error=str(e))
        return []


async def fetch_source(file: str, line: int, github_token: str, repo: str, ref: str) -> str:
    """Read source via GitHub API."""
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            r = await client.get(
                f"https://api.github.com/repos/{repo}/contents/{file}?ref={ref}",
                headers={"Authorization": f"Bearer {github_token}"},
            )
            r.raise_for_status()
            return base64.b64decode(r.json()["content"]).decode("utf-8", errors="ignore")
    except Exception as e:
        logger.warning("source_fetch_failed", file=file, error=str(e))
        return ""


def parse_stack(stack: str) -> tuple[str | None, int | None]:
    """Parse Go/Rust/Python stack trace to find file:line."""
    patterns = [
        r"([\w/\-_]+\.go):(\d+)",   # Go
        r"([\w/\-_]+\.rs):(\d+):\d+",  # Rust
        r'File "([\w/\-_]+\.py)", line (\d+)',  # Python
        r"([\w/\-_]+\.(?:ts|tsx|js|jsx)):(\d+)",  # TS/JS
    ]
    for p in patterns:
        m = re.search(p, stack or "")
        if m:
            return m.group(1), int(m.group(2))
    return None, None


def build_prompt(error: str, stack: str | None, logs: list[dict], source: str) -> str:
    log_lines = "\n".join(
        f"[{l.get('level', '?')}] {l.get('message', '')[:200]} ({l.get('caller_file', '')}:{l.get('caller_line', '')})"
        for l in logs[:30]
    )
    src_excerpt = source[:5000] if source else "(source not available)"
    return f"""You are an expert Site Reliability Engineer AI. Analyze the incident below and provide a Root Cause Analysis (RCA).

INCIDENT:
- Error: {error[:500]}
- Stack trace: {(stack or '(no stack)')[:1500]}

RECENT LOGS (last 50):
{log_lines[:3000]}

SOURCE (relevant snippet):
```
{src_excerpt}
```

Respond with ONLY a valid JSON object, no markdown:
{{
  "root_cause": "1-2 sentence summary of the root cause",
  "why": "Detailed explanation why this happened",
  "suggested_fix": "Concrete code fix (only the changed lines, with brief context)",
  "prevention": "Long-term preventive measures",
  "file": "exact path to file to modify (e.g. pkg/auth/paseto.go)",
  "line": integer line number,
  "confidence": float 0.0-1.0 indicating your confidence
}}

Be concise and technical."""


async def call_llm(prompt: str, vllm_url: str, model: str) -> str:
    """Call vLLM serving DeepSeek-Coder."""
    try:
        async with httpx.AsyncClient(timeout=60.0) as client:
            r = await client.post(
                f"{vllm_url}/v1/chat/completions",
                json={
                    "model": model,
                    "messages": [{"role": "user", "content": prompt}],
                    "max_tokens": 2000,
                    "temperature": 0.1,
                },
            )
            r.raise_for_status()
            return r.json()["choices"][0]["message"]["content"]
    except Exception as e:
        logger.error("llm_call_failed", error=str(e))
        raise


def parse_llm_response(text: str) -> dict:
    """Parse JSON from LLM response, fallback to regex."""
    # Try direct JSON parse
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        pass

    # Try extracting JSON block
    json_match = re.search(r"\{.*\}", text, re.DOTALL)
    if json_match:
        try:
            return json.loads(json_match.group(0))
        except json.JSONDecodeError:
            pass

    # Fallback
    return {
        "root_cause": text[:500],
        "why": "Could not parse structured response from LLM",
        "suggested_fix": text[500:1500] if len(text) > 500 else "",
        "prevention": "",
        "file": None,
        "line": None,
        "confidence": 0.3,
    }


async def notify_telegram(incident: Incident, rca: dict):
    """Send Telegram notification."""
    token = os.getenv("TELEGRAM_BOT_TOKEN", "")
    chat_id = os.getenv("TELEGRAM_CHAT_ID", "")
    if not token or not chat_id:
        return
    msg = f"""🚨 *{incident.severity}* – `{incident.service}`

*Error*: `{incident.error[:300]}`

*RCA*: {rca.get('root_cause', '?')[:800]}

*Fix*:
```
{rca.get('suggested_fix', '?')[:1500]}
```

*Confidence*: {rca.get('confidence', 0):.0%}

Trace: `{incident.trace_id or '-'}`
[Open Sentry]({incident.sentry_url or '#'})
"""
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            await client.post(
                f"https://api.telegram.org/bot{token}/sendMessage",
                json={"chat_id": chat_id, "text": msg, "parse_mode": "Markdown"},
            )
    except Exception as e:
        logger.warning("telegram_notify_failed", error=str(e))


async def auto_create_pr(rca: dict, source: str, github_token: str, repo: str):
    """Create a GitHub PR with the suggested fix."""
    branch = f"hotfix/ai-{rca.get('incident_id', 'unknown')[:8]}"
    file_path = rca.get("file")
    if not file_path:
        return None

    try:
        async with httpx.AsyncClient(timeout=30.0) as client:
            # 1. Get default branch SHA
            r = await client.get(
                f"https://api.github.com/repos/{repo}/git/refs/heads/main",
                headers={"Authorization": f"Bearer {github_token}"},
            )
            base_sha = r.json()["object"]["sha"]

            # 2. Create new branch
            await client.post(
                f"https://api.github.com/repos/{repo}/git/refs",
                headers={"Authorization": f"Bearer {github_token}"},
                json={"ref": f"refs/heads/{branch}", "sha": base_sha},
            )

            # 3. Update file
            r = await client.get(
                f"https://api.github.com/repos/{repo}/contents/{file_path}?ref=main",
                headers={"Authorization": f"Bearer {github_token}"},
            )
            current_sha = r.json()["sha"]
            current_content = base64.b64decode(r.json()["content"]).decode("utf-8")

            new_content = apply_fix(current_content, rca.get("line", 0), rca.get("suggested_fix", ""))

            await client.put(
                f"https://api.github.com/repos/{repo}/contents/{file_path}",
                headers={"Authorization": f"Bearer {github_token}"},
                json={
                    "message": f"fix: {rca.get('root_cause', 'AI fix')[:60]}",
                    "branch": branch,
                    "content": base64.b64encode(new_content.encode()).decode(),
                    "sha": current_sha,
                },
            )

            # 4. Create PR
            r = await client.post(
                f"https://api.github.com/repos/{repo}/pulls",
                headers={"Authorization": f"Bearer {github_token}"},
                json={
                    "title": f"[AI Hotfix] {rca.get('root_cause', 'Fix')[:60]}",
                    "head": branch,
                    "base": "main",
                    "body": f"""## AI SRE Auto-Fix

**Root Cause**: {rca.get('root_cause')}
**Confidence**: {rca.get('confidence', 0):.0%}

### Suggested Fix
```diff
{rca.get('suggested_fix', '')}
```

### Prevention
{rca.get('prevention', '')}

---
Auto-generated by AI SRE Worker. Please review carefully before merging.
""",
                },
            )
            return r.json().get("html_url")
    except Exception as e:
        logger.error("auto_pr_failed", error=str(e))
        return None


def apply_fix(source: str, line: int, fix_code: str) -> str:
    """Apply fix at specific line. Simple version: just append comment."""
    lines = source.split("\n")
    if line < 1 or line > len(lines):
        return source
    lines.insert(line, fix_code)
    return "\n".join(lines)


@app.post("/analyze", response_model=RCAResult)
async def analyze(incident: Incident, bg: BackgroundTasks):
    """Trigger RCA ngay."""
    logger.info(
        "incident_received",
        incident_id=incident.incident_id,
        service=incident.service,
        severity=incident.severity,
    )

    # Load config
    vllm_url = os.getenv("VLLM_URL", "http://vllm:8000")
    model = os.getenv("LLM_MODEL", "deepseek-coder-v2-lite-instruct")
    github_token = os.getenv("GITHUB_TOKEN", "")
    repo = os.getenv("GITHUB_REPO", "itdoanh/rinco")
    ref = os.getenv("GITHUB_REF", "main")
    clickhouse_url = os.getenv("CLICKHOUSE_URL", "http://clickhouse:8123")

    # 1. Gather context
    logs = await fetch_logs(incident.trace_id or "", clickhouse_url)
    file, line = parse_stack(incident.stack or "")
    if not file and incident.files:
        file = incident.files[0].get("path")
        line = incident.files[0].get("line", 0)
    source = ""
    if file and line:
        source = await fetch_source(file, line, github_token, repo, ref)

    # 2. LLM analysis
    prompt = build_prompt(incident.error, incident.stack, logs, source)
    try:
        llm_response = await call_llm(prompt, vllm_url, model)
    except Exception as e:
        logger.error("llm_failed", error=str(e))
        raise HTTPException(status_code=503, detail="LLM unavailable")

    rca = parse_llm_response(llm_response)
    rca["incident_id"] = incident.incident_id
    rca.setdefault("file", file)
    rca.setdefault("line", line)

    # 3. Notify (Telegram)
    await notify_telegram(incident, rca)
    rca["notify_sent"] = True

    # 4. Auto-PR if confidence high and severity critical
    if rca.get("confidence", 0) > 0.8 and incident.severity in ("P0", "P1") and rca.get("file"):
        pr_url = await auto_create_pr(rca, source, github_token, repo)
        rca["pr_url"] = pr_url

    return RCAResult(**rca)


@app.get("/health")
async def health():
    return {"status": "ok", "service": "ai-sre"}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8090)
