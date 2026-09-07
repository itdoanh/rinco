"""Pydantic schemas."""
from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


class ChatRequest(BaseModel):
    query: str = Field(..., min_length=1)
    history: List[Dict[str, str]] = Field(default_factory=list)
    collection: Optional[str] = None
    system_prompt: Optional[str] = None
    temperature: float = 0.2
    top_k: int = 20
    stream: bool = True
    metadata_filter: Dict[str, Any] = Field(default_factory=dict)


class Citation(BaseModel):
    chunk_id: str
    score: float
    source_url: Optional[str] = None
    doc_type: Optional[str] = None
    text: str
    span: Optional[Dict[str, int]] = None


class ChatResponse(BaseModel):
    answer: str
    citations: List[Citation] = Field(default_factory=list)
    model: str
    tokens_used: int
    latency_ms: int
    collection: str
    feedback_id: Optional[str] = None


class IngestRequest(BaseModel):
    source: str
    kind: str = "text"
    collection: Optional[str] = None
    metadata: Dict[str, Any] = Field(default_factory=dict)


class IngestResponse(BaseModel):
    doc_id: str
    chunks_created: int
    collection: str
    bytes_ingested: int
    duration_ms: int


class SearchRequest(BaseModel):
    query: str
    collection: Optional[str] = None
    top_k: int = 20
    metadata_filter: Dict[str, Any] = Field(default_factory=dict)


class SearchHit(BaseModel):
    chunk_id: str
    score: float
    text: str
    source_url: Optional[str] = None
    metadata: Dict[str, Any] = Field(default_factory=dict)


class SearchResponse(BaseModel):
    hits: List[SearchHit]
    collection: str
    query: str


class FeedbackRequest(BaseModel):
    chat_id: Optional[str] = None
    query: str
    answer: str
    rating: int
    comment: Optional[str] = None


class CreateCollectionRequest(BaseModel):
    name: Optional[str] = None
    vector_size: int = 1024
    distance: str = "Cosine"