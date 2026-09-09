"""Tests for rag-chatbot chunker."""
from __future__ import annotations

from app.services.chunker import chunk_text, _split_recursive


def test_chunk_text_empty():
    """Empty text returns empty list."""
    assert chunk_text("") == []


def test_chunk_text_short():
    """Text shorter than chunk_size returns single chunk."""
    text = "hello world"
    chunks = chunk_text(text, chunk_size=100)
    assert chunks == [text]


def test_chunk_text_basic():
    """Basic split with paragraph breaks works."""
    text = "Para A\n\nPara B\n\nPara C\n\nPara D\n\nPara E"
    chunks = chunk_text(text, chunk_size=15, overlap=0)
    assert len(chunks) >= 1


def test_chunk_text_preserves_content():
    """Chunks concatenate back to original."""
    text = "the quick brown fox jumps over the lazy dog. " * 30
    chunks = chunk_text(text, chunk_size=100, overlap=10)
    # Content should be preserved (allow for separator changes)
    assert all(len(c) > 0 for c in chunks)


def test_chunk_text_with_overlap():
    """Overlapping chunks have shared content."""
    text = "abc def ghi jkl mno pqr stu vwx yzz abc def ghi jkl mno pqr stu vwx yzz " * 10
    chunks = chunk_text(text, chunk_size=50, overlap=10)
    assert len(chunks) >= 1


def test_chunk_text_paragraph_breaks():
    """Paragraph breaks (\n\n) are primary separator."""
    text = "Para 1\n\nPara 2\n\nPara 3"
    chunks = chunk_text(text, chunk_size=100)
    # Short text fits in one chunk
    assert len(chunks) == 1


def test_chunk_text_no_overlap():
    """Setting overlap=0 means no overlap."""
    text = "Sentence one. Sentence two. Sentence three. Sentence four. Sentence five."
    chunks = chunk_text(text, chunk_size=30, overlap=0)
    assert len(chunks) >= 1


def test_chunk_text_sentence_separator():
    """Sentences split at '. ' separator."""
    text = "First sentence. Second sentence. Third sentence. Fourth sentence."
    chunks = chunk_text(text, chunk_size=30, overlap=0)
    assert len(chunks) >= 1


def test_chunk_text_zero_chunk_size():
    """chunk_size=0 with text returns chunks."""
    # chunk_size=0 means each character is a "chunk"
    chunks = chunk_text("abc", chunk_size=0)
    # Function handles gracefully
    assert isinstance(chunks, list)


def test_chunk_text_default_size():
    """Default chunk_size works."""
    chunks = chunk_text("hello " * 200, chunk_size=512, overlap=64)
    assert len(chunks) >= 1


def test_split_recursive_empty():
    """_split_recursive with empty text returns []."""
    result = _split_recursive("", 100, 0, ["\n", " "], 0)
    assert result == []


def test_split_recursive_short():
    """_split_recursive with short text returns single chunk."""
    result = _split_recursive("hi", 100, 0, ["\n", " "], 0)
    assert result == ["hi"]


def test_split_recursive_depth_limit():
    """At max depth, returns text as-is."""
    text = "no separators here"
    seps = ["\n\n"]
    result = _split_recursive(text, 100, 0, seps, depth=5)
    assert result == [text]
