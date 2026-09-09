"""Hotfix generator using vLLM DeepSeekSeek-Coder."""
from __future__ import annotations

import json
import os
import re
from typing import Any

import httpx

from app.core import get_logger

logger = get_logger("hotfix_generator")

# Re-export constants so callers can import from app.services without import cycles
from app.core import VLLM_URL as _VLLM_URL
from app.core import LLM_MODEL as _LLM_MODEL


async def generate_hotfix(
    error: str,
    stack: str | None,
    source: str,
    context: str,
) -> dict[str, Any]:
    """
    Generate a code-level hotfix suggestion using vLLM.

    Builds a structured prompt from the incident context, sends it to the
    vLLM DeepSeek-Coder endpoint, and parses the response for a diff.

    Args:
        error: Error message or exception text.
        stack: Optional stack trace string.
        source: Current source code around the error location.
        context: Additional context (logs, metrics summary, etc.).

    Returns:
        {
            "diff": str,          # Unified diff of the suggested change
            "file": str,          # File path targeted for the change
            "line": int,          # Approximate line number for the change
            "confidence": float,  # 0.0-1.0 confidence score
        }
    """
    prompt = _build_hotfix_prompt(error, stack, source, context)

    try:
        response_text = await _call_vllm(prompt)
        parsed = _parse_hotfix_response(response_text)
        logger.info("hotfix_generated", confidence=parsed.get("confidence", 0))
        return parsed
    except Exception as e:
        logger.warning("hotfix_generation_failed", error=str(e))
        return {
            "diff": "",
            "file": "",
            "line": 0,
            "confidence": 0.0,
        }


def _build_hotfix_prompt(error: str, stack: str | None, source: str, context: str) -> str:
    """Build the prompt sent to DeepSeek-Coder for hotfix generation."""
    stack_snippet = (stack or "(no stack trace)")[:1500]
    source_snippet = source[:6000] if source else "(source not available)"
    context_snippet = context[:3000] if context else "(no additional context)"

    return f"""You are an expert Site Reliability Engineer AI. Given the incident below, generate a precise code-level hotfix.

## INCIDENT
Error: {error[:500]}
Stack trace: {stack_snippet}

## SOURCE CODE (context around error)
```
{source_snippet}
```

## ADDITIONAL CONTEXT
{context_snippet}

## TASK
Analyze the error and source code, then output a JSON object describing the minimal fix.
Respond with ONLY valid JSON, no markdown or extra text:
{{
  "root_cause": "1-2 sentence root cause summary",
  "diff": "unified diff showing the changes (e.g. -old line\\n+new line)",
  "file": "exact file path to modify",
  "line": integer line number where the change should start,
  "confidence": float 0.0-1.0 indicating confidence in the fix
}}
"""


async def _call_vllm(prompt: str) -> str:
    """Call the vLLM chat completions endpoint."""
    async with httpx.AsyncClient(timeout=60.0) as client:
        r = await client.post(
            f"{_VLLM_URL}/v1/chat/completions",
            json={
                "model": _LLM_MODEL,
                "messages": [{"role": "user", "content": prompt}],
                "max_tokens": 2000,
                "temperature": 0.1,
            },
        )
        r.raise_for_status()
        return r.json()["choices"][0]["message"]["content"]


DEFAULT_HOTFIX_KEYS = ("diff", "file", "line", "confidence", "root_cause")


def _merge_defaults(parsed: dict[str, Any]) -> dict[str, Any]:
    """Return a copy of ``parsed`` with all known keys filled with defaults.

    The LLM response is best-effort: it may omit fields.  Callers should
    always be able to rely on the canonical schema, with sensible defaults
    for any missing keys.
    """
    result = {
        "diff": "",
        "file": "",
        "line": 0,
        "confidence": 0.3,
        "root_cause": "",
    }
    for key in DEFAULT_HOTFIX_KEYS:
        if key in parsed and parsed[key] is not None:
            result[key] = parsed[key]
    # Preserve any extra keys the model returned (callers may want to inspect).
    for key, value in parsed.items():
        if key not in result:
            result[key] = value
    return result


def _parse_hotfix_response(text: str) -> dict[str, Any]:
    """Parse JSON hotfix response from LLM with multiple fallback strategies."""
    parsed: dict[str, Any] | None = None

    # Strategy 1: Direct JSON parse
    try:
        parsed = json.loads(text)
    except json.JSONDecodeError:
        pass

    # Strategy 2: Extract JSON from code block
    if parsed is None:
        json_match = re.search(r"```(?:json)?\s*(\{.*?\})\s*```", text, re.DOTALL)
        if json_match:
            try:
                parsed = json.loads(json_match.group(1))
            except json.JSONDecodeError:
                pass

    # Strategy 3: Extract any {...} block
    if parsed is None:
        brace_match = re.search(r"\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}", text, re.DOTALL)
        if brace_match:
            try:
                parsed = json.loads(brace_match.group(0))
            except json.JSONDecodeError:
                # Try to coerce JavaScript-style object literals (unquoted keys)
                # into valid JSON by quoting the keys.
                coerced = _coerce_unquoted_keys(brace_match.group(0))
                try:
                    parsed = json.loads(coerced)
                except json.JSONDecodeError:
                    pass

    if parsed is not None and isinstance(parsed, dict):
        return _merge_defaults(parsed)

    # Strategy 4: Fallback — extract known fields via regex
    # Match both quoted keys ("diff":) and unquoted keys (diff:)
    diff_match = re.search(r'(?:"diff"|diff)\s*:\s*"([^"]*)"', text, re.DOTALL)
    file_match = re.search(r'(?:"file"|file)\s*:\s*"([^"]*)"', text)
    line_match = re.search(r'(?:"line"|line)\s*:\s*(\d+)', text)
    conf_match = re.search(r'(?:"confidence"|confidence)\s*:\s*([0-9.]+)', text)
    root_match = re.search(r'(?:"root_cause"|root_cause)\s*:\s*"([^"]*)"', text)

    return {
        "diff": (diff_match.group(1) if diff_match else ""),
        "file": (file_match.group(1) if file_match else ""),
        "line": (int(line_match.group(1)) if line_match else 0),
        "confidence": (float(conf_match.group(1)) if conf_match else 0.3),
        "root_cause": (
            root_match.group(1) if root_match else (text[:500].strip() if text else "")
        ),
    }


def _coerce_unquoted_keys(blob: str) -> str:
    """Quote bare identifier keys in a JS-style object literal.

    Example:
        >>> _coerce_unquoted_keys('{ a: 1, b: "two" }')
        '{ "a": 1, "b": "two" }'
    """
    # Match identifier-like words followed by ``:`` that are not already
    # inside double quotes.  This is intentionally permissive: it will not
    # attempt to handle string values containing colons.
    return re.sub(
        r"([\{,]\s*)([A-Za-z_][A-Za-z0-9_]*)(\s*:)",
        r'\1"\2"\3',
        blob,
    )
