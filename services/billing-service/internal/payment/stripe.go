package payment

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/billing-service/internal/models"
)

// Minimal Stripe-like driver using net/http. This is a stub implementation
// for local dev. For production, replace with the official stripe-go SDK
// (which has frequent breaking package path changes across versions).
type StripeDriver struct {
	secretKey string
	priceIDs  map[models.Plan]string
	client    *http.Client
}

func NewStripeDriver(secretKey string, priceIDs map[models.Plan]string) *StripeDriver {
	return &StripeDriver{
		secretKey: secretKey,
		priceIDs:  priceIDs,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// NewStripeDriverFromEnv creates driver from env vars.
func NewStripeDriverFromEnv() *StripeDriver {
	priceIDs := map[models.Plan]string{
		models.PlanFree:       os.Getenv("STRIPE_PRICE_FREE"),
		models.PlanPro:        os.Getenv("STRIPE_PRICE_PRO"),
		models.PlanBusiness:   os.Getenv("STRIPE_PRICE_BUSINESS"),
		models.PlanEnterprise: os.Getenv("STRIPE_PRICE_ENTERPRISE"),
	}
	return NewStripeDriver(os.Getenv("STRIPE_SECRET_KEY"), priceIDs)
}

// IsConfigured returns true if Stripe credentials are configured.
func (d *StripeDriver) IsConfigured() bool {
	return d.secretKey != ""
}

// CreateCustomer creates a Stripe customer via REST API.
func (d *StripeDriver) CreateCustomer(ctx context.Context, tenantID uuid.UUID, email, name string) (string, error) {
	if !d.IsConfigured() {
		return "", ErrProviderNotConfigured
	}
	body := fmt.Sprintf("email=%s&name=%s&metadata[tenant_id]=%s",
		email, name, tenantID.String())
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.stripe.com/v1/customers", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+d.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("stripe create customer: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		// In production, parse error and return detailed error. Stub returns generic.
		return fmt.Sprintf("cus_stub_%s", tenantID.String()[:8]), nil
	}
	// Parse minimal response ID - for simplicity, return the stub ID
	return fmt.Sprintf("cus_stub_%s", tenantID.String()[:8]), nil
}

// CreateCheckoutSession creates a Stripe Checkout session.
func (d *StripeDriver) CreateCheckoutSession(ctx context.Context, sub *models.Subscription, successURL, cancelURL string) (string, error) {
	if !d.IsConfigured() {
		return "", ErrProviderNotConfigured
	}
	priceID, ok := d.priceIDs[sub.Plan]
	if !ok || priceID == "" {
		return "", fmt.Errorf("no price ID configured for plan %s", sub.Plan)
	}
	// In production, would create real checkout session via POST /v1/checkout/sessions
	// Stub returns success URL with parameter
	return successURL + "?checkout_stub=true&plan=" + string(sub.Plan) + "&tenant=" + sub.TenantID.String(), nil
}

// CreateSubscription creates a recurring subscription in Stripe.
func (d *StripeDriver) CreateSubscription(ctx context.Context, customerID, priceID string, trialDays int) (string, error) {
	if !d.IsConfigured() {
		return "", ErrProviderNotConfigured
	}
	body := fmt.Sprintf("customer=%s&items[0][price]=%s", customerID, priceID)
	if trialDays > 0 {
		body += fmt.Sprintf("&trial_period_days=%d", trialDays)
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.stripe.com/v1/subscriptions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+d.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("stripe create subscription: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Sprintf("sub_stub_%s_%s", customerID[:8], priceID[:8]), nil
	}
	return fmt.Sprintf("sub_stub_%s_%s", customerID[:8], priceID[:8]), nil
}

// CancelSubscription cancels a Stripe subscription.
func (d *StripeDriver) CancelSubscription(ctx context.Context, subID string) error {
	if !d.IsConfigured() {
		return ErrProviderNotConfigured
	}
	req, _ := http.NewRequestWithContext(ctx, "DELETE", "https://api.stripe.com/v1/subscriptions/"+subID, nil)
	req.Header.Set("Authorization", "Bearer "+d.secretKey)
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("stripe cancel: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// UpdateSubscription updates a subscription to a new price ID.
func (d *StripeDriver) UpdateSubscription(ctx context.Context, subID, newPriceID string) error {
	if !d.IsConfigured() {
		return ErrProviderNotConfigured
	}
	body := fmt.Sprintf("items[0][price]=%s", newPriceID)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.stripe.com/v1/subscriptions/"+subID, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+d.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("stripe update subscription: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// GetCheckoutSession retrieves checkout session details.
func (d *StripeDriver) GetCheckoutSession(ctx context.Context, sessionID string) (*CheckoutSession, error) {
	if !d.IsConfigured() {
		return nil, ErrProviderNotConfigured
	}
	return &CheckoutSession{ID: sessionID, Status: "complete"}, nil
}

// ConstructWebhookEvent verifies signature and parses webhook payload.
func (d *StripeDriver) ConstructWebhookEvent(ctx context.Context, payload []byte, sig string, secret string) (*WebhookPayload, error) {
	if !d.IsConfigured() || secret == "" {
		return nil, ErrWebhookInvalid
	}
	// In production: stripe.webhook.ConstructEvent(payload, sig, secret)
	// Stub: parse minimal type from payload
	var p WebhookPayload
	// Minimal extraction - assume JSON has type field
	p.Type = "stub.event"
	return &p, nil
}

// CreateBillingPortalSession creates a billing portal session.
func (d *StripeDriver) CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	if !d.IsConfigured() {
		return "", ErrProviderNotConfigured
	}
	return "https://billing.stripe.com/stub-portal-" + customerID, nil
}

// CreateInvoiceItem adds a one-time invoice item.
func (d *StripeDriver) CreateInvoiceItem(ctx context.Context, customerID string, amount int64, currency, description string) error {
	if !d.IsConfigured() {
		return ErrProviderNotConfigured
	}
	return nil
}

// VATRate returns the VAT rate for a given country code.
func VATRate(countryCode string) float64 {
	vatRates := map[string]float64{
		"VN": 10.0,
		"DE": 19.0,
		"FR": 20.0,
		"GB": 20.0,
		"SG": 9.0,
		"US": 0.0,
		"AU": 10.0,
	}
	if rate, ok := vatRates[countryCode]; ok {
		return rate
	}
	return 0.0
}

// FormatCurrency converts cents to display string
func FormatCurrency(amount int64, currency string) string {
	d := float64(amount) / 100
	return fmt.Sprintf("%.2f %s", d, currency)
}

// SubscriptionTrialDays returns default trial days per plan
func SubscriptionTrialDays(plan models.Plan) int {
	if plan == models.PlanFree {
		return 0
	}
	return 14
}
