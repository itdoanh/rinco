// Package main — RINCO CRM Service entrypoint.
//
// CRM service with LTREE-based organizational tree, CRUD for
// contacts, companies, deals, activities, notes, tags, and custom fields.
// Built on Echo + pgx/v5 + go-redis, runs embedded SQL migrations on
// startup, and participates in RINCO multi-tenant pool via RLS GUCs.
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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	crmhandler "github.com/itdoanh/rinco/services/crm-service/internal/handler"
	"github.com/itdoanh/rinco/services/crm-service/internal/middleware"
	"github.com/itdoanh/rinco/services/crm-service/internal/platform"
)

const (
	serviceName = "crm-service"
	version     = "1.0.0"
)

// =============================================================================
// Types
// =============================================================================

type server struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
}

type config struct {
	Env            string
	HTTPAddr       string
	DatabaseURL    string
	ValkeyAddr     string
	ValkeyPassword string
	ValkeyDB       int
	OTLP           string
}

// =============================================================================
// Server metrics (crm_service_http_requests_total and friends are registered
// in internal/middleware so the cmd package doesn't re-register them and
// trigger a duplicate-registration panic at init time.)
// =============================================================================

// =============================================================================
// main
// =============================================================================

func main() {
	cfg := loadConfig()
	logger := platform.InitLogger(serviceName, cfg.Env, version)
	logger.Info("crm-service starting", slog.String("addr", cfg.HTTPAddr))

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tp, shutdownTracer, err := initTracer(rootCtx, cfg.OTLP, cfg.Env)
	if err == nil && tp != nil {
		otel.SetTracerProvider(tp)
		defer func() { _ = shutdownTracer(context.Background()) }()
	}

	pool, err := platform.OpenPool(rootCtx, platform.DBConfigFromEnv("CRM_"), serviceName)
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

	srv := &server{
		pool: pool,
		rdb:  rdb,
	}

	h := crmhandler.NewServer(pool, &crmhandler.RedisClient{
		Addr:     cfg.ValkeyAddr,
		Password: cfg.ValkeyPassword,
		DB:       cfg.ValkeyDB,
	})

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
	logger.Info("crm-service stopped")
}

// =============================================================================
// Config
// =============================================================================

func loadConfig() *config {
	return &config{
		Env:            platform.Getenv("ENV", "development"),
		HTTPAddr:       platform.Getenv("CRM_HTTP_ADDR", ":8083"),
		DatabaseURL:    os.Getenv("CRM_DATABASE_URL"),
		ValkeyAddr:     platform.Getenv("CRM_VALKEY_URL", "localhost:6379"),
		ValkeyPassword: platform.Getenv("CRM_VALKEY_PASSWORD", "rinco_dev_password"),
		ValkeyDB:       platform.GetenvInt("CRM_VALKEY_DB", 0),
		OTLP:           os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
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
		{"0001_companies", migration0001},
		{"0002_contacts", migration0002},
		{"0003_deals", migration0003},
		{"0004_activities", migration0004},
		{"0005_notes", migration0005},
		{"0006_tags", migration0006},
		{"0007_custom_fields", migration0007},
		{"0008_users_tree", migration0008},
		{"0009_invite_links", migration0009},
		{"0010_rls", migration0010},
		{"0011_lead_pipeline", migration0011},
		{"0012_workflows", migration0012},
		{"0013_audit_log", migration0013},
		{"0014_notifications", migration0014},
	}
}

// =============================================================================
// Echo setup
// =============================================================================

