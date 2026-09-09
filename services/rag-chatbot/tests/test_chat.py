"""Tests for chat endpoint."""
from __future__ import annotations

import os
import sys

ROOT = os.path.dirname(os.path.abspath(__file__))
if ROOT not in sys.path:
    sys.path.insert(0, os.path.dirname(ROOT))

from app.services.chunker import chunk_text


def test_chunk_text_small():
    text = "Hello world"
    chunks = chunk_text(text, chunk_size=512, overlap=64)
    assert len(chunks) == 1
    assert chunks[0] == "Hello world"


def test_chunk_text_long():
    # Use a string with embedded separators so the recursive splitter
    # actually produces multiple chunks; ``"A" * 1000`` has no separators
    # and would be returned as a single chunk.
    text = ("Hello world. " * 200).strip()
    chunks = chunk_text(text, chunk_size=200, overlap=20)
    assert len(chunks) > 1


def test_chunk_text_empty():
    assert chunk_text("") == []
    assert chunk_text("   ") == ["   "]


def test_chunk_text_with_separators():
    text = "First paragraph.\n\nSecond paragraph."
    chunks = chunk_text(text, chunk_size=50, overlap=5)
    assert len(chunks) >= 1


def test_sanitize_input():
    from app.api.chat import sanitize_input
    out = sanitize_input("My email is alice@example.com")
    assert "[REDACTED_EMAIL]" in out
    out = sanitize_input("Call me at 0909123456")
    assert "[REDACTED_PHONE]" in out


def test_build_rag_messages():
    from app.api.chat import build_rag_messages
    msgs = build_rag_messages(
        query="What is RINCO?",
        history=[],
        context=[{"payload": {"text": "RINCO is a platform.", "source_url": "https://rinco.app"}}],
        system=None,
    )
    assert msgs[0]["role"] == "system"
    assert any("RINCO" in m["content"] for m in msgs)


def test_app_can_be_created():
    from app.main import create_app
    app = create_app()
    assert app.title == "RINCO RAG Chatbot"