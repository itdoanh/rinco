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

-- -- Tenant domains are defined inline above (INSERT INTO tenant.tenant_domains)
-- -- Tenant sites are defined inline above (INSERT INTO tenant.tenant_sites)
-- -- tenant_branding and tenant_settings tables do not exist — skipped
