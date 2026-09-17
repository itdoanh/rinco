-- ============================================================
-- WS-B LOOP 1: DEALS + ACTIVITIES EXPANSION
-- 50+ deals/tenant, 300+ activities/tenant, additional notes
-- ============================================================

-- ============================================================
-- APEX FINTECH: 60 EXTRA DEALS (from new leads)
-- ============================================================
INSERT INTO crm.deals (
  id, tenant_id, lead_id, owner_user_id, pipeline_id, stage, title, value, currency,
  probability, expected_close_date, status, data, created_at, updated_at
)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000001',
  lead.id,
  lead.owner_user_id,
  'c1000001-0000-0000-0000-000000000001',
  CASE lead.stage
    WHEN 'qualified' THEN 'proposal'
    WHEN 'proposal' THEN 'underwriting'
    WHEN 'won' THEN 'won'
    WHEN 'lost' THEN 'lost'
    WHEN 'contacted' THEN 'contacted'
    ELSE 'contacted'
  END,
  'Khoản vay ' || lead.full_name || ' - ' || (lead.custom_fields->>'loan_purpose'),
  CASE (lead.custom_fields->>'budget_range')
    WHEN '5-10tr' THEN (5 + random()*5)::numeric(18,2)*1000000
    WHEN '10-30tr' THEN (10 + random()*20)::numeric(18,2)*1000000
    WHEN '30-100tr' THEN (30 + random()*70)::numeric(18,2)*1000000
    ELSE (100 + random()*400)::numeric(18,2)*1000000
  END,
  'VND',
  CASE lead.stage WHEN 'qualified' THEN 65 WHEN 'proposal' THEN 80 WHEN 'won' THEN 100 ELSE 25 END,
  CURRENT_DATE + ((random()*90)::int),
  CASE WHEN lead.stage = 'won' THEN 'WON' WHEN lead.stage = 'lost' THEN 'LOST' ELSE 'OPEN' END,
  jsonb_build_object(
    'source',lead.source,
    'utm_campaign',lead.utm_campaign,
    'city',lead.data->>'city',
    'loan_product',(ARRAY['personal_loan','home_loan','auto_loan','business_loan','credit_card'])[ceil(random()*5)],
    'term_months',(ARRAY[6,12,24,36,48,60])[ceil(random()*6)],
    'interest_rate',(0.08 + random()*0.12)::numeric(4,3)
  ),
  lead.created_at + ((random()*5)::int || ' days')::interval,
  NOW()
FROM leads.leads lead
WHERE lead.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
  AND lead.stage IN ('contacted','qualified','proposal','won')
  AND NOT EXISTS (
    SELECT 1 FROM crm.deals d WHERE d.lead_id = lead.id
  )
ORDER BY random()
LIMIT 60
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- HCT CONSULTING: 60 EXTRA DEALS
-- ============================================================
INSERT INTO crm.deals (
  id, tenant_id, lead_id, owner_user_id, pipeline_id, stage, title, value, currency,
  probability, expected_close_date, status, data, created_at, updated_at
)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000002',
  lead.id,
  lead.owner_user_id,
  'c1000002-0000-0000-0000-000000000001',
  CASE lead.stage
    WHEN 'qualified' THEN 'viewing'
    WHEN 'proposal' THEN 'negotiation'
    WHEN 'won' THEN 'won'
    WHEN 'lost' THEN 'lost'
    WHEN 'contacted' THEN 'contacted'
    ELSE 'contacted'
  END,
  'BĐS ' || COALESCE(lead.custom_fields->>'property_type','can_ho') || ' - ' || lead.full_name,
  CASE (lead.custom_fields->>'budget_range')
    WHEN '2-5 ty' THEN (2 + random()*3)::numeric(18,2)*1000000000
    WHEN '5-10 ty' THEN (5 + random()*5)::numeric(18,2)*1000000000
    WHEN '10-20 ty' THEN (10 + random()*10)::numeric(18,2)*1000000000
    WHEN '20-50 ty' THEN (20 + random()*30)::numeric(18,2)*1000000000
    ELSE (50 + random()*50)::numeric(18,2)*1000000000
  END,
  'VND',
  CASE lead.stage WHEN 'qualified' THEN 40 WHEN 'proposal' THEN 75 WHEN 'won' THEN 100 ELSE 20 END,
  CURRENT_DATE + ((random()*120)::int),
  CASE WHEN lead.stage = 'won' THEN 'WON' WHEN lead.stage = 'lost' THEN 'LOST' ELSE 'OPEN' END,
  jsonb_build_object(
    'property_type',lead.custom_fields->>'property_type',
    'location',lead.custom_fields->>'preferred_location',
    'investment',lead.custom_fields->>'investment_horizon',
    'project',(ARRAY['Vinhomes Grand Park','Masterise Eco','Sun Group','Vinhomes Ocean Park','Imperia Smart City','The Matrix One'])[ceil(random()*6)],
    'unit_code','SH-' || lpad((random()*999)::int::text, 3, '0'),
    'area_sqm',(50 + random()*250)::int
  ),
  lead.created_at + ((random()*7)::int || ' days')::interval,
  NOW()
