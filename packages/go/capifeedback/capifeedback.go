// Package capifeedback publishes CRM conversion events back to Facebook CAPI.
//
// Per master design (Master Doc section "Feedback Loop Cho Facebook CAPI"):
// When a Lead advances through the CRM funnel and reaches a "won" /
// "converted" / "purchase" milestone, CRM Service should publish a
// server-side Purchase / High_Value_Lead event to Meta so the ad
// optimizer can re-train on real conversions instead of guessing.
//
// This package is intentionally lightweight: it does NOT require NATS
// or the landing-service to be reachable. It uses an outbound HTTP
// POST to whatever gateway the operator has configured (typically the
// landing-service's /internal/capi/conversion endpoint). If the
// gateway is down, the publisher retries with exponential backoff
// but never blocks the CRM write path.
package capifeedback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Publisher posts conversion events to a downstream CAPI gateway.
type Publisher struct {
	endpoint   string
	httpClient *http.Client
	mu         sync.Mutex
	retry      backoff.BackOff
}

// PublisherConfig holds publisher settings.
type PublisherConfig struct {
	// Endpoint is the absolute URL of the CAPI gateway (e.g.
	// "http://landing-service:8080/internal/capi/conversion").
	// If empty, the publisher becomes a no-op.
	Endpoint string

	// AccessToken is the per-tenant access token. CRM typically
	// receives this from request headers and forwards.
	AccessToken string

	// PixelID for downstream Meta event
	PixelID string

	// TestEventCode enables Meta test-mode (optional)
	TestEventCode string

	// Timeout is the HTTP request timeout. Default: 5s.
	Timeout time.Duration

	// MaxRetries caps retry attempts. Default: 3.
	MaxRetries int
}

// Event represents a conversion feedback to publish.
type Event struct {
	// EventID is the original landing-page event_id (UUIDv7).
	// Required for Meta dedup with the original Lead event.
	EventID string

	// EventName: capi.EventPurchase / capi.EventLead / custom
	EventName string

	// EventTime (Unix seconds)
	EventTime int64

	// Email (plain; will be SHA-256 hashed by gateway)
	Email string

	// Phone (E.164 or local; normalized by gateway)
	Phone string

	// FBClickID and FBP cookie from the original Lead capture
	FBClickID string
	FBPCookie string

	// Value (monetary) and Currency
	Value    float64
	Currency string

	// ContentIDs / ContentName for Purchase events
	ContentIDs  []string
	ContentName string

	// TenantID for multi-tenant routing
	TenantID string

	// ContactID and DealID for audit trail
	ContactID string
	DealID    string
}

// New builds a Publisher from config.
func New(cfg PublisherConfig) *Publisher {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 200 * time.Millisecond
	bo.MaxInterval = 5 * time.Second
	bo.MaxElapsedTime = 30 * time.Second

	return &Publisher{
		endpoint: cfg.Endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		retry: bo,
	}
}

// NewFromEnv builds a Publisher using CAPI_FEEDBACK_URL env var.
func NewFromEnv() *Publisher {
	return New(PublisherConfig{
		Endpoint:     os.Getenv("CAPI_FEEDBACK_URL"),
		AccessToken:  os.Getenv("CAPI_ACCESS_TOKEN"),
		PixelID:      os.Getenv("CAPI_PIXEL_ID"),
		TestEventCode: os.Getenv("CAPI_TEST_EVENT_CODE"),
		Timeout:      parseDuration(os.Getenv("CAPI_FEEDBACK_TIMEOUT"), 5*time.Second),
		MaxRetries:   parseInt(os.Getenv("CAPI_FEEDBACK_MAX_RETRIES"), 3),
	})
}

// Enabled reports whether the publisher is wired to a real endpoint.
func (p *Publisher) Enabled() bool {
	return p != nil && p.endpoint != ""
}

// Publish posts the event to the downstream gateway with retry.
// Errors are returned but the caller may choose to swallow them —
// the CRM write path should not block on a Meta outage.
func (p *Publisher) Publish(ctx context.Context, ev Event) error {
	if !p.Enabled() {
		return nil
	}
	body, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("capi_feedback: marshal: %w", err)
	}

	op := func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", ev.TenantID)
		req.Header.Set("X-CAPI-Source", "rinco-crm")
		resp, err := p.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		return fmt.Errorf("capi_feedback: gateway returned %d", resp.StatusCode)
	}

	bo := backoff.WithContext(p.retry, ctx)
	return backoff.Retry(op, bo)
}

// PublishWon publishes a "deal won" conversion event with sensible defaults.
// Convenience helper that fills EventTime if missing.
func (p *Publisher) PublishWon(ctx context.Context, ev Event) error {
	if ev.EventTime == 0 {
		ev.EventTime = time.Now().Unix()
	}
	if ev.EventName == "" {
		ev.EventName = "Purchase"
	}
	return p.Publish(ctx, ev)
}

// PublishAsync publishes without blocking the caller. The returned
// channel emits any error encountered during retry.
func (p *Publisher) PublishAsync(ev Event) <-chan error {
	out := make(chan error, 1)
	if !p.Enabled() {
		out <- nil
		close(out)
		return out
	}
	go func() {
		defer close(out)
		// Use a background context with 60s timeout so the goroutine
		// is bounded even when the parent request already returned.
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		out <- p.PublishWon(ctx, ev)
	}()
	return out
}

func parseDuration(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
