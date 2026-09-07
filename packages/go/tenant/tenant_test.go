package tenant

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDefaults(t *testing.T) {
	good := []string{"abc", "tenant-1", "a1b2c3", strings.Repeat("a", 64)}
	for _, id := range good {
		assert.NoError(t, Validate(id), "expected %q valid", id)
	}

	bad := []string{"", "ab", "UPPER", "with space", "with_underscore", strings.Repeat("a", 65)}
	for _, id := range bad {
		assert.Error(t, Validate(id), "expected %q invalid", id)
	}
}

func TestSetValidator(t *testing.T) {
	original := defaultValidator
	defer SetValidator(original)

	called := 0
	SetValidator(func(id string) error {
		called++
		return nil
	})
	assert.NoError(t, Validate("anything"))
	assert.Equal(t, 1, called)

	SetValidator(nil)
	assert.NotNil(t, defaultValidator)
	assert.NoError(t, Validate("valid-id"))
}

func TestWithAndFrom(t *testing.T) {
	ctx := WithTenant(context.Background(), "tenant-1")
	assert.Equal(t, "tenant-1", IDFromContext(ctx))

	ctx = WithTenantSlug(ctx, "acme")
	assert.Equal(t, "acme", SlugFromContext(ctx))

	ctx = WithTenantRole(ctx, "admin")
	assert.Equal(t, "admin", RoleFromContext(ctx))

	assert.False(t, BypassFromContext(ctx))
}

func TestEnsure(t *testing.T) {
	ctx := WithTenant(context.Background(), "abc")
	require.NoError(t, Ensure(ctx))

	// Missing
	assert.True(t, errors.Is(Ensure(context.Background()), ErrMissingTenant))

	// Bypass skips missing
	bypassCtx := WithBypass(context.Background())
	assert.NoError(t, Ensure(bypassCtx))

	// Invalid id rejected
	assert.Error(t, Ensure(WithTenant(context.Background(), "BAD")))
}

func TestInherit(t *testing.T) {
	src := WithTenantRole(WithTenantSlug(WithTenant(context.Background(), "id-1"), "slug-1"), "admin")
	src = WithBypass(src)

	dst := Inherit(context.Background(), src)
	assert.Equal(t, "id-1", IDFromContext(dst))
	assert.Equal(t, "slug-1", SlugFromContext(dst))
	assert.Equal(t, "admin", RoleFromContext(dst))
	assert.True(t, BypassFromContext(dst))

	// No values in src → dst unchanged
	dst2 := Inherit(context.Background(), context.Background())
	assert.Equal(t, "", IDFromContext(dst2))
}

func TestScopeRoundtrip(t *testing.T) {
	src := IntoContext(context.Background(), Scope{
		ID: "id-1", Slug: "slug-1", Role: "owner", Bypass: true,
	})
	got := FromContext(src)
	assert.Equal(t, "id-1", got.ID)
	assert.Equal(t, "slug-1", got.Slug)
	assert.Equal(t, "owner", got.Role)
	assert.True(t, got.Bypass)

	// Empty scope
	got = FromContext(context.Background())
	assert.Equal(t, Scope{}, got)
}