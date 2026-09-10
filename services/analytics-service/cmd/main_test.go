// Tests for analytics-service cmd helpers.
package main

import "testing"

func TestEnvOr_DefaultWhenEmpty(t *testing.T) {
	t.Setenv("ANALYTICS_TEST_VAR", "")
	if got := envOr("ANALYTICS_TEST_VAR", "default"); got != "default" {
		t.Errorf("envOr empty: got %q, want default", got)
	}
}

func TestEnvOr_ReturnsValue(t *testing.T) {
	t.Setenv("ANALYTICS_TEST_VAR", "actual")
	if got := envOr("ANALYTICS_TEST_VAR", "default"); got != "actual" {
		t.Errorf("envOr: got %q, want actual", got)
	}
}

func TestEnvOr_UnsetReturnsDefault(t *testing.T) {
	t.Setenv("ANALYTICS_TEST_VAR_UNSET", "")
	// Even if the var was previously set, clearing should make it return default.
	if got := envOr("ANALYTICS_NEVER_SET_XYZ_42", "fallback"); got != "fallback" {
		t.Errorf("envOr unset: got %q, want fallback", got)
	}
}
