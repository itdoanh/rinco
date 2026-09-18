-- ============================================================
-- WS-D: ADDITIONAL DEALS + ACTIVITIES SPANNING 18 MONTHS
-- For each of 5 tenants: 30-50 deals + 200-400 activities
-- Deal amounts realistic VND range (15M-2B depending on tenant type)
-- Activities: realistic mix of call/email/meeting/demo/follow_up
-- ============================================================================

-- ============================================================================
-- APEX FINTECH — 40 deals + 350 activities (loan origination domain)
-- ============================================================================
INSERT INTO crm.deals (id, tenant_id, owner_user_id, name, stage, amount, currency, probability, expected_close_date, pipeline_id, source, custom_fields, description, created_at, updated_at, closed_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000001',
  owner_pool.id,
  'Gói vay ' || amount_pool.amount_label || ' - ' || person.full_name,
  stage_pool.stage,
  amount_pool.amount_vnd,
  'VND',
  amount_pool.probability,
  NOW() - (random()*30 || ' days')::interval,
  (SELECT id FROM crm.pipelines WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001' AND name LIKE '%Vay%' LIMIT 1),
  src.source,
  jsonb_build_object(
    'loan_term_months', (6+floor(random()*54)::int),
    'interest_rate', (8+random()*8)::numeric(4,2),
    'collateral_type', (ARRAY['Tin chap','The chap nha','The chap xe','The chap dat'])[1+floor(random()*4)::int],
    'product_type', (ARRAY['Vay mua nha','Vay mua xe','Vay kinh doanh','Vay tieu dung'])[1+floor(random()*4)::int]
  ),
  'Lead từ ' || src.source || ', quan tâm gói vay ' || amount_pool.amount_label,
  NOW() - (random()*540 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval,
  CASE WHEN stage_pool.stage='won' THEN NOW() - (random()*30 || ' days')::interval ELSE NULL END
FROM GENERATE_SERIES(1, 40) AS gs(n)
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan','Cao','Tạ','Châu','Đinh','Lý','Trịnh','Lâm','Đào','Lưu','Nghiêm','Võ','Phùng','Dương','Tống'])[1+floor(random()*29)::int]
    || ' ' ||
    (ARRAY['Văn','Thị','Hoàng','Hồng','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Kim','Sỹ','Quốc','Công','Thanh','Trọng','Mạnh','Tấn','Hải','Gia'])[1+floor(random()*20)::int]
    || ' ' ||
    (ARRAY['An','Bình','Cường','Dung','Em','Phượng','Giang','Hoa','Hải','Khánh','Long','Mai','Nam','Oanh','Phong','Quỳnh','Rạng','Sương','Tài','Uyên','Vinh','Xuân','Yên','Ánh','Bảo','Châu','Diệu','Hạnh','Kiều','Lan','Mận','Nga','Phương','Quang','Rồng','Sửu','Tỵ','Uyển','Vũ','Xuyến','Yến','Hà','Tuấn','Anh','Thư','Vy','Trâm','Huy','Hoa','Chi','Cường','Đạt','Hòa','Minh','Tâm','Long','Hưng','Linh','Phúc','Quân','Trung','Phong','Hải','Duy','Hiếu','Tú','Quy','Phước'])[1+floor(random()*67)::int]
    AS full_name
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001' AND role='member' AND parent_id IS NOT NULL
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  -- Realistic loan amounts: 15M - 500M VND
  SELECT
    CASE
        WHEN gs.n % 5 = 0 THEN (15+random()*35)::numeric(15,0)*1000000  -- 15-50M
        WHEN gs.n % 5 = 1 THEN (50+random()*150)::numeric(15,0)*1000000 -- 50-200M
        WHEN gs.n % 5 = 2 THEN (200+random()*300)::numeric(15,0)*1000000 -- 200-500M
        WHEN gs.n % 5 = 3 THEN (500+random()*500)::numeric(15,0)*1000000 -- 500M-1B
        ELSE (15+random()*85)::numeric(15,0)*1000000                       -- 15-100M
      END AS amount_vnd,
      CASE
        WHEN gs.n % 5 = 0 THEN '15-50 trieu'
        WHEN gs.n % 5 = 1 THEN '50-200 trieu'
        WHEN gs.n % 5 = 2 THEN '200-500 trieu'
        WHEN gs.n % 5 = 3 THEN '500M-1 ty'
        ELSE '15-100 trieu'
      END AS amount_label,
      CASE
        WHEN stage_pool.stage='won' THEN 0.9
        WHEN stage_pool.stage='proposal' THEN 0.6
        WHEN stage_pool.stage='qualified' THEN 0.4
        ELSE 0.2
      END AS probability
) AS amount_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Facebook Ads','Google Ads','TikTok Ads','Referral','Website'])[1+floor(random()*5)::int] AS source
) AS src
CROSS JOIN LATERAL (
  SELECT CASE
    WHEN gs.n % 20 IN (0,1,2,3,4,5,6) THEN 'prospecting'
    WHEN gs.n % 20 IN (7,8,9,10,11) THEN 'qualified'
    WHEN gs.n % 20 IN (12,13,14) THEN 'proposal'
    WHEN gs.n % 20 IN (15,16) THEN 'negotiation'
    WHEN gs.n % 20 IN (17,18) THEN 'won'
    WHEN gs.n % 20 = 19 THEN 'lost'
    ELSE 'prospecting'
  END AS stage
) AS stage_pool
ON CONFLICT DO NOTHING;

-- APEX activities (7+ per deal = 350 activities)
INSERT INTO crm.activities (id, tenant_id, deal_id, owner_user_id, type, subject, description, due_date, completed_at, created_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000001',
  deal_pool.id,
  deal_pool.owner_user_id,
  type_pool.activity_type,
  type_pool.subject,
  type_pool.description,
  NOW() - (random()*540 || ' days')::interval,
  CASE WHEN type_pool.completion>0.7 THEN NOW() - (random()*540 || ' days')::interval ELSE NULL END,
  NOW() - (random()*540 || ' days')::interval
FROM GENERATE_SERIES(1, 350) AS gs(n)
CROSS JOIN LATERAL (
  SELECT id, owner_user_id FROM crm.deals WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000001' ORDER BY random() LIMIT 1
) AS deal_pool
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['call','email','meeting','demo','whatsapp','note','follow_up'])[1+floor(random()*7)::int] AS activity_type,
    CASE
      WHEN activity_type='call' THEN 'Gọi điện tư vấn gói vay'
      WHEN activity_type='email' THEN 'Gửi email thông tin lãi suất'
      WHEN activity_type='meeting' THEN 'Hẹn gặp thẩm định tài sản'
      WHEN activity_type='demo' THEN 'Demo quy trình đăng ký online'
      WHEN activity_type='whatsapp' THEN 'Zalo tư vấn nhanh'
      WHEN activity_type='note' THEN 'Ghi chú lead yêu cầu gọi lại chiều'
      ELSE 'Follow-up sau khi gửi báo giá'
    END AS subject,
    CASE
      WHEN activity_type='call' THEN 'Lead quan tâm gói vay mua xe, cần tư vấn lãi suất cố định vs thả nổi'
      WHEN activity_type='email' THEN 'Gửi bảng tính lãi suất kèm điều kiện vay + hồ sơ cần chuẩn bị'
      WHEN activity_type='meeting' THEN 'Đặt lịch thẩm định giá tài sản tại ngân hàng, chuẩn bị CMND + sổ đỏ'
      WHEN activity_type='demo' THEN 'Hướng dẫn đăng ký app Apex Fintech, chụp CCCD, xác minh danh tính'
      WHEN activity_type='whatsapp' THEN 'Gửi Zalo: báo giá nhanh + file PDF điều khoản vay'
      WHEN activity_type='note' THEN 'Lead nói sẽ gọi lại sau giờ hành chính, prefer kênh Zalo hơn điện thoại'
      ELSE 'Gọi lại sau 2 ngày, xác nhận đã đọc email báo giá'
    END AS description,
    (0.3+random()*0.7)::numeric(3,2) AS completion
) AS type_pool
ON CONFLICT DO NOTHING;

