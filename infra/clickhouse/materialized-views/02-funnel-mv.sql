-- ============================================================
-- Materialized view — funnel step counts per hour
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
