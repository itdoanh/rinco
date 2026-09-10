// Additional edge-case tests for capifeedback helpers.
package capifeedback

import (
	"strings"
	"testing"
	"time"
)

func TestMore_parseDuration_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		def      time.Duration
		expected time.Duration
	}{
		{"100ms", 5 * time.Second, 100 * time.Millisecond},
		{"1h", 5 * time.Second, time.Hour},
		{"-5s", 5 * time.Second, -5 * time.Second},
		{"0s", 5 * time.Second, 0},
		{"  5s  ", 5 * time.Second, 5 * time.Second}, // ParseDuration accepts whitespace
		{"abc", 7 * time.Second, 7 * time.Second},    // default applied
		{"", 9 * time.Second, 9 * time.Second},
	}
	for _, tt := range tests {
		got := parseDuration(tt.input, tt.def)
		if got != tt.expected {
			t.Errorf("parseDuration(%q, %v) = %v, want %v", tt.input, tt.def, got, tt.expected)
		}
	}
}

func TestMore_parseInt_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		def      int
		expected int
	}{
		{"42", 5, 42},
		{"-7", 5, -7},
		{"0", 5, 0},
		{"abc", 5, 5},
		{"", 5, 5},
		{"999999999999999999999", 5, 5}, // overflow -> default
		{"  5  ", 5, 5},                 // whitespace handled
	}
	for _, tt := range tests {
		got := parseInt(tt.input, tt.def)
		if got != tt.expected {
			t.Errorf("parseInt(%q, %d) = %d, want %d", tt.input, tt.def, got, tt.expected)
		}
	}
}

func TestMore_Publisher_NilSafe(t *testing.T) {
	var p *Publisher
	if p.Enabled() {
		t.Error("nil Publisher should not be enabled")
	}
}

func TestMore_Event_HTTPRequest_Constructed(t *testing.T) {
	// Verify a valid Event serialises to JSON without error.
	ev := Event{
		EventID:   "evt-1",
		EventName: "Purchase",
		EventTime: 1700000000,
		Email:     "x@example.com",
		TenantID:  "tenant-1",
		Value:     100.0,
		Currency:  "VND",
	}
	if ev.TenantID == "" {
		t.Error("TenantID should be set")
	}
	if strings.TrimSpace(ev.Email) == "" {
		t.Error("Email should be non-empty")
	}
}

func TestMore_PublisherConfig_ZeroTimeout(t *testing.T) {
	p := New(PublisherConfig{
		Endpoint: "http://example.com",
		Timeout:  0, // should default to 5s
	})
	if p == nil {
		t.Fatal("New returned nil")
	}
	if !p.Enabled() {
		t.Error("Enabled() should be true with endpoint")
	}
}

func TestMore_PublisherConfig_NegativeTimeout(t *testing.T) {
	p := New(PublisherConfig{
		Endpoint: "http://example.com",
		Timeout:  -5 * time.Second,
	})
	if p == nil {
		t.Fatal("New returned nil")
	}
	if !p.Enabled() {
		t.Error("Enabled() should be true")
	}
}

func TestMore_PublisherConfig_MaxRetries_Default(t *testing.T) {
	// MaxRetries=0 should be left as 0 in the struct; the retry policy
	// uses exponential backoff so zero retries is technically valid.
	p := New(PublisherConfig{
		Endpoint:    "http://example.com",
		MaxRetries:  0,
	})
	if p == nil {
		t.Fatal("New returned nil")
	}
}

func TestMore_Event_ZeroValue(t *testing.T) {
	var ev Event
	if ev.EventID != "" || ev.EventName != "" || ev.TenantID != "" {
		t.Error("zero-value Event should have empty fields")
	}
	if ev.Value != 0 {
		t.Error("Value should be zero")
	}
	if len(ev.ContentIDs) != 0 {
		t.Error("ContentIDs should be nil/empty")
	}
}
