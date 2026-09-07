// Package platform wraps infrastructure concerns for the observability
// service: structured logging, configuration, HTTP clients for the
// upstream observability stack (Loki / Prometheus / Jaeger / Tempo /
// ClickHouse), and embedded SQL migrations applied at boot.
//
// The service is a thin aggregator over self-hosted observability
// backends.  It does not store logs, traces or metrics itself; it only
// stores alerts (Postgres) and application audit (Postgres + ClickHouse
// rollups).
//
// ClickHouse is reached via its native binary protocol (TCP :9000) using
// clickhouse-go v2; the connection is best-effort — when CH is
// unreachable the service still serves traffic and writes audit rows
// only to Postgres.
package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

	// ClickHouse is reached via its HTTP gateway (port 8123).  See
	// EnsureClickHouseDDL and clickhouse.Writer for the HTTP-based
	// implementations; we deliberately avoid clickhouse-go's native
	// driver because of its tight coupling with ch-go versions that
	// break under independent version bumps.
)

const serviceName = "observability-service"
const version = "1.0.0"

// Config bundles runtime configuration.
type Config struct {
	Env             string
	HTTPAddr        string
	DatabaseURL     string
	ClickHouseURL   string
	PrometheusURL   string
	LokiURL         string
	JaegerURL       string
	TempoURL        string
	OTLP            string
	RateLimitPerMin int
}

// LoadConfig reads environment variables.
func LoadConfig() *Config {
	return &Config{
		Env:             Getenv("ENV", "development"),
		HTTPAddr:        Getenv("OBSERVABILITY_HTTP_ADDR", ":8099"),
		DatabaseURL:     os.Getenv("OBSERVABILITY_DATABASE_URL"),
		ClickHouseURL:   Getenv("OBSERVABILITY_CLICKHOUSE_URL", "http://localhost:8123"),
		PrometheusURL:   Getenv("OBSERVABILITY_PROMETHEUS_URL", "http://localhost:9090"),
		LokiURL:         Getenv("OBSERVABILITY_LOKI_URL", "http://localhost:3100"),
		JaegerURL:       Getenv("OBSERVABILITY_JAEGER_URL", "http://localhost:16686"),
		TempoURL:        Getenv("OBSERVABILITY_TEMPO_URL", "http://localhost:3200"),
		OTLP:            os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		RateLimitPerMin: GetenvInt("OBSERVABILITY_RATE_LIMIT", 600),
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

// GetenvBool reads a bool env.
func GetenvBool(key string, fallback bool) bool {
	v := strings.ToLower(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}

// InitLogger configures slog with JSON output.
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

// OpenPool opens a pgxpool.
func OpenPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		dsn = "postgres://rinco:rinco_dev_password@localhost:5432/rinco?sslmode=disable"
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 20
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

// InitTracer sets up an OTLP trace exporter.
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
// HTTP JSON helper used by every upstream client.
// =============================================================================

// JSONClient is a tiny HTTP wrapper with timeout + JSON helper.
type JSONClient struct {
	BaseURL string
	APIKey  string
	HC      *http.Client
}

// NewJSONClient builds a JSON client.
func NewJSONClient(baseURL, apiKey string, timeout time.Duration) *JSONClient {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &JSONClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HC:      &http.Client{Timeout: timeout},
	}
}

// Get issues a GET and returns parsed JSON.
func (c *JSONClient) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// GetRaw issues a GET and returns raw bytes.
func (c *JSONClient) GetRaw(ctx context.Context, path string) ([]byte, error) {
	return c.doRaw(ctx, http.MethodGet, path, nil)
}

// Post issues a POST and returns parsed JSON.
func (c *JSONClient) Post(ctx context.Context, path string, body any, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

// PostRaw posts a raw body and returns raw bytes.
func (c *JSONClient) PostRaw(ctx context.Context, path string, body []byte, contentType string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HC.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// do is the common request helper.
func (c *JSONClient) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HC.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(buf))
	}
	if out == nil {
		_, err = io.Copy(io.Discard, resp.Body)
		return err
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *JSONClient) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HC.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(buf))
	}
	return io.ReadAll(resp.Body)
}

// =============================================================================
// Embedded SQL migrations (alerts + audit + cache).
// =============================================================================

