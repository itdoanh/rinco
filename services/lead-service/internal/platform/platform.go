// Package platform wraps shared infrastructure concerns for lead service.
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
)

// InitLogger configures slog with JSON output.
func InitLogger(service, env, version string) *slog.Logger {
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
		slog.String("service", service),
		slog.String("env", env),
		slog.String("version", version),
	)
	slog.SetDefault(l)
	return l
}

// Getenv reads env, falling back to def.
func Getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// GetenvInt reads int env.
func GetenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return def
}

// GetenvBool reads bool env.
func GetenvBool(key string, def bool) bool {
	v := strings.ToLower(os.Getenv(key))
	if v == "" {
		return def
	}
	return v == "true" || v == "1" || v == "yes"
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// DBConfig holds Postgres pool configuration.
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// DBConfigFromEnv populates from env vars.
func DBConfigFromEnv(prefix string) DBConfig {
	dsn := getenv(prefix+"DATABASE_URL", "")
	if dsn == "" {
		dsn = "postgres://rinco:rinco_dev_password@localhost:5432/rinco_lead?sslmode=disable"
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return DBConfig{Host: "localhost", Port: 5432, User: "rinco", Password: "rinco_dev_password", Database: "rinco_lead", SSLMode: "disable"}
	}
	host, port := u.Hostname(), 5432
	if p := u.Port(); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}
	pw, _ := u.User.Password()
	db := strings.TrimPrefix(u.Path, "/")
	mode := "disable"
	if v := u.Query().Get("sslmode"); v != "" {
		mode = v
	}
	return DBConfig{Host: host, Port: port, User: u.User.Username(), Password: pw, Database: db, SSLMode: mode}
}

// OpenPool opens a pgxpool.Pool with sane defaults.
func OpenPool(ctx context.Context, cfg DBConfig, applicationName string) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.QueryEscape(cfg.User), url.QueryEscape(cfg.Password),
		cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode)
	if applicationName != "" {
		dsn += "&application_name=" + url.QueryEscape(applicationName)
	}
	pcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx parse: %w", err)
	}
	pcfg.MaxConns = 25
	pcfg.MinConns = 2
	pcfg.MaxConnLifetime = 30 * time.Minute
	pcfg.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgx ping: %w", err)
	}
	return pool, nil
}

// === Context helpers ===

type ctxKey string

const (
	tenantIDKey ctxKey = "tenant_id"
	userIDKey   ctxKey = "user_id"
	isAdminKey  ctxKey = "is_admin"
	traceIDKey  ctxKey = "trace_id"
)

// WithTenant returns a context with tenant_id + user_id + isAdmin set.
func WithTenant(ctx context.Context, tenantID, userID string, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, tenantIDKey, tenantID)
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, isAdminKey, isAdmin)
	return ctx
}

// TenantFromContext extracts tenant_id.
func TenantFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}

// UserFromContext extracts user_id.
func UserFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// IsAdminFromContext returns whether current context is super-admin.
func IsAdminFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(isAdminKey).(bool)
	return v
}

// WithTraceID attaches trace id to context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// TraceFromContext returns current trace id.
func TraceFromContext(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}

// IsNotFound reports whether err represents a "not found" condition.
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
