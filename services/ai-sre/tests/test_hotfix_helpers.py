"""Tests for ai-sre hotfix_generator helper functions."""
from __future__ import annotations

from app.services.hotfix_generator import (
    _parse_hotfix_response,
    _build_hotfix_prompt,
)


def test_extra_parse_direct_json():
    """Strategy 1: direct JSON parse."""
    text = '{"diff": "x", "file": "y", "line": 5, "confidence": 0.9}'
    result = _parse_hotfix_response(text)
    assert result["diff"] == "x"
    assert result["file"] == "y"
    assert result["line"] == 5


def test_extra_parse_json_in_code_block():
    """Strategy 2: JSON in ```json``` code block."""
    text = '```json\n{"diff": "x", "file": "y", "line": 5, "confidence": 0.9}\n```'
    result = _parse_hotfix_response(text)
    assert result["diff"] == "x"


def test_extra_parse_json_in_code_block_no_lang():
    """Strategy 2: JSON in ``` ``` block (no language)."""
    text = '```\n{"diff": "x", "file": "y", "line": 5, "confidence": 0.9}\n```'
    result = _parse_hotfix_response(text)
    assert result["diff"] == "x"


def test_extra_parse_brace_extraction():
    """Strategy 3: extract first {...} block."""
    text = 'Some text {"diff": "x", "file": "y", "line": 5, "confidence": 0.9} more text'
    result = _parse_hotfix_response(text)
    assert result["diff"] == "x"


def test_extra_parse_regex_fallback():
    """Strategy 4: regex fallback when no JSON found."""
    text = 'random text "diff": "fixed code", "file": "main.go", "line": 42, "confidence": 0.7'
    result = _parse_hotfix_response(text)
    assert result["diff"] == "fixed code"
    assert result["file"] == "main.go"
    assert result["line"] == 42
    assert abs(result["confidence"] - 0.7) < 0.001


def test_extra_parse_empty_string():
    """Empty string should return defaults."""
    result = _parse_hotfix_response("")
    assert result["diff"] == ""
    assert result["file"] == ""
    assert result["line"] == 0
    assert result["confidence"] == 0.3


def test_extra_parse_partial_json():
    """Partial JSON should fall back to regex or defaults."""
    text = 'Some text "diff": "partial"'
    result = _parse_hotfix_response(text)
    assert "diff" in result


def test_extra_parse_root_cause_extracted():
    """root_cause should be extracted."""
    text = '{"root_cause": "DB connection lost", "diff": "x", "file": "y", "line": 1, "confidence": 0.5}'
    result = _parse_hotfix_response(text)
    assert result.get("root_cause") == "DB connection lost"


def test_extra_parse_invalid_confidence_defaults():
    """Invalid confidence falls back to 0.3."""
    text = '"diff": "x"'
    result = _parse_hotfix_response(text)
    assert result["confidence"] == 0.3


def test_extra_build_prompt_basic():
    """Build prompt should include all sections."""
    prompt = _build_hotfix_prompt(
        error="Database connection failed",
        stack="at line 42",
        source="def hello(): pass",
        context="prod env",
    )
    assert "Database connection failed" in prompt
    assert "at line 42" in prompt
    assert "def hello()" in prompt
    assert "prod env" in prompt
    assert "INCIDENT" in prompt
    assert "SOURCE CODE" in prompt
    assert "TASK" in prompt


def test_extra_build_prompt_no_stack():
    """No stack trace should show '(no stack trace)'."""
    prompt = _build_hotfix_prompt("err", None, "src", "ctx")
    assert "(no stack trace)" in prompt


def test_extra_build_prompt_no_source():
    """No source should show '(source not available)'."""
    prompt = _build_hotfix_prompt("err", "stk", "", "ctx")
    assert "(source not available)" in prompt


def test_extra_build_prompt_no_context():
    """No context should show '(no additional context)'."""
    prompt = _build_hotfix_prompt("err", "stk", "src", "")
    assert "(no additional context)" in prompt


def test_extra_build_prompt_truncates_long_inputs():
    """Long inputs should be truncated."""
    long_error = "x" * 1000
    long_stack = "y" * 3000
    long_source = "z" * 10000
    long_context = "w" * 5000
    prompt = _build_hotfix_prompt(long_error, long_stack, long_source, long_context)
    # Error should be truncated to 500 chars
    assert "x" * 500 in prompt
    assert "x" * 501 not in prompt
    # Stack should be truncated to 1500
    assert "y" * 1500 in prompt


def test_extra_build_prompt_includes_json_template():
    """Prompt should include JSON template for response."""
    prompt = _build_hotfix_prompt("err", None, "src", "ctx")
    assert '"diff"' in prompt
    assert '"file"' in prompt
    assert '"line"' in prompt
    assert '"confidence"' in prompt


def test_extra_parse_unicode():
    """Unicode content should be preserved."""
    text = '{"diff": "Xin chào 你好", "file": "main.go", "line": 1, "confidence": 0.5}'
    result = _parse_hotfix_response(text)
    assert "Xin" in result["diff"]


def test_extra_parse_multiline_json():
    """Multi-line JSON should parse correctly."""
    text = '''{
  "diff": "multi
line diff",
  "file": "main.go",
  "line": 10,
  "confidence": 0.85
}'''
    result = _parse_hotfix_response(text)
    assert result["file"] == "main.go"
    assert result["line"] == 10
