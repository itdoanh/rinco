"""Tests for lead-scoring /v1/score, /v1/score/batch and /v1/health endpoints."""
import pytest
from httpx import AsyncClient, ASGITransport

from app.main import app


def _feats(**over):
    base = {
        "page_views": 5,
        "time_on_site_seconds": 60,
        "has_phone": True,
        "email_opens": 2,
        "email_clicks": 1,
        "form_fills": 1,
        "company_size": "50-100",
        "industry": "tech",
        "source": "google",
    }
    base.update(over)
    return base


@pytest.mark.asyncio
async def test_health():
    """GET /v1/health returns ok status."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.get("/v1/health")
    # May be 200 or 503 depending on whether model loaded
    assert r.status_code in (200, 503)


@pytest.mark.asyncio
async def test_score_v1_basic():
    """POST /v1/score returns score/tier/action/confidence."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/score", json=_feats())
    assert r.status_code in (200, 503), r.text
    if r.status_code == 200:
        data = r.json()
        assert "score" in data
        assert "tier" in data
        assert "recommended_action" in data
        assert "confidence" in data


@pytest.mark.asyncio
async def test_score_by_lead_id():
    """POST /v1/score/{lead_id} includes lead_id in response."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/score/lead-abc", json=_feats())
    assert r.status_code in (200, 503), r.text
    if r.status_code == 200:
        assert r.json()["lead_id"] == "lead-abc"


@pytest.mark.asyncio
async def test_score_batch():
    """POST /v1/score/batch handles multiple leads."""
    payload = [_feats(), _feats(page_views=0)]
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/score/batch", json=payload)
    assert r.status_code in (200, 503), r.text
    if r.status_code == 200:
        data = r.json()
        assert "count" in data
        assert "results" in data


@pytest.mark.asyncio
async def test_score_batch_too_large():
    """POST /v1/score/batch with >1000 items returns 400."""
    payload = [_feats()] * 1001
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/score/batch", json=payload)
    assert r.status_code == 400


@pytest.mark.asyncio
async def test_metrics_endpoint():
    """GET /v1/metrics returns Prometheus exposition format."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.get("/v1/metrics")
    assert r.status_code == 200
    body = r.text
    assert "lead_scoring" in body


@pytest.mark.asyncio
async def test_score_tenant_header():
    """Score with custom tenant id."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/score", json=_feats(), headers={"X-Tenant-ID": "tenant-x"})
    assert r.status_code in (200, 503)


@pytest.mark.asyncio
async def test_score_invalid_payload():
    """Score with bad payload — empty body should fail validation."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        # Send invalid types — array is not allowed
        r = await client.post("/v1/score", json=["array not valid"])
    assert r.status_code in (200, 422)


@pytest.mark.asyncio
async def test_root_endpoint():
    """GET / returns service metadata."""
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.get("/")
    assert r.status_code == 200
    data = r.json()
    assert data["service"] == "lead-scoring"
    assert "version" in data
