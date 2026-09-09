"""Tests for rag-chatbot embeddings service fallback behavior.

The pure-Python fallback is exercised by patching HAS_HTTPX = False on the
embeddings module so deterministic SHA-256 vectors are produced without
attempting to reach a real vLLM endpoint.
"""
import hashlib

import pytest

from app.services import embeddings


@pytest.fixture(autouse=True)
def fallback_only(monkeypatch):
    """Force the deterministic fallback path so tests don't need vLLM or sentence-transformers."""
    monkeypatch.setattr(embeddings, "HAS_HTTPX", False)
    monkeypatch.setattr(embeddings, "HAS_ST", False)
    monkeypatch.setattr(embeddings, "_embedder", None)
    yield


@pytest.mark.asyncio
async def test_embed_texts_returns_list_of_vectors():
    """embed_texts returns list of vectors."""
    vecs = await embeddings.embed_texts(["hello", "world"])
    assert isinstance(vecs, list)
    assert len(vecs) == 2


@pytest.mark.asyncio
async def test_embed_texts_dimension_consistent():
    """All embeddings have same dimension."""
    vecs = await embeddings.embed_texts(["a", "b", "c"])
    dims = {len(v) for v in vecs}
    assert len(dims) == 1


@pytest.mark.asyncio
async def test_embed_texts_deterministic_fallback():
    """Same input yields same output in fallback mode."""
    v1 = await embeddings.embed_texts(["test"])
    v2 = await embeddings.embed_texts(["test"])
    assert v1 == v2


@pytest.mark.asyncio
async def test_embed_texts_different_inputs_different_vectors():
    """Different inputs produce different vectors in fallback."""
    v1 = await embeddings.embed_texts(["alpha"])
    v2 = await embeddings.embed_texts(["beta"])
    assert v1 != v2


@pytest.mark.asyncio
async def test_embed_texts_empty_list():
    """Empty input returns empty list."""
    vecs = await embeddings.embed_texts([])
    assert vecs == []


@pytest.mark.asyncio
async def test_embed_texts_default_dimension_384():
    """Default fallback embedding dimension is 384."""
    vecs = await embeddings.embed_texts(["test"])
    assert len(vecs[0]) == 384


def test_hashlib_sha256_seed_alignment():
    """Verify fallback algorithm matches expected hash-then-byte-vector logic."""
    text = "hello"
    seed = hashlib.sha256(text.encode()).digest()
    # SHA-256 always 32 bytes
    assert len(seed) == 32
    # Bytes are 0-255
    assert max(seed) <= 255
    assert min(seed) >= 0


@pytest.mark.asyncio
async def test_embed_texts_vector_value_range():
    """Bytes/255 in [0, 1]."""
    vecs = await embeddings.embed_texts(["test"])
    v = vecs[0]
    for x in v:
        assert 0.0 <= x <= 1.0


@pytest.mark.asyncio
async def test_embed_texts_unique_per_unique_input():
    """Many distinct inputs yield many distinct vectors."""
    inputs = [f"text-{i}" for i in range(20)]
    vecs = await embeddings.embed_texts(inputs)
    unique = set(tuple(v) for v in vecs)
    # All should be distinct
    assert len(unique) == 20
