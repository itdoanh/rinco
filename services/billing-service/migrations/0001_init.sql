-- Billing Service Schema
-- Run: psql $DATABASE_URL -f 0001_init.sql

BEGIN;

-- Subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    plan            TEXT NOT NULL CHECK (plan IN ('free', 'pro', 'business', 'enterprise')),
    status          TEXT NOT NULL DEFAULT 'active',
    stripe_sub_id   TEXT UNIQUE,
    stripe_customer_id TEXT,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end   TIMESTAMPTZ NOT NULL,
    trial_ends_at   TIMESTAMPTZ,
    cancelled_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);

-- Invoices
CREATE TABLE IF NOT EXISTS invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    invoice_number  TEXT UNIQUE NOT NULL,
    amount          BIGINT NOT NULL, -- in cents
    currency        TEXT NOT NULL DEFAULT 'usd',
    status          TEXT NOT NULL DEFAULT 'open',
    stripe_invoice_id TEXT UNIQUE,
    tax_amount      BIGINT NOT NULL DEFAULT 0,
    tax_percent     NUMERIC(5,2) NOT NULL DEFAULT 0,
    period_start    TIMESTAMPTZ,
    period_end      TIMESTAMPTZ,
    due_date        TIMESTAMPTZ NOT NULL,
    invoice_date    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_invoices_tenant ON invoices(tenant_id);
CREATE INDEX idx_invoices_subscription ON invoices(subscription_id);
CREATE INDEX idx_invoices_status ON invoices(status);

-- Usage records
CREATE TABLE IF NOT EXISTS usage_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    month           INT NOT NULL CHECK (month BETWEEN 1 AND 12),
    year            INT NOT NULL,
    api_requests    BIGINT NOT NULL DEFAULT 0,
    storage_gb      NUMERIC(10,3) NOT NULL DEFAULT 0,
    ai_calls        BIGINT NOT NULL DEFAULT 0,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, month, year)
);
CREATE INDEX idx_usage_tenant ON usage_records(tenant_id);

-- Payment methods
CREATE TABLE IF NOT EXISTS payment_methods (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    stripe_pm_id    TEXT UNIQUE NOT NULL,
    type            TEXT NOT NULL,
    last4           TEXT,
    brand            TEXT,
    exp_month       INT,
    exp_year        INT,
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_payment_methods_tenant ON payment_methods(tenant_id);

-- Discount codes
CREATE TABLE IF NOT EXISTS discount_codes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            TEXT UNIQUE NOT NULL,
    discount_type   TEXT NOT NULL CHECK (discount_type IN ('percentage', 'fixed_amount')),
    discount_value  BIGINT NOT NULL,
    currency        TEXT,
    max_redemptions INT NOT NULL DEFAULT 0,
    max_uses        INT NOT NULL DEFAULT 0,
    redemption_count INT NOT NULL DEFAULT 0,
    current_uses    INT NOT NULL DEFAULT 0,
    expires_at      TIMESTAMPTZ,
    valid_from      TIMESTAMPTZ,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Webhook events (idempotency)
CREATE TABLE IF NOT EXISTS webhook_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id        TEXT,
    stripe_event_id TEXT UNIQUE,
    event_type      TEXT,
    type            TEXT,
    processed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payload         JSONB,
    data_payload    JSONB,
    attempts        INT NOT NULL DEFAULT 0,
    error           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;
