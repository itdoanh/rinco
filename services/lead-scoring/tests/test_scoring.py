"""Tests for the scoring endpoint and inference pipeline."""
from __future__ import annotations

import asyncio
import os
import sys

import pytest

# Ensure we can import the app package
ROOT = os.path.dirname(os.path.abspath(__file__))
if ROOT not in sys.path:
    sys.path.insert(0, os.path.dirname(ROOT))

from app.schemas.lead import LeadFeatures
from app.services.inference import (
    GLOBAL_MODEL,
    _initialise_default_model,
    confidence_from_features,
    predict_score,
    recommended_action,
    tier_from_score,
)
from app.services.features import DEFAULT_FEATURES, featurize, feature_names


def test_feature_names_returns_default():
    names = feature_names()
    assert isinstance(names, list)
    assert "has_phone" in names
    assert "page_views" in names


def test_featurize_returns_row_with_all_defaults():
    feats = LeadFeatures()
    df = featurize(feats, DEFAULT_FEATURES)
    assert len(df) == 1
    # Default values should not crash.
    row = df.iloc[0]
    assert row["has_phone"] == 0
    assert row["has_email"] == 1


def test_featurize_handles_job_title_b2b():
    feats = LeadFeatures(job_title="Senior Manager", company="ACME", country="VN")
    df = featurize(feats, DEFAULT_FEATURES)
    assert df.iloc[0]["is_b2b"] == 1
    assert df.iloc[0]["country_vn"] == 1
    assert df.iloc[0]["company_length"] == 4


def test_tier_from_score_thresholds():
    assert tier_from_score(0.95) == "very-hot"
    assert tier_from_score(0.7) == "hot"
    assert tier_from_score(0.4) == "warm"
    assert tier_from_score(0.1) == "cold"


def test_recommended_action_for_each_tier():
    assert recommended_action("very-hot") == "call-now"
    assert recommended_action("hot") == "call-soon"
    assert recommended_action("warm") == "email"
    assert recommended_action("cold") == "nurture"
    assert recommended_action("unknown") == "nurture"


def test_inference_pipeline_returns_score():
    if GLOBAL_MODEL is None:
        m = _initialise_default_model()
    else:
        m = GLOBAL_MODEL
    feats = LeadFeatures(
        company="ACME",
        page_views=5,
        time_on_site_seconds=120,
        email_opens=3,
        email_clicks=2,
    )
    X = featurize(feats, m.features)
    score = predict_score(m, X)
    assert 0.0 <= score <= 1.0


def test_confidence_in_range():
    if GLOBAL_MODEL is None:
        m = _initialise_default_model()
    else:
        m = GLOBAL_MODEL
    feats = LeadFeatures()
    X = featurize(feats, m.features)
    conf = confidence_from_features(m, X)
    assert 0.5 <= conf <= 0.99


def test_app_can_be_created():
    """Smoke test: the FastAPI app should import without error."""
    from app.main import create_app

    app = create_app()
    assert app.title == "RINCO Lead Scoring"


@pytest.mark.asyncio
async def test_score_v1_endpoint():
    """Smoke test the POST /v1/score endpoint."""
    try:
        from httpx import ASGITransport, AsyncClient
    except Exception:
        pytest.skip("httpx not installed")
    from app.main import create_app

    app = create_app()
    payload = {
        "email": "alice@example.com",
        "company": "ACME",
        "job_title": "Director of Marketing",
        "country": "US",
        "page_views": 8,
        "time_on_site_seconds": 250,
        "email_opens": 4,
        "email_clicks": 1,
        "device_type": "desktop",
    }
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        resp = await client.post("/v1/score", json=payload, headers={"X-Tenant-ID": "test"})
    assert resp.status_code == 200
    body = resp.json()
    assert "score" in body
    assert "tier" in body
    assert "recommended_action" in body
    assert "model_version" in body