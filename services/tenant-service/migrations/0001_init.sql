-- 0001_init.sql — tenants core table
CREATE SCHEMA IF NOT EXISTS tenant;

CREATE TABLE IF NOT EXISTS tenant.tenants (
    id            UUID PRIMARY KEY,
    slug          TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'trial'
                  CHECK (status IN ('trial', 'active', 'suspended', 'deleted')),
    plan          TEXT NOT NULL DEFAULT 'starter'
                  CHECK (plan IN ('starter', 'pro', 'business', 'enterprise')),
    settings_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    branding_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    suspended_at  TIMESTAMPTZ,
    suspended_reason TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tenants_status  ON tenant.tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_plan    ON tenant.tenants(plan)   WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_created ON tenant.tenants(created_at DESC);

-- Audit: track tenant lifecycle changes
CREATE TABLE IF NOT EXISTS tenant.tenant_audit (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    actor_id    TEXT,
    actor_email TEXT,
    action      TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenant_audit_tenant ON tenant.tenant_audit(tenant_id, created_at DESC);