FROM leads.leads lead
WHERE lead.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000002'
  AND lead.stage IN ('contacted','qualified','proposal','won')
  AND NOT EXISTS (
    SELECT 1 FROM crm.deals d WHERE d.lead_id = lead.id
  )
ORDER BY random()
LIMIT 60
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DEMO COMPANY: 30 EXTRA DEALS
-- ============================================================
INSERT INTO crm.deals (
  id, tenant_id, lead_id, owner_user_id, pipeline_id, stage, title, value, currency,
  probability, expected_close_date, status, data, created_at, updated_at
)
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
  CURRENT_DATE + ((random()*60)::int),
  CASE WHEN lead.stage = 'won' THEN 'WON' WHEN lead.stage = 'lost' THEN 'LOST' ELSE 'OPEN' END,
  jsonb_build_object('demo_tier',(ARRAY['free','pro','enterprise'])[ceil(random()*3)]),
  lead.created_at + ((random()*3)::int || ' days')::interval,
  NOW()
FROM leads.leads lead
WHERE lead.tenant_id = 'bbbbbbbb-0000-0000-0000-000000000003'
  AND lead.stage IN ('contacted','qualified','proposal','won')
  AND NOT EXISTS (
    SELECT 1 FROM crm.deals d WHERE d.lead_id = lead.id
  )
ORDER BY random()
LIMIT 30
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- APEX FINTECH: 300 EXTRA ACTIVITIES (more variety)
-- ============================================================
INSERT INTO crm.activities (
  id, tenant_id, user_id, lead_id, deal_id, type, title, description,
  due_at, completed_at, status, priority, data, created_at
)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000001',
  act.owner,
  act.lead_id,
  act.deal_id,
  act.type,
  act.title,
  act.desc,
  act.due_at,
  CASE WHEN act.type IN ('call','email','meeting','note') THEN act.due_at ELSE NULL END,
  CASE WHEN act.type IN ('call','email','meeting','note') THEN 'completed' ELSE 'pending' END,
  (ARRAY['low','medium','high','urgent'])[ceil(random()*4)],
  jsonb_build_object(
    'duration_minutes',(15+random()*105)::int,
    'result',act.result,
    'channel',(ARRAY['phone','zalo','email','in_person','video_call'])[ceil(random()*5)],
    'outcome',(ARRAY['interested','not_interested','callback','no_answer','follow_up'])[ceil(random()*5)]
  ),
  act.due_at - INTERVAL '2 hours'
