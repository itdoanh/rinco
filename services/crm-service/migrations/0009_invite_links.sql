-- Migration 0009: Invite Links
-- =============================================================================

CREATE TABLE IF NOT EXISTS invite_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    parent_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_role TEXT NOT NULL CHECK (target_role IN (
        'admin', 'manager', 'team_lead', 'member', 'viewer'
    )),
    token TEXT NOT NULL UNIQUE,
    max_uses INT DEFAULT 1,
    current_uses INT DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id)
);

CREATE INDEX idx_invite_links_token ON invite_links(token);
CREATE INDEX idx_invite_links_tenant ON invite_links(tenant_id);
CREATE INDEX idx_invite_links_parent ON invite_links(tenant_id, parent_user_id);
CREATE INDEX idx_invite_links_expires ON invite_links(expires_at) WHERE revoked_at IS NULL AND expires_at > NOW();

ALTER TABLE invite_links ENABLE ROW LEVEL SECURITY;

CREATE POLICY invite_links_tenant_isolation ON invite_links
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Invite usage history
CREATE TABLE IF NOT EXISTS invite_usage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invite_id UUID NOT NULL REFERENCES invite_links(id) ON DELETE CASCADE,
    used_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_invite_usage_invite ON invite_usage_history(invite_id);
