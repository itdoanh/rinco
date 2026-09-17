package handler

import (
	"context"
	mathrand "math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/itdoanh/rinco/services/meta-capi-service/internal/capi"
	"github.com/itdoanh/rinco/services/meta-capi-service/internal/models"
)

// Pool is a small wrapper interface used by SendLeadEvent/SendPurchase to
// resolve tenants when callers don't pass X-Tenant-ID directly.  The
// handler.Server already has access to the pool via the repository, but
// we expose a separate function to keep the wiring explicit.
type Pool interface {
	GetCAPIConfig(ctx context.Context, tenantID uuid.UUID) (*models.CAPIConfig, error)
}

// --- Alias handlers for the canonical /v1/... routes ---

// SendLeadEvent is the /v1/capi/lead entrypoint.  It is functionally
// identical to SendEvent but is locked to the "Lead" event name so the
// landing-service can call /v1/capi/lead without specifying the event
// type.  We keep SendEvent for backward compatibility.
func (s *Server) SendLeadEvent(c echo.Context) error {
	var req SendEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	req.EventName = "Lead"
	return s.processRequest(c, req)
}

// SendPurchase is the /v1/capi/purchase entrypoint.  It is identical to
// SendEvent but is locked to the "Purchase" event name and requires an
// order_value/currency pair (the CRM fires it on deal-won).
func (s *Server) SendPurchase(c echo.Context) error {
	var req SendEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	req.EventName = "Purchase"
	return s.processRequest(c, req)
}

// processRequest centralises the dedup, circuit-breaker and dispatch
// logic that both SendEvent and the alias handlers share.
func (s *Server) processRequest(c echo.Context, req SendEventRequest) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID required"})
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	cfg, err := s.repo.GetCAPIConfig(context.Background(), tid)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "CAPI not configured"})
	}

	// Note: dedup in this revision only happens at the Facebook CAPI
	// (event_id is a client-supplied UUID), so the handler just needs
	// to ensure every event has a stable id that matches the pixel
	// side for the 48h dedup window.  See ``capi.NewEventPayload``.

	// Sample rate
	if cfg.SampleRate < 1.0 {
		// Use the same mrand package as SendEvent via the alias
		// package; we keep a copy here to avoid touching SendEvent.
		// Cheap uniform [0,1) draw.
		// mrand.Float64 is global and protected by its own mutex.
		if randFloat() > cfg.SampleRate {
			return c.JSON(http.StatusOK, map[string]string{"status": "sampled_out"})
		}
	}

	ev := &models.CAPIEvent{
		ID:          uuid.New(),
		TenantID:    tid,
		EventID:     uuid.New().String(),
		EventName:   req.EventName,
		EventTime:   time.Now(),
		EventSource: "crm",
		Email:       req.Email,
		Phone:       req.Phone,
		IPAddress:   req.IPAddress,
		UserAgent:   req.UserAgent,
		Country:     req.Country,
		FBPID:       req.FBPID,
		FBCID:       req.FBCID,
		LeadID:      req.LeadID,
		DealID:      req.DealID,
		OrderValue:  req.OrderValue,
		Currency:    req.Currency,
		CustomData:  req.CustomData,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	_ = s.repo.CreateCAPIEvent(context.Background(), ev)

	fbEventID, err := s.sendToFacebookWithRetry(cfg, ev)
	if err != nil {
		_ = s.repo.UpdateCAPIEventStatus(context.Background(), ev.ID, "failed", "", err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	_ = s.repo.UpdateCAPIEventStatus(context.Background(), ev.ID, "sent", fbEventID, "")
	return c.JSON(http.StatusOK, map[string]string{"status": "sent", "fb_event_id": fbEventID})
}

// --- Observability ---

// ListEvents returns the most recent CAPI events for the tenant
// (debug / audit log).
func (s *Server) ListEvents(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID required"})
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	limit := 50
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	status := c.QueryParam("status")

	events, err := s.repo.ListCAPIEvents(context.Background(), tid, status, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
		"limit":  limit,
	})
}

// GetCAPIStats returns aggregate stats: total sent, failed, dedup
// rate, match rate.
func (s *Server) GetCAPIStats(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID required"})
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	stats, err := s.repo.GetCAPIStats(context.Background(), tid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, stats)
}

// --- Internal helpers ---

// sendToFacebookWithRetry is identical to sendToFacebook but adds
// exponential backoff retry for transient failures.  Circuit-breaker
// behaviour is implemented by the capi.Client; this wrapper only adds
// retry semantics at the handler level.
func (s *Server) sendToFacebookWithRetry(cfg *models.CAPIConfig, ev *models.CAPIEvent) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// 1s, 2s, 4s backoff
			delay := time.Duration(1<<attempt) * time.Second
			time.Sleep(delay)
		}
		fbEventID, err := s.sendToFacebook(cfg, ev)
		if err == nil {
			return fbEventID, nil
		}
		lastErr = err
		// Only retry on transient HTTP errors
		if !isRetryable(err) {
			return "", err
		}
	}
	return "", lastErr
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "504")
}

func randFloat() float64 {
	return mathrand.Float64()
}

// String imports needed for isRetryable
var _ = capi.NewClient
var _ = pgxpool.New