func newEcho(srv *server, h *crmhandler.Server) *echo.Echo {
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

	// Companies
	v1 := e.Group("/v1")
	v1.Use(middleware.TenantMW())
	v1.GET("/companies", h.ListCompanies)
	v1.POST("/companies", h.CreateCompany)
	v1.GET("/companies/:id", h.GetCompany)
	v1.PUT("/companies/:id", h.UpdateCompany)
	v1.DELETE("/companies/:id", h.DeleteCompany)

	// Contacts
	v1.GET("/contacts", h.ListContacts)
	v1.POST("/contacts", h.CreateContact)
	v1.GET("/contacts/:id", h.GetContact)
	v1.PUT("/contacts/:id", h.UpdateContact)
	v1.DELETE("/contacts/:id", h.DeleteContact)

	// Deals
	v1.GET("/deals", h.ListDeals)
	v1.POST("/deals", h.CreateDeal)
	v1.GET("/deals/:id", h.GetDeal)
	v1.PUT("/deals/:id", h.UpdateDeal)
	v1.DELETE("/deals/:id", h.DeleteDeal)
	v1.POST("/deals/:id/stage", h.MoveDealStage)

	// Activities
	v1.GET("/activities", h.ListActivities)
	v1.POST("/activities", h.CreateActivity)
	v1.GET("/activities/:id", h.GetActivity)
	v1.PUT("/activities/:id", h.UpdateActivity)
	v1.DELETE("/activities/:id", h.DeleteActivity)
	v1.GET("/activities/timeline/:contact_id", h.GetContactTimeline)

	// Notes
	v1.GET("/notes", h.ListNotes)
	v1.POST("/notes", h.CreateNote)
	v1.GET("/notes/:id", h.GetNote)
	v1.PUT("/notes/:id", h.UpdateNote)
	v1.DELETE("/notes/:id", h.DeleteNote)

	// Tags
	v1.GET("/tags", h.ListTags)
	v1.POST("/tags", h.CreateTag)
	v1.POST("/contacts/:id/tags", h.AddTagToContact)

	// Custom Fields
	v1.GET("/custom-fields", h.ListCustomFields)
	v1.POST("/custom-fields", h.CreateCustomField)

	// Tree (LTREE)
	tree := e.Group("/v1/tree")
	tree.GET("/:tenant_id/users", h.ListTreeUsers)
	tree.POST("/:tenant_id/users", h.CreateTreeUser)
	tree.GET("/:tenant_id/users/:id", h.GetUserTree)
	tree.PUT("/:tenant_id/users/:id", h.UpdateTreeUser)
	tree.DELETE("/:tenant_id/users/:id", h.DeleteTreeUser)
	tree.POST("/:tenant_id/move", h.MoveSubtree)
	tree.GET("/:tenant_id/path/:user_id", h.GetUserPath)
	tree.GET("/:tenant_id/subordinates/:user_id", h.GetSubordinates)
	tree.GET("/:tenant_id/ancestors/:user_id", h.GetAncestors)
	tree.POST("/:tenant_id/invite-link", h.CreateInviteLink)
	tree.GET("/:tenant_id/users/:id/subordinates-count", h.GetSubordinatesCount)

	// Reports
	v1.GET("/reports/pipeline", h.GetPipelineReport)
	v1.GET("/reports/conversion", h.GetConversionReport)
	v1.GET("/reports/leaderboard", h.GetLeaderboard)

	// Connect-RPC endpoints (using standard HTTP POST for now)
	rpc := e.Group("/internal/crm.v1.CRMService")
	rpc.POST("/GetContact", h.GetContact)
	rpc.POST("/ListDeals", h.ListDeals)
	rpc.POST("/CreateActivity", h.CreateActivity)
	rpc.POST("/MoveSubtree", h.MoveSubtree)
	rpc.POST("/GetUserTree", h.GetUserTree)

	// Leads (new in loop 203)
	v1.POST("/leads", h.CreateLead)
	v1.GET("/leads", h.ListLeads)
	v1.GET("/leads/:id", h.GetLead)
	v1.PUT("/leads/:id", h.UpdateLead)
	v1.DELETE("/leads/:id", h.DeleteLead)
	v1.POST("/leads/:id/assign", h.AssignLead)
	v1.POST("/leads/:id/stage", h.MoveLeadStage)
	v1.POST("/leads/:id/convert", h.ConvertLead)
	v1.POST("/leads/bulk", h.BulkLeadImport)

	// Pipelines
	v1.POST("/pipelines", h.CreatePipeline)
	v1.GET("/pipelines", h.ListPipelines)
	v1.GET("/pipelines/:id", h.GetPipeline)

	// Workflows
	v1.POST("/workflows", h.CreateWorkflow)
	v1.GET("/workflows", h.ListWorkflows)
	v1.GET("/workflows/:id", h.GetWorkflow)
	v1.PUT("/workflows/:id", h.UpdateWorkflow)
	v1.DELETE("/workflows/:id", h.DeleteWorkflow)
	v1.POST("/workflows/:id/trigger", h.TriggerWorkflow)

	// Notifications
	v1.POST("/notifications/broadcast", h.BroadcastNotification)
	v1.GET("/notifications/inbox", h.NotificationInbox)
	v1.PATCH("/notifications/:id/read", h.MarkNotificationRead)
	v1.POST("/notifications/:id/dismiss", h.DismissNotification)

	// Audit
	v1.GET("/audit", h.ListAudit)

	// Sessions
	v1.GET("/sessions", h.ListSessions)
	v1.POST("/sessions/:id/revoke", h.RevokeSession)

	// Tags update/delete
	v1.PUT("/tags/:id", h.UpdateTag)
	v1.DELETE("/tags/:id", h.DeleteTag)

	// Custom Fields update/delete
	v1.PUT("/custom-fields/:id", h.UpdateCustomField)
	v1.DELETE("/custom-fields/:id", h.DeleteCustomField)

	// User promote/demote (sibling of move)
	tree.PUT("/:tenant_id/users/:id/promote", h.PromoteUser)
	tree.PUT("/:tenant_id/users/:id/demote", h.DemoteUser)

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
CREATE EXTENSION IF NOT EXISTS "ltree";

CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    industry TEXT,
    size TEXT CHECK (size IN ('startup', 'small', 'medium', 'large', 'enterprise')),
    website TEXT,
    phone TEXT,
    email TEXT,
    address TEXT,
    city TEXT,
    country TEXT DEFAULT 'Vietnam',
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_companies_tenant ON companies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_companies_name ON companies USING gin(to_tsvector('simple', name));
CREATE INDEX IF NOT EXISTS idx_companies_industry ON companies(tenant_id, industry);
`

const migration0002 = `
CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    job_title TEXT,
    department TEXT,
    owner_user_id UUID,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'lead')),
    source TEXT CHECK (source IN ('organic', 'referral', 'social', 'ads', 'direct', 'other')),
    custom_fields JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    last_contacted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_contacts_tenant ON contacts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_contacts_company ON contacts(tenant_id, company_id);
