"""Tests for rag-chatbot llm vLLM HTTP calls via httpx MockTransport."""
from __future__ import annotations

import json
import os

import httpx
import pytest

# Save env vars so we don't pollute other tests.
os.environ.setdefault("VLLM_URL", "http://vllm.test/v1")
os.environ.setdefault("VLLM_API_KEY", "")
os.environ.setdefault("LLM_MODEL", "test-model")


@pytest.mark.asyncio
async def test_extra_call_vllm_nonstream_returns_content(monkeypatch):
    """call_vllm_nonstream returns choices[0].message.content."""
    from app.services import llm

    captured = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["url"] = str(request.url)
        captured["body"] = json.loads(request.content.decode())
        captured["headers"] = dict(request.headers)
        return httpx.Response(
            200,
            json={
                "choices": [
                    {"message": {"role": "assistant", "content": "Hello there"}}
                ]
            },
        )

    transport = httpx.MockTransport(handler)
    # Patch AsyncClient to use our transport.
    real_init = httpx.AsyncClient.__init__

    def patched_init(self, *args, **kwargs):
        kwargs["transport"] = transport
        real_init(self, *args, **kwargs)

    monkeypatch.setattr(httpx.AsyncClient, "__init__", patched_init)
    out = await llm.call_vllm_nonstream([{"role": "user", "content": "hi"}])
    assert out == "Hello there"
    assert captured["body"]["messages"][0]["content"] == "hi"
    assert captured["body"]["stream"] is False


@pytest.mark.asyncio
async def test_extra_call_vllm_nonstream_uses_bearer_when_key_set(monkeypatch):
    """When VLLM_API_KEY is set, Authorization: Bearer is sent."""
    from app.services import llm
    # Override the module-level constant directly.
    monkeypatch.setattr(llm, "VLLM_API_KEY", "sk-test")

    captured = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["auth"] = request.headers.get("Authorization")
        return httpx.Response(200, json={"choices": [{"message": {"content": "ok"}}]})

    transport = httpx.MockTransport(handler)
    real_init = httpx.AsyncClient.__init__

    def patched_init(self, *args, **kwargs):
        kwargs["transport"] = transport
        real_init(self, *args, **kwargs)

    monkeypatch.setattr(httpx.AsyncClient, "__init__", patched_init)
    out = await llm.call_vllm_nonstream([{"role": "user", "content": "x"}])
    assert out == "ok"
    assert captured["auth"] == "Bearer sk-test"


@pytest.mark.asyncio
async def test_extra_call_vllm_nonstream_raises_on_5xx(monkeypatch):
    """5xx response raises RuntimeError with status code in message."""
    from app.services import llm

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(500, text="upstream down")

    transport = httpx.MockTransport(handler)
    real_init = httpx.AsyncClient.__init__

    def patched_init(self, *args, **kwargs):
        kwargs["transport"] = transport
        real_init(self, *args, **kwargs)

    monkeypatch.setattr(httpx.AsyncClient, "__init__", patched_init)
    with pytest.raises(RuntimeError) as exc:
        await llm.call_vllm_nonstream([{"role": "user", "content": "x"}])
    assert "500" in str(exc.value)


@pytest.mark.asyncio
async def test_extra_call_vllm_stream_yields_lines(monkeypatch):
    """Streaming response yields SSE ``data:`` payloads (prefix stripped)."""
    from app.services import llm

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            headers={"content-type": "text/event-stream"},
            content=b"data: token1\n\ndata: token2\n\n",
        )

    transport = httpx.MockTransport(handler)
    real_init = httpx.AsyncClient.__init__

    def patched_init(self, *args, **kwargs):
        kwargs["transport"] = transport
        real_init(self, *args, **kwargs)

    monkeypatch.setattr(httpx.AsyncClient, "__init__", patched_init)
    out = []
    async for token in llm.call_vllm_stream([{"role": "user", "content": "x"}]):
        out.append(token)
    assert out == ["token1", "token2"]


@pytest.mark.asyncio
async def test_extra_call_vllm_stream_handles_blank_lines(monkeypatch):
    """Blank lines and ``data:`` prefix lines are skipped."""
    from app.services import llm

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            headers={"content-type": "text/event-stream"},
            content=b"\ndata: a\n\n   \ndata: b\n",
        )

    transport = httpx.MockTransport(handler)
    real_init = httpx.AsyncClient.__init__

    def patched_init(self, *args, **kwargs):
        kwargs["transport"] = transport
        real_init(self, *args, **kwargs)

    monkeypatch.setattr(httpx.AsyncClient, "__init__", patched_init)
    out = []
    async for token in llm.call_vllm_stream([{"role": "user", "content": "x"}]):
        out.append(token)
    assert out == ["a", "b"]
