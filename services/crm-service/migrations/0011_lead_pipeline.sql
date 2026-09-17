-- =============================================================================
-- Migration 0011: Lead lifecycle, pipelines, auto-assignment rules
-- =============================================================================
-- Loop 203 — CRM Tree enterprise completion.
--
-- Tách "leads" ra khỏi "contacts" để pipeline thuần (raw inbound):
--   * Lead: từ landing page, có UTM, fbclid, fbp, score, status, owner
--   * Contact: thực thể CRM "đã qualify", 1-1 với person/account
--
-- Pipeline là danh sách stage cấu hình được per tenant (mặc định
-- cung cấp bộ 7 stage trùng với deals.pipeline_stage).
-- =============================================================================

CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_pipelines_tenant ON pipelines(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_pipelines_default ON pipelines(tenant_id) WHERE is_default = true AND deleted_at IS NULL;

ALTER TABLE pipelines ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS pipelines_tenant_isolation ON pipelines;
CREATE POLICY pipelines_tenant_isolation ON pipelines
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            current_setting('app.is_admin', true) = 'true'
            OR EXISTS (
                SELECT 1 FROM users
                WHERE id = current_setting('app.current_user_id', true)::UUID
                AND tenant_id = pipelines.tenant_id
            )
        )
    );

CREATE TABLE IF NOT EXISTS pipeline_stages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    color TEXT DEFAULT '#6366f1',
    position INT NOT NULL DEFAULT 0,
    probability INT NOT NULL DEFAULT 50 CHECK (probability BETWEEN 0 AND 100),
    sla_hours INT,
    is_won BOOLEAN DEFAULT false,
    is_lost BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (pipeline_id, code)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stages_pipeline ON pipeline_stages(pipeline_id, position);

ALTER TABLE pipeline_stages ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS pipeline_stages_tenant_isolation ON pipeline_stages;
CREATE POLICY pipeline_stages_tenant_isolation ON pipeline_stages
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    pipeline_id UUID REFERENCES pipelines(id) ON DELETE SET NULL,
    current_stage_id UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    converted_contact_id UUID, -- FK tới contacts sau khi convert
    converted_deal_id UUID,    -- FK tới deals nếu auto-create deal
    full_name TEXT,
    email TEXT,
    phone TEXT,
    company_name TEXT,
    source TEXT,                  -- 'facebook', 'website', 'referral', ...
    source_detail TEXT,
    utm JSONB DEFAULT '{}'::jsonb,
    fbclid TEXT,
    fbp TEXT,
    fbc TEXT,
    ip_address INET,
    user_agent TEXT,
    score DECIMAL(5,2),           -- 0..100
    estimated_value DECIMAL(15,2),
    currency TEXT DEFAULT 'VND',
    status TEXT NOT NULL DEFAULT 'NEW' CHECK (status IN (
        'NEW', 'CONTACTED', 'QUALIFIED', 'PROPOSAL', 'WON', 'LOST', 'ARCHIVED'
    )),
    lost_reason TEXT,
    custom_fields JSONB DEFAULT '{}'::jsonb,
    tags TEXT[] DEFAULT '{}',
    next_followup_at TIMESTAMPTZ,
    last_contacted_at TIMESTAMPTZ,
    converted_at TIMESTAMPTZ,
    assigned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_leads_tenant_status ON leads(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_tenant_owner ON leads(tenant_id, owner_user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_tenant_stage ON leads(tenant_id, current_stage_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_score ON leads(tenant_id, score DESC) WHERE score IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_followup ON leads(tenant_id, next_followup_at) WHERE next_followup_at IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_fbclid ON leads(tenant_id, fbclid) WHERE fbclid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_leads_email ON leads(tenant_id, email) WHERE email IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_custom_fields ON leads USING GIN(custom_fields);

ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS leads_tenant_isolation ON leads;
CREATE POLICY leads_tenant_isolation ON leads
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            current_setting('app.is_admin', true) = 'true'
            OR owner_user_id = current_setting('app.current_user_id', true)::UUID
            OR owner_user_id IN (
                SELECT id FROM users
                WHERE tenant_id = leads.tenant_id
                AND deleted_at IS NULL
                AND path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID)
            )
        )
    );

CREATE TABLE IF NOT EXISTS lead_stage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    from_stage_id UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    to_stage_id UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lead_stage_history_lead ON lead_stage_history(lead_id, changed_at DESC);

ALTER TABLE lead_stage_history ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS lead_stage_history_tenant_isolation ON lead_stage_history;
CREATE POLICY lead_stage_history_tenant_isolation ON lead_stage_history
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Auto-assignment rules: round-robin per role in subtree, or skill match.
CREATE TABLE IF NOT EXISTS lead_assignment_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    strategy TEXT NOT NULL DEFAULT 'round_robin' CHECK (strategy IN (
        'round_robin', 'least_loaded', 'skill_match', 'manual'
    )),
    target_role TEXT,                -- assignee role filter
    source_filter JSONB DEFAULT '{}'::jsonb, -- e.g. {"source": ["facebook"]}
    min_score DECIMAL(5,2),          -- only assign if lead.score >= min_score
    is_active BOOLEAN DEFAULT true,
    last_assigned_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    total_assigned INT DEFAULT 0,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lead_assign_rules_active ON lead_assignment_rules(tenant_id) WHERE is_active = true;

ALTER TABLE lead_assignment_rules ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS lead_assignment_rules_tenant_isolation ON lead_assignment_rules;
CREATE POLICY lead_assignment_rules_tenant_isolation ON lead_assignment_rules
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND current_setting('app.is_admin', true) = 'true'
    );

-- Seed default pipeline (function-defined to keep migration idempotent).
CREATE OR REPLACE FUNCTION seed_default_pipeline_for_tenant(p_tenant UUID)
RETURNS void AS $$
DECLARE
    pid UUID;
BEGIN
    -- Skip if tenant already has any pipeline
    IF EXISTS (SELECT 1 FROM pipelines WHERE tenant_id = p_tenant AND deleted_at IS NULL) THEN
        RETURN;
    END IF;
    INSERT INTO pipelines (tenant_id, name, description, is_default)
    VALUES (p_tenant, 'Default Sales Pipeline', 'Default 7-stage sales pipeline', true)
    RETURNING id INTO pid;

    INSERT INTO pipeline_stages (pipeline_id, tenant_id, name, code, color, position, probability, is_won, is_lost) VALUES
        (pid, p_tenant, 'New',         'NEW',         '#3b82f6', 1, 10, false, false),
        (pid, p_tenant, 'Contacted',   'CONTACTED',   '#06b6d4', 2, 25, false, false),
        (pid, p_tenant, 'Qualified',   'QUALIFIED',   '#10b981', 3, 50, false, false),
        (pid, p_tenant, 'Proposal',    'PROPOSAL',    '#f59e0b', 4, 70, false, false),
        (pid, p_tenant, 'Negotiation', 'NEGOTIATION', '#f97316', 5, 85, false, false),
        (pid, p_tenant, 'Won',         'WON',         '#22c55e', 6, 100, true,  false),
        (pid, p_tenant, 'Lost',        'LOST',        '#ef4444', 7, 0,   false, true);
END;
$$ LANGUAGE plpgsql;

-- Activity extension: link to lead_id (activity can refer to contact OR lead)
ALTER TABLE activities ADD COLUMN IF NOT EXISTS lead_id UUID REFERENCES leads(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_activities_lead ON activities(tenant_id, lead_id) WHERE lead_id IS NOT NULL;
