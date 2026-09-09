// Tests for crm-service logger.
package logger

import (
	"context"
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	log := Init("test", "test", "v1")
	if log == nil {
		t.Fatal("Init returned nil")
	}
}

func TestInit_DebugLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")
	log := Init("test", "dev", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInit_WarnLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	defer os.Unsetenv("LOG_LEVEL")
	log := Init("test", "prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInit_ErrorLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "error")
	defer os.Unsetenv("LOG_LEVEL")
	log := Init("test", "prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestInit_InvalidLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "garbage")
	defer os.Unsetenv("LOG_LEVEL")
	log := Init("test", "prod", "v1")
	if log == nil {
		t.Fatal("nil logger")
	}
}

func TestWithTraceID(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-123")
	got := FromContext(ctx)
	if got != "trace-123" {
		t.Errorf("FromContext: got %s", got)
	}
}

func TestFromContext_Empty(t *testing.T) {
	if got := FromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestFromContext_WrongType(t *testing.T) {
	// Wrong type stored in context - should return empty
	ctx := context.WithValue(context.Background(), traceIDKey, 123)
	if got := FromContext(ctx); got != "" {
		t.Errorf("expected empty for non-string, got %s", got)
	}
}

func TestLogError_NoTrace(t *testing.T) {
	// Should not panic
	LogError(context.Background(), "test error", errTest("fake error"))
}

func TestLogError_WithTrace(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-456")
	LogError(ctx, "test error", errTest("another error"))
}

func TestLogInfo_NoArgs(t *testing.T) {
	LogInfo(context.Background(), "test info")
}

func TestLogInfo_WithArgs(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-789")
	LogInfo(ctx, "test", "key", "value")
}

func TestLogDebug(t *testing.T) {
	LogDebug(context.Background(), "test debug")
}

func errTest(s string) error {
	return &testErr{s}
}

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
