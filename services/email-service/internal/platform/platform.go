// Package platform wraps infrastructure concerns for the email service:
// structured logging, configuration, Postgres connection pool, Redis/Valkey
// client, NATS pub/sub, OpenTelemetry initialisation, embedded SQL
// migrations, and basic context helpers.
package platform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const serviceName = "email-service"
const version = "1.0.0"

// Config bundles all runtime configuration for the service.
type Config struct {
	Env               string
	HTTPAddr          string
	DatabaseURL       string
	ValkeyURL         string
	NatsURL           string
	OTLP              string
	Driver            string
	SMTPHost          string
	SMTPPort          int
	SMTPUser          string
	SMTPPass          string
	ResendAPIKey      string
	SendGridAPIKey    string
	AWSRegion         string
	AWSAccessKey      string
	AWSSecretKey      string
	FromDefault       string
	WebBaseURL        string
	S3BucketHot       string
	S3BucketCold      string
	S3Endpoint        string
	S3AccessKey       string
	S3SecretKey       string
	S3UseSSL          bool
	MaxHotAgeDays     int
	WebhookSecret     string
}

// LoadConfig reads environment variables into a Config struct.
func LoadConfig() *Config {
	return &Config{
		Env:           Getenv("ENV", "development"),
		HTTPAddr:      Getenv("EMAIL_HTTP_ADDR", ":8087"),
		DatabaseURL:   os.Getenv("EMAIL_DATABASE_URL"),
		ValkeyURL:     Getenv("EMAIL_VALKEY_URL", "localhost:6379"),
		NatsURL:       os.Getenv("EMAIL_NATS_URL"),
		OTLP:          os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Driver:        Getenv("EMAIL_DRIVER", "console"),
		SMTPHost:      os.Getenv("EMAIL_SMTP_HOST"),
		SMTPPort:      GetenvInt("EMAIL_SMTP_PORT", 587),
		SMTPUser:      os.Getenv("EMAIL_SMTP_USER"),
		SMTPPass:      os.Getenv("EMAIL_SMTP_PASS"),
		ResendAPIKey:  os.Getenv("EMAIL_RESEND_API_KEY"),
		SendGridAPIKey: os.Getenv("EMAIL_SENDGRID_API_KEY"),
		AWSRegion:     os.Getenv("EMAIL_AWS_REGION"),
		AWSAccessKey:  os.Getenv("EMAIL_AWS_ACCESS_KEY"),
		AWSSecretKey:  os.Getenv("EMAIL_AWS_SECRET"),
		FromDefault:   Getenv("EMAIL_FROM_DEFAULT", "no-reply@rinco.app"),
		WebBaseURL:    Getenv("EMAIL_WEB_BASE_URL", "https://track.rinco.app"),
		S3BucketHot:   Getenv("EMAIL_S3_BUCKET_HOT", "rinco-email-hot"),
		S3BucketCold:  Getenv("EMAIL_S3_BUCKET_COLD", "rinco-email-cold"),
		S3Endpoint:    os.Getenv("EMAIL_S3_ENDPOINT"),
		S3AccessKey:   os.Getenv("EMAIL_S3_ACCESS_KEY"),
		S3SecretKey:   os.Getenv("EMAIL_S3_SECRET_KEY"),
		S3UseSSL:      strings.EqualFold(Getenv("EMAIL_S3_SSL", "false"), "true"),
		MaxHotAgeDays: GetenvInt("EMAIL_S3_HOT_DAYS", 30),
		WebhookSecret: os.Getenv("EMAIL_WEBHOOK_SECRET"),
	}
}

// Getenv reads env or returns the fallback.
func Getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetenvInt reads an int env, returning fallback on error.
func GetenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return fallback
}

// GetenvBool reads a bool env (true/1/yes).
func GetenvBool(key string, fallback bool) bool {
	v := strings.ToLower(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}

// InitLogger configures slog with JSON output and service metadata.
func InitLogger(env string) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level, AddSource: env == "development"})
	l := slog.New(h).With(
		slog.String("service", serviceName),
		slog.String("env", env),
		slog.String("version", version),
	)
	slog.SetDefault(l)
	return l
}

