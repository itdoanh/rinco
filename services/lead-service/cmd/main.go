// Package main — RINCO Lead Service entrypoint.
//
// Lead Service handles lead capture, scoring, assignment, conversion.
// Built on Echo + pgx/v5 + go-redis + NATS, runs embedded SQL migrations
// on startup, and participates in RINCO multi-tenant pool via RLS GUCs.
package main

import (
	"context"
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

	leadhdl "github.com/itdoanh/rinco/services/lead-service/internal/handler"
	leadnats "github.com/itdoanh/rinco/services/lead-service/internal/nats"
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
	nats *leadnats.Client
	h    *leadhdl.Server
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
	leadsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lead_service_leads_created_total", Help: "Leads created",
	}, []string{"source"})
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
		Password: cfg.ValkeyPwd,
		DB:       cfg.ValkeyDB,
		PoolSize: 20,
	}
	rdb = redis.NewClient(rdbOpts)
	if err := rdb.Ping(rootCtx).Err(); err != nil {
		logger.Warn("valkey ping failed; cache disabled", slog.String("error", err.Error()))
	} else {
		defer func() { _ = rdb.Close() }()
	}

	// NATS
	natsClient, err := leadnats.NewClient(cfg.NATSURL)
	if err != nil {
		logger.Warn("nats connect failed; events disabled", slog.String("error", err.Error()))
	}
	if natsClient != nil {
		defer natsClient.Close()
	}

	h := leadhdl.NewServer(pool, &leadhdl.RedisClient{
		Addr:     cfg.ValkeyAddr,
		Password: cfg.ValkeyPwd,
		DB:       cfg.ValkeyDB,
	}, natsClient, cfg.ScoringURL)

	srv := &server{
		pool: pool,
		rdb:  rdb,
		nats: natsClient,
		h:    h,
	}

	e := newEcho(srv)
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

type config struct {
	Env         string
	HTTPAddr    string
	DatabaseURL string
	ValkeyAddr  string
	ValkeyPwd   string
	ValkeyDB    int
	NATSURL     string
	OTLP        string
	ScoringURL  string
	CAPIURL     string
}

func loadConfig() *config {
	return &config{
		Env:         platform.Getenv("ENV", "development"),
		HTTPAddr:    platform.Getenv("LEAD_HTTP_ADDR", ":8083"),
		DatabaseURL: os.Getenv("LEAD_DATABASE_URL"),
		ValkeyAddr:  platform.Getenv("LEAD_VALKEY_URL", "localhost:6379"),
		ValkeyPwd:   platform.Getenv("LEAD_VALKEY_PASSWORD", "rinco_dev_password"),
		ValkeyDB:    platform.GetenvInt("LEAD_VALKEY_DB", 0),
		NATSURL:     os.Getenv("LEAD_NATS_URL"),
		OTLP:        os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		ScoringURL:  os.Getenv("LEAD_SCORING_URL"),
		CAPIURL:     os.Getenv("LEAD_CAPI_WORKER_URL"),
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
		{"0001_lead_sources", migration0001},
		{"0002_pipelines", migration0002},
		{"0003_leads", migration0003},
		{"0004_lead_notes_activities", migration0004},
	}
}

// =============================================================================
// Echo setup
// =============================================================================

