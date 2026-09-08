-- +goose Up
-- +goose StatementBegin

-- Subscriptions table for tenant billing
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    plan TEXT NOT NULL CHECK (plan IN ('FREE', 'PRO', 'BUSINESS', 'ENTERPRISE')),
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'pending', 'cancelled', 'past_due', 'trialing', 'incomplete')),
    stripe_sub_id TEXT,
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMPTZ NOT NULL,
    trial_ends_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_subs_tenant_active ON subscriptions (tenant_id) WHERE status != 'cancelled';
CREATE INDEX idx_subs_stripe ON subscriptions (stripe_sub_id) WHERE stripe_sub_id IS NOT NULL;
CREATE INDEX idx_subs_plan_status ON subscriptions (plan, status);
CREATE INDEX idx_subs_period_end ON subscriptions (current_period_end) WHERE status = 'active';

-- Trigger update updated_at
CREATE OR REPLACE FUNCTION subscriptions_update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER subscriptions_update_trigger
BEFORE UPDATE ON subscriptions
FOR EACH ROW EXECUTE FUNCTION subscriptions_update_updated_at();

-- Invoices table for billing invoices
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    number TEXT UNIQUE NOT NULL,
    status TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('draft', 'open', 'paid', 'void', 'uncollectible')),
    amount BIGINT NOT NULL CHECK (amount >= 0),
    currency TEXT NOT NULL DEFAULT 'USD',
    tax_amount BIGINT NOT NULL DEFAULT 0,
    tax_percent NUMERIC(5,2) NOT NULL DEFAULT 0,
    stripe_invoice_id TEXT,
    paid_at TIMESTAMPTZ,
    due_date TIMESTAMPTZ NOT NULL,
    invoice_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inv_tenant ON invoices (tenant_id, created_at DESC);
CREATE INDEX idx_inv_status ON invoices (status, due_date);
CREATE INDEX idx_inv_subscription ON invoices (subscription_id);
CREATE INDEX idx_inv_stripe ON invoices (stripe_invoice_id) WHERE stripe_invoice_id IS NOT NULL;

-- Invoice line items
CREATE TABLE invoice_line_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    unit_amount BIGINT NOT NULL CHECK (unit_amount >= 0),
    amount BIGINT NOT NULL CHECK (amount >= 0),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_invoice_items ON invoice_line_items (invoice_id);

-- Usage records (for usage-based billing)
CREATE TABLE usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    month INTEGER NOT NULL CHECK (month BETWEEN 1 AND 12),
    year INTEGER NOT NULL CHECK (year BETWEEN 2020 AND 2100),
    api_requests BIGINT NOT NULL DEFAULT 0,
    storage_gb NUMERIC(12,4) NOT NULL DEFAULT 0,
    ai_calls BIGINT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, month, year)
);

CREATE INDEX idx_usage_tenant ON usage_records (tenant_id, year DESC, month DESC);

-- Payment methods
CREATE TABLE payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    stripe_pm_id TEXT NOT NULL,
    brand TEXT NOT NULL,
    last4 TEXT NOT NULL,
    exp_month INTEGER NOT NULL,
    exp_year INTEGER NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (stripe_pm_id)
);

CREATE INDEX idx_payment_methods_tenant ON payment_methods (tenant_id, is_default);

-- Discount codes
CREATE TABLE discount_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT UNIQUE NOT NULL,
    discount_type TEXT NOT NULL CHECK (discount_type IN ('percentage', 'fixed_amount')),
    discount_value BIGINT NOT NULL CHECK (discount_value >= 0),
    currency TEXT,
    max_uses INTEGER NOT NULL DEFAULT 0,
    current_uses INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_discount_codes_active ON discount_codes (is_active, code) WHERE is_active = true;

-- Webhook events (Stripe webhook audit/replay)
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stripe_event_id TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    data_payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_events_type ON webhook_events (type, created_at DESC);
CREATE INDEX idx_webhook_events_unprocessed ON webhook_events (created_at DESC) WHERE processed_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS subscriptions_update_trigger ON subscriptions;
DROP FUNCTION IF EXISTS subscriptions_update_updated_at();
DROP TABLE IF EXISTS webhook_events;
DROP TABLE IF EXISTS discount_codes;
DROP TABLE IF EXISTS payment_methods;
DROP TABLE IF EXISTS usage_records;
DROP TABLE IF EXISTS invoice_line_items;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS subscriptions;
-- +goose StatementEnd
