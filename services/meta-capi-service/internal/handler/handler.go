package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/itdoanh/rinco/services/meta-capi-service/internal/capi"
	"github.com/itdoanh/rinco/services/meta-capi-service/internal/models"
	"github.com/itdoanh/rinco/services/meta-capi-service/internal/repository"
)

type Server struct {
	repo  *repository.Repository
	capi  *capi.Client
	log   *slog.Logger
}

func New(repo *repository.Repository) *Server {
	return &Server{repo: repo, capi: capi.NewClient(), log: slog.Default()}
}

// --- Configuration Endpoints ---

type SetupCAPIRequest struct {
	TenantID     string   `json:"tenant_id"`
	PixelID      string   `json:"pixel_id"`
	AccessToken  string   `json:"access_token"`
	TestEventCode string  `json:"test_event_code,omitempty"`
	EventTypes   []string `json:"event_types,omitempty"`
	SampleRate   float64  `json:"sample_rate"`
}

func (s *Server) SetupCAPI(c echo.Context) error {
	var req SetupCAPIRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	cfg := &models.CAPIConfig{
		ID:            uuid.New(),
		TenantID:      tenantID,
		PixelID:       req.PixelID,
		AccessToken:   req.AccessToken,
		TestEventCode: req.TestEventCode,
		IsEnabled:     true,
		EventTypes:    req.EventTypes,
		SampleRate:    req.SampleRate,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if cfg.EventTypes == nil {
		cfg.EventTypes = []string{"Lead", "Purchase", "Contact", "ViewContent"}
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 1.0
	}

	if err := s.repo.UpsertCAPIConfig(context.Background(), cfg); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "CAPI configured"})
}

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
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "CAPI not configured"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Mask access token
	masked := cfg.AccessToken
	if len(cfg.AccessToken) > 8 {
		masked = cfg.AccessToken[:4] + "****" + cfg.AccessToken[len(cfg.AccessToken)-4:]
	}
	cfg.AccessToken = masked

	return c.JSON(http.StatusOK, cfg)
}

// --- Event Ingestion Endpoints ---

type SendEventRequest struct {
	EventName  string            `json:"event_name"`
	Email      string            `json:"email,omitempty"`
	Phone      string            `json:"phone,omitempty"`
	LeadID     string            `json:"lead_id,omitempty"`
	DealID     string            `json:"deal_id,omitempty"`
	OrderValue float64           `json:"order_value,omitempty"`
	Currency   string            `json:"currency,omitempty"`
	IPAddress  string            `json:"ip_address,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
	Country    string            `json:"country,omitempty"`
	FBPID      string            `json:"fbp_id,omitempty"`
	FBCID      string            `json:"fbc_id,omitempty"`
	CustomData map[string]string `json:"custom_data,omitempty"`
}

func (s *Server) SendEvent(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID required"})
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	var req SendEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	cfg, err := s.repo.GetCAPIConfig(context.Background(), tid)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "CAPI not configured"})
	}

	// Sample: skip if above sample rate
	if cfg.SampleRate < 1.0 && float64(time.Now().UnixNano()%10000)/100.0 > cfg.SampleRate*100 {
		return c.JSON(http.StatusOK, map[string]string{"status": "sampled_out"})
	}

	// Create event record
	ev := &models.CAPIEvent{
		ID:          uuid.New(),
		TenantID:    tid,
		EventID:     uuid.New().String(), // deduplication ID
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

	if err := s.repo.CreateCAPIEvent(context.Background(), ev); err != nil {
		s.log.Error("store capi event failed", "err", err)
	}

	// Send to Facebook
	fbEventID, err := s.sendToFacebook(cfg, ev)
	if err != nil {
		_ = s.repo.UpdateCAPIEventStatus(context.Background(), ev.ID, "failed", "", err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	_ = s.repo.UpdateCAPIEventStatus(context.Background(), ev.ID, "sent", fbEventID, "")

	return c.JSON(http.StatusOK, map[string]string{"status": "sent", "fb_event_id": fbEventID})
}

func (s *Server) sendToFacebook(cfg *models.CAPIConfig, ev *models.CAPIEvent) (string, error) {
	actionSource := "website"
	if ev.EventSource == "offline" {
		actionSource = "offline"
	}

	payload := capi.EventPayload{
		Data: []capi.CAPIEventData{{
			EventID:      ev.EventID,
			EventName:    ev.EventName,
			EventTime:    ev.EventTime.Unix(),
			EventSource:  ev.EventSource,
			ActionSource: actionSource,
			UserData: capi.UserData{
				Email:      ev.Email,
				Phone:      ev.Phone,
				IPAddress:  ev.IPAddress,
				UserAgent:  ev.UserAgent,
				Country:    ev.Country,
				FBPIDCookieID: ev.FBPID,
				FBCookieID:    ev.FBCID,
				ExternalID:  ev.LeadID,
			},
			CustomData: capi.CustomData{
				Value:    ev.OrderValue,
				Currency: ev.Currency,
				OrderID:  ev.DealID,
			},
		}},
	}

	results, err := s.capi.SendEvents(context.Background(), cfg.AccessToken, cfg.PixelID, payload, cfg.TestEventCode != "")
	if err != nil {
		return "", err
	}

	if len(results) > 0 {
		return results[0].FBEventID, nil
	}
	return "", nil
}

// --- CRM Event Bridge (receives events from CRM/NATS) ---

type CRMBridgeRequest struct {
	CRMEventType string            `json:"crm_event_type"`
	TenantID     string            `json:"tenant_id"`
	Email        string            `json:"email,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	LeadID       string            `json:"lead_id,omitempty"`
	DealID       string            `json:"deal_id,omitempty"`
	DealValue    float64           `json:"deal_value,omitempty"`
	Currency     string            `json:"currency,omitempty"`
	IPAddress    string            `json:"ip_address,omitempty"`
	UserAgent    string            `json:"user_agent,omitempty"`
	CustomData   map[string]string `json:"custom_data,omitempty"`
}

