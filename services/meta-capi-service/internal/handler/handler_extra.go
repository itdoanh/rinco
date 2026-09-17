// Package handler provides additional HTTP handlers for the meta-capi-service.
//
// These handlers complement the core handlers in handler.go and handler_v1.go.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/itdoanh/rinco/services/meta-capi-service/internal/capi"
	"github.com/itdoanh/rinco/services/meta-capi-service/internal/models"
)

// SendEventRequest is the shape accepted by SendLeadEvent and SendPurchase.
type SendEventRequest struct {
	EventName  string            `json:"event_name"`
	Email      string            `json:"email,omitempty"`
	Phone      string            `json:"phone,omitempty"`
	IPAddress  string            `json:"ip_address,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
	Country    string            `json:"country,omitempty"`
	FBPID      string            `json:"fbp_id,omitempty"`
	FBCID      string            `json:"fbc_id,omitempty"`
	LeadID     string            `json:"lead_id,omitempty"`
	DealID     string            `json:"deal_id,omitempty"`
	OrderValue float64           `json:"order_value,omitempty"`
	Currency   string            `json:"currency,omitempty"`
	CustomData map[string]string `json:"custom_data,omitempty"`
}

// sendToFacebook sends a CAPI event to Meta Graph API using the capi client.
func (s *Server) sendToFacebook(cfg *models.CAPIConfig, ev *models.CAPIEvent) (string, error) {
	if s.capiClient == nil {
		s.capiClient = capi.NewClient()
	}

	payload := capi.NewEventPayload()

	userData := capi.UserData{
		Email:         ev.Email,
		Phone:         ev.Phone,
		FBCookieID:    ev.FBCID,
		FBPIDCookieID: ev.FBPID,
		IPAddress:     ev.IPAddress,
		UserAgent:     ev.UserAgent,
		Country:       ev.Country,
	}

	customData := capi.CustomData{
		Value:    ev.OrderValue,
		Currency: ev.Currency,
	}

	eventData := capi.CAPIEventData{
		EventID:      ev.EventID,
		EventName:    ev.EventName,
		EventTime:    ev.EventTime.Unix(),
		ActionSource: "website",
		UserData:     userData,
		CustomData:   customData,
	}

	if ev.EventSource != "" {
		eventData.EventSource = ev.EventSource
	}

	payload.AppendEvent(eventData)

	// Use test event code if configured
	if cfg.TestEventCode != "" {
		payload.Debug = &capi.DebugMode{Mode: 1}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := s.capiClient.SendEvents(ctx, cfg.AccessToken, cfg.PixelID, payload, cfg.TestEventCode != "")
	if err != nil {
		return "", err
	}

	if len(results) > 0 && results[0].FBEventID != "" {
		return results[0].FBEventID, nil
	}

	return "", nil
}

// SendEvent handles POST /capi/events — generic CAPI event sender.
func (s *Server) SendEvent(c echo.Context) error {
	var req SendEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	return s.processRequest(c, req)
}

// SendEventInternal is an alias for processRequest.
func (s *Server) SendEventInternal(c echo.Context) error {
	return nil
}

// CRMBridgeRequest is the shape accepted by the CRM bridge endpoint.
type CRMBridgeRequest struct {
	EventType  string            `json:"event_type"`
	EventName  string            `json:"event_name"`
	LeadID     string            `json:"lead_id,omitempty"`
	DealID     string            `json:"deal_id,omitempty"`
	Email      string            `json:"email,omitempty"`
	Phone      string            `json:"phone,omitempty"`
	FBCLID     string            `json:"fbclid,omitempty"`
	FBPID      string            `json:"fbp_id,omitempty"`
	OrderValue float64           `json:"order_value,omitempty"`
	Currency   string            `json:"currency,omitempty"`
	IPAddress  string            `json:"ip_address,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
	Country    string            `json:"country,omitempty"`
	CustomData map[string]string `json:"custom_data,omitempty"`
	Timestamp  int64             `json:"timestamp,omitempty"`
	EventID    string            `json:"event_id,omitempty"`
}

