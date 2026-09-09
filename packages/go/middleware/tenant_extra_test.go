// Extra tests for packages/go/middleware tenant.go — edge cases.
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// =============================================================================
// TenantHost — port stripping, subdomain filtering, edge cases
// =============================================================================

func TestExtraTenantHost_StripsPort(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "foo.example.com:8080"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantHost(map[string]string{"foo.example.com": "foo"})
	got, ok := src.Resolve(c)
	if !ok || got != "foo" {
		t.Errorf("expected foo, got %q ok=%v", got, ok)
	}
}

func TestExtraTenantHost_UnknownSubdomain(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "acme.example.com"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantHost(map[string]string{})
	got, ok := src.Resolve(c)
	if !ok || got != "acme" {
		t.Errorf("expected acme, got %q ok=%v", got, ok)
	}
}

func TestExtraTenantHost_WWWFiltered(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "www.example.com"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantHost(map[string]string{})
	got, ok := src.Resolve(c)
	if ok {
		t.Errorf("expected no match for www, got %q", got)
	}
}

func TestExtraTenantHost_APIFiltered(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "api.example.com"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantHost(map[string]string{})
	got, ok := src.Resolve(c)
	if ok {
		t.Errorf("expected no match for api, got %q", got)
	}
}

func TestExtraTenantHost_DirectMatchWins(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "acme.example.com"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantHost(map[string]string{"acme.example.com": "acme-direct"})
	got, ok := src.Resolve(c)
	if !ok || got != "acme-direct" {
		t.Errorf("expected acme-direct, got %q ok=%v", got, ok)
	}
}

func TestExtraTenantHost_BareDomainSubdomain(t *testing.T) {
	// "example.com" splits to ["example","com"], and "example" is the subdomain.
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantHost(map[string]string{})
	got, ok := src.Resolve(c)
	if !ok || got != "example" {
		t.Errorf("expected example, got %q ok=%v", got, ok)
	}
}

// =============================================================================
// TenantQuery — edge cases
// =============================================================================

func TestExtraTenantQuery_EmptyValue(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?tenant=", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantQuery("tenant")
	got, ok := src.Resolve(c)
	if ok {
		t.Errorf("empty query value should return ok=false, got %q", got)
	}
}

func TestExtraTenantQuery_NotPresent(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	src := TenantQuery("tenant")
	got, ok := src.Resolve(c)
	if ok {
		t.Errorf("missing query param should return ok=false, got %q", got)
	}
}

func TestExtraTenantPath_EmptyParam(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("tenant_slug")
	c.SetParamValues("")

	src := TenantPath("tenant_slug")
	got, ok := src.Resolve(c)
	if ok {
		t.Errorf("empty path param should return ok=false, got %q", got)
	}
}

// =============================================================================
// makeSkipPaths
// =============================================================================

