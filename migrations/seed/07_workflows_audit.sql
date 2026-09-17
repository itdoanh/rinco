-- ============================================================
-- WORKFLOWS + AUDIT + API REQUESTS SEED (Loop 202)
-- ============================================================

-- ===== WORKFLOW ENTITY DEFINITIONS =====
INSERT INTO workflow.entity_definitions (id, tenant_id, entity_code, display_name, icon, description, fields, workflows, indexes, version, is_active, created_by, created_at)
VALUES
  ('ee110001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','real_estate_apex','Bất động sản','home','Real estate listings for Apex Fintech investment clients',
    '[{"name":"title","type":"string"},{"name":"price","type":"float"},{"name":"location","type":"string"},{"name":"bedrooms","type":"int"}]'::jsonb,
    '[]'::jsonb,
    '[{"name":"idx_re_price","fields":["price"]},{"name":"idx_re_loc","fields":["location"]}]'::jsonb,
    1,true,'a0000001-0000-0000-0000-000000000002',NOW() - INTERVAL '60 days'),
  ('ee110002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bds_property','BĐS Cao Cấp','building','High-end real estate for HCT clients',
    '[{"name":"title","type":"string"},{"name":"price","type":"float"},{"name":"area_sqm","type":"int"},{"name":"location","type":"string"}]'::jsonb,
    '[]'::jsonb,
    '[{"name":"idx_bds_price","fields":["price"]}]'::jsonb,
    1,true,'a0000002-0000-0000-0000-000000000001',NOW() - INTERVAL '40 days'),
  ('ee110003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','demo_item','Demo Items','box','Demo entity for showcase',
    '[{"name":"name","type":"string"}]'::jsonb,
    '[]'::jsonb,
    '[]'::jsonb,
    1,true,'a0000003-0000-0000-0000-000000000001',NOW() - INTERVAL '5 days')
ON CONFLICT (tenant_id, entity_code) DO NOTHING;

-- ===== WORKFLOWS (5 per tenant-ish) =====
INSERT INTO workflow.workflows (id, tenant_id, name, description, trigger_entity, trigger_event, trigger_config, conditions, actions, is_active, created_at)
VALUES
  ('ee200001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001',
    'Auto-assign lead theo region',
    'Tự động assign lead về agent tương ứng với khu vực: HN, HCM, DN',
    'lead','create',
    '{"field":"data->>''city''"}'::jsonb,
    '[{"type":"always"}]'::jsonb,
    '[{"type":"assign","rules":[{"if":"data.city==\"HN\"","to":"a0000001-0000-0000-0000-000000000011"},{"if":"data.city==\"HCM\"","to":"a0000001-0000-0000-0000-000000000010"},{"if":"data.city==\"DN\"","to":"a0000001-0000-0000-0000-000000000012"}]}]'::jsonb,
    true,NOW() - INTERVAL '60 days'),

  ('ee200001-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000001',
    'Auto-tag theo UTM source',
    'Tag lead theo UTM source để dễ tracking',
    'lead','create',
    '{}'::jsonb,
    '[]'::jsonb,
    '[{"type":"add_tag","rules":[{"if":"utm_source==\"facebook\"","tag":"facebook"},{"if":"utm_source==\"google\"","tag":"google"},{"if":"utm_source==\"tiktok\"","tag":"tiktok"}]}]'::jsonb,
    true,NOW() - INTERVAL '55 days'),

  ('ee200001-0000-0000-0000-000000000003','aaaaaaaa-0000-0000-0000-000000000001',
    'Alert khi score > 80',
    'Gửi notification cho manager khi có lead score cao',
    'lead','update',
    '{"field":"score"}'::jsonb,
    '[{"type":"gt","field":"score","value":80}]'::jsonb,
    '[{"type":"notify","channel":"in_app","template":"hot_lead_alert","recipients":["a0000001-0000-0000-0000-000000000002"]}]'::jsonb,
    true,NOW() - INTERVAL '40 days'),

  ('ee200002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002',
    'Notify team khi có viewing mới',
    'Gửi in-app notification khi lead đặt lịch xem BĐS',
    'lead','update',
    '{"field":"stage"}'::jsonb,
    '[{"type":"eq","field":"stage","value":"viewing"}]'::jsonb,
    '[{"type":"notify","channel":"in_app","template":"viewing_scheduled"}]'::jsonb,
    true,NOW() - INTERVAL '30 days'),

  ('ee200002-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000002',
    'Auto-assign BDS consultant theo location',
    'Assign consultant BĐS chuyên khu vực',
    'lead','create',
    '{"field":"custom_fields->>''preferred_location''"}'::jsonb,
    '[{"type":"always"}]'::jsonb,
    '[{"type":"assign","rules":[{"if":"preferred_location in [\"Ba Dinh\",\"Hoan Kiem\",\"Hai Ba Trung\"]","to":"a0000002-0000-0000-0000-000000000010"}]}]'::jsonb,
    true,NOW() - INTERVAL '25 days'),

  ('ee200003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003',
    'Demo: Auto welcome email',
    'Gửi email chào mừng khi có lead mới',
    'lead','create',
    '{}'::jsonb,
    '[]'::jsonb,
    '[{"type":"send_email","template":"welcome_demo"}]'::jsonb,
    true,NOW() - INTERVAL '3 days')
