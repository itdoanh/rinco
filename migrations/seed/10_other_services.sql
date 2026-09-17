-- ============================================================
-- EMAIL TEMPLATES + LOGS + LANDING PAGES + FB PIXEL + OBSERVABILITY + RECORDINGS (Loop 202)
-- ============================================================

-- ============================================================
-- EMAIL TEMPLATES (8 per tenant)
-- ============================================================
INSERT INTO email.email_templates (id, tenant_id, name, subject, body, body_type, vars, type, is_active, version, created_by, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  tpl.name,
  tpl.subject,
  tpl.body,
  tpl.body_type,
  '[]'::jsonb,
  tpl.type,
  true,
  1,
  (SELECT id FROM auth.users WHERE tenant_id=t.id AND parent_id IS NULL LIMIT 1),
  NOW() - (random()*60 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Welcome Email','Chào mừng bạn đến với RINCO','<h1>Chào {{.name}}</h1><p>Cảm ơn bạn đã đăng ký. Trợ lý AI sẽ hỗ trợ bạn trong 24h.</p>','html','transactional'),
  ('Lead Assigned','Bạn có lead mới','<p>Bạn vừa được assign một lead mới từ {{.source}}.</p><p>Lead: {{.lead_name}} - Score: {{.score}}</p>','html','transactional'),
  ('Deal Won Confirmation','Chúc mừng! Deal đã chốt','<h1>Chúc mừng {{.agent_name}}</h1><p>Deal #{{.deal_id}} với giá trị {{.value}} đã chốt thành công.</p>','html','transactional'),
  ('Weekly Report','Báo cáo tuần - {{.week}}','<h2>Tổng quan tuần {{.week}}</h2><ul><li>Leads mới: {{.new_leads}}</li><li>Deals chốt: {{.won_deals}}</li><li>Doanh thu: {{.revenue}}</li></ul>','html','marketing'),
  ('Forgot Password','Khôi phục mật khẩu','<p>Click vào link sau để reset: <a href="{{.reset_link}}">{{.reset_link}}</a></p>','html','transactional'),
  ('Invoice','Hóa đơn {{.invoice_number}}','<p>Hóa đơn của bạn: {{.total}} VND. Trạng thái: {{.status}}</p>','html','transactional'),
  ('Demo Booking','Lịch demo đã được đặt','<p>Demo vào {{.date}} lúc {{.time}} với {{.rep_name}}.</p>','html','transactional'),
  ('Newsletter Tháng 9','Bản tin tháng 9','<h2>Tin tức tháng 9</h2><p>Nhiều tính năng mới vừa được phát hành...</p>','html','marketing')
) AS tpl(name, subject, body, body_type, type)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');

-- ============================================================
-- EMAIL LOGS (50 entries)
-- ============================================================
INSERT INTO email.email_logs (id, tenant_id, template_id, recipient_email, subject, body, status, provider_message_id, error, sent_at, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  (SELECT id FROM email.email_templates WHERE tenant_id=t.id ORDER BY random() LIMIT 1),
  'lead.' || gs || '@example.com',
  'Email subject #' || gs,
  'Email body content',
  (ARRAY['sent','delivered','opened','clicked','bounced','failed'])[ceil(random()*6)],
  'msg-' || md5(random()::text),
  CASE WHEN random()<0.1 THEN 'smtp_timeout' ELSE NULL END,
  NOW() - (random()*30 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN generate_series(1,20) AS gs
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');

-- ============================================================
-- EMAIL WEBHOOKS (3 per tenant)
-- ============================================================
INSERT INTO email.email_webhooks (id, tenant_id, name, url, events, secret, is_active, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  w.name,
  w.url,
  ARRAY[w.event_1, w.event_2]::TEXT[],
  md5(random()::text),
  true,
  NOW() - (random()*30 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Mailgun Forwarder','https://api.apexfintech.vn/email/inbound','delivered','opened'),
  ('Slack Notifier','https://hooks.slack.com/services/T0/B0/XXX','bounced','failed'),
  ('CRM Sync','https://hooks.zapier.com/abc','clicked','opened')
) AS w(name, url, event_1, event_2)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');

-- ============================================================
-- LANDING PAGES (5 per tenant)
-- ============================================================
INSERT INTO landing.landing_pages (id, tenant_id, slug, title, meta, design_schema, status, published_at, version, created_at, updated_at) VALUES
  ('l0000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','home','Vay Nhanh Online 24/7',
    '{"seo_title":"Apex Fintech - Vay nhanh trong 5 phút","description":"Vay tín chấp online, duyệt trong 5 phút, giải ngân trong ngày"}'::jsonb,
    '{"blocks":[{"type":"hero","title":"Vay nhanh 50 triệu, duyệt trong 5 phút","cta":"Đăng ký ngay"},{"type":"features","items":3},{"type":"form","fields":["name","phone","income"]}]}'::jsonb,
    'published',NOW() - INTERVAL '110 days',3,NOW() - INTERVAL '120 days',NOW() - INTERVAL '5 days'),

  ('l0000001-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000001','vay-mua-xe','Vay Mua Xe Ô Tô',
    '{"seo_title":"Vay mua xe - Lãi suất ưu đãi","description":"Vay mua ô tô với lãi suất chỉ từ 7.5%/năm"}'::jsonb,
    '{"blocks":[{"type":"hero","title":"Vay mua xe ô tô - Lãi suất 7.5%"},{"type":"form","fields":["name","phone","car_brand","car_price"]}]}'::jsonb,
    'published',NOW() - INTERVAL '90 days',2,NOW() - INTERVAL '100 days',NOW() - INTERVAL '10 days'),

  ('l0000001-0000-0000-0000-000000000003','aaaaaaaa-0000-0000-0000-000000000001','vay-mua-nha','Vay Mua Nhà',
    '{"seo_title":"Vay mua nhà - Lãi suất 6.8%/năm","description":"Vay mua nhà với thời hạn tối đa 25 năm"}'::jsonb,
    '{"blocks":[{"type":"hero"},{"type":"form"}]}'::jsonb,
    'draft',NULL,1,NOW() - INTERVAL '30 days',NOW() - INTERVAL '25 days'),

  ('l0000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','can-ho-cao-cap','Căn Hộ Cao Cấp - Vinhomes Grand Park',
    '{"seo_title":"Căn hộ Vinhomes Grand Park - Giá từ 2.5 tỷ","description":"Cơ hội đầu tư căn hộ cao cấp tại TP.HCM"}'::jsonb,
    '{"blocks":[{"type":"hero","title":"Căn hộ Vinhomes Grand Park"},{"type":"gallery"},{"type":"form","fields":["name","phone","budget"]}]}'::jsonb,
    'published',NOW() - INTERVAL '50 days',2,NOW() - INTERVAL '55 days',NOW() - INTERVAL '3 days'),

  ('l0000002-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000002','dat-nen-long-bien','Đất Nền Long Biên - Cơ Hội Đầu Tư',
    '{"seo_title":"Đất nền Long Biên - Sinh lời 30%/năm","description":"Đất nền thổ cư, sổ đỏ lâu dài"}'::jsonb,
    '{"blocks":[{"type":"hero"},{"type":"form"}]}'::jsonb,
    'published',NOW() - INTERVAL '40 days',2,NOW() - INTERVAL '45 days',NOW() - INTERVAL '1 day'),

  ('l0000002-0000-0000-0000-000000000003','aaaaaaaa-0000-0000-0000-000000000002','seminar-thang-9','Seminar Đầu Tư BĐS Tháng 9',
    '{"seo_title":"Seminar đầu tư BĐS - Miễn phí tham dự"}'::jsonb,
    '{"blocks":[{"type":"hero","title":"Seminar Đầu tư BĐS tháng 9"},{"type":"form","fields":["name","email","phone"]}]}'::jsonb,
    'published',NOW() - INTERVAL '20 days',1,NOW() - INTERVAL '30 days',NOW() - INTERVAL '15 days'),

  ('l0000003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','demo-home','Demo Landing Page',
    '{"description":"Demo page"}'::jsonb,
    '{"blocks":[{"type":"hero","title":"Demo RINCO Platform"},{"type":"form","fields":["name","email"]}]}'::jsonb,
    'draft',NULL,1,NOW() - INTERVAL '5 days',NOW())
ON CONFLICT (id) DO NOTHING;

-- CRM landing_pages (legacy crm-service schema)
INSERT INTO crm.landing_pages (id, tenant_id, slug, title, description, domain, status, is_default, seo_settings, theme, blocks, published_at, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  lp.slug,
  lp.title,
  lp.description,
  (SELECT hostname FROM tenant.tenant_domains WHERE tenant_id=t.id LIMIT 1),
  lp.status,
  false,
  lp.seo_settings,
  lp.theme,
  lp.blocks,
  lp.published_at,
  lp.created_at
FROM tenant.tenants t
CROSS JOIN LATERAL (
  SELECT * FROM (VALUES
    ('home-a','Trang Chủ','Trang chủ tổng quan','published','{}'::jsonb,'{}'::jsonb,'[]'::jsonb,NOW() - INTERVAL '100 days'),
    ('landing-spring','Vay Mùa Xuân','Landing page chiến dịch mùa xuân','published','{"theme":"spring"}'::jsonb,'{"color":"#FFB7C5"}'::jsonb,'[]'::jsonb,NOW() - INTERVAL '60 days'),
    ('landing-summer','Vay Mùa Hè','Landing page chiến dịch mùa hè','published','{}'::jsonb,'{}'::jsonb,'[]'::jsonb,NOW() - INTERVAL '30 days'),
    ('landing-blackfriday','Black Friday','Landing page Black Friday','draft','{}'::jsonb,'{}'::jsonb,'[]'::jsonb,NOW() - INTERVAL '5 days'),
    ('default','Default Landing','Default landing page','published','{}'::jsonb,'{}'::jsonb,'[]'::jsonb,NOW() - INTERVAL '100 days')
  ) AS lp(slug, title, description, status, seo_settings, theme, blocks, published_at)
  WHERE lp.slug IN ('home-a','landing-spring','default')
) lp
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT DO NOTHING;

-- ============================================================
-- LANDING FORMS (multiple per tenant)
-- ============================================================
INSERT INTO landing.landing_forms (id, tenant_id, name, fields, submit_action, success_message, is_active, created_at)
SELECT gen_random_uuid(), t.id, f.name, f.fields::jsonb, f.action::jsonb, f.success::jsonb, true, NOW() - (random()*30 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Hero Contact','[{"name":"full_name","type":"text","required":true},{"name":"phone","type":"phone","required":true},{"name":"email","type":"email","required":false},{"name":"loan_purpose","type":"select","options":["Mua nha","Mua xe","Kinh doanh","No tien mat"]}]',
   '{"type":"webhook","url":"https://api.apexfintech.vn/leads/intake"}',
   '{"title":"Cảm ơn bạn!","body":"Chúng tôi sẽ liên hệ trong 30 phút."}'),
  ('Quick Apply','[{"name":"phone","type":"phone","required":true}]',
   '{"type":"webhook","url":"https://api.apexfintech.vn/leads/quick"}',
   '{"title":"Thành công","body":"Yêu cầu đã được gửi."}'),
  ('Newsletter','[{"name":"email","type":"email","required":true}]',
   '{"type":"email_list","list_id":"newsletter_vi"}',
   '{"title":"Đã đăng ký!","body":"Cảm ơn bạn."}')
) AS f(name, fields, action, success)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');

-- ============================================================
-- FB PIXEL CONFIGS (legacy landing-service)
-- ============================================================
INSERT INTO landing.fb_pixel_configs (id, tenant_id, pixel_id, access_token, test_event_code, is_active, created_at)
SELECT gen_random_uuid(), t.id, t.slug || '_pixel_' || (100 + (random()*999)::int), 'EAAB' || md5(random()::text), 'TEST' || md5(random()::text)[:8], true, NOW() - INTERVAL '60 days'
FROM tenant.tenants t
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');

-- ============================================================
-- TRACKING CLICKS (300 entries — for analytics)
-- ============================================================
INSERT INTO landing.tracking_clicks (id, tenant_id, target_url, utm, clicks, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  (ARRAY['https://apexfintech.vn/loan','https://hct.vn/projects','https://demo.rinco.app'])[ceil(random()*3)],
  jsonb_build_object('utm_source',src.utm_source,'utm_campaign',src.utm_campaign,'utm_medium',src.utm_medium),
  (1 + random()*500)::int,
  NOW() - (random()*30 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('facebook','spring_loan','cpc'),
  ('google','q3_personal','cpc'),
  ('tiktok','cash_loan','cpc'),
  ('linkedin','b2b','social'),
  ('email','weekly','email')
) AS src(utm_source, utm_campaign, utm_medium)
CROSS JOIN generate_series(1,20)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');

-- ============================================================
-- META CAPI EVENTS (50 per tenant)
-- ============================================================
INSERT INTO meta_capi.events (id, tenant_id, event_name, event_id, event_time, user_email, user_phone, click_id, action_source, event_source_url, metadata, sent_to_meta, sent_at, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  evt.name,
  gen_random_uuid(),
  NOW() - (random()*30 || ' days')::interval,
  'lead' || gs || '@example.com',
  '+8490' || lpad((7000000 + (random()*999999)::int)::text, 7, '0'),
  CASE WHEN random()<0.7 THEN 'fb.' || md5(random()::text) ELSE NULL END,
  'website',
  'https://' || t.slug || '.rinco.app/landing',
  jsonb_build_object('value',(random()*5)::numeric*1000000,'currency','VND'),
  random()<0.8,
  NOW() - (random()*30 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Lead'),
  ('Purchase'),
  ('CompleteRegistration'),
  ('AddToCart'),
  ('InitiateCheckout')
) AS evt(name)
CROSS JOIN generate_series(1,10) AS gs
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');

-- ============================================================
-- OBSERVABILITY ALERTS (10 per tenant)
-- ============================================================
INSERT INTO observability.alerts (id, tenant_id, name, description, query, severity, channels, is_active, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  al.name,
  al.description,
  al.query,
  al.severity::text,
  ARRAY['email','slack']::TEXT[],
  true,
  NOW() - (random()*30 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('High CPU usage','CPU > 90% for 5 minutes','avg(cpu_usage) > 90','critical'),
  ('High memory usage','Memory > 85% for 5 minutes','avg(mem_usage) > 85','warning'),
  ('API errors spike','error_rate > 5% in 5min','rate(errors[5m]) > 0.05','critical'),
  ('Lead flow drop','Leads/hour < 10','rate(leads[1h]) < 10','warning'),
  ('Database connection pool','pg active connections > 80%','pg_active / pg_max > 0.8','warning'),
  ('Disk usage','Disk > 90%','disk_used / disk_total > 0.9','critical'),
  ('Slow queries','p95 query > 1s','quantile(0.95, query_time) > 1','warning'),
  ('Failed logins spike','failed_login > 50/min','rate(failed_logins[5m]) > 50','warning'),
  ('ScyllaDB latency','scylla p99 > 50ms','quantile(0.99, scylla_latency) > 0.05','critical'),
  ('ClickHouse ingestion lag','lag > 60s','ch_lag > 60','warning')
) AS al(name, description, query, severity)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');

-- ============================================================
-- RECORDINGS (5 per tenant ended rooms)
-- ============================================================
INSERT INTO recordings (id, room_id, tenant_id, title, started_at, ended_at, duration_seconds, size_bytes, s3_key, s3_bucket, tier, format, status, transcript_url, thumbnail_url, created_at)
SELECT
  gen_random_uuid(),
  mr.id,
  t.id,
  mr.title || ' - Recording',
  mr.started_at,
  mr.ended_at,
  mr.duration_seconds,
  (mr.duration_seconds * 250000)::bigint,
  'recordings/' || t.slug || '/' || mr.id || '.mp4',
  'chat-attachments',
  (ARRAY['hot','warm','cold'])[ceil(random()*3)],
  'mp4',
  'ready',
  'https://recordings.rinco.app/' || t.slug || '/' || mr.id || '/transcript.json',
  'https://recordings.rinco.app/' || t.slug || '/' || mr.id || '/thumb.jpg',
  mr.started_at
FROM crm.meeting_rooms mr
JOIN tenant.tenants t ON t.id = mr.tenant_id
WHERE mr.ended_at IS NOT NULL
  AND t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (id) DO NOTHING;

-- Transcript segments (a few per recording)
INSERT INTO transcript_segments (recording_id, start_ms, end_ms, text, confidence, speaker)
SELECT r.id, ms * 5000, ms * 5000 + 4500, sentences.s, 0.95, 'Speaker_' || (ms % 3)
FROM recordings r
CROSS JOIN generate_series(0,20) AS ms
CROSS JOIN (VALUES
  ('Xin chào, cảm ơn anh đã tham gia buổi họp hôm nay.'),
  ('Vâng, mình rất vui khi được tham gia.'),
  ('Hôm nay chúng ta sẽ thảo luận về dự án đầu tư mới.'),
  ('Tôi đã review proposal rồi, khá tốt.'),
  ('Bạn có câu hỏi gì không?'),
  ('Vâng, tôi muốn hỏi về lịch trình triển khai.')
) sentences(s)
WHERE r.started_at > NOW() - INTERVAL '60 days'
LIMIT 100;
