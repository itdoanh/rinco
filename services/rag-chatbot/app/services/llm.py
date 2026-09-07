"""LLM service: calls vLLM OpenAI-compatible endpoint."""
from __future__ import annotations

import json
import os
from typing import Any, AsyncGenerator, Dict, List

try:
    import httpx  # type: ignore
    HAS_HTTPX = True
except Exception:  # pragma: no cover
    HAS_HTTPX = False
    httpx = None  # type: ignore

from app.core import VLLM_API_KEY, VLLM_URL, LLM_MODEL, METRICS, get_logger

log = get_logger("rag-chatbot.llm")


async def call_vllm_stream(
    messages: List[Dict[str, str]],
    model: str | None = None,
    temperature: float = 0.2,
    max_tokens: int = 512,
) -> AsyncGenerator[str, None]:
    """Yield raw SSE tokens from the vLLM chat completions stream."""
    if not HAS_HTTPX:
        return
    body = {
        "model": model or LLM_MODEL,
        "messages": messages,
        "stream": True,
        "temperature": temperature,
        "max_tokens": max_tokens,
    }
    headers = {"Content-Type": "application/json"}
    if VLLM_API_KEY:
        headers["Authorization"] = f"Bearer {VLLM_API_KEY}"
    url = VLLM_URL.rstrip("/") + "/chat/completions"
    try:
        async with httpx.AsyncClient(timeout=60) as client:
            async with client.stream("POST", url, json=body, headers=headers) as r:
                async for line in r.aiter_lines():
                    if not line or not line.startswith("data: "):
                        continue
                    yield line[6:]  # strip "data: "
    except Exception as exc:
        log.warning("vllm_stream_failed", error=str(exc))


async def call_vllm_nonstream(
    messages: List[Dict[str, str]],
    model: str | None = None,
    temperature: float = 0.2,
    max_tokens: int = 512,
) -> str:
    """Return the full assistant response."""
    if not HAS_HTTPX:
        return ""
    body = {
        "model": model or LLM_MODEL,
        "messages": messages,
        "stream": False,
        "temperature": temperature,
        "max_tokens": max_tokens,
    }
    headers = {"Content-Type": "application/json"}
    if VLLM_API_KEY:
        headers["Authorization"] = f"Bearer {VLLM_API_KEY}"
    url = VLLM_URL.rstrip("/") + "/chat/completions"
    async with httpx.AsyncClient(timeout=60) as client:
        r = await client.post(url, json=body, headers=headers)
    if r.status_code >= 300:
        raise RuntimeError(f"vllm {r.status_code}: {r.text}")
    return r.json()["choices"][0]["message"]["content"]


def estimate_tokens(text: str) -> int:
    return max(1, len(text) // 4)