// Extra tests for landing-service middleware.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestExtraRecovery_NoPanic(t *testing.T) {
	e := echo.New()
	e.GET("/ok", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(Recovery())

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestExtraRecovery_WithPanic(t *testing.T) {
	e := echo.New()
	e.GET("/panic", func(c echo.Context) error {
		panic("boom")
	})
	e.Use(Recovery())

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestExtraRequestLog_OK(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(RequestLog())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestExtraCORS_RegularRequest(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(CORS())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("CORS header missing")
	}
}

func TestExtraCORS_Preflight(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(CORS())

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}

func TestExtraSecurityHeaders(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(SecurityHeaders())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options")
	}
}

func TestExtraEditorAuth_NoHeaders(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(EditorAuth)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestExtraEditorAuth_HasUserID(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(EditorAuth)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", "u1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestExtraEditorAuth_HasAuth(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(EditorAuth)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestExtraTenantHelper(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-x")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if got := Tenant(c); got != "tenant-x" {
		t.Errorf("got %q, want tenant-x", got)
	}
}

func TestExtraTenantHelper_Empty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if got := Tenant(c); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtraUserHelper(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "user-x")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if got := User(c); got != "user-x" {
		t.Errorf("got %q, want user-x", got)
	}
}

func TestExtraUserHelper_Empty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if got := User(c); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
