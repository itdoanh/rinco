// Additional tests for observability-service handler helpers.
package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// =============================================================================
// composeStatus
// =============================================================================

func TestExtraComposeStatus_NoAlerts(t *testing.T) {
	cases := []struct {
		base   string
		alerts int
		want   string
	}{
		{"ok", 0, "ok"},
		{"healthy", 0, "healthy"},
		{"", 0, "unknown"},
		{"unknown", 0, "unknown"},
	}
	for _, c := range cases {
		if got := composeStatus(c.base, c.alerts); got != c.want {
			t.Errorf("composeStatus(%q, %d) = %q, want %q", c.base, c.alerts, got, c.want)
		}
	}
}

func TestExtraComposeStatus_FewAlertsDegraded(t *testing.T) {
	for n := 1; n <= 4; n++ {
		if got := composeStatus("ok", n); got != "degraded" {
			t.Errorf("alerts=%d: got %q, want degraded", n, got)
		}
	}
}

func TestExtraComposeStatus_FivePlusAlertsCritical(t *testing.T) {
	for _, n := range []int{5, 6, 10, 100, 1000} {
		if got := composeStatus("ok", n); got != "critical" {
			t.Errorf("alerts=%d: got %q, want critical", n, got)
		}
	}
}

func TestExtraComposeStatus_AlertsOverrideBase(t *testing.T) {
	// Even if base is "healthy", alerts change the status.
	if got := composeStatus("healthy", 1); got != "degraded" {
		t.Errorf("expected degraded override, got %q", got)
	}
	if got := composeStatus("healthy", 5); got != "critical" {
		t.Errorf("expected critical override, got %q", got)
	}
}

// =============================================================================
// parseWindow
// =============================================================================

func TestExtraParseWindow_BothEmpty(t *testing.T) {
	from, to := parseWindow("", "")
	now := time.Now()
	// Both should be within a few seconds of now (from = -1h, to = now).
	if to.After(now.Add(5*time.Second)) || to.Before(now.Add(-5*time.Second)) {
		t.Errorf("to drift: %v vs %v", to, now)
	}
	if from.After(to) {
		t.Errorf("from (%v) after to (%v)", from, to)
	}
}

func TestExtraParseWindow_OnlyFrom(t *testing.T) {
	fromStr := "2026-01-01T10:00:00Z"
	from, to := parseWindow(fromStr, "")
	if from.Format(time.RFC3339) != fromStr {
		t.Errorf("from: got %v", from)
	}
	now := time.Now()
	if to.Before(now.Add(-5*time.Second)) {
		t.Errorf("to should default to now, got %v", to)
	}
}

func TestExtraParseWindow_OnlyTo(t *testing.T) {
	toStr := "2026-01-01T10:00:00Z"
	from, to := parseWindow("", toStr)
	if to.Format(time.RFC3339) != toStr {
		t.Errorf("to: got %v", to)
	}
	now := time.Now()
	if from.After(now) {
		t.Errorf("from should default to now-1h, got %v", from)
	}
}

func TestExtraParseWindow_BothSet(t *testing.T) {
	fromStr := "2026-01-01T10:00:00Z"
	toStr := "2026-01-02T10:00:00Z"
	from, to := parseWindow(fromStr, toStr)
	if from.Format(time.RFC3339) != fromStr {
		t.Errorf("from: got %v", from)
	}
	if to.Format(time.RFC3339) != toStr {
		t.Errorf("to: got %v", to)
	}
}

func TestExtraParseWindow_InvalidFromDefaultsToMinus1h(t *testing.T) {
	from, _ := parseWindow("garbage", "")
	now := time.Now()
	expectedFrom := now.Add(-1 * time.Hour)
	// Allow 1-second tolerance.
	if from.Before(expectedFrom.Add(-1*time.Second)) || from.After(expectedFrom.Add(1*time.Second)) {
		t.Errorf("invalid from: got %v, want near %v", from, expectedFrom)
	}
}

