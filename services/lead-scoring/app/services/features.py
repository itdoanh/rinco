"""Feature engineering for lead scoring.

We engineer ~40 features covering demographic, behavioural, engagement
and firmographic signals.  Missing values are filled with the default
neutral value (0 or 0.5).
"""
from __future__ import annotations

from typing import Dict, List

import numpy as np

try:  # pandas is optional at import time
    import pandas as pd  # type: ignore

    _HAS_PANDAS = True
except Exception:  # pragma: no cover
    pd = None  # type: ignore
    _HAS_PANDAS = False

from app.schemas.lead import LeadFeatures

DEFAULT_FEATURES: List[str] = [
    # demographic
    "has_phone", "has_email", "company_length", "name_length",
    "country_vn", "country_us", "country_other",
    # behavioural
    "page_views", "time_on_site_log", "repeat_visits",
    "fbclid_present", "gclid_present", "ttclid_present",
    "is_mobile", "is_tablet", "is_desktop",
    # engagement
    "email_opens", "email_clicks", "form_fills",
    "abandoned_carts", "open_rate", "click_through_rate",
    # firmographic
    "is_b2b", "company_size_sm", "company_size_md", "company_size_lg",
    "company_revenue_log",
    # source / channel
    "source_paid", "source_organic", "source_direct", "source_referral",
    "source_quality", "channel_conv_rate",
    # utm
    "utm_has_campaign", "utm_has_medium",
    # device
    "device_mobile_ratio",
]


def feature_names() -> List[str]:
    return list(DEFAULT_FEATURES)


def featurize(features: LeadFeatures, names: List[str] | None = None) -> "pd.DataFrame":  # type: ignore[name-defined]
    """Return a single-row DataFrame aligned with `names` (or DEFAULT_FEATURES)."""
    requested = list(names) if names else DEFAULT_FEATURES
    row = _compute_row(features)
    for n in requested:
        row.setdefault(n, 0.0)
    if _HAS_PANDAS:
        df = pd.DataFrame([row])
        return df[requested]
    # Fallback: a list-of-dicts representation when pandas is missing.
    return _DictFrame([{n: row[n] for n in requested}], requested)


def _compute_row(s: LeadFeatures) -> Dict[str, float]:
    title = (s.job_title or "").lower()
    is_b2b = int(
        any(
            k in title
            for k in ("manager", "director", "ceo", "cto", "founder", "owner", "head")
        )
    )
    country = (s.country or "").upper()
    src = (s.source or "").lower()
    device = (s.device_type or "").lower()
    size = (s.company_size or "").lower()

    return {
        "has_phone": int(s.has_phone),
        "has_email": int(s.has_email),
        "company_length": len(s.company or ""),
        "name_length": len(s.full_name or ""),
        "country_vn": int(country == "VN"),
        "country_us": int(country == "US"),
        "country_other": int(country not in ("VN", "US", "")),
        "page_views": float(s.page_views),
        "time_on_site_log": float(np.log1p(max(0, s.time_on_site_seconds))),
        "repeat_visits": float(s.repeat_visits),
        "fbclid_present": int(s.fbclid is not None),
        "gclid_present": int(s.gclid is not None),
        "ttclid_present": int(bool(s.utm_source and "tiktok" in s.utm_source)),
        "is_mobile": int(device == "mobile"),
        "is_tablet": int(device == "tablet"),
        "is_desktop": int(device == "desktop"),
        "email_opens": float(s.email_opens),
        "email_clicks": float(s.email_clicks),
        "form_fills": float(s.form_fills),
        "abandoned_carts": float(s.abandoned_carts),
        "open_rate": s.email_opens / max(1, s.email_opens + s.email_clicks + 5),
        "click_through_rate": s.email_clicks / max(1, s.email_opens + 1),
        "is_b2b": float(is_b2b),
        "company_size_sm": int(size in ("1-10", "1-50", "small")),
        "company_size_md": int(size in ("11-50", "51-200", "medium")),
        "company_size_lg": int(size in ("200+", "500+", "1000+", "large")),
        "company_revenue_log": float(np.log1p(s.company_revenue or 0)),
        "source_paid": int(src in ("facebook_ads", "google_ads", "tiktok_ads")),
        "source_organic": int(src in ("organic", "search")),
        "source_direct": int(src in ("direct", "none")),
        "source_referral": int(src in ("referral", "partner")),
        "source_quality": float(s.source_quality),
        "channel_conv_rate": float(s.channel_conversion_rate),
        "utm_has_campaign": int(bool(s.utm_campaign)),
        "utm_has_medium": int(bool(s.utm_medium)),
        "device_mobile_ratio": float(int(device == "mobile")),
    }


class _DictFrame:
    """Minimal DataFrame-like fallback used when pandas is unavailable."""

    def __init__(self, rows: List[Dict[str, float]], columns: List[str]) -> None:
        self.rows = rows
        self.columns = columns

    def __getitem__(self, key):
        if isinstance(key, list):
            return _DictFrame(
                [{k: r[k] for k in key} for r in self.rows], key
            )
        if isinstance(key, str):
            return [r[key] for r in self.rows]
        raise TypeError("unsupported index")

    def to_numpy(self, dtype=None):  # pragma: no cover
        import numpy as np

        return np.array([[r[k] for k in self.columns] for r in self.rows], dtype=dtype)