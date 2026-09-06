-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0008 — Dynamic model schema tables
-- ============================================================
-- Each (tenant, entity_code) gets:
--   * row in  workflow.entity_definitions      -- schema (JSON)
--   * rows in workflow.entity_field_defs       -- column-level typed metadata
--   * rows in workflow.entity_workflows        -- automation pipelines
-- Actual record data lives in MongoDB
-- (rinco.dynamic_records) but a sparse Postgres mirror is kept
-- so RLS / SQL queries / dashboards work.
-- ============================================================

CREATE TABLE IF NOT EXISTS workflow.entity_definitions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    entity_code       CITEXT NOT NULL,            -- 'real_estate', 'bds_property'
    display_name      TEXT NOT NULL,
    icon              TEXT,
    description       TEXT,
    fields            JSONB NOT NULL DEFAULT '[]'::jsonb,    -- JSON Schema array
    workflows         JSONB NOT NULL DEFAULT '[]'::jsonb,
    indexes           JSONB NOT NULL DEFAULT '[]'::jsonb,
    version           INT NOT NULL DEFAULT 1,
    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_by        UUID REFERENCES auth.users(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, entity_code)
);
CREATE INDEX IF NOT EXISTS idx_edef_tenant  ON workflow.entity_definitions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_edef_active  ON workflow.entity_definitions(tenant_id, is_active);

CREATE TRIGGER trg_edef_updated_at
    BEFORE UPDATE ON workflow.entity_definitions
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- Column-level typed field definitions (analytics / validation)
CREATE TABLE IF NOT EXISTS workflow.entity_field_defs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    entity_type       CITEXT NOT NULL,             -- 'lead', 'real_estate', ...
    name              TEXT NOT NULL,
    label             TEXT,
    field_type        TEXT NOT NULL
                      CHECK (field_type IN (
                          'string','text','int','bigint','float','bool','date',
                          'datetime','enum','json','uuid','ref','file','formula'
                      )),
    is_required       BOOLEAN NOT NULL DEFAULT false,
    is_searchable     BOOLEAN NOT NULL DEFAULT false,
    default_value     JSONB,
    options           JSONB NOT NULL DEFAULT '[]'::jsonb,
    validation_rules  JSONB NOT NULL DEFAULT '{}'::jsonb,
    display_order     INT  NOT NULL DEFAULT 0,
    is_system         BOOLEAN NOT NULL DEFAULT false,
    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, entity_type, name)
);
CREATE INDEX IF NOT EXISTS idx_efd_tenant  ON workflow.entity_field_defs(tenant_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_efd_active  ON workflow.entity_field_defs(tenant_id, entity_type, is_active);

CREATE TRIGGER trg_efd_updated_at
    BEFORE UPDATE ON workflow.entity_field_defs
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- Workflow automation pipelines
CREATE TABLE IF NOT EXISTS workflow.workflows (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    description       TEXT,
    trigger_entity    TEXT NOT NULL,
    trigger_event     TEXT NOT NULL
                      CHECK (trigger_event IN ('create','update','delete','custom','schedule')),
    trigger_config    JSONB NOT NULL DEFAULT '{}'::jsonb,
    conditions        JSONB NOT NULL DEFAULT '[]'::jsonb,
    actions           JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_wf_tenant  ON workflow.workflows(tenant_id);
CREATE INDEX IF NOT EXISTS idx_wf_active  ON workflow.workflows(tenant_id, is_active);

CREATE TRIGGER trg_workflows_updated_at
    BEFORE UPDATE ON workflow.workflows
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- Workflow execution log
CREATE TABLE IF NOT EXISTS workflow.workflow_executions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id       UUID NOT NULL REFERENCES workflow.workflows(id) ON DELETE CASCADE,
    tenant_id         UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    trigger_data      JSONB NOT NULL DEFAULT '{}'::jsonb,
    status            TEXT NOT NULL DEFAULT 'running'
                      CHECK (status IN ('running','completed','failed','cancelled')),
    started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at      TIMESTAMPTZ,
    error_message     TEXT,
    execution_log     JSONB NOT NULL DEFAULT '[]'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_wfe_workflow ON workflow.workflow_executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_wfe_status   ON workflow.workflow_executions(status);
CREATE INDEX IF NOT EXISTS idx_wfe_started  ON workflow.workflow_executions(started_at DESC);

-- Postgres mirror of Mongo dynamic_records (sparse projection for
-- SQL queries: created_at, owner, filtered columns)
CREATE TABLE IF NOT EXISTS workflow.dynamic_records (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    entity_code       CITEXT NOT NULL,
    owner_user_id     UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    mongo_doc_id      TEXT UNIQUE,            -- pointer to Mongo
    data              JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_dr_data_gin         ON workflow.dynamic_records USING GIN(data);
CREATE INDEX IF NOT EXISTS idx_dr_tenant_entity    ON workflow.dynamic_records(tenant_id, entity_code);
CREATE INDEX IF NOT EXISTS idx_dr_owner            ON workflow.dynamic_records(tenant_id, owner_user_id);
CREATE INDEX IF NOT EXISTS idx_dr_email            ON workflow.dynamic_records((data->>'email'));
CREATE INDEX IF NOT EXISTS idx_dr_phone            ON workflow.dynamic_records((data->>'phone'));

CREATE TRIGGER trg_dynamic_records_updated_at
    BEFORE UPDATE ON workflow.dynamic_records
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- ============================================================
-- RLS
-- ============================================================
ALTER TABLE workflow.entity_definitions    ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow.entity_definitions    FORCE  ROW LEVEL SECURITY;
ALTER TABLE workflow.entity_field_defs     ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow.entity_field_defs     FORCE  ROW LEVEL SECURITY;
ALTER TABLE workflow.workflows             ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow.workflows             FORCE  ROW LEVEL SECURITY;
ALTER TABLE workflow.workflow_executions   ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow.workflow_executions   FORCE  ROW LEVEL SECURITY;
ALTER TABLE workflow.dynamic_records       ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow.dynamic_records       FORCE  ROW LEVEL SECURITY;

CREATE POLICY edef_tenant ON workflow.entity_definitions
    FOR ALL USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    ) WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY efd_tenant ON workflow.entity_field_defs
    FOR ALL USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    ) WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY workflows_tenant ON workflow.workflows
    FOR ALL USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    ) WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY wfe_tenant ON workflow.workflow_executions
    FOR ALL USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    ) WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY dr_tenant ON workflow.dynamic_records
    FOR ALL USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    ) WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS workflow.dynamic_records        CASCADE;
DROP TABLE IF EXISTS workflow.workflow_executions    CASCADE;
DROP TABLE IF EXISTS workflow.workflows              CASCADE;
DROP TABLE IF EXISTS workflow.entity_field_defs      CASCADE;
DROP TABLE IF EXISTS workflow.entity_definitions     CASCADE;
-- +goose StatementEnd
