// Extra tests for landing-service platform helpers.
package platform

import (
	"os"
	"testing"
)

func TestGetenv_DefaultEx(t *testing.T) {
	os.Unsetenv("LANDING_TEST_VAR")
	if got := Getenv("LANDING_TEST_VAR", "default"); got != "default" {
		t.Errorf("got %s", got)
	}
}

func TestGetenv_SetEx(t *testing.T) {
	os.Setenv("LANDING_TEST_VAR", "value")
	defer os.Unsetenv("LANDING_TEST_VAR")
	if got := Getenv("LANDING_TEST_VAR", "default"); got != "value" {
		t.Errorf("got %s", got)
	}
}

func TestGetenvInt_DefaultEx(t *testing.T) {
	os.Unsetenv("LANDING_TEST_INT")
	if got := GetenvInt("LANDING_TEST_INT", 42); got != 42 {
		t.Errorf("got %d", got)
	}
}

func TestGetenvInt_SetEx(t *testing.T) {
	os.Setenv("LANDING_TEST_INT", "100")
	defer os.Unsetenv("LANDING_TEST_INT")
	if got := GetenvInt("LANDING_TEST_INT", 42); got != 100 {
		t.Errorf("got %d", got)
	}
}

func TestGetenvInt_InvalidEx(t *testing.T) {
	os.Setenv("LANDING_TEST_INT", "not-a-number")
	defer os.Unsetenv("LANDING_TEST_INT")
	if got := GetenvInt("LANDING_TEST_INT", 42); got != 42 {
		t.Errorf("got %d", got)
	}
}

func TestInitLogger_DefaultEx(t *testing.T) {
	log := InitLogger("test", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_Debug(t *testing.T) {
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("dev", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_Warn(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_Error(t *testing.T) {
	os.Setenv("LOG_LEVEL", "error")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitTracer_EmptyEndpoint(t *testing.T) {
	tp, err := InitTracer(t.Context(), "", "production")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tp != nil {
		t.Error("expected nil provider for empty endpoint")
	}
}

func TestInitTracer_DevEnv(t *testing.T) {
	tp, err := InitTracer(t.Context(), "http://otel-collector:4317", "development")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tp != nil {
		t.Error("expected nil for dev env")
	}
}

func TestInitTracer_BadURL(t *testing.T) {
	tp, err := InitTracer(t.Context(), ":::bad url:::", "production")
	if err == nil {
		t.Error("expected error for bad URL")
	}
	_ = tp
}

func TestBootstrapSQL_ContainsSchemas(t *testing.T) {
	if !contains(bootstrapSQL, "CREATE SCHEMA IF NOT EXISTS landing") {
		t.Error("missing landing schema")
	}
}

func TestBootstrapSQL_ContainsTables(t *testing.T) {
	tables := []string{
		"landing.landing_pages",
		"landing.form_submissions",
		"landing.tracking_events",
		"landing.form_definitions",
		"landing.conversion_goals",
		"landing.fb_pixel_configs",
		"landing.tracking_clicks",
	}
	for _, table := range tables {
		if !contains(bootstrapSQL, table) {
			t.Errorf("missing %s", table)
		}
	}
}

func TestBootstrapSQL_ContainsIndexes(t *testing.T) {
	indexes := []string{
		"idx_landing_pages_slug",
		"idx_landing_form_submissions_event",
		"idx_landing_form_submissions_idem",
		"idx_landing_tracking_event_id",
		"idx_landing_tracking_events_tenant_created",
	}
	for _, idx := range indexes {
		if !contains(bootstrapSQL, idx) {
			t.Errorf("missing index %s", idx)
		}
	}
}

func TestServiceName(t *testing.T) {
	if serviceName != "landing-service" {
		t.Errorf("got %s", serviceName)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
