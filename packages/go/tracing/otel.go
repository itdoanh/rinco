// Package tracing - auto-instrumentation helpers for common libraries.
//
// Placeholder for OTLP exporter wiring.
// Full implementations require:
//   - go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
//   - go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
package tracing

import "context"

// InitHTTPExporter is a no-op placeholder for OTLP/HTTP trace exporter init.
// To enable, install the otlptracehttp package and wire it into tracing.Init.
func InitHTTPExporter(ctx context.Context, cfg Config) error {
	if cfg.OTLPEndpoint == "" {
		return nil
	}
	return nil
}

// InitGRPCExporter is a no-op placeholder for OTLP/gRPC trace exporter init.
// To enable, install the otlptracegrpc package and wire it into tracing.Init.
func InitGRPCExporter(ctx context.Context, cfg Config) error {
	if cfg.OTLPEndpoint == "" {
		return nil
	}
	return nil
}
