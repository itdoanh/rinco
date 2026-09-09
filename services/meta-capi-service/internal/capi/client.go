package capi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps Facebook Conversions API v19.0
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new CAPI client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// EventPayload represents the Facebook CAPI event payload
type EventPayload struct {
	Data  []CAPIEventData `json:"data"`
	Debug *DebugMode      `json:"debug,omitempty"`
}

// NewEventPayload constructs an empty payload with a non-nil Data
// slice so that the marshalled JSON always contains ``"data":[]``
// instead of ``"data":null`` (which the Facebook CAPI rejects).
func NewEventPayload() EventPayload {
	return EventPayload{Data: []CAPIEventData{}}
}

// AppendEvent adds an event to the payload, allocating the slice if
// nil so the marshalled result is always a JSON array.
func (p *EventPayload) AppendEvent(e CAPIEventData) {
	if p.Data == nil {
		p.Data = []CAPIEventData{}
	}
	p.Data = append(p.Data, e)
}

// CAPIEventData represents a single event in the CAPI payload
type CAPIEventData struct {
	EventID       string            `json:"event_id,omitempty"`
	EventName     string            `json:"event_name"`
	EventTime     int64             `json:"event_time"` // Unix timestamp
	EventSource   string            `json:"event_source_url,omitempty"`
	ActionSource  string            `json:"action_source"`
	UserData      UserData          `json:"user_data"`
	CustomData    CustomData        `json:"custom_data,omitempty"`
	OptOut        bool              `json:"opt_out,omitempty"`
	ProcessingOptions ProcessingOptions `json:"processing_options,omitempty"`
}

// UserData contains user identifiers
type UserData struct {
	Email         string `json:"em,omitempty"`
	Phone         string `json:"ph,omitempty"`
	FirstName     string `json:"fn,omitempty"`
	LastName      string `json:"ln,omitempty"`
	City          string `json:"ct,omitempty"`
	State         string `json:"st,omitempty"`
	Zip           string `json:"zp,omitempty"`
	Country       string `json:"country,omitempty"`
	ExternalID    string `json:"external_id,omitempty"`
	FBCookieID    string `json:"fbc,omitempty"`
	FBPIDCookieID string `json:"fbp,omitempty"`
	IPAddress     string `json:"client_ip_address,omitempty"`
	UserAgent     string `json:"client_user_agent,omitempty"`
}

// CustomData contains event-specific data
type CustomData struct {
	Value       float64            `json:"value,omitempty"`
	Currency    string             `json:"currency,omitempty"`
	ContentName string             `json:"content_name,omitempty"`
	ContentType string             `json:"content_type,omitempty"`
	Contents    []ContentItem      `json:"contents,omitempty"`
	OrderID     string             `json:"order_id,omitempty"`
	CustomProps map[string]string  `json:"custom_properties,omitempty"`
}

// ContentItem represents a content item
type ContentItem struct {
	ID       string  `json:"id,omitempty"`
	Quantity int     `json:"quantity,omitempty"`
	Price    float64 `json:"item_price,omitempty"`
}

// ProcessingOptions controls how Facebook processes the event
type ProcessingOptions struct {
	AllowNoSales       bool `json:"allow_unofficial_click identifiers,omitempty"`
	OverrideUserData    bool `json:"override_user_data,omitempty"`
	OverrideMatch      bool `json:"override_match,omitempty"`
}

// DebugMode enables debug mode for CAPI
type DebugMode struct {
	Mode int `json:"mode,omitempty"` // 0 = production, 1 = debug
}

// APIResponse is the response from Facebook CAPI
type APIResponse struct {
	Events     []EventResult `json:"events_received,omitempty"`
	Messages   []string      `json:"messages,omitempty"`
	DebugMsg   []string      `json:"debug_messages,omitempty"`
	ID         string        `json:"id,omitempty"`
	FBEventID  string        `json:"fb_event_id,omitempty"`
}

// EventResult is the result for a single event
type EventResult struct {
	EventID      string `json:"event_id,omitempty"`
	FBEventID    string `json:"fb_event_id,omitempty"`
	Tracepoint   string `json:"tracepoint,omitempty"`
	ErrorCode    int    `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// SendEvents sends events to Facebook Conversions API
func (c *Client) SendEvents(ctx context.Context, accessToken, pixelID string, payload EventPayload, testMode bool) ([]EventResult, error) {
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/events?access_token=%s", pixelID, accessToken)
	return c.SendEventsToURL(ctx, url, payload)
}

// SendEventsToURL posts ``payload`` to the given fully-qualified URL.  It is
// the same as :func:`SendEvents` but accepts an arbitrary endpoint, which is
// useful for unit tests that hit a local ``httptest`` server.  Production
// callers should prefer :func:`SendEvents`.
func (c *Client) SendEventsToURL(ctx context.Context, url string, payload EventPayload) ([]EventResult, error) {
	_ = ctx
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result APIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Messages != nil && len(result.Messages) > 0 {
		// Some events failed
		var errors []string
		for _, msg := range result.Messages {
			errors = append(errors, msg)
		}
		return result.Events, fmt.Errorf("CAPI errors: %v", errors)
	}

	return result.Events, nil
}

// TestConnection tests if the CAPI access token is valid
func (c *Client) TestConnection(ctx context.Context, accessToken, pixelID string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s?access_token=%s", pixelID, accessToken)
	return c.TestConnectionToURL(ctx, url)
}

// TestConnectionToURL issues a GET against ``url`` and expects a 200 OK.
// Like :func:`SendEventsToURL` this exists primarily so unit tests can
// point at a local ``httptest`` server.
func (c *Client) TestConnectionToURL(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("test failed: %s", string(body))
	}
	return nil
}
