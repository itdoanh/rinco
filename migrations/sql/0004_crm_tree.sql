-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0004 — CRM tree (LTREE materialized path)
-- ============================================================
-- Implements org-chart / reporting hierarchy. The `path` column
-- stores `/<uuidA>/<uuidB>/.../self`, enabling fast sub-tree
-- queries via GIST index.
-- ============================================================

CREATE TABLE IF NOT EXISTS crm.departments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    code          TEXT,
    description   TEXT,
    manager_id    UUID REFERENCES auth.users(id),
    parent_id     UUID REFERENCES crm.departments(id),
    path          LTREE NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name, parent_id)
);
CREATE INDEX IF NOT EXISTS idx_departments_tenant ON crm.departments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_departments_path   ON crm.departments USING GIST(path);

CREATE TRIGGER trg_departments_updated_at
    BEFORE UPDATE ON crm.departments
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- CRM user hierarchy — separate from auth.users; one row per
-- (tenant, user) so a user can switch teams etc.
CREATE TABLE IF NOT EXISTS crm.user_hierarchy (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES auth.users(id)   ON DELETE CASCADE,
    department_id UUID REFERENCES crm.departments(id),
    manager_id    UUID REFERENCES auth.users(id),
    parent_id     UUID REFERENCES crm.user_hierarchy(id),
    path          LTREE NOT NULL,
    depth         INT   NOT NULL DEFAULT 0,
    position      TEXT,
    is_manager    BOOLEAN NOT NULL DEFAULT false,
    start_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date      DATE,
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_user_hierarchy_path         ON crm.user_hierarchy USING GIST(path);
CREATE INDEX IF NOT EXISTS idx_user_hierarchy_tenant_user  ON crm.user_hierarchy(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_user_hierarchy_manager     ON crm.user_hierarchy(manager_id) WHERE manager_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_hierarchy_parent      ON crm.user_hierarchy(parent_id);

CREATE TRIGGER trg_user_hierarchy_updated_at
    BEFORE UPDATE ON crm.user_hierarchy
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- ============================================================
-- RLS
-- ============================================================
ALTER TABLE crm.departments    ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.departments    FORCE  ROW LEVEL SECURITY;
ALTER TABLE crm.user_hierarchy ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.user_hierarchy FORCE  ROW LEVEL SECURITY;

CREATE POLICY departments_tenant ON crm.departments
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY user_hierarchy_tenant ON crm.user_hierarchy
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

-- Additional SELECT filter: only see rows in caller's subtree
CREATE POLICY user_hierarchy_subtree_select ON crm.user_hierarchy
    FOR SELECT
    USING (
        app.bypass_rls_check()
        OR path <@ (
            SELECT path FROM crm.user_hierarchy
            WHERE user_id = app.current_user_id()
        )
        OR user_id = app.current_user_id()
    );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS crm.user_hierarchy CASCADE;
DROP TABLE IF EXISTS crm.departments    CASCADE;
-- +goose StatementEnd
