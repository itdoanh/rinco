"""Lead input schema."""
from __future__ import annotations

from typing import Any, Dict, Optional

from pydantic import BaseModel, ConfigDict, Field


class LeadFeatures(BaseModel):
    """Input features for a lead."""

    model_config = ConfigDict(extra="allow")

    email: Optional[str] = None
    phone: Optional[str] = None
    full_name: Optional[str] = None
    company: Optional[str] = None
    job_title: Optional[str] = None
    industry: Optional[str] = None
    source: str = "unknown"
    campaign_id: Optional[str] = None

    page_views: int = 0
    time_on_site_seconds: int = 0
    has_phone: bool = False
    has_email: bool = True

    utm_source: Optional[str] = None
    utm_medium: Optional[str] = None
    utm_campaign: Optional[str] = None
    fbclid: Optional[str] = None
    gclid: Optional[str] = None
    referrer_domain: Optional[str] = None
    device_type: str = "desktop"
    country: Optional[str] = None

    email_opens: int = 0
    email_clicks: int = 0
    form_fills: int = 0
    abandoned_carts: int = 0
    repeat_visits: int = 0
    source_quality: float = 0.5

    company_size: Optional[str] = None
    company_revenue: Optional[float] = None
    channel_conversion_rate: float = 0.05

    custom_fields: Dict[str, Any] = Field(default_factory=dict)


__all__ = ["LeadFeatures"]