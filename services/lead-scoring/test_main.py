"""Tests cho Lead Scoring service."""
import pytest
from fastapi.testclient import TestClient
from main import app
import os
import tempfile
import json
import joblib
from unittest.mock import MagicMock

client = TestClient(app)


@pytest.fixture
def sample_lead():
    return {
        "email": "test@example.com",
        "phone": "0912345678",
        "full_name": "Nguyen Van A",
        "company": "ABC Corp",
        "job_title": "CEO",
        "source": "facebook_ads",
        "page_views": 5,
        "time_on_site_seconds": 300,
        "has_phone": True,
        "has_email": True,
        "fbclid": "fb.test.123",
        "country": "VN",
        "device_type": "desktop",
    }


def test_health():
    r = client.get("/health")
    assert r.status_code == 200
    assert r.json()["status"] == "ok"


def test_score_missing_tenant():
    r = client.post("/score", json={"email": "test@example.com"})
    assert r.status_code == 422  # Missing X-Tenant-ID header


def test_score_invalid_email():
    payload = {"email": "not-an-email", "phone": "0912345678"}
    r = client.post("/score", json=payload, headers={"X-Tenant-ID": "demo"})
    # Should still process (no strict validation in score function)
    # But will likely 503 if no model loaded
    assert r.status_code in (200, 503)


def test_score_band_logic():
    """Verify band boundaries."""
    from main import score_band_from_score
    assert score_band_from_score(0.9) == "hot"
    assert score_band_from_score(0.7) == "warm"
    assert score_band_from_score(0.3) == "cold"
    assert score_band_from_score(0.1) == "low"
    assert score_band_from_score(0.8) == "hot"  # boundary
    assert score_band_from_score(0.5) == "warm"  # boundary
    assert score_band_from_score(0.2) == "cold"  # boundary


def test_generate_explanation():
    """Verify explanation generation."""
    from main import generate_explanation
    importances = {"page_views": 0.3, "has_phone": 0.2, "fbclid_present": 0.15}
    exp = generate_explanation(importances, "hot")
    assert "hot" in exp
    assert "page_views" in exp


def test_featurize_returns_dataframe():
    """Verify feature engineering produces correct columns."""
    from main import featurize, LeadFeatures
    features = LeadFeatures(
        email="test@example.com",
        phone="0912345678",
        page_views=5,
        time_on_site_seconds=300,
        fbclid="abc",
    )
    feature_names = ["has_phone", "has_email", "page_views", "fbclid_present"]
    df = featurize(features, feature_names)
    assert len(df) == 1
    assert df["has_phone"].iloc[0] == 1
    assert df["has_email"].iloc[0] == 1
    assert df["page_views"].iloc[0] == 5
    assert df["fbclid_present"].iloc[0] == 1


def test_batch_score_limit():
    """Verify batch limit enforcement."""
    items = [{"email": "a@b.c"}] * 1001
    r = client.post("/batch-score", json=items, headers={"X-Tenant-ID": "demo"})
    assert r.status_code == 400
    assert "Max 1000" in r.json()["detail"]
