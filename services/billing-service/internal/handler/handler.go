package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/itdoanh/rinco/services/billing-service/internal/models"
	"github.com/itdoanh/rinco/services/billing-service/internal/payment"
	"github.com/itdoanh/rinco/services/billing-service/internal/repository"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Server handles billing HTTP endpoints
type Server struct {
	repo   *repository.Repository
	driver payment.Driver
	log    *slog.Logger
}

func New(repo *repository.Repository, driver payment.Driver) *Server {
	return &Server{repo: repo, driver: driver, log: slog.Default()}
}

// --- Subscription Endpoints ---

type CreateSubscriptionRequest struct {
	TenantID    string `json:"tenant_id"`
	Plan        string `json:"plan"`
	Email       string `json:"email"`
	CompanyName string `json:"company_name"`
}

type CreateSubscriptionResponse struct {
	CheckoutURL string `json:"checkout_url"`
	Subscription *models.Subscription `json:"subscription,omitempty"`
}

func (s *Server) CreateSubscription(c echo.Context) error {
	var req CreateSubscriptionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	plan := models.Plan(req.Plan)
	if _, ok := models.PlanPricing[plan]; !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid plan"})
	}

	if plan == models.PlanFree {
		// Free plan: no checkout needed, create subscription directly
		now := time.Now()
		sub := &models.Subscription{
			ID:                 uuid.New(),
			TenantID:          tenantID,
			Plan:              plan,
			Status:             "active",
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   now.AddDate(0, 1, 0),
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := s.repo.CreateSubscription(context.Background(), sub); err != nil {
			s.log.Error("create free subscription failed", "err", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create subscription"})
		}
		return c.JSON(http.StatusCreated, CreateSubscriptionResponse{Subscription: sub})
	}

	// Paid plan: create Stripe checkout session
	if s.driver == nil || !s.isDriverConfigured() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "payment provider not configured"})
	}

	now := time.Now()
	sub := &models.Subscription{
		ID:                 uuid.New(),
		TenantID:          tenantID,
		Plan:              plan,
		Status:             "pending",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	successURL := c.Scheme() + "://" + c.Request().Host+ "/billing/success?session_id={CHECKOUT_SESSION_ID}"
	cancelURL := c.Scheme() + "://" + c.Request().Host+ "/billing/cancel"

	checkoutURL, err := s.driver.CreateCheckoutSession(context.Background(), sub, successURL, cancelURL)
	if err != nil {
		s.log.Error("create checkout session failed", "err", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create checkout"})
	}

	return c.JSON(http.StatusOK, CreateSubscriptionResponse{CheckoutURL: checkoutURL})
}

func (s *Server) GetSubscription(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID required"})
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	sub, err := s.repo.GetSubscriptionByTenant(context.Background(), tid)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "no subscription found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get subscription"})
	}
	return c.JSON(http.StatusOK, sub)
}

type UpdateSubscriptionRequest struct {
	Plan string `json:"plan"`
}

func (s *Server) UpdateSubscription(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID required"})
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	var req UpdateSubscriptionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	sub, err := s.repo.GetSubscriptionByTenant(context.Background(), tid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "no subscription found"})
	}

	newPlan := models.Plan(req.Plan)
	if _, ok := models.PlanPricing[newPlan]; !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid plan"})
	}

	if sub.StripeSubID != "" && s.isDriverConfigured() {
		priceID := envOr("STRIPE_PRICE_"+string(newPlan), "")
		if err := s.driver.UpdateSubscription(context.Background(), sub.StripeSubID, priceID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update subscription"})
		}
	}

	sub.Plan = newPlan
	sub.UpdatedAt = time.Now()
	if err := s.repo.UpdateSubscription(context.Background(), sub); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update subscription"})
	}

	return c.JSON(http.StatusOK, sub)
}

