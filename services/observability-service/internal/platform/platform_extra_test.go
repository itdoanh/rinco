// Extra tests for observability-service platform.go helpers.
package platform

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestExtraGetenv_ReturnsEnv(t *testing.T) {
	os.Setenv("TEST_VAR_OBS", "val123")
	defer os.Unsetenv("TEST_VAR_OBS")
	got := Getenv("TEST_VAR_OBS", "fallback")
	if got != "val123" {
		t.Errorf("got %q, want val123", got)
	}
}

func TestExtraGetenv_ReturnsFallback(t *testing.T) {
	os.Unsetenv("TEST_MISSING")
	got := Getenv("TEST_MISSING", "default")
	if got != "default" {
		t.Errorf("got %q, want default", got)
	}
}

func TestExtraGetenv_EmptyEnv(t *testing.T) {
	os.Setenv("TEST_EMPTY_VAR", "")
	defer os.Unsetenv("TEST_EMPTY_VAR")
	got := Getenv("TEST_EMPTY_VAR", "fallback")
	if got != "fallback" {
		t.Errorf("empty env: got %q", got)
	}
}

func TestExtraGetenvInt_Valid(t *testing.T) {
	os.Setenv("TEST_INT", "123")
	defer os.Unsetenv("TEST_INT")
	got := GetenvInt("TEST_INT", 0)
	if got != 123 {
		t.Errorf("got %d", got)
	}
}

func TestExtraGetenvInt_Invalid(t *testing.T) {
	os.Setenv("TEST_INT_INV", "not-number")
	defer os.Unsetenv("TEST_INT_INV")
	got := GetenvInt("TEST_INT_INV", 99)
	if got != 99 {
		t.Errorf("got fallback %d", got)
	}
}

func TestExtraGetenvInt_Empty(t *testing.T) {
	os.Unsetenv("TEST_INT_EMPTY")
	got := GetenvInt("TEST_INT_EMPTY", 77)
	if got != 77 {
		t.Errorf("got %d", got)
	}
}

func TestExtraGetenvBool_TrueVariants(t *testing.T) {
	for _, v := range []string{"true", "TRUE", "True", "1", "yes"} {
		os.Setenv("TEST_BOOL", v)
		got := GetenvBool("TEST_BOOL", false)
		if !got {
			t.Errorf("GetenvBool(%q)=false, want true", v)
		}
		os.Unsetenv("TEST_BOOL")
	}
}

func TestExtraGetenvBool_FalseVariants(t *testing.T) {
	for _, v := range []string{"false", "0", "no", "anything"} {
		os.Setenv("TEST_BOOL", v)
		got := GetenvBool("TEST_BOOL", false)
		if got {
			t.Errorf("GetenvBool(%q)=true, want false", v)
		}
		os.Unsetenv("TEST_BOOL")
	}
}

func TestExtraGetenvBool_EmptyFallback(t *testing.T) {
	os.Unsetenv("TEST_BOOL_EMP")
	got := GetenvBool("TEST_BOOL_EMP", true)
	if !got {
		t.Error("empty should return fallback true")
	}
}

func TestExtraLoadConfig_Defaults(t *testing.T) {
	os.Unsetenv("OBSERVABILITY_HTTP_ADDR")
	os.Unsetenv("OBSERVABILITY_RATE_LIMIT")
	cfg := LoadConfig()
	if cfg.HTTPAddr == "" {
		t.Error("HTTPAddr missing default")
	}
	if cfg.RateLimitPerMin == 0 {
		t.Error("RateLimitPerMin missing default")
	}
}

func TestExtraLoadConfig_Overrides(t *testing.T) {
	os.Setenv("OBSERVABILITY_HTTP_ADDR", ":9999")
	os.Setenv("ENV", "production")
	defer func() {
		os.Unsetenv("OBSERVABILITY_HTTP_ADDR")
		os.Unsetenv("ENV")
	}()
	cfg := LoadConfig()
	if cfg.HTTPAddr != ":9999" {
		t.Errorf("got %s", cfg.HTTPAddr)
	}
	if cfg.Env != "production" {
		t.Errorf("got %s", cfg.Env)
	}
}

func TestExtraTenantContext_Helpers(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenant(ctx, "t1", "u1", true)

	if TenantFromContext(ctx) != "t1" {
		t.Errorf("got %s", TenantFromContext(ctx))
	}
	if UserFromContext(ctx) != "u1" {
		t.Errorf("got %s", UserFromContext(ctx))
	}
	if !IsAdminFromContext(ctx) {
		t.Error("want true")
	}
}