FROM (
  SELECT l.owner_user_id AS owner, l.id AS lead_id,
    d.id AS deal_id,
    (ARRAY['call','call','call','email','email','email','meeting','meeting','note','note','task','task','whatsapp','sms','zalo'])[ceil(random()*15)] AS type,
    (ARRAY[
      'Gọi điện giới thiệu sản phẩm mới',
      'Follow-up sau email đầu tiên',
      'Gặp mặt trực tiếp tư vấn khoản vay',
      'Gửi báo giá chi tiết',
      'Gọi xác nhận nhu cầu vay',
      'Gửi hồ sơ vay mẫu',
      'Họp team review deal',
      'Gửi email cảm ơn sau meeting',
      'Gọi check-in định kỳ',
      'Gửi tài liệu pháp lý dự án',
      'Zalo tư vấn nhanh',
      'Demo app cho khách hàng',
      'Gọi xác nhận giấy tờ',
      'Gặp tại ngân hàng',
      'Gửi hợp đồng qua email'
    ])[ceil(random()*15)] AS title,
    (ARRAY[
      'Khách hàng quan tâm, hẹn gặp tuần sau',
      'Đã gửi email, chờ phản hồi trong 48h',
      'Gặp gỡ tại văn phòng, khách hàng rất hào hứng',
      'Đang so sánh với đối thủ, cần thêm thông tin',
      'Đủ điều kiện vay, tiến hành thủ tục',
      'Khách hàng cần thêm thời gian suy nghĩ',
      'Đã xác nhận deal, chờ ký hợp đồng',
      'Cần follow-up lại trong 1 tuần',
      'Đã gửi báo giá, chờ phản hồi',
      'Khách hàng yêu cầu gặp lại sau',
      'Đã nhận hồ sơ đầy đủ',
      'Đang chờ phê duyệt từ ngân hàng',
      'Deal đang đàm phán giá',
      'Khách hàng muốn xem căn mẫu',
      'Đã ký hợp đồng thành công'
    ])[ceil(random()*15)] AS desc,
    (ARRAY['Tích cực','Trung lập','Tiêu cực','Hẹn lại','Đã chốt'])[ceil(random()*5)] AS result,
    NOW() - (random()*60 || ' days')::interval AS due_at
  FROM leads.leads l
  LEFT JOIN crm.deals d ON d.lead_id = l.id AND d.tenant_id = l.tenant_id
  WHERE l.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
  ORDER BY random()
  LIMIT 300
) AS act
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- HCT CONSULTING: 250 EXTRA ACTIVITIES (real estate)
-- ============================================================
INSERT INTO crm.activities (
  id, tenant_id, user_id, lead_id, deal_id, type, title, description,
  due_at, completed_at, status, priority, created_at
)
SELECT
  gen_random_uuid(),
  'aaaaaaaa-0000-0000-0000-000000000002',
  act.owner,
  act.lead_id,
  act.deal_id,
  act.type,
  act.title,
  act.desc,
  act.due_at,
  CASE WHEN act.type IN ('call','email','meeting','note') THEN act.due_at ELSE NULL END,
  CASE WHEN act.type IN ('call','email','meeting','note') THEN 'completed' ELSE 'pending' END,
  (ARRAY['low','medium','high'])[ceil(random()*3)],
  act.due_at - INTERVAL '1 hour'
