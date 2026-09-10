// Tests for crm-service cmd helpers (loadConfig, migrationsList, initTracer).
package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear all known CRM_ env vars to ensure defaults take effect.
	t.Setenv("CRM_HTTP_ADDR", "")
	t.Setenv("CRM_DATABASE_URL", "")
	t.Setenv("CRM_VALKEY_URL", "")
	t.Setenv("CRM_VALKEY_PASSWORD", "")
	t.Setenv("CRM_VALKEY_DB", "")
	t.Setenv("ENV", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	cfg := loadConfig()
	if cfg.HTTPAddr != ":8082" {
		t.Errorf("HTTPAddr default: got %q, want :8082", cfg.HTTPAddr)
	}
	if cfg.Env != "development" {
		t.Errorf("Env default: got %q, want development", cfg.Env)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL default: got %q, want empty", cfg.DatabaseURL)
	}
	if cfg.ValkeyAddr != "localhost:6379" {
		t.Errorf("ValkeyAddr default: got %q, want localhost:6379", cfg.ValkeyAddr)
	}
	if cfg.ValkeyPassword != "rinco_dev_password" {
		t.Errorf("ValkeyPassword default: got %q, want rinco_dev_password", cfg.ValkeyPassword)
	}
}

func TestLoadConfig_Overrides(t *testing.T) {
	t.Setenv("CRM_HTTP_ADDR", ":9999")
	t.Setenv("CRM_DATABASE_URL", "postgres://u:p@h/d")
	t.Setenv("CRM_VALKEY_URL", "redis://x:1")
	t.Setenv("CRM_VALKEY_PASSWORD", "secret")
	t.Setenv("CRM_VALKEY_DB", "3")
	t.Setenv("ENV", "staging")

	cfg := loadConfig()
	if cfg.HTTPAddr != ":9999" {
		t.Errorf("HTTPAddr override: got %q", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL != "postgres://u:p@h/d" {
		t.Errorf("DatabaseURL override: got %q", cfg.DatabaseURL)
	}
	if cfg.ValkeyAddr != "redis://x:1" {
		t.Errorf("ValkeyAddr override: got %q", cfg.ValkeyAddr)
	}
	if cfg.ValkeyPassword != "secret" {
		t.Errorf("ValkeyPassword override: got %q", cfg.ValkeyPassword)
	}
	if cfg.ValkeyDB != 3 {
		t.Errorf("ValkeyDB override: got %d, want 3", cfg.ValkeyDB)
	}
	if cfg.Env != "staging" {
		t.Errorf("Env override: got %q", cfg.Env)
	}
}

func TestMigrationsList_CountAndOrder(t *testing.T) {
	list := migrationsList()
	if len(list) != 10 {
		t.Errorf("migrations count: got %d, want 10", len(list))
	}
	wantOrder := []string{
		"0001_companies", "0002_contacts", "0003_deals", "0004_activities",
		"0005_notes", "0006_tags", "0007_custom_fields", "0008_users_tree",
		"0009_invite_links", "0010_rls",
	}
	for i, m := range list {
		if m.name != wantOrder[i] {
			t.Errorf("migration[%d].name: got %q, want %q", i, m.name, wantOrder[i])
		}
		if strings.TrimSpace(m.body) == "" {
			t.Errorf("migration[%d].body is empty", i)
		}
	}
}

func TestMigrationsList_BodiesContainExpectedSchema(t *testing.T) {
	list := migrationsList()
	byName := map[string]string{}
	for _, m := range list {
		byName[m.name] = m.body
	}
	expectations := map[string][]string{
		"0001_companies":      {"CREATE TABLE IF NOT EXISTS companies", "ltree", "tenant_id"},
		"0002_contacts":       {"CREATE TABLE IF NOT EXISTS contacts", "tenant_id"},
		"0003_deals":          {"CREATE TABLE IF NOT EXISTS deals", "tenant_id"},
		"0004_activities":     {"CREATE TABLE IF NOT EXISTS activities", "tenant_id"},
		"0005_notes":          {"CREATE TABLE IF NOT EXISTS notes", "tenant_id"},
		"0006_tags":           {"CREATE TABLE IF NOT EXISTS tags", "tenant_id"},
		"0007_custom_fields":  {"CREATE TABLE IF NOT EXISTS custom_fields", "tenant_id"},
		"0008_users_tree":     {"CREATE TABLE IF NOT EXISTS users", "ltree"},
		"0009_invite_links":   {"CREATE TABLE IF NOT EXISTS invite_links", "tenant_id"},
		"0010_rls":            {"RLS", "tenant_id"},
	}
	for name, snippets := range expectations {
		body, ok := byName[name]
		if !ok {
			t.Errorf("migration %q not found", name)
			continue
		}
		for _, snippet := range snippets {
			if !strings.Contains(strings.ToLower(body), strings.ToLower(snippet)) {
				t.Errorf("migration %q missing %q", name, snippet)
			}
		}
	}
}

func TestInitTracer_NoEndpoint(t *testing.T) {
	tp, shutdown, err := initTracer(context.Background(), "", "development")
	if err != nil {
		t.Fatalf("initTracer no endpoint: %v", err)
	}
	if tp != nil {
		t.Errorf("expected nil TracerProvider when endpoint is empty; got %T", tp)
	}
	if shutdown == nil {
		t.Error("shutdown func should not be nil even on no-op path")
	}
	// Calling the no-op shutdown should not panic.
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("no-op shutdown should return nil; got %v", err)
	}
}

func TestInitTracer_DevelopmentShortCircuit(t *testing.T) {
	tp, shutdown, err := initTracer(context.Background(), "http://otel:4317", "development")
	if err != nil {
		t.Fatalf("initTracer dev: %v", err)
	}
	if tp != nil {
		t.Errorf("expected nil TracerProvider in development env; got %T", tp)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown: %v", err)
	}
}

var errFake = &fakeErr{msg: "boom"}

type fakeErr struct{ msg string }

func (e *fakeErr) Error() string { return e.msg }

// Verify that initTracer in development never tries to dial an endpoint by
// checking the call returns within a fraction of a second.
func TestInitTracer_DevelopmentFast(t *testing.T) {
	done := make(chan struct{})
	go func() {
		_, _, _ = initTracer(context.Background(), "http://invalid.local:4317", "development")
		close(done)
	}()
	select {
	case <-done:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("initTracer in development should be near-instant; it took >2s")
	}
}
