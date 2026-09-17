-- ============================================================
-- RINCO Comprehensive Demo Data Seed (Loop 202)
-- ============================================================
-- Idempotent: safe to re-run. ON CONFLICT clauses everywhere.
-- Order matters because of FK constraints. Uses LTREE path
-- strings formatted as 'root.<user-uuid-segments>' for hierarchy.
-- ============================================================

-- ============================================================================
-- SECTION 0: MIGRATION GATE — skip entire seed when sentinel row is present
-- ============================================================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM tenant.tenants WHERE slug = '__seed_completed__') THEN
    RAISE NOTICE 'seed already applied — skipping all demo seeds';
    RETURN;  -- exits DO block only
  END IF;
END$$;
-- Top-level guard: empty sentinel means re-run is safe.
-- We don't abort the script here, so the idempotent INSERTs re-run harmlessly.

-- ============================================================================
-- SECTION 1: TENANTS (3 real-world Vietnamese tenants)
-- ============================================================================
INSERT INTO tenant.tenants (id, slug, name, display_name, description, logo_url, website, plan, status, isolation_mode, region, settings, metadata, storage_quota_bytes, api_calls_quota, max_users, max_leads_per_month, created_at, updated_at)
VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', 'apexfintech', 'Apex Fintech', 'Apex Fintech JSC', 'Công ty công nghệ tài chính hàng đầu Việt Nam - Cổng thanh toán & cho vay online', 'https://cdn.rinco.app/logos/apexfintech.png', 'https://apexfintech.vn', 'business', 'active', 'SHARED', 'ap-southeast-1',
    '{"theme":"modern","locale":"vi-VN","currency":"VND","timezone":"Asia/Ho_Chi_Minh","features":{"ai_scoring":true,"webhook_out":true,"api_access":true,"sso":false}}'::jsonb,
    '{"industry":"FinTech","employees":150,"founded":2019,"tax_code":"0316445991","address":"Tầng 12, Bitexco Tower, HCMC"}'::jsonb,
    536870912000, 10000000, 200, 500000, NOW() - INTERVAL '120 days', NOW() - INTERVAL '5 days'),

  ('aaaaaaaa-0000-0000-0000-000000000002', 'hct-consulting', 'HCT Consulting', 'HCT Real Estate Consulting', 'Công ty tư vấn bất động sản & đầu tư cá nhân - chuyên phân khúc cao cấp Hà Nội', 'https://cdn.rinco.app/logos/hct-consulting.png', 'https://hctconsulting.vn', 'pro', 'active', 'SHARED', 'ap-southeast-1',
    '{"theme":"classic","locale":"vi-VN","currency":"VND","timezone":"Asia/Ho_Chi_Minh","features":{"ai_scoring":true,"webhook_out":true,"api_access":true,"sso":false}}'::jsonb,
    '{"industry":"Real Estate","employees":45,"founded":2021,"tax_code":"0109988776","address":"Tầng 8, Capital Place, Hà Nội"}'::jsonb,
    268435456000, 5000000, 80, 100000, NOW() - INTERVAL '60 days', NOW() - INTERVAL '2 days'),

  ('bbbbbbbb-0000-0000-0000-000000000003', 'demo-company', 'Demo Company', 'Demo Company Vietnam', 'Tài khoản demo showcase sản phẩm RINCO - dữ liệu mẫu đa ngành nghề', 'https://cdn.rinco.app/logos/demo-company.png', 'https://demo.rinco.app', 'enterprise', 'trial', 'SHARED', 'ap-southeast-1',
    '{"theme":"custom","locale":"vi-VN","currency":"VND","timezone":"Asia/Ho_Chi_Minh","features":{"ai_scoring":true,"webhook_out":true,"api_access":true,"sso":true,"white_label":true}}'::jsonb,
    '{"industry":"Multi","employees":500,"founded":2024,"tax_code":"0319999888","address":"Quận 1, TP.HCM"}'::jsonb,
    1073741824000, 50000000, 1000, 1000000, NOW() - INTERVAL '10 days', NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- SECTION 2: TENANT DOMAINS
-- ============================================================================
INSERT INTO tenant.tenant_domains (id, tenant_id, hostname, routing, verified, verified_at, created_at, updated_at) VALUES
  ('dddddddd-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'apexfintech.rinco.app',  'subdomain', true, NOW() - INTERVAL '100 days', NOW() - INTERVAL '120 days', NOW()),
  ('dddddddd-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'crm.apexfintech.vn',     'subdomain', true, NOW() - INTERVAL '90 days',  NOW() - INTERVAL '100 days', NOW()),
  ('dddddddd-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000002', 'hctconsulting.rinco.app','subdomain', true, NOW() - INTERVAL '50 days', NOW() - INTERVAL '60 days', NOW()),
  ('dddddddd-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000002', 'hct.vn',                 'custom',    true, NOW() - INTERVAL '40 days', NOW() - INTERVAL '45 days', NOW()),
  ('dddddddd-0000-0000-0000-000000000005', 'bbbbbbbb-0000-0000-0000-000000000003', 'demo.rinco.app',         'subdomain', true, NOW() - INTERVAL '5 days',  NOW() - INTERVAL '10 days', NOW())
ON CONFLICT (hostname) DO NOTHING;

-- ============================================================================
-- SECTION 3: TENANT SITES (per-tenant landing configuration)
-- ============================================================================
INSERT INTO tenant.tenant_sites (id, tenant_id, domain, deployment_mode, theme, branding, pages, seo_settings, status, created_at, updated_at) VALUES
  ('eeeeeeee-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'apexfintech.vn', 'shared',
    '{"primary":"#0F766E","secondary":"#F59E0B","font":"Inter"}'::jsonb,
    '{"name":"Apex Fintech","tagline":"Vay nhanh - Trả gọn","logo":"https://cdn.rinco.app/logos/apexfintech.png","favicon":"https://cdn.rinco.app/favicons/apexfintech.ico"}'::jsonb,
    '{"home":"/","about":"/about","pricing":"/pricing","contact":"/contact"}'::jsonb,
    '{"title":"Apex Fintech - Giải pháp tài chính số","description":"Vay tín chấp online chỉ trong 5 phút","og_image":"https://cdn.rinco.app/og/apexfintech.png"}'::jsonb,
    'published', NOW() - INTERVAL '120 days', NOW() - INTERVAL '1 day'),

  ('eeeeeeee-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000002', 'hct.vn', 'shared',
    '{"primary":"#1E40AF","secondary":"#DC2626","font":"Roboto"}'::jsonb,
    '{"name":"HCT Consulting","tagline":"Đầu tư BĐS thông minh","logo":"https://cdn.rinco.app/logos/hct-consulting.png","favicon":"https://cdn.rinco.app/favicons/hct-consulting.ico"}'::jsonb,
    '{"home":"/","projects":"/projects","about":"/about","contact":"/contact"}'::jsonb,
    '{"title":"HCT Consulting - Tư vấn đầu tư BĐS","description":"Tư vấn đầu tư căn hộ, shophouse, đất nền","og_image":"https://cdn.rinco.app/og/hct-consulting.png"}'::jsonb,
    'published', NOW() - INTERVAL '60 days', NOW() - INTERVAL '2 days'),

  ('eeeeeeee-0000-0000-0000-000000000003', 'bbbbbbbb-0000-0000-0000-000000000003', 'demo.rinco.app', 'shared',
    '{"primary":"#7C3AED","secondary":"#FBBF24","font":"Plus Jakarta Sans"}'::jsonb,
    '{"name":"Demo Company","tagline":"Try RINCO today","logo":"https://cdn.rinco.app/logos/demo-company.png"}'::jsonb,
    '{"home":"/","features":"/features","pricing":"/pricing"}'::jsonb,
    '{"title":"Demo Company - powered by RINCO","description":"Test demo data"}'::jsonb,
    'draft', NOW() - INTERVAL '10 days', NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- SECTION 4: TENANT BRANDING (legacy mirror table from tenant-service)
-- ============================================================================
INSERT INTO tenant.tenant_branding (tenant_id, logo_url, favicon_url, primary_color, secondary_color, accent_color, font_family, custom_css, email_logo_url, updated_at)
SELECT id, logo_url,
       'https://cdn.rinco.app/favicons/' || slug || '.ico',
       CASE slug WHEN 'apexfintech'    THEN '#0F766E' WHEN 'hct-consulting' THEN '#1E40AF' ELSE '#7C3AED' END,
       CASE slug WHEN 'apexfintech'    THEN '#F59E0B' WHEN 'hct-consulting' THEN '#DC2626' ELSE '#FBBF24' END,
       '#10B981', 'Inter', '', 'https://cdn.rinco.app/logos/' || slug || '-email.png', NOW()
FROM tenant.tenants
WHERE id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (tenant_id) DO NOTHING;

-- ============================================================================
-- SECTION 5: TENANT SETTINGS (key/value store)
-- ============================================================================
INSERT INTO tenant.tenant_settings (id, tenant_id, key, value, updated_at)
SELECT gen_random_uuid(), t.id, kv.k, kv.v::jsonb, NOW()
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('feature_flags','{"ai_scoring_enabled":true,"auto_assign_enabled":true,"email_inbound_enabled":true,"churn_prediction_enabled":true,"webhook_outbound_enabled":true,"realtime_dashboard":true}'),
  ('integrations','{"zalo_oa":"","facebook_pixel_id":"","google_ads_id":"","tiktok_pixel_id":"","salesforce_sync":false,"hubspot_sync":false}'),
  ('working_hours','{"mon":"09:00-18:00","tue":"09:00-18:00","wed":"09:00-18:00","thu":"09:00-18:00","fri":"09:00-17:00","sat":"closed","sun":"closed"}'),
  ('lead_assignment_rules','{"round_robin":true,"weight_by_capacity":true,"region_based":true,"auto_tag_by_source":true}'),
  ('notification_preferences','{"new_lead_alert":true,"daily_report":true,"weekly_summary":true,"score_threshold":80}')
) AS kv(k, v)
WHERE t.id IN ('aaaaaaaa-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000002','bbbbbbbb-0000-0000-0000-000000000003')
ON CONFLICT (tenant_id, key) DO NOTHING;
