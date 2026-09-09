"""Extra tests for app.services.features — feature engineering edge cases."""
from __future__ import annotations

import math

import pytest

from app.schemas.lead import LeadFeatures
from app.services.features import (
    DEFAULT_FEATURES,
    _compute_row,
    feature_names,
    featurize,
)


def test_extra_feature_names_returns_default():
    names = feature_names()
    assert names == DEFAULT_FEATURES
    assert isinstance(names, list)
    assert len(names) > 30


def test_extra_feature_names_is_copy():
    """feature_names() should return a copy, not the module-level list."""
    a = feature_names()
    a.append("hacked")
    assert "hacked" not in DEFAULT_FEATURES


def test_extra_default_features_contains_expected():
    expected = [
        "has_phone", "has_email", "company_length", "name_length",
        "page_views", "email_opens", "form_fills",
        "is_b2b", "company_revenue_log",
        "source_quality", "channel_conv_rate",
    ]
    for f in expected:
        assert f in DEFAULT_FEATURES


def test_extra_compute_row_minimal_features():
    """With default LeadFeatures, compute_row should not crash."""
    s = LeadFeatures()
    row = _compute_row(s)
    assert row["has_phone"] == 0
    assert row["has_email"] == 1
    assert row["country_vn"] == 0
    assert row["country_us"] == 0
    assert row["country_other"] == 0
    assert row["is_b2b"] == 0
    assert row["is_mobile"] == 0
    assert row["is_desktop"] == 1  # default device_type is "desktop"
    assert row["source_paid"] == 0
    assert row["source_organic"] == 0


def test_extra_compute_row_country_vn():
    s = LeadFeatures(country="VN")
    row = _compute_row(s)
    assert row["country_vn"] == 1
    assert row["country_us"] == 0
    assert row["country_other"] == 0


def test_extra_compute_row_country_us():
    s = LeadFeatures(country="US")
    row = _compute_row(s)
    assert row["country_us"] == 1
    assert row["country_vn"] == 0


def test_extra_compute_row_country_other():
    s = LeadFeatures(country="FR")
    row = _compute_row(s)
    assert row["country_other"] == 1


def test_extra_compute_row_country_case_insensitive():
    s = LeadFeatures(country="vn")
    row = _compute_row(s)
    assert row["country_vn"] == 1


def test_extra_compute_row_is_b2b_manager():
    s = LeadFeatures(job_title="Engineering Manager")
    assert _compute_row(s)["is_b2b"] == 1.0


def test_extra_compute_row_is_b2b_director():
    s = LeadFeatures(job_title="Director of Sales")
    assert _compute_row(s)["is_b2b"] == 1.0


def test_extra_compute_row_is_b2b_ceo():
    s = LeadFeatures(job_title="CEO")
    assert _compute_row(s)["is_b2b"] == 1.0


def test_extra_compute_row_is_b2b_cto():
    s = LeadFeatures(job_title="CTO")
    assert _compute_row(s)["is_b2b"] == 1.0


def test_extra_compute_row_is_b2b_founder():
    s = LeadFeatures(job_title="Co-Founder")
    assert _compute_row(s)["is_b2b"] == 1.0


def test_extra_compute_row_is_b2b_owner():
    s = LeadFeatures(job_title="Business Owner")
    assert _compute_row(s)["is_b2b"] == 1.0


def test_extra_compute_row_is_b2b_head():
    s = LeadFeatures(job_title="Head of Marketing")
    assert _compute_row(s)["is_b2b"] == 1.0


def test_extra_compute_row_not_b2b():
    s = LeadFeatures(job_title="Student")
    assert _compute_row(s)["is_b2b"] == 0


def test_extra_compute_row_empty_title():
    s = LeadFeatures(job_title=None)
    assert _compute_row(s)["is_b2b"] == 0


def test_extra_compute_row_device_mobile():
    s = LeadFeatures(device_type="mobile")
    row = _compute_row(s)
    assert row["is_mobile"] == 1
    assert row["is_tablet"] == 0
    assert row["is_desktop"] == 0


def test_extra_compute_row_device_tablet():
    s = LeadFeatures(device_type="tablet")
    row = _compute_row(s)
    assert row["is_mobile"] == 0
    assert row["is_tablet"] == 1
    assert row["is_desktop"] == 0


def test_extra_compute_row_device_desktop():
    s = LeadFeatures(device_type="desktop")
    row = _compute_row(s)
    assert row["is_mobile"] == 0
    assert row["is_tablet"] == 0
    assert row["is_desktop"] == 1


def test_extra_compute_row_device_unknown():
    s = LeadFeatures(device_type="smartwatch")
    row = _compute_row(s)
    assert row["is_mobile"] == 0
    assert row["is_tablet"] == 0
    assert row["is_desktop"] == 0


