// Package tracing - OpenTelemetry initialization and configuration.
//
// Provides:
//   - OTLP/HTTP and OTLP/gRPC exporters
//   - TracerProvider, MeterProvider, and LoggerProvider setup
//   - Resource attributes (service.name, version, env, host)
//   - Auto-instrumentation helpers for popular libraries
//   - W3C TraceContext and B3 propagation
//
// Usage:
//
//	cfg := tracing.Config{
//	    ServiceName:    "auth-service",
//	    ServiceVersion: "1.0.0",
//	    Env:            "production",
//	    OTLPEndpoint:   "otel-collector:4317",
//	    OTLPProtocol:   "grpc", // or "http"
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
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
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
	OTLPEndpoint string // e.g., "otel-collector:4317" or "otel-collector:4318"
	OTLPProtocol string // "grpc" (default) or "http"
	OTLPHeaders  map[string]string
	OTLPInsecure bool

	// Optional: Use Prometheus exporter for metrics
	Prometheus bool

	// Sampler configuration
	SampleRatio float64 // 0.0 to 1.0, default 1.0 (always sample)
}

// shutdownFuncs holds all shutdown functions for cleanup.
type shutdownFuncs struct {
	mu     sync.Mutex
	funcs  []func(context.Context) error
	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc
}

var globalShutdown *shutdownFuncs

// Init initializes OpenTelemetry with the given configuration.
// Returns a shutdown function that should be called during graceful shutdown.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if cfg.OTLPEndpoint == "" || cfg.Env == "development" {
		// In development mode, just use noop providers
		return func(context.Context) error { return nil }, nil
	}

	// Create resource with standard attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Env),
			semconv.HostName(hostname()),
			semconv.TelemetrySDKVersion("1.27.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	// Initialize shutdown tracking
	s := &shutdownFuncs{}
	s.ctx, s.cancel = context.WithCancel(context.Background())
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

	// Initialize meter provider
	mp, mpShutdown, err := initMeterProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("initializing meter provider: %w", err)
	}
	if mp != nil {
		otel.SetMeterProvider(mp)
		s.addShutdown(mpShutdown)
	}

	// Initialize logger provider
	lp, lpShutdown, err := initLoggerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("initializing logger provider: %w", err)
	}
	if lp != nil {
		otel.SetLoggerProvider(lp)
		s.addShutdown(lpShutdown)
	}

	// Set up propagator (W3C TraceContext + Baggage)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return s.shutdown, nil
}

// initTracerProvider sets up the trace provider with OTLP exporter.
func initTracerProvider(ctx context.Context, cfg Config, res *resource.Resource) (*trace.TracerProvider, func(context.Context) error, error) {
	var exp *otlptraceExporterWrapper
	var err error

	if cfg.OTLPProtocol == "http" {
		exp, err = newOTLPTraceHTTPExporter(ctx, cfg)
	} else {
		exp, err = newOTLPTraceGRPCExporter(ctx, cfg)
	}
	if err != nil {
		return nil, nil, err
	}

	// Determine sampler
	var sampler trace.Sampler
	if cfg.SampleRatio > 0 && cfg.SampleRatio < 1.0 {
		sampler = trace.ParentBased(trace.TraceIDRatioBased(cfg.SampleRatio))
	} else {
		sampler = trace.ParentBased(trace.AlwaysSample())
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp, // batch span processor
			trace.WithBatchTimeout(5*time.Second),
			trace.WithMaxExportBatchSize(512),
		),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	return tp, tp.Shutdown, nil
}

// initMeterProvider sets up the meter provider with OTLP or Prometheus exporter.
func initMeterProvider(ctx context.Context, cfg Config, res *resource.Resource) (*metric.MeterProvider, func(context.Context) error, error) {
	var exp metric.Exporter
	var err error

	if cfg.Prometheus {
		// Prometheus exporter handles its own server
		exp, err = prometheus.New()
		if err != nil {
			return nil, nil, fmt.Errorf("creating prometheus exporter: %w", err)
		}
	} else if cfg.OTLPEndpoint != "" {
		if cfg.OTLPProtocol == "http" {
			exp, err = newOTLPMetricHTTPExporter(ctx, cfg)
		} else {
			exp, err = newOTLPMetricGRPCExporter(ctx, cfg)
		}
		if err != nil {
			return nil, nil, err
		}
	}

	if exp == nil {
		return nil, nil, nil
	}

	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exp,
			metric.WithInterval(30*time.Second),
			metric.WithTimeout(10*time.Second),
		)),
	)

	return mp, mp.Shutdown, nil
}

// initLoggerProvider sets up the logger provider with OTLP exporter.
func initLoggerProvider(ctx context.Context, cfg Config, res *resource.Resource) (*log.LoggerProvider, func(context.Context) error, error) {
	if cfg.OTLPEndpoint == "" {
		return nil, nil, nil
	}

	var exp *otlplogExporterWrapper
	var err error

	if cfg.OTLPProtocol == "http" {
		exp, err = newOTLPLogHTTPExporter(ctx, cfg)
	} else {
		exp, err = newOTLPLogGRPCExporter(ctx, cfg)
	}
	if err != nil {
		return nil, nil, err
	}

	lp := log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(log.NewBatchProcessor(exp)),
	)

	return lp, lp.Shutdown, nil
}

func (s *shutdownFuncs) addShutdown(fn func(context.Context) error) {
	s.mu.Lock()
	s.funcs = append(s.funcs, fn)
	s.mu.Unlock()
}

func (s *shutdownFuncs) shutdown(ctx context.Context) error {
	s.once.Do(func() {
		s.cancel()
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

// Tracer returns a tracer with the given name.
func Tracer(name string) interface{ TracerName() string } {
	return otel.Tracer(name)
}