ON CONFLICT (id) DO NOTHING;

-- ===== WORKFLOW EXECUTIONS (historical log) =====
INSERT INTO workflow.workflow_executions (id, workflow_id, tenant_id, trigger_data, status, started_at, completed_at, execution_log)
SELECT
  gen_random_uuid(),
  w.id,
  w.tenant_id,
  jsonb_build_object('lead_id', gen_random_uuid(), 'event','create', 'ts', NOW()),
  (ARRAY['completed','completed','completed','failed'])[ceil(random()*4)],
  NOW() - (random()*20 || ' days')::interval,
  NOW() - (random()*20 || ' days')::interval + INTERVAL '2 seconds',
  jsonb_build_array(
    jsonb_build_object('step','validation','ok',true),
    jsonb_build_object('step','action','name', w.name,'ok',true)
  )
FROM workflow.workflows w
CROSS JOIN generate_series(1,15)
ON CONFLICT (id) DO NOTHING;

-- ===== AUDIT LOGS (50+ entries) =====
INSERT INTO audit.audit_logs (tenant_id, actor_user_id, actor_ip, action, entity_type, entity_id, before, after, request_id, trace_id, user_agent, extra, created_at)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000001','203.0.113.10','create_lead','lead',gen_random_uuid(),NULL,jsonb_build_object('source','Facebook Ads','score',85),'req-001','trace-001','Mozilla/5.0 Chrome','{"test":true}',NOW() - INTERVAL '1 day'),
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000001','203.0.113.10','create_deal','deal',gen_random_uuid(),NULL,jsonb_build_object('value',500000000),'req-002','trace-002','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '1 day'),
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000002','203.0.113.11','update_lead','lead',gen_random_uuid(),jsonb_build_object('stage','new'),jsonb_build_object('stage','contacted'),'req-003','trace-003','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '2 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000001','203.0.113.10','login_success','session',gen_random_uuid(),NULL,jsonb_build_object('ip','203.0.113.10'),'req-004','trace-004','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '2 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000001','203.0.113.10','create_user','user','a0000001-0000-0000-0000-000000000015',NULL,jsonb_build_object('role','agent'),'req-005','trace-005','Postman','{}',NOW() - INTERVAL '15 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000003','203.0.113.12','workflow_executed','workflow','ee200001-0000-0000-0000-000000000003',NULL,jsonb_build_object('status','completed'),'req-006','trace-006','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '3 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001',NULL,'203.0.113.13','failed_login','session',gen_random_uuid(),NULL,NULL,'req-007','trace-007','curl/7.81','{"failure_reason":"bad_password"}',NOW() - INTERVAL '4 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000004','203.0.113.14','campaign_launched','campaign',gen_random_uuid(),NULL,jsonb_build_object('platform','facebook','budget_vnd',50000000),'req-008','trace-008','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '7 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000002','203.0.113.11','meeting_scheduled','activity',gen_random_uuid(),NULL,jsonb_build_object('attendees',3),'req-009','trace-009','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '5 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000001','203.0.113.10','settings_updated','tenant_settings','aaaaaaaa-0000-0000-0000-000000000001',jsonb_build_object('plan','pro'),jsonb_build_object('plan','business'),'req-010','trace-010','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '20 days'),
  ('aaaaaaaa-0000-0000-0000-000000000002','a0000002-0000-0000-0000-000000000001','203.0.113.20','create_lead','lead',gen_random_uuid(),NULL,jsonb_build_object('source','Facebook','property_type','can_ho'),'req-011','trace-011','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '10 days'),
  ('aaaaaaaa-0000-0000-0000-000000000002','a0000002-0000-0000-0000-000000000002','203.0.113.21','viewing_scheduled','activity',gen_random_uuid(),NULL,jsonb_build_object('property','Vinhomes Grand Park'),'req-012','trace-012','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '8 days'),
  ('aaaaaaaa-0000-0000-0000-000000000002','a0000002-0000-0000-0000-000000000003','203.0.113.22','deal_won','deal',gen_random_uuid(),NULL,jsonb_build_object('value',5000000000),'req-013','trace-013','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '5 days'),
  ('aaaaaaaa-0000-0000-0000-000000000002','a0000002-0000-0000-0000-000000000001','203.0.113.20','login_success','session',gen_random_uuid(),NULL,NULL,'req-014','trace-014','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '1 day'),
  ('aaaaaaaa-0000-0000-0000-000000000002','a0000002-0000-0000-0000-000000000004','203.0.113.23','report_generated','report',gen_random_uuid(),NULL,jsonb_build_object('type','monthly_sales'),'req-015','trace-015','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '15 days'),
  ('bbbbbbbb-0000-0000-0000-000000000003','a0000003-0000-0000-0000-000000000001','203.0.113.30','create_lead','lead',gen_random_uuid(),NULL,jsonb_build_object('source','demo'),'req-016','trace-016','Mozilla/5.0 Chrome','{}',NOW() - INTERVAL '5 days'),
  ('bbbbbbbb-0000-0000-0000-000000000003','a0000003-0000-0000-0000-000000000001','203.0.113.30','tenant_created','tenant','bbbbbbbb-0000-0000-0000-000000000003',NULL,NULL,'req-017','trace-017','Mozilla/5.0 Chrome','{"plan":"enterprise"}',NOW() - INTERVAL '9 days'),
  ('bbbbbbbb-0000-0000-0000-000000000003','a0000003-0000-0000-0000-000000000002','203.0.113.31','bulk_import','lead','bulk',NULL,jsonb_build_object('count',50),'req-018','trace-018','postman','{}',NOW() - INTERVAL '4 days')
;

-- ===== LOGIN HISTORY (30 entries) =====
INSERT INTO audit.login_history (tenant_id, user_id, email, success, failure_reason, ip_address, user_agent, mfa_used, created_at)
SELECT
  'aaaaaaaa-0000-0000-0000-000000000001',
  u.id,
  u.email,
  random()<0.9,
  CASE WHEN random()<0.1 THEN 'invalid_password' ELSE NULL END,
  ('203.0.113.' || (10 + (random()*200)::int)::text)::inet,
  (ARRAY['Mozilla/5.0 Chrome','Mozilla/5.0 Safari','Mozilla/5.0 Firefox','PostmanRuntime/7.32','curl/7.81'])[ceil(random()*5)],
  random()<0.2,
  NOW() - (random()*30 || ' days')::interval
FROM auth.users u
WHERE u.tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002')
LIMIT 30;

-- ===== API REQUESTS (200 entries) =====
INSERT INTO audit.api_requests (tenant_id, user_id, service, method, route, status, duration_ms, bytes_in, bytes_out, ip_address, user_agent, trace_id, created_at)
SELECT
  t.id,
  u.id,
  svc.name,
  svc.method,
  svc.route,
  (ARRAY[200,200,200,200,201,204,400,401,500])[ceil(random()*9)],
  (5 + random()*995)::int,
  (random()*1024)::int,
  (random()*20480)::int,
  ('203.0.113.' || (10 + (random()*200)::int)::text)::inet,
  'Mozilla/5.0 Chrome',
  'trace-' || md5(random()::text),
  NOW() - (random()*30 || ' days')::interval
FROM tenant.tenants t
JOIN auth.users u ON u.tenant_id = t.id
CROSS JOIN (VALUES
  ('crm-service','GET','/api/v1/leads'),
  ('crm-service','POST','/api/v1/leads'),
  ('crm-service','GET','/api/v1/deals'),
  ('crm-service','PATCH','/api/v1/leads/{id}'),
  ('lead-service','POST','/api/v1/scoring/batch'),
  ('notification-service','POST','/api/v1/notifications'),
  ('auth-service','POST','/api/v1/auth/login'),
  ('auth-service','POST','/api/v1/auth/refresh'),
  ('tenant-service','GET','/api/v1/tenant/quota'),
  ('landing-service','POST','/api/v1/track/event')
) AS svc(name, method, route)
CROSS JOIN generate_series(1,20)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003');
