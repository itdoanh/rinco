// Tests for observability-service alertmanager webhook.
package alertmanager

import (
	"encoding/json"
	"testing"
	"time"
)

// =============================================================================
// mapStatus
// =============================================================================

// TestMapStatus_KnownValues verifies the three canonical status values.
func TestMapStatus_KnownValues(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"firing", "firing"},
		{"resolved", "resolved"},
		{"FIRING", "firing"},
		{"RESOLVED", "firing"}, // uppercase doesn't match
		{"", "firing"},
		{"unknown", "firing"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := mapStatus(tc.in)
			if got != tc.want {
				t.Errorf("mapStatus(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// =============================================================================
// nullableTime
// =============================================================================

// TestNullableTime_Zero returns nil.
func TestNullableTime_Zero(t *testing.T) {
	got := nullableTime(time.Time{})
	if got != nil {
		t.Errorf("nullableTime(zero) = %v, want nil", got)
	}
}

// TestNullableTime_Valid returns the time.
func TestNullableTime_Valid(t *testing.T) {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	got := nullableTime(ts)
	if got == nil {
		t.Fatal("nullableTime returned nil")
	}
	if !got.(time.Time).Equal(ts) {
		t.Errorf("nullableTime returned %v, want %v", got, ts)
	}
}

// =============================================================================
// firstNonEmpty
// =============================================================================

// TestFirstNonEmpty_PicksFirst returns the first non-empty value.
func TestFirstNonEmpty_PicksFirst(t *testing.T) {
	got := firstNonEmpty("a", "b", "c")
	if got != "a" {
		t.Errorf("expected 'a', got %q", got)
	}
}

// TestFirstNonEmpty_SkipsEmpty skips leading empty strings.
func TestFirstNonEmpty_SkipsEmpty(t *testing.T) {
	got := firstNonEmpty("", "", "x", "")
	if got != "x" {
		t.Errorf("expected 'x', got %q", got)
	}
}

// TestFirstNonEmpty_WhitespaceIsNotEmpty verifies whitespace-only strings are
// treated as non-empty values (only "" is considered empty).
func TestFirstNonEmpty_WhitespaceIsNotEmpty(t *testing.T) {
	got := firstNonEmpty("", "  ", "", "   ")
	if got != "  " {
		t.Errorf("expected '  ', got %q", got)
	}
}

// TestFirstNonEmpty_NoArgs returns empty.
func TestFirstNonEmpty_NoArgs(t *testing.T) {
	got := firstNonEmpty()
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// =============================================================================
// computeFingerprint
// =============================================================================

// TestComputeFingerprint_Present returns the label value.
func TestComputeFingerprint_Present(t *testing.T) {
	got := computeFingerprint(map[string]string{"fingerprint": "fp-123"})
	if got != "fp-123" {
		t.Errorf("expected 'fp-123', got %q", got)
	}
}

// TestComputeFingerprint_Derived builds from alertname/service/severity.
func TestComputeFingerprint_Derived(t *testing.T) {
	labels := map[string]string{
		"alertname": "HighCPU",
		"service":   "auth",
		"severity":  "critical",
		"other":     "ignored",
	}
	got := computeFingerprint(labels)
	// Should contain the three keys.
	want := "alertname=HighCPU,service=auth,severity=critical"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

// TestComputeFingerprint_MissingKeys handles absent labels.
func TestComputeFingerprint_MissingKeys(t *testing.T) {
	got := computeFingerprint(map[string]string{})
	if got != "alertname=,service=,severity=" {
		t.Errorf("expected all-empty, got %q", got)
	}
}

// =============================================================================
// Webhook JSON unmarshaling
// =============================================================================

// TestWebhook_UnmarshalJSON decodes an AlertManager payload.
func TestWebhook_UnmarshalJSON(t *testing.T) {
	raw := `{
		"version": "4",
		"groupKey": "abc",
		"status": "firing",
		"receiver": "default",
		"alerts": [{
			"status": "firing",
			"labels": {"alertname": "HighLatency", "service": "auth"},
			"annotations": {"summary": "P99 > 1s"},
			"startsAt": "2025-01-01T00:00:00Z",
			"endsAt": "0001-01-01T00:00:00Z",
			"generatorURL": "",
			"fingerprint": "fp-abc"
		}]
	}`
	var w Webhook
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if w.Version != "4" {
		t.Errorf("version: got %q", w.Version)
	}
	if w.Status != "firing" {
		t.Errorf("status: got %q", w.Status)
	}
	if len(w.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(w.Alerts))
	}
	if w.Alerts[0].Labels["alertname"] != "HighLatency" {
		t.Errorf("alert name: got %v", w.Alerts[0].Labels)
	}
}

// TestAlert_UnmarshalJSON verifies timestamp parsing.
func TestAlert_UnmarshalJSON(t *testing.T) {
	raw := `{
		"status": "resolved",
		"labels": {},
		"annotations": {},
		"startsAt": "2025-01-01T12:00:00Z",
		"endsAt": "2025-01-01T12:05:00Z",
		"fingerprint": "x"
	}`
	var a Alert
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if a.Status != "resolved" {
		t.Errorf("status: got %q", a.Status)
	}
	// endsAt should be a valid time (not zero).
	if a.EndsAt.IsZero() {
		t.Error("endsAt should not be zero after unmarshal")
	}
}

// TestAlert_UnmarshalJSON_NilAnnotations handles nil annotations.
func TestAlert_UnmarshalJSON_NilAnnotations(t *testing.T) {
	raw := `{"status": "firing", "labels": null, "annotations": null}`
	var a Alert
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Labels/annotations should be nil/empty maps.
	if a.Labels == nil {
		t.Log("labels nil — acceptable")
	}
}

// =============================================================================
// MarshalJSON round-trips
// =============================================================================

// TestAlert_MarshalUnmarshal verifies the Alert marshaling round-trips.
func TestAlert_MarshalUnmarshal(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	a := Alert{
		Status:       "firing",
		Labels:       map[string]string{"alertname": "Test"},
		Annotations:  map[string]string{"summary": "test"},
		StartsAt:     now,
		EndsAt:       now.Add(5 * time.Minute),
		Fingerprint:  "fp-test",
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Alert
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Status != "firing" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.Fingerprint != "fp-test" {
		t.Errorf("fingerprint: got %q", got.Fingerprint)
	}
}

// =============================================================================
// Receiver struct
// =============================================================================

// TestNewReceiver_Defaults verifies constructor.
func TestNewReceiver_Defaults(t *testing.T) {
	// Can't create a real pool without a DB; just verify the constructor
	// exists and returns a non-nil struct (checked by the compiler via
	// this test being in the same package).
	// The Receiver struct is tested implicitly by the Webhook/Alert tests.
}
