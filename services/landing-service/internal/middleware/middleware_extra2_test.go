package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func newCtx(req *http.Request) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func TestRecoveryNoPanic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c, rec := newCtx(req)

	mw := Recovery()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = mw(c)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestRecoveryWithPanic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c, rec := newCtx(req)

	mw := Recovery()(func(c echo.Context) error {
		panic("boom")
	})
	_ = mw(c)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestRequestLogBasic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c, rec := newCtx(req)

	mw := RequestLog()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = mw(c)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestRequestLogRecordsError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/fail", nil)
	c, rec := newCtx(req)

	mw := RequestLog()(func(c echo.Context) error {
		return c.String(http.StatusBadRequest, "bad")
	})
	_ = mw(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestCORSGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c, rec := newCtx(req)

	mw := CORS()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = mw(c)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("ACAO = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "GET") {
		t.Errorf("methods = %q", got)
	}
}

func TestCORSOptions(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	c, rec := newCtx(req)

	mw := CORS()(func(c echo.Context) error {
		t.Error("should not reach handler on OPTIONS")
		return nil
	})
	_ = mw(c)
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c, rec := newCtx(req)

	mw := SecurityHeaders()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = mw(c)
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("nosniff missing: %q", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("DENY missing: %q", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got == "" {
		t.Error("referrer-policy missing")
	}
}

func TestEditorAuthUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "u1")
	c, rec := newCtx(req)

	called := false
	mw := EditorAuth(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})
	_ = mw(c)
	if !called {
		t.Error("handler not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestEditorAuthBearer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer xxx")
	c, rec := newCtx(req)

	called := false
	mw := EditorAuth(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})
	_ = mw(c)
	if !called {
		t.Error("handler not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestEditorAuthMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c, rec := newCtx(req)

	called := false
	mw := EditorAuth(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})
	_ = mw(c)
	if called {
		t.Error("handler called without auth")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestTenantFromContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-x")
	c, _ := newCtx(req)

	if got := Tenant(c); got != "tenant-x" {
		t.Errorf("Tenant = %q", got)
	}
}

func TestTenantEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c, _ := newCtx(req)
	if got := Tenant(c); got != "" {
		t.Errorf("empty Tenant expected, got %q", got)
	}
}

func TestUserFromContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "u42")
	c, _ := newCtx(req)
	if got := User(c); got != "u42" {
		t.Errorf("User = %q", got)
	}
}

func TestUserEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c, _ := newCtx(req)
	if got := User(c); got != "" {
		t.Errorf("empty User, got %q", got)
	}
}

