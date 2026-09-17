"""Extended mock data for the lead-scoring service (WS-B Loop 9).

Provides 100+ realistic leads across the 3 demo tenants and
500 training rows with feature/label distributions that make the
ML model pick up realistic patterns.
"""
from __future__ import annotations

import random
from typing import Any, Dict, List

from app.schemas.lead import LeadFeatures

__all__ = [
    "EXTENDED_MOCK_LEADS",
    "EXTENDED_TRAINING_DATASET",
]


# ---------------------------------------------------------------------------
# Vietnamese / international names (deterministic — full names)
# ---------------------------------------------------------------------------
_FIRST_NAMES = [
    "Nguyen Van", "Tran Thi", "Le Hoang", "Pham Thi", "Hoang Van",
    "Vu Thi", "Dang Van", "Bui Thi", "Do Van", "Ngo Thi",
    "Hoang Mai", "Nguyen Bao", "Tran Duc", "Le Thi", "Pham Van",
    "Vu My", "Dang Quoc", "Bui Thi", "Do Van", "Ngo Cam",
    "Hoang Thanh", "To Khanh", "Phung Thi", "Ly Van", "Truong Bich",
]

_LAST_NAMES = [
    "An", "Binh", "Cuong", "Dung", "Em",
    "Phuong", "Giang", "Hoa", "Ich", "Khanh",
    "Lam", "Mai", "Nam", "Oanh", "Phuc",
    "Quynh", "Suong", "Tai", "Uyen", "Vinh",
    "Xuan", "Yen", "Anh", "Bich", "Cuc",
]

_INDUSTRIES = ["saas", "fintech", "manufacturing", "retail", "healthcare", "education", "real_estate", "logistics", "hospitality", "ecommerce"]
_SOURCES = ["referral", "linkedin", "google-ads", "facebook", "tiktok", "zalo", "email", "organic", "webinar", "demo"]
_COUNTRIES = ["VN", "US", "SG", "JP", "AU", "KR", "GB", "DE", "FR", "CA"]
_DEVICE_TYPES = ["desktop", "mobile", "tablet"]
_COMPANY_SIZES = ["1-10", "11-50", "51-200", "201-1000", "1000+"]
_JOB_TITLES = ["CEO", "CTO", "CFO", "VP of Engineering", "Marketing Manager", "Sales Lead", "Product Manager", "Director of Sales", "Operations Manager", "Customer Success Lead"]


