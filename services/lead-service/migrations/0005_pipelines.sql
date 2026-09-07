-- Migration 0005: Pipelines and Stages
-- =============================================================================

CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_pipelines_tenant ON pipelines(tenant_id);

ALTER TABLE pipelines ENABLE ROW LEVEL SECURITY;

CREATE POLICY pipelines_tenant_isolation ON pipelines
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE TABLE IF NOT EXISTS pipeline_stages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    order_index INT NOT NULL DEFAULT 0,
    probability INT NOT NULL DEFAULT 0 CHECK (probability >= 0 AND probability <= 100),
    color TEXT DEFAULT '#6366f1',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(pipeline_id, order_index)
);

CREATE INDEX idx_pipeline_stages_pipeline ON pipeline_stages(pipeline_id, order_index);

ALTER TABLE pipeline_stages ENABLE ROW LEVEL SECURITY;

CREATE POLICY pipeline_stages_tenant_isolation ON pipeline_stages
    USING (pipeline_id IN (
        SELECT id FROM pipelines WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID
    ));

-- Lead stage history
CREATE TABLE IF NOT EXISTS lead_stage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    from_stage TEXT,
    to_stage TEXT NOT NULL,
    pipeline_stage_id UUID,
    changed_by UUID,
    notes TEXT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lead_stage_history_lead ON lead_stage_history(lead_id, changed_at DESC);

ALTER TABLE lead_stage_history ENABLE ROW LEVEL SECURITY;

CREATE POLICY lead_stage_history_tenant_isolation ON lead_stage_history
    USING (lead_id IN (
        SELECT id FROM leads WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID
    ));
