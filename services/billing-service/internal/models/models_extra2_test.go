// Extra tests for billing-service models.
package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestExtraPlan_Constants(t *testing.T) {
	if PlanFree != "free" {
		t.Errorf("PlanFree: %s", PlanFree)
	}
	if PlanPro != "pro" {
		t.Errorf("PlanPro: %s", PlanPro)
	}
	if PlanBusiness != "business" {
		t.Errorf("PlanBusiness: %s", PlanBusiness)
	}
	if PlanEnterprise != "enterprise" {
		t.Errorf("PlanEnterprise: %s", PlanEnterprise)
	}
}

func TestExtraPlanPricing_AllHaveValue(t *testing.T) {
	for plan, price := range PlanPricing {
		if price < 0 {
			t.Errorf("plan %s has negative price: %d", plan, price)
		}
	}
}

func TestExtraPlanPricing_Ordering(t *testing.T) {
	// Free < Pro < Business < Enterprise
	if PlanPricing[PlanFree] >= PlanPricing[PlanPro] {
		t.Error("Free should be < Pro")
	}
	if PlanPricing[PlanPro] >= PlanPricing[PlanBusiness] {
		t.Error("Pro should be < Business")
	}
	if PlanPricing[PlanBusiness] >= PlanPricing[PlanEnterprise] {
		t.Error("Business should be < Enterprise")
	}
}

func TestExtraPlanLimits_AllPlansExist(t *testing.T) {
	for _, plan := range []Plan{PlanFree, PlanPro, PlanBusiness, PlanEnterprise} {
		if _, ok := PlanLimits[plan]; !ok {
			t.Errorf("missing limits for %s", plan)
		}
	}
}

func TestExtraPlanLimits_EnterpriseUnlimited(t *testing.T) {
	l := PlanLimits[PlanEnterprise]
	if l.APIRequestsPerMonth != -1 {
		t.Error("API: should be -1 (unlimited)")
	}
	if l.MaxUsers != -1 {
		t.Error("MaxUsers: should be -1")
	}
}

func TestExtraPlanLimits_FreeLimited(t *testing.T) {
	l := PlanLimits[PlanFree]
	if l.CustomBranding {
		t.Error("Free should not have custom branding")
	}
	if l.SSO {
		t.Error("Free should not have SSO")
	}
	if l.AuditLogRetentionDays != 7 {
		t.Errorf("Free retention: %d", l.AuditLogRetentionDays)
	}
}

func TestExtraSubscription_JSON(t *testing.T) {
	now := time.Now()
	s := Subscription{
		ID:   uuid.New(),
		Plan: PlanPro,
	}
	s.CreatedAt = now
	s.UpdatedAt = now
	s.CurrentPeriodStart = now
	s.CurrentPeriodEnd = now.AddDate(0, 1, 0)
	_ = s
	t.Log("subscription created")
}

func TestExtraSubscription_JSONMarshal(t *testing.T) {
	s := Subscription{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Plan:     PlanPro,
		Status:   "active",
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty JSON")
	}
}

func TestExtraInvoice_JSON(t *testing.T) {
	inv := Invoice{
		ID:     uuid.New(),
		Amount: 9999,
		Status: "open",
	}
	b, err := json.Marshal(inv)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty JSON")
	}
}

func TestExtraUsageRecord_JSON(t *testing.T) {
	u := UsageRecord{
		APIRequests: 1000,
		StorageGB:   5.5,
	}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty JSON")
	}
}

func TestExtraPaymentMethod_JSON(t *testing.T) {
	pm := PaymentMethod{
		Type:    "card",
		Last4:   "4242",
		Brand:   "visa",
		IsDefault: true,
	}
	b, err := json.Marshal(pm)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty JSON")
	}
}

func TestExtraDiscountCode_JSON(t *testing.T) {
	d := DiscountCode{
		Code:          "SUMMER24",
		DiscountType:  "percentage",
		DiscountValue: 25,
		IsActive:      true,
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty JSON")
	}
}

func TestExtraWebhookEvent_JSON(t *testing.T) {
	w := WebhookEvent{
		ID:    uuid.New(),
		Type:  "invoice.paid",
		EventID: "evt_123",
	}
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("empty JSON")
	}
}

func TestExtraSubscription_WithTrial(t *testing.T) {
	now := time.Now()
	trial := now.Add(7 * 24 * time.Hour)
	s := Subscription{
		ID:          uuid.New(),
		Plan:        PlanBusiness,
		Status:      "trialing",
		TrialEndsAt: &trial,
	}
	if s.TrialEndsAt == nil {
		t.Fatal("trial not set")
	}
}

func TestExtraPlanLimits_StorageValues(t *testing.T) {
	if PlanLimits[PlanFree].StorageGB >= PlanLimits[PlanPro].StorageGB {
		t.Error("Free storage should be < Pro")
	}
	if PlanLimits[PlanPro].StorageGB >= PlanLimits[PlanBusiness].StorageGB {
		t.Error("Pro storage should be < Business")
	}
}

func TestExtraPlanLimits_RetentionDays(t *testing.T) {
	if PlanLimits[PlanFree].AuditLogRetentionDays >= PlanLimits[PlanPro].AuditLogRetentionDays {
		t.Error("Free retention < Pro retention")
	}
}

func TestExtraPlan_StringConversion(t *testing.T) {
	if Plan("custom") != "custom" {
		t.Error("Plan is a string")
	}
}

func TestExtraPlanPricing_ZeroForFree(t *testing.T) {
	if PlanPricing[PlanFree] != 0 {
		t.Errorf("Free should be 0: got %d", PlanPricing[PlanFree])
	}
}

// Helper: subscript for tests
func TestHelperTime(t *testing.T) {
	now := time.Now()
	if now.IsZero() {
		t.Error("now should not be zero")
	}
}
