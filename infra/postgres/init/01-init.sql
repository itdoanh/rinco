-- ============================================================
-- RINCO PostgreSQL Database Init
-- ============================================================
-- Extensions, schemas, and initial setup
-- ============================================================

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS ltree;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS postgis;

-- Schemas
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS crm;
CREATE SCHEMA IF NOT EXISTS leads;
CREATE SCHEMA IF NOT EXISTS tenant;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS workflow;

-- ============================================================
-- TENANT SCHEMA
-- ============================================================
CREATE TYPE tenant.status AS ENUM ('active', 'suspended', 'trial', 'cancelled');

CREATE TABLE tenant.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    plan TEXT NOT NULL DEFAULT 'starter',
    status tenant.status NOT NULL DEFAULT 'trial',
    settings JSONB NOT NULL DEFAULT '{}',
    storage_quota_bytes BIGINT NOT NULL DEFAULT 107374182400, -- 100GB
    api_calls_quota INT NOT NULL DEFAULT 1000000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tenants_status ON tenant.tenants(status) WHERE deleted_at IS NULL;

-- ============================================================
-- AUTH SCHEMA
-- ============================================================
CREATE TABLE auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    password_hash TEXT,
    full_name TEXT,
    path LTREE NOT NULL DEFAULT 'root',
    roles TEXT[] NOT NULL DEFAULT ARRAY['member']::TEXT[],
    permissions JSONB NOT NULL DEFAULT '[]',
    is_super_admin BOOLEAN NOT NULL DEFAULT false,
    fido2_credentials JSONB NOT NULL DEFAULT '[]',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant ON auth.users(tenant_id);
CREATE INDEX idx_users_path_gist ON auth.users USING GIST(path);
CREATE INDEX idx_users_path_btree ON auth.users USING BTREE(path);
CREATE INDEX idx_users_email_trgm ON auth.users USING GIN(email gin_trgm_ops);
CREATE INDEX idx_users_super_admin ON auth.users(is_super_admin) WHERE is_super_admin = true;

-- Sessions / Refresh tokens
CREATE TABLE auth.refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    device_fingerprint TEXT,
    ip_address INET,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON auth.refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON auth.refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires ON auth.refresh_tokens(expires_at);

-- FIDO2 challenges (anti-replay)
CREATE TABLE auth.fido2_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    challenge TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('reg', 'auth')),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fido2_challenges_user ON auth.fido2_challenges(user_id);
CREATE INDEX idx_fido2_challenges_expires ON auth.fido2_challenges(expires_at);

-- ============================================================
-- CRM SCHEMA (employee hierarchy)
-- ============================================================
CREATE TYPE crm.employment_status AS ENUM ('active', 'suspended', 'terminated');

CREATE TABLE crm.departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    parent_id UUID REFERENCES crm.departments(id),
    path LTREE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name, parent_id)
);

CREATE INDEX idx_departments_tenant ON crm.departments(tenant_id);
CREATE INDEX idx_departments_path ON crm.departments USING GIST(path);

-- ============================================================
-- LEADS SCHEMA (CRM Core)
-- ============================================================
CREATE TABLE leads.leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    owner_user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    email TEXT,
    phone TEXT,
    full_name TEXT,
    source TEXT,
    campaign_id UUID,
    custom_fields JSONB NOT NULL DEFAULT '{}',
    score NUMERIC(5, 2) NOT NULL DEFAULT 0,
    score_band TEXT GENERATED ALWAYS AS (
        CASE
            WHEN score >= 80 THEN 'hot'
            WHEN score >= 50 THEN 'warm'
            WHEN score >= 20 THEN 'cold'
            ELSE 'low'
        END
    ) STORED,
    stage TEXT NOT NULL DEFAULT 'new',
    last_contacted_at TIMESTAMPTZ,
    converted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_leads_tenant ON leads.leads(tenant_id);
CREATE INDEX idx_leads_owner ON leads.leads(owner_user_id);
CREATE INDEX idx_leads_score ON leads.leads(tenant_id, score DESC);
CREATE INDEX idx_leads_stage ON leads.leads(tenant_id, stage);
CREATE INDEX idx_leads_custom_fields ON leads.leads USING GIN(custom_fields);
CREATE INDEX idx_leads_created_at ON leads.leads(tenant_id, created_at DESC);

-- ============================================================
-- WORKFLOW / DYNAMIC MODEL SCHEMA
-- ============================================================
CREATE TABLE workflow.entity_field_defs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    name TEXT NOT NULL,
    label TEXT,
    field_type TEXT NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT false,
    default_value JSONB,
    validation JSONB NOT NULL DEFAULT '{}',
    options JSONB,
    display_order INT NOT NULL DEFAULT 0,
    is_system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, entity_type, name)
);

CREATE INDEX idx_entity_field_defs_tenant ON workflow.entity_field_defs(tenant_id, entity_type);

CREATE TABLE workflow.workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    trigger_entity TEXT NOT NULL,
    trigger_event TEXT NOT NULL,
    actions JSONB NOT NULL DEFAULT '[]',
    conditions JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflows_tenant ON workflow.workflows(tenant_id);

