-- ============================================================
-- CRM TREE SEED (Loop 202)
-- Departments + Pipelines + Lead sources per tenant
-- ============================================================

-- ========== DEPARTMENTS (per tenant) ==========
INSERT INTO crm.departments (id, tenant_id, name, code, description, manager_id, parent_id, path, created_at, updated_at) VALUES
  -- Apex Fintech org
  ('d1000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','Board of Directors','BOD',NULL,'a0000001-0000-0000-0000-000000000001',NULL,text2ltree('d1000001-0000-0000-0000-000000000001'),NOW() - INTERVAL '110 days',NOW()),
  ('d1000001-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000001','Sales','SLS','Tư vấn & chốt deals tài chính','a0000001-0000-0000-0000-000000000002','d1000001-0000-0000-0000-000000000001',text2ltree('d1000001-0000-0000-0000-000000000001.d1000001-0000-0000-0000-000000000002'),NOW() - INTERVAL '110 days',NOW()),
  ('d1000001-0000-0000-0000-000000000003','aaaaaaaa-0000-0000-0000-000000000001','Customer Support','CS','Hỗ trợ khách hàng sau bán','a0000001-0000-0000-0000-000000000003','d1000001-0000-0000-0000-000000000001',text2ltree('d1000001-0000-0000-0000-000000000001.d1000001-0000-0000-0000-000000000003'),NOW() - INTERVAL '110 days',NOW()),
  ('d1000001-0000-0000-0000-000000000004','aaaaaaaa-0000-0000-0000-000000000001','Marketing','MKT','Digital marketing & branding','a0000001-0000-0000-0000-000000000004','d1000001-0000-0000-0000-000000000001',text2ltree('d1000001-0000-0000-0000-000000000001.d1000001-0000-0000-0000-000000000004'),NOW() - INTERVAL '110 days',NOW()),
  ('d1000001-0000-0000-0000-000000000005','aaaaaaaa-0000-0000-0000-000000000001','Risk & Underwriting','RISK','Phê duyệt khoản vay','a0000001-0000-0000-0000-000000000006','d1000001-0000-0000-0000-000000000001',text2ltree('d1000001-0000-0000-0000-000000000001.d1000001-0000-0000-0000-000000000005'),NOW() - INTERVAL '110 days',NOW()),
  -- HCT org
  ('d1000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','Ban Lãnh Đạo','BOD',NULL,'a0000002-0000-0000-0000-000000000001',NULL,text2ltree('d1000002-0000-0000-0000-000000000001'),NOW() - INTERVAL '55 days',NOW()),
  ('d1000002-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000002','Tư Vấn BĐS','BDS','Tư vấn dự án BĐS cao cấp','a0000002-0000-0000-0000-000000000002','d1000002-0000-0000-0000-000000000001',text2ltree('d1000002-0000-0000-0000-000000000001.d1000002-0000-0000-0000-000000000002'),NOW() - INTERVAL '55 days',NOW()),
  ('d1000002-0000-0000-0000-000000000003','aaaaaaaa-0000-0000-0000-000000000002','Phòng Đầu Tư','INV','Phân tích & đầu tư BĐS','a0000002-0000-0000-0000-000000000003','d1000002-0000-0000-0000-000000000001',text2ltree('d1000002-0000-0000-0000-000000000001.d1000002-0000-0000-0000-000000000003'),NOW() - INTERVAL '55 days',NOW()),
  ('d1000002-0000-0000-0000-000000000004','aaaaaaaa-0000-0000-0000-000000000002','Marketing','MKT','Quảng cáo & nội dung','a0000002-0000-0000-0000-000000000004','d1000002-0000-0000-0000-000000000001',text2ltree('d1000002-0000-0000-0000-000000000001.d1000002-0000-0000-0000-000000000004'),NOW() - INTERVAL '55 days',NOW()),
  ('d1000002-0000-0000-0000-000000000005','aaaaaaaa-0000-0000-0000-000000000002','Chăm Sóc Khách Hàng','CSKH','Hỗ trợ khách sau bán','a0000002-0000-0000-0000-000000000040','d1000002-0000-0000-0000-000000000001',text2ltree('d1000002-0000-0000-0000-000000000001.d1000002-0000-0000-0000-000000000005'),NOW() - INTERVAL '55 days',NOW()),
  -- Demo org
  ('d1000003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','Board','BOD',NULL,'a0000003-0000-0000-0000-000000000001',NULL,text2ltree('d1000003-0000-0000-0000-000000000001'),NOW() - INTERVAL '9 days',NOW()),
  ('d1000003-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003','Sales','SLS',NULL,'a0000003-0000-0000-0000-000000000002','d1000003-0000-0000-0000-000000000001',text2ltree('d1000003-0000-0000-0000-000000000001.d1000003-0000-0000-0000-000000000002'),NOW() - INTERVAL '9 days',NOW()),
  ('d1000003-0000-0000-0000-000000000003','bbbbbbbb-0000-0000-0000-000000000003','Operations','OPS',NULL,'a0000003-0000-0000-0000-000000000003','d1000003-0000-0000-0000-000000000001',text2ltree('d1000003-0000-0000-0000-000000000001.d1000003-0000-0000-0000-000000000003'),NOW() - INTERVAL '9 days',NOW()),
  ('d1000003-0000-0000-0000-000000000004','bbbbbbbb-0000-0000-0000-000000000003','Marketing','MKT',NULL,'a0000003-0000-0000-0000-000000000004','d1000003-0000-0000-0000-000000000001',text2ltree('d1000003-0000-0000-0000-000000000001.d1000003-0000-0000-0000-000000000004'),NOW() - INTERVAL '9 days',NOW())
