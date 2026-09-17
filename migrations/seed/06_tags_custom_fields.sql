-- ============================================================
-- TAGS + CUSTOM FIELDS DEFINITIONS SEED (Loop 202)
-- ============================================================

-- ===== TAGS (10 per tenant) =====
INSERT INTO tags (id, tenant_id, name, color, description, usage_count, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  tag.k,
  tag.color,
  tag.v,
  (random()*200)::int,
  NOW() - (random()*60 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('VIP','#DC2626','Khách hàng quan trọng, doanh thu cao'),
  ('Hot','#F59E0B','Lead tiềm năng cao, score > 80'),
  ('Cold','#3B82F6','Lead chưa quan tâm'),
  ('Follow-up','#8B5CF6','Cần follow up trong tuần này'),
  ('Blocked','#6B7280','Lead bị chặn, không liên hệ'),
  ('Repeat Customer','#10B981','Khách hàng quay lại'),
  ('Influencer','#EC4899','Người có tầm ảnh hưởng'),
  ('Enterprise','#0EA5E9','Doanh nghiệp lớn'),
  ('SMB','#84CC16','Doanh nghiệp vừa và nhỏ'),
  ('Trial','#A855F7','Đang dùng thử')
) AS tag(k, color, v)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (id) DO NOTHING;

-- ===== LEGACY CONTACT TAGS (assign 5-10 random tags to first 50 leads in legacy contacts table) =====
INSERT INTO contact_tags (contact_id, tag_id)
SELECT c.id, t.id
FROM (
  SELECT c.id, ROW_NUMBER() OVER (PARTITION BY c.tenant_id ORDER BY c.id) AS rn
  FROM contacts c
  WHERE c.tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002')
    AND c.deleted_at IS NULL
) c
CROSS JOIN LATERAL (
  SELECT id FROM tags
   WHERE tenant_id = c.tenant_id
   ORDER BY random() LIMIT 3
) t
WHERE c.rn <= 30
ON CONFLICT DO NOTHING;

-- ===== CUSTOM FIELDS DEFINITIONS =====
INSERT INTO custom_fields (id, tenant_id, entity_type, name, field_key, field_type, description, options, default_value, validation_rules, is_required, is_unique, display_order, width, is_active, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  cf.entity_type,
  cf.name,
  cf.field_key,
  cf.field_type,
  cf.description,
  cf.options::jsonb,
  cf.default_value::jsonb,
  cf.validation_rules::jsonb,
  cf.is_required::boolean,
  cf.is_unique::boolean,
  cf.display_order::int,
  cf.width::text,
  true,
  NOW() - (random()*40 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('lead','Lead Source','source_detail','select','Chi tiết nguồn lead (Facebook campaign, Google ad group, etc.)',
    '["Facebook - Spring Loan","Facebook - Personal Loan","Google - Brand","Google - Generic","TikTok - Cash Loan","Zalo OA","Walk-in","Referral"]',
    '"Zalo OA"',
    '{}',false,false,1,'full'),
  ('lead','Interest Level','interest_level','select','Mức độ quan tâm',
    '["very_high","high","medium","low"]',
    '"medium"',
    '{}',true,false,2,'half'),
  ('lead','Budget Range','budget_range','number','Ngân sách dự kiến (VND)',
    '[]',
    '50000000',
    '{"min":0,"max":10000000000}',
    false,false,3,'half'),
  ('lead','Preferred Location','preferred_location','select','Khu vực ưa thích',
    '["Ho Chi Minh","Ha Noi","Da Nang","Can Tho","Other"]',
    '"Ho Chi Minh"',
    '{}',false,false,4,'full'),
  ('lead','Decision Maker','decision_maker','boolean','Có phải người quyết định?',
    '[]','true','{}',false,false,5,'half'),
  ('contact','Last Meeting Date','last_meeting_date','date','Ngày gặp gỡ gần nhất',
    '[]','null','{}',false,false,1,'half'),
  ('contact','Preferred Channel','preferred_channel','select','Kênh liên lạc ưa thích',
    '["phone","email","zalo","telegram","whatsapp"]','"phone"','{}',false,false,2,'full'),
  ('deal','Lost Reason','lost_reason','select','Lý do mất deal',
    '["price","competitor","no_budget","no_need","timing","other"]',
    'null','{}',false,false,1,'full'),
  ('deal','Commission %','commission_pct','number','% hoa hồng',
    '[]','5',
    '{"min":0,"max":50}',
    false,false,2,'half'),
  ('activity','Outcome','outcome','select','Kết quả cuộc gọi/meeting',
    '["interested","not_interested","callback","no_answer","wrong_number","do_not_call"]',
    'null','{}',false,false,1,'full')
) AS cf(entity_type, name, field_key, field_type, description, options, default_value, validation_rules, is_required, is_unique, display_order, width)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (id) DO NOTHING;

-- ===== DYNAMIC FIELD DEFS (workflow.entity_field_defs) =====
INSERT INTO workflow.entity_field_defs (id, tenant_id, entity_type, name, label, field_type, is_required, is_searchable, options, default_value, validation_rules, display_order, is_system, is_active, created_at)
SELECT
  gen_random_uuid(),
  t.id,
  ef.entity_type,
  ef.name,
  ef.label,
  ef.field_type,
  ef.is_required::boolean,
  ef.is_searchable::boolean,
  '[]'::jsonb,
  'null'::jsonb,
  '{}'::jsonb,
  ef.display_order::int,
  false,
  true,
  NOW() - (random()*30 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('real_estate','title','Property Title','string',true,true,1),
  ('real_estate','location','Location','string',true,true,2),
  ('real_estate','price','Price (VND)','float',true,false,3),
  ('real_estate','area_sqm','Area (m2)','int',false,false,4),
  ('real_estate','bedrooms','Bedrooms','int',false,false,5),
  ('real_estate','property_type','Type','enum',false,false,6),
  ('real_estate','owner_name','Owner Name','string',false,true,7),
  ('real_estate','owner_phone','Owner Phone','string',false,true,8),
  ('real_estate','description','Description','text',false,false,9),
  ('real_estate','published_at','Published At','datetime',false,false,10),
  ('contact','email','Email','string',true,true,1),
  ('contact','phone','Phone','string',true,true,2),
  ('contact','company','Company Name','string',false,true,3),
  ('contact','job_title','Job Title','string',false,false,4),
  ('deal','value','Deal Value','float',true,false,1),
  ('deal','probability','Probability %','int',false,false,2),
  ('deal','expected_close_date','Expected Close','date',false,false,3)
) AS ef(entity_type, name, label, field_type, is_required, is_searchable, display_order)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (id) DO NOTHING;
