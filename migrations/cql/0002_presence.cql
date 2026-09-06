-- ============================================================
-- CQL migration 0002 — presence (60s TTL)
-- ============================================================

USE chat;

CREATE TABLE IF NOT EXISTS chat.presence_by_user (
    tenant_id        uuid,
    user_id          uuid,
    device_id        text,
    status           text,
    last_heartbeat   timestamp,
    location         text,
    PRIMARY KEY ((tenant_id, user_id), device_id)
) WITH default_time_to_live = 60;

CREATE TABLE IF NOT EXISTS chat.channel_presence (
    tenant_id      uuid,
    channel_id     uuid,
    online_users   set<uuid>,
    last_update    timestamp,
    PRIMARY KEY (tenant_id, channel_id)
);
