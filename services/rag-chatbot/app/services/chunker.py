"""Recursive character text splitter for RAG chunking."""
from __future__ import annotations

from typing import List


def chunk_text(text: str, chunk_size: int = 512, overlap: int = 64) -> List[str]:
    """Split text into chunks of ~chunk_size tokens using recursive separators."""
    if not text:
        return []
    if len(text) <= chunk_size:
        return [text]
    return _split_recursive(
        text, chunk_size, overlap,
        ["\n\n", "\n", ". ", " "], depth=0
    )


def _split_recursive(
    text: str,
    size: int,
    overlap: int,
    seps: List[str],
    depth: int,
) -> List[str]:
    if not text:
        return []
    if len(text) <= size or depth >= len(seps):
        return [text] if text else []
    sep = seps[depth]
    parts = text.split(sep)
    chunks: List[str] = []
    cur = ""
    for p in parts:
        piece = (p + sep) if cur else p
        if len(cur) + len(piece) <= size:
            cur += piece
        else:
            if cur:
                chunks.append(cur)
                # carry overlap
                if overlap > 0 and len(cur) > overlap:
                    cur = cur[-overlap:] + piece
                else:
                    cur = piece
            else:
                # single piece > size
                chunks.extend(_split_recursive(p, size, overlap, seps, depth + 1))
                cur = ""
    if cur:
        chunks.append(cur)
    return chunks