func TestExtraParseWindow_InvalidToDefaultsToNow(t *testing.T) {
	_, to := parseWindow("", "garbage")
	now := time.Now()
	if to.Before(now.Add(-1*time.Second)) || to.After(now.Add(1*time.Second)) {
		t.Errorf("invalid to: got %v, want near now", to)
	}
}

func TestExtraParseWindow_OneHourSpan(t *testing.T) {
	from, to := parseWindow("", "")
	span := to.Sub(from)
	if span < 59*time.Minute || span > 61*time.Minute {
		t.Errorf("expected ~1h span, got %v", span)
	}
}

// =============================================================================
// atoiDefault
// =============================================================================

func TestExtraAtoiDefault_EmptyReturnsDefault(t *testing.T) {
	if got := atoiDefault("", 42); got != 42 {
		t.Errorf("empty: got %d, want 42", got)
	}
}

func TestExtraAtoiDefault_ValidNumber(t *testing.T) {
	if got := atoiDefault("123", 42); got != 123 {
		t.Errorf("got %d, want 123", got)
	}
}

func TestExtraAtoiDefault_InvalidReturnsDefault(t *testing.T) {
	cases := []string{"abc", "12.5", "12x", "  ", "1 2", "0x1A"}
	for _, c := range cases {
		if got := atoiDefault(c, 99); got != 99 {
			t.Errorf("atoiDefault(%q) = %d, want 99", c, got)
		}
	}
}

func TestExtraAtoiDefault_Negative(t *testing.T) {
	if got := atoiDefault("-5", 42); got != -5 {
		t.Errorf("negative: got %d, want -5", got)
	}
}

func TestExtraAtoiDefault_Zero(t *testing.T) {
	if got := atoiDefault("0", 42); got != 0 {
		t.Errorf("zero: got %d, want 0", got)
	}
}

func TestExtraAtoiDefault_DefaultZero(t *testing.T) {
	if got := atoiDefault("abc", 0); got != 0 {
		t.Errorf("default zero: got %d, want 0", got)
	}
}

// =============================================================================
// pgErr
// =============================================================================

func TestExtraPgErr_NilError(t *testing.T) {
	if pgErr(nil) {
		t.Error("nil error should not be pgErr")
	}
}

func TestExtraPgErr_NoRows(t *testing.T) {
	if !pgErr(pgx.ErrNoRows) {
		t.Error("pgx.ErrNoRows should be detected")
	}
}

func TestExtraPgErr_OtherError(t *testing.T) {
	if pgErr(errTestSimple("some other error")) {
		t.Error("other errors should not be detected as pgx.ErrNoRows")
	}
}

// =============================================================================
// alertView / auditView JSON shape
// =============================================================================

func TestExtraAlertView_JSONShape(t *testing.T) {
	a := alertView{
		ID:          "id-1",
		Fingerprint: "fp-1",
		Status:      "firing",
		Severity:    "critical",
		Labels:      map[string]string{"alertname": "HighCPU"},
		Annotations: map[string]string{"summary": "CPU > 90%"},
		Title:       "High CPU",
		FiredAt:     time.Now(),
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"id":"id-1"`, `"fingerprint":"fp-1"`, `"status":"firing"`, `"severity":"critical"`} {
		if !strings.Contains(string(b), key) {
			t.Errorf("missing %s in JSON: %s", key, string(b))
		}
	}
}

func TestExtraAuditView_JSONShape(t *testing.T) {
	a := auditView{
		ID:           "audit-1",
		TenantID:     "tenant-1",
		Action:       "create",
		ResourceType: strPtr("contact"),
		ResourceID:   strPtr("c-1"),
		TS:           time.Now(),
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"id":"audit-1"`, `"tenant_id":"tenant-1"`, `"action":"create"`, `"resource_type":"contact"`} {
		if !strings.Contains(string(b), key) {
			t.Errorf("missing %s in JSON: %s", key, string(b))
		}
	}
}

func strPtr(s string) *string { return &s }

func errTestSimple(s string) error {
	return &basicErr{msg: s}
}

type basicErr struct{ msg string }

func (e *basicErr) Error() string { return e.msg }
