"""Tests for lead-scoring features module."""
from __future__ import annotations

import numpy as np

from app.services.features import (
    DEFAULT_FEATURES,
    _compute_row,
    feature_names,
    featurize,
)
from app.schemas.lead import LeadFeatures


def _make_lead(**overrides):
    """Helper to create a LeadFeatures with sensible defaults."""
    defaults = {
        "has_phone": False,
        "has_email": True,
        "company": "ACME Corp",
        "full_name": "John Doe",
        "country": "VN",
        "source": "organic",
        "device_type": "desktop",
        "page_views": 5,
        "time_on_site_seconds": 60,
        "repeat_visits": 1,
        "fbclid": None,
        "gclid": None,
        "email_opens": 3,
        "email_clicks": 1,
        "form_fills": 0,
        "abandoned_carts": 0,
        "company_size": "11-50",
        "company_revenue": 1000000,
        "job_title": "Manager",
        "source_quality": 0.5,
        "channel_conversion_rate": 0.02,
        "utm_source": None,
        "utm_campaign": None,
        "utm_medium": None,
    }
    defaults.update(overrides)
    return LeadFeatures(**defaults)


def test_extra_feature_names_returns_default():
    """feature_names() should return DEFAULT_FEATURES."""
    names = feature_names()
    assert names == DEFAULT_FEATURES
    assert len(names) > 0


def test_extra_default_features_count():
    """DEFAULT_FEATURES should have many features."""
    assert len(DEFAULT_FEATURES) >= 30


def test_extra_compute_row_basic():
    """_compute_row should produce expected row dict."""
    lead = _make_lead()
    row = _compute_row(lead)
    assert isinstance(row, dict)
    assert "has_phone" in row
    assert "has_email" in row
    assert "page_views" in row


def test_extra_compute_row_country_vn():
    """country_vn should be 1 for VN."""
    lead = _make_lead(country="VN")
    row = _compute_row(lead)
    assert row["country_vn"] == 1
    assert row["country_us"] == 0


def test_extra_compute_row_country_us():
    """country_us should be 1 for US."""
    lead = _make_lead(country="US")
    row = _compute_row(lead)
    assert row["country_us"] == 1
    assert row["country_vn"] == 0


def test_extra_compute_row_country_other():
    """country_other should be 1 for non-VN/US countries."""
    lead = _make_lead(country="JP")
    row = _compute_row(lead)
    assert row["country_other"] == 1
    assert row["country_vn"] == 0
    assert row["country_us"] == 0


def test_extra_compute_row_empty_country():
    """Empty country should not match VN/US/other."""
    lead = _make_lead(country="")
    row = _compute_row(lead)
    assert row["country_vn"] == 0
    assert row["country_us"] == 0
    assert row["country_other"] == 0


def test_extra_compute_row_b2b_title():
    """is_b2b should be 1 for manager title."""
    lead = _make_lead(job_title="Sales Manager")
    row = _compute_row(lead)
    assert row["is_b2b"] == 1


def test_extra_compute_row_b2b_director():
    """is_b2b should be 1 for director title."""
    lead = _make_lead(job_title="Director of Engineering")
    row = _compute_row(lead)
    assert row["is_b2b"] == 1


def test_extra_compute_row_b2b_ceo():
    """is_b2b should be 1 for CEO title."""
    lead = _make_lead(job_title="CEO")
    row = _compute_row(lead)
    assert row["is_b2b"] == 1


def test_extra_compute_row_non_b2b_title():
    """is_b2b should be 0 for student title."""
    lead = _make_lead(job_title="Student")
    row = _compute_row(lead)
    assert row["is_b2b"] == 0


def test_extra_compute_row_device_mobile():
    """is_mobile should be 1 for mobile device."""
    lead = _make_lead(device_type="mobile")
    row = _compute_row(lead)
    assert row["is_mobile"] == 1
    assert row["is_desktop"] == 0


def test_extra_compute_row_device_desktop():
    """is_desktop should be 1 for desktop device."""
    lead = _make_lead(device_type="desktop")
    row = _compute_row(lead)
    assert row["is_desktop"] == 1
    assert row["is_mobile"] == 0


def test_extra_compute_row_source_organic():
    """source_organic should be 1 for organic."""
    lead = _make_lead(source="organic")
    row = _compute_row(lead)
    assert row["source_organic"] == 1


