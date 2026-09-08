package models

import (
	"time"

	"github.com/google/uuid"
)

// Plan represents a billing plan
type Plan string

const (
	PlanFree       Plan = "free"
	PlanPro        Plan = "pro"
	PlanBusiness   Plan = "business"
	PlanEnterprise Plan = "enterprise"
)

// Subscription represents a tenant subscription
type Subscription struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	Plan                 Plan       `json:"plan"`
	Status               string     `json:"status"` // active, pending, cancelled, past_due
	StripeSubID          string     `json:"stripe_sub_id,omitempty"`
	StripeCustomerID     string     `json:"stripe_customer_id,omitempty"`
	CurrentPeriodStart   time.Time  `json:"current_period_start"`
	CurrentPeriodEnd     time.Time  `json:"current_period_end"`
	TrialEndsAt          *time.Time `json:"trial_ends_at,omitempty"`
	CancelledAt          *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// Invoice represents a billing invoice
type Invoice struct {
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	SubscriptionID   uuid.UUID `json:"subscription_id"`
	Number           string    `json:"number"`
	InvoiceNumber    string    `json:"invoice_number"`
	Amount           int64     `json:"amount"` // in cents
	Currency         string    `json:"currency"`
	TaxAmount        int64     `json:"tax_amount"` // in cents
	TaxPercent       float64   `json:"tax_percent"`
	Status           string    `json:"status"` // paid, open, void, uncollectible
	StripeInvoiceID  string    `json:"stripe_invoice_id,omitempty"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	DueDate          time.Time `json:"due_date"`
	InvoiceDate      time.Time `json:"invoice_date"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// UsageRecord tracks monthly usage for a tenant
type UsageRecord struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Month        int       `json:"month"`
	Year         int       `json:"year"`
	APIRequests  int64     `json:"api_requests"`
	StorageGB    float64   `json:"storage_gb"`
	AIcalls      int64     `json:"ai_calls"`
	RecordedAt   time.Time `json:"recorded_at"`
}

// PaymentMethod represents a stored payment method
type PaymentMethod struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	StripePMID      string    `json:"stripe_pm_id"`
	Type            string    `json:"type"` // card, bank_account
	Last4           string    `json:"last4"`
	Brand           string    `json:"brand"` // visa, mastercard, etc.
	ExpMonth        int       `json:"exp_month"`
	ExpYear         int       `json:"exp_year"`
	IsDefault       bool      `json:"is_default"`
	CreatedAt       time.Time `json:"created_at"`
}

// DiscountCode represents a discount or coupon
type DiscountCode struct {
	ID             uuid.UUID  `json:"id"`
	Code           string    `json:"code"`
	DiscountType   string    `json:"discount_type"` // percentage, fixed_amount
	DiscountValue  int64     `json:"discount_value"`
	Currency       string    `json:"currency,omitempty"`
	MaxRedemptions int       `json:"max_redemptions"`
	MaxUses        int       `json:"max_uses"`
	RedemptionCount int       `json:"redemption_count"`
	CurrentUses    int       `json:"current_uses"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	ValidFrom      *time.Time `json:"valid_from,omitempty"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

// WebhookEvent stores processed webhook events for idempotency
type WebhookEvent struct {
	ID            uuid.UUID `json:"id"`
	EventID       string    `json:"event_id"`
	StripeEventID string    `json:"stripe_event_id"`
	EventType     string    `json:"event_type"`
	Type          string    `json:"type"`
	ProcessedAt   time.Time `json:"processed_at"`
	Payload       []byte    `json:"payload"`
	DataPayload   []byte    `json:"data_payload"`
	Attempts      int       `json:"attempts"`
	Error         string    `json:"error"`
	CreatedAt     time.Time `json:"created_at"`
}

// PlanPricing defines pricing for each plan (in cents per month)
var PlanPricing = map[Plan]int64{
	PlanFree:       0,
	PlanPro:        9900,       // $99/month
	PlanBusiness:   29900,      // $299/month
	PlanEnterprise: 99900,      // $999/month
}

// PlanLimits defines resource limits per plan
var PlanLimits = map[Plan]PlanLimitsConfig{
	PlanFree: {
		APIRequestsPerMonth:   10000,
		StorageGB:             1,
		AIcallsPerMonth:       100,
		MaxUsers:              5,
		MaxLeads:              500,
		MaxLandingPages:       1,
		MaxSeats:              5,
		CustomBranding:        false,
		SSO:                   false,
		AuditLogRetentionDays: 7,
	},
	PlanPro: {
		APIRequestsPerMonth:   100000,
		StorageGB:             10,
		AIcallsPerMonth:       1000,
		MaxUsers:              50,
		MaxLeads:              10000,
		MaxLandingPages:       5,
		MaxSeats:              50,
		CustomBranding:        true,
		SSO:                   false,
		AuditLogRetentionDays: 30,
	},
	PlanBusiness: {
		APIRequestsPerMonth:   1000000,
		StorageGB:             100,
		AIcallsPerMonth:       10000,
		MaxUsers:              100,
		MaxLeads:              100000,
		MaxLandingPages:       50,
		MaxSeats:              100,
		CustomBranding:        true,
		SSO:                   true,
		AuditLogRetentionDays: 90,
	},
	PlanEnterprise: {
		APIRequestsPerMonth:   -1, // unlimited
		StorageGB:             -1,
		AIcallsPerMonth:       -1,
		MaxUsers:              -1,
		MaxLeads:              -1,
		MaxLandingPages:       -1,
		MaxSeats:              -1,
		CustomBranding:        true,
		SSO:                   true,
		AuditLogRetentionDays: 365,
	},
}

// PlanLimitsConfig defines resource limits for a plan
type PlanLimitsConfig struct {
	APIRequestsPerMonth   int
	StorageGB             int
	AIcallsPerMonth       int
	MaxUsers              int
	MaxLeads              int
	MaxLandingPages       int
	MaxSeats              int
	CustomBranding        bool
	SSO                   bool
	AuditLogRetentionDays int
}
