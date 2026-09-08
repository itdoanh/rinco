// Tests for lead-service platform.
package platform

import (
	"context"
	"os"
	"testing"
)

func TestGetenv_Default(t *testing.T) {
	os.Unsetenv("LEAD_TEST_VAR")
	if got := Getenv("LEAD_TEST_VAR", "default"); got != "default" {
		t.Errorf("got %s, want default", got)
	}
}

func TestGetenv_Set(t *testing.T) {
	os.Setenv("LEAD_TEST_VAR", "value")
	defer os.Unsetenv("LEAD_TEST_VAR")
	if got := Getenv("LEAD_TEST_VAR", "default"); got != "value" {
		t.Errorf("got %s, want value", got)
	}
}

func TestGetenv_EmptyFallback(t *testing.T) {
	os.Setenv("LEAD_TEST_VAR", "")
	defer os.Unsetenv("LEAD_TEST_VAR")
	if got := Getenv("LEAD_TEST_VAR", "default"); got != "default" {
		t.Errorf("got %s, want default", got)
	}
}

func TestGetenvInt_Default(t *testing.T) {
	os.Unsetenv("LEAD_TEST_INT")
	if got := GetenvInt("LEAD_TEST_INT", 42); got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestGetenvInt_Set(t *testing.T) {
	os.Setenv("LEAD_TEST_INT", "100")
	defer os.Unsetenv("LEAD_TEST_INT")
	if got := GetenvInt("LEAD_TEST_INT", 42); got != 100 {
		t.Errorf("got %d, want 100", got)
	}
}

func TestGetenvInt_Invalid(t *testing.T) {
	os.Setenv("LEAD_TEST_INT", "not-a-number")
	defer os.Unsetenv("LEAD_TEST_INT")
	if got := GetenvInt("LEAD_TEST_INT", 42); got != 42 {
		t.Errorf("got %d, want default 42", got)
	}
}

func TestGetenvBool_Default(t *testing.T) {
	os.Unsetenv("LEAD_TEST_BOOL")
	if got := GetenvBool("LEAD_TEST_BOOL", true); !got {
		t.Error("got false, want true (default)")
	}
}

func TestGetenvBool_True(t *testing.T) {
	os.Setenv("LEAD_TEST_BOOL", "true")
	defer os.Unsetenv("LEAD_TEST_BOOL")
	if !GetenvBool("LEAD_TEST_BOOL", false) {
		t.Error("expected true")
	}
}

func TestGetenvBool_One(t *testing.T) {
	os.Setenv("LEAD_TEST_BOOL", "1")
	defer os.Unsetenv("LEAD_TEST_BOOL")
	if !GetenvBool("LEAD_TEST_BOOL", false) {
		t.Error("expected true for '1'")
	}
}

func TestGetenvBool_Yes(t *testing.T) {
	os.Setenv("LEAD_TEST_BOOL", "yes")
	defer os.Unsetenv("LEAD_TEST_BOOL")
	if !GetenvBool("LEAD_TEST_BOOL", false) {
		t.Error("expected true for 'yes'")
	}
}

func TestGetenvBool_False(t *testing.T) {
	os.Setenv("LEAD_TEST_BOOL", "false")
	defer os.Unsetenv("LEAD_TEST_BOOL")
	if GetenvBool("LEAD_TEST_BOOL", true) {
		t.Error("expected false")
	}
}

func TestGetenvBool_Bogus(t *testing.T) {
	os.Setenv("LEAD_TEST_BOOL", "bogus")
	defer os.Unsetenv("LEAD_TEST_BOOL")
	if GetenvBool("LEAD_TEST_BOOL", true) {
		t.Error("expected default for invalid value")
	}
}

func TestDBConfigFromEnv_Default(t *testing.T) {
	os.Unsetenv("LEAD_TEST_DATABASE_URL")
	cfg := DBConfigFromEnv("LEAD_TEST_")
	if cfg.Host == "" {
		t.Error("Host should have a default")
	}
}

func TestDBConfigFromEnv_WithURL(t *testing.T) {
	os.Setenv("LEAD_TEST_DATABASE_URL", "postgres://user:pass@example.com:5433/mydb?sslmode=require")
	defer os.Unsetenv("LEAD_TEST_DATABASE_URL")
	cfg := DBConfigFromEnv("LEAD_TEST_")
	if cfg.Host != "example.com" {
		t.Errorf("Host: got %s", cfg.Host)
	}
	if cfg.Port != 5433 {
		t.Errorf("Port: got %d", cfg.Port)
	}
	if cfg.User != "user" {
		t.Errorf("User: got %s", cfg.User)
	}
	if cfg.Password != "pass" {
		t.Errorf("Password: got %s", cfg.Password)
	}
	if cfg.Database != "mydb" {
		t.Errorf("Database: got %s", cfg.Database)
	}
	if cfg.SSLMode != "require" {
		t.Errorf("SSLMode: got %s", cfg.SSLMode)
	}
}

func TestDBConfigFromEnv_BadURL(t *testing.T) {
	os.Setenv("LEAD_TEST_DATABASE_URL", "://not-a-valid-url")
	defer os.Unsetenv("LEAD_TEST_DATABASE_URL")
	cfg := DBConfigFromEnv("LEAD_TEST_")
	if cfg.Host != "localhost" {
		t.Errorf("expected fallback to localhost, got %s", cfg.Host)
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
	if got := IsAdminFromContext(ctx); !got {
		t.Error("IsAdmin should be true")
	}
	if got := TraceFromContext(ctx); got != "" {
		t.Errorf("TraceID: got %s", got)
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
	if got := IsAdminFromContext(ctx); got {
		t.Error("expected false")
	}
}

func TestInitLogger_Default(t *testing.T) {
	log := InitLogger("test-svc", "test", "v1.0.0")
	if log == nil {
		t.Fatal("InitLogger returned nil")
	}
}

func TestInitLogger_DebugLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("test", "dev", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_WarnLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("test", "prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_ErrorLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "error")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("test", "prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_InvalidLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "garbage")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("test", "prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}
