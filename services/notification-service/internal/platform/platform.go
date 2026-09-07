// Package platform wraps infrastructure concerns for the notification service:
// structured logging, configuration, Postgres connection pool, Redis/Valkey
// client, NATS pub/sub, OpenTelemetry initialisation, and embedded SQL
// migrations applied at startup.
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

const serviceName = "notification-service"
const version = "1.0.0"

// Config bundles all runtime configuration.
type Config struct {
	Env                string
	HTTPAddr           string
	DatabaseURL        string
	ValkeyURL          string
	NatsURL            string
	OTLP               string
	EmailRPCURL        string
	TwilioSID          string
	TwilioToken        string
	TwilioFrom         string
	FCMProjectID       string
	FCMCredentialsFile string
	VAPIDPublic        string
	VAPIDPrivate       string
	VAPIDSubject       string
	WebBaseURL         string
	SlackWebhook       string
	TelegramBotToken   string
}

// LoadConfig reads environment variables into a Config struct.
func LoadConfig() *Config {
	return &Config{
		Env:                Getenv("ENV", "development"),
		HTTPAddr:           Getenv("NOTIF_HTTP_ADDR", ":8088"),
		DatabaseURL:        os.Getenv("NOTIF_DATABASE_URL"),
		ValkeyURL:         Getenv("NOTIF_VALKEY_URL", "localhost:6379"),
		NatsURL:           os.Getenv("NOTIF_NATS_URL"),
		OTLP:              os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		EmailRPCURL:        Getenv("NOTIF_EMAIL_RPC_URL", "http://email-service:8087"),
		TwilioSID:          os.Getenv("NOTIF_TWILIO_SID"),
		TwilioToken:        os.Getenv("NOTIF_TWILIO_TOKEN"),
		TwilioFrom:         os.Getenv("NOTIF_TWILIO_FROM"),
		FCMProjectID:       os.Getenv("NOTIF_FCM_PROJECT_ID"),
		FCMCredentialsFile: os.Getenv("NOTIF_FCM_CREDENTIALS_FILE"),
		VAPIDPublic:        os.Getenv("NOTIF_VAPID_PUBLIC"),
		VAPIDPrivate:       os.Getenv("NOTIF_VAPID_PRIVATE"),
		VAPIDSubject:       os.Getenv("NOTIF_VAPID_SUBJECT"),
		WebBaseURL:        Getenv("NOTIF_WEB_BASE_URL", "https://app.rinco.app"),
		SlackWebhook:       os.Getenv("NOTIF_SLACK_WEBHOOK"),
		TelegramBotToken:   os.Getenv("NOTIF_TELEGRAM_BOT_TOKEN"),
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

// OpenNATS returns a NATS connection (nil when URL is empty).
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

// TenantToCtx decorates a context with tenant metadata.
func TenantToCtx(ctx context.Context, tenantID, userID string, isAdmin bool) context.Context {
	return WithTenant(ctx, tenantID, userID, isAdmin)
}

// =============================================================================
// Embedded SQL migrations
// =============================================================================

const migration0001 = `
CREATE TABLE IF NOT EXISTS notification.notifications (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID NOT NULL,
	user_id TEXT NOT NULL,
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	body TEXT NOT NULL,
	icon TEXT,
	category TEXT,
	priority TEXT NOT NULL DEFAULT 'normal' CHECK (priority IN ('high','normal','low')),
	channels_resolved JSONB NOT NULL DEFAULT '[]'::jsonb,
	data JSONB NOT NULL DEFAULT '{}'::jsonb,
	status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','delivered','read','archived')),
	read_at TIMESTAMPTZ,
	sent_at TIMESTAMPTZ,
	expires_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notif_tenant_user ON notification.notifications(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_notif_user_status ON notification.notifications(user_id, status);
CREATE INDEX IF NOT EXISTS idx_notif_created ON notification.notifications(tenant_id, created_at DESC);
ALTER TABLE notification.notifications ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS notif_tenant ON notification.notifications;
CREATE POLICY notif_tenant ON notification.notifications
	USING (
		tenant_id = current_setting('app.current_tenant_id', true)::UUID
		OR current_setting('app.is_admin', true) = 'true'
	);
`

const migration0002 = `
CREATE TABLE IF NOT EXISTS notification.notification_preferences (
	user_id TEXT NOT NULL,
	notif_type TEXT NOT NULL,
	channel TEXT NOT NULL CHECK (channel IN ('in_app','email','sms','push','slack','discord','telegram','webhook')),
	enabled BOOLEAN NOT NULL DEFAULT true,
	quiet_start INT CHECK (quiet_start >= 0 AND quiet_start <= 23),
	quiet_end INT CHECK (quiet_end >= 0 AND quiet_end <= 23),
	digest_mode TEXT NOT NULL DEFAULT 'none' CHECK (digest_mode IN ('none','daily','weekly')),
	PRIMARY KEY (user_id, notif_type, channel)
);
CREATE INDEX IF NOT EXISTS idx_prefs_user ON notification.notification_preferences(user_id);
`

const migration0003 = `
CREATE TABLE IF NOT EXISTS notification.push_subscriptions (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id TEXT NOT NULL,
	endpoint TEXT NOT NULL UNIQUE,
	p256dh TEXT NOT NULL,
	auth TEXT NOT NULL,
	keys JSONB NOT NULL DEFAULT '{}'::jsonb,
	vapid_public_key TEXT,
	last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	expires_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_push_user ON notification.push_subscriptions(user_id);

CREATE TABLE IF NOT EXISTS notification.fcm_subscriptions (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id TEXT NOT NULL,
	device_token TEXT NOT NULL UNIQUE,
	platform TEXT NOT NULL CHECK (platform IN ('android','ios','web')),
	app_version TEXT,
	last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	expires_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_fcm_user ON notification.fcm_subscriptions(user_id);
`

const migration0004 = `
CREATE TABLE IF NOT EXISTS notification.notification_delivery_logs (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	notification_id UUID NOT NULL REFERENCES notification.notifications(id) ON DELETE CASCADE,
	channel TEXT NOT NULL CHECK (channel IN ('in_app','email','sms','push','slack','discord','telegram','webhook')),
	status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','delivered','failed')),
	sent_at TIMESTAMPTZ,
	delivered_at TIMESTAMPTZ,
	error_msg TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dl_notif ON notification.notification_delivery_logs(notification_id);

CREATE TABLE IF NOT EXISTS notification.daily_aggregates (
	date DATE NOT NULL,
	tenant_id UUID NOT NULL,
	type TEXT NOT NULL,
	channel TEXT NOT NULL,
	count_sent BIGINT NOT NULL DEFAULT 0,
	count_delivered BIGINT NOT NULL DEFAULT 0,
	count_failed BIGINT NOT NULL DEFAULT 0,
	PRIMARY KEY (date, tenant_id, type, channel)
);
CREATE INDEX IF NOT EXISTS idx_agg_tenant_date ON notification.daily_aggregates(tenant_id, date DESC);
`

// migrations lists every SQL block to be applied at startup.
func migrations() []migrationFile {
	return []migrationFile{
		{name: "0001_notifications", body: migration0001},
		{name: "0002_preferences", body: migration0002},
		{name: "0003_subscriptions", body: migration0003},
		{name: "0004_aggregate", body: migration0004},
	}
}

type migrationFile struct {
	name, body string
}

// RunMigrations applies embedded SQL files idempotently with bookkeeping.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS notification;
		CREATE TABLE IF NOT EXISTS notification.schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	rows, err := pool.Query(ctx, `SELECT version FROM notification.schema_migrations`)
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
		if _, err := pool.Exec(ctx, `INSERT INTO notification.schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, m.name); err != nil {
			return fmt.Errorf("bookkeeping %s: %w", m.name, err)
		}
		slog.Info("applied migration", slog.String("name", m.name))
	}
	return nil
}