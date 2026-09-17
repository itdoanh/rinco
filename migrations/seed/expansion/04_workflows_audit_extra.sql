-- ============================================================
-- WS-B LOOP 3: WORKFLOWS + AUDIT EXPANSION
-- 10+ workflows + audit entries per tenant
-- ============================================================

-- ============================================================
-- APEX FINTECH: 6 EXTRA WORKFLOWS
-- ============================================================
INSERT INTO workflow.workflows (id, tenant_id, name, description, trigger_entity, trigger_event, trigger_config, conditions, actions, is_active, created_at)
VALUES
  ('ee200001-0000-0000-0000-000000000010','aaaaaaaa-0000-0000-0000-000000000001',
    'Auto-send welcome email cho lead mới',
    'Gửi email welcome ngay khi lead mới được tạo',
    'lead','create',
    '{}'::jsonb,
    '[{"type":"always"}]'::jsonb,
    '[{"type":"send_email","template":"welcome_loan","recipients":["owner"]},{"type":"add_tag","tag":"new_lead"}]'::jsonb,
    true,NOW() - INTERVAL '30 days'),

  ('ee200001-0000-0000-0000-000000000011','aaaaaaaa-0000-0000-0000-000000000001',
    'Auto-nurture email sequence cho qualified',
    'Gửi email nurturing 3 ngày sau khi qualified',
    'lead','update',
    '{"field":"stage","delay_days":3}'::jsonb,
    '[{"type":"eq","field":"stage","value":"qualified"}]'::jsonb,
    '[{"type":"send_email","template":"nurture_qualified"}]'::jsonb,
    true,NOW() - INTERVAL '25 days'),

  ('ee200001-0000-0000-0000-000000000012','aaaaaaaa-0000-0000-0000-000000000001',
    'Slack notification khi deal lớn',
    'Gửi Slack alert khi deal > 500 triệu',
    'deal','create',
    '{"field":"value"}'::jsonb,
    '[{"type":"gt","field":"value","value":500000000}]'::jsonb,
    '[{"type":"slack_notify","channel":"#sales-big-deals","recipients":["manager"]}]'::jsonb,
    true,NOW() - INTERVAL '20 days'),

  ('ee200001-0000-0000-0000-000000000013','aaaaaaaa-0000-0000-0000-000000000001',
    'Auto-assign theo loan amount',
    'Phân deal theo loan amount: <100tr junior, 100-500tr senior, >500tr manager',
    'deal','create',
    '{"field":"value"}'::jsonb,
    '[{"type":"always"}]'::jsonb,
    '[{"type":"assign","rules":[{"if":"value<100000000","to":"a0000001-0000-0000-0000-000000000014"},{"if":"value<500000000","to":"b0000001-0000-0000-0000-000000000010"},{"if":"value>=500000000","to":"a0000001-0000-0000-0000-000000000002"}]}]'::jsonb,
    true,NOW() - INTERVAL '15 days'),

  ('ee200001-0000-0000-0000-000000000014','aaaaaaaa-0000-0000-0000-000000000001',
    'Stale lead warning',
    'Cảnh báo lead > 7 ngày không contact',
    'lead','update',
    '{"field":"last_contacted_at","interval_days":7}'::jsonb,
    '[{"type":"gt","field":"days_since_contact","value":7}]'::jsonb,
    '[{"type":"notify","channel":"in_app","template":"stale_lead_warning"},{"type":"add_tag","tag":"stale"}]'::jsonb,
    true,NOW() - INTERVAL '10 days'),

  ('ee200001-0000-0000-0000-000000000015','aaaaaaaa-0000-0000-0000-000000000001',
    'Auto-sync Meta CAPI cho won deals',
    'Gửi Meta Conversions API khi deal won',
    'deal','update',
    '{"field":"status"}'::jsonb,
    '[{"type":"eq","field":"status","value":"WON"}]'::jsonb,
    '[{"type":"meta_capi","event_name":"Purchase","value_field":"value"}]'::jsonb,
    true,NOW() - INTERVAL '5 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- HCT CONSULTING: 5 EXTRA WORKFLOWS
-- ============================================================
INSERT INTO workflow.workflows (id, tenant_id, name, description, trigger_entity, trigger_event, trigger_config, conditions, actions, is_active, created_at)
VALUES
  ('ee200002-0000-0000-0000-000000000010','aaaaaaaa-0000-0000-0000-000000000002',
    'Auto-schedule viewing khi qualified',
    'Tự động tạo activity "viewing" khi lead qualified',
    'lead','update',
    '{"field":"stage"}'::jsonb,
    '[{"type":"eq","field":"stage","value":"qualified"}]'::jsonb,
    '[{"type":"create_activity","activity_type":"viewing","due_days":3,"priority":"high"}]'::jsonb,
    true,NOW() - INTERVAL '30 days'),

  ('ee200002-0000-0000-0000-000000000011','aaaaaaaa-0000-0000-0000-000000000002',
    'Investor notification cho high-value deal',
    'Alert board khi deal > 10 tỷ',
    'deal','create',
    '{"field":"value"}'::jsonb,
    '[{"type":"gt","field":"value","value":10000000000}]'::jsonb,
    '[{"type":"notify","channel":"email","template":"vip_deal_alert","recipients":["admin","manager"]},{"type":"notify","channel":"sms","template":"vip_sms"}]'::jsonb,
    true,NOW() - INTERVAL '20 days'),

  ('ee200002-0000-0000-0000-000000000012','aaaaaaaa-0000-0000-0000-000000000002',
    'Auto-follow-up sau viewing',
    'Schedule call 2 ngày sau khi viewing hoàn thành',
    'activity','update',
    '{"field":"status","related_to":"viewing"}'::jsonb,
    '[{"type":"eq","field":"status","value":"completed"},{"type":"eq","field":"type","value":"viewing"}]'::jsonb,
    '[{"type":"create_activity","activity_type":"call","due_days":2,"priority":"high","assign_to":"owner"}]'::jsonb,
    true,NOW() - INTERVAL '15 days'),

  ('ee200002-0000-0000-0000-000000000013','aaaaaaaa-0000-0000-0000-000000000002',
    'Document collection workflow',
    'Tự động request docs khi vào negotiation',
    'deal','update',
    '{"field":"stage"}'::jsonb,
    '[{"type":"eq","field":"stage","value":"negotiation"}]'::jsonb,
    '[{"type":"send_email","template":"doc_request"},{"type":"create_task","task":"collect_documents","due_days":7}]'::jsonb,
    true,NOW() - INTERVAL '10 days'),

  ('ee200002-0000-0000-0000-000000000014','aaaaaaaa-0000-0000-0000-000000000002',
    'Competitor mention tag',
    'Tag deal khi competitor được nhắc đến',
    'note','create',
    '{"field":"body","contains":["vinhomes","masterise","sun_group","novaland","dat_xanh"]}'::jsonb,
    '[]'::jsonb,
    '[{"type":"add_tag","tag":"competitor_mentioned"},{"type":"notify","channel":"in_app","template":"competitor_alert"}]'::jsonb,
    true,NOW() - INTERVAL '5 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DEMO COMPANY: 3 EXTRA WORKFLOWS
-- ============================================================
INSERT INTO workflow.workflows (id, tenant_id, name, description, trigger_entity, trigger_event, trigger_config, conditions, actions, is_active, created_at)
VALUES
  ('ee200003-0000-0000-0000-000000000010','bbbbbbbb-0000-0000-0000-000000000003',
    'Demo: auto-trial activation',
    'Kích hoạt trial 14 ngày khi có lead mới',
    'lead','create',
    '{}'::jsonb,
    '[{"type":"always"}]'::jsonb,
    '[{"type":"send_email","template":"trial_activation"},{"type":"add_tag","tag":"trial"}]'::jsonb,
    true,NOW() - INTERVAL '5 days'),

  ('ee200003-0000-0000-0000-000000000011','bbbbbbbb-0000-0000-0000-000000000003',
    'Demo: conversion scoring trigger',
    'Calculate AI score khi lead qualified',
    'lead','update',
    '{"field":"stage"}'::jsonb,
    '[{"type":"eq","field":"stage","value":"qualified"}]'::jsonb,
    '[{"type":"enrichment","service":"lead_scoring","async":true}]'::jsonb,
    true,NOW() - INTERVAL '4 days'),

  ('ee200003-0000-0000-0000-000000000012','bbbbbbbb-0000-0000-0000-000000000003',
    'Demo: instant webhook',
    'Push webhook đến mock endpoint khi deal won',
    'deal','update',
    '{"field":"status"}'::jsonb,
    '[{"type":"eq","field":"status","value":"WON"}]'::jsonb,
    '[{"type":"webhook","url":"http://localhost:9999/demo/webhook","async":true}]'::jsonb,
    true,NOW() - INTERVAL '2 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- WORKFLOW ENTITY FIELD DEFS EXTRA (15+ dynamic schema fields)
-- ============================================================
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
  ('[' || ef.options_array || ']')::jsonb,
  'null'::jsonb,
  ef.validation_rules::jsonb,
  ef.display_order::int,
  false,
  true,
  NOW() - (random()*30 || ' days')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('real_estate','legal_status','Legal Status','enum',false,true,'["clean","pending","disputed"]','{}',17),
  ('real_estate','floor','Floor','int',false,false,'[]','{"min":-2,"max":60}',18),
  ('real_estate','view_direction','View Direction','enum',false,false,'["north","south","east","west","southeast","southwest"]','{}',19),
  ('real_estate','furnishing','Furnishing','enum',false,true,'["unfurnished","basic","full","luxury"]','{}',20),
  ('real_estate','handover_date','Handover Date','date',false,false,'[]','{}',21),
  ('real_estate','developer','Developer','string',false,true,'[]','{}',22),
  ('real_estate','amenities','Amenities','array',false,false,'[]','{}',23),
  ('real_estate','monthly_fee','Monthly Fee (VND)','float',false,false,'[]','{"min":0,"max":50000000}',24),
  ('real_estate','rental_yield','Rental Yield %','float',false,false,'[]','{"min":0,"max":20}',25),
  ('real_estate','investment_score','Investment Score','int',false,true,'[]','{"min":0,"max":100}',26),
  ('contact','linkedin_url','LinkedIn','url',false,true,'[]','{}',5),
  ('contact','facebook_url','Facebook','url',false,false,'[]','{}',6),
  ('contact','zalo_phone','Zalo Phone','string',false,true,'[]','{}',7),
  ('contact','birthday','Birthday','date',false,false,'[]','{}',8),
  ('contact','preferred_language','Preferred Language','enum',false,false,'["vi","en","ja","ko","zh"]','{}',9),
  ('deal','commission_pct','Commission %','float',false,false,'[]','{"min":0,"max":50}',4),
  ('deal','expected_revenue','Expected Revenue','float',false,false,'[]','{}',5),
  ('deal','close_reason','Close Reason','enum',false,false,'["price","feature","relationship","urgent","competition"]','{}',6),
  ('deal','payment_terms','Payment Terms','text',false,false,'[]','{}',7),
  ('deal','contract_url','Contract URL','url',false,false,'[]','{}',8),
  ('activity','call_recording_url','Call Recording','url',false,false,'[]','{}',2),
  ('activity','participants','Participants','array',false,false,'[]','{}',3),
  ('activity','follow_up_action','Follow-up Action','enum',false,false,'["none","call","email","meeting","send_proposal","close"]','{}',4)
) AS ef(entity_type, name, label, field_type, is_required, is_searchable, options_array, validation_rules, display_order)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- WORKFLOW EXECUTIONS (more historical data)
-- ============================================================
INSERT INTO workflow.workflow_executions (id, workflow_id, tenant_id, trigger_data, status, started_at, completed_at, execution_log)
SELECT
  gen_random_uuid(),
  w.id,
  w.tenant_id,
  jsonb_build_object('event_id', gen_random_uuid(), 'event', 'create', 'ts', NOW()),
  (ARRAY['completed','completed','completed','completed','failed','running'])[ceil(random()*6)],
  NOW() - (random()*45 || ' days')::interval,
  NOW() - (random()*45 || ' days')::interval + (random()*5 || ' seconds')::interval,
  jsonb_build_array(
    jsonb_build_object('step','validation','ok',true),
    jsonb_build_object('step','condition_eval','matched',random()<0.7),
    jsonb_build_object('step','action','name',w.name,'ok',random()<0.9)
  )
FROM workflow.workflows w
CROSS JOIN generate_series(1, 30)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- AUDIT LOGS EXPANDED (200+ more entries)
-- ============================================================
INSERT INTO audit.audit_logs (tenant_id, actor_user_id, actor_ip, action, entity_type, entity_id, before, after, request_id, trace_id, user_agent, extra, created_at)
SELECT
  t.id,
  (SELECT id FROM auth.users WHERE tenant_id = t.id ORDER BY random() LIMIT 1),
  ('203.0.113.' || (10 + (random()*200)::int))::inet,
  al.action,
  al.entity_type,
  gen_random_uuid(),
  NULL,
  jsonb_build_object('demo', true, 'iteration', gs),
  'req-' || md5(random()::text),
  'trace-' || md5(random()::text),
  (ARRAY['Mozilla/5.0 Chrome','Mozilla/5.0 Safari','Postman','curl/7.81'])[ceil(random()*4)],
  jsonb_build_object('seed_loop','WS-B'),
  NOW() - ((gs*2 + random()*3) || ' hours')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('lead_created','lead'),
  ('lead_updated','lead'),
  ('lead_assigned','lead'),
  ('lead_score_calculated','lead'),
  ('deal_created','deal'),
  ('deal_stage_changed','deal'),
  ('deal_won','deal'),
  ('deal_lost','deal'),
  ('activity_logged','activity'),
  ('note_created','note'),
  ('workflow_executed','workflow'),
  ('campaign_launched','campaign'),
  ('webhook_sent','webhook'),
  ('api_key_used','api_key'),
  ('login_success','session'),
  ('login_failed','session'),
  ('password_reset','user'),
  ('mfa_enabled','user'),
  ('permission_granted','user'),
  ('settings_updated','tenant_settings')
) AS al(action, entity_type)
CROSS JOIN generate_series(1, 20) AS gs
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT DO NOTHING;

-- ============================================================
-- LOGIN HISTORY EXPANDED
-- ============================================================
INSERT INTO audit.login_history (tenant_id, user_id, email, success, failure_reason, ip_address, user_agent, mfa_used, created_at)
SELECT
  u.tenant_id,
  u.id,
  u.email,
  random() < 0.9,
  CASE WHEN random() < 0.1 THEN (ARRAY['invalid_password','mfa_failed','account_locked','expired_token'])[ceil(random()*4)] ELSE NULL END,
  ('203.0.113.' || (10 + (random()*200)::int))::inet,
  (ARRAY['Mozilla/5.0 Chrome','Mozilla/5.0 Safari','Mozilla/5.0 Firefox','PostmanRuntime','curl/7.81','Mozilla/5.0 Mobile Safari'])[ceil(random()*6)],
  random() < 0.3,
  NOW() - ((random()*60)::int || ' hours')::interval
FROM auth.users u
WHERE u.tenant_id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
  AND u.id LIKE 'a0000%'
CROSS JOIN generate_series(1, 5)
ON CONFLICT DO NOTHING;

-- ============================================================
-- API REQUESTS (additional 500 entries)
-- ============================================================
INSERT INTO audit.api_requests (tenant_id, user_id, service, method, route, status, duration_ms, bytes_in, bytes_out, ip_address, user_agent, trace_id, created_at)
SELECT
  t.id,
  (SELECT id FROM auth.users WHERE tenant_id = t.id ORDER BY random() LIMIT 1),
  svc.name,
  svc.method,
  svc.route,
  (ARRAY[200,200,201,204,400,401,403,404,500])[ceil(random()*9)],
  (5 + random()*995)::int,
  (random()*2048)::int,
  (random()*40960)::int,
  ('203.0.113.' || (10 + (random()*200)::int))::inet,
  'Mozilla/5.0',
  'trace-' || md5(random()::text),
  NOW() - ((random()*45)::int || ' hours')::interval
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('crm-service','GET','/api/v1/leads'),
  ('crm-service','POST','/api/v1/leads'),
  ('crm-service','PATCH','/api/v1/leads/{id}'),
  ('crm-service','DELETE','/api/v1/leads/{id}'),
  ('crm-service','GET','/api/v1/deals'),
  ('crm-service','POST','/api/v1/deals'),
  ('crm-service','PATCH','/api/v1/deals/{id}/stage'),
  ('lead-service','POST','/api/v1/leads'),
  ('lead-service','POST','/api/v1/scoring/batch'),
  ('lead-service','POST','/api/v1/scoring/single'),
  ('notification-service','POST','/api/v1/notifications'),
  ('notification-service','GET','/api/v1/notifications'),
  ('auth-service','POST','/api/v1/auth/login'),
  ('auth-service','POST','/api/v1/auth/refresh'),
  ('auth-service','POST','/api/v1/auth/logout'),
  ('tenant-service','GET','/api/v1/tenant/quota'),
  ('tenant-service','GET','/api/v1/tenant/billing'),
  ('landing-service','POST','/api/v1/track/event'),
  ('landing-service','GET','/api/v1/landing/pages'),
  ('analytics-service','POST','/api/v1/events/batch'),
  ('workflow-service','POST','/api/v1/workflows/{id}/execute'),
  ('recording-service','GET','/api/v1/recordings'),
  ('recording-service','GET','/api/v1/recordings/{id}/transcript'),
  ('stt-service','POST','/api/v1/transcribe'),
  ('chat-engine','POST','/api/v1/messages'),
  ('chat-engine','GET','/api/v1/channels'),
  ('webrtc-sfu','POST','/api/v1/rooms'),
  ('search-service','POST','/api/v1/search'),
  ('email-service','POST','/api/v1/email/send'),
  ('email-service','POST','/api/v1/email/template')
) AS svc(name, method, route)
CROSS JOIN generate_series(1, 6)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT DO NOTHING;

SELECT 'expansion_04_workflows_audit (WS-B Loop 3)' AS section, COUNT(*) AS new_workflows
FROM workflow.workflows
WHERE id LIKE 'ee200%010%' OR id LIKE 'ee200%011%' OR id LIKE 'ee200%012%' OR id LIKE 'ee200%013%' OR id LIKE 'ee200%014%' OR id LIKE 'ee200%015%';