ON CONFLICT (id) DO NOTHING;

-- ========== LEAD SOURCES (per tenant) ==========
INSERT INTO crm.lead_sources (id, tenant_id, name, utm_source, utm_medium, is_active, created_at)
SELECT gen_random_uuid(), t.id, s.name, s.utm_source, s.utm_medium, true, NOW() - INTERVAL '60 days'
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('Facebook Ads','facebook','cpc'),
  ('Google Ads','google','cpc'),
  ('TikTok Ads','tiktok','cpc'),
  ('Zalo OA','zalo','social'),
  ('Website Organic','google','organic'),
  ('Referral','referral','referral'),
  ('LinkedIn','linkedin','social'),
  ('Email Campaign','mailchimp','email'),
  ('Walk-in','direct','none'),
  ('Seminh','event','offline')
) AS s(name, utm_source, utm_medium)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');

-- ========== PIPELINES (per tenant) ==========
INSERT INTO crm.pipelines (id, tenant_id, name, is_default, stages, created_at, updated_at) VALUES
  -- Apex Fintech: Sales pipeline
  ('c1000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','Loan Application Sales Funnel',true,
    '[{"key":"new","label":"Mới tiếp nhận","color":"#94A3B8","probability":5},{"key":"contacted","label":"Đã liên hệ","color":"#60A5FA","probability":15},{"key":"qualified","label":"Đủ điều kiện","color":"#34D399","probability":35},{"key":"proposal","label":"Gửi đề xuất","color":"#FBBF24","probability":55},{"key":"underwriting","label":"Thẩm định","color":"#A78BFA","probability":75},{"key":"won","label":"Phê duyệt","color":"#10B981","probability":100},{"key":"lost","label":"Từ chối","color":"#EF4444","probability":0}]'::jsonb,
    NOW() - INTERVAL '100 days',NOW() - INTERVAL '10 days'),
  -- HCT: Real Estate pipeline
  ('c1000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','BĐS Investment Funnel',true,
    '[{"key":"new","label":"Mới quan tâm","color":"#94A3B8","probability":10},{"key":"contacted","label":"Đã tư vấn","color":"#60A5FA","probability":20},{"key":"qualified","label":"Khả thi","color":"#34D399","probability":40},{"key":"viewing","label":"Đi xem","color":"#FBBF24","probability":60},{"key":"negotiation","label":"Đàm phán","color":"#A78BFA","probability":80},{"key":"won","label":"Chốt deal","color":"#10B981","probability":100},{"key":"lost","label":"Mất deal","color":"#EF4444","probability":0}]'::jsonb,
    NOW() - INTERVAL '55 days',NOW() - INTERVAL '2 days'),
  -- Demo: Generic
  ('c1000003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','Demo Sales Funnel',true,
    '[{"key":"new","label":"New","color":"#94A3B8","probability":5},{"key":"contacted","label":"Contacted","color":"#60A5FA","probability":20},{"key":"qualified","label":"Qualified","color":"#34D399","probability":45},{"key":"proposal","label":"Proposal","color":"#FBBF24","probability":65},{"key":"won","label":"Won","color":"#10B981","probability":100},{"key":"lost","label":"Lost","color":"#EF4444","probability":0}]'::jsonb,
    NOW() - INTERVAL '8 days',NOW())
ON CONFLICT (id) DO NOTHING;

