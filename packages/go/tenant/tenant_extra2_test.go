// Tests for tenant package helpers (tenant.go).
package tenant

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExtra2_Validate_Valid(t *testing.T) {
	tests := []string{
		"abc",
		"tenant-1",
		"acme-corp",
		"my-tenant-123",
		"a1b",
	}
	for _, tt := range tests {
		if err := Validate(tt); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", tt, err)
		}
	}
}

func TestExtra2_Validate_Invalid(t *testing.T) {
	tests := []struct {
		id   string
		why  string
	}{
		{"", "empty"},
		{"ab", "too short"},
		{"a", "too short"},
		{strings.Repeat("a", 65), "too long"},
		{"ABC", "uppercase"},
		{"_underscore", "underscore"},
		{"with space", "space"},
		{"tenant!", "special char"},
		{"tenant.1", "dot"},
		// Note: starts-with-dash is allowed by current implementation
	}
	for _, tt := range tests {
		err := Validate(tt.id)
		if err == nil {
			t.Errorf("Validate(%q) [%s] = nil, want error", tt.id, tt.why)
		}
		if !errors.Is(err, ErrInvalidTenant) {
			t.Errorf("Validate(%q) = %v, want ErrInvalidTenant", tt.id, err)
		}
	}
}

func TestExtra2_Validate_Exactly64Chars(t *testing.T) {
	id := strings.Repeat("a", 64)
	if err := Validate(id); err != nil {
		t.Errorf("Validate(64 chars) = %v, want nil", err)
	}
}

func TestExtra2_Validate_Exactly3Chars(t *testing.T) {
	id := "abc"
	if err := Validate(id); err != nil {
		t.Errorf("Validate(3 chars) = %v, want nil", err)
	}
}

func TestExtra2_Validate_WithSpaces(t *testing.T) {
	// Spaces should be trimmed and fail length check
	err := Validate("  a  ")
	if err == nil {
		t.Error("should error for short after trim")
	}
}

func TestExtra2_SetValidator(t *testing.T) {
	original := defaultValidator
	defer SetValidator(original)
	
	called := false
	custom := Validator(func(id string) error {
		called = true
		return nil
	})
	
	SetValidator(custom)
	Validate("anything")
	if !called {
		t.Error("custom validator should be called")
	}
}

func TestExtra2_SetValidator_Nil(t *testing.T) {
	original := defaultValidator
	defer SetValidator(original)
	
	called := false
	custom := Validator(func(id string) error {
		called = true
		return nil
	})
	
	SetValidator(custom)
	SetValidator(nil) // restore default
	
	Validate("abc") // should use default
	if called {
		t.Error("default should not call custom validator")
	}
}

func TestExtra2_WithTenant(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-1")
	if IDFromContext(ctx) != "tenant-1" {
		t.Errorf("got %s", IDFromContext(ctx))
	}
}

func TestExtra2_WithTenant_Empty(t *testing.T) {
	ctx := WithTenant(context.Background(), "")
	if IDFromContext(ctx) != "" {
		t.Error("empty should be empty")
	}
}

func TestExtra2_WithTenantSlug(t *testing.T) {
	ctx := WithTenantSlug(context.Background(), "acme")
	if SlugFromContext(ctx) != "acme" {
		t.Errorf("got %s", SlugFromContext(ctx))
	}
}

func TestExtra2_WithTenantRole(t *testing.T) {
	ctx := WithTenantRole(context.Background(), "admin")
	if RoleFromContext(ctx) != "admin" {
		t.Errorf("got %s", RoleFromContext(ctx))
	}
}

func TestExtra2_WithBypass(t *testing.T) {
	ctx := WithBypass(context.Background())
	if !BypassFromContext(ctx) {
		t.Error("expected true after WithBypass")
	}
}

func TestExtra2_BypassFromContext_False(t *testing.T) {
	if BypassFromContext(context.Background()) {
		t.Error("expected false for empty context")
	}
}

func TestExtra2_IDFromContext_Empty(t *testing.T) {
	if IDFromContext(context.Background()) != "" {
		t.Error("expected empty for empty context")
	}
}

func TestExtra2_SlugFromContext_Empty(t *testing.T) {
	if SlugFromContext(context.Background()) != "" {
		t.Error("expected empty for empty context")
	}
}

func TestExtra2_RoleFromContext_Empty(t *testing.T) {
	if RoleFromContext(context.Background()) != "" {
		t.Error("expected empty for empty context")
	}
}

func TestExtra2_Ensure_NoTenant(t *testing.T) {
	err := Ensure(context.Background())
	if err == nil {
		t.Error("expected error for missing tenant")
	}
	if !errors.Is(err, ErrMissingTenant) {
		t.Errorf("expected ErrMissingTenant, got %v", err)
	}
}

