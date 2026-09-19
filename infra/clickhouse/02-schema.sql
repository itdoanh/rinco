-- ============================================================
-- Events raw â€” MergeTree, partitioned by month, ordered by
-- (tenant_id, event_time, user_id) for fast per-tenant scans
-- ============================================================

CREATE TABLE IF NOT EXISTS rinco_analytics.events_raw
(
    event_date      Date                 MATERIALIZED toDate(event_time),
    event_time      DateTime64(3),
    event_id        UUID,
    tenant_id       LowCardinality(String),
    session_id      UUID,
    anonymous_id    String,
    user_id         UUID,
    event_name      LowCardinality(String),
    source          LowCardinality(String),
    utm_source      LowCardinality(String),
    utm_medium      LowCardinality(String),
    utm_campaign    LowCardinality(String),
    utm_content     String,
    utm_term        String,
    fbclid          String,
    gclid           String,
    country         LowCardinality(String),
    city            String,
    device_type     LowCardinality(String),
    browser         LowCardinality(String),
    os              LowCardinality(String),
    ip_address      IPv4,
    user_agent      String,
    referrer        String,
    page_url        String,
    properties      JSON,
    INDEX idx_event_name event_name TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_utm       utm_source TYPE bloom_filter(0.01) GRANULARITY 4
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_time, user_id)
TTL event_date + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- ============================================================
-- User sessions â€” derived from events_raw
-- ============================================================

CREATE TABLE IF NOT EXISTS rinco_analytics.user_sessions
(
    session_id      UUID,
    tenant_id       LowCardinality(String),
    user_id         UUID,
    anonymous_id    String,
    started_at      DateTime,
    ended_at        DateTime,
    duration_sec    UInt32,
    page_view_count UInt32,
    event_count     UInt32,
    entry_url       String,
    exit_url        String,
    country         LowCardinality(String),
    device_type     LowCardinality(String),
    browser         LowCardinality(String),
    os              LowCardinality(String),
    utm_source      LowCardinality(String),
    utm_campaign    LowCardinality(String)
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(started_at)
ORDER BY (tenant_id, started_at, session_id)
TTL started_at + INTERVAL 180 DAY
SETTINGS index_granularity = 8192;

-- ============================================================
-- Funnel events / conversions â€” wide denormalised table for
-- product-level conversion analytics
-- ============================================================

CREATE TABLE IF NOT EXISTS rinco_analytics.conversions
(
    event_time      DateTime,
    tenant_id       LowCardinality(String),
    user_id         UUID,
    campaign_id     UUID,
    funnel_id       LowCardinality(String),
    step            LowCardinality(String),     -- e.g. 'view','add_to_cart','checkout','purchase'
    order_id        UUID,
    order_value     Decimal(18, 4),
    currency        LowCardinality(String),
    properties      JSON
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_time)
ORDER BY (tenant_id, funnel_id, event_time, user_id)
TTL event_time + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

-- ============================================================
-- Materialized view â€” events aggregated by hour
-- ============================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS rinco_analytics.events_hourly_mv
ENGINE = SummingMergeTree
PARTITION BY (toYYYYMM(hour), tenant_id)
ORDER BY (tenant_id, event_name, source, hour)
AS SELECT
    toStartOfHour(event_time)              AS hour,
    tenant_id,
    event_name,
    source,
    utm_source,
    utm_campaign,
    device_type,
    country,
    count()                                AS event_count,
    uniqState(user_id)                     AS unique_users,
    uniqState(session_id)                  AS unique_sessions
FROM rinco_analytics.events_raw
GROUP BY hour, tenant_id, event_name, source, utm_source, utm_campaign, device_type, country;

-- ============================================================
-- Materialized view â€” funnel step counts per hour
-- ============================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS rinco_analytics.funnel_hourly_mv
ENGINE = SummingMergeTree
PARTITION BY (toYYYYMM(hour), tenant_id)
ORDER BY (tenant_id, funnel_id, step, hour)
AS SELECT
    toStartOfHour(event_time)              AS hour,
    tenant_id,
    funnel_id,
    step,
    count()                                AS step_count,
    uniqState(user_id)                     AS unique_users,
    sumState(order_value)                  AS total_value
FROM rinco_analytics.conversions
GROUP BY hour, tenant_id, funnel_id, step;

