-- ============================================================
-- RINCO ScyllaDB Comprehensive Demo Seed
-- Loop 202 — Chat Engine + Events
-- ============================================================
-- Apply: cqlsh localhost 9042 -f seed/external/scylla/seed.sql
-- Uses idempotent statements where possible
-- ============================================================

-- ============================================================
-- SECTION 1: KEYSPACES
-- ============================================================
CREATE KEYSPACE IF NOT EXISTS rinco_chat
  WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'datacenter1': 1
  }
  AND durable_writes = true;

CREATE KEYSPACE IF NOT EXISTS rinco_events
  WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'datacenter1': 1
  }
  AND durable_writes = true;

CREATE KEYSPACE IF NOT EXISTS rinco_webrtc
  WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'datacenter1': 1
  }
  AND durable_writes = true;

-- ============================================================
-- SECTION 2: CHAT TABLES (rinco_chat keyspace)
-- ============================================================
USE rinco_chat;

-- Conversations (group metadata)
CREATE TABLE IF NOT EXISTS rinco_chat.conversations (
    tenant_id            uuid,
    conversation_id      uuid,
    type                text,  -- 'direct','group','channel'
    title               text,
    description          text,
    member_ids          set<uuid>,
    admin_ids           set<uuid>,
    created_by          uuid,
    created_at         timestamp,
    updated_at          timestamp,
    last_message_at     timestamp,
    last_message_preview text,
    avatar_url          text,
    metadata            map<text, text>,
    PRIMARY KEY ((tenant_id), conversation_id)
) WITH CLUSTERING ORDER BY (conversation_id ASC)
  AND default_time_to_live = 0;

CREATE TABLE IF NOT EXISTS rinco_chat.messages_by_channel (
    tenant_id      uuid,
    channel_id     uuid,
    msg_id         timeuuid,
    sender_id      uuid,
    sender_name    text,
    msg_type       text,  -- 'text','image','file','system','call','reaction'
    body           text,
    metadata       map<text, text>,
    attachments    list<uuid>,
    reactions      map<text, int>,  -- emoji -> count
    parent_msg_id  uuid,
    edited_at      timestamp,
    deleted_at     timestamp,
    created_at     timestamp,
    PRIMARY KEY ((tenant_id, channel_id), msg_id)
) WITH CLUSTERING ORDER BY (msg_id DESC)
  AND default_time_to_live = 2592000  -- 30 days
  AND compaction = {
    'class': 'TimeWindowCompactionStrategy',
    'compaction_window_size': '1',
    'compaction_window_unit': 'DAYS'
  };

CREATE INDEX IF NOT EXISTS idx_messages_by_channel_parent
    ON rinco_chat.messages_by_channel (parent_msg_id);

CREATE TABLE IF NOT EXISTS rinco_chat.messages_by_user (
    tenant_id      uuid,
    user_id        uuid,
    channel_id     uuid,
    msg_id         timeuuid,
    sender_id      uuid,
    msg_type       text,
    body           text,
    created_at     timestamp,
    PRIMARY KEY ((tenant_id, user_id), msg_id)
) WITH CLUSTERING ORDER BY (msg_id DESC)
  AND default_time_to_live = 2592000;

CREATE TABLE IF NOT EXISTS rinco_chat.reactions (
    tenant_id      uuid,
    channel_id     uuid,
    msg_id         uuid,
    emoji          text,
    user_ids       set<uuid>,
    count          int,
    updated_at     timestamp,
    PRIMARY KEY ((tenant_id, channel_id, msg_id), emoji)
) WITH CLUSTERING ORDER BY (emoji ASC);

CREATE TABLE IF NOT EXISTS rinco_chat.read_receipts (
    tenant_id            uuid,
    channel_id           uuid,
    user_id              uuid,
    last_read_msg_id     uuid,
    last_read_at         timestamp,
    unread_count         int,
    PRIMARY KEY ((tenant_id, channel_id), user_id)
);

CREATE TABLE IF NOT EXISTS rinco_chat.typing_indicators (
    tenant_id     uuid,
    channel_id    uuid,
    user_id       uuid,
    started_at    timestamp,
    PRIMARY KEY ((tenant_id, channel_id), user_id)
) WITH default_time_to_live = 10;  -- Auto-expire after 10s

CREATE TABLE IF NOT EXISTS rinco_chat.channel_members (
    tenant_id     uuid,
    channel_id    uuid,
    user_id       uuid,
    joined_at     timestamp,
    last_read_at  timestamp,
    unread_count  int,
    muted         boolean,
    PRIMARY KEY ((tenant_id, channel_id), user_id)
);

-- ============================================================
-- SECTION 3: EVENTS TABLES (rinco_events keyspace)
-- ============================================================
USE rinco_events;

