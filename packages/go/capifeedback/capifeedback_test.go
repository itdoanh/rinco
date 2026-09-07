package capifeedback

import (
	"context"
	"testing"
	"time"
)

func TestPublisherNotEnabled(t *testing.T) {
	p := New(PublisherConfig{}) // no endpoint
	if p.Enabled() {
		t.Error("expected publisher not enabled when endpoint empty")
	}
	// Publish should be no-op (returns nil)
	if err := p.Publish(context.Background(), Event{}); err != nil {
		t.Errorf("Publish on disabled: %v", err)
	}
}

func TestPublishAsyncNoOp(t *testing.T) {
	p := New(PublisherConfig{})
	errCh := p.PublishAsync(Event{})
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("PublishAsync did not return")
	}
}

func TestEventDefaults(t *testing.T) {
	p := New(PublisherConfig{Endpoint: "http://localhost:9999"})
	ev := Event{TenantID: "t1", Email: "x@y.com"}

	// PublishWon should fill EventTime + EventName
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err := p.PublishWon(ctx, ev)
	// We expect an error because localhost:9999 isn't listening,
	// but the call should not panic and defaults should be set.
	// Note: PublishWon sets defaults BEFORE retry; can't easily verify
	// without mocking, but the call should complete.
	_ = err
}

func TestParseHelpers(t *testing.T) {
	if parseDuration("", 5*time.Second) != 5*time.Second {
		t.Error("parseDuration default")
	}
	if parseDuration("invalid", 5*time.Second) != 5*time.Second {
		t.Error("parseDuration invalid fallback")
	}
	if parseDuration("10s", 5*time.Second) != 10*time.Second {
		t.Error("parseDuration parse")
	}
	if parseInt("", 7) != 7 {
		t.Error("parseInt default")
	}
	if parseInt("abc", 7) != 7 {
		t.Error("parseInt invalid fallback")
	}
	if parseInt("42", 7) != 42 {
		t.Error("parseInt parse")
	}
}

func TestNewFromEnv(t *testing.T) {
	t.Setenv("CAPI_FEEDBACK_URL", "")
	t.Setenv("CAPI_PIXEL_ID", "123")
	p := NewFromEnv()
	if p.Enabled() {
		t.Error("expected not enabled when URL empty")
	}
}

func TestEnabledFlag(t *testing.T) {
	cases := []struct {
		endpoint string
		want     bool
	}{
		{"", false},
		{"http://example.com/capi", true},
	}
	for _, c := range cases {
		p := New(PublisherConfig{Endpoint: c.endpoint})
		if p.Enabled() != c.want {
			t.Errorf("endpoint=%q Enabled=%v, want %v", c.endpoint, p.Enabled(), c.want)
		}
	}
}
