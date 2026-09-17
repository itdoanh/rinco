-- ============================================================
-- LEGACY SCHEMA SETUP for Demo Seed (Loop 202)
-- Creates tables that exist in per-service databases but
-- may not be installed in the unified 'rinco' database.
-- All statements are idempotent (CREATE TABLE IF NOT EXISTS).
-- ============================================================

-- (Schemas already created in master; just safety re-create here.)
CREATE SCHEMA IF NOT EXISTS crm;
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS leads;
CREATE SCHEMA IF NOT EXISTS workflow;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS billing;
CREATE SCHEMA IF NOT EXISTS tenant;
CREATE SCHEMA IF NOT EXISTS notification;
CREATE SCHEMA IF NOT EXISTS email;
CREATE SCHEMA IF NOT EXISTS landing;
CREATE SCHEMA IF NOT EXISTS observability;
CREATE SCHEMA IF NOT EXISTS meta_capi;
CREATE SCHEMA IF NOT EXISTS app;
CREATE SCHEMA IF NOT EXISTS public;

-- ============================================================
-- Legacy crm-service tables (no ltree path required, no RLS bypass).
-- ============================================================
CREATE TABLE IF NOT EXISTS public.companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    industry TEXT,
    size TEXT CHECK (size IN ('startup', 'small', 'medium', 'large', 'enterprise')),
    website TEXT,
    phone TEXT,
    email TEXT,
    address TEXT,
    city TEXT,
    country TEXT DEFAULT 'Vietnam',
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_companies_tenant ON public.companies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_companies_industry ON public.companies(tenant_id, industry);

CREATE TABLE IF NOT EXISTS public.contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    company_id UUID REFERENCES public.companies(id) ON DELETE SET NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    job_title TEXT,
    department TEXT,
    owner_user_id UUID,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'lead')),
    source TEXT CHECK (source IN ('organic', 'referral', 'social', 'ads', 'direct', 'other')),
    custom_fields JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    last_contacted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_contacts_tenant ON public.contacts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_contacts_company ON public.contacts(tenant_id, company_id);
CREATE INDEX IF NOT EXISTS idx_contacts_owner ON public.contacts(tenant_id, owner_user_id);

CREATE TABLE IF NOT EXISTS public.deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    contact_id UUID REFERENCES public.contacts(id) ON DELETE SET NULL,
    owner_user_id UUID,
    name TEXT NOT NULL,
    value DECIMAL(15, 2) DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'VND',
    stage TEXT NOT NULL DEFAULT 'prospecting' CHECK (stage IN (
        'prospecting', 'qualification', 'proposal', 'negotiation',
        'won', 'lost', 'on_hold'
    )),
    probability INT CHECK (probability >= 0 AND probability <= 100),
    expected_close_date DATE,
    actual_close_date DATE,
    lost_reason TEXT,
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_deals_tenant ON public.deals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_deals_contact ON public.deals(tenant_id, contact_id);
CREATE INDEX IF NOT EXISTS idx_deals_owner ON public.deals(tenant_id, owner_user_id);
CREATE INDEX IF NOT EXISTS idx_deals_stage ON public.deals(tenant_id, stage);

CREATE TABLE IF NOT EXISTS public.deal_stage_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id UUID NOT NULL REFERENCES public.deals(id) ON DELETE CASCADE,
    from_stage TEXT,
    to_stage TEXT NOT NULL,
    changed_by UUID,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT
);
CREATE INDEX IF NOT EXISTS idx_deal_stage_history_deal ON public.deal_stage_history(deal_id, changed_at DESC);

CREATE TABLE IF NOT EXISTS public.tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    color TEXT DEFAULT '#6366f1',
    description TEXT,
    usage_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_tags_tenant ON public.tags(tenant_id);

CREATE TABLE IF NOT EXISTS public.contact_tags (
    contact_id UUID NOT NULL REFERENCES public.contacts(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES public.tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (contact_id, tag_id)
);

CREATE TABLE IF NOT EXISTS public.custom_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('contact', 'company', 'deal', 'lead', 'activity')),
    name TEXT NOT NULL,
    field_key TEXT NOT NULL,
    field_type TEXT NOT NULL CHECK (field_type IN (
        'text', 'textarea', 'number', 'currency', 'date', 'datetime',
        'boolean', 'select', 'multiselect', 'email', 'phone', 'url'
    )),
    description TEXT,
    options JSONB DEFAULT '[]',
    default_value JSONB,
    validation_rules JSONB DEFAULT '{}',
    is_required BOOLEAN DEFAULT false,
    is_unique BOOLEAN DEFAULT false,
    display_order INT DEFAULT 0,
    width TEXT DEFAULT 'full' CHECK (width IN ('full', 'half', 'third')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, entity_type, field_key)
);
CREATE INDEX IF NOT EXISTS idx_custom_fields_tenant ON public.custom_fields(tenant_id);
CREATE INDEX IF NOT EXISTS idx_custom_fields_entity ON public.custom_fields(tenant_id, entity_type);

CREATE TABLE IF NOT EXISTS public.activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    contact_id UUID REFERENCES public.contacts(id) ON DELETE CASCADE,
    deal_id UUID REFERENCES public.deals(id) ON DELETE SET NULL,
    type TEXT NOT NULL CHECK (type IN ('call', 'email', 'meeting', 'task', 'note')),
    subject TEXT NOT NULL,
    body TEXT,
    due_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    priority TEXT DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    reminder_at TIMESTAMPTZ,
    owner_user_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_activities_tenant ON public.activities(tenant_id);
