-- 000_init_dynamic_model.sql
-- Schema for the Dynamic Model / Meta-Schema Engine.
-- All tables live in the "model" schema and obey Row Level Security
-- using the values set by the gateway / service middleware:
--     SET LOCAL app.current_tenant_id  = '...'
--     SET LOCAL app.current_user_id    = '...'
--     SET LOCAL app.is_super_admin     = 'true'

CREATE SCHEMA IF NOT EXISTS model;
SET search_path = model, public;

-- ---------------------------------------------------------------------------
-- models: header / metadata about a user-defined entity type.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS model.models (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID         NOT NULL,
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(255) NOT NULL,
    version         INT          NOT NULL DEFAULT 1,
    status          VARCHAR(32)  NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'published', 'archived')),
    description     TEXT,
    created_by      UUID         NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (tenant_id, slug, version)
);

CREATE INDEX IF NOT EXISTS idx_models_tenant        ON model.models (tenant_id);
CREATE INDEX IF NOT EXISTS idx_models_status        ON model.models (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_models_deleted_at    ON model.models (deleted_at);

-- ---------------------------------------------------------------------------
-- model_fields: per-model field definitions (JSONB-based, JSON Schema lite).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS model.model_fields (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id           UUID         NOT NULL REFERENCES model.models (id) ON DELETE CASCADE,
    name               VARCHAR(128) NOT NULL,
    label              VARCHAR(255),
    type               VARCHAR(32)  NOT NULL
                       CHECK (type IN (
                           'string','number','integer','boolean','date','datetime','time',
                           'json','enum','array','object','relation','file','ref','email',
                           'phone','url','color'
                       )),
    required           BOOLEAN      NOT NULL DEFAULT false,
    "default"          JSONB,
    validation_rules   JSONB        NOT NULL DEFAULT '{}'::jsonb,
    ui_config          JSONB        NOT NULL DEFAULT '{}'::jsonb,
    display_order      INT          NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (model_id, name)
);

CREATE INDEX IF NOT EXISTS idx_model_fields_model ON model.model_fields (model_id);

-- ---------------------------------------------------------------------------
-- model_records: dynamic instance data (all custom fields stored in JSONB).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS model.model_records (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL,
    model_id    UUID         NOT NULL REFERENCES model.models (id) ON DELETE CASCADE,
    data        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_by  UUID         NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- GIN index on data for fast JSONB searches.
CREATE INDEX IF NOT EXISTS idx_model_records_tenant ON model.model_records (tenant_id);
CREATE INDEX IF NOT EXISTS idx_model_records_model  ON model.model_records (model_id);
CREATE INDEX IF NOT EXISTS idx_model_records_data   ON model.model_records USING gin (data);
CREATE INDEX IF NOT EXISTS idx_model_records_mtime  ON model.model_records (updated_at DESC);

-- ---------------------------------------------------------------------------
-- model_versions: immutable snapshots of a model schema (auditing + rollback).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS model.model_versions (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id       UUID         NOT NULL REFERENCES model.models (id) ON DELETE CASCADE,
    version        INT          NOT NULL,
    snapshot       JSONB        NOT NULL,
    created_by     UUID         NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    note           TEXT,
    UNIQUE (model_id, version)
);

CREATE INDEX IF NOT EXISTS idx_model_versions_model ON model.model_versions (model_id);

-- ---------------------------------------------------------------------------
-- model_import_jobs: tracks CSV import progress / errors per model.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS model.model_import_jobs (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL,
    model_id      UUID         NOT NULL REFERENCES model.models (id) ON DELETE CASCADE,
    status        VARCHAR(32)  NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','running','done','failed')),
    total_rows    INT          NOT NULL DEFAULT 0,
    success_rows  INT          NOT NULL DEFAULT 0,
    failed_rows   INT          NOT NULL DEFAULT 0,
    errors        JSONB        NOT NULL DEFAULT '[]'::jsonb,
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    created_by    UUID         NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_model_import_jobs_model ON model.model_import_jobs (model_id);

-- ---------------------------------------------------------------------------
-- Row Level Security — enforce tenant_id from session vars.
-- ---------------------------------------------------------------------------
ALTER TABLE model.models          ENABLE ROW LEVEL SECURITY;
ALTER TABLE model.model_fields    ENABLE ROW LEVEL SECURITY;
ALTER TABLE model.model_records   ENABLE ROW LEVEL SECURITY;
ALTER TABLE model.model_versions  ENABLE ROW LEVEL SECURITY;
ALTER TABLE model.model_import_jobs ENABLE ROW LEVEL SECURITY;

-- Drop previous policies (idempotent).
DO $$
DECLARE t TEXT;
BEGIN
  FOR t IN
    SELECT tablename FROM pg_tables WHERE schemaname = 'model'
  LOOP
    EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON model.%I', t);
    EXECUTE format('CREATE POLICY tenant_isolation ON model.%I
                    USING (
                        current_setting(''app.is_super_admin'', true) = ''true''
                        OR (tenant_id::text = current_setting(''app.current_tenant_id'', true))
                    )', t);
  END LOOP;
END$$;
