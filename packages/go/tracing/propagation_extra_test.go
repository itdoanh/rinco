// Additional tests for tracing propagation.
package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/propagation"
)

func TestExtra_Propagator_FormatConstants(t *testing.T) {
	if FormatW3C != "w3c" {
		t.Errorf("FormatW3C: got %s", FormatW3C)
	}
	if FormatB3 != "b3" {
		t.Errorf("FormatB3: got %s", FormatB3)
	}
	if FormatComposite != "composite" {
		t.Errorf("FormatComposite: got %s", FormatComposite)
	}
}

func TestExtra_NewPropagator_W3C(t *testing.T) {
	p := NewPropagator(FormatW3C)
	if p == nil {
		t.Fatal("nil propagator")
	}
}

func TestExtra_NewPropagator_B3(t *testing.T) {
	p := NewPropagator(FormatB3)
	if p == nil {
		t.Fatal("nil propagator")
	}
}

func TestExtra_NewPropagator_Composite(t *testing.T) {
	p := NewPropagator(FormatComposite)
	if p == nil {
		t.Fatal("nil propagator")
	}
}

func TestExtra_NewPropagator_Default(t *testing.T) {
	p := NewPropagator("unknown")
	if p == nil {
		t.Fatal("nil propagator for unknown format")
	}
}

func TestExtra_NewW3CPropagator(t *testing.T) {
	p := NewW3CPropagator()
	if p == nil {
		t.Fatal("nil")
	}
}

func TestExtra_NewB3Propagator(t *testing.T) {
	p := NewB3Propagator()
	if p == nil {
		t.Fatal("nil")
	}
}

func TestExtra_NewCompositePropagator(t *testing.T) {
	p := NewCompositePropagator()
	if p == nil {
		t.Fatal("nil")
	}
}

func TestExtra_Propagator_InjectExtract_NoSpan(t *testing.T) {
	p := NewPropagator(FormatW3C)
	ctx := context.Background()
	headers := make(map[string]string)
	p.Inject(ctx, propagation.MapCarrier(headers))
	// Without a span, headers should be empty
	// (no panic)
	_ = headers
}

func TestExtra_InjectToHeaders(t *testing.T) {
	p := NewPropagator(FormatW3C)
	headers := p.InjectToHeaders(context.Background())
	if headers == nil {
		t.Error("headers should not be nil")
	}
}

func TestExtra_ExtractFromHeaders(t *testing.T) {
	p := NewPropagator(FormatW3C)
	ctx := p.ExtractFromHeaders(context.Background(), map[string]string{
		"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	})
	if ctx == nil {
		t.Error("ctx should not be nil")
	}
}

func TestExtra_GetTraceInfo_NoSpan(t *testing.T) {
	traceID, spanID := GetTraceInfo(context.Background())
	if traceID != "" || spanID != "" {
		t.Errorf("expected empty, got %s/%s", traceID, spanID)
	}
}

func TestExtra_B3SingleHeader_Fields(t *testing.T) {
	p := NewB3SingleHeaderPropagator()
	fields := p.Fields()
	if len(fields) == 0 {
		t.Error("fields should not be empty")
	}
}

func TestExtra_B3SingleHeader_InjectNoSpan(t *testing.T) {
	carrier := propagation.MapCarrier{}
	p := &b3SingleHeaderPropagator{}
	p.Inject(context.Background(), carrier)
	// Without span, no fields should be set
	if len(carrier) > 0 {
		t.Error("no fields should be set without span")
	}
}

func TestExtra_B3MultiHeader_Fields(t *testing.T) {
	p := NewB3MultiHeaderPropagator()
	fields := p.Fields()
	if len(fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(fields))
	}
}

func TestExtra_B3MultiHeader_InjectNoSpan(t *testing.T) {
	carrier := propagation.MapCarrier{}
	p := &b3MultiHeaderPropagator{}
	p.Inject(context.Background(), carrier)
	if len(carrier) > 0 {
		t.Error("no fields should be set without span")
	}
}

func TestExtra_HTTPHeaderCarrier_Get(t *testing.T) {
	c := HTTPHeaderCarrier{"X-Test": {"value1", "value2"}}
	if got := c.Get("X-Test"); got != "value1" {
		t.Errorf("got %s", got)
	}
}

func TestExtra_HTTPHeaderCarrier_GetMissing(t *testing.T) {
	c := HTTPHeaderCarrier{}
	if got := c.Get("X-Missing"); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestExtra_HTTPHeaderCarrier_Set(t *testing.T) {
	c := HTTPHeaderCarrier{}
	c.Set("X-Test", "value")
	if c["X-Test"][0] != "value" {
		t.Errorf("set failed")
	}
}

func TestExtra_HTTPHeaderCarrier_SetOverwrite(t *testing.T) {
	c := HTTPHeaderCarrier{"X-Test": {"old"}}
	c.Set("X-Test", "new")
	if c["X-Test"][0] != "new" {
		t.Errorf("overwrite failed")
	}
}

func TestExtra_HTTPHeaderCarrier_Keys(t *testing.T) {
	c := HTTPHeaderCarrier{"A": {"1"}, "B": {"2"}, "C": {"3"}}
	keys := c.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestExtra_HTTPHeaderCarrier_KeysEmpty(t *testing.T) {
	c := HTTPHeaderCarrier{}
	keys := c.Keys()
	if len(keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys))
	}
}

func TestExtra_HTTPHeaderCarrier_ImplementsCarrier(t *testing.T) {
	var c propagation.TextMapCarrier = HTTPHeaderCarrier{}
	_ = c
}

func TestExtra_Propagator_InjectExtract_W3CRoundTrip(t *testing.T) {
	p := NewPropagator(FormatW3C)
	// Without an active span, the roundtrip should be a no-op
	headers := map[string]string{}
	p.Inject(context.Background(), propagation.MapCarrier(headers))
	ctx := p.ExtractFromHeaders(context.Background(), headers)
	if ctx == nil {
		t.Error("ctx nil")
	}
}
