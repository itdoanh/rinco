-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0005 — Deals + Activities
-- ============================================================

CREATE TABLE IF NOT EXISTS crm.deals (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    lead_id               UUID,  -- FK declared after leads migration
    owner_user_id         UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    team_id               UUID,
    pipeline_id           UUID,
    stage                 TEXT NOT NULL DEFAULT 'new',
    title                 TEXT NOT NULL,
    value                 NUMERIC(18, 2),
    currency              TEXT NOT NULL DEFAULT 'VND',
    probability           INT CHECK (probability BETWEEN 0 AND 100),
    expected_close_date   DATE,
    actual_close_date     DATE,
    status                TEXT NOT NULL DEFAULT 'OPEN'
                          CHECK (status IN ('OPEN','WON','LOST','ABANDONED')),
    data                  JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_deals_tenant_stage ON crm.deals(tenant_id, stage);
CREATE INDEX IF NOT EXISTS idx_deals_owner        ON crm.deals(tenant_id, owner_user_id);
CREATE INDEX IF NOT EXISTS idx_deals_pipeline     ON crm.deals(tenant_id, pipeline_id);
CREATE INDEX IF NOT EXISTS idx_deals_close_date   ON crm.deals(tenant_id, expected_close_date) WHERE status = 'OPEN';
CREATE INDEX IF NOT EXISTS idx_deals_value        ON crm.deals(tenant_id, value)  WHERE status = 'OPEN';
CREATE INDEX IF NOT EXISTS idx_deals_data_gin     ON crm.deals USING GIN(data);

CREATE TRIGGER trg_deals_updated_at
    BEFORE UPDATE ON crm.deals
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- Activities
CREATE TABLE IF NOT EXISTS crm.activities (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    user_id       UUID REFERENCES auth.users(id),
    lead_id       UUID,
    deal_id       UUID REFERENCES crm.deals(id) ON DELETE CASCADE,
    type          TEXT NOT NULL
                  CHECK (type IN ('call','email','meeting','note','task','whatsapp','telegram','sms')),
    title         TEXT,
    description   TEXT,
    due_at        TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    status        TEXT DEFAULT 'pending' CHECK (status IN ('pending','completed','cancelled')),
    data          JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_activities_user_due   ON crm.activities(user_id, due_at);
CREATE INDEX IF NOT EXISTS idx_activities_lead      ON crm.activities(lead_id);
CREATE INDEX IF NOT EXISTS idx_activities_deal      ON crm.activities(deal_id);
CREATE INDEX IF NOT EXISTS idx_activities_pending   ON crm.activities(user_id, due_at) WHERE status = 'pending';

-- Pipelines (collection of stages)
CREATE TABLE IF NOT EXISTS crm.pipelines (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    is_default   BOOLEAN NOT NULL DEFAULT false,
    stages       JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_pipelines_tenant ON crm.pipelines(tenant_id);

-- ============================================================
-- RLS
-- ============================================================
ALTER TABLE crm.deals       ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.deals       FORCE  ROW LEVEL SECURITY;
ALTER TABLE crm.activities  ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.activities  FORCE  ROW LEVEL SECURITY;
ALTER TABLE crm.pipelines   ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.pipelines   FORCE  ROW LEVEL SECURITY;

CREATE POLICY deals_tenant ON crm.deals
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

-- Subtree visibility: own + reports
CREATE POLICY deals_subtree ON crm.deals
    FOR SELECT
    USING (
        app.bypass_rls_check()
        OR owner_user_id = app.current_user_id()
        OR owner_user_id IN (
            SELECT id FROM auth.users
            WHERE path <@ (SELECT path FROM auth.users WHERE id = app.current_user_id())
        )
    );

CREATE POLICY activities_tenant ON crm.activities
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY pipelines_tenant ON crm.pipelines
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS crm.pipelines   CASCADE;
DROP TABLE IF EXISTS crm.activities  CASCADE;
DROP TABLE IF EXISTS crm.deals       CASCADE;
-- +goose StatementEnd