CREATE INDEX IF NOT EXISTS idx_contacts_owner ON contacts(tenant_id, owner_user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts(tenant_id, email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts USING gin(to_tsvector('simple', first_name || ' ' || last_name));
CREATE INDEX IF NOT EXISTS idx_contacts_status ON contacts(tenant_id, status);
`

const migration0003 = `
CREATE TABLE IF NOT EXISTS deals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    owner_user_id UUID,
    name TEXT NOT NULL,
    value DECIMAL(15, 2) DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'VND',
    stage TEXT NOT NULL DEFAULT 'prospecting' CHECK (stage IN (
        'prospecting', 'qualification', 'proposal', 'negotiation', 
        'won', 'lost', 'on_hold'
    )),
    probability INT CHECK (probability >= 0 AND probability <= 100),
    expected_close_date DATE,
    actual_close_date DATE,
    lost_reason TEXT,
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_deals_tenant ON deals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_deals_contact ON deals(tenant_id, contact_id);
CREATE INDEX IF NOT EXISTS idx_deals_owner ON deals(tenant_id, owner_user_id);
CREATE INDEX IF NOT EXISTS idx_deals_stage ON deals(tenant_id, stage);

CREATE TABLE IF NOT EXISTS deal_stage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    from_stage TEXT,
    to_stage TEXT NOT NULL,
    changed_by UUID,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_deal_stage_history_deal ON deal_stage_history(deal_id, changed_at DESC);
`

const migration0004 = `
CREATE TABLE IF NOT EXISTS activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    contact_id UUID REFERENCES contacts(id) ON DELETE CASCADE,
    deal_id UUID REFERENCES deals(id) ON DELETE SET NULL,
    type TEXT NOT NULL CHECK (type IN ('call', 'email', 'meeting', 'task', 'note')),
    subject TEXT NOT NULL,
    body TEXT,
    due_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    priority TEXT DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    reminder_at TIMESTAMPTZ,
    owner_user_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_activities_tenant ON activities(tenant_id);
CREATE INDEX IF NOT EXISTS idx_activities_contact ON activities(tenant_id, contact_id);
CREATE INDEX IF NOT EXISTS idx_activities_deal ON activities(tenant_id, deal_id);
CREATE INDEX IF NOT EXISTS idx_activities_type ON activities(tenant_id, type);
CREATE INDEX IF NOT EXISTS idx_activities_due ON activities(tenant_id, due_at) WHERE due_at IS NOT NULL;
`

const migration0005 = `
CREATE TABLE IF NOT EXISTS notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    contact_id UUID REFERENCES contacts(id) ON DELETE CASCADE,
    deal_id UUID REFERENCES deals(id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    author_id UUID,
    is_pinned BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_notes_tenant ON notes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_notes_contact ON notes(tenant_id, contact_id);
CREATE INDEX IF NOT EXISTS idx_notes_deal ON notes(tenant_id, deal_id);
CREATE INDEX IF NOT EXISTS idx_notes_body ON notes USING gin(to_tsvector('simple', body));
`

const migration0006 = `
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    color TEXT DEFAULT '#6366f1',
    description TEXT,
    usage_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_tags_tenant ON tags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(tenant_id, name);

CREATE TABLE IF NOT EXISTS contact_tags (
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (contact_id, tag_id)
);
`

const migration0007 = `
CREATE TABLE IF NOT EXISTS custom_fields (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('contact', 'company', 'deal', 'lead', 'activity')),
    name TEXT NOT NULL,
    field_key TEXT NOT NULL,
    field_type TEXT NOT NULL CHECK (field_type IN (
        'text', 'textarea', 'number', 'currency', 'date', 'datetime',
        'boolean', 'select', 'multiselect', 'email', 'phone', 'url'
    )),
    description TEXT,
    options JSONB DEFAULT '[]',
    default_value JSONB,
    validation_rules JSONB DEFAULT '{}',
    is_required BOOLEAN DEFAULT false,
    is_unique BOOLEAN DEFAULT false,
    display_order INT DEFAULT 0,
    width TEXT DEFAULT 'full' CHECK (width IN ('full', 'half', 'third')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, entity_type, field_key)
);

CREATE INDEX IF NOT EXISTS idx_custom_fields_tenant ON custom_fields(tenant_id);
CREATE INDEX IF NOT EXISTS idx_custom_fields_entity ON custom_fields(tenant_id, entity_type);
`

const migration0008 = `
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    email TEXT NOT NULL,
    full_name TEXT NOT NULL,
    phone TEXT,
    avatar_url TEXT,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN (
        'owner', 'admin', 'manager', 'team_lead', 'member', 'viewer'
    )),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'invited', 'pending')),
    path LTREE NOT NULL,
    depth INT NOT NULL DEFAULT 1,
    parent_id UUID,
    department TEXT,
    position TEXT,
    password_hash TEXT,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, email)
);

