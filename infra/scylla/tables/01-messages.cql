-- ============================================================
-- Chat messages — recent first, 30-day TTL
-- ============================================================

USE chat;

-- Conversations between two (or more) users.
CREATE TABLE IF NOT EXISTS chat.conversations (
    tenant_id              uuid,
    conversation_id        uuid,
    type                   text,         -- 'direct','group','channel'
    title                  text,
    member_ids             set<uuid>,
    last_message_at        timestamp,
    last_message_preview   text,
    created_by             uuid,
    created_at             timestamp,
    updated_at             timestamp,
    PRIMARY KEY (tenant_id, conversation_id)
) WITH CLUSTERING ORDER BY (conversation_id ASC)
  AND default_time_to_live = 0
  AND comment = 'Conversations metadata (tenant scoped)';

-- Messages — partition by channel, cluster by ts DESC for tail reads.
CREATE TABLE IF NOT EXISTS chat.messages_by_channel (
    tenant_id      uuid,
    channel_id     uuid,
    msg_id         timeuuid,
    sender_id      uuid,
    msg_type       text,             -- 'text','image','file','system','call'
    body           text,
    metadata       map<text, text>,
    attachments    list<uuid>,      -- IDs into MinIO
    reactions      map<text, int>,  -- emoji -> count
    parent_msg_id  uuid,            -- thread / reply-to
    edited_at      timestamp,
    deleted_at     timestamp,
    PRIMARY KEY ((tenant_id, channel_id), msg_id)
) WITH CLUSTERING ORDER BY (msg_id DESC)
  AND default_time_to_live = 2592000
  AND compaction = {
    'class': 'TimeWindowCompactionStrategy',
    'compaction_window_size': '1',
    'compaction_window_unit': 'DAYS'
  }
  AND comment = 'Channel message stream. TTL 30 days.';

CREATE INDEX IF NOT EXISTS idx_messages_by_channel_parent
    ON chat.messages_by_channel (parent_msg_id);

-- Inbox: messages grouped by sender (for "messages I sent" queries).
CREATE TABLE IF NOT EXISTS chat.messages_by_user (
    tenant_id      uuid,
    user_id        uuid,
    channel_id     uuid,
    msg_id         timeuuid,
    msg_type       text,
    body           text,
    PRIMARY KEY ((tenant_id, user_id), msg_id)
) WITH CLUSTERING ORDER BY (msg_id DESC)
  AND default_time_to_live = 2592000
  AND comment = 'Sent messages per user, TTL 30d.';

-- Reactions aggregate (per channel + emoji).
CREATE TABLE IF NOT EXISTS chat.reactions (
    tenant_id      uuid,
    channel_id     uuid,
    msg_id         uuid,
    emoji          text,
    user_ids       set<uuid>,
    count          int,
    updated_at     timestamp,
    PRIMARY KEY ((tenant_id, channel_id, msg_id), emoji)
) WITH CLUSTERING ORDER BY (emoji ASC)
  AND comment = 'Per message / per emoji reactions.';

-- Read receipts (per user per channel).
CREATE TABLE IF NOT EXISTS chat.read_receipts (
    tenant_id             uuid,
    channel_id            uuid,
    user_id               uuid,
    last_read_msg_id      uuid,
    last_read_at          timestamp,
    PRIMARY KEY ((tenant_id, channel_id), user_id)
) WITH comment = 'Per-channel read-cursor.';

-- Typing indicators — short TTL (handled by Scylla TTL).
CREATE TABLE IF NOT EXISTS chat.typing_indicators (
    tenant_id     uuid,
    channel_id    uuid,
    user_id       uuid,
    started_at    timestamp,
    PRIMARY KEY ((tenant_id, channel_id), user_id)
) WITH default_time_to_live = 10
  AND comment = 'Typing indicators. Auto-expire 10s.';
