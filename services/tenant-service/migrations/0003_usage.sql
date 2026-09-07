-- 0003_usage.sql — quota usage tracking

CREATE TABLE IF NOT EXISTS tenant.tenant_usage (
    tenant_id   UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    period      TEXT NOT NULL,                 -- YYYY-MM
    metric      TEXT NOT NULL,                 -- users, leads, deals, storage_bytes, api_calls, ccu
    value       BIGINT NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, period, metric)
);

CREATE INDEX IF NOT EXISTS idx_tenant_usage_period ON tenant.tenant_usage(period DESC);

-- Quota limits per plan (single source of truth, can be overridden per tenant in settings)
CREATE TABLE IF NOT EXISTS tenant.plan_quotas (
    plan           TEXT PRIMARY KEY,
    max_users      INT NOT NULL DEFAULT 5,
    max_storage_mb INT NOT NULL DEFAULT 1024,
    max_api_calls  INT NOT NULL DEFAULT 10000,
    max_ccu        INT NOT NULL DEFAULT 50,
    features       JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO tenant.plan_quotas (plan, max_users, max_storage_mb, max_api_calls, max_ccu, features) VALUES
    ('starter',   5,    1024,  10000,   50,  '{"webauthn":true,"oauth":true,"custom_domain":false}'::jsonb),
    ('pro',       50,   10240, 100000,  200, '{"webauthn":true,"oauth":true,"custom_domain":true}'::jsonb),
    ('business',  500,  51200, 1000000, 1000,'{"webauthn":true,"oauth":true,"custom_domain":true,"sso":true}'::jsonb),
    ('enterprise', 999999, 1024000, 99999999, 99999, '{"webauthn":true,"oauth":true,"custom_domain":true,"sso":true,"audit_log":true}'::jsonb)
ON CONFLICT (plan) DO NOTHING;
