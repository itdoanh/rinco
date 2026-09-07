"""OpenTelemetry tracing helpers."""
from __future__ import annotations

import os
from typing import Any


def configure_tracing(service_name: str, otlp_endpoint: str | None = None) -> Any:
    """Configure OTel tracing; safe to call when no exporter is configured."""
    try:
        from opentelemetry import trace
        from opentelemetry.sdk.resources import Resource
        from opentelemetry.sdk.trace import TracerProvider
        from opentelemetry.sdk.trace.export import BatchSpanProcessor
        from opentelemetry.semconv.resource import ResourceAttributes
    except Exception:  # pragma: no cover
        return None

    endpoint = otlp_endpoint or os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
    resource = Resource.create(
        {
            ResourceAttributes.SERVICE_NAME: service_name,
            ResourceAttributes.SERVICE_VERSION: os.getenv(
                "LEAD_SCORING_VERSION", "2.0.0"
            ),
        }
    )
    provider = TracerProvider(resource=resource)
    if endpoint:
        try:
            from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import (
                OTLPSpanExporter,
            )

            provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(endpoint=endpoint)))
        except Exception:  # pragma: no cover
            pass
    trace.set_tracer_provider(provider)
    return trace.get_tracer(service_name)


def instrument_fastapi(app: Any) -> None:
    """Best-effort OTel FastAPI instrumentation."""
    try:
        from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

        FastAPIInstrumentor.instrument_app(app)
    except Exception:  # pragma: no cover
        pass