func TestExtraTenantContext_EmptyContext(t *testing.T) {
	ctx := context.Background()
	if TenantFromContext(ctx) != "" {
		t.Error("expected empty")
	}
	if IsAdminFromContext(ctx) {
		t.Error("expected false")
	}
}

func TestExtraIsNotFound_Nil(t *testing.T) {
	if IsNotFound(nil) {
		t.Error("nil should not be not-found")
	}
}

func TestExtraIsNotFound_ErrNotFound(t *testing.T) {
	if !IsNotFound(ErrNotFound) {
		t.Error("ErrNotFound should be not-found")
	}
}

func TestExtraIsNotFound_NoRows(t *testing.T) {
	err := &wrappedErr{msg: "no rows in result set"}
	if !IsNotFound(err) {
		t.Error("no rows should be not-found")
	}
}

func TestExtraIsNotFound_NotFound(t *testing.T) {
	err := &wrappedErr{msg: "not found"}
	if !IsNotFound(err) {
		t.Error("not found should be not-found")
	}
}

func TestExtraIsNotFound_Other(t *testing.T) {
	err := &wrappedErr{msg: "connection refused"}
	if IsNotFound(err) {
		t.Error("other errors should not be not-found")
	}
}

type wrappedErr struct {
	msg   string
	inner error
}

func (w *wrappedErr) Error() string { return w.msg }
func (w *wrappedErr) Unwrap() error { return w.inner }

func TestExtraErrNotFound_Sentinel(t *testing.T) {
	if ErrNotFound.Error() != "not found" {
		t.Errorf("got %s", ErrNotFound.Error())
	}
}

func TestExtraSplitDDL_Empty(t *testing.T) {
	parts := splitDDL("")
	if len(parts) != 0 {
		t.Errorf("empty: got %d parts", len(parts))
	}
}

func TestExtraSplitDDL_Single(t *testing.T) {
	parts := splitDDL("SELECT 1;")
	if len(parts) != 1 {
		t.Errorf("single: got %d", len(parts))
	}
	if parts[0] != "SELECT 1" {
		t.Errorf("got %q", parts[0])
	}
}

func TestExtraSplitDDL_Multiple(t *testing.T) {
	ddl := "SELECT 1;  SELECT 2; CREATE TABLE t ();"
	parts := splitDDL(ddl)
	if len(parts) != 3 {
		t.Errorf("got %d parts: %v", len(parts), parts)
	}
}

func TestExtraSplitDDL_Whitespace(t *testing.T) {
	parts := splitDDL("  SELECT 1  ;  ;  CREATE TABLE  ")
	if len(parts) != 2 {
		t.Errorf("got %d", len(parts))
	}
}

func TestExtraMigrations_AllPresent(t *testing.T) {
	ms := migrations()
	if len(ms) < 2 {
		t.Errorf("expected at least 2, got %d", len(ms))
	}
	names := make(map[string]bool)
	for _, m := range ms {
		if names[m.name] {
			t.Errorf("duplicate: %s", m.name)
		}
		names[m.name] = true
		if m.body == "" {
			t.Errorf("migration %s empty", m.name)
		}
	}
}

func TestExtraNewJSONClient_Defaults(t *testing.T) {
	c := NewJSONClient("http://localhost:9090", "key123", 10_000_000_000)
	if c.BaseURL != "http://localhost:9090" {
		t.Errorf("BaseURL: %s", c.BaseURL)
	}
	if c.APIKey != "key123" {
		t.Errorf("APIKey: %s", c.APIKey)
	}
	if c.HC == nil {
		t.Error("HC nil")
	}
}

func TestExtraNewJSONClient_ZeroTimeout(t *testing.T) {
	c := NewJSONClient("http://localhost:9090", "", 0)
	if c.HC.Timeout != 10*time.Second {
		t.Errorf("timeout: %v", c.HC.Timeout)
	}
}

func TestExtraNewJSONClient_TrimsTrailingSlash(t *testing.T) {
	c := NewJSONClient("http://localhost:9090/", "", 0)
	if c.BaseURL != "http://localhost:9090" {
		t.Errorf("got %s", c.BaseURL)
	}
}
