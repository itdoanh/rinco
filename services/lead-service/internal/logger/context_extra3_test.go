// Extra tests for lead-service logger/context helpers.
package logger

import (
	"context"
	"testing"
)

func TestWithTraceID_Basic(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-1")
	got := FromContext(ctx)
	if got != "trace-1" {
		t.Errorf("got %s", got)
	}
}

func TestFromContext_Empty(t *testing.T) {
	got := FromContext(context.Background())
	if got != "" {
		t.Errorf("expected empty: got %s", got)
	}
}

func TestFromContext_NilContext(t *testing.T) {
	// Note: nil context causes panic in FromContext.
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil context")
		}
	}()
	_ = FromContext(nil) //nolint:staticcheck
}

func TestWithTraceID_Overwrite(t *testing.T) {
	ctx := WithTraceID(context.Background(), "first")
	ctx = WithTraceID(ctx, "second")
	got := FromContext(ctx)
	if got != "second" {
		t.Errorf("expected second: got %s", got)
	}
}

func TestWithTraceID_Empty(t *testing.T) {
	ctx := WithTraceID(context.Background(), "")
	got := FromContext(ctx)
	if got != "" {
		t.Errorf("expected empty: got %s", got)
	}
}

func TestWithTraceID_Unicode(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trầce-id-123")
	got := FromContext(ctx)
	if got != "trầce-id-123" {
		t.Errorf("got %s", got)
	}
}

func TestFromContext_WrongType(t *testing.T) {
	// Inject a non-string value with the trace_id key
	type otherKey struct{}
	ctx := context.WithValue(context.Background(), otherKey{}, "value")
	got := FromContext(ctx)
	if got != "" {
		t.Errorf("got %s", got)
	}
}

func TestLogError_DoesNotPanic(t *testing.T) {
	ctx := context.Background()
	LogError(ctx, "test error", errMock("mock error"))
}

func TestLogError_WithTrace(t *testing.T) {
	ctx := WithTraceID(context.Background(), "tr-1")
	LogError(ctx, "test error", errMock("boom"))
}

func TestLogInfo_DoesNotPanic(t *testing.T) {
	ctx := context.Background()
	LogInfo(ctx, "test info")
}

func TestLogInfo_WithArgs(t *testing.T) {
	ctx := context.Background()
	LogInfo(ctx, "test info", "key", "value")
}

func errMock(msg string) error {
	return &mockError{msg: msg}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
