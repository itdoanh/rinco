// Tests for observability-service jaeger client.
package jaeger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// stringify
// =============================================================================

// TestStringify_StringValue verifies string passthrough.
func TestStringify_StringValue(t *testing.T) {
	if stringify("hello") != "hello" {
		t.Error("string passthrough failed")
	}
}

// TestStringify_FloatValue verifies float formatting.
func TestStringify_FloatValue(t *testing.T) {
	if got := stringify(1.5); got != "1.5" {
		t.Errorf("float: got %q", got)
	}
}

// TestStringify_IntValue handles integer-valued floats.
func TestStringify_IntValue(t *testing.T) {
	if got := stringify(42.0); got != "42" {
		t.Errorf("int: got %q", got)
	}
}

// TestStringify_BoolValue verifies bool formatting.
func TestStringify_BoolValue(t *testing.T) {
	if got := stringify(true); got != "true" {
		t.Errorf("bool: got %q", got)
	}
}

// TestStringify_NilValue returns empty string.
func TestStringify_NilValue(t *testing.T) {
	if got := stringify(nil); got != "" {
		t.Errorf("nil: got %q", got)
	}
}

// TestStringify_OtherType uses fmt.Sprintf fallback.
func TestStringify_OtherType(t *testing.T) {
	got := stringify([]int{1, 2})
	if got == "" {
		t.Error("slice should produce some string")
	}
}

// =============================================================================
// IsLikelyTraceID
// =============================================================================

// TestIsLikelyTraceID_ValidHex accepts 16-char hex.
func TestIsLikelyTraceID_ValidHex(t *testing.T) {
	if !IsLikelyTraceID("0123456789abcdef") {
		t.Error("valid hex should be accepted")
	}
}

// TestIsLikelyTraceID_ValidW3CAccepts accepts W3C trace context.
func TestIsLikelyTraceID_ValidW3CAccepts(t *testing.T) {
	// W3C: trace-id-span-id-format
	if !IsLikelyTraceID("abc123-def456") {
		t.Error("W3C format should be accepted")
	}
}

// TestIsLikelyTraceID_TooShort rejects 8-char strings.
func TestIsLikelyTraceID_TooShort(t *testing.T) {
	if IsLikelyTraceID("abcd1234") {
		t.Error("8-char hex should be rejected (no dash, len < 16)")
	}
}

// TestIsLikelyTraceID_NonHex rejects non-hex chars.
func TestIsLikelyTraceID_NonHex(t *testing.T) {
	if IsLikelyTraceID("xyz123-def456") {
		t.Error("non-hex should be rejected")
	}
}

// TestIsLikelyTraceID_Empty rejects empty.
func TestIsLikelyTraceID_Empty(t *testing.T) {
	if IsLikelyTraceID("") {
		t.Error("empty should be rejected")
	}
}

// TestIsLikelyTraceID_UppercaseHex accepts uppercase.
func TestIsLikelyTraceID_UppercaseHex(t *testing.T) {
	if !IsLikelyTraceID("ABCDEF0123456789") {
		t.Error("uppercase hex should be accepted")
	}
}

// =============================================================================
// Client.GetTrace
// =============================================================================

// TestGetTrace_ParsesResponse decodes a Jaeger trace response.
func TestGetTrace_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/traces/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{
				"traceID": "trace-1",
				"spans": [{
					"spanID": "span-1",
					"operationName": "authenticate",
					"processID": "p1",
					"startTime": 1700000000000,
					"duration": 5000000,
					"tags": [
						{"key": "http.method", "value": "GET"},
						{"key": "http.status_code", "value": 200.0}
					],
					"logs": []
				}],
				"processes": {
					"p1": {"serviceName": "auth-service"}
				}
			}]
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	trace, err := c.GetTrace(context.Background(), "trace-1")
	if err != nil {
		t.Fatalf("gettrace: %v", err)
	}
	if trace.TraceID != "trace-1" {
		t.Errorf("traceID: got %q", trace.TraceID)
	}
	if len(trace.Spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(trace.Spans))
	}
	if trace.Spans[0].Operation != "authenticate" {
		t.Errorf("operation: got %q", trace.Spans[0].Operation)
	}
	if trace.Spans[0].Service != "auth-service" {
		t.Errorf("service: got %q", trace.Spans[0].Service)
	}
	if trace.Spans[0].StartMs != 1700000000 {
		t.Errorf("startMs: got %d (should be 1700000000000/1000)", trace.Spans[0].StartMs)
	}
	if trace.Spans[0].Tags["http.method"] != "GET" {
		t.Errorf("tag: got %v", trace.Spans[0].Tags)
	}
}

