-- ============================================================
-- DEALS + ACTIVITIES + NOTES SEED (Loop 202)
-- ============================================================

-- ===== APEX FINTECH DEALS (30) =====
INSERT INTO crm.deals (id, tenant_id, lead_id, owner_user_id, pipeline_id, stage, title, value, currency, probability, expected_close_date, status, data, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000001',
  lead.id,
  lead.owner_user_id,
  'c1000001-0000-0000-0000-000000000001',
  CASE lead.stage WHEN 'qualified' THEN 'proposal' WHEN 'proposal' THEN 'won' WHEN 'won' THEN 'won' ELSE 'contacted' END,
  'Khoản vay ' || lead.full_name || ' - ' || (lead.custom_fields->>'loan_purpose'),
  (10 + random()*4900)::numeric(18,2)*1000000,
  'VND',
  CASE lead.stage WHEN 'qualified' THEN 65 WHEN 'proposal' THEN 80 WHEN 'won' THEN 100 ELSE 25 END,
  CURRENT_DATE + (random()*60)::int,
  CASE WHEN lead.stage = 'won' THEN 'WON' WHEN lead.stage = 'lost' THEN 'LOST' ELSE 'OPEN' END,
  jsonb_build_object('source',lead.source,'utm_campaign',lead.utm_campaign,'city',lead.data->>'city'),
  lead.created_at + INTERVAL '1 day',
  NOW()
FROM leads.leads lead
WHERE lead.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
  AND lead.stage IN ('qualified','proposal','won')
ORDER BY random()
LIMIT 30
ON CONFLICT (id) DO NOTHING;

-- ===== HCT CONSULTING DEALS (30) =====
INSERT INTO crm.deals (id, tenant_id, lead_id, owner_user_id, pipeline_id, stage, title, value, currency, probability, expected_close_date, status, data, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000002',
  lead.id,
  lead.owner_user_id,
  'c1000002-0000-0000-0000-000000000001',
  CASE lead.stage WHEN 'qualified' THEN 'viewing' WHEN 'proposal' THEN 'negotiation' WHEN 'won' THEN 'won' ELSE 'contacted' END,
  'Mua BĐS ' || lead.full_name || ' - ' || COALESCE((lead.custom_fields->>'property_type'),'can_ho'),
  (2 + random()*98)::numeric(18,2)*1000000000,
  'VND',
  CASE lead.stage WHEN 'qualified' THEN 40 WHEN 'proposal' THEN 75 WHEN 'won' THEN 100 ELSE 20 END,
  CURRENT_DATE + (random()*90)::int,
  CASE WHEN lead.stage = 'won' THEN 'WON' WHEN lead.stage = 'lost' THEN 'LOST' ELSE 'OPEN' END,
  jsonb_build_object('property_type',lead.custom_fields->>'property_type','location',lead.custom_fields->>'preferred_location','investment',lead.custom_fields->>'investment_horizon'),
  lead.created_at + INTERVAL '2 days',
  NOW()
FROM leads.leads lead
WHERE lead.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000002'
  AND lead.stage IN ('qualified','proposal','won')
ORDER BY random()
LIMIT 30
ON CONFLICT (id) DO NOTHING;

-- ===== DEMO DEALS (15) =====
INSERT INTO crm.deals (id, tenant_id, lead_id, owner_user_id, pipeline_id, stage, title, value, currency, probability, expected_close_date, status, data, created_at, updated_at)
SELECT
  gen_random_uuid(),
  'bbbbbbbb-0000-0000-0000-000000000003',
  lead.id,
  lead.owner_user_id,
  'c1000003-0000-0000-0000-000000000001',
  CASE lead.stage WHEN 'qualified' THEN 'proposal' WHEN 'won' THEN 'won' ELSE 'contacted' END,
  'Demo Deal - ' || lead.full_name,
  (1 + random()*99)::numeric(18,2)*1000000,
  'VND',
  CASE lead.stage WHEN 'qualified' THEN 60 WHEN 'won' THEN 100 ELSE 30 END,
  CURRENT_DATE + (random()*30)::int,
  CASE WHEN lead.stage = 'won' THEN 'WON' WHEN lead.stage = 'lost' THEN 'LOST' ELSE 'OPEN' END,
  '{}'::jsonb,
  lead.created_at + INTERVAL '1 day',
  NOW()
FROM leads.leads lead
WHERE lead.tenant_id = 'bbbbbbbb-0000-0000-0000-000000000003'
  AND lead.stage IN ('qualified','proposal','won')
ORDER BY random()
LIMIT 15
ON CONFLICT (id) DO NOTHING;

-- ===== ACTIVITIES (100+ per tenant) =====
INSERT INTO crm.activities (id, tenant_id, user_id, lead_id, deal_id, type, title, description, due_at, completed_at, status, priority, data, created_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000001',
  act.owner,
  act.lead_id,
  NULL::uuid,
  act.type,
  act.title,
  act.desc,
  act.due_at,
  CASE WHEN act.type != 'task' THEN act.due_at ELSE NULL END,
  CASE WHEN act.type != 'task' THEN 'completed' ELSE 'pending' END,
  (ARRAY['low','medium','high','urgent'])[ceil(random()*4)],
  jsonb_build_object('duration_minutes',(15+random()*105)::int,'result',act.result),
  act.due_at - INTERVAL '2 hours'
FROM (
  SELECT l.owner_user_id AS owner, l.id AS lead_id,
    (ARRAY['call','call','email','email','meeting','meeting','note','task'])[ceil(random()*8)] AS type,
    (ARRAY[
      'Gọi điện giới thiệu sản phẩm',
      'Follow-up sau email đầu tiên',
      'Gặp mặt trực tiếp tư vấn',
      'Gửi báo giá chi tiết',
      'Gọi xác nhận nhu cầu',
      'Gửi hồ sơ vay mẫu',
      'Họp team review deal',
      'Gửi email cảm ơn sau meeting',
      'Gọi check-in định kỳ',
      'Gửi tài liệu pháp lý'
    ])[ceil(random()*10)] AS title,
    (ARRAY[
      'Khách hàng quan tâm, hẹn gặp tuần sau',
      'Đã gửi email, chờ phản hồi trong 48h',
      'Gặp gỡ tại văn phòng, khách hàng rất hào hứng',
      'Đang so sánh với đối thủ, cần thêm thông tin',
      'Đủ điều kiện vay, tiến hành thủ tục',
      'Khách hàng cần thêm thời gian suy nghĩ',
      'Đã xác nhận deal, chờ ký hợp đồng',
      'Cần follow-up lại trong 1 tuần'
    ])[ceil(random()*8)] AS desc,
    'Tích cực' AS result,
    NOW() - (random()*25 || ' days')::interval AS due_at
  FROM leads.leads l
  WHERE l.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
  ORDER BY random()
  LIMIT 120
) AS act
ON CONFLICT (id) DO NOTHING;

-- HCT activities
INSERT INTO crm.activities (id, tenant_id, user_id, lead_id, deal_id, type, title, description, due_at, completed_at, status, priority, created_at)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000002',
  act.owner,
  act.lead_id,
  NULL::uuid,
  act.type,
  act.title,
  act.desc,
  act.due_at,
  CASE WHEN act.type != 'task' THEN act.due_at ELSE NULL END,
  CASE WHEN act.type != 'task' THEN 'completed' ELSE 'pending' END,
  (ARRAY['low','medium','high'])[ceil(random()*3)],
  act.due_at - INTERVAL '1 hour'
FROM (
  SELECT l.owner_user_id AS owner, l.id AS lead_id,
    (ARRAY['call','call','meeting','meeting','note','task'])[ceil(random()*6)] AS type,
    (ARRAY[
      'Gọi giới thiệu dự án Vinhomes',
      'Hẹn đi xem căn hộ mẫu',
      'Gửi bảng giá shophouse',
      'Tư vấn pháp lý đất nền',
      'Gọi follow-up sau buổi xem',
      'Họp phân tích deal đầu tư',
      'Gửi hồ sơ tín dụng',
      'Đàm phán giá cuối cùng'
    ])[ceil(random()*8)] AS title,
    (ARRAY[
      'Khách rất quan tâm, muốn mua 2 căn',
      'Đã đi xem, đang so sánh với đối thủ',
      'Cần tài liệu pháp lý chi tiết hơn',
      'Chờ quyết định từ gia đình',
      'Deal tiềm năng cao, nên theo sát',
      'Khách đã có financing, chỉ cần deal',
      'Cần gặp lại tuần này',
      'Đàm phán giá thành công'
    ])[ceil(random()*8)] AS desc,
    NOW() - (random()*25 || ' days')::interval AS due_at
  FROM leads.leads l
  WHERE l.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000002'
  ORDER BY random()
  LIMIT 100
) AS act
ON CONFLICT (id) DO NOTHING;

-- Demo activities
INSERT INTO crm.activities (id, tenant_id, user_id, lead_id, type, title, description, due_at, completed_at, status, created_at)
SELECT
  gen_random_uuid(),
  'bbbbbbbb-0000-0000-0000-000000000003',
  l.owner_user_id,
  l.id,
  (ARRAY['call','email','meeting','note'])[ceil(random()*4)],
  'Demo activity - ' || l.full_name,
  'Automated demo activity for testing',
  NOW() - (random()*10 || ' days')::interval,
  NOW() - (random()*10 || ' days')::interval,
  'completed',
  NOW() - (random()*10 || ' days')::interval
FROM leads.leads l
WHERE l.tenant_id = 'bbbbbbbb-0000-0000-0000-000000000003'
ORDER BY random()
LIMIT 50
ON CONFLICT (id) DO NOTHING;

-- ===== NOTES (30 per tenant) =====
INSERT INTO crm.notes (id, tenant_id, contact_id, deal_id, body, author_id, is_pinned, created_at)
SELECT gen_random_uuid(), n.tenant_id, NULL::uuid, d.id, n.body, n.author, random()<0.3, n.created_at
FROM (
  SELECT tenant_id, deal_id, author, body, created_at,
    ROW_NUMBER() OVER(PARTITION BY tenant_id ORDER BY random()) AS rn
  FROM (
    SELECT d.tenant_id, d.id AS deal_id, d.owner_user_id AS author,
      (ARRAY[
        'Khách hàng rất quan tâm, đã xác nhận nhu cầu mua.',
        'Đã gửi báo giá chi tiết qua email. Chờ phản hồi.',
        'Cần theo dõi sát sao trong tuần này.',
        'Đã ký hợp đồng thành công. Chuyển sang onboarding.',
        'Khách hàng từ chối, lý do: chưa đủ điều kiện tài chính.',
        'Hẹn gặp lại vào thứ 6 tuần sau.',
        'Đang chờ phê duyệt từ bộ phận thẩm định.',
        'Cần bổ sung giấy tờ chứng minh thu nhập.',
        'Deal đang ở giai đoạn đàm phán cuối cùng.',
        'Đã chuyển hồ sơ sang bộ phận pháp lý.'
      ])[ceil(random()*10)] AS body,
      NOW() - (random()*20 || ' days')::interval AS created_at
    FROM crm.deals d
    WHERE d.tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
  ) AS sub
  WHERE rn <= 30
) AS n
ON CONFLICT (id) DO NOTHING;

-- ===== DEAL STAGE HISTORY =====
INSERT INTO deal_stage_history (id, deal_id, from_stage, to_stage, changed_by, changed_at, notes)
SELECT gen_random_uuid(), d.id,
  (ARRAY['new','contacted','qualified','proposal'])[ceil(random()*4)],
  d.stage,
  d.owner_user_id,
  d.created_at + (random()*10 || ' days')::interval,
  'Auto stage update during demo data creation'
FROM crm.deals d
WHERE d.stage NOT IN ('new','contacted')
LIMIT 50
ON CONFLICT (id) DO NOTHING;
