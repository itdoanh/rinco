"""POST /v1/runbook/create — create a Confluence runbook."""
from __future__ import annotations

from fastapi import APIRouter, HTTPException

from app.core import get_logger
from app.schemas import RunbookCreateRequest, RunbookCreateResponse

from app.services import runbook_writer

router = APIRouter(prefix="/v1/runbook", tags=["runbook"])
logger = get_logger("api.runbook")


@router.post("/create", response_model=RunbookCreateResponse)
async def create_runbook(req: RunbookCreateRequest) -> RunbookCreateResponse:
    """
    Create a runbook page in Confluence.

    Provide the HTML body content and Confluence space key. The page is
    created under the root of the space (no parent).
    """
    if not req.title:
        raise HTTPException(status_code=400, detail="title is required")
    if not req.content_html:
        raise HTTPException(status_code=400, detail="content_html is required")

    url = await runbook_writer.create_runbook(
        title=req.title,
        content_html=req.content_html,
        space=req.space or "RINCO",
    )

    if not url:
        raise HTTPException(
            status_code=502,
            detail="Failed to create Confluence page. Check credentials and space key.",
        )

    # Extract page_id from URL query param
    page_id = ""
    if "pageId=" in url:
        page_id = url.split("pageId=")[-1]

    logger.info("runbook_created", title=req.title, url=url)
    return RunbookCreateResponse(url=url, page_id=page_id or None)


@router.post("/build-html")
async def build_runbook_html(
    incident_id: str,
    service: str,
    root_cause: str,
    suggested_fix: str,
    prevention: str,
    metrics: dict | None = None,
    logs: list[dict] | None = None,
) -> dict:
    """
    Build an HTML runbook body from structured data.

    This is a helper endpoint that formats incident data into Confluence-compatible
    HTML without actually creating the page. Use `/v1/runbook/create` to publish.
    """
    html = runbook_writer.build_runbook_html(
        incident_id=incident_id,
        service=service,
        root_cause=root_cause,
        suggested_fix=suggested_fix,
        prevention=prevention,
        metrics=metrics or {},
        logs=logs or [],
    )
    return {"html": html}
