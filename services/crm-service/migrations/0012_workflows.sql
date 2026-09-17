-- =============================================================================
-- Migration 0012: Workflow engine (triggers, conditions, actions)
-- =============================================================================
-- Loop 203 — CRM Tree enterprise completion.
--
-- Workflow:
--   trigger.type : 'stage_change' | 'field_change' | 'time_based' | 'manual'
--   condition    : JSON DSL ({"all":[...],"any":[...]}) đơn giản
--   actions      : JSON array [{type:"assign_user",...}, {type:"create_activity",...}, ...]
-- execution     : log mỗi lần workflow chạy (audit + retry).
-- =============================================================================

CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    trigger_type TEXT NOT NULL CHECK (trigger_type IN (
        'stage_change', 'field_change', 'time_based', 'manual', 'lead_created', 'lead_assigned'
    )),
    trigger_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    conditions JSONB NOT NULL DEFAULT '{}'::jsonb,
    actions JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN DEFAULT true,
    run_count INT DEFAULT 0,
    last_run_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_workflows_tenant_active ON workflows(tenant_id) WHERE is_active = true AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_workflows_trigger ON workflows(tenant_id, trigger_type) WHERE is_active = true AND deleted_at IS NULL;

ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS workflows_tenant_isolation ON workflows;
CREATE POLICY workflows_tenant_isolation ON workflows
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            current_setting('app.is_admin', true) = 'true'
            OR EXISTS (
                SELECT 1 FROM users
                WHERE id = current_setting('app.current_user_id', true)::UUID
                AND tenant_id = workflows.tenant_id
                AND deleted_at IS NULL
                AND (role IN ('owner', 'admin', 'manager'))
            )
        )
    );

CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,         -- 'lead', 'deal', 'contact'
    entity_id UUID NOT NULL,
    triggered_by TEXT NOT NULL,         -- 'event' | 'manual' | 'system'
    status TEXT NOT NULL DEFAULT 'success' CHECK (status IN ('success', 'failed', 'partial', 'skipped')),
    inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    actions_executed JSONB NOT NULL DEFAULT '[]'::jsonb,
    error TEXT,
    execution_ms INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workflow_exec_workflow ON workflow_executions(workflow_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_exec_entity ON workflow_executions(entity_type, entity_id);

ALTER TABLE workflow_executions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS workflow_exec_tenant_isolation ON workflow_executions;
CREATE POLICY workflow_exec_tenant_isolation ON workflow_executions
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
