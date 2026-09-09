// Tests for db rls helpers (rls.go).
package db

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestExtra_RLSContext_Empty(t *testing.T) {
	// Verify TenantIDFromContext returns empty for empty context
	got := TenantIDFromContext(context.Background())
	if got != "" {
		t.Errorf("expected empty: %s", got)
	}
}

func TestExtra_RLSContext_UserEmpty(t *testing.T) {
	// Verify UserIDFromContext returns empty for empty context
	got := UserIDFromContext(context.Background())
	if got != "" {
		t.Errorf("expected empty: %s", got)
	}
}

func TestExtra_IsSuperAdmin_NoKey(t *testing.T) {
	if IsSuperAdmin(context.Background()) {
		t.Error("expected false for no context key")
	}
}

func TestExtra_BypassRLSFromContext_True(t *testing.T) {
	ctx := WithBypassRLS(context.Background())
	if !BypassRLSFromContext(ctx) {
		t.Error("expected true after WithBypassRLS")
	}
}

func TestExtra_BypassRLSFromContext_False(t *testing.T) {
	if BypassRLSFromContext(context.Background()) {
		t.Error("expected false")
	}
}

func TestExtra_SetTenantContext(t *testing.T) {
	ctx := SetTenantContext(context.Background(), "tenant-1", "user-1", false)
	if TenantIDFromContext(ctx) != "tenant-1" {
		t.Error("tenant not set")
	}
	if UserIDFromContext(ctx) != "user-1" {
		t.Error("user not set")
	}
}

func TestExtra_SetTenantContext_AsAdmin(t *testing.T) {
	ctx := SetTenantContext(context.Background(), "tenant-1", "user-1", true)
	if !IsSuperAdmin(ctx) {
		t.Error("admin not set")
	}
}

func TestExtra_RLSSetupSQL_NotEmpty(t *testing.T) {
	if RLSSetupSQL == "" {
		t.Error("RLSSetupSQL should not be empty")
	}
	if !strings.Contains(RLSSetupSQL, "ALTER DATABASE") {
		t.Error("RLSSetupSQL should contain ALTER DATABASE")
	}
	if !strings.Contains(RLSSetupSQL, "CREATE OR REPLACE FUNCTION") {
		t.Error("RLSSetupSQL should contain CREATE OR REPLACE FUNCTION")
	}
}

func TestExtra_ExamplePolicySQL_NotEmpty(t *testing.T) {
	if ExamplePolicySQL == "" {
		t.Error("ExamplePolicySQL should not be empty")
	}
	if !strings.Contains(ExamplePolicySQL, "ROW LEVEL SECURITY") {
		t.Error("ExamplePolicySQL should mention ROW LEVEL SECURITY")
	}
}

func TestExtra_IsPgError_Nil(t *testing.T) {
	_, ok := IsPgError(nil)
	if ok {
		t.Error("expected false for nil error")
	}
}

func TestExtra_IsPgError_RegularError(t *testing.T) {
	_, ok := IsPgError(errors.New("regular error"))
	if ok {
		t.Error("expected false for non-pg error")
	}
}

func TestExtra_IsPgError_PgError(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:       "23505",
		Message:    "unique violation",
		Detail:     "key already exists",
		TableName:  "users",
	}
	result, ok := IsPgError(pgErr)
	if !ok {
		t.Error("expected true for pg error")
	}
	if result == nil {
		t.Error("result should not be nil")
	}
	if result.Code != "23505" {
		t.Errorf("got code %s", result.Code)
	}
}

func TestExtra_IsPgError_WrappedPgError(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:    "23505",
		Message: "unique violation",
	}
	wrapped := &wrappedError{msg: "wrapped", inner: pgErr}
	_, ok := IsPgError(wrapped)
	if !ok {
		t.Error("expected true for wrapped pg error")
	}
}

func TestExtra_WrapPoolWithRLS(t *testing.T) {
	// Can't fully test without a real pool, just verify it doesn't panic
	p := WrapPoolWithRLS(nil)
	if p == nil {
		t.Error("WrapPoolWithRLS should return non-nil")
	}
}

func TestExtra_TenantIDFromContext_Set(t *testing.T) {
	ctx := SetTenantContext(context.Background(), "tenant-123", "", false)
	if TenantIDFromContext(ctx) != "tenant-123" {
		t.Errorf("got %s", TenantIDFromContext(ctx))
	}
}

func TestExtra_UserIDFromContext_Set(t *testing.T) {
	ctx := SetTenantContext(context.Background(), "", "user-123", false)
	if UserIDFromContext(ctx) != "user-123" {
		t.Errorf("got %s", UserIDFromContext(ctx))
	}
}

func TestExtra_PoolWithRLS_Pool(t *testing.T) {
	p := WrapPoolWithRLS(nil)
	// p.Pool() should return the underlying pool (nil in this test)
	if p.Pool() != nil {
		t.Error("Pool should return nil since we passed nil")
	}
}

// wrappedError wraps an error with a message (for testing IsPgError)
type wrappedError struct {
	msg   string
	inner error
}

func (w *wrappedError) Error() string { return w.msg + ": " + w.inner.Error() }
func (w *wrappedError) Unwrap() error { return w.inner }
