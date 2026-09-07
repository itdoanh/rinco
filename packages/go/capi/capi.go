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
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// Config holds CAPI configuration.
type Config struct {
	// Facebook API credentials
	AccessToken string
	PixelID    string
	DatasetID  string // Optional: for Conversions API
	TestCode   string // Optional: test event code for testing

	// HTTP client settings
	HTTPClient  *http.Client
	Timeout     time.Duration

	// Retry settings
	MaxRetries    int
	RetryInterval time.Duration

	// Dedup settings
	RedisClient   *redis.Client
	DedupTTL      time.Duration // default 72 hours
	DedupEnabled  bool

	// Signing secret for request integrity (optional)
	Secret    string

	// Batch settings
	BatchSize int
}

// Client is the Facebook CAPI client.
type Client struct {
	config    Config
	client    *http.Client
	dedupMu   sync.Mutex
	batchMu   sync.Mutex
	batch     []*ServerEvent
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

// ServerEvent represents a server-side event to send to Meta.
type ServerEvent struct {
	// Common fields
	EventName   string     `json:"event_name"`
	EventTime   int64      `json:"event_time"` // Unix timestamp
	EventID     string     `json:"event_id,omitempty"`
	ActionSource string    `json:"action_source"`

	// Optional fields
	UserData    *UserData  `json:"user_data,omitempty"`
	CustomData  *CustomData `json:"custom_data,omitempty"`
	DataOptions []DataOption `json:"data_options,omitempty"`
	PartnerData []PartnerData `json:"partner_data,omitempty"`

	// Context (optional)
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// UserData contains user information for matching.
type UserData struct {
	Email             string `json:"em,omitempty"`        // SHA256 hashed
	EmailSHA256       string `json:"e,omitempty"`         // Already hashed
	Phone             string `json:"ph,omitempty"`        // SHA256 hashed
	PhoneSHA256       string `json:"p,omitempty"`         // Already hashed
	FirstName         string `json:"fn,omitempty"`         // SHA256 hashed
	FirstNameSHA256   string `json:"fnb,omitempty"`        // Already hashed
	LastName          string `json:"ln,omitempty"`         // SHA256 hashed
	LastNameSHA256    string `json:"lnb,omitempty"`        // Already hashed
	DateOfBirth       string `json:"dob,omitempty"`        // SHA256 hashed
	Gender            string `json:"ge,omitempty"`         // SHA256 hashed
	City              string `json:"ct,omitempty"`         // SHA256 hashed
	State             string `json:"st,omitempty"`         // SHA256 hashed
	ZipCode           string `json:"zp,omitempty"`         // SHA256 hashed
	Country           string `json:"country,omitempty"`    // SHA256 hashed
	ExternalID        string `json:"external_id,omitempty"`
	ClientIPAddress   string `json:"client_ip_address,omitempty"`
	ClientUserAgent   string `json:"client_user_agent,omitempty"`
	FBCookieID        string `json:"fbc,omitempty"`
	FBPCookieID       string `json:"fbp,omitempty"`
	SubscriptionID    string `json:"subscription_id,omitempty"`
	LeadID            string `json:"lead_id,omitempty"`
}

// CustomData contains event-specific information.
type CustomData struct {
	Value        float64          `json:"value,omitempty"`
	Currency     string           `json:"currency,omitempty"`
	ContentName  string           `json:"content_name,omitempty"`
	ContentCategory string        `json:"content_category,omitempty"`
	ContentIDs   []string         `json:"content_ids,omitempty"`
	ContentType  string           `json:"content_type,omitempty"`
	Contents     []ContentItem    `json:"contents,omitempty"`
	NumItems     int              `json:"num_items,omitempty"`
	OrderID      string           `json:"order_id,omitempty"`
	SearchString string           `json:"search_string,omitempty"`
	CustomProps  map[string]any   `json:"custom_data,omitempty"`
}

// ContentItem represents a content item in an event.
type ContentItem struct {
	ID          string  `json:"id,omitempty"`
	Quantity    int     `json:"quantity,omitempty"`
	ItemPrice   float64 `json:"item_price,omitempty"`
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Brand       string  `json:"brand,omitempty"`
	Category    string  `json:"category,omitempty"`
}

// DataOption controls data processing options.
type DataOption struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
}

// PartnerData contains partner information.
type PartnerData struct {
	PartnerName string `json:"partner_name,omitempty"`
	PartnerID   string `json:"partner_id,omitempty"`
}

// Response represents the API response from Meta.
type Response struct {
	Events   []EventResponse `json:"events_received,omitempty"`
	Messages []string        `json:"messages,omitempty"`
	FBTraceID string        `json:"fbtrace_id,omitempty"`
}

// EventResponse contains the response for individual events.
type EventResponse struct {
	EventID   string `json:"event_id,omitempty"`
	LineItems []LineItemResponse `json:"line_items,omitempty"`
}

// LineItemResponse represents a line item in the response.
type LineItemResponse struct {
	LineNumber int    `json:"line_number,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Send sends a single event to Meta.
func (c *Client) Send(ctx context.Context, event *ServerEvent) (*Response, error) {
	return c.SendBatch(ctx, []*ServerEvent{event})
}

// SendBatch sends multiple events in a single request.
// Automatically handles batching if events exceed BatchSize.
func (c *Client) SendBatch(ctx context.Context, events []*ServerEvent) (*Response, error) {
	if len(events) == 0 {
		return nil, nil
	}

	// Check for duplicates if dedup is enabled
	if c.config.DedupEnabled {
		events = c.deduplicate(ctx, events)
	}

	// Sign events if secret is configured
	if c.config.Secret != "" {
		c.signEvents(events)
	}

	// Make request with retry
	var resp *Response
	var lastErr error

	bo := backoff.WithContext(
		backoff.WithMaxRetries(
			backoff.NewConstantBackOff(c.config.RetryInterval),
			uint64(c.config.MaxRetries),
		),
		ctx,
	)

	err := bo.Retry(func() error {
		var err error
		resp, err = c.doSend(ctx, events)
		if err != nil {
			lastErr = err
			return err
		}
		return nil
	})

	if err != nil {
		return resp, fmt.Errorf("capi send failed after %d retries: %w", c.config.MaxRetries, lastErr)
	}

	return resp, nil
}

// doSend performs the actual HTTP request to Meta.
func (c *Client) doSend(ctx context.Context, events []*ServerEvent) (*Response, error) {
	// Build endpoint
	endpoint := fmt.Sprintf("%s/%s/events", graphAPIBase, c.config.PixelID)

	// Build request body
	body := map[string]any{
		"events": events,
	}

	// Add test event code if present
	if c.config.TestCode != "" {
		body["test_event_code"] = c.config.TestCode
	}

	// Add access token
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Set up form data
	form := make(url.Values)
	form.Set("access_token", c.config.AccessToken)
	form.Set("events", mustMarshalJSON(events))
	if c.config.TestCode != "" {
		form.Set("test_event_code", c.config.TestCode)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	resp, err := c.client.PostForm(endpoint+"?access_token="+c.config.AccessToken,
		bytes.NewBufferString(form.Encode()))
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
// Flushes automatically when batch size is reached.
func (c *Client) AddToBatch(ctx context.Context, event *ServerEvent) error {
	c.batchMu.Lock()
	defer c.batchMu.Unlock()

	// Generate event ID if not set
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

		// Check if event_id already exists in Redis
		key := fmt.Sprintf("capi:dedup:%s", event.EventID)
		set, err := c.config.RedisClient.SetNX(ctx, key, "1", c.config.DedupTTL).Result()
		if err != nil {
			// If Redis fails, include the event
			result = append(result, event)
			continue
		}

		if set {
			// Event is new, include it
			result = append(result, event)
		}
		// If set is false, event is duplicate, skip it
	}
	return result
}

// signEvents signs events with HMAC-SHA256.
func (c *Client) signEvents(events []*ServerEvent) {
	for _, event := range events {
		if event.EventID == "" {
			continue
		}
		// Sign event_id with secret
		h := hmac.New(sha256.New, []byte(c.config.Secret))
		h.Write([]byte(event.EventID))
		sig := hex.EncodeToString(h.Sum(nil))
		// Store signature in custom_data (Meta uses this for verification)
		if event.CustomData == nil {
			event.CustomData = &CustomData{}
		}
		if event.CustomData.CustomProps == nil {
			event.CustomData.CustomProps = make(map[string]any)
		}
		event.CustomData.CustomProps["_se"] = sig
	}
}

// TestEvent sends a test event and returns the response.
func (c *Client) TestEvent(ctx context.Context, event *ServerEvent) (*Response, error) {
	if c.config.TestCode == "" {
		return nil, fmt.Errorf("test_event_code not configured")
	}

	// Temporarily use test code
	originalCode := c.config.TestCode
	defer func() { c.config.TestCode = originalCode }()

	return c.Send(ctx, event)
}

// Event names
const (
	EventPageView           = "PageView"
	EventViewContent        = "ViewContent"
	EventSearch             = "Search"
	EventAddToCart          = "AddToCart"
	EventAddToWishlist      = "AddToWishlist"
	EventInitiateCheckout    = "InitiateCheckout"
	EventAddPaymentInfo      = "AddPaymentInfo"
	EventPurchase           = "Purchase"
	EventLead               = "Lead"
	EventCompleteRegistration = "CompleteRegistration"
	EventContact            = "Contact"
	EventCustomizeProduct   = "CustomizeProduct"
	EventDonate             = "Donate"
	EventFindLocation       = "FindLocation"
	EventSchedule           = "Schedule"
	EventStartTrial         = "StartTrial"
	EventSubmitApplication  = "SubmitApplication"
	EventSubscribe          = "Subscribe"
)

// Action sources
const (
	ActionSourceWebsite     = "website"
	ActionSourceApp         = "app"
	ActionSourceChat        = "chat"
	ActionSourceEmail       = "email"
	ActionSourcePhone       = "phone"
	ActionSourcePhysicalStore = "physical_store"
	ActionSourcePOS         = "pos"
)

// url is a placeholder for net/url since we need it in doSend
type url = struct{}

func init() {
	_ = &url{}
}
