-- Migration 0003: Deals table
-- =============================================================================

CREATE TABLE IF NOT EXISTS deals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    owner_user_id UUID, -- Will reference users table after migration 0008
    name TEXT NOT NULL,
    value DECIMAL(15, 2) DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'VND',
    stage TEXT NOT NULL DEFAULT 'prospecting' CHECK (stage IN (
        'prospecting', 'qualification', 'proposal', 'negotiation', 
        'won', 'lost', 'on_hold'
    )),
    probability INT CHECK (probability >= 0 AND probability <= 100),
    expected_close_date DATE,
    actual_close_date DATE,
    lost_reason TEXT,
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_deals_tenant ON deals(tenant_id);
CREATE INDEX idx_deals_contact ON deals(tenant_id, contact_id);
CREATE INDEX idx_deals_owner ON deals(tenant_id, owner_user_id);
CREATE INDEX idx_deals_stage ON deals(tenant_id, stage);
CREATE INDEX idx_deals_status ON deals(tenant_id, status);
CREATE INDEX idx_deals_expected_close ON deals(tenant_id, expected_close_date) 
    WHERE expected_close_date IS NOT NULL;

ALTER TABLE deals ENABLE ROW LEVEL SECURITY;

CREATE POLICY deals_tenant_isolation ON deals
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Stage history for deal tracking
CREATE TABLE IF NOT EXISTS deal_stage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    from_stage TEXT,
    to_stage TEXT NOT NULL,
    changed_by UUID, -- Will reference users table after migration 0008
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT
);

CREATE INDEX idx_deal_stage_history_deal ON deal_stage_history(deal_id, changed_at DESC);
