"""Output schemas for scoring / training / explain."""
from __future__ import annotations

from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


class ScoreResponse(BaseModel):
    """Result of a single-lead scoring request."""

    lead_id: Optional[str] = None
    score: float
    tier: str
    recommended_action: str
    confidence: float
    model_version: str
    feature_importances: Dict[str, float] = Field(default_factory=dict)
    shap_top: Dict[str, float] = Field(default_factory=dict)
    explanation: Optional[str] = None
    latency_ms: int


class TrainRequest(BaseModel):
    """Trigger for a (re-)training job."""

    tenant_id: Optional[str] = None
    notes: Optional[str] = None


class TrainResponse(BaseModel):
    status: str
    tenant_id: str
    model_version: str
    metrics: Dict[str, float] = Field(default_factory=dict)
    fallback_mode: bool


class ModelInfo(BaseModel):
    tenant_id: str
    model_version: str
    n_estimators: int
    features: List[str]
    trained_at: str
    metrics: Dict[str, float] = Field(default_factory=dict)
    fallback_mode: bool


class HealthResponse(BaseModel):
    status: str
    service: str
    version: str
    models_loaded: List[str] = Field(default_factory=list)
    fallback_mode: bool = False


class ExplainResponse(BaseModel):
    lead_id: str
    tenant_id: str
    shap: Dict[str, float] = Field(default_factory=dict)
    features: List[str] = Field(default_factory=list)


__all__ = [
    "ScoreResponse",
    "TrainRequest",
    "TrainResponse",
    "ModelInfo",
    "HealthResponse",
    "ExplainResponse",
]