CREATE INDEX IF NOT EXISTS idx_users_path ON users USING GIST (path);
CREATE INDEX IF NOT EXISTS idx_users_path_btree ON users USING BTREE (path);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_parent ON users(tenant_id, parent_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(tenant_id, role);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(tenant_id, status);

CREATE MATERIALIZED VIEW IF NOT EXISTS users_with_depth AS
SELECT id, tenant_id, email, full_name, role, status, path, depth, parent_id, department
FROM users WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_uwd_id ON users_with_depth(id);
CREATE INDEX IF NOT EXISTS idx_uwd_tenant ON users_with_depth(tenant_id);
CREATE INDEX IF NOT EXISTS idx_uwd_path ON users_with_depth USING GIST (path);

-- Add foreign keys
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'contacts_owner_user_id_fkey') THEN
        ALTER TABLE contacts ADD CONSTRAINT contacts_owner_user_id_fkey REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'deals_owner_user_id_fkey') THEN
        ALTER TABLE deals ADD CONSTRAINT deals_owner_user_id_fkey REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'activities_owner_user_id_fkey') THEN
        ALTER TABLE activities ADD CONSTRAINT activities_owner_user_id_fkey REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'notes_author_id_fkey') THEN
        ALTER TABLE notes ADD CONSTRAINT notes_author_id_fkey REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'deal_stage_history_changed_by_fkey') THEN
        ALTER TABLE deal_stage_history ADD CONSTRAINT deal_stage_history_changed_by_fkey REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'users_parent_id_fkey') THEN
        ALTER TABLE users ADD CONSTRAINT users_parent_id_fkey REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;
