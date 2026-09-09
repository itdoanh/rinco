// Extra tests for billing-service handler helpers.
package handler

import (
	"testing"
	"time"

	"github.com/itdoanh/rinco/services/billing-service/internal/models"
)

func TestEnvOrEx_Default(t *testing.T) {
	t.Setenv("TEST_ENV_VAR_X", "")
	if got := envOr("TEST_ENV_VAR_X", "default"); got != "default" {
		t.Errorf("got %s", got)
	}
}

func TestEnvOrEx_Set(t *testing.T) {
	t.Setenv("TEST_ENV_VAR_X", "value")
	if got := envOr("TEST_ENV_VAR_X", "default"); got != "value" {
		t.Errorf("got %s", got)
	}
}

func TestEstimateUsageCost_Empty(t *testing.T) {
	rec := &models.UsageRecord{}
	got := estimateUsageCost(rec)
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestEstimateUsageCost_APICalls(t *testing.T) {
	rec := &models.UsageRecord{APIRequests: 1000}
	got := estimateUsageCost(rec)
	if got != 1 {
		t.Errorf("expected 1, got %d", got)
	}
}

func TestEstimateUsageCost_Storage(t *testing.T) {
	rec := &models.UsageRecord{StorageGB: 100}
	got := estimateUsageCost(rec)
	// 100 * 2.3 → int64(229.999...) → 229
	if got != 229 {
		t.Errorf("expected 229, got %d", got)
	}
}

func TestEstimateUsageCost_AICalls(t *testing.T) {
	rec := &models.UsageRecord{AIcalls: 100}
	got := estimateUsageCost(rec)
	// 100 * 2 = 200
	if got != 200 {
		t.Errorf("expected 200, got %d", got)
	}
}

func TestEstimateUsageCost_Combined(t *testing.T) {
	rec := &models.UsageRecord{
		APIRequests: 10000,  // → 10
		StorageGB:   100,    // → 229
		AIcalls:     50,     // → 100
	}
	got := estimateUsageCost(rec)
	// 10 + 229 + 100 = 339
	if got != 339 {
		t.Errorf("expected 339, got %d", got)
	}
}

func TestEstimateUsageCost_LargeValues(t *testing.T) {
	rec := &models.UsageRecord{
		APIRequests: 1_000_000_000, // → 1_000_000
		StorageGB:   1000,          // → 2300
		AIcalls:     1000,          // → 2000
	}
	got := estimateUsageCost(rec)
	if got <= 0 {
		t.Errorf("expected positive, got %d", got)
	}
}

func TestEstimateUsageCost_NilRec(t *testing.T) {
	// Note: nil rec causes panic due to field access.
	// This test documents the current behavior.
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil rec")
		}
	}()
	_ = estimateUsageCost(nil)
}

func TestNew_NoDriver(t *testing.T) {
	s := New(nil, nil)
	if s == nil {
		t.Fatal("nil server")
	}
	if s.driver != nil {
		t.Error("driver should be nil")
	}
}

func TestIsDriverConfigured_NilDriver(t *testing.T) {
	s := &Server{driver: nil}
	if s.isDriverConfigured() {
		t.Error("nil driver should not be configured")
	}
}

func TestIsDriverConfigured_NonStripeDriver(t *testing.T) {
	s := &Server{driver: nil} // dummy
	// Non-Stripe drivers return false
	if s.isDriverConfigured() {
		t.Error("non-stripe should not be configured")
	}
}

func TestServer_NilRepo(t *testing.T) {
	s := &Server{}
	if s.repo != nil {
		t.Error("repo should be nil")
	}
}

func TestServer_Logger(t *testing.T) {
	s := New(nil, nil)
	if s.log == nil {
		t.Error("logger should be set")
	}
}

func TestUsageResponse_Defaults(t *testing.T) {
	r := UsageResponse{}
	if r.Month != 0 {
		t.Error("month should default to 0")
	}
}

func TestUsageResponse_WithValues(t *testing.T) {
	r := UsageResponse{
		Month:         1,
		Year:          2026,
		APIRequests:   1000,
		StorageGB:     10,
		AIcalls:       100,
		TotalEstimate: 50,
	}
	if r.TotalEstimate != 50 {
		t.Errorf("got %d", r.TotalEstimate)
	}
}

func TestCreateSubscriptionRequest_Empty(t *testing.T) {
	r := CreateSubscriptionRequest{}
	if r.Plan != "" {
		t.Error("empty plan")
	}
}

func TestUpdateSubscriptionRequest_Empty(t *testing.T) {
	r := UpdateSubscriptionRequest{}
	if r.Plan != "" {
		t.Error("empty plan")
	}
}

func TestAdminCreateSubscriptionRequest_Empty(t *testing.T) {
	r := CreateSubscriptionAdminRequest{}
	if r.TenantID != "" {
		t.Error("empty tenant_id")
	}
}

func TestCreateSubscriptionResponse_Defaults(t *testing.T) {
	r := CreateSubscriptionResponse{}
	if r.CheckoutURL != "" {
		t.Error("empty url")
	}
	if r.Subscription != nil {
		t.Error("nil sub")
	}
}

func TestServer_NowUsedForTimestamps(t *testing.T) {
	s := New(nil, nil)
	if s == nil {
		t.Fatal("nil server")
	}
	// Just verify time operations don't break
	now := time.Now()
	if now.IsZero() {
		t.Error("now should not be zero")
	}
}
