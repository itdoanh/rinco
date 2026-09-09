"""Tests for rag-chatbot llm service."""
from __future__ import annotations

from app.services.llm import estimate_tokens


def test_extra_estimate_tokens_empty():
    """Empty text should still return at least 1 token."""
    assert estimate_tokens("") >= 1


def test_extra_estimate_tokens_short():
    """Short text should be 1 token."""
    assert estimate_tokens("hi") == 1  # 2 chars / 4 = 0, max(1, 0) = 1


def test_extra_estimate_tokens_medium():
    """Medium text should be approximately len/4 tokens."""
    text = "x" * 100
    assert estimate_tokens(text) == 25  # 100 / 4


def test_extra_estimate_tokens_long():
    """Long text should follow len/4 formula."""
    text = "y" * 400
    assert estimate_tokens(text) == 100  # 400 / 4


def test_extra_estimate_tokens_realistic():
    """Realistic English text estimate."""
    text = "The quick brown fox jumps over the lazy dog"
    tokens = estimate_tokens(text)
    # Should be roughly len/4, with min 1
    assert tokens >= 1
    assert tokens == max(1, len(text) // 4)


def test_extra_estimate_tokens_unicode():
    """Unicode text should be estimated."""
    text = "Xin chào"
    tokens = estimate_tokens(text)
    assert tokens >= 1
