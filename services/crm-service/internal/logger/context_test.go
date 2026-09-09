package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(handler)
}

func runWithLogger(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(newTestLogger(&buf))
	defer slog.SetDefault(orig)
	fn()
	return buf.String()
}

func TestWithTraceIDRoundtrip(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-abc-123")
	if id := FromContext(ctx); id != "trace-abc-123" {
		t.Errorf("got %q", id)
	}
}

func TestWithTraceIDOverwrite(t *testing.T) {
	ctx := WithTraceID(context.Background(), "first")
	ctx = WithTraceID(ctx, "second")
	if id := FromContext(ctx); id != "second" {
		t.Errorf("got %q", id)
	}
}

func TestFromContextEmpty(t *testing.T) {
	if id := FromContext(context.Background()); id != "" {
		t.Errorf("expected empty, got %q", id)
	}
}

func TestFromContextWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), traceIDKey, 42)
	if id := FromContext(ctx); id != "" {
		t.Errorf("wrong type should yield empty, got %q", id)
	}
}

func TestLogError(t *testing.T) {
	out := runWithLogger(t, func() {
		ctx := WithTraceID(context.Background(), "tr-1")
		LogError(ctx, "failed to load", errSentinel)
	})
	if !strings.Contains(out, "failed to load") {
		t.Errorf("missing message: %s", out)
	}
	if !strings.Contains(out, "tr-1") {
		t.Errorf("missing trace_id: %s", out)
	}
	if !strings.Contains(out, "boom") {
		t.Errorf("missing error: %s", out)
	}
	var record map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &record); err != nil {
		t.Fatalf("not JSON: %s", out)
	}
	if record["level"] != "ERROR" {
		t.Errorf("level = %v", record["level"])
	}
}

func TestLogInfoWithArgs(t *testing.T) {
	out := runWithLogger(t, func() {
		ctx := WithTraceID(context.Background(), "tr-2")
		LogInfo(ctx, "info message", slog.String("k", "v"))
	})
	if !strings.Contains(out, "info message") {
		t.Errorf("missing message: %s", out)
	}
	// LogInfo does not add trace_id automatically (unlike LogError)
	if !strings.Contains(out, `"k":"v"`) {
		t.Errorf("missing k=v: %s", out)
	}
}

func TestLogDebugEmittedAtDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	orig := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(orig)

	LogDebug(context.Background(), "debug message", slog.Int("count", 7))

	if !strings.Contains(buf.String(), "debug message") {
		t.Errorf("missing debug message: %s", buf.String())
	}
}

func TestLogDebugNotEmittedAtInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	orig := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(orig)

	LogDebug(context.Background(), "debug message")

	if strings.Contains(buf.String(), "debug message") {
		t.Errorf("debug should be filtered at info level: %s", buf.String())
	}
}

type sentinel struct{}

func (sentinel) Error() string { return "boom" }

var errSentinel error = sentinel{}
