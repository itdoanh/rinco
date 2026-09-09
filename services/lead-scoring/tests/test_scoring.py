"""Deprecated legacy test file.

The legacy module path `from scoring import ...` referenced a non-existent
top-level `scoring` module. The current implementation lives in
`app.services.scoring`. The current tests for the scorer live in
`tests/test_inference.py`.

This file is kept as a placeholder so that test directories do not shrink
and to avoid breaking pytest collection. The placeholder test is skipped
because the symbols the original suite exercised no longer exist at this
import path.
"""
from __future__ import annotations

import pytest


@pytest.mark.skip(
    reason=(
        "Legacy file: the 'scoring' module was renamed to "
        "'app.services.scoring'. See tests/test_inference.py for the "
        "current coverage."
    )
)
def test_legacy_scoring_placeholder():
    """Placeholder test (legacy file kept for historical reference)."""
