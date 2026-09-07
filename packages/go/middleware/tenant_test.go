package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestTenantHeaderResolution(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-abc")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantHeader("X-Tenant-ID")
	got, ok := src.Resolve(c)
	if !ok || got != "tenant-abc" {
		t.Errorf("got %q ok=%v, want tenant-abc/true", got, ok)
	}
}

func TestTenantHeaderMissing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantHeader("X-Tenant-ID")
	got, ok := src.Resolve(c)
	if ok || got != "" {
		t.Errorf("got %q ok=%v, want empty/false", got, ok)
	}
}

func TestTenantPathResolution(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/apexfintech/landing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("tenant_slug")
	c.SetParamValues("apexfintech")

	src := TenantPath("tenant_slug")
	got, ok := src.Resolve(c)
	if !ok || got != "apexfintech" {
		t.Errorf("got %q ok=%v", got, ok)
	}
}

func TestTenantQueryResolution(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?tenant=acme", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantQuery("tenant")
	got, ok := src.Resolve(c)
	if !ok || got != "acme" {
		t.Errorf("got %q ok=%v", got, ok)
	}
}

func TestTenantHostResolution(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "foo.example.com"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantHost(map[string]string{"foo.example.com": "foo"})
	got, ok := src.Resolve(c)
	if !ok || got != "foo" {
		t.Errorf("got %q ok=%v", got, ok)
	}
}

func TestTenantHostSubdomainFallback(t *testing.T) {
	// When no mapping matches, the first subdomain segment is returned
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "apex.example.com"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantHost(map[string]string{}) // empty
	got, ok := src.Resolve(c)
	if !ok || got != "apex" {
		t.Errorf("got %q ok=%v, want apex/true", got, ok)
	}
}

func TestTenantMiddleware(t *testing.T) {
	e := echo.New()
	var seenTenant string
	handler := func(c echo.Context) error {
		seenTenant = TenantIDFromContext(c.Request().Context())
		return c.NoContent(http.StatusOK)
	}

	mw := Tenant(TenantConfig{
		Sources:       []TenantSource{TenantHeader("X-Tenant-ID")},
		DefaultTenant: "default",
		Required:      false,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := mw(handler)(c); err != nil {
		t.Errorf("middleware err: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if seenTenant != "tenant-1" {
		t.Errorf("expected tenant_id=tenant-1, got %q", seenTenant)
	}
	if rec.Header().Get("X-Tenant-ID") != "tenant-1" {
		t.Errorf("response header X-Tenant-ID = %q, want tenant-1", rec.Header().Get("X-Tenant-ID"))
	}
}

func TestTenantMiddlewareDefaultFallback(t *testing.T) {
	e := echo.New()
	var seenTenant string
	handler := func(c echo.Context) error {
		seenTenant = TenantIDFromContext(c.Request().Context())
		return c.NoContent(http.StatusOK)
	}
	mw := Tenant(TenantConfig{
		Sources:       []TenantSource{TenantHeader("X-Tenant-ID")},
		DefaultTenant: "default",
		Required:      false,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := mw(handler)(c); err != nil {
		t.Errorf("middleware err: %v", err)
	}
	if seenTenant != "default" {
		t.Errorf("expected default tenant, got %q", seenTenant)
	}
}

func TestTenantFromContext(t *testing.T) {
	ctx := WithTenantID(context.Background(), "tenant-1")
	if got := TenantIDFromContext(ctx); got != "tenant-1" {
		t.Errorf("got %q, want tenant-1", got)
	}
}

func TestTenantFromContextMissing(t *testing.T) {
	got := TenantIDFromContext(context.Background())
	if got != "" {
		t.Errorf("expected empty tenant, got %q", got)
	}
}

func TestTenantInfoRoundTrip(t *testing.T) {
	info := &TenantInfo{
		ID:     "tenant-1",
		Name:   "Apex Fintech",
		Status: "active",
		Plan:   "pro",
	}
	ctx := WithTenantInfo(context.Background(), info)
	got := TenantInfoFromContext(ctx)
	if got == nil || got.ID != "tenant-1" || got.Plan != "pro" {
		t.Errorf("TenantInfo round-trip failed: %+v", got)
	}
}

func TestTenantScope(t *testing.T) {
	ctx := TenantScope(context.Background(), "tenant-scope")
	if got := TenantIDFromContext(ctx); got != "tenant-scope" {
		t.Errorf("got %q, want tenant-scope", got)
	}
}

func TestTenantDefaultConfig(t *testing.T) {
	cfg := TenantDefaultConfig()
	if len(cfg.Sources) == 0 {
		t.Error("default config should have sources")
	}
	if !cfg.Required {
		t.Error("default config should require tenant")
	}
}
