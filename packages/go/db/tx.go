// Package db - transaction helper với retry on serialization failure + context propagation.
//
// Usage:
//
//	db.WithTx(ctx, pool, func(tx pgx.Tx) error {
//	    _, err := tx.Exec(ctx, "INSERT ...", args...)
//	    return err
//	})
//
// Mặc định retry tối đa 3 lần khi gặp SQLSTATE 40001 (serialization_failure)
// hoặc 40P01 (deadlock_detected).
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxOptions cho phép override isolation level + retries.
type TxOptions struct {
	IsoLevel    pgx.TxIsoLevel
	AccessMode  pgx.TxAccessMode
	MaxRetries  int
	RetryDelay  time.Duration
	OnRetry     func(attempt int, err error)
}

// defaults fills zero-value fields with safe defaults. The receiver is a
// value so the function is not a method.
func (o TxOptions) defaults() TxOptions {
	if o.MaxRetries == 0 {
		o.MaxRetries = 3
	}
	if o.RetryDelay == 0 {
		o.RetryDelay = 50 * time.Millisecond
	}
	if o.IsoLevel == "" {
		o.IsoLevel = pgx.ReadCommitted
	}
	return o
}

// SerializableTxOptions preset cho high-stakes transactions.
func SerializableTxOptions() TxOptions {
	return TxOptions{
		IsoLevel:   pgx.Serializable,
		MaxRetries: 5,
		RetryDelay: 100 * time.Millisecond,
	}
}

// isRetryable kiểm tra lỗi có nên retry không (SQLSTATE 40001 hoặc 40P01).
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	// pgx-specific serialization errors
	if errors.Is(err, pgx.ErrTxClosed) {
		return false
	}
	return false
}

// WithTx chạy fn trong transaction với retry on serialization failure.
//
// Trước khi fn chạy, sẽ set RLS context (tenant_id, user_id, is_super_admin, bypass_rls).
// Tất cả errors từ fn sẽ rollback transaction (trừ khi fn gọi panic).
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error, opts ...TxOptions) error {
	o := TxOptions{}.defaults()
	if len(opts) > 0 {
		o = opts[0].defaults()
	}

	var lastErr error
	for attempt := 0; attempt <= o.MaxRetries; attempt++ {
		if attempt > 0 {
			if o.OnRetry != nil {
				o.OnRetry(attempt, lastErr)
			}
			// Exponential backoff
			delay := o.RetryDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := func() error {
			tx, err := pool.BeginTx(ctx, pgx.TxOptions{
				IsoLevel:   o.IsoLevel,
				AccessMode: o.AccessMode,
			})
			if err != nil {
				return fmt.Errorf("begin tx: %w", err)
			}
			defer func() {
				if p := recover(); p != nil {
					_ = tx.Rollback(ctx)
					panic(p)
				}
			}()

			// Set RLS context (inside transaction)
			if err := setRLSContext(ctx, tx); err != nil {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("set rls: %w", err)
			}

			if err := fn(tx); err != nil {
				_ = tx.Rollback(ctx)
				return err
			}
			return tx.Commit(ctx)
		}()

		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetryable(err) {
			return err
		}
	}
	return fmt.Errorf("withTx: max retries exceeded: %w", lastErr)
}

// setRLSContext set các biến session để RLS policies dùng.
// Đặt trong transaction nên SET LOCAL tự hết hạn khi commit/rollback.
func setRLSContext(ctx context.Context, tx pgx.Tx) error {
	tenantID := TenantIDFromContext(ctx)
	userID := UserIDFromContext(ctx)
	isAdmin := IsSuperAdmin(ctx)
	bypass := BypassRLSFromContext(ctx)

	if tenantID != "" {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
			return err
		}
	}
	if userID != "" {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", userID); err != nil {
			return err
		}
	}
	if bypass {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.bypass_rls', 'true', true)"); err != nil {
			return err
		}
	}
	if isAdmin {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.is_super_admin', 'true', true)"); err != nil {
			return err
		}
	}
	return nil
}

// ===== Context helpers =====

// SetTenantContext gắn tenant_id và user_id vào context.
// Trả về context mới (không mutate input).
func SetTenantContext(ctx context.Context, tenantID, userID string, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, UserIDKey, userID)
	if isAdmin {
		ctx = context.WithValue(ctx, IsAdminKey, true)
	}
	ctx = context.WithValue(ctx, IsAuthKey, true)
	return ctx
}

// WithBypassRLS đánh dấu context để bypass Row-Level Security.
func WithBypassRLS(ctx context.Context) context.Context {
	return context.WithValue(ctx, BypassRLSKey, true)
}

// TenantIDFromContext extracts tenant_id từ context.
func TenantIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(TenantIDKey).(string)
	return v
}

// UserIDFromContext extracts user_id từ context.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(UserIDKey).(string)
	return v
}

// IsSuperAdmin checks if context is super admin.
func IsSuperAdmin(ctx context.Context) bool {
	v, _ := ctx.Value(IsAdminKey).(bool)
	return v
}

// IsAuthenticated checks if context có authentication info.
func IsAuthenticated(ctx context.Context) bool {
	v, _ := ctx.Value(IsAuthKey).(bool)
	return v
}

// BypassRLSFromContext checks if context bypass RLS.
func BypassRLSFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(BypassRLSKey).(bool)
	return v
}