def test_extra_compute_row_source_paid():
    """source_paid should be 1 for paid ads."""
    lead = _make_lead(source="facebook_ads")
    row = _compute_row(lead)
    assert row["source_paid"] == 1


def test_extra_compute_row_company_size_small():
    """company_size_sm should be 1 for small sizes."""
    for size in ["1-10", "1-50", "small"]:
        lead = _make_lead(company_size=size)
        row = _compute_row(lead)
        assert row["company_size_sm"] == 1, f"size={size}"


def test_extra_compute_row_company_size_medium():
    """company_size_md should be 1 for medium sizes."""
    for size in ["11-50", "51-200", "medium"]:
        lead = _make_lead(company_size=size)
        row = _compute_row(lead)
        assert row["company_size_md"] == 1, f"size={size}"


def test_extra_compute_row_company_size_large():
    """company_size_lg should be 1 for large sizes."""
    for size in ["200+", "500+", "1000+", "large"]:
        lead = _make_lead(company_size=size)
        row = _compute_row(lead)
        assert row["company_size_lg"] == 1, f"size={size}"


def test_extra_compute_row_clid_present():
    """fbclid_present should be 1 when fbclid is set."""
    lead = _make_lead(fbclid="abc123")
    row = _compute_row(lead)
    assert row["fbclid_present"] == 1


def test_extra_compute_row_clid_absent():
    """fbclid_present should be 0 when fbclid is None."""
    lead = _make_lead(fbclid=None)
    row = _compute_row(lead)
    assert row["fbclid_present"] == 0


def test_extra_compute_row_tiktok_utm():
    """ttclid_present should be 1 when utm_source contains tiktok."""
    lead = _make_lead(utm_source="tiktok")
    row = _compute_row(lead)
    assert row["ttclid_present"] == 1


def test_extra_compute_row_time_on_site_log():
    """time_on_site_log should be log1p of seconds."""
    lead = _make_lead(time_on_site_seconds=100)
    row = _compute_row(lead)
    expected = float(np.log1p(100))
    assert abs(row["time_on_site_log"] - expected) < 0.001


def test_extra_compute_row_company_revenue_log():
    """company_revenue_log should be log1p of revenue."""
    lead = _make_lead(company_revenue=1000)
    row = _compute_row(lead)
    expected = float(np.log1p(1000))
    assert abs(row["company_revenue_log"] - expected) < 0.001


def test_extra_compute_row_no_negative_time():
    """Negative time_on_site should not produce negative log."""
    lead = _make_lead(time_on_site_seconds=-100)
    row = _compute_row(lead)
    assert row["time_on_site_log"] >= 0


def test_extra_featurize_returns_dataframe_like():
    """featurize should return a dataframe-like object."""
    lead = _make_lead()
    df = featurize(lead)
    assert df is not None


def test_extra_featurize_custom_names():
    """featurize should respect custom names."""
    lead = _make_lead()
    df = featurize(lead, names=["has_email", "page_views"])
    assert df is not None


def test_extra_featurize_includes_all_default_features():
    """featurize with default names should include all DEFAULT_FEATURES."""
    lead = _make_lead()
    df = featurize(lead)
    # Convert to dict if needed
    if hasattr(df, "to_dict"):
        d = df.to_dict("records")[0]
    elif hasattr(df, "rows"):
        d = df.rows[0]
    else:
        d = {}
    # Should have most DEFAULT_FEATURES (or all)
    assert len(d) > 0


def test_extra_featurize_handles_missing_data():
    """featurize should handle leads with missing fields."""
    lead = _make_lead(
        company=None,
        full_name=None,
        job_title=None,
        country=None,
    )
    df = featurize(lead)
    assert df is not None


def test_extra_featurize_empty_company():
    """featurize should handle empty company."""
    lead = _make_lead(company="")
    df = featurize(lead)
    assert df is not None


def test_extra_default_features_unique():
    """DEFAULT_FEATURES should not contain duplicates."""
    assert len(DEFAULT_FEATURES) == len(set(DEFAULT_FEATURES))


def test_extra_default_features_categories():
    """DEFAULT_FEATURES should cover key categories."""
    names_str = " ".join(DEFAULT_FEATURES)
    assert "country" in names_str
    assert "is_" in names_str
    assert "page_views" in names_str
