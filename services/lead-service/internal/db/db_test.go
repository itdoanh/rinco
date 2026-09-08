// Tests for Lead service db package.
package db

import (
	"context"
	"testing"
)

func TestSetRLS_EmptyTenantID(t *testing.T) {
	err := SetRLS(context.Background(), nil, "", "user-1", false)
	if err == nil {
		t.Fatal("expected error for empty tenantID")
	}
}

func TestSetRLS_InvalidTenantUUID(t *testing.T) {
	err := SetRLS(context.Background(), nil, "not-a-uuid", "user-1", false)
	if err == nil {
		t.Fatal("expected error for invalid tenant UUID")
	}
}

func TestSetRLSTx_EmptyTenantID(t *testing.T) {
	err := SetRLSTx(context.Background(), nil, "", "user-1", false)
	if err == nil {
		t.Fatal("expected error for empty tenantID")
	}
}

func TestSetRLSTx_InvalidTenantUUID(t *testing.T) {
	err := SetRLSTx(context.Background(), nil, "not-a-uuid", "user-1", false)
	if err == nil {
		t.Fatal("expected error for invalid tenant UUID")
	}
}