-- ============================================================================
-- HCT CONSULTING — 30 deals (BDS domain, much larger amounts)
-- ============================================================================
INSERT INTO crm.deals (id, tenant_id, owner_user_id, name, stage, amount, currency, probability, expected_close_date, pipeline_id, source, custom_fields, description, created_at, updated_at, closed_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000002',
  owner_pool.id,
  'Booking ' || property_pool.property_type || ' ' || property_pool.project || ' - ' || person.full_name,
  stage_pool.stage,
  property_pool.amount_vnd,
  'VND',
  property_pool.probability,
  NOW() - (random()*30 || ' days')::interval,
  (SELECT id FROM crm.pipelines WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000002' LIMIT 1),
  src.source,
  jsonb_build_object(
    'project', property_pool.project,
    'unit_code', 'A-' || (101+floor(random()*899)::int)::text,
    'floor', (5+floor(random()*30)::int),
    'area_m2', (40+random()*120)::numeric(5,1),
    'bedrooms', (1+floor(random()*4)::int),
    'view', (ARRAY['Thành phố','Sông','Hồ bơi','Công viên','Biển'])[1+floor(random()*5)::int]
  ),
  'Booking ' || property_pool.property_type || ' tại ' || property_pool.project,
  NOW() - (random()*540 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval,
  CASE WHEN stage_pool.stage='won' THEN NOW() - (random()*30 || ' days')::interval ELSE NULL END
FROM GENERATE_SERIES(1, 30) AS gs(n)
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan','Cao','Tạ','Châu','Đinh','Lý','Trịnh','Lâm','Đào','Lưu','Nghiêm','Võ','Phùng','Dương','Tống'])[1+floor(random()*29)::int]
    || ' ' ||
    (ARRAY['Văn','Thị','Hoàng','Hồng','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Kim','Sỹ','Quốc','Công','Thanh','Trọng','Mạnh','Tấn','Hải','Gia'])[1+floor(random()*20)::int]
    || ' ' ||
    (ARRAY['An','Bình','Cường','Dung','Em','Phượng','Giang','Hoa','Hải','Khánh','Long','Mai','Nam','Oanh','Phong','Quỳnh','Rạng','Sương','Tài','Uyên','Vinh','Xuân','Yên','Ánh','Bảo','Châu','Diệu','Hạnh','Kiều','Lan','Mận','Nga','Phương','Quang','Rồng','Sửu','Tỵ','Uyển','Vũ','Xuyến','Yến','Hà','Tuấn','Anh','Thư','Vy','Trâm','Huy','Hoa','Chi','Cường','Đạt','Hòa','Minh','Tâm','Long','Hưng','Linh','Phúc','Quân','Trung','Phong','Hải','Duy','Hiếu','Tú','Quy','Phước'])[1+floor(random()*67)::int]
    AS full_name
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000002' AND role='member'
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  -- BDS amounts: 2B - 50B VND
  SELECT
    CASE
        WHEN gs.n % 5 = 0 THEN (2+random()*5)::numeric(15,1)*1000000000   -- 2-7B (can ho)
        WHEN gs.n % 5 = 1 THEN (5+random()*15)::numeric(15,1)*1000000000  -- 5-20B (shophouse)
        WHEN gs.n % 5 = 2 THEN (10+random()*40)::numeric(15,1)*1000000000 -- 10-50B (penthouse/villa)
        WHEN gs.n % 5 = 3 THEN (1+random()*3)::numeric(15,1)*1000000000   -- 1-4B (dat nen)
        ELSE (3+random()*8)::numeric(15,1)*1000000000
      END AS amount_vnd,
      (ARRAY['Vinhomes Grand Park','Vinhomes Ocean Park','Masteri Thao Dien','The Maris','Lumiere Riverside','The Sun Avenue'])[1+floor(random()*6)::int] AS project,
      (ARRAY['can ho','shophouse','penthouse','dat nen','villa','nha pho'])[1+floor(random()*6)::int] AS property_type,
      CASE
        WHEN stage_pool.stage='won' THEN 0.9
        WHEN stage_pool.stage='proposal' THEN 0.6
        WHEN stage_pool.stage='qualified' THEN 0.4
        ELSE 0.2
      END AS probability
) AS property_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Facebook Ads','Google Ads','Referral','LinkedIn','Bao chi','Event'])[1+floor(random()*6)::int] AS source
) AS src
CROSS JOIN LATERAL (
  SELECT CASE
    WHEN gs.n % 20 IN (0,1,2,3,4,5,6) THEN 'prospecting'
    WHEN gs.n % 20 IN (7,8,9,10,11) THEN 'qualified'
    WHEN gs.n % 20 IN (12,13,14) THEN 'proposal'
    WHEN gs.n % 20 IN (15,16) THEN 'negotiation'
    WHEN gs.n % 20 IN (17,18) THEN 'won'
    WHEN gs.n % 20 = 19 THEN 'lost'
    ELSE 'prospecting'
  END AS stage
) AS stage_pool
ON CONFLICT DO NOTHING;

-- HCT activities (300)
INSERT INTO crm.activities (id, tenant_id, deal_id, owner_user_id, type, subject, description, due_date, completed_at, created_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000002',
  deal_pool.id,
  deal_pool.owner_user_id,
  type_pool.activity_type,
  type_pool.subject,
  type_pool.description,
  NOW() - (random()*540 || ' days')::interval,
  CASE WHEN type_pool.completion>0.7 THEN NOW() - (random()*540 || ' days')::interval ELSE NULL END,
  NOW() - (random()*540 || ' days')::interval
FROM GENERATE_SERIES(1, 300) AS gs(n)
CROSS JOIN LATERAL (
  SELECT id, owner_user_id FROM crm.deals WHERE tenant_id='aaaaaaaa-0000-0000-0000-000000000002' ORDER BY random() LIMIT 1
) AS deal_pool
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['call','meeting','viewing','email','whatsapp','note','follow_up','contract'])[1+floor(random()*8)::int] AS activity_type,
    CASE
      WHEN activity_type='call' THEN 'Gọi điện xác nhận nhu cầu mua BĐS'
      WHEN activity_type='meeting' THEN 'Hẹn cafe thảo luận ngân sách'
      WHEN activity_type='viewing' THEN 'Đưa khách đi xem căn hộ mẫu'
      WHEN activity_type='email' THEN 'Gửi bảng giá + chính sách thanh toán'
      WHEN activity_type='whatsapp' THEN 'Zalo gửi video flycam dự án'
      WHEN activity_type='note' THEN 'Khách cần tư vấn thêm về pháp lý, hẹn gặp luật sư'
      WHEN activity_type='follow_up' THEN 'Follow-up sau khi khách đi xem nhà'
      ELSE 'Chuẩn bị hợp đồng đặt cọc'
    END AS subject,
    CASE
      WHEN activity_type='call' THEN 'Lead quan tâm căn 3PN, view sông, tầng 18-25, tài chính sẵn 30%'
      WHEN activity_type='meeting' THEN 'Gặp khách tại cafe Giảng, thảo luận tài chính và chính sách vay NH'
      WHEN activity_type='viewing' THEN 'Đưa khách xem căn B-1805, view hồ bơi + công viên, 3PN, 89m2'
      WHEN activity_type='email' THEN 'Gửi file PDF bảng giá + chính sách ưu đãi quý 3/2025'
      WHEN activity_type='whatsapp' THEN 'Gửi link YouTube flycam toàn cảnh dự án + 3D căn hộ mẫu'
      WHEN activity_type='note' THEN 'Khách lo về pháp lý sổ đỏ, hẹn gặp luật sư công ty vào thứ 6 tuần sau'
      WHEN activity_type='follow_up' THEN 'Gọi lại sau khi khách đi xem căn 1806 để confirm quyết định'
      ELSE 'Soạn hợp đồng đặt cọc 100 triệu, gửi phòng pháp chế review'
    END AS description,
    (0.3+random()*0.7)::numeric(3,2) AS completion
) AS type_pool
ON CONFLICT DO NOTHING;

