package models

import (
	"time"

	"github.com/google/uuid"
)

// CAPIEvent represents a Facebook CAPI event
type CAPIEvent struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	EventID      string    `json:"event_id"`      // unique per event instance (for deduplication)
	EventName    string    `json:"event_name"`    // e.g., "Purchase", "Lead", "Contact"
	EventTime    time.Time `json:"event_time"`
	EventSource  string    `json:"event_source"`   // "web", "app", "offline", "crm"
	Email        string    `json:"email,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	Country      string    `json:"country,omitempty"`
	FBPID        string    `json:"fbp_id,omitempty"` // Facebook Pixel ID
	FBCID        string    `json:"fbc_id,omitempty"` // Facebook Click ID
	LeadID       string    `json:"lead_id,omitempty"`
	DealID       string    `json:"deal_id,omitempty"`
	OrderValue   float64   `json:"order_value,omitempty"`
	Currency     string    `json:"currency,omitempty"`
	CustomData   map[string]string `json:"custom_data,omitempty"`
	Status       string    `json:"status"` // pending, sent, delivered, failed, suppressed
	FBEventID    string    `json:"fb_event_id,omitempty"` // returned by Facebook
	ErrorMessage string    `json:"error_message,omitempty"`
	RetryCount   int       `json:"retry_count"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// CAPIConfig holds Facebook CAPI credentials per tenant
type CAPIConfig struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	PixelID        string    `json:"pixel_id"`
	AccessToken    string    `json:"access_token"` // encrypted at rest
	TestEventCode  string    `json:"test_event_code,omitempty"`
	IsEnabled      bool      `json:"is_enabled"`
	EventTypes     []string  `json:"event_types"` // which events to send
	SampleRate     float64   `json:"sample_rate"` // 0.0-1.0 for sampling
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// FeedbackEvent represents CAPI feedback from Facebook (hashed events returned)
type FeedbackEvent struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	CAPIEventID     uuid.UUID `json:"capi_event_id"`
	FBEventID       string    `json:"fb_event_id"`
	FBEventName     string    `json:"fb_event_name"`
	FBEventTime     time.Time `json:"fb_event_time"`
	FBPartnerName   string    `json:"fb_partner_name"`
	FBPartnerID     string    `json:"fb_partner_id"`
	FBArtistID      string    `json:"fb_artist_id,omitempty"`
	FBClaimCode     string    `json:"fb_claim_code,omitempty"`
	FBDisaggregate string    `json:"fb_disaggregate,omitempty"`
	ProcessedAt     time.Time `json:"processed_at"`
}

// ConversionMapping maps CRM events to Facebook CAPI events
type ConversionMapping struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	CRMEventType string    `json:"crm_event_type"` // e.g., "deal_won", "lead_created"
	CAPIEventName string   `json:"capi_event_name"` // e.g., "Purchase", "Lead"
	IsActive     bool      `json:"is_active"`
	ValueField   string    `json:"value_field,omitempty"` // which field to use for order_value
	CreatedAt    time.Time `json:"created_at"`
}

// AggregatedConversion holds aggregated conversion data for reporting
type AggregatedConversion struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	Date          time.Time `json:"date"`
	EventName     string    `json:"event_name"`
	TotalEvents   int64     `json:"total_events"`
	Delivered     int64     `json:"delivered"`
	Failed        int64     `json:"failed"`
	Suppressed    int64     `json:"suppressed"`
	TotalValue    float64   `json:"total_value"`
	Currency      string    `json:"currency"`
}
