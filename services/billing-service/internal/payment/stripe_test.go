package payment_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/billing-service/internal/models"
	"github.com/itdoanh/rinco/services/billing-service/internal/payment"
)

func TestStripeDriverNotConfigured(t *testing.T) {
	d := payment.NewStripeDriver("", map[models.Plan]string{})
	if d.IsConfigured() {
		t.Error("empty secret key should not be configured")
	}
}

func TestStripeDriverConfigured(t *testing.T) {
	d := payment.NewStripeDriver("sk_test_abc", map[models.Plan]string{
		models.PlanPro: "price_abc123",
	})
	if !d.IsConfigured() {
		t.Error("non-empty secret key should be configured")
	}
}

func TestStripeDriverErrorsWhenNotConfigured(t *testing.T) {
	d := payment.NewStripeDriver("", nil)
	ctx := context.Background()
	tenantID := uuid.New()

	_, err := d.CreateCustomer(ctx, tenantID, "test@example.com", "Test")
	if err == nil {
		t.Error("CreateCustomer should fail when not configured")
	}

	_, err = d.CreateCheckoutSession(ctx, &models.Subscription{
		TenantID: tenantID,
		Plan:    models.PlanPro,
	}, "https://success", "https://cancel")
	if err == nil {
		t.Error("CreateCheckoutSession should fail when not configured")
	}

	err = d.CancelSubscription(ctx, "sub_123")
	if err == nil {
		t.Error("CancelSubscription should fail when not configured")
	}
}

func TestVATRate(t *testing.T) {
	tests := []struct {
		country string
		want    float64
	}{
		{"VN", 10.0},
		{"DE", 19.0},
		{"US", 0.0},
		{"XX", 0.0},
	}
	for _, tc := range tests {
		got := payment.VATRate(tc.country)
		if got != tc.want {
			t.Errorf("VATRate(%s) = %f, want %f", tc.country, got, tc.want)
		}
	}
}

func TestSubscriptionTrialDays(t *testing.T) {
	if payment.SubscriptionTrialDays(models.PlanFree) != 0 {
		t.Error("Free plan should have 0 trial days")
	}
	if payment.SubscriptionTrialDays(models.PlanPro) != 14 {
		t.Error("Pro plan should have 14 trial days")
	}
}

func TestFormatCurrency(t *testing.T) {
	s := payment.FormatCurrency(9900, "USD")
	if s != "99.00 USD" {
		t.Errorf("expected '99.00 USD', got %s", s)
	}
}