-- ============================================================================
-- DEMO COMPANY — 20 deals (smaller, recent)
-- ============================================================================
INSERT INTO crm.deals (id, tenant_id, owner_user_id, name, stage, amount, currency, probability, expected_close_date, pipeline_id, source, custom_fields, description, created_at, updated_at, closed_at)
SELECT
  gen_random_uuid(),
  'bbbbbbbb-0000-0000-0000-000000000003',
  owner_pool.id,
  'Demo subscription - ' || person.full_name,
  stage_pool.stage,
  ((2+random()*8)::numeric(15,0)*1000000)::numeric(15,2),  -- 2-10M VND
  'VND',
  CASE
    WHEN stage_pool.stage='won' THEN 0.9
    WHEN stage_pool.stage='proposal' THEN 0.6
    WHEN stage_pool.stage='qualified' THEN 0.4
    ELSE 0.2
  END,
  NOW() - (random()*10 || ' days')::interval,
  (SELECT id FROM crm.pipelines WHERE tenant_id='bbbbbbbb-0000-0000-0000-000000000003' LIMIT 1),
  'Demo Signup',
  jsonb_build_object('plan',(ARRAY['starter','pro','business'])[1+floor(random()*3)::int],'seats',(3+floor(random()*15)::int)),
  'Free trial customer - now in conversion pipeline',
  NOW() - (random()*10 || ' days')::interval,
  NOW() - (random()*5 || ' days')::interval,
  CASE WHEN stage_pool.stage='won' THEN NOW() - (random()*5 || ' days')::interval ELSE NULL END
