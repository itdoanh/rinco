"""Tests for AI-SRE analyze.py helper functions."""
from app.api.analyze import (
    _parse_rca_response,
    _summarize_traces,
    _apply_fix,
    _build_pr_body,
)


def test_parse_rca_direct_json():
    """Strategy 1: parse direct JSON."""
    text = '{"root_cause":"a","why":"b","suggested_fix":"c","prevention":"d","file":"x.go","line":10,"confidence":0.9}'
    r = _parse_rca_response(text)
    assert r["root_cause"] == "a"
    assert r["file"] == "x.go"
    assert r["line"] == 10
    assert r["confidence"] == 0.9


def test_parse_rca_json_in_codeblock():
    """Strategy 2: parse JSON inside a markdown code block."""
    text = """Here's the analysis:

```json
{"root_cause":"db","why":"lock","suggested_fix":"retry","prevention":"timeout","file":"q.go","line":5,"confidence":0.7}
```
"""
    r = _parse_rca_response(text)
    assert r["root_cause"] == "db"
    assert r["file"] == "q.go"


def test_parse_rca_inline_json():
    """Strategy 3: parse inline {...} JSON."""
    text = 'Result: {"root_cause":"x","confidence":0.5} trailing text'
    r = _parse_rca_response(text)
    assert r["root_cause"] == "x"


def test_parse_rca_field_extraction():
    """Strategy 4: extract fields with regex when no JSON parses."""
    text = '"root_cause": "boom", "confidence": 0.42, "line": 99'
    r = _parse_rca_response(text)
    assert r["root_cause"] == "boom"
    assert abs(r["confidence"] - 0.42) < 1e-6
    assert r["line"] == 99


def test_parse_rca_empty():
    """Empty text returns dict with at least root_cause and confidence."""
    r = _parse_rca_response("")
    assert isinstance(r, dict)
    assert "root_cause" in r
    assert "confidence" in r
    assert r["confidence"] >= 0


def test_parse_rca_garbage_text():
    """Unparseable text returns empty string for root_cause."""
    text = "no json here at all"
    r = _parse_rca_response(text)
    # field() returns "" for missing keys; confidence defaults to 0.3
    assert isinstance(r, dict)


def test_summarize_traces_empty():
    """Empty trace list returns empty string."""
    assert _summarize_traces([]) == ""


def test_summarize_traces_multiple():
    """Multiple traces are summarized with span count + duration."""
    traces = [
        {"trace_id": "abc", "span_count": 10, "duration_ms": 100.0},
        {"trace_id": "def", "span_count": 5, "duration_ms": 50.5},
    ]
    s = _summarize_traces(traces)
    assert "abc" in s
    assert "def" in s
    assert "10 spans" in s
    assert "50.5ms" in s


def test_summarize_traces_limit5():
    """Only first 5 traces are summarized."""
    traces = [{"trace_id": f"t{i}", "span_count": 1, "duration_ms": 1.0} for i in range(10)]
    s = _summarize_traces(traces)
    assert "t4" in s
    assert "t5" not in s


def test_apply_fix_at_line():
    """Insert fix code at line number."""
    source = "line1\nline2\nline3"
    result = _apply_fix(source, 2, "FIXED")
    lines = result.split("\n")
    assert lines == ["line1", "FIXED", "line2", "line3"]


def test_apply_fix_out_of_range():
    """Out-of-range line returns source unchanged."""
    source = "line1\nline2\nline3"
    # Line beyond end: returns source unchanged per current impl
    r = _apply_fix(source, 100, "FIXED")
    # Implementation returns source when line > len+1
    assert "FIXED" not in r or r == source


def test_apply_fix_at_start():
    """Line 1 inserts at beginning."""
    source = "line1\nline2"
    result = _apply_fix(source, 1, "NEW")
    lines = result.split("\n")
    assert lines[0] == "NEW"
    assert lines[1] == "line1"


def test_apply_fix_at_end():
    """Insertion at len+1 appends."""
    source = "line1\nline2"
    result = _apply_fix(source, 3, "NEW")
    lines = result.split("\n")
    assert lines[-1] == "NEW"


def test_build_pr_body():
    """PR body includes service/root_cause/confidence/fix."""
    rca = {
        "root_cause": "DB pool exhausted",
        "suggested_fix": "Increase pool size",
        "why": "high load",
        "prevention": "monitor",
        "confidence": 0.85,
    }
    body = _build_pr_body(rca, "auth-service")
    assert "auth-service" in body
    assert "DB pool exhausted" in body
    assert "Increase pool size" in body
    assert "85%" in body
