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

-- Add foreign key constraints that were deferred
DO $$
BEGIN
    -- Add foreign key to contacts.owner_user_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'contacts_owner_user_id_fkey'
    ) THEN
        ALTER TABLE contacts ADD CONSTRAINT contacts_owner_user_id_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    
    -- Add foreign key to deals.owner_user_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'deals_owner_user_id_fkey'
    ) THEN
        ALTER TABLE deals ADD CONSTRAINT deals_owner_user_id_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    
    -- Add foreign key to deals.changed_by
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'deal_stage_history_changed_by_fkey'
    ) THEN
        ALTER TABLE deal_stage_history ADD CONSTRAINT deal_stage_history_changed_by_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    
    -- Add foreign key to activities.owner_user_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'activities_owner_user_id_fkey'
    ) THEN
        ALTER TABLE activities ADD CONSTRAINT activities_owner_user_id_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    
    -- Add foreign key to notes.author_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'notes_author_id_fkey'
    ) THEN
        ALTER TABLE notes ADD CONSTRAINT notes_author_id_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

-- Add self-referential constraint for parent_id
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'users_parent_id_fkey'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_parent_id_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;
