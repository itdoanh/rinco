// Additional tests for tenant package.
package tenant

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExtra_Validate_TooShort(t *testing.T) {
	if err := Validate("ab"); err == nil {
		t.Error("too short should fail")
	}
}

func TestExtra_Validate_TooLong(t *testing.T) {
	long := strings.Repeat("a", 65)
	if err := Validate(long); err == nil {
		t.Error("too long should fail")
	}
}

func TestExtra_Validate_ValidSimple(t *testing.T) {
	if err := Validate("abc"); err != nil {
		t.Errorf("'abc' should be valid: %v", err)
	}
}

func TestExtra_Validate_ValidWithDash(t *testing.T) {
	if err := Validate("my-tenant-1"); err != nil {
		t.Errorf("'my-tenant-1' should be valid: %v", err)
	}
}

func TestExtra_Validate_InvalidUppercase(t *testing.T) {
	if err := Validate("MyTenant"); err == nil {
		t.Error("uppercase should fail")
	}
}

func TestExtra_Validate_InvalidSpecialChar(t *testing.T) {
	if err := Validate("tenant_1"); err == nil {
		t.Error("underscore should fail")
	}
}

func TestExtra_Validate_InvalidSpace(t *testing.T) {
	if err := Validate("my tenant"); err == nil {
		t.Error("space should fail")
	}
}

func TestExtra_Validate_Empty(t *testing.T) {
	if err := Validate(""); err == nil {
		t.Error("empty should fail")
	}
}

func TestExtra_SetValidator_Custom(t *testing.T) {
	called := 0
	SetValidator(func(id string) error {
		called++
		return nil
	})
	defer SetValidator(nil) // restore default

	if err := Validate("test"); err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if called != 1 {
		t.Errorf("custom validator not called: %d", called)
	}
}

func TestExtra_SetValidator_CustomError(t *testing.T) {
	custom := errors.New("custom error")
	SetValidator(func(id string) error {
		return custom
	})
	defer SetValidator(nil)

	if err := Validate("test"); err != custom {
		t.Errorf("expected custom error: %v", err)
	}
}

func TestExtra_WithTenant_AndBack(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-1")
	if got := IDFromContext(ctx); got != "tenant-1" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_IDFromContext_Nil(t *testing.T) {
	if got := IDFromContext(context.Background()); got != "" {
		t.Errorf("expected empty: got %s", got)
	}
}

func TestExtra_SlugFromContext(t *testing.T) {
	ctx := WithTenantSlug(context.Background(), "my-slug")
	if got := SlugFromContext(ctx); got != "my-slug" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_RoleFromContext(t *testing.T) {
	ctx := WithTenantRole(context.Background(), "admin")
	if got := RoleFromContext(ctx); got != "admin" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_BypassFromContext(t *testing.T) {
	ctx := WithBypass(context.Background())
	if !BypassFromContext(ctx) {
		t.Error("bypass should be true")
	}
}

func TestExtra_BypassFromContext_Default(t *testing.T) {
	if BypassFromContext(context.Background()) {
		t.Error("bypass should default to false")
	}
}

func TestExtra_Ensure_Missing(t *testing.T) {
	err := Ensure(context.Background())
	if err != ErrMissingTenant {
		t.Errorf("expected ErrMissingTenant: got %v", err)
	}
}

func TestExtra_Ensure_Bypass(t *testing.T) {
	ctx := WithBypass(context.Background())
	if err := Ensure(ctx); err != nil {
		t.Errorf("bypass should skip: got %v", err)
	}
}

func TestExtra_Ensure_Valid(t *testing.T) {
	ctx := WithTenant(context.Background(), "valid-tenant")
	if err := Ensure(ctx); err != nil {
		t.Errorf("expected nil: got %v", err)
	}
}

func TestExtra_Ensure_Invalid(t *testing.T) {
	ctx := WithTenant(context.Background(), "INVALID")
	if err := Ensure(ctx); err != ErrInvalidTenant {
		t.Errorf("expected ErrInvalidTenant: got %v", err)
	}
}

func TestExtra_Scope_FromContext(t *testing.T) {
	ctx := WithTenant(context.Background(), "t1")
	ctx = WithTenantSlug(ctx, "slug1")
	ctx = WithTenantRole(ctx, "admin")
	ctx = WithBypass(ctx)

	s := FromContext(ctx)
	if s.ID != "t1" {
		t.Errorf("ID: got %s", s.ID)
	}
	if s.Slug != "slug1" {
		t.Errorf("Slug: got %s", s.Slug)
	}
	if s.Role != "admin" {
		t.Errorf("Role: got %s", s.Role)
	}
	if !s.Bypass {
		t.Error("Bypass should be true")
	}
}

func TestExtra_Scope_IntoContext(t *testing.T) {
	s := Scope{ID: "t1", Slug: "slug1", Role: "admin", Bypass: true}
	ctx := IntoContext(context.Background(), s)

	if IDFromContext(ctx) != "t1" {
		t.Error("ID not set")
	}
	if SlugFromContext(ctx) != "slug1" {
		t.Error("Slug not set")
	}
	if RoleFromContext(ctx) != "admin" {
		t.Error("Role not set")
	}
	if !BypassFromContext(ctx) {
		t.Error("Bypass not set")
	}
}

func TestExtra_Scope_EmptyIntoContext(t *testing.T) {
	// Empty scope should leave ctx unchanged
	ctx := IntoContext(context.Background(), Scope{})
	if IDFromContext(ctx) != "" {
		t.Error("ID should not be set for empty scope")
	}
}

func TestExtra_Inherit_CopiesValues(t *testing.T) {
	src := WithTenant(context.Background(), "t1")
	src = WithTenantSlug(src, "slug1")
	src = WithTenantRole(src, "admin")
	src = WithBypass(src)

	dst := context.Background()
	dst = Inherit(dst, src)

	if IDFromContext(dst) != "t1" {
		t.Errorf("ID not inherited: %s", IDFromContext(dst))
	}
	if SlugFromContext(dst) != "slug1" {
		t.Errorf("Slug not inherited: %s", SlugFromContext(dst))
	}
	if RoleFromContext(dst) != "admin" {
		t.Errorf("Role not inherited: %s", RoleFromContext(dst))
	}
	if !BypassFromContext(dst) {
		t.Error("Bypass not inherited")
	}
}

func TestExtra_Inherit_EmptySource(t *testing.T) {
	src := context.Background()
	dst := Inherit(context.Background(), src)
	if IDFromContext(dst) != "" {
		t.Error("ID should not be set")
	}
}

func TestExtra_Validate_NumericOnly(t *testing.T) {
	if err := Validate("123"); err != nil {
		t.Errorf("numeric-only should be valid: %v", err)
	}
}

func TestExtra_Validate_TrailingDash(t *testing.T) {
	// 'abc-' ends with dash — should be valid (only leading dash is restricted by regex)
	if err := Validate("abc-"); err != nil {
		t.Errorf("trailing dash should be valid: %v", err)
	}
}
