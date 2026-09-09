// Tests for logger context helpers.
package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestExtra_WithTraceID(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-123")
	if got := TraceIDFromContext(ctx); got != "trace-123" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_WithSpanID(t *testing.T) {
	ctx := WithSpanID(context.Background(), "span-456")
	if got := SpanIDFromContext(ctx); got != "span-456" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_WithRequestID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-789")
	if got := RequestIDFromContext(ctx); got != "req-789" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_WithTenantID(t *testing.T) {
	ctx := WithTenantID(context.Background(), "tenant-1")
	if got := TenantIDFromContext(ctx); got != "tenant-1" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_WithUserID(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-1")
	if got := UserIDFromContext(ctx); got != "user-1" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_TraceIDFromContext_Empty(t *testing.T) {
	if got := TraceIDFromContext(context.Background()); got != "" {
		t.Errorf("expected empty: got %s", got)
	}
}

func TestExtra_SpanIDFromContext_Empty(t *testing.T) {
	if got := SpanIDFromContext(context.Background()); got != "" {
		t.Errorf("expected empty: got %s", got)
	}
}

func TestExtra_RequestIDFromContext_Empty(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("expected empty: got %s", got)
	}
}

func TestExtra_TenantIDFromContext_Empty(t *testing.T) {
	if got := TenantIDFromContext(context.Background()); got != "" {
		t.Errorf("expected empty: got %s", got)
	}
}

func TestExtra_UserIDFromContext_Empty(t *testing.T) {
	if got := UserIDFromContext(context.Background()); got != "" {
		t.Errorf("expected empty: got %s", got)
	}
}

func TestExtra_EnrichLogger_NilLogger(t *testing.T) {
	// nil logger should return slog.Default()
	original := slog.Default()
	defer slog.SetDefault(original)

	buf := &bytes.Buffer{}
	l := slog.New(slog.NewTextHandler(buf, nil))
	slog.SetDefault(l)

	ctx := WithTraceID(context.Background(), "trace-1")
	enriched := enrichLogger(nil, ctx)
	if enriched == nil {
		t.Error("expected non-nil")
	}
}

func TestExtra_EnrichLogger_NoContext(t *testing.T) {
	buf := &bytes.Buffer{}
	l := slog.New(slog.NewTextHandler(buf, nil))

	enriched := enrichLogger(l, context.Background())
	if enriched == nil {
		t.Error("expected non-nil")
	}
}

func TestExtra_EnrichLogger_WithTrace(t *testing.T) {
	buf := &bytes.Buffer{}
	l := slog.New(slog.NewTextHandler(buf, nil))

	ctx := WithTraceID(context.Background(), "trace-1")
	enriched := enrichLogger(l, ctx)
	enriched.Info("test")

	if !strings.Contains(buf.String(), "trace_id=trace-1") {
		t.Errorf("missing trace_id: %s", buf.String())
	}
}

func TestExtra_EnrichLogger_WithAll(t *testing.T) {
	buf := &bytes.Buffer{}
	l := slog.New(slog.NewTextHandler(buf, nil))

	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-1")
	ctx = WithSpanID(ctx, "span-1")
	ctx = WithRequestID(ctx, "req-1")
	ctx = WithTenantID(ctx, "tenant-1")
	ctx = WithUserID(ctx, "user-1")

	enriched := enrichLogger(l, ctx)
	enriched.Info("test")

	s := buf.String()
	for _, key := range []string{"trace_id", "span_id", "request_id", "tenant_id", "user_id"} {
		if !strings.Contains(s, key) {
			t.Errorf("missing %s in: %s", key, s)
		}
	}
}

func TestExtra_EnrichLogger_Partial(t *testing.T) {
	buf := &bytes.Buffer{}
	l := slog.New(slog.NewTextHandler(buf, nil))

	ctx := WithTenantID(context.Background(), "tenant-1")
	enriched := enrichLogger(l, ctx)
	enriched.Info("test")

	s := buf.String()
	if !strings.Contains(s, "tenant_id=tenant-1") {
		t.Errorf("missing tenant_id: %s", s)
	}
	if strings.Contains(s, "trace_id=") {
		t.Errorf("unexpected trace_id: %s", s)
	}
}

func TestExtra_ContextKeyIsolation(t *testing.T) {
	// Different keys should not collide
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-1")
	if SpanIDFromContext(ctx) != "" {
		t.Error("traceID key should not affect spanID")
	}
	if RequestIDFromContext(ctx) != "" {
		t.Error("traceID key should not affect requestID")
	}
}

func TestExtra_OverwriteContext(t *testing.T) {
	// Setting twice should overwrite
	ctx := WithTraceID(context.Background(), "first")
	ctx = WithTraceID(ctx, "second")
	if got := TraceIDFromContext(ctx); got != "second" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_EnrichLogger_NoAttrs(t *testing.T) {
	buf := &bytes.Buffer{}
	l := slog.New(slog.NewTextHandler(buf, nil))

	// Empty context, enrich should return l
	enriched := enrichLogger(l, context.Background())
	if enriched != l {
		t.Error("should return same logger when no attrs")
	}
}
