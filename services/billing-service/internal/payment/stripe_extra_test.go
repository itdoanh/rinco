package payment_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/billing-service/internal/models"
	"github.com/itdoanh/rinco/services/billing-service/internal/payment"
)

// =============================================================================
// VATRate — full coverage of all countries + edge cases
// =============================================================================

func TestExtraVATRate_AllDocumentedCountries(t *testing.T) {
	cases := []struct {
		country string
		want    float64
	}{
		{"VN", 10.0},
		{"DE", 19.0},
		{"FR", 20.0},
		{"GB", 20.0},
		{"SG", 9.0},
		{"US", 0.0},
		{"AU", 10.0},
	}
	for _, c := range cases {
		if got := payment.VATRate(c.country); got != c.want {
			t.Errorf("VATRate(%s) = %v, want %v", c.country, got, c.want)
		}
	}
}

func TestExtraVATRate_UnknownCountry_ReturnsZero(t *testing.T) {
	for _, c := range []string{"ZZ", "XX", "moon", "", "123"} {
		if got := payment.VATRate(c); got != 0.0 {
			t.Errorf("VATRate(%q) = %v, want 0.0", c, got)
		}
	}
}

func TestExtraVATRate_CaseSensitivity(t *testing.T) {
	// Map keys are uppercase; lowercase should not match.
	if got := payment.VATRate("vn"); got != 0.0 {
		t.Errorf("lowercase 'vn' should return 0.0, got %v", got)
	}
	if got := payment.VATRate("de"); got != 0.0 {
		t.Errorf("lowercase 'de' should return 0.0, got %v", got)
	}
}

// =============================================================================
// FormatCurrency
// =============================================================================

func TestExtraFormatCurrency_Zero(t *testing.T) {
	if got := payment.FormatCurrency(0, "USD"); got != "0.00 USD" {
		t.Errorf("FormatCurrency(0, USD) = %q, want %q", got, "0.00 USD")
	}
}

func TestExtraFormatCurrency_Negative(t *testing.T) {
	// Function does not handle negatives specially — verify the math.
	got := payment.FormatCurrency(-100, "EUR")
	if got != "-1.00 EUR" {
		t.Errorf("FormatCurrency(-100, EUR) = %q, want %q", got, "-1.00 EUR")
	}
}

func TestExtraFormatCurrency_LargeAmounts(t *testing.T) {
	if got := payment.FormatCurrency(123456789, "USD"); got != "1234567.89 USD" {
		t.Errorf("FormatCurrency(123456789, USD) = %q, want %q", got, "1234567.89 USD")
	}
}

func TestExtraFormatCurrency_DifferentCurrencies(t *testing.T) {
	cases := []struct {
		amount   int64
		currency string
		want     string
	}{
		{100, "VND", "1.00 VND"},
		{99, "JPY", "0.99 JPY"},
		{1000000, "KRW", "10000.00 KRW"},
	}
	for _, c := range cases {
		if got := payment.FormatCurrency(c.amount, c.currency); got != c.want {
			t.Errorf("FormatCurrency(%d, %s) = %q, want %q", c.amount, c.currency, got, c.want)
		}
	}
}

func TestExtraFormatCurrency_AlwaysTwoDecimals(t *testing.T) {
	// 1 cent = 0.01
	if got := payment.FormatCurrency(1, "USD"); !strings.HasSuffix(got, "0.01 USD") {
		t.Errorf("FormatCurrency(1, USD) = %q, expected '...0.01 USD'", got)
	}
}

// =============================================================================
// SubscriptionTrialDays
// =============================================================================

func TestExtraSubscriptionTrialDays_AllPlans(t *testing.T) {
	cases := []struct {
		plan models.Plan
		want int
	}{
		{models.PlanFree, 0},
		{models.PlanPro, 14},
		{models.PlanBusiness, 14},
		{models.PlanEnterprise, 14},
	}
	for _, c := range cases {
		if got := payment.SubscriptionTrialDays(c.plan); got != c.want {
			t.Errorf("SubscriptionTrialDays(%s) = %d, want %d", c.plan, got, c.want)
		}
	}
}

func TestExtraSubscriptionTrialDays_UnknownPlan(t *testing.T) {
	// Unknown plan (not equal to PlanFree) defaults to 14 days.
	if got := payment.SubscriptionTrialDays(models.Plan("unknown")); got != 14 {
		t.Errorf("unknown plan should default to 14, got %d", got)
	}
}

// =============================================================================
// NewStripeDriver / IsConfigured
// =============================================================================

func TestExtraNewStripeDriver_StoresPriceIDs(t *testing.T) {
	prices := map[models.Plan]string{
		models.PlanPro:      "price_pro_123",
		models.PlanBusiness: "price_biz_456",
	}
	d := payment.NewStripeDriver("sk_test_xxx", prices)
	if !d.IsConfigured() {
		t.Error("should be configured with secret key")
	}
}

