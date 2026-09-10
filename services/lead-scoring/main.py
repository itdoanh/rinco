"""Uvicorn entrypoint.

Run::

    uvicorn main:app --host 0.0.0.0 --port 8092

The app is defined in `app.main`; this file simply re-exports it so
`uvicorn main:app` works out of the box.  A handful of legacy symbols
(``score_band_from_score``, ``generate_explanation``, ``featurize``,
``LeadFeatures``) and backward-compatible routes (``/health``,
``/score``, ``/batch-score``) are exposed here for older test suites.
"""
from __future__ import annotations

from fastapi import Body, Header, HTTPException, Request
from fastapi.responses import JSONResponse
from typing import List

from app.main import app, create_app
from app.schemas.lead import LeadFeatures
from app.services.features import featurize as _featurize
from app.services.inference import (
    GLOBAL_MODEL,
    load_model,
    predict_score,
    shap_top_k,
    tier_from_score,
)

# Backwards-compatible band logic used by the original test suite.
# Different thresholds than `tier_from_score` for legacy callers.
_BAND_BOUNDARIES = (
    (0.8, "hot"),
    (0.5, "warm"),
    (0.2, "cold"),
)


def score_band_from_score(score: float) -> str:
    """Legacy band helper: 0.8+ = hot, 0.5+ = warm, 0.2+ = cold, else low."""
    for threshold, band in _BAND_BOUNDARIES:
        if score >= threshold:
            return band
    return "low"


def generate_explanation(importances: dict, band: str) -> str:
    """Produce a short human-readable explanation string from SHAP-like
    feature importances and the chosen band label.
    """
    parts = ", ".join(f"{k}={v:+.2f}" for k, v in (importances or {}).items())
    return f"Lead is {band}; top factors: {parts}."


# Re-export under the names the legacy tests expect.
featurize = _featurize


# -----------------------------------------------------------------------------
# Legacy routes — kept for backwards compatibility with the original test
# suite.  All routes are also available under ``/v1/...``.
# -----------------------------------------------------------------------------


@app.get("/health")
async def _legacy_health() -> JSONResponse:
    """Legacy health endpoint used by early integration tests."""
    return JSONResponse(
        {
            "status": "ok",
            "service": "lead-scoring",
            "models_loaded": [],
            "global_model": GLOBAL_MODEL is not None,
        }
    )


@app.post("/score")
async def _legacy_score(
    payload: dict,
    x_tenant_id: str = Header(..., alias="X-Tenant-ID"),
) -> JSONResponse:
    """Legacy single-lead scoring endpoint.  Requires ``X-Tenant-ID``."""
    feats = LeadFeatures(**payload)
    model = load_model(x_tenant_id)
    X = _featurize(feats, model.features)
    proba = predict_score(model, X)
    score = round(proba * 100, 2)
    tier = tier_from_score(proba)
    band = score_band_from_score(proba)
    shap = shap_top_k(model, X)
    return JSONResponse(
        {
            "score": score,
            "tier": tier,
            "band": band,
            "model_version": model.version,
            "shap_top": shap,
            "explanation": generate_explanation(shap, band),
        }
    )


@app.post("/batch-score")
async def _legacy_batch_score(
    items: List[dict] = Body(...),
    x_tenant_id: str = Header("default", alias="X-Tenant-ID"),
) -> JSONResponse:
    """Legacy batch endpoint.  Hard cap at 1000 items per request."""
    if len(items) > 1000:
        raise HTTPException(status_code=400, detail="Max 1000 leads per batch")
    model = load_model(x_tenant_id)
    out: list[dict] = []
    for item in items:
        feats = LeadFeatures(**item)
        X = _featurize(feats, model.features)
        proba = predict_score(model, X)
        score = round(proba * 100, 2)
        tier = tier_from_score(proba)
        band = score_band_from_score(proba)
        out.append(
            {
                "score": score,
                "tier": tier,
                "band": band,
            }
        )
    return JSONResponse({"count": len(out), "results": out})


__all__ = [
    "app",
    "create_app",
    "featurize",
    "LeadFeatures",
    "score_band_from_score",
    "generate_explanation",
]


if __name__ == "__main__":  # pragma: no cover
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8092)