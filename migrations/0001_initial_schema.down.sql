-- Migration 0001 Rollback: Drop Initial Schema

BEGIN;

DROP POLICY IF EXISTS tenant_isolation_users ON auth.users;
DROP POLICY IF EXISTS tenant_isolation_hierarchy ON crm.user_hierarchy;
DROP POLICY IF EXISTS tenant_isolation_leads ON leads.leads;
DROP POLICY IF EXISTS tenant_isolation_lead_activities ON leads.lead_activities;
DROP POLICY IF EXISTS tenant_isolation_entity_fields ON crm.entity_field_defs;
DROP POLICY IF EXISTS tenant_isolation_workflows ON workflow.workflows;
DROP POLICY IF EXISTS tenant_isolation_audit ON audit.audit_logs;

DROP TABLE IF EXISTS audit.audit_logs CASCADE;
DROP TABLE IF EXISTS workflow.workflow_executions CASCADE;
DROP TABLE IF EXISTS workflow.workflows CASCADE;
DROP TABLE IF EXISTS crm.entity_field_defs CASCADE;
DROP TABLE IF EXISTS leads.lead_activities CASCADE;
DROP TABLE IF EXISTS leads.leads CASCADE;
DROP TABLE IF EXISTS leads.lead_pipelines CASCADE;
DROP TABLE IF EXISTS crm.user_hierarchy CASCADE;
DROP TABLE IF EXISTS crm.departments CASCADE;
DROP TABLE IF EXISTS auth.fido2_credentials CASCADE;
DROP TABLE IF EXISTS auth.refresh_tokens CASCADE;
DROP TABLE IF EXISTS auth.users CASCADE;
DROP TABLE IF EXISTS tenant.tenant_sites CASCADE;
DROP TABLE IF EXISTS tenant.tenants CASCADE;

DROP SCHEMA IF EXISTS audit CASCADE;
DROP SCHEMA IF EXISTS workflow CASCADE;
DROP SCHEMA IF EXISTS leads CASCADE;
DROP SCHEMA IF EXISTS crm CASCADE;
DROP SCHEMA IF EXISTS auth CASCADE;
DROP SCHEMA IF EXISTS tenant CASCADE;

COMMIT;
