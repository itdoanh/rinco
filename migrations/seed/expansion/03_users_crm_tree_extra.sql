-- ============================================================
-- WS-B LOOP 2: USERS + CRM TREE EXPANSION
-- 5-7 more users per tenant covering Manager/Team Lead/Agent/Viewer
-- Additional departments, sub-pipelines, lead sources
-- ============================================================

-- ============================================================
-- APEX FINTECH: 7 NEW USERS (Team Lead, Senior Agent, Junior, Viewer)
-- ============================================================
INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, phone, parent_id, path, depth, role, roles, permissions, metadata, last_login_at, created_at) VALUES
  -- Team Lead under Sales Manager (a...002)
  ('b0000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','teamlead1@apexfintech.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Hồ Bích Hà','+84904111201','a0000001-0000-0000-0000-000000000002',text2ltree('root.a0000001-0000-0000-0000-000000000001.a0000001-0000-0000-0000-000000000002.b0000001-0000-0000-0000-000000000001'),3,'member',ARRAY['team_lead','agent']::TEXT[],'["leads.read","leads.write","deals.read","reports.read"]'::jsonb,'{"position":"Team Lead HCM","department":"Sales","joined":"2022-06-15"}'::jsonb,NOW() - INTERVAL '4 hours',NOW() - INTERVAL '40 days'),
  ('b0000001-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000001','teamlead2@apexfintech.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Lý Quang Hải','+84904111202','a0000001-0000-0000-0000-000000000002',text2ltree('root.a0000001-0000-0000-0000-000000000001.a0000001-0000-0000-0000-000000000002.b0000001-0000-0000-0000-000000000002'),3,'member',ARRAY['team_lead','agent']::TEXT[],'["leads.read","leads.write","deals.read","reports.read"]'::jsonb,'{"position":"Team Lead HN","department":"Sales","joined":"2022-07-20"}'::jsonb,NOW() - INTERVAL '2 days',NOW() - INTERVAL '38 days'),
  -- Senior agent under teamlead1
  ('b0000001-0000-0000-0000-000000000010','aaaaaaaa-0000-0000-0000-000000000001','senior1@apexfintech.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Phùng Mỹ Linh','+84904111210','b0000001-0000-0000-0000-000000000001',text2ltree('root.a0000001-0000-0000-0000-000000000001.a0000001-0000-0000-0000-000000000002.b0000001-0000-0000-0000-000000000001.b0000001-0000-0000-0000-000000000010'),4,'member',ARRAY['senior_agent']::TEXT[],'["leads.read","leads.write","deals.write"]'::jsonb,'{"position":"Senior Sales","region":"HCM","deals_closed_ytd":18}'::jsonb,NOW() - INTERVAL '6 hours',NOW() - INTERVAL '30 days'),
  -- Junior agents
  ('b0000001-0000-0000-0000-000000000020','aaaaaaaa-0000-0000-0000-000000000001','junior1@apexfintech.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Đặng Thị Thúy','+84904111220','b0000001-0000-0000-0000-000000000001',text2ltree('root.a0000001-0000-0000-0000-000000000001.a0000001-0000-0000-0000-000000000002.b0000001-0000-0000-0000-000000000001.b0000001-0000-0000-0000-000000000020'),4,'member',ARRAY['junior_agent']::TEXT[],'["leads.read","leads.write"]'::jsonb,'{"position":"Junior Sales","region":"HCM","deals_closed_ytd":3}'::jsonb,NOW() - INTERVAL '1 day',NOW() - INTERVAL '20 days'),
  ('b0000001-0000-0000-0000-000000000021','aaaaaaaa-0000-0000-0000-000000000001','junior2@apexfintech.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Trương Quốc Đạt','+84904111221','b0000001-0000-0000-0000-000000000002',text2ltree('root.a0000001-0000-0000-0000-000000000001.a0000001-0000-0000-0000-000000000002.b0000001-0000-0000-0000-000000000002.b0000001-0000-0000-0000-000000000021'),4,'member',ARRAY['junior_agent']::TEXT[],'["leads.read","leads.write"]'::jsonb,'{"position":"Junior Sales","region":"HN","deals_closed_ytd":1}'::jsonb,NOW() - INTERVAL '5 days',NOW() - INTERVAL '15 days'),
  -- Viewer (external stakeholder)
  ('b0000001-0000-0000-0000-000000000030','aaaaaaaa-0000-0000-0000-000000000001','viewer.investor@apexfintech.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Lâm Quốc Bảo','+84904111230','a0000001-0000-0000-0000-000000000001',text2ltree('root.a0000001-0000-0000-0000-000000000001.b0000001-0000-0000-0000-000000000030'),2,'member',ARRAY['viewer']::TEXT[],'["reports.read","dashboards.read"]'::jsonb,'{"position":"Board observer","type":"external"}'::jsonb,NULL,NOW() - INTERVAL '50 days'),
  -- Intern
  ('b0000001-0000-0000-0000-000000000040','aaaaaaaa-0000-0000-0000-000000000001','intern@apexfintech.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Hồ Khánh Linh','+84904111240','a0000001-0000-0000-0000-000000000004',text2ltree('root.a0000001-0000-0000-0000-000000000001.a0000001-0000-0000-0000-000000000004.b0000001-0000-0000-0000-000000000040'),3,'member',ARRAY['intern']::TEXT[],'["leads.read","campaigns.read"]'::jsonb,'{"position":"Marketing Intern","school":"FTU"}'::jsonb,NOW() - INTERVAL '12 hours',NOW() - INTERVAL '10 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- HCT CONSULTING: 6 NEW USERS
-- ============================================================
INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, phone, parent_id, path, depth, role, roles, permissions, metadata, last_login_at, created_at) VALUES
  -- Team Lead BDS
  ('b0000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','teamlead.hn@hct.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Tống Diệu Hương','+84905111201','a0000002-0000-0000-0000-000000000002',text2ltree('root.a0000002-0000-0000-0000-000000000001.a0000002-0000-0000-0000-000000000002.b0000002-0000-0000-0000-000000000001'),3,'member',ARRAY['team_lead','agent']::TEXT[],'["leads.read","leads.write","deals.write","properties.read"]'::jsonb,'{"position":"Team Lead BDS HN","specialty":"Vinhomes","deals_closed_ytd":12}'::jsonb,NOW() - INTERVAL '3 hours',NOW() - INTERVAL '45 days'),
  -- Senior consultant
  ('b0000002-0000-0000-0000-000000000010','aaaaaaaa-0000-0000-0000-000000000002','senior.consultant@hct.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Hà Mạnh Quân','+84905111210','b0000002-0000-0000-0000-000000000001',text2ltree('root.a0000002-0000-0000-0000-000000000001.a0000002-0000-0000-0000-000000000002.b0000002-0000-0000-0000-000000000001.b0000002-0000-0000-0000-000000000010'),4,'member',ARRAY['senior_agent']::TEXT[],'["leads.read","leads.write","deals.write","viewings.schedule"]'::jsonb,'{"position":"Senior Consultant","specialty":"Shophouse cao cấp","deals_closed_ytd":8}'::jsonb,NOW() - INTERVAL '8 hours',NOW() - INTERVAL '35 days'),
  ('b0000002-0000-0000-0000-000000000011','aaaaaaaa-0000-0000-0000-000000000002','senior2@hct.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Phạm Hồng Nhung','+84905111211','b0000002-0000-0000-0000-000000000001',text2ltree('root.a0000002-0000-0000-0000-000000000001.a0000002-0000-0000-0000-000000000002.b0000002-0000-0000-0000-000000000001.b0000002-0000-0000-0000-000000000011'),4,'member',ARRAY['senior_agent']::TEXT[],'["leads.read","leads.write","deals.write"]'::jsonb,'{"position":"Senior Consultant","specialty":"Penthouse","deals_closed_ytd":6}'::jsonb,NOW() - INTERVAL '1 day',NOW() - INTERVAL '30 days'),
  -- Junior consultant
  ('b0000002-0000-0000-0000-000000000020','aaaaaaaa-0000-0000-0000-000000000002','junior1@hct.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Trần Diệu Anh','+84905111220','b0000002-0000-0000-0000-000000000001',text2ltree('root.a0000002-0000-0000-0000-000000000001.a0000002-0000-0000-0000-000000000002.b0000002-0000-0000-0000-000000000001.b0000002-0000-0000-0000-000000000020'),4,'member',ARRAY['junior_agent']::TEXT[],'["leads.read","leads.write"]'::jsonb,'{"position":"Junior Consultant","deals_closed_ytd":2}'::jsonb,NOW() - INTERVAL '6 hours',NOW() - INTERVAL '20 days'),
  -- Viewer (external investor)
  ('b0000002-0000-0000-0000-000000000030','aaaaaaaa-0000-0000-0000-000000000002','viewer.investor@hct.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Ngô Mạnh Hùng','+84905111230','a0000002-0000-0000-0000-000000000001',text2ltree('root.a0000002-0000-0000-0000-000000000001.b0000002-0000-0000-0000-000000000030'),2,'member',ARRAY['viewer']::TEXT[],'["reports.read","properties.read"]'::jsonb,'{"position":"Investor viewer","type":"external"}'::jsonb,NULL,NOW() - INTERVAL '40 days'),
  -- Intern
  ('b0000002-0000-0000-0000-000000000040','aaaaaaaa-0000-0000-0000-000000000002','intern@hct.vn','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Đinh Quốc Việt','+84905111240','a0000002-0000-0000-0000-000000000004',text2ltree('root.a0000002-0000-0000-0000-000000000001.a0000002-0000-0000-0000-000000000004.b0000002-0000-0000-0000-000000000040'),3,'member',ARRAY['intern']::TEXT[],'["leads.read","campaigns.read"]'::jsonb,'{"position":"Marketing Intern","school":"NEU"}'::jsonb,NOW() - INTERVAL '18 hours',NOW() - INTERVAL '12 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DEMO COMPANY: 5 NEW USERS
-- ============================================================
INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, phone, parent_id, path, depth, role, roles, permissions, metadata, last_login_at, created_at) VALUES
  -- Team Lead
  ('b0000003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','teamlead@demo.com','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Phùng Thế Anh','+84906111201','a0000003-0000-0000-0000-000000000002',text2ltree('root.a0000003-0000-0000-0000-000000000001.a0000003-0000-0000-0000-000000000002.b0000003-0000-0000-0000-000000000001'),3,'member',ARRAY['team_lead','agent']::TEXT[],'["leads.read","leads.write","deals.read"]'::jsonb,'{"position":"Team Lead Demo","joined":"2024-08-01"}'::jsonb,NOW() - INTERVAL '1 hour',NOW() - INTERVAL '5 days'),
  -- Senior Agent
  ('b0000003-0000-0000-0000-000000000010','bbbbbbbb-0000-0000-0000-000000000003','senior@demo.com','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Trương Quỳnh Hoa','+84906111210','b0000003-0000-0000-0000-000000000001',text2ltree('root.a0000003-0000-0000-0000-000000000001.a0000003-0000-0000-0000-000000000002.b0000003-0000-0000-0000-000000000001.b0000003-0000-0000-0000-000000000010'),4,'member',ARRAY['senior_agent']::TEXT[],'["leads.read","leads.write","deals.write"]'::jsonb,'{"position":"Senior Demo Specialist"}'::jsonb,NOW() - INTERVAL '4 hours',NOW() - INTERVAL '3 days'),
  -- Junior
  ('b0000003-0000-0000-0000-000000000020','bbbbbbbb-0000-0000-0000-000000000003','junior@demo.com','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Võ Thanh Huy','+84906111220','b0000003-0000-0000-0000-000000000001',text2ltree('root.a0000003-0000-0000-0000-000000000001.a0000003-0000-0000-0000-000000000002.b0000003-0000-0000-0000-000000000001.b0000003-0000-0000-0000-000000000020'),4,'member',ARRAY['junior_agent']::TEXT[],'["leads.read","leads.write"]'::jsonb,'{"position":"Junior Demo"}'::jsonb,NOW() - INTERVAL '2 days',NOW() - INTERVAL '2 days'),
  -- Viewer
  ('b0000003-0000-0000-0000-000000000030','bbbbbbbb-0000-0000-0000-000000000003','viewer@demo.com','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Bùi Khánh Toàn','+84906111230','a0000003-0000-0000-0000-000000000001',text2ltree('root.a0000003-0000-0000-0000-000000000001.b0000003-0000-0000-0000-000000000030'),2,'member',ARRAY['viewer']::TEXT[],'["reports.read"]'::jsonb,'{"position":"Demo investor viewer"}'::jsonb,NULL,NOW() - INTERVAL '2 days'),
  -- Intern
  ('b0000003-0000-0000-0000-000000000040','bbbbbbbb-0000-0000-0000-000000000003','intern2@demo.com','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','Hoàng Mỹ Tâm','+84906111240','a0000003-0000-0000-0000-000000000004',text2ltree('root.a0000003-0000-0000-0000-000000000001.a0000003-0000-0000-0000-000000000004.b0000003-0000-0000-0000-000000000040'),3,'member',ARRAY['intern']::TEXT[],'["leads.read"]'::jsonb,'{"position":"Demo marketing intern"}'::jsonb,NOW() - INTERVAL '12 hours',NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- ADDITIONAL CRM DEPARTMENTS (sub-teams per tenant)
-- ============================================================
INSERT INTO crm.departments (id, tenant_id, name, code, description, manager_id, parent_id, path, created_at, updated_at)
SELECT
  'd' || lpad((2000000 + (gs + t.row_n * 100))::text, 8, '0') || '-0000-0000-0000-000000000000',
  t.id,
  d.name,
  d.code,
  d.description,
  d.manager_id,
  d.parent_id,
  text2ltree(d.parent_path || '.d' || lpad((2000000 + (gs + t.row_n * 100))::text, 8, '0') || '-0000-0000-0000-000000000000'),
  NOW() - INTERVAL '40 days',
  NOW()
FROM tenant.tenants t
CROSS JOIN LATERAL (
  SELECT * FROM (VALUES
    (1, 'Sales HCM', 'SLS-HCM', 'Team sales khu vực TP.HCM', 'b0000001-0000-0000-0000-000000000001'::uuid, 'd1000001-0000-0000-0000-000000000002', 'd1000001-0000-0000-0000-000000000001.d1000001-0000-0000-0000-000000000002'),
    (2, 'Sales HN', 'SLS-HN', 'Team sales khu vực Hà Nội', 'b0000001-0000-0000-0000-000000000002'::uuid, 'd1000001-0000-0000-0000-000000000002', 'd1000001-0000-0000-0000-000000000001.d1000001-0000-0000-0000-000000000002'),
    (3, 'Underwriting Team', 'UW-TEAM', 'Team thẩm định khoản vay', 'a0000001-0000-0000-0000-000000000006'::uuid, 'd1000001-0000-0000-0000-000000000005', 'd1000001-0000-0000-0000-000000000001.d1000001-0000-0000-0000-000000000005')
  ) AS v(gs, name, code, description, manager_id, parent_id, parent_path)
) d
CROSS JOIN LATERAL (
  SELECT ROW_NUMBER() OVER() AS row_n FROM generate_series(1,1)
) gs_arr
WHERE t.id = 'aaaaaaaa-0000-0000-0000-000000000001'
ON CONFLICT (id) DO NOTHING;

INSERT INTO crm.departments (id, tenant_id, name, code, description, manager_id, parent_id, path, created_at, updated_at)
SELECT
  'd' || lpad((3000000 + (gs + t.row_n * 100))::text, 8, '0') || '-0000-0000-0000-000000000000',
  t.id,
  d.name,
  d.code,
  d.description,
  d.manager_id,
  d.parent_id,
  text2ltree(d.parent_path || '.d' || lpad((3000000 + (gs + t.row_n * 100))::text, 8, '0') || '-0000-0000-0000-000000000000'),
  NOW() - INTERVAL '40 days',
  NOW()
FROM tenant.tenants t
CROSS JOIN LATERAL (
  SELECT * FROM (VALUES
    (1, 'BDS Consultants HN', 'BDS-HN', 'Tư vấn BĐS khu vực Hà Nội', 'b0000002-0000-0000-0000-000000000001'::uuid, 'd1000002-0000-0000-0000-000000000002', 'd1000002-0000-0000-0000-000000000001.d1000002-0000-0000-0000-000000000002'),
    (2, 'Investment Advisors', 'INV-ADV', 'Cố vấn đầu tư BĐS', 'a0000002-0000-0000-0000-000000000003'::uuid, 'd1000002-0000-0000-0000-000000000003', 'd1000002-0000-0000-0000-000000000001.d1000002-0000-0000-0000-000000000003')
  ) AS v(gs, name, code, description, manager_id, parent_id, parent_path)
) d
CROSS JOIN LATERAL (
  SELECT ROW_NUMBER() OVER() AS row_n FROM generate_series(1,1)
) gs_arr
WHERE t.id = 'aaaaaaaa-0000-0000-0000-000000000002'
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- CRM COMPANIES EXTRA (15+ extra companies across tenants)
-- ============================================================
INSERT INTO companies (id, tenant_id, name, industry, size, website, phone, email, address, city, country, custom_fields, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  c.name,
  c.industry,
  c.size,
  c.website,
  c.phone,
  c.email,
  c.address,
  c.city,
  'Vietnam',
  jsonb_build_object('source','demo_expansion'),
  NOW() - ((random()*90)::int || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Techcombank','Banking','enterprise','https://techcombank.com','+842839222222','contact@techcombank.com','191 Ba Trieu, Hai Ba Trung','Ha Noi'),
  ('VPBank','Banking','enterprise','https://vpbank.com.vn','+842839006699','info@vpbank.com.vn','Lang Ha, Dong Da','Ha Noi'),
  ('MB Bank','Banking','enterprise','https://mbbank.com.vn','+842837644268','info@mbbank.com','21 Cat Linh, Dong Da','Ha Noi'),
  ('ACB','Banking','enterprise','https://acb.com.vn','+842839006688','info@acb.com.vn','442 Nguyen Thi Minh Khai','Ho Chi Minh'),
  ('BIDV','Banking','enterprise','https://bidv.com.vn','+842842004368','contact@bidv.com.vn','35 Hang Chuoi, Hai Ba Trung','Ha Noi'),
  ('Masan','Conglomerate','enterprise','https://masangroup.com','+842862513025','info@masangroup.com','Suite 802, 8th Fl, Capital Place','Ha Noi'),
  ('FPT Retail','Retail','large','https://fpt.com.vn','+842873012345','contact@fptretail.com','261 Tran Hung Dao','Ho Chi Minh'),
  ('Ho Chi Minh City Development JSC','Real Estate','enterprise','https://hdcd.com.vn','+842838229922','info@hdc.com.vn','60 Vo Van Kiet','Ho Chi Minh'),
  ('Novaland Group','Real Estate','enterprise','https://novaland.com.vn','+842838229900','info@novaland.com.vn','65 Nguyen Du','Ho Chi Minh'),
  ('Tan Hoang Minh Group','Real Estate','large','https://thm.com.vn','+842839333333','info@thm.com.vn','22 Hang Bai, Hoan Kiem','Ha Noi'),
  ('Dat Xanh Group','Real Estate','large','https://datxanh.com.vn','+842839002020','info@datxanh.com.vn','27 Vo Thi Sau','Ho Chi Minh'),
  ('Khang Dien House','Real Estate','large','https://khangdien.com.vn','+842839009999','info@khangdien.com.vn','20 Pham Hung, Nam Tu Liem','Ha Noi'),
  ('Hung Thinh Land','Real Estate','large','https://hungthinhland.com','+842839228888','info@hungthinhland.com','53 Truong Son, Tan Binh','Ho Chi Minh'),
  ('Sovico Group','Conglomerate','enterprise','https://sovico.vn','+842835210101','info@sovico.vn','12A Pham Huy Thong','Ha Noi'),
  ('HAGL Agrico','Agriculture','large','https://hagl.com.vn','+842662220222','contact@hagl.com.vn','15 Hoang Hoa Tham, Pleiku','Gia Lai')
) AS c(name, industry, size, website, phone, email, address, city)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- CONTACTS (legacy crm-service) - 100+ entries per tenant
-- ============================================================
INSERT INTO contacts (id, tenant_id, first_name, last_name, email, phone, company_id, job_title, source, lead_source, owner_user_id, address, city, country, tags, custom_fields, created_at, updated_at)
SELECT
  gen_random_uuid(),
  t.id,
  split_part(p.full_name, ' ', 1),
  split_part(p.full_name, ' ', 2),
  lower(replace(p.full_name, ' ', '')) || p.email_suffix,
  '+849' || (random()*9000000 + 1000000)::int::text,
  (SELECT id FROM companies WHERE tenant_id = t.id ORDER BY random() LIMIT 1),
  p.job_title,
  p.source,
  p.source,
  (SELECT id FROM auth.users WHERE tenant_id = t.id AND role = 'member' AND parent_id IS NOT NULL ORDER BY random() LIMIT 1),
  (random()*999)::int || ' ' || p.street || ', ' || p.district,
  p.city,
  'Vietnam',
  ARRAY[p.tag1, p.tag2],
  jsonb_build_object('age',(22 + random()*40)::int,'industry',p.industry),
  NOW() - ((random()*60)::int || ' days')::interval,
  NOW()
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Nguyễn Văn An','an.nguyen','@gmail.com','CEO','Facebook Ads','Hanoi Tech Park','Cau Giay','Ha Noi','technology','VIP','Hot'),
  ('Trần Thị Bình','binh.tran','@yahoo.com','CFO','Google Ads','Tran Hung Dao','Quan 1','Ho Chi Minh','finance','Hot','Follow-up'),
  ('Lê Hoàng Cường','cuong.le','@outlook.com','Marketing Manager','LinkedIn','Ly Tu Trong','Quan 1','Ho Chi Minh','marketing','VIP','Cold'),
  ('Phạm Thị Dung','dung.pham','@company.vn','CTO','TikTok Ads','Nguyen Hue','Quan 1','Ho Chi Minh','technology','Hot','Enterprise'),
  ('Hoàng Văn Em','em.hoang','@gmail.com','COO','Referral','Ba Trieu','Quan 3','Ho Chi Minh','operations','SMB','Follow-up'),
  ('Vũ Thị Phương','phuong.vu','@vnn.vn','Sales Manager','Zalo OA','Le Loi','Quan 1','Ho Chi Minh','sales','Hot','VIP'),
  ('Đặng Văn Giang','giang.dang','@gmail.com','Director','Facebook Ads','Dien Bien Phu','Ba Dinh','Ha Noi','finance','VIP','Hot'),
  ('Bùi Thị Hoa','hoa.bui','@yahoo.com','VP Sales','Google Ads','Tran Phu','Hai Ba Trung','Ha Noi','sales','Enterprise','Follow-up'),
  ('Đỗ Văn Ích','ich.do','@outlook.com','Product Manager','LinkedIn','Phan Xich Long','Phu Nhuan','Ho Chi Minh','technology','VIP','Hot'),
  ('Ngô Thị Khánh','khanh.ngo','@company.vn','HR Manager','TikTok Ads','Vo Thi Sau','Quan 3','Ho Chi Minh','hr','SMB','Cold'),
  ('Hồ Văn Lâm','lam.ho','@gmail.com','Investment Manager','Referral','Trang Tien','Hoan Kiem','Ha Noi','finance','VIP','Enterprise'),
  ('Tô Thị Mai','mai.to','@vnn.vn','BDS Consultant','Zalo OA','Phan Chu Trinh','Hoan Kiem','Ha Noi','real_estate','Hot','VIP'),
  ('Phùng Văn Nam','nam.phung','@gmail.com','Analyst','Facebook Ads','Xuan Thuy','Cau Giay','Ha Noi','finance','Trial','Follow-up'),
  ('Lý Thị Oanh','oanh.ly','@yahoo.com','Designer','Google Ads','Hai Ba Trung','Quan 3','Ho Chi Minh','creative','SMB','Cold'),
  ('Trương Văn Phúc','phuc.truong','@outlook.com','Fullstack Dev','LinkedIn','Nguyen Trai','Quan 1','Ho Chi Minh','technology','VIP','Hot')
) AS p(full_name, short_name, email_suffix, job_title, source, street, district, city, industry, tag1, tag2)
CROSS JOIN generate_series(1, 8) AS gs
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
LIMIT 360
ON CONFLICT (id) DO NOTHING;

-- Sync new users into crm.user_hierarchy mirror
INSERT INTO crm.user_hierarchy (id, tenant_id, user_id, manager_id, parent_id, path, depth, position, is_manager, start_date, metadata)
SELECT
  id,
  tenant_id,
  id AS user_id,
  parent_id,
  NULL::UUID,
  path,
  depth,
  COALESCE(metadata->>'position','Member'),
  (role IN ('manager','tenant_admin','super_admin')),
  (created_at)::date,
  metadata
FROM auth.users
WHERE tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
  AND id LIKE 'b000000%'
ON CONFLICT (id) DO NOTHING;

-- Stats
SELECT 'expansion_03_users_crm_tree (WS-B Loop 2)' AS section, COUNT(*) AS new_users
FROM auth.users
WHERE tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
  AND id LIKE 'b000000%';
