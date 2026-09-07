"""Tests for the feature engineering module."""
from __future__ import annotations

import math
import os
import sys

ROOT = os.path.dirname(os.path.abspath(__file__))
if ROOT not in sys.path:
    sys.path.insert(0, os.path.dirname(ROOT))

from app.schemas.lead import LeadFeatures
from app.services.features import DEFAULT_FEATURES, featurize, feature_names


def test_default_feature_names_is_a_list():
    names = feature_names()
    assert isinstance(names, list)
    assert len(names) > 10
    assert all(isinstance(n, str) for n in names)


def test_featurize_fills_unknown_features_with_zero():
    feats = LeadFeatures()
    df = featurize(feats, DEFAULT_FEATURES)
    # Every requested feature should be present.
    for col in DEFAULT_FEATURES:
        assert col in df.columns


def test_featurize_log_transform_for_time():
    feats = LeadFeatures(time_on_site_seconds=0)
    df = featurize(feats, DEFAULT_FEATURES)
    assert df.iloc[0]["time_on_site_log"] == 0.0

    feats = LeadFeatures(time_on_site_seconds=7200)
    df = featurize(feats, DEFAULT_FEATURES)
    # log1p(7200) ≈ 8.88
    assert abs(df.iloc[0]["time_on_site_log"] - math.log1p(7200)) < 1e-6


def test_featurize_country_one_hot():
    feats_vn = LeadFeatures(country="VN")
    feats_us = LeadFeatures(country="US")
    feats_unknown = LeadFeatures(country="JP")
    df_vn = featurize(feats_vn, DEFAULT_FEATURES)
    df_us = featurize(feats_us, DEFAULT_FEATURES)
    df_other = featurize(feats_unknown, DEFAULT_FEATURES)
    assert df_vn.iloc[0]["country_vn"] == 1
    assert df_vn.iloc[0]["country_us"] == 0
    assert df_us.iloc[0]["country_us"] == 1
    assert df_us.iloc[0]["country_vn"] == 0
    assert df_other.iloc[0]["country_other"] == 1


def test_featurize_device_detection():
    feats = LeadFeatures(device_type="mobile")
    df = featurize(feats, DEFAULT_FEATURES)
    assert df.iloc[0]["is_mobile"] == 1
    assert df.iloc[0]["is_tablet"] == 0
    assert df.iloc[0]["is_desktop"] == 0


def test_featurize_source_classification():
    feats = LeadFeatures(source="facebook_ads")
    df = featurize(feats, DEFAULT_FEATURES)
    assert df.iloc[0]["source_paid"] == 1
    assert df.iloc[0]["source_organic"] == 0


def test_featurize_open_rate_denominator_safe():
    feats = LeadFeatures(email_opens=0, email_clicks=0)
    df = featurize(feats, DEFAULT_FEATURES)
    # open_rate is `opens / max(1, opens + clicks + 5)`
    assert 0.0 <= df.iloc[0]["open_rate"] <= 1.0


def test_featurize_company_size_buckets():
    small = LeadFeatures(company_size="1-10")
    medium = LeadFeatures(company_size="51-200")
    large = LeadFeatures(company_size="1000+")
    df_small = featurize(small, DEFAULT_FEATURES)
    df_medium = featurize(medium, DEFAULT_FEATURES)
    df_large = featurize(large, DEFAULT_FEATURES)
    assert df_small.iloc[0]["company_size_sm"] == 1
    assert df_medium.iloc[0]["company_size_md"] == 1
    assert df_large.iloc[0]["company_size_lg"] == 1


def test_featurize_b2b_detection():
    feats = LeadFeatures(job_title="VP of Engineering")
    df = featurize(feats, DEFAULT_FEATURES)
    assert df.iloc[0]["is_b2b"] == 1

    feats = LeadFeatures(job_title="student")
    df = featurize(feats, DEFAULT_FEATURES)
    assert df.iloc[0]["is_b2b"] == 0


def test_featurize_aligns_to_feature_names():
    feats = LeadFeatures()
    # Use a subset to confirm alignment.
    df = featurize(feats, ["page_views", "country_vn"])
    assert list(df.columns) == ["page_views", "country_vn"]