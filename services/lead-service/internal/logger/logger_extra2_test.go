// Extra tests for lead-service logger/context helpers.
package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

// =============================================================================
// WithTraceID / FromContext
// =============================================================================

func TestExtra_WithTraceID_StoresValue(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-123")
	if got := FromContext(ctx); got != "trace-123" {
		t.Errorf("expected trace-123, got %q", got)
	}
}

func TestExtra_WithTraceID_EmptyString(t *testing.T) {
	ctx := WithTraceID(context.Background(), "")
	if got := FromContext(ctx); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtra_WithTraceID_Overwrite(t *testing.T) {
	ctx := WithTraceID(context.Background(), "first")
	ctx = WithTraceID(ctx, "second")
	if got := FromContext(ctx); got != "second" {
		t.Errorf("expected second, got %q", got)
	}
}

func TestExtra_WithTraceID_NestedContext(t *testing.T) {
	type otherKey string
	inner := context.WithValue(context.Background(), otherKey("foo"), "bar")
	ctx := WithTraceID(inner, "trace-nested")
	if got := FromContext(ctx); got != "trace-nested" {
		t.Errorf("nested: expected trace-nested, got %q", got)
	}
}

func TestExtra_FromContext_EmptyContext(t *testing.T) {
	if got := FromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtra_FromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), traceIDKey, 12345)
	if got := FromContext(ctx); got != "" {
		t.Errorf("expected empty for non-string value, got %q", got)
	}
}

// =============================================================================
// LogError
// =============================================================================

func TestExtra_LogError_EmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(prev)

	ctx := WithTraceID(context.Background(), "abc-xyz")
	LogError(ctx, "operation failed", errors.New("something broke"))

	out := buf.String()
	if !strings.Contains(out, `"level":"ERROR"`) {
		t.Errorf("expected ERROR level, got %s", out)
	}
	if !strings.Contains(out, `"msg":"operation failed"`) {
		t.Errorf("expected message, got %s", out)
	}
	if !strings.Contains(out, `"error":"something broke"`) {
		t.Errorf("expected error field, got %s", out)
	}
	if !strings.Contains(out, `"trace_id":"abc-xyz"`) {
		t.Errorf("expected trace_id field, got %s", out)
	}
}

// =============================================================================
// LogInfo
// =============================================================================

func TestExtra_LogInfo_EmitsInfo(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(prev)

	LogInfo(context.Background(), "hello", slog.String("k", "v"))

	out := buf.String()
	if !strings.Contains(out, `"level":"INFO"`) {
		t.Errorf("expected INFO, got %s", out)
	}
	if !strings.Contains(out, `"msg":"hello"`) {
		t.Errorf("expected message, got %s", out)
	}
	if !strings.Contains(out, `"k":"v"`) {
		t.Errorf("expected extra k=v, got %s", out)
	}
}

func TestExtra_LogInfo_NoExtraArgs(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(prev)

	LogInfo(context.Background(), "minimal")

	out := buf.String()
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		// JSON might include extra stderr output; do regex check instead
		re := regexp.MustCompile(`"msg":"minimal"`)
		if !re.MatchString(out) {
			t.Fatalf("missing msg field in %s", out)
		}
		return
	}
	if m["msg"] != "minimal" {
		t.Errorf("expected msg=minimal, got %v", m["msg"])
	}
}
