package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/billing-service/internal/models"
)

func TestPlanPricingExists(t *testing.T) {
	plans := []models.Plan{models.PlanFree, models.PlanPro, models.PlanBusiness, models.PlanEnterprise}
	for _, p := range plans {
		price, ok := models.PlanPricing[p]
		if !ok {
			t.Errorf("missing pricing for plan %s", p)
		}
		if price < 0 {
			t.Errorf("price for plan %s should be >= 0, got %d", p, price)
		}
	}
}

func TestPlanLimits(t *testing.T) {
	freeLimits := models.PlanLimits[models.PlanFree]
	if freeLimits.MaxUsers != 5 {
		t.Errorf("Free plan should allow 5 users, got %d", freeLimits.MaxUsers)
	}
	proLimits := models.PlanLimits[models.PlanPro]
	if proLimits.MaxUsers != 50 {
		t.Errorf("Pro plan should allow 50 users, got %d", proLimits.MaxUsers)
	}
}

func TestSubscriptionCreation(t *testing.T) {
	now := time.Now()
	sub := &models.Subscription{
		ID:                 uuid.New(),
		TenantID:          uuid.New(),
		Plan:              models.PlanPro,
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if sub.Plan != models.PlanPro {
		t.Errorf("expected PlanPro, got %s", sub.Plan)
	}
	if sub.Status != "active" {
		t.Errorf("expected status active, got %s", sub.Status)
	}
}

func TestInvoiceCreation(t *testing.T) {
	now := time.Now()
	inv := &models.Invoice{
		ID:              uuid.New(),
		TenantID:       uuid.New(),
		SubscriptionID: uuid.New(),
		Number:         "INV-2026-00001",
		Status:         "open",
		Amount:         9900,
		Currency:       "USD",
		DueDate:         now.AddDate(0, 0, 14),
		InvoiceDate:     now,
		CreatedAt:       now,
	}
	if inv.Amount != 9900 {
		t.Errorf("expected amount 9900, got %d", inv.Amount)
	}
	if inv.Number != "INV-2026-00001" {
		t.Errorf("expected INV-2026-00001, got %s", inv.Number)
	}
}

func TestUsageRecord(t *testing.T) {
	rec := &models.UsageRecord{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		Month:       12,
		Year:        2026,
		APIRequests: 1000,
		StorageGB:   5.5,
		AIcalls:     100,
		RecordedAt:  time.Now(),
	}
	if rec.Month != 12 {
		t.Error("month should be 12")
	}
	if rec.Year != 2026 {
		t.Error("year should be 2026")
	}
}