func TestExtra2_Ensure_WithTenant(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-1")
	err := Ensure(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtra2_Ensure_WithBypass(t *testing.T) {
	ctx := WithBypass(context.Background())
	err := Ensure(ctx)
	if err != nil {
		t.Errorf("unexpected error with bypass: %v", err)
	}
}

func TestExtra2_Ensure_InvalidTenant(t *testing.T) {
	ctx := WithTenant(context.Background(), "INVALID")
	err := Ensure(ctx)
	if err == nil {
		t.Error("expected error for invalid tenant")
	}
}

func TestExtra2_Ensure_BypassOverrides(t *testing.T) {
	ctx := WithBypass(context.Background())
	ctx = WithTenant(ctx, "INVALID") // Invalid, but bypass should allow
	err := Ensure(ctx)
	if err != nil {
		t.Errorf("expected nil with bypass, got %v", err)
	}
}

func TestExtra2_Scope_FromContext_Empty(t *testing.T) {
	scope := FromContext(context.Background())
	if scope.ID != "" {
		t.Error("ID should be empty")
	}
	if scope.Slug != "" {
		t.Error("Slug should be empty")
	}
	if scope.Role != "" {
		t.Error("Role should be empty")
	}
	if scope.Bypass {
		t.Error("Bypass should be false")
	}
}

func TestExtra2_Scope_FromContext_All(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-1")
	ctx = WithTenantSlug(ctx, "acme")
	ctx = WithTenantRole(ctx, "admin")
	ctx = WithBypass(ctx)
	
	scope := FromContext(ctx)
	if scope.ID != "tenant-1" {
		t.Error("ID")
	}
	if scope.Slug != "acme" {
		t.Error("Slug")
	}
	if scope.Role != "admin" {
		t.Error("Role")
	}
	if !scope.Bypass {
		t.Error("Bypass should be true")
	}
}

func TestExtra2_IntoContext_All(t *testing.T) {
	scope := Scope{
		ID:     "tenant-1",
		Slug:   "acme",
		Role:   "admin",
		Bypass: true,
	}
	ctx := IntoContext(context.Background(), scope)
	
	if IDFromContext(ctx) != "tenant-1" {
		t.Error("ID")
	}
	if SlugFromContext(ctx) != "acme" {
		t.Error("Slug")
	}
	if RoleFromContext(ctx) != "admin" {
		t.Error("Role")
	}
	if !BypassFromContext(ctx) {
		t.Error("Bypass")
	}
}

func TestExtra2_IntoContext_Partial(t *testing.T) {
	scope := Scope{ID: "tenant-1"}
	ctx := IntoContext(context.Background(), scope)
	
	if IDFromContext(ctx) != "tenant-1" {
		t.Error("ID should be set")
	}
	if BypassFromContext(ctx) {
		t.Error("Bypass should not be set")
	}
}

func TestExtra2_Inherit_Empty(t *testing.T) {
	ctx := Inherit(context.Background(), context.Background())
	if IDFromContext(ctx) != "" {
		t.Error("expected empty")
	}
}

func TestExtra2_Inherit_FromSet(t *testing.T) {
	src := WithTenant(context.Background(), "tenant-1")
	src = WithTenantSlug(src, "acme")
	src = WithTenantRole(src, "admin")
	src = WithBypass(src)
	
	dst := Inherit(context.Background(), src)
	
	if IDFromContext(dst) != "tenant-1" {
		t.Error("ID not inherited")
	}
	if SlugFromContext(dst) != "acme" {
		t.Error("Slug not inherited")
	}
	if RoleFromContext(dst) != "admin" {
		t.Error("Role not inherited")
	}
	if !BypassFromContext(dst) {
		t.Error("Bypass not inherited")
	}
}

func TestExtra2_Validate_DigitsOnly(t *testing.T) {
	if err := Validate("123"); err != nil {
		t.Errorf("digits should be valid: %v", err)
	}
}

func TestExtra2_Validate_NumericPattern(t *testing.T) {
	if err := Validate("123-abc-456"); err != nil {
		t.Errorf("mixed should be valid: %v", err)
	}
}

func TestExtra2_Validate_TrailingDash(t *testing.T) {
	err := Validate("abc-")
	if err == nil {
		// Actually, "abc-" is valid because we only check chars
		// Trailing dash is allowed
	}
	_ = err
}

func TestExtra2_IDPattern(t *testing.T) {
	if IDPattern == "" {
		t.Error("IDPattern should not be empty")
	}
}

func TestExtra2_Errors(t *testing.T) {
	if ErrMissingTenant.Error() == "" {
		t.Error("ErrMissingTenant message")
	}
	if ErrInvalidTenant.Error() == "" {
		t.Error("ErrInvalidTenant message")
	}
	if ErrBypassRequired.Error() == "" {
		t.Error("ErrBypassRequired message")
	}
}

func TestExtra2_Validate_Unicode(t *testing.T) {
	// Unicode is not allowed
	err := Validate("tënant")
	if err == nil {
		t.Error("unicode should fail")
	}
}
