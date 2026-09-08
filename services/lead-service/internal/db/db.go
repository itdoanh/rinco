// Package db provides database access patterns for lead service.
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetRLS sets Row Level Security context variables on a pool.
//
// KNOWN LIMITATION: SET LOCAL outside a tx has no effect. This applies
// only the state to the connection's *current* statement; subsequent
// statements on a different pool conn see NULL settings. Use SetRLSTx
// for proper RLS isolation.
//
// SQL injection is prevented via UUID validation + parameterized queries.
func SetRLS(ctx context.Context, pool *pgxpool.Pool, tenantID, userID string, isAdmin bool) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}
	if _, err := uuid.Parse(tenantID); err != nil {
		return fmt.Errorf("invalid tenant_id: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		return fmt.Errorf("set tenant_id: %w", err)
	}
	if userID != "" {
		if _, err := uuid.Parse(userID); err != nil {
			return fmt.Errorf("invalid user_id: %w", err)
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", userID); err != nil {
			return fmt.Errorf("set user_id: %w", err)
		}
	}
	adminVal := "false"
	if isAdmin {
		adminVal = "true"
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.is_admin', $1, true)", adminVal); err != nil {
		return fmt.Errorf("set is_admin: %w", err)
	}
	return tx.Commit(ctx)
}

// SetRLSTx sets RLS on a transaction. This is the proper RLS path: the
// variables are bound to the tx's lifetime and reset on commit/rollback.
//
// All inputs validated as UUIDs; queries parameterized via set_config().
func SetRLSTx(ctx context.Context, tx pgx.Tx, tenantID, userID string, isAdmin bool) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}
	if _, err := uuid.Parse(tenantID); err != nil {
		return fmt.Errorf("invalid tenant_id: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		return fmt.Errorf("set tenant_id: %w", err)
	}
	if userID != "" {
		if _, err := uuid.Parse(userID); err != nil {
			return fmt.Errorf("invalid user_id: %w", err)
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", userID); err != nil {
			return fmt.Errorf("set user_id: %w", err)
		}
	}
	adminVal := "false"
	if isAdmin {
		adminVal = "true"
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.is_admin', $1, true)", adminVal); err != nil {
		return fmt.Errorf("set is_admin: %w", err)
	}
	return nil
}

// ExecTx executes fn within a transaction with RLS applied.
func ExecTx(ctx context.Context, pool *pgxpool.Pool, tenantID, userID string, isAdmin bool, fn func(ctx context.Context, tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := SetRLSTx(ctx, tx, tenantID, userID, isAdmin); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			slog.Error("tx rollback failed", slog.String("error", rbErr.Error()))
		}
		return err
	}

	return tx.Commit(ctx)
}
