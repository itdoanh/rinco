// Tests for email-service platform.
package platform

import (
	"context"
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := LoadConfig()
	if cfg.Env == "" {
		t.Error("Env should have default")
	}
	if cfg.HTTPAddr == "" {
		t.Error("HTTPAddr should have default")
	}
	if cfg.Driver == "" {
		t.Error("Driver should have default")
	}
}

func TestGetenv_Default(t *testing.T) {
	os.Unsetenv("EMAIL_TEST_VAR")
	if got := Getenv("EMAIL_TEST_VAR", "default"); got != "default" {
		t.Errorf("got %s, want default", got)
	}
}

func TestGetenv_Set(t *testing.T) {
	os.Setenv("EMAIL_TEST_VAR", "value")
	defer os.Unsetenv("EMAIL_TEST_VAR")
	if got := Getenv("EMAIL_TEST_VAR", "default"); got != "value" {
		t.Errorf("got %s, want value", got)
	}
}

func TestGetenv_EmptyEnv(t *testing.T) {
	os.Setenv("EMAIL_TEST_VAR", "")
	defer os.Unsetenv("EMAIL_TEST_VAR")
	if got := Getenv("EMAIL_TEST_VAR", "default"); got != "default" {
		t.Errorf("got %s, want default", got)
	}
}

func TestGetenvInt_Default(t *testing.T) {
	os.Unsetenv("EMAIL_TEST_INT")
	if got := GetenvInt("EMAIL_TEST_INT", 42); got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestGetenvInt_Set(t *testing.T) {
	os.Setenv("EMAIL_TEST_INT", "100")
	defer os.Unsetenv("EMAIL_TEST_INT")
	if got := GetenvInt("EMAIL_TEST_INT", 42); got != 100 {
		t.Errorf("got %d, want 100", got)
	}
}

func TestGetenvInt_Invalid(t *testing.T) {
	os.Setenv("EMAIL_TEST_INT", "not-a-number")
	defer os.Unsetenv("EMAIL_TEST_INT")
	if got := GetenvInt("EMAIL_TEST_INT", 42); got != 42 {
		t.Errorf("got %d, want default 42", got)
	}
}

func TestGetenvBool_Default(t *testing.T) {
	os.Unsetenv("EMAIL_TEST_BOOL")
	if !GetenvBool("EMAIL_TEST_BOOL", true) {
		t.Error("expected true default")
	}
}

func TestGetenvBool_True(t *testing.T) {
	os.Setenv("EMAIL_TEST_BOOL", "true")
	defer os.Unsetenv("EMAIL_TEST_BOOL")
	if !GetenvBool("EMAIL_TEST_BOOL", false) {
		t.Error("expected true")
	}
}

func TestGetenvBool_One(t *testing.T) {
	os.Setenv("EMAIL_TEST_BOOL", "1")
	defer os.Unsetenv("EMAIL_TEST_BOOL")
	if !GetenvBool("EMAIL_TEST_BOOL", false) {
		t.Error("expected true for '1'")
	}
}

func TestGetenvBool_Yes(t *testing.T) {
	os.Setenv("EMAIL_TEST_BOOL", "yes")
	defer os.Unsetenv("EMAIL_TEST_BOOL")
	if !GetenvBool("EMAIL_TEST_BOOL", false) {
		t.Error("expected true for 'yes'")
	}
}

func TestGetenvBool_False(t *testing.T) {
	os.Setenv("EMAIL_TEST_BOOL", "false")
	defer os.Unsetenv("EMAIL_TEST_BOOL")
	if GetenvBool("EMAIL_TEST_BOOL", true) {
		t.Error("expected false")
	}
}

func TestInitLogger_Default(t *testing.T) {
	log := InitLogger("test")
	if log == nil {
		t.Fatal("InitLogger returned nil")
	}
}

func TestInitLogger_DebugLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("dev")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_WarnLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("prod")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_ErrorLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "error")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("prod")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_InvalidLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "garbage")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("prod")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestOpenRedis(t *testing.T) {
	rdb := OpenRedis("localhost:6379")
	if rdb == nil {
		t.Error("expected redis client")
	}
}

func TestOpenNATS_Empty(t *testing.T) {
	nc, err := OpenNATS("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if nc != nil {
		t.Error("expected nil nats for empty URL")
	}
}

func TestWithTenant_Context(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenant(ctx, "tenant-1", "user-1", true)

	if got := TenantFromContext(ctx); got != "tenant-1" {
		t.Errorf("TenantID: got %s", got)
	}
	if got := UserFromContext(ctx); got != "user-1" {
		t.Errorf("UserID: got %s", got)
	}
	if !IsAdminFromContext(ctx) {
		t.Error("IsAdmin should be true")
	}
}

func TestTenantFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	if got := TenantFromContext(ctx); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
	if got := UserFromContext(ctx); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
	if IsAdminFromContext(ctx) {
		t.Error("expected false")
	}
}

func TestTenantToCtx(t *testing.T) {
	ctx := context.Background()
	ctx = TenantToCtx(ctx, "t1", "u1", false)
	if got := TenantFromContext(ctx); got != "t1" {
		t.Errorf("TenantID: got %s", got)
	}
}

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")
	if got := TraceFromContext(ctx); got != "trace-123" {
		t.Errorf("TraceID: got %s", got)
	}
}

func TestIsNotFound(t *testing.T) {
	if IsNotFound(nil) {
		t.Error("nil should not be not-found")
	}
	if !IsNotFound(ErrNotFound) {
		t.Error("ErrNotFound should be not-found")
	}
}
