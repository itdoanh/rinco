-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0006 — leads + AI-scoring fields
-- ============================================================

CREATE TABLE IF NOT EXISTS leads.leads (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    owner_user_id           UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    full_name               TEXT NOT NULL,
    email                   CITEXT,
    phone                   TEXT,
    source                  TEXT,
    utm_source              TEXT,
    utm_medium              TEXT,
    utm_campaign            TEXT,
    utm_content             TEXT,
    utm_term                TEXT,
    fbclid                  TEXT,
    gclid                   TEXT,
    ttclid                  TEXT,
    campaign_id             UUID,
    score                   INT CHECK (score BETWEEN 0 AND 100),
    score_band              TEXT GENERATED ALWAYS AS (
        CASE
            WHEN score >= 80 THEN 'hot'
            WHEN score >= 50 THEN 'warm'
            WHEN score >= 20 THEN 'cold'
            ELSE 'low'
        END
    ) STORED,
    quality_score           TEXT CHECK (quality_score IN ('high','medium','low')),
    predicted_ltv           DECIMAL(18, 2),
    conversion_probability  DECIMAL(5, 4),
    custom_fields           JSONB NOT NULL DEFAULT '{}'::jsonb,
    data                    JSONB NOT NULL DEFAULT '{}'::jsonb,
    consent_given           BOOLEAN NOT NULL DEFAULT false,
    consent_at              TIMESTAMPTZ,
    source_meta             JSONB NOT NULL DEFAULT '{}'::jsonb,
    crm_synced              BOOLEAN NOT NULL DEFAULT false,
    capi_sent               BOOLEAN NOT NULL DEFAULT false,
    capi_event_id           UUID,
    stage                   TEXT NOT NULL DEFAULT 'new',
    last_contacted_at       TIMESTAMPTZ,
    converted_at            TIMESTAMPTZ,
    metadata                JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_leads_tenant          ON leads.leads(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_owner           ON leads.leads(tenant_id, owner_user_id);
CREATE INDEX IF NOT EXISTS idx_leads_score           ON leads.leads(tenant_id, score DESC) WHERE score IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_leads_stage           ON leads.leads(tenant_id, stage);
CREATE INDEX IF NOT EXISTS idx_leads_status_active   ON leads.leads(tenant_id, stage) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_phone           ON leads.leads(tenant_id, phone);
CREATE INDEX IF NOT EXISTS idx_leads_email           ON leads.leads(tenant_id, email);
CREATE INDEX IF NOT EXISTS idx_leads_data_phone_gin  ON leads.leads USING GIN((data->'phone'));
CREATE INDEX IF NOT EXISTS idx_leads_data_gin        ON leads.leads USING GIN(data);
CREATE INDEX IF NOT EXISTS idx_leads_created_at      ON leads.leads(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_leads_hot             ON leads.leads(tenant_id, owner_user_id, created_at DESC)
                                                  WHERE stage IN ('new','contacted','qualified');
CREATE INDEX IF NOT EXISTS idx_leads_dashboard       ON leads.leads(tenant_id, stage, owner_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_leads_fbclid          ON leads.leads(tenant_id, fbclid) WHERE fbclid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_leads_phone_trgm      ON leads.leads USING gin(phone gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_leads_owner_covering  ON leads.leads(tenant_id, owner_user_id)
                                                  INCLUDE (stage, score, created_at);

CREATE TRIGGER trg_leads_updated_at
    BEFORE UPDATE ON leads.leads
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- Now wire the deals FK back to leads (deferred from 0005)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE constraint_name = 'deals_lead_id_fkey'
      AND table_name = 'deals'
      AND table_schema = 'crm'
  ) THEN
    ALTER TABLE crm.deals
      ADD CONSTRAINT deals_lead_id_fkey
      FOREIGN KEY (lead_id) REFERENCES leads.leads(id) ON DELETE SET NULL;
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE constraint_name = 'activities_lead_id_fkey'
      AND table_name = 'activities'
      AND table_schema = 'crm'
  ) THEN
    ALTER TABLE crm.activities
      ADD CONSTRAINT activities_lead_id_fkey
      FOREIGN KEY (lead_id) REFERENCES leads.leads(id) ON DELETE CASCADE;
  END IF;
END$$;

-- ============================================================
-- RLS
-- ============================================================
ALTER TABLE leads.leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads.leads FORCE  ROW LEVEL SECURITY;

CREATE POLICY leads_tenant ON leads.leads
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY leads_user_scope ON leads.leads
    FOR SELECT
    USING (
        app.bypass_rls_check()
        OR owner_user_id = app.current_user_id()
        OR owner_user_id IN (
            SELECT id FROM auth.users
            WHERE path <@ (
                SELECT path FROM auth.users
                WHERE id = app.current_user_id()
            )
        )
    );

-- ============================================================
-- Views
-- ============================================================
CREATE OR REPLACE VIEW leads.v_lead_funnel AS
SELECT
    tenant_id,
    DATE_TRUNC('day', created_at) AS day,
    source,
    utm_campaign,
    COUNT(*) FILTER (WHERE stage = 'new')      AS new_count,
    COUNT(*) FILTER (WHERE stage = 'qualified') AS qualified_count,
    COUNT(*) FILTER (WHERE stage = 'converted') AS converted_count,
    COUNT(*) FILTER (WHERE stage = 'lost')     AS lost_count,
    AVG(score) FILTER (WHERE score IS NOT NULL) AS avg_score
FROM leads.leads
WHERE deleted_at IS NULL
GROUP BY tenant_id, DATE_TRUNC('day', created_at), source, utm_campaign;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS leads.v_lead_funnel;
ALTER TABLE crm.activities DROP CONSTRAINT IF EXISTS activities_lead_id_fkey;
ALTER TABLE crm.deals     DROP CONSTRAINT IF EXISTS deals_lead_id_fkey;
DROP TABLE IF EXISTS leads.leads CASCADE;
-- +goose StatementEnd
