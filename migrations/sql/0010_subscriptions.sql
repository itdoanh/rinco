-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0010 — Subscriptions / Billing
-- ============================================================

CREATE TABLE IF NOT EXISTS billing.subscriptions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    plan_code               TEXT NOT NULL,                  -- 'starter','business','enterprise'
    status                  TEXT NOT NULL DEFAULT 'active'
                            CHECK (status IN ('trialing','active','past_due','cancelled','unpaid')),
    trial_ends_at           TIMESTAMPTZ,
    current_period_start    TIMESTAMPTZ NOT NULL,
    current_period_end      TIMESTAMPTZ NOT NULL,
    cancel_at               TIMESTAMPTZ,
    cancelled_at            TIMESTAMPTZ,
    billing_interval        TEXT NOT NULL DEFAULT 'month'
                            CHECK (billing_interval IN ('month','year')),
    seats_limit             INT NOT NULL DEFAULT 50,
    storage_limit_bytes     BIGINT NOT NULL DEFAULT 107374182400,
    monthly_leads_limit     BIGINT NOT NULL DEFAULT 50000,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_subs_tenant ON billing.subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_subs_status ON billing.subscriptions(status);

CREATE TRIGGER trg_subs_updated_at
    BEFORE UPDATE ON billing.subscriptions
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- Invoices
CREATE TABLE IF NOT EXISTS billing.invoices (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    subscription_id     UUID REFERENCES billing.subscriptions(id),
    invoice_number      TEXT UNIQUE NOT NULL,
    period_start        TIMESTAMPTZ NOT NULL,
    period_end          TIMESTAMPTZ NOT NULL,
    subtotal            NUMERIC(18, 2) NOT NULL DEFAULT 0,
    tax                 NUMERIC(18, 2) NOT NULL DEFAULT 0,
    total               NUMERIC(18, 2) NOT NULL DEFAULT 0,
    currency            TEXT NOT NULL DEFAULT 'VND',
    status              TEXT NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft','open','paid','void','uncollectible')),
    paid_at             TIMESTAMPTZ,
    due_at              TIMESTAMPTZ,
    line_items          JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_inv_tenant ON billing.invoices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_inv_status ON billing.invoices(status);
CREATE INDEX IF NOT EXISTS idx_inv_period ON billing.invoices(period_end);

-- Usage tracking (counters per billing period)
CREATE TABLE IF NOT EXISTS billing.usage_counters (
    id                  BIGSERIAL PRIMARY KEY,
    tenant_id           UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    metric              TEXT NOT NULL,            -- 'api_calls','storage_bytes','leads','seats'
    period_start        TIMESTAMPTZ NOT NULL,
    period_end          TIMESTAMPTZ NOT NULL,
    value               BIGINT NOT NULL DEFAULT 0,
    metadata            JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, metric, period_start)
);
CREATE INDEX IF NOT EXISTS idx_usage_tenant ON billing.usage_counters(tenant_id);
CREATE INDEX IF NOT EXISTS idx_usage_period ON billing.usage_counters(period_start);

-- Payment methods (tokenized reference only — full PCI via gateway)
CREATE TABLE IF NOT EXISTS billing.payment_methods (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    gateway             TEXT NOT NULL,                 -- 'stripe','vnpay','momo'
    gateway_pm_id       TEXT NOT NULL,
    brand               TEXT,
    last4               TEXT(4),
    exp_month           INT,
    exp_year            INT,
    is_default          BOOLEAN NOT NULL DEFAULT false,
    metadata            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (gateway, gateway_pm_id)
);
CREATE INDEX IF NOT EXISTS idx_pm_tenant ON billing.payment_methods(tenant_id);

-- ============================================================
-- RLS
-- ============================================================
ALTER TABLE billing.subscriptions    ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing.subscriptions    FORCE  ROW LEVEL SECURITY;
ALTER TABLE billing.invoices         ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing.invoices         FORCE  ROW LEVEL SECURITY;
ALTER TABLE billing.usage_counters   ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing.usage_counters   FORCE  ROW LEVEL SECURITY;
ALTER TABLE billing.payment_methods  ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing.payment_methods  FORCE  ROW LEVEL SECURITY;

CREATE POLICY subs_tenant ON billing.subscriptions
    FOR ALL USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    ) WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY inv_tenant ON billing.invoices
    FOR ALL USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    ) WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY usage_tenant ON billing.usage_counters
    FOR ALL USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    ) WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY pm_tenant ON billing.payment_methods
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
DROP TABLE IF EXISTS billing.payment_methods CASCADE;
DROP TABLE IF EXISTS billing.usage_counters  CASCADE;
DROP TABLE IF EXISTS billing.invoices        CASCADE;
DROP TABLE IF EXISTS billing.subscriptions    CASCADE;
-- +goose StatementEnd
