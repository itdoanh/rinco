-- ============================================================
-- LEADS SEED (Loop 202)
-- ~75 leads per tenant (Apex), ~75 (HCT), ~50 (Demo) = 200+ leads
-- Vietnamese full names, +84 phones, varied UTM sources & stages.
-- All data inserted with timestamped created_at spread across 30 days.
-- ============================================================

-- Helper CTE: Vietnamese first/middle/last names
-- We'll generate 200 leads inline.

-- ===== APEX FINTECH LEADS (75) =====
INSERT INTO leads.leads (id, tenant_id, owner_user_id, full_name, email, phone, source, utm_source, utm_medium, utm_campaign, utm_content, fbclid, gclid, ttclid, campaign_id, score, quality_score, predicted_ltv, conversion_probability, custom_fields, data, consent_given, consent_at, source_meta, stage, last_contacted_at, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000001',
  owner_pool.id,
  person.first_name || ' ' || person.last_name,
  lower(replace(person.first_name,' ','')) || '.' || lower(replace(person.last_name,' ','')) || person.email_domain,
  '+8490' || lpad((8000000 + (random()*999999)::int)::text, 7, '0'),
  src.source,
  src.utm_source,
  src.utm_medium,
  src.utm_campaign,
  src.utm_content,
  CASE WHEN src.utm_source='facebook' THEN 'fb.' || md5(random()::text) END,
  CASE WHEN src.utm_source='google' THEN 'g.' || md5(random()::text) END,
  CASE WHEN src.utm_source='tiktok' THEN 'tt.' || md5(random()::text) END,
  gen_random_uuid(),
  (random()*100)::int,
  CASE WHEN random()<0.3 THEN 'high' WHEN random()<0.6 THEN 'medium' ELSE 'low' END,
  (random()*50 + 5)::numeric(18,2)*1000000,
  (random()*0.8)::numeric(5,4),
  jsonb_build_object(
    'interest_level', (ARRAY['very_high','high','medium','low'])[ceil(random()*4)],
    'budget_range', (ARRAY['5-10tr','10-30tr','30-100tr','100tr+'])[ceil(random()*4)],
    'preferred_location', (ARRAY['Ho Chi Minh','Ha Noi','Da Nang','Can Tho'])[ceil(random()*4)],
    'decision_maker', random()<0.5,
    'loan_purpose', (ARRAY['Mua nha','Mua xe','Kinh doanh','No tien mat','Tra no'])[ceil(random()*5)]
  ),
  jsonb_build_object('city',(ARRAY['HCMC','HN','DN','CT'])[ceil(random()*4)],'age',(22+random()*40)::int),
  random()<0.8,
  NOW() - (random()*30 || ' days')::interval,
  jsonb_build_object('campaign_id', md5(random()::text),'adgroup_id', md5(random()::text)),
  stage_pool.stage,
  NOW() - (random()*30 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval
FROM (
  SELECT unnest(ARRAY[
    'Nguyễn Văn An','Trần Thị Bình','Lê Hoàng Cường','Phạm Thị Dung','Hoàng Văn Em',
    'Vũ Thị Phương','Đặng Văn Giang','Bùi Thị Hoa','Đỗ Văn Ích','Ngô Thị Khánh',
    'Hồ Văn Lâm','Tô Thị Mai','Phùng Văn Nam','Lý Thị Oanh','Trương Văn Phúc',
    'Cao Thị Quỳnh','Đinh Văn Rồng','Dương Thị Sương','Phan Văn Tài','Châu Thị Uyên',
    'Tống Văn Vinh','Hà Thị Xuân','Lại Văn Yên','Trịnh Thị Ánh','Mai Văn Bảo',
    'Đoàn Thị Chi','Tạ Văn Dũng','Võ Thị Én','Lưu Văn Phong','Kiều Thị Giang',
    'Lâm Văn Hào','Từ Thị Huyền','Quách Văn Khôi','Tiêu Thị Lan','Giang Văn Minh',
    'Mạc Thị Nhung','Ninh Văn Oai','Ứng Thị Phượng','Vĩnh Văn Quang','Xung Thị Rô',
    'Yên Văn Sĩ','Zương Thị Tâm','An Văn Ưng','Bình Thị Vân','Cam Văn Xuyến',
    'Dương Văn Yên','Hạnh Thị Anh','Kha Văn Bình','Minh Văn Châu','Nghiêm Thị Diệu',
    'Quan Văn Hiếu','Sơn Thị Kiều','Thạch Văn Long','Uyên Thị Mỹ','Vạn Văn Ngọc',
    'Xuân Thị Oanh','Yến Văn Phú','Anh Thị Quyên','Bích Văn Sinh','Cúc Thị Thanh',
    'Duy Văn Uy','Giao Thị Vân','Hải Văn Xanh','Khuê Thị Yến','Lâm Văn Tân',
    'Mai Thị Tường','Nghĩa Văn Vĩ','Phượng Thị Xuân','Quang Văn Yên','Rồng Thị Ánh',
    'Sơn Văn Bảo','Tuyết Thị Cúc','Uyên Văn Duy','Vàng Thị Én','Xanh Văn Phong',
    'Yên Thị Giang'
  ]) AS full_name,
  ROW_NUMBER() OVER() AS rn
  FROM generate_series(1,1)
) AS raw_data
CROSS JOIN LATERAL (
  SELECT split_part(full_name,' ',1) AS first_name, split_part(full_name,' ',2) AS last_name,
         (ARRAY['@gmail.com','@yahoo.com','@outlook.com','@company.vn'])[ceil(random()*4)] AS email_domain
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001' AND role='member' AND parent_id IS NOT NULL
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['new','new','new','contacted','contacted','qualified','qualified','proposal','won','lost'])[ceil(random()*10)] AS stage
) AS stage_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Facebook Ads','Google Ads','TikTok Ads','Zalo OA','Website','Referral'])[ceil(random()*6)] AS source,
         (ARRAY['facebook','google','tiktok','zalo','google','direct'])[ceil(random()*6)] AS utm_source,
         (ARRAY['cpc','social','cpc','social','organic','referral'])[ceil(random()*6)] AS utm_medium,
         (ARRAY['spring_loan_v1','q3_personal_loan','cashback_50','instant_v2','tiktok_creative_a'])[ceil(random()*5)] AS utm_campaign,
         md5(random()::text) AS utm_content
) AS src
ON CONFLICT (id) DO NOTHING;

