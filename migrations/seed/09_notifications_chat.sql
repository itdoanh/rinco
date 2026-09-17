-- ============================================================
-- NOTIFICATIONS / CHAT CHANNELS / MEETING ROOMS / API KEYS (Loop 202)
-- ============================================================

-- ============================================================
-- TENANT NOTIFICATIONS (uses auth-scoped users table from notification-service)
-- ============================================================
INSERT INTO notification.notifications (id, tenant_id, user_id, type, title, body, category, priority, channels_resolved, data, status, read_at, created_at)
SELECT
  gen_random_uuid(),
  u.tenant_id::text,
  u.id::text,
  n.type,
  n.title,
  n.body,
  n.category,
  n.priority,
  n.channels_resolved::jsonb,
  n.data::jsonb,
  n.status,
  CASE WHEN n.status='read' THEN n.created_at + INTERVAL '1 hour' ELSE NULL END,
  NOW() - (random()*15 || ' days')::interval
FROM auth.users u
CROSS JOIN (VALUES
  ('lead_assigned','Khách hàng mới được phân công','Bạn vừa nhận được một lead mới từ Facebook Ads','crm','high',
    '[{"channel":"in_app","sent_at":"2026-09-10T10:00:00Z"}]',
    '{"lead_id":"lead-001","source":"facebook"}',
    'read'),
  ('deal_won','Deal đã chốt thành công','Chúc mừng! Deal #D-123 đã được chốt thành công','crm','high',
    '[{"channel":"in_app"},{"channel":"email"}]','{"deal_id":"deal-001","value":500000000}','delivered'),
  ('lead_score_high','Lead có điểm cao','Lead từ Zalo có score 92, cần follow-up ngay','ai_scoring','urgent',
    '[{"channel":"in_app"},{"channel":"push"}]','{"lead_id":"lead-002","score":92}','pending'),
  ('meeting_reminder','Nhắc lịch họp','Bạn có cuộc họp với khách hàng ABC vào 14:00 hôm nay','calendar','normal',
    '[{"channel":"in_app"}]','{"meeting_id":"m-001"}','sent'),
  ('workflow_triggered','Workflow đã chạy','Auto-assign workflow vừa được trigger','automation','low',
    '[{"channel":"in_app"}]','{"workflow_id":"wf-001"}','read'),
  ('payment_received','Thanh toán đã nhận','Invoice INV-2026-0001 đã được thanh toán thành công','billing','normal',
    '[{"channel":"in_app"},{"channel":"email"}]','{"invoice_id":"inv-001","amount":4389000}','delivered'),
  ('system_update','Hệ thống đã cập nhật','RINCO v2.5 vừa release, xem chi tiết tại changelog','system','low',
    '[{"channel":"in_app"}]','{"version":"2.5"}','read'),
  ('team_invitation','Lời mời vào team','Bạn được mời vào team Sales HCM','team','normal',
    '[{"channel":"email"}]','{"team_id":"team-001"}','delivered')
) AS n(type, title, body, category, priority, channels_resolved, data, status)
WHERE u.tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
  AND u.role IN ('tenant_admin','manager','member')
LIMIT 80;

-- ============================================================
-- NOTIFICATION PREFERENCES
-- ============================================================
INSERT INTO notification.notification_preferences (user_id, notif_type, channel, enabled, quiet_start, quiet_end, digest_mode)
SELECT
  u.id::text,
  n.type,
  c.channel,
  c.enabled,
  22,7,
  c.digest_mode
FROM auth.users u
CROSS JOIN (VALUES
  ('lead_assigned','in_app',true,'none'),
  ('lead_assigned','email',true,'daily'),
  ('lead_assigned','push',true,'none'),
  ('lead_assigned','sms',false,'none'),
  ('deal_won','in_app',true,'none'),
  ('deal_won','email',true,'none'),
  ('lead_score_high','in_app',true,'none'),
  ('lead_score_high','push',true,'none'),
  ('meeting_reminder','in_app',true,'none'),
  ('meeting_reminder','email',true,'none'),
  ('payment_received','email',true,'weekly'),
  ('system_update','in_app',false,'weekly')
) AS n(type, c1, c2, c3)
JOIN (VALUES
  ('in_app',true,'none'::text),
  ('email',true,'none'::text),
  ('push',true,'none'::text),
  ('sms',false,'none'::text)
) c(channel, enabled, digest_mode) ON c.channel = n.c1
WHERE u.tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
  AND u.role IN ('tenant_admin','manager')
