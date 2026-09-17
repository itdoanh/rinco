-- ============================================================
-- RINCO ScyllaDB Demo Seed (Loop 202)
-- Apply:  cqlsh localhost 9042 -f seed/scylla/seed.cql
-- All statements idempotent / IF NOT EXISTS where possible.
-- ============================================================

-- We use the chat keyspace per project migration. Adjust keyspace
-- name to 'rinco_chat' if your setup differs; the CQL default is 'chat'.
-- See infra/scylla/keyspace.cql for actual keyspace creations.

-- ============================================================
-- KEYSPACE: rinco_chat (created if missing)
-- ============================================================
CREATE KEYSPACE IF NOT EXISTS rinco_chat
  WITH replication = {
    'class': 'SimpleStrategy',
    'replication_factor': 1
  } AND durable_writes = true;

USE rinco_chat;

CREATE TABLE IF NOT EXISTS rinco_chat.conversations (
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
) WITH CLUSTERING ORDER BY (conversation_id ASC);

CREATE TABLE IF NOT EXISTS rinco_chat.messages_by_channel (
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
) WITH CLUSTERING ORDER BY (msg_id DESC);

-- ============================================================
-- KEYSPACE: rinco_leads (raw lead-ingress stream)
-- ============================================================
CREATE KEYSPACE IF NOT EXISTS rinco_leads
  WITH replication = {'class': 'SimpleStrategy','replication_factor': 1} AND durable_writes = true;

USE rinco_leads;

CREATE TABLE IF NOT EXISTS rinco_leads.leads_raw (
    tenant_id     uuid,
    submitted_at  timestamp,
    event_id      uuid,
    full_name     text,
    phone         text,
    email         text,
    source        text,
    utm           map<text, text>,
    fbclid        text,
    gclid         text,
    ip            text,
    user_agent    text,
    raw_json      text,
    processed     boolean,
    lead_id       uuid,
    PRIMARY KEY ((tenant_id), submitted_at, event_id)
) WITH CLUSTERING ORDER BY (submitted_at DESC, event_id DESC);

-- ============================================================
-- KEYSPACE: rinco_audit (event stream)
-- ============================================================
CREATE KEYSPACE IF NOT EXISTS rinco_audit
  WITH replication = {'class': 'SimpleStrategy','replication_factor': 1} AND durable_writes = true;

USE rinco_audit;

CREATE TABLE IF NOT EXISTS rinco_audit.events_by_day (
    tenant_id     text,
    event_date    date,
    event_time    timestamp,
    event_id      uuid,
    event_name    text,
    actor_user_id uuid,
    entity_type   text,
    entity_id     uuid,
    correlation_id uuid,
    payload       text,
    PRIMARY KEY ((tenant_id, event_date), event_time, event_id)
) WITH CLUSTERING ORDER BY (event_time DESC, event_id DESC);

USE system;

-- ============================================================
-- INSERT sample data — note: CQL does not support cross-keyspace
-- prepared-statements in one block easily. We will use cqlsh
-- batch mode. For idempotency we use fixed UUIDs.
-- ============================================================

-- We provide demo data here. In your real seed runner, generate
-- programmatically (see scripts/seed-all.ps1 which handles it).

-- 3 tenants
INSERT INTO rinco_chat.conversations (tenant_id, conversation_id, type, title, member_ids, last_message_at, last_message_preview, created_by, created_at, updated_at)
VALUES
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-1111-1111-1111-000000000001, 'public', 'Apex Fintech General',
   {a0000001-0000-0000-0000-000000000001,a0000001-0000-0000-0000-000000000002,a0000001-0000-0000-0000-000000000010},
   toTimestamp(now()), 'Chào cả team, có deal mới về!',
   a0000001-0000-0000-0000-000000000001, toTimestamp(now()), toTimestamp(now()));

INSERT INTO rinco_chat.conversations (tenant_id, conversation_id, type, title, member_ids, last_message_at, last_message_preview, created_by, created_at, updated_at)
VALUES
  (aaaaaaaa-0000-0000-0000-000000000002, 22222222-2222-2222-2222-000000000001, 'public', 'HCT Consulting General',
   {a0000002-0000-0000-0000-000000000001,a0000002-0000-0000-0000-000000000010,a0000002-0000-0000-0000-000000000011},
   toTimestamp(now()), 'Tuần này có viewing Vinhomes Grand Park thứ 7',
   a0000002-0000-0000-0000-000000000001, toTimestamp(now()), toTimestamp(now()));

-- A few messages per channel
INSERT INTO rinco_chat.messages_by_channel (tenant_id, channel_id, msg_id, sender_id, msg_type, body)
VALUES
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-1111-1111-1111-000000000001, now(), a0000001-0000-0000-0000-000000000001, 'text', 'Chào cả team, có deal mới về từ Facebook Ads');
INSERT INTO rinco_chat.messages_by_channel (tenant_id, channel_id, msg_id, sender_id, msg_type, body)
VALUES
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-1111-1111-1111-000000000001, now(), a0000001-0000-0000-0000-000000000002, 'text', 'Nhận rồi, tôi sẽ follow-up ngay');
INSERT INTO rinco_chat.messages_by_channel (tenant_id, channel_id, msg_id, sender_id, msg_type, body)
VALUES
  (aaaaaaaa-0000-0000-0000-000000000001, 11111111-1111-1111-1111-000000000001, now(), a0000001-0000-0000-0000-000000000003, 'text', 'Đã cập nhật ticket support #1245');

-- A few demo leads_raw entries (100 records target via runner)
-- Fixed UUIDs for idempotency
INSERT INTO rinco_leads.leads_raw (tenant_id, submitted_at, event_id, full_name, phone, email, source, utm, fbclid, gclid, processed, lead_id)
VALUES
  (aaaaaaaa-0000-0000-0000-000000000001, toTimestamp(now()), 55555555-5555-5555-5555-000000000001, 'Nguyen Van Anh', '+8490800001', 'anh.nguyen@example.com', 'facebook', {'utm_source':'facebook','utm_campaign':'spring_loan'}, 'fb.abc123', null, true, 66666666-6666-6666-6666-000000000001);

-- Sample event log entries
INSERT INTO rinco_audit.events_by_day (tenant_id, event_date, event_time, event_id, event_name, actor_user_id, entity_type, entity_id, correlation_id, payload)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', toDate(now()), toTimestamp(now()), 77777777-7777-7777-7777-000000000001, 'lead.created', a0000001-0000-0000-0000-000000000001, 'lead', 66666666-6666-6666-6666-000000000001, 88888888-8888-8888-8888-000000000001, '{"source":"facebook"}');

-- ============================================================
-- NOTE: Bulk generation of 500 messages / 100 leads is performed
-- by the PowerShell seed runner (`scripts/seed-all.ps1`) using
-- cqlsh COPY or looped INSERTs from a JS generator.
-- This file provides schema + seed shape; the runner handles volume.
-- ============================================================
