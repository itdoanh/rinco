-- ============================================================
-- LEGACY SERVICE TABLES (lead-service, crm-service, dynamic_model) - Loop 202
-- These services often run on separate databases. Idempotent.
-- ============================================================

-- ============================================================
-- LEADS (legacy lead-service migrations/0003_leads.sql schema)
-- ============================================================
INSERT INTO leads (id, tenant_id, source_id, owner_user_id, pipeline_id, stage_id, full_name, email, phone, company_name, job_title, status, score, score_tier, estimated_value, custom_fields, utm, ip, user_agent, referrer, fbclid, gclid, ttclid, landing_page, campaign_id, tags, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000001',
  (SELECT id FROM lead_sources WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001' ORDER BY random() LIMIT 1),
  (SELECT id FROM auth.users WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001' AND role='member' ORDER BY random() LIMIT 1),
  '11000001-0000-0000-0000-000000000001',
  CASE (SELECT stage FROM leads.leads WHERE id = (SELECT id FROM leads.leads WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001' LIMIT 1)) WHEN 'new' THEN '21000001-0000-0000-0000-000000000001' WHEN 'contacted' THEN '21000001-0000-0000-0000-000000000002' WHEN 'qualified' THEN '21000001-0000-0000-0000-000000000003' WHEN 'proposal' THEN '21000001-0000-0000-0000-000000000004' WHEN 'won' THEN '21000001-0000-0000-0000-000000000005' ELSE '21000001-0000-0000-0000-000000000006' END,
  'L ' || gs || ' - ' || person.full_name,
  lower(replace(person.full_name,' ','')) || '@legacy.com',
  '+8490' || lpad((8000000 + gs + (random()*99999)::int)::text, 7, '0'),
  'Legacy Co ' || gs,
  (ARRAY['CEO','CTO','Marketing Manager','Sales Lead'])[ceil(random()*4)],
  'new',
  (random()*100)::numeric(5,2),
  CASE WHEN random()<0.3 THEN 'hot' WHEN random()<0.6 THEN 'warm' ELSE 'cold' END,
  (random()*5)::numeric(15,2)*1000000,
  '{"source":"facebook"}'::jsonb,
  '{"utm_source":"facebook","utm_medium":"cpc"}'::jsonb,
  ('203.0.113.' || (10 + (random()*200)::int)::text)::inet,
  'Mozilla/5.0',
  'https://facebook.com',
  CASE WHEN random()<0.7 THEN 'fb.' || md5(random()::text) ELSE NULL END,
  CASE WHEN random()<0.3 THEN 'g.' || md5(random()::text) ELSE NULL END,
  CASE WHEN random()<0.2 THEN 'tt.' || md5(random()::text) ELSE NULL END,
  'https://apexfintech.vn/landing/home',
  'campaign-' || gs,
  ARRAY['hot','facebook']::TEXT[],
  NOW() - (random()*20 || ' days')::interval,
  NOW()
FROM (
  SELECT unnest(ARRAY['Nguyen Van A','Tran Thi B','Le Van C','Pham Thi D','Hoang Van E','Vu Thi F','Dang Van G','Bui Thi H']) AS full_name
) person
CROSS JOIN generate_series(1,15) AS gs
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- LEAD NOTES (lead-service legacy)
-- ============================================================
INSERT INTO lead_notes (id, tenant_id, lead_id, author_id, body, is_pinned, created_at)
SELECT
  gen_random_uuid(),
  l.tenant_id,
  l.id,
  l.owner_user_id,
  (ARRAY['Gọi điện intro','Follow up sau email','Đã gửi quote','Khách hẹn meeting tuần sau','Cần check income'])[ceil(random()*5)],
  random()<0.2,
  NOW() - (random()*15 || ' days')::interval
FROM leads l
WHERE l.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
LIMIT 30
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- LEAD ACTIVITIES (lead-service legacy)
-- ============================================================
INSERT INTO lead_activities (id, tenant_id, lead_id, type, payload, actor_id, description, created_at)
SELECT
  gen_random_uuid(),
  l.tenant_id,
  l.id,
  (ARRAY['CALL','EMAIL','MEETING','NOTE','SCORE_UPDATE','ASSIGN'])[ceil(random()*6)],
  jsonb_build_object('result','contacted','duration_min', 5+random()*55),
  l.owner_user_id,
  'Legacy activity',
  NOW() - (random()*15 || ' days')::interval
FROM leads l
WHERE l.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
LIMIT 40
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- LEAD ASSIGNMENTS (lead-service legacy)
-- ============================================================
INSERT INTO lead_assignments (id, tenant_id, lead_id, from_user_id, to_user_id, reason, assigned_by, created_at)
SELECT
  gen_random_uuid(),
  l.tenant_id,
  l.id,
  NULL,
  l.owner_user_id,
  'Auto round-robin assignment',
  NULL,
  NOW() - (random()*15 || ' days')::interval
FROM leads l
WHERE l.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
LIMIT 20
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- LEAD STAGE HISTORY (lead-service legacy)
-- ============================================================
INSERT INTO lead_stage_history (id, tenant_id, lead_id, from_stage, to_stage, changed_by, notes, changed_at)
SELECT
  gen_random_uuid(),
  l.tenant_id,
  l.id,
  'new',
  'contacted',
  l.owner_user_id,
  'Auto stage move',
  NOW() - (random()*15 || ' days')::interval
FROM leads l
WHERE l.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
LIMIT 20
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- CLICKHOUSE EVENTS META (lead-service 0006)
-- ============================================================
INSERT INTO clickhouse_events (id, tenant_id, event_type, lead_id, user_id, properties, event_date, synced_at, created_at)
SELECT
  gen_random_uuid(),
  l.tenant_id,
  evt.name,
  l.id,
  l.owner_user_id,
  jsonb_build_object('source',l.utm->>'utm_source','score',l.score),
  CURRENT_DATE - (random()*30)::int,
  NOW() - INTERVAL '1 hour',
  NOW() - (random()*30 || ' days')::interval
FROM leads l
CROSS JOIN (VALUES ('lead.created'),('lead.viewed'),('lead.contacted'),('lead.score_updated')) evt(name)
WHERE l.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
LIMIT 80
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- CONTACTS (legacy crm-service)
-- ============================================================
INSERT INTO contacts (id, tenant_id, company_id, first_name, last_name, email, phone, job_title, department, owner_user_id, status, source, custom_fields, tags, last_contacted_at, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  (SELECT id FROM companies WHERE tenant_id=t.id ORDER BY random() LIMIT 1),
  split_part(person.full_name,' ',1),
  split_part(person.full_name,' ',2),
  lower(replace(person.full_name,' ','')) || '@contact.vn',
  '+8490' || lpad((8000000 + (random()*999999)::int)::text, 7, '0'),
  (ARRAY['CEO','CTO','CFO','Manager','Director'])[ceil(random()*5)],
  (ARRAY['Sales','Operations','Marketing','IT'])[ceil(random()*4)],
  (SELECT id FROM auth.users WHERE tenant_id=t.id AND role='member' ORDER BY random() LIMIT 1),
  'active',
  (ARRAY['organic','referral','social','ads','direct'])[ceil(random()*5)],
  jsonb_build_object('seniority', (ARRAY['junior','mid','senior'])[ceil(random()*3)]),
  ARRAY['hot','vip']::TEXT[],
  NOW() - (random()*15 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Nguyen Van A'),('Tran Thi B'),('Le Van C'),('Pham Thi D'),('Hoang Van E'),
  ('Vu Thi F'),('Dang Van G'),('Bui Thi H'),('Do Van I'),('Ngo Thi J'),
  ('Ho Van K'),('To Thi L'),('Phung Van M'),('Ly Thi N'),('Truong Van O'),
  ('Cao Thi P'),('Dinh Van Q'),('Duong Thi R'),('Phan Van S'),('Chau Thi T'),
  ('Tong Van U'),('Ha Thi V'),('Lai Van W'),('Trinh Thi X'),('Mai Van Y'),
  ('Doan Thi Z'),('Ta Van AA'),('Vo Thi AB'),('Luu Van AC'),('Kieu Thi AD')
) person(full_name)
CROSS JOIN generate_series(1,2) gs
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DEALS (legacy crm-service)
-- ============================================================
INSERT INTO deals (id, tenant_id, contact_id, owner_user_id, name, value, currency, stage, probability, expected_close_date, custom_fields, created_at)
SELECT
  gen_random_uuid(),
  c.tenant_id,
  c.id,
  c.owner_user_id,
  'Deal - ' || c.last_name || ' - ' || (ARRAY['New','Renewal','Upsell'])[ceil(random()*3)],
  (1 + random()*99)::numeric(15,2)*1000000,
  'VND',
  (ARRAY['prospecting','qualification','proposal','negotiation','won','lost','on_hold'])[ceil(random()*7)],
  (random()*100)::int,
  CURRENT_DATE + (random()*60)::int,
  '{}'::jsonb,
  c.created_at + INTERVAL '3 days'
FROM contacts c
LIMIT 50
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DYNAMIC RECORDS (workflow.dynamic_records - Postgres mirror of Mongo)
-- ============================================================
INSERT INTO workflow.dynamic_records (id, tenant_id, entity_code, owner_user_id, data, created_at, updated_at)
SELECT
  gen_random_uuid(),
  t.id,
  'real_estate',
  (SELECT id FROM auth.users WHERE tenant_id=t.id AND role='member' ORDER BY random() LIMIT 1),
  jsonb_build_object(
    'title', p.title,
    'location', p.location,
    'price', (1 + random()*49)::numeric(18,2) * 1000000000,
    'bedrooms', p.bedrooms,
    'owner_name', p.full_name,
    'owner_phone', '+8490' || lpad((8000000 + (random()*999999)::int)::text, 7, '0')
  ),
  NOW() - (random()*30 || ' days')::interval,
  NOW()
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Vinhomes Grand Park - 2PN','TP.HCM',2),
  ('Masterise Eco Smart City - 3PN','TP.HCM',3),
  ('Sun Group Long Biên','Ha Noi',2),
  ('The Matrix One - 4PN','Ha Noi',4),
  ('Imperia Smart City - 1PN','Ha Noi',1),
  ('Vinhomes Ocean Park - 3PN','Ha Noi',3),
  ('Vinhomes Symphony - 2PN','Ha Noi',2),
  ('The Maris - 2PN','Da Nang',2),
  ('Apec Mandala - Studio','Da Nang',1),
  ('Aqua City - 2PN','Dong Nai',2)
) p(title, location, bedrooms)
CROSS JOIN generate_series(1,3) gs
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');

-- ============================================================
-- INVITE LINKS (crm-service 0009)
-- ============================================================
INSERT INTO invite_links (id, tenant_id, parent_user_id, target_role, token, max_uses, current_uses, expires_at, created_by, created_at)
SELECT
  gen_random_uuid(),
  u.tenant_id,
  u.id,
  'member',
  md5(random()::text),
  5,
  (random()*5)::int,
  NOW() + INTERVAL '7 days',
  u.id,
  NOW() - (random()*7 || ' days')::interval
FROM auth.users u
WHERE u.tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
  AND u.role = 'manager'
LIMIT 10
ON CONFLICT (token) DO NOTHING;
