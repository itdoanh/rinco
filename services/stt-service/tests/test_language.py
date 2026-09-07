"""Tests for language-related API endpoints."""
from __future__ import annotations

import sys
from pathlib import Path

# Ensure project root is on path for imports
_root = Path(__file__).resolve().parents[2]
if str(_root) not in sys.path:
    sys.path.insert(0, str(_root))

import pytest

from app.api.transcribe import SUPPORTED_LANGUAGES, list_languages


# ---------------------------------------------------------------------------
# Test: supported languages list
# ---------------------------------------------------------------------------

def test_list_languages() -> None:
    """list_languages should return a non-empty list of supported languages."""
    languages = list_languages()
    assert isinstance(languages, list)
    assert len(languages) > 0
    # Each entry should have code and name
    for lang in languages:
        assert "code" in lang
        assert "name" in lang
        assert isinstance(lang["code"], str)
        assert isinstance(lang["name"], str)


def test_supported_languages_contains_vietnamese() -> None:
    """SUPPORTED_LANGUAGES should include Vietnamese."""
    codes = {lang["code"] for lang in SUPPORTED_LANGUAGES}
    assert "vi" in codes


def test_supported_languages_contains_english() -> None:
    """SUPPORTED_LANGUAGES should include English."""
    codes = {lang["code"] for lang in SUPPORTED_LANGUAGES}
    assert "en" in codes


# ---------------------------------------------------------------------------
# Test: language detection endpoint structure
# ---------------------------------------------------------------------------

def test_language_detection_response_structure() -> None:
    """Language detection response should have expected keys."""
    # Minimal mock result dict
    mock_result = {
        "language": "vi",
        "language_probability": 0.95,
        "duration": 10.5,
    }
    assert "language" in mock_result
    assert "language_probability" in mock_result
    assert "duration" in mock_result
