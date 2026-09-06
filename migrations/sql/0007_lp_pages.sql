-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0007 — Landing pages storage (Postgres + Mongo mirror)
-- Postgres holds metadata + version history, Mongo stores the
-- actual block-tree (see mongo migration 0001).
-- ============================================================

CREATE TABLE IF NOT EXISTS crm.landing_pages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    slug            CITEXT NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT,
    domain          CITEXT,
    form_id         UUID,
    mongo_doc_id    TEXT,                          -- pointer into Mongo collections
    template_id     UUID,
    ab_test_id      UUID,
    status          TEXT NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','published','paused','archived')),
    is_default      BOOLEAN NOT NULL DEFAULT false,
    seo_settings    JSONB NOT NULL DEFAULT '{}'::jsonb,
    theme           JSONB NOT NULL DEFAULT '{}'::jsonb,
    blocks          JSONB NOT NULL DEFAULT '[]'::jsonb, -- copy-on-write snapshot
    analytics       JSONB NOT NULL DEFAULT '{}'::jsonb,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, slug, domain)
);

CREATE INDEX IF NOT EXISTS idx_lp_tenant  ON crm.landing_pages(tenant_id);
CREATE INDEX IF NOT EXISTS idx_lp_domain   ON crm.landing_pages(domain);
CREATE INDEX IF NOT EXISTS idx_lp_status   ON crm.landing_pages(status);
CREATE INDEX IF NOT EXISTS idx_lp_form     ON crm.landing_pages(form_id);

CREATE TRIGGER trg_landing_pages_updated_at
    BEFORE UPDATE ON crm.landing_pages
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- Version history (every save)
CREATE TABLE IF NOT EXISTS crm.landing_page_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id         UUID NOT NULL REFERENCES crm.landing_pages(id) ON DELETE CASCADE,
    version         INT  NOT NULL,
    blocks          JSONB NOT NULL,
    created_by      UUID REFERENCES auth.users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (page_id, version)
);

CREATE INDEX IF NOT EXISTS idx_lpv_page ON crm.landing_page_versions(page_id);

-- A/B test variants
CREATE TABLE IF NOT EXISTS crm.landing_page_ab_variants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id         UUID NOT NULL REFERENCES crm.landing_pages(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    weight          INT NOT NULL DEFAULT 50 CHECK (weight BETWEEN 0 AND 100),
    blocks          JSONB NOT NULL,
    metrics         JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_lp_ab_page ON crm.landing_page_ab_variants(page_id);

-- ============================================================
-- RLS
-- ============================================================
ALTER TABLE crm.landing_pages            ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.landing_pages            FORCE  ROW LEVEL SECURITY;
ALTER TABLE crm.landing_page_versions    ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.landing_page_versions    FORCE  ROW LEVEL SECURITY;
ALTER TABLE crm.landing_page_ab_variants ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm.landing_page_ab_variants FORCE  ROW LEVEL SECURITY;

CREATE POLICY lp_tenant ON crm.landing_pages
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY lpv_tenant ON crm.landing_page_versions
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR page_id IN (SELECT id FROM crm.landing_pages WHERE tenant_id = app.current_tenant_id())
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR page_id IN (SELECT id FROM crm.landing_pages WHERE tenant_id = app.current_tenant_id())
    );

CREATE POLICY lpvab_tenant ON crm.landing_page_ab_variants
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR page_id IN (SELECT id FROM crm.landing_pages WHERE tenant_id = app.current_tenant_id())
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR page_id IN (SELECT id FROM crm.landing_pages WHERE tenant_id = app.current_tenant_id())
    );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS crm.landing_page_ab_variants CASCADE;
DROP TABLE IF EXISTS crm.landing_page_versions    CASCADE;
DROP TABLE IF EXISTS crm.landing_pages            CASCADE;
-- +goose StatementEnd
