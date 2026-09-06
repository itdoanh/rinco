-- ============================================================
-- RINCO ClickHouse Schema
-- ============================================================

CREATE DATABASE IF NOT EXISTS rinco_logs;

-- ============================================================
-- APPLICATION LOGS
-- ============================================================
CREATE TABLE IF NOT EXISTS rinco_logs.app_logs (
    timestamp DateTime64(3),
    tenant_id UUID,
    service LowCardinality(String),
    level Enum8('debug' = 1, 'info' = 2, 'warn' = 3, 'error' = 4, 'fatal' = 5),
    message String CODEC(ZSTD(3)),
    trace_id String,
    span_id String,
    request_id String,
    user_id UUID,
    caller_file LowCardinality(String),
    caller_line UInt32,
    http_method LowCardinality(String),
    http_route LowCardinality(String),
    http_status UInt16,
    duration_ms UInt32,
    attributes JSON CODEC(ZSTD(3))
) ENGINE = MergeTree
PARTITION BY (toYYYYMM(timestamp), tenant_id)
ORDER BY (tenant_id, service, timestamp)
TTL toDate(timestamp) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- ============================================================
-- AUDIT LOGS
-- ============================================================
CREATE TABLE IF NOT EXISTS rinco_logs.audit_logs (
    timestamp DateTime64(3),
    tenant_id UUID,
    actor_user_id UUID,
    actor_ip IPv4,
    action LowCardinality(String),
    entity_type LowCardinality(String),
    entity_id String,
    before JSON,
    after JSON,
    request_id String,
    trace_id String,
    user_agent String
) ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (tenant_id, timestamp)
TTL toDate(timestamp) + INTERVAL 365 DAY;

-- ============================================================
-- METRICS SAMPLES (long-term storage)
-- ============================================================
CREATE TABLE IF NOT EXISTS rinco_logs.metrics_samples (
    timestamp DateTime64(3),
    tenant_id UUID,
    service LowCardinality(String),
    metric_name LowCardinality(String),
    labels Map(LowCardinality(String), String),
    value Float64
) ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (service, metric_name, timestamp)
TTL toDate(timestamp) + INTERVAL 365 DAY;

-- ============================================================
-- DISTRIBUTED TRACING SPANS
-- ============================================================
CREATE TABLE IF NOT EXISTS rinco_logs.traces_spans (
    timestamp DateTime64(3),
    tenant_id UUID,
    trace_id String,
    span_id String,
    parent_span_id String,
    service LowCardinality(String),
    operation_name LowCardinality(String),
    duration_ms UInt32,
    status LowCardinality(String),
    attributes JSON
) ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (trace_id, timestamp)
TTL toDate(timestamp) + INTERVAL 30 DAY;

-- ============================================================
-- FACEBOOK CAPI EVENTS LOG
-- ============================================================
CREATE TABLE IF NOT EXISTS rinco_logs.capi_events (
    timestamp DateTime64(3),
    tenant_id UUID,
    event_id UUID,
    event_name LowCardinality(String),
    fbclid String,
    fb_event_id String,
    payload_size_bytes UInt32,
    response_status UInt16,
    response_body String,
    duration_ms UInt32,
    error String
) ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (tenant_id, timestamp)
TTL toDate(timestamp) + INTERVAL 90 DAY;

-- ============================================================
-- AGGREGATE VIEWS
-- ============================================================
-- Hourly error counts per service
CREATE MATERIALIZED VIEW IF NOT EXISTS rinco_logs.app_logs_hourly_errors
ENGINE = SummingMergeTree
PARTITION BY (toYYYYMM(hour), tenant_id)
ORDER BY (tenant_id, service, hour, level)
AS SELECT
    toStartOfHour(timestamp) AS hour,
    tenant_id,
    service,
    level,
    count() AS event_count
FROM rinco_logs.app_logs
WHERE level IN ('error', 'fatal', 'warn')
GROUP BY hour, tenant_id, service, level;

-- API request stats
CREATE MATERIALIZED VIEW IF NOT EXISTS rinco_logs.api_request_stats
ENGINE = SummingMergeTree
PARTITION BY (toYYYYMM(hour), tenant_id)
ORDER BY (tenant_id, service, http_route, hour)
AS SELECT
    toStartOfHour(timestamp) AS hour,
    tenant_id,
    service,
    http_route,
    http_status,
    count() AS request_count,
    avg(duration_ms) AS avg_duration_ms,
    quantile(0.95)(duration_ms) AS p95_duration_ms,
    quantile(0.99)(duration_ms) AS p99_duration_ms
FROM rinco_logs.app_logs
WHERE http_route != ''
GROUP BY hour, tenant_id, service, http_route, http_status;