FROM GENERATE_SERIES(1, 20) AS gs(n)
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan'])[1+floor(random()*15)::int]
    || ' ' ||
    (ARRAY['Văn','Thị','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Hồng','Tấn'])[1+floor(random()*10)::int]
    || ' ' ||
    (ARRAY['An','Bình','Cường','Dung','Hải','Khánh','Long','Mai','Nam','Phong','Quỳnh','Tài','Uyên','Vinh','Xuân','Ánh','Châu','Hạnh','Lan','Phương','Yến'])[1+floor(random()*21)::int]
    AS full_name
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='bbbbbbbb-0000-0000-0000-000000000003' AND role='member'
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT CASE
    WHEN gs.n % 10 IN (0,1,2,3,4,5) THEN 'trial'
    WHEN gs.n % 10 IN (6,7) THEN 'qualified'
    WHEN gs.n % 10 = 8 THEN 'proposal'
    ELSE 'won'
  END AS stage
) AS stage_pool
ON CONFLICT DO NOTHING;

-- DEMO activities (200)
INSERT INTO crm.activities (id, tenant_id, deal_id, owner_user_id, type, subject, description, due_date, completed_at, created_at)
SELECT
  gen_random_uuid(),
  'bbbbbbbb-0000-0000-0000-000000000003',
  deal_pool.id,
  deal_pool.owner_user_id,
  type_pool.activity_type,
  type_pool.subject,
  type_pool.description,
  NOW() - (random()*10 || ' days')::interval,
  CASE WHEN type_pool.completion>0.7 THEN NOW() - (random()*10 || ' days')::interval ELSE NULL END,
  NOW() - (random()*10 || ' days')::interval