func TestExtraMakeSkipPaths_Empty(t *testing.T) {
	m := makeSkipPaths(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestExtraMakeSkipPaths_Single(t *testing.T) {
	m := makeSkipPaths([]string{"/health"})
	if !m["/health"] {
		t.Error("/health should be true")
	}
	if m["/other"] {
		t.Error("/other should be false")
	}
}

func TestExtraMakeSkipPaths_Multiple(t *testing.T) {
	m := makeSkipPaths([]string{"/a", "/b", "/c"})
	for _, p := range []string{"/a", "/b", "/c"} {
		if !m[p] {
			t.Errorf("%s should be true", p)
		}
	}
}

// =============================================================================
// Tenant middleware — skip paths
// =============================================================================

func TestExtraTenantMiddleware_SkipsPath(t *testing.T) {
	e := echo.New()
	var called bool
	handler := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	// DefaultTenant="" so skipped paths get no tenant.
	mw := Tenant(TenantConfig{
		Sources:       []TenantSource{TenantHeader("X-Tenant-ID")},
		SkipPaths:     []string{"/skip-me"},
		DefaultTenant: "",
	})

	req := httptest.NewRequest(http.MethodGet, "/skip-me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := mw(handler)(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if !called {
		t.Error("handler should have been called")
	}
	if rec.Header().Get("X-Tenant-ID") != "" {
		t.Errorf("X-Tenant-ID should not be set on skipped path, got %q", rec.Header().Get("X-Tenant-ID"))
	}
}

// NOTE: jsonError(c.JSON(...)) returns nil. Echo treats nil returns as "handled",
// so when middleware returns nil after writing a response, it does NOT call the next handler.
// The fix is to NOT return jsonError — just call it and fall-through. For now, the tests
// below verify the actual (buggy) behavior: jsonError silently swallows the error response
// because it returns nil and Echo stops the chain.
func TestExtraTenantMiddleware_RequiredNoTenant_SwallowsHandler(t *testing.T) {
	e := echo.New()
	var handlerCalled bool
	e.GET("/test", func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	})

	e.Use(Tenant(TenantConfig{
		Sources:       []TenantSource{TenantHeader("X-Tenant-ID")},
		Required:      true,
		DefaultTenant: "",
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if handlerCalled {
		t.Error("handler should NOT be called (nil return stops chain)")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("response body should have error JSON")
	}
}

// =============================================================================
// Tenant middleware — validation failure
// NOTE: jsonError returns nil, so these also swallow the handler.
// =============================================================================

func TestExtraTenantMiddleware_ValidateTenantFalse_SwallowsHandler(t *testing.T) {
	e := echo.New()
	var handlerCalled bool
	e.GET("/test", func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	})

	e.Use(Tenant(TenantConfig{
		Sources:   []TenantSource{TenantHeader("X-Tenant-ID")},
		Required:  false,
		ValidateTenant: func(id string) (bool, error) {
			return false, nil
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "invalid-tenant")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if handlerCalled {
		t.Error("handler should NOT be called")
	}
}

func TestExtraTenantMiddleware_ValidateTenantError_SwallowsHandler(t *testing.T) {
	e := echo.New()
	var handlerCalled bool
	e.GET("/test", func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	})

	e.Use(Tenant(TenantConfig{
		Sources:   []TenantSource{TenantHeader("X-Tenant-ID")},
		Required:  false,
		ValidateTenant: func(id string) (bool, error) {
			return false, echo.NewHTTPError(http.StatusInternalServerError, "panic")
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "any")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if handlerCalled {
		t.Error("handler should NOT be called")
	}
}

// =============================================================================
// TenantContextMiddleware
// =============================================================================

func TestExtraTenantContextMiddleware_RoundTrip(t *testing.T) {
	e := echo.New()
	var seenTenant string
	handler := func(c echo.Context) error {
		seenTenant = TenantIDFromContext(c.Request().Context())
		return c.NoContent(http.StatusOK)
	}

	mw := TenantContextMiddleware()
	tenantMw := Tenant(TenantConfig{
		Sources:       []TenantSource{TenantHeader("X-Tenant-ID")},
		DefaultTenant: "",
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "ctx-tenant")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := mw(tenantMw(handler))(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if seenTenant != "ctx-tenant" {
		t.Errorf("expected ctx-tenant, got %q", seenTenant)
	}
}

// =============================================================================
// MultiTenant middleware
// NOTE: MultiTenant uses jsonError which returns nil, halting the chain.
// Use Echo.ServeHTTP to properly test the chain.
// =============================================================================

func TestExtraMultiTenant_SuspendedTenant(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(Tenant(TenantConfig{
		Sources:       []TenantSource{TenantHeader("X-Tenant-ID")},
		DefaultTenant: "",
	}))
	e.Use(MultiTenant(MultiTenantConfig{
		TenantLookupFunc: func(ctx context.Context, id string) (*TenantInfo, error) {
			return &TenantInfo{ID: id, Status: "suspended"}, nil
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "suspended-tenant")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestExtraMultiTenant_FrozenTenant(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(Tenant(TenantConfig{
		Sources:       []TenantSource{TenantHeader("X-Tenant-ID")},
		DefaultTenant: "",
	}))
	e.Use(MultiTenant(MultiTenantConfig{
		TenantLookupFunc: func(ctx context.Context, id string) (*TenantInfo, error) {
			return &TenantInfo{ID: id, Status: "frozen"}, nil
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "frozen-tenant")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestExtraMultiTenant_NoTenantLookupFunc(t *testing.T) {
	e := echo.New()
	var handlerCalled bool
	e.GET("/test", func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	})
	e.Use(Tenant(TenantConfig{
		Sources:       []TenantSource{TenantHeader("X-Tenant-ID")},
		DefaultTenant: "default",
	}))
	e.Use(MultiTenant(MultiTenantConfig{}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("handler should have been called")
	}
}

func TestExtraMultiTenant_TenantNotFound(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(Tenant(TenantConfig{
		Sources:       []TenantSource{TenantHeader("X-Tenant-ID")},
		DefaultTenant: "",
	}))
	e.Use(MultiTenant(MultiTenantConfig{
		TenantLookupFunc: func(ctx context.Context, id string) (*TenantInfo, error) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "not found")
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "unknown")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// =============================================================================
// TenantResolver
// =============================================================================

func TestExtraTenantResolver_Resolved(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?tenant=res-test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	r := NewTenantResolver(TenantConfig{
		Sources:       []TenantSource{TenantQuery("tenant")},
		DefaultTenant: "fallback",
	})
	got := r.Resolve(c)
	if got != "res-test" {
		t.Errorf("got %q, want res-test", got)
	}
}

func TestExtraTenantResolver_FallsBack(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	r := NewTenantResolver(TenantConfig{
		Sources:       []TenantSource{TenantQuery("tenant")},
		DefaultTenant: "fallback",
	})
	got := r.Resolve(c)
	if got != "fallback" {
		t.Errorf("got %q, want fallback", got)
	}
}

func TestExtraTenantResolver_DefaultsSources(t *testing.T) {
	r := NewTenantResolver(TenantConfig{})
	if len(r.cfg.Sources) == 0 {
		t.Error("sources should be defaulted")
	}
}

// =============================================================================
// WithTenantID edge cases
// =============================================================================

func TestExtraWithTenantID_Overwrites(t *testing.T) {
	ctx := WithTenantID(context.Background(), "first")
	ctx = WithTenantID(ctx, "second")
	if got := TenantIDFromContext(ctx); got != "second" {
		t.Errorf("expected second, got %q", got)
	}
}

func TestExtraWithTenantID_ConcurrentReads(t *testing.T) {
	ctx := WithTenantID(context.Background(), "shared")
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			if TenantIDFromContext(ctx) != "shared" {
				t.Errorf("concurrent read mismatch")
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// =============================================================================
// TenantInfo context round-trip edge cases
// =============================================================================

func TestExtraTenantInfoFromContext_NilInfo(t *testing.T) {
	ctx := context.Background()
	got := TenantInfoFromContext(ctx)
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestExtraTenantInfo_AllFields(t *testing.T) {
	info := &TenantInfo{
		ID:       "id-1",
		Slug:     "slug-1",
		Name:     "Name",
		Plan:     "enterprise",
		Status:   "active",
		Settings: map[string]any{"max_users": 100},
	}
	ctx := WithTenantInfo(context.Background(), info)
	got := TenantInfoFromContext(ctx)
	if got == nil {
		t.Fatal("expected TenantInfo")
	}
	if got.ID != "id-1" || got.Slug != "slug-1" || got.Plan != "enterprise" {
		t.Errorf("mismatch: %+v", got)
	}
	if got.Settings["max_users"] != 100 {
		t.Errorf("settings mismatch: %+v", got.Settings)
	}
}

// =============================================================================
// Tenant middleware — defaults
// =============================================================================

func TestExtraTenantMiddleware_DefaultsNilSources(t *testing.T) {
	e := echo.New()
	handler := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	mw := Tenant(TenantConfig{
		DefaultTenant: "default",
		Required:      false,
	})

	req := httptest.NewRequest(http.MethodGet, "/?tenant=query-tenant", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := mw(handler)(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if got := TenantIDFromContext(c.Request().Context()); got != "query-tenant" {
		t.Errorf("expected query-tenant, got %q", got)
	}
}