CREATE TABLE IF NOT EXISTS rinco_events.event_log (
    tenant_id         text,
    event_date        date,
    event_id          uuid,
    event_time        timestamp,
    event_name        text,   -- 'lead.created','deal.won',...
    actor_user_id     uuid,
    actor_type        text,   -- 'user','system','integration'
    entity_type       text,   -- 'lead','deal','user',...
    entity_id         uuid,
    correlation_id    uuid,
    payload           text,   -- JSON string
    source_service    text,
    metadata          map<text, text>,
    PRIMARY KEY ((tenant_id, event_date), event_time, event_id)
) WITH CLUSTERING ORDER BY (event_time DESC, event_id DESC)
  AND default_time_to_live = 31536000  -- 365 days
  AND compaction = {
    'class': 'TimeWindowCompactionStrategy',
    'compaction_window_size': '1',
    'compaction_window_unit': 'DAYS'
  };

CREATE TABLE IF NOT EXISTS rinco_events.landing_submissions (
    tenant_id        text,
    submit_date      date,
    event_id         uuid,
    event_time       timestamp,
    full_name        text,
    phone            text,
    email            text,
    utm              map<text, text>,
    fbclid           text,
    gclid            text,
    ttclid           text,
    idempotency_key  text,
    raw_json         text,
    processed        boolean,
    lead_id          uuid,
    PRIMARY KEY ((tenant_id, submit_date), event_time, event_id)
) WITH CLUSTERING ORDER BY (event_time DESC, event_id DESC)
  AND default_time_to_live = 7776000;  -- 90 days

CREATE TABLE IF NOT EXISTS rinco_events.webhook_deliveries (
    tenant_id        text,
    delivery_date    date,
    delivery_id      uuid,
    webhook_id       uuid,
    target_url       text,
    event_name       text,
    request_body     text,
    response_status  int,
    response_body    text,
    attempt_count    int,
    delivered_at     timestamp,
    error            text,
    PRIMARY KEY ((tenant_id, delivery_date), delivered_at, delivery_id)
) WITH CLUSTERING ORDER BY (delivered_at DESC, delivery_id DESC);

-- ============================================================
-- SECTION 4: SEED DATA - CONVERSATIONS
-- ============================================================

-- Apex Fintech conversations
INSERT INTO rinco_chat.conversations
  (tenant_id, conversation_id, type, title, description, member_ids, created_by, created_at, updated_at, last_message_at, last_message_preview)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000001'), 'group', 'Sales Team', 'Sales department channel',
   {uuid('a0000001-0000-0000-0000-000000000001'), uuid('a0000001-0000-0000-0000-000000000002'), uuid('a0000001-0000-0000-0000-000000000010'), uuid('a0000001-0000-0000-0000-000000000011')},
   uuid('a0000001-0000-0000-0000-000000000001'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Welcome to Sales Team channel!');

INSERT INTO rinco_chat.conversations
  (tenant_id, conversation_id, type, title, description, member_ids, created_by, created_at, updated_at, last_message_at, last_message_preview)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000002'), 'group', 'Support Team', 'Customer support channel',
   {uuid('a0000001-0000-0000-0000-000000000001'), uuid('a0000001-0000-0000-0000-000000000003'), uuid('a0000001-0000-0000-0000-000000000020'), uuid('a0000001-0000-0000-0000-000000000021')},
   uuid('a0000001-0000-0000-0000-000000000003'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'New support ticket assigned');

INSERT INTO rinco_chat.conversations
  (tenant_id, conversation_id, type, title, member_ids, created_by, created_at, updated_at, last_message_at, last_message_preview)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000003'), 'group', 'Marketing Team',
   {uuid('a0000001-0000-0000-0000-000000000001'), uuid('a0000001-0000-0000-0000-000000000004'), uuid('a0000001-0000-0000-0000-000000000030'), uuid('a0000001-0000-0000-0000-000000000031')},
   uuid('a0000001-0000-0000-0000-000000000004'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'New campaign launched!');

-- HCT Consulting conversations
INSERT INTO rinco_chat.conversations
  (tenant_id, conversation_id, type, title, description, member_ids, created_by, created_at, updated_at, last_message_at, last_message_preview)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000001'), 'group', 'BDS Consultants', 'Real estate sales team',
   {uuid('a0000002-0000-0000-0000-000000000001'), uuid('a0000002-0000-0000-0000-000000000002'), uuid('a0000002-0000-0000-0000-000000000010'), uuid('a0000002-0000-0000-0000-000000000011')},
   uuid('a0000002-0000-0000-0000-000000000001'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Viewing trip scheduled for Saturday');

INSERT INTO rinco_chat.conversations
  (tenant_id, conversation_id, type, title, member_ids, created_by, created_at, updated_at, last_message_at, last_message_preview)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000002'), 'group', 'Investment Team',
   {uuid('a0000002-0000-0000-0000-000000000001'), uuid('a0000002-0000-0000-0000-000000000003'), uuid('a0000002-0000-0000-0000-000000000020'), uuid('a0000002-0000-0000-0000-000000000021')},
   uuid('a0000002-0000-0000-0000-000000000003'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'New deal analysis ready');

-- Demo Company conversations
INSERT INTO rinco_chat.conversations
  (tenant_id, conversation_id, type, title, member_ids, created_by, created_at, updated_at, last_message_at, last_message_preview)
VALUES
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000001'), 'group', 'Demo General',
   {uuid('a0000003-0000-0000-0000-000000000001'), uuid('a0000003-0000-0000-0000-000000000002'), uuid('a0000003-0000-0000-0000-000000000010')},
   uuid('a0000003-0000-0000-0000-000000000001'), toTimestamp(now()), toTimestamp(now()), toTimestamp(now()), 'Welcome to RINCO Demo!');

-- ============================================================
-- SECTION 5: SEED DATA - MESSAGES (recent messages per channel)
-- ============================================================

-- Apex Sales Team messages
INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, created_at)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000001'), now(), uuid('a0000001-0000-0000-0000-000000000002'), 'Phạm Thị Mai', 'text', 'Chào cả team! Có lead mới từ Facebook Ads, score 85', toTimestamp(now() - 3600));

INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, created_at)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000001'), now(), uuid('a0000001-0000-0000-0000-000000000010'), 'Nguyễn Thị Lan', 'text', 'Đã tiếp nhận, sẽ follow-up trong 30 phút', toTimestamp(now() - 1800));

INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, created_at)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000001'), now(), uuid('a0000001-0000-0000-0000-000000000002'), 'Phạm Thị Mai', 'text', 'Vừa chốt deal 500 triệu từ khách hàng FPT! 🎉', toTimestamp(now() - 900));

INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, created_at)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000001'), now(), uuid('a0000001-0000-0000-0000-000000000011'), 'Hoàng Minh Tuấn', 'text', 'Chúc mừng team! Mình cũng có 2 qualified leads mới', toTimestamp(now() - 600));

INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, created_at)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000001'), uuid('11111111-1111-1111-1111-000000000001'), now(), uuid('a0000001-0000-0000-0000-000000000001'), 'Trần Minh Quân', 'text', 'Tuyệt vời! Keep up the good work team 🚀', toTimestamp(now() - 300));

-- HCT BDS Consultants messages
INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, created_at)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000001'), now(), uuid('a0000002-0000-0000-0000-000000000002'), 'Tạ Kim Phượng', 'text', 'Tuần này có viewing Vinhomes Grand Park thứ 7, ai muốn đi thì đăng ký nhé', toTimestamp(now() - 7200));

INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, created_at)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000001'), now(), uuid('a0000002-0000-0000-0000-000000000010'), 'Bùi Mạnh Hưng', 'text', 'Mình đăng ký 1 slot! Có khách hàng muốn xem căn 2PN', toTimestamp(now() - 3600));

INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, created_at)
VALUES
  (uuid('aaaaaaaa-0000-0000-0000-000000000002'), uuid('22222222-2222-2222-2222-000000000001'), now(), uuid('a0000002-0000-0000-0000-000000000011'), 'Tống Diệu Linh', 'text', 'Mình cũng đăng ký, có 2 khách hàng quan tâm shophouse', toTimestamp(now() - 1800));

-- Demo General messages
INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, created_at)
VALUES
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000001'), now(), uuid('a0000003-0000-0000-0000-000000000001'), 'Demo Admin', 'text', 'Welcome to RINCO! Explore the demo features here.', toTimestamp(now() - 1800));

INSERT INTO rinco_chat.messages_by_channel
  (tenant_id, channel_id, msg_id, sender_id, sender_name, msg_type, body, created_at)
VALUES
  (uuid('bbbbbbbb-0000-0000-0000-000000000003'), uuid('33333333-3333-3333-3333-000000000001'), now(), uuid('a0000003-0000-0000-0000-000000000002'), 'Lê Minh Châu', 'text', 'Great platform! The lead scoring feature is amazing', toTimestamp(now() - 900));

-- ============================================================
-- SECTION 6: SEED DATA - EVENTS
-- ============================================================
USE rinco_events;

-- Apex Fintech events
INSERT INTO rinco_events.event_log
  (tenant_id, event_date, event_id, event_time, event_name, actor_user_id, actor_type, entity_type, entity_id, correlation_id, payload, source_service)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now()), uuid('e1111111-1111-1111-1111-000000000001'), toTimestamp(now() - 86400), 'lead.created', uuid('a0000001-0000-0000-0000-000000000010'), 'user', 'lead', uuid('l1111111-1111-1111-1111-000000000001'), uuid('c1111111-1111-1111-1111-000000000001'), '{"source":"facebook","score":85}', 'lead-service');

