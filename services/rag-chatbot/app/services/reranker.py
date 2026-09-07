"""Reranker using BGE-reranker-base cross-encoder."""
from __future__ import annotations

from typing import Any, Dict, List

try:
    from sentence_transformers import CrossEncoder  # type: ignore
    HAS_ST = True
except Exception:  # pragma: no cover
    HAS_ST = False
    CrossEncoder = None  # type: ignore

from app.core import RERANKER_MODEL, get_logger

log = get_logger("rag-chatbot.reranker")

_reranker: Any = None


def load_reranker() -> Any:
    global _reranker
    if _reranker is not None:
        return _reranker
    if HAS_ST:
        try:
            log.info("loading_reranker", model=RERANKER_MODEL)
            _reranker = CrossEncoder(RERANKER_MODEL)
            return _reranker
        except Exception as exc:
            log.warning("reranker_load_failed", error=str(exc))
    return None


def rerank(query: str, hits: List[Dict[str, Any]], top_n: int) -> List[Dict[str, Any]]:
    """Re-rank hits by cross-encoder score and return top_n."""
    reranker = load_reranker()
    if reranker is None or not hits:
        return hits[:top_n]
    pairs = [[query, h.get("payload", {}).get("text", "")] for h in hits]
    try:
        scores = reranker.predict(pairs)
    except Exception as exc:
        log.warning("rerank_failed", error=str(exc))
        return hits[:top_n]
    scored = []
    for h, s in zip(hits, scores):
        h["rerank_score"] = float(s)
        scored.append(h)
    scored.sort(key=lambda x: x.get("rerank_score", 0), reverse=True)
    return scored[:top_n]