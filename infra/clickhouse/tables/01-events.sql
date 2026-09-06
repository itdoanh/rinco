-- ============================================================
-- Events raw — MergeTree, partitioned by month, ordered by
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
