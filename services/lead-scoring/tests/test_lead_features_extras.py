"""Extra tests for LeadFeatures schema validation and edge cases."""
from __future__ import annotations

import pytest
from pydantic import ValidationError

from app.schemas.lead import LeadFeatures


def test_extra_lead_minimal_required():
    """Only source has a non-default required value; everything else is optional with defaults."""
    lead = LeadFeatures()
    assert lead.source == "unknown"
    assert lead.page_views == 0
    assert lead.time_on_site_seconds == 0
    assert lead.has_email is True
    assert lead.device_type == "desktop"
    assert lead.source_quality == 0.5
    assert lead.channel_conversion_rate == 0.05


def test_extra_lead_extra_fields_allowed():
    """extra='allow' permits unknown fields."""
    lead = LeadFeatures(custom_field_xyz="value123", another_one=42)
    assert lead.custom_field_xyz == "value123"
    assert lead.another_one == 42


def test_extra_lead_email_optional():
    lead = LeadFeatures()
    assert lead.email is None


def test_extra_lead_full_data():
    lead = LeadFeatures(
        email="test@example.com",
        phone="+1234567890",
        full_name="John Doe",
        company="ACME",
        job_title="CTO",
        industry="tech",
        source="google_ads",
        campaign_id="camp-1",
        page_views=10,
        time_on_site_seconds=300,
        has_phone=True,
        has_email=True,
        utm_source="google",
        utm_medium="cpc",
        utm_campaign="summer",
        fbclid="fb.1.test",
        gclid="gclid.1.test",
        referrer_domain="google.com",
        device_type="mobile",
        country="US",
        email_opens=5,
        email_clicks=2,
        form_fills=1,
        abandoned_carts=0,
        repeat_visits=3,
        source_quality=0.9,
        company_size="100-500",
        company_revenue=50_000_000.0,
        channel_conversion_rate=0.12,
        custom_fields={"vip": True},
    )
    assert lead.email == "test@example.com"
    assert lead.page_views == 10
    assert lead.company_revenue == 50_000_000.0
    assert lead.custom_fields == {"vip": True}


def test_extra_lead_unicode():
    lead = LeadFeatures(
        email="người_dùng@example.com",
        full_name="Nguyễn Văn A",
        company="Công ty ABC",
        industry="công nghệ",
        country="VN",
    )
    assert lead.full_name == "Nguyễn Văn A"
    assert lead.country == "VN"


def test_extra_lead_custom_fields_default_is_empty_dict():
    """Each instance should get its own empty dict (not shared)."""
    a = LeadFeatures()
    b = LeadFeatures()
    assert a.custom_fields == {}
    assert b.custom_fields == {}
    a.custom_fields["x"] = "y"
    assert b.custom_fields == {}  # not shared


def test_extra_lead_source_required_as_string():
    """source is a required string (no default in signature)."""
    with pytest.raises(ValidationError):
        LeadFeatures(source=123)  # type: ignore[arg-type]


def test_extra_lead_device_type_required_as_string():
    with pytest.raises(ValidationError):
        LeadFeatures(device_type=["mobile"])  # type: ignore[arg-type]


def test_extra_lead_negative_integers_allowed():
    """By default, int fields accept negative values (no constraints)."""
    lead = LeadFeatures(page_views=-1, email_opens=-10)
    assert lead.page_views == -1
    assert lead.email_opens == -10


def test_extra_lead_float_fields():
    lead = LeadFeatures(source_quality=0.0, channel_conversion_rate=1.0, company_revenue=0.0)
    assert lead.source_quality == 0.0
    assert lead.channel_conversion_rate == 1.0
    assert lead.company_revenue == 0.0


def test_extra_lead_bool_fields():
    lead = LeadFeatures(has_phone=False, has_email=False)
    assert lead.has_phone is False
    assert lead.has_email is False


def test_extra_lead_json_serialization_roundtrip():
    lead = LeadFeatures(email="x@y.com", page_views=5)
    json = lead.model_dump_json()
    parsed = LeadFeatures.model_validate_json(json)
    assert parsed.email == lead.email
    assert parsed.page_views == lead.page_views


def test_extra_lead_model_dump_contains_all_fields():
    lead = LeadFeatures()
    data = lead.model_dump()
    assert "email" in data
    assert "phone" in data
    assert "source" in data
    assert "custom_fields" in data


def test_extra_lead_model_config_extra_allow():
    """Model config allows extra fields."""
    assert LeadFeatures.model_config.get("extra") == "allow"


def test_extra_lead_utm_fields_optional():
    """utm fields default to None."""
    lead = LeadFeatures()
    assert lead.utm_source is None
    assert lead.utm_medium is None
    assert lead.utm_campaign is None
    assert lead.fbclid is None
    assert lead.gclid is None
    assert lead.referrer_domain is None


def test_extra_lead_company_size_string():
    lead = LeadFeatures(company_size="1000+", company_revenue=1_000_000_000.0)
    assert lead.company_size == "1000+"


def test_extra_lead_negative_zero_metrics():
    """All-zero metrics should be valid."""
    lead = LeadFeatures(
        page_views=0,
        time_on_site_seconds=0,
        email_opens=0,
        email_clicks=0,
        form_fills=0,
        abandoned_carts=0,
        repeat_visits=0,
    )
    assert lead.page_views == 0


def test_extra_lead_extra_field_preserved():
    """Custom fields are accessible via attribute access (extra=allow)."""
    lead = LeadFeatures(score_priority=10, lead_tier="A")
    assert lead.score_priority == 10
    assert lead.lead_tier == "A"
