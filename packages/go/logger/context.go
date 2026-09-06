// Package logger - context-aware helpers.
package logger

import (
	"context"
	"log/slog"
)

type ctxKey string

const (
	traceIDKey   ctxKey = "trace_id"
	spanIDKey    ctxKey = "span_id"
	requestIDKey ctxKey = "request_id"
	tenantIDKey  ctxKey = "tenant_id"
	userIDKey    ctxKey = "user_id"
)

// WithTraceID returns a new context with the given trace ID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithSpanID returns a new context with the given span ID.
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, spanIDKey, spanID)
}

// WithRequestID returns a new context with the given request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithTenantID returns a new context with the given tenant ID.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// WithUserID returns a new context with the given user ID.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// TraceIDFromContext extracts trace ID from context.
func TraceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}

// SpanIDFromContext extracts span ID.
func SpanIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(spanIDKey).(string)
	return v
}

// RequestIDFromContext extracts request ID.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// TenantIDFromContext extracts tenant ID.
func TenantIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}

// UserIDFromContext extracts user ID.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// enrichLogger thêm các context attrs vào logger.
func enrichLogger(l *slog.Logger, ctx context.Context) *slog.Logger {
	if l == nil {
		return slog.Default()
	}
	attrs := make([]slog.Attr, 0, 5)
	if v := TraceIDFromContext(ctx); v != "" {
		attrs = append(attrs, slog.String("trace_id", v))
	}
	if v := SpanIDFromContext(ctx); v != "" {
		attrs = append(attrs, slog.String("span_id", v))
	}
	if v := RequestIDFromContext(ctx); v != "" {
		attrs = append(attrs, slog.String("request_id", v))
	}
	if v := TenantIDFromContext(ctx); v != "" {
		attrs = append(attrs, slog.String("tenant_id", v))
	}
	if v := UserIDFromContext(ctx); v != "" {
		attrs = append(attrs, slog.String("user_id", v))
	}
	if len(attrs) == 0 {
		return l
	}
	return l.With(attrs...)
}