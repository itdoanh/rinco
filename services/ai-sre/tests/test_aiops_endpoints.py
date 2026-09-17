"""Integration tests for the new AI SRE aiops endpoints."""
from __future__ import annotations

import pytest
from httpx import ASGITransport, AsyncClient


@pytest.mark.asyncio
async def test_anomalies_check_detects_outlier():
    from app.main import app

    samples = [10.0 + i * 0.01 for i in range(30)] + [95.0]
    payload = {"metric": "cpu", "samples": samples}
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/anomalies/check", json=payload)
    assert r.status_code == 200
    data = r.json()
    assert data["metric"] == "cpu"
    assert isinstance(data["anomalies"], list)
    assert data["summary"]["total"] >= 1
    assert any(a["detector"] == "zscore" for a in data["anomalies"])


@pytest.mark.asyncio
async def test_anomalies_check_too_few_samples():
    from app.main import app

    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/anomalies/check", json={"metric": "x", "samples": [1, 2]})
    assert r.status_code == 400


@pytest.mark.asyncio
async def test_anomalies_observe_pushes_sample():
    from app.main import app

    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        # Push a clearly anomalous value to force a trigger.
        for _ in range(15):
            await client.post("/v1/anomalies/observe", json={"metric": "memory", "value": 50.0})
        r = await client.post("/v1/anomalies/observe", json={"metric": "memory", "value": 500.0})
    assert r.status_code == 200
    data = r.json()
    assert data["metric"] == "memory"
    assert isinstance(data["anomalies"], list)


@pytest.mark.asyncio
async def test_anomalies_list_endpoint():
    from app.main import app

    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        for _ in range(10):
            await client.post("/v1/anomalies/observe", json={"metric": "rpc", "value": 100.0})
        await client.post("/v1/anomalies/observe", json={"metric": "rpc", "value": 9999.0})
        r = await client.get("/v1/anomalies")
    assert r.status_code == 200
    data = r.json()
    assert "count" in data
    assert "by_metric" in data
    assert "rpc" in data["by_metric"]


@pytest.mark.asyncio
async def test_scaling_recommend_hold():
    from app.main import app

    payload = {
        "service": "crm-core",
        "metric": "cpu",
        "current_replicas": 4,
        "forecast_horizon_minutes": 10,
        "predicted_peak": 0.5,
        "target_utilization": 0.8,
        "min_replicas": 1,
        "max_replicas": 10,
        "safety_margin": 1.0,
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/scaling/recommend", json=payload)
    assert r.status_code == 200
    data = r.json()
    assert data["service"] == "crm-core"
    assert data["recommendation"]["current_replicas"] == 4


@pytest.mark.asyncio
async def test_scaling_recommend_scale_up():
    from app.main import app

    payload = {
        "service": "chat-engine",
        "metric": "cpu",
        "current_replicas": 2,
        "forecast_horizon_minutes": 30,
        "predicted_peak": 1.0,  # 0.7 target * 2 = 1.4 -> double replicas
        "target_utilization": 0.7,
        "safety_margin": 1.0,
        "min_replicas": 1,
        "max_replicas": 20,
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/scaling/recommend", json=payload)
    assert r.status_code == 200
    data = r.json()
    assert data["recommendation"]["action"] in ("scale_up", "add_capacity")
    assert data["recommendation"]["recommended_replicas"] >= 3


@pytest.mark.asyncio
async def test_capacity_plan_with_multiple_metrics():
    from app.main import app

    payload = {
        "service": "meeting-ui",
        "current_replicas": 3,
        "metrics": {
            "cpu": [0.5] * 30,
            "memory": [0.4] * 30,
            "rpc_latency": [0.05 + i * 0.001 for i in range(30)],
        },
        "horizon_minutes": 30,
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/scaling/plan", json=payload)
    assert r.status_code == 200
    data = r.json()
    assert data["service"] == "meeting-ui"
    assert len(data["recommendations"]) == 3
    assert data["summary"]["count"] == 3


@pytest.mark.asyncio
async def test_capacity_plan_empty_metrics_returns_400():
    from app.main import app

    payload = {
        "service": "x",
        "current_replicas": 1,
        "metrics": {},
        "horizon_minutes": 10,
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/scaling/plan", json=payload)
    assert r.status_code == 400


@pytest.mark.asyncio
async def test_forecast_endpoint_returns_predictions():
    from app.main import app

    samples = [float(i) for i in range(30)]
    payload = {
        "metric": "rpc_latency",
        "samples": samples,
        "horizon_minutes": 15,
        "sample_interval_seconds": 60.0,
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        r = await client.post("/v1/forecast", json=payload)
    assert r.status_code == 200
    data = r.json()
    assert data["metric"] == "rpc_latency"
    assert len(data["predicted"]) == 15
    assert data["trend_per_minute"] > 0
