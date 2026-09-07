-- Migration 0004: RLS policies
-- =============================================================================

-- Lead sources RLS
CREATE POLICY lead_sources_tenant_isolation ON lead_sources
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
    );

-- Pipelines RLS
CREATE POLICY pipelines_tenant_isolation ON pipelines
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
    );

-- Pipeline stages RLS
CREATE POLICY pipeline_stages_tenant_isolation ON pipeline_stages
    FOR ALL
    USING (
        pipeline_id IN (
            SELECT id FROM pipelines 
            WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID
        )
    );

-- Lead notes RLS
CREATE POLICY lead_notes_tenant_isolation ON lead_notes
    FOR ALL
    USING (
        lead_id IN (
            SELECT id FROM leads 
            WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID
        )
    );

-- Lead activities RLS
CREATE POLICY lead_activities_tenant_isolation ON lead_activities
    FOR ALL
    USING (
        lead_id IN (
            SELECT id FROM leads 
            WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID
        )
    );

-- Lead assignments RLS
CREATE POLICY lead_assignments_tenant_isolation ON lead_assignments
    FOR ALL
    USING (
        lead_id IN (
            SELECT id FROM leads 
            WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID
        )
    );

-- Lead stage history RLS
CREATE POLICY lead_stage_history_tenant_isolation ON lead_stage_history
    FOR ALL
    USING (
        lead_id IN (
            SELECT id FROM leads 
            WHERE tenant_id = current_setting('app.current_tenant_id', true)::UUID
        )
    );