func TestExtraNewStripeDriverFromEnv_EmptyEnv(t *testing.T) {
	// NewStripeDriverFromEnv uses os.Getenv which is empty in tests.
	// Should produce a non-nil but unconfigured driver.
	d := payment.NewStripeDriverFromEnv()
	if d == nil {
		t.Fatal("expected non-nil driver")
	}
	if d.IsConfigured() {
		t.Error("empty env should produce unconfigured driver")
	}
}

// =============================================================================
// CreateCheckoutSession — valid input path
// =============================================================================

func TestExtraCreateCheckoutSession_NoPriceConfigured(t *testing.T) {
	d := payment.NewStripeDriver("sk_test_xxx", map[models.Plan]string{})
	ctx := context.Background()
	sub := &models.Subscription{
		TenantID: uuid.New(),
		Plan:     models.PlanPro,
	}
	_, err := d.CreateCheckoutSession(ctx, sub, "https://ok", "https://cancel")
	if err == nil {
		t.Error("missing price ID should produce error")
	}
}

func TestExtraCreateCheckoutSession_StubURL(t *testing.T) {
	d := payment.NewStripeDriver("sk_test_xxx", map[models.Plan]string{
		models.PlanPro: "price_pro_123",
	})
	ctx := context.Background()
	tenantID := uuid.New()
	sub := &models.Subscription{
		TenantID: tenantID,
		Plan:     models.PlanPro,
	}
	url, err := d.CreateCheckoutSession(ctx, sub, "https://success.example.com", "https://cancel.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "checkout_stub=true") {
		t.Errorf("expected stub marker in URL, got %s", url)
	}
	if !strings.Contains(url, "plan=pro") {
		t.Errorf("expected plan=pro in URL, got %s", url)
	}
	if !strings.Contains(url, tenantID.String()) {
		t.Errorf("expected tenant ID in URL, got %s", url)
	}
}

// =============================================================================
// GetCheckoutSession / ConstructWebhookEvent / CreateBillingPortalSession /
// CreateInvoiceItem — unconfigured paths
// =============================================================================

func TestExtraGetCheckoutSession_NotConfigured(t *testing.T) {
	d := payment.NewStripeDriver("", nil)
	_, err := d.GetCheckoutSession(context.Background(), "sess_1")
	if err == nil {
		t.Error("expected ErrProviderNotConfigured")
	}
}

func TestExtraGetCheckoutSession_Configured(t *testing.T) {
	d := payment.NewStripeDriver("sk_test_xxx", nil)
	s, err := d.GetCheckoutSession(context.Background(), "sess_123")
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "sess_123" {
		t.Errorf("ID: got %s, want sess_123", s.ID)
	}
	if s.Status != "complete" {
		t.Errorf("Status: got %s, want complete", s.Status)
	}
}

func TestExtraConstructWebhookEvent_NotConfigured(t *testing.T) {
	d := payment.NewStripeDriver("", nil)
	_, err := d.ConstructWebhookEvent(context.Background(), []byte(`{}`), "sig", "secret")
	if err == nil {
		t.Error("expected ErrWebhookInvalid")
	}
}

func TestExtraConstructWebhookEvent_EmptySecret(t *testing.T) {
	d := payment.NewStripeDriver("sk_test_xxx", nil)
	_, err := d.ConstructWebhookEvent(context.Background(), []byte(`{}`), "sig", "")
	if err == nil {
		t.Error("expected ErrWebhookInvalid with empty secret")
	}
}

func TestExtraConstructWebhookEvent_Configured(t *testing.T) {
	d := payment.NewStripeDriver("sk_test_xxx", nil)
	p, err := d.ConstructWebhookEvent(context.Background(), []byte(`{"type":"x"}`), "sig", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("expected non-nil payload")
	}
	if p.Type != "stub.event" {
		t.Errorf("Type: got %s, want stub.event", p.Type)
	}
}

func TestExtraCreateBillingPortalSession_NotConfigured(t *testing.T) {
	d := payment.NewStripeDriver("", nil)
	_, err := d.CreateBillingPortalSession(context.Background(), "cus_1", "https://return")
	if err == nil {
		t.Error("expected ErrProviderNotConfigured")
	}
}

func TestExtraCreateBillingPortalSession_Configured(t *testing.T) {
	d := payment.NewStripeDriver("sk_test_xxx", nil)
	url, err := d.CreateBillingPortalSession(context.Background(), "cus_abc123", "https://return")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "cus_abc123") {
		t.Errorf("expected customer ID in URL, got %s", url)
	}
}

func TestExtraCreateInvoiceItem_NotConfigured(t *testing.T) {
	d := payment.NewStripeDriver("", nil)
	err := d.CreateInvoiceItem(context.Background(), "cus_1", 100, "USD", "test")
	if err == nil {
		t.Error("expected ErrProviderNotConfigured")
	}
}

func TestExtraCreateInvoiceItem_Configured(t *testing.T) {
	d := payment.NewStripeDriver("sk_test_xxx", nil)
	if err := d.CreateInvoiceItem(context.Background(), "cus_1", 100, "USD", "test"); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
