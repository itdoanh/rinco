-- Migration 0001: notifications
-- Per-user notifications with status tracking, delivery channels, and metadata.
CREATE TABLE IF NOT EXISTS notification.notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    icon TEXT,
    category TEXT,
    priority TEXT NOT NULL DEFAULT 'normal' CHECK (priority IN ('high','normal','low')),
    channels_resolved JSONB NOT NULL DEFAULT '[]'::jsonb,
    data JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'delivered', 'read', 'archived')),
    read_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notif_tenant_user ON notification.notifications(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_notif_user_status ON notification.notifications(user_id, status);
CREATE INDEX IF NOT EXISTS idx_notif_created ON notification.notifications(tenant_id, created_at DESC);

ALTER TABLE notification.notifications ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS notif_tenant ON notification.notifications;
CREATE POLICY notif_tenant ON notification.notifications
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        OR current_setting('app.is_admin', true) = 'true'
    );