`

const migration0009 = `
CREATE TABLE IF NOT EXISTS invite_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    parent_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_role TEXT NOT NULL CHECK (target_role IN (
        'admin', 'manager', 'team_lead', 'member', 'viewer'
    )),
    token TEXT NOT NULL UNIQUE,
    max_uses INT DEFAULT 1,
    current_uses INT DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_invite_links_token ON invite_links(token);
CREATE INDEX IF NOT EXISTS idx_invite_links_tenant ON invite_links(tenant_id);
CREATE INDEX IF NOT EXISTS idx_invite_links_expires ON invite_links(expires_at) WHERE revoked_at IS NULL AND expires_at > NOW();

CREATE TABLE IF NOT EXISTS invite_usage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invite_id UUID NOT NULL REFERENCES invite_links(id) ON DELETE CASCADE,
    used_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);
`

const migration0010 = `
-- Enhanced RLS Policies
CREATE OR REPLACE FUNCTION get_user_subtree_path(user_id UUID)
RETURNS LTREE AS $$
DECLARE
    user_path LTREE;
BEGIN
    SELECT u.path INTO user_path
    FROM users u
    WHERE u.id = user_id AND u.deleted_at IS NULL;
    RETURN user_path;
END;
$$ LANGUAGE plpgsql STABLE;

-- Contacts RLS
DROP POLICY IF EXISTS contacts_tenant_isolation ON contacts;
CREATE POLICY contacts_tenant_isolation ON contacts
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (owner_user_id = current_setting('app.current_user_id', true)::UUID)
            OR
            (owner_user_id IN (
                SELECT id FROM users 
                WHERE path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID)
            ))
        )
    );

-- Companies RLS
DROP POLICY IF EXISTS companies_tenant_isolation ON companies;
CREATE POLICY companies_tenant_isolation ON companies
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
    );

-- Deals RLS
DROP POLICY IF EXISTS deals_tenant_isolation ON deals;
CREATE POLICY deals_tenant_isolation ON deals
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (owner_user_id = current_setting('app.current_user_id', true)::UUID)
            OR
            (owner_user_id IN (
                SELECT id FROM users 
                WHERE path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID)
            ))
        )
    );

-- Activities RLS
DROP POLICY IF EXISTS activities_tenant_isolation ON activities;
CREATE POLICY activities_tenant_isolation ON activities
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (owner_user_id = current_setting('app.current_user_id', true)::UUID)
            OR
            (owner_user_id IN (
                SELECT id FROM users 
                WHERE path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID)
            ))
        )
    );

-- Notes RLS
DROP POLICY IF EXISTS notes_tenant_isolation ON notes;
CREATE POLICY notes_tenant_isolation ON notes
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (author_id = current_setting('app.current_user_id', true)::UUID)
        )
    );

-- Tags RLS
DROP POLICY IF EXISTS tags_tenant_isolation ON tags;
CREATE POLICY tags_tenant_isolation ON tags
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
    );

-- Custom Fields RLS
DROP POLICY IF EXISTS custom_fields_tenant_isolation ON custom_fields;
CREATE POLICY custom_fields_tenant_isolation ON custom_fields
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
    );

-- Users RLS
DROP POLICY IF EXISTS users_tenant_isolation ON users;
CREATE POLICY users_tenant_isolation ON users
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID))
        )
    );

-- Invite Links RLS
DROP POLICY IF EXISTS invite_links_tenant_isolation ON invite_links;
CREATE POLICY invite_links_tenant_isolation ON invite_links
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (current_setting('app.is_admin', true) = 'true')
    );

