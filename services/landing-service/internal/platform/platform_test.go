// Tests for landing-service platform.
package platform

import (
	"os"
	"testing"
)

func TestGetenv_Default(t *testing.T) {
	os.Unsetenv("LANDING_TEST_VAR")
	if got := Getenv("LANDING_TEST_VAR", "default"); got != "default" {
		t.Errorf("got %s, want default", got)
	}
}

func TestGetenv_Set(t *testing.T) {
	os.Setenv("LANDING_TEST_VAR", "value")
	defer os.Unsetenv("LANDING_TEST_VAR")
	if got := Getenv("LANDING_TEST_VAR", "default"); got != "value" {
		t.Errorf("got %s, want value", got)
	}
}

func TestGetenv_EmptyFallback(t *testing.T) {
	os.Setenv("LANDING_TEST_VAR", "")
	defer os.Unsetenv("LANDING_TEST_VAR")
	if got := Getenv("LANDING_TEST_VAR", "default"); got != "default" {
		t.Errorf("got %s, want default", got)
	}
}

func TestGetenvInt_Default(t *testing.T) {
	os.Unsetenv("LANDING_TEST_INT")
	if got := GetenvInt("LANDING_TEST_INT", 42); got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestGetenvInt_Set(t *testing.T) {
	os.Setenv("LANDING_TEST_INT", "100")
	defer os.Unsetenv("LANDING_TEST_INT")
	if got := GetenvInt("LANDING_TEST_INT", 42); got != 100 {
		t.Errorf("got %d, want 100", got)
	}
}

func TestGetenvInt_Invalid(t *testing.T) {
	os.Setenv("LANDING_TEST_INT", "not-a-number")
	defer os.Unsetenv("LANDING_TEST_INT")
	if got := GetenvInt("LANDING_TEST_INT", 42); got != 42 {
		t.Errorf("got %d, want default 42", got)
	}
}

func TestInitLogger_Default(t *testing.T) {
	log := InitLogger("test", "v1")
	if log == nil {
		t.Fatal("InitLogger returned nil")
	}
}

func TestInitLogger_DebugLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("dev", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_WarnLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_ErrorLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "error")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInitLogger_InvalidLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "garbage")
	defer os.Unsetenv("LOG_LEVEL")
	log := InitLogger("prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}
