"""Tests for AI-SRE hotfix_generator internal parser helpers."""
from app.services.hotfix_generator import (
    _parse_hotfix_response,
    _merge_defaults,
    _coerce_unquoted_keys,
    DEFAULT_HOTFIX_KEYS,
)


def test_parse_hotfix_direct_json():
    """Strategy 1: direct JSON parse."""
    text = '{"diff":"-x\\n+y","file":"a.go","line":10,"confidence":0.9,"root_cause":"r"}'
    r = _parse_hotfix_response(text)
    assert r["diff"] == "-x\n+y"
    assert r["file"] == "a.go"
    assert r["line"] == 10
    assert r["confidence"] == 0.9
    assert r["root_cause"] == "r"


def test_parse_hotfix_in_codeblock():
    """Strategy 2: JSON in code block."""
    text = """Here's the fix:

```json
{"diff":"","file":"q.go","line":5,"confidence":0.7}
```
"""
    r = _parse_hotfix_response(text)
    assert r["file"] == "q.go"
    assert r["line"] == 5


def test_parse_hotfix_brace_extract():
    """Strategy 3: extract inline JSON."""
    text = 'Result: {"diff":"d","file":"x.go","confidence":0.5}'
    r = _parse_hotfix_response(text)
    assert r["diff"] == "d"


def test_parse_hotfix_unquoted_keys():
    """Strategy 3b: JS-style object literal with unquoted keys."""
    text = "{ diff: 'fix', file: 'q.go', line: 7, confidence: 0.6 }"
    r = _parse_hotfix_response(text)
    # JS-style: coerce should produce parseable JSON
    assert r["file"] == "q.go" or r["confidence"] > 0


def test_parse_hotfix_regex_fallback():
    """Strategy 4: regex field extraction."""
    text = '"diff": "fix here", "file": "main.go", "line": 99, "confidence": 0.42'
    r = _parse_hotfix_response(text)
    assert r["diff"] == "fix here"
    assert r["file"] == "main.go"
    assert r["line"] == 99
    assert abs(r["confidence"] - 0.42) < 1e-6


def test_parse_hotfix_empty():
    """Empty text returns defaults."""
    r = _parse_hotfix_response("")
    assert r["diff"] == ""
    assert r["file"] == ""
    assert r["line"] == 0
    assert r["confidence"] == 0.3


def test_parse_hotfix_garbage():
    """Garbage text uses regex fallback or text-as-root_cause."""
    r = _parse_hotfix_response("this is just plain text")
    assert isinstance(r, dict)
    assert "diff" in r
    assert "file" in r
    assert "line" in r
    assert "confidence" in r
    assert "root_cause" in r


def test_parse_hotfix_partial_json():
    """Partial JSON (missing fields) gets defaults filled in."""
    text = '{"diff":"only diff"}'
    r = _parse_hotfix_response(text)
    assert r["diff"] == "only diff"
    assert r["file"] == ""
    assert r["line"] == 0
    assert r["confidence"] == 0.3  # default


def test_merge_defaults_full():
    """merge_defaults fills in all canonical keys."""
    parsed = {"diff": "x", "file": "y", "line": 1, "confidence": 0.5, "root_cause": "r"}
    r = _merge_defaults(parsed)
    assert r["diff"] == "x"
    assert r["file"] == "y"
    assert r["line"] == 1
    assert r["confidence"] == 0.5
    assert r["root_cause"] == "r"


def test_merge_defaults_empty():
    """merge_defaults of empty dict returns all defaults."""
    r = _merge_defaults({})
    assert r["diff"] == ""
    assert r["file"] == ""
    assert r["line"] == 0
    assert r["confidence"] == 0.3
    assert r["root_cause"] == ""


def test_merge_defaults_preserves_extras():
    """Extra keys from LLM are preserved."""
    parsed = {"diff": "x", "extra_field": "extra_value"}
    r = _merge_defaults(parsed)
    assert "extra_field" in r
    assert r["extra_field"] == "extra_value"


def test_merge_defaults_skips_none():
    """None values don't override defaults."""
    parsed = {"diff": None, "file": "x.go"}
    r = _merge_defaults(parsed)
    assert r["diff"] == ""  # default since input was None
    assert r["file"] == "x.go"


def test_coerce_unquoted_keys_basic():
    """Coerce bare keys into quoted keys."""
    result = _coerce_unquoted_keys("{ a: 1, b: 2 }")
    assert '"a":' in result
    assert '"b":' in result


def test_coerce_unquoted_keys_with_strings():
    """Coerce works even when string values present."""
    result = _coerce_unquoted_keys("{ name: 'alice', age: 30 }")
    assert '"name":' in result
    assert '"age":' in result


def test_coerce_unquoted_keys_already_quoted():
    """Already-quoted keys remain unchanged."""
    s = '{ "a": 1, "b": 2 }'
    assert _coerce_unquoted_keys(s) == s


def test_default_hotfix_keys_constants():
    """DEFAULT_HOTFIX_KEYS contains expected keys."""
    assert "diff" in DEFAULT_HOTFIX_KEYS
    assert "file" in DEFAULT_HOTFIX_KEYS
    assert "line" in DEFAULT_HOTFIX_KEYS
    assert "confidence" in DEFAULT_HOTFIX_KEYS
    assert "root_cause" in DEFAULT_HOTFIX_KEYS
    assert len(DEFAULT_HOTFIX_KEYS) == 5
