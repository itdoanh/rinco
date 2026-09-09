// Tests for middleware auth helpers (auth.go).
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestExtra_BearerTokenExtractor_Valid(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	token, err := BearerTokenExtractor(c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if token != "secret123" {
		t.Errorf("got %q", token)
	}
}

func TestExtra_BearerTokenExtractor_MissingHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := BearerTokenExtractor(c)
	if err == nil {
		t.Error("expected error for missing header")
	}
}

func TestExtra_BearerTokenExtractor_InvalidScheme(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic secret123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := BearerTokenExtractor(c)
	if err == nil {
		t.Error("expected error for invalid scheme")
	}
}

func TestExtra_BearerTokenExtractor_EmptyToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := BearerTokenExtractor(c)
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestExtra_BearerTokenExtractor_BearerOnly(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := BearerTokenExtractor(c)
	if err == nil {
		t.Error("expected error for no space after Bearer")
	}
}

func TestExtra_APIKeyExtractor_Valid(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "my-api-key")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	apiKey, err := APIKeyExtractor(c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if apiKey != "my-api-key" {
		t.Errorf("got %q", apiKey)
	}
}

func TestExtra_APIKeyExtractor_Missing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := APIKeyExtractor(c)
	if err == nil {
		t.Error("expected error for missing header")
	}
}

func TestExtra_GetUserID(t *testing.T) {
	claims := &Claims{UserID: "user-1"}
	ctx := SetClaims(context.Background(), claims)
	if got := GetUserID(ctx); got != "user-1" {
		t.Errorf("got %q", got)
	}
}

func TestExtra_GetUserID_Empty(t *testing.T) {
	if got := GetUserID(context.Background()); got != "" {
		t.Errorf("expected empty: %q", got)
	}
}

func TestExtra_GetTenantID(t *testing.T) {
	claims := &Claims{TenantID: "tenant-1"}
	ctx := SetClaims(context.Background(), claims)
	if got := GetTenantID(ctx); got != "tenant-1" {
		t.Errorf("got %q", got)
	}
}

func TestExtra_GetTenantID_Empty(t *testing.T) {
	if got := GetTenantID(context.Background()); got != "" {
		t.Errorf("expected empty: %q", got)
	}
}

func TestExtra_GetRoles(t *testing.T) {
	claims := &Claims{Roles: []string{"admin", "editor"}}
	ctx := SetClaims(context.Background(), claims)
	got := GetRoles(ctx)
	if len(got) != 2 || got[0] != "admin" {
		t.Errorf("got %v", got)
	}
}

func TestExtra_GetRoles_Empty(t *testing.T) {
	got := GetRoles(context.Background())
	if got != nil {
		t.Errorf("expected nil: %v", got)
	}
}

func TestExtra_GetScope(t *testing.T) {
	claims := &Claims{Scope: "read write"}
	ctx := SetClaims(context.Background(), claims)
	if got := GetScope(ctx); got != "read write" {
		t.Errorf("got %q", got)
	}
}

func TestExtra_GetScope_Empty(t *testing.T) {
	if got := GetScope(context.Background()); got != "" {
		t.Errorf("expected empty: %q", got)
	}
}

func TestExtra_IsAdmin_True(t *testing.T) {
	claims := &Claims{Roles: []string{"admin"}}
	ctx := SetClaims(context.Background(), claims)
	if got := IsAdmin(ctx); !got {
		t.Error("expected true")
	}
}

func TestExtra_IsAdmin_SuperAdmin(t *testing.T) {
	claims := &Claims{Roles: []string{"super_admin"}}
	ctx := SetClaims(context.Background(), claims)
	if got := IsAdmin(ctx); !got {
		t.Error("expected true for super_admin")
	}
}

func TestExtra_IsAdmin_False(t *testing.T) {
	claims := &Claims{Roles: []string{"editor"}}
	ctx := SetClaims(context.Background(), claims)
	if got := IsAdmin(ctx); got {
		t.Error("expected false")
	}
}

func TestExtra_IsAdmin_Empty(t *testing.T) {
	if got := IsAdmin(context.Background()); got {
		t.Error("expected false for empty context")
	}
}

func TestExtra_SetClaims(t *testing.T) {
	claims := &Claims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		Roles:    []string{"admin"},
		Scope:    "read",
	}
	ctx := SetClaims(context.Background(), claims)
	if GetUserID(ctx) != "user-1" {
		t.Error("UserID not set")
	}
	if GetTenantID(ctx) != "tenant-1" {
		t.Error("TenantID not set")
	}
	if len(GetRoles(ctx)) != 1 {
		t.Error("Roles not set")
	}
	if GetScope(ctx) != "read" {
		t.Error("Scope not set")
	}
	if !IsAdmin(ctx) {
		t.Error("IsAdmin not set")
	}
}

