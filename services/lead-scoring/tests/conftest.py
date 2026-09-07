"""Pytest configuration."""
from __future__ import annotations

import os
import sys

# Make the app package importable when running `pytest tests/` from the
# service root.
ROOT = os.path.dirname(os.path.abspath(__file__))
SERVICE_ROOT = os.path.dirname(ROOT)
if SERVICE_ROOT not in sys.path:
    sys.path.insert(0, SERVICE_ROOT)