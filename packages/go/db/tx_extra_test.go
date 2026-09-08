// Additional tests for db package tx.go.
package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestExtra_TxOptions_Defaults(t *testing.T) {
	o := TxOptions{}.defaults()
	if o.MaxRetries != 3 {
		t.Errorf("MaxRetries: got %d, want 3", o.MaxRetries)
	}
	if o.RetryDelay != 50*time.Millisecond {
		t.Errorf("RetryDelay: got %v", o.RetryDelay)
	}
	if o.IsoLevel != pgx.ReadCommitted {
		t.Errorf("IsoLevel: got %v", o.IsoLevel)
	}
}

func TestExtra_TxOptions_Custom(t *testing.T) {
	o := TxOptions{
		MaxRetries: 10,
		RetryDelay: 100 * time.Millisecond,
	}.defaults()
	if o.MaxRetries != 10 {
		t.Errorf("MaxRetries: got %d", o.MaxRetries)
	}
	if o.RetryDelay != 100*time.Millisecond {
		t.Errorf("RetryDelay: got %v", o.RetryDelay)
	}
}

func TestExtra_SerializableTxOptions(t *testing.T) {
	o := SerializableTxOptions()
	if o.IsoLevel != pgx.Serializable {
		t.Errorf("IsoLevel: got %v, want Serializable", o.IsoLevel)
	}
	if o.MaxRetries != 5 {
		t.Errorf("MaxRetries: got %d", o.MaxRetries)
	}
}

func TestExtra_IsRetryable_NilError(t *testing.T) {
	if isRetryable(nil) {
		t.Error("nil error should not be retryable")
	}
}

func TestExtra_IsRetryable_SerializationFailure(t *testing.T) {
	err := &pgconn.PgError{Code: "40001"}
	if !isRetryable(err) {
		t.Error("40001 should be retryable")
	}
}

func TestExtra_IsRetryable_DeadlockDetected(t *testing.T) {
	err := &pgconn.PgError{Code: "40P01"}
	if !isRetryable(err) {
		t.Error("40P01 should be retryable")
	}
}

func TestExtra_IsRetryable_OtherError(t *testing.T) {
	err := &pgconn.PgError{Code: "23505"} // unique_violation
	if isRetryable(err) {
		t.Error("23505 should NOT be retryable")
	}
}

func TestExtra_IsRetryable_RandomError(t *testing.T) {
	if isRetryable(extraErr("random error")) {
		t.Error("random error should NOT be retryable")
	}
}

func TestExtra_IsRetryable_TxClosed(t *testing.T) {
	if isRetryable(pgx.ErrTxClosed) {
		t.Error("ErrTxClosed should NOT be retryable")
	}
}

func TestExtra_SetTenantContext_Admin(t *testing.T) {
	ctx := SetTenantContext(context.Background(), "tenant-1", "user-1", true)
	if !IsSuperAdmin(ctx) {
		t.Error("IsSuperAdmin should be true")
	}
}

func TestExtra_WithBypassRLS_EmptyContext(t *testing.T) {
	if BypassRLSFromContext(context.Background()) {
		t.Error("empty context should NOT have bypass RLS")
	}
}

func TestExtra_TenantIDFromContext_Empty(t *testing.T) {
	if TenantIDFromContext(context.Background()) != "" {
		t.Error("empty context should return empty tenant_id")
	}
}

func TestExtra_UserIDFromContext_Empty(t *testing.T) {
	if UserIDFromContext(context.Background()) != "" {
		t.Error("empty context should return empty user_id")
	}
}

func TestExtra_IsSuperAdmin_Empty(t *testing.T) {
	if IsSuperAdmin(context.Background()) {
		t.Error("empty context should NOT be admin")
	}
}

func TestExtra_IsAuthenticated_Empty(t *testing.T) {
	if IsAuthenticated(context.Background()) {
		t.Error("empty context should NOT be authenticated")
	}
}

// extraErr is a simple error type for testing
type extraErr string

func (e extraErr) Error() string { return string(e) }
