-- ============================================================
-- Funnel events / conversions — wide denormalised table for
-- product-level conversion analytics
-- ============================================================

CREATE TABLE IF NOT EXISTS rinco_analytics.conversions
(
    event_time      DateTime64(3),
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
