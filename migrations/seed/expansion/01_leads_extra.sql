-- ============================================================
-- WS-B LOOP 1: LEADS EXPANSION (~200+ leads per demo tenant)
-- Adds 200+ additional leads per tenant on top of base 04_leads.sql
-- All idempotent (ON CONFLICT DO NOTHING) and uses gen_random_uuid
-- ============================================================

-- ============================================================
-- APEX FINTECH: 250 EXTRA LEADS (real Vietnamese names)
-- ============================================================
INSERT INTO leads.leads (
  id, tenant_id, owner_user_id, full_name, email, phone,
  source, utm_source, utm_medium, utm_campaign, utm_content,
  fbclid, gclid, ttclid, campaign_id,
  score, quality_score, predicted_ltv, conversion_probability,
  custom_fields, data, consent_given, consent_at, source_meta,
  stage, last_contacted_at, created_at, updated_at
)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000001',
  owner_pool.id,
  person.full_name,
  lower(replace(person.full_name,' ','')) || person.email_domain,
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
    'preferred_location', (ARRAY['Ho Chi Minh','Ha Noi','Da Nang','Can Tho','Hai Phong','Bien Hoa'])[ceil(random()*6)],
    'decision_maker', random()<0.5,
    'loan_purpose', (ARRAY['Mua nha','Mua xe','Kinh doanh','No tien mat','Tra no','Dau tu','Hoc tap'])[ceil(random()*7)],
    'age', (22 + (random()*40)::int),
    'occupation', (ARRAY['Nhan vien van phong','Kinh doanh tu do','Ky su','Bac si','Giao vien','Cong nhan','Quan ly','Ke toan'])[ceil(random()*8)]
  ),
  jsonb_build_object(
    'city',(ARRAY['HCMC','HN','DN','CT','HP','BD'])[ceil(random()*6)],
    'district',(ARRAY['Quan 1','Quan 2','Quan 3','Quan 4','Quan 7','Quan Binh Thanh','Quan Go Vap','Quan Tan Binh','Quan 10','Quan Phu Nhuan'])[ceil(random()*10)],
    'source_detail',(ARRAY['facebook_ads_spring','facebook_ads_personal','google_brand','google_generic','tiktok_creative_v1','tiktok_creative_v2','zalo_oa_v3','referral_program'])[ceil(random()*8)]
  ),
  random()<0.8,
  NOW() - (random()*60 || ' days')::interval,
  jsonb_build_object('campaign_id', md5(random()::text),'adgroup_id', md5(random()::text),'ad_id', md5(random()::text)),
  stage_pool.stage,
  NOW() - (random()*30 || ' days')::interval,
  NOW() - (random()*60 || ' days')::interval,
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
    'Yên Thị Giang','Tịnh Văn An','Kha Thị Bình','Hảo Văn Cúc','Khải Văn Duyên',
    'Đức Văn Hà','Minh Văn Khoa','Phước Văn Lan','Trọng Văn Mỹ','Quý Văn Nga',
    'Thiện Văn Oanh','Huy Văn Phú','Hòa Văn Quyên','Sáng Văn Rô','Thương Văn Sương',
    'Tường Văn Tâm','Uy Văn Uyên','Vinh Văn Vân','Xuân Văn Xanh','Yến Văn Yên',
    'An Văn Ánh','Bảo Văn Bình','Cường Văn Cúc','Dũng Văn Duy','Em Văn Én',
    'Phong Văn Giang','Hào Văn Hà','Khôi Văn Khang','Lan Văn Long','Mỹ Văn Mai',
    'Nhung Văn Nam','Oai Văn Oanh','Phượng Văn Phúc','Quang Văn Quỳnh','Rô Văn Rồng',
    'Sĩ Văn Sương','Tâm Văn Tài','Ưng Văn Uyên','Vân Văn Vinh','Xuyến Văn Xuân',
    'Yên Văn Yến','Ánh Văn An','Bảo Thị Bình','Cúc Văn Cường','Duy Thị Dung',
    'Én Văn Em','Giang Thị Phương','Hà Văn Giang','Khang Thị Hoa','Long Văn Ích',
    'Mai Thị Khánh','Nam Văn Lâm','Oanh Thị Mai','Phúc Văn Nam','Quỳnh Thị Oanh',
    'Rồng Văn Phúc','Sương Thị Quỳnh','Tài Văn Rồng','Uyên Thị Sương','Vinh Văn Tài',
    'Xuân Thị Uyên','Yến Văn Vinh','An Thị Xuân','Bình Văn Yên','Cường Thị Ánh',
    'Dung Văn Bảo','Em Thị Chi','Giang Văn Dũng','Hoa Văn Én','Ích Thị Phong',
    'Khánh Văn Giang','Lâm Thị Hào','Mai Văn Huyền','Nam Thị Khôi','Oanh Văn Lan',
    'Phúc Thị Minh','Quỳnh Văn Mỹ','Rồng Thị Nhung','Sương Văn Oai','Tài Thị Phượng',
    'Uyên Văn Quang','Vinh Thị Rô','Xuân Văn Sĩ','Yến Thị Tâm','An Văn Ưng',
    'Bình Thị Vân','Cường Văn Xuyến','Dung Thị Yên','Em Văn Ánh','Giang Thị Bảo',
    'Hoa Văn Cúc','Ích Thị Duy','Khánh Văn Én','Lâm Thị Phong','Mai Văn Giang',
    'Nam Thị Hào','Oanh Văn Huyền','Phúc Thị Khôi','Quỳnh Văn Lan','Rồng Thị Minh',
    'Sương Văn Mỹ','Tài Thị Nhung','Uyên Văn Oai','Vinh Thị Phượng','Xuân Văn Quang'
  ]) AS full_name,
  ROW_NUMBER() OVER() AS rn
  FROM generate_series(1,1)
) AS raw_data
CROSS JOIN LATERAL (
  SELECT split_part(full_name,' ',1) AS first_name, split_part(full_name,' ',2) AS last_name,
         (ARRAY['@gmail.com','@yahoo.com','@outlook.com','@company.vn','@vnn.vn'])[ceil(random()*5)] AS email_domain
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001' AND role='member' AND parent_id IS NOT NULL
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['new','new','new','contacted','contacted','contacted','qualified','qualified','proposal','won','lost'])[ceil(random()*11)] AS stage
) AS stage_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Facebook Ads','Google Ads','TikTok Ads','Zalo OA','Website','Referral'])[ceil(random()*6)] AS source,
         (ARRAY['facebook','google','tiktok','zalo','google','direct'])[ceil(random()*6)] AS utm_source,
         (ARRAY['cpc','social','cpc','social','organic','referral'])[ceil(random()*6)] AS utm_medium,
         (ARRAY['spring_loan_v1','q3_personal_loan','cashback_50','instant_v2','tiktok_creative_a','summer_loan','autumn_v3','winter_promo'])[ceil(random()*8)] AS utm_campaign,
         md5(random()::text) AS utm_content
) AS src
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- HCT CONSULTING: 200 EXTRA LEADS (real estate investors)
-- ============================================================
INSERT INTO leads.leads (
  id, tenant_id, owner_user_id, full_name, email, phone,
  source, utm_source, utm_medium, utm_campaign, utm_content,
  fbclid, gclid, score, quality_score, predicted_ltv, conversion_probability,
  custom_fields, data, consent_given, consent_at, source_meta,
  stage, last_contacted_at, created_at, updated_at
)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000002',
  owner_pool.id,
  person.full_name,
  lower(replace(person.full_name,' ','')) || '@hcmail.vn',
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
    'property_type', (ARRAY['can_ho','shophouse','dat_nen','biet_thu','lien_ke','penthouse','officetel'])[ceil(random()*7)],
    'budget_range', (ARRAY['2-5 ty','5-10 ty','10-20 ty','20-50 ty','50 ty+','100 ty+'])[ceil(random()*6)],
    'preferred_location', (ARRAY['Ba Dinh','Hoan Kiem','Tay Ho','Long Bien','Nam Tu Liem','Hai Ba Trung','Dong Da','Cau Giay','Thanh Xuan'])[ceil(random()*9)],
    'decision_maker', random()<0.6,
    'investment_horizon', (ARRAY['short_term','medium_term','long_term'])[ceil(random()*3)],
    'loan_needed', random()<0.5,
    'age', (28 + (random()*40)::int),
    'investor_type', (ARRAY['first_time','experienced','professional','foreign'])[ceil(random()*4)]
  ),
  jsonb_build_object(
    'city','Ha Noi',
    'district',(ARRAY['Ba Dinh','Hoan Kiem','Tay Ho','Long Bien','Nam Tu Liem','Hai Ba Trung','Dong Da','Cau Giay','Thanh Xuan','Hoang Mai'])[ceil(random()*10)],
    'source_detail',(ARRAY['vinhomes_landing','masterise_landing','sun_group_landing','referral_investor','newsletter','youtube_ad'])[ceil(random()*6)]
  ),
  random()<0.85,
  NOW() - (random()*60 || ' days')::interval,
  jsonb_build_object('campaign_id', md5(random()::text),'utm_content', md5(random()::text)),
  stage_pool.stage,
  NOW() - (random()*30 || ' days')::interval,
  NOW() - (random()*60 || ' days')::interval,
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
    'Sơn Văn Phong','Tuyết Mỹ Giang','Uyên Văn Hào','Vàng Xuân Khôi','Xanh Văn Lan',
    'An Hà Bình','Bảo Mỹ Cúc','Cường Hà Duy','Dũng Khánh Én','Em Phong Giang',
    'Phong Mỹ Hà','Hào Văn Khang','Khôi Thị Long','Lan Mỹ Mai','Mỹ Văn Nam',
    'Nhung Thị Oanh','Oai Văn Phúc','Phượng Văn Quỳnh','Quang Văn Rồng','Rô Thị Sương',
    'Sĩ Văn Tài','Tâm Văn Uyên','Ưng Thị Vinh','Vân Văn Xuân','Xuyến Hà Yến',
    'Yên Phương Ánh','Ánh Văn Bảo','Bảo Thị Cúc','Cúc Văn Duy','Duy Mỹ Én',
    'Én Văn Giang','Giang Hà Hào','Hà Văn Huyền','Khang Thị Khôi','Long Văn Lan',
    'Mỹ Văn Mỹ','Nam Thị Nhung','Oanh Văn Oai','Phúc Hà Phượng','Quỳnh Thị Quang',
    'Sương Văn Rô','Tài Hà Sĩ','Uyên Văn Tâm','Vinh Thị Ưng','Xuân Văn Vân',
    'Yến Hà Xuyến','An Phương Yên','Bình Mỹ Ánh','Cúc Văn Bảo','Duy Hà Cúc',
    'Én Mỹ Duy','Giang Văn Én','Hà Văn Giang','Hào Thị Hào','Khôi Văn Khôi',
    'Lan Mỹ Lan','Mai Văn Mai','Nam Thị Nam','Oanh Văn Oanh','Phúc Mỹ Phúc',
    'Quỳnh Văn Quỳnh','Rồng Thị Rồng','Sương Văn Sương','Tài Thị Tài','Uyên Văn Uyên'
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
  SELECT (ARRAY['new','new','new','contacted','contacted','qualified','qualified','viewing','negotiation','won','lost'])[ceil(random()*11)] AS stage
) AS stage_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Facebook Ads','Zalo OA','LinkedIn','TikTok Ads','Google Ads'])[ceil(random()*5)] AS source,
         (ARRAY['facebook','zalo','linkedin','tiktok','google'])[ceil(random()*5)] AS utm_source,
         (ARRAY['cpc','social','social','cpc','cpc'])[ceil(random()*5)] AS utm_medium,
         (ARRAY['vinhomes_grand_park','masterise_eco','sun_group_30ha','nhan_dinh_q3','vinhomes_ocean_park','vinhomes_symphony','tay_ho_view','long_bien_residence'])[ceil(random()*8)] AS utm_campaign,
         md5(random()::text) AS utm_content
) AS src
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DEMO COMPANY: 100 EXTRA LEADS
-- ============================================================
INSERT INTO leads.leads (
  id, tenant_id, owner_user_id, full_name, email, phone,
  source, utm_source, utm_medium, utm_campaign, score, custom_fields,
  data, stage, consent_given, created_at
)
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
  jsonb_build_object(
    'interest_level', (ARRAY['high','medium','low'])[ceil(random()*3)],
    'age', (20 + (random()*40)::int),
    'company_size', (ARRAY['solo','small','medium','large'])[ceil(random()*4)]
  ),
  jsonb_build_object('city',(ARRAY['Ho Chi Minh','Ha Noi','Da Nang'])[ceil(random()*3)],'source_detail',(ARRAY['demo_spring','demo_free_trial','demo_youtube','referral_demo','docs_demo'])[ceil(random()*5)]),
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
    'Helm Chart','Terra Form','Vault Secret','Mesh Wire','Edge Cut',
    'Phoenix Blaze','Titan Forge','Orion Drive','Sirius Loop','Vega Pulse',
    'Andromeda Code','Cygnus Stack','Lyra Logic','Draco Run','Hydra Hook',
    'Centaur Node','Pegasus Build','Pisces Script','Aquarius Data','Aries Bot',
    'Taurus Cache','Gemini Twin','Leo King','Libra Scale','Scorpio Sting',
    'Sagittarius Aim','Capricorn Goal','Virgo Detail','Cancer Shield','Leo Prime'
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
         (ARRAY['demo_spring','demo_free_trial','demo_youtube','demo_q3','demo_partner'])[ceil(random()*5)] AS utm_campaign
) AS src
ON CONFLICT (id) DO NOTHING;

-- Update capi_sent flag for some 'won' leads
UPDATE leads.leads SET crm_synced = true, capi_sent = true
WHERE tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002')
  AND stage IN ('qualified','proposal','won','lost','viewing','negotiation')
  AND crm_synced = false;

-- Stats
SELECT 'expansion_01_leads_extra (WS-B Loop 1)' AS section,
       tenant_id, COUNT(*) AS total_leads
FROM leads.leads
WHERE tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
GROUP BY tenant_id
ORDER BY tenant_id;
