"""Extra tests for rag-chatbot schemas."""
from __future__ import annotations

import pytest
from pydantic import ValidationError

from app.schemas import (
    ChatRequest,
    ChatResponse,
    Citation,
    IngestRequest,
    IngestResponse,
    SearchRequest,
    SearchHit,
    SearchResponse,
    FeedbackRequest,
    CreateCollectionRequest,
)


def test_chat_request_minimal():
    req = ChatRequest(query="hello")
    assert req.query == "hello"
    assert req.history == []
    assert req.collection is None
    assert req.temperature == 0.2
    assert req.top_k == 20
    assert req.stream is True
    assert req.metadata_filter == {}


def test_chat_request_with_history():
    req = ChatRequest(query="hi", history=[{"role": "user", "content": "hello"}])
    assert len(req.history) == 1
    assert req.history[0]["role"] == "user"


def test_chat_request_empty_query_rejected():
    with pytest.raises(ValidationError):
        ChatRequest(query="")


def test_chat_request_collection_optional():
    req = ChatRequest(query="x", collection="docs")
    assert req.collection == "docs"


def test_chat_request_temperature_range_accepted():
    req = ChatRequest(query="x", temperature=1.5)
    assert req.temperature == 1.5


def test_citation_required_fields():
    c = Citation(chunk_id="c1", score=0.95, text="hello")
    assert c.chunk_id == "c1"
    assert c.score == 0.95
    assert c.text == "hello"
    assert c.source_url is None
    assert c.doc_type is None
    assert c.span is None


def test_citation_optional_fields():
    c = Citation(
        chunk_id="c2", score=0.8, text="x",
        source_url="https://example.com",
        doc_type="pdf",
        span={"start": 0, "end": 10},
    )
    assert c.source_url == "https://example.com"
    assert c.doc_type == "pdf"
    assert c.span["start"] == 0


def test_chat_response_defaults():
    r = ChatResponse(answer="ok", model="m", tokens_used=10, latency_ms=100, collection="c")
    assert r.answer == "ok"
    assert r.citations == []
    assert r.feedback_id is None


def test_chat_response_with_citations():
    c = Citation(chunk_id="c1", score=0.9, text="t")
    r = ChatResponse(
        answer="ans",
        citations=[c],
        model="model",
        tokens_used=50,
        latency_ms=200,
        collection="col",
    )
    assert len(r.citations) == 1
    assert r.citations[0].chunk_id == "c1"


def test_ingest_request_defaults():
    req = IngestRequest(source="text content")
    assert req.source == "text content"
    assert req.kind == "text"
    assert req.collection is None
    assert req.metadata == {}


def test_ingest_request_with_metadata():
    req = IngestRequest(source="x", kind="pdf", collection="docs", metadata={"author": "alice"})
    assert req.kind == "pdf"
    assert req.metadata["author"] == "alice"


def test_ingest_response():
    r = IngestResponse(doc_id="doc1", chunks_created=10, collection="c", bytes_ingested=1000, duration_ms=500)
    assert r.doc_id == "doc1"
    assert r.chunks_created == 10
    assert r.duration_ms == 500


def test_search_request_defaults():
    r = SearchRequest(query="search")
    assert r.top_k == 20
    assert r.collection is None
    assert r.metadata_filter == {}


def test_search_request_custom_topk():
    r = SearchRequest(query="x", top_k=100)
    assert r.top_k == 100


def test_search_hit_required():
    h = SearchHit(chunk_id="c1", score=0.5, text="text")
    assert h.chunk_id == "c1"
    assert h.source_url is None
    assert h.metadata == {}


def test_search_hit_with_metadata():
    h = SearchHit(chunk_id="c1", score=0.5, text="t", source_url="https://x.com", metadata={"k": "v"})
    assert h.metadata["k"] == "v"


def test_search_response():
    h = SearchHit(chunk_id="c1", score=0.9, text="t")
    r = SearchResponse(hits=[h], collection="c", query="q")
    assert len(r.hits) == 1
    assert r.query == "q"


def test_feedback_request_required():
    req = FeedbackRequest(query="q", answer="a", rating=5)
    assert req.rating == 5
    assert req.chat_id is None
    assert req.comment is None


def test_feedback_request_with_comment():
    req = FeedbackRequest(query="q", answer="a", rating=4, comment="good")
    assert req.comment == "good"
    assert req.chat_id is None


def test_feedback_request_with_chat_id():
    req = FeedbackRequest(chat_id="chat-1", query="q", answer="a", rating=3)
    assert req.chat_id == "chat-1"


def test_create_collection_defaults():
    req = CreateCollectionRequest()
    assert req.name is None
    assert req.vector_size == 1024
    assert req.distance == "Cosine"


def test_create_collection_custom():
    req = CreateCollectionRequest(name="docs", vector_size=384, distance="Euclid")
    assert req.name == "docs"
    assert req.vector_size == 384
    assert req.distance == "Euclid"
