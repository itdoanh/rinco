"""Training endpoints (admin)."""
from __future__ import annotations

from fastapi import APIRouter, Header, HTTPException

from app.core.metrics import TRAINING_RUNS_TOTAL
from app.schemas.score import ModelInfo, TrainRequest, TrainResponse
from app.services.training import load_model, train as train_fn

router = APIRouter()


@router.post("/v1/train", response_model=TrainResponse)
async def train(
    req: TrainRequest,
    x_admin: bool = Header(False, alias="X-Is-Super-Admin"),
):
    """Re-train the model (admin only)."""
    import os

    if not x_admin and os.getenv("ALLOW_PUBLIC_TRAIN", "0") != "1":
        raise HTTPException(403, detail="admin only")
    tenant_id = req.tenant_id or "default"
    TRAINING_RUNS_TOTAL.labels(tenant_id=tenant_id, status="started").inc()
    try:
        result = train_fn(tenant_id=tenant_id, notes=req.notes)
    except Exception as exc:
        TRAINING_RUNS_TOTAL.labels(tenant_id=tenant_id, status="failed").inc()
        raise HTTPException(500, detail=str(exc))
    if result.get("status") == "failed":
        TRAINING_RUNS_TOTAL.labels(tenant_id=tenant_id, status="failed").inc()
        raise HTTPException(500, detail=result.get("error", "training failed"))
    TRAINING_RUNS_TOTAL.labels(tenant_id=tenant_id, status="ok").inc()
    return TrainResponse(
        status=result["status"],
        tenant_id=result["tenant_id"],
        model_version=result["model_version"],
        metrics=result.get("metrics", {}),
        fallback_mode=result.get("fallback_mode", True),
    )


@router.get("/v1/model/info", response_model=ModelInfo)
async def model_info(x_tenant_id: str = Header("default", alias="X-Tenant-ID")):
    m = load_model(x_tenant_id)
    return ModelInfo(
        tenant_id=x_tenant_id,
        model_version=m.version,
        n_estimators=(getattr(m.xgb, "n_estimators", None) or 0) if m.xgb else 0,
        features=m.features,
        trained_at=m.trained_at.isoformat(),
        metrics=m.metrics,
        fallback_mode=m.fallback_mode,
    )


__all__ = ["router"]