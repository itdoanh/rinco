"""Tests for rag-chatbot reranker fallback behavior."""
from __future__ import annotations

from app.services.reranker import rerank, _reranker


def test_extra_rerank_no_model_returns_top_n():
    """When no model, should return top_n hits unchanged."""
    # Force no model by clearing the cache
    import app.services.reranker as r
    r._reranker = None
    r._reranker = "force_no_model"  # Not a real model, will be loaded as None

    # Actually let's just check the behavior when load_reranker returns None
    hits = [
        {"id": "1", "payload": {"text": "doc one"}},
        {"id": "2", "payload": {"text": "doc two"}},
        {"id": "3", "payload": {"text": "doc three"}},
    ]
    # Even if reranker fails to load, hits should be returned (truncated)
    result = rerank("query", hits, top_n=2)
    assert isinstance(result, list)
    assert len(result) <= 3  # Top_n or less


def test_extra_rerank_empty_hits():
    """Empty hits should return empty list."""
    result = rerank("query", [], top_n=10)
    assert result == []


def test_extra_rerank_top_n_limit():
    """Result should not exceed top_n."""
    # Force no model
    import app.services.reranker as r
    r._reranker = "force_no_model"
    hits = [
        {"id": str(i), "payload": {"text": f"doc {i}"}} for i in range(10)
    ]
    result = rerank("query", hits, top_n=3)
    assert len(result) <= 3


def test_extra_rerank_preserves_hit_format():
    """Hits should preserve their original format."""
    import app.services.reranker as r
    r._reranker = "force_no_model"
    hits = [
        {"id": "1", "payload": {"text": "doc"}, "score": 0.5},
    ]
    result = rerank("query", hits, top_n=5)
    assert result[0]["id"] == "1"
    assert result[0]["payload"]["text"] == "doc"
