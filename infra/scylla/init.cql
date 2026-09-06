-- ============================================================
-- RINCO ScyllaDB Schema
-- ============================================================

-- ============================================================
-- CHAT KEYSPACE
-- ============================================================
CREATE KEYSPACE IF NOT EXISTS chat WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'replication_factor': 1
} AND durable_writes = true;

USE chat;

-- Conversations
CREATE TABLE IF NOT EXISTS chat.conversations (
    tenant_id UUID,
    conversation_id UUID,
    user_id_a UUID,
    user_id_b UUID,
    type TEXT,
    title TEXT,
    last_message_at TIMESTAMP,
    last_message_preview TEXT,
    created_at TIMESTAMP,
    PRIMARY KEY (tenant_id, conversation_id)
) WITH CLUSTERING ORDER BY (conversation_id ASC)
  AND default_time_to_live = 0;

-- Messages per conversation (recent first via DESC clustering)
CREATE TABLE IF NOT EXISTS chat.messages_by_conversation (
    tenant_id UUID,
    conversation_id UUID,
    message_id TIMEUUID,
    sender_id UUID,
    content TEXT,
    message_type TEXT,
    metadata MAP<TEXT, TEXT>,
    edited_at TIMESTAMP,
    deleted_at TIMESTAMP,
    PRIMARY KEY ((tenant_id, conversation_id), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC)
  AND default_time_to_live = 0;

-- Messages by sender (for inbox view)
CREATE TABLE IF NOT EXISTS chat.messages_by_sender (
    tenant_id UUID,
    sender_id UUID,
    message_id TIMEUUID,
    conversation_id UUID,
    content TEXT,
    PRIMARY KEY ((tenant_id, sender_id), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC)
  AND default_time_to_live = 0;

-- Presence
CREATE TABLE IF NOT EXISTS chat.presence_by_user (
    tenant_id UUID,
    user_id UUID,
    online_status TEXT,
    last_seen TIMESTAMP,
    device_id TEXT,
    PRIMARY KEY (tenant_id, user_id)
);

-- Read receipts
CREATE TABLE IF NOT EXISTS chat.read_receipts (
    tenant_id UUID,
    conversation_id UUID,
    user_id UUID,
    last_read_message_id UUID,
    last_read_at TIMESTAMP,
    PRIMARY KEY ((tenant_id, conversation_id), user_id)
);

-- Typing indicators (short TTL)
CREATE TABLE IF NOT EXISTS chat.typing (
    tenant_id UUID,
    conversation_id UUID,
    user_id UUID,
    started_at TIMESTAMP,
    PRIMARY KEY ((tenant_id, conversation_id), user_id)
) WITH default_time_to_live = 10;

-- ============================================================
-- WEBRTC KEYSPACE
-- ============================================================
CREATE KEYSPACE IF NOT EXISTS webrtc WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'replication_factor': 1
} AND durable_writes = true;

USE webrtc;

-- Meeting sessions
CREATE TABLE IF NOT EXISTS webrtc.sessions (
    tenant_id UUID,
    session_id UUID,
    room_id UUID,
    host_id UUID,
    title TEXT,
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    duration_seconds INT,
    participants SET<UUID>,
    metadata MAP<TEXT, TEXT>,
    recording_url TEXT,
    PRIMARY KEY (tenant_id, session_id)
) WITH CLUSTERING ORDER BY (session_id DESC);

-- Sessions by user (history)
CREATE TABLE IF NOT EXISTS webrtc.sessions_by_user (
    tenant_id UUID,
    user_id UUID,
    session_id UUID,
    room_id UUID,
    title TEXT,
    started_at TIMESTAMP,
    duration_seconds INT,
    PRIMARY KEY ((tenant_id, user_id), started_at, session_id)
) WITH CLUSTERING ORDER BY (started_at DESC);

-- ============================================================
-- LEADS RAW (Landing Ingest)
-- ============================================================
CREATE KEYSPACE IF NOT EXISTS rinco_leads WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'replication_factor': 1
} AND durable_writes = true;

USE rinco_leads;

-- Raw leads from landing ingestion (time-bucketed partition)
CREATE TABLE IF NOT EXISTS rinco_leads.leads_raw (
    tenant_id TEXT,
    event_time BIGINT,
    event_id UUID,
    idempotency_key TEXT,
    full_name TEXT,
    phone TEXT,
    email TEXT,
    utm_source TEXT,
    utm_medium TEXT,
    utm_campaign TEXT,
    utm_content TEXT,
    utm_term TEXT,
    fbclid TEXT,
    gclid TEXT,
    ttclid TEXT,
    ip_address TEXT,
    user_agent TEXT,
    source TEXT,
    form_id TEXT,
    extra MAP<TEXT, TEXT>,
    processed BOOLEAN,
    lead_id UUID,
    fb_capi_sent BOOLEAN,
    PRIMARY KEY ((tenant_id, event_time), event_id)
) WITH CLUSTERING ORDER BY (event_id DESC);

-- CAPI events log (audit + dedup)
CREATE TABLE IF NOT EXISTS rinco_leads.capi_events (
    tenant_id TEXT,
    event_id UUID,
    event_name TEXT,
    fb_event_id TEXT,
    payload TEXT,
    sent_at TIMESTAMP,
    response TEXT,
    status_code INT,
    PRIMARY KEY (tenant_id, event_id)
);
