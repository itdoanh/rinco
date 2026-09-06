// Package logger provides structured logging with trace/tenant context propagation.
//
// Usage:
//
//	import "github.com/rinco/go/pkg/logger"
//
//	logger.Init("auth-service", "development", "1.0.0")
//
//	ctx = logger.WithTraceID(ctx, "abc-123")
//	ctx = logger.WithTenantID(ctx, "tenant-id")
//
//	logger.Info(ctx, "user logged in", zap.String("user_id", "..."))
package logger

import (
	"context"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey string

const (
	traceIDKey   ctxKey = "trace_id"
	tenantIDKey  ctxKey = "tenant_id"
	userIDKey    ctxKey = "user_id"
	spanIDKey    ctxKey = "span_id"
	requestIDKey ctxKey = "request_id"
)

var (
	baseLogger *zap.Logger
	once       sync.Once
)

// Init khởi tạo global logger một lần.
func Init(service, env, version string) {
	once.Do(func() {
		encoderCfg := zap.NewProductionEncoderConfig()
		encoderCfg.TimeKey = "timestamp"
		encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderCfg.MessageKey = "message"
		encoderCfg.LevelKey = "level"
		encoderCfg.CallerKey = "caller"

		level := zap.InfoLevel
		if env == "development" {
			level = zap.DebugLevel
		}

		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.Lock(os.Stdout),
			zap.NewAtomicLevelAt(level),
		)

		baseLogger = zap.New(core,
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.Fields(
				zap.String("service", service),
				zap.String("env", env),
				zap.String("version", version),
				zap.String("host", hostname()),
			),
		)
	})
}

// WithContext trích context fields rồi gắn vào logger.
func WithContext(ctx context.Context) *zap.Logger {
	l := baseLogger
	if v, ok := ctx.Value(traceIDKey).(string); ok && v != "" {
		l = l.With(zap.String("trace_id", v))
	}
	if v, ok := ctx.Value(tenantIDKey).(string); ok && v != "" {
		l = l.With(zap.String("tenant_id", v))
	}
	if v, ok := ctx.Value(userIDKey).(string); ok && v != "" {
		l = l.With(zap.String("user_id", v))
	}
	if v, ok := ctx.Value(spanIDKey).(string); ok && v != "" {
		l = l.With(zap.String("span_id", v))
	}
	if v, ok := ctx.Value(requestIDKey).(string); ok && v != "" {
		l = l.With(zap.String("request_id", v))
	}
	return l
}

// Info logs info level.
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Info(msg, fields...)
}

// Warn logs warning level.
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Warn(msg, fields...)
}

// Error logs error level with error field.
func Error(ctx context.Context, msg string, err error, fields ...zap.Field) {
	all := append([]zap.Field{zap.Error(err)}, fields...)
	WithContext(ctx).Error(msg, all...)
}

// Fatal logs fatal level and exits.
func Fatal(ctx context.Context, msg string, err error, fields ...zap.Field) {
	all := append([]zap.Field{zap.Error(err)}, fields...)
	WithContext(ctx).Fatal(msg, all...)
}

// Debug logs debug level.
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Debug(msg, fields...)
}

// ============ Context helpers ============

// WithTraceID returns a new context with the given trace ID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithTenantID returns a new context with the given tenant ID.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// WithUserID returns a new context with the given user ID.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithSpanID returns a new context with the given span ID.
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, spanIDKey, spanID)
}

// WithRequestID returns a new context with the given request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// TraceIDFromContext extracts the trace ID from the context.
func TraceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}

// TenantIDFromContext extracts the tenant ID from the context.
func TenantIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// ============ Internal ============

func hostname() string {
	h, _ := os.Hostname()
	return h
}

// Sync flushes any buffered log entries.
func Sync() {
	if baseLogger != nil {
		_ = baseLogger.Sync()
	}
}

// NowRFC3339 returns current time in RFC3339 format.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
