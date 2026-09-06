-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0001 — extensions & RLS scaffolding
-- ============================================================
-- This migration creates the foundational extensions, schemas, and
-- helper functions required by RLS. Idempotent so it can be safely
-- re-applied on a clean database.
-- ============================================================

-- Extensions (must be created by superuser; will fail silently if missing)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp"     WITH SCHEMA ext;
CREATE EXTENSION IF NOT EXISTS pgcrypto        WITH SCHEMA ext;
CREATE EXTENSION IF NOT EXISTS ltree           WITH SCHEMA ext;
CREATE EXTENSION IF NOT EXISTS pg_trgm          WITH SCHEMA ext;
CREATE EXTENSION IF NOT EXISTS btree_gist      WITH SCHEMA ext;
CREATE EXTENSION IF NOT EXISTS citext          WITH SCHEMA ext;
CREATE EXTENSION IF NOT EXISTS vector          WITH SCHEMA ext;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements WITH SCHEMA ext;

-- Logical schemas
CREATE SCHEMA IF NOT EXISTS app;
CREATE SCHEMA IF NOT EXISTS tenant;
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS crm;
CREATE SCHEMA IF NOT EXISTS leads;
CREATE SCHEMA IF NOT EXISTS workflow;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS billing;

-- ============================================================
-- Helper functions (RLS context)
-- ============================================================
CREATE OR REPLACE FUNCTION app.current_user_id()
RETURNS UUID AS $$
BEGIN
  RETURN NULLIF(current_setting('app.current_user_id', true), '')::UUID;
EXCEPTION WHEN OTHERS THEN
  RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE PARALLEL SAFE;

CREATE OR REPLACE FUNCTION app.current_tenant_id()
RETURNS UUID AS $$
BEGIN
  RETURN NULLIF(current_setting('app.current_tenant_id', true), '')::UUID;
EXCEPTION WHEN OTHERS THEN
  RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE PARALLEL SAFE;

CREATE OR REPLACE FUNCTION app.is_super_admin()
RETURNS BOOLEAN AS $$
BEGIN
  RETURN coalesce(current_setting('app.is_super_admin', true), 'false') = 'true';
END;
$$ LANGUAGE plpgsql STABLE PARALLEL SAFE;

CREATE OR REPLACE FUNCTION app.bypass_rls_check()
RETURNS BOOLEAN AS $$
BEGIN
  IF app.is_super_admin() THEN
    RETURN TRUE;
  END IF;
  RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE PARALLEL SAFE;

CREATE OR REPLACE FUNCTION app.set_rls_context(
  p_tenant_id  UUID,
  p_user_id    UUID,
  p_is_super   BOOLEAN DEFAULT FALSE
) RETURNS VOID AS $$
BEGIN
  PERFORM set_config('app.current_tenant_id', COALESCE(p_tenant_id::text, ''), true);
  PERFORM set_config('app.current_user_id',  COALESCE(p_user_id::text,    ''), true);
  PERFORM set_config('app.is_super_admin',
         CASE WHEN p_is_super THEN 'true' ELSE 'false' END, true);
END;
$$ LANGUAGE plpgsql;

-- Generic set_updated_at trigger function
CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Application role
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'rinco_app') THEN
    CREATE ROLE rinco_app LOGIN PASSWORD 'rinco_dev_password';
  END IF;
END$$;
GRANT CONNECT ON DATABASE rinco TO rinco_app;
GRANT USAGE ON SCHEMA app, tenant, auth, crm, leads, workflow, audit, billing, public TO rinco_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS billing CASCADE;
DROP SCHEMA IF EXISTS audit   CASCADE;
DROP SCHEMA IF EXISTS workflow CASCADE;
DROP SCHEMA IF EXISTS leads   CASCADE;
DROP SCHEMA IF EXISTS crm     CASCADE;
DROP SCHEMA IF EXISTS auth    CASCADE;
DROP SCHEMA IF EXISTS tenant  CASCADE;
DROP SCHEMA IF EXISTS app     CASCADE;
-- +goose StatementEnd
