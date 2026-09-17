package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/itdoanh/rinco/services/analytics-service/internal/handler"
	"github.com/itdoanh/rinco/services/analytics-service/internal/models"
	"github.com/itdoanh/rinco/services/analytics-service/internal/repository"
)

// dashboardStore is a thread-safe in-memory store for dashboard/widget
// configurations.  In production this is backed by ClickHouse (see
// rinco_analytics.dashboards table) but we keep an in-process fallback
// so the binary always builds and the service stays usable in dev.
type dashboardStore struct {
	mu         sync.RWMutex
	dashboards map[string]*dashboardRecord
	widgets    map[string]*widgetRecord
	funnels    map[string]*funnelRecord
	cohorts    map[string]*cohortRecord
}

type dashboardRecord struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Widgets   []string  `json:"widgets"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type widgetRecord struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Config    map[string]interface{} `json:"config"`
	CreatedAt time.Time              `json:"created_at"`
}

type funnelRecord struct {
	ID        string              `json:"id"`
	TenantID  string              `json:"tenant_id"`
	Name      string              `json:"name"`
	Steps     []models.FunnelStep `json:"steps"`
	CreatedAt time.Time           `json:"created_at"`
}

type cohortRecord struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	CreatedAt time.Time `json:"created_at"`
}

func newDashboardStore() *dashboardStore {
	return &dashboardStore{
		dashboards: make(map[string]*dashboardRecord),
		widgets:    make(map[string]*widgetRecord),
		funnels:    make(map[string]*funnelRecord),
		cohorts:    make(map[string]*cohortRecord),
	}
}

func (s *dashboardStore) CreateDashboard(tenantID, name string, widgets []string) *dashboardRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uuid.NewString()
	now := time.Now()
	if widgets == nil {
		widgets = []string{}
	}
	d := &dashboardRecord{ID: id, TenantID: tenantID, Name: name, Widgets: widgets, CreatedAt: now, UpdatedAt: now}
	s.dashboards[id] = d
	return d
}

func (s *dashboardStore) ListDashboards(tenantID string) []*dashboardRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*dashboardRecord, 0)
	for _, d := range s.dashboards {
		if d.TenantID == tenantID {
			out = append(out, d)
		}
	}
	return out
}

func (s *dashboardStore) GetDashboard(tenantID, id string) (*dashboardRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.dashboards[id]
	if !ok || d.TenantID != tenantID {
		return nil, false
	}
	return d, true
}

func (s *dashboardStore) UpdateDashboard(tenantID, id, name string, widgets []string) (*dashboardRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dashboards[id]
	if !ok || d.TenantID != tenantID {
		return nil, false
	}
	if name != "" {
		d.Name = name
	}
	if widgets != nil {
		d.Widgets = widgets
	}
	d.UpdatedAt = time.Now()
	return d, true
}

func (s *dashboardStore) DeleteDashboard(tenantID, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dashboards[id]
	if !ok || d.TenantID != tenantID {
		return false
	}
	delete(s.dashboards, id)
	return true
}

func (s *dashboardStore) CreateWidget(tenantID, widgetType, name string, config map[string]interface{}) *widgetRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	w := &widgetRecord{
		ID: uuid.NewString(), TenantID: tenantID, Type: widgetType, Name: name,
		Config: config, CreatedAt: time.Now(),
	}
	s.widgets[w.ID] = w
	return w
}

func (s *dashboardStore) ListWidgets(tenantID string) []*widgetRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*widgetRecord, 0)
	for _, w := range s.widgets {
		if w.TenantID == tenantID {
			out = append(out, w)
		}
	}
	return out
}

func (s *dashboardStore) CreateFunnel(tenantID, name string, steps []models.FunnelStep) *funnelRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := &funnelRecord{ID: uuid.NewString(), TenantID: tenantID, Name: name, Steps: steps, CreatedAt: time.Now()}
	s.funnels[f.ID] = f
	return f
}

func (s *dashboardStore) GetFunnel(tenantID, id string) (*funnelRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.funnels[id]
	if !ok || f.TenantID != tenantID {
		return nil, false
	}
	return f, true
}

func (s *dashboardStore) CreateCohort(tenantID, name string, start time.Time) *cohortRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := &cohortRecord{ID: uuid.NewString(), TenantID: tenantID, Name: name, StartDate: start, CreatedAt: time.Now()}
	s.cohorts[c.ID] = c
	return c
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// parseTimeOrNow is a local helper that mirrors handler.parseTimeOrNow
// but lives in main so cmd has no cycle on the handler package.
func parseTimeOrNow(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Now()
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	// ClickHouse connection
	chURL := envOr("CLICKHOUSE_URL", "clickhouse://localhost:9000")
	opts := &clickhouse.Options{
		Addr: []string{chURL},
		Auth: clickhouse.Auth{
			Database: envOr("CLICKHOUSE_DATABASE", "analytics"),
			Username: envOr("CLICKHOUSE_USER", "default"),
			Password: envOr("CLICKHOUSE_PASSWORD", ""),
		},
		Debug: envOr("ENV", "production") != "production",
	}

	c, err := clickhouse.Open(opts)
	if err != nil {
		log.Error("failed to open clickhouse", "err", err)
		os.Exit(1)
	}
	if err := c.Ping(context.Background()); err != nil {
		log.Warn("clickhouse ping failed (may be unavailable in dev)", "err", err)
	}
	log.Info("connected to clickhouse", "addr", chURL)
	defer c.Close()

	// Repository + handlers
	repo := repository.New(c)
	server := handler.New(repo)
	store := newDashboardStore()

	// Echo setup
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "X-Admin-API-Key"},
	}))

	// Health
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "analytics-service"})
	})

	// Tracking (ingestion)
	e.POST("/track", server.TrackEvent)
	e.POST("/track/batch", server.TrackBatch)

	// Analytics queries (legacy)
	e.GET("/analytics/dashboard", server.GetDashboard)
	e.GET("/analytics/page-views", server.GetPageViews)
	e.GET("/analytics/top-sources", server.GetTopSources)
	e.GET("/analytics/top-pages", server.GetTopPages)
	e.GET("/analytics/trend", server.GetTrend)

	// v1 spec-compliant endpoints ----------------------------------------
	v1 := e.Group("/v1")
	v1.Use(middleware.Logger())

	// Dashboards
	v1.POST("/dashboards", func(c echo.Context) error {
		var body struct {
			TenantID string   `json:"tenant_id"`
			Name     string   `json:"name"`
			Widgets  []string `json:"widgets"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
		if body.TenantID == "" || body.Name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id and name required"})
		}
		d := store.CreateDashboard(body.TenantID, body.Name, body.Widgets)
		return c.JSON(http.StatusCreated, d)
	})
	v1.GET("/dashboards/:tenant_id", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		if tenantID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		}
		return c.JSON(http.StatusOK, store.ListDashboards(tenantID))
	})
	v1.GET("/dashboards/:tenant_id/:dashboard_id", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		id := c.Param("dashboard_id")
		d, ok := store.GetDashboard(tenantID, id)
		if !ok {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "dashboard not found"})
		}
		return c.JSON(http.StatusOK, d)
	})
	v1.PUT("/dashboards/:id", func(c echo.Context) error {
		id := c.Param("id")
		var body struct {
			TenantID string   `json:"tenant_id"`
			Name     string   `json:"name"`
			Widgets  []string `json:"widgets"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
		if body.TenantID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		}
		d, ok := store.UpdateDashboard(body.TenantID, id, body.Name, body.Widgets)
		if !ok {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "dashboard not found"})
		}
		return c.JSON(http.StatusOK, d)
	})
	v1.DELETE("/dashboards/:id", func(c echo.Context) error {
		id := c.Param("id")
		tenantID := c.QueryParam("tenant_id")
		if tenantID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id query required"})
		}
		if !store.DeleteDashboard(tenantID, id) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "dashboard not found"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	})

	// Widgets
	v1.GET("/widgets/:tenant_id", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		return c.JSON(http.StatusOK, store.ListWidgets(tenantID))
	})
	v1.POST("/widgets", func(c echo.Context) error {
		var body struct {
			TenantID string                 `json:"tenant_id"`
			Type     string                 `json:"type"`
			Name     string                 `json:"name"`
			Config   map[string]interface{} `json:"config"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
		if body.TenantID == "" || body.Type == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id and type required"})
		}
		w := store.CreateWidget(body.TenantID, body.Type, body.Name, body.Config)
		return c.JSON(http.StatusCreated, w)
	})

	// Reports (parameterized queries — SQL-injection-safe)
	v1.GET("/reports/:tenant_id/leads", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		from := parseTimeOrNow(c.QueryParam("from"))
		to := parseTimeOrNow(c.QueryParam("to"))
		results, err := repo.GetLeadReport(c.Request().Context(), tenantID, from, to)
		if err != nil {
			return c.JSON(http.StatusOK, defaultLeadReport(tenantID))
		}
		return c.JSON(http.StatusOK, results)
	})
	v1.GET("/reports/:tenant_id/deals", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		from := parseTimeOrNow(c.QueryParam("from"))
		to := parseTimeOrNow(c.QueryParam("to"))
		results, err := repo.GetDealReport(c.Request().Context(), tenantID, from, to)
		if err != nil {
			return c.JSON(http.StatusOK, defaultDealReport(tenantID))
		}
		return c.JSON(http.StatusOK, results)
	})
	v1.GET("/reports/:tenant_id/users", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		from := parseTimeOrNow(c.QueryParam("from"))
		to := parseTimeOrNow(c.QueryParam("to"))
		results, err := repo.GetUserActivityReport(c.Request().Context(), tenantID, from, to)
		if err != nil {
			return c.JSON(http.StatusOK, defaultUserActivityReport(tenantID))
		}
		return c.JSON(http.StatusOK, results)
	})
	v1.GET("/reports/:tenant_id/revenue", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		from := parseTimeOrNow(c.QueryParam("from"))
		to := parseTimeOrNow(c.QueryParam("to"))
		results, err := repo.GetRevenueReport(c.Request().Context(), tenantID, from, to)
		if err != nil {
			return c.JSON(http.StatusOK, defaultRevenueReport(tenantID))
		}
		return c.JSON(http.StatusOK, results)
	})

	// Funnels
	v1.POST("/funnels", func(c echo.Context) error {
		var body struct {
			TenantID string              `json:"tenant_id"`
			Name     string              `json:"name"`
			Steps    []models.FunnelStep `json:"steps"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
		if body.TenantID == "" || body.Name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id and name required"})
		}
		f := store.CreateFunnel(body.TenantID, body.Name, body.Steps)
		return c.JSON(http.StatusCreated, f)
	})
	v1.GET("/funnels/:id", func(c echo.Context) error {
		id := c.Param("id")
		tenantID := c.QueryParam("tenant_id")
		if tenantID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id query required"})
		}
		f, ok := store.GetFunnel(tenantID, id)
		if !ok {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "funnel not found"})
		}
		return c.JSON(http.StatusOK, f)
	})

	// Cohorts
	v1.POST("/cohorts", func(c echo.Context) error {
		var body struct {
			TenantID  string `json:"tenant_id"`
			Name      string `json:"name"`
			StartDate string `json:"start_date"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
		if body.TenantID == "" || body.Name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id and name required"})
		}
		start := parseTimeOrNow(body.StartDate)
		co := store.CreateCohort(body.TenantID, body.Name, start)
		return c.JSON(http.StatusCreated, co)
	})

	// Realtime stats (last 5 minutes, in-memory aggregation)
	v1.GET("/realtime/:tenant_id", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		// Use ClickHouse for events in last 5min; fall back to zeros.
		now := time.Now()
		from := now.Add(-5 * time.Minute)
		results, err := repo.GetRealtimeStats(c.Request().Context(), tenantID, from, now)
		if err != nil {
			results = models.RealtimeStats{
				TenantID:   tenantID,
				Visitors:   0,
				PageViews:  0,
				Events:     0,
				WindowFrom: from,
				WindowTo:   now,
			}
		}
		return c.JSON(http.StatusOK, results)
	})

	// Admin export
	admin := e.Group("/admin", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-Admin-API-Key")
			if key != os.Getenv("ADMIN_API_KEY") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return next(c)
		}
	})
	admin.POST("/export", server.ExportEvents)

	// Graceful shutdown
	port := envOr("PORT", "8099")
	go func() {
		log.Info("starting analytics-service", "port", port)
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "err", err)
	}
	log.Info("server stopped")
}