LIMIT 200
ON CONFLICT DO NOTHING;

-- ============================================================
-- PUSH SUBSCRIPTIONS (a few devices registered)
-- ============================================================
INSERT INTO notification.push_subscriptions (id, user_id, endpoint, p256dh, auth, keys, vapid_public_key, last_seen, created_at)
VALUES
  ('pp000001-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000001',
   'https://fcm.googleapis.com/fcm/send/AAAApex001','BNcRdreALRFXTkBlK1ByMCA3-1_3','tBHItjj5x57fbA1Lxczg_w','{"p256dh":"BNcRdreALRFXTkBlK1ByMCA3-1_3","auth":"tBHItjj5x57fbA1Lxczg_w"}'::jsonb,
   'BElXdJ8VjN9QX9kYx9Y7zJNyy1YbL2cOKxW0qh9mZg',NOW() - INTERVAL '2 days',NOW() - INTERVAL '100 days'),
  ('pp000002-0000-0000-0000-000000000001','a0000002-0000-0000-0000-000000000001',
   'https://fcm.googleapis.com/fcm/send/AAAAhct001','BNcRdreALRFXTkBlK1ByMCA3-2_4','tBHItjj5x57fbA1Lxczg_x','{"p256dh":"BNcRdreALRFXTkBlK1ByMCA3-2_4","auth":"tBHItjj5x57fbA1Lxczg_x"}'::jsonb,
   'BElXdJ8VjN9QX9kYx9Y7zJNyy1YbL2cOKxW0qh9mZg',NOW() - INTERVAL '3 days',NOW() - INTERVAL '50 days'),
  ('pp000003-0000-0000-0000-000000000001','a0000003-0000-0000-0000-000000000001',
   'https://fcm.googleapis.com/fcm/send/AAAAdemo001','BNcRdreALRFXTkBlK1ByMCA3-3_5','tBHItjj5x57fbA1Lxczg_y','{"p256dh":"BNcRdreALRFXTkBlK1ByMCA3-3_5","auth":"tBHItjj5x57fbA1Lxczg_y"}'::jsonb,
   'BElXdJ8VjN9QX9kYx9Y7zJNyy1YbL2cOKxW0qh9mZg',NOW() - INTERVAL '1 day',NOW() - INTERVAL '8 days')
ON CONFLICT (endpoint) DO NOTHING;

-- ============================================================
-- FCM SUBSCRIPTIONS (mobile device tokens)
-- ============================================================
INSERT INTO notification.fcm_subscriptions (id, user_id, device_token, platform, app_version, last_seen, created_at)
VALUES
  ('ff000001-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000001','fcm-apex-mobile-001-android','android','3.2.1',NOW() - INTERVAL '2 days',NOW() - INTERVAL '90 days'),
  ('ff000002-0000-0000-0000-000000000001','a0000002-0000-0000-0000-000000000001','fcm-hct-mobile-001-ios','ios','3.2.1',NOW() - INTERVAL '4 days',NOW() - INTERVAL '40 days'),
  ('ff000003-0000-0000-0000-000000000001','a0000003-0000-0000-0000-000000000001','fcm-demo-mobile-001-android','android','3.2.1',NOW() - INTERVAL '1 day',NOW() - INTERVAL '5 days')
ON CONFLICT (device_token) DO NOTHING;