func TestExtra_GetClaims(t *testing.T) {
	claims := &Claims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		Roles:    []string{"admin"},
		Scope:    "read",
	}
	ctx := SetClaims(context.Background(), claims)
	got := GetClaims(ctx)
	if got.UserID != "user-1" {
		t.Error("UserID mismatch")
	}
	if got.TenantID != "tenant-1" {
		t.Error("TenantID mismatch")
	}
}

func TestExtra_LoggedIn(t *testing.T) {
	claims := &Claims{UserID: "user-1"}
	ctx := SetClaims(context.Background(), claims)
	if !LoggedIn(ctx) {
		t.Error("expected true")
	}
}

func TestExtra_LoggedIn_False(t *testing.T) {
	if LoggedIn(context.Background()) {
		t.Error("expected false")
	}
}

func TestExtra_HasRole(t *testing.T) {
	claims := &Claims{Roles: []string{"admin", "editor"}}
	ctx := SetClaims(context.Background(), claims)
	if !HasRole(ctx, "admin") {
		t.Error("expected true for admin")
	}
	if !HasRole(ctx, "editor") {
		t.Error("expected true for editor")
	}
	if HasRole(ctx, "super_admin") {
		t.Error("expected false")
	}
}

func TestExtra_HasRole_SuperAdminBypasses(t *testing.T) {
	claims := &Claims{Roles: []string{"super_admin"}}
	ctx := SetClaims(context.Background(), claims)
	if !HasRole(ctx, "admin") {
		t.Error("super_admin should bypass role check")
	}
}

func TestExtra_HasRole_NoRoles(t *testing.T) {
	ctx := context.Background()
	if HasRole(ctx, "admin") {
		t.Error("expected false")
	}
}

func TestExtra_GetContextValue(t *testing.T) {
	ctx := SetContextValue(context.Background(), "custom_key", "custom_value")
	got := GetContextValue(ctx, "custom_key")
	if got != "custom_value" {
		t.Errorf("got %v", got)
	}
}

func TestExtra_GetContextValue_Missing(t *testing.T) {
	got := GetContextValue(context.Background(), "missing")
	if got != nil {
		t.Errorf("expected nil: %v", got)
	}
}

func TestExtra_SetContextValue(t *testing.T) {
	ctx := SetContextValue(context.Background(), "key1", "val1")
	ctx = SetContextValue(ctx, "key2", 42)
	if GetContextValue(ctx, "key1") != "val1" {
		t.Error("key1 not set")
	}
	if GetContextValue(ctx, "key2") != 42 {
		t.Error("key2 not set")
	}
}

func TestExtra_InjectContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	mw := InjectContext(map[string]any{"injected": "value"})
	h := mw(func(c echo.Context) error {
		called = true
		val := GetContextValue(c.Request().Context(), "injected")
		if val != "value" {
			t.Errorf("got %v", val)
		}
		return nil
	})
	if err := h(c); err != nil {
		t.Errorf("unexpected: %v", err)
	}
	if !called {
		t.Error("next not called")
	}
}

func TestExtra_Claims_JSON(t *testing.T) {
	claims := &Claims{
		UserID:   "u1",
		TenantID: "t1",
		Roles:    []string{"admin"},
		Scope:    "read",
		Email:    "u1@example.com",
		Exp:      9999999999,
		Iat:      1000000000,
		Sub:      "u1",
		Iss:      "auth-service",
		Aud:      "api",
	}
	if claims.UserID != "u1" {
		t.Error("UserID")
	}
	if claims.TenantID != "t1" {
		t.Error("TenantID")
	}
	if len(claims.Roles) != 1 {
		t.Error("Roles")
	}
	if claims.Scope != "read" {
		t.Error("Scope")
	}
}

func TestExtra_Claims_EmptyRoles(t *testing.T) {
	claims := &Claims{UserID: ""}  // No user ID
	ctx := SetClaims(context.Background(), claims)
	if IsAdmin(ctx) {
		t.Error("empty roles should not be admin")
	}
	// LoggedIn checks GetUserID != "", which is empty, so LoggedIn should be false
	if LoggedIn(ctx) {
		t.Error("user with empty UserID should not be logged in")
	}
}

func TestExtra_GetUserClaims(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	claims := &Claims{UserID: "u1", TenantID: "t1"}
	c.SetRequest(c.Request().WithContext(SetClaims(c.Request().Context(), claims)))

	got := GetUserClaims(c)
	if got.UserID != "u1" {
		t.Error("UserID mismatch")
	}
}