-- ========== LEAD PIPELINES (lead-service migration 0005) ==========
-- Map to lead-service pipelines (some services duplicate this table)
INSERT INTO pipelines (id, tenant_id, name, description, is_default, color, created_at, updated_at) VALUES
  ('11000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','Loan Funnel','Khoản vay mới',true,'#0F766E',NOW() - INTERVAL '90 days',NOW()),
  ('11000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','BDS Funnel','BĐS cao cấp',true,'#1E40AF',NOW() - INTERVAL '50 days',NOW()),
  ('11000003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','Demo Funnel','Test pipeline',true,'#7C3AED',NOW() - INTERVAL '8 days',NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO pipeline_stages (id, pipeline_id, tenant_id, name, display_order, probability, color, is_won, is_lost, created_at) VALUES
  ('21000001-0000-0000-0000-000000000001','11000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','New',0,5,'#94A3B8',false,false,NOW() - INTERVAL '90 days'),
  ('21000001-0000-0000-0000-000000000002','11000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','Contacted',1,15,'#60A5FA',false,false,NOW() - INTERVAL '90 days'),
  ('21000001-0000-0000-0000-000000000003','11000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','Qualified',2,35,'#34D399',false,false,NOW() - INTERVAL '90 days'),
  ('21000001-0000-0000-0000-000000000004','11000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','Proposal',3,55,'#FBBF24',false,false,NOW() - INTERVAL '90 days'),
  ('21000001-0000-0000-0000-000000000005','11000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','Won',4,100,'#10B981',true,false,NOW() - INTERVAL '90 days'),
  ('21000001-0000-0000-0000-000000000006','11000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','Lost',5,0,'#EF4444',false,true,NOW() - INTERVAL '90 days'),
  ('21000002-0000-0000-0000-000000000001','11000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','Inquiry',0,10,'#94A3B8',false,false,NOW() - INTERVAL '50 days'),
  ('21000002-0000-0000-0000-000000000002','11000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','Viewing',1,30,'#60A5FA',false,false,NOW() - INTERVAL '50 days'),
  ('21000002-0000-0000-0000-000000000003','11000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','Negotiation',2,70,'#FBBF24',false,false,NOW() - INTERVAL '50 days'),
  ('21000002-0000-0000-0000-000000000004','11000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','Closed',3,100,'#10B981',true,false,NOW() - INTERVAL '50 days'),
  ('21000003-0000-0000-0000-000000000001','11000003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','New',0,10,'#94A3B8',false,false,NOW() - INTERVAL '8 days'),
  ('21000003-0000-0000-0000-000000000002','11000003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','Won',1,100,'#10B981',true,false,NOW() - INTERVAL '8 days');

-- ========== CRM COMPANIES (legacy crm-service) ==========
INSERT INTO companies (id, tenant_id, name, industry, size, website, phone, email, address, city, country, custom_fields, created_at) VALUES
  ('c0000001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','FPT Software','Technology','large','https://fptsoftware.com','+84287301234','contact@fptsoftware.com','17 Duy Tan, Cau Giay','Ha Noi','Vietnam','{"deal_size":"1B+"}'::jsonb,NOW() - INTERVAL '80 days'),
  ('c0000001-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000001','Viettel','Telecom','enterprise','https://viettel.vn','+84286322234','crm@viettel.vn','1 Tran Huu Duc','Ha Noi','Vietnam','{"deal_size":"5B+"}'::jsonb,NOW() - INTERVAL '75 days'),
  ('c0000001-0000-0000-0000-000000000003','aaaaaaaa-0000-0000-0000-000000000001','VNG Corporation','Tech','large','https://vng.com.vn','+84287305678','biz@vng.com.vn','Z06 13A, Tan Thuan','Ho Chi Minh','Vietnam','{"deal_size":"500M"}'::jsonb,NOW() - INTERVAL '70 days'),
  ('c0000001-0000-0000-0000-000000000004','aaaaaaaa-0000-0000-0000-000000000001','Vietcombank','Banking','enterprise','https://vcb.com.vn','+842839342342','hello@vcb.com.vn','198 Tran Quang Khai','Ha Noi','Vietnam','{}'::jsonb,NOW() - INTERVAL '60 days'),
  ('c0000002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','Vinhomes','Real Estate','enterprise','https://vinhomes.vn','+842439888888','sales@vinhomes.vn','Symphony Office, Long Bien','Ha Noi','Vietnam','{}'::jsonb,NOW() - INTERVAL '45 days'),
  ('c0000002-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000002','Sun Group','Real Estate','enterprise','https://sungroup.com.vn','+842439333333','info@sungroup.com.vn','29 Lieu Giai, Ba Dinh','Ha Noi','Vietnam','{}'::jsonb,NOW() - INTERVAL '40 days'),
  ('c0000002-0000-0000-0000-000000000003','aaaaaaaa-0000-0000-0000-000000000002','Masterise','Real Estate','large','https://masterisehomes.com','+842839223366','contact@masterise.com','20 Pham Hung, Nam Tu Liem','Ha Noi','Vietnam','{}'::jsonb,NOW() - INTERVAL '30 days'),
  ('c0000003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','MIX Corp','Multi','medium','https://mixcorp.demo','+84906112233','hello@mixcorp.demo','1 Me Linh Square','Ho Chi Minh','Vietnam','{"demo":"yes"}'::jsonb,NOW() - INTERVAL '5 days'),
  ('c0000003-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003','GAMMA Studio','Creative','small','https://gamma.demo','+84906112244','hi@gamma.demo','2 Pasteur','Ho Chi Minh','Vietnam','{}'::jsonb,NOW() - INTERVAL '4 days')
ON CONFLICT (id) DO NOTHING;