-- Refresh MV function
CREATE OR REPLACE FUNCTION refresh_users_materialized_view()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY users_with_depth;
END;
$$ LANGUAGE plpgsql;
`

// =============================================================================
// Migration 0011 (Lead pipeline)
// =============================================================================
const migration0011 = `
CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_pipelines_tenant ON pipelines(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_pipelines_default ON pipelines(tenant_id) WHERE is_default = true AND deleted_at IS NULL;
ALTER TABLE pipelines ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS pipelines_tenant_isolation ON pipelines;
CREATE POLICY pipelines_tenant_isolation ON pipelines
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            current_setting('app.is_admin', true) = 'true'
            OR EXISTS (
                SELECT 1 FROM users
                WHERE id = current_setting('app.current_user_id', true)::UUID
                AND tenant_id = pipelines.tenant_id
            )
        )
    );

CREATE TABLE IF NOT EXISTS pipeline_stages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    color TEXT DEFAULT '#6366f1',
    position INT NOT NULL DEFAULT 0,
    probability INT NOT NULL DEFAULT 50 CHECK (probability BETWEEN 0 AND 100),
    sla_hours INT,
    is_won BOOLEAN DEFAULT false,
    is_lost BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (pipeline_id, code)
);
CREATE INDEX IF NOT EXISTS idx_pipeline_stages_pipeline ON pipeline_stages(pipeline_id, position);
ALTER TABLE pipeline_stages ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS pipeline_stages_tenant_isolation ON pipeline_stages;
CREATE POLICY pipeline_stages_tenant_isolation ON pipeline_stages
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    pipeline_id UUID REFERENCES pipelines(id) ON DELETE SET NULL,
    current_stage_id UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    converted_contact_id UUID,
    converted_deal_id UUID,
    full_name TEXT,
    email TEXT,
    phone TEXT,
    company_name TEXT,
    source TEXT,
    source_detail TEXT,
    utm JSONB DEFAULT '{}'::jsonb,
    fbclid TEXT,
    fbp TEXT,
    fbc TEXT,
    ip_address INET,
    user_agent TEXT,
    score DECIMAL(5,2),
    estimated_value DECIMAL(15,2),
    currency TEXT DEFAULT 'VND',
    status TEXT NOT NULL DEFAULT 'NEW' CHECK (status IN (
        'NEW', 'CONTACTED', 'QUALIFIED', 'PROPOSAL', 'WON', 'LOST', 'ARCHIVED'
    )),
    lost_reason TEXT,
    custom_fields JSONB DEFAULT '{}'::jsonb,
    tags TEXT[] DEFAULT '{}',
    next_followup_at TIMESTAMPTZ,
    last_contacted_at TIMESTAMPTZ,
    converted_at TIMESTAMPTZ,
    assigned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_leads_tenant_status ON leads(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_tenant_owner ON leads(tenant_id, owner_user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_tenant_stage ON leads(tenant_id, current_stage_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_score ON leads(tenant_id, score DESC) WHERE score IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_followup ON leads(tenant_id, next_followup_at) WHERE next_followup_at IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_fbclid ON leads(tenant_id, fbclid) WHERE fbclid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_leads_email ON leads(tenant_id, email) WHERE email IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_custom_fields ON leads USING GIN(custom_fields);
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS leads_tenant_isolation ON leads;
CREATE POLICY leads_tenant_isolation ON leads
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            current_setting('app.is_admin', true) = 'true'
            OR owner_user_id = current_setting('app.current_user_id', true)::UUID
            OR owner_user_id IN (
                SELECT id FROM users
                WHERE tenant_id = leads.tenant_id
                AND deleted_at IS NULL
                AND path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID)
            )
        )
    );

