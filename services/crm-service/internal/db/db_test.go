// Tests for CRM service db package.
package db

import (
	"context"
	"testing"
)

// SetRLS_EmptyTenantID: pre-flight empty check before pool access.
func TestSetRLS_EmptyTenantID(t *testing.T) {
	err := SetRLS(context.Background(), nil, "", "user-1", false)
	if err == nil {
		t.Fatal("expected error for empty tenantID")
	}
}

// SetRLS_InvalidTenantUUID: pre-flight UUID check before pool access.
func TestSetRLS_InvalidTenantUUID(t *testing.T) {
	err := SetRLS(context.Background(), nil, "not-a-uuid", "user-1", false)
	if err == nil {
		t.Fatal("expected error for invalid tenant UUID")
	}
}

// SetRLSTx_EmptyTenantID: pre-flight empty check before tx access.
func TestSetRLSTx_EmptyTenantID(t *testing.T) {
	err := SetRLSTx(context.Background(), nil, "", "user-1", false)
	if err == nil {
		t.Fatal("expected error for empty tenantID")
	}
}

// SetRLSTx_InvalidTenantUUID: pre-flight UUID check before tx access.
func TestSetRLSTx_InvalidTenantUUID(t *testing.T) {
	err := SetRLSTx(context.Background(), nil, "not-a-uuid", "user-1", false)
	if err == nil {
		t.Fatal("expected error for invalid tenant UUID")
	}
}
