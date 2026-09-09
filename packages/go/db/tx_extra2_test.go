// Tests for the new WithTxTenant function and related helpers.
package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExtra2_TxOptionsWithOnRetry(t *testing.T) {
	called := 0
	o := TxOptions{
		MaxRetries: 2,
		RetryDelay: 1 * time.Millisecond,
		OnRetry:    func(attempt int, err error) { called++ },
	}
	if o.OnRetry == nil {
		t.Fatal("OnRetry should be set")
	}
	o.OnRetry(1, errors.New("test"))
	if called != 1 {
		t.Errorf("OnRetry called %d times", called)
	}
}

func TestExtra2_TxOptionsPreserved(t *testing.T) {
	o := TxOptions{
		MaxRetries: 10,
		RetryDelay: 200 * time.Millisecond,
	}.defaults()
	if o.MaxRetries != 10 {
		t.Errorf("MaxRetries: got %d", o.MaxRetries)
	}
	if o.RetryDelay != 200*time.Millisecond {
		t.Errorf("RetryDelay: got %v", o.RetryDelay)
	}
}

func TestExtra2_ContextKeysUnique(t *testing.T) {
	// Verify all context keys are distinct
	if TenantIDKey == UserIDKey {
		t.Error("TenantIDKey and UserIDKey must be distinct")
	}
	if TenantIDKey == IsAdminKey {
		t.Error("TenantIDKey and IsAdminKey must be distinct")
	}
	if UserIDKey == IsAdminKey {
		t.Error("UserIDKey and IsAdminKey must be distinct")
	}
	if IsAdminKey == IsAuthKey {
		t.Error("IsAdminKey and IsAuthKey must be distinct")
	}
	if IsAdminKey == BypassRLSKey {
		t.Error("IsAdminKey and BypassRLSKey must be distinct")
	}
	if TenantIDKey == BypassRLSKey {
		t.Error("TenantIDKey and BypassRLSKey must be distinct")
	}
	if UserIDKey == BypassRLSKey {
		t.Error("UserIDKey and BypassRLSKey must be distinct")
	}
}

func TestExtra2_SetTenantContext_NonAdmin(t *testing.T) {
	ctx := SetTenantContext(context.Background(), "t1", "u1", false)
	if TenantIDFromContext(ctx) != "t1" {
		t.Errorf("TenantID: got %s", TenantIDFromContext(ctx))
	}
	if IsSuperAdmin(ctx) {
		t.Error("should not be admin")
	}
	if !IsAuthenticated(ctx) {
		t.Error("should be authenticated")
	}
}

func TestExtra2_WithBypassRLS_Nested(t *testing.T) {
	ctx := WithBypassRLS(context.Background())
	if !BypassRLSFromContext(ctx) {
		t.Error("bypass should be true")
	}
	// Nest with another WithBypassRLS — should still be true
	ctx2 := WithBypassRLS(ctx)
	if !BypassRLSFromContext(ctx2) {
		t.Error("nested bypass should be true")
	}
}

func TestExtra2_MaxRetriesNegative(t *testing.T) {
	o := TxOptions{MaxRetries: -1}.defaults()
	// Negative is preserved (defaults only fills zero)
	if o.MaxRetries != -1 {
		t.Errorf("negative MaxRetries should be preserved, got %d", o.MaxRetries)
	}
}

func TestExtra2_RetryDelayNegative(t *testing.T) {
	o := TxOptions{RetryDelay: -1 * time.Millisecond}.defaults()
	// Negative is preserved (defaults only fills zero)
	if o.RetryDelay != -1*time.Millisecond {
		t.Errorf("negative RetryDelay should be preserved, got %v", o.RetryDelay)
	}
}