CREATE TABLE IF NOT EXISTS lead_stage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    from_stage_id UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    to_stage_id UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_lead_stage_history_lead ON lead_stage_history(lead_id, changed_at DESC);
ALTER TABLE lead_stage_history ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS lead_stage_history_tenant_isolation ON lead_stage_history;
CREATE POLICY lead_stage_history_tenant_isolation ON lead_stage_history
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE TABLE IF NOT EXISTS lead_assignment_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    strategy TEXT NOT NULL DEFAULT 'round_robin' CHECK (strategy IN (
        'round_robin', 'least_loaded', 'skill_match', 'manual'
    )),
    target_role TEXT,
    source_filter JSONB DEFAULT '{}'::jsonb,
    min_score DECIMAL(5,2),
    is_active BOOLEAN DEFAULT true,
    last_assigned_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    total_assigned INT DEFAULT 0,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_lead_assign_rules_active ON lead_assignment_rules(tenant_id) WHERE is_active = true;
ALTER TABLE lead_assignment_rules ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS lead_assignment_rules_tenant_isolation ON lead_assignment_rules;
CREATE POLICY lead_assignment_rules_tenant_isolation ON lead_assignment_rules
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND current_setting('app.is_admin', true) = 'true'
    );

CREATE OR REPLACE FUNCTION seed_default_pipeline_for_tenant(p_tenant UUID)
RETURNS void AS $$
DECLARE
    pid UUID;
BEGIN
    IF EXISTS (SELECT 1 FROM pipelines WHERE tenant_id = p_tenant AND deleted_at IS NULL) THEN
        RETURN;
    END IF;
    INSERT INTO pipelines (tenant_id, name, description, is_default)
    VALUES (p_tenant, 'Default Sales Pipeline', 'Default 7-stage sales pipeline', true)
    RETURNING id INTO pid;

    INSERT INTO pipeline_stages (pipeline_id, tenant_id, name, code, color, position, probability, is_won, is_lost) VALUES
        (pid, p_tenant, 'New',         'NEW',         '#3b82f6', 1, 10, false, false),
        (pid, p_tenant, 'Contacted',   'CONTACTED',   '#06b6d4', 2, 25, false, false),
        (pid, p_tenant, 'Qualified',   'QUALIFIED',   '#10b981', 3, 50, false, false),
        (pid, p_tenant, 'Proposal',    'PROPOSAL',    '#f59e0b', 4, 70, false, false),
        (pid, p_tenant, 'Negotiation', 'NEGOTIATION', '#f97316', 5, 85, false, false),
        (pid, p_tenant, 'Won',         'WON',         '#22c55e', 6, 100, true,  false),
        (pid, p_tenant, 'Lost',        'LOST',        '#ef4444', 7, 0,   false, true);
END;
$$ LANGUAGE plpgsql;

ALTER TABLE activities ADD COLUMN IF NOT EXISTS lead_id UUID REFERENCES leads(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_activities_lead ON activities(tenant_id, lead_id) WHERE lead_id IS NOT NULL;
`

// =============================================================================
// Migration 0012 (Workflows)
// =============================================================================
const migration0012 = `
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    trigger_type TEXT NOT NULL CHECK (trigger_type IN (
        'stage_change', 'field_change', 'time_based', 'manual', 'lead_created', 'lead_assigned'
    )),
    trigger_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    conditions JSONB NOT NULL DEFAULT '{}'::jsonb,
    actions JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN DEFAULT true,
    run_count INT DEFAULT 0,
    last_run_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_workflows_tenant_active ON workflows(tenant_id) WHERE is_active = true AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_workflows_trigger ON workflows(tenant_id, trigger_type) WHERE is_active = true AND deleted_at IS NULL;
ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS workflows_tenant_isolation ON workflows;
CREATE POLICY workflows_tenant_isolation ON workflows
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            current_setting('app.is_admin', true) = 'true'
            OR EXISTS (
                SELECT 1 FROM users
                WHERE id = current_setting('app.current_user_id', true)::UUID
                AND tenant_id = workflows.tenant_id
                AND deleted_at IS NULL
                AND role IN ('owner', 'admin', 'manager')
            )
        )
    );

CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    triggered_by TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'success' CHECK (status IN ('success', 'failed', 'partial', 'skipped')),
    inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    actions_executed JSONB NOT NULL DEFAULT '[]'::jsonb,
    error TEXT,
    execution_ms INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_workflow_exec_workflow ON workflow_executions(workflow_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_exec_entity ON workflow_executions(entity_type, entity_id);
ALTER TABLE workflow_executions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS workflow_exec_tenant_isolation ON workflow_executions;
CREATE POLICY workflow_exec_tenant_isolation ON workflow_executions
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
`

// =============================================================================
// Migration 0013 (Audit log + sessions)
// =============================================================================
const migration0013 = `
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_email TEXT,
    action TEXT NOT NULL,
    target_type TEXT,
    target_id UUID,
    old_value JSONB,
    new_value JSONB,
    ip_address INET,
    user_agent TEXT,
    trace_id TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_actor ON audit_log(tenant_id, actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_action ON audit_log(tenant_id, action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_target ON audit_log(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_audit_trace ON audit_log(trace_id) WHERE trace_id IS NOT NULL;
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS audit_log_tenant_isolation ON audit_log;
CREATE POLICY audit_log_tenant_isolation ON audit_log
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            current_setting('app.is_admin', true) = 'true'
            OR actor_id = current_setting('app.current_user_id', true)::UUID
            OR EXISTS (
                SELECT 1 FROM users
                WHERE id = current_setting('app.current_user_id', true)::UUID
                AND tenant_id = audit_log.tenant_id
                AND role IN ('owner', 'admin', 'manager')
                AND deleted_at IS NULL
            )
        )
    );

CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    token_id TEXT UNIQUE NOT NULL,
    device_fingerprint TEXT,
    ip_address INET,
    user_agent TEXT,
    login_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN DEFAULT false,
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT
);
CREATE INDEX IF NOT EXISTS idx_user_sessions_active ON user_sessions(user_id) WHERE NOT revoked;
ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS user_sessions_tenant_isolation ON user_sessions;
CREATE POLICY user_sessions_tenant_isolation ON user_sessions
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            user_id = current_setting('app.current_user_id', true)::UUID
            OR current_setting('app.is_admin', true) = 'true'
            OR EXISTS (
                SELECT 1 FROM users
                WHERE id = current_setting('app.current_user_id', true)::UUID
                AND tenant_id = user_sessions.tenant_id
                AND role IN ('owner', 'admin')
                AND deleted_at IS NULL
            )
        )
    );
`

// =============================================================================
// Migration 0014 (Notifications + departments)
// =============================================================================
const migration0014 = `
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    leader_id UUID REFERENCES users(id) ON DELETE SET NULL,
    parent_department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_departments_tenant ON departments(tenant_id) WHERE deleted_at IS NULL;
ALTER TABLE departments ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS departments_tenant_isolation ON departments;
CREATE POLICY departments_tenant_isolation ON departments
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE TABLE IF NOT EXISTS department_members (
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    role TEXT DEFAULT 'member' CHECK (role IN ('leader', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (department_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_dept_members_user ON department_members(user_id);
CREATE INDEX IF NOT EXISTS idx_dept_members_tenant ON department_members(tenant_id);
ALTER TABLE department_members ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS department_members_tenant_isolation ON department_members;
CREATE POLICY department_members_tenant_isolation ON department_members
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sender_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    type TEXT NOT NULL CHECK (type IN (
        'broadcast', 'mention', 'lead_assigned', 'task_due',
        'workflow', 'invite', 'system'
    )),
    title TEXT NOT NULL,
    body TEXT,
    link TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMPTZ,
    priority TEXT NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('LOW','NORMAL','HIGH','URGENT')),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notif_user_inbox ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_tenant_priority ON notifications(tenant_id, priority, created_at);
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS notifications_tenant_isolation ON notifications;
CREATE POLICY notifications_tenant_isolation ON notifications
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            user_id = current_setting('app.current_user_id', true)::UUID
            OR current_setting('app.is_admin', true) = 'true'
        )
    );

CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    email_enabled BOOLEAN DEFAULT true,
    push_enabled BOOLEAN DEFAULT true,
    inapp_enabled BOOLEAN DEFAULT true,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    digest_mode TEXT DEFAULT 'off' CHECK (digest_mode IN ('off','daily','weekly')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS notif_prefs_tenant_isolation ON notification_preferences;
CREATE POLICY notif_prefs_tenant_isolation ON notification_preferences
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND user_id = current_setting('app.current_user_id', true)::UUID
    );
`
