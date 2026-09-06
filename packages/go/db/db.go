// Package db cung cấp PostgreSQL connection với RLS enforcement.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// ctxKey type for context values.
type ctxKey string

const (
	TenantIDKey   ctxKey = "tenant_id"
	UserIDKey     ctxKey = "user_id"
	IsAdminKey    ctxKey = "is_super_admin"
	IsAuthKey     ctxKey = "is_authenticated"
	RequestIDKey  ctxKey = "request_id"
)

// Config chứa cấu hình database connection.
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN trả về connection string.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// New tạo connection pool mới.
func New(cfg Config) (*sql.DB, error) {
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 25
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 5
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 5 * time.Minute
	}

	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}

// WithRLS bắt đầu transaction với RLS context.
func WithRLS(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	tenantID, _ := ctx.Value(TenantIDKey).(string)
	userID, _ := ctx.Value(UserIDKey).(string)
	isAdmin, _ := ctx.Value(IsAdminKey).(bool)

	if tenantID != "" {
		if _, err := tx.ExecContext(ctx,
			fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", escapeValue(tenantID))); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("set tenant: %w", err)
		}
	}
	if userID != "" {
		if _, err := tx.ExecContext(ctx,
			fmt.Sprintf("SET LOCAL app.current_user_id = '%s'", escapeValue(userID))); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("set user: %w", err)
		}
	}
	if isAdmin {
		if _, err := tx.ExecContext(ctx, "SET LOCAL app.is_super_admin = 'true'"); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("set admin: %w", err)
		}
	}

	return tx, nil
}

// SetTenantContext gắn tenant_id vào context.
func SetTenantContext(ctx context.Context, tenantID, userID string, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, UserIDKey, userID)
	if isAdmin {
		ctx = context.WithValue(ctx, IsAdminKey, true)
	}
	ctx = context.WithValue(ctx, IsAuthKey, true)
	return ctx
}

// TenantIDFromContext extracts tenant_id from context.
func TenantIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(TenantIDKey).(string)
	return v
}

// UserIDFromContext extracts user_id from context.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(UserIDKey).(string)
	return v
}

// IsSuperAdmin checks if context is super admin.
func IsSuperAdmin(ctx context.Context) bool {
	v, _ := ctx.Value(IsAdminKey).(bool)
	return v
}

// escapeValue escapes single quotes for SQL safety (RLS context).
func escapeValue(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' {
			result = append(result, '\'', '\'')
		} else {
			result = append(result, c)
		}
	}
	return string(result)
}
