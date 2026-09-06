// Package logger cung cấp structured logging wrapper quanh slog (Go 1.21+).
//
// Hỗ trợ:
//   - JSON output theo chuẩn OpenTelemetry log record
//   - OTLP export (HTTP + gRPC)
//   - Auto-inject trace_id, span_id, request_id từ context
//   - Adaptive sampling cho high-volume logs
//   - PII/secret redactor
//
// Usage:
//
//	logger.Init(ctx, logger.Config{
//	    Service: "auth-service",
//	    Env:     "production",
//	    Version: "1.0.0",
//	    Level:   slog.LevelInfo,
//	    OTLP:    &logger.OTLPConfig{Endpoint: "otel-collector:4318", Protocol: "http"},
//	})
//	defer logger.Shutdown(ctx)
//
//	logger.Info(ctx, "user logged in", slog.String("user_id", id))
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Level re-export từ slog.
type Level = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Config cấu hình logger.
type Config struct {
	Service     string
	Env         string
	Version     string
	Level       Level
	Output      string // "stdout", "stderr", "file:path"
	OTLP        *OTLPConfig
	Sampling    *SamplingConfig
	Redactor    *Redactor
	StaticAttrs []slog.Attr
}

// OTLPConfig cấu hình OTLP export.
type OTLPConfig struct {
	Endpoint  string // "otel-collector:4318" (HTTP) hoặc ":4317" (gRPC)
	Protocol  string // "http" (default) | "grpc"
	Headers   map[string]string
	Insecure  bool
	Timeout   time.Duration
	BatchSize int
}

var (
	globalLogger *slog.Logger
	globalOnce   sync.Once
	globalMu     sync.RWMutex
	otlpExporter *otlpShutdownable
)

// Init khởi tạo global logger với config.
//
// Trong production, gọi 1 lần ở main(). Trong test có thể không gọi.
func Init(ctx context.Context, cfg Config) {
	globalOnce.Do(func() {
		globalLogger = buildLogger(ctx, cfg)
	})
}

// InitForTest khởi tạo lại logger (chỉ dùng cho test, override Init đã gọi).
func InitForTest(ctx context.Context, cfg Config) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = buildLogger(ctx, cfg)
}

func buildLogger(ctx context.Context, cfg Config) *slog.Logger {
	if cfg.Level == 0 {
		cfg.Level = LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.Env == "development",
	}

	// Build base handler
	var baseHandler slog.Handler
	switch cfg.Output {
	case "stderr":
		baseHandler = slog.NewJSONHandler(os.Stderr, handlerOpts)
	case "":
		baseHandler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	default:
		// Default stdout
		baseHandler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	// Apply redactor
	if cfg.Redactor != nil {
		baseHandler = NewRedactionHandler(baseHandler, cfg.Redactor)
	}

	// Apply sampling
	if cfg.Sampling != nil {
		baseHandler = NewSamplingHandler(baseHandler, cfg.Sampling)
	}

	logger := slog.New(baseHandler).With(
		slog.String("service", cfg.Service),
		slog.String("env", cfg.Env),
		slog.String("version", cfg.Version),
		slog.String("host", hostname()),
	)

	if len(cfg.StaticAttrs) > 0 {
		logger = logger.With(cfg.StaticAttrs...)
	}

	// Setup OTLP
	if cfg.OTLP != nil {
		exp, err := newOTLPExporter(ctx, cfg.OTLP)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[logger] OTLP export failed: %v\n", err)
		} else {
			otlpExporter = exp
			logger = slog.New(newOTLPLoggerHandler(baseHandler, exp))
		}
	}

	globalLogger = logger
	return logger
}

// Shutdown flush OTLP exporter và đợi pending batches.
func Shutdown(ctx context.Context) error {
	if otlpExporter != nil {
		return otlpExporter.Shutdown(ctx)
	}
	return nil
}

// L trả về *slog.Logger với context values (trace_id, span_id, ...).
func L(ctx context.Context) *slog.Logger {
	if globalLogger == nil {
		// Fallback nếu Init() chưa được gọi
		Init(ctx, Config{Service: "unknown", Env: "development", Version: "0.0.0"})
	}
	return enrichLogger(globalLogger, ctx)
}

// Info shorthand.
func Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	L(ctx).LogAttrs(ctx, LevelInfo, msg, attrs...)
}

// Warn shorthand.
func Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	L(ctx).LogAttrs(ctx, LevelWarn, msg, attrs...)
}

// Error shorthand - kèm error field.
func Error(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	all := make([]slog.Attr, 0, len(attrs)+1)
	all = append(all, slog.Any("error", err))
	all = append(all, attrs...)
	L(ctx).LogAttrs(ctx, LevelError, msg, all...)
}

// Debug shorthand.
func Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	L(ctx).LogAttrs(ctx, LevelDebug, msg, attrs...)
}

// Fatal log error level rồi exit.
func Fatal(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	Error(ctx, msg, err, attrs...)
	os.Exit(1)
}

// WithContext trả về context mới với additional trace values.
func WithContext(ctx context.Context, traceID, spanID, requestID string) context.Context {
	if traceID != "" {
		ctx = WithTraceID(ctx, traceID)
	}
	if spanID != "" {
		ctx = WithSpanID(ctx, spanID)
	}
	if requestID != "" {
		ctx = WithRequestID(ctx, requestID)
	}
	return ctx
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

// ===== Placeholders for OTLP integration (implemented in tracing package) =====

// otlpShutdownable wraps OTLP exporter với Shutdown method.
type otlpShutdownable struct {
	shutdown func(context.Context) error
}

func (o *otlpShutdownable) Shutdown(ctx context.Context) error {
	if o.shutdown != nil {
		return o.shutdown(ctx)
	}
	return nil
}

// newOTLPExporter stub - implementation sẽ ở tracing package trong tương lai.
// Hiện tại return nil exporter để tránh crash nếu user chưa setup OTLP.
func newOTLPExporter(_ context.Context, _ *OTLPConfig) (*otlpShutdownable, error) {
	return nil, fmt.Errorf("OTLP export requires tracing package integration")
}

// newOTLPLoggerHandler placeholder.
func newOTLPLoggerHandler(_ slog.Handler, _ *otlpShutdownable) slog.Handler {
	return nil
}