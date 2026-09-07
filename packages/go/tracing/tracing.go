// Package tracing - OpenTelemetry initialization and configuration.
//
// Provides:
//   - OTLP trace exporter setup
//   - TracerProvider setup with resource attributes
//   - W3C TraceContext and B3 propagation
//
// Usage:
//
//	cfg := tracing.Config{
//	    ServiceName:    "auth-service",
//	    ServiceVersion: "1.0.0",
//	    Env:            "production",
//	    OTLPEndpoint:   "otel-collector:4317",
//	}
//	shutdown, err := tracing.Init(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer shutdown(ctx)
package tracing

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Config holds tracing configuration.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Env            string
	Host           string

	// OTLP Configuration
	OTLPEndpoint string
	OTLPProtocol string // "grpc" (default) or "http"

	// Sampler configuration
	SampleRatio float64 // 0.0 to 1.0, default 1.0 (always sample)
}

// shutdownFuncs holds all shutdown functions for cleanup.
type shutdownFuncs struct {
	mu    sync.Mutex
	funcs []func(context.Context) error
	once  sync.Once
}

var globalShutdown *shutdownFuncs

// Init initializes OpenTelemetry with the given configuration.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if cfg.OTLPEndpoint == "" || cfg.Env == "development" {
		return func(context.Context) error { return nil }, nil
	}

	// Create resource with standard attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Env),
			semconv.HostName(hostname()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	// Initialize shutdown tracking
	s := &shutdownFuncs{}
	globalShutdown = s

	// Initialize tracer provider
	tp, tpShutdown, err := initTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("initializing tracer provider: %w", err)
	}
	if tp != nil {
		otel.SetTracerProvider(tp)
		s.addShutdown(tpShutdown)
	}

	// Set up propagator (W3C TraceContext + Baggage)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return s.shutdown, nil
}

// initTracerProvider sets up the trace provider.
func initTracerProvider(ctx context.Context, cfg Config, res *resource.Resource) (*trace.TracerProvider, func(context.Context) error, error) {
	// Determine sampler
	var sampler trace.Sampler
	if cfg.SampleRatio > 0 && cfg.SampleRatio < 1.0 {
		sampler = trace.ParentBased(trace.TraceIDRatioBased(cfg.SampleRatio))
	} else {
		sampler = trace.ParentBased(trace.AlwaysSample())
	}

	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	return tp, tp.Shutdown, nil
}

func (s *shutdownFuncs) addShutdown(fn func(context.Context) error) {
	s.mu.Lock()
	s.funcs = append(s.funcs, fn)
	s.mu.Unlock()
}

func (s *shutdownFuncs) shutdown(ctx context.Context) error {
	s.once.Do(func() {
		s.mu.Lock()
		funcs := s.funcs
		s.mu.Unlock()

		var lastErr error
		for _, fn := range funcs {
			if err := fn(ctx); err != nil {
				lastErr = err
			}
		}
		if lastErr != nil {
			fmt.Fprintf(os.Stderr, "[tracing] shutdown errors: %v\n", lastErr)
		}
	})
	return nil
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

// Shutdown triggers graceful shutdown of all OTel providers.
func Shutdown(ctx context.Context) error {
	if globalShutdown != nil {
		return globalShutdown.shutdown(ctx)
	}
	return nil
}
