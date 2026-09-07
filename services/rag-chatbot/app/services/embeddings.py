"""Embedding service: BGE-M3 via sentence-transformers or vLLM."""
from __future__ import annotations

import hashlib
import os
from typing import Any, List

from app.core import EMBEDDING_MODEL, METRICS, get_logger

log = get_logger("rag-chatbot.embeddings")

# ---------------------------------------------------------------- optional imports
try:
    from sentence_transformers import SentenceTransformer  # type: ignore
    HAS_ST = True
except Exception:  # pragma: no cover
    HAS_ST = False
    SentenceTransformer = None  # type: ignore

try:
    import httpx  # type: ignore
    HAS_HTTPX = True
except Exception:  # pragma: no cover
    HAS_HTTPX = False
    httpx = None  # type: ignore


_embedder: Any = None


def load_embedder() -> Any:
    global _embedder
    if _embedder is not None:
        return _embedder
    if HAS_ST:
        try:
            log.info("loading_embedder", model=EMBEDDING_MODEL)
            _embedder = SentenceTransformer(EMBEDDING_MODEL)
            return _embedder
        except Exception as exc:
            log.warning("embedder_load_failed", error=str(exc))
    return None


async def embed_texts(texts: List[str]) -> List[List[float]]:
    """Return a list of dense vectors (one per text)."""
    embedder = load_embedder()
    if embedder is not None:
        try:
            vecs = embedder.encode(texts, normalize_embeddings=True)
            return vecs.tolist()
        except Exception as exc:
            log.warning("embed_encode_failed", error=str(exc))

    if HAS_HTTPX:
        out: List[List[float]] = []
        url = os.getenv("VLLM_URL", "http://vllm:8000/v1").rstrip("/") + "/embeddings"
        api_key = os.getenv("VLLM_API_KEY", "")
        headers = {"Content-Type": "application/json"}
        if api_key:
            headers["Authorization"] = f"Bearer {api_key}"
        async with httpx.AsyncClient(timeout=30) as client:
            for t in texts:
                r = await client.post(
                    url,
                    headers=headers,
                    json={"model": EMBEDDING_MODEL, "input": t},
                )
                if r.status_code >= 300:
                    raise RuntimeError(f"embedding failed {r.status_code}: {r.text}")
                data = r.json()
                out.append(data["data"][0]["embedding"])
        return out

    # Deterministic fake embeddings (dev fallback).
    dim = 384
    out: List[List[float]] = []
    for t in texts:
        seed = hashlib.sha256(t.encode()).digest()
        out.append([(seed[i % len(seed)] / 255.0) for i in range(dim)])
    return out