-- ============================================================
-- CHAT CHANNELS — represented as new tables we create in app schema for legacy use
-- We use the crm chat-engine legacy schema (some services use channel_meta JSON)
-- ============================================================
CREATE TABLE IF NOT EXISTS crm.chat_channels (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT,
    type            TEXT NOT NULL DEFAULT 'public' CHECK (type IN ('public','private','announcement','direct')),
    created_by      UUID REFERENCES auth.users(id),
    member_ids      UUID[] NOT NULL DEFAULT ARRAY[]::UUID[],
    last_message_at TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_chat_channels_tenant ON crm.chat_channels(tenant_id);

CREATE TABLE IF NOT EXISTS crm.chat_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    channel_id      UUID NOT NULL REFERENCES crm.chat_channels(id) ON DELETE CASCADE,
    sender_id       UUID REFERENCES auth.users(id),
    sender_name     TEXT,
    body            TEXT NOT NULL,
    type            TEXT NOT NULL DEFAULT 'text' CHECK (type IN ('text','file','system','call')),
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    reply_to_id     UUID REFERENCES crm.chat_messages(id),
    edited_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_channel ON crm.chat_messages(channel_id, created_at DESC);

-- 5 channels per tenant + messages
INSERT INTO crm.chat_channels (id, tenant_id, name, description, type, created_by, member_ids, last_message_at)
VALUES
  -- Apex Fintech channels
  ('cc000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','General','Kênh chung cho cả team','public','a0000001-0000-0000-0000-000000000001',
    ARRAY['a0000001-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000002','a0000001-0000-0000-0000-000000000003','a0000001-0000-0000-0000-000000000010','a0000001-0000-0000-0000-000000000011']::UUID[],
    NOW() - INTERVAL '2 hours'),
  ('cc000001-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000001','Sales','Kênh riêng phòng sales','private','a0000001-0000-0000-0000-000000000002',
    ARRAY['a0000001-0000-0000-0000-000000000002','a0000001-0000-0000-0000-000000000010','a0000001-0000-0000-0000-000000000011','a0000001-0000-0000-0000-000000000012']::UUID[],
    NOW() - INTERVAL '1 hour'),
  ('cc000001-0000-0000-0000-000000000003','aaaaaaaa-0000-0000-0000-000000000001','Support','Kênh CSKH','public','a0000001-0000-0000-0000-000000000003',
    ARRAY['a0000001-0000-0000-0000-000000000003','a0000001-0000-0000-0000-000000000020','a0000001-0000-0000-0000-000000000021']::UUID[],
    NOW() - INTERVAL '30 minutes'),
  ('cc000001-0000-0000-0000-000000000004','aaaaaaaa-0000-0000-0000-000000000001','Random','Chat phiếm, cafe','public','a0000001-0000-0000-0000-000000000010',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001')::UUID[],
    NOW() - INTERVAL '15 minutes'),
  ('cc000001-0000-0000-0000-000000000005','aaaaaaaa-0000-0000-0000-000000000001','Announcements','Thông báo nội bộ','announcement','a0000001-0000-0000-0000-000000000001',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001')::UUID[],
    NOW() - INTERVAL '12 hours'),
  -- HCT channels
  ('cc000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','General','Kênh chung','public','a0000002-0000-0000-0000-000000000001',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000002')::UUID[],
    NOW() - INTERVAL '3 hours'),
  ('cc000002-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000002','Sales','BĐS sales','private','a0000002-0000-0000-0000-000000000002',
    ARRAY['a0000002-0000-0000-0000-000000000002','a0000002-0000-0000-0000-000000000010','a0000002-0000-0000-0000-000000000011','a0000002-0000-0000-0000-000000000012']::UUID[],
    NOW() - INTERVAL '45 minutes'),
  ('cc000002-0000-0000-0000-000000000003','aaaaaaaa-0000-0000-0000-000000000002','Viewing Trips','Lịch đi xem BĐS','public','a0000002-0000-0000-0000-000000000003',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000002')::UUID[],
    NOW() - INTERVAL '8 hours'),
  ('cc000002-0000-0000-0000-000000000004','aaaaaaaa-0000-0000-0000-000000000002','Marketing','MKT team','private','a0000002-0000-0000-0000-000000000004',
    ARRAY['a0000002-0000-0000-0000-000000000004','a0000002-0000-0000-0000-000000000030','a0000002-0000-0000-0000-000000000031']::UUID[],
    NOW() - INTERVAL '6 hours'),
  ('cc000002-0000-0000-0000-000000000005','aaaaaaaa-0000-0000-0000-000000000002','Announcements','Thông báo','announcement','a0000002-0000-0000-0000-000000000001',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000002')::UUID[],
    NOW() - INTERVAL '1 day'),
  -- Demo channels
  ('cc000003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','General','Demo general','public','a0000003-0000-0000-0000-000000000001',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='bbbbbbbb-0000-0000-0000-000000000003')::UUID[],
    NOW() - INTERVAL '4 hours'),
  ('cc000003-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003','Demo Sales','Sales demo','public','a0000003-0000-0000-0000-000000000002',
    ARRAY['a0000003-0000-0000-0000-000000000002','a0000003-0000-0000-0000-000000000010','a0000003-0000-0000-0000-000000000011']::UUID[],
    NOW() - INTERVAL '2 hours'),
  ('cc000003-0000-0000-0000-000000000003','bbbbbbbb-0000-0000-0000-000000000003','Random','Demo chitchat','public','a0000003-0000-0000-0000-000000000001',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='bbbbbbbb-0000-0000-0000-000000000003')::UUID[],
    NOW() - INTERVAL '20 minutes'),
  ('cc000003-0000-0000-0000-000000000004','bbbbbbbb-0000-0000-0000-000000000003','Demo Announcements','Demo thông báo','announcement','a0000003-0000-0000-0000-000000000001',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='bbbbbbbb-0000-0000-0000-000000000003')::UUID[],
    NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;

