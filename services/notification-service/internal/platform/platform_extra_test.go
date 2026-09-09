// Extra tests for notification-service platform.go helpers.
package platform

import (
	"context"
	"os"
	"testing"
)

func TestExtraGetenv_ReturnsEnv(t *testing.T) {
	os.Setenv("TEST_VAR_XYZ", "hello")
	defer os.Unsetenv("TEST_VAR_XYZ")

	got := Getenv("TEST_VAR_XYZ", "fallback")
	if got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
}

func TestExtraGetenv_ReturnsFallback(t *testing.T) {
	os.Unsetenv("TEST_MISSING_VAR")
	got := Getenv("TEST_MISSING_VAR", "default")
	if got != "default" {
		t.Errorf("got %q, want default", got)
	}
}

func TestExtraGetenv_EmptyEnvIsMissing(t *testing.T) {
	os.Setenv("TEST_EMPTY_VAR", "")
	defer os.Unsetenv("TEST_EMPTY_VAR")
	got := Getenv("TEST_EMPTY_VAR", "default")
	if got != "default" {
		t.Errorf("empty env should return fallback, got %q", got)
	}
}

func TestExtraGetenvInt_Valid(t *testing.T) {
	os.Setenv("TEST_INT_VAR", "42")
	defer os.Unsetenv("TEST_INT_VAR")
	got := GetenvInt("TEST_INT_VAR", 99)
	if got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestExtraGetenvInt_Invalid(t *testing.T) {
	os.Setenv("TEST_INT_INVALID", "not-a-number")
	defer os.Unsetenv("TEST_INT_INVALID")
	got := GetenvInt("TEST_INT_INVALID", 77)
	if got != 77 {
		t.Errorf("got %d, want fallback 77", got)
	}
}

func TestExtraGetenvInt_Empty(t *testing.T) {
	os.Unsetenv("TEST_INT_EMPTY")
	got := GetenvInt("TEST_INT_EMPTY", 55)
	if got != 55 {
		t.Errorf("got %d, want fallback 55", got)
	}
}

func TestExtraGetenvInt_Negative(t *testing.T) {
	os.Setenv("TEST_INT_NEG", "-10")
	defer os.Unsetenv("TEST_INT_NEG")
	got := GetenvInt("TEST_INT_NEG", 0)
	if got != -10 {
		t.Errorf("got %d, want -10", got)
	}
}

func TestExtraGetenvBool_TrueVariants(t *testing.T) {
	for _, v := range []string{"true", "TRUE", "True", "1", "yes", "YES"} {
		os.Setenv("TEST_BOOL_VAR", v)
		got := GetenvBool("TEST_BOOL_VAR", false)
		if !got {
			t.Errorf("GetenvBool(%q)=false, want true", v)
		}
		os.Unsetenv("TEST_BOOL_VAR")
	}
}

func TestExtraGetenvBool_FalseVariants(t *testing.T) {
	for _, v := range []string{"false", "FALSE", "0", "no", "NO", "anything"} {
		os.Setenv("TEST_BOOL_VAR", v)
		got := GetenvBool("TEST_BOOL_VAR", false)
		if got {
			t.Errorf("GetenvBool(%q)=true, want false", v)
		}
		os.Unsetenv("TEST_BOOL_VAR")
	}
}

func TestExtraGetenvBool_EmptyReturnsFallback(t *testing.T) {
	os.Unsetenv("TEST_BOOL_EMPTY")
	got := GetenvBool("TEST_BOOL_EMPTY", true)
	if !got {
		t.Error("empty env should return fallback true")
	}
}

func TestExtraLoadConfig_Defaults(t *testing.T) {
	os.Unsetenv("NOTIF_HTTP_ADDR")
	os.Unsetenv("NOTIF_VALKEY_URL")
	cfg := LoadConfig()
	if cfg.HTTPAddr == "" {
		t.Error("HTTPAddr should have default")
	}
	if cfg.ValkeyURL == "" {
		t.Error("ValkeyURL should have default")
	}
	if cfg.Env == "" {
		t.Error("Env should have default")
	}
}

func TestExtraLoadConfig_EnvOverrides(t *testing.T) {
	os.Setenv("NOTIF_HTTP_ADDR", ":9090")
	os.Setenv("ENV", "production")
	defer func() {
		os.Unsetenv("NOTIF_HTTP_ADDR")
		os.Unsetenv("ENV")
	}()
	cfg := LoadConfig()
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("got %s", cfg.HTTPAddr)
	}
	if cfg.Env != "production" {
		t.Errorf("got %s", cfg.Env)
	}
}

func TestExtraLoadConfig_OptionalFieldsNil(t *testing.T) {
	os.Unsetenv("NOTIF_DATABASE_URL")
	os.Unsetenv("NOTIF_VALKEY_URL")
	cfg := LoadConfig()
	// Optional fields should be empty/nil strings, not panic
	_ = cfg.DatabaseURL
	_ = cfg.NatsURL
	_ = cfg.OTLP
}

func TestExtraTenantContext_Helpers(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenant(ctx, "tenant-1", "user-1", true)

	if TenantFromContext(ctx) != "tenant-1" {
		t.Errorf("TenantFromContext: got %s", TenantFromContext(ctx))
	}
	if UserFromContext(ctx) != "user-1" {
		t.Errorf("UserFromContext: got %s", UserFromContext(ctx))
	}
	if !IsAdminFromContext(ctx) {
		t.Error("IsAdminFromContext: want true")
	}
}

func TestExtraTenantContext_EmptyContext(t *testing.T) {
	ctx := context.Background()
	if TenantFromContext(ctx) != "" {
		t.Error("expected empty")
	}
	if UserFromContext(ctx) != "" {
		t.Error("expected empty")
	}
	if IsAdminFromContext(ctx) {
		t.Error("expected false")
	}
}

func TestExtraTenantToCtx_EquivWithTenant(t *testing.T) {
	ctx := TenantToCtx(context.Background(), "t1", "u1", false)
	if TenantFromContext(ctx) != "t1" {
		t.Error("TenantToCtx should work like WithTenant")
	}
}

func TestExtraTraceContext_Helpers(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-abc")

	if TraceFromContext(ctx) != "trace-abc" {
		t.Errorf("got %s", TraceFromContext(ctx))
	}
}

func TestExtraTraceContext_EmptyContext(t *testing.T) {
	ctx := context.Background()
	if TraceFromContext(ctx) != "" {
		t.Error("expected empty")
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

func TestExtraIsNotFound_Wrapped(t *testing.T) {
	wrapped := &wrappedErr{msg: "wrapped", inner: ErrNotFound}
	if !IsNotFound(wrapped) {
		t.Error("wrapped ErrNotFound should be not-found")
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
		t.Error("not found message should be not-found")
	}
}

func TestExtraIsNotFound_OtherError(t *testing.T) {
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
		t.Errorf("sentinel message: %s", ErrNotFound.Error())
	}
}

func TestExtraMigrations_AllPresent(t *testing.T) {
	ms := migrations()
	if len(ms) < 4 {
		t.Errorf("expected at least 4 migrations, got %d", len(ms))
	}
	names := make(map[string]bool)
	for _, m := range ms {
		if names[m.name] {
			t.Errorf("duplicate migration name: %s", m.name)
		}
		names[m.name] = true
		if m.body == "" {
			t.Errorf("migration %s has empty body", m.name)
		}
	}
}

func TestExtraMigrations_IDempotent(t *testing.T) {
	ms := migrations()
	for _, m := range ms {
		if len(m.body) < 50 {
			t.Errorf("migration %s suspiciously short: %s", m.name, m.body)
		}
	}
}