INSERT INTO rinco_events.event_log
  (tenant_id, event_date, event_id, event_time, event_name, actor_user_id, actor_type, entity_type, entity_id, correlation_id, payload, source_service)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now()), uuid('e1111111-1111-1111-1111-000000000002'), toTimestamp(now() - 43200), 'lead.qualified', uuid('a0000001-0000-0000-0000-000000000002'), 'user', 'lead', uuid('l1111111-1111-1111-1111-000000000001'), uuid('c1111111-1111-1111-1111-000000000001'), '{"score":92}', 'ai-scoring');

INSERT INTO rinco_events.event_log
  (tenant_id, event_date, event_id, event_time, event_name, actor_user_id, actor_type, entity_type, entity_id, correlation_id, payload, source_service)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now()), uuid('e1111111-1111-1111-1111-000000000003'), toTimestamp(now() - 7200), 'deal.won', uuid('a0000001-0000-0000-0000-000000000002'), 'user', 'deal', uuid('d1111111-1111-1111-1111-000000000001'), uuid('c1111111-1111-1111-1111-000000000002'), '{"value":500000000,"currency":"VND"}', 'crm-service');

INSERT INTO rinco_events.event_log
  (tenant_id, event_date, event_id, event_time, event_name, actor_user_id, actor_type, entity_type, entity_id, correlation_id, payload, source_service)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now() - 1), uuid('e1111111-1111-1111-1111-000000000004'), toTimestamp(now() - 86400 * 2), 'user.login', uuid('a0000001-0000-0000-0000-000000000001'), 'user', 'user', uuid('a0000001-0000-0000-0000-000000000001'), uuid('s1111111-1111-1111-1111-000000000001'), '{"ip":"203.0.113.10"}', 'auth-service');

-- HCT Consulting events
INSERT INTO rinco_events.event_log
  (tenant_id, event_date, event_id, event_time, event_name, actor_user_id, actor_type, entity_type, entity_id, correlation_id, payload, source_service)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now()), uuid('e2222222-2222-2222-2222-000000000001'), toTimestamp(now() - 7200), 'property.viewing.scheduled', uuid('a0000002-0000-0000-0000-000000000010'), 'user', 'property', uuid('p2222222-2222-2222-2222-000000000001'), uuid('c2222222-2222-2222-2222-000000000001'), '{"property":"Vinhomes Grand Park","date":"2026-09-20"}', 'crm-service');

INSERT INTO rinco_events.event_log
  (tenant_id, event_date, event_id, event_time, event_name, actor_user_id, actor_type, entity_type, entity_id, correlation_id, payload, source_service)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000002', toDate(now()), uuid('e2222222-2222-2222-2222-000000000002'), toTimestamp(now() - 3600), 'lead.created', uuid('a0000002-0000-0000-0000-000000000011'), 'user', 'lead', uuid('l2222222-2222-2222-2222-000000000001'), uuid('c2222222-2222-2222-2222-000000000001'), '{"source":"zalo","score":78}', 'landing-service');

-- Demo Company events
INSERT INTO rinco_events.event_log
  (tenant_id, event_date, event_id, event_time, event_name, actor_user_id, actor_type, entity_type, entity_id, correlation_id, payload, source_service)
VALUES
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now()), uuid('e3333333-3333-3333-3333-000000000001'), toTimestamp(now() - 1800), 'demo.started', uuid('a0000003-0000-0000-0000-000000000001'), 'user', 'demo', uuid('x3333333-3333-3333-3333-000000000001'), uuid('s3333333-3333-3333-3333-000000000001'), '{"plan":"enterprise"}', 'auth-service');

INSERT INTO rinco_events.event_log
  (tenant_id, event_date, event_id, event_time, event_name, actor_user_id, actor_type, entity_type, entity_id, correlation_id, payload, source_service)
VALUES
  ('bbbbbbbb-0000-0000-0000-000000000003', toDate(now()), uuid('e3333333-3333-3333-3333-000000000002'), toTimestamp(now() - 900), 'lead.created', uuid('a0000003-0000-0000-0000-000000000002'), 'user', 'lead', uuid('l3333333-3333-3333-3333-000000000001'), uuid('c3333333-3333-3333-3333-000000000001'), '{"source":"demo"}', 'lead-service');

-- ============================================================
-- VERIFICATION
-- ============================================================
SELECT 'rinco_chat.conversations' AS table_name, count(*) AS total_rows FROM rinco_chat.conversations;
SELECT 'rinco_chat.messages_by_channel' AS table_name, count(*) AS total_rows FROM rinco_chat.messages_by_channel;
SELECT 'rinco_events.event_log' AS table_name, count(*) AS total_rows FROM rinco_events.event_log;

SELECT 'ScyllaDB comprehensive seed complete!' AS status;
