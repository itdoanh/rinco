"""POST /v1/chat — simple LLM chat with optional incident context."""
from __future__ import annotations

from typing import Any

import httpx

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from app.core import get_logger, METRICS
from app.core import VLLM_URL, LLM_MODEL
from app.services import incident_store, incident_correlator

router = APIRouter(prefix="/v1/chat", tags=["chat"])
logger = get_logger("api.chat")


class ChatRequest(BaseModel):
    """Request for the chat endpoint."""
    message: str
    incident_id: str | None = None
    service: str | None = None
    time_window: str = "1h"


class ChatResponse(BaseModel):
    """Response from the chat endpoint."""
    reply: str
    sources: list[str] = []


@router.post("", response_model=ChatResponse)
async def chat(req: ChatRequest) -> ChatResponse:
    """
    Simple LLM chat endpoint. If an incident_id or service is provided,
    relevant context (logs, traces, metrics) is fetched and included in
    the prompt to give the LLM situational awareness.
    """
    if not req.message:
        raise HTTPException(status_code=400, detail="message is required")

    context_parts: list[str] = []
    sources: list[str] = []

    # Enrich with incident context if available
    if req.incident_id:
        incident = await incident_store.get_incident(req.incident_id)
        if incident:
            context_parts.append(
                f"INCIDENT [{req.incident_id}]: service={incident.get('service')}, "
                f"error={incident.get('error', '')[:300]}, severity={incident.get('severity')}"
            )
            # Correlate additional observability data
            correlation = await incident_correlator.correlate_incident(
                service=incident.get("service", ""),
                trace_id=incident.get("trace_id"),
                time_window=req.time_window,
            )
            context_parts.append(f"CORRELATION SUMMARY:\n{correlation.get('summary', '')}")
            sources.append(f"incident:{req.incident_id}")

    elif req.service:
        correlation = await incident_correlator.correlate_incident(
            service=req.service,
            time_window=req.time_window,
        )
        context_parts.append(f"SERVICE CONTEXT for {req.service}:\n{correlation.get('summary', '')}")
        sources.append(f"service:{req.service}")

    context_block = ""
    if context_parts:
        context_block = "\n\n## INCIDENT CONTEXT\n" + "\n\n".join(context_parts)

    prompt = f"""You are an expert AI Site Reliability Engineer assistant.

{context_block}

## USER QUESTION
{req.message}

Respond as a helpful SRE assistant. Be concise and technical."""

    try:
        async with httpx.AsyncClient(timeout=60.0) as client:
            r = await client.post(
                f"{VLLM_URL}/v1/chat/completions",
                json={
                    "model": LLM_MODEL,
                    "messages": [{"role": "user", "content": prompt}],
                    "max_tokens": 1500,
                    "temperature": 0.3,
                },
            )
            r.raise_for_status()
            reply = r.json()["choices"][0]["message"]["content"]
    except httpx.HTTPStatusError as e:
        logger.warning("chat_llm_failed", status=e.response.status_code)
        raise HTTPException(status_code=503, detail="LLM service unavailable")
    except Exception as e:
        logger.error("chat_error", error=str(e))
        raise HTTPException(status_code=500, detail="Chat processing failed")

    return ChatResponse(reply=reply, sources=sources)
