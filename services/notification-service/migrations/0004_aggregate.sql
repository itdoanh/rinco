-- Migration 0004: notification_delivery_logs + daily_aggregates
-- Per-notification delivery log and daily aggregate counters per (tenant, type, channel).
CREATE TABLE IF NOT EXISTS notification.notification_delivery_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notification.notifications(id) ON DELETE CASCADE,
    channel TEXT NOT NULL
        CHECK (channel IN ('in_app','email','sms','push','slack','discord','telegram','webhook')),
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','sent','delivered','failed')),
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    error_msg TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dl_notif ON notification.notification_delivery_logs(notification_id);

CREATE TABLE IF NOT EXISTS notification.daily_aggregates (
    date DATE NOT NULL,
    tenant_id UUID NOT NULL,
    type TEXT NOT NULL,
    channel TEXT NOT NULL,
    count_sent BIGINT NOT NULL DEFAULT 0,
    count_delivered BIGINT NOT NULL DEFAULT 0,
    count_failed BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (date, tenant_id, type, channel)
);

CREATE INDEX IF NOT EXISTS idx_agg_tenant_date ON notification.daily_aggregates(tenant_id, date DESC);