-- Migration 0001: Core Lead Tables
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    source_id UUID,
    contact_id UUID,
    owner_user_id UUID,
    full_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    company_name TEXT,
    job_title TEXT,
    status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'contacted', 'qualified', 'lost', 'won', 'archived')),
    score DECIMAL(5, 2) DEFAULT 0 CHECK (score >= 0 AND score <= 100),
    score_tier TEXT DEFAULT 'cold' CHECK (score_tier IN ('cold', 'warm', 'hot')),
    custom_fields JSONB DEFAULT '{}',
    utm JSONB DEFAULT '{}',
    ip INET,
    user_agent TEXT,
    referrer TEXT,
    fbclid TEXT,
    fbp TEXT,
    gclid TEXT,
    landing_page TEXT,
    campaign_id UUID,
    estimated_value DECIMAL(15, 2),
    lost_reason TEXT,
    next_followup_at TIMESTAMPTZ,
    last_contacted_at TIMESTAMPTZ,
    converted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_leads_tenant ON leads(tenant_id);
CREATE INDEX idx_leads_owner ON leads(tenant_id, owner_user_id);
CREATE INDEX idx_leads_status ON leads(tenant_id, status);
CREATE INDEX idx_leads_score ON leads(tenant_id, score DESC) WHERE score IS NOT NULL;
CREATE INDEX idx_leads_email ON leads(tenant_id, email) WHERE email IS NOT NULL;
CREATE INDEX idx_leads_phone ON leads(tenant_id, phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_leads_source ON leads(tenant_id, source_id);
CREATE INDEX idx_leads_campaign ON leads(tenant_id, campaign_id);
CREATE INDEX idx_leads_created ON leads(tenant_id, created_at DESC);
CREATE INDEX idx_leads_custom ON leads USING GIN (custom_fields);
CREATE INDEX idx_leads_utm ON leads USING GIN (utm);

ALTER TABLE leads ENABLE ROW LEVEL SECURITY;

CREATE POLICY leads_tenant_isolation ON leads
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
