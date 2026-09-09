// Tests for lead-service logger.
package logger

import (
	"os"
	"testing"
)

func TestExtra_Init(t *testing.T) {
	l := Init("test", "test", "v1")
	if l == nil {
		t.Fatal("nil")
	}
}

func TestExtra_Init_DebugLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")
	l := Init("test", "dev", "v1")
	if l == nil {
		t.Fatal("nil")
	}
}

func TestExtra_Init_WarnLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	defer os.Unsetenv("LOG_LEVEL")
	l := Init("test", "prod", "v1")
	if l == nil {
		t.Fatal("nil")
	}
}

func TestExtra_Init_ErrorLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "error")
	defer os.Unsetenv("LOG_LEVEL")
	l := Init("test", "prod", "v1")
	if l == nil {
		t.Fatal("nil")
	}
}

func TestExtra_Init_InvalidLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "garbage")
	defer os.Unsetenv("LOG_LEVEL")
	l := Init("test", "prod", "v1")
	if l == nil {
		t.Fatal("nil")
	}
}
