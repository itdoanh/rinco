// Tenant Service — CRUD for tenants, domains, branding, settings, and usage.
// Multi-tenant aware via PostgreSQL RLS on `current_tenant_id` (set by gateway / Connect-RPC).
package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const serviceName = "tenant-service"

// =============================================================================
// Config
// =============================================================================

type config struct {
	HTTPAddr     string
	DatabaseURL  string
	ValkeyURL    string
	OTLPEndpoint string
	LogLevel     slog.Level
	AdminAPIKey  string
	Environment  string
}

func loadConfig() config {
	c := config{
		HTTPAddr:     getEnv("TENANT_HTTP_ADDR", ":8082"),
		DatabaseURL:  os.Getenv("TENANT_DATABASE_URL"),
		ValkeyURL:    getEnv("TENANT_VALKEY_URL", "redis://localhost:6379/0"),
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		AdminAPIKey:  getEnv("TENANT_ADMIN_API_KEY", "dev_admin_key_change_me"),
		Environment:  getEnv("ENV", "development"),
	}
	switch strings.ToLower(getEnv("LOG_LEVEL", "info")) {
	case "debug":
		c.LogLevel = slog.LevelDebug
	case "warn":
		c.LogLevel = slog.LevelWarn
	case "error":
		c.LogLevel = slog.LevelError
	default:
		c.LogLevel = slog.LevelInfo
	}
	if c.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "TENANT_DATABASE_URL is required")
		os.Exit(1)
	}
	return c
}

// =============================================================================
// OpenTelemetry
// =============================================================================

func initTracer(ctx context.Context, endpoint string) (func(context.Context) error, error) {
	if endpoint == "" {
		// Don't override the global TracerProvider — default is a no-op already.
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion("1.0.0"),
		attribute.String("deployment.environment", getEnv("ENV", "development")),
	))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// =============================================================================
// Database helpers
// =============================================================================

type server struct {
	pool  *pgxpool.Pool
	rdb   *redis.Client
	cfg   config
	log   *slog.Logger
	ready *sync.RWMutex
	ok    bool
}

func newPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second
	return pgxpool.NewWithConfig(ctx, cfg)
}

// withTenant runs fn inside a transaction with RLS context applied.
// Deprecated: use packages/go/db.WithTxTenant instead.
func (s *server) withTenant(ctx context.Context, tenantID string, fn func(pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if tenantID != "" {
		// Validate UUID format and use parameterized SET LOCAL to prevent
		// SQL injection. set_config(name, value, is_local=true) is the safe
		// equivalent of SET LOCAL.
		if _, err := uuid.Parse(tenantID); err != nil {
			return fmt.Errorf("withTenant: invalid tenant_id: %w", err)
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
			return err
		}
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// =============================================================================
// Migrations
// =============================================================================

func (s *server) runMigrations(ctx context.Context) error {
	for _, m := range Migrations {
		s.log.Info("applying migration", slog.String("name", m.Name))
		if _, err := s.pool.Exec(ctx, m.SQL); err != nil {
			return fmt.Errorf("migration %s: %w", m.Name, err)
		}
	}
	return nil
}

// =============================================================================
// Middleware
// =============================================================================

func (s *server) recoveryMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					s.log.Error("panic", slog.Any("panic", r), slog.String("path", c.Request().URL.Path))
					_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal_error"})
				}
			}()
			return next(c)
		}
	}
}

func traceMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get("X-Trace-ID")
			if id == "" {
				id = uuid.NewString()
			}
			c.Response().Header().Set("X-Trace-ID", id)
			c.Set("trace_id", id)
			return next(c)
		}
	}
}

var httpReqs = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "http_requests_total", Help: "HTTP requests."},
	[]string{"method", "path", "status"})
var httpDur = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "Duration.", Buckets: prometheus.DefBuckets},
	[]string{"method", "path"})

func init() {
	prometheus.MustRegister(httpReqs, httpDur)
}

func metricsMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			status := c.Response().Status
			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}
			httpReqs.WithLabelValues(c.Request().Method, path, strconv.Itoa(status)).Inc()
			httpDur.WithLabelValues(c.Request().Method, path).Observe(time.Since(start).Seconds())
			return err
		}
	}
}

func loggingMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			status := c.Response().Status
			if err != nil || status >= 500 {
				slog.Warn("http",
					slog.String("service", serviceName),
					slog.String("method", c.Request().Method),
					slog.String("path", c.Request().URL.Path),
					slog.Int("status", status),
					slog.Duration("dur", time.Since(start)),
					slog.String("trace_id", fmt.Sprint(c.Get("trace_id"))),
				)
			} else {
				slog.Info("http",
					slog.String("service", serviceName),
					slog.String("method", c.Request().Method),
					slog.String("path", c.Request().URL.Path),
					slog.Int("status", status),
					slog.Duration("dur", time.Since(start)),
				)
			}
			return err
		}
	}
}

func corsMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("Access-Control-Allow-Origin", "*")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Trace-ID, X-Tenant-ID, X-Admin-Key")
			if c.Request().Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	}
}

func securityHeadersMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			return next(c)
		}
	}
}

func (s *server) adminAuthMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-Admin-Key")
			if key == "" || !subtleCompare(key, s.cfg.AdminAPIKey) {
				return jsonErr(c, 401, "UNAUTHORIZED", "admin key required")
			}
			return next(c)
		}
	}
}

func (s *server) requireTenantMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tid := c.Request().Header.Get("X-Tenant-ID")
			if tid == "" {
				if uid := c.Get("user_id"); uid != nil {
					if s, ok := uid.(string); ok {
						_ = s
					}
				}
			}
			if tid == "" {
				return jsonErr(c, 400, "TENANT_REQUIRED", "X-Tenant-ID header required")
			}
			c.Set("tenant_id", tid)
			return next(c)
		}
	}
}

func subtleCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// =============================================================================
// Helpers
// =============================================================================

