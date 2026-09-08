-- Analytics Service: ClickHouse schema
-- Run against ClickHouse

-- Events table (MergeTree, partitioned by month)
CREATE TABLE IF NOT EXISTS analytics.events
(
    id              UUID,
    tenant_id       UUID,
    user_id         String,
    session_id      String,
    event_type      LowCardinality(String),
    source          LowCardinality(String),
    campaign        String,
    url             String,
    referrer        String,
    user_agent      String,
    country         LowCardinality(String),
    device          LowCardinality(String),
    browser         LowCardinality(String),
    os              LowCardinality(String),
    value           Float64 DEFAULT 0,
    props           String DEFAULT '{}',
    timestamp       DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (tenant_id, event_type, timestamp)
TTL timestamp + INTERVAL 365 DAY;
-- SETTINGS index_granularity = 8192;

-- Materialized view for real-time aggregations (per hour)
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.hourly_events
ENGINE = SummingMergeTree()
ORDER BY (tenant_id, event_type, hour)
AS SELECT
    tenant_id,
    event_type,
    source,
    device,
    toStartOfHour(timestamp) as hour,
    count() as cnt,
    sum(value) as total_value,
    uniq(user_id) as unique_users
FROM analytics.events
GROUP BY tenant_id, event_type, source, device, hour;

-- Dashboard summary table (updated by scheduled jobs)
CREATE TABLE IF NOT EXISTS analytics.daily_summary
(
    tenant_id       UUID,
    date            Date,
    page_views      Int64 DEFAULT 0,
    unique_visitors Int64 DEFAULT 0,
    conversions     Int64 DEFAULT 0,
    conv_rate       Float64 DEFAULT 0,
    bounce_rate     Float64 DEFAULT 0,
    avg_session_sec Float64 DEFAULT 0,
    updated_at      DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (tenant_id, date);