func (s *Server) CancelSubscription(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID required"})
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	sub, err := s.repo.GetSubscriptionByTenant(context.Background(), tid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "no subscription found"})
	}

	if sub.StripeSubID != "" && s.isDriverConfigured() {
		if err := s.driver.CancelSubscription(context.Background(), sub.StripeSubID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to cancel subscription"})
		}
	}

	now := time.Now()
	sub.Status = "cancelled"
	sub.CancelledAt = &now
	sub.UpdatedAt = now
	if err := s.repo.UpdateSubscription(context.Background(), sub); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to cancel subscription"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) GetBillingPortal(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID required"})
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	sub, err := s.repo.GetSubscriptionByTenant(context.Background(), tid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "no subscription found"})
	}

	if !s.isDriverConfigured() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "payment provider not configured"})
	}

	portalURL, err := s.driver.CreateBillingPortalSession(context.Background(), sub.StripeSubID, c.Scheme()+"://"+c.Request().Host)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create portal session"})
	}

	return c.JSON(http.StatusOK, map[string]string{"portal_url": portalURL})
}

// --- Invoice Endpoints ---

func (s *Server) ListInvoices(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID required"})
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	invoices, err := s.repo.ListInvoices(context.Background(), tid, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list invoices"})
	}
	return c.JSON(http.StatusOK, invoices)
}

func (s *Server) GetInvoice(c echo.Context) error {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invoice_id"})
	}

	inv, err := s.repo.GetInvoice(context.Background(), uid)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "invoice not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get invoice"})
	}
	return c.JSON(http.StatusOK, inv)
}

// --- Usage Endpoints ---

type UsageResponse struct {
	Month       int     `json:"month"`
	Year        int     `json:"year"`
	APIRequests int64   `json:"api_requests"`
	StorageGB   float64 `json:"storage_gb"`
	AIcalls     int64   `json:"ai_calls"`
	TotalEstimate int64  `json:"total_estimate"` // estimated cost in cents
}

func (s *Server) GetUsage(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID required"})
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	month, _ := strconv.Atoi(c.QueryParam("month"))
	year, _ := strconv.Atoi(c.QueryParam("year"))
	if month == 0 {
		month = int(time.Now().Month())
	}
	if year == 0 {
		year = time.Now().Year()
	}

	rec, err := s.repo.GetUsage(context.Background(), tid, month, year)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusOK, UsageResponse{Month: month, Year: year})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get usage"})
	}

	// Estimate cost
	estimate := estimateUsageCost(rec)

	return c.JSON(http.StatusOK, UsageResponse{
		Month:         rec.Month,
		Year:          rec.Year,
		APIRequests:   rec.APIRequests,
		StorageGB:     rec.StorageGB,
		AIcalls:       rec.AIcalls,
		TotalEstimate: estimate,
	})
}

func estimateUsageCost(rec *models.UsageRecord) int64 {
	// API: $0.001 per 1000 requests
	apiCost := int64(rec.APIRequests / 1000)
	// Storage: $0.023 per GB/month
	storageCost := int64(rec.StorageGB * 2.3)
	// AI calls: $0.002 per call
	aiCost := rec.AIcalls * 2
	return apiCost + storageCost + aiCost
}

// --- Admin Endpoints ---

type CreateSubscriptionAdminRequest struct {
	TenantID string `json:"tenant_id"`
	Plan    string `json:"plan"`
	Email   string `json:"email"`
}

func (s *Server) AdminCreateSubscription(c echo.Context) error {
	var req CreateSubscriptionAdminRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}
	plan := models.Plan(req.Plan)

	now := time.Now()
	sub := &models.Subscription{
		ID:                 uuid.New(),
		TenantID:          tenantID,
		Plan:              plan,
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.repo.CreateSubscription(context.Background(), sub); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, sub)
}

func (s *Server) isDriverConfigured() bool {
	if s.driver == nil {
		return false
	}
	if d, ok := s.driver.(*payment.StripeDriver); ok {
		return d.IsConfigured()
	}
	return false
}
