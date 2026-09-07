//! Tracing / OTel / Prometheus initialization.

use opentelemetry::{global, trace::TracerProvider as _, KeyValue};
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{
    trace::{Config as TraceConfig, Sampler},
    Resource,
};
use tracing_subscriber::{layer::SubscriberExt, EnvFilter};

use crate::error::ChatResult;

/// Telemetry guard returned from [`init`]. Drop to flush.
pub struct TelemetryGuard {
    _prometheus: metrics_exporter_prometheus::PrometheusHandle,
}

/// Build a `tracing` subscriber that writes JSON to stdout and forwards spans
/// to the configured OTLP collector. Also installs a Prometheus recorder.
pub fn init(service_name: &str, otlp_endpoint: &str) -> ChatResult<TelemetryGuard> {
    global::set_text_map_propagator(opentelemetry_sdk::propagation::TraceContextPropagator::new());

    let exporter = opentelemetry_otlp::SpanExporter::builder()
        .with_tonic()
        .with_endpoint(otlp_endpoint)
        .build()
        .map_err(|e| crate::error::ChatError::Other(anyhow::anyhow!(e)))?;

    let provider = opentelemetry_sdk::trace::TracerProvider::builder()
        .with_batch_exporter(exporter, opentelemetry_sdk::runtime::Tokio)
        .with_config(
            TraceConfig::default()
                .with_sampler(Sampler::ParentBased(Box::new(Sampler::AlwaysOn)))
                .with_resource(Resource::new(vec![KeyValue::new(
                    "service.name",
                    service_name.to_string(),
                )])),
        )
        .build();

    let tracer = provider.tracer(service_name.to_string());
    let otel_layer = tracing_opentelemetry::layer().with_tracer(tracer);

    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info,chat_engine=debug"));

    let fmt_layer = tracing_subscriber::fmt::layer()
        .json()
        .with_current_span(true)
        .with_span_list(false);

    tracing::subscriber::set_global_default(
        tracing_subscriber::Registry::default()
            .with(env_filter)
            .with(fmt_layer)
            .with(otel_layer),
    )
    .map_err(|e| crate::error::ChatError::Other(anyhow::anyhow!(e)))?;

    let prom_handle = metrics_exporter_prometheus::PrometheusBuilder::new()
        .install_recorder()
        .map_err(|e| crate::error::ChatError::Other(anyhow::anyhow!(e)))?;

    Ok(TelemetryGuard { _prometheus: prom_handle })
}

/// Returns the current Prometheus scrape body.
pub fn render_metrics() -> String {
    metrics_exporter_prometheus::PrometheusBuilder::new()
        .install_recorder()
        .map(|h| h.render())
        .unwrap_or_default()
}