func newEcho(srv *server) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic recovered", slog.Any("panic", r))
					_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
				}
			}()
			return next(c)
		}
	})

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start).Seconds()
			route := c.Path()
			if route == "" {
				route = c.Request().URL.Path
			}
			httpReqs.WithLabelValues(c.Request().Method, route, "200").Inc()
			httpDur.WithLabelValues(c.Request().Method, route).Observe(latency)
			slog.Info("request",
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", c.Response().Status),
				slog.Float64("latency_s", latency),
			)
			return err
		}
	})

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Access-Control-Allow-Origin", "*")
			c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-User-ID, X-Admin")
			if c.Request().Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	})

	e.Use(otelecho.Middleware(serviceName))

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
	})
	e.GET("/readyz", readyz(srv))
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/version", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"service": serviceName, "version": version})
	})

	v1 := e.Group("/v1")
	v1.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Request().Header.Get("X-Tenant-ID")
			if tenantID == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID header required"})
			}
			c.Set("tenant_id", tenantID)
			c.Set("user_id", c.Request().Header.Get("X-User-ID"))
			c.Set("is_admin", c.Request().Header.Get("X-Admin") == "true")
			return next(c)
		}
	})

	// Leads
	v1.GET("/leads", srv.h.ListLeads)
	v1.POST("/leads", srv.h.CreateLead)
	v1.GET("/leads/:id", srv.h.GetLead)
	v1.PUT("/leads/:id", srv.h.UpdateLead)
	v1.DELETE("/leads/:id", srv.h.DeleteLead)
	v1.POST("/leads/import", srv.h.ImportLeads)
	v1.GET("/leads/export", srv.h.ExportLeads)
	v1.POST("/leads/:id/assign", srv.h.AssignLead)
	v1.POST("/leads/:id/status", srv.h.UpdateLeadStatus)
	v1.POST("/leads/:id/score", srv.h.LeadScore)
	v1.POST("/leads/:id/convert", srv.h.ConvertLead)
	v1.GET("/leads/:id/notes", srv.h.ListLeadNotes)
	v1.POST("/leads/:id/notes", srv.h.CreateLeadNote)
	v1.GET("/leads/:id/timeline", srv.h.GetLeadTimeline)
	v1.GET("/leads/by-source/:source", srv.h.GetLeadsBySource)
	v1.GET("/leads/stream", srv.h.StreamLeads)
	v1.GET("/leads/stats", srv.h.GetLeadStats)

	// Lead Sources
	v1.GET("/lead-sources", srv.h.ListLeadSources)
	v1.POST("/lead-sources", srv.h.CreateLeadSource)

	// Pipelines
	v1.GET("/pipelines", srv.h.ListPipelines)
	v1.POST("/pipelines", srv.h.CreatePipeline)

	// Stages
	v1.GET("/stages", srv.h.ListStages)
	v1.POST("/stages", srv.h.CreateStage)

	// Connect-RPC endpoints
	rpc := e.Group("/internal/lead.v1.LeadService")
	rpc.POST("/CreateLead", srv.h.CreateLead)
	rpc.POST("/AssignLead", srv.h.AssignLead)
	rpc.POST("/UpdateLeadStatus", srv.h.UpdateLeadStatus)
	rpc.POST("/GetLead", srv.h.GetLead)
	rpc.POST("/StreamLeads", srv.h.StreamLeads)
	rpc.POST("/ConvertLead", srv.h.ConvertLead)
	rpc.POST("/ComputeScore", srv.h.LeadScore)

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
CREATE TABLE IF NOT EXISTS lead_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    utm_source TEXT,
    utm_medium TEXT,
    utm_campaign TEXT,
    utm_term TEXT,
    utm_content TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_lead_sources_tenant ON lead_sources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_lead_sources_active ON lead_sources(tenant_id, is_active) WHERE is_active = true;
`

const migration0002 = `
CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    color TEXT DEFAULT '#6366f1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_pipelines_tenant ON pipelines(tenant_id);
CREATE INDEX IF NOT EXISTS idx_pipelines_default ON pipelines(tenant_id, is_default) WHERE is_default = true;

CREATE TABLE IF NOT EXISTS pipeline_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    probability INT DEFAULT 50 CHECK (probability >= 0 AND probability <= 100),
    color TEXT DEFAULT '#6366f1',
    is_won BOOLEAN DEFAULT false,
    is_lost BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stages_pipeline ON pipeline_stages(pipeline_id, display_order);
