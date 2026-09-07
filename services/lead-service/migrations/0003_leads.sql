-- Migration 0003: Leads table
-- =============================================================================

CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    source_id UUID REFERENCES lead_sources(id) ON DELETE SET NULL,
    contact_id UUID,
    owner_user_id UUID,
    pipeline_id UUID REFERENCES pipelines(id) ON DELETE SET NULL,
    stage_id UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
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
    fbp TEXT,
    fbc TEXT,
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

CREATE INDEX idx_leads_tenant ON leads(tenant_id);
CREATE INDEX idx_leads_owner ON leads(tenant_id, owner_user_id);
CREATE INDEX idx_leads_source ON leads(tenant_id, source_id);
CREATE INDEX idx_leads_status ON leads(tenant_id, status);
CREATE INDEX idx_leads_pipeline ON leads(tenant_id, pipeline_id);
CREATE INDEX idx_leads_stage ON leads(tenant_id, stage_id);
CREATE INDEX idx_leads_score ON leads(tenant_id, score DESC) WHERE score IS NOT NULL;
CREATE INDEX idx_leads_email ON leads(tenant_id, email) WHERE email IS NOT NULL;
CREATE INDEX idx_leads_phone ON leads(tenant_id, phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_leads_tags ON leads USING GIN (tags);
CREATE INDEX idx_leads_utm ON leads USING GIN (utm);
CREATE INDEX idx_leads_custom_fields ON leads USING GIN (custom_fields);
CREATE INDEX idx_leads_followup ON leads(tenant_id, next_followup_at) WHERE next_followup_at IS NOT NULL;
CREATE INDEX idx_leads_created ON leads(tenant_id, created_at DESC);

ALTER TABLE leads ENABLE ROW LEVEL SECURITY;

CREATE POLICY leads_tenant_isolation ON leads
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (owner_user_id = current_setting('app.current_user_id', true)::UUID)
        )
    );
