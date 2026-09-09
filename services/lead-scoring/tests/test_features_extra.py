"""Tests for lead-scoring feature engineering."""
from __future__ import annotations

from app.schemas.lead import LeadFeatures
from app.services.features import (
    DEFAULT_FEATURES,
    _DictFrame,
    _compute_row,
    feature_names,
    featurize,
)


def _make_lead(**overrides) -> LeadFeatures:
    """Build a LeadFeatures with sensible defaults and per-test overrides."""
    defaults = dict(
        job_title="",
        company="",
        full_name="",
        country="",
        source="organic",
        device_type="desktop",
        company_size="",
        company_revenue=0,
        has_phone=False,
        has_email=False,
        page_views=0,
        time_on_site_seconds=0,
        repeat_visits=0,
        fbclid=None,
        gclid=None,
        utm_source=None,
        utm_campaign=None,
        utm_medium=None,
        email_opens=0,
        email_clicks=0,
        form_fills=0,
        abandoned_carts=0,
        source_quality=0.5,
        channel_conversion_rate=0.0,
    )
    defaults.update(overrides)
    return LeadFeatures(**defaults)


def test_extra_feature_names_returns_default():
    """feature_names() should return a fresh copy of DEFAULT_FEATURES."""
    names = feature_names()
    assert names == DEFAULT_FEATURES
    # Should be a copy, not the same list.
    assert names is not DEFAULT_FEATURES


def test_extra_default_features_length():
    """DEFAULT_FEATURES should have a known, fixed length (smoke check)."""
    assert len(DEFAULT_FEATURES) >= 30
    # No duplicates.
    assert len(set(DEFAULT_FEATURES)) == len(DEFAULT_FEATURES)


def test_extra_compute_row_basic_flags():
    """Boolean flags (has_phone/has_email) are 0/1."""
    row = _compute_row(_make_lead(has_phone=True, has_email=False))
    assert row["has_phone"] == 1
    assert row["has_email"] == 0


def test_extra_compute_row_country_indicators():
    """Country one-hots: VN, US, other, none."""
    row_vn = _compute_row(_make_lead(country="VN"))
    row_us = _compute_row(_make_lead(country="US"))
    row_de = _compute_row(_make_lead(country="DE"))
    row_empty = _compute_row(_make_lead(country=""))
    assert row_vn["country_vn"] == 1 and row_vn["country_us"] == 0
    assert row_us["country_us"] == 1 and row_us["country_vn"] == 0
    assert row_de["country_other"] == 1
    assert row_empty["country_other"] == 0 and row_empty["country_vn"] == 0


def test_extra_compute_row_is_b2b_titles():
    """Job titles containing B2B keywords set is_b2b=1."""
    for title in ["CEO", "Engineering Manager", "Director of Sales", "Founder", "Owner", "Head of Marketing"]:
        row = _compute_row(_make_lead(job_title=title))
        assert row["is_b2b"] == 1, f"expected B2B for {title!r}, got {row['is_b2b']}"
    # Non-B2B titles should give 0.
    for title in ["Student", "Intern", "Junior Developer", "Volunteer"]:
        row = _compute_row(_make_lead(job_title=title))
        assert row["is_b2b"] == 0, f"expected non-B2B for {title!r}"


def test_extra_compute_row_device_types():
    """Device type indicator maps to the correct one-hot."""
    for device, expected in [
        ("mobile", "is_mobile"),
        ("tablet", "is_tablet"),
        ("desktop", "is_desktop"),
    ]:
        row = _compute_row(_make_lead(device_type=device))
        assert row[expected] == 1, f"{device} should set {expected}"
        # Other flags must be 0.
        for other in ["is_mobile", "is_tablet", "is_desktop"]:
            if other != expected:
                assert row[other] == 0


def test_extra_compute_row_company_size_buckets():
    """Company size maps to the right bucket."""
    assert _compute_row(_make_lead(company_size="1-10"))["company_size_sm"] == 1
    assert _compute_row(_make_lead(company_size="51-200"))["company_size_md"] == 1
    assert _compute_row(_make_lead(company_size="1000+"))["company_size_lg"] == 1
    assert _compute_row(_make_lead(company_size="unknown"))["company_size_sm"] == 0
    assert _compute_row(_make_lead(company_size="unknown"))["company_size_md"] == 0
    assert _compute_row(_make_lead(company_size="unknown"))["company_size_lg"] == 0


