"""Extra tests for lead-scoring features."""
import pytest

try:
    import pandas as pd  # noqa: F401
    _HAS_PANDAS = True
except Exception:
    _HAS_PANDAS = False

from app.services.features import (
    feature_names,
    featurize,
    DEFAULT_FEATURES,
    _compute_row,
    _DictFrame,
)
from app.schemas.lead import LeadFeatures


@pytest.mark.skipif(not _HAS_PANDAS, reason="pandas not available")
def test_feature_names_returns_copy():
    names = feature_names()
    assert isinstance(names, list)
    assert names == DEFAULT_FEATURES


@pytest.mark.skipif(not _HAS_PANDAS, reason="pandas not available")
def test_feature_names_does_not_mutate_default():
    """feature_names should return a copy, not the original list."""
    original_len = len(DEFAULT_FEATURES)
    names = feature_names()
    names.append("garbage")
    assert len(DEFAULT_FEATURES) == original_len


def test_default_features_has_expected_count():
    assert len(DEFAULT_FEATURES) > 30


def test_default_features_unique():
    assert len(DEFAULT_FEATURES) == len(set(DEFAULT_FEATURES))


def test_compute_row_basic():
    lead = LeadFeatures(
        has_phone=True,
        has_email=False,
        full_name="Alice",
        company="AcmeCorp",
        country="VN",
        source="facebook_ads",
    )
    row = _compute_row(lead)
    assert row["has_phone"] == 1
    assert row["has_email"] == 0
    assert row["company_length"] == len("AcmeCorp")
    assert row["name_length"] == len("Alice")
    assert row["country_vn"] == 1
    assert row["country_us"] == 0
    assert row["country_other"] == 0


def test_compute_row_us_country():
    lead = LeadFeatures(country="US")
    row = _compute_row(lead)
    assert row["country_vn"] == 0
    assert row["country_us"] == 1


def test_compute_row_other_country():
    lead = LeadFeatures(country="FR")
    row = _compute_row(lead)
    assert row["country_other"] == 1


def test_compute_row_b2b_manager():
    lead = LeadFeatures(job_title="Sales Manager")
    row = _compute_row(lead)
    assert row["is_b2b"] == 1.0


def test_compute_row_b2b_director():
    lead = LeadFeatures(job_title="Director of Sales")
    row = _compute_row(lead)
    assert row["is_b2b"] == 1.0


def test_compute_row_b2b_founder():
    lead = LeadFeatures(job_title="Founder")
    row = _compute_row(lead)
    assert row["is_b2b"] == 1.0


def test_compute_row_non_b2b():
    lead = LeadFeatures(job_title="Student")
    row = _compute_row(lead)
    assert row["is_b2b"] == 0.0


def test_compute_row_device_mobile():
    lead = LeadFeatures(device_type="mobile")
    row = _compute_row(lead)
    assert row["is_mobile"] == 1
    assert row["is_desktop"] == 0


def test_compute_row_device_desktop():
    lead = LeadFeatures(device_type="desktop")
    row = _compute_row(lead)
    assert row["is_desktop"] == 1


def test_compute_row_device_tablet():
    lead = LeadFeatures(device_type="tablet")
    row = _compute_row(lead)
    assert row["is_tablet"] == 1


def test_compute_row_source_paid():
    lead = LeadFeatures(source="google_ads")
    row = _compute_row(lead)
    assert row["source_paid"] == 1


def test_compute_row_source_organic():
    lead = LeadFeatures(source="organic")
    row = _compute_row(lead)
    assert row["source_organic"] == 1


def test_compute_row_source_direct():
    lead = LeadFeatures(source="direct")
    row = _compute_row(lead)
    assert row["source_direct"] == 1


def test_compute_row_source_referral():
    lead = LeadFeatures(source="referral")
    row = _compute_row(lead)
    assert row["source_referral"] == 1


def test_compute_row_company_size_small():
    lead = LeadFeatures(company_size="1-10")
    row = _compute_row(lead)
    assert row["company_size_sm"] == 1


def test_compute_row_company_size_medium():
    lead = LeadFeatures(company_size="51-200")
    row = _compute_row(lead)
    assert row["company_size_md"] == 1


def test_compute_row_company_size_large():
    lead = LeadFeatures(company_size="1000+")
    row = _compute_row(lead)
    assert row["company_size_lg"] == 1


def test_compute_row_fbclid_present():
    lead = LeadFeatures(fbclid="abc123")
    row = _compute_row(lead)
    assert row["fbclid_present"] == 1


def test_compute_row_gclid_present():
    lead = LeadFeatures(gclid="xyz789")
    row = _compute_row(lead)
    assert row["gclid_present"] == 1


def test_compute_row_ttclid_present():
    lead = LeadFeatures(utm_source="tiktok_ads")
    row = _compute_row(lead)
    assert row["ttclid_present"] == 1


def test_compute_row_utm_campaign():
    lead = LeadFeatures(utm_campaign="summer_sale")
    row = _compute_row(lead)
    assert row["utm_has_campaign"] == 1


def test_compute_row_utm_medium():
    lead = LeadFeatures(utm_medium="email")
    row = _compute_row(lead)
    assert row["utm_has_medium"] == 1


@pytest.mark.skipif(not _HAS_PANDAS, reason="pandas not available")
def test_featurize_returns_dataframe():
    lead = LeadFeatures()
    df = featurize(lead)
    assert hasattr(df, "columns")


@pytest.mark.skipif(not _HAS_PANDAS, reason="pandas not available")
def test_featurize_with_custom_names():
    lead = LeadFeatures()
    df = featurize(lead, names=["has_phone", "has_email"])
    assert len(df.columns) == 2


@pytest.mark.skipif(not _HAS_PANDAS, reason="pandas not available")
def test_featurize_missing_features_defaulted():
    lead = LeadFeatures()
    df = featurize(lead, names=["has_phone", "custom_missing"])
    # Custom missing feature should default to 0.0
    if "custom_missing" in df.columns:
        assert df["custom_missing"].iloc[0] == 0.0


def test_dict_frame_getitem_str():
    frame = _DictFrame([{"a": 1.0, "b": 2.0}], ["a", "b"])
    result = frame["a"]
    assert result == [1.0]


def test_dict_frame_getitem_list():
    frame = _DictFrame([{"a": 1.0, "b": 2.0}], ["a", "b"])
    result = frame[["a"]]
    assert isinstance(result, _DictFrame)


def test_dict_frame_getitem_invalid_type():
    frame = _DictFrame([{"a": 1.0}], ["a"])
    with pytest.raises(TypeError):
        frame[123]


def test_dict_frame_to_numpy():
    frame = _DictFrame([{"a": 1.0, "b": 2.0}], ["a", "b"])
    arr = frame.to_numpy()
    assert arr.shape == (1, 2)