CREATE INDEX IF NOT EXISTS idx_activities_contact ON public.activities(tenant_id, contact_id);

CREATE TABLE IF NOT EXISTS public.notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    contact_id UUID REFERENCES public.contacts(id) ON DELETE CASCADE,
    deal_id UUID REFERENCES public.deals(id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    author_id UUID,
    is_pinned BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- ============================================================
-- Legacy lead-service tables (public schema)
-- ============================================================
CREATE TABLE IF NOT EXISTS public.lead_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    utm_source TEXT,
    utm_medium TEXT,
    utm_campaign TEXT,
    utm_term TEXT,
    utm_content TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_lead_sources_tenant ON public.lead_sources(tenant_id);

CREATE TABLE IF NOT EXISTS public.pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    color TEXT DEFAULT '#6366f1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, name)
);

CREATE TABLE IF NOT EXISTS public.pipeline_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES public.pipelines(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    probability INT DEFAULT 50 CHECK (probability >= 0 AND probability <= 100),
    color TEXT DEFAULT '#6366f1',
    is_won BOOLEAN DEFAULT false,
    is_lost BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    source_id UUID REFERENCES public.lead_sources(id) ON DELETE SET NULL,
    contact_id UUID,
    owner_user_id UUID,
    pipeline_id UUID REFERENCES public.pipelines(id) ON DELETE SET NULL,
    stage_id UUID REFERENCES public.pipeline_stages(id) ON DELETE SET NULL,
    full_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    company_name TEXT,
    job_title TEXT,
    status TEXT NOT NULL DEFAULT 'new' CHECK (status IN (
        'new', 'contacted', 'qualified', 'proposal', 'won', 'lost', 'archived'
    )),
    score DECIMAL(5, 2) DEFAULT 0,
    score_tier TEXT CHECK (score_tier IN ('cold', 'warm', 'hot')),
    estimated_value DECIMAL(15, 2) DEFAULT 0,
    custom_fields JSONB DEFAULT '{}',
    utm JSONB DEFAULT '{}',
    ip INET,
    user_agent TEXT,
    referrer TEXT,
    fbclid TEXT,
    gclid TEXT,
    ttclid TEXT,
    landing_page TEXT,
    campaign_id TEXT,
    tags TEXT[] DEFAULT '{}',
    next_followup_at TIMESTAMPTZ,
    last_contacted_at TIMESTAMPTZ,
    converted_at TIMESTAMPTZ,
    lost_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS public.lead_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES public.leads(id) ON DELETE CASCADE,
    author_id UUID,
    body TEXT NOT NULL,
    is_pinned BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS public.lead_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES public.leads(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('CALL', 'EMAIL', 'MEETING', 'NOTE', 'STATUS_CHANGE', 'ASSIGN', 'TASK', 'SCORE_UPDATE')),
    payload JSONB DEFAULT '{}',
    actor_id UUID,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.lead_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES public.leads(id) ON DELETE CASCADE,
    from_user_id UUID,
    to_user_id UUID,
    reason TEXT,
    assigned_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.lead_stage_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES public.leads(id) ON DELETE CASCADE,
    from_stage TEXT,
    to_stage TEXT NOT NULL,
    changed_by UUID,
    notes TEXT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.clickhouse_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    lead_id UUID,
    user_id UUID,
    properties JSONB DEFAULT '{}',
    event_date DATE NOT NULL DEFAULT CURRENT_DATE,
    synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- landing-service tables (public.legacy_landing_*)
-- ============================================================
CREATE TABLE IF NOT EXISTS public.landing_forms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    fields JSONB NOT NULL DEFAULT '[]',
    submit_action JSONB NOT NULL DEFAULT '{}',
    success_message JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.landing_fb_pixel_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    pixel_id TEXT NOT NULL,
    access_token TEXT,
    test_event_code TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.tracking_clicks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    target_url TEXT NOT NULL,
    utm JSONB DEFAULT '{}',
    clicks INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- email-service tables
-- ============================================================
CREATE TABLE IF NOT EXISTS email.email_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    template_id UUID,
    recipient_email TEXT,
    subject TEXT,
    body TEXT,
    status TEXT DEFAULT 'pending',
    provider_message_id TEXT,
    error TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_email_logs_tenant ON email.email_logs(tenant_id);

CREATE TABLE IF NOT EXISTS email.email_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    events TEXT[] DEFAULT '{}',
    secret TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- observability-service tables
-- ============================================================
CREATE TABLE IF NOT EXISTS observability.alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    name TEXT NOT NULL,
    description TEXT,
    query TEXT,
    severity TEXT,
    channels TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- meta-capi-service tables
-- ============================================================
CREATE TABLE IF NOT EXISTS meta_capi.events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_name TEXT NOT NULL,
    event_id UUID,
    event_time TIMESTAMPTZ DEFAULT NOW(),
    user_email TEXT,
    user_phone TEXT,
    click_id TEXT,
    action_source TEXT,
    event_source_url TEXT,
    metadata JSONB DEFAULT '{}',
    sent_to_meta BOOLEAN DEFAULT false,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- billing service legacy tenants_usage (already in 0010_subs.sql via tenant.tenant_usage)
-- ============================================================
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
