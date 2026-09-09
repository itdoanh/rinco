// Extra tests for observability-service loki.go helpers.
package loki

import (
	"testing"
	"time"
)

// =============================================================================
// buildQuery
// =============================================================================

func TestExtraBuildQuery_ServiceOnly(t *testing.T) {
	q := Query{Service: "auth"}
	got := buildQuery(q)
	if got == "" {
		t.Fatal("empty query")
	}
	if !contains(got, `service="auth"`) {
		t.Errorf("missing service label: %s", got)
	}
}

func TestExtraBuildQuery_AllFilters(t *testing.T) {
	q := Query{
		Service:  "auth",
		Level:    "error",
		TenantID: "tenant-123",
		Pattern:  "panic",
	}
	got := buildQuery(q)
	if !contains(got, `service="auth"`) {
		t.Errorf("missing service: %s", got)
	}
	if !contains(got, `level="error"`) {
		t.Errorf("missing level: %s", got)
	}
	if !contains(got, `tenant_id="tenant-123"`) {
		t.Errorf("missing tenant_id: %s", got)
	}
	if !contains(got, `|= "panic"`) {
		t.Errorf("missing pattern: %s", got)
	}
}

func TestExtraBuildQuery_EmptyService(t *testing.T) {
	q := Query{Level: "info"}
	got := buildQuery(q)
	// Should have empty label set or at least not panic
	if got == "" {
		t.Error("query should not be empty")
	}
}

func TestExtraBuildQuery_NoFilters(t *testing.T) {
	q := Query{}
	got := buildQuery(q)
	// Should be "{}"
	if got != "{}" {
		t.Errorf("empty query: got %s", got)
	}
}

func TestExtraBuildQuery_PatternOnly(t *testing.T) {
	q := Query{Pattern: "failed"}
	got := buildQuery(q)
	if !contains(got, `|= "failed"`) {
		t.Errorf("missing pattern: %s", got)
	}
}

func TestExtraBuildQuery_EscapesQuotes(t *testing.T) {
	q := Query{Service: `service"with"quotes`}
	got := buildQuery(q)
	// The escaped quotes should appear as \"
	if !contains(got, `\"`) {
		t.Errorf("missing escaped quotes: %s", got)
	}
}

// =============================================================================
// LogLine
// =============================================================================

func TestExtraLogLine_Fields(t *testing.T) {
	ll := LogLine{
		Time:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Line:    "test log line",
		Labels:  map[string]string{"service": "auth"},
		Tenant:  "tenant-1",
		Service: "auth",
		Level:   "error",
	}
	if ll.Line != "test log line" {
		t.Errorf("Line: %s", ll.Line)
	}
	if ll.Level != "error" {
		t.Errorf("Level: %s", ll.Level)
	}
}

// =============================================================================
// Query struct
// =============================================================================

func TestExtraQuery_DefaultValues(t *testing.T) {
	q := Query{}
	if q.Limit != 0 {
		t.Errorf("Limit: got %d", q.Limit)
	}
	if q.Service != "" {
		t.Errorf("Service: got %s", q.Service)
	}
}

func TestExtraQuery_CustomValues(t *testing.T) {
	q := Query{
		Service:  "auth",
		Level:    "info",
		TenantID: "t1",
		From:     "2026-01-01T00:00:00Z",
		To:       "2026-01-02T00:00:00Z",
		Limit:    500,
		Pattern:  "error",
		LogQL:    "rate({service=\"auth\"}[5m])",
	}
	if q.Limit != 500 {
		t.Errorf("Limit: got %d", q.Limit)
	}
	if q.LogQL == "" {
		t.Error("LogQL should be set")
	}
}

// =============================================================================
// New client
// =============================================================================

func TestExtraNew_ClientCreated(t *testing.T) {
	c := New("http://localhost:3100")
	if c == nil {
		t.Error("New should return non-nil client")
	}
	if c.JC == nil {
		t.Error("JSONClient should be non-nil")
	}
}

func TestExtraNew_EmptyURL(t *testing.T) {
	// Should not panic
	c := New("")
	if c == nil {
		t.Error("New with empty URL should return client")
	}
}

// =============================================================================
// QueryRangeResponse
// =============================================================================

func TestExtraQueryRangeResponse_EmptyResult(t *testing.T) {
	r := QueryRangeResponse{}
	if r.Status != "" {
		t.Errorf("default status: %s", r.Status)
	}
	if len(r.Data.Result) != 0 {
		t.Errorf("default result: %v", r.Data.Result)
	}
}

// =============================================================================
// contains helper (local, mirrors test patterns)
// =============================================================================

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
