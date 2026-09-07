"""NATS consumer for `lead.created` events.

This module wires an async JetStream subscription that calls the
inference pipeline for every incoming lead and republishes the score to
`lead.scored`.  When NATS is unavailable the consumer is a no-op so the
service can boot without infrastructure.
"""
from __future__ import annotations

import asyncio
import json
import os
from typing import Any

from app.core.logging import get_logger

log = get_logger("lead-scoring.nats")


async def nats_consumer(app: Any, subject: str = "lead.created", url: str | None = None) -> None:
    """Listen to `lead.created` and score each lead automatically."""
    try:
        import nats as nats_lib  # type: ignore
    except Exception:
        log.info("nats_unavailable_skipping_consumer")
        return

    target_url = url or os.getenv("NATS_URL") or "nats://nats:4222"
    try:
        nc = await nats_lib.connect(target_url, connect_timeout=3)
        js = nc.jetstream()

        async def cb(msg: Any) -> None:
            try:
                payload = json.loads(msg.data.decode())
                lead_id = payload.get("lead_id")
                from app.schemas.lead import LeadFeatures  # local to avoid import-time side-effects

                feats = LeadFeatures(**payload.get("features", {}))
                resp = await _run_scoring(feats, lead_id=lead_id,
                                          tenant_id=payload.get("tenant_id"))
                await js.publish(
                    "lead.scored",
                    json.dumps(
                        {
                            "lead_id": lead_id,
                            "tenant_id": payload.get("tenant_id"),
                            **resp.model_dump(mode="json"),
                        }
                    ).encode(),
                )
                await msg.ack()
            except Exception as exc:  # pragma: no cover
                log.error("nats_consumer_error", error=str(exc))
                try:
                    await msg.nak()
                except Exception:  # pragma: no cover
                    pass

        await js.subscribe(subject, cb=cb)
        log.info("nats_consumer_started", subject=subject, url=target_url)
        while True:
            await asyncio.sleep(60)
    except Exception as exc:  # pragma: no cover
        log.warning("nats_connect_failed", error=str(exc))


async def _run_scoring(features: "LeadFeatures", lead_id: str | None = None,
                       tenant_id: str = "default") -> "ScoreResponse":
    """Run scoring using the same logic as the HTTP endpoint."""
    from app.schemas.lead import LeadFeatures  # type: ignore
    from app.schemas.score import ScoreResponse  # type: ignore
    from app.services.inference import (
        _initialise_default_model,
        confidence_from_features,
        load_model,
        predict_score,
        recommended_action,
        shap_top_k,
        tier_from_score,
    )
    import time

    start = time.perf_counter()
    model = load_model(tenant_id)
    from app.services.features import featurize  # type: ignore

    X = featurize(features, model.features)
    proba = predict_score(model, X)
    score_0_100 = round(proba * 100, 2)
    tier = tier_from_score(proba)
    action = recommended_action(tier)
    confidence = round(confidence_from_features(model, X), 3)
    shap_top = shap_top_k(model, X)
    latency_ms = int((time.perf_counter() - start) * 1000)
    return ScoreResponse(
        lead_id=lead_id,
        score=score_0_100,
        tier=tier,
        recommended_action=action,
        confidence=confidence,
        model_version=model.version,
        shap_top=shap_top,
        latency_ms=latency_ms,
    )


def warm_global_model() -> None:
    """Ensure the global fallback model is initialised at startup."""
    from app.services.inference import GLOBAL_MODEL, _initialise_default_model, load_model

    global GLOBAL_MODEL
    if GLOBAL_MODEL is None:
        GLOBAL_MODEL = _initialise_default_model()
    _ = load_model


__all__ = ["nats_consumer", "warm_global_model"]