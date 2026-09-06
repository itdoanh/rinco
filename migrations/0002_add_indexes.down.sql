-- Migration 0002 Rollback: Drop Performance Indexes

BEGIN;

DROP INDEX IF EXISTS leads.idx_leads_fulltext;
DROP INDEX IF EXISTS leads.idx_lead_activities_created;
DROP INDEX IF EXISTS leads.idx_leads_tenant_status;
DROP INDEX IF EXISTS leads.idx_leads_tenant_stage;
DROP INDEX IF EXISTS tenant.idx_tenants_slug_active;
DROP INDEX IF EXISTS tenant.idx_tenant_sites_active;
DROP INDEX IF EXISTS audit.idx_audit_logs_action;
DROP INDEX IF EXISTS crm.idx_user_hierarchy_tenant_path;

COMMIT;
