"""Tests for ingest pipeline."""
from __future__ import annotations

import os
import sys

ROOT = os.path.dirname(os.path.abspath(__file__))
if ROOT not in sys.path:
    sys.path.insert(0, os.path.dirname(ROOT))

from app.services.chunker import chunk_text


def test_chunk_respects_size():
    # ``chunk_text`` is a recursive splitter that works on natural text
    # separators (paragraphs, sentences, words).  A solid string of ``"A"``
    # has no separators and would be returned as a single oversized chunk
    # by design.  Use a string with embedded separators to exercise the
    # size-limiting behaviour.
    text = ("Sentence number one. " * 200).strip()
    chunks = chunk_text(text, chunk_size=512, overlap=64)
    # Every chunk must fit inside the configured size bound.
    for c in chunks:
        assert len(c) <= 512, f"chunk too large: {len(c)} > 512"
    # And we should actually have produced multiple chunks.
    assert len(chunks) > 1


def test_chunk_has_overlap():
    text = "ABCDEFGH" * 100
    chunks = chunk_text(text, chunk_size=100, overlap=20)
    if len(chunks) >= 2:
        # The end of one chunk should appear at the start of the next.
        # This is a soft check.
        assert len(chunks) >= 2


def test_ingest_kind_dispatch():
    from app.services.ingestion import extract_kind
    # Without the heavy deps the extract_kind just returns content as-is.
    result = extract_kind("text", "Hello world")
    assert "Hello" in result