FROM (
  SELECT l.owner_user_id AS owner, l.id AS lead_id,
    d.id AS deal_id,
    (ARRAY['call','call','meeting','meeting','meeting','note','note','task','viewing','email','zalo','whatsapp'])[ceil(random()*12)] AS type,
    (ARRAY[
      'Gọi giới thiệu dự án Vinhomes Grand Park',
      'Hẹn đi xem căn hộ mẫu',
      'Gửi bảng giá shophouse',
      'Tư vấn pháp lý đất nền',
      'Gọi follow-up sau buổi xem',
      'Họp phân tích deal đầu tư',
      'Gửi hồ sơ tín dụng',
      'Đàm phán giá cuối cùng',
      'Gặp tại dự án cuối tuần',
      'Gửi tài liệu pháp lý chi tiết',
      'Xem căn 2PN tầng 12',
      'Thương thảo điều khoản thanh toán'
    ])[ceil(random()*12)] AS title,
    (ARRAY[
      'Khách rất quan tâm, muốn mua 2 căn',
      'Đã đi xem, đang so sánh với đối thủ',
      'Cần tài liệu pháp lý chi tiết hơn',
      'Chờ quyết định từ gia đình',
      'Deal tiềm năng cao, nên theo sát',
      'Khách đã có financing, chỉ cần deal',
      'Cần gặp lại tuần này',
      'Đàm phán giá thành công',
      'Khách hàng VIP, đề xuất chiết khấu thêm',
      'Cần xem thêm 2 căn nữa',
      'Chốt deal ký hợp đồng tuần sau',
      'Đã đặt cọc 100 triệu'
    ])[ceil(random()*12)] AS desc,
    NOW() - (random()*60 || ' days')::interval AS due_at
  FROM leads.leads l
  LEFT JOIN crm.deals d ON d.lead_id = l.id AND d.tenant_id = l.tenant_id
  WHERE l.tenant_id = 'aaaaaaaa-0000-0000-0000-000000000002'
  ORDER BY random()
  LIMIT 250
) AS act
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DEMO COMPANY: 80 EXTRA ACTIVITIES
-- ============================================================
INSERT INTO crm.activities (
  id, tenant_id, user_id, lead_id, deal_id, type, title, description,
  due_at, completed_at, status, created_at
)
SELECT
  gen_random_uuid(),
  'bbbbbbbb-0000-0000-0000-000000000003',
  l.owner_user_id,
  l.id,
  d.id,
  (ARRAY['call','email','meeting','note','demo','task'])[ceil(random()*6)],
  'Activity - ' || (ARRAY['intro','follow-up','demo','pricing','closing'])[ceil(random()*5)] || ' - ' || l.full_name,
  'Demo activity description: ' || (ARRAY[
    'Khách quan tâm, cần demo chi tiết',
    'Gửi trial account cho khách',
    'Demo screen share 30 phút',
    'Gửi bảng giá enterprise',
    'Cần call với CTO khách hàng'
  ])[ceil(random()*5)],
  NOW() - (random()*30 || ' days')::interval,
  NOW() - (random()*30 || ' days')::interval,
  (ARRAY['completed','pending','in_progress'])[ceil(random()*3)],
  NOW() - (random()*30 || ' days')::interval
FROM leads.leads l
LEFT JOIN crm.deals d ON d.lead_id = l.id AND d.tenant_id = l.tenant_id
WHERE l.tenant_id = 'bbbbbbbb-0000-0000-0000-000000000003'
ORDER BY random()
LIMIT 80
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- ADDITIONAL NOTES (50+ per tenant)
-- ============================================================
INSERT INTO crm.notes (
  id, tenant_id, contact_id, deal_id, body, author_id, is_pinned, created_at
)
SELECT gen_random_uuid(), n.tenant_id, NULL::uuid, d.id, n.body, n.author, random()<0.2, n.created_at
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
        'Đã chuyển hồ sơ sang bộ phận pháp lý.',
        'Khách hàng VIP, cần chăm sóc đặc biệt.',
        'Đề xuất giảm giá 5% để chốt deal.',
        'Đã đồng ý điều khoản thanh toán.',
        'Chờ khách hàng confirm lịch xem.',
        'Đã gửi hợp đồng điện tử cho khách ký.'
      ])[ceil(random()*15)] AS body,
      NOW() - (random()*40 || ' days')::interval AS created_at
    FROM crm.deals d
    WHERE d.tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
  ) AS sub
  WHERE rn <= 60
) AS n
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DEAL STAGE HISTORY (additional for tracking)
-- ============================================================
INSERT INTO deal_stage_history (id, deal_id, from_stage, to_stage, changed_by, changed_at, notes)
SELECT gen_random_uuid(), d.id,
  (ARRAY['new','contacted','qualified','proposal','viewing','negotiation'])[ceil(random()*6)],
  d.stage,
  d.owner_user_id,
  d.created_at + (random()*15 || ' days')::interval,
  'Stage progression tracked for demo data expansion'
FROM crm.deals d
WHERE d.created_at > NOW() - INTERVAL '60 days'
LIMIT 200
ON CONFLICT (id) DO NOTHING;

-- Stats
SELECT 'expansion_02_deals_activities (WS-B Loop 1)' AS section,
       tenant_id, COUNT(*) AS total_activities
FROM crm.activities
WHERE tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
GROUP BY tenant_id
ORDER BY tenant_id;
