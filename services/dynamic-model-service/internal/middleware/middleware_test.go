// Tests for dynamic-model-service middleware.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestExtra_Recovery(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := Recovery()(func(c echo.Context) error {
		panic("test")
	})
	_ = mw(c)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500: got %d", rec.Code)
	}
}

func TestExtra_RequestLog(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := RequestLog()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
}

func TestExtra_CORS(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := CORS()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS")
	}
}

func TestExtra_CORS_Preflight(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	mw := CORS()(func(c echo.Context) error {
		called = true
		return nil
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if called {
		t.Error("next should not be called")
	}
}

func TestExtra_SecurityHeaders(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := SecurityHeaders()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := mw(c); err != nil {
		t.Errorf("middleware: %v", err)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing header")
	}
}

func TestExtra_Tenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if got := Tenant(c); got != "tenant-1" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_Tenant_Empty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if got := Tenant(c); got != "" {
		t.Errorf("expected empty: got %s", got)
	}
}

func TestExtra_User(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if got := User(c); got != "user-1" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_IsAdmin_True(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Is-Super-Admin", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if !IsAdmin(c) {
		t.Error("should be admin")
	}
}

func TestExtra_IsAdmin_False(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if IsAdmin(c) {
		t.Error("should not be admin")
	}
}