-- ===== HCT CONSULTING LEADS (75) =====
INSERT INTO leads.leads (id, tenant_id, owner_user_id, full_name, email, phone, source, utm_source, utm_medium, utm_campaign, utm_content, fbclid, gclid, score, quality_score, predicted_ltv, conversion_probability, custom_fields, data, consent_given, consent_at, source_meta, stage, last_contacted_at, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000002',
  owner_pool.id,
  person.first_name || ' ' || person.last_name,
  lower(replace(person.first_name,' ','')) || '.' || lower(replace(person.last_name,' ','')) || '@hcmail.vn',
  '+8491' || lpad((9000000 + (random()*999999)::int)::text, 7, '0'),
  src.source,
  src.utm_source,
  src.utm_medium,
  src.utm_campaign,
  src.utm_content,
  CASE WHEN src.utm_source='facebook' THEN 'fb.' || md5(random()::text) END,
  CASE WHEN src.utm_source='google' THEN 'g.' || md5(random()::text) END,
  (random()*100)::int,
  CASE WHEN random()<0.3 THEN 'high' WHEN random()<0.6 THEN 'medium' ELSE 'low' END,
  (random()*500 + 50)::numeric(18,2)*1000000,
  (random()*0.7)::numeric(5,4),
  jsonb_build_object(
    'property_type', (ARRAY['can_ho','shophouse','dat_nen','biet_thu','lien_ke'])[ceil(random()*5)],
    'budget_range', (ARRAY['2-5 ty','5-10 ty','10-20 ty','20-50 ty','50 ty+'])[ceil(random()*5)],
    'preferred_location', (ARRAY['Ba Dinh','Hoan Kiem','Tay Ho','Long Bien','Nam Tu Liem','Hai Ba Trung'])[ceil(random()*6)],
    'decision_maker', random()<0.6,
    'investment_horizon', (ARRAY['short_term','medium_term','long_term'])[ceil(random()*3)],
    'loan_needed', random()<0.5
  ),
  jsonb_build_object('city','Ha Noi','age',(28+random()*40)::int),
  random()<0.85,
  NOW() - (random()*30 || ' days')::interval,
  jsonb_build_object('campaign_id', md5(random()::text)),
  stage_pool.stage,
  NOW() - (random()*30 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval
FROM (
  SELECT unnest(ARRAY[
    'Hoàng Mai Anh','Nguyễn Bảo Châu','Trần Đức Duy','Lê Thị Hà','Phạm Văn Khoa',
    'Vũ Mỹ Linh','Đặng Quốc Minh','Bùi Thị Nga','Đỗ Văn Phú','Ngô Cẩm Quỳnh',
    'Hồ Thanh Sơn','Tô Khánh Tâm','Phùng Thị Uyên','Lý Văn Việt','Trương Bích Xuân',
    'Cao Văn Ý','Đinh Hà An','Dương Văn Bình','Phan Thị Chi','Châu Hồng Diệu',
    'Tống Văn Giang','Hà Hải Yến','Lại Văn Khang','Trịnh Văn Long','Mai Thị Minh',
    'Đoàn Văn Nghị','Tạ Thị Oanh','Võ Văn Phong','Lưu Thị Quyên','Kiều Văn Rô',
    'Lâm Mỹ Sen','Từ Văn Thái','Quách Thị Uy','Tiêu Văn Vinh','Giang Khánh Vy',
    'Mạc Văn Xuân','Ninh Hải Yến','Ứng Văn An','Vĩnh Thị Bình','Xung Văn Cúc',
    'Yên Thanh Duy','Zương Ngọc Em','An Văn Phong','Bình Mỹ Giang','Cam Văn Hào',
    'Dương Khánh Huyền','Hạnh Văn Khôi','Kha Thị Lan','Minh Văn Mỹ','Nghiêm Văn Nhung',
    'Quan Hà Oai','Sơn Văn Phượng','Thạch Văn Quang','Uyên Thị Rô','Vạn Văn Sĩ',
    'Xuân Tâm Tân','Yến Phương Thị','Anh Văn Uyên','Bích Văn Vân','Cúc Hà Xanh',
    'Duy Mỹ Yến','Giao Văn Tường','Hải Văn Vĩ','Khuê Thị Xuân','Lâm Hà Yến',
    'Mai Văn Ánh','Nghĩa Thị Bảo','Phượng Văn Cúc','Quang Văn Duy','Rồng Ngọc Én',
    'Sơn Văn Phong','Tuyết Mỹ Giang','Uyên Văn Hào','Vàng Xuân Khôi','Xanh Văn Lan'
  ]) AS full_name,
  ROW_NUMBER() OVER() AS rn
  FROM generate_series(1,1)
) AS raw_data
CROSS JOIN LATERAL (
  SELECT split_part(full_name,' ',1) AS first_name, split_part(full_name,' ',2) AS last_name
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000002' AND role='member' AND parent_id IS NOT NULL
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['new','new','new','contacted','contacted','qualified','qualified','proposal','won','lost'])[ceil(random()*10)] AS stage
) AS stage_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Facebook Ads','Zalo OA','LinkedIn','TikTok Ads','Google Ads'])[ceil(random()*5)] AS source,
         (ARRAY['facebook','zalo','linkedin','tiktok','google'])[ceil(random()*5)] AS utm_source,
         (ARRAY['cpc','social','social','cpc','cpc'])[ceil(random()*5)] AS utm_medium,
         (ARRAY['vinhomes_grand_park','masterise_eco','sun_group_30ha','nhan_dinh_q3'])[ceil(random()*4)] AS utm_campaign,
         md5(random()::text) AS utm_content
) AS src
ON CONFLICT (id) DO NOTHING;

