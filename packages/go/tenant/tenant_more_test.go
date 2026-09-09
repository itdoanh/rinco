package tenant

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestErrorsDefined(t *testing.T) {
	if ErrMissingTenant == nil {
		t.Error("ErrMissingTenant nil")
	}
	if ErrInvalidTenant == nil {
		t.Error("ErrInvalidTenant nil")
	}
	if ErrBypassRequired == nil {
		t.Error("ErrBypassRequired nil")
	}
}

func TestValidateValid(t *testing.T) {
	valid := []string{
		"abc", "tenant-1", "abc-123", "abc-123-xyz",
		"a1b2c3", "abc-def", strings.Repeat("a", 63),
	}
	for _, v := range valid {
		if err := Validate(v); err != nil {
			t.Errorf("Validate(%q) unexpected error: %v", v, err)
		}
	}
}

func TestValidateInvalid(t *testing.T) {
	invalid := []string{
		"", "a", "ab", "ABC", "abc.def", "abc def",
		strings.Repeat("a", 65), "abc_def", "abc/def",
	}
	for _, v := range invalid {
		if err := Validate(v); err == nil {
			t.Errorf("Validate(%q) should fail", v)
		}
	}
}

func TestSetValidatorCustom(t *testing.T) {
	custom := func(id string) error {
		if id != "ok" {
			return errors.New("not ok")
		}
		return nil
	}
	SetValidator(custom)
	defer SetValidator(nil)
	if err := Validate("ok"); err != nil {
		t.Errorf("expected ok, got %v", err)
	}
	if err := Validate("bad"); err == nil {
		t.Error("expected error for bad")
	}
}

func TestSetValidatorNilRestores(t *testing.T) {
	SetValidator(nil)
	if err := Validate("abc"); err != nil {
		t.Errorf("default validator broken: %v", err)
	}
}

func TestWithTenantRoundtrip(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-x")
	if id := IDFromContext(ctx); id != "tenant-x" {
		t.Errorf("ID = %q", id)
	}
}

func TestWithTenantSlugRoundtrip(t *testing.T) {
	ctx := WithTenantSlug(context.Background(), "acme-corp")
	if s := SlugFromContext(ctx); s != "acme-corp" {
		t.Errorf("slug = %q", s)
	}
}

func TestWithTenantRoleRoundtrip(t *testing.T) {
	ctx := WithTenantRole(context.Background(), "admin")
	if r := RoleFromContext(ctx); r != "admin" {
		t.Errorf("role = %q", r)
	}
}

func TestWithBypassRoundtrip(t *testing.T) {
	ctx := WithBypass(context.Background())
	if !BypassFromContext(ctx) {
		t.Error("bypass should be true")
	}
}

func TestIDFromContextEmpty(t *testing.T) {
	if id := IDFromContext(context.Background()); id != "" {
		t.Errorf("expected empty, got %q", id)
	}
}

func TestSlugFromContextEmpty(t *testing.T) {
	if s := SlugFromContext(context.Background()); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

func TestRoleFromContextEmpty(t *testing.T) {
	if r := RoleFromContext(context.Background()); r != "" {
		t.Errorf("expected empty, got %q", r)
	}
}

func TestBypassFromContextEmpty(t *testing.T) {
	if BypassFromContext(context.Background()) {
		t.Error("bypass should be false")
	}
}

func TestEnsureMissing(t *testing.T) {
	err := Ensure(context.Background())
	if !errors.Is(err, ErrMissingTenant) {
		t.Errorf("expected ErrMissingTenant, got %v", err)
	}
}

func TestEnsureInvalid(t *testing.T) {
	ctx := WithTenant(context.Background(), "INVALID")
	err := Ensure(ctx)
	if !errors.Is(err, ErrInvalidTenant) {
		t.Errorf("expected ErrInvalidTenant, got %v", err)
	}
}

func TestEnsureValid(t *testing.T) {
	ctx := WithTenant(context.Background(), "valid-tenant")
	if err := Ensure(ctx); err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestEnsureBypass(t *testing.T) {
	// Even without tenant, bypass allows
	ctx := WithBypass(context.Background())
	if err := Ensure(ctx); err != nil {
		t.Errorf("bypass should skip validation: %v", err)
	}
}

func TestFromContext(t *testing.T) {
	ctx := WithTenant(context.Background(), "t1")
	ctx = WithTenantSlug(ctx, "s1")
	ctx = WithTenantRole(ctx, "admin")
	ctx = WithBypass(ctx)
	s := FromContext(ctx)
	if s.ID != "t1" {
		t.Errorf("ID = %q", s.ID)
	}
	if s.Slug != "s1" {
		t.Errorf("Slug = %q", s.Slug)
	}
	if s.Role != "admin" {
		t.Errorf("Role = %q", s.Role)
	}
	if !s.Bypass {
		t.Error("Bypass should be true")
	}
}

func TestIntoContextEmpty(t *testing.T) {
	ctx := IntoContext(context.Background(), Scope{})
	if id := IDFromContext(ctx); id != "" {
		t.Errorf("expected empty, got %q", id)
	}
}

func TestIntoContextFull(t *testing.T) {
	ctx := IntoContext(context.Background(), Scope{
		ID:     "t1",
		Slug:   "s1",
		Role:   "admin",
		Bypass: true,
	})
	s := FromContext(ctx)
	if s.ID != "t1" || s.Slug != "s1" || s.Role != "admin" || !s.Bypass {
		t.Errorf("full Scope mismatch: %+v", s)
	}
}

func TestInheritCopiesAll(t *testing.T) {
	src := WithTenant(context.Background(), "t1")
	src = WithTenantSlug(src, "s1")
	src = WithTenantRole(src, "admin")
	src = WithBypass(src)

	dst := Inherit(context.Background(), src)
	s := FromContext(dst)
	if s.ID != "t1" {
		t.Errorf("ID = %q", s.ID)
	}
	if s.Slug != "s1" {
		t.Errorf("Slug = %q", s.Slug)
	}
	if s.Role != "admin" {
		t.Errorf("Role = %q", s.Role)
	}
	if !s.Bypass {
		t.Error("Bypass should be true")
	}
}

func TestInheritEmptySrc(t *testing.T) {
	dst := Inherit(context.Background(), context.Background())
	s := FromContext(dst)
	if s.ID != "" || s.Slug != "" || s.Role != "" || s.Bypass {
		t.Errorf("expected all empty, got %+v", s)
	}
}

func TestIDPattern(t *testing.T) {
	// Just check pattern is defined and non-empty
	if IDPattern == "" {
		t.Error("IDPattern empty")
	}
}

func TestValidateTrimsSpaces(t *testing.T) {
	// Leading/trailing spaces get trimmed and result is valid
	if err := Validate("  abc  "); err != nil {
		t.Errorf("leading/trailing spaces should be trimmed: %v", err)
	}
	// Middle space is invalid
	if err := Validate("a b"); err == nil {
		t.Error("middle space should fail")
	}
}

func TestWithTenantDoesNotMutateBase(t *testing.T) {
	base := context.Background()
	with1 := WithTenant(base, "t1")
	with2 := WithTenant(base, "t2")
	if FromContext(with1).ID != "t1" {
		t.Errorf("with1.ID = %q", FromContext(with1).ID)
	}
	if FromContext(with2).ID != "t2" {
		t.Errorf("with2.ID = %q", FromContext(with2).ID)
	}
	if id := IDFromContext(base); id != "" {
		t.Errorf("base context mutated: %q", id)
	}
}
