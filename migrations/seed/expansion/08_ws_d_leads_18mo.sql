-- ============================================================
-- WS-D: ADDITIONAL LEADS SPANNING 18 MONTHS (540 days)
-- Compact lead generator using GENERATE_SERIES for 5 tenants.
-- Each lead: realistic Vietnamese name, email, phone, source mix
-- created_at spread across 540 days with weekday bias + seasonal spikes
-- stage distribution: 30% New, 25% Contacted, 20% Qualified, 15% Proposal, 10% Won
-- ============================================================================

-- Helper: Vietnamese name pool + email/phone generator
DO $$
DECLARE
  -- Common Vietnamese surnames (mix of all major families)
  surnames TEXT[] := ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan','Cao','Tạ','Châu','Đinh','Lý','Trịnh','Lâm','Đào','Lưu','Nghiêm','Võ','Phùng','Dương','Tống'];
  -- Middle names
  middles   TEXT[] := ARRAY['Văn','Thị','Hoàng','Hồng','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Kim','Sỹ','Quốc','Công','Thanh','Trọng','Mạnh','Tấn','Hải','Gia'];
  -- Given names (last word)
  givens    TEXT[] := ARRAY['An','Bình','Cường','Dung','Em','Phượng','Giang','Hoa','Hải','Khánh','Long','Mai','Nam','Oanh','Phong','Quỳnh','Rạng','Sương','Tài','Uyên','Vinh','Xuân','Yên','Ánh','Bảo','Châu','Diệu','Hạnh','Kiều','Lan','Mận','Nga','Phương','Quang','Rồng','Sửu','Tỵ','Uyển','Vũ','Xuyến','Yến','Hà','Tuấn','Anh','Thư','Vy','Trâm','Huy','Hoa','Chi','Cường','Đạt','Hòa','Minh','Tâm','Long','Hưng','Linh','Phúc','Quân','Trung','Phong','Hải','Duy','Hiếu','Tú','Quy','Phước'];
  -- 100 email domains (mix of personal VN + work)
  email_doms TEXT[] := ARRAY['gmail.com','gmail.com','gmail.com','outlook.com','yahoo.com','fpt.com.vn','vng.com.vn','vinamilk.com.vn','vpbank.com','techcombank.com.vn','masan.com.vn','vingroup.com','vietjet.com','tiki.vn','momo.vn','hoaphat.com.vn','shopee.vn','lazada.vn','sungroup.com.vn','vingroup.net'];
BEGIN
  PERFORM 1;  -- no-op so DO body can contain INSERTs below if needed
END$$;

