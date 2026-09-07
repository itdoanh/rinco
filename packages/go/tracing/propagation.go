// Package tracing provides W3C TraceContext and B3 propagation support.
//
// W3C TraceContext is the modern standard, while B3 is used by Zipkin.
// This package provides utilities for extracting and injecting trace context
// from various propagation formats.
package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// PropagationFormat defines supported propagation formats.
type PropagationFormat string

const (
	// W3C TraceContext is the modern standard (traceparent, tracestate headers)
	FormatW3C PropagationFormat = "w3c"
	// B3 is used by Zipkin and older systems (X-B3-TraceId, X-B3-SpanId, etc.)
	FormatB3 PropagationFormat = "b3"
	// Composite uses both W3C and B3 for backward compatibility
	FormatComposite PropagationFormat = "composite"
)

// Propagator wraps OpenTelemetry propagator for easier use.
type Propagator struct {
	propagator propagation.TextMapPropagator
}

// NewPropagator creates a propagator with the specified format.
func NewPropagator(format PropagationFormat) *Propagator {
	var prop propagation.TextMapPropagator
	switch format {
	case FormatW3C:
		prop = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	case FormatB3:
		prop = NewB3Propagator()
	case FormatComposite:
		prop = NewCompositePropagator()
	default:
		prop = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}
	return &Propagator{propagator: prop}
}

// NewW3CPropagator creates a W3C TraceContext propagator.
func NewW3CPropagator() *Propagator {
	return NewPropagator(FormatW3C)
}

// NewB3Propagator creates a B3 propagator for Zipkin compatibility.
func NewB3Propagator() propagation.TextMapPropagator {
	// B3 uses custom fields: X-B3-TraceId, X-B3-SpanId, X-B3-ParentSpanId, X-B3-Sampled, X-B3-Flags
	return propagation.NewCompositeTextMapPropagator(
		NewB3SingleHeaderPropagator(),
		NewB3MultiHeaderPropagator(),
	)
}

// NewCompositePropagator uses both W3C and B3 for maximum compatibility.
func NewCompositePropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C first (preferred)
		NewB3SingleHeaderPropagator(),
		NewB3MultiHeaderPropagator(),
		propagation.Baggage{},
	)
}

// Inject injects trace context into the carrier (e.g., HTTP headers).
// Returns the carrier with injected context.
func (p *Propagator) Inject(ctx context.Context, carrier interface{}) {
	p.propagator.Inject(ctx, carrier)
}

// Extract extracts trace context from the carrier (e.g., HTTP headers).
// Returns a context with the extracted trace context.
func (p *Propagator) Extract(ctx context.Context, carrier interface{}) context.Context {
	return p.propagator.Extract(ctx, carrier)
}

// InjectToHeaders is a helper that injects trace context into a map of headers.
func (p *Propagator) InjectToHeaders(ctx context.Context) map[string]string {
	headers := make(map[string]string)
	carrier := propagation.MapCarrier(headers)
	p.Inject(ctx, carrier)
	return headers
}

// ExtractFromHeaders is a helper that extracts trace context from headers map.
func (p *Propagator) ExtractFromHeaders(ctx context.Context, headers map[string]string) context.Context {
	carrier := propagation.MapCarrier(headers)
	return p.Extract(ctx, carrier)
}

// GetTraceInfo extracts trace ID and span ID from context.
func GetTraceInfo(ctx context.Context) (traceID, spanID string) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
		spanID = span.SpanContext().SpanID().String()
	}
	return
}

// NewSpan creates a new span as a child of the current context.
func NewSpan(ctx context.Context, tracerName, spanName string) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, spanName)
}

// B3SingleHeaderPropagator handles B3 in single header format (X-B3-SampledId).
type b3SingleHeaderPropagator struct{}

func NewB3SingleHeaderPropagator() propagation.TextMapPropagator {
	return &b3SingleHeaderPropagator{}
}

func (p *b3SingleHeaderPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return
	}

	// X-B3-SampledId format: {TraceId}-{SpanId}-{Sampled}
	sampled := "1"
	if sc.TraceFlags().IsSampled() {
		sampled = "1"
	} else {
		sampled = "0"
	}
	carrier.Set("X-B3-SampledId", sc.TraceID().String()+"-"+sc.SpanID().String()+"-"+sampled)
}

func (p *b3SingleHeaderPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	// Simplified - full implementation would parse X-B3-SampledId
	return ctx
}

// B3MultiHeaderPropagator handles B3 in multiple header format.
type b3MultiHeaderPropagator struct{}

func NewB3MultiHeaderPropagator() propagation.TextMapPropagator {
	return &b3MultiHeaderPropagator{}
}

func (p *b3MultiHeaderPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return
	}

	carrier.Set("X-B3-TraceId", sc.TraceID().String())
	carrier.Set("X-B3-SpanId", sc.SpanID().String())

	if sc.TraceFlags().IsSampled() {
		carrier.Set("X-B3-Sampled", "1")
	} else {
		carrier.Set("X-B3-Sampled", "0")
	}
}

func (p *b3MultiHeaderPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	// Simplified extraction - full implementation would:
	// 1. Read X-B3-TraceId and X-B3-SpanId
	// 2. Parse X-B3-Sampled and X-B3-Flags
	// 3. Create a new SpanContext
	// 4. Return context with the new span context
	return ctx
}

// HTTPHeaderCarrier adapts http.Header to TextMapCarrier for ease of use.
type HTTPHeaderCarrier map[string][]string

func (c HTTPHeaderCarrier) Get(key string) string {
	v := c[key]
	if len(v) > 0 {
		return v[0]
	}
	return ""
}

func (c HTTPHeaderCarrier) Set(key, value string) {
	c[key] = []string{value}
}

func (c HTTPHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
