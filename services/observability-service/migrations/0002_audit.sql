-- 0002_audit.sql
-- Adds row-level security policies for audit_logs.

ALTER TABLE observability.audit_logs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS audit_logs_tenant ON observability.audit_logs;
CREATE POLICY audit_logs_tenant ON observability.audit_logs
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        OR current_setting('app.is_admin', true) = 'true'
    );