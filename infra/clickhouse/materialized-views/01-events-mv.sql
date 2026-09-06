-- ============================================================
-- Materialized view — events aggregated by hour
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
