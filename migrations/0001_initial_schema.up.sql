-- Migration 0001: Initial Schema
-- Creates the foundational tables for RINCO platform

BEGIN;

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS ltree;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- ============================================
-- SCHEMAS
-- ============================================
CREATE SCHEMA IF NOT EXISTS tenant;
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS crm;
CREATE SCHEMA IF NOT EXISTS leads;
CREATE SCHEMA IF NOT EXISTS workflow;
CREATE SCHEMA IF NOT EXISTS audit;

-- ============================================
-- TENANT SCHEMA
-- ============================================

CREATE TABLE tenant.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    display_name TEXT,
    description TEXT,
    logo_url TEXT,
    website TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'trial', 'expired', 'deleted')),
    tier TEXT NOT NULL DEFAULT 'starter' CHECK (tier IN ('starter', 'pro', 'enterprise')),
    region TEXT NOT NULL DEFAULT 'ap-southeast-1',
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    max_users INTEGER NOT NULL DEFAULT 10,
    max_leads_per_month INTEGER NOT NULL DEFAULT 1000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tenants_slug ON tenant.tenants(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_status ON tenant.tenants(status) WHERE deleted_at IS NULL;

CREATE TABLE tenant.tenant_sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    domain TEXT NOT NULL UNIQUE,
    subdomain TEXT UNIQUE,
    site_type TEXT NOT NULL DEFAULT 'landing' CHECK (site_type IN ('landing', 'portal', 'custom')),
    is_primary BOOLEAN NOT NULL DEFAULT false,
    ssl_enabled BOOLEAN NOT NULL DEFAULT true,
    ssl_cert TEXT,
    ssl_key TEXT,
    ssl_expires_at TIMESTAMPTZ,
    custom_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_sites_tenant ON tenant.tenant_sites(tenant_id);
CREATE INDEX idx_tenant_sites_domain ON tenant.tenant_sites(domain);

-- ============================================
-- AUTH SCHEMA
-- ============================================

CREATE TABLE auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    password_hash TEXT,
    full_name TEXT NOT NULL,
    avatar_url TEXT,
    phone TEXT,
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('super_admin', 'tenant_admin', 'manager', 'user', 'guest')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_email_verified BOOLEAN NOT NULL DEFAULT false,
    is_phone_verified BOOLEAN NOT NULL DEFAULT false,
    mfa_enabled BOOLEAN NOT NULL DEFAULT false,
    mfa_secret TEXT,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_email ON auth.users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_tenant ON auth.users(tenant_id) WHERE deleted_at IS NULL;

CREATE TABLE auth.refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    device_info JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON auth.refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires ON auth.refresh_tokens(expires_at);

CREATE TABLE auth.fido2_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    counter BIGINT NOT NULL DEFAULT 0,
    transports TEXT[],
    aaguid TEXT,
    name TEXT,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fido2_user ON auth.fido2_credentials(user_id);

-- ============================================
-- CRM SCHEMA
-- ============================================

CREATE TABLE crm.departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    code TEXT,
    description TEXT,
    manager_id UUID REFERENCES auth.users(id),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_departments_tenant ON crm.departments(tenant_id);

CREATE TABLE crm.user_hierarchy (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    manager_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    path LTREE NOT NULL,
    level INTEGER NOT NULL DEFAULT 0,
    position TEXT,
    department_id UUID REFERENCES crm.departments(id),
    is_manager BOOLEAN NOT NULL DEFAULT false,
    start_date DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date DATE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_hierarchy_path ON crm.user_hierarchy USING GIST(path);
CREATE INDEX idx_user_hierarchy_tenant_user ON crm.user_hierarchy(tenant_id, user_id);
CREATE INDEX idx_user_hierarchy_manager ON crm.user_hierarchy(manager_id) WHERE manager_id IS NOT NULL;

-- ============================================
-- LEADS SCHEMA
-- ============================================

CREATE TABLE leads.lead_pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    is_default BOOLEAN NOT NULL DEFAULT false,
    stages JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lead_pipelines_tenant ON leads.lead_pipelines(tenant_id);

CREATE TABLE leads.leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    full_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    company TEXT,
    position TEXT,
    source TEXT NOT NULL DEFAULT 'unknown',
    source_detail JSONB NOT NULL DEFAULT '{}'::jsonb,
    pipeline_id UUID REFERENCES leads.lead_pipelines(id),
    stage TEXT NOT NULL DEFAULT 'new',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'won', 'lost', 'disqualified')),
    score INTEGER NOT NULL DEFAULT 0,
    p_ltv NUMERIC(12, 2),
    assigned_to UUID REFERENCES auth.users(id),
    custom_fields JSONB NOT NULL DEFAULT '{}'::jsonb,
    utm_source TEXT,
    utm_medium TEXT,
    utm_campaign TEXT,
    utm_term TEXT,
    utm_content TEXT,
    fbclid TEXT,
    gclid TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    converted_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_leads_tenant ON leads.leads(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_leads_email ON leads.leads(email) WHERE email IS NOT NULL;
CREATE INDEX idx_leads_phone ON leads.leads(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_leads_assigned ON leads.leads(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_leads_score ON leads.leads(score DESC);
CREATE INDEX idx_leads_custom_fields ON leads.leads USING GIN(custom_fields);

CREATE TABLE leads.lead_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    lead_id UUID NOT NULL REFERENCES leads.leads(id) ON DELETE CASCADE,
    user_id UUID REFERENCES auth.users(id),
    activity_type TEXT NOT NULL,
    description TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lead_activities_lead ON leads.lead_activities(lead_id, created_at DESC);
CREATE INDEX idx_lead_activities_tenant ON leads.lead_activities(tenant_id);

-- ============================================
-- WORKFLOW SCHEMA
-- ============================================

CREATE TABLE workflow.workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    trigger_type TEXT NOT NULL,
    trigger_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    steps JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflows_tenant ON workflow.workflows(tenant_id) WHERE is_active = true;

CREATE TABLE workflow.workflow_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflow.workflows(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    trigger_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed', 'cancelled')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    execution_log JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE INDEX idx_workflow_executions_workflow ON workflow.workflow_executions(workflow_id);
CREATE INDEX idx_workflow_executions_status ON workflow.workflow_executions(status);

-- ============================================
-- DYNAMIC MODEL ENGINE
-- ============================================

CREATE TABLE crm.entity_field_defs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    field_name TEXT NOT NULL,
    field_label TEXT NOT NULL,
    field_type TEXT NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT false,
    is_searchable BOOLEAN NOT NULL DEFAULT false,
    default_value JSONB,
    options JSONB NOT NULL DEFAULT '[]'::jsonb,
    validation_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    display_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, entity_type, field_name)
);

CREATE INDEX idx_entity_field_defs_tenant ON crm.entity_field_defs(tenant_id, entity_type);

-- ============================================
-- AUDIT LOG
-- ============================================

CREATE TABLE audit.audit_logs (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenant.tenants(id),
    user_id UUID REFERENCES auth.users(id),
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_tenant ON audit.audit_logs(tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit.audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_user ON audit.audit_logs(user_id);

-- ============================================
-- ROW LEVEL SECURITY
-- ============================================

ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.user_hierarchy ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads.leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads.lead_activities ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.entity_field_defs ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow.workflows ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit.audit_logs ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY tenant_isolation_users ON auth.users
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID OR current_setting('app.is_super_admin', true) = 'true');

CREATE POLICY tenant_isolation_hierarchy ON crm.user_hierarchy
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID OR current_setting('app.is_super_admin', true) = 'true');

CREATE POLICY tenant_isolation_leads ON leads.leads
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID OR current_setting('app.is_super_admin', true) = 'true');

CREATE POLICY tenant_isolation_lead_activities ON leads.lead_activities
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID OR current_setting('app.is_super_admin', true) = 'true');

CREATE POLICY tenant_isolation_entity_fields ON crm.entity_field_defs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID OR current_setting('app.is_super_admin', true) = 'true');

CREATE POLICY tenant_isolation_workflows ON workflow.workflows
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID OR current_setting('app.is_super_admin', true) = 'true');

CREATE POLICY tenant_isolation_audit ON audit.audit_logs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID OR current_setting('app.is_super_admin', true) = 'true');

-- ============================================
-- SEED DATA: Demo Tenant
-- ============================================

INSERT INTO tenant.tenants (id, slug, name, display_name, tier, status)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'demo',
    'Demo Tenant',
    'Công ty Demo',
    'pro',
    'active'
);

INSERT INTO auth.users (id, tenant_id, email, password_hash, full_name, role)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'admin@demo.com',
    '$argon2id$v=19$m=65536,t=3,p=4$dGVzdHNhbHQ$Yp0Yt8wQq7t8q0Yp0Yt8wQq7t8q0Yp0Yt8wQq7t8q0Y',
    'Demo Admin',
    'tenant_admin'
);

COMMIT;
