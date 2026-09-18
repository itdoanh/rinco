-- ============================================================
-- WS-D: TENANT EXPANSION (2 new tenants to reach 5+ total)
-- Vinamilk Distribution (B2C, FMCG distribution channel)
-- VNG Corporation (B2B tech — Zalo, Zing, gaming)
-- Each tenant gets: row in tenant.tenants + tenant_domains + tenant_sites + tenant_branding + tenant_settings
-- All INSERTs idempotent via ON CONFLICT.
-- ============================================================

-- ============================================================================
-- VINAMILK DISTRIBUTION (cccccccc-0000-0000-0000-000000000004) — created 540d ago
-- ============================================================================
INSERT INTO tenant.tenants (id, slug, name, display_name, description, logo_url, website, plan, status, isolation_mode, region, settings, metadata, storage_quota_bytes, api_calls_quota, max_users, max_leads_per_month, created_at, updated_at)
VALUES
  ('cccccccc-0000-0000-0000-000000000004', 'vinamilk-dist', 'Vinamilk Distribution', 'Vinamilk Distribution Network', 'Hệ thống phân phối sữa & FMCG - quản lý hơn 200 đại lý trên toàn quốc', 'https://cdn.rinco.app/logos/vinamilk-dist.png', 'https://distribution.vinamilk.com.vn', 'enterprise', 'active', 'SHARED', 'ap-southeast-1',
    '{"theme":"fresh","locale":"vi-VN","currency":"VND","timezone":"Asia/Ho_Chi_Minh","features":{"ai_scoring":true,"webhook_out":true,"api_access":true,"sso":true,"white_label":false}}'::jsonb,
    '{"industry":"FMCG Distribution","employees":1200,"founded":1976,"tax_code":"0300588569","address":"Số 10 Tôn Đức Thắng, Quận 1, TP.HCM","parent":"Vinamilk JSC"}'::jsonb,
    2147483648000, 20000000, 500, 2000000, NOW() - INTERVAL '540 days', NOW() - INTERVAL '3 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- VNG CORPORATION (dddddddd-0000-0000-0000-000000000005) — created 365d ago
-- ============================================================================
INSERT INTO tenant.tenants (id, slug, name, display_name, description, logo_url, website, plan, status, isolation_mode, region, settings, metadata, storage_quota_bytes, api_calls_quota, max_users, max_leads_per_month, created_at, updated_at)
VALUES
  ('dddddddd-0000-0000-0000-000000000005', 'vng-corp', 'VNG Corporation', 'VNG Corporation - Zalo, Zing, Gaming', 'Tập đoàn công nghệ VNG - phát triển Zalo, Zing MP3, game studio, cloud services', 'https://cdn.rinco.app/logos/vng-corp.png', 'https://vng.com.vn', 'enterprise', 'active', 'SHARED', 'ap-southeast-1',
    '{"theme":"zalo","locale":"vi-VN","currency":"VND","timezone":"Asia/Ho_Chi_Minh","features":{"ai_scoring":true,"webhook_out":true,"api_access":true,"sso":true,"white_label":true,"api_quotas_higher":true}}'::jsonb,
    '{"industry":"Internet & Software","employees":3500,"founded":2004,"tax_code":"0305286165","address":"Tầng 17, Tòa nhà VNG, Đường Số 1, Thảo Điền, Quận 2, TP.HCM","products":["Zalo","Zing MP3","VNG Cloud","Game Studios"]}'::jsonb,
    4294967296000, 100000000, 2000, 5000000, NOW() - INTERVAL '365 days', NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- TENANT DOMAINS (new tenants)
-- ============================================================================
INSERT INTO tenant.tenant_domains (id, tenant_id, hostname, routing, verified, verified_at, created_at, updated_at) VALUES
  ('dddddddd-1000-0000-0000-000000000001', 'cccccccc-0000-0000-0000-000000000004', 'vinamilk.rinco.app',       'subdomain', true, NOW() - INTERVAL '500 days', NOW() - INTERVAL '540 days', NOW()),
  ('dddddddd-1000-0000-0000-000000000002', 'cccccccc-0000-0000-0000-000000000004', 'crm.vinamilk.com.vn',       'custom',    true, NOW() - INTERVAL '480 days', NOW() - INTERVAL '530 days', NOW()),
  ('dddddddd-1000-0000-0000-000000000003', 'cccccccc-0000-0000-0000-000000000004', 'dist.vinamilk-dist.vn',     'subdomain', true, NOW() - INTERVAL '300 days', NOW() - INTERVAL '310 days', NOW()),
  ('dddddddd-1000-0000-0000-000000000004', 'dddddddd-0000-0000-0000-000000000005', 'vng.rinco.app',             'subdomain', true, NOW() - INTERVAL '350 days', NOW() - INTERVAL '365 days', NOW()),
  ('dddddddd-1000-0000-0000-000000000005', 'dddddddd-0000-0000-0000-000000000005', 'crm.vng.com.vn',            'custom',    true, NOW() - INTERVAL '340 days', NOW() - INTERVAL '360 days', NOW()),
  ('dddddddd-1000-0000-0000-000000000006', 'dddddddd-0000-0000-0000-000000000005', 'sales.zalo.me',             'custom',    true, NOW() - INTERVAL '200 days', NOW() - INTERVAL '210 days', NOW())
ON CONFLICT (hostname) DO NOTHING;

-- ============================================================================
-- TENANT SITES (per-tenant landing config)
-- ============================================================================
INSERT INTO tenant.tenant_sites (id, tenant_id, domain, deployment_mode, theme, branding, pages, seo_settings, status, created_at, updated_at) VALUES
  ('eeeeeeee-1000-0000-0000-000000000004', 'cccccccc-0000-0000-0000-000000000004', 'vinamilk-dist.vn', 'shared',
    '{"primary":"#0066B3","secondary":"#FFC20E","font":"Roboto"}'::jsonb,
    '{"name":"Vinamilk Distribution","tagline":"Sữa tươi từ trang trại đến ngôi nhà bạn","logo":"https://cdn.rinco.app/logos/vinamilk-dist.png"}'::jsonb,
    '{"home":"/","about":"/about","products":"/products","agents":"/agents","contact":"/contact"}'::jsonb,
    '{"title":"Vinamilk Distribution - Mạng lưới đại lý toàn quốc","description":"Quản lý đại lý, đơn hàng, kho và giao nhận","og_image":"https://cdn.rinco.app/og/vinamilk-dist.png"}'::jsonb,
    'published', NOW() - INTERVAL '540 days', NOW() - INTERVAL '3 days'),

  ('eeeeeeee-1000-0000-0000-000000000005', 'dddddddd-0000-0000-0000-000000000005', 'vng.com.vn', 'shared',
    '{"primary":"#0068FF","secondary":"#00C7B7","font":"Inter"}'::jsonb,
    '{"name":"VNG Corporation","tagline":"Make the Internet a better place for Vietnamese","logo":"https://cdn.rinco.app/logos/vng-corp.png"}'::jsonb,
    '{"home":"/","products":"/products","careers":"/careers","about":"/about","contact":"/contact"}'::jsonb,
    '{"title":"VNG Corporation - Zalo, Zing, Cloud","description":"Tập đoàn công nghệ hàng đầu Việt Nam","og_image":"https://cdn.rinco.app/og/vng-corp.png"}'::jsonb,
    'published', NOW() - INTERVAL '365 days', NOW() - INTERVAL '1 day')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- TENANT BRANDING (legacy mirror)
-- ============================================================================
INSERT INTO tenant.tenant_branding (tenant_id, logo_url, favicon_url, primary_color, secondary_color, accent_color, font_family, custom_css, email_logo_url, updated_at)
SELECT id, logo_url,
       'https://cdn.rinco.app/favicons/' || slug || '.ico',
       CASE slug WHEN 'vinamilk-dist' THEN '#0066B3' WHEN 'vng-corp' THEN '#0068FF' END,
       CASE slug WHEN 'vinamilk-dist' THEN '#FFC20E' WHEN 'vng-corp' THEN '#00C7B7' END,
       '#10B981', 'Inter', '', 'https://cdn.rinco.app/logos/' || slug || '-email.png', NOW()
FROM tenant.tenants
WHERE id IN ('cccccccc-0000-0000-0000-000000000004','dddddddd-0000-0000-0000-000000000005')
ON CONFLICT (tenant_id) DO NOTHING;

-- ============================================================================
-- TENANT SETTINGS (key/value)
-- ============================================================================
INSERT INTO tenant.tenant_settings (id, tenant_id, key, value, updated_at)
SELECT gen_random_uuid(), t.id, kv.k, kv.v::jsonb, NOW()
FROM tenant.tenants t
CROSS JOIN (VALUES
  ('feature_flags','{"ai_scoring_enabled":true,"auto_assign_enabled":true,"email_inbound_enabled":true,"churn_prediction_enabled":true,"webhook_outbound_enabled":true,"realtime_dashboard":true,"distributor_route_planning":true}'),
  ('integrations','{"zalo_oa":"","facebook_pixel_id":"","google_ads_id":"","tiktok_pixel_id":"","salesforce_sync":false,"hubspot_sync":false,"sap_sync":true}'),
  ('working_hours','{"mon":"08:00-17:30","tue":"08:00-17:30","wed":"08:00-17:30","thu":"08:00-17:30","fri":"08:00-17:00","sat":"08:00-12:00","sun":"closed"}'),
  ('lead_assignment_rules','{"round_robin":true,"weight_by_capacity":true,"region_based":true,"auto_tag_by_source":true,"tier_based":true}'),
  ('notification_preferences','{"new_lead_alert":true,"daily_report":true,"weekly_summary":true,"score_threshold":75,"sms_alert_enabled":true}')
) AS kv(k, v)
WHERE t.id IN ('cccccccc-0000-0000-0000-000000000004','dddddddd-0000-0000-0000-000000000005')
ON CONFLICT (tenant_id, key) DO NOTHING;