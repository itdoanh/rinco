-- ============================================================
-- RINCO Row-Level Security (RLS) Setup
-- ============================================================
-- GUC (Grand Unified Configuration) variables:
--   app.current_user_id     - current authenticated user UUID
--   app.current_tenant_id   - current tenant UUID
--   app.is_super_admin      - 'true' if super-admin bypass
-- ============================================================

-- ============================================================
-- HELPER FUNCTIONS
-- ============================================================

-- Returns the current authenticated user UUID (or NULL).
CREATE OR REPLACE FUNCTION app.current_user_id()
RETURNS UUID AS $$
BEGIN
  RETURN NULLIF(current_setting('app.current_user_id', true), '')::UUID;
EXCEPTION WHEN OTHERS THEN
  RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE PARALLEL SAFE;

-- Returns the current tenant UUID (or NULL).
CREATE OR REPLACE FUNCTION app.current_tenant_id()
RETURNS UUID AS $$
BEGIN
  RETURN NULLIF(current_setting('app.current_tenant_id', true), '')::UUID;
END;
$$ LANGUAGE plpgsql STABLE PARALLEL SAFE;

-- Returns TRUE if current request is from a super-admin.
CREATE OR REPLACE FUNCTION app.is_super_admin()
RETURNS BOOLEAN AS $$
BEGIN
  RETURN coalesce(current_setting('app.is_super_admin', true), 'false') = 'true';
END;
$$ LANGUAGE plpgsql STABLE PARALLEL SAFE;

-- Internal: returns NULL if any required context is missing
-- (policies that mandate tenant context reject those rows).
CREATE OR REPLACE FUNCTION app.bypass_rls_check()
RETURNS BOOLEAN AS $$
BEGIN
  -- Super-admins always bypass RLS (FORCE RLS still applies).
  IF app.is_super_admin() THEN
    RETURN TRUE;
  END IF;
  RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE PARALLEL SAFE;

-- Wrap a function/transaction with RLS context.
-- Usage: BEGIN; SELECT app.set_rls_context(...); ... COMMIT;
CREATE OR REPLACE FUNCTION app.set_rls_context(
  p_tenant_id  UUID,
  p_user_id    UUID,
  p_is_super   BOOLEAN DEFAULT FALSE
) RETURNS VOID AS $$
BEGIN
  PERFORM set_config('app.current_tenant_id', COALESCE(p_tenant_id::text, ''), true);
  PERFORM set_config('app.current_user_id',  COALESCE(p_user_id::text,    ''), true);
  PERFORM set_config('app.is_super_admin',   CASE WHEN p_is_super THEN 'true' ELSE 'false' END, true);
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- DEFAULT DENY ROLE
-- ============================================================
-- The application connects as a non-superuser (e.g. rinco_app).
-- We FORCE row-level security on tenant-scoped tables so they
-- can't accidentally be skipped.
-- ============================================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'rinco_app') THEN
    CREATE ROLE rinco_app LOGIN PASSWORD 'rinco_dev_password';
  END IF;
EXCEPTION WHEN OTHERS THEN
  -- ignore if cannot create
  NULL;
END$$;

GRANT CONNECT ON DATABASE rinco TO rinco_app;
GRANT USAGE ON SCHEMA app, tenant, auth, crm, leads, workflow, audit, billing, public TO rinco_app;
