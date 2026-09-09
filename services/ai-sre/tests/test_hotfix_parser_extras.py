"""Extra tests for the hotfix_generator parser helpers.

This file covers the secondary parsing helpers that are exercised by the
public ``_parse_hotfix_response`` function but warrant their own
behavioural assertions:

* :func:`_coerce_unquoted_keys` — quote JS-style bare identifier keys
* :func:`_merge_defaults` — ensure canonical schema is always returned
* :func:`DEFAULT_HOTFIX_KEYS` — the list of canonical keys
"""
from __future__ import annotations

import sys
import os

sys.path.insert(
    0,
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
)

import pytest

try:
    from app.services import hotfix_generator
except Exception as e:
    pytest.skip(f"Cannot import app.services.hotfix_generator: {e}", allow_module_level=True)


# ---------------------------------------------------------------- coerce
class TestCoerceUnquotedKeys:
    def test_quotes_simple_keys(self):
        result = hotfix_generator._coerce_unquoted_keys("{ a: 1, b: 2 }")
        assert '"a": 1' in result
        assert '"b": 2' in result

    def test_keeps_quoted_keys_unchanged(self):
        text = '{ "a": 1, "b": 2 }'
        result = hotfix_generator._coerce_unquoted_keys(text)
        assert result == text

    def test_handles_mixed_quoted_and_unquoted(self):
        result = hotfix_generator._coerce_unquoted_keys(
            '{ "a": 1, b: 2, "c": 3 }'
        )
        assert '"a": 1' in result
        assert '"b": 2' in result
        assert '"c": 3' in result

    def test_does_not_quote_values(self):
        # Make sure we don't accidentally quote string values.
        result = hotfix_generator._coerce_unquoted_keys(
            '{ key: "value with: colon" }'
        )
        assert '"key"' in result
        assert '"value with: colon"' in result

    def test_empty_input(self):
        assert hotfix_generator._coerce_unquoted_keys("") == ""

    def test_nested_object(self):
        text = '{ outer: { inner: 1 } }'
        result = hotfix_generator._coerce_unquoted_keys(text)
        assert '"outer"' in result
        assert '"inner"' in result


# ---------------------------------------------------------------- merge
class TestMergeDefaults:
    def test_fills_missing_keys(self):
        result = hotfix_generator._merge_defaults({"diff": "x"})
        assert result["diff"] == "x"
        assert result["file"] == ""
        assert result["line"] == 0
        assert result["confidence"] == 0.3
        assert result["root_cause"] == ""

    def test_does_not_overwrite_explicit_values(self):
        result = hotfix_generator._merge_defaults(
            {"diff": "x", "file": "f", "line": 5, "confidence": 0.9, "root_cause": "rc"}
        )
        assert result["diff"] == "x"
        assert result["file"] == "f"
        assert result["line"] == 5
        assert result["confidence"] == 0.9
        assert result["root_cause"] == "rc"

    def test_treats_none_as_missing(self):
        result = hotfix_generator._merge_defaults({"line": None})
        assert result["line"] == 0

    def test_preserves_extra_keys(self):
        result = hotfix_generator._merge_defaults({"diff": "x", "extra": "info"})
        assert result["extra"] == "info"

    def test_empty_input_returns_full_defaults(self):
        result = hotfix_generator._merge_defaults({})
        for key in hotfix_generator.DEFAULT_HOTFIX_KEYS:
            assert key in result


class TestDefaultHotfixKeys:
    def test_contains_canonical_keys(self):
        assert "diff" in hotfix_generator.DEFAULT_HOTFIX_KEYS
        assert "file" in hotfix_generator.DEFAULT_HOTFIX_KEYS
        assert "line" in hotfix_generator.DEFAULT_HOTFIX_KEYS
        assert "confidence" in hotfix_generator.DEFAULT_HOTFIX_KEYS
        assert "root_cause" in hotfix_generator.DEFAULT_HOTFIX_KEYS


# ---------------------------------------------------------------- extras
class TestParseHotfixResponseExtras:
    def test_returns_defaults_for_partial_json(self):
        """Partial JSON missing several keys gets defaults merged in."""
        text = '{"diff": "x"}'
        result = hotfix_generator._parse_hotfix_response(text)
        assert result["diff"] == "x"
        assert result["file"] == ""
        assert result["line"] == 0
        assert result["confidence"] == 0.3

    def test_handles_unquoted_keys(self):
        text = '{ diff: "+x", file: "a.go", line: 7, confidence: 0.5, root_cause: "rc" }'
        result = hotfix_generator._parse_hotfix_response(text)
        assert result["diff"] == "+x"
        assert result["file"] == "a.go"
        assert result["line"] == 7

    def test_uses_text_as_root_cause_in_fallback(self):
        """When no JSON is found, the raw text is treated as root_cause."""
        text = "Database connection failed because of bad credentials."
        result = hotfix_generator._parse_hotfix_response(text)
        assert "database" in result["root_cause"].lower()
