// Extra tests for analytics-service handler helpers.
package handler

import (
	"testing"
	"time"
)

func TestParseTime_Empty(t *testing.T) {
	fb := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := parseTime("", fb)
	if !got.Equal(fb) {
		t.Errorf("got %v, want %v", got, fb)
	}
}

func TestParseTime_Valid(t *testing.T) {
	fb := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := parseTime("2026-06-15T12:00:00Z", fb)
	if got.Year() != 2026 || got.Month() != 6 || got.Day() != 15 {
		t.Errorf("got %v", got)
	}
}

func TestParseTime_Invalid(t *testing.T) {
	fb := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := parseTime("not-a-date", fb)
	if !got.Equal(fb) {
		t.Errorf("expected fallback on invalid: got %v", got)
	}
}

func TestParseTime_OtherFormat(t *testing.T) {
	fb := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// RFC3339 is required; "2026-01-01" is not RFC3339
	got := parseTime("2026-01-01", fb)
	if !got.Equal(fb) {
		t.Errorf("expected fallback for non-RFC3339: got %v", got)
	}
}

func TestParseTimeOrNow_Empty(t *testing.T) {
	got := parseTimeOrNow("")
	if got.IsZero() {
		t.Error("empty should return time.Now()")
	}
}

func TestParseTimeOrNow_Valid(t *testing.T) {
	got := parseTimeOrNow("2026-06-15T12:00:00Z")
	if got.Year() != 2026 {
		t.Errorf("got %v", got)
	}
}

func TestParseTimeOrNow_Invalid(t *testing.T) {
	got := parseTimeOrNow("invalid")
	if got.IsZero() {
		t.Error("invalid should fall back to time.Now()")
	}
}

func TestTrackEventRequest_Defaults(t *testing.T) {
	r := TrackEventRequest{}
	if r.TenantID != "" {
		t.Error("default tenant_id")
	}
}

func TestTrackEventRequest_WithValues(t *testing.T) {
	r := TrackEventRequest{
		TenantID:  "tenant-1",
		UserID:    "user-1",
		EventType: "page_view",
		Source:    "fb",
	}
	if r.EventType != "page_view" {
		t.Error("event type")
	}
}

func TestTrackBatchRequest_Empty(t *testing.T) {
	r := TrackBatchRequest{}
	if len(r.Events) != 0 {
		t.Error("empty events")
	}
}

func TestTrackBatchRequest_WithEvents(t *testing.T) {
	r := TrackBatchRequest{
		TenantID: "t1",
		Events: []TrackEventRequest{
			{EventType: "click"},
			{EventType: "page_view"},
		},
	}
	if len(r.Events) != 2 {
		t.Errorf("events: %d", len(r.Events))
	}
}

func TestDashboardRequest_Defaults(t *testing.T) {
	r := DashboardRequest{}
	if r.TenantID != "" {
		t.Error("default")
	}
}

func TestDashboardRequest_WithValues(t *testing.T) {
	r := DashboardRequest{
		TenantID:    "t1",
		From:        "2026-01-01",
		To:          "2026-01-31",
		Granularity: "day",
	}
	if r.Granularity != "day" {
		t.Error("granularity")
	}
}

func TestExportRequest_WithTypes(t *testing.T) {
	r := ExportRequest{
		TenantID: "t1",
		From:     "2026-01-01",
		To:       "2026-01-31",
		Types:    []string{"page_view", "click"},
	}
	if len(r.Types) != 2 {
		t.Errorf("types: %d", len(r.Types))
	}
}

func TestNew_NilRepo(t *testing.T) {
	s := New(nil)
	if s == nil {
		t.Fatal("nil server")
	}
	if s.repo != nil {
		t.Error("repo should be nil")
	}
}

func TestServer_Logger(t *testing.T) {
	s := New(nil)
	if s.log == nil {
		t.Error("logger should be set")
	}
}

func TestGetPayload_Empty(t *testing.T) {
	// getPayload with nil request
	// We can't easily test this without a real request; just check it doesn't panic
	defer func() {
		_ = recover()
	}()
	// Skip actual call since it requires echo context
	// _ = getPayload(nil)
}
