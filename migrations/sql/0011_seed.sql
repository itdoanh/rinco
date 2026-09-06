-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0011 — Super-admin seed data
-- ============================================================

INSERT INTO tenant.tenants (id, slug, name, plan, status, isolation_mode)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    'rinco-root',
    'RINCO Root',
    'enterprise',
    'active',
    'SHARED'
)
ON CONFLICT (id) DO NOTHING;

-- Super admin user (default password "rinco_dev_password")
-- bcrypt hash generated with cost=10
INSERT INTO auth.users (
    id, tenant_id, email, password_hash, full_name, role, roles,
    is_super_admin, path, mfa_enabled
)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000000',
    'admin@rinco.app',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'Super Admin',
    'super_admin',
    ARRAY['super_admin']::TEXT[],
    true,
    text2ltree('root'),
    false
)
ON CONFLICT (id) DO NOTHING;

-- Demo tenant
INSERT INTO tenant.tenants (id, slug, name, plan, status)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'demo',
    'Demo Company',
    'pro',
    'active'
)
ON CONFLICT (id) DO NOTHING;

-- Demo admin user
INSERT INTO auth.users (
    id, tenant_id, email, password_hash, full_name, role, roles, path
)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    '11111111-1111-1111-1111-111111111111',
    'demo@demo.com',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'Demo Admin',
    'tenant_admin',
    ARRAY['admin']::TEXT[],
    text2ltree('root')
)
ON CONFLICT (id) DO NOTHING;

-- Demo subscription
INSERT INTO billing.subscriptions (
    id, tenant_id, plan_code, status,
    current_period_start, current_period_end
)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    '11111111-1111-1111-1111-111111111111',
    'pro',
    'active',
    NOW(),
    NOW() + INTERVAL '30 days'
)
ON CONFLICT DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM billing.subscriptions WHERE id IN (
    '22222222-2222-2222-2222-222222222222'
);
DELETE FROM auth.users WHERE id IN (
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000001'
);
DELETE FROM tenant.tenants WHERE id IN (
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111'
);
-- +goose StatementEnd
