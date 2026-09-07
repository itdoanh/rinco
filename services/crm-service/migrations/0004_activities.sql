-- Migration 0004: Activities table
-- =============================================================================

CREATE TABLE IF NOT EXISTS activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    contact_id UUID REFERENCES contacts(id) ON DELETE CASCADE,
    deal_id UUID REFERENCES deals(id) ON DELETE SET NULL,
    type TEXT NOT NULL CHECK (type IN ('call', 'email', 'meeting', 'task', 'note')),
    subject TEXT NOT NULL,
    body TEXT,
    due_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    priority TEXT DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    reminder_at TIMESTAMPTZ,
    owner_user_id UUID, -- Will reference users table after migration 0008
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_activities_tenant ON activities(tenant_id);
CREATE INDEX idx_activities_contact ON activities(tenant_id, contact_id);
CREATE INDEX idx_activities_deal ON activities(tenant_id, deal_id);
CREATE INDEX idx_activities_type ON activities(tenant_id, type);
CREATE INDEX idx_activities_due ON activities(tenant_id, due_at) WHERE due_at IS NOT NULL;
CREATE INDEX idx_activities_owner ON activities(tenant_id, owner_user_id);
CREATE INDEX idx_activities_status ON activities(tenant_id, status);

ALTER TABLE activities ENABLE ROW LEVEL SECURITY;

CREATE POLICY activities_tenant_isolation ON activities
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
