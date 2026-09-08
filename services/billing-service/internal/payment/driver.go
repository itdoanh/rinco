package payment

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/itdoanh/rinco/services/billing-service/internal/models"
)

// Driver defines the payment provider interface
type Driver interface {
	// CreateCheckoutSession creates a Stripe Checkout session for subscription
	CreateCheckoutSession(ctx context.Context, sub *models.Subscription, successURL, cancelURL string) (string, error)
	// CreateCustomer creates a customer in the payment provider
	CreateCustomer(ctx context.Context, tenantID uuid.UUID, email, name string) (string, error)
	// CreateSubscription creates a recurring subscription
	CreateSubscription(ctx context.Context, customerID string, priceID string, trialDays int) (string, error)
	// CancelSubscription cancels a subscription
	CancelSubscription(ctx context.Context, subID string) error
	// UpdateSubscription updates plan
	UpdateSubscription(ctx context.Context, subID, newPriceID string) error
	// GetCheckoutSession retrieves checkout session details
	GetCheckoutSession(ctx context.Context, sessionID string) (*CheckoutSession, error)
	// ConstructWebhookEvent verifies and parses webhook payload
	ConstructWebhookEvent(ctx context.Context, payload []byte, sig string, secret string) (*WebhookPayload, error)
	// CreateBillingPortalSession creates Stripe Customer Portal session
	CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (string, error)
	// CreateInvoiceItem adds a one-time charge
	CreateInvoiceItem(ctx context.Context, customerID string, amount int64, currency, description string) error
}

type CheckoutSession struct {
	ID         string
	URL        string
	Status     string
	CustomerID string
	SubID      string
}

type WebhookPayload struct {
	Type         string
	SubID        string
	CustomerID   string
	InvoiceID    string
	Status       string
	AmountPaid   int64
	Currency     string
	Email        string
	CustomerAddr string
}

// Common errors
var (
	ErrProviderNotConfigured = errors.New("payment provider not configured")
	ErrCheckoutFailed        = errors.New("checkout session creation failed")
	ErrSubscriptionFailed    = errors.New("subscription creation failed")
	ErrCancellationFailed    = errors.New("subscription cancellation failed")
	ErrWebhookInvalid       = errors.New("invalid webhook signature")
)
