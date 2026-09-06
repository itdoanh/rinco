-- ============================================================
-- ClickHouse migration 0001 — analytics events_raw
-- ============================================================
-- Apply with:
--   clickhouse-client --multiquery < migrations/ch/0001_events.sql
-- ============================================================

CREATE DATABASE IF NOT EXISTS rinco_analytics;

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
    properties      JSON
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_time, user_id)
TTL event_date + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Per-tenant row-level isolation example (production)
CREATE ROW POLICY IF NOT EXISTS tenant_rp ON rinco_analytics.events_raw
  USING tenant_id = currentSetting('tenant_id') TO default;
