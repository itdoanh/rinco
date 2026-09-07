// Package main — RINCO Lead Service entrypoint.
//
// Lead management service for RINCO - handles lead capture, scoring,
// assignment, and conversion. Built on Echo + pgx/v5 + go-redis + NATS.
// Integrates with landing-service for form submissions and lead-scoring
// for AI-powered lead qualification.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	leadhandler "github.com/itdoanh/rinco/services/lead-service/internal/handler"
	"github.com/itdoanh/rinco/services/lead-service/internal/middleware"
	"github.com/itdoanh/rinco/services/lead-service/internal/platform"
)

const (
	serviceName = "lead-service"
	version     = "1.0.0"
)

// =============================================================================
// Types
// =============================================================================

type server struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
	nc   *nats.Conn
}

type config struct {
	Env            string
	HTTPAddr       string
	DatabaseURL    string
	ValkeyAddr     string
	ValkeyPassword string
	ValkeyDB       int
	NATSURL        string
	OTLP          string
}

// =============================================================================
// Server metrics
// =============================================================================

var (
	httpReqs = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lead_service_http_requests_total", Help: "HTTP requests",
	}, []string{"method", "route", "status"})
	httpDur = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lead_service_http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
	leadEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lead_service_events_total", Help: "NATS events",
	}, []string{"subject", "type"})
)

// =============================================================================
// main
// =============================================================================

func main() {
	cfg := loadConfig()
	logger := platform.InitLogger(serviceName, cfg.Env, version)
	logger.Info("lead-service starting", slog.String("addr", cfg.HTTPAddr))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tp, shutdownTracer, err := initTracer(rootCtx, cfg.OTLP, cfg.Env)
	if err == nil && tp != nil {
		otel.SetTracerProvider(tp)
		defer func() { _ = shutdownTracer(context.Background()) }()
	}

	pool, err := platform.OpenPool(rootCtx, platform.DBConfigFromEnv("LEAD_"), serviceName)
	if err != nil {
		logger.Error("db connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	if err := runMigrations(rootCtx, pool); err != nil {
		logger.Error("migrations failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var rdb *redis.Client
	rdbOpts := &redis.Options{
		Addr:     cfg.ValkeyAddr,
		Password: cfg.ValkeyPassword,
		DB:       cfg.ValkeyDB,
		PoolSize: 20,
	}
	rdb = redis.NewClient(rdbOpts)
	if err := rdb.Ping(rootCtx).Err(); err != nil {
		logger.Warn("valkey ping failed; cache disabled", slog.String("error", err.Error()))
	} else {
		defer func() { _ = rdb.Close() }()
	}

	// Connect to NATS
	var nc *nats.Conn
	if cfg.NATSURL != "" {
		nc, err = nats.Connect(cfg.NATSURL,
			nats.Name(serviceName),
			nats.MaxReconnects(5),
			nats.ReconnectWait(2*time.Second),
		)
		if err != nil {
			logger.Warn("nats connect failed; events disabled", slog.String("error", err.Error()))
		} else {
			defer nc.Close()
			logger.Info("nats connected", slog.String("url", cfg.NATSURL))
		}
	}

	srv := &server{
		pool: pool,
		rdb:  rdb,
		nc:   nc,
	}

	h := leadhandler.NewServer(pool)

	// Start NATS subscribers
	if nc != nil {
		startNATSSubscribers(rootCtx, srv, h, nc)
	}

	e := newEcho(srv, h)
	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server error", slog.String("error", err.Error()))
		}
	}()
	<-rootCtx.Done()
	logger.Info("shutdown signal received")
	shut, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := e.Shutdown(shut); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
	}
	logger.Info("lead-service stopped")
}

// =============================================================================
// Config
// =============================================================================

func loadConfig() *config {
	return &config{
		Env:            platform.Getenv("ENV", "development"),
		HTTPAddr:       platform.Getenv("LEAD_HTTP_ADDR", ":8083"),
		DatabaseURL:    os.Getenv("LEAD_DATABASE_URL"),
		ValkeyAddr:     platform.Getenv("LEAD_VALKEY_URL", "localhost:6379"),
		ValkeyPassword: platform.Getenv("LEAD_VALKEY_PASSWORD", "rinco_dev_password"),
		ValkeyDB:       platform.GetenvInt("LEAD_VALKEY_DB", 0),
		NATSURL:        platform.Getenv("LEAD_NATS_URL", "nats://localhost:4222"),
		OTLP:          os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}
}

