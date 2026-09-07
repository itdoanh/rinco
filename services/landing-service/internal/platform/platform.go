package platform

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const serviceName = "landing-service"

func Getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" { return value }
	return fallback
}

func GetenvInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key)); if err != nil { return fallback }; return value
}

func InitLogger(env, version string) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) { case "debug": level = slog.LevelDebug; case "warn": level = slog.LevelWarn; case "error": level = slog.LevelError }
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level, AddSource: env == "development"}).WithAttrs([]slog.Attr{slog.String("service", serviceName), slog.String("env", env), slog.String("version", version)}))
	slog.SetDefault(logger)
	return logger
}

func OpenPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" { dsn = "postgres://rinco:rinco_dev_password@localhost:5432/rinco?sslmode=disable" }
	cfg, err := pgxpool.ParseConfig(dsn); if err != nil { return nil, fmt.Errorf("parse database url: %w", err) }
	cfg.MaxConns, cfg.MinConns = 25, 2
	cfg.MaxConnLifetime, cfg.MaxConnIdleTime = 30*time.Minute, 5*time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg); if err != nil { return nil, fmt.Errorf("open database: %w", err) }
	if err := pool.Ping(ctx); err != nil { pool.Close(); return nil, fmt.Errorf("ping database: %w", err) }
	return pool, nil
}

func InitTracer(ctx context.Context, endpoint, env string) (*sdktrace.TracerProvider, error) {
	if endpoint == "" || env == "development" { return nil, nil }
	u, err := url.Parse(endpoint); if err != nil { return nil, err }
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(u.Host), otlptracegrpc.WithInsecure()); if err != nil { return nil, err }
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName), semconv.ServiceVersion("1.0.0"), semconv.DeploymentEnvironment(env))); if err != nil { return nil, err }
	provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res), sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())))
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return provider, nil
}

// Migrations are managed via SQL files in the migrations/ folder. We
// apply them lazily by inspecting sql files, but for production the
// images are expected to be initialised out-of-band (Helm chart runs
// `psql -f`). When LANDING_AUTOMIGRATE=true we run the embedded SQL
// instead.
const bootstrapSQL = `
CREATE SCHEMA IF NOT EXISTS landing;
CREATE TABLE IF NOT EXISTS landing.landing_pages (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID,
	tenant_slug TEXT,
	slug TEXT,
	page_slug TEXT,
	title TEXT NOT NULL DEFAULT '',
	meta JSONB NOT NULL DEFAULT '{}'::jsonb,
	design_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
	mongo_page_id TEXT,
	status TEXT NOT NULL DEFAULT 'draft',
	version INT NOT NULL DEFAULT 1,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE landing.landing_pages ADD COLUMN IF NOT EXISTS tenant_slug TEXT;
ALTER TABLE landing.landing_pages ADD COLUMN IF NOT EXISTS slug TEXT;
ALTER TABLE landing.landing_pages ADD COLUMN IF NOT EXISTS page_slug TEXT;
ALTER TABLE landing.landing_pages ADD COLUMN IF NOT EXISTS mongo_page_id TEXT;
ALTER TABLE landing.landing_pages ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1;
ALTER TABLE landing.landing_pages ADD COLUMN IF NOT EXISTS meta JSONB NOT NULL DEFAULT '{}'::jsonb;
CREATE INDEX IF NOT EXISTS idx_landing_pages_slug ON landing.landing_pages(tenant_slug, slug);
CREATE INDEX IF NOT EXISTS idx_landing_pages_page_slug ON landing.landing_pages(tenant_slug, page_slug);

CREATE TABLE IF NOT EXISTS landing.form_submissions (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID,
	form_id UUID,
	form_slug TEXT,
	payload JSONB NOT NULL DEFAULT '{}'::jsonb,
	status TEXT NOT NULL DEFAULT 'accepted',
	event_id TEXT,
	ip_address INET,
	user_agent TEXT,
	idempotency_key TEXT,
	lead_id UUID,
	utm JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE landing.form_submissions ADD COLUMN IF NOT EXISTS event_id TEXT;
ALTER TABLE landing.form_submissions ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
ALTER TABLE landing.form_submissions ADD COLUMN IF NOT EXISTS ip_address INET;
ALTER TABLE landing.form_submissions ADD COLUMN IF NOT EXISTS user_agent TEXT;
ALTER TABLE landing.form_submissions ADD COLUMN IF NOT EXISTS form_slug TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_landing_form_submissions_event ON landing.form_submissions(tenant_id,event_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_landing_form_submissions_idem ON landing.form_submissions(tenant_id,idempotency_key);

CREATE TABLE IF NOT EXISTS landing.tracking_events (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID,
	event_name TEXT NOT NULL,
	event_type TEXT NOT NULL DEFAULT 'custom',
	event_id TEXT,
	page_slug TEXT,
	session_id TEXT,
	properties JSONB NOT NULL DEFAULT '{}'::jsonb,
	payload JSONB NOT NULL DEFAULT '{}'::jsonb,
	ip_address INET,
	user_agent TEXT,
	referer TEXT,
	utm JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE landing.tracking_events ADD COLUMN IF NOT EXISTS event_id TEXT;
ALTER TABLE landing.tracking_events ADD COLUMN IF NOT EXISTS session_id TEXT;
ALTER TABLE landing.tracking_events ADD COLUMN IF NOT EXISTS payload JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE landing.tracking_events ADD COLUMN IF NOT EXISTS ip_address INET;
ALTER TABLE landing.tracking_events ADD COLUMN IF NOT EXISTS user_agent TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_landing_tracking_event_id ON landing.tracking_events(tenant_id,event_id);
CREATE INDEX IF NOT EXISTS idx_landing_tracking_events_tenant_created ON landing.tracking_events(tenant_id,created_at DESC);

CREATE TABLE IF NOT EXISTS landing.form_definitions (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID,
	form_slug TEXT,
	name TEXT NOT NULL DEFAULT '',
	fields JSONB NOT NULL DEFAULT '[]'::jsonb,
	schema JSONB NOT NULL DEFAULT '{}'::jsonb,
	model_id UUID,
	active BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE landing.form_definitions ADD COLUMN IF NOT EXISTS schema JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE landing.form_definitions ADD COLUMN IF NOT EXISTS model_id UUID;
ALTER TABLE landing.form_definitions ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE IF NOT EXISTS landing.conversion_goals (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID,
	name TEXT NOT NULL,
	event_name TEXT NOT NULL,
	config JSONB NOT NULL DEFAULT '{}'::jsonb,
	conditions JSONB NOT NULL DEFAULT '{}'::jsonb,
	value NUMERIC(10,2) DEFAULT 0,
	active BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS landing.fb_pixel_configs (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID NOT NULL UNIQUE,
	pixel_id TEXT NOT NULL,
	app_secret TEXT,
	access_token TEXT,
	test_event_code TEXT,
	is_active BOOLEAN NOT NULL DEFAULT true,
	enabled BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE landing.fb_pixel_configs ADD COLUMN IF NOT EXISTS app_secret TEXT;
ALTER TABLE landing.fb_pixel_configs ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE IF NOT EXISTS landing.tracking_clicks (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID,
	target_url TEXT NOT NULL,
	utm JSONB NOT NULL DEFAULT '{}'::jsonb,
	clicks INT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if strings.EqualFold(os.Getenv("LANDING_AUTOMIGRATE"), "true") {
		if _, err := pool.Exec(ctx, bootstrapSQL); err != nil { return fmt.Errorf("automigrate: %w", err) }
	}
	return pool.Ping(ctx)
}