def test_extra_compute_row_source_paid():
    for src in ("facebook_ads", "google_ads", "tiktok_ads"):
        s = LeadFeatures(source=src)
        assert _compute_row(s)["source_paid"] == 1


def test_extra_compute_row_source_organic():
    for src in ("organic", "search"):
        s = LeadFeatures(source=src)
        assert _compute_row(s)["source_organic"] == 1


def test_extra_compute_row_source_direct():
    for src in ("direct", "none"):
        s = LeadFeatures(source=src)
        assert _compute_row(s)["source_direct"] == 1


def test_extra_compute_row_source_referral():
    for src in ("referral", "partner"):
        s = LeadFeatures(source=src)
        assert _compute_row(s)["source_referral"] == 1


def test_extra_compute_row_company_size_sm():
    for size in ("1-10", "1-50", "small"):
        s = LeadFeatures(company_size=size)
        row = _compute_row(s)
        assert row["company_size_sm"] == 1
        assert row["company_size_md"] == 0
        assert row["company_size_lg"] == 0


def test_extra_compute_row_company_size_md():
    for size in ("11-50", "51-200", "medium"):
        s = LeadFeatures(company_size=size)
        row = _compute_row(s)
        assert row["company_size_md"] == 1
        assert row["company_size_sm"] == 0
        assert row["company_size_lg"] == 0


def test_extra_compute_row_company_size_lg():
    for size in ("200+", "500+", "1000+", "large"):
        s = LeadFeatures(company_size=size)
        row = _compute_row(s)
        assert row["company_size_lg"] == 1
        assert row["company_size_sm"] == 0
        assert row["company_size_md"] == 0


def test_extra_compute_row_company_size_unknown():
    s = LeadFeatures(company_size="unknown")
    row = _compute_row(s)
    assert row["company_size_sm"] == 0
    assert row["company_size_md"] == 0
    assert row["company_size_lg"] == 0


def test_extra_compute_row_company_length():
    s = LeadFeatures(company="ACME")
    assert _compute_row(s)["company_length"] == 4
    s = LeadFeatures(company="")
    assert _compute_row(s)["company_length"] == 0
    s = LeadFeatures(company=None)
    assert _compute_row(s)["company_length"] == 0


def test_extra_compute_row_name_length():
    s = LeadFeatures(full_name="John Doe")
    assert _compute_row(s)["name_length"] == 8
    s = LeadFeatures(full_name="")
    assert _compute_row(s)["name_length"] == 0


def test_extra_compute_row_fbclid_present():
    s = LeadFeatures(fbclid="fb.1.xyz")
    assert _compute_row(s)["fbclid_present"] == 1


def test_extra_compute_row_fbclid_absent():
    s = LeadFeatures()
    assert _compute_row(s)["fbclid_present"] == 0


def test_extra_compute_row_gclid_present():
    s = LeadFeatures(gclid="gclid.1.xyz")
    assert _compute_row(s)["gclid_present"] == 1


def test_extra_compute_row_ttclid_via_utm():
    s = LeadFeatures(utm_source="tiktok")
    assert _compute_row(s)["ttclid_present"] == 1


def test_extra_compute_row_ttclid_absent():
    s = LeadFeatures(utm_source="facebook")
    assert _compute_row(s)["ttclid_present"] == 0


def test_extra_compute_row_time_on_site_log():
    """time_on_site_log = log1p(time_on_site_seconds)."""
    s = LeadFeatures(time_on_site_seconds=0)
    assert _compute_row(s)["time_on_site_log"] == 0.0
    s = LeadFeatures(time_on_site_seconds=10)
    assert abs(_compute_row(s)["time_on_site_log"] - math.log1p(10)) < 1e-6
    s = LeadFeatures(time_on_site_seconds=1000)
    assert abs(_compute_row(s)["time_on_site_log"] - math.log1p(1000)) < 1e-6


def test_extra_compute_row_negative_time_clamped_to_zero():
    """Negative time_on_site is clamped via max(0, ...)."""
    s = LeadFeatures(time_on_site_seconds=-100)
    assert _compute_row(s)["time_on_site_log"] == 0.0


def test_extra_compute_row_company_revenue_log():
    s = LeadFeatures(company_revenue=0)
    assert _compute_row(s)["company_revenue_log"] == 0.0
    s = LeadFeatures(company_revenue=1_000_000)
    assert abs(_compute_row(s)["company_revenue_log"] - math.log1p(1_000_000)) < 1e-6
    s = LeadFeatures(company_revenue=None)
    assert _compute_row(s)["company_revenue_log"] == 0.0


def test_extra_compute_row_open_rate():
    """open_rate = opens / max(1, opens + clicks + 5)."""
    s = LeadFeatures(email_opens=10, email_clicks=5)
    expected = 10 / max(1, 10 + 5 + 5)
    assert abs(_compute_row(s)["open_rate"] - expected) < 1e-6