`

const migration0003 = `
CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    source_id UUID REFERENCES lead_sources(id) ON DELETE SET NULL,
    contact_id UUID,
    owner_user_id UUID,
    pipeline_id UUID REFERENCES pipelines(id) ON DELETE SET NULL,
    stage_id UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    full_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    company_name TEXT,
    job_title TEXT,
    status TEXT NOT NULL DEFAULT 'new' CHECK (status IN (
        'new', 'contacted', 'qualified', 'proposal', 'won', 'lost', 'archived'
    )),
    score DECIMAL(5, 2) DEFAULT 0,
    score_tier TEXT CHECK (score_tier IN ('cold', 'warm', 'hot')),
    estimated_value DECIMAL(15, 2) DEFAULT 0,
    custom_fields JSONB DEFAULT '{}',
    utm JSONB DEFAULT '{}',
    ip INET,
    user_agent TEXT,
    referrer TEXT,
    fbclid TEXT,
    fbp TEXT,
    fbc TEXT,
    gclid TEXT,
    ttclid TEXT,
    landing_page TEXT,
    campaign_id TEXT,
    tags TEXT[] DEFAULT '{}',
    next_followup_at TIMESTAMPTZ,
    last_contacted_at TIMESTAMPTZ,
    converted_at TIMESTAMPTZ,
    lost_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_leads_tenant ON leads(tenant_id);
CREATE INDEX IF NOT EXISTS idx_leads_owner ON leads(tenant_id, owner_user_id);
CREATE INDEX IF NOT EXISTS idx_leads_source ON leads(tenant_id, source_id);
CREATE INDEX IF NOT EXISTS idx_leads_status ON leads(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_leads_pipeline ON leads(tenant_id, pipeline_id);
CREATE INDEX IF NOT EXISTS idx_leads_stage ON leads(tenant_id, stage_id);
CREATE INDEX IF NOT EXISTS idx_leads_score ON leads(tenant_id, score DESC) WHERE score IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_leads_email ON leads(tenant_id, email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_leads_phone ON leads(tenant_id, phone) WHERE phone IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_leads_tags ON leads USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_leads_utm ON leads USING GIN (utm);
CREATE INDEX IF NOT EXISTS idx_leads_custom_fields ON leads USING GIN (custom_fields);
CREATE INDEX IF NOT EXISTS idx_leads_followup ON leads(tenant_id, next_followup_at) WHERE next_followup_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_leads_created ON leads(tenant_id, created_at DESC);

ALTER TABLE leads ENABLE ROW LEVEL SECURITY;

CREATE POLICY leads_tenant_isolation ON leads
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (owner_user_id = current_setting('app.current_user_id', true)::UUID)
        )
    );
`

const migration0004 = `
CREATE TABLE IF NOT EXISTS lead_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    author_id UUID,
    body TEXT NOT NULL,
    is_pinned BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_lead_notes_tenant ON lead_notes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_lead_notes_lead ON lead_notes(lead_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lead_notes_body ON lead_notes USING gin(to_tsvector('simple', body));

CREATE TABLE IF NOT EXISTS lead_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('CALL', 'EMAIL', 'MEETING', 'NOTE', 'STATUS_CHANGE', 'ASSIGN', 'TASK', 'SCORE_UPDATE')),
    payload JSONB DEFAULT '{}',
    actor_id UUID,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lead_activities_tenant ON lead_activities(tenant_id);
CREATE INDEX IF NOT EXISTS idx_lead_activities_lead ON lead_activities(lead_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lead_activities_type ON lead_activities(tenant_id, type);

CREATE TABLE IF NOT EXISTS lead_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    from_user_id UUID,
    to_user_id UUID,
    reason TEXT,
    assigned_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lead_assignments_tenant ON lead_assignments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_lead_assignments_lead ON lead_assignments(lead_id, created_at DESC);

CREATE TABLE IF NOT EXISTS lead_stage_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    from_stage TEXT,
    to_stage TEXT NOT NULL,
    changed_by UUID,
    notes TEXT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lead_stage_history_lead ON lead_stage_history(lead_id, changed_at DESC);
`
