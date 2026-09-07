"""Tests for search endpoint."""
from __future__ import annotations

import os
import sys

ROOT = os.path.dirname(os.path.abspath(__file__))
if ROOT not in sys.path:
    sys.path.insert(0, os.path.dirname(ROOT))


def test_user_collection_naming():
    from app.services.qdrant_client import user_collection
    assert user_collection("abc-123") == "tenant_abc_123_kb"
    assert user_collection("tenant-xyz") == "tenant_tenant_xyz_kb"