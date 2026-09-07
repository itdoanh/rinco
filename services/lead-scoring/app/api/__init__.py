"""FastAPI routers: scoring, training, explain."""
from __future__ import annotations

from app.api.scoring import router as scoring_router
from app.api.training import router as training_router
from app.api.explain import router as explain_router

__all__ = ["scoring_router", "training_router", "explain_router"]