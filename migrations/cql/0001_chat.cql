-- ============================================================
-- CQL migration 0001 — chat keyspace + tables
-- ============================================================
-- Idempotent (IF NOT EXISTS). Apply with:
--   cqlsh $SCYLLA_HOST 9042 -f migrations/cql/0001_chat.cql
-- ============================================================

CREATE KEYSPACE IF NOT EXISTS chat
  WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'replication_factor': 3
  }
  AND durable_writes = true;

USE chat;

CREATE TABLE IF NOT EXISTS chat.conversations (
    tenant_id              uuid,
    conversation_id        uuid,
    type                   text,
    title                  text,
    member_ids             set<uuid>,
    last_message_at        timestamp,
    last_message_preview   text,
    created_by             uuid,
    created_at             timestamp,
    updated_at             timestamp,
    PRIMARY KEY (tenant_id, conversation_id)
) WITH CLUSTERING ORDER BY (conversation_id ASC)
  AND default_time_to_live = 0;

CREATE TABLE IF NOT EXISTS chat.messages_by_channel (
    tenant_id      uuid,
    channel_id     uuid,
    msg_id         timeuuid,
    sender_id      uuid,
    msg_type       text,
    body           text,
    metadata       map<text, text>,
    parent_msg_id  uuid,
    edited_at      timestamp,
    deleted_at     timestamp,
    PRIMARY KEY ((tenant_id, channel_id), msg_id)
) WITH CLUSTERING ORDER BY (msg_id DESC)
  AND default_time_to_live = 2592000
  AND compaction = {
    'class': 'TimeWindowCompactionStrategy',
    'compaction_window_size': '1',
    'compaction_window_unit': 'DAYS'
  };

CREATE INDEX IF NOT EXISTS idx_messages_channel_parent
    ON chat.messages_by_channel (parent_msg_id);

CREATE TABLE IF NOT EXISTS chat.messages_by_user (
    tenant_id      uuid,
    user_id        uuid,
    channel_id     uuid,
    msg_id         timeuuid,
    msg_type       text,
    body           text,
    PRIMARY KEY ((tenant_id, user_id), msg_id)
) WITH CLUSTERING ORDER BY (msg_id DESC)
  AND default_time_to_live = 2592000;

CREATE TABLE IF NOT EXISTS chat.reactions (
    tenant_id      uuid,
    channel_id     uuid,
    msg_id         uuid,
    emoji          text,
    user_ids       set<uuid>,
    count          int,
    updated_at     timestamp,
    PRIMARY KEY ((tenant_id, channel_id, msg_id), emoji)
);