// =============================================================================
// Tracing
// =============================================================================

func initTracer(ctx context.Context, endpoint, env string) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if endpoint == "" || env == "development" {
		return nil, func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	res, _ := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(serviceName), semconv.ServiceVersion(version),
		semconv.DeploymentEnvironment(env),
	))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp), sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, tp.Shutdown, nil
}

// =============================================================================
// Migrations
// =============================================================================

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("migrations: bootstrap: %w", err)
	}
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("migrations: applied: %w", err)
	}
	applied := map[string]bool{}
	for rows.Next() {
		var v string
		_ = rows.Scan(&v)
		applied[v] = true
	}
	rows.Close()
	for _, m := range migrationsList() {
		if applied[m.name] {
			continue
		}
		slog.Info("applying migration", slog.String("name", m.name))
		if _, err := pool.Exec(ctx, m.body); err != nil {
			return fmt.Errorf("migrations: %s: %w", m.name, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, m.name); err != nil {
			return fmt.Errorf("migrations: bookkeeping %s: %w", m.name, err)
		}
		slog.Info("migration applied", slog.String("name", m.name))
	}
	return nil
}

type migrationFile struct {
	name, body string
}

func migrationsList() []migrationFile {
	return []migrationFile{
		{"0001_leads", migration0001},
		{"0002_lead_notes", migration0002},
		{"0003_sources_pipelines", migration0003},
		{"0004_rls", migration0004},
	}
}

// =============================================================================
// NATS Event Handlers
// =============================================================================