FROM GENERATE_SERIES(1, 200) AS gs(n)
CROSS JOIN LATERAL (
  SELECT id, owner_user_id FROM crm.deals WHERE tenant_id='bbbbbbbb-0000-0000-0000-000000000003' ORDER BY random() LIMIT 1
) AS deal_pool
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['email','call','demo','note'])[1+floor(random()*4)::int] AS activity_type,
    CASE
      WHEN activity_type='email' THEN 'Gửi email onboarding trial user'
      WHEN activity_type='call' THEN 'Gọi check-in sau 3 ngày trial'
      WHEN activity_type='demo' THEN 'Demo tính năng CRM + Lead Scoring'
      ELSE 'Note: user quan tâm module Báo cáo AI'
    END AS subject,
    CASE
      WHEN activity_type='email' THEN 'Cảm ơn đã đăng ký trial, hướng dẫn bắt đầu với 3 bước đơn giản'
      WHEN activity_type='call' THEN 'User đang dùng trial, hỏi về pricing nâng cấp pro'
      WHEN activity_type='demo' THEN 'Demo 30 phút tính năng CRM + AI scoring, user impressed'
      ELSE 'User tâm đắc module báo cáo AI, hỏi giá enterprise'
    END AS description,
    (0.4+random()*0.6)::numeric(3,2) AS completion
) AS type_pool
ON CONFLICT DO NOTHING;