func jsonErr(c echo.Context, status int, code, msg string) error {
	return c.JSON(status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}

func newID() string {
	u, err := uuid.NewV7()
	if err != nil {
		return uuid.New().String()
	}
	return u.String()
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		return s == "true" || s == "1"
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func auditLog(ctx context.Context, tx pgx.Tx, tenantID, actorID, actorEmail, action string, payload map[string]any) {
	b, _ := json.Marshal(payload)
	if _, err := tx.Exec(ctx,
		`INSERT INTO tenant.tenant_audit (tenant_id, actor_id, actor_email, action, payload) VALUES ($1, $2, $3, $4, $5::jsonb)`,
		tenantID, nullStr(actorID), nullStr(actorEmail), action, string(b)); err != nil {
		slog.Warn("audit_log_failed", slog.String("err", err.Error()))
	}
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// =============================================================================
// Models
// =============================================================================

type tenant struct {
	ID            string         `json:"id"`
	Slug          string         `json:"slug"`
	Name          string         `json:"name"`
	Status        string         `json:"status"`
	Plan          string         `json:"plan"`
	Settings      map[string]any `json:"settings"`
	Branding      map[string]any `json:"branding"`
	SuspendedAt   *time.Time     `json:"suspended_at,omitempty"`
	SuspendedReason string       `json:"suspended_reason,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type tenantDomain struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Domain     string     `json:"domain"`
	Type       string     `json:"type"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	SSLStatus  string     `json:"ssl_status"`
	SSLIssuer  string     `json:"ssl_issuer,omitempty"`
	SSLExpires *time.Time `json:"ssl_expires_at,omitempty"`
	IsPrimary  bool       `json:"is_primary"`
	CreatedAt  time.Time  `json:"created_at"`
}

type tenantSetting struct {
	Key       string    `json:"key"`
	Value     any       `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type tenantBranding struct {
	TenantID       string    `json:"tenant_id"`
	LogoURL        string    `json:"logo_url,omitempty"`
	FaviconURL     string    `json:"favicon_url,omitempty"`
	PrimaryColor   string    `json:"primary_color,omitempty"`
	SecondaryColor string    `json:"secondary_color,omitempty"`
	AccentColor    string    `json:"accent_color,omitempty"`
	FontFamily     string    `json:"font_family,omitempty"`
	CustomCSS      string    `json:"custom_css,omitempty"`
	EmailLogoURL   string    `json:"email_logo_url,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type tenantUsage struct {
	TenantID string `json:"tenant_id"`
	Period   string `json:"period"`
	Metric   string `json:"metric"`
	Value    int64  `json:"value"`
}

type tenantStats struct {
	TenantID      string `json:"tenant_id"`
	Users         int64  `json:"users"`
	Leads         int64  `json:"leads"`
	Deals         int64  `json:"deals"`
	StorageBytes  int64  `json:"storage_bytes"`
	APICallsMonth int64  `json:"api_calls_month"`
	CCU           int    `json:"ccu"`
	Plan          string `json:"plan"`
}

// =============================================================================
// Handlers — tenants
// =============================================================================

type createTenantReq struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Plan  string `json:"plan"`
	Email string `json:"email"`
}

func (s *server) createTenant(c echo.Context) error {
	var req createTenantReq
	if err := c.Bind(&req); err != nil {
		return jsonErr(c, 400, "BAD_REQUEST", err.Error())
	}
	if req.Slug == "" || req.Name == "" {
		return jsonErr(c, 400, "VALIDATION", "slug and name are required")
	}
	if req.Plan == "" {
		req.Plan = "starter"
	}
	ctx := c.Request().Context()
	id := newID()
	now := time.Now()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO tenant.tenants (id, slug, name, status, plan, created_at, updated_at)
		VALUES ($1, $2, $3, 'trial', $4, $5, $5)
	`, id, req.Slug, req.Name, req.Plan, now); err != nil {
		if isUniqueViolation(err) {
			return jsonErr(c, 409, "SLUG_TAKEN", "tenant slug already exists")
		}
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	actor := c.Get("admin_email")
	auditLog(ctx, tx, id, "admin", asString(actor), "created", map[string]any{"plan": req.Plan})
	if err := tx.Commit(ctx); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{"id": id, "slug": req.Slug, "name": req.Name, "plan": req.Plan, "status": "trial"})
}

func (s *server) listTenants(c echo.Context) error {
	ctx := c.Request().Context()
	limit := atoiDefault(c.QueryParam("limit"), 50)
	offset := atoiDefault(c.QueryParam("offset"), 0)
	status := c.QueryParam("status")
	plan := c.QueryParam("plan")
	q := strings.Builder{}
	q.WriteString(`SELECT id::text, slug, name, status, COALESCE(plan,'starter'), settings_json, branding_json,
		created_at, updated_at FROM tenant.tenants WHERE deleted_at IS NULL`)
	args := []any{}
	if status != "" {
		args = append(args, status)
		q.WriteString(fmt.Sprintf(" AND status = $%d", len(args)))
	}
	if plan != "" {
		args = append(args, plan)
		q.WriteString(fmt.Sprintf(" AND plan = $%d", len(args)))
	}
	args = append(args, limit)
	q.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args)))
	args = append(args, offset)
	q.WriteString(fmt.Sprintf(" OFFSET $%d", len(args)))

	rows, err := s.pool.Query(ctx, q.String(), args...)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	defer rows.Close()
	out := []tenant{}
	for rows.Next() {
		t, err := scanTenant(rows)
		if err != nil {
			s.log.Warn("scan_tenant", slog.String("err", err.Error()))
			continue
		}
		out = append(out, t)
	}
	return c.JSON(http.StatusOK, map[string]any{"count": len(out), "tenants": out, "limit": limit, "offset": offset})
}

func (s *server) getTenant(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()
	row := s.pool.QueryRow(ctx, `SELECT id::text, slug, name, status, COALESCE(plan,'starter'), settings_json, branding_json, created_at, updated_at FROM tenant.tenants WHERE id = $1 AND deleted_at IS NULL`, id)
	t, err := scanTenant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return jsonErr(c, 404, "NOT_FOUND", "tenant not found")
		}
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, t)
}

type updateTenantReq struct {
	Name     *string        `json:"name,omitempty"`
	Plan     *string        `json:"plan,omitempty"`
	Status   *string        `json:"status,omitempty"`
	Settings map[string]any `json:"settings,omitempty"`
	Branding map[string]any `json:"branding,omitempty"`
}

func (s *server) updateTenant(c echo.Context) error {
	id := c.Param("id")
	var req updateTenantReq
	if err := c.Bind(&req); err != nil {
		return jsonErr(c, 400, "BAD_REQUEST", err.Error())
	}
	ctx := c.Request().Context()
	settings, _ := json.Marshal(firstMap(req.Settings))
	branding, _ := json.Marshal(firstMap(req.Branding))
	if _, err := s.pool.Exec(ctx, `
		UPDATE tenant.tenants
		   SET name = COALESCE($2, name),
		       plan = COALESCE($3, plan),
		       status = COALESCE($4, status),
		       settings_json = CASE WHEN $5 = '{}'::jsonb THEN settings_json ELSE $5::jsonb END,
		       branding_json = CASE WHEN $6 = '{}'::jsonb THEN branding_json ELSE $6::jsonb END,
		       updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL
	`, id, req.Name, req.Plan, req.Status, string(settings), string(branding)); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	actor := asString(c.Get("admin_email"))
	if err := s.withTenant(ctx, "", func(tx pgx.Tx) error {
		auditLog(ctx, tx, id, "admin", actor, "updated", map[string]any{"name": req.Name, "plan": req.Plan, "status": req.Status})
		return nil
	}); err != nil {
		s.log.Warn("audit_log", slog.String("err", err.Error()))
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (s *server) deleteTenant(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()
	res, err := s.pool.Exec(ctx, `UPDATE tenant.tenants SET status='deleted', deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	if res.RowsAffected() == 0 {
		return jsonErr(c, 404, "NOT_FOUND", "tenant not found")
	}
	actor := asString(c.Get("admin_email"))
	_ = s.withTenant(ctx, "", func(tx pgx.Tx) error {
		auditLog(ctx, tx, id, "admin", actor, "deleted", nil)
		return nil
	})
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (s *server) suspendTenant(c echo.Context) error {
	id := c.Param("id")
	reason := c.QueryParam("reason")
	ctx := c.Request().Context()
	if _, err := s.pool.Exec(ctx, `UPDATE tenant.tenants SET status='suspended', suspended_at = now(), suspended_reason = $2 WHERE id = $1`, id, reason); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	actor := asString(c.Get("admin_email"))
	_ = s.withTenant(ctx, "", func(tx pgx.Tx) error {
		auditLog(ctx, tx, id, "admin", actor, "suspended", map[string]any{"reason": reason})
		return nil
	})
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "status": "suspended"})
}

func (s *server) activateTenant(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()
	if _, err := s.pool.Exec(ctx, `UPDATE tenant.tenants SET status='active', suspended_at = NULL, suspended_reason = NULL WHERE id = $1`, id); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	actor := asString(c.Get("admin_email"))
	_ = s.withTenant(ctx, "", func(tx pgx.Tx) error {
		auditLog(ctx, tx, id, "admin", actor, "activated", nil)
		return nil
	})
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "status": "active"})
}

