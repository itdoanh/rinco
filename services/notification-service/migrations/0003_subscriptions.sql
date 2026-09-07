-- Migration 0003: push_subscriptions + fcm_subscriptions
-- Web Push (VAPID) and FCM device registration per user.
CREATE TABLE IF NOT EXISTS notification.push_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    endpoint TEXT NOT NULL UNIQUE,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    keys JSONB NOT NULL DEFAULT '{}'::jsonb,
    vapid_public_key TEXT,
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_push_user ON notification.push_subscriptions(user_id);

CREATE TABLE IF NOT EXISTS notification.fcm_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    device_token TEXT NOT NULL UNIQUE,
    platform TEXT NOT NULL CHECK (platform IN ('android','ios','web')),
    app_version TEXT,
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fcm_user ON notification.fcm_subscriptions(user_id);