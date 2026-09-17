-- ============================================================
-- BILLING + SUBSCRIPTIONS SEED (Loop 202)
-- ============================================================

-- ===== SUBSCRIPTIONS =====
INSERT INTO billing.subscriptions (id, tenant_id, plan_code, status, trial_ends_at, current_period_start, current_period_end, billing_interval, seats_limit, storage_limit_bytes, monthly_leads_limit, created_at, updated_at)
VALUES
  ('22222222-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','business','active',NULL,NOW() - INTERVAL '15 days',NOW() + INTERVAL '15 days','month',200,536870912000,500000,NOW() - INTERVAL '120 days',NOW() - INTERVAL '15 days'),
  ('22222222-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000002','pro','active',NULL,NOW() - INTERVAL '10 days',NOW() + INTERVAL '20 days','month',80,268435456000,100000,NOW() - INTERVAL '60 days',NOW() - INTERVAL '10 days'),
  ('22222222-0000-0000-0000-000000000003','bbbbbbbb-0000-0000-0000-000000000003','enterprise','trialing',NOW() + INTERVAL '20 days',NOW() - INTERVAL '10 days',NOW() + INTERVAL '20 days','year',1000,1073741824000,1000000,NOW() - INTERVAL '10 days',NOW())
ON CONFLICT (id) DO NOTHING;

-- ===== INVOICES =====
INSERT INTO billing.invoices (id, tenant_id, subscription_id, invoice_number, period_start, period_end, subtotal, tax, total, currency, status, paid_at, due_at, line_items)
VALUES
  (gen_random_uuid(),'aaaaaaaa-0000-0000-0000-000000000001','22222222-0000-0000-0000-000000000001',
   'INV-2026-0001',NOW() - INTERVAL '45 days',NOW() - INTERVAL '15 days',
   3990000,399000,4389000,'VND','paid',NOW() - INTERVAL '14 days',NOW() - INTERVAL '15 days + 7 days',
   '[{"description":"Business plan - monthly","quantity":1,"unit_price":3990000}]'::jsonb),
  (gen_random_uuid(),'aaaaaaaa-0000-0000-0000-000000000001','22222222-0000-0000-0000-000000000001',
   'INV-2026-0002',NOW() - INTERVAL '15 days',NOW() + INTERVAL '15 days',
   3990000,399000,4389000,'VND','open',NULL,NOW() + INTERVAL '7 days',
   '[{"description":"Business plan - monthly","quantity":1,"unit_price":3990000}]'::jsonb),
  (gen_random_uuid(),'aaaaaaaa-0000-0000-0000-000000000002','22222222-0000-0000-0000-000000000002',
   'INV-2026-0003',NOW() - INTERVAL '40 days',NOW() - INTERVAL '10 days',
   1990000,199000,2189000,'VND','paid',NOW() - INTERVAL '10 days',NOW() - INTERVAL '10 days + 7 days',
   '[{"description":"Pro plan - monthly","quantity":1,"unit_price":1990000}]'::jsonb),
  (gen_random_uuid(),'bbbbbbbb-0000-0000-0000-000000000003','22222222-0000-0000-0000-000000000003',
   'INV-2026-0004',NOW() - INTERVAL '10 days',NOW() + INTERVAL '20 days',
   0,0,0,'VND','open',NULL,NOW() + INTERVAL '20 days',
   '[{"description":"Enterprise trial","quantity":1,"unit_price":0}]'::jsonb)
ON CONFLICT (invoice_number) DO NOTHING;