// TestGetTrace_EmptyResponse handles empty data array.
func TestGetTrace_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "")
	trace, err := c.GetTrace(context.Background(), "missing")
	if err != nil {
		t.Fatalf("gettrace: %v", err)
	}
	if trace.TraceID != "missing" {
		t.Errorf("traceID: got %q", trace.TraceID)
	}
	if len(trace.Spans) != 0 {
		t.Errorf("expected empty spans, got %d", len(trace.Spans))
	}
}

// TestGetTrace_HTTPError surfaces server errors.
func TestGetTrace_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := New(srv.URL, "")
	_, err := c.GetTrace(context.Background(), "x")
	if err == nil {
		t.Error("expected error from 500")
	}
}

// =============================================================================
// Client.Search
// =============================================================================

// TestSearch_ParsesResponse decodes a Jaeger search response.
func TestSearch_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{
				"traceID": "trace-1",
				"spans": [{
					"spanID": "span-1",
					"operationName": "op",
					"processID": "p1",
					"startTime": 0,
					"duration": 0,
					"tags": [{"key": "k", "value": "v"}]
				}],
				"processes": {"p1": {"serviceName": "x"}}
			}]
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	traces, err := c.Search(context.Background(), SearchQuery{Service: "x"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(traces) != 1 {
		t.Errorf("expected 1 trace, got %d", len(traces))
	}
}

// TestSearch_QueryParams verifies URL params are encoded.
func TestSearch_QueryParams(t *testing.T) {
	var hitQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, _ = c.Search(context.Background(), SearchQuery{
		Service:   "auth-service",
		Operation: "login",
		Tags:      "error=true",
		Limit:     50,
	})
	if !strings.Contains(hitQuery, "service=auth-service") {
		t.Errorf("expected service param, got %q", hitQuery)
	}
	if !strings.Contains(hitQuery, "operation=login") {
		t.Errorf("expected operation param, got %q", hitQuery)
	}
	if !strings.Contains(hitQuery, "limit=50") {
		t.Errorf("expected limit=50, got %q", hitQuery)
	}
}

// TestSearch_HTTPError surfaces server errors.
func TestSearch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	c := New(srv.URL, "")
	_, err := c.Search(context.Background(), SearchQuery{Service: "x"})
	if err == nil {
		t.Error("expected error from 502")
	}
}

// =============================================================================
// Constructor
// =============================================================================

// TestNew_Defaults checks the constructor.
func TestNew_Defaults(t *testing.T) {
	c := New("http://jaeger:16686", "")
	if c == nil {
		t.Fatal("nil client")
	}
	if c.JC == nil {
		t.Error("JC should be set")
	}
	if c.UseTempo {
		t.Error("UseTempo should be false when only jaeger URL is given")
	}
}

// TestNew_TempoEnabled when both URLs are different.
func TestNew_TempoEnabled(t *testing.T) {
	c := New("http://jaeger:16686", "http://tempo:3200")
	if !c.UseTempo {
		t.Error("UseTempo should be true when URLs differ")
	}
}

// TestNew_SameURLs_NoTempo when both URLs are the same.
func TestNew_SameURLs_NoTempo(t *testing.T) {
	c := New("http://tracing:16686", "http://tracing:16686")
	if c.UseTempo {
		t.Error("UseTempo should be false when URLs match")
	}
}
