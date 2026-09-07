// Package db provides database access patterns for lead service.
package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetRLS sets Row Level Security context variables.
func SetRLS(ctx context.Context, pool *pgxpool.Pool, tenantID, userID string, isAdmin bool) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)); err != nil {
		return fmt.Errorf("set tenant_id: %w", err)
	}
	if userID != "" {
		if _, err := pool.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_user_id = '%s'", userID)); err != nil {
			return fmt.Errorf("set user_id: %w", err)
		}
	}
	adminVal := "false"
	if isAdmin {
		adminVal = "true"
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf("SET LOCAL app.is_admin = '%s'", adminVal)); err != nil {
		return fmt.Errorf("set is_admin: %w", err)
	}
	return nil
}

// SetRLSTx sets RLS on a transaction.
func SetRLSTx(ctx context.Context, tx pgx.Tx, tenantID, userID string, isAdmin bool) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)); err != nil {
		return fmt.Errorf("set tenant_id: %w", err)
	}
	if userID != "" {
		if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_user_id = '%s'", userID)); err != nil {
			return fmt.Errorf("set user_id: %w", err)
		}
	}
	adminVal := "false"
	if isAdmin {
		adminVal = "true"
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.is_admin = '%s'", adminVal)); err != nil {
		return fmt.Errorf("set is_admin: %w", err)
	}
	return nil
}

// ExecTx executes fn within a transaction.
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
