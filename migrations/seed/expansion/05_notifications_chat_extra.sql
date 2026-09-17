-- ============================================================
-- WS-B LOOP 4: NOTIFICATIONS + CHAT CHANNELS + TAGS + CUSTOM FIELDS
-- 100+ notifications, 5+ chat channels/tenant, 20+ tags, 20+ custom fields
-- ============================================================

-- ============================================================
-- APEX FINTECH: 5 EXTRA CHAT CHANNELS
-- ============================================================
INSERT INTO crm.chat_channels (id, tenant_id, name, description, type, created_by, member_ids, last_message_at)
VALUES
  ('cc000001-0000-0000-0000-000000000006','aaaaaaaa-0000-0000-0000-000000000001','Team HCM','Kênh riêng team sales HCM','private','b0000001-0000-0000-0000-000000000001',
    ARRAY['b0000001-0000-0000-0000-000000000001','b0000001-0000-0000-0000-000000000010','b0000001-0000-0000-0000-000000000020','a0000001-0000-0000-0000-000000000010','a0000001-0000-0000-0000-000000000011']::UUID[],
    NOW() - INTERVAL '5 minutes'),
  ('cc000001-0000-0000-0000-000000000007','aaaaaaaa-0000-0000-0000-000000000001','Team HN','Kênh riêng team sales HN','private','b0000001-0000-0000-0000-000000000002',
    ARRAY['b0000001-0000-0000-0000-000000000002','b0000001-0000-0000-0000-000000000021','a0000001-0000-0000-0000-000000000012','a0000001-0000-0000-0000-000000000013']::UUID[],
    NOW() - INTERVAL '15 minutes'),
  ('cc000001-0000-0000-0000-000000000008','aaaaaaaa-0000-0000-0000-000000000001','Underwriting','Kênh team thẩm định','private','a0000001-0000-0000-0000-000000000006',
    ARRAY['a0000001-0000-0000-0000-000000000006','a0000001-0000-0000-0000-000000000002']::UUID[],
    NOW() - INTERVAL '2 hours'),
  ('cc000001-0000-0000-0000-000000000009','aaaaaaaa-0000-0000-0000-000000000001','VIP Deals','Kênh chia sẻ deal VIP','announcement','a0000001-0000-0000-0000-000000000002',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001')::UUID[],
    NOW() - INTERVAL '30 minutes'),
  ('cc000001-0000-0000-0000-00000000000a','aaaaaaaa-0000-0000-0000-000000000001','AI Insights','Kênh chia sẻ insights từ AI scoring','public','a0000001-0000-0000-0000-000000000004',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001')::UUID[],
    NOW() - INTERVAL '1 hour')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- HCT CONSULTING: 5 EXTRA CHAT CHANNELS
