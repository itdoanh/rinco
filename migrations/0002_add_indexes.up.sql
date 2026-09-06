-- Migration 0002: Add Performance Indexes
-- Creates additional indexes for better query performance

BEGIN;

-- Full-text search on leads
CREATE INDEX idx_leads_fulltext ON leads.leads 
    USING GIN(to_tsvector('simple', 
        COALESCE(full_name, '') || ' ' || 
        COALESCE(email, '') || ' ' || 
        COALESCE(company, '')
    ));

-- Lead activities timestamp indexes
CREATE INDEX idx_lead_activities_created ON leads.lead_activities(created_at DESC);

-- Composite index for common lead queries
CREATE INDEX idx_leads_tenant_status ON leads.leads(tenant_id, status, score DESC) 
    WHERE deleted_at IS NULL;

CREATE INDEX idx_leads_tenant_stage ON leads.leads(tenant_id, stage) 
    WHERE deleted_at IS NULL AND status = 'active';

-- Index for tenant lookups by various identifiers
CREATE INDEX idx_tenants_slug_active ON tenant.tenants(slug) 
    WHERE status = 'active' AND deleted_at IS NULL;

CREATE INDEX idx_tenant_sites_active ON tenant.tenant_sites(tenant_id) 
    WHERE is_active = true;

-- Index for audit log queries
CREATE INDEX idx_audit_logs_action ON audit.audit_logs(action, created_at DESC);

-- Index for user hierarchy queries
CREATE INDEX idx_user_hierarchy_tenant_path ON crm.user_hierarchy(tenant_id, path);

-- Statistics target
ALTER TABLE leads.leads SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE audit.audit_logs SET (autovacuum_vacuum_scale_factor = 0.05);

COMMIT;