-- Messages in channels
INSERT INTO crm.chat_messages (id, tenant_id, channel_id, sender_id, sender_name, body, type, created_at)
SELECT
  gen_random_uuid(),
  c.tenant_id,
  c.id,
  (SELECT id FROM auth.users WHERE tenant_id=c.tenant_id ORDER BY random() LIMIT 1),
  u.full_name,
  m.body,
  'text',
  NOW() - ((20 - gs) || ' hours')::interval
FROM crm.chat_channels c
CROSS JOIN LATERAL (
  SELECT id, full_name FROM auth.users WHERE tenant_id=c.tenant_id ORDER BY random() LIMIT 1
) u
CROSS JOIN generate_series(1,8) AS gs
CROSS JOIN (VALUES
  ('Chào cả team! Hôm nay có update quan trọng'),
  ('Lead mới về từ Facebook Ads, score 85'),
  ('Vừa chốt được deal lớn 500 triệu!'),
  ('Ai rảnh xử lý ticket #1245 giúp mình với'),
  ('Meeting tuần sau lúc 10h, confirm nhanh nhé'),
  ('Đã gửi báo cáo monthly trong #reports'),
  ('Cafe ai đi không?'),
  ('Reminder: deadline dự án vào 5h chiều nay')
) AS m(body)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- MEETING ROOMS (legacy crm recording metadata table)
-- ============================================================
CREATE TABLE IF NOT EXISTS crm.meeting_rooms (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT,
    host_user_id    UUID REFERENCES auth.users(id),
    participant_ids UUID[] NOT NULL DEFAULT ARRAY[]::UUID[],
    room_url        TEXT,
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,
    duration_seconds INT DEFAULT 0,
    recording_id    UUID,
    status          TEXT NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled','live','ended','cancelled')),
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_meeting_rooms_tenant ON crm.meeting_rooms(tenant_id);

INSERT INTO crm.meeting_rooms (id, tenant_id, name, description, host_user_id, participant_ids, room_url, started_at, ended_at, duration_seconds, status, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  m.name,
  m.desc,
  (SELECT id FROM auth.users WHERE tenant_id=t.id AND parent_id IS NULL LIMIT 1),
  ARRAY(SELECT id FROM auth.users WHERE tenant_id=t.id ORDER BY random() LIMIT 4),
  'https://meet.rinco.app/' || md5(random()::text),
  CASE WHEN m.status='scheduled' THEN NULL ELSE m.start_time END,
  CASE WHEN m.status='ended' THEN m.start_time + (m.duration_minutes || ' minutes')::interval ELSE NULL END,
  CASE WHEN m.status='ended' THEN m.duration_minutes * 60 ELSE 0 END,
  m.status,
  m.start_time - INTERVAL '2 days'
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Sales Daily Standup','Daily standup với team sales','scheduled'::text, NOW() + INTERVAL '4 hours', 0),
  ('Demo Khách Hàng VNPT','Demo product cho khách hàng VNPT','scheduled'::text, NOW() + INTERVAL '1 day', 0),
  ('Review Tuần Sales HN','Review tuần với team sales Hà Nội','ended'::text, NOW() - INTERVAL '3 days', 45),
  ('All Hands Meeting Q3','All-hands meeting toàn công ty','ended'::text, NOW() - INTERVAL '7 days', 90),
  ('Demo Vinhomes Grand Park','Demo BĐS cho khách VIP','ended'::text, NOW() - INTERVAL '5 days', 60)
) AS m(name, desc, status, start_time, duration_minutes)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- API KEYS (per tenant — 5 per tenant)
-- ============================================================
CREATE TABLE IF NOT EXISTS auth.api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    key_prefix      TEXT NOT NULL,         -- 'apx_live_', 'hct_live_', 'demo_test_'
    key_hash        TEXT NOT NULL,         -- bcrypt hash of full key
    scopes          TEXT[] NOT NULL DEFAULT ARRAY['read']::TEXT[],
    created_by      UUID REFERENCES auth.users(id),
    last_used_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON auth.api_keys(tenant_id);

INSERT INTO auth.api_keys (tenant_id, name, key_prefix, key_hash, scopes, created_by, last_used_at, expires_at, created_at) VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001','Production CRM Integration','apx_live_','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',ARRAY['read','write']::TEXT[],'a0000001-0000-0000-0000-000000000001',NOW() - INTERVAL '5 minutes',NOW() + INTERVAL '1 year',NOW() - INTERVAL '100 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','Mobile App Backend','apx_live_','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',ARRAY['read','write']::TEXT[],'a0000001-0000-0000-0000-000000000001',NOW() - INTERVAL '1 hour',NOW() + INTERVAL '1 year',NOW() - INTERVAL '95 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','Webhook Receiver (Meta CAPI)','apx_live_','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',ARRAY['webhook']::TEXT[],'a0000001-0000-0000-0000-000000000004',NOW() - INTERVAL '10 minutes',NOW() + INTERVAL '180 days',NOW() - INTERVAL '50 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','Read-only analytics key','apx_live_','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',ARRAY['read','analytics']::TEXT[],'a0000001-0000-0000-0000-000000000005',NOW() - INTERVAL '2 hours',NOW() + INTERVAL '1 year',NOW() - INTERVAL '40 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','Staging (legacy)','apx_test_','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',ARRAY['read']::TEXT[],'a0000001-0000-0000-0000-000000000005',NULL,NOW() + INTERVAL '30 days',NOW() - INTERVAL '20 days'),

  ('aaaaaaaa-0000-0000-0000-000000000002','Production HCT API','hct_live_','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',ARRAY['read','write']::TEXT[],'a0000002-0000-0000-0000-000000000001',NOW() - INTERVAL '20 minutes',NOW() + INTERVAL '1 year',NOW() - INTERVAL '55 days'),
  ('aaaaaaaa-0000-0000-0000-000000000002','Zalo OA Integration','hct_live_','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',ARRAY['webhook']::TEXT[],'a0000002-0000-0000-0000-000000000004',NOW() - INTERVAL '30 minutes',NOW() + INTERVAL '90 days',NOW() - INTERVAL '40 days'),
  ('aaaaaaaa-0000-0000-0000-000000000002','Mobile app backend','hct_live_','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',ARRAY['read','write']::TEXT[],'a0000002-0000-0000-0000-000000000002',NOW() - INTERVAL '15 minutes',NULL,NOW() - INTERVAL '35 days'),

  ('bbbbbbbb-0000-0000-0000-000000000003','Demo API Key (full access)','demo_live_','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',ARRAY['read','write','admin']::TEXT[],'a0000003-0000-0000-0000-000000000001',NOW() - INTERVAL '1 minute',NULL,NOW() - INTERVAL '9 days'),
  ('bbbbbbbb-0000-0000-0000-000000000003','Demo Read-only','demo_test_','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',ARRAY['read']::TEXT[],'a0000003-0000-0000-0000-000000000001',NOW() - INTERVAL '5 minutes',NOW() + INTERVAL '90 days',NOW() - INTERVAL '5 days')
ON CONFLICT DO NOTHING;
