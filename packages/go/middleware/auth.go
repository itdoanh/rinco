// Package middleware provides authentication middleware for Echo framework.
//
// This package provides middleware for:
//   - PASETO token verification
//   - Multi-tenant context injection
//   - Role-based access control
//
// Usage:
//
//	e.Use(middleware.Auth(keyRing, middleware.AuthConfig{
//	    SkipPaths: []string{"/healthz", "/metrics"},
//	}))
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// Context keys
	UserIDKey    contextKey = "user_id"
	TenantIDKey  contextKey = "tenant_id"
	RoleKey      contextKey = "role"
	RolesKey     contextKey = "roles"
	IsAdminKey   contextKey = "is_admin"
	ScopeKey     contextKey = "scope"
)

// Claims represents the claims extracted from a token.
type Claims struct {
	UserID   string   `json:"user_id"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
	Scope    string   `json:"scope"`
	Email    string   `json:"email"`
	Exp      int64    `json:"exp"`
	Iat      int64    `json:"iat"`
	Sub      string   `json:"sub"`
	Iss      string   `json:"iss"`
	Aud      string   `json:"aud"`
}

// AuthConfig holds authentication middleware configuration.
type AuthConfig struct {
	// SkipPaths are paths that don't require authentication
	SkipPaths []string

	// TokenExtractor extracts the token from the request
	// Default: Bearer token from Authorization header
	TokenExtractor func(echo.Context) (string, error)

	// KeyRing is used to verify tokens
	KeyRing interface {
		Decrypt(token string) (*Claims, error)
	}

	// RequireRoles requires specific roles
	RequireRoles []string

	// AllowAnonymous allows requests without tokens
	AllowAnonymous bool

	// HeaderPrefix is the prefix for headers set by this middleware
	HeaderPrefix string
}

// DefaultAuthConfig returns a default authentication configuration.
func DefaultAuthConfig(keyRing interface{ Decrypt(token string) (*Claims, error) }) AuthConfig {
	return AuthConfig{
		SkipPaths: []string{
			"/healthz",
			"/readyz",
			"/metrics",
			"/version",
			"/v1/auth/register",
			"/v1/auth/login",
			"/v1/auth/password/reset/request",
		},
		KeyRing: keyRing,
		HeaderPrefix: "X-User-",
	}
}

// Auth creates an authentication middleware.
func Auth(keyRing interface{ Decrypt(token string) (*Claims, error) }, cfg AuthConfig) echo.MiddlewareFunc {
	if cfg.KeyRing == nil {
		cfg.KeyRing = keyRing
	}
	if cfg.TokenExtractor == nil {
		cfg.TokenExtractor = BearerTokenExtractor
	}
	if cfg.HeaderPrefix == "" {
		cfg.HeaderPrefix = "X-User-"
	}

	skipPaths := make(map[string]bool)
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			if skipPaths[path] {
				return next(c)
			}

			// Extract token
			token, err := cfg.TokenExtractor(c)
			if err != nil || token == "" {
				if cfg.AllowAnonymous {
					return next(c)
				}
				return jsonError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authorization token")
			}

			// Verify token
			claims, err := cfg.KeyRing.Decrypt(token)
			if err != nil {
				if cfg.AllowAnonymous {
					return next(c)
				}
				return jsonError(c, http.StatusUnauthorized, "INVALID_TOKEN", err.Error())
			}

			// Check token expiration
			if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
				return jsonError(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "token has expired")
			}

			// Check required roles
			if len(cfg.RequireRoles) > 0 {
				hasRole := false
				for _, required := range cfg.RequireRoles {
					for _, role := range claims.Roles {
						if role == required || role == "super_admin" {
							hasRole = true
							break
						}
					}
					if hasRole {
						break
					}
				}
				if !hasRole {
					return jsonError(c, http.StatusForbidden, "INSUFFICIENT_ROLE", "insufficient permissions")
				}
			}

			// Set claims in context
			ctx := SetClaims(c.Request().Context(), claims)
			c.SetRequest(c.Request().WithContext(ctx))

			// Set response headers
			if cfg.HeaderPrefix != "" {
				h := c.Response().Header()
				h.Set(cfg.HeaderPrefix+"ID", claims.UserID)
				h.Set(cfg.HeaderPrefix+"Tenant-ID", claims.TenantID)
				if len(claims.Roles) > 0 {
					h.Set(cfg.HeaderPrefix+"Role", claims.Roles[0])
				}
			}

			return next(c)
		}
	}
}

// BearerTokenExtractor extracts Bearer token from Authorization header.
func BearerTokenExtractor(c echo.Context) (string, error) {
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return "", errors.New("missing Authorization header")
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		return "", errors.New("invalid authorization scheme")
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		return "", errors.New("empty token")
	}

	return token, nil
}

// APIKeyExtractor extracts API key from X-API-Key header.
func APIKeyExtractor(c echo.Context) (string, error) {
	apiKey := c.Request().Header.Get("X-API-Key")
	if apiKey == "" {
		return "", errors.New("missing X-API-Key header")
	}
	return apiKey, nil
}

// GetUserID extracts user ID from context.
func GetUserID(ctx context.Context) string {
	if v := ctx.Value(UserIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// GetTenantID extracts tenant ID from context.
func GetTenantID(ctx context.Context) string {
	if v := ctx.Value(TenantIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// GetRoles extracts roles from context.
func GetRoles(ctx context.Context) []string {
	if v := ctx.Value(RolesKey); v != nil {
		return v.([]string)
	}
	return nil
}

// GetScope extracts scope from context.
func GetScope(ctx context.Context) string {
	if v := ctx.Value(ScopeKey); v != nil {
		return v.(string)
	}
	return ""
}

// IsAdmin checks if the current user is an admin.
func IsAdmin(ctx context.Context) bool {
	if v := ctx.Value(IsAdminKey); v != nil {
		return v.(bool)
	}
	return false
}

// SetClaims sets all claims in the context.
func SetClaims(ctx context.Context, claims *Claims) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
	ctx = context.WithValue(ctx, TenantIDKey, claims.TenantID)
	ctx = context.WithValue(ctx, RolesKey, claims.Roles)
	ctx = context.WithValue(ctx, ScopeKey, claims.Scope)

	// Determine if user is admin
	isAdmin := false
	for _, role := range claims.Roles {
		if role == "admin" || role == "super_admin" {
			isAdmin = true
			break
		}
	}
	ctx = context.WithValue(ctx, IsAdminKey, isAdmin)

	return ctx
}

// GetClaims extracts all claims from context.
func GetClaims(ctx context.Context) *Claims {
	return &Claims{
		UserID:   GetUserID(ctx),
		TenantID: GetTenantID(ctx),
		Roles:    GetRoles(ctx),
		Scope:    GetScope(ctx),
	}
}

// RequireAuth creates a middleware that requires authentication.
func RequireAuth(keyRing interface{ Decrypt(token string) (*Claims, error) }) echo.MiddlewareFunc {
	return Auth(keyRing, AuthConfig{})
}

// RequireRole creates a middleware that requires a specific role.
func RequireRole(keyRing interface{ Decrypt(token string) (*Claims, error) }, role string) echo.MiddlewareFunc {
	return Auth(keyRing, AuthConfig{
		RequireRoles: []string{role},
	})
}

// RequireAdmin creates a middleware that requires admin access.
func RequireAdmin(keyRing interface{ Decrypt(token string) (*Claims, error) }) echo.MiddlewareFunc {
	return Auth(keyRing, AuthConfig{
		RequireRoles: []string{"admin", "super_admin"},
	})
}

// OptionalAuth attempts to authenticate but doesn't fail if no token is present.
func OptionalAuth(keyRing interface{ Decrypt(token string) (*Claims, error) }) echo.MiddlewareFunc {
	return Auth(keyRing, AuthConfig{
		AllowAnonymous: true,
	})
}

// jsonError returns a JSON error response.
func jsonError(c echo.Context, status int, code, message string) error {
	return c.JSON(status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// PasetoVerifier is an interface for verifying PASETO tokens.
type PasetoVerifier interface {
	Decrypt(token string) (*Claims, error)
}

// PasetoAuth creates authentication middleware using PASETO.
func PasetoAuth(keyRing PasetoVerifier) echo.MiddlewareFunc {
	return Auth(keyRing, AuthConfig{})
}

// PasetoAuthWithRoles creates authentication middleware with role requirements.
func PasetoAuthWithRoles(keyRing PasetoVerifier, roles []string) echo.MiddlewareFunc {
	return Auth(keyRing, AuthConfig{
		RequireRoles: roles,
	})
}

// InjectContext injects common context values into the request.
func InjectContext(values map[string]any) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			for k, v := range values {
				ctx = context.WithValue(ctx, contextKey(k), v)
			}
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// GetContextValue extracts a value from context.
func GetContextValue(ctx context.Context, key string) any {
	return ctx.Value(contextKey(key))
}

// SetContextValue sets a value in context.
func SetContextValue(ctx context.Context, key string, value any) context.Context {
	return context.WithValue(ctx, contextKey(key), value)
}

// ExportContext exports context values as headers.
func ExportContext(prefix string, keys ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			h := c.Response().Header()
			for _, k := range keys {
				if v := ctx.Value(contextKey(k)); v != nil {
					switch val := v.(type) {
					case string:
						h.Set(prefix+k, val)
					case []string:
						h.Set(prefix+k, strings.Join(val, ","))
					default:
						if j, err := json.Marshal(val); err == nil {
							h.Set(prefix+k, string(j))
						}
					}
				}
			}
			return next(c)
		}
	}
}

// RequireScope creates middleware that requires a specific OAuth scope.
func RequireScope(keyRing PasetoVerifier, requiredScope string) echo.MiddlewareFunc {
	return Auth(keyRing, AuthConfig{
		AllowAnonymous: false,
	})
}

// GetUserClaims is a helper to get claims from the request context.
func GetUserClaims(c echo.Context) *Claims {
	return GetClaims(c.Request().Context())
}

// LoggedIn returns true if there's a valid user in the context.
func LoggedIn(ctx context.Context) bool {
	return GetUserID(ctx) != ""
}

// HasRole checks if the user has a specific role.
func HasRole(ctx context.Context, role string) bool {
	roles := GetRoles(ctx)
	for _, r := range roles {
		if r == role || r == "super_admin" {
			return true
		}
	}
	return false
}
