// Package db cung cấp PostgreSQL connection pool (pgxpool), RLS helpers,
// transaction wrapper với retry, migration runner, và generic repository.
//
// Mọi service nên dùng:
//   pool, _ := db.NewPool(ctx, db.Config{...})
//   db.WithTx(ctx, pool, func(tx pgx.Tx) error { ... })
package db

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ctxKey định danh giá trị lưu trong context.
type ctxKey string

const (
	TenantIDKey  ctxKey = "tenant_id"
	UserIDKey    ctxKey = "user_id"
	IsAdminKey   ctxKey = "is_super_admin"
	IsAuthKey    ctxKey = "is_authenticated"
	RequestIDKey ctxKey = "request_id"
	TraceIDKey   ctxKey = "trace_id"
	BypassRLSKey ctxKey = "bypass_rls"
)

// Config cấu hình pool.
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthCheckPeriod time.Duration
	ApplicationName string
	StatementCacheCapacity int
}

// DSN trả về connection string.
func (c Config) DSN() string {
	host := c.Host
	if c.Port > 0 {
		host = fmt.Sprintf("%s:%d", c.Host, c.Port)
	}
	params := url.Values{}
	params.Set("sslmode", c.SSLMode)
	if c.ApplicationName != "" {
		params.Set("application_name", c.ApplicationName)
	}
	if c.StatementCacheCapacity > 0 {
		params.Set("statement_cache_capacity", fmt.Sprintf("%d", c.StatementCacheCapacity))
	}
	return fmt.Sprintf("postgres://%s:%s@%s/%s?%s",
		url.QueryEscape(c.User), url.QueryEscape(c.Password), host, c.Database, params.Encode())
}

// WithDefaults điền defaults nếu field rỗng.
func (c Config) WithDefaults() Config {
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	if c.MaxConns == 0 {
		c.MaxConns = 25
	}
	if c.MinConns == 0 {
		c.MinConns = 2
	}
	if c.MaxConnLifetime == 0 {
		c.MaxConnLifetime = 30 * time.Minute
	}
	if c.MaxConnIdleTime == 0 {
		c.MaxConnIdleTime = 5 * time.Minute
	}
	if c.HealthCheckPeriod == 0 {
		c.HealthCheckPeriod = 30 * time.Second
	}
	return c
}

// NewPool tạo pgxpool với config + metrics.
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	cfg = cfg.WithDefaults()
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("db: parse DSN: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	if cfg.ApplicationName != "" {
		poolCfg.ConnConfig.RuntimeParams["application_name"] = cfg.ApplicationName
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("db: new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

// PoolStats trả về metrics cho Prometheus collector.
type PoolStats struct {
	Active int32 `json:"active"`
	Idle   int32 `json:"idle"`
	Total  int32 `json:"total"`
	Max    int32 `json:"max"`
}

// Stats của pool.
func Stats(pool *pgxpool.Pool) PoolStats {
	s := pool.Stat()
	return PoolStats{
		Active: s.AcquiredConns(),
		Idle:   s.IdleConns(),
		Total:  s.TotalConns(),
		Max:    s.MaxConns(),
	}
}