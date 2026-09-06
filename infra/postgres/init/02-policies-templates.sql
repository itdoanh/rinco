-- ============================================================
-- RINCO RLS Policy Templates
-- ============================================================
-- Reusable template policies. Apply selectively to each table
-- via `migrations/sql/000X_*.sql`. These statements are commented
-- out and used as REFERENCES — copy/paste the policy block into
-- each table migration and adjust the table name if needed.
-- ============================================================

-- -------------------------------------------------------------
-- TEMPLATE 1: Tenant isolation (mandatory, every tenant_id col)
-- -------------------------------------------------------------
-- ALTER TABLE <schema>.<table> ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE <schema>.<table> FORCE  ROW LEVEL SECURITY;
--
-- CREATE POLICY <table>_tenant_isolation ON <schema>.<table>
--   FOR ALL
--   USING (
--        app.bypass_rls_check()
--     OR tenant_id = app.current_tenant_id()
--   )
--   WITH CHECK (
--        app.bypass_rls_check()
--     OR tenant_id = app.current_tenant_id()
--   );

-- -------------------------------------------------------------
-- TEMPLATE 2: User subtree (CRM). Visible if owner is in the
-- caller's org subtree (LTREE).
-- -------------------------------------------------------------
-- CREATE POLICY <table>_user_subtree ON <schema>.<table>
--   FOR ALL
--   USING (
--        app.bypass_rls_check()
--     OR owner_user_id = app.current_user_id()
--     OR owner_user_id IN (
--         SELECT id FROM auth.users
--         WHERE  path <@ (
--             SELECT path FROM auth.users
--             WHERE id = app.current_user_id()
--         )
--     )
--   );

-- -------------------------------------------------------------
-- TEMPLATE 3: Tenant-scoped SELECT, super-admin bypass
-- -------------------------------------------------------------
-- CREATE POLICY <table>_super_admin_select ON <schema>.<table>
--   FOR SELECT
--   USING ( app.bypass_rls_check() );

-- -------------------------------------------------------------
-- TEMPLATE 4: Read-only for non-admins (audit log)
-- -------------------------------------------------------------
-- CREATE POLICY <table>_audit_read ON audit.audit_logs
--   FOR SELECT
--   USING (
--        app.bypass_rls_check()
--     OR ( tenant_id = app.current_tenant_id()
--          AND app.is_super_admin() = FALSE )
--   );
-- CREATE POLICY <table>_audit_insert ON audit.audit_logs
--   FOR INSERT
--   WITH CHECK ( app.bypass_rls_check() OR tenant_id = app.current_tenant_id() );

-- -------------------------------------------------------------
-- EXAMPLE — applied to a real table:
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS app._rls_template_examples (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    UUID NOT NULL,
  owner_id     UUID,
  payload      JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE app._rls_template_examples ENABLE ROW LEVEL SECURITY;
ALTER TABLE app._rls_template_examples FORCE  ROW LEVEL SECURITY;

CREATE POLICY _rls_template_examples_tenant ON app._rls_template_examples
  FOR ALL
  USING (
       app.bypass_rls_check()
    OR tenant_id = app.current_tenant_id()
  )
  WITH CHECK (
       app.bypass_rls_check()
    OR tenant_id = app.current_tenant_id()
  );

DROP TABLE IF EXISTS app._rls_template_examples;