def test_extra_compute_row_source_channels():
    """Source channel maps to the right bucket."""
    assert _compute_row(_make_lead(source="facebook_ads"))["source_paid"] == 1
    assert _compute_row(_make_lead(source="organic"))["source_organic"] == 1
    assert _compute_row(_make_lead(source="direct"))["source_direct"] == 1
    assert _compute_row(_make_lead(source="referral"))["source_referral"] == 1
    assert _compute_row(_make_lead(source=""))["source_organic"] == 0
    assert _compute_row(_make_lead(source=""))["source_direct"] == 0


def test_extra_compute_row_click_ids_present():
    """Click ID flags set to 1 when the ID is non-empty."""
    row = _compute_row(_make_lead(fbclid="abc", gclid=None))
    assert row["fbclid_present"] == 1
    assert row["gclid_present"] == 0


def test_extra_compute_row_utm_flags():
    """UTM campaign/medium flag set when non-empty."""
    row = _compute_row(_make_lead(utm_campaign="x", utm_medium=None))
    assert row["utm_has_campaign"] == 1
    assert row["utm_has_medium"] == 0


def test_extra_compute_row_open_rate_in_unit_interval():
    """open_rate should be in [0, 1] for valid inputs."""
    row = _compute_row(_make_lead(email_opens=10, email_clicks=2))
    assert 0.0 <= row["open_rate"] <= 1.0
    row2 = _compute_row(_make_lead(email_opens=0, email_clicks=0))
    assert 0.0 <= row2["open_rate"] <= 1.0


def test_extra_compute_row_no_negative_time_log():
    """Negative time_on_site_seconds should be clamped to 0 in log."""
    row = _compute_row(_make_lead(time_on_site_seconds=-100))
    # log1p(0) = 0, never negative.
    assert row["time_on_site_log"] == 0.0


def test_extra_compute_row_zero_revenue_log():
    """Zero revenue → log1p(0) = 0."""
    row = _compute_row(_make_lead(company_revenue=0))
    assert row["company_revenue_log"] == 0.0


def test_extra_compute_row_high_revenue_log_positive():
    """Positive revenue → positive log."""
    row = _compute_row(_make_lead(company_revenue=1000))
    assert row["company_revenue_log"] > 0


def test_extra_featurize_returns_correct_columns():
    """featurize() returns object with the requested columns."""
    df = featurize(_make_lead())
    # pandas DataFrame or _DictFrame fallback both have ``columns``.
    assert list(df.columns) == DEFAULT_FEATURES
    assert len(df.rows) == 1


def test_extra_featurize_custom_columns():
    """featurize() with custom names returns only those columns."""
    df = featurize(_make_lead(), names=["has_phone", "has_email"])
    assert list(df.columns) == ["has_phone", "has_email"]
    row = df.rows[0]
    assert "has_phone" in row
    assert "has_email" in row


def test_extra_featurize_fills_missing_columns():
    """featurize() should fill missing columns with 0.0."""
    df = featurize(_make_lead(), names=["has_phone", "this_column_does_not_exist"])
    row = df.rows[0]
    assert row["this_column_does_not_exist"] == 0.0


def test_extra_dictframe_getitem_list():
    """_DictFrame supports list indexing (column projection)."""
    df = _DictFrame([{"a": 1, "b": 2}], ["a", "b"])
    proj = df[["a"]]
    assert isinstance(proj, _DictFrame)
    assert proj.columns == ["a"]
    assert proj.rows == [{"a": 1}]


def test_extra_dictframe_getitem_string():
    """_DictFrame supports string indexing (column)."""
    df = _DictFrame([{"a": 1, "b": 2}], ["a", "b"])
    assert df["a"] == [1]
    assert df["b"] == [2]


def test_extra_dictframe_getitem_invalid_type():
    """_DictFrame raises TypeError on invalid index."""
    df = _DictFrame([{"a": 1}], ["a"])
    import pytest
    with pytest.raises(TypeError):
        _ = df[42]  # type: ignore[arg-type]


def test_extra_dictframe_to_numpy():
    """_DictFrame.to_numpy returns a numpy array."""
    df = _DictFrame([{"a": 1, "b": 2}], ["a", "b"])
    import numpy as np
    arr = df.to_numpy()
    assert arr.shape == (1, 2)
    assert arr[0, 0] == 1
    assert arr[0, 1] == 2
