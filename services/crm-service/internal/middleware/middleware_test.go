// Tests for crm-service middleware.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRecoveryMW(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := RecoveryMW()(func(c echo.Context) error {
		panic("test panic")
	})
	if err := mw(c); err != nil {
		t.Errorf("recovery should swallow panic: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestLoggingMW(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := LoggingMW()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
}

func TestCORSMW(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := CORSMW()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS origin header")
	}
}

func TestCORSMW_Preflight(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := CORSMW()(func(c echo.Context) error {
		t.Error("next should not be called for OPTIONS")
		return nil
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestSecurityHeadersMW(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := SecurityHeadersMW()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options")
	}
}

func TestMetricsMW(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := MetricsMW()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
}

func TestTraceMW(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Trace-ID", "trace-abc")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	mw := TraceMW()(func(c echo.Context) error {
		called = true
		traceID := c.Request().Context().Value("trace_id")
		if traceID != "trace-abc" {
			t.Errorf("expected trace_id in context, got %v", traceID)
		}
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
	if !called {
		t.Error("next was not called")
	}
}

func TestTraceMW_NoHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := TraceMW()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
}

func TestTenantMW_MissingTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := TenantMW()(func(c echo.Context) error {
		t.Error("next should not be called without tenant")
		return nil
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestTenantMW_WithTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("X-Admin", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := TenantMW()(func(c echo.Context) error {
		if c.Get("tenant_id") != "tenant-1" {
			t.Error("tenant_id not set in context")
		}
		if c.Get("user_id") != "user-1" {
			t.Error("user_id not set in context")
		}
		if c.Get("is_admin") != true {
			t.Error("is_admin not set")
		}
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
}

func TestTenantMW_AdminFalse(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Admin", "false")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := TenantMW()(func(c echo.Context) error {
		if c.Get("is_admin") != false {
			t.Error("is_admin should be false")
		}
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
}

func TestAuthMW_NoTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := AuthMW()(func(c echo.Context) error {
		t.Error("next should not be called without auth")
		return nil
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMW_WithTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")

	called := false
	mw := AuthMW()(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware error: %v", err)
	}
	if !called {
		t.Error("next should be called")
	}
}
