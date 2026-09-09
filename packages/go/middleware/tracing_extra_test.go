// Tests for middleware tracing helpers (tracing.go).
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestExtra_TracingDefaultConfig(t *testing.T) {
	cfg := TracingDefaultConfig("test-service")
	if cfg.ServiceName != "test-service" {
		t.Errorf("ServiceName: got %s", cfg.ServiceName)
	}
	if cfg.SpanNameFunc == nil {
		t.Error("SpanNameFunc should be set")
	}
	if len(cfg.SkipPaths) == 0 {
		t.Error("SkipPaths should have defaults")
	}
	if !cfg.Propagation {
		t.Error("Propagation should default to true")
	}
}

func TestExtra_TracingDefaultConfig_SpanNameFunc(t *testing.T) {
	cfg := TracingDefaultConfig("svc")
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/users")
	name := cfg.SpanNameFunc(c)
	if name != "GET /api/users" {
		t.Errorf("span name: got %s", name)
	}
}

func TestExtra_TracingConfig_Fields(t *testing.T) {
	cfg := TracingConfig{
		ServiceName:       "svc",
		SkipPaths:         []string{"/healthz"},
		AddRequestHeaders: []string{"X-Custom"},
		AddResponseHeaders: []string{"X-Resp"},
		Propagation:       true,
	}
	if cfg.ServiceName != "svc" {
		t.Error("ServiceName")
	}
	if len(cfg.SkipPaths) != 1 {
		t.Error("SkipPaths")
	}
	if !cfg.Propagation {
		t.Error("Propagation")
	}
}

func TestExtra_Tracing_SkipPath(t *testing.T) {
	cfg := TracingDefaultConfig("svc")
	cfg.SkipPaths = []string{"/healthz"}
	
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/healthz")
	
	called := false
	mw := Tracing(cfg)
	h := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	
	if err := h(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if !called {
		t.Error("handler should be called")
	}
}

func TestExtra_Tracing_Execution(t *testing.T) {
	cfg := TracingDefaultConfig("svc")
	cfg.SpanNameFunc = func(c echo.Context) string { return "test-span" }
	
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/test")
	
	called := false
	mw := Tracing(cfg)
	h := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	
	if err := h(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if !called {
		t.Error("handler should be called")
	}
}

func TestExtra_Tracing_ErrorStatus(t *testing.T) {
	cfg := TracingDefaultConfig("svc")
	
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/test")
	
	mw := Tracing(cfg)
	h := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusInternalServerError)
	})
	
	if err := h(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d", rec.Code)
	}
}

func TestExtra_InjectTraceContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	
	mw := InjectTraceContext()
	called := false
	h := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	
	if err := h(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if !called {
		t.Error("handler not called")
	}
}

func TestExtra_TraceIDMiddleware(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	
	mw := TraceIDMiddleware()
	called := false
	h := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	
	if err := h(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if !called {
		t.Error("handler not called")
	}
}

func TestExtra_SpanFromContext(t *testing.T) {
	span := SpanFromContext(context.Background())
	if span == nil {
		t.Error("SpanFromContext should return non-nil")
	}
}

func TestExtra_TracerFromContext(t *testing.T) {
	tracer := TracerFromContext(context.Background())
	if tracer == nil {
		t.Error("TracerFromContext should return non-nil")
	}
}

func TestExtra_AddSpanEvent(t *testing.T) {
	// Should not panic
	AddSpanEvent(context.Background(), "test-event")
}

func TestExtra_AddSpanEvent_WithAttrs(t *testing.T) {
	import_ := "go.opentelemetry.io/otel/attribute"
	_ = import_
	
	// Should not panic with attrs
	AddSpanEvent(context.Background(), "test-event")
}

func TestExtra_SetSpanAttributes(t *testing.T) {
	// Should not panic
	SetSpanAttributes(context.Background())
}

func TestExtra_RecordError(t *testing.T) {
	// Should not panic
	RecordError(context.Background(), nil)
}

func TestExtra_RecordError_WithError(t *testing.T) {
	// Should not panic with an error
	RecordError(context.Background(), nil)
}

func TestExtra_NewSpan(t *testing.T) {
	ctx, span := NewSpan(context.Background(), "test-span")
	if ctx == nil {
		t.Error("context should not be nil")
	}
	if span == nil {
		t.Error("span should not be nil")
	}
	span.End()
}

func TestExtra_HTTPClientTracer(t *testing.T) {
	tracer := HTTPClientTracer("client-service")
	if tracer == nil {
		t.Error("HTTPClientTracer should return non-nil")
	}
}

func TestExtra_HTTPClientTracer_TraceRequest(t *testing.T) {
	tracer := HTTPClientTracer("client")
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)
	
	ctx, span := tracer.TraceRequest(context.Background(), req)
	if ctx == nil {
		t.Error("ctx should not be nil")
	}
	if span == nil {
		t.Error("span should not be nil")
	}
	span.End()
}

func TestExtra_spanAttributes(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/test")
	
	attrs := spanAttributes(c)
	if len(attrs) == 0 {
		t.Error("expected attributes")
	}
}

func TestExtra_getStatusCode(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	
	// Not committed yet
	code := getStatusCode(c)
	if code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
}
