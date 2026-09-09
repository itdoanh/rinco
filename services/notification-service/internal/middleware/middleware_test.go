// Tests for notification-service middleware.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestExtra_RecoveryMW(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := RecoveryMW()(func(c echo.Context) error {
		panic("test")
	})
	if err := mw(c); err != nil {
		t.Errorf("recovery should swallow: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500: got %d", rec.Code)
	}
}

func TestExtra_LoggingMW(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := LoggingMW()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
}

func TestExtra_CORSMW(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := CORSMW()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}

func TestExtra_CORSMW_Preflight(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	mw := CORSMW()(func(c echo.Context) error {
		called = true
		return nil
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if called {
		t.Error("next should not be called for OPTIONS")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204: got %d", rec.Code)
	}
}

func TestExtra_SecurityHeadersMW(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := SecurityHeadersMW()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing header")
	}
}

func TestExtra_TenantMW_Missing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	mw := TenantMW()(func(c echo.Context) error {
		called = true
		return nil
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if called {
		t.Error("next should not be called without tenant")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400: got %d", rec.Code)
	}
}

func TestExtra_TenantMW_Present(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := TenantMW()(func(c echo.Context) error {
		if c.Get("tenant_id") != "tenant-1" {
			t.Error("tenant_id not set")
		}
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
}

func TestExtra_TenantMW_Admin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Admin", "TRUE")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := TenantMW()(func(c echo.Context) error {
		if c.Get("is_admin") != true {
			t.Error("is_admin should be true")
		}
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
}

func TestExtra_AuthMW_NoAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := AuthMW()(func(c echo.Context) error {
		t.Error("next should not be called")
		return nil
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401: got %d", rec.Code)
	}
}

func TestExtra_AuthMW_WithTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tenant-1")

	called := false
	mw := AuthMW()(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if !called {
		t.Error("next should be called")
	}
}

func TestExtra_AuthMW_WithHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer xxx")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	mw := AuthMW()(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if !called {
		t.Error("next should be called")
	}
}

func TestExtra_RateLimitMW_NilRedis(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	mw := RateLimitMW(nil, 100, time.Second)(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if !called {
		t.Error("nil redis should be no-op")
	}
}
