-- Migration 0008: Users Tree with LTREE
-- =============================================================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    email TEXT NOT NULL,
    full_name TEXT NOT NULL,
    phone TEXT,
    avatar_url TEXT,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN (
        'owner', 'admin', 'manager', 'team_lead', 'member', 'viewer'
    )),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'invited', 'pending')),
    path LTREE NOT NULL,
    depth INT NOT NULL DEFAULT 1,
    parent_id UUID,
    department TEXT,
    position TEXT,
    password_hash TEXT,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, email)
);

-- GIST index for LTREE operations
CREATE INDEX idx_users_path ON users USING GIST (path);
CREATE INDEX idx_users_path_btree ON users USING BTREE (path);
CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_parent ON users(tenant_id, parent_id);
CREATE INDEX idx_users_role ON users(tenant_id, role);
CREATE INDEX idx_users_status ON users(tenant_id, status);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY users_tenant_isolation ON users
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Materialized view for fast subtree lookups
CREATE MATERIALIZED VIEW IF NOT EXISTS users_with_depth AS
SELECT 
    id, tenant_id, email, full_name, role, status, 
    path, depth, parent_id, department
FROM users 
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_uwd_id ON users_with_depth(id);
CREATE INDEX idx_uwd_tenant ON users_with_depth(tenant_id);
CREATE INDEX idx_uwd_path ON users_with_depth USING GIST (path);

-- Self-referential parent_id FK (deferred to migration 0015)
-- users_parent_id_fkey will be added in 0015_users_fk_constraints.sql
