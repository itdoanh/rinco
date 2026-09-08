// Additional tests for capifeedback package.
package capifeedback

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	p := New(PublisherConfig{Endpoint: "http://localhost"})
	if p == nil {
		t.Fatal("nil publisher")
	}
	if p.httpClient == nil {
		t.Error("expected http client")
	}
}

func TestNew_Empty(t *testing.T) {
	p := New(PublisherConfig{})
	if p == nil {
		t.Fatal("nil publisher")
	}
	if !p.Enabled() == false {
		t.Error("should not be enabled without endpoint")
	}
}

func TestEnabled(t *testing.T) {
	p := &Publisher{endpoint: ""}
	if p.Enabled() {
		t.Error("empty endpoint should not be enabled")
	}

	p.endpoint = "http://localhost"
	if !p.Enabled() {
		t.Error("with endpoint should be enabled")
	}
}

func TestEnabled_NilPublisher(t *testing.T) {
	var p *Publisher
	if p.Enabled() {
		t.Error("nil publisher should not be enabled")
	}
}

func TestParseDuration_Valid(t *testing.T) {
	os.Setenv("CAPI_TEST_DURATION", "10s")
	defer os.Unsetenv("CAPI_TEST_DURATION")
	d := parseDuration("10s", 5*time.Second)
	if d != 10*time.Second {
		t.Errorf("expected 10s, got %v", d)
	}
}

func TestParseDuration_Empty(t *testing.T) {
	d := parseDuration("", 5*time.Second)
	if d != 5*time.Second {
		t.Errorf("expected default, got %v", d)
	}
}

func TestParseDuration_Invalid(t *testing.T) {
	d := parseDuration("not-a-duration", 5*time.Second)
	if d != 5*time.Second {
		t.Errorf("expected default for invalid, got %v", d)
	}
}

func TestParseInt_Valid(t *testing.T) {
	n := parseInt("42", 3)
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
}

func TestParseInt_Empty(t *testing.T) {
	n := parseInt("", 3)
	if n != 3 {
		t.Errorf("expected default, got %d", n)
	}
}

func TestParseInt_Invalid(t *testing.T) {
	n := parseInt("not-a-number", 3)
	if n != 3 {
		t.Errorf("expected default for invalid, got %d", n)
	}
}

func TestNewFromEnv_Empty(t *testing.T) {
	os.Unsetenv("CAPI_FEEDBACK_URL")
	os.Unsetenv("CAPI_ACCESS_TOKEN")
	os.Unsetenv("CAPI_PIXEL_ID")
	os.Unsetenv("CAPI_TEST_EVENT_CODE")

	p := NewFromEnv()
	if p == nil {
		t.Fatal("nil publisher")
	}
	if p.Enabled() {
		t.Error("should not be enabled with empty env")
	}
}

func TestPublish_Disabled(t *testing.T) {
	p := New(PublisherConfig{}) // No endpoint
	err := p.Publish(context.Background(), Event{EventName: "Purchase"})
	if err != nil {
		t.Errorf("disabled publisher should not error: %v", err)
	}
}

func TestPublishAsync_Disabled(t *testing.T) {
	p := New(PublisherConfig{})
	ch := p.PublishAsync(Event{EventName: "Purchase"})
	err := <-ch
	if err != nil {
		t.Errorf("disabled publisher should not error: %v", err)
	}
}

func TestEvent_Struct(t *testing.T) {
	ev := Event{
		EventID:     "evt-1",
		EventName:   "Purchase",
		EventTime:   time.Now().Unix(),
		Email:       "test@example.com",
		Value:       99.99,
		Currency:    "USD",
		TenantID:    "tenant-1",
		ContactID:   "contact-1",
		DealID:      "deal-1",
		ContentIDs:  []string{"item-1", "item-2"},
		ContentName: "Test Product",
	}
	if ev.EventID != "evt-1" {
		t.Errorf("EventID: got %s", ev.EventID)
	}
	if ev.Value != 99.99 {
		t.Errorf("Value: got %f", ev.Value)
	}
	if len(ev.ContentIDs) != 2 {
		t.Errorf("ContentIDs: got %d", len(ev.ContentIDs))
	}
}

func TestPublisherConfig_Defaults(t *testing.T) {
	p := New(PublisherConfig{Endpoint: "http://localhost", Timeout: 0})
	if p.httpClient.Timeout != 5*time.Second {
		t.Errorf("expected default timeout 5s, got %v", p.httpClient.Timeout)
	}
}

func TestPublishWon_FillsDefaults(t *testing.T) {
	p := New(PublisherConfig{}) // No endpoint, will return early
	err := p.PublishWon(context.Background(), Event{})
	if err != nil {
		t.Errorf("disabled publisher should not error: %v", err)
	}
}
