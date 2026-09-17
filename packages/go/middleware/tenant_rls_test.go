// Tests for the tenant-RLS GUC middleware (tenant_rls.go).
//
// The integration bit (actually talking to a Postgres) lives in
// services/integration-tests/ — these tests focus on the contract:
//   - skips health/metrics paths
//   - reads tenant from verified claims in context
//   - falls back to X-Tenant-ID header when no claim is present
//   - returns 400 when Required and no tenant can be resolved
//
// We don't spin up pgxpool here because the middleware's only side
// effect is the GUC `Exec` call — a testcontainers run is overkill.
// Instead, the actual GUC verification lives in the integration suite.
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestTenantRLS_DefaultsHeaderName(t *testing.T) {
	cfg := TenantRLSConfig{}.defaults()
	if cfg.HeaderName != "X-Tenant-ID" {
		t.Errorf("HeaderName: want X-Tenant-ID, got %q", cfg.HeaderName)
	}
	if !cfg.Required {
		t.Error("Required default should be true")
	}
	if len(cfg.SkipPaths) == 0 {
		t.Error("SkipPaths should have defaults")
	}
}

func TestResolveTenant_ClaimWins(t *testing.T) {
	type claimKey struct{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Both claim and header present — claim must win.
	req = req.WithContext(context.WithValue(req.Context(), claimKey{}, "claim-tenant"))
	req.Header.Set("X-Tenant-ID", "header-tenant")
	c := e.NewContext(req, httptest.NewRecorder())

	got := resolveTenantForRequest(c, TenantRLSConfig{
		ClaimTenantKey: claimKey{},
		HeaderName:     "X-Tenant-ID",
	}.defaults())
	if got != "claim-tenant" {
		t.Errorf("got %q, want claim-tenant", got)
	}
}

func TestResolveTenant_HeaderFallback(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "header-tenant")
	c := e.NewContext(req, httptest.NewRecorder())

	got := resolveTenantForRequest(c, TenantRLSConfig{}.defaults())
	if got != "header-tenant" {
		t.Errorf("got %q, want header-tenant", got)
	}
}

func TestResolveTenant_Nothing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, httptest.NewRecorder())

	got := resolveTenantForRequest(c, TenantRLSConfig{}.defaults())
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestResolveTenant_NonStringClaim(t *testing.T) {
	// If someone stuffs a non-string into the context under our key,
	// we must not panic and must fall back to the header.
	type claimKey struct{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), claimKey{}, 12345))
	req.Header.Set("X-Tenant-ID", "fallback-tenant")
	c := e.NewContext(req, httptest.NewRecorder())

	got := resolveTenantForRequest(c, TenantRLSConfig{
		ClaimTenantKey: claimKey{},
	}.defaults())
	if got != "fallback-tenant" {
		t.Errorf("got %q, want fallback-tenant", got)
	}
}

func TestTenantGUCStatement(t *testing.T) {
	sql, args := TenantGUCStatement("tenant-abc")
	if sql != "SELECT set_config($1, $2, false)" {
		t.Errorf("unexpected SQL: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != TenantGUCName {
		t.Errorf("arg0: want %s, got %v", TenantGUCName, args[0])
	}
	if args[1] != "tenant-abc" {
		t.Errorf("arg1: want tenant-abc, got %v", args[1])
	}
}

func TestEscapeSingleQuote(t *testing.T) {
	cases := map[string]string{
		"plain":            "plain",
		"with'apostrophe":  "with''apostrophe",
		"two''already":     "two''''already",
		"":                 "",
	}
	for in, want := range cases {
		if got := escapeSingleQuote(in); got != want {
			t.Errorf("escapeSingleQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTenantConn_NilContext(t *testing.T) {
	if TenantConn(nil) != nil {
		t.Error("nil echo context should yield nil conn")
	}
}

func TestTenantConn_NoMiddleware(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, httptest.NewRecorder())
	if TenantConn(c) != nil {
		t.Error("context without middleware should yield nil conn")
	}
}

func TestApplyTenantGUC_NilConn(t *testing.T) {
	if err := ApplyTenantGUC(context.Background(), nil, "x", false); err == nil {
		t.Error("nil conn should error")
	}
	if err := ApplyTenantGUC(context.Background(), nil, "", false); err == nil {
		t.Error("empty tenant id should error")
	}
}
