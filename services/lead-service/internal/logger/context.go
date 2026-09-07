// Package logger provides structured logging helpers.
package logger

import (
	"context"
	"log/slog"
)

type ctxKey string

const traceIDKey ctxKey = "trace_id"

// WithTraceID adds trace_id to context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// FromContext extracts trace_id from context.
func FromContext(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}

// LogError logs an error with context.
func LogError(ctx context.Context, msg string, err error) {
	slog.Error(msg,
		slog.String("error", err.Error()),
		slog.String("trace_id", FromContext(ctx)),
	)
}

// LogInfo logs an info message with context.
func LogInfo(ctx context.Context, msg string, args ...any) {
	slog.Info(msg, args...)
}