func (s *server) getTenantStats(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()
	var t tenant
	row := s.pool.QueryRow(ctx, `SELECT id::text, slug, name, status, COALESCE(plan,'starter'), settings_json, branding_json, created_at, updated_at FROM tenant.tenants WHERE id = $1`, id)
	t, err := scanTenant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return jsonErr(c, 404, "NOT_FOUND", "tenant not found")
		}
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	period := time.Now().Format("2006-01")
	rows, err := s.pool.Query(ctx, `SELECT metric, value FROM tenant.tenant_usage WHERE tenant_id = $1 AND period = $2`, id, period)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	defer rows.Close()
	stats := tenantStats{TenantID: id, Plan: t.Plan}
	for rows.Next() {
		var m string
		var v int64
		if err := rows.Scan(&m, &v); err == nil {
			switch m {
			case "users":
				stats.Users = v
			case "leads":
				stats.Leads = v
			case "deals":
				stats.Deals = v
			case "storage_bytes":
				stats.StorageBytes = v
			case "api_calls":
				stats.APICallsMonth = v
			case "ccu":
				stats.CCU = int(v)
			}
		}
	}
	return c.JSON(http.StatusOK, stats)
}

// =============================================================================
// Handlers — domains
// =============================================================================

func (s *server) listDomains(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()
	rows, err := s.pool.Query(ctx, `SELECT id::text, tenant_id::text, domain, type, verified_at, ssl_status, COALESCE(ssl_issuer,''), ssl_expires_at, is_primary, created_at FROM tenant.tenant_domains WHERE tenant_id = $1 ORDER BY created_at`, id)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	defer rows.Close()
	out := []tenantDomain{}
	for rows.Next() {
		var d tenantDomain
		if err := rows.Scan(&d.ID, &d.TenantID, &d.Domain, &d.Type, &d.VerifiedAt, &d.SSLStatus, &d.SSLIssuer, &d.SSLExpires, &d.IsPrimary, &d.CreatedAt); err == nil {
			out = append(out, d)
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"count": len(out), "domains": out})
}

