"""Routers: chat, ingest, collections, search."""
from __future__ import annotations

from app.api.chat import router as chat_router
from app.api.ingest import router as ingest_router
from app.api.collections import router as collections_router
from app.api.search import router as search_router

__all__ = ["chat_router", "ingest_router", "collections_router", "search_router"]