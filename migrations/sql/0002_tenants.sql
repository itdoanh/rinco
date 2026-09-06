-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0002 — tenants table + RLS
-- ============================================================

CREATE TABLE IF NOT EXISTS tenant.tenants (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug               CITEXT UNIQUE NOT NULL,
    name               TEXT NOT NULL,
    display_name       TEXT,
    description        TEXT,
    logo_url           TEXT,
    website            TEXT,
    plan               TEXT NOT NULL DEFAULT 'starter'
                       CHECK (plan IN ('free','starter','business','enterprise')),
    status             TEXT NOT NULL DEFAULT 'active'
                       CHECK (status IN ('active','suspended','trial','cancelled','archived')),
    isolation_mode     TEXT NOT NULL DEFAULT 'SHARED'
                       CHECK (isolation_mode IN ('SHARED','ISOLATED_VPS','HYBRID')),
    region             TEXT NOT NULL DEFAULT 'ap-southeast-1',
    vps_node_id        UUID,
    settings           JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata           JSONB NOT NULL DEFAULT '{}'::jsonb,
    storage_quota_bytes BIGINT NOT NULL DEFAULT 107374182400,  -- 100 GB
    api_calls_quota     BIGINT NOT NULL DEFAULT 1000000,
    max_users           INTEGER NOT NULL DEFAULT 50,
    max_leads_per_month INTEGER NOT NULL DEFAULT 1000,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug   ON tenant.tenants(slug)  WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenant.tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_plan   ON tenant.tenants(plan);
CREATE INDEX IF NOT EXISTS idx_tenants_region ON tenant.tenants(region);

CREATE TRIGGER trg_tenants_updated_at
    BEFORE UPDATE ON tenant.tenants
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- Domains (subpath / subdomain / custom)
CREATE TABLE IF NOT EXISTS tenant.tenant_domains (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    hostname        CITEXT UNIQUE NOT NULL,
    routing         TEXT NOT NULL CHECK (routing IN ('subpath','subdomain','custom')),
    verified        BOOLEAN NOT NULL DEFAULT false,
    verified_at     TIMESTAMPTZ,
    tls_cert_id     UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tenant_domains_tenant ON tenant.tenant_domains(tenant_id);

-- Tenant sites (one tenant can own multiple sites)
CREATE TABLE IF NOT EXISTS tenant.tenant_sites (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    domain          CITEXT UNIQUE NOT NULL,
    deployment_mode TEXT NOT NULL DEFAULT 'shared',
    vps_node_id     UUID,
    theme           JSONB NOT NULL DEFAULT '{}'::jsonb,
    branding        JSONB NOT NULL DEFAULT '{}'::jsonb,
    pages           JSONB NOT NULL DEFAULT '{}'::jsonb,
    seo_settings    JSONB NOT NULL DEFAULT '{}'::jsonb,
    status          TEXT NOT NULL DEFAULT 'draft',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_tenant_sites_tenant ON tenant.tenant_sites(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_sites_status ON tenant.tenant_sites(status);

CREATE TRIGGER trg_tenant_sites_updated_at
    BEFORE UPDATE ON tenant.tenant_sites
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- ============================================================
-- RLS
-- ============================================================
ALTER TABLE tenant.tenants       ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant.tenants       FORCE  ROW LEVEL SECURITY;
ALTER TABLE tenant.tenant_domains ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant.tenant_domains FORCE  ROW LEVEL SECURITY;
ALTER TABLE tenant.tenant_sites  ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant.tenant_sites  FORCE  ROW LEVEL SECURITY;

-- Tenants: everyone in a tenant can see their own; super-admin sees all
CREATE POLICY tenants_self_select ON tenant.tenants
    FOR SELECT USING (
        app.bypass_rls_check()
        OR id = app.current_tenant_id()
    );

CREATE POLICY tenants_super_admin_modify ON tenant.tenants
    FOR ALL
    USING (app.bypass_rls_check())
    WITH CHECK (app.bypass_rls_check());

CREATE POLICY tenant_domains_tenant ON tenant.tenant_domains
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY tenant_sites_tenant ON tenant.tenant_sites
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tenant.tenant_sites    CASCADE;
DROP TABLE IF EXISTS tenant.tenant_domains  CASCADE;
DROP TABLE IF EXISTS tenant.tenants         CASCADE;
-- +goose StatementEnd
