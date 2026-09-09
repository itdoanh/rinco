// Extra tests for observability-service loki client helpers.
package loki

import (
	"strings"
	"testing"
)

func TestBuildQuery_AllFieldsEx(t *testing.T) {
	q := Query{
		Service:  "auth-service",
		Level:    "error",
		TenantID: "tenant-1",
		Pattern:  "exception",
	}
	got := buildQuery(q)
	for _, expected := range []string{"auth-service", "error", "tenant-1", "exception"} {
		if !strings.Contains(got, expected) {
			t.Errorf("missing %s in %s", expected, got)
		}
	}
}

func TestBuildQuery_StartsWithBraceEx(t *testing.T) {
	q := Query{Service: "auth"}
	got := buildQuery(q)
	if !strings.HasPrefix(got, "{") {
		t.Errorf("should start with {: %s", got)
	}
	if !strings.Contains(got, "}") {
		t.Errorf("missing }: %s", got)
	}
}

func TestNewEx(t *testing.T) {
	c := New("http://loki:3100")
	if c == nil {
		t.Fatal("nil client")
	}
	if c.JC == nil {
		t.Error("nil JSONClient")
	}
}

func TestQuery_StructDefaultsEx(t *testing.T) {
	q := Query{}
	if q.Limit != 0 {
		t.Error("default limit should be 0")
	}
	if q.Service != "" {
		t.Error("default service should be empty")
	}
}

func TestLogLine_FieldsEx(t *testing.T) {
	ll := LogLine{
		Line:    "test message",
		Tenant:  "tenant-1",
		Service: "auth",
		Level:   "info",
	}
	if ll.Line != "test message" {
		t.Error("line")
	}
	if ll.Tenant != "tenant-1" {
		t.Error("tenant")
	}
}

func TestQueryRangeResponse_StatusEx(t *testing.T) {
	r := QueryRangeResponse{Status: "success"}
	if r.Status != "success" {
		t.Error("status")
	}
}
