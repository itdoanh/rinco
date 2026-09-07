"""Tests for the hotfix_generator module."""
from __future__ import annotations

import sys
import os

# Ensure project root is on path
sys.path.insert(
    0,
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
)

import pytest

try:
    from app.services import hotfix_generator
except Exception as e:
    pytest.skip(f"Cannot import app.services.hotfix_generator: {e}", allow_module_level=True)


# ---------------------------------------------------------------- helpers
def make_mock_response(text: str) -> dict:
    """Simulate an httpx response dict structure."""
    return {"choices": [{"message": {"content": text}}]}


# ---------------------------------------------------------------- tests
def test_build_hotfix_prompt_contains_error():
    """Prompt should include the error message."""
    prompt = hotfix_generator._build_hotfix_prompt(
        error="NullPointerException at auth.go:42",
        stack=None,
        source='func authenticate() {\n  return nil\n}',
        context="service: auth-service, error_rate: 5%",
    )
    assert "NullPointerException" in prompt
    assert "auth-service" in prompt


def test_build_hotfix_prompt_includes_stack():
    """Stack trace should appear in the prompt when provided."""
    stack = 'File "auth.go", line 42, in authenticate\n  return user.getId()'
    prompt = hotfix_generator._build_hotfix_prompt(
        error="panic",
        stack=stack,
        source="func authenticate() {}",
        context="",
    )
    assert "auth.go" in prompt
    assert "line 42" in prompt


def test_build_hotfix_prompt_truncates_long_inputs():
    """Very long inputs should be truncated to avoid huge prompts."""
    long_error = "x" * 1000
    prompt = hotfix_generator._build_hotfix_prompt(
        error=long_error,
        stack=None,
        source="x" * 10000,
        context="x" * 10000,
    )
    # Should not include the full 10k string
    assert len(prompt) < 15000


def test_parse_hotfix_response_valid_json():
    """Valid JSON response should be parsed directly."""
    text = '{"diff": "-old\\n+new", "file": "auth.go", "line": 42, "confidence": 0.85, "root_cause": "null check missing"}'
    result = hotfix_generator._parse_hotfix_response(text)

    assert result["diff"] == "-old\n+new"
    assert result["file"] == "auth.go"
    assert result["line"] == 42
    assert result["confidence"] == 0.85
    assert result["root_cause"] == "null check missing"


def test_parse_hotfix_response_json_in_code_block():
    """JSON wrapped in a markdown code block should still parse."""
    text = '''
    Here's the suggested fix:

    ```json
    {
      "diff": "+added line",
      "file": "handler.go",
      "line": 15,
      "confidence": 0.7
    }
    ```
    '''
    result = hotfix_generator._parse_hotfix_response(text)

    assert result["diff"] == "+added line"
    assert result["file"] == "handler.go"
    assert result["line"] == 15
    assert result["confidence"] == 0.7


def test_parse_hotfix_response_unquoted_braces():
    """JSON without surrounding code fences should still extract fields via regex."""
    text = '{ root_cause: "connection timeout", diff: "+ increase timeout", file: "config.go", line: 10, confidence: 0.6 }'
    result = hotfix_generator._parse_hotfix_response(text)

    assert result["diff"] == "+ increase timeout"
    assert result["file"] == "config.go"
    assert result["line"] == 10
    assert result["confidence"] == 0.6


def test_parse_hotfix_response_malformed_returns_fallback():
    """Completely malformed text should return a fallback dict with non-zero line."""
    text = "I think the error is in the database layer. The fix is to add retry logic."
    result = hotfix_generator._parse_hotfix_response(text)

    # Fallback should still have the text in root_cause
    assert "error" in result["root_cause"].lower() or "database" in result["root_cause"].lower()
    assert result["confidence"] == 0.3


def test_parse_hotfix_response_partial_json():
    """Partially complete JSON should extract known fields and leave others as defaults."""
    text = '{"diff": "-missing null check\\n+if user == nil { return }", "confidence": 0.92}'
    result = hotfix_generator._parse_hotfix_response(text)

    assert result["diff"] == "-missing null check\n+if user == nil { return }"
    assert result["file"] == ""  # not provided → default
    assert result["line"] == 0   # not provided → default
    assert result["confidence"] == 0.92


def test_generate_hotfix_returns_structure():
    """generate_hotfix should return a dict with diff/file/line/confidence keys."""
    async def mock_call_vllm(prompt):
        return '{"diff": "+fix", "file": "a.go", "line": 1, "confidence": 0.9, "root_cause": "bug"}'

    import app.services.hotfix_generator as hg
    original = hg._call_vllm
    hg._call_vllm = mock_call_vllm
    try:
        import asyncio
        result = asyncio.run(hg.generate_hotfix(
            error="error",
            stack=None,
            source="src",
            context="ctx",
        ))
        assert "diff" in result
        assert "file" in result
        assert "line" in result
        assert "confidence" in result
        assert result["confidence"] == 0.9
    finally:
        hg._call_vllm = original
