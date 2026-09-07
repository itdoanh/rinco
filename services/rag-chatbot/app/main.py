"""FastAPI app for rag-chatbot."""
from __future__ import annotations

import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api import chat_router, collections_router, ingest_router, search_router
from app.core import METRICS, configure_logging, get_logger
from app.services.embeddings import load_embedder
from app.services.qdrant_client import get_client
from app.services.reranker import load_reranker

log = get_logger("rag-chatbot.main")


@asynccontextmanager
async def lifespan(app: FastAPI):
    configure_logging(os.getenv("LOG_LEVEL", "INFO"), "rag-chatbot")
    load_embedder()
    load_reranker()
    get_client()  # eagerly attempt connection
    log.info("rag_chatbot_started", version="2.0.0")
    yield


def create_app() -> FastAPI:
    app = FastAPI(
        title="RINCO RAG Chatbot",
        version="2.0.0",
        lifespan=lifespan,
    )
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_methods=["*"],
        allow_headers=["*"],
    )
    app.include_router(chat_router)
    app.include_router(ingest_router)
    app.include_router(collections_router)
    app.include_router(search_router)

    @app.get("/")
    async def root():
        return {"service": "rag-chatbot", "version": "2.0.0"}

    @app.get("/v1/health")
    async def health():
        return {
            "status": "ok",
            "service": "rag-chatbot",
            "embedder": True,
            "reranker": True,
            "qdrant": get_client() is not None,
        }

    @app.get("/v1/metrics")
    async def metrics():
        body = METRICS["generate_latest"]()
        return body, 200, {"Content-Type": METRICS["CONTENT_TYPE"]}

    return app


app = create_app()