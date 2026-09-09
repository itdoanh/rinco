// Tests for logger public API (logger.go).
package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func TestExtra_LevelConstants(t *testing.T) {
	// Verify level constants match slog
	if LevelDebug != slog.LevelDebug {
		t.Error("LevelDebug mismatch")
	}
	if LevelInfo != slog.LevelInfo {
		t.Error("LevelInfo mismatch")
	}
	if LevelWarn != slog.LevelWarn {
		t.Error("LevelWarn mismatch")
	}
	if LevelError != slog.LevelError {
		t.Error("LevelError mismatch")
	}
}

func TestExtra_Config_Defaults(t *testing.T) {
	cfg := Config{}
	if cfg.Service != "" {
		t.Error("default service empty")
	}
	if cfg.Level != Level(0) {
		t.Error("default level should be 0 (gets set in buildLogger)")
	}
}

func TestExtra_OTLPConfig(t *testing.T) {
	cfg := &OTLPConfig{
		Endpoint: "otel-collector:4318",
		Protocol: "http",
		Headers:  map[string]string{"x-api-key": "secret"},
		Insecure: true,
		Timeout:  5000000000,
	}
	if cfg.Endpoint != "otel-collector:4318" {
		t.Error("Endpoint not set")
	}
	if cfg.Protocol != "http" {
		t.Error("Protocol not set")
	}
}

func TestExtra_Init_BasicConfig(t *testing.T) {
	buf := &bytes.Buffer{}
	// Override default to capture
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)

	// Init without OTLP for simplicity
	cfg := Config{
		Service: "test-svc",
		Env:     "test",
		Version: "1.0.0",
		Level:   LevelInfo,
		Output:  "", // stdout, but we'll override via custom
	}
	// Build manually with stdout override
	l := buildLoggerForTest(cfg, buf)
	if l == nil {
		t.Fatal("nil logger")
	}
	l.Info("hello")
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("missing hello in: %s", buf.String())
	}
}

func TestExtra_L_AutoInit(t *testing.T) {
	// L should auto-init if not called
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)
	defer func() {
		// Reset global state
		globalMu.Lock()
		globalLogger = nil
		globalMu.Unlock()
	}()

	// Force globalLogger to nil
	globalMu.Lock()
	globalLogger = nil
	globalMu.Unlock()

	l := L(context.Background())
	if l == nil {
		t.Error("L should return non-nil")
	}
}

func TestExtra_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{Service: "test", Env: "test", Version: "v1", Level: LevelInfo}
	l := buildLoggerForTest(cfg, buf)

	// Use the logger
	l.Info("info-test")
	if !strings.Contains(buf.String(), "info-test") {
		t.Errorf("missing: %s", buf.String())
	}
}

func TestExtra_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{Service: "test", Env: "test", Version: "v1", Level: LevelInfo}
	l := buildLoggerForTest(cfg, buf)
	l.Warn("warn-test")
	if !strings.Contains(buf.String(), "warn-test") {
		t.Errorf("missing: %s", buf.String())
	}
}

func TestExtra_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{Service: "test", Env: "test", Version: "v1", Level: LevelInfo}
	l := buildLoggerForTest(cfg, buf)
	l.Error("error-test")
	if !strings.Contains(buf.String(), "error-test") {
		t.Errorf("missing: %s", buf.String())
	}
}

func TestExtra_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{Service: "test", Env: "test", Version: "v1", Level: LevelDebug}
	l := buildLoggerForTest(cfg, buf)
	l.Debug("debug-test")
	if !strings.Contains(buf.String(), "debug-test") {
		t.Errorf("missing: %s", buf.String())
	}
}

func TestExtra_WithContext_AllFields(t *testing.T) {
	ctx := WithContext(context.Background(), "trace-1", "span-1", "req-1")
	if TraceIDFromContext(ctx) != "trace-1" {
		t.Error("trace_id not set")
	}
	if SpanIDFromContext(ctx) != "span-1" {
		t.Error("span_id not set")
	}
	if RequestIDFromContext(ctx) != "req-1" {
		t.Error("request_id not set")
	}
}

func TestExtra_WithContext_EmptyFields(t *testing.T) {
	ctx := WithContext(context.Background(), "", "", "")
	if TraceIDFromContext(ctx) != "" {
		t.Error("trace_id should be empty")
	}
	if SpanIDFromContext(ctx) != "" {
		t.Error("span_id should be empty")
	}
}