-- ============================================================
INSERT INTO crm.chat_channels (id, tenant_id, name, description, type, created_by, member_ids, last_message_at)
VALUES
  ('cc000002-0000-0000-0000-000000000006','aaaaaaaa-0000-0000-0000-000000000002','BDS Team HN','Team BDS Hà Nội','private','b0000002-0000-0000-0000-000000000001',
    ARRAY['b0000002-0000-0000-0000-000000000001','b0000002-0000-0000-0000-000000000010','b0000002-0000-0000-0000-000000000011','b0000002-0000-0000-0000-000000000020','a0000002-0000-0000-0000-000000000010','a0000002-0000-0000-0000-000000000011','a0000002-0000-0000-0000-000000000012','a0000002-0000-0000-0000-000000000013']::UUID[],
    NOW() - INTERVAL '8 minutes'),
  ('cc000002-0000-0000-0000-000000000007','aaaaaaaa-0000-0000-0000-000000000002','Investment Talks','Thảo luận đầu tư','private','a0000002-0000-0000-0000-000000000003',
    ARRAY['a0000002-0000-0000-0000-000000000003','a0000002-0000-0000-0000-000000000020','a0000002-0000-0000-0000-000000000021','b0000002-0000-0000-0000-000000000030']::UUID[],
    NOW() - INTERVAL '3 hours'),
  ('cc000002-0000-0000-0000-000000000008','aaaaaaaa-0000-0000-0000-000000000002','Vinhomes Project','Dự án Vinhomes','public','b0000002-0000-0000-0000-000000000010',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000002')::UUID[],
    NOW() - INTERVAL '20 minutes'),
  ('cc000002-0000-0000-0000-000000000009','aaaaaaaa-0000-0000-0000-000000000002','Masterise Project','Dự án Masterise','public','b0000002-0000-0000-0000-000000000011',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000002')::UUID[],
    NOW() - INTERVAL '45 minutes'),
  ('cc000002-0000-0000-0000-00000000000a','aaaaaaaa-0000-0000-0000-000000000002','Investors Lounge','Phòng chờ investors','public','b0000002-0000-0000-0000-000000000030',
    ARRAY['b0000002-0000-0000-0000-000000000030','a0000002-0000-0000-0000-000000000001','a0000002-0000-0000-0000-000000000002','a0000002-0000-0000-0000-000000000003']::UUID[],
    NOW() - INTERVAL '5 hours')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DEMO COMPANY: 3 EXTRA CHAT CHANNELS
-- ============================================================
INSERT INTO crm.chat_channels (id, tenant_id, name, description, type, created_by, member_ids, last_message_at)
VALUES
  ('cc000003-0000-0000-0000-000000000005','bbbbbbbb-0000-0000-0000-000000000003','Demo Engineering','Demo team kỹ thuật','private','b0000003-0000-0000-0000-000000000001',
    ARRAY['b0000003-0000-0000-0000-000000000001','b0000003-0000-0000-0000-000000000010','b0000003-0000-0000-0000-000000000020','a0000003-0000-0000-0000-000000000010','a0000003-0000-0000-0000-000000000011','a0000003-0000-0000-0000-000000000012','a0000003-0000-0000-0000-000000000013','a0000003-0000-0000-0000-000000000014']::UUID[],
    NOW() - INTERVAL '5 minutes'),
  ('cc000003-0000-0000-0000-000000000006','bbbbbbbb-0000-0000-0000-000000000003','Demo Feedback','Feedback khách hàng demo','public','b0000003-0000-0000-0000-000000000010',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='bbbbbbbb-0000-0000-0000-000000000003')::UUID[],
    NOW() - INTERVAL '1 hour'),
  ('cc000003-0000-0000-0000-000000000007','bbbbbbbb-0000-0000-0000-000000000003','Demo Random','Demo chitchat','public','a0000003-0000-0000-0000-000000000001',
    ARRAY(SELECT id FROM auth.users WHERE tenant_id='bbbbbbbb-0000-0000-0000-000000000003')::UUID[],
    NOW() - INTERVAL '2 hours')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- ADDITIONAL CHAT MESSAGES (200+ extra per tenant)
-- ============================================================
INSERT INTO crm.chat_messages (id, tenant_id, channel_id, sender_id, sender_name, body, type, metadata, created_at)
SELECT
  gen_random_uuid(),
  c.tenant_id,
  c.id,
  (SELECT id FROM auth.users WHERE tenant_id=c.tenant_id ORDER BY random() LIMIT 1),
  (SELECT full_name FROM auth.users WHERE tenant_id=c.tenant_id ORDER BY random() LIMIT 1),
  m.body,
  (ARRAY['text','text','text','text','text','file','reaction','system'])[ceil(random()*8)],
  CASE WHEN random() < 0.3 THEN jsonb_build_object('reactions', jsonb_build_object('👍', (random()*10)::int, '🎉', (random()*5)::int)) ELSE '{}'::jsonb END,
  NOW() - ((gs + random()*24) || ' hours')::interval
