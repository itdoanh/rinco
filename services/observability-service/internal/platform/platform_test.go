// Tests for observability-service platform helpers.
package platform

import (
	"testing"
)

func TestGetenv_Set(t *testing.T) {
	t.Setenv("TEST_KEY", "value")
	got := Getenv("TEST_KEY", "fallback")
	if got != "value" {
		t.Errorf("expected 'value', got %q", got)
	}
}

func TestGetenv_NotSet(t *testing.T) {
	t.Setenv("TEST_KEY", "")
	got := Getenv("TEST_KEY", "fallback")
	if got != "fallback" {
		t.Errorf("expected 'fallback', got %q", got)
	}
}

func TestGetenvInt_Valid(t *testing.T) {
	t.Setenv("TEST_INT_KEY", "42")
	got := GetenvInt("TEST_INT_KEY", 0)
	if got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestGetenvInt_Invalid(t *testing.T) {
	t.Setenv("TEST_INT_KEY", "not-a-number")
	got := GetenvInt("TEST_INT_KEY", 10)
	if got != 10 {
		t.Errorf("expected fallback 10, got %d", got)
	}
}

func TestGetenvInt_Empty(t *testing.T) {
	t.Setenv("TEST_INT_KEY", "")
	got := GetenvInt("TEST_INT_KEY", 99)
	if got != 99 {
		t.Errorf("expected fallback 99, got %d", got)
	}
}

func TestGetenvBool(t *testing.T) {
	tests := []struct {
		value    string
		fallback bool
		want     bool
	}{
		{"", true, true},
		{"", false, false},
		{"true", false, true},
		{"TRUE", false, true},
		{"True", false, true},
		{"false", true, false},
		{"FALSE", true, false},
		{"1", false, true},
		{"yes", false, true},
		{"YES", false, true},
		{"0", true, false},
		{"no", true, false},
		{"invalid", true, false},
		{"invalid", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.value+"_fb_"+boolStr(tt.fallback), func(t *testing.T) {
			t.Setenv("TEST_BOOL_KEY", tt.value)
			got := GetenvBool("TEST_BOOL_KEY", tt.fallback)
			if got != tt.want {
				t.Errorf("GetenvBool(%q, %v) = %v, want %v", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
