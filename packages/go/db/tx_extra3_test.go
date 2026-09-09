// Tests for db tx helpers (tx.go).
package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestExtra3_TxOptions_defaults(t *testing.T) {
	o := TxOptions{}
	got := o.defaults()
	if got.MaxRetries != 3 {
		t.Errorf("MaxRetries: got %d", got.MaxRetries)
	}
	if got.RetryDelay != 50*time.Millisecond {
		t.Errorf("RetryDelay: got %v", got.RetryDelay)
	}
	if got.IsoLevel != pgx.ReadCommitted {
		t.Errorf("IsoLevel: got %v", got.IsoLevel)
	}
}

func TestExtra3_TxOptions_defaults_PreservesSet(t *testing.T) {
	o := TxOptions{
		MaxRetries: 10,
		RetryDelay: 100 * time.Millisecond,
		IsoLevel:   pgx.Serializable,
	}
	got := o.defaults()
	if got.MaxRetries != 10 {
		t.Errorf("MaxRetries: got %d", got.MaxRetries)
	}
	if got.RetryDelay != 100*time.Millisecond {
		t.Errorf("RetryDelay: got %v", got.RetryDelay)
	}
	if got.IsoLevel != pgx.Serializable {
		t.Errorf("IsoLevel: got %v", got.IsoLevel)
	}
}

func TestExtra3_SerializableTxOptions(t *testing.T) {
	o := SerializableTxOptions()
	if o.IsoLevel != pgx.Serializable {
		t.Errorf("IsoLevel: got %v", o.IsoLevel)
	}
	if o.MaxRetries != 5 {
		t.Errorf("MaxRetries: got %d", o.MaxRetries)
	}
	if o.RetryDelay != 100*time.Millisecond {
		t.Errorf("RetryDelay: got %v", o.RetryDelay)
	}
}

func TestExtra3_isRetryable_Nil(t *testing.T) {
	if isRetryable(nil) {
		t.Error("nil should not be retryable")
	}
}

func TestExtra3_isRetryable_RegularError(t *testing.T) {
	if isRetryable(errors.New("some error")) {
		t.Error("regular error should not be retryable")
	}
}

func TestExtra3_isRetryable_SerializationFailure(t *testing.T) {
	err := &pgconn.PgError{Code: "40001", Message: "serialization_failure"}
	if !isRetryable(err) {
		t.Error("serialization_failure should be retryable")
	}
}

func TestExtra3_isRetryable_Deadlock(t *testing.T) {
	err := &pgconn.PgError{Code: "40P01", Message: "deadlock_detected"}
	if !isRetryable(err) {
		t.Error("deadlock should be retryable")
	}
}

func TestExtra3_isRetryable_OtherPgError(t *testing.T) {
	err := &pgconn.PgError{Code: "23505", Message: "unique_violation"}
	if isRetryable(err) {
		t.Error("unique violation should not be retryable")
	}
}

func TestExtra3_isRetryable_TxClosed(t *testing.T) {
	if isRetryable(pgx.ErrTxClosed) {
		t.Error("ErrTxClosed should not be retryable")
	}
}

func TestExtra3_IsAuthenticated_False(t *testing.T) {
	if IsAuthenticated(context.Background()) {
		t.Error("expected false for empty context")
	}
}

func TestExtra3_IsAuthenticated_True(t *testing.T) {
	ctx := SetTenantContext(context.Background(), "t1", "u1", false)
	if !IsAuthenticated(ctx) {
		t.Error("expected true after SetTenantContext")
	}
}

func TestExtra3_SetTenantContext_NotAdmin(t *testing.T) {
	ctx := SetTenantContext(context.Background(), "t1", "u1", false)
	if IsSuperAdmin(ctx) {
		t.Error("should not be admin")
	}
	if !IsAuthenticated(ctx) {
		t.Error("should be authenticated")
	}
	if TenantIDFromContext(ctx) != "t1" {
		t.Error("TenantID mismatch")
	}
	if UserIDFromContext(ctx) != "u1" {
		t.Error("UserID mismatch")
	}
}

func TestExtra3_WithBypassRLS(t *testing.T) {
	ctx := WithBypassRLS(context.Background())
	if !BypassRLSFromContext(ctx) {
		t.Error("expected true after WithBypassRLS")
	}
}

func TestExtra3_WithBypassRLS_NonBypassContext(t *testing.T) {
	if BypassRLSFromContext(context.Background()) {
		t.Error("expected false for non-bypass context")
	}
}

func TestExtra3_TxOptions_OnRetry(t *testing.T) {
	o := TxOptions{
		OnRetry: func(attempt int, err error) {},
	}
	if o.OnRetry == nil {
		t.Error("OnRetry should be set")
	}
}

func TestExtra3_TxOptions_AccessMode(t *testing.T) {
	o := TxOptions{
		AccessMode: pgx.ReadOnly,
	}
	if o.AccessMode != pgx.ReadOnly {
		t.Errorf("AccessMode: got %v", o.AccessMode)
	}
}

func TestExtra3_TxOptions_IsoLevelEmpty(t *testing.T) {
	o := TxOptions{}
	got := o.defaults()
	if got.IsoLevel == "" {
		t.Error("IsoLevel default should not be empty")
	}
}

func TestExtra3_isRetryable_Wrapped(t *testing.T) {
	// Wrap an SQLSTATE error - errors.As should find it
	pgErr := &pgconn.PgError{Code: "40001", Message: "serialization_failure"}
	wrapped := &wrappedError{msg: "wrapped", inner: pgErr}
	if !isRetryable(wrapped) {
		t.Error("wrapped serialization_failure should be retryable via errors.As")
	}
}

func TestExtra3_TxOptions_FullDefaults(t *testing.T) {
	o := TxOptions{
		MaxRetries: 0, // will be defaulted
		RetryDelay: 0, // will be defaulted
	}
	got := o.defaults()
	if got.MaxRetries == 0 {
		t.Error("MaxRetries defaulting should set non-zero")
	}
	if got.RetryDelay == 0 {
		t.Error("RetryDelay defaulting should set non-zero")
	}
}
