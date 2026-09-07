//! OTel / Prometheus / tracing setup for the SFU.

use opentelemetry::{global, trace::TracerProvider as _, KeyValue};
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{
    trace::{Config as TraceConfig, Sampler},
    Resource,
};
use tracing_subscriber::{layer::SubscriberExt, EnvFilter};

/// Telemetry guard.
pub struct TelemetryGuard {
    _prom: metrics_exporter_prometheus::PrometheusHandle,
}

/// Initialize tracing + OTel + Prometheus recorder.
pub fn init(service_name: &str, otlp_endpoint: &str) -> anyhow::Result<TelemetryGuard> {
    global::set_text_map_propagator(opentelemetry_sdk::propagation::TraceContextPropagator::new());

    let exporter = opentelemetry_otlp::SpanExporter::builder()
        .with_tonic()
        .with_endpoint(otlp_endpoint)
        .build()?;

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

    let env_filter = EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info,webrtc_sfu=debug"));

    let fmt_layer = tracing_subscriber::fmt::layer().json();

    tracing::subscriber::set_global_default(
        tracing_subscriber::Registry::default()
            .with(env_filter)
            .with(fmt_layer)
            .with(otel_layer),
    )?;

    let prom = metrics_exporter_prometheus::PrometheusBuilder::new()
        .install_recorder()?;
    Ok(TelemetryGuard { _prom: prom })
}

/// Render Prometheus metrics body.
pub fn render_metrics() -> String {
    metrics_exporter_prometheus::PrometheusBuilder::new()
        .install_recorder()
        .map(|h| h.render())
        .unwrap_or_default()
}
