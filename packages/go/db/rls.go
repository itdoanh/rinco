// Package db - Row-Level Security helpers.
//
// PostgreSQL RLS yêu cầu SET app.current_tenant_id, app.current_user_id,
// app.is_super_admin, app.bypass_rls trước khi queries chạy.
// Các helper này wrap pgxpool + pgx.Tx để tự động set các biến này
// dựa trên values có trong context.
//
// Usage:
//
//   pool, _ := db.NewPool(...)
//   ctx := db.SetTenantContext(ctx, tenantID, userID, false)
//   db.WithRLS(ctx, pool, func(tx pgx.Tx) error {
//       // queries inside auto have RLS applied
//   })
//
//   // Hoặc gắn RLS vào mọi acquire:
//   wrapped := db.WrapPoolWithRLS(pool, ctx)
//   rows, _ := wrapped.Query(ctx, "SELECT * FROM leads.leads")
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolWithRLS wrap pgxpool với hook trước khi acquire connection để
// tự động set RLS context variables.
type PoolWithRLS struct {
	pool    *pgxpool.Pool
	ctx     context.Context
}

// WrapPoolWithRLS trả về pool wrapper. Mỗi Acquire sẽ set RLS context
// dựa trên ctx truyền vào (nếu ctx không có values thì no-op).
func WrapPoolWithRLS(pool *pgxpool.Pool) *PoolWithRLS {
	return &PoolWithRLS{pool: pool}
}

// Acquire lấy connection từ pool và set RLS context từ ctx parameter.
func (p *PoolWithRLS) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	if err := applyRLSContext(ctx, conn.Conn()); err != nil {
		conn.Release()
		return nil, fmt.Errorf("rls: apply: %w", err)
	}
	return conn, nil
}

// AcquireFunc tương tự Acquire nhưng dùng callback.
func (p *PoolWithRLS) AcquireFunc(ctx context.Context, fn func(*pgxpool.Conn) error) error {
	conn, err := p.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	return fn(conn)
}

// Pool trả về underlying pool.
func (p *PoolWithRLS) Pool() *pgxpool.Pool {
	return p.pool
}

// applyRLSContext set các biến session cho RLS.
func applyRLSContext(ctx context.Context, conn *pgx.Conn) error {
	tenantID := TenantIDFromContext(ctx)
	userID := UserIDFromContext(ctx)
	isAdmin := IsSuperAdmin(ctx)
	bypass := BypassRLSFromContext(ctx)

	if tenantID == "" && userID == "" && !isAdmin && !bypass {
		// Nothing to set, skip.
		return nil
	}

	// SET LOCAL chỉ áp dụng trong transaction; dùng set_config() thay thế.
	if tenantID != "" {
		if _, err := conn.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID); err != nil {
			return err
		}
	}
	if userID != "" {
		if _, err := conn.Exec(ctx, "SELECT set_config('app.current_user_id', $1, false)", userID); err != nil {
			return err
		}
	}
	if isAdmin {
		if _, err := conn.Exec(ctx, "SELECT set_config('app.is_super_admin', 'true', false)"); err != nil {
			return err
		}
	}
	if bypass {
		if _, err := conn.Exec(ctx, "SELECT set_config('app.bypass_rls', 'true', false)"); err != nil {
			return err
		}
	}
	return nil
}

// RLSSetupSQL trả về DDL bootstrap cho RLS (idempotent).
// Migration runner sẽ apply file này trước khi tạo các bảng.
//
// Bao gồm:
//   - Tạo roles: app_user, app_admin, app_super_admin
//   - Tạo helper functions: app_current_tenant_id(), app_current_user_id(), ...
//   - Tạo GUC defaults
const RLSSetupSQL = `
-- GUC defaults
ALTER DATABASE rinco SET app.current_tenant_id = '';
ALTER DATABASE rinco SET app.current_user_id = '';
ALTER DATABASE rinco SET app.is_super_admin = 'false';
ALTER DATABASE rinco SET app.bypass_rls = 'false';

-- Helper functions
CREATE OR REPLACE FUNCTION app_current_tenant_id() RETURNS uuid AS $$
BEGIN
  RETURN NULLIF(current_setting('app.current_tenant_id', true), '')::uuid;
EXCEPTION WHEN OTHERS THEN
  RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION app_current_user_id() RETURNS uuid AS $$
BEGIN
  RETURN NULLIF(current_setting('app.current_user_id', true), '')::uuid;
EXCEPTION WHEN OTHERS THEN
  RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION app_is_super_admin() RETURNS boolean AS $$
BEGIN
  RETURN COALESCE(NULLIF(current_setting('app.is_super_admin', true), ''), 'false')::boolean;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION app_bypass_rls() RETURNS boolean AS $$
BEGIN
  RETURN COALESCE(NULLIF(current_setting('app.bypass_rls', true), ''), 'false')::boolean;
END;
$$ LANGUAGE plpgsql STABLE;
`

// ExamplePolicySQL demo một RLS policy mẫu.
const ExamplePolicySQL = `
-- Ví dụ: enable RLS cho bảng leads.leads
ALTER TABLE leads.leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads.leads FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON leads.leads
  USING (
    app_bypass_rls() OR
    app_is_super_admin() OR
    tenant_id = app_current_tenant_id()
  )
  WITH CHECK (
    app_bypass_rls() OR
    app_is_super_admin() OR
    tenant_id = app_current_tenant_id()
  );
`

// IsPgError trả về *pgconn.PgError nếu err là PostgreSQL error.
func IsPgError(err error) (*pgconn.PgError, bool) {
	var pgErr *pgconn.PgError
	if errAs := pgx.ErrTxClosed; errAs != nil {
		_ = errAs
	}
	if err == nil {
		return nil, false
	}
	pgErr = nil
	for {
		if e, ok := err.(*pgconn.PgError); ok {
			return e, true
		}
		type unwrap interface{ Unwrap() error }
		u, ok := err.(unwrap)
		if !ok {
			break
		}
		err = u.Unwrap()
		if err == nil {
			break
		}
	}
	return pgErr, false
}