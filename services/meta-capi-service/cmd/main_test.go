// Tests for meta-capi-service cmd helpers.
package main

import (
	"os"
	"testing"
)

func TestEnvOr_Unset(t *testing.T) {
	os.Unsetenv("RINCO_TEST_UNSET")
	if got := envOr("RINCO_TEST_UNSET", "fallback"); got != "fallback" {
		t.Errorf("unset: want fallback, got %s", got)
	}
}

func TestEnvOr_Set(t *testing.T) {
	os.Setenv("RINCO_TEST_SET", "real")
	defer os.Unsetenv("RINCO_TEST_SET")
	if got := envOr("RINCO_TEST_SET", "fallback"); got != "real" {
		t.Errorf("set: want real, got %s", got)
	}
}

func TestEnvOr_Empty(t *testing.T) {
	os.Setenv("RINCO_TEST_EMPTY", "")
	defer os.Unsetenv("RINCO_TEST_EMPTY")
	if got := envOr("RINCO_TEST_EMPTY", "fallback"); got != "fallback" {
		t.Errorf("empty: want fallback, got %s", got)
	}
}