-- ============================================================================
-- VINAMILK DISTRIBUTION — 35 deals (dealer orders)
-- ============================================================================
INSERT INTO crm.deals (id, tenant_id, owner_user_id, name, stage, amount, currency, probability, expected_close_date, pipeline_id, source, custom_fields, description, created_at, updated_at, closed_at)
SELECT
  gen_random_uuid(),
  'cccccccc-0000-0000-0000-000000000004',
  owner_pool.id,
  'Đơn hàng ' || product_pool.product || ' - ' || person.full_name,
  stage_pool.stage,
  ((random()*800+50)::numeric(15,0)*1000000)::numeric(15,2),  -- 50M-850M VND
  'VND',
  CASE
    WHEN stage_pool.stage='won' THEN 0.9
    WHEN stage_pool.stage='proposal' THEN 0.6
    ELSE 0.3
  END,
  NOW() - (random()*30 || ' days')::interval,
  (SELECT id FROM crm.pipelines WHERE tenant_id='cccccccc-0000-0000-0000-000000000004' LIMIT 1),
  src.source,
  jsonb_build_object(
    'product', product_pool.product,
    'volume_cases', (100+floor(random()*5000)::int),
    'delivery_region', (ARRAY['Miền Bắc','Miền Trung','Miền Nam'])[1+floor(random()*3)::int],
    'delivery_window', (ARRAY['Tuần 1','Tuần 2','Tuần 3','Tuần 4'])[1+floor(random()*4)::int]
  ),
  'Đơn hàng ' || product_pool.product || ' đại lý ' || person.full_name,
  NOW() - (random()*540 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval,
  CASE WHEN stage_pool.stage='won' THEN NOW() - (random()*30 || ' days')::interval ELSE NULL END
FROM GENERATE_SERIES(1, 35) AS gs(n)
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan','Cao','Tạ','Châu','Đinh','Lý','Trịnh','Lâm','Đào','Lưu','Nghiêm','Võ','Phùng','Dương','Tống'])[1+floor(random()*29)::int]
    || ' ' ||
    (ARRAY['Văn','Thị','Hoàng','Hồng','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Kim','Sỹ','Quốc','Công','Thanh','Trọng','Mạnh','Tấn','Hải','Gia'])[1+floor(random()*20)::int]
    || ' ' ||
    (ARRAY['An','Bình','Cường','Dung','Em','Phượng','Giang','Hoa','Hải','Khánh','Long','Mai','Nam','Oanh','Phong','Quỳnh','Rạng','Sương','Tài','Uyên','Vinh','Xuân','Yên','Ánh','Bảo','Châu','Diệu','Hạnh','Kiều','Lan','Mận','Nga','Phương','Quang','Rồng','Sửu','Tỵ','Uyển','Vũ','Xuyến','Yến','Hà','Tuấn','Anh','Thư','Vy','Trâm','Huy','Hoa','Chi','Cường','Đạt','Hòa','Minh','Tâm','Long','Hưng','Linh','Phúc','Quân','Trung','Phong','Hải','Duy','Hiếu','Tú','Quy','Phước'])[1+floor(random()*67)::int]
    AS full_name
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='cccccccc-0000-0000-0000-000000000004' AND role='member'
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Sua tuoi Vinamilk','Sua chua YoMost','Sua bot Dielac','Nuoc giai khat','Sua dong hop'])[1+floor(random()*5)::int] AS product
) AS product_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['Field Sales','Phone Outbound','Trade Show'])[1+floor(random()*3)::int] AS source
) AS src
CROSS JOIN LATERAL (
  SELECT CASE
    WHEN gs.n % 20 IN (0,1,2,3,4,5,6,7,8,9,10) THEN 'prospecting'
    WHEN gs.n % 20 IN (11,12,13,14) THEN 'qualified'
    WHEN gs.n % 20 IN (15,16,17) THEN 'proposal'
    WHEN gs.n % 20 IN (18,19) THEN 'won'
    WHEN random()<0.05 THEN 'lost'
    ELSE 'prospecting'
  END AS stage
) AS stage_pool
ON CONFLICT DO NOTHING;

-- VINAMILK activities (250)
INSERT INTO crm.activities (id, tenant_id, deal_id, owner_user_id, type, subject, description, due_date, completed_at, created_at)
SELECT
  gen_random_uuid(),
  'cccccccc-0000-0000-0000-000000000004',
  deal_pool.id,
  deal_pool.owner_user_id,
  type_pool.activity_type,
  type_pool.subject,
  type_pool.description,
  NOW() - (random()*540 || ' days')::interval,
  CASE WHEN type_pool.completion>0.7 THEN NOW() - (random()*540 || ' days')::interval ELSE NULL END,
  NOW() - (random()*540 || ' days')::interval
FROM GENERATE_SERIES(1, 250) AS gs(n)
CROSS JOIN LATERAL (
  SELECT id, owner_user_id FROM crm.deals WHERE tenant_id='cccccccc-0000-0000-0000-000000000004' ORDER BY random() LIMIT 1
) AS deal_pool
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['visit','call','order','note','follow_up'])[1+floor(random()*5)::int] AS activity_type,
    CASE
      WHEN activity_type='visit' THEN 'Thăm đại lý giới thiệu sản phẩm mới'
      WHEN activity_type='call' THEN 'Gọi confirm đơn hàng tháng'
      WHEN activity_type='order' THEN 'Nhận đơn hàng và sắp xếp giao nhận'
      WHEN activity_type='note' THEN 'Đại lý yêu cầu bổ sung chương trình khuyến mãi'
      ELSE 'Follow-up đơn hàng tháng sau'
    END AS subject,
    CASE
      WHEN activity_type='visit' THEN 'Đến đại lý cấp 1 tại quận Bình Thạnh, trình bày combo sữa tươi + sữa chua'
      WHEN activity_type='call' THEN 'Confirm đơn 500 thùng sữa tươi, giao vào thứ 2 tuần sau'
      WHEN activity_type='order' THEN 'Đơn 200 thùng YoMost 1L, giao tới kho đại lý Bình Dương trong 3 ngày'
      WHEN activity_type='note' THEN 'Đại lý phản ánh sữa Dielake bị cạnh tranh bởi Nutifood, cần chiết khấu thêm'
      ELSE 'Check-in đại lý trước khi chốt đơn tháng sau'
    END AS description,
    (0.5+random()*0.5)::numeric(3,2) AS completion
) AS type_pool
ON CONFLICT DO NOTHING;