-- ===== DEMO LEADS (50) =====
INSERT INTO leads.leads (id, tenant_id, owner_user_id, full_name, email, phone, source, utm_source, utm_medium, utm_campaign, score, custom_fields, stage, consent_given, created_at)
SELECT
  gen_random_uuid(),
  'bbbbbbbb-0000-0000-0000-000000000003',
  owner_pool.id,
  person.full_name,
  lower(replace(person.full_name,' ','')) || '@demo.io',
  '+8492' || lpad((1000000 + (random()*999999)::int)::text, 7, '0'),
  src.source,
  src.utm_source,
  src.utm_medium,
  src.utm_campaign,
  (random()*100)::int,
  jsonb_build_object('interest_level', (ARRAY['high','medium','low'])[ceil(random()*3)]),
  (ARRAY['new','contacted','qualified','won','lost'])[ceil(random()*5)],
  true,
  NOW() - (random()*30 || ' days')::interval
FROM (
  SELECT unnest(ARRAY[
    'Alpha Beta','Gamma Delta','Epsilon Zeta','Eta Theta','Iota Kappa',
    'Lambda Mu','Nu Xi','Omicron Pi','Rho Sigma','Tau Upsilon',
    'Phi Chi','Psi Omega','Nova Star','Luna Sky','Solar Ray',
    'Quantum Flux','Pixel Wave','Data Stream','Code Ninja','Cloud Surfer',
    'Byte Master','Cache Lord','Hash Whisper','Binary Bloom','Algorithm Ace',
    'Stack Trace','Heap Hero','Null Pointer','Void Walker','Memory Lane',
    'Thread Safe','Async Await','Promise Keeper','Function Pure','Arrow Head',
    'Lambda Lift','Spark Plug','Cluster Node','Docker Ship','Kube Pilot',
    'Helm Chart','Terra Form','Vault Secret','Mesh Wire','Edge Cut'
  ]) AS full_name,
  ROW_NUMBER() OVER() AS rn
  FROM generate_series(1,1)
) AS raw_data
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='bbbbbbbb-0000-0000-0000-000000000003' AND role='member' AND parent_id IS NOT NULL
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Facebook Ads','Google Ads','TikTok Ads','Referral','Demo'])[ceil(random()*5)] AS source,
         (ARRAY['facebook','google','tiktok','direct','demo'])[ceil(random()*5)] AS utm_source,
         (ARRAY['cpc','cpc','cpc','referral','none'])[ceil(random()*5)] AS utm_medium,
         (ARRAY['demo_spring','demo_free_trial','demo_youtube'])[ceil(random()*3)] AS utm_campaign
) AS src
ON CONFLICT (id) DO NOTHING;

-- Update first 30 leads to have crm_synced + capi_sent for variety
UPDATE leads.leads SET crm_synced = true, capi_sent = true
WHERE tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002')
  AND stage IN ('qualified','proposal','won','lost')
  AND crm_synced = false
LIMIT 60;
