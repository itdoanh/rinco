-- ============================================================
-- User sessions — derived from events_raw
-- ============================================================

CREATE TABLE IF NOT EXISTS rinco_analytics.user_sessions
(
    session_id      UUID,
    tenant_id       LowCardinality(String),
    user_id         UUID,
    anonymous_id    String,
    started_at      DateTime64(3),
    ended_at        DateTime64(3),
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