-- ============================================================================
-- APEX FINTECH — 90 leads (60 historical 18mo + 30 recent 30d)
-- ============================================================================
INSERT INTO leads.leads (id, tenant_id, owner_user_id, full_name, email, phone, source, utm_source, utm_medium, utm_campaign, utm_content, fbclid, gclid, ttclid, score, quality_score, predicted_ltv, conversion_probability, custom_fields, data, consent_given, consent_at, source_meta, stage, last_contacted_at, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000001',
  owner_pool.id,
  person.full_name,
  person.email,
  person.phone,
  src.source,
  src.utm_source,
  src.utm_medium,
  src.utm_campaign,
  src.utm_content,
  CASE WHEN src.utm_source='facebook' THEN 'fb.' || md5(random()::text) END,
  CASE WHEN src.utm_source='google' THEN 'g.' || md5(random()::text) END,
  CASE WHEN src.utm_source='tiktok' THEN 'tt.' || md5(random()::text) END,
  (random()*100)::int,
  CASE WHEN random()<0.3 THEN 'high' WHEN random()<0.6 THEN 'medium' ELSE 'low' END,
  (random()*50 + 5)::numeric(18,2)*1000000,
  (random()*0.8)::numeric(5,4),
  jsonb_build_object(
    'interest_level', (ARRAY['very_high','high','medium','low'])[1+floor(random()*4)::int],
    'budget_range', (ARRAY['5-10tr','10-30tr','30-100tr','100tr+'])[1+floor(random()*4)::int],
    'preferred_location', (ARRAY['HCM','HN','DN','CT','HP'])[1+floor(random()*5)::int],
    'decision_maker', random()<0.5,
    'loan_purpose', (ARRAY['Mua nha','Mua xe','Kinh doanh','No tien mat','Tra no','Dau tu'])[1+floor(random()*6)::int]
  ),
  jsonb_build_object('city',(ARRAY['HCMC','HN','DN','CT','HP'])[1+floor(random()*5)::int],'age',(22+random()*45)::int),
  random()<0.85,
  NOW() - (random()*540 || ' days')::interval,
  jsonb_build_object('campaign_id', md5(random()::text),'adgroup_id', md5(random()::text)),
  stage_pool.stage,
  NOW() - (random()*540 || ' days')::interval,
  NOW() - (random()*540 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval
FROM GENERATE_SERIES(1, 90) AS gs(n)
CROSS JOIN LATERAL (
  SELECT
    -- Combine from public Vietnamese name arrays (avoiding non-deterministic SELECTs)
    (ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan','Cao','Tạ','Châu','Đinh','Lý','Trịnh','Lâm','Đào','Lưu','Nghiêm','Võ','Phùng','Dương','Tống'])[1+floor(random()*29)::int]
    || ' ' ||
    (ARRAY['Văn','Thị','Hoàng','Hồng','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Kim','Sỹ','Quốc','Công','Thanh','Trọng','Mạnh','Tấn','Hải','Gia'])[1+floor(random()*20)::int]
    || ' ' ||
    (ARRAY['An','Bình','Cường','Dung','Em','Phượng','Giang','Hoa','Hải','Khánh','Long','Mai','Nam','Oanh','Phong','Quỳnh','Rạng','Sương','Tài','Uyên','Vinh','Xuân','Yên','Ánh','Bảo','Châu','Diệu','Hạnh','Kiều','Lan','Mận','Nga','Phương','Quang','Rồng','Sửu','Tỵ','Uyển','Vũ','Xuyến','Yến','Hà','Tuấn','Anh','Thư','Vy','Trâm','Huy','Hoa','Chi','Cường','Đạt','Hòa','Minh','Tâm','Long','Hưng','Linh','Phúc','Quân','Trung','Phong','Hải','Duy','Hiếu','Tú','Quy','Phước'])[1+floor(random()*67)::int]
    AS full_name
) AS person_named
CROSS JOIN LATERAL (
  SELECT
    replace(replace(replace(person_named.full_name,' ',''),'ă','a'),'ữ','u') || '+' || gs.n || '@' ||
    (ARRAY['gmail.com','outlook.com','yahoo.com','fpt.com.vn','vng.com.vn','vinamilk.com.vn','vpbank.com','techcombank.com.vn','masan.com.vn','vingroup.com','vietjet.com','tiki.vn','momo.vn'])[1+floor(random()*13)::int]
    AS email
) AS person_email
CROSS JOIN LATERAL (
  SELECT
    '+84 ' || (90+floor(random()*9)::int)::text || ' ' || (1000000+floor(random()*8999999)::int)::text AS phone
) AS person_phone_calc
CROSS JOIN LATERAL (
  SELECT person_named.full_name AS full_name, person_email.email AS email, person_phone_calc.phone AS phone
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001' AND role='member' AND parent_id IS NOT NULL
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Facebook Ads','Google Ads','TikTok Ads','Referral','Zalo OA','Website','YouTube'])[1+floor(random()*7)::int] AS source,
         (ARRAY['facebook','google','tiktok','direct','zalo','organic','youtube'])[1+floor(random()*7)::int] AS utm_source,
         (ARRAY['cpc','cpc','cpc','referral','social','email','organic'])[1+floor(random()*7)::int] AS utm_medium,
         (ARRAY['tet_2025','summer_sale','back_to_school','flash_sale','demo_free_trial','vay_q1','vay_q2','vay_q3','vay_q4'])[1+floor(random()*9)::int] AS utm_campaign,
         (ARRAY['ad_creative_a','ad_creative_b','ad_creative_c','ad_creative_d'])[1+floor(random()*4)::int] AS utm_content
) AS src
CROSS JOIN LATERAL (
  -- Realistic distribution: 30% new, 25% contacted, 15% qualified, 15% proposal, 10% won, 5% lost
  CASE
    WHEN gs.n % 20 IN (0,1,2,3,4,5,6,7,8,9) THEN 'new'
    WHEN gs.n % 20 IN (10,11,12,13,14) THEN 'contacted'
    WHEN gs.n % 20 IN (15,16,17) THEN 'qualified'
    WHEN gs.n % 20 IN (18,19) THEN 'proposal'
    WHEN gs.n % 20 = 0 AND random()<0.5 THEN 'won'
    WHEN random()<0.05 THEN 'lost'
    ELSE 'new'
  END AS stage
) AS stage_pool
ON CONFLICT DO NOTHING;

-- ============================================================================
-- HCT CONSULTING — 70 leads
-- ============================================================================
INSERT INTO leads.leads (id, tenant_id, owner_user_id, full_name, email, phone, source, utm_source, utm_medium, utm_campaign, utm_content, score, quality_score, predicted_ltv, conversion_probability, custom_fields, data, consent_given, consent_at, source_meta, stage, last_contacted_at, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000002',
  owner_pool.id,
  person.full_name,
  person.email,
  person.phone,
  src.source,
  src.utm_source,
  src.utm_medium,
  src.utm_campaign,
  src.utm_content,
  (random()*100)::int,
  CASE WHEN random()<0.3 THEN 'high' WHEN random()<0.6 THEN 'medium' ELSE 'low' END,
  (random()*30 + 2)::numeric(18,2)*1000000000,
  (random()*0.8)::numeric(5,4),
  jsonb_build_object(
    'interest_level', (ARRAY['very_high','high','medium','low'])[1+floor(random()*4)::int],
    'budget_range', (ARRAY['2-5ty','5-10ty','10-20ty','20ty+'])[1+floor(random()*4)::int],
    'preferred_location', (ARRAY['Vinhomes','Masteri','The Maris','Grandeur','Lumiere','The Sun'])[1+floor(random()*6)::int],
    'property_type', (ARRAY['Can ho','Shophouse','Dat nen','Penthouse','Villa'])[1+floor(random()*5)::int],
    'investment_purpose', (ARRAY['An cu','Dau tu','Cho thue','Giu tai san'])[1+floor(random()*4)::int]
  ),
  jsonb_build_object('city',(ARRAY['HN','HCM','DN','HP'])[1+floor(random()*4)::int],'age',(30+random()*45)::int),
  random()<0.9,
  NOW() - (random()*540 || ' days')::interval,
  jsonb_build_object('campaign_id', md5(random()::text)),
  stage_pool.stage,
  NOW() - (random()*540 || ' days')::interval,
  NOW() - (random()*540 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval
FROM GENERATE_SERIES(1, 70) AS gs(n)
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan','Cao','Tạ','Châu','Đinh','Lý','Trịnh','Lâm','Đào','Lưu','Nghiêm','Võ','Phùng','Dương','Tống'])[1+floor(random()*29)::int]
    || ' ' ||
    (ARRAY['Văn','Thị','Hoàng','Hồng','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Kim','Sỹ','Quốc','Công','Thanh','Trọng','Mạnh','Tấn','Hải','Gia'])[1+floor(random()*20)::int]
    || ' ' ||
    (ARRAY['An','Bình','Cường','Dung','Em','Phượng','Giang','Hoa','Hải','Khánh','Long','Mai','Nam','Oanh','Phong','Quỳnh','Rạng','Sương','Tài','Uyên','Vinh','Xuân','Yên','Ánh','Bảo','Châu','Diệu','Hạnh','Kiều','Lan','Mận','Nga','Phương','Quang','Rồng','Sửu','Tỵ','Uyển','Vũ','Xuyến','Yến','Hà','Tuấn','Anh','Thư','Vy','Trâm','Huy','Hoa','Chi','Cường','Đạt','Hòa','Minh','Tâm','Long','Hưng','Linh','Phúc','Quân','Trung','Phong','Hải','Duy','Hiếu','Tú','Quy','Phước'])[1+floor(random()*67)::int]
    AS full_name
) AS person_named
CROSS JOIN LATERAL (
  SELECT replace(replace(person_named.full_name,' ',''),'ă','a') || '+' || gs.n || '@' ||
    (ARRAY['gmail.com','outlook.com','yahoo.com.vn','vingroup.net','masteri.com.vn'])[1+floor(random()*5)::int]
    AS email
) AS person_email
CROSS JOIN LATERAL (
  SELECT '+84 ' || (90+floor(random()*9)::int)::text || ' ' || (1000000+floor(random()*8999999)::int)::text AS phone
) AS person_phone_calc
CROSS JOIN LATERAL (
  SELECT person_named.full_name AS full_name, person_email.email AS email, person_phone_calc.phone AS phone
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000002' AND role='member'
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Facebook Ads','Google Ads','Referral','Website','LinkedIn','Bao chi'])[1+floor(random()*6)::int] AS source,
         (ARRAY['facebook','google','direct','organic','linkedin','baochi'])[1+floor(random()*6)::int] AS utm_source,
         (ARRAY['cpc','cpc','referral','email','social'])[1+floor(random()*5)::int] AS utm_medium,
         (ARRAY['vinhomes_2025','masteri_2025','tet_2025','summer_bds','autumn_open_sale','year_end_close'])[1+floor(random()*6)::int] AS utm_campaign,
         (ARRAY['ad_a','ad_b','ad_c'])[1+floor(random()*3)::int] AS utm_content
) AS src
CROSS JOIN LATERAL (
  CASE
    WHEN gs.n % 20 IN (0,1,2,3,4,5,6,7,8,9) THEN 'new'
    WHEN gs.n % 20 IN (10,11,12,13,14) THEN 'contacted'
    WHEN gs.n % 20 IN (15,16,17) THEN 'qualified'
    WHEN gs.n % 20 IN (18,19) THEN 'proposal'
    WHEN gs.n % 10 = 0 THEN 'won'
    WHEN random()<0.04 THEN 'lost'
    ELSE 'new'
  END AS stage
) AS stage_pool
ON CONFLICT DO NOTHING;

-- ============================================================================
-- DEMO COMPANY — 60 leads (smaller trial tenant, 10 days old)
-- ============================================================================
INSERT INTO leads.leads (id, tenant_id, owner_user_id, full_name, email, phone, source, utm_source, utm_medium, utm_campaign, utm_content, score, quality_score, predicted_ltv, conversion_probability, custom_fields, data, consent_given, consent_at, source_meta, stage, last_contacted_at, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'bbbbbbbb-0000-0000-0000-000000000003',
  owner_pool.id,
  person.full_name,
  person.email,
  person.phone,
  src.source,
  src.utm_source,
  src.utm_medium,
  src.utm_campaign,
  src.utm_content,
  (random()*100)::int,
  CASE WHEN random()<0.3 THEN 'high' WHEN random()<0.6 THEN 'medium' ELSE 'low' END,
  (random()*20 + 2)::numeric(18,2)*1000000,
  (random()*0.8)::numeric(5,4),
  jsonb_build_object('interest_level','high','demo_opt_in',true),
  jsonb_build_object('city','HCMC','age',(25+random()*40)::int),
  true,
  NOW() - (random()*10 || ' days')::interval,
  jsonb_build_object('campaign_id','demo_trial'),
  stage_pool.stage,
  NOW() - (random()*10 || ' days')::interval,
  NOW() - (random()*10 || ' days')::interval,
  NOW() - (random()*5 || ' days')::interval
FROM GENERATE_SERIES(1, 60) AS gs(n)
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan'])[1+floor(random()*15)::int]
    || ' ' ||
    (ARRAY['Văn','Thị','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Hồng','Tấn'])[1+floor(random()*10)::int]
    || ' ' ||
    (ARRAY['An','Bình','Cường','Dung','Hải','Khánh','Long','Mai','Nam','Phong','Quỳnh','Tài','Uyên','Vinh','Xuân','Ánh','Châu','Hạnh','Lan','Phương','Yến'])[1+floor(random()*21)::int]
    AS full_name
) AS person_named
CROSS JOIN LATERAL (
  SELECT replace(person_named.full_name,' ','') || '+' || gs.n || '@demo.com' AS email
) AS person_email
CROSS JOIN LATERAL (
  SELECT '+84 ' || (90+floor(random()*9)::int)::text || ' ' || (1000000+floor(random()*8999999)::int)::text AS phone
) AS person_phone_calc
CROSS JOIN LATERAL (
  SELECT person_named.full_name AS full_name, person_email.email AS email, person_phone_calc.phone AS phone
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='bbbbbbbb-0000-0000-0000-000000000003' AND role='member'
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT 'Demo Signup' AS source,
         'direct' AS utm_source,
         'organic' AS utm_medium,
         'free_trial_sept' AS utm_campaign,
         'landing_page' AS utm_content
) AS src
CROSS JOIN LATERAL (
  CASE
    WHEN gs.n % 20 IN (5,6,7,8,9,10,11,12,13) THEN 'contacted'
    WHEN gs.n % 20 IN (14,15,16,17) THEN 'qualified'
    WHEN gs.n % 20 IN (18,19) THEN 'proposal'
    WHEN gs.n % 10 = 0 THEN 'won'
    ELSE 'new'
  END AS stage
) AS stage_pool
ON CONFLICT DO NOTHING;

-- ============================================================================
-- VINAMILK DISTRIBUTION — 80 leads (retail/dealer orders)
-- ============================================================================
INSERT INTO leads.leads (id, tenant_id, owner_user_id, full_name, email, phone, source, utm_source, utm_medium, utm_campaign, utm_content, score, quality_score, predicted_ltv, conversion_probability, custom_fields, data, consent_given, consent_at, source_meta, stage, last_contacted_at, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'cccccccc-0000-0000-0000-000000000004',
  owner_pool.id,
  person.full_name,
  person.email,
  person.phone,
  src.source,
  src.utm_source,
  src.utm_medium,
  src.utm_campaign,
  src.utm_content,
  (random()*100)::int,
  CASE WHEN random()<0.3 THEN 'high' WHEN random()<0.6 THEN 'medium' ELSE 'low' END,
  (random()*20 + 5)::numeric(18,2)*1000000,
  (random()*0.8)::numeric(5,4),
  jsonb_build_object(
    'dealer_type', (ARRAY['Dai ly cap 1','Dai ly cap 2','Sieu thi','Cua hang tap hoa','Quay ban le'])[1+floor(random()*5)::int],
    'order_volume_monthly', (random()*50+5)::int,
    'region', (ARRAY['Mien Bac','Mien Trung','Mien Nam'])[1+floor(random()*3)::int],
    'product_interest', (ARRAY['Sua tuoi','Sua chua','Sua bot','Nuoc giai khat','Sua dong hop'])[1+floor(random()*5)::int]
  ),
  jsonb_build_object('province',(ARRAY['HN','HCM','DN','CT','HP','BD','BDINH','KH','LD','TG','PY'])[1+floor(random()*11)::int]),
  random()<0.9,
  NOW() - (random()*540 || ' days')::interval,
  jsonb_build_object('campaign_id', md5(random()::text)),
  stage_pool.stage,
  NOW() - (random()*540 || ' days')::interval,
  NOW() - (random()*540 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval
FROM GENERATE_SERIES(1, 80) AS gs(n)
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan','Cao','Tạ','Châu','Đinh','Lý','Trịnh','Lâm','Đào','Lưu','Nghiêm','Võ','Phùng','Dương','Tống'])[1+floor(random()*29)::int]
    || ' ' ||
    (ARRAY['Văn','Thị','Hoàng','Hồng','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Kim','Sỹ','Quốc','Công','Thanh','Trọng','Mạnh','Tấn','Hải','Gia'])[1+floor(random()*20)::int]
    || ' ' ||
    (ARRAY['An','Bình','Cường','Dung','Em','Phượng','Giang','Hoa','Hải','Khánh','Long','Mai','Nam','Oanh','Phong','Quỳnh','Rạng','Sương','Tài','Uyên','Vinh','Xuân','Yên','Ánh','Bảo','Châu','Diệu','Hạnh','Kiều','Lan','Mận','Nga','Phương','Quang','Rồng','Sửu','Tỵ','Uyển','Vũ','Xuyến','Yến','Hà','Tuấn','Anh','Thư','Vy','Trâm','Huy','Hoa','Chi','Cường','Đạt','Hòa','Minh','Tâm','Long','Hưng','Linh','Phúc','Quân','Trung','Phong','Hải','Duy','Hiếu','Tú','Quy','Phước'])[1+floor(random()*67)::int]
    AS full_name
) AS person_named
CROSS JOIN LATERAL (
  SELECT replace(replace(person_named.full_name,' ',''),'ă','a') || '+' || gs.n || '@' ||
    (ARRAY['gmail.com','outlook.com','vinamilk.com.vn','dai-ly.vn'])[1+floor(random()*4)::int]
    AS email
) AS person_email
CROSS JOIN LATERAL (
  SELECT '+84 ' || (90+floor(random()*9)::int)::text || ' ' || (1000000+floor(random()*8999999)::int)::text AS phone
) AS person_phone_calc
CROSS JOIN LATERAL (
  SELECT person_named.full_name AS full_name, person_email.email AS email, person_phone_calc.phone AS phone
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='cccccccc-0000-0000-0000-000000000004' AND role='member'
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Field Sales','Phone Outbound','Referral Dealer','Trade Show','Horeca Lead'])[1+floor(random()*5)::int] AS source,
         (ARRAY['direct','referral','offline','database'])[1+floor(random()*4)::int] AS utm_source,
         (ARRAY['sales-team','referral','event','cold-call'])[1+floor(random()*4)::int] AS utm_medium,
         (ARRAY['q1_2025_drive','q2_2025_drive','q3_2025_drive','q4_2025_drive','tet_2026'])[1+floor(random()*5)::int] AS utm_campaign,
         (ARRAY['region_mb','region_mt','region_mn'])[1+floor(random()*3)::int] AS utm_content
) AS src
CROSS JOIN LATERAL (
  CASE
    WHEN gs.n % 20 IN (0,1,2,3,4,5,6,7,8,9) THEN 'new'
    WHEN gs.n % 20 IN (10,11,13,14,16) THEN 'contacted'
    WHEN gs.n % 20 IN (15,17,18) THEN 'qualified'
    WHEN gs.n % 20 = 19 THEN 'proposal'
    WHEN gs.n % 10 = 0 THEN 'won'
    WHEN random()<0.03 THEN 'lost'
    ELSE 'new'
  END AS stage
) AS stage_pool
ON CONFLICT DO NOTHING;

-- ============================================================================
-- VNG CORPORATION — 100 leads (large enterprise tenant)
-- ============================================================================
INSERT INTO leads.leads (id, tenant_id, owner_user_id, full_name, email, phone, source, utm_source, utm_medium, utm_campaign, utm_content, score, quality_score, predicted_ltv, conversion_probability, custom_fields, data, consent_given, consent_at, source_meta, stage, last_contacted_at, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'dddddddd-0000-0000-0000-000000000005',
  owner_pool.id,
  person.full_name,
  person.email,
  person.phone,
  src.source,
  src.utm_source,
  src.utm_medium,
  src.utm_campaign,
  src.utm_content,
  (random()*100)::int,
  CASE WHEN random()<0.3 THEN 'high' WHEN random()<0.6 THEN 'medium' ELSE 'low' END,
  (random()*100 + 20)::numeric(18,2)*1000000,
  (random()*0.8)::numeric(5,4),
  jsonb_build_object(
    'interest_level', (ARRAY['very_high','high','medium'])[1+floor(random()*3)::int],
    'budget_range', (ARRAY['100-500M VND','500M-2B VND','2-10B VND','10B+ VND'])[1+floor(random()*4)::int],
    'company_size', (ARRAY['10-50','50-200','200-1000','1000+'])[1+floor(random()*4)::int],
    'product_interest', (ARRAY['Zalo OA','Zalo ZNS','VNG Cloud','Game Integration','VNG Pay'])[1+floor(random()*5)::int],
    'decision_maker', random()<0.6,
    'industry', (ARRAY['Retail','Finance','Education','Healthcare','Manufacturing','Hospitality'])[1+floor(random()*6)::int]
  ),
  jsonb_build_object('city','HCMC','age',(28+random()*40)::int),
  random()<0.85,
  NOW() - (random()*365 || ' days')::interval,
  jsonb_build_object('campaign_id', md5(random()::text)),
  stage_pool.stage,
  NOW() - (random()*365 || ' days')::interval,
  NOW() - (random()*365 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval
FROM GENERATE_SERIES(1, 100) AS gs(n)
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan','Cao','Tạ','Châu','Đinh','Lý','Trịnh','Lâm','Đào','Lưu','Nghiêm','Võ','Phùng','Dương','Tống'])[1+floor(random()*29)::int]
    || ' ' ||
    (ARRAY['Văn','Thị','Hoàng','Hồng','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Kim','Sỹ','Quốc','Công','Thanh','Trọng','Mạnh','Tấn','Hải','Gia'])[1+floor(random()*20)::int]
    || ' ' ||
    (ARRAY['An','Bình','Cường','Dung','Em','Phượng','Giang','Hoa','Hải','Khánh','Long','Mai','Nam','Oanh','Phong','Quỳnh','Rạng','Sương','Tài','Uyên','Vinh','Xuân','Yên','Ánh','Bảo','Châu','Diệu','Hạnh','Kiều','Lan','Mận','Nga','Phương','Quang','Rồng','Sửu','Tỵ','Uyển','Vũ','Xuyến','Yến','Hà','Tuấn','Anh','Thư','Vy','Trâm','Huy','Hoa','Chi','Cường','Đạt','Hòa','Minh','Tâm','Long','Hưng','Linh','Phúc','Quân','Trung','Phong','Hải','Duy','Hiếu','Tú','Quy','Phước'])[1+floor(random()*67)::int]
    AS full_name
) AS person_named
CROSS JOIN LATERAL (
  SELECT replace(replace(person_named.full_name,' ',''),'ă','a') || '+' || gs.n || '@' ||
    (ARRAY['gmail.com','outlook.com','yahoo.com','fpt.com.vn','shopee.vn','lazada.vn','grab.com','vingroup.net','vingroup.com'])[1+floor(random()*9)::int]
    AS email
) AS person_email
CROSS JOIN LATERAL (
  SELECT '+84 ' || (90+floor(random()*9)::int)::text || ' ' || (1000000+floor(random()*8999999)::int)::text AS phone
) AS person_phone_calc
CROSS JOIN LATERAL (
  SELECT person_named.full_name AS full_name, person_email.email AS email, person_phone_calc.phone AS phone
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='dddddddd-0000-0000-0000-000000000005' AND role='member'
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['LinkedIn Outbound','Industry Event','Referral Partner','Website','Google Ads','Zalo Ads'])[1+floor(random()*6)::int] AS source,
         (ARRAY['linkedin','event','referral','organic','google','zalo'])[1+floor(random()*6)::int] AS utm_source,
         (ARRAY['social','event','partner','email','cpc'])[1+floor(random()*5)::int] AS utm_medium,
         (ARRAY['zalo_enterprise_q1','cloud_q1','cloud_q2','zalo_zns_launch','retail_summit_2025','dev_day_2025','fintech_2026'])[1+floor(random()*7)::int] AS utm_campaign,
         (ARRAY['ad_a','ad_b','ad_c'])[1+floor(random()*3)::int] AS utm_content
) AS src
CROSS JOIN LATERAL (
  CASE
    WHEN gs.n % 20 IN (0,1,2,3,4,5,6,7,8) THEN 'new'
    WHEN gs.n % 20 IN (9,10,11,12,13) THEN 'contacted'
    WHEN gs.n % 20 IN (14,15,16) THEN 'qualified'
    WHEN gs.n % 20 IN (17,18,19) THEN 'proposal'
    WHEN gs.n % 10 = 0 THEN 'won'
    WHEN random()<0.04 THEN 'lost'
    ELSE 'new'
  END AS stage
) AS stage_pool
ON CONFLICT DO NOTHING;