const migration0001 = `
CREATE SCHEMA IF NOT EXISTS observability;
CREATE TABLE IF NOT EXISTS observability.alerts (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	fingerprint TEXT UNIQUE,
	status TEXT NOT NULL DEFAULT 'firing' CHECK (status IN ('firing','resolved','ack')),
	severity TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('critical','warning','info')),
	labels JSONB NOT NULL DEFAULT '{}'::jsonb,
	annotations JSONB NOT NULL DEFAULT '{}'::jsonb,
	service TEXT,
	tenant_id UUID,
	title TEXT NOT NULL,
	message TEXT,
	fired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	resolved_at TIMESTAMPTZ,
	ack_by UUID,
	ack_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON observability.alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_service ON observability.alerts(service);
CREATE INDEX IF NOT EXISTS idx_alerts_tenant ON observability.alerts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_alerts_fired_at ON observability.alerts(fired_at DESC);

CREATE TABLE IF NOT EXISTS observability.alert_history (
	id BIGSERIAL PRIMARY KEY,
	alert_id UUID NOT NULL REFERENCES observability.alerts(id) ON DELETE CASCADE,
	event TEXT NOT NULL,
	payload JSONB NOT NULL DEFAULT '{}'::jsonb,
	ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_alert_history_alert ON observability.alert_history(alert_id, ts DESC);

CREATE TABLE IF NOT EXISTS observability.audit_logs (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	tenant_id UUID NOT NULL,
	actor_user_id UUID,
	actor_ip TEXT,
	action TEXT NOT NULL,
	resource_type TEXT,
	resource_id TEXT,
	payload JSONB NOT NULL DEFAULT '{}'::jsonb,
	ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_tenant ON observability.audit_logs(tenant_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action ON observability.audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON observability.audit_logs(actor_user_id);

CREATE TABLE IF NOT EXISTS observability.service_health_cache (
	service TEXT PRIMARY KEY,
	status TEXT NOT NULL DEFAULT 'unknown',
	error_rate DOUBLE PRECISION,
	p99_latency DOUBLE PRECISION,
	active_alerts INT NOT NULL DEFAULT 0,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

const migration0002 = `
ALTER TABLE observability.audit_logs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS audit_logs_tenant ON observability.audit_logs;
CREATE POLICY audit_logs_tenant ON observability.audit_logs
	USING (
		tenant_id = current_setting('app.current_tenant_id', true)::UUID
		OR current_setting('app.is_admin', true) = 'true'
	);
`

// migrations lists every SQL block to be applied at startup.
func migrations() []migrationFile {
	return []migrationFile{
		{name: "0001_alerts", body: migration0001},
		{name: "0002_audit", body: migration0002},
	}
}

type migrationFile struct {
	name, body string
}

// RunMigrations applies embedded SQL files idempotently.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS observability;
		CREATE TABLE IF NOT EXISTS observability.schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	rows, err := pool.Query(ctx, `SELECT version FROM observability.schema_migrations`)
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
		if _, err := pool.Exec(ctx, `INSERT INTO observability.schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, m.name); err != nil {
			return fmt.Errorf("bookkeeping %s: %w", m.name, err)
		}
		slog.Info("applied migration", slog.String("name", m.name))
	}
	return nil
}

// =============================================================================
// ClickHouse DDL (run on startup, best-effort).
// =============================================================================

// ClickHouseDDL contains the table DDL statements that should exist in the
// `observability` database.
const ClickHouseDDL = `
CREATE DATABASE IF NOT EXISTS observability;

CREATE TABLE IF NOT EXISTS observability.app_audit_logs (
	id String,
	tenant_id String,
	actor_user_id String,
	actor_ip String,
	action LowCardinality(String),
	resource_type LowCardinality(String),
	resource_id String,
	payload String,
	ts DateTime64(3)
) ENGINE = MergeTree
PARTITION BY toYYYYMM(ts)
ORDER BY (tenant_id, ts, actor_user_id);

CREATE TABLE IF NOT EXISTS observability.app_incidents (
	id String,
	service LowCardinality(String),
	severity LowCardinality(String),
	title String,
	started_at DateTime64(3),
	resolved_at Nullable(DateTime64(3)),
	root_cause String,
	runbook_url String,
	status LowCardinality(String) DEFAULT 'open'
) ENGINE = MergeTree
PARTITION BY toYYYYMM(started_at)
ORDER BY (service, started_at);

CREATE TABLE IF NOT EXISTS observability.app_metric_rollup (
	service TEXT NOT NULL,
	metric TEXT NOT NULL,
	bucket DateTime64(3) NOT NULL,
	value Float64,
	labels Map(LowCardinality(String), String),
	ts DateTime64(3)
) ENGINE = MergeTree
PARTITION BY toYYYYMM(ts)
ORDER BY (service, metric, ts);
`

// CHConn is a sentinel interface kept for forward-compatibility.  When
// the observability service is extended to use the native driver the
// implementation will live alongside clickhouse.Writer.
type CHConn interface {
	Exec(ctx context.Context, query string, args ...any) error
	Close() error
}

// OpenClickHouse returns nil and a sentinel error.  Audit writes go
// through the HTTP gateway (see EnsureClickHouseDDL and
// clickhouse.NewWriter).  The function exists so callers have a single
// place to upgrade to a native driver in future.
func OpenClickHouse(chURL string) (any, error) {
	if chURL == "" {
		return nil, errors.New("clickhouse url is empty")
	}
	return nil, errors.New("native clickhouse driver disabled; using HTTP gateway")
}

// EnsureClickHouseDDL posts the bundled DDL statements to the ClickHouse
// HTTP gateway and is best-effort.
func EnsureClickHouseDDL(ctx context.Context, chURL string) error {
	if chURL == "" {
		return errors.New("clickhouse url is empty")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, stmt := range splitDDL(ClickHouseDDL) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, chURL+"/?database=observability", strings.NewReader(stmt))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("clickhouse ddl HTTP %d", resp.StatusCode)
		}
	}
	return nil
}

func splitDDL(ddl string) []string {
	parts := strings.Split(ddl, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}