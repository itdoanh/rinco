// Package middleware provides tenant resolution middleware for multi-tenant applications.
//
// This middleware resolves tenant from multiple sources:
//   - X-Tenant-ID header
//   - Path parameters (/:tenant_slug/)
//   - Query parameters (?tenant=)
//   - JWT/Token claims
//   - Domain/Host header
//
// Usage:
//
//	e.Use(middleware.Tenant(middleware.TenantConfig{
//	    Sources: []middleware.TenantSource{
//	        middleware.TenantHeader("X-Tenant-ID"),
//	        middleware.TenantPath("tenant_slug"),
//	        middleware.TenantQuery("tenant"),
//	    },
//	    DefaultTenant: "default",
//	}))
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// TenantSource defines where to get tenant information from.
type TenantSource interface {
	Resolve(c echo.Context) (string, bool)
}

// TenantHeader resolves tenant from HTTP header.
func TenantHeader(name string) TenantSource {
	return &headerSource{name: name}
}

type headerSource struct{ name string }

func (s *headerSource) Resolve(c echo.Context) (string, bool) {
	v := c.Request().Header.Get(s.name)
	return v, v != ""
}

// TenantPath resolves tenant from path parameter.
func TenantPath(param string) TenantSource {
	return &pathSource{param: param}
}

type pathSource struct{ param string }

func (s *pathSource) Resolve(c echo.Context) (string, bool) {
	v := c.Param(s.param)
	return v, v != ""
}

// TenantQuery resolves tenant from query parameter.
func TenantQuery(param string) TenantSource {
	return &querySource{param: param}
}

type querySource struct{ param string }

func (s *querySource) Resolve(c echo.Context) (string, bool) {
	v := c.QueryParam(s.param)
	return v, v != ""
}

// TenantHost resolves tenant from Host header.
func TenantHost(mapping map[string]string) TenantSource {
	return &hostSource{mapping: mapping}
}

type hostSource struct{ mapping map[string]string }

func (s *hostSource) Resolve(c echo.Context) string {
	host := c.Request().Host
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	// Direct match
	if tenant, ok := s.mapping[host]; ok {
		return tenant
	}
	// Subdomain match
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		// Check if first part is a known subdomain
		subdomain := parts[0]
		if subdomain != "www" && subdomain != "api" {
			return subdomain
		}
	}
	return ""
}

// TenantToken resolves tenant from JWT/PASETO token claims.
func TenantToken() TenantSource {
	return &tokenSource{}
}

type tokenSource struct{}

func (s *tokenSource) Resolve(c echo.Context) (string, bool) {
	claims := GetClaims(c.Request().Context())
	if claims != nil && claims.TenantID != "" {
		return claims.TenantID, true
	}
	return "", false
}

// TenantConfig holds tenant resolution configuration.
type TenantConfig struct {
	// Sources to try in order
	Sources []TenantSource

	// Default tenant if none found
	DefaultTenant string

	// Whether tenant is required
	Required bool

	// Skip paths that don't need tenant
	SkipPaths []string

	// Cache tenant resolution for the request
	CachePerRequest bool

	// ValidateTenant is called to validate the resolved tenant ID
	ValidateTenant func(tenantID string) (bool, error)
}

// DefaultTenantConfig returns a sensible default configuration.
func TenantDefaultConfig() TenantConfig {
	return TenantConfig{
		Sources: []TenantSource{
			TenantHeader("X-Tenant-ID"),
			TenantHeader("X-TenantID"),
			TenantPath("tenant_slug"),
			TenantPath("tenant"),
			TenantQuery("tenant"),
			TenantQuery("tenant_id"),
			TenantToken(),
		},
		Required:        true,
		CachePerRequest: true,
	}
}

// SkipPaths converts a slice of paths to a map for fast lookup.
func makeSkipPaths(paths []string) map[string]bool {
	m := make(map[string]bool)
	for _, p := range paths {
		m[p] = true
	}
	return m
}

// Tenant creates a tenant resolution middleware.
func Tenant(cfg TenantConfig) echo.MiddlewareFunc {
	skipPaths := makeSkipPaths(cfg.SkipPaths)

	// Set defaults
	if cfg.Sources == nil {
		cfg.Sources = TenantDefaultConfig().Sources
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			if skipPaths[path] {
				return next(c)
			}

			// Try each source in order
			var tenantID string
			for _, source := range cfg.Sources {
				if tid, ok := source.Resolve(c); ok && tid != "" {
					tenantID = tid
					break
				}
			}

			// Apply default if no tenant found
			if tenantID == "" {
				tenantID = cfg.DefaultTenant
			}

			// Validate tenant if configured
			if tenantID != "" && cfg.ValidateTenant != nil {
				valid, err := cfg.ValidateTenant(tenantID)
				if err != nil {
					return jsonError(c, http.StatusInternalServerError, "TENANT_VALIDATION_ERROR", err.Error())
				}
				if !valid {
					return jsonError(c, http.StatusBadRequest, "INVALID_TENANT", "invalid tenant: "+tenantID)
				}
			}

			// Check if tenant is required
			if tenantID == "" && cfg.Required {
				return jsonError(c, http.StatusBadRequest, "TENANT_REQUIRED", "tenant not found in request")
			}

			// Inject into context
			ctx := c.Request().Context()
			ctx = WithTenantID(ctx, tenantID)
			c.SetRequest(c.Request().WithContext(ctx))

			// Set response header
			if tenantID != "" {
				c.Response().Header().Set("X-Tenant-ID", tenantID)
			}

			return next(c)
		}
	}
}

