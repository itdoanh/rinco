package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/bridge/opentracing"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/trace"
)

// otlptraceExporterWrapper wraps both HTTP and gRPC OTLP trace exporters.
type otlptraceExporterWrapper struct {
	grpc *otlptracegrpc.Exporter
	http *otlptracehttp.Exporter
}

func (e *otlptraceExporterWrapper) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	if e.grpc != nil {
		return e.grpc.ExportSpans(ctx, spans)
	}
	if e.http != nil {
		return e.http.ExportSpans(ctx, spans)
	}
	return nil
}

func (e *otlptraceExporterWrapper) Shutdown(ctx context.Context) error {
	var err error
	if e.grpc != nil {
		if sErr := e.grpc.Shutdown(ctx); sErr != nil {
			err = sErr
		}
	}
	if e.http != nil {
		if sErr := e.http.Shutdown(ctx); sErr != nil {
			err = sErr
		}
	}
	return err
}

func newOTLPTraceHTTPExporter(ctx context.Context, cfg Config) (*otlptraceExporterWrapper, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.OTLPInsecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	if len(cfg.OTLPHeaders) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.OTLPHeaders))
	}
	exp, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP/HTTP trace exporter: %w", err)
	}
	return &otlptraceExporterWrapper{http: exp}, nil
}

func newOTLPTraceGRPCExporter(ctx context.Context, cfg Config) (*otlptraceExporterWrapper, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.OTLPInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	if len(cfg.OTLPHeaders) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.OTLPHeaders))
	}
	exp, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP/gRPC trace exporter: %w", err)
	}
	return &otlptraceExporterWrapper{grpc: exp}, nil
}

// otlpmetricExporterWrapper wraps both HTTP and gRPC OTLP metric exporters.
type otlpmetricExporterWrapper struct {
	grpc *otlpmetricgrpc.Exporter
	http *otlpmetrichttp.Exporter
}

func newOTLPMetricHTTPExporter(ctx context.Context, cfg Config) (*otlpmetrichttp.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.OTLPInsecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}
	exp, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP/HTTP metric exporter: %w", err)
	}
	return exp, nil
}

func newOTLPMetricGRPCExporter(ctx context.Context, cfg Config) (*otlpmetricgrpc.Exporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.OTLPInsecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	exp, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP/gRPC metric exporter: %w", err)
	}
	return exp, nil
}

// otlplogExporterWrapper wraps both HTTP and gRPC OTLP log exporters.
type otlplogExporterWrapper struct {
	grpc *otlploggrpc.Exporter
	http *otlploghttp.Exporter
}

func (e *otlplogExporterWrapper) Export(ctx context.Context, logs []interface{}) error {
	if e.grpc != nil {
		return e.grpc.Export(ctx, logs)
	}
	if e.http != nil {
		return e.http.Export(ctx, logs)
	}
	return nil
}

func (e *otlplogExporterWrapper) Shutdown(ctx context.Context) error {
	var err error
	if e.grpc != nil {
		if sErr := e.grpc.Shutdown(ctx); sErr != nil {
			err = sErr
		}
	}
	if e.http != nil {
		if sErr := e.http.Shutdown(ctx); sErr != nil {
			err = sErr
		}
	}
	return err
}

func newOTLPLogHTTPExporter(ctx context.Context, cfg Config) (*otlplogExporterWrapper, error) {
	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.OTLPInsecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}
	exp, err := otlploghttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP/HTTP log exporter: %w", err)
	}
	return &otlplogExporterWrapper{http: exp}, nil
}

func newOTLPLogGRPCExporter(ctx context.Context, cfg Config) (*otlplogExporterWrapper, error) {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.OTLPInsecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	exp, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP/gRPC log exporter: %w", err)
	}
	return &otlplogExporterWrapper{grpc: exp}, nil
}

// InitOTelBridge initializes the OpenTracing bridge for compatibility with
// libraries that still use opentracing.
func InitOTelBridge() {
	// Register OpenTracing bridge
	tracerProvider := otel.GetTracerProvider()
	_, _ = opentracing.NewTracerPair(tracerProvider)
}
