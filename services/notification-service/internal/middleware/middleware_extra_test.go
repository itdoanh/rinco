// Extra tests for notification-service middleware.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestExtraRecoveryMW_PanicsRecovered(t *testing.T) {
	e := echo.New()
	e.GET("/panic", func(c echo.Context) error {
		panic("boom")
	})
	e.Use(RecoveryMW())

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestExtraRecoveryMW_NoPanic(t *testing.T) {
	e := echo.New()
	e.GET("/ok", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.Use(RecoveryMW())

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestExtraCORSMW_Options(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(CORSMW())

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("CORS header missing")
	}
}

func TestExtraCORSMW_ActualRequest(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(CORSMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Access-Control-Allow-Methods missing")
	}
}

func TestExtraSecurityHeadersMW(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(SecurityHeadersMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options missing")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options missing")
	}
	if rec.Header().Get("Referrer-Policy") == "" {
		t.Error("Referrer-Policy missing")
	}
}

func TestExtraTenantMW_ValidHeader(t *testing.T) {
	e := echo.New()
	var capturedTenant string
	e.GET("/test", func(c echo.Context) error {
		capturedTenant = c.Get("tenant_id").(string)
		return c.NoContent(http.StatusOK)
	})
	e.Use(TenantMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "my-tenant")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if capturedTenant != "my-tenant" {
		t.Errorf("tenant = %q, want my-tenant", capturedTenant)
	}
}

func TestExtraTenantMW_MissingHeader(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(TenantMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestExtraTenantMW_WhitespaceTrimmed(t *testing.T) {
	e := echo.New()
	var capturedTenant string
	e.GET("/test", func(c echo.Context) error {
		capturedTenant = c.Get("tenant_id").(string)
		return c.NoContent(http.StatusOK)
	})
	e.Use(TenantMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "  my-tenant  ")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if capturedTenant != "my-tenant" {
		t.Errorf("tenant = %q, want my-tenant", capturedTenant)
	}
}

func TestExtraTenantMW_StoresUserID(t *testing.T) {
	e := echo.New()
	var capturedUser string
	e.GET("/test", func(c echo.Context) error {
		capturedUser = c.Get("user_id").(string)
		return c.NoContent(http.StatusOK)
	})
	e.Use(TenantMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-User-ID", "u123")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if capturedUser != "u123" {
		t.Errorf("user = %q, want u123", capturedUser)
	}
}

func TestExtraTenantMW_StoresAdminFlag(t *testing.T) {
	e := echo.New()
	var capturedAdmin bool
	e.GET("/test", func(c echo.Context) error {
		capturedAdmin = c.Get("is_admin").(bool)
		return c.NoContent(http.StatusOK)
	})
	e.Use(TenantMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Admin", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if !capturedAdmin {
		t.Error("is_admin should be true")
	}
}

func TestExtraTenantMW_AdminCaseInsensitive(t *testing.T) {
	e := echo.New()
	var capturedAdmin bool
	e.GET("/test", func(c echo.Context) error {
		capturedAdmin = c.Get("is_admin").(bool)
		return c.NoContent(http.StatusOK)
	})
	e.Use(TenantMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Admin", "TRUE")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if !capturedAdmin {
		t.Error("is_admin should be true for TRUE")
	}
}

func TestExtraAuthMW_NoAuth(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(TenantMW())
	e.Use(AuthMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestExtraAuthMW_MissingTenantAndAuth(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(AuthMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestExtraAuthMW_HasBearer(t *testing.T) {
	e := echo.New()
	var called bool
	e.GET("/test", func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	e.Use(AuthMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !called {
		t.Error("handler should be called")
	}
}

func TestExtraRateLimitMW_NilRedis_NoOp(t *testing.T) {
	e := echo.New()
	var called bool
	e.GET("/test", func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	e.Use(RateLimitMW(nil, 10, 0))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should be called with nil Redis")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestExtraLoggingMW_CallsNext(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusCreated)
	})
	e.Use(LoggingMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
}
