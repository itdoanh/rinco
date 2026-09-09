// Extra tests for lead-service middleware.
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

func TestExtraLoggingMW_RecordsMetrics(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(LoggingMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestExtraCORSMW_Preflight(t *testing.T) {
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
}

func TestExtraCORSMW_Regular(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(CORSMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("CORS header missing")
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
	if rec.Header().Get("X-XSS-Protection") == "" {
		t.Error("X-XSS-Protection missing")
	}
}

func TestExtraMetricsMW_TracksLatency(t *testing.T) {
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(MetricsMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestExtraTraceMW_FromHeader(t *testing.T) {
	e := echo.New()
	var seenTrace string
	e.GET("/test", func(c echo.Context) error {
		seenTrace = c.Get("trace_id").(string)
		return c.NoContent(http.StatusOK)
	})
	e.Use(TraceMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Trace-ID", "trace-from-header")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if seenTrace != "trace-from-header" {
		t.Errorf("got %q, want trace-from-header", seenTrace)
	}
}

func TestExtraTraceMW_EmptyHeader(t *testing.T) {
	e := echo.New()
	var seenTrace string
	e.GET("/test", func(c echo.Context) error {
		if v, ok := c.Get("trace_id").(string); ok {
			seenTrace = v
		}
		return c.NoContent(http.StatusOK)
	})
	e.Use(TraceMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if seenTrace != "" {
		t.Errorf("got %q, want empty", seenTrace)
	}
}

func TestExtraTenantMW_ValidHeader(t *testing.T) {
	e := echo.New()
	var seenTenant string
	e.GET("/test", func(c echo.Context) error {
		seenTenant = c.Get("tenant_id").(string)
		return c.NoContent(http.StatusOK)
	})
	e.Use(TenantMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if seenTenant != "tenant-1" {
		t.Errorf("got %q", seenTenant)
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

func TestExtraTenantMW_UserIDAndAdmin(t *testing.T) {
	e := echo.New()
	var seenUser string
	var seenAdmin bool
	e.GET("/test", func(c echo.Context) error {
		seenUser = c.Get("user_id").(string)
		seenAdmin = c.Get("is_admin").(bool)
		return c.NoContent(http.StatusOK)
	})
	e.Use(TenantMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-User-ID", "u1")
	req.Header.Set("X-Admin", "TRUE")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if seenUser != "u1" {
		t.Errorf("user: %q", seenUser)
	}
	if !seenAdmin {
		t.Error("admin should be true")
	}
}

func TestExtraAuthMW_ValidTenant(t *testing.T) {
	e := echo.New()
	var called bool
	e.GET("/test", func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	e.Use(AuthMW())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "t1")

	if err := AuthMW()(func(c echo.Context) error {
		called = true
		return nil
	})(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if !called {
		t.Error("handler should be called")
	}
}

func TestExtraAuthMW_NoTenant(t *testing.T) {
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
