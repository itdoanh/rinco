"""Seed mock data for the lead-scoring service.

Provides deterministic, ready-to-use mock leads and training data so the
service can serve realistic responses out-of-the-box without needing the
full CRM / data-pipeline.

This module is imported lazily by ``app.main`` if the env var
``LEAD_SCORING_SEED=1`` is set.
"""
from __future__ import annotations

import os
import random
from typing import Any, Dict, List

from app.schemas.lead import LeadFeatures

__all__ = [
    "MOCK_LEADS",
    "TRAINING_DATASET",
    "seed_tenant_model",
    "initialise_seed_data",
]


# ---------------------------------------------------------------------------
# Mock leads (deterministic) — covers all tiers.
# ---------------------------------------------------------------------------

MOCK_LEADS: List[Dict[str, Any]] = [
    {
        "lead_id": "lead-001",
        "tenant_id": "demo-tenant",
        "full_name": "Alice Nguyen",
        "email": "alice@acme-corp.com",
        "company": "Acme Corp",
        "job_title": "VP of Engineering",
        "industry": "saas",
        "source": "referral",
        "page_views": 42,
        "time_on_site_seconds": 1240,
        "has_email": True,
        "has_phone": True,
        "email_opens": 8,
        "email_clicks": 4,
        "form_fills": 3,
        "repeat_visits": 6,
        "company_size": "1000+",
        "company_revenue": 250_000_000.0,
        "country": "US",
        "device_type": "desktop",
    },
    {
        "lead_id": "lead-002",
        "tenant_id": "demo-tenant",
        "full_name": "Bob Tran",
        "email": "bob@startup.io",
        "company": "Startup.io",
        "job_title": "Founder",
        "industry": "fintech",
        "source": "linkedin",
        "page_views": 15,
        "time_on_site_seconds": 460,
        "email_opens": 3,
        "email_clicks": 1,
        "form_fills": 1,
        "company_size": "11-50",
        "company_revenue": 5_000_000.0,
        "country": "VN",
        "device_type": "mobile",
    },
    {
        "lead_id": "lead-003",
        "tenant_id": "demo-tenant",
        "full_name": "Carol Le",
        "email": "carol@enterprise.co",
        "company": "Enterprise Co",
        "job_title": "Director of Sales",
        "industry": "manufacturing",
        "source": "google-ads",
        "page_views": 5,
        "time_on_site_seconds": 90,
        "email_opens": 1,
        "email_clicks": 0,
        "form_fills": 1,
        "company_size": "500+",
        "company_revenue": 80_000_000.0,
        "country": "US",
        "device_type": "desktop",
    },
    {
        "lead_id": "lead-004",
        "tenant_id": "demo-tenant",
        "full_name": "David Pham",
        "email": "david@smallbiz.com",
        "company": "SmallBiz LLC",
        "job_title": "Owner",
        "industry": "retail",
        "source": "facebook",
        "page_views": 2,
        "time_on_site_seconds": 30,
        "email_opens": 0,
        "form_fills": 0,
        "company_size": "1-10",
        "country": "VN",
        "device_type": "mobile",
    },
]


# ---------------------------------------------------------------------------
# Synthetic training dataset — small enough to fit in CI but informative.
# ---------------------------------------------------------------------------


def _random_label(features: List[float]) -> int:
    """Derive a label from a handful of features so XGBoost picks them up."""
    page_views, time_on_site, opens, clicks, form_fills = features[:5]
    score = (
        page_views * 0.04
        + time_on_site * 0.001
        + opens * 0.05
        + clicks * 0.10
        + form_fills * 0.25
    )
    return int(score > 0.45)


def _build_training_dataset(n: int = 200) -> List[Dict[str, Any]]:
    rng = random.Random(42)
    rows: List[Dict[str, Any]] = []
    for i in range(n):
        page_views = rng.randint(0, 60)
        time_on_site = rng.randint(0, 2400)
        opens = rng.randint(0, 12)
        clicks = rng.randint(0, 6)
        form_fills = rng.randint(0, 5)
        repeat_visits = rng.randint(0, 8)
        abandoned = rng.randint(0, 3)
        feats = [page_views, time_on_site, opens, clicks, form_fills]
        label = _random_label(feats)
        rows.append(
            {
                "row_id": f"row-{i:04d}",
                "features": {
                    "page_views": page_views,
                    "time_on_site_seconds": time_on_site,
                    "email_opens": opens,
                    "email_clicks": clicks,
                    "form_fills": form_fills,
                    "repeat_visits": repeat_visits,
                    "abandoned_carts": abandoned,
                },
                "label": label,
            }
        )
    return rows


TRAINING_DATASET: List[Dict[str, Any]] = _build_training_dataset()


# ---------------------------------------------------------------------------
# Wiring: pre-load a default tenant model when seeded.
# ---------------------------------------------------------------------------


def seed_tenant_model(tenant_id: str = "demo-tenant") -> None:
    """Boot the global model and register a per-tenant alias."""
    import app.services.inference as _inference

    if _inference.GLOBAL_MODEL is None:
        _inference.GLOBAL_MODEL = _inference._initialise_default_model()
    if tenant_id not in _inference.TENANT_MODELS and _inference.GLOBAL_MODEL is not None:
        _inference.TENANT_MODELS[tenant_id] = _inference.GLOBAL_MODEL


def initialise_seed_data() -> None:
    """Called once at app startup (when ``LEAD_SCORING_SEED=1``)."""
    if os.getenv("LEAD_SCORING_SEED", "1") not in ("1", "true", "TRUE", "yes"):
        return
    seed_tenant_model("demo-tenant")
