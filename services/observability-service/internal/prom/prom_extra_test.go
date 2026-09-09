// Extra tests for observability-service prom.go helpers.
package prom

import (
	"testing"
	"time"
)

// =============================================================================
// valueAt
// =============================================================================

func TestExtraValueAt_Empty(t *testing.T) {
	if v := valueAt(nil); v != 0 {
		t.Errorf("nil: got %f", v)
	}
}

func TestExtraValueAt_SingleElement(t *testing.T) {
	if v := valueAt([]any{1.0}); v != 0 {
		t.Errorf("single: got %f", v)
	}
}

func TestExtraValueAt_Float64(t *testing.T) {
	v := valueAt([]any{123.456, 99.5})
	if v != 99.5 {
		t.Errorf("float64: got %f", v)
	}
}

func TestExtraValueAt_String(t *testing.T) {
	v := valueAt([]any{0, "42.5"})
	if v != 42.5 {
		t.Errorf("string: got %f", v)
	}
}

func TestExtraValueAt_StringInteger(t *testing.T) {
	v := valueAt([]any{0, "100"})
	if v != 100.0 {
		t.Errorf("string int: got %f", v)
	}
}

func TestExtraValueAt_StringInvalid(t *testing.T) {
	v := valueAt([]any{0, "not-a-number"})
	if v != 0 {
		t.Errorf("invalid string: got %f", v)
	}
}

func TestExtraValueAt_OtherType(t *testing.T) {
	v := valueAt([]any{0, 123})
	if v != 0 {
		t.Errorf("other type: got %f", v)
	}
}

// =============================================================================
// TopByLabel
// =============================================================================

func TestExtraTopByLabel_Empty(t *testing.T) {
	r := TopByLabel([]Result{}, 5)
	if len(r) != 0 {
		t.Errorf("empty: got %d", len(r))
	}
}

func TestExtraTopByLabel_Single(t *testing.T) {
	r := TopByLabel([]Result{{Value: []any{0, 1.0}}}, 5)
	if len(r) != 1 || r[0].Value[1] != 1.0 {
		t.Errorf("single: unexpected %v", r)
	}
}

func TestExtraTopByLabel_OrderDescending(t *testing.T) {
	r := TopByLabel([]Result{
		{Metric: map[string]string{"svc": "a"}, Value: []any{0, 1.0}},
		{Metric: map[string]string{"svc": "b"}, Value: []any{0, 5.0}},
		{Metric: map[string]string{"svc": "c"}, Value: []any{0, 3.0}},
	}, 5)
	if r[0].Metric["svc"] != "b" {
		t.Errorf("first should be b: %v", r[0].Metric)
	}
	if r[2].Metric["svc"] != "a" {
		t.Errorf("last should be a: %v", r[2].Metric)
	}
}

func TestExtraTopByLabel_Limit(t *testing.T) {
	r := TopByLabel([]Result{
		{Metric: map[string]string{"svc": "a"}, Value: []any{0, 1.0}},
		{Metric: map[string]string{"svc": "b"}, Value: []any{0, 5.0}},
		{Metric: map[string]string{"svc": "c"}, Value: []any{0, 3.0}},
	}, 2)
	if len(r) != 2 {
		t.Errorf("limit: got %d", len(r))
	}
	if r[0].Metric["svc"] != "b" {
		t.Errorf("first after limit: %v", r[0].Metric)
	}
}

func TestExtraTopByLabel_ZeroLimit(t *testing.T) {
	r := TopByLabel([]Result{
		{Metric: map[string]string{"svc": "a"}, Value: []any{0, 1.0}},
		{Metric: map[string]string{"svc": "b"}, Value: []any{0, 5.0}},
	}, 0)
	if len(r) != 2 {
		t.Errorf("zero limit: got %d", len(r))
	}
}

func TestExtraTopByLabel_NegativeLimit(t *testing.T) {
	r := TopByLabel([]Result{
		{Metric: map[string]string{"svc": "a"}, Value: []any{0, 1.0}},
		{Metric: map[string]string{"svc": "b"}, Value: []any{0, 5.0}},
	}, -1)
	if len(r) != 2 {
		t.Errorf("negative limit: got %d", len(r))
	}
}

// =============================================================================
// FormatStep
// =============================================================================

func TestExtraFormatStep_Empty(t *testing.T) {
	d, err := FormatStep("")
	if err != nil {
		t.Errorf("empty: unexpected error %v", err)
	}
	if d != 30*time.Second {
		t.Errorf("default: got %v", d)
	}
}

func TestExtraFormatStep_Valid(t *testing.T) {
	d, err := FormatStep("1m")
	if err != nil {
		t.Errorf("1m: unexpected error %v", err)
	}
	if d != time.Minute {
		t.Errorf("got %v", d)
	}
}

func TestExtraFormatStep_Seconds(t *testing.T) {
	d, err := FormatStep("15s")
	if err != nil {
		t.Errorf("15s: unexpected error %v", err)
	}
	if d != 15*time.Second {
		t.Errorf("got %v", d)
	}
}

func TestExtraFormatStep_Hours(t *testing.T) {
	d, err := FormatStep("2h")
	if err != nil {
		t.Errorf("2h: unexpected error %v", err)
	}
	if d != 2*time.Hour {
		t.Errorf("got %v", d)
	}
}

func TestExtraFormatStep_Invalid(t *testing.T) {
	_, err := FormatStep("not-a-duration")
	if err == nil {
		t.Error("invalid should error")
	}
}

// =============================================================================
// HealthSummary
// =============================================================================

func TestExtraHealthSummary_Healthy(t *testing.T) {
	hs := HealthSummary{Service: "auth", Status: "healthy", Up: 1.0, ErrorRate: 0.01, P99Latency: 0.05}
	if hs.Status != "healthy" {
		t.Errorf("status: %s", hs.Status)
	}
}

func TestExtraHealthSummary_Degraded(t *testing.T) {
	hs := HealthSummary{Service: "auth", Status: "degraded", Up: 1.0, ErrorRate: 0.1, P99Latency: 2.0}
	if hs.Status != "degraded" {
		t.Errorf("status: %s", hs.Status)
	}
}

func TestExtraHealthSummary_Down(t *testing.T) {
	hs := HealthSummary{Service: "auth", Status: "down", Up: 0.0, ErrorRate: 0.8, P99Latency: 10.0}
	if hs.Status != "down" {
		t.Errorf("status: %s", hs.Status)
	}
}

// =============================================================================
// New client
// =============================================================================

func TestExtraNew_ClientCreated(t *testing.T) {
	c := New("http://localhost:9090")
	if c == nil {
		t.Error("New should return non-nil client")
	}
	if c.JC == nil {
		t.Error("JSONClient should be non-nil")
	}
}