-- ===== USAGE COUNTERS =====
INSERT INTO billing.usage_counters (tenant_id, metric, period_start, period_end, value, metadata, updated_at)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001','api_calls',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',324516,'{"endpoint":"/api/v1/leads"}',NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000001','leads',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',1287,'{}',NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000001','storage_bytes',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',15728640000,'{}',NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000001','seats',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',18,'{}',NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000002','api_calls',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',89521,'{}',NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000002','leads',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',482,'{}',NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000002','storage_bytes',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',5368709120,'{}',NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000002','seats',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',14,'{}',NOW()),
  ('bbbbbbbb-0000-0000-0000-000000000003','api_calls',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',8521,'{}',NOW()),
  ('bbbbbbbb-0000-0000-0000-000000000003','leads',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',124,'{}',NOW()),
  ('bbbbbbbb-0000-0000-0000-000000000003','seats',DATE_TRUNC('month',NOW()),DATE_TRUNC('month',NOW()) + INTERVAL '1 month',17,'{}',NOW())
ON CONFLICT (tenant_id, metric, period_start) DO NOTHING;

-- ===== PAYMENT METHODS =====
INSERT INTO billing.payment_methods (id, tenant_id, gateway, gateway_pm_id, brand, last4, exp_month, exp_year, is_default, created_at)
VALUES
  ('33300001-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','vnpay','VNPAY-PM-123456','VISA','4242',12,2028,true,NOW() - INTERVAL '100 days'),
  ('33300002-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','momo','MOMO-PM-789012','Mastercard','5412',8,2027,true,NOW() - INTERVAL '50 days'),
  ('33300003-0000-0000-0000-000000000001','bbbbbbbb-0000-0000-0000-000000000003','stripe','STRIPE-PM-345678','Visa','0006',3,2026,true,NOW() - INTERVAL '5 days')
ON CONFLICT (gateway, gateway_pm_id) DO NOTHING;

-- ===== TENANT USAGE (legacy tenant-service table) =====
INSERT INTO tenant.tenant_usage (tenant_id, period, metric, value, updated_at)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001',TO_CHAR(NOW(),'YYYY-MM'),'users',20,NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000001',TO_CHAR(NOW(),'YYYY-MM'),'leads',1287,NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000001',TO_CHAR(NOW(),'YYYY-MM'),'api_calls',324516,NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000001',TO_CHAR(NOW(),'YYYY-MM'),'storage_bytes',15728640000,NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000002',TO_CHAR(NOW(),'YYYY-MM'),'users',16,NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000002',TO_CHAR(NOW(),'YYYY-MM'),'leads',482,NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000002',TO_CHAR(NOW(),'YYYY-MM'),'api_calls',89521,NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000003',TO_CHAR(NOW(),'YYYY-MM'),'users',18,NOW()),
  ('aaaaaaaa-0000-0000-0000-000000000003',TO_CHAR(NOW(),'YYYY-MM'),'leads',124,NOW())
ON CONFLICT (tenant_id, period, metric) DO NOTHING;

-- ===== TENANT AUDIT (tenant-service legacy) =====
INSERT INTO tenant.tenant_audit (tenant_id, actor_id, actor_email, action, payload, created_at)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000001','admin@apexfintech.vn','plan_upgraded','{"from":"pro","to":"business"}',NOW() - INTERVAL '100 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000001','admin@apexfintech.vn','domain_verified','{"domain":"crm.apexfintech.vn"}',NOW() - INTERVAL '90 days'),
  ('aaaaaaaa-0000-0000-0000-000000000001','a0000001-0000-0000-0000-000000000001','admin@apexfintech.vn','user_invited','{"email":"agent2@apexfintech.vn"}',NOW() - INTERVAL '78 days'),
  ('aaaaaaaa-0000-0000-0000-000000000002','a0000002-0000-0000-0000-000000000001','admin@hct.vn','tenant_created','{}',NOW() - INTERVAL '55 days'),
  ('aaaaaaaa-0000-0000-0000-000000000002','a0000002-0000-0000-0000-000000000001','admin@hct.vn','subscription_started','{"plan":"pro"}',NOW() - INTERVAL '55 days'),
  ('bbbbbbbb-0000-0000-0000-000000000003','a0000003-0000-0000-0000-000000000001','demo@demo.com','tenant_created','{}',NOW() - INTERVAL '10 days');