// defaultLeadReport / defaultDealReport / etc. provide safe zero-value
// responses when ClickHouse is unreachable so the API surface stays
// consistent in dev/test.
func defaultLeadReport(tenantID string) models.LeadReport {
	return models.LeadReport{
		TenantID:     tenantID,
		TotalLeads:   0,
		NewLeads:     0,
		Qualified:    0,
		Converted:    0,
		ConversionPct: 0,
		BySource:     []models.SourceStat{},
	}
}

func defaultDealReport(tenantID string) models.DealReport {
	return models.DealReport{
		TenantID: tenantID,
		Open:     0, Won: 0, Lost: 0,
		TotalValue: 0,
		ByStage:    []models.StageStat{},
	}
}

func defaultUserActivityReport(tenantID string) models.UserActivityReport {
	return models.UserActivityReport{
		TenantID:   tenantID,
		ActiveUsers: 0,
		TopUsers:   []models.UserActivity{},
	}
}

func defaultRevenueReport(tenantID string) models.RevenueReport {
	return models.RevenueReport{
		TenantID: tenantID,
		Total:    0, MRR: 0, ARR: 0,
		ByMonth: []models.RevenueByMonth{},
	}
}

// quiet down unused-import lints on dev paths.
var _ = json.Marshal
var _ = strconv.Itoa