func (s *Server) CRMBridge(c echo.Context) error {
	var req CRMBridgeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	// Look up conversion mapping
	mappings, err := s.repo.GetActiveMappings(context.Background(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	capiEventName := req.CRMEventType
	for _, m := range mappings {
		if m.CRMEventType == req.CRMEventType {
			capiEventName = m.CAPIEventName
			break
		}
	}

	// Default mapping if no custom mapping exists
	eventName := mapCRMEvents(capiEventName)

	// Extract value from mapped field
	orderValue := req.DealValue
	if orderValue == 0 {
		if val, ok := req.CustomData["value"]; ok {
			if v, err := parseFloat(val); err == nil {
				orderValue = v
			}
		}
	}

	sendReq := SendEventRequest{
		EventName:  eventName,
		Email:      req.Email,
		Phone:      req.Phone,
		LeadID:     req.LeadID,
		DealID:     req.DealID,
		OrderValue: orderValue,
		Currency:   req.Currency,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
	}

	return s.processSend(c, tenantID, sendReq)
}

func (s *Server) processSend(c echo.Context, tenantID uuid.UUID, req SendEventRequest) error {
	cfg, err := s.repo.GetCAPIConfig(context.Background(), tenantID)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "CAPI not configured"})
	}

	ev := &models.CAPIEvent{
		ID:          uuid.New(),
		TenantID:    tenantID,
		EventID:     uuid.New().String(),
		EventName:   req.EventName,
		EventTime:   time.Now(),
		EventSource: "crm",
		Email:       req.Email,
		Phone:       req.Phone,
		IPAddress:   req.IPAddress,
		UserAgent:   req.UserAgent,
		LeadID:      req.LeadID,
		DealID:      req.DealID,
		OrderValue:  req.OrderValue,
		Currency:    req.Currency,
		CustomData:  req.CustomData,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	_ = s.repo.CreateCAPIEvent(context.Background(), ev)

	fbEventID, err := s.sendToFacebook(cfg, ev)
	if err != nil {
		_ = s.repo.UpdateCAPIEventStatus(context.Background(), ev.ID, "failed", "", err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	_ = s.repo.UpdateCAPIEventStatus(context.Background(), ev.ID, "sent", fbEventID, "")

	return c.JSON(http.StatusOK, map[string]string{"status": "sent", "fb_event_id": fbEventID})
}

func mapCRMEvents(crmEvent string) string {
	switch strings.ToLower(crmEvent) {
	case "deal_won", "deal_closed_won":
		return "Purchase"
	case "lead_created", "lead_new":
		return "Lead"
	case "contact", "contact_created":
		return "Contact"
	case "page_view", "landing_view":
		return "ViewContent"
	case "form_submit", "form_filled":
		return "CompleteRegistration"
	case "add_to_cart":
		return "AddToCart"
	case "checkout":
		return "InitiateCheckout"
	case "subscribe":
		return "Subscribe"
	default:
		return crmEvent
	}
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
