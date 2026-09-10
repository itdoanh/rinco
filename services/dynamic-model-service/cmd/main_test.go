// Tests for dynamic-model-service cmd helpers.
package main

import "testing"

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("DYNAMIC_MODEL_HTTP_ADDR", "")
	t.Setenv("DYNAMIC_MODEL_DATABASE_URL", "")
	t.Setenv("DYNAMIC_MODEL_VALKEY_URL", "")
	t.Setenv("ENV", "")

	cfg := loadConfig()
	if cfg.HTTPAddr != ":8084" {
		t.Errorf("HTTPAddr default: got %q, want :8084", cfg.HTTPAddr)
	}
	if cfg.Env != "development" {
		t.Errorf("Env default: got %q, want development", cfg.Env)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL default: got %q, want empty", cfg.DatabaseURL)
	}
	if cfg.ValkeyURL != "localhost:6379" {
		t.Errorf("ValkeyURL default: got %q, want localhost:6379", cfg.ValkeyURL)
	}
}

func TestLoadConfig_Overrides(t *testing.T) {
	t.Setenv("DYNAMIC_MODEL_HTTP_ADDR", ":9999")
	t.Setenv("DYNAMIC_MODEL_DATABASE_URL", "postgres://u:p@h/d")
	t.Setenv("DYNAMIC_MODEL_VALKEY_URL", "redis://valkey:6379")
	t.Setenv("ENV", "production")

	cfg := loadConfig()
	if cfg.HTTPAddr != ":9999" {
		t.Errorf("HTTPAddr override: got %q", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL != "postgres://u:p@h/d" {
		t.Errorf("DatabaseURL override: got %q", cfg.DatabaseURL)
	}
	if cfg.ValkeyURL != "redis://valkey:6379" {
		t.Errorf("ValkeyURL override: got %q", cfg.ValkeyURL)
	}
	if cfg.Env != "production" {
		t.Errorf("Env override: got %q", cfg.Env)
	}
}
