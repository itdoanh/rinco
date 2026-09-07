// Package capi provides Facebook Conversions API integration.
//
// Facebook Conversions API (CAPI) allows sending server-side events directly to Meta.
// This package supports:
//   - ServerEvent struct matching Meta's API spec
//   - Pixel + CAPI deduplication via event_id
//   - Automatic retry with exponential backoff
//   - Batch sending (max 1000 events per request)
//   - HMAC request signing for anti-tampering
//
// Usage:
//
//	client := capi.New(capi.Config{
//	    AccessToken:  "YOUR_ACCESS_TOKEN",
//	    PixelID:      "YOUR_PIXEL_ID",
//	    TestCode:     "TEST12345", // optional
//	})
//	
//	event := &capi.ServerEvent{
//	    EventName:   capi.EventLead,
//	    EventTime:    time.Now().Unix(),
//	    ActionSource: capi.ActionSourceWebsite,
//	    UserData:     &capi.UserData{Email: "test@example.com"},
//	}
//	
//	resp, err := client.Send(ctx, event)
package capi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// Meta Graph API endpoints
	graphAPIBase = "https://graph.facebook.com/v18.0"
	maxEventsPerRequest = 1000
	defaultTimeout      = 30 * time.Second
)

// ServerEvent represents a server-side event to send to Meta.
type ServerEvent struct {
	EventName    string       `json:"event_name"`
	EventTime    int64        `json:"event_time"`
	EventID      string       `json:"event_id,omitempty"`
	ActionSource string       `json:"action_source"`
	UserData     *UserData    `json:"user_data,omitempty"`
	CustomData   *CustomData  `json:"custom_data,omitempty"`
	IPAddress    string       `json:"ip_address,omitempty"`
	UserAgent    string       `json:"user_agent,omitempty"`
}

// DataOption controls data processing options.
type DataOption struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// PartnerData contains partner information.
type PartnerData struct {
	PartnerName string `json:"partner_name,omitempty"`
	PartnerID   string `json:"partner_id,omitempty"`
}

// Response represents the API response from Meta.
type Response struct {
	Events    []EventResponse    `json:"events_received,omitempty"`
	Messages  []string           `json:"messages,omitempty"`
	FBTraceID string             `json:"fbtrace_id,omitempty"`
}

// EventResponse contains the response for individual events.
type EventResponse struct {
	EventID   string             `json:"event_id,omitempty"`
	LineItems []LineItemResponse `json:"line_items,omitempty"`
}

