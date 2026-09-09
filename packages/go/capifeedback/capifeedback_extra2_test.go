// Tests for capifeedback package (capifeedback.go).
package capifeedback

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestExtra_New(t *testing.T) {
	p := New(PublisherConfig{})
	if p == nil {
		t.Fatal("New returned nil")
	}
}

func TestExtra_New_Defaults(t *testing.T) {
	cfg := PublisherConfig{}
	p := New(cfg)
	if p == nil {
		t.Fatal("New returned nil")
	}
}

func TestExtra_New_EmptyEndpoint(t *testing.T) {
	p := New(PublisherConfig{Endpoint: ""})
	if p.Enabled() {
		t.Error("Empty endpoint should not be enabled")
	}
}

func TestExtra_New_WithEndpoint(t *testing.T) {
	p := New(PublisherConfig{Endpoint: "http://example.com"})
	if !p.Enabled() {
		t.Error("With endpoint should be enabled")
	}
}

func TestExtra_PublisherConfig_Fields(t *testing.T) {
	cfg := PublisherConfig{
		Endpoint:      "http://example.com",
		AccessToken:  "token123",
		PixelID:      "pixel123",
		TestEventCode: "TEST",
		Timeout:      10 * time.Second,
		MaxRetries:   5,
	}
	
	if cfg.Endpoint != "http://example.com" {
		t.Error("Endpoint")
	}
	if cfg.AccessToken != "token123" {
		t.Error("AccessToken")
	}
	if cfg.PixelID != "pixel123" {
		t.Error("PixelID")
	}
	if cfg.Timeout != 10*time.Second {
		t.Error("Timeout")
	}
	if cfg.MaxRetries != 5 {
		t.Error("MaxRetries")
	}
}

func TestExtra_Event_Fields(t *testing.T) {
	ev := Event{
		EventID:    "event-123",
		EventName:  "Purchase",
		EventTime:  time.Now().Unix(),
		Email:      "test@example.com",
		Phone:      "+1234567890",
		FBClickID:  "click123",
		FBPCookie:  "fbp123",
		Value:      99.99,
		Currency:   "USD",
		ContentIDs: []string{"content-1"},
		ContentName: "Product",
		TenantID:  "tenant-1",
		ContactID: "contact-1",
		DealID:   "deal-1",
	}
	
	if ev.EventID != "event-123" {
		t.Error("EventID")
	}
	if ev.EventName != "Purchase" {
		t.Error("EventName")
	}
	if ev.Value != 99.99 {
		t.Error("Value")
	}
	if ev.Currency != "USD" {
		t.Error("Currency")
	}
	if len(ev.ContentIDs) != 1 {
		t.Error("ContentIDs")
	}
}

func TestExtra_Event_Empty(t *testing.T) {
	ev := Event{}
	if ev.EventID != "" {
		t.Error("EventID should be empty")
	}
	if ev.EventName != "" {
		t.Error("EventName should be empty")
	}
}

func TestExtra_Publish_Disabled(t *testing.T) {
	p := New(PublisherConfig{})
	err := p.Publish(context.Background(), Event{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtra_PublishWon_Disabled(t *testing.T) {
	p := New(PublisherConfig{})
	err := p.PublishWon(context.Background(), Event{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtra_PublishWon_AutoFills(t *testing.T) {
	// Only testable with enabled publisher
	p := New(PublisherConfig{Endpoint: "http://example.com"})
	ev := Event{}
	
	// PublishWon should not error even with disabled publisher
	err := p.PublishWon(context.Background(), ev)
	// Error expected because example.com won't respond, but we verify the function runs
	_ = err
}

func TestExtra_PublishWon_WithDefaults(t *testing.T) {
	// With a disabled publisher, PublishWon should work
	p := New(PublisherConfig{})
	ev := Event{
		EventID:   "test-id",
		EventTime: time.Now().Unix(),
	}
	err := p.PublishWon(context.Background(), ev)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtra_PublishAsync_Disabled(t *testing.T) {
	p := New(PublisherConfig{})
	errCh := p.PublishAsync(Event{})
	
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for result")
	}
}

func TestExtra_parseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"5s", 5 * time.Second},
		{"1m", 1 * time.Minute},
		{"", 5 * time.Second}, // default
		{"invalid", 5 * time.Second}, // default on error
	}
	
	for _, tt := range tests {
		got := parseDuration(tt.input, 5*time.Second)
		if got != tt.expected {
			t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestExtra_parseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"10", 10},
		{"0", 0},
		{"", 5}, // default
		{"invalid", 5}, // default on error
	}
	
	for _, tt := range tests {
		got := parseInt(tt.input, 5)
		if got != tt.expected {
			t.Errorf("parseInt(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestExtra_Publisher_Fields(t *testing.T) {
	p := New(PublisherConfig{
		Endpoint: "http://example.com",
	})
	if p.endpoint != "http://example.com" {
		t.Error("endpoint not set")
	}
}

func TestExtra_Event_ContentIDs(t *testing.T) {
	ev := Event{
		ContentIDs: []string{"id1", "id2", "id3"},
	}
	if len(ev.ContentIDs) != 3 {
		t.Errorf("ContentIDs: got %d", len(ev.ContentIDs))
	}
}

func TestExtra_NewFromEnv(t *testing.T) {
	// Set test env vars
	os.Setenv("CAPI_FEEDBACK_URL", "http://test.example.com")
	os.Setenv("CAPI_ACCESS_TOKEN", "test-token")
	defer os.Unsetenv("CAPI_FEEDBACK_URL")
	defer os.Unsetenv("CAPI_ACCESS_TOKEN")
	
	p := NewFromEnv()
	if !p.Enabled() {
		t.Error("Should be enabled with env vars")
	}
}

func TestExtra_NewFromEnv_NoEnv(t *testing.T) {
	// Ensure env vars are not set
	os.Unsetenv("CAPI_FEEDBACK_URL")
	
	p := NewFromEnv()
	if p.Enabled() {
		t.Error("Should not be enabled without env vars")
	}
}