type addDomainReq struct {
	Domain    string `json:"domain"`
	Type      string `json:"type"`
	IsPrimary bool   `json:"is_primary"`
}

func (s *server) addDomain(c echo.Context) error {
	id := c.Param("id")
	var req addDomainReq
	if err := c.Bind(&req); err != nil {
		return jsonErr(c, 400, "BAD_REQUEST", err.Error())
	}
	if req.Domain == "" {
		return jsonErr(c, 400, "VALIDATION", "domain is required")
	}
	if req.Type == "" {
		req.Type = "custom"
	}
	ctx := c.Request().Context()
	did := newID()
	if _, err := s.pool.Exec(ctx, `INSERT INTO tenant.tenant_domains (id, tenant_id, domain, type, is_primary) VALUES ($1, $2, $3, $4, $5)`,
		did, id, req.Domain, req.Type, req.IsPrimary); err != nil {
		if isUniqueViolation(err) {
			return jsonErr(c, 409, "DOMAIN_TAKEN", "domain already mapped")
		}
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	// Invalidate Valkey cache for domain → tenant
	_ = s.rdb.Del(ctx, "domain:"+strings.ToLower(req.Domain)).Err()
	return c.JSON(http.StatusCreated, map[string]any{"id": did, "domain": req.Domain, "type": req.Type, "ssl_status": "pending"})
}

func (s *server) deleteDomain(c echo.Context) error {
	tenantID := c.Param("id")
	domainID := c.Param("domain_id")
	ctx := c.Request().Context()
	var domain string
	row := s.pool.QueryRow(ctx, `SELECT domain FROM tenant.tenant_domains WHERE id = $1 AND tenant_id = $2`, domainID, tenantID)
	if err := row.Scan(&domain); err != nil {
		return jsonErr(c, 404, "NOT_FOUND", "domain not found")
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM tenant.tenant_domains WHERE id = $1 AND tenant_id = $2`, domainID, tenantID); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	_ = s.rdb.Del(ctx, "domain:"+strings.ToLower(domain)).Err()
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (s *server) resolveByDomain(c echo.Context) error {
	domain := strings.ToLower(c.Param("domain"))
	if domain == "" {
		return jsonErr(c, 400, "VALIDATION", "domain required")
	}
	ctx := c.Request().Context()
	// Try Valkey cache
	if v, err := s.rdb.Get(ctx, "domain:"+domain).Result(); err == nil && v != "" {
		return c.JSON(http.StatusOK, map[string]any{"tenant_id": v, "domain": domain, "source": "cache"})
	}
	var tenantID string
	row := s.pool.QueryRow(ctx, `SELECT tenant_id::text FROM tenant.tenant_domains WHERE LOWER(domain) = $1 AND verified_at IS NOT NULL`, domain)
	if err := row.Scan(&tenantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return jsonErr(c, 404, "NOT_FOUND", "no tenant for domain")
		}
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	_ = s.rdb.Set(ctx, "domain:"+domain, tenantID, 5*time.Minute).Err()
	return c.JSON(http.StatusOK, map[string]any{"tenant_id": tenantID, "domain": domain, "source": "db"})
}

// =============================================================================
// Handlers — branding
// =============================================================================

func (s *server) getBranding(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()
	row := s.pool.QueryRow(ctx, `SELECT tenant_id::text, COALESCE(logo_url,''), COALESCE(favicon_url,''), COALESCE(primary_color,''), COALESCE(secondary_color,''), COALESCE(accent_color,''), COALESCE(font_family,''), COALESCE(custom_css,''), COALESCE(email_logo_url,''), updated_at FROM tenant.tenant_branding WHERE tenant_id = $1`, id)
	var b tenantBranding
	if err := row.Scan(&b.TenantID, &b.LogoURL, &b.FaviconURL, &b.PrimaryColor, &b.SecondaryColor, &b.AccentColor, &b.FontFamily, &b.CustomCSS, &b.EmailLogoURL, &b.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// fallback to JSONB snapshot on tenants
			return c.JSON(http.StatusOK, tenantBranding{TenantID: id})
		}
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, b)
}

type brandingReq struct {
	LogoURL        string `json:"logo_url"`
	FaviconURL     string `json:"favicon_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
	FontFamily     string `json:"font_family"`
	CustomCSS      string `json:"custom_css"`
	EmailLogoURL   string `json:"email_logo_url"`
}

func (s *server) upsertBranding(c echo.Context) error {
	id := c.Param("id")
	var req brandingReq
	if err := c.Bind(&req); err != nil {
		return jsonErr(c, 400, "BAD_REQUEST", err.Error())
	}
	ctx := c.Request().Context()
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO tenant.tenant_branding (tenant_id, logo_url, favicon_url, primary_color, secondary_color, accent_color, font_family, custom_css, email_logo_url, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (tenant_id) DO UPDATE SET
			logo_url = EXCLUDED.logo_url,
			favicon_url = EXCLUDED.favicon_url,
			primary_color = EXCLUDED.primary_color,
			secondary_color = EXCLUDED.secondary_color,
			accent_color = EXCLUDED.accent_color,
			font_family = EXCLUDED.font_family,
			custom_css = EXCLUDED.custom_css,
			email_logo_url = EXCLUDED.email_logo_url,
			updated_at = now()
	`, id, nullStr(req.LogoURL), nullStr(req.FaviconURL), nullStr(req.PrimaryColor), nullStr(req.SecondaryColor), nullStr(req.AccentColor), nullStr(req.FontFamily), nullStr(req.CustomCSS), nullStr(req.EmailLogoURL)); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	// Also snapshot into tenants.branding_json for fast reads
	b, _ := json.Marshal(req)
	_, _ = s.pool.Exec(ctx, `UPDATE tenant.tenants SET branding_json = $1::jsonb, updated_at = now() WHERE id = $2`, string(b), id)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (s *server) deleteBranding(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()
	if _, err := s.pool.Exec(ctx, `DELETE FROM tenant.tenant_branding WHERE tenant_id = $1`, id); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	_, _ = s.pool.Exec(ctx, `UPDATE tenant.tenants SET branding_json = '{}'::jsonb, updated_at = now() WHERE id = $1`, id)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

// =============================================================================
// Handlers — usage
// =============================================================================

type recordUsageReq struct {
	Period string `json:"period"`
	Metric string `json:"metric"`
	Value  int64  `json:"value"`
}

func (s *server) recordUsage(c echo.Context) error {
	id := c.Param("id")
	var req recordUsageReq
	if err := c.Bind(&req); err != nil {
		return jsonErr(c, 400, "BAD_REQUEST", err.Error())
	}
	if req.Period == "" {
		req.Period = time.Now().Format("2006-01")
	}
	ctx := c.Request().Context()
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO tenant.tenant_usage (tenant_id, period, metric, value, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (tenant_id, period, metric) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, id, req.Period, req.Metric, req.Value); err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (s *server) getUsage(c echo.Context) error {
	id := c.Param("id")
	period := c.QueryParam("period")
	if period == "" {
		period = time.Now().Format("2006-01")
	}
	ctx := c.Request().Context()
	rows, err := s.pool.Query(ctx, `SELECT metric, value FROM tenant.tenant_usage WHERE tenant_id = $1 AND period = $2`, id, period)
	if err != nil {
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	defer rows.Close()
	out := []tenantUsage{}
	for rows.Next() {
		var u tenantUsage
		u.TenantID = id
		u.Period = period
		if err := rows.Scan(&u.Metric, &u.Value); err == nil {
			out = append(out, u)
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"tenant_id": id, "period": period, "usage": out})
}

// =============================================================================
// Handlers — readiness / health
// =============================================================================

func (s *server) readyz(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
	defer cancel()
	if err := s.pool.Ping(ctx); err != nil {
		return jsonErr(c, 503, "DB_DOWN", err.Error())
	}
	if _, err := s.rdb.Ping(ctx).Result(); err != nil {
		return jsonErr(c, 503, "VALKEY_DOWN", err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "service": serviceName})
}

// =============================================================================
// Connect-RPC (internal)
// =============================================================================

type rpcReq struct {
	Slug string `json:"slug"`
}

func (s *server) rpcGetTenantBySlug(c echo.Context) error {
	var req rpcReq
	if err := c.Bind(&req); err != nil {
		return jsonErr(c, 400, "BAD_REQUEST", err.Error())
	}
	ctx := c.Request().Context()
	row := s.pool.QueryRow(ctx, `SELECT id::text, slug, name, status, COALESCE(plan,'starter'), settings_json, branding_json, created_at, updated_at FROM tenant.tenants WHERE slug = $1 AND deleted_at IS NULL`, req.Slug)
	t, err := scanTenant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return jsonErr(c, 404, "NOT_FOUND", "tenant not found")
		}
		return jsonErr(c, 500, "DB_ERROR", err.Error())
	}
	return c.JSON(http.StatusOK, t)
}

type resolveReq struct {
	Domain string `json:"domain"`
}

func (s *server) rpcResolveDomain(c echo.Context) error {
	var req resolveReq
	if err := c.Bind(&req); err != nil {
		return jsonErr(c, 400, "BAD_REQUEST", err.Error())
	}
	domain := strings.ToLower(req.Domain)
	ctx := c.Request().Context()
	if v, err := s.rdb.Get(ctx, "domain:"+domain).Result(); err == nil && v != "" {
		return c.JSON(http.StatusOK, map[string]any{"tenant_id": v})
	}
	var tenantID string
	row := s.pool.QueryRow(ctx, `SELECT tenant_id::text FROM tenant.tenant_domains WHERE LOWER(domain) = $1 AND verified_at IS NOT NULL`, domain)
	if err := row.Scan(&tenantID); err != nil {
		return jsonErr(c, 404, "NOT_FOUND", "no tenant for domain")
	}
	_ = s.rdb.Set(ctx, "domain:"+domain, tenantID, 5*time.Minute).Err()
	return c.JSON(http.StatusOK, map[string]any{"tenant_id": tenantID})
}

type quotaReq struct {
	TenantID string `json:"tenant_id"`
	Metric   string `json:"metric"`
}

func (s *server) rpcGetQuota(c echo.Context) error {
	var req quotaReq
	if err := c.Bind(&req); err != nil {
		return jsonErr(c, 400, "BAD_REQUEST", err.Error())
	}
	ctx := c.Request().Context()
	row := s.pool.QueryRow(ctx, `
		SELECT q.max_users, q.max_storage_mb, q.max_api_calls, q.max_ccu, q.features
		FROM tenant.plan_quotas q
		JOIN tenant.tenants t ON t.plan = q.plan
		WHERE t.id = $1
	`, req.TenantID)
	var mu, ms, ma, mc int
	var features map[string]any
	if err := row.Scan(&mu, &ms, &ma, &mc, &features); err != nil {
		return jsonErr(c, 404, "NOT_FOUND", "quota not found")
	}
	limit := int64(0)
	switch req.Metric {
	case "users":
		limit = int64(mu)
	case "storage_bytes":
		limit = int64(ms) * 1024 * 1024
	case "api_calls":
		limit = int64(ma)
	case "ccu":
		limit = int64(mc)
	default:
		return jsonErr(c, 400, "BAD_METRIC", "metric must be one of users, storage_bytes, api_calls, ccu")
	}
	period := time.Now().Format("2006-01")
	var used int64
	_ = s.pool.QueryRow(ctx, `SELECT value FROM tenant.tenant_usage WHERE tenant_id = $1 AND period = $2 AND metric = $3`, req.TenantID, period, req.Metric).Scan(&used)
	return c.JSON(http.StatusOK, map[string]any{"tenant_id": req.TenantID, "metric": req.Metric, "limit": limit, "used": used, "features": features})
}

// =============================================================================
// Routing
// =============================================================================

func (s *server) routes(e *echo.Echo) {
	e.GET("/healthz", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName}) })
	e.GET("/readyz", s.readyz)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	v1 := e.Group("/v1")
	v1.POST("/tenants", s.createTenant, s.adminAuthMW())
	v1.GET("/tenants", s.listTenants, s.adminAuthMW())
	v1.GET("/tenants/:id", s.getTenant, s.adminAuthMW())
	v1.PUT("/tenants/:id", s.updateTenant, s.adminAuthMW())
	v1.DELETE("/tenants/:id", s.deleteTenant, s.adminAuthMW())
	v1.POST("/tenants/:id/suspend", s.suspendTenant, s.adminAuthMW())
	v1.POST("/tenants/:id/activate", s.activateTenant, s.adminAuthMW())
	v1.GET("/tenants/:id/stats", s.getTenantStats, s.adminAuthMW())
	v1.GET("/tenants/by-domain/:domain", s.resolveByDomain)
	v1.GET("/tenants/:id/domains", s.listDomains, s.requireTenantMW())
	v1.POST("/tenants/:id/domains", s.addDomain, s.adminAuthMW())
	v1.DELETE("/tenants/:id/domains/:domain_id", s.deleteDomain, s.adminAuthMW())
	v1.GET("/tenants/:id/branding", s.getBranding)
	v1.POST("/tenants/:id/branding", s.upsertBranding, s.adminAuthMW())
	v1.DELETE("/tenants/:id/branding", s.deleteBranding, s.adminAuthMW())
	v1.GET("/tenants/:id/usage", s.getUsage)
	v1.POST("/tenants/:id/usage", s.recordUsage, s.requireTenantMW())

	// Connect-RPC style endpoints
	rpc := e.Group("/rpc/tenant.v1.TenantService")
	rpc.POST("/GetTenantBySlug", s.rpcGetTenantBySlug)
	rpc.POST("/ResolveDomain", s.rpcResolveDomain)
	rpc.POST("/GetQuota", s.rpcGetQuota)
}

// =============================================================================
// Helpers — DB scanning
// =============================================================================

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTenant(r rowScanner) (tenant, error) {
	var t tenant
	var settingsJSON, brandingJSON []byte
	if err := r.Scan(&t.ID, &t.Slug, &t.Name, &t.Status, &t.Plan, &settingsJSON, &brandingJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return t, err
	}
	if len(settingsJSON) > 0 {
		_ = json.Unmarshal(settingsJSON, &t.Settings)
	}
	if len(brandingJSON) > 0 {
		_ = json.Unmarshal(brandingJSON, &t.Branding)
	}
	if t.Settings == nil {
		t.Settings = map[string]any{}
	}
	if t.Branding == nil {
		t.Branding = map[string]any{}
	}
	return t, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key value")
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func firstMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	// Defensive copy so callers can mutate without side-effecting the original
	// (and so a nil receiver becomes a non-nil empty map).
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// =============================================================================
// main
// =============================================================================

func main() {
	cfg := loadConfig()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)
	logger.Info("starting", slog.String("service", serviceName), slog.String("addr", cfg.HTTPAddr), slog.String("env", cfg.Environment))

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracer, err := initTracer(rootCtx, cfg.OTLPEndpoint)
	if err != nil {
		logger.Warn("tracer init failed", slog.String("err", err.Error()))
	}

	pool, err := newPool(rootCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db pool", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	opt, err := redis.ParseURL(cfg.ValkeyURL)
	if err != nil {
		logger.Error("redis parse", slog.String("err", err.Error()))
		os.Exit(1)
	}
	rdb := redis.NewClient(opt)
	defer func() { _ = rdb.Close() }()
	if _, err := rdb.Ping(rootCtx).Result(); err != nil {
		logger.Warn("redis ping failed", slog.String("err", err.Error()))
	}

	srv := &server{pool: pool, rdb: rdb, cfg: cfg, log: logger, ready: &sync.RWMutex{}}
	if err := srv.runMigrations(rootCtx); err != nil {
		logger.Error("migrations", slog.String("err", err.Error()))
		os.Exit(1)
	}
	srv.ok = true

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(srv.recoveryMW())
	e.Use(traceMW())
	e.Use(loggingMW())
	e.Use(metricsMW())
	e.Use(corsMW())
	e.Use(securityHeadersMW())
	e.Use(otelecho.Middleware(serviceName))

	srv.routes(e)

	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: e, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen", slog.String("err", err.Error()))
			cancel()
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sig:
		logger.Info("shutdown signal")
	case <-rootCtx.Done():
	}

	shutdownCtx, sCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer sCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http shutdown", slog.String("err", err.Error()))
	}
	if err := shutdownTracer(shutdownCtx); err != nil {
		logger.Warn("tracer shutdown", slog.String("err", err.Error()))
	}
	logger.Info("bye")
}
