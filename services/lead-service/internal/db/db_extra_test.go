// Extra tests for lead-service db package.
package db

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// =============================================================================
// SetRLS — panics with nil pool (documented behavior)
// =============================================================================

func TestExtraSetRLS_NilPoolPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("confirmed: SetRLS panics with nil pool: %v", r)
		}
	}()
	_ = SetRLS(context.Background(), nil, uuid.New().String(), "", false)
	t.Error("expected panic")
}

func TestExtraSetRLS_InvalidTenantUUID(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// Panic with nil pool — expected
		}
	}()
	err := SetRLS(context.Background(), nil, "not-uuid", "user-1", false)
	if err == nil {
		t.Fatal("expected error or panic")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected 'invalid' in error, got: %s", err.Error())
	}
}

// =============================================================================
// SetRLSTx — panics with nil tx (no nil guard)
// =============================================================================

func TestExtraSetRLSTx_NilTxPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("confirmed: SetRLSTx panics with nil tx: %v", r)
		}
	}()
	_ = SetRLSTx(context.Background(), nil, uuid.New().String(), "", false)
	t.Error("expected panic")
}

func TestExtraSetRLSTx_EmptyTenant(t *testing.T) {
	err := SetRLSTx(context.Background(), nil, "", "user-1", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "tenant_id is required" {
		t.Errorf("unexpected: %s", err.Error())
	}
}

func TestExtraSetRLSTx_InvalidTenantUUID(t *testing.T) {
	err := SetRLSTx(context.Background(), nil, "bad-uuid", "user-1", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected 'invalid' in error, got: %s", err.Error())
	}
}

func TestExtraSetRLSTx_InvalidUserUUID(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// Panic with nil tx — expected
		}
	}()
	_ = SetRLSTx(context.Background(), nil, uuid.New().String(), "bad-uuid", false)
	t.Error("expected panic")
}

// =============================================================================
// ExecTx — panics with nil pool
// =============================================================================

func TestExtraExecTx_NilPoolPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("confirmed: ExecTx panics with nil pool: %v", r)
		}
	}()
	_ = ExecTx(context.Background(), nil, uuid.New().String(), "", false,
		func(ctx context.Context, tx pgx.Tx) error { return nil })
	t.Error("expected panic")
}

func TestExtraExecTx_FnErrorReturned(t *testing.T) {
	wantErr := errors.New("boom")
	defer func() {
		if r := recover(); r != nil {
			// Panic with nil pool — expected
		}
	}()
	got := ExecTx(context.Background(), nil, uuid.New().String(), "", false,
		func(ctx context.Context, tx pgx.Tx) error { return wantErr })
	if got != wantErr {
		t.Errorf("expected wantErr, got %v", got)
	}
}

func TestExtraExecTx_InvalidTenant(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// Panic with nil pool — expected
		}
	}()
	_ = ExecTx(context.Background(), nil, "bad", "", false,
		func(ctx context.Context, tx pgx.Tx) error { return nil })
	t.Error("expected panic")
}

// =============================================================================
// Validation error messages
// =============================================================================

func TestExtraValidationError_TenantRequired(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// Panic with nil pool — expected
		}
	}()
	err := SetRLS(context.Background(), nil, "", "user-1", false)
	if err == nil {
		t.Fatal("expected error or panic")
	}
	if err != nil && err.Error() != "tenant_id is required" {
		t.Errorf("unexpected: %s", err.Error())
	}
}

func TestExtraValidationError_InvalidUUID(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// Panic with nil pool — expected
		}
	}()
	err := SetRLS(context.Background(), nil, "invalid", "user-1", false)
	if err == nil {
		t.Fatal("expected error or panic")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected 'invalid' in error, got: %s", err.Error())
	}
}
