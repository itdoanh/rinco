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


def _parse_hotfix_response(text: str) -> dict[str, Any]:
    """Parse JSON hotfix response from LLM with multiple fallback strategies."""
    # Strategy 1: Direct JSON parse
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        pass

    # Strategy 2: Extract JSON from code block
    json_match = re.search(r"```(?:json)?\s*(\{.*?\})\s*```", text, re.DOTALL)
    if json_match:
        try:
            return json.loads(json_match.group(1))
        except json.JSONDecodeError:
            pass

    # Strategy 3: Extract any {...} block
    brace_match = re.search(r"\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}", text, re.DOTALL)
    if brace_match:
        try:
            return json.loads(brace_match.group(0))
        except json.JSONDecodeError:
            pass

    # Strategy 4: Fallback — extract known fields via regex
    diff_match = re.search(r'"diff"\s*:\s*"([^"]*)"', text, re.DOTALL)
    file_match = re.search(r'"file"\s*:\s*"([^"]*)"', text)
    line_match = re.search(r'"line"\s*:\s*(\d+)', text)
    conf_match = re.search(r'"confidence"\s*:\s*([0-9.]+)', text)
    root_match = re.search(r'"root_cause"\s*:\s*"([^"]*)"', text)

    return {
        "diff": (diff_match.group(1) if diff_match else ""),
        "file": (file_match.group(1) if file_match else ""),
        "line": (int(line_match.group(1)) if line_match else 0),
        "confidence": (float(conf_match.group(1)) if conf_match else 0.3),
        "root_cause": (root_match.group(1) if root_match else ""),
    }