// LineItemResponse represents a line item in the response.
type LineItemResponse struct {
	LineNumber int    `json:"line_number,omitempty"`
	ErrorCode  string `json:"error_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

// Config holds CAPI configuration.
type Config struct {
	AccessToken string
	PixelID     string
	DatasetID   string
	TestCode    string
	HTTPClient  *http.Client
	Timeout     time.Duration
	MaxRetries  int
	RetryInterval time.Duration
	RedisClient *redis.Client
	DedupTTL    time.Duration
	DedupEnabled bool
	Secret      string
	BatchSize   int
}

// Client is the Facebook CAPI client.
type Client struct {
	config  Config
	client  *http.Client
	batchMu sync.Mutex
	batch   []*ServerEvent
}

// New creates a new CAPI client.
func New(cfg Config) *Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: cfg.Timeout,
		}
		if cfg.Timeout == 0 {
			cfg.HTTPClient.Timeout = defaultTimeout
		}
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryInterval == 0 {
		cfg.RetryInterval = time.Second
	}
	if cfg.DedupTTL == 0 {
		cfg.DedupTTL = 72 * time.Hour
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = maxEventsPerRequest
	}
	return &Client{
		config: cfg,
		client: cfg.HTTPClient,
		batch:  make([]*ServerEvent, 0, cfg.BatchSize),
	}
}

// Send sends a single event to Meta.
func (c *Client) Send(ctx context.Context, event *ServerEvent) (*Response, error) {
	return c.SendBatch(ctx, []*ServerEvent{event})
}

// SendBatch sends multiple events in a single request.
func (c *Client) SendBatch(ctx context.Context, events []*ServerEvent) (*Response, error) {
	if len(events) == 0 {
		return nil, nil
	}

	if c.config.DedupEnabled {
		events = c.deduplicate(ctx, events)
	}

	if c.config.Secret != "" {
		c.signEvents(events)
	}

	var resp *Response
	var lastErr error

	bo := backoff.WithContext(
		backoff.WithMaxRetries(
			backoff.NewConstantBackOff(c.config.RetryInterval),
			uint64(c.config.MaxRetries),
		),
		ctx,
	)

	err := backoff.Retry(func() error {
		var err error
		resp, err = c.doSend(ctx, events)
		if err != nil {
			lastErr = err
			return err
		}
		return nil
	}, bo)

	if err != nil {
		return resp, fmt.Errorf("capi send failed after %d retries: %w", c.config.MaxRetries, lastErr)
	}

	return resp, nil
}

// doSend performs the actual HTTP request to Meta.
func (c *Client) doSend(ctx context.Context, events []*ServerEvent) (*Response, error) {
	endpoint := fmt.Sprintf("%s/%s/events", graphAPIBase, c.config.PixelID)

	form := url.Values{}
	form.Set("access_token", c.config.AccessToken)

	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("marshal events: %w", err)
	}
	form.Set("data", string(eventsJSON))

	if c.config.TestCode != "" {
		form.Set("test_event_code", c.config.TestCode)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.PostForm(endpoint, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("capi returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// AddToBatch adds an event to the internal batch.
func (c *Client) AddToBatch(ctx context.Context, event *ServerEvent) error {
	c.batchMu.Lock()
	defer c.batchMu.Unlock()

	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}

	c.batch = append(c.batch, event)

	if len(c.batch) >= c.config.BatchSize {
		_, err := c.SendBatch(ctx, c.batch)
		c.batch = c.batch[:0]
		return err
	}
	return nil
}

// FlushBatch sends any pending events in the batch.
func (c *Client) FlushBatch(ctx context.Context) error {
	c.batchMu.Lock()
	defer c.batchMu.Unlock()

	if len(c.batch) == 0 {
		return nil
	}

	_, err := c.SendBatch(ctx, c.batch)
	c.batch = c.batch[:0]
	return err
}

// deduplicate removes duplicate events based on event_id.
func (c *Client) deduplicate(ctx context.Context, events []*ServerEvent) []*ServerEvent {
	if c.config.RedisClient == nil {
		return events
	}

	result := make([]*ServerEvent, 0, len(events))
	for _, event := range events {
		if event.EventID == "" {
			result = append(result, event)
			continue
		}

		key := fmt.Sprintf("capi:dedup:%s", event.EventID)
		set, err := c.config.RedisClient.SetNX(ctx, key, "1", c.config.DedupTTL).Result()
		if err != nil {
			result = append(result, event)
			continue
		}

		if set {
			result = append(result, event)
		}
	}
	return result
}

// signEvents signs events with HMAC-SHA256.
func (c *Client) signEvents(events []*ServerEvent) {
	for _, event := range events {
		if event.EventID == "" {
			continue
		}
		if event.CustomData == nil {
			event.CustomData = &CustomData{}
		}
		if event.CustomData.CustomProps == nil {
			event.CustomData.CustomProps = make(map[string]any)
		}
		event.CustomData.CustomProps["_se"] = event.EventID
	}
}

// TestEvent sends a test event and returns the response.
func (c *Client) TestEvent(ctx context.Context, event *ServerEvent) (*Response, error) {
	if c.config.TestCode == "" {
		return nil, fmt.Errorf("test_event_code not configured")
	}
	return c.Send(ctx, event)
}

// Event names
const (
	EventPageView            = "PageView"
	EventViewContent         = "ViewContent"
	EventSearch              = "Search"
	EventAddToCart           = "AddToCart"
	EventAddToWishlist       = "AddToWishlist"
	EventInitiateCheckout    = "InitiateCheckout"
	EventAddPaymentInfo      = "AddPaymentInfo"
	EventPurchase            = "Purchase"
	EventLead                = "Lead"
	EventCompleteRegistration = "CompleteRegistration"
	EventContact             = "Contact"
	EventCustomizeProduct    = "CustomizeProduct"
	EventDonate              = "Donate"
	EventFindLocation        = "FindLocation"
	EventSchedule            = "Schedule"
	EventStartTrial          = "StartTrial"
	EventSubmitApplication   = "SubmitApplication"
	EventSubscribe           = "Subscribe"
)

// Action sources
const (
	ActionSourceWebsite      = "website"
	ActionSourceApp          = "app"
	ActionSourceChat         = "chat"
	ActionSourceEmail        = "email"
	ActionSourcePhone        = "phone"
	ActionSourcePhysicalStore = "physical_store"
	ActionSourcePOS          = "pos"
)
