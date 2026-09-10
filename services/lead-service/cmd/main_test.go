// Tests for lead-service cmd helpers (loadConfig, migrationsList, initTracer).
package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("LEAD_HTTP_ADDR", "")
	t.Setenv("LEAD_DATABASE_URL", "")
	t.Setenv("LEAD_VALKEY_URL", "")
	t.Setenv("LEAD_VALKEY_PASSWORD", "")
	t.Setenv("LEAD_VALKEY_DB", "")
	t.Setenv("LEAD_NATS_URL", "")
	t.Setenv("ENV", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("LEAD_SCORING_URL", "")
	t.Setenv("LEAD_CAPI_WORKER_URL", "")

	cfg := loadConfig()
	if cfg.HTTPAddr != ":8083" {
		t.Errorf("HTTPAddr default: got %q, want :8083", cfg.HTTPAddr)
	}
	if cfg.Env != "development" {
		t.Errorf("Env default: got %q, want development", cfg.Env)
	}
	if cfg.ValkeyAddr != "localhost:6379" {
		t.Errorf("ValkeyAddr default: got %q, want localhost:6379", cfg.ValkeyAddr)
	}
	if cfg.ValkeyPwd != "rinco_dev_password" {
		t.Errorf("ValkeyPwd default: got %q", cfg.ValkeyPwd)
	}
	if cfg.NATSURL != "" {
		t.Errorf("NATSURL default: got %q, want empty", cfg.NATSURL)
	}
}

func TestLoadConfig_Overrides(t *testing.T) {
	t.Setenv("LEAD_HTTP_ADDR", ":9999")
	t.Setenv("LEAD_DATABASE_URL", "postgres://u:p@h/d")
	t.Setenv("LEAD_VALKEY_URL", "redis://x:1")
	t.Setenv("LEAD_VALKEY_PASSWORD", "secret")
	t.Setenv("LEAD_VALKEY_DB", "5")
	t.Setenv("LEAD_NATS_URL", "nats://nats:4222")
	t.Setenv("ENV", "production")
	t.Setenv("LEAD_SCORING_URL", "http://scoring:8092")
	t.Setenv("LEAD_CAPI_WORKER_URL", "http://capi:8091")

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
	if cfg.ValkeyPwd != "secret" {
		t.Errorf("ValkeyPwd override: got %q", cfg.ValkeyPwd)
	}
	if cfg.ValkeyDB != 5 {
		t.Errorf("ValkeyDB override: got %d, want 5", cfg.ValkeyDB)
	}
	if cfg.NATSURL != "nats://nats:4222" {
		t.Errorf("NATSURL override: got %q", cfg.NATSURL)
	}
	if cfg.Env != "production" {
		t.Errorf("Env override: got %q", cfg.Env)
	}
	if cfg.ScoringURL != "http://scoring:8092" {
		t.Errorf("ScoringURL override: got %q", cfg.ScoringURL)
	}
	if cfg.CAPIURL != "http://capi:8091" {
		t.Errorf("CAPIURL override: got %q", cfg.CAPIURL)
	}
}

func TestMigrationsList_CountAndOrder(t *testing.T) {
	list := migrationsList()
	if len(list) != 4 {
		t.Errorf("migrations count: got %d, want 4", len(list))
	}
	wantOrder := []string{
		"0001_lead_sources", "0002_pipelines", "0003_leads", "0004_lead_notes_activities",
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
		"0001_lead_sources":       {"lead_sources", "tenant_id"},
		"0002_pipelines":          {"pipelines", "tenant_id"},
		"0003_leads":              {"leads", "tenant_id"},
		"0004_lead_notes_activities": {"lead_notes", "lead_activities", "tenant_id"},
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

// Verify that initTracer in development never tries to dial an endpoint.
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