-- ============================================================================
-- VNG CORPORATION — 50 deals (enterprise B2B)
-- ============================================================================
INSERT INTO crm.deals (id, tenant_id, owner_user_id, name, stage, amount, currency, probability, expected_close_date, pipeline_id, source, custom_fields, description, created_at, updated_at, closed_at)
SELECT
  gen_random_uuid(),
  'dddddddd-0000-0000-0000-000000000005',
  owner_pool.id,
  product_pool.product || ' - ' || person.full_name || ' - ' || company_pool.company,
  stage_pool.stage,
  ((50+random()*950)::numeric(15,0)*1000000)::numeric(15,2),  -- 50M-1B VND
  'VND',
  CASE
    WHEN stage_pool.stage='won' THEN 0.9
    WHEN stage_pool.stage='proposal' THEN 0.6
    WHEN stage_pool.stage='qualified' THEN 0.4
    ELSE 0.2
  END,
  NOW() - (random()*30 || ' days')::interval,
  (SELECT id FROM crm.pipelines WHERE tenant_id='dddddddd-0000-0000-0000-000000000005' LIMIT 1),
  src.source,
  jsonb_build_object(
    'product', product_pool.product,
    'company_size', company_pool.size,
    'industry', company_pool.industry,
    'annual_contract', random()<0.6,
    'demo_completed', random()<0.7
  ),
  'Enterprise deal - ' || product_pool.product || ' for ' || company_pool.company,
  NOW() - (random()*365 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval,
  CASE WHEN stage_pool.stage='won' THEN NOW() - (random()*30 || ' days')::interval ELSE NULL END
FROM GENERATE_SERIES(1, 50) AS gs(n)
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan','Cao','Tạ','Châu','Đinh','Lý','Trịnh','Lâm','Đào','Lưu','Nghiêm','Võ','Phùng','Dương','Tống'])[1+floor(random()*29)::int]
    || ' ' ||
    (ARRAY['Văn','Thị','Hoàng','Hồng','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Kim','Sỹ','Quốc','Công','Thanh','Trọng','Mạnh','Tấn','Hải','Gia'])[1+floor(random()*20)::int]
    || ' ' ||
    (ARRAY['An','Bình','Cường','Dung','Em','Phượng','Giang','Hoa','Hải','Khánh','Long','Mai','Nam','Oanh','Phong','Quỳnh','Rạng','Sương','Tài','Uyên','Vinh','Xuân','Yên','Ánh','Bảo','Châu','Diệu','Hạnh','Kiều','Lan','Mận','Nga','Phương','Quang','Rồng','Sửu','Tỵ','Uyển','Vũ','Xuyến','Yến','Hà','Tuấn','Anh','Thư','Vy','Trâm','Huy','Hoa','Chi','Cường','Đạt','Hòa','Minh','Tâm','Long','Hưng','Linh','Phúc','Quân','Trung','Phong','Hải','Duy','Hiếu','Tú','Quy','Phước'])[1+floor(random()*67)::int]
    AS full_name
) AS person
CROSS JOIN LATERAL (
  SELECT id FROM auth.users
   WHERE tenant_id='dddddddd-0000-0000-0000-000000000005' AND role='member'
   ORDER BY random() LIMIT 1
) AS owner_pool
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['Zalo OA Enterprise','Zalo ZNS','VNG Cloud VM','VNG Cloud Storage','Game SDK','VNG Pay'])[1+floor(random()*6)::int] AS product
) AS product_pool
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['FPT','Vinamilk','VPBank','Techcombank','Masan','Vietjet','Tiki','MoMo','Hòa Phát','Sun Group','CellphoneS','The Coffee House','Phúc Long','Highlands','Bách Hóa Xanh'])[1+floor(random()*15)::int] AS company,
    (ARRAY['10-50','50-200','200-1000','1000-5000','5000+'])[1+floor(random()*5)::int] AS size,
    (ARRAY['Retail','Finance','Education','Healthcare','Manufacturing','Hospitality','Logistics'])[1+floor(random()*7)::int] AS industry
) AS company_pool
CROSS JOIN LATERAL (
  SELECT (ARRAY['LinkedIn Outbound','Industry Event','Referral Partner','Website'])[1+floor(random()*4)::int] AS source
) AS src
CROSS JOIN LATERAL (
  SELECT CASE
    WHEN gs.n % 20 IN (0,1,2,3,4,5,6) THEN 'prospecting'
    WHEN gs.n % 20 IN (7,8,9,10,11) THEN 'qualified'
    WHEN gs.n % 20 IN (12,13,14) THEN 'proposal'
    WHEN gs.n % 20 IN (15,16) THEN 'negotiation'
    WHEN gs.n % 20 IN (17,18,19) THEN 'won'
    ELSE 'prospecting'
  END AS stage
) AS stage_pool
ON CONFLICT DO NOTHING;

