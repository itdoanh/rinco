-- ============================================================
-- RINCO Comprehensive Demo Seed Master File (Loop 202)
-- Runs all seed files in dependency order.
-- Idempotent: every INSERT uses ON CONFLICT to ensure re-runnability.
-- Creates a 'rinco-root' sentinel after completion (SECTION 99).
-- ============================================================
DO $$
BEGIN
  RAISE NOTICE '===========================================';
  RAISE NOTICE 'RINCO Demo Seed — Loop 202 (comprehensive)';
  RAISE NOTICE '===========================================';
END$$;

\set ON_ERROR_STOP off

-- Service schemas check
DO $$
BEGIN
  PERFORM 1;
  RAISE NOTICE '--- Creating missing schemas (auth, crm, leads, workflow, audit, billing, tenant, notification, email, landing, observability, recordings) ---';
END$$;
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS crm;
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

\echo '--- [1/11] Tenants + domains + branding ---'
\i 01_tenants.sql

\echo '--- [2/11] Users + LTREE hierarchy ---'
\i 02_users.sql

\echo '--- [3/11] CRM tree (departments, sources, pipelines, companies) ---'
\i 03_crm_tree.sql

\echo '--- [4/11] Leads (200+ realistic Vietnamese) ---'
\i 04_leads.sql

\echo '--- [5/11] Deals + Activities + Notes ---'
\i 05_deals_activities.sql

\echo '--- [6/11] Tags + Custom Field Definitions ---'
\i 06_tags_custom_fields.sql

\echo '--- [7/11] Workflows + Audit Logs ---'
\i 07_workflows_audit.sql

\echo '--- [8/11] Billing + Subscriptions ---'
\i 08_billing.sql

\echo '--- [9/11] Notifications + Chat + API Keys ---'
\i 09_notifications_chat.sql

\echo '--- [10/11] Email/Landing/MetaCapi/Observability/Recordings ---'
\i 10_other_services.sql

\echo '--- [11/11] Legacy service tables (lead-service, crm-service, dynamic_model) ---'
\i 11_legacy_services.sql

-- ============================================================
-- SECTION 99: Mark seed as completed (sentinel tenant row)
-- ============================================================
INSERT INTO tenant.tenants (id, slug, name, plan, status)
VALUES (
  '__seed_sentinel_202__',
  '__seed_completed__',
  'Loop 202 Demo Seed Sentinel (do not delete)',
  'starter',
  'archived'
)
ON CONFLICT (slug) DO NOTHING;

-- ============================================================
-- WS-B expansion (additional demo data — idempotent)
-- ============================================================
\echo '--- [12/N] WS-B Loop expansion seeds (extra leads/deals/users/etc) ---'
\i expansion/01_leads_extra.sql
\i expansion/02_deals_activities_extra.sql
\i expansion/03_users_crm_tree_extra.sql
\i expansion/04_workflows_audit_extra.sql
\i expansion/05_notifications_chat_extra.sql

\echo ''
\echo '==========================================='
\echo 'RINCO Demo Seed complete. Summary:'
SELECT
  'tenants' AS table_name, COUNT(*) AS cnt FROM tenant.tenants WHERE deleted_at IS NULL AND slug NOT LIKE '__%'
UNION ALL SELECT 'users', COUNT(*) FROM auth.users WHERE deleted_at IS NULL
UNION ALL SELECT 'leads (leads.leads)', COUNT(*) FROM leads.leads WHERE deleted_at IS NULL
UNION ALL SELECT 'deals (crm.deals)', COUNT(*) FROM crm.deals WHERE deleted_at IS NULL
UNION ALL SELECT 'deals (crm-service deals)', COUNT(*) FROM deals WHERE deleted_at IS NULL
UNION ALL SELECT 'activities (crm)', COUNT(*) FROM crm.activities
UNION ALL SELECT 'companies', COUNT(*) FROM companies WHERE deleted_at IS NULL
UNION ALL SELECT 'contacts (crm-service)', COUNT(*) FROM contacts WHERE deleted_at IS NULL
UNION ALL SELECT 'tags', COUNT(*) FROM tags WHERE deleted_at IS NULL
UNION ALL SELECT 'custom_fields', COUNT(*) FROM custom_fields
UNION ALL SELECT 'workflow.entity_field_defs', COUNT(*) FROM workflow.entity_field_defs WHERE is_active
UNION ALL SELECT 'workflow.workflows', COUNT(*) FROM workflow.workflows WHERE is_active
UNION ALL SELECT 'audit.audit_logs', COUNT(*) FROM audit.audit_logs
UNION ALL SELECT 'audit.api_requests', COUNT(*) FROM audit.api_requests
UNION ALL SELECT 'subscriptions', COUNT(*) FROM billing.subscriptions
UNION ALL SELECT 'invoices', COUNT(*) FROM billing.invoices
UNION ALL SELECT 'payment_methods', COUNT(*) FROM billing.payment_methods
UNION ALL SELECT 'notifications', COUNT(*) FROM notification.notifications
UNION ALL SELECT 'push_subscriptions', COUNT(*) FROM notification.push_subscriptions
UNION ALL SELECT 'chat_channels', COUNT(*) FROM crm.chat_channels
UNION ALL SELECT 'chat_messages', COUNT(*) FROM crm.chat_messages
UNION ALL SELECT 'meeting_rooms', COUNT(*) FROM crm.meeting_rooms
UNION ALL SELECT 'api_keys', COUNT(*) FROM auth.api_keys
UNION ALL SELECT 'email_templates', COUNT(*) FROM email.email_templates
UNION ALL SELECT 'landing_pages (crm)', COUNT(*) FROM crm.landing_pages
UNION ALL SELECT 'landing_pages (landing)', COUNT(*) FROM landing.landing_pages
UNION ALL SELECT 'fb_pixel_configs', COUNT(*) FROM landing.fb_pixel_configs
UNION ALL SELECT 'recordings', COUNT(*) FROM recordings
UNION ALL SELECT 'departments', COUNT(*) FROM crm.departments
UNION ALL SELECT 'pipelines (crm)', COUNT(*) FROM crm.pipelines
UNION ALL SELECT 'pipelines (lead-service)', COUNT(*) FROM pipelines
UNION ALL SELECT 'tenant_domains', COUNT(*) FROM tenant.tenant_domains
UNION ALL SELECT 'tenant_sites', COUNT(*) FROM tenant.tenant_sites
UNION ALL SELECT 'tenant_settings', COUNT(*) FROM tenant.tenant_settings
UNION ALL SELECT 'dynamic_records', COUNT(*) FROM workflow.dynamic_records
ORDER BY table_name;
\echo '==========================================='
