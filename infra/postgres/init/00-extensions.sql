-- ============================================================
-- PostgreSQL Extensions (RINCO)
-- ============================================================
-- Required extensions for RINCO platform.
-- Loaded first (alphabetical: 00-) so they exist before all
-- subsequent init scripts reference them.
-- ============================================================

-- Cryptographically-strong UUIDs (Gen5+)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA ext;

-- Crypto functions (gen_random_uuid, digest, etc.)
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA ext;

-- Hierarchical tree labels for CRM org chart
CREATE EXTENSION IF NOT EXISTS ltree WITH SCHEMA ext;

-- Trigram fuzzy search
CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA ext;

-- Case-insensitive text
CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA ext;

-- Composite B-tree GiST indexes
CREATE EXTENSION IF NOT EXISTS btree_gist WITH SCHEMA ext;

-- Vector similarity search (pgvector)
CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA ext;

-- Query statistics
CREATE EXTENSION IF NOT EXISTS pg_stat_statements WITH SCHEMA ext;

-- Background workers, scheduled jobs
CREATE EXTENSION IF NOT EXISTS pg_cron WITH SCHEMA ext;

-- Schemas
CREATE SCHEMA IF NOT EXISTS app;
CREATE SCHEMA IF NOT EXISTS tenant;
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS crm;
CREATE SCHEMA IF NOT EXISTS leads;
CREATE SCHEMA IF NOT EXISTS workflow;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS billing;
CREATE SCHEMA IF NOT EXISTS public;

-- Set the search_path for convenience
ALTER DATABASE rinco SET search_path TO app, public, tenant, auth, crm, leads, workflow, audit, billing;

-- Convenience: extension visibility
GRANT USAGE ON SCHEMA ext TO PUBLIC;