FROM crm.chat_channels c
CROSS JOIN generate_series(1, 35) AS gs
CROSS JOIN (VALUES
  ('Có ai rảnh review deal này giúp mình không?'),
  ('Vừa gọi xong khách, họ muốn schedule meeting tuần sau'),
  ('Update: lead từ Facebook Ads đã qualified'),
  ('Gửi proposal xong rồi, chờ phản hồi 48h'),
  ('Team mình đạt 120% target tuần này!'),
  ('Khách mới yêu cầu demo tính năng AI scoring'),
  ('Có bug gì ai biết không? Service chậm'),
  ('Meeting lúc 14h confirm đi mọi người'),
  ('Hot lead mới về, score 95, cần xử lý ngay'),
  ('Reminder: deadline báo cáo tháng vào thứ 6'),
  ('Ai có template email follow-up không?'),
  ('Deal lớn vừa chốt, 2 tỷ! 🎉'),
  ('Khách hỏi về pricing, ai support giúp?'),
  ('Training session thứ 4 tuần này, ai đến?'),
  ('Update docs mới lên Confluence rồi nhé'),
  ('Khách complain về response time, cần xử lý'),
  ('New campaign launched: spring_loan_v2'),
  ('Có 5 leads mới từ Zalo OA sáng nay'),
  ('Reminder: cập nhật CRM trước 18h'),
  ('Deal trên 500tr cần manager approve'),
  ('Test mới release, ai verify giúp?'),
  ('Khách VIP yêu cầu dedicated agent'),
  ('Sprint planning thứ 2, mang laptop nhé'),
  ('Database backup done at 02:00 AM'),
  ('Lunch ai đi không? Cafe Dinh rất ngon'),
  ('Email marketing scheduled cho 9h sáng mai'),
  ('Khách hẹn gặp lại thứ 7 tuần sau'),
  ('New hire onboard tuần này, welcome!'),
  ('Performance review deadline 30/9'),
  ('Customer survey results đã có')
) AS m(body)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- EXTRA MEETING ROOMS (15+ entries)
-- ============================================================
INSERT INTO crm.meeting_rooms (id, tenant_id, name, description, host_user_id, participant_ids, room_url, started_at, ended_at, duration_seconds, status, metadata, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  m.name,
  m.desc,
  (SELECT id FROM auth.users WHERE tenant_id = t.id AND parent_id IS NULL ORDER BY random() LIMIT 1),
  ARRAY(SELECT id FROM auth.users WHERE tenant_id = t.id ORDER BY random() LIMIT (3 + (random()*5)::int)),
  'https://meet.rinco.app/' || md5(random()::text),
  m.start_time,
  CASE WHEN m.status='ended' THEN m.start_time + (m.duration_minutes || ' minutes')::interval ELSE NULL END,
  CASE WHEN m.status='ended' THEN m.duration_minutes * 60 ELSE 0 END,
  m.status,
  jsonb_build_object('recording_enabled', random()<0.5, 'topic', m.name),
  m.start_time - INTERVAL '5 days'
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Daily Standup', 'Daily team sync', 'ended'::text, NOW() - INTERVAL '1 day', 15),
  ('Demo Khách FPT', 'Demo cho khách FPT', 'ended'::text, NOW() - INTERVAL '3 days', 60),
  ('Review Tuần', 'Weekly review meeting', 'ended'::text, NOW() - INTERVAL '4 days', 45),
  ('Demo Vinhomes', 'Demo dự án Vinhomes', 'ended'::text, NOW() - INTERVAL '5 days', 90),
  ('Khách VNG', 'Demo cho VNG', 'ended'::text, NOW() - INTERVAL '6 days', 45),
  ('Demo Masan', 'Demo Masan Group', 'ended'::text, NOW() - INTERVAL '7 days', 60),
  ('BDS View Trip', 'Viewing trip BDS cao cấp', 'ended'::text, NOW() - INTERVAL '8 days', 180),
  ('Investor Meeting', 'Meeting với investors', 'ended'::text, NOW() - INTERVAL '10 days', 90),
  ('Khách Techcombank', 'Demo ngân hàng Techcombank', 'ended'::text, NOW() - INTERVAL '12 days', 60),
  ('Workshop AI', 'Workshop về AI features', 'ended'::text, NOW() - INTERVAL '14 days', 120),
  ('Demo Plan', 'Demo planning session', 'scheduled'::text, NOW() + INTERVAL '2 hours', 30),
  ('Buyer Meeting', 'Meeting với buyer mới', 'scheduled'::text, NOW() + INTERVAL '1 day', 45),
  ('Sales Training', 'Training sales team', 'scheduled'::text, NOW() + INTERVAL '2 days', 90),
  ('Board Meeting', 'Board of directors meeting', 'scheduled'::text, NOW() + INTERVAL '5 days', 120),
  ('Demo Q4', 'Demo quarterly Q4', 'scheduled'::text, NOW() + INTERVAL '7 days', 60),
  ('Khách VPBank', 'Demo cho VPBank', 'live'::text, NOW() - INTERVAL '5 minutes', 0),
  ('Demo Khách ACB', 'Demo ACB ngân hàng', 'cancelled'::text, NOW() - INTERVAL '2 days', 0)
) AS m(name, desc, status, start_time, duration_minutes)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- ADDITIONAL TAGS (20+ per tenant)
-- ============================================================
INSERT INTO tags (id, tenant_id, name, color, description, usage_count, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  tag.k,
  tag.color,
  tag.v,
  (random()*200)::int,
  NOW() - (random()*60 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('High Value','#DC2626','Khách giá trị cao, LTV lớn'),
  ('Returning','#059669','Khách quay lại sau 6 tháng'),
  ('New Customer','#3B82F6','Khách mới trong 30 ngày'),
  ('Loyal','#7C3AED','Khách trung thành > 1 năm'),
  ('Inactive','#6B7280','Khách không tương tác > 60 ngày'),
  ('Newsletter','#0EA5E9','Subscribe newsletter'),
  ('Webinar','#F59E0B','Đã tham gia webinar'),
  ('Free Trial','#10B981','Đang dùng trial'),
  ('Paid','#84CC16','Đã trả phí'),
  ('Churned','#EF4444','Đã churn'),
  ('Resurrected','#F97316','Đã quay lại sau churn'),
  ('Referral Source','#8B5CF6','Đến từ referral'),
  ('Organic','#06B6D4','Đến từ organic search'),
  ('Paid Ads','#EC4899','Đến từ paid ads'),
  ('Social','#3B82F6','Đến từ social media'),
  ('Direct','#14B8A6','Direct traffic'),
  ('Email Campaign','#A855F7','Từ email campaign'),
  ('SMS Campaign','#F59E0B','Từ SMS campaign'),
  ('Push Notification','#10B981','Từ push notification'),
  ('Cold Outreach','#64748B','Outbound cold'),
  ('Inbound','#22C55E','Inbound lead'),
  ('Webinar Attendee','#F97316','Đã tham dự webinar'),
  ('Whitepaper Download','#0EA5E9','Đã tải whitepaper'),
  ('Contact Request','#FACC15','Yêu cầu liên hệ')
) AS tag(k, color, v)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- MORE CUSTOM FIELDS (20+ extra per entity_type per tenant)
-- ============================================================
INSERT INTO custom_fields (id, tenant_id, entity_type, name, field_key, field_type, description, options, default_value, validation_rules, is_required, is_unique, display_order, width, is_active, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  cf.entity_type,
  cf.name,
  cf.field_key,
  cf.field_type,
  cf.description,
  cf.options::jsonb,
  cf.default_value::jsonb,
  cf.validation_rules::jsonb,
  cf.is_required::boolean,
  cf.is_unique::boolean,
  cf.display_order::int,
  cf.width::text,
  true,
  NOW() - ((random()*40)::int || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  -- lead entity
  ('lead','Age','age','number','Tuổi của lead',
    '[]','null','{"min":18,"max":100}',false,false,6,'half'),
  ('lead','Gender','gender','select','Giới tính',
    '["male","female","other"]','null','{}',false,false,7,'half'),
  ('lead','Income Range','income_range','select','Mức thu nhập hàng tháng',
    '["under_10m","10-20m","20-50m","50-100m","over_100m"]','null','{}',false,false,8,'full'),
  ('lead','Family Status','family_status','select','Tình trạng gia đình',
    '["single","married","divorced","widowed"]','null','{}',false,false,9,'half'),
  ('lead','Children','children','number','Số con',
    '[]','0','{"min":0,"max":10}',false,false,10,'half'),
  ('lead','Education','education','select','Trình độ học vấn',
    '["high_school","bachelor","master","phd","other"]','null','{}',false,false,11,'full'),
  ('lead','Marital Status','marital','select','Hôn nhân',
    '["single","married","divorced","widowed"]','null','{}',false,false,12,'half'),
  ('lead','Lead Temperature','temperature','select','Nhiệt độ lead',
    '["hot","warm","cold"]','warm','{}',true,false,13,'half'),
  ('lead','Lead Source Detail','source_detail','text','Chi tiết nguồn',
    '[]','""','{"max_length":500}',false,false,14,'full'),
  ('lead','Referrer','referrer','text','Người giới thiệu',
    '[]','""','{}',false,false,15,'full'),
  -- contact entity
  ('contact','Zalo OA','zalo_oa','string','Zalo Official Account',
    '[]','""','{}',false,false,5,'half'),
  ('contact','WhatsApp','whatsapp','string','WhatsApp number',
    '[]','""','{}',false,false,6,'half'),
  ('contact','Telegram','telegram','string','Telegram handle',
    '[]','""','{}',false,false,7,'half'),
  ('contact','Birthday','birthday','date','Sinh nhật',
    '[]','null','{}',false,false,8,'half'),
  ('contact','Anniversary','anniversary','date','Ngày kỷ niệm',
    '[]','null','{}',false,false,9,'half'),
  ('contact','Spouse Name','spouse_name','string','Tên vợ/chồng',
    '[]','""','{}',false,false,10,'full'),
  ('contact','Children Count','children','number','Số con',
    '[]','0','{"min":0,"max":15}',false,false,11,'half'),
  -- deal entity
  ('deal','Source Detail','source_detail','text','Chi tiết nguồn deal',
    '[]','""','{}',false,false,3,'full'),
  ('deal','Campaign Name','campaign','text','Tên campaign',
    '[]','""','{}',false,false,4,'full'),
  ('deal','Interest Rate','interest_rate','number','Lãi suất %',
    '[]','0','{"min":0,"max":50}',false,false,5,'half'),
  ('deal','Term','term_months','number','Kỳ hạn (tháng)',
    '[]','12','{"min":1,"max":360}',false,false,6,'half'),
  ('deal','Collateral','collateral','text','Tài sản đảm bảo',
    '[]','""','{}',false,false,7,'full'),
  ('deal','Co-borrowers','co_borrowers','number','Số người đồng vay',
    '[]','0','{"min":0,"max":10}',false,false,8,'half'),
  -- activity entity
  ('activity','Location','location','string','Địa điểm',
    '[]','""','{}',false,false,2,'full'),
  ('activity','Follow-up Date','followup_date','date','Ngày follow-up',
    '[]','null','{}',false,false,3,'half'),
  ('activity','Sentiment','sentiment','select','Cảm xúc khách hàng',
    '["positive","neutral","negative"]','neutral','{}',false,false,4,'half')
) AS cf(entity_type, name, field_key, field_type, description, options, default_value, validation_rules, is_required, is_unique, display_order, width)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- EXTRA NOTIFICATIONS (100+)
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
  ('[' || n.channels || ']')::jsonb,
  ('{' || n.data || '}')::jsonb,
  n.status,
  CASE WHEN n.status='read' THEN NOW() - ((random()*10)::int || ' hours')::interval ELSE NULL END,
  NOW() - ((random()*30)::int || ' hours')::interval
FROM auth.users u
CROSS JOIN (VALUES
  ('lead_assigned','Lead mới từ Facebook Ads','Bạn được assign lead từ campaign spring_loan_v2','crm','high','{"channel":"in_app"}','"source":"facebook"','read'),
  ('deal_stage_changed','Deal chuyển sang Proposal','Khách hàng FPT đã accept proposal','crm','normal','{"channel":"in_app"}','"deal_id":"deal-001"','delivered'),
  ('lead_score_high','Lead score cao','Lead mới có score 92, cần follow-up','ai_scoring','urgent','{"channel":"in_app"},{"channel":"push"}','"score":92','pending'),
  ('meeting_reminder','Meeting trong 30 phút','Họp với team sales HCM lúc 14h','calendar','normal','{"channel":"in_app"}','"time":"14:00"','read'),
  ('workflow_executed','Workflow chạy thành công','Auto-assign workflow vừa trigger','automation','low','{"channel":"in_app"}','"workflow":"auto_assign"','read'),
  ('payment_received','Thanh toán mới','Invoice INV-2026-0099 đã được thanh toán','billing','normal','{"channel":"in_app"},{"channel":"email"}','"amount":4389000','delivered'),
  ('new_message','Tin nhắn mới từ khách','Khách hàng Nguyễn Văn A vừa nhắn tin','chat','normal','{"channel":"in_app"}','"channel_id":"cc-001"','pending'),
  ('mention','Ai đó mention bạn','Phạm Thị Mai mention bạn trong #sales','chat','normal','{"channel":"in_app"},{"channel":"email"}','"message_id":"msg-123"','pending'),
  ('document_signed','Hợp đồng đã ký','Khách hàng ký hợp đồng điện tử thành công','document','high','{"channel":"in_app"},{"channel":"email"},{"channel":"push"}','"contract_id":"c-999"','delivered'),
  ('lead_bulk_import','Import 50 leads thành công','File CSV đã được xử lý','crm','low','{"channel":"in_app"}','"count":50','read'),
  ('integration_error','Lỗi Meta CAPI','API Meta bị rate limit','integration','high','{"channel":"in_app"},{"channel":"email"}','"service":"meta_capi"','pending'),
  ('tenant_quota_warning','Sắp hết quota','Bạn đã dùng 95% quota tháng này','billing','urgent','{"channel":"in_app"},{"channel":"email"}','"quota":"leads"','pending'),
  ('feature_release','Tính năng mới','AI Lead Scoring v3 vừa được release','system','low','{"channel":"in_app"}','"version":"3.0"','read'),
  ('security_alert','Login từ IP lạ','Có login từ IP 198.51.100.5','security','urgent','{"channel":"in_app"},{"channel":"email"},{"channel":"push"}','"ip":"198.51.100.5"','pending'),
  ('report_ready','Báo cáo đã sẵn sàng','Báo cáo tuần 38 đã được tạo','reporting','normal','{"channel":"in_app"}','"report_id":"r-001"','delivered')
) AS n(type, title, body, category, priority, channels, data, status)
CROSS JOIN generate_series(1, 3) AS gs
WHERE u.tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
  AND u.role IN ('tenant_admin','manager','member')
ON CONFLICT (id) DO NOTHING;

-- Stats
SELECT 'expansion_05_notifications_chat (WS-B Loop 4)' AS section, COUNT(*) AS new_notifs
FROM notification.notifications
WHERE created_at > NOW() - INTERVAL '60 days'
  AND tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');