// contextKey type for tenant context
type tenantContextKey string

const tenantContextKeyID tenantContextKey = "tenant_id"

// WithTenantID adds tenant ID to context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantContextKeyID, tenantID)
}

// TenantIDFromContext extracts tenant ID from context.
func TenantIDFromContext(ctx context.Context) string {
	if v := ctx.Value(tenantContextKeyID); v != nil {
		return v.(string)
	}
	return ""
}

// GetTenant is a helper to get tenant from echo context.
func GetTenant(c echo.Context) string {
	return TenantIDFromContext(c.Request().Context())
}

// TenantResolver provides tenant resolution for various contexts.
type TenantResolver struct {
	cfg TenantConfig
}

// NewTenantResolver creates a new tenant resolver.
func NewTenantResolver(cfg TenantConfig) *TenantResolver {
	if cfg.Sources == nil {
		cfg.Sources = TenantDefaultConfig().Sources
	}
	return &TenantResolver{cfg: cfg}
}

// Resolve resolves tenant from an echo context.
func (r *TenantResolver) Resolve(c echo.Context) string {
	var tenantID string
	for _, source := range r.cfg.Sources {
		if tid, ok := source.Resolve(c); ok && tid != "" {
			tenantID = tid
			break
		}
	}
	if tenantID == "" {
		tenantID = r.cfg.DefaultTenant
	}
	return tenantID
}

// TenantContextMiddleware injects tenant information into context for downstream handlers.
func TenantContextMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := GetTenant(c)
			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, tenantContextKeyID, tenantID)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// MultiTenantConfig holds configuration for multi-tenant routing.
type MultiTenantConfig struct {
	// TenantLookupFunc retrieves tenant information
	TenantLookupFunc func(ctx context.Context, tenantID string) (*TenantInfo, error)

	// CreateContext creates a database connection for the tenant
	// This is called when switching tenant context
	CreateTenantContext func(ctx context.Context, tenantID string) (context.Context, func(), error)
}

// TenantInfo contains information about a tenant.
type TenantInfo struct {
	ID       string
	Slug     string
	Name     string
	Plan     string
	Status   string
	Settings map[string]any
}

// MultiTenant creates middleware for multi-tenant database routing.
func MultiTenant(cfg MultiTenantConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := GetTenant(c)
			if tenantID == "" {
				return next(c)
			}

			ctx := c.Request().Context()

			// Look up tenant info
			if cfg.TenantLookupFunc != nil {
				info, err := cfg.TenantLookupFunc(ctx, tenantID)
				if err != nil {
					return jsonError(c, http.StatusNotFound, "TENANT_NOT_FOUND", "tenant not found: "+tenantID)
				}
				if info.Status == "suspended" || info.Status == "frozen" {
					return jsonError(c, http.StatusForbidden, "TENANT_SUSPENDED", "tenant is "+info.Status)
				}
				ctx = WithTenantInfo(ctx, info)
			}

			// Create tenant-specific context
			if cfg.CreateTenantContext != nil {
				ctx, cleanup, err := cfg.CreateTenantContext(ctx, tenantID)
				if err != nil {
					return jsonError(c, http.StatusInternalServerError, "TENANT_CONTEXT_ERROR", err.Error())
				}
				defer cleanup()
				c.SetRequest(c.Request().WithContext(ctx))
			}

			return next(c)
		}
	}
}

// TenantInfoContextKey is the context key for tenant info.
const TenantInfoContextKey tenantContextKey = "tenant_info"

// WithTenantInfo adds tenant info to context.
func WithTenantInfo(ctx context.Context, info *TenantInfo) context.Context {
	return context.WithValue(ctx, TenantInfoContextKey, info)
}

// TenantInfoFromContext extracts tenant info from context.
func TenantInfoFromContext(ctx context.Context) *TenantInfo {
	if v := ctx.Value(TenantInfoContextKey); v != nil {
		return v.(*TenantInfo)
	}
	return nil
}

// TenantScope creates a context with tenant scope for database queries.
func TenantScope(ctx context.Context, tenantID string) context.Context {
	return WithTenantID(ctx, tenantID)
}