// OpenPool opens a pgxpool with sane defaults.
func OpenPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		dsn = "postgres://rinco:rinco_dev_password@localhost:5432/rinco?sslmode=disable"
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 25
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// OpenRedis returns a configured Redis client.
func OpenRedis(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr})
}

// OpenNATS returns a NATS connection (may be nil when URL is empty).
func OpenNATS(url string) (*nats.Conn, error) {
	if url == "" {
		return nil, nil
	}
	conn, err := nats.Connect(url, nats.MaxReconnects(-1), nats.ReconnectWait(2*time.Second))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// InitTracer sets up an OTLP trace exporter; no-op for local development.
func InitTracer(ctx context.Context, endpoint, env string) (*sdktrace.TracerProvider, error) {
	if endpoint == "" || env == "development" {
		return nil, nil
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(u.Host), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(version),
		semconv.DeploymentEnvironment(env),
	))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

// =============================================================================
// Context helpers
// =============================================================================

type ctxKey string

const (
	tenantIDKey ctxKey = "tenant_id"
	userIDKey   ctxKey = "user_id"
	isAdminKey  ctxKey = "is_admin"
	traceIDKey  ctxKey = "trace_id"
)

// WithTenant decorates a context with tenant metadata.
func WithTenant(ctx context.Context, tenantID, userID string, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, tenantIDKey, tenantID)
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, isAdminKey, isAdmin)
	return ctx
}

// TenantFromContext extracts the tenant identifier.
func TenantFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}

// UserFromContext extracts the user identifier.
func UserFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// IsAdminFromContext reports whether the context is a super-admin.
func IsAdminFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(isAdminKey).(bool)
	return v
}

// TenantToCtx decorates a context with tenant metadata; used by Echo
// middleware so that downstream code (including background workers) can
// retrieve identifiers via TenantFromContext / UserFromContext.
func TenantToCtx(ctx context.Context, tenantID, userID string, isAdmin bool) context.Context {
	return WithTenant(ctx, tenantID, userID, isAdmin)
}

// WithTraceID attaches a trace id to the context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// TraceFromContext returns the trace id.
func TraceFromContext(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}

// IsNotFound reports whether an error is a not-found sentinel.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrNotFound) ||
		strings.Contains(err.Error(), "no rows") ||
		strings.Contains(err.Error(), "not found")
}

// ErrNotFound sentinel.
var ErrNotFound = errors.New("not found")

// =============================================================================
// Embedded SQL migrations
// =============================================================================

const migration0001 = `
CREATE TABLE IF NOT EXISTS email.email_templates (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID NOT NULL,
	name TEXT NOT NULL,
	subject TEXT NOT NULL,
	body TEXT NOT NULL,
	body_type TEXT NOT NULL DEFAULT 'html' CHECK (body_type IN ('text','html','markdown')),
	vars JSONB NOT NULL DEFAULT '{}'::jsonb,
	type TEXT NOT NULL DEFAULT 'transactional',
	is_active BOOLEAN NOT NULL DEFAULT true,
	version INT NOT NULL DEFAULT 1,
	created_by UUID,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_email_templates_tenant ON email.email_templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_email_templates_tenant_name ON email.email_templates(tenant_id, name) WHERE deleted_at IS NULL;
ALTER TABLE email.email_templates ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS email_templates_tenant ON email.email_templates;
CREATE POLICY email_templates_tenant ON email.email_templates
	USING (
		tenant_id = current_setting('app.current_tenant_id', true)::UUID
		OR current_setting('app.is_admin', true) = 'true'
	);
`