// CRMBridge handles POST /bridge/crm and /v1/capi/bridge/crm — receives events from CRM.
func (s *Server) CRMBridge(c echo.Context) error {
	var req CRMBridgeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

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
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "CAPI not configured for tenant"})
	}

	// Determine event name
	eventName := req.EventName
	if eventName == "" {
		switch req.EventType {
		case "deal_won", "purchase":
			eventName = "Purchase"
		case "lead_created", "lead_updated":
			eventName = "Lead"
		case "checkout_started":
			eventName = "InitiateCheckout"
		case "contact":
			eventName = "Contact"
		default:
			eventName = "Custom"
		}
	}

	eventID := req.EventID
	if eventID == "" {
		eventID = uuid.New().String()
	}

	timestamp := req.Timestamp
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}

	ev := &models.CAPIEvent{
		ID:          uuid.New(),
		TenantID:    tid,
		EventID:     eventID,
		EventName:   eventName,
		EventTime:   time.Unix(timestamp, 0),
		EventSource: "crm",
		Email:       req.Email,
		Phone:       req.Phone,
		IPAddress:   req.IPAddress,
		UserAgent:   req.UserAgent,
		Country:     req.Country,
		FBPID:       req.FBPID,
		FBCID:       req.FBCLID,
		LeadID:      req.LeadID,
		DealID:      req.DealID,
		OrderValue:  req.OrderValue,
		Currency:    req.Currency,
		CustomData:  req.CustomData,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateCAPIEvent(context.Background(), ev); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	fbEventID, err := s.sendToFacebookWithRetry(cfg, ev)
	if err != nil {
		_ = s.repo.UpdateCAPIEventStatus(context.Background(), ev.ID, "failed", "", err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	_ = s.repo.UpdateCAPIEventStatus(context.Background(), ev.ID, "sent", fbEventID, "")

	return c.JSON(http.StatusOK, map[string]string{
		"status":      "sent",
		"event_id":    eventID,
		"fb_event_id": fbEventID,
	})
}

// SetupCAPIRequest is the shape accepted by the SetupCAPI endpoint.
type SetupCAPIRequest struct {
	TenantID      string   `json:"tenant_id"`
	PixelID       string   `json:"pixel_id"`
	AccessToken   string   `json:"access_token"`
	TestEventCode string   `json:"test_event_code,omitempty"`
	EventTypes    []string `json:"event_types,omitempty"`
	SampleRate    float64  `json:"sample_rate,omitempty"`
	IsEnabled     bool     `json:"is_enabled"`
}

// SetupCAPI handles POST /admin/capi/setup — configure CAPI for a tenant.
func (s *Server) SetupCAPI(c echo.Context) error {
	var req SetupCAPIRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.TenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
	}
	if req.PixelID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "pixel_id required"})
	}
	if req.AccessToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "access_token required"})
	}

	tid, err := uuid.Parse(req.TenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	eventTypes := req.EventTypes
	if len(eventTypes) == 0 {
		eventTypes = []string{"Lead", "Purchase", "Contact", "ViewContent"}
	}

	sampleRate := req.SampleRate
	if sampleRate <= 0 || sampleRate > 1 {
		sampleRate = 1.0
	}

	cfg := &models.CAPIConfig{
		ID:            uuid.New(),
		TenantID:      tid,
		PixelID:       req.PixelID,
		AccessToken:   req.AccessToken,
		TestEventCode: req.TestEventCode,
		IsEnabled:     req.IsEnabled,
		EventTypes:    eventTypes,
		SampleRate:    sampleRate,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.UpsertCAPIConfig(context.Background(), cfg); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if s.capiClient == nil {
		s.capiClient = capi.NewClient()
	}
	testErr := s.capiClient.TestConnection(context.Background(), req.AccessToken, req.PixelID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":          "configured",
		"tenant_id":       req.TenantID,
		"pixel_id":        req.PixelID,
		"connection_test": testErr == nil,
	})
}

// GetCAPIStatus returns the current CAPI status for the tenant.
func (s *Server) GetCAPIStatus(c echo.Context) error {
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
		return c.JSON(http.StatusNotFound, map[string]string{
			"status":    "not_configured",
			"tenant_id": tenantID,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":      "configured",
		"tenant_id":   tenantID,
		"pixel_id":    cfg.PixelID,
		"is_enabled":  cfg.IsEnabled,
		"event_types": cfg.EventTypes,
		"sample_rate": cfg.SampleRate,
	})
}
