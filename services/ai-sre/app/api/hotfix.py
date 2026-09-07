"""POST /v1/hotfix/generate — generate a code-level hotfix via vLLM."""
from __future__ import annotations

import time

from fastapi import APIRouter, HTTPException

from app.core import get_logger, METRICS
from app.schemas import HotfixGenerateRequest, HotfixGenerateResponse

from app.services import hotfix_generator

router = APIRouter(prefix="/v1/hotfix", tags=["hotfix"])
logger = get_logger("api.hotfix")


@router.post("/generate", response_model=HotfixGenerateResponse)
async def generate_hotfix(req: HotfixGenerateRequest) -> HotfixGenerateResponse:
    """
    Generate a code-level hotfix suggestion using vLLM DeepSeek-Coder.

    Provide the error message, optional stack trace, source code context,
    and any additional context (logs, metrics). The model returns a diff
    targeting the most likely root cause.
    """
    if not req.error:
        raise HTTPException(status_code=400, detail="error field is required")

    t0 = time.monotonic()

    try:
        result = await hotfix_generator.generate_hotfix(
            error=req.error,
            stack=req.stack,
            source=req.source,
            context=req.context,
        )
    except Exception as e:
        logger.error("hotfix_generate_error", error=str(e))
        raise HTTPException(status_code=503, detail="Hotfix generation failed")

    elapsed = time.monotonic() - t0
    logger.info(
        "hotfix_generated",
        confidence=result.get("confidence", 0),
        file=result.get("file", ""),
        elapsed_s=round(elapsed, 2),
    )

    # Record metrics
    metrics = METRICS
    if metrics.get("analysis_requests"):
        metrics["analysis_requests"].labels(tenant_id="default", result="hotfix").inc()
    if metrics.get("analysis_latency"):
        metrics["analysis_latency"].observe(elapsed)

    return HotfixGenerateResponse(
        diff=result.get("diff", ""),
        file=result.get("file", ""),
        line=result.get("line", 0),
        confidence=result.get("confidence", 0.0),
        root_cause=result.get("root_cause", ""),
    )