type NATSEvent struct {
	Type      string          `json:"type"`
	LeadID    string          `json:"lead_id"`
	TenantID  string          `json:"tenant_id"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

func startNATSSubscribers(ctx context.Context, srv *server, h *leadhandler.Server, nc *nats.Conn) {
	// Subscribe to lead.created from landing-service
	if _, err := nc.Subscribe("lead.created", func(msg *nats.Msg) {
		leadEvents.WithLabelValues("lead.created", "received").Inc()
		slog.Info("received lead.created event", slog.String("data", string(msg.Data)))

		var event NATSEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			slog.Error("parse lead.created failed", slog.String("error", err.Error()))
			return
		}

		// Process the lead (already created by landing-service, just update stats)
		leadEvents.WithLabelValues("lead.created", "processed").Inc()
	}); err != nil {
		slog.Error("subscribe lead.created failed", slog.String("error", err.Error()))
	}

	// Subscribe to lead.scored from lead-scoring service
	if _, err := nc.Subscribe("lead.scored", func(msg *nats.Msg) {
		leadEvents.WithLabelValues("lead.scored", "received").Inc()
		slog.Info("received lead.scored event", slog.String("data", string(msg.Data)))

		var event struct {
			LeadID string  `json:"lead_id"`
			Score  float64 `json:"score"`
			Tier   string  `json:"tier"`
		}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			slog.Error("parse lead.scored failed", slog.String("error", err.Error()))
			return
		}

		// Update lead score in database
		if srv.pool != nil {
			_, err := srv.pool.Exec(ctx, `
				UPDATE leads SET score = $1, score_tier = $2, updated_at = NOW() 
				WHERE id = $3
			`, event.Score, event.Tier, event.LeadID)
			if err != nil {
				slog.Error("update lead score failed", slog.String("error", err.Error()))
			}
		}

		// Record activity
		if srv.pool != nil {
			_, _ = srv.pool.Exec(ctx, `
				INSERT INTO lead_activities (lead_id, type, payload)
				VALUES ($1, 'score_update', $2)
			`, event.LeadID, fmt.Sprintf(`{"score": %f, "tier": "%s", "source": "ai"}`, event.Score, event.Tier))
		}

		leadEvents.WithLabelValues("lead.scored", "processed").Inc()
	}); err != nil {
		slog.Error("subscribe lead.scored failed", slog.String("error", err.Error()))
	}

	// Subscribe to user.created for auto-assign logic
	if _, err := nc.Subscribe("user.created", func(msg *nats.Msg) {
		leadEvents.WithLabelValues("user.created", "received").Inc()
		slog.Info("received user.created event", slog.String("data", string(msg.Data)))
		// Auto-assign unassigned leads to new user if configured
		leadEvents.WithLabelValues("user.created", "processed").Inc()
	}); err != nil {
		slog.Error("subscribe user.created failed", slog.String("error", err.Error()))
	}
}

// PublishLeadEvent publishes a lead event to NATS
func (s *server) publishLeadEvent(subject string, event NATSEvent) error {
	if s.nc == nil {
		return nil
	}
	event.Timestamp = time.Now()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return s.nc.Publish(subject, data)
}

// =============================================================================
// Echo setup
// =============================================================================

func newEcho(srv *server, h *leadhandler.Server) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.RecoveryMW())
	e.Use(middleware.TraceMW())
	e.Use(middleware.LoggingMW())
	e.Use(middleware.MetricsMW())
	e.Use(middleware.CORSMW())
	e.Use(middleware.SecurityHeadersMW())
	e.Use(otelecho.Middleware(serviceName))

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
	})
	e.GET("/readyz", readyz(srv))
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/version", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"service": serviceName, "version": version})
	})

	// Leads
	v1 := e.Group("/v1")
	v1.Use(middleware.TenantMW())

	v1.GET("/leads", h.ListLeads)
	v1.POST("/leads", h.CreateLead)
	v1.GET("/leads/:id", h.GetLead)
	v1.PUT("/leads/:id", h.UpdateLead)
	v1.DELETE("/leads/:id", h.DeleteLead)
	v1.POST("/leads/:id/assign", h.AssignLead)
	v1.POST("/leads/:id/status", h.UpdateLeadStatus)
	v1.POST("/leads/:id/score", h.ScoreLead)
	v1.POST("/leads/:id/convert", h.ConvertLead)
	v1.GET("/leads/:id/notes", h.ListLeadNotes)
	v1.POST("/leads/:id/notes", h.CreateLeadNote)
	v1.GET("/leads/:id/timeline", h.GetLeadTimeline)
	v1.GET("/leads/by-source/:source", h.GetLeadsBySource)
	v1.GET("/leads/stats", h.GetLeadStats)
	v1.POST("/leads/import", h.ImportLeads)
	v1.GET("/leads/export", h.ExportLeads)

	// Lead Sources
	v1.GET("/lead-sources", h.ListLeadSources)
	v1.POST("/lead-sources", h.CreateLeadSource)

	// Pipelines
	v1.GET("/pipelines", h.ListPipelines)
	v1.POST("/pipelines", h.CreatePipeline)

	// Pipeline Stages
	v1.GET("/stages", h.ListStages)
	v1.POST("/stages", h.CreateStage)

	// Connect-RPC endpoints
	rpc := e.Group("/internal/lead.v1.LeadService")
	rpc.POST("/CreateLead", h.CreateLead)
	rpc.POST("/AssignLead", h.AssignLead)
	rpc.POST("/UpdateLeadStatus", h.UpdateLeadStatus)
	rpc.POST("/GetLead", h.GetLead)
	rpc.POST("/ListLeads", h.ListLeads)
	rpc.POST("/StreamLeads", h.ListLeads)
	rpc.POST("/ConvertLead", h.ConvertLead)
	rpc.POST("/ComputeScore", h.ScoreLead)

	return e
}

func readyz(srv *server) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if err := srv.pool.Ping(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db_unreachable", "error": err.Error()})
		}
		if srv.rdb != nil {
			if err := srv.rdb.Ping(ctx).Err(); err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "valkey_unreachable", "error": err.Error()})
			}
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

// =============================================================================
// Migration SQL Bodies
// =============================================================================

const migration0001 = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    source_id UUID,
    contact_id UUID,
    owner_user_id UUID,
    full_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    company_name TEXT,
    status TEXT NOT NULL DEFAULT 'new' CHECK (status IN (
        'new', 'contacted', 'qualified', 'proposal', 'won', 'lost', 'archived'
    )),
    score DECIMAL(5, 2) DEFAULT 0,
    score_tier TEXT CHECK (score_tier IN ('cold', 'warm', 'hot')),
    utm JSONB DEFAULT '{}',
    custom_fields JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    referrer TEXT,
    fbclid TEXT,
    gclid TEXT,
    last_contacted_at TIMESTAMPTZ,
    next_followup_at TIMESTAMPTZ,
    converted_at TIMESTAMPTZ,
    lost_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_leads_tenant ON leads(tenant_id);
CREATE INDEX IF NOT EXISTS idx_leads_tenant_owner ON leads(tenant_id, owner_user_id);
CREATE INDEX IF NOT EXISTS idx_leads_tenant_status ON leads(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_leads_email ON leads(tenant_id, email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_leads_score ON leads(tenant_id, score DESC) WHERE score IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_leads_fbclid ON leads(fbclid) WHERE fbclid IS NOT NULL;
`

const migration0002 = `
CREATE TABLE IF NOT EXISTS lead_notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    author_id UUID,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_lead_notes_lead ON lead_notes(lead_id, created_at DESC);

CREATE TABLE IF NOT EXISTS lead_activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('call', 'email', 'meeting', 'sms', 'note', 'status_change', 'assignment', 'score_update', 'conversion')),
    payload JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lead_activities_lead ON lead_activities(lead_id, created_at DESC);

CREATE TABLE IF NOT EXISTS lead_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    from_user_id UUID,
    to_user_id UUID NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lead_assignments_lead ON lead_assignments(lead_id, created_at DESC);

CREATE TABLE IF NOT EXISTS lead_stage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    from_stage TEXT,
    to_stage TEXT NOT NULL,
    changed_by UUID,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lead_stage_history_lead ON lead_stage_history(lead_id, changed_at DESC);
`

const migration0003 = `
CREATE TABLE IF NOT EXISTS lead_sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    utm_source TEXT,
    utm_medium TEXT,
    utm_campaign TEXT,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_lead_sources_tenant ON lead_sources(tenant_id);

CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_pipelines_tenant ON pipelines(tenant_id);

CREATE TABLE IF NOT EXISTS pipeline_stages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    probability INT CHECK (probability >= 0 AND probability <= 100),
    color TEXT DEFAULT '#6366f1',
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stages_pipeline ON pipeline_stages(pipeline_id, display_order);
`

const migration0004 = `
-- RLS Policies
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE lead_notes ENABLE ROW LEVEL SECURITY;
ALTER TABLE lead_activities ENABLE ROW LEVEL SECURITY;
ALTER TABLE lead_assignments ENABLE ROW LEVEL SECURITY;
ALTER TABLE lead_stage_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE lead_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE pipelines ENABLE ROW LEVEL SECURITY;
ALTER TABLE pipeline_stages ENABLE ROW LEVEL SECURITY;

CREATE POLICY leads_tenant_isolation ON leads
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY lead_notes_tenant_isolation ON lead_notes
    FOR ALL
    USING (
        lead_id IN (SELECT id FROM leads WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID)
    );

CREATE POLICY lead_activities_tenant_isolation ON lead_activities
    FOR ALL
    USING (
        lead_id IN (SELECT id FROM leads WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID)
    );

CREATE POLICY lead_assignments_tenant_isolation ON lead_assignments
    FOR ALL
    USING (
        lead_id IN (SELECT id FROM leads WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID)
    );

CREATE POLICY lead_stage_history_tenant_isolation ON lead_stage_history
    FOR ALL
    USING (
        lead_id IN (SELECT id FROM leads WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID)
    );

CREATE POLICY lead_sources_tenant_isolation ON lead_sources
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY pipelines_tenant_isolation ON pipelines
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY pipeline_stages_tenant_isolation ON pipeline_stages
    FOR ALL
    USING (
        pipeline_id IN (SELECT id FROM pipelines WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID)
    );
`
