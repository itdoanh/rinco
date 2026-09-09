// Extra tests for landing-service cmd/main.go loadConfig helpers.
package main

import (
	"os"
	"testing"

	"github.com/rinco/services/landing-service/internal/platform"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear env vars that loadConfig reads
	for _, k := range []string{
		"ENV", "LANDING_HTTP_ADDR", "LANDING_MONGO_URL",
		"LANDING_S3_BUCKET_HOT", "LANDING_S3_BUCKET_COLD",
		"LANDING_S3_SSL", "LANDING_VALKEY_URL",
		"LANDING_TRACKING_SALT",
	} {
		os.Unsetenv(k)
	}

	cfg := loadConfig()
	if cfg.Env != "development" {
		t.Errorf("Env: got %q, want development", cfg.Env)
	}
	if cfg.HTTPAddr != ":8086" {
		t.Errorf("HTTPAddr: got %q, want :8086", cfg.HTTPAddr)
	}
	if cfg.HotBucket != "rinco-hot-ssd" {
		t.Errorf("HotBucket: got %q", cfg.HotBucket)
	}
	if cfg.ColdBucket != "rinco-cold-hdd" {
		t.Errorf("ColdBucket: got %q", cfg.ColdBucket)
	}
	if cfg.ValkeyURL != "localhost:6379" {
		t.Errorf("ValkeyURL: got %q", cfg.ValkeyURL)
	}
	if cfg.TrackingSalt != "rinco-tracking-salt" {
		t.Errorf("TrackingSalt: got %q", cfg.TrackingSalt)
	}
}

func TestLoadConfig_Overrides(t *testing.T) {
	os.Setenv("LANDING_HTTP_ADDR", ":9999")
	os.Setenv("LANDING_S3_BUCKET_HOT", "custom-hot")
	os.Setenv("LANDING_S3_BUCKET_COLD", "custom-cold")
	os.Setenv("LANDING_TRACKING_SALT", "custom-salt")
	defer func() {
		os.Unsetenv("LANDING_HTTP_ADDR")
		os.Unsetenv("LANDING_S3_BUCKET_HOT")
		os.Unsetenv("LANDING_S3_BUCKET_COLD")
		os.Unsetenv("LANDING_TRACKING_SALT")
	}()

	cfg := loadConfig()
	if cfg.HTTPAddr != ":9999" {
		t.Errorf("HTTPAddr: got %q", cfg.HTTPAddr)
	}
	if cfg.HotBucket != "custom-hot" {
		t.Errorf("HotBucket: got %q", cfg.HotBucket)
	}
	if cfg.ColdBucket != "custom-cold" {
		t.Errorf("ColdBucket: got %q", cfg.ColdBucket)
	}
	if cfg.TrackingSalt != "custom-salt" {
		t.Errorf("TrackingSalt: got %q", cfg.TrackingSalt)
	}
}

func TestServiceConstants(t *testing.T) {
	if serviceName != "landing-service" {
		t.Errorf("serviceName: got %q", serviceName)
	}
	if version == "" {
		t.Error("version should be non-empty")
	}
}

// Validate that platform.Getenv respects empty as fallback.
func TestPlatformGetenvFallback(t *testing.T) {
	os.Unsetenv("LANDING_TEST_VAR")
	got := platform.Getenv("LANDING_TEST_VAR", "default-val")
	if got != "default-val" {
		t.Errorf("got %q", got)
	}
}

func TestPlatformGetenvValue(t *testing.T) {
	os.Setenv("LANDING_TEST_VAR", "actual-val")
	defer os.Unsetenv("LANDING_TEST_VAR")
	got := platform.Getenv("LANDING_TEST_VAR", "default-val")
	if got != "actual-val" {
		t.Errorf("got %q", got)
	}
}

func TestPlatformGetenvEmptyIsFallback(t *testing.T) {
	os.Setenv("LANDING_TEST_VAR", "")
	defer os.Unsetenv("LANDING_TEST_VAR")
	got := platform.Getenv("LANDING_TEST_VAR", "default-val")
	if got != "default-val" {
		t.Errorf("got %q, want fallback for empty", got)
	}
}
