package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/itdoanh/rinco/services/analytics-service/internal/models"
	"github.com/itdoanh/rinco/services/analytics-service/internal/repository"
)

type Server struct {
	repo *repository.Repository
	log  *slog.Logger
}

func New(repo *repository.Repository) *Server {
	return &Server{repo: repo, log: slog.Default()}
}

// --- Tracking Endpoints ---

type TrackEventRequest struct {
	TenantID  string            `json:"tenant_id"`
	UserID    string            `json:"user_id,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	EventType string            `json:"event_type"`
	Source    string            `json:"source,omitempty"`
	Campaign  string            `json:"campaign,omitempty"`
	URL       string            `json:"url,omitempty"`
	Referrer  string            `json:"referrer,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	Country   string            `json:"country,omitempty"`
	Device    string            `json:"device,omitempty"`
	Browser   string            `json:"browser,omitempty"`
	OS        string            `json:"os,omitempty"`
	Value     float64           `json:"value,omitempty"`
	Props     map[string]string `json:"props,omitempty"`
}

func (s *Server) TrackEvent(c echo.Context) error {
	var req TrackEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	ev := &models.Event{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    req.UserID,
		SessionID: req.SessionID,
		EventType: req.EventType,
		Source:    req.Source,
		Campaign:  req.Campaign,
		URL:       req.URL,
		Referrer:  req.Referrer,
		UserAgent: req.UserAgent,
		Country:   req.Country,
		Device:    req.Device,
		Browser:   req.Browser,
		OS:        req.OS,
		Value:     req.Value,
		Props:     req.Props,
		Timestamp: time.Now(),
	}

	if err := s.repo.TrackEvent(context.Background(), ev); err != nil {
		s.log.Error("track event failed", "err", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to track event"})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "ok"})
}

type TrackBatchRequest struct {
	TenantID string                `json:"tenant_id"`
	Events   []TrackEventRequest  `json:"events"`
}

func (s *Server) TrackBatch(c echo.Context) error {
	var req TrackBatchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	events := make([]*models.Event, 0, len(req.Events))
	for _, e := range req.Events {
		events = append(events, &models.Event{
			ID:        uuid.New(),
			TenantID:  tenantID,
			UserID:    e.UserID,
			SessionID: e.SessionID,
			EventType: e.EventType,
			Source:    e.Source,
			Campaign:  e.Campaign,
			URL:       e.URL,
			Referrer:  e.Referrer,
			UserAgent: e.UserAgent,
			Country:   e.Country,
			Device:    e.Device,
			Browser:   e.Browser,
			OS:        e.OS,
			Value:     e.Value,
			Props:     e.Props,
			Timestamp: time.Now(),
		})
	}

	if err := s.repo.TrackEventsBatch(context.Background(), events); err != nil {
		s.log.Error("track batch failed", "err", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to track events"})
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"status": "ok",
		"count":  strconv.Itoa(len(events)),
	})
}

// --- Analytics Endpoints ---

type DashboardRequest struct {
	TenantID   string `query:"tenant_id"`
	From       string `query:"from"`
	To         string `query:"to"`
	Granularity string `query:"granularity"`
}

func (s *Server) GetDashboard(c echo.Context) error {
	tenantID, err := uuid.Parse(c.QueryParam("tenant_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	from := parseTime(c.QueryParam("from"), time.Now().AddDate(0, 0, -30))
	to := parseTime(c.QueryParam("to"), time.Now())
	granularity := c.QueryParam("granularity")
	if granularity == "" {
		granularity = "day"
	}

	ctx := context.Background()

	pageViews, err := s.repo.GetPageViews(ctx, tenantID, from, to, granularity)
	if err != nil {
		s.log.Warn("get page views failed", "err", err)
	}

	visitors, _ := s.repo.GetUniqueVisitors(ctx, tenantID, from, to)
	conversions, convRate, _ := s.repo.GetConversions(ctx, tenantID, from, to)
	sources, _ := s.repo.GetTopSources(ctx, tenantID, from, to, 5)
	pages, _ := s.repo.GetTopPages(ctx, tenantID, from, to, 5)
	trend, _ := s.repo.GetTrend(ctx, tenantID, "page_view", from, to, granularity)

	summary := models.DashboardSummary{
		TenantID:         tenantID,
		TotalPageViews:   0,
		UniqueVisitors:   visitors,
		TotalConversions: conversions,
		ConversionRate:   convRate,
		TopSources:       sources,
		TopPages:         pages,
		Trend:            trend,
	}

	for _, p := range pageViews {
		summary.TotalPageViews += int64(p.Value)
	}

	return c.JSON(http.StatusOK, summary)
}

func (s *Server) GetPageViews(c echo.Context) error {
	tenantID, err := uuid.Parse(c.QueryParam("tenant_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}
	from := parseTime(c.QueryParam("from"), time.Now().AddDate(0, 0, -30))
	to := parseTime(c.QueryParam("to"), time.Now())
	granularity := c.QueryParam("granularity")
	if granularity == "" {
		granularity = "day"
	}

	results, err := s.repo.GetPageViews(context.Background(), tenantID, from, to, granularity)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func (s *Server) GetTopSources(c echo.Context) error {
	tenantID, err := uuid.Parse(c.QueryParam("tenant_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}
	from := parseTime(c.QueryParam("from"), time.Now().AddDate(0, 0, -30))
	to := parseTime(c.QueryParam("to"), time.Now())
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 10
	}

	results, err := s.repo.GetTopSources(context.Background(), tenantID, from, to, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func (s *Server) GetTopPages(c echo.Context) error {
	tenantID, err := uuid.Parse(c.QueryParam("tenant_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}
	from := parseTime(c.QueryParam("from"), time.Now().AddDate(0, 0, -30))
	to := parseTime(c.QueryParam("to"), time.Now())
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 10
	}

	results, err := s.repo.GetTopPages(context.Background(), tenantID, from, to, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func (s *Server) GetTrend(c echo.Context) error {
	tenantID, err := uuid.Parse(c.QueryParam("tenant_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}
	eventType := c.QueryParam("event_type")
	if eventType == "" {
		eventType = "page_view"
	}
	from := parseTime(c.QueryParam("from"), time.Now().AddDate(0, 0, -30))
	to := parseTime(c.QueryParam("to"), time.Now())
	granularity := c.QueryParam("granularity")
	if granularity == "" {
		granularity = "day"
	}

	results, err := s.repo.GetTrend(context.Background(), tenantID, eventType, from, to, granularity)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func parseTime(s string, fallback time.Time) time.Time {
	if s == "" {
		return fallback
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fallback
	}
	return t
}

// --- Admin: raw event export ---

type ExportRequest struct {
	TenantID string   `json:"tenant_id"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Types    []string `json:"types,omitempty"`
}

func (s *Server) ExportEvents(c echo.Context) error {
	var req ExportRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	// Admin-only: return JSON export of events (paginated)
	return c.JSON(http.StatusOK, map[string]string{"status": "export not implemented in stub"})
}

// getPayload safely extracts JSON payload from request
func getPayload(c echo.Context) map[string]any {
	var payload map[string]any
	json.NewDecoder(c.Request().Body).Decode(&payload)
	return payload
}

func parseTimeOrNow(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Now()
}
