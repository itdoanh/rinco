-- ============================================================
-- Presence — write-time 60s TTL (auto-expire when heartbeat stops)
-- ============================================================

USE chat;

-- Per-user presence state per device.
CREATE TABLE IF NOT EXISTS chat.presence_by_user (
    tenant_id        uuid,
    user_id          uuid,
    device_id        text,
    status           text,           -- 'online','away','dnd','offline'
    last_heartbeat   timestamp,
    location         text,           -- optional geo
    PRIMARY KEY ((tenant_id, user_id), device_id)
) WITH default_time_to_live = 60
  AND comment = 'Per-device presence with 60s TTL.';

-- Aggregate online user set per channel (computed by chat-engine).
CREATE TABLE IF NOT EXISTS chat.channel_presence (
    tenant_id      uuid,
    channel_id     uuid,
    online_users   set<uuid>,
    last_update    timestamp,
    PRIMARY KEY (tenant_id, channel_id)
) WITH comment = 'Channel-wide online user set, kept in sync via chat-engine.';
