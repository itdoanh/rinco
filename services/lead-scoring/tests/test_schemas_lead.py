"""Tests for lead-scoring schemas."""
from __future__ import annotations

from app.schemas.lead import LeadFeatures


def test_extra_lead_features_minimal():
    """LeadFeatures with no arguments should have defaults."""
    l = LeadFeatures()
    assert l.source == "unknown"
    assert l.device_type == "desktop"
    assert l.has_phone is False
    assert l.has_email is True
    assert l.page_views == 0
    assert l.time_on_site_seconds == 0
    assert l.email_opens == 0
    assert l.email_clicks == 0
    assert l.form_fills == 0
    assert l.abandoned_carts == 0
    assert l.repeat_visits == 0
    assert l.source_quality == 0.5
    assert l.channel_conversion_rate == 0.05


def test_extra_lead_features_with_data():
    """LeadFeatures should accept all fields."""
    l = LeadFeatures(
        email="test@example.com",
        phone="+84901234567",
        full_name="John Doe",
        company="ACME",
        job_title="CEO",
        source="facebook_ads",
        page_views=10,
        time_on_site_seconds=120,
        has_phone=True,
        has_email=True,
        country="VN",
        company_size="100+",
        company_revenue=5000000,
    )
    assert l.email == "test@example.com"
    assert l.phone == "+84901234567"
    assert l.company == "ACME"
    assert l.job_title == "CEO"
    assert l.country == "VN"


def test_extra_lead_features_custom_fields():
    """LeadFeatures should accept custom_fields dict."""
    l = LeadFeatures(custom_fields={"key1": "value1", "key2": 42})
    assert l.custom_fields["key1"] == "value1"
    assert l.custom_fields["key2"] == 42


def test_extra_lead_features_extra_allowed():
    """LeadFeatures should allow extra fields (model_config)."""
    l = LeadFeatures(unknown_field="anything")
    assert l.unknown_field == "anything"


def test_extra_lead_features_negative_ints():
    """LeadFeatures should accept negative ints (no validation)."""
    l = LeadFeatures(page_views=-1, time_on_site_seconds=-10)
    assert l.page_views == -1


def test_extra_lead_features_uuid_utm():
    """LeadFeatures should accept UTM params."""
    l = LeadFeatures(
        utm_source="google",
        utm_medium="cpc",
        utm_campaign="summer-sale",
    )
    assert l.utm_source == "google"
    assert l.utm_campaign == "summer-sale"


def test_extra_lead_features_clids():
    """LeadFeatures should accept click IDs."""
    l = LeadFeatures(
        fbclid="fb.1.xyz",
        gclid="gclid.123",
    )
    assert l.fbclid == "fb.1.xyz"
    assert l.gclid == "gclid.123"


def test_extra_lead_features_device_types():
    """LeadFeatures should accept device types."""
    for dt in ["desktop", "mobile", "tablet"]:
        l = LeadFeatures(device_type=dt)
        assert l.device_type == dt


def test_extra_lead_features_country_codes():
    """LeadFeatures should accept ISO country codes."""
    for cc in ["VN", "US", "JP", "DE", "FR"]:
        l = LeadFeatures(country=cc)
        assert l.country == cc


def test_extra_lead_features_source_variations():
    """LeadFeatures should accept various source values."""
    for src in ["organic", "facebook_ads", "google_ads", "tiktok_ads", "direct", "referral"]:
        l = LeadFeatures(source=src)
        assert l.source == src


def test_extra_lead_features_company_size_variations():
    """LeadFeatures should accept company_size strings."""
    for size in ["1-10", "11-50", "51-200", "200+", "1000+", "small", "medium", "large"]:
        l = LeadFeatures(company_size=size)
        assert l.company_size == size


def test_extra_lead_features_unicode():
    """LeadFeatures should support unicode."""
    l = LeadFeatures(
        full_name="Nguyễn Văn A",
        company="Công ty ABC",
        country="VN",
    )
    assert "Nguyễn" in l.full_name


def test_extra_lead_features_repr_doesnt_error():
    """LeadFeatures repr should not error."""
    l = LeadFeatures(email="test@example.com")
    s = repr(l)
    assert "LeadFeatures" in s


def test_extra_lead_features_dict_roundtrip():
    """LeadFeatures should round-trip via dict()."""
    l = LeadFeatures(
        email="a@b.c",
        page_views=5,
    )
    d = l.model_dump()
    l2 = LeadFeatures(**d)
    assert l.email == l2.email
    assert l.page_views == l2.page_views
