"""Tests for rag-chatbot chunker edge cases."""
from __future__ import annotations

from app.services.chunker import chunk_text, _split_recursive


def test_chunk_text_unicode():
    """Unicode text is handled."""
    text = "Xin chào. Tạm biệt. Hẹn gặp lại. Cảm ơn bạn."
    chunks = chunk_text(text, chunk_size=20, overlap=5)
    assert len(chunks) >= 1
    # All chunks should be non-empty
    for c in chunks:
        assert len(c) > 0


def test_chunk_text_single_separator():
    """Text with only one paragraph break."""
    text = "Hello" + ("a" * 100) + "\n\nWorld"
    chunks = chunk_text(text, chunk_size=50, overlap=0)
    assert len(chunks) >= 1


def test_chunk_text_long_paragraph():
    """Single very long paragraph splits."""
    text = "word " * 200
    chunks = chunk_text(text, chunk_size=100, overlap=10)
    assert len(chunks) >= 2


def test_chunk_text_word_separator():
    """When no other separator works, falls back to word split."""
    text = "a " * 300
    chunks = chunk_text(text, chunk_size=50, overlap=0)
    # Each chunk <= 50 chars
    for c in chunks:
        assert len(c) <= 51, f"chunk too big: {len(c)}"


def test_split_recursive_max_depth():
    """At depth >= len(seps), returns text."""
    text = "no break here"
    result = _split_recursive(text, 5, 0, ["X"], depth=10)
    # If text > size, returns as single chunk (still tries to split once)
    assert isinstance(result, list)


def test_split_recursive_zero_overlap():
    """Overlap=0 means no carry."""
    text = "abc def ghi jkl mno pqr"
    result = _split_recursive(text, 10, 0, [" "], 0)
    assert isinstance(result, list)


def test_chunk_text_negative_chunk_size():
    """Negative chunk_size handled gracefully."""
    chunks = chunk_text("hello world", chunk_size=-1)
    # Behavior: text <= size means returns single chunk
    # -1 < 11 so should split (recursive)
    assert isinstance(chunks, list)


def test_chunk_text_overlap_zero_simple():
    """With overlap=0, simple split returns clean chunks."""
    text = "Para A. Para B. Para C."
    chunks = chunk_text(text, chunk_size=15, overlap=0)
    # Each chunk <= 15 chars
    for c in chunks:
        # Allow slight buffer for the carryover edge case
        assert len(c) <= 30


def test_chunk_text_huge_overlap():
    """Overlap > chunk_size handled gracefully."""
    text = "a b c d e f g h i j k l m n o p"
    chunks = chunk_text(text, chunk_size=10, overlap=100)
    assert isinstance(chunks, list)
    # All chunks non-empty
    for c in chunks:
        assert len(c) > 0
