// Extra tests for tenant-service helpers and middleware.
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestExtraGetEnv_DefaultWhenEmpty(t *testing.T) {
	t.Setenv("EXTRA_TEST_VAR", "")
	if got := getEnv("EXTRA_TEST_VAR", "default"); got != "default" {
		t.Errorf("getEnv empty: got %q, want default", got)
	}
}

func TestExtraGetEnv_ReturnsValue(t *testing.T) {
	t.Setenv("EXTRA_TEST_VAR", "actual")
	if got := getEnv("EXTRA_TEST_VAR", "default"); got != "actual" {
		t.Errorf("getEnv: got %q, want actual", got)
	}
}

func TestExtraAsString_NonString(t *testing.T) {
	if got := asString(42); got != "" {
		t.Errorf("asString int: got %q", got)
	}
}

func TestExtraAsString_String(t *testing.T) {
	if got := asString("hello"); got != "hello" {
		t.Errorf("asString: got %q", got)
	}
}

func TestExtraAsString_Nil(t *testing.T) {
	if got := asString(nil); got != "" {
		t.Errorf("asString nil: got %q", got)
	}
}

func TestExtraAsBool_True(t *testing.T) {
	if !asBool(true) {
		t.Error("true should return true")
	}
}

func TestExtraAsBool_False(t *testing.T) {
	if asBool(false) {
		t.Error("false should return false")
	}
}

func TestExtraAsBool_StringTrue(t *testing.T) {
	if !asBool("true") {
		t.Error("string 'true' should return true")
	}
}

func TestExtraAsBool_StringOne(t *testing.T) {
	if !asBool("1") {
		t.Error("string '1' should return true")
	}
}

func TestExtraAsBool_StringFalse(t *testing.T) {
	if asBool("false") {
		t.Error("string 'false' should return false")
	}
}

func TestExtraAsBool_Int(t *testing.T) {
	if asBool(1) {
		t.Error("int 1 (not string/bool) should return false")
	}
}

func TestExtraFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "x", "y"); got != "x" {
		t.Errorf("firstNonEmpty: got %q", got)
	}
}

func TestExtraFirstNonEmpty_AllEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", ""); got != "" {
		t.Errorf("firstNonEmpty all empty: got %q", got)
	}
}

func TestExtraNullStr_Empty(t *testing.T) {
	if got := nullStr(""); got != nil {
		t.Errorf("nullStr empty: got %v", got)
	}
}

func TestExtraNullStr_NonEmpty(t *testing.T) {
	if got := nullStr("hello"); got != "hello" {
		t.Errorf("nullStr: got %v", got)
	}
}

func TestExtraAtoiDefault_Empty(t *testing.T) {
	if got := atoiDefault("", 99); got != 99 {
		t.Errorf("atoiDefault empty: got %d", got)
	}
}

func TestExtraAtoiDefault_BadStr(t *testing.T) {
	if got := atoiDefault("not-a-number", 99); got != 99 {
		t.Errorf("atoiDefault bad: got %d", got)
	}
}

func TestExtraAtoiDefault_Valid(t *testing.T) {
	if got := atoiDefault("42", 0); got != 42 {
		t.Errorf("atoiDefault valid: got %d", got)
	}
}

func TestExtraFirstMap_Nil(t *testing.T) {
	m := firstMap(nil)
	if m == nil {
		t.Error("firstMap nil: got nil, want empty")
	}
	if len(m) != 0 {
		t.Errorf("firstMap nil: len = %d", len(m))
	}
}

func TestExtraFirstMap_NonNil(t *testing.T) {
	in := map[string]any{"a": "b"}
	out := firstMap(in)
	if out == nil {
		t.Fatal("firstMap non-nil: got nil")
	}
	if out["a"] != "b" {
		t.Errorf("firstMap lost data")
	}
}

func TestExtraIsUniqueViolation_True(t *testing.T) {
	if !isUniqueViolation(&echo.HTTPError{Message: "duplicate key value"}) {
		t.Error("should be detected as unique violation")
	}
}

func TestExtraIsUniqueViolation_False(t *testing.T) {
	if isUniqueViolation(&echo.HTTPError{Message: "other error"}) {
		t.Error("should not be unique violation")
	}
}

func TestExtraIsUniqueViolation_Nil(t *testing.T) {
	if isUniqueViolation(nil) {
		t.Error("nil error should not be unique violation")
	}
}

func TestExtraJsonErr(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := jsonErr(c, 400, "TEST", "test msg")
	if err != nil {
		t.Fatalf("jsonErr returned err: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestExtraSubtleCompare_Equal(t *testing.T) {
	if !subtleCompare("abc", "abc") {
		t.Error("equal strings should match")
	}
}

func TestExtraSubtleCompare_DiffLen(t *testing.T) {
	if subtleCompare("abc", "abcd") {
		t.Error("different length should not match")
	}
}

func TestExtraSubtleCompare_Diff(t *testing.T) {
	if subtleCompare("abc", "xyz") {
		t.Error("different strings should not match")
	}
}

func TestExtraSubtleCompare_Empty(t *testing.T) {
	if !subtleCompare("", "") {
		t.Error("empty should match empty")
	}
}

func TestExtraRecoveryMW_NormalFlow(t *testing.T) {
	e := echo.New()
	srv := &server{log: nil}
	mw := srv.recoveryMW()

	called := false
	handler := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	e.GET("/", handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestExtraTraceMW_GeneratesID(t *testing.T) {
	e := echo.New()
	mw := traceMW()
	called := false
	handler := mw(func(c echo.Context) error {
		called = true
		tid := c.Get("trace_id")
		if tid == nil || tid == "" {
			t.Error("trace_id should be set")
		}
		return c.NoContent(http.StatusOK)
	})
	e.GET("/", handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should be called")
	}
}

func TestExtraTraceMW_UsesExistingID(t *testing.T) {
	e := echo.New()
	mw := traceMW()
	handler := mw(func(c echo.Context) error {
		tid := c.Get("trace_id")
		if tid != "existing-id" {
			t.Errorf("trace_id: got %v", tid)
		}
		return c.NoContent(http.StatusOK)
	})
	e.GET("/", handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Trace-ID", "existing-id")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
}

func TestExtraCorsMW_Headers(t *testing.T) {
	e := echo.New()
	e.Use(corsMW())
	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("missing CORS origin")
	}
}

func TestExtraCorsMW_Preflight(t *testing.T) {
	e := echo.New()
	e.Use(corsMW())
	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight: got %d, want 204", rec.Code)
	}
}

func TestExtraSecurityHeadersMW(t *testing.T) {
	e := echo.New()
	e.Use(securityHeadersMW())
	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options")
	}
}

func TestExtraRequireTenantMW_Missing(t *testing.T) {
	e := echo.New()
	srv := &server{}
	e.Use(srv.requireTenantMW())
	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing tenant: got %d", rec.Code)
	}
}

func TestExtraRequireTenantMW_Present(t *testing.T) {
	e := echo.New()
	srv := &server{}
	e.Use(srv.requireTenantMW())
	e.GET("/", func(c echo.Context) error {
		if c.Get("tenant_id") != "t1" {
			t.Errorf("tenant_id not set: %v", c.Get("tenant_id"))
		}
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestExtraNewID_Unique(t *testing.T) {
	a := newID()
	b := newID()
	if a == b {
		t.Error("newID should return unique IDs")
	}
	if len(a) < 32 {
		t.Errorf("newID too short: %s", a)
	}
}