-- ============================================================
-- AUDIT SCHEMA
-- ============================================================
CREATE TABLE audit.audit_logs (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID,
    actor_user_id UUID,
    actor_ip INET,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT,
    before JSONB,
    after JSONB,
    request_id TEXT,
    trace_id TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_tenant_time ON audit.audit_logs(tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit.audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_actor ON audit.audit_logs(actor_user_id, created_at DESC);

-- ============================================================
-- TENANT SITES
-- ============================================================
CREATE TABLE tenant.tenant_sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    domain TEXT UNIQUE NOT NULL,
    deployment_mode TEXT NOT NULL DEFAULT 'shared',
    vps_node_id UUID,
    theme JSONB NOT NULL DEFAULT '{}',
    branding JSONB NOT NULL DEFAULT '{}',
    pages JSONB NOT NULL DEFAULT '{}',
    seo_settings JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_sites_tenant ON tenant.tenant_sites(tenant_id);

-- ============================================================
-- HELPER FUNCTIONS
-- ============================================================

-- owner_in_subtree: check if current user owns the target user (in subtree)
CREATE OR REPLACE FUNCTION auth.owner_in_subtree(target_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    current_path LTREE;
    target_path LTREE;
BEGIN
    SELECT path INTO current_path FROM auth.users
    WHERE id = current_setting('app.current_user_id', true)::UUID;
    SELECT path INTO target_path FROM auth.users WHERE id = target_user_id;

    IF current_path IS NULL OR target_path IS NULL THEN
        RETURN FALSE;
    END IF;

    RETURN target_path <@ current_path;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================
-- ROW-LEVEL SECURITY (RLS) - MULTI-TENANT ISOLATION
-- ============================================================

-- ============ TENANTS ============
ALTER TABLE tenant.tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant.tenants FORCE ROW LEVEL SECURITY;

-- Super admin sees all
CREATE POLICY tenants_super_admin ON tenant.tenants
  FOR ALL
  USING (current_setting('app.is_super_admin', true) = 'true');

-- Tenant users see only own tenant
CREATE POLICY tenants_self ON tenant.tenants
  FOR SELECT
  USING (id = current_setting('app.current_tenant_id', true)::UUID);

-- ============ USERS ============
ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.users FORCE ROW LEVEL SECURITY;

CREATE POLICY users_self_tenant ON auth.users
  FOR ALL
  USING (
    tenant_id = current_setting('app.current_tenant_id', true)::UUID
    OR current_setting('app.is_super_admin', true) = 'true'
  );

CREATE POLICY users_subtree ON auth.users
  FOR SELECT
  USING (
    current_setting('app.is_super_admin', true) = 'true'
    OR path <@ (
      SELECT path FROM auth.users
      WHERE id = current_setting('app.current_user_id', true)::UUID
    )
    OR id = current_setting('app.current_user_id', true)::UUID
  );

-- ============ LEADS ============
ALTER TABLE leads.leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads.leads FORCE ROW LEVEL SECURITY;

CREATE POLICY leads_tenant ON leads.leads
  FOR ALL
  USING (
    tenant_id = current_setting('app.current_tenant_id', true)::UUID
    OR current_setting('app.is_super_admin', true) = 'true'
  );

CREATE POLICY leads_user_scope ON leads.leads
  FOR ALL
  USING (
    current_setting('app.is_super_admin', true) = 'true'
    OR owner_user_id = current_setting('app.current_user_id', true)::UUID
    OR owner_user_id IN (
      SELECT id FROM auth.users
      WHERE path <@ (
        SELECT path FROM auth.users
        WHERE id = current_setting('app.current_user_id', true)::UUID
      )
    )
  );

-- ============ WORKFLOW ============
ALTER TABLE workflow.entity_field_defs ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow.entity_field_defs FORCE ROW LEVEL SECURITY;

CREATE POLICY entity_field_defs_tenant ON workflow.entity_field_defs
  FOR ALL
  USING (
    tenant_id = current_setting('app.current_tenant_id', true)::UUID
    OR current_setting('app.is_super_admin', true) = 'true'
  );

ALTER TABLE workflow.workflows ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow.workflows FORCE ROW LEVEL SECURITY;

CREATE POLICY workflows_tenant ON workflow.workflows
  FOR ALL
  USING (
    tenant_id = current_setting('app.current_tenant_id', true)::UUID
    OR current_setting('app.is_super_admin', true) = 'true'
  );

-- ============ TENANT SITES ============
ALTER TABLE tenant.tenant_sites ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant.tenant_sites FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_sites_tenant ON tenant.tenant_sites
  FOR ALL
  USING (
    tenant_id = current_setting('app.current_tenant_id', true)::UUID
    OR current_setting('app.is_super_admin', true) = 'true'
  );

-- ============================================================
-- SEED DATA
-- ============================================================

-- Create root tenant for super admin operations
INSERT INTO tenant.tenants (id, slug, name, plan, status)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    'rinco-root',
    'RINCO Root',
    'enterprise',
    'active'
) ON CONFLICT DO NOTHING;

-- Create super admin user
INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, roles, is_super_admin, path)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000000',
    'admin@rinco.app',
    -- bcrypt hash of "rinco_dev_password"
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'Super Admin',
    ARRAY['super_admin']::TEXT[],
    true,
    'root'
) ON CONFLICT DO NOTHING;

-- Create demo tenant
INSERT INTO tenant.tenants (id, slug, name, plan, status)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'demo',
    'Demo Company',
    'pro',
    'active'
) ON CONFLICT DO NOTHING;

-- Create demo admin user
INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, roles, path)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    '11111111-1111-1111-1111-111111111111',
    'demo@demo.com',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'Demo Admin',
    ARRAY['admin']::TEXT[],
    'root'
) ON CONFLICT DO NOTHING;

-- ============================================================
-- Updated_at triggers
-- ============================================================
CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_tenants_updated_at
    BEFORE UPDATE ON tenant.tenants
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_leads_updated_at
    BEFORE UPDATE ON leads.leads
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_tenant_sites_updated_at
    BEFORE UPDATE ON tenant.tenant_sites
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_workflows_updated_at
    BEFORE UPDATE ON workflow.workflows
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();