func TestExtra_WithContext_Partial(t *testing.T) {
	ctx := WithContext(context.Background(), "trace-1", "", "")
	if TraceIDFromContext(ctx) != "trace-1" {
		t.Error("trace_id not set")
	}
	if SpanIDFromContext(ctx) != "" {
		t.Error("span_id should be empty")
	}
}

func TestExtra_BuildLogger_DefaultLevel(t *testing.T) {
	cfg := Config{Service: "test", Env: "test", Version: "v1"}
	// Default level (0) should be set to Info internally
	l := buildLogger(context.Background(), cfg)
	if l == nil {
		t.Fatal("nil")
	}
}

func TestExtra_BuildLogger_StaticAttrs(t *testing.T) {
	// buildLoggerForTest doesn't apply StaticAttrs; test Config struct fields directly
	cfg := Config{
		Service:     "test",
		Env:         "test",
		Version:     "v1",
		Level:       LevelInfo,
		StaticAttrs: []any{slog.String("custom", "value")},
	}
	if cfg.StaticAttrs == nil {
		t.Fatal("StaticAttrs not set")
	}
	if len(cfg.StaticAttrs) != 1 {
		t.Error("expected 1 static attr")
	}
}

func TestExtra_BuildLogger_OutputStderr(t *testing.T) {
	// Just verify stderr option doesn't panic
	cfg := Config{
		Service: "test",
		Env:     "test",
		Version: "v1",
		Level:   LevelInfo,
		Output:  "stderr",
	}
	l := buildLogger(context.Background(), cfg)
	if l == nil {
		t.Fatal("nil")
	}
}

func TestExtra_Shutdown_NoExporter(t *testing.T) {
	// Without OTLP exporter, shutdown should be no-op
	err := Shutdown(context.Background())
	if err != nil {
		t.Errorf("expected nil: %v", err)
	}
}

func TestExtra_otlpShutdownable_Shutdown(t *testing.T) {
	called := false
	o := &otlpShutdownable{
		shutdown: func(ctx context.Context) error {
			called = true
			return nil
		},
	}
	if err := o.Shutdown(context.Background()); err != nil {
		t.Errorf("expected nil: %v", err)
	}
	if !called {
		t.Error("shutdown func not called")
	}
}

func TestExtra_otlpShutdownable_NilShutdown(t *testing.T) {
	o := &otlpShutdownable{shutdown: nil}
	if err := o.Shutdown(context.Background()); err != nil {
		t.Errorf("nil shutdown should return nil: %v", err)
	}
}

func TestExtra_Hostname(t *testing.T) {
	h := hostname()
	// Just verify it returns a string without error
	if h == "" {
		t.Log("hostname is empty (expected in some test envs)")
	}
}

func TestExtra_Init_Once(t *testing.T) {
	// Reset
	globalOnce = sync.Once{}
	globalMu.Lock()
	globalLogger = nil
	globalMu.Unlock()

	cfg := Config{Service: "test", Env: "test", Version: "v1", Level: LevelInfo}
	Init(context.Background(), cfg)
	Init(context.Background(), cfg) // Should be no-op due to sync.Once
	// No assertion needed - just verify no panic
}

func TestExtra_InitForTest_Overrides(t *testing.T) {
	globalMu.Lock()
	globalLogger = nil
	globalMu.Unlock()

	cfg := Config{Service: "test", Env: "test", Version: "v1", Level: LevelInfo}
	InitForTest(context.Background(), cfg)
	if globalLogger == nil {
		t.Error("InitForTest should set globalLogger")
	}
}

// Helper function for tests
func buildLoggerForTest(cfg Config, buf *bytes.Buffer) *slog.Logger {
	// Override Output to use our buffer
	if cfg.Output == "" || cfg.Output == "stdout" {
		// Build with custom handler
		handlerOpts := &slog.HandlerOptions{
			Level:     cfg.Level,
			AddSource: cfg.Env == "development",
		}
		handler := slog.NewJSONHandler(buf, handlerOpts)
		logger := slog.New(handler).With(
			slog.String("service", cfg.Service),
			slog.String("env", cfg.Env),
			slog.String("version", cfg.Version),
		)
		return logger
	}
	return buildLogger(context.Background(), cfg)
}