-- VNG activities (400)
INSERT INTO crm.activities (id, tenant_id, deal_id, owner_user_id, type, subject, description, due_date, completed_at, created_at)
SELECT
  gen_random_uuid(),
  'dddddddd-0000-0000-0000-000000000005',
  deal_pool.id,
  deal_pool.owner_user_id,
  type_pool.activity_type,
  type_pool.subject,
  type_pool.description,
  NOW() - (random()*365 || ' days')::interval,
  CASE WHEN type_pool.completion>0.7 THEN NOW() - (random()*365 || ' days')::interval ELSE NULL END,
  NOW() - (random()*365 || ' days')::interval
FROM GENERATE_SERIES(1, 400) AS gs(n)
CROSS JOIN LATERAL (
  SELECT id, owner_user_id FROM crm.deals WHERE tenant_id='dddddddd-0000-0000-0000-000000000005' ORDER BY random() LIMIT 1
) AS deal_pool
CROSS JOIN LATERAL (
  SELECT
    (ARRAY['call','meeting','demo','email','linkedin','note','proposal','security_review','legal_review'])[1+floor(random()*9)::int] AS activity_type,
    CASE
      WHEN activity_type='call' THEN 'Discovery call với CTO/CMO'
      WHEN activity_type='meeting' THEN 'Onsite presentation cho leadership team'
      WHEN activity_type='demo' THEN 'Demo sản phẩm Zalo OA cho team marketing'
      WHEN activity_type='email' THEN 'Gửi RFP response + pricing breakdown'
      WHEN activity_type='linkedin' THEN 'LinkedIn outreach follow-up'
      WHEN activity_type='note' THEN 'Note: khách cần bảo mật SOC2 compliance'
      WHEN activity_type='proposal' THEN 'Gửi proposal v2 sau khi tinh chỉnh giá'
      WHEN activity_type='security_review' THEN 'Customer security team review architecture'
      ELSE 'Legal review hợp đồng + SLA'
    END AS subject,
    CASE
      WHEN activity_type='call' THEN 'CTO quan tâm ZNS cho gửi OTP, scale 100K msg/day, cần pricing tier'
      WHEN activity_type='meeting' THEN 'Onsite tại VPBank tower, trình bày cho CTO + CISO + procurement lead'
      WHEN activity_type='demo' THEN 'Demo 60 phút cho team marketing Techcombank, highlight Zalo ZNS CRM integration'
      WHEN activity_type='email' THEN 'RFP response 12 trang, bao gồm pricing tier enterprise + SLA 99.99%'
      WHEN activity_type='linkedin' THEN 'Follow-up sau hội nghị Retail Tech Summit, gửi case study Masan'
      WHEN activity_type='note' THEN 'Khách yêu cầu chứng nhận ISO 27001 + SOC2 Type II, security team cần review'
      WHEN activity_type='proposal' THEN 'Proposal v2: discount 15% cho 3-year commitment, thêm 200K free ZNS credits'
      WHEN activity_type='security_review' THEN 'Security team review kiến trúc multi-region, penetration test results'
      ELSE 'Legal review: thỏa thuận xử lý dữ liệu (DPA), điều khoản thanh toán 60 ngày'
    END AS description,
    (0.3+random()*0.7)::numeric(3,2) AS completion
) AS type_pool
ON CONFLICT DO NOTHING;
