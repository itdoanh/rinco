"""Tests for lead-scoring LeadFeatures schema."""
import pytest
from pydantic import ValidationError

from app.schemas.lead import LeadFeatures


def test_lead_features_default_construction():
    lf = LeadFeatures()
    assert lf.email is None
    assert lf.phone is None
    assert lf.full_name is None
    assert lf.company is None
    assert lf.job_title is None
    assert lf.industry is None
    assert lf.source == "unknown"
    assert lf.page_views == 0
    assert lf.time_on_site_seconds == 0
    assert lf.has_phone is False
    assert lf.has_email is True
    assert lf.utm_source is None
    assert lf.device_type == "desktop"
    assert lf.email_opens == 0
    assert lf.email_clicks == 0
    assert lf.form_fills == 0
    assert lf.abandoned_carts == 0
    assert lf.repeat_visits == 0
    assert lf.source_quality == 0.5
    assert lf.channel_conversion_rate == 0.05
    assert lf.custom_fields == {}


def test_lead_features_full_construction():
    lf = LeadFeatures(
        email="a@b.com",
        phone="+1234",
        full_name="Alice",
        company="ACME",
        job_title="CEO",
        industry="Tech",
        source="web",
        campaign_id="c1",
        page_views=10,
        time_on_site_seconds=300,
        has_phone=True,
        has_email=True,
        utm_source="google",
        utm_medium="cpc",
        utm_campaign="spring",
        fbclid="fb1",
        gclid="g1",
        referrer_domain="google.com",
        device_type="mobile",
        country="US",
        email_opens=2,
        email_clicks=1,
        form_fills=3,
        abandoned_carts=0,
        repeat_visits=2,
        source_quality=0.8,
        company_size="large",
        company_revenue=1000000.0,
        channel_conversion_rate=0.15,
        custom_fields={"foo": "bar"},
    )
    assert lf.email == "a@b.com"
    assert lf.phone == "+1234"
    assert lf.full_name == "Alice"
    assert lf.company == "ACME"
    assert lf.job_title == "CEO"
    assert lf.industry == "Tech"
    assert lf.source == "web"
    assert lf.campaign_id == "c1"
    assert lf.page_views == 10
    assert lf.time_on_site_seconds == 300
    assert lf.has_phone is True
    assert lf.has_email is True
    assert lf.utm_source == "google"
    assert lf.utm_medium == "cpc"
    assert lf.utm_campaign == "spring"
    assert lf.fbclid == "fb1"
    assert lf.gclid == "g1"
    assert lf.referrer_domain == "google.com"
    assert lf.device_type == "mobile"
    assert lf.country == "US"
    assert lf.email_opens == 2
    assert lf.email_clicks == 1
    assert lf.form_fills == 3
    assert lf.abandoned_carts == 0
    assert lf.repeat_visits == 2
    assert lf.source_quality == 0.8
    assert lf.company_size == "large"
    assert lf.company_revenue == 1000000.0
    assert lf.channel_conversion_rate == 0.15
    assert lf.custom_fields == {"foo": "bar"}


def test_lead_features_extra_allowed():
    # extra="allow" should not raise on unknown fields
    lf = LeadFeatures(unknown_field="hello")
    assert lf.model_extra["unknown_field"] == "hello"


def test_lead_features_dict_roundtrip():
    data = {
        "email": "a@b.com",
        "page_views": 5,
        "source": "fb",
    }
    lf = LeadFeatures(**data)
    out = lf.model_dump()
    for k, v in data.items():
        assert out[k] == v


def test_lead_features_invalid_int():
    with pytest.raises(ValidationError):
        LeadFeatures(page_views="not-an-int")


def test_lead_features_custom_fields_default_factory():
    a = LeadFeatures()
    b = LeadFeatures()
    # default_factory creates a new dict each time
    assert a.custom_fields is not b.custom_fields
    assert a.custom_fields == b.custom_fields == {}


def test_lead_features_custom_fields_modification():
    a = LeadFeatures()
    a.custom_fields["foo"] = "bar"
    b = LeadFeatures()
    assert b.custom_fields == {}


def test_lead_features_model_config_extra():
    # Verify config
    cfg = LeadFeatures.model_config
    assert cfg.get("extra") == "allow"
