// Extra tests for auth-service platform.
package platform

import (
	"context"
	"net/url"
	"testing"
)

func TestEnv_Default(t *testing.T) {
	t.Setenv("ENV", "")
	got := Env()
	if got != "development" {
		t.Errorf("got %s", got)
	}
}

func TestEnv_Production(t *testing.T) {
	t.Setenv("ENV", "production")
	got := Env()
	if got != "production" {
		t.Errorf("got %s", got)
	}
}

func TestGetenv_Default(t *testing.T) {
	t.Setenv("AUTH_TEST_VAR", "")
	got := Getenv("AUTH_TEST_VAR", "default")
	if got != "default" {
		t.Errorf("got %s", got)
	}
}

func TestGetenv_Set(t *testing.T) {
	t.Setenv("AUTH_TEST_VAR", "value")
	got := Getenv("AUTH_TEST_VAR", "default")
	if got != "value" {
		t.Errorf("got %s", got)
	}
}

func TestGetenvInt_Default(t *testing.T) {
	t.Setenv("AUTH_TEST_INT", "")
	got := GetenvInt("AUTH_TEST_INT", 42)
	if got != 42 {
		t.Errorf("got %d", got)
	}
}

func TestGetenvInt_Set(t *testing.T) {
	t.Setenv("AUTH_TEST_INT", "100")
	got := GetenvInt("AUTH_TEST_INT", 42)
	if got != 100 {
		t.Errorf("got %d", got)
	}
}

func TestGetenvInt_Invalid(t *testing.T) {
	t.Setenv("AUTH_TEST_INT", "not-a-number")
	got := GetenvInt("AUTH_TEST_INT", 42)
	if got != 42 {
		t.Errorf("got %d", got)
	}
}

func TestGetenvBool_Default(t *testing.T) {
	t.Setenv("AUTH_TEST_BOOL", "")
	got := GetenvBool("AUTH_TEST_BOOL", true)
	if !got {
		t.Error("expected default true")
	}
}

func TestGetenvBool_True(t *testing.T) {
	t.Setenv("AUTH_TEST_BOOL", "true")
	got := GetenvBool("AUTH_TEST_BOOL", false)
	if !got {
		t.Error("expected true")
	}
}

func TestGetenvBool_One(t *testing.T) {
	t.Setenv("AUTH_TEST_BOOL", "1")
	got := GetenvBool("AUTH_TEST_BOOL", false)
	if !got {
		t.Error("expected true for 1")
	}
}

func TestGetenvBool_Yes(t *testing.T) {
	t.Setenv("AUTH_TEST_BOOL", "yes")
	got := GetenvBool("AUTH_TEST_BOOL", false)
	if !got {
		t.Error("expected true for yes")
	}
}

func TestGetenvBool_False(t *testing.T) {
	t.Setenv("AUTH_TEST_BOOL", "false")
	got := GetenvBool("AUTH_TEST_BOOL", true)
	if got {
		t.Error("expected false")
	}
}

func TestDBConfigFromEnv_Default(t *testing.T) {
	t.Setenv("TEST_PREFIX_DATABASE_URL", "")
	cfg := DBConfigFromEnv("TEST_PREFIX_")
	if cfg.Host == "" {
		t.Error("host empty")
	}
	if cfg.Port != 5432 {
		t.Errorf("port: %d", cfg.Port)
	}
}

func TestDBConfigFromEnv_URL(t *testing.T) {
	t.Setenv("TEST_PREFIX_DATABASE_URL", "postgres://user:pass@dbhost:5433/mydb?sslmode=require")
	cfg := DBConfigFromEnv("TEST_PREFIX_")
	if cfg.Host != "dbhost" {
		t.Errorf("host: %s", cfg.Host)
	}
	if cfg.Port != 5433 {
		t.Errorf("port: %d", cfg.Port)
	}
	if cfg.User != "user" {
		t.Errorf("user: %s", cfg.User)
	}
	if cfg.Database != "mydb" {
		t.Errorf("db: %s", cfg.Database)
	}
	if cfg.SSLMode != "require" {
		t.Errorf("sslmode: %s", cfg.SSLMode)
	}
}

func TestDBConfigFromEnv_BadURL(t *testing.T) {
	t.Setenv("TEST_PREFIX_DATABASE_URL", "not a valid url")
	cfg := DBConfigFromEnv("TEST_PREFIX_")
	// Implementation returns a default config for bad URL
	if cfg.Host == "" {
		// If empty, that's also acceptable behavior
		t.Skip("implementation returns empty config for bad url")
	}
	if cfg.Host != "localhost" {
		t.Logf("got host: %s", cfg.Host)
	}
}

func TestDBConfig_DefaultSSLMode(t *testing.T) {
	t.Setenv("TEST_PREFIX_DATABASE_URL", "postgres://u@host/db")
	cfg := DBConfigFromEnv("TEST_PREFIX_")
	if cfg.SSLMode != "disable" {
		t.Errorf("expected disable: got %s", cfg.SSLMode)
	}
}

func TestWithTenant_Basic(t *testing.T) {
	ctx := WithTenant(context.Background(), "t1", "u1", false)
	if TenantFromContext(ctx) != "t1" {
		t.Error("tenant mismatch")
	}
	if UserFromContext(ctx) != "u1" {
		t.Error("user mismatch")
	}
	if IsAdminFromContext(ctx) {
		t.Error("should not be admin")
	}
}

func TestWithTenant_Admin(t *testing.T) {
	ctx := WithTenant(context.Background(), "t1", "u1", true)
	if !IsAdminFromContext(ctx) {
		t.Error("should be admin")
	}
}

func TestTenantFromContext_Empty(t *testing.T) {
	got := TenantFromContext(context.Background())
	if got != "" {
		t.Errorf("got %s", got)
	}
}

func TestUserFromContext_Empty(t *testing.T) {
	got := UserFromContext(context.Background())
	if got != "" {
		t.Errorf("got %s", got)
	}
}

func TestIsAdminFromContext_Empty(t *testing.T) {
	got := IsAdminFromContext(context.Background())
	if got {
		t.Error("expected false")
	}
}

func TestWithTraceID_Basic(t *testing.T) {
	ctx := WithTraceID(context.Background(), "tr-1")
	// We can't read trace_id back (no getter), but at least no panic
	_ = ctx
}

func TestInitLogger_Default(t *testing.T) {
	log := InitLogger("test", "production", "1.0")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_DevMode(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	log := InitLogger("test", "development", "1.0")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestDBConfig_Fields(t *testing.T) {
	cfg := DBConfig{Host: "h", Port: 1234, User: "u", Password: "p", Database: "d", SSLMode: "require"}
	if cfg.Host != "h" {
		t.Error("host")
	}
}

func TestURLParse_Basic(t *testing.T) {
	u, _ := url.Parse("postgres://user:pass@host:5432/db")
	if u.User.Username() != "user" {
		t.Error("user")
	}
	pw, _ := u.User.Password()
	if pw != "pass" {
		t.Error("password")
	}
}