def test_extra_compute_row_click_through_rate():
    """click_through_rate = clicks / max(1, opens + 1)."""
    s = LeadFeatures(email_opens=10, email_clicks=5)
    expected = 5 / max(1, 10 + 1)
    assert abs(_compute_row(s)["click_through_rate"] - expected) < 1e-6


def test_extra_compute_row_open_rate_zero_opens():
    s = LeadFeatures(email_opens=0, email_clicks=0)
    rate = _compute_row(s)["open_rate"]
    assert rate == 0 / max(1, 0 + 0 + 5)  # = 0


def test_extra_featurize_returns_single_row():
    """featurize returns exactly one row."""
    s = LeadFeatures(country="VN")
    df = featurize(s)
    # Depending on whether pandas is available, df is DataFrame or _DictFrame.
    if hasattr(df, "__len__") and not hasattr(df, "rows"):
        assert len(df) == 1
    else:
        # _DictFrame has .rows attribute
        assert len(df.rows) == 1


def test_extra_featurize_default_columns():
    """featurize with no names uses DEFAULT_FEATURES."""
    s = LeadFeatures()
    df = featurize(s)
    cols = list(df.columns)
    assert cols == DEFAULT_FEATURES


def test_extra_featurize_custom_names():
    """featurize with custom names returns only those columns."""
    s = LeadFeatures()
    df = featurize(s, names=["has_phone", "has_email"])
    cols = list(df.columns)
    assert cols == ["has_phone", "has_email"]


def test_extra_featurize_extra_names_defaulted_to_zero():
    """If names include unknown features, they're filled with 0."""
    s = LeadFeatures()
    df = featurize(s, names=["has_phone", "unknown_feature_xyz"])
    # Pull column value (works for both DataFrame and _DictFrame).
    col = df["unknown_feature_xyz"]
    if isinstance(col, list):
        assert col[0] == 0.0
    else:
        assert col.iloc[0] == 0.0


def test_extra_featurize_row_values():
    s = LeadFeatures(country="VN", has_phone=True)
    df = featurize(s, names=["country_vn", "has_phone", "has_email"])
    for col_name, expected in [("country_vn", 1.0), ("has_phone", 1.0), ("has_email", 1.0)]:
        col = df[col_name]
        if isinstance(col, list):
            assert col[0] == expected, f"{col_name}: {col[0]} != {expected}"
        else:
            assert col.iloc[0] == expected, f"{col_name}: {col.iloc[0]} != {expected}"


def test_extra_featurize_no_pandas_uses_dictframe():
    """Force the no-pandas fallback path."""
    import app.services.features as f_mod

    original_has_pandas = f_mod._HAS_PANDAS
    f_mod._HAS_PANDAS = False
    try:
        s = LeadFeatures()
        result = f_mod.featurize(s, names=["has_phone"])
        # Should be a _DictFrame instance
        assert isinstance(result, f_mod._DictFrame)
        assert result.columns == ["has_phone"]
        assert result.rows[0]["has_phone"] == 0.0
    finally:
        f_mod._HAS_PANDAS = original_has_pandas


def test_extra_dictframe_to_numpy():
    """_DictFrame.to_numpy() returns a numpy array."""
    import numpy as np

    from app.services.features import _DictFrame

    frame = _DictFrame([{"a": 1.0, "b": 2.0}], ["a", "b"])
    arr = frame.to_numpy()
    assert arr.shape == (1, 2)
    assert arr[0][0] == 1.0
    assert arr[0][1] == 2.0


def test_extra_dictframe_to_numpy_dtype():
    import numpy as np

    from app.services.features import _DictFrame

    frame = _DictFrame([{"a": 1, "b": 2}], ["a", "b"])
    arr = frame.to_numpy(dtype=np.float32)
    assert arr.dtype == np.float32


def test_extra_dictframe_getitem_str():
    from app.services.features import _DictFrame

    frame = _DictFrame([{"a": 1.0, "b": 2.0}, {"a": 3.0, "b": 4.0}], ["a", "b"])
    assert frame["a"] == [1.0, 3.0]
    assert frame["b"] == [2.0, 4.0]


def test_extra_dictframe_getitem_list():
    from app.services.features import _DictFrame

    frame = _DictFrame([{"a": 1.0, "b": 2.0, "c": 3.0}], ["a", "b", "c"])
    sub = frame[["a", "c"]]
    assert isinstance(sub, _DictFrame)
    assert sub.columns == ["a", "c"]
    assert sub.rows[0]["a"] == 1.0
    assert sub.rows[0]["c"] == 3.0


def test_extra_dictframe_getitem_invalid_type():
    from app.services.features import _DictFrame

    frame = _DictFrame([{"a": 1.0}], ["a"])
    with pytest.raises(TypeError):
        frame[42]  # type: ignore[arg-type]