const migration0002 = `
CREATE TABLE IF NOT EXISTS email.email_logs (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID NOT NULL,
	msg_id TEXT NOT NULL UNIQUE,
	template_id UUID,
	from_addr TEXT NOT NULL,
	to_addrs JSONB NOT NULL DEFAULT '[]'::jsonb,
	cc JSONB NOT NULL DEFAULT '[]'::jsonb,
	bcc JSONB NOT NULL DEFAULT '[]'::jsonb,
	subject TEXT NOT NULL,
	body_rendered TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','sending','sent','delivered','opened','clicked','bounced','complained','failed','retrying')),
	driver TEXT NOT NULL DEFAULT 'console',
	driver_msg_id TEXT,
	priority TEXT NOT NULL DEFAULT 'normal' CHECK (priority IN ('high','normal','low')),
	retry_count INT NOT NULL DEFAULT 0,
	last_error TEXT,
	sent_at TIMESTAMPTZ,
	delivered_at TIMESTAMPTZ,
	opened_at TIMESTAMPTZ,
	clicked_at TIMESTAMPTZ,
	bounced_at TIMESTAMPTZ,
	complained_at TIMESTAMPTZ,
	failed_at TIMESTAMPTZ,
	scheduled_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_email_logs_tenant ON email.email_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_email_logs_status ON email.email_logs(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_email_logs_template ON email.email_logs(template_id);
CREATE INDEX IF NOT EXISTS idx_email_logs_created ON email.email_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_email_logs_msg ON email.email_logs(msg_id);
ALTER TABLE email.email_logs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS email_logs_tenant ON email.email_logs;
CREATE POLICY email_logs_tenant ON email.email_logs
	USING (
		tenant_id = current_setting('app.current_tenant_id', true)::UUID
		OR current_setting('app.is_admin', true) = 'true'
	);

CREATE TABLE IF NOT EXISTS email.email_attachments (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	msg_id TEXT NOT NULL,
	tenant_id UUID NOT NULL,
	filename TEXT NOT NULL,
	content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
	size_bytes BIGINT NOT NULL DEFAULT 0,
	s3_key TEXT NOT NULL,
	s3_bucket TEXT NOT NULL,
	tier TEXT NOT NULL DEFAULT 'hot' CHECK (tier IN ('hot','cold')),
	uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_email_attachments_msg ON email.email_attachments(msg_id);
CREATE INDEX IF NOT EXISTS idx_email_attachments_tenant ON email.email_attachments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_email_attachments_tier ON email.email_attachments(tier, uploaded_at);

CREATE TABLE IF NOT EXISTS email.tenant_settings (
	tenant_id UUID PRIMARY KEY,
	tier_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
	driver TEXT NOT NULL DEFAULT 'console',
	default_from TEXT,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

const migration0003 = `
CREATE TABLE IF NOT EXISTS email.email_webhooks (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID NOT NULL,
	provider TEXT NOT NULL,
	event_type TEXT NOT NULL,
	payload JSONB NOT NULL DEFAULT '{}'::jsonb,
	processed_at TIMESTAMPTZ,
	received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_email_webhooks_tenant ON email.email_webhooks(tenant_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_email_webhooks_provider ON email.email_webhooks(provider, event_type);
`

// migrations lists every SQL block to be applied at startup.
func migrations() []migrationFile {
	return []migrationFile{
		{name: "0001_templates", body: migration0001},
		{name: "0002_logs", body: migration0002},
		{name: "0003_webhooks", body: migration0003},
	}
}

type migrationFile struct {
	name, body string
}

// RunMigrations applies embedded SQL files idempotently with bookkeeping.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS email;
		CREATE TABLE IF NOT EXISTS email.schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	rows, err := pool.Query(ctx, `SELECT version FROM email.schema_migrations`)
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	applied := map[string]bool{}
	for rows.Next() {
		var v string
		_ = rows.Scan(&v)
		applied[v] = true
	}
	rows.Close()
	for _, m := range migrations() {
		if applied[m.name] {
			continue
		}
		if _, err := pool.Exec(ctx, m.body); err != nil {
			return fmt.Errorf("apply %s: %w", m.name, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO email.schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, m.name); err != nil {
			return fmt.Errorf("bookkeeping %s: %w", m.name, err)
		}
		slog.Info("applied migration", slog.String("name", m.name))
	}
	return nil
}