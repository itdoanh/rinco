"""Tests for rag-chatbot embeddings fallback behavior."""
from __future__ import annotations

import hashlib

import pytest

from app.services import embeddings


@pytest.mark.asyncio
async def test_extra_embed_texts_no_libs_returns_deterministic(monkeypatch):
    """When neither sentence-transformers nor httpx is available, fall back to deterministic SHA-256-based vectors."""
    monkeypatch.setattr(embeddings, "_embedder", None)
    monkeypatch.setattr(embeddings, "HAS_ST", False)
    monkeypatch.setattr(embeddings, "HAS_HTTPX", False)
    out = await embeddings.embed_texts(["hello", "world"])
    assert len(out) == 2
    # Deterministic: same text → same vector.
    again = await embeddings.embed_texts(["hello"])
    assert out[0] == again[0]
    # Different texts → different vectors.
    assert out[0] != out[1]


@pytest.mark.asyncio
async def test_extra_embed_texts_dim_is_384(monkeypatch):
    """Fallback vectors are 384-dimensional."""
    monkeypatch.setattr(embeddings, "_embedder", None)
    monkeypatch.setattr(embeddings, "HAS_ST", False)
    monkeypatch.setattr(embeddings, "HAS_HTTPX", False)
    out = await embeddings.embed_texts(["test"])
    assert len(out[0]) == 384


@pytest.mark.asyncio
async def test_extra_embed_texts_values_in_unit_interval(monkeypatch):
    """Fallback values are normalized to [0, 1]."""
    monkeypatch.setattr(embeddings, "_embedder", None)
    monkeypatch.setattr(embeddings, "HAS_ST", False)
    monkeypatch.setattr(embeddings, "HAS_HTTPX", False)
    out = await embeddings.embed_texts(["test"])
    for v in out[0]:
        assert 0.0 <= v <= 1.0


@pytest.mark.asyncio
async def test_extra_embed_texts_uses_sha256_seed(monkeypatch):
    """First 4 values of fallback vector equal hash bytes / 255."""
    monkeypatch.setattr(embeddings, "_embedder", None)
    monkeypatch.setattr(embeddings, "HAS_ST", False)
    monkeypatch.setattr(embeddings, "HAS_HTTPX", False)
    text = "my text"
    expected_seed = hashlib.sha256(text.encode()).digest()
    expected_first4 = [expected_seed[i] / 255.0 for i in range(4)]
    out = await embeddings.embed_texts([text])
    actual_first4 = out[0][:4]
    for exp, got in zip(expected_first4, actual_first4):
        assert abs(exp - got) < 1e-9


@pytest.mark.asyncio
async def test_extra_embed_texts_empty_list(monkeypatch):
    """Empty input returns empty list."""
    monkeypatch.setattr(embeddings, "_embedder", None)
    monkeypatch.setattr(embeddings, "HAS_ST", False)
    monkeypatch.setattr(embeddings, "HAS_HTTPX", False)
    out = await embeddings.embed_texts([])
    assert out == []
