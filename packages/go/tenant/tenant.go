// Package tenant provides multi-tenant context helpers used across services
// to read, write, and validate tenant identifiers without depending on a
// specific HTTP framework.
//
// This package complements the db package's TenantID context key by exposing
// a tenant-aware key that other middleware (auth, audit, logging) can read.
// It is intentionally framework-agnostic: the actual request handler lives
// in the middleware package.
package tenant

import (
	"context"
	"errors"
	"strings"
)

type ctxKey string

const (
	tenantIDKey   ctxKey = "tenant_id"
	tenantSlugKey ctxKey = "tenant_slug"
	tenantRoleKey ctxKey = "tenant_role"
	bypassKey     ctxKey = "bypass_tenant"
)

// Errors returned by Ensure/Validate.
var (
	ErrMissingTenant  = errors.New("tenant: missing tenant id in context")
	ErrInvalidTenant  = errors.New("tenant: invalid tenant id")
	ErrBypassRequired = errors.New("tenant: bypass required for cross-tenant operation")
)

// IDPattern is a conservative tenant ID pattern: lowercase letters, digits,
// dashes, between 3 and 64 chars. Override via SetValidator if needed.
const IDPattern = "^[a-z0-9][a-z0-9-]{2,63}$"

// Validator returns nil if the tenant ID is valid.
type Validator func(tenantID string) error

var defaultValidator Validator = defaultValidate

func defaultValidate(id string) error {
	id = strings.TrimSpace(id)
	if len(id) < 3 || len(id) > 64 {
		return ErrInvalidTenant
	}
	for _, c := range id {
		ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-'
		if !ok {
			return ErrInvalidTenant
		}
	}
	return nil
}

// SetValidator replaces the package-level validator. Pass nil to restore
// the default.
func SetValidator(v Validator) {
	if v == nil {
		defaultValidator = defaultValidate
		return
	}
	defaultValidator = v
}

// Validate returns nil if id is a valid tenant identifier.
func Validate(id string) error { return defaultValidator(id) }

// ===== Context helpers =====

// WithTenant returns a new context with tenantID attached.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// WithTenantSlug returns a new context with the tenant slug attached (used by
// systems that look up tenants by slug).
func WithTenantSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, tenantSlugKey, slug)
}

// WithTenantRole returns a new context with the role the caller plays inside
// the tenant (e.g. "owner", "admin", "member").
func WithTenantRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, tenantRoleKey, role)
}

// WithBypass marks the context as allowed to bypass tenant scoping (used by
// super-admin jobs and migrations).
func WithBypass(ctx context.Context) context.Context {
	return context.WithValue(ctx, bypassKey, true)
}

// IDFromContext extracts the tenant ID, or "" if not set.
func IDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}

// SlugFromContext extracts the tenant slug, or "".
func SlugFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantSlugKey).(string)
	return v
}

// RoleFromContext extracts the tenant role, or "".
func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantRoleKey).(string)
	return v
}

// BypassFromContext reports whether bypass mode is set.
func BypassFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(bypassKey).(bool)
	return v
}

// Ensure returns an error when no tenant ID is attached to the context and
// bypass is not set. Useful in repository layers that must enforce scoping.
func Ensure(ctx context.Context) error {
	if BypassFromContext(ctx) {
		return nil
	}
	id := IDFromContext(ctx)
	if id == "" {
		return ErrMissingTenant
	}
	return Validate(id)
}

// Scope is a tenant-scoped context value object that callers can pass around
// without dealing with raw context.Value lookups.
type Scope struct {
	ID    string
	Slug  string
	Role  string
	Bypass bool
}

// FromContext returns the current scope.
func FromContext(ctx context.Context) Scope {
	return Scope{
		ID:     IDFromContext(ctx),
		Slug:   SlugFromContext(ctx),
		Role:   RoleFromContext(ctx),
		Bypass: BypassFromContext(ctx),
	}
}

// IntoContext stores a Scope back into a context. Useful for tests.
func IntoContext(ctx context.Context, s Scope) context.Context {
	if s.ID != "" {
		ctx = WithTenant(ctx, s.ID)
	}
	if s.Slug != "" {
		ctx = WithTenantSlug(ctx, s.Slug)
	}
	if s.Role != "" {
		ctx = WithTenantRole(ctx, s.Role)
	}
	if s.Bypass {
		ctx = WithBypass(ctx)
	}
	return ctx
}

// Inherit copies the tenant-related values from src to dst, returning dst.
func Inherit(dst, src context.Context) context.Context {
	if id := IDFromContext(src); id != "" {
		dst = WithTenant(dst, id)
	}
	if slug := SlugFromContext(src); slug != "" {
		dst = WithTenantSlug(dst, slug)
	}
	if role := RoleFromContext(src); role != "" {
		dst = WithTenantRole(dst, role)
	}
	if BypassFromContext(src) {
		dst = WithBypass(dst)
	}
	return dst
}