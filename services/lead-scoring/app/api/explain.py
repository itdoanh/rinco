"""SHAP-based explain endpoint."""
from __future__ import annotations

from fastapi import APIRouter, Header, Request

from app.schemas.lead import LeadFeatures
from app.schemas.score import ExplainResponse
from app.services.features import featurize
from app.services.inference import load_model, shap_top_k

router = APIRouter()


@router.post("/v1/explain/{lead_id}", response_model=ExplainResponse)
async def explain(
    lead_id: str,
    request: Request,
    x_tenant_id: str = Header("default", alias="X-Tenant-ID"),
):
    body = await request.json()
    feats = LeadFeatures(**body)
    model = load_model(x_tenant_id)
    X = featurize(feats, model.features)
    shap_full = shap_top_k(model, X, k=len(model.features))
    return ExplainResponse(
        lead_id=lead_id,
        tenant_id=x_tenant_id,
        shap=shap_full,
        features=model.features,
    )


__all__ = ["router"]