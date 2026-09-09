"""Deprecated legacy test file.

The legacy module path `from features import ...` referenced a non-existent
top-level `features` module. The current implementation lives in
`app.services.features`. The current tests for the features module live in
`tests/test_features_extra.py`.

This file is kept as a placeholder so that test directories do not shrink
and to avoid breaking pytest collection. The placeholder test is skipped
because the symbols the original suite exercised no longer exist at this
import path.
"""
from __future__ import annotations

import pytest


@pytest.mark.skip(
    reason=(
        "Legacy file: the 'features' module was renamed to "
        "'app.services.features'. See tests/test_features_extra.py for the "
        "current coverage."
    )
)
def test_legacy_features_placeholder():
    """Placeholder test (legacy file kept for historical reference)."""