def _make_lead(i: int, tenant_slug: str, tenant_id: str) -> Dict[str, Any]:
    """Deterministically create a realistic lead."""
    first = _FIRST_NAMES[i % len(_FIRST_NAMES)]
    last = _LAST_NAMES[(i // len(_FIRST_NAMES)) % len(_LAST_NAMES)]
    full_name = f"{first} {last} {i}"
    industry = _INDUSTRIES[i % len(_INDUSTRIES)]
    source = _SOURCES[i % len(_SOURCES)]
    country = _COUNTRIES[i % len(_COUNTRIES)]
    company_size = _COMPANY_SIZES[(i // 4) % len(_COMPANY_SIZES)]

    # Engagement: 0-100 scale — roughly correlated with company size
    base_eng = (i * 7 + 13) % 100
    page_views = max(1, base_eng // 2)
    time_on_site = base_eng * 12
    email_opens = base_eng // 10
    email_clicks = email_opens // 3
    form_fills = base_eng // 25
    repeat_visits = base_eng // 15

    return {
        "lead_id": f"lead-{tenant_slug}-{i:04d}",
        "tenant_id": tenant_id,
        "full_name": full_name,
        "email": f"{first.lower().replace(' ', '')}.{last.lower()}{i}@{tenant_slug}.demo",
        "company": f"{last} {industry.title()} {i}",
        "job_title": _JOB_TITLES[i % len(_JOB_TITLES)],
        "industry": industry,
        "source": source,
        "page_views": page_views,
        "time_on_site_seconds": time_on_site,
        "has_email": email_opens > 0,
        "has_phone": i % 3 != 0,
        "email_opens": email_opens,
        "email_clicks": email_clicks,
        "form_fills": form_fills,
        "repeat_visits": repeat_visits,
        "company_size": company_size,
        "company_revenue": float(((i % 10) + 1) * 50_000_000),
        "country": country,
        "device_type": _DEVICE_TYPES[i % 3],
        "utm_source": source,
        "utm_medium": ["cpc", "organic", "social", "referral"][i % 4],
        "utm_campaign": [f"campaign_{i % 8}", f"promo_q{i % 4}", f"launch_v{i % 3}"][i % 3],
    }


# ---------------------------------------------------------------------------
# Build extended mock leads (100+ across 3 demo tenants)
# ---------------------------------------------------------------------------
_TENANTS = [
    ("apexfintech", "demo-tenant-apexfintech"),
    ("hct-consulting", "demo-tenant-hct-consulting"),
    ("demo-company", "demo-tenant"),
]

EXTENDED_MOCK_LEADS: List[Dict[str, Any]] = []
for tenant_slug, tenant_id in _TENANTS:
    for i in range(35):
        EXTENDED_MOCK_LEADS.append(_make_lead(i, tenant_slug, tenant_id))
# Total: 105 leads


# ---------------------------------------------------------------------------
# Build extended training dataset (500 rows)
# ---------------------------------------------------------------------------
def _extended_training(n: int = 500) -> List[Dict[str, Any]]:
    """Generate 500 rows with realistic feature distribution.

    The label is computed using a non-linear rule so XGBoost has
    something interesting to learn (vs. trivial linear separation).
    """
    rng = random.Random(2026)
    rows: List[Dict[str, Any]] = []
    for i in range(n):
        # Heterogeneous distributions per tenant
        industry = _INDUSTRIES[i % len(_INDUSTRIES)]
        source = _SOURCES[i % len(_SOURCES)]
        company_size = _COMPANY_SIZES[(i // 10) % len(_COMPANY_SIZES)]

        # Features correlated with industry/source/company_size
        size_factor = {"1-10": 0.3, "11-50": 0.5, "51-200": 0.7, "201-1000": 0.9, "1000+": 1.0}[company_size]
        source_factor = {
            "referral": 1.4, "linkedin": 1.2, "google-ads": 1.0,
            "facebook": 0.8, "tiktok": 0.6, "zalo": 0.9,
            "email": 1.0, "organic": 1.1, "webinar": 1.3, "demo": 1.5
        }[source]

        page_views = max(0, int(rng.gauss(20 * size_factor, 12)))
        time_on_site = max(0, int(rng.gauss(800 * size_factor, 400)))
        opens = max(0, int(rng.gauss(4 * size_factor, 2)))
        clicks = max(0, int(min(opens, rng.gauss(1.5 * size_factor, 1))))
        form_fills = max(0, int(min(3, rng.gauss(1.0 * size_factor, 0.7))))
        repeat_visits = max(0, int(rng.gauss(2 * size_factor, 1.5)))
        abandoned = max(0, int(rng.gauss(0.8, 0.6)))
        video_watched = max(0, int(rng.gauss(2, 1.5)))

        # Non-linear combination + noise
        score = (
            page_views * 0.02
            + time_on_site * 0.0008
            + opens * 0.08
            + clicks * 0.15
            + form_fills * 0.30
            + repeat_visits * 0.05
            - abandoned * 0.10
            + video_watched * 0.04
            + size_factor * 0.5
            + source_factor * 0.3
            + rng.gauss(0, 0.3)
        )
        label = 1 if score > 0.85 else 0

        rows.append({
            "row_id": f"row-ext-{i:04d}",
            "industry": industry,
            "source": source,
            "company_size": company_size,
            "features": {
                "page_views": page_views,
                "time_on_site_seconds": time_on_site,
                "email_opens": opens,
                "email_clicks": clicks,
                "form_fills": form_fills,
                "repeat_visits": repeat_visits,
                "abandoned_carts": abandoned,
                "video_watched": video_watched,
            },
            "label": label,
            "score_raw": round(score, 4),
        })
    return rows


EXTENDED_TRAINING_DATASET: List[Dict[str, Any]] = _extended_training()


# ---------------------------------------------------------------------------
# Vietnamese contact directory for enrichment / lookup
# ---------------------------------------------------------------------------
VIETNAMESE_CONTACTS: List[Dict[str, Any]] = [
    {
        "email": f"{first.lower().replace(' ', '')}.{last.lower()}@apexfintech.demo",
        "full_name": f"{first} {last}",
        "tenant": "apexfintech",
        "phone": f"+8490{(i*97 + 1234567) % 10000000:07d}",
        "company": f"{last} Corp",
        "job_title": _JOB_TITLES[i % len(_JOB_TITLES)],
    }
    for i, (first, last) in enumerate([
        ("Nguyen Van", "An"), ("Tran Thi", "Binh"), ("Le Hoang", "Cuong"),
        ("Pham Thi", "Dung"), ("Hoang Van", "Em"), ("Vu Thi", "Phuong"),
        ("Dang Van", "Giang"), ("Bui Thi", "Hoa"), ("Do Van", "Ich"),
        ("Ngo Thi", "Khanh"), ("Hoang Thanh", "Son"), ("To Khanh", "Tam"),
        ("Phung Thi", "Uyen"), ("Ly Van", "Viet"), ("Truong Bich", "Xuan"),
    ])
]


# ---------------------------------------------------------------------------
# MLflow tags/params for the demo training runs
# ---------------------------------------------------------------------------
MLFLOW_PARAMS = {
    "tenant_id": "demo-tenant",
    "model_type": "xgboost",
    "n_estimators": 200,
    "max_depth": 6,
    "learning_rate": 0.05,
    "subsample": 0.8,
    "colsample_bytree": 0.8,
    "random_state": 42,
    "n_training_rows": 500,
    "n_features": 8,
}

MLFLOW_METRICS_TYPICAL = {
    "accuracy": 0.87,
    "auc": 0.93,
    "f1": 0.85,
    "precision": 0.88,
    "recall": 0.83,
}
