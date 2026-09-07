// Package tracing provides W3C TraceContext and B3 propagation support.
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
	FormatW3C       PropagationFormat = "w3c"
	FormatB3        PropagationFormat = "b3"
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
	return propagation.NewCompositeTextMapPropagator(
		NewB3SingleHeaderPropagator(),
		NewB3MultiHeaderPropagator(),
	)
}

// NewCompositePropagator uses both W3C and B3 for maximum compatibility.
func NewCompositePropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		NewB3SingleHeaderPropagator(),
		NewB3MultiHeaderPropagator(),
		propagation.Baggage{},
	)
}

// Inject injects trace context into the carrier.
func (p *Propagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	p.propagator.Inject(ctx, carrier)
}

// Extract extracts trace context from the carrier.
func (p *Propagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
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

// B3SingleHeaderPropagator handles B3 in single header format.
type b3SingleHeaderPropagator struct{}

// NewB3SingleHeaderPropagator creates a new B3 single-header propagator.
func NewB3SingleHeaderPropagator() propagation.TextMapPropagator {
	return &b3SingleHeaderPropagator{}
}

func (p *b3SingleHeaderPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return
	}

	sampled := "0"
	if sc.TraceFlags().IsSampled() {
		sampled = "1"
	}
	carrier.Set("X-B3-SampledId", sc.TraceID().String()+"-"+sc.SpanID().String()+"-"+sampled)
}

func (p *b3SingleHeaderPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return ctx
}

// Fields returns the list of fields this propagator injects/extracts.
func (p *b3SingleHeaderPropagator) Fields() []string {
	return []string{"X-B3-SampledId"}
}

// B3MultiHeaderPropagator handles B3 in multiple header format.
type b3MultiHeaderPropagator struct{}

// NewB3MultiHeaderPropagator creates a new B3 multi-header propagator.
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
	return ctx
}

// Fields returns the list of fields this propagator injects/extracts.
func (p *b3MultiHeaderPropagator) Fields() []string {
	return []string{"X-B3-TraceId", "X-B3-SpanId", "X-B3-Sampled"}
}

// HTTPHeaderCarrier adapts http.Header to TextMapCarrier.
type HTTPHeaderCarrier map[string][]string

// Get returns the first value for the given key.
func (c HTTPHeaderCarrier) Get(key string) string {
	v := c[key]
	if len(v) > 0 {
		return v[0]
	}
	return ""
}

// Set sets a header value.
func (c HTTPHeaderCarrier) Set(key, value string) {
	c[key] = []string{value}
}

// Keys returns all header keys.
func (c HTTPHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
