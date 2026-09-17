"""Seed data and mock data initialization for lead-scoring service."""
from __future__ import annotations

from typing import Dict, List
from app.schemas.lead import LeadFeatures

# Sample leads for testing
SAMPLE_LEADS: List[LeadFeatures] = [
    LeadFeatures(
        email="john.smith@techcorp.com",
        phone="+1-555-0101",
        full_name="John Smith",
        company="TechCorp Inc",
        job_title="CTO",
        industry="Technology",
        source="linkedin",
        page_views=45,
        time_on_site_seconds=1200,
        has_phone=True,
        has_email=True,
        utm_source="linkedin",
        device_type="desktop",
        country="US",
        email_opens=12,
        email_clicks=5,
        form_fills=3,
        repeat_visits=8,
        source_quality=0.9,
        company_size="500-1000",
    ),
    LeadFeatures(
        email="sarah.jones@startup.io",
        phone="+1-555-0102",
        full_name="Sarah Jones",
        company="Startup.io",
        job_title="Founder",
        industry="SaaS",
        source="organic",
        page_views=25,
        time_on_site_seconds=600,
        has_phone=True,
        has_email=True,
        utm_source="google",
        device_type="desktop",
        country="UK",
        email_opens=8,
        email_clicks=3,
        form_fills=1,
        repeat_visits=4,
        source_quality=0.7,
        company_size="10-50",
    ),
    LeadFeatures(
        email="mike.wilson@enterprise.com",
        phone="+1-555-0103",
        full_name="Mike Wilson",
        company="Enterprise Solutions",
        job_title="VP Sales",
        industry="Enterprise Software",
        source="referral",
        page_views=60,
        time_on_site_seconds=1800,
        has_phone=True,
        has_email=True,
        utm_source="referral",
        device_type="desktop",
        country="US",
        email_opens=20,
        email_clicks=10,
        form_fills=5,
        repeat_visits=15,
        source_quality=0.95,
        company_size="1000+",
    ),
    LeadFeatures(
        email="emma.davis@consulting.co",
        full_name="Emma Davis",
        company="Davis Consulting",
        job_title="Managing Partner",
        industry="Consulting",
        source="webinar",
        page_views=10,
        time_on_site_seconds=300,
        has_phone=False,
        has_email=True,
        device_type="mobile",
        country="CA",
        email_opens=2,
        email_clicks=0,
        form_fills=0,
        repeat_visits=1,
        source_quality=0.4,
        company_size="50-200",
    ),
    LeadFeatures(
        email="alex.chen@fintech.sg",
        phone="+65-5555-0104",
        full_name="Alex Chen",
        company="FinTech Solutions",
        job_title="Head of Technology",
        industry="FinTech",
        source="linkedin",
        page_views=35,
        time_on_site_seconds=900,
        has_phone=True,
        has_email=True,
        utm_source="linkedin",
        utm_medium=" sponsored",
        device_type="desktop",
        country="SG",
        email_opens=15,
        email_clicks=7,
        form_fills=2,
        repeat_visits=6,
        source_quality=0.85,
        company_size="200-500",
    ),
]

# Tenant configurations
TENANT_CONFIGS: Dict[str, dict] = {
    "default": {
        "model_version": "1.0.0",
        "thresholds": {"hot": 0.7, "warm": 0.4, "cold": 0.2},
        "features_enabled": True,
    },
    "tenant_acme": {
        "model_version": "1.0.0",
        "thresholds": {"hot": 0.75, "warm": 0.45, "cold": 0.25},
        "features_enabled": True,
    },
    "tenant_globex": {
        "model_version": "1.0.0",
        "thresholds": {"hot": 0.8, "warm": 0.5, "cold": 0.3},
        "features_enabled": True,
    },
}


def get_seed_leads() -> List[LeadFeatures]:
    """Return the sample leads for seeding."""
    return SAMPLE_LEADS


def get_tenant_configs() -> Dict[str, dict]:
    """Return tenant configurations."""
    return TENANT_CONFIGS
