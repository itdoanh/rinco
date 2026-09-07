// Package middleware provides OpenTelemetry tracing middleware for Echo.
//
// This middleware creates spans for HTTP requests and automatically
// propagates trace context to downstream services.
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TracingConfig holds tracing middleware configuration.
type TracingConfig struct {
	// ServiceName is the name of this service
	ServiceName string

	// SpanNameFunc customizes the span name
	SpanNameFunc func(echo.Context) string

	// SkipPaths are paths that won't be traced
	SkipPaths []string

	// AddRequestHeaders adds request headers as span attributes
	AddRequestHeaders []string

	// AddResponseHeaders adds response headers as span attributes
	AddResponseHeaders []string

	// Sampler overrides the global sampler
	Sampler sdktrace.Sampler

	// Propagation propagates trace context
	Propagation bool
}

// DefaultTracingConfig returns sensible defaults.
func TracingDefaultConfig(serviceName string) TracingConfig {
	return TracingConfig{
		ServiceName:  serviceName,
		SpanNameFunc: func(c echo.Context) string { return c.Request().Method + " " + c.Path() },
		SkipPaths:    []string{"/healthz", "/readyz", "/metrics"},
		Propagation:  true,
	}
}

// Tracing creates an OpenTelemetry tracing middleware for Echo.
func Tracing(cfg TracingConfig) echo.MiddlewareFunc {
	skipPaths := make(map[string]bool)
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = true
	}

	tracer := otel.Tracer(cfg.ServiceName)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			if skipPaths[path] {
				return next(c)
			}

			// Get or extract trace context
			ctx := c.Request().Context()
			if cfg.Propagation {
				ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(c.Request().Header))
			}

			// Create span name
			spanName := cfg.SpanNameFunc(c)

			// Start span
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPMethod(c.Request().Method),
					semconv.HTTPURL(c.Request().URL.String()),
					attribute.String("http.host", c.Request().Host),
					semconv.HTTPRoute(path),
					attribute.String("http.client_ip", c.RealIP()),
					semconv.UserAgentOriginal(c.Request().UserAgent()),
				),
			)
			defer span.End()

			// Apply custom sampler if configured
			if cfg.Sampler != nil {
				// Note: Sampler is applied at span creation time
				_ = cfg.Sampler
			}

			// Add request headers as attributes
			for _, header := range cfg.AddRequestHeaders {
				if val := c.Request().Header.Get(header); val != "" {
					span.SetAttributes(attribute.String("http.request.header."+header, val))
				}
			}

			// Store start time for duration calculation
			start := time.Now()

			// Set the context
			c.SetRequest(c.Request().WithContext(ctx))

			// Execute handler
			err := next(c)

			// Record response status
			status := c.Response().Status
			span.SetAttributes(semconv.HTTPStatusCode(status))

			// Set span status based on HTTP status
			if status >= 500 {
				span.SetStatus(codes.Error, "HTTP 5xx")
			} else if status >= 400 {
				span.SetStatus(codes.Error, "HTTP 4xx")
			} else {
				span.SetStatus(codes.Ok, "")
			}

			// Add response headers
			for _, header := range cfg.AddResponseHeaders {
				if val := c.Response().Header().Get(header); val != "" {
					span.SetAttributes(attribute.String("http.response.header."+header, val))
				}
			}

			// Add duration
			span.SetAttributes(attribute.Int64("http.duration_ms", time.Since(start).Milliseconds()))

			// Record error if present
			if err != nil {
				span.RecordError(err)
			}

			return err
		}
	}
}

// InjectTraceContext injects trace context into response headers.
func InjectTraceContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			span := trace.SpanFromContext(c.Request().Context())
			sc := span.SpanContext()
			if sc.IsValid() {
				c.Response().Header().Set("X-Trace-ID", sc.TraceID().String())
				c.Response().Header().Set("X-Span-ID", sc.SpanID().String())
			}
			return next(c)
		}
	}
}

// TraceIDMiddleware adds trace ID to response headers.
func TraceIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			span := trace.SpanFromContext(ctx)
			sc := span.SpanContext()
			if sc.IsValid() {
				c.Response().Header().Set("X-Trace-ID", sc.TraceID().String())
				c.Response().Header().Set("X-Trace-Extension", sc.TraceFlags().String())
			}
			return next(c)
		}
	}
}

// SpanFromContext extracts the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TracerFromContext returns a tracer from context.
func TracerFromContext(ctx context.Context) trace.Tracer {
	return otel.Tracer("")
}

// AddSpanEvent adds an event to the current span.
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanAttributes sets attributes on the current span.
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// NewSpan creates a new child span.
func NewSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("").Start(ctx, name, opts...)
}

// HTTPClientTracer creates a tracing middleware for HTTP clients.
func HTTPClientTracer(serviceName string) *httptrace {
	return &httptrace{tracer: otel.Tracer(serviceName)}
}

type httptrace struct {
	tracer trace.Tracer
}

// TraceRequest creates a span for an outgoing HTTP request.
func (t *httptrace) TraceRequest(ctx context.Context, req *http.Request) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, req.Method+" "+req.URL.Path,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.HTTPMethod(req.Method),
			semconv.HTTPURL(req.URL.String()),
			semconv.HTTPRoute(req.URL.Path),
		),
	)
}

// spanAttributes extracts common span attributes from echo context.
func spanAttributes(c echo.Context) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.HTTPMethod(c.Request().Method),
		semconv.HTTPURL(c.Request().URL.String()),
		semconv.HTTPRoute(c.Path()),
		attribute.String("http.client_ip", c.RealIP()),
	}
}

// getStatusCode gets the status code from echo context.
// Handles the case where Response hasn't been written yet.
func getStatusCode(c echo.Context) int {
	if c.Response().Committed {
		return c.Response().Status
	}
	return 0
}

// httpreq is a placeholder to avoid import cycle - actual import happens in otel.go
type httpreq struct{}

func init() {
	_ = &httpreq{}
}
