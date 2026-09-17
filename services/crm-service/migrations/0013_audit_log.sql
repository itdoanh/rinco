-- =============================================================================
-- Migration 0013: Audit log + user sessions
-- =============================================================================
-- Loop 203 — CRM Tree enterprise completion.
-- =============================================================================

CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_email TEXT,
    action TEXT NOT NULL,        -- 'lead.create', 'lead.convert', 'tree.move_subtree', ...
    target_type TEXT,            -- 'lead', 'deal', 'user', 'workflow', ...
    target_id UUID,
    old_value JSONB,
    new_value JSONB,
    ip_address INET,
    user_agent TEXT,
    trace_id TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_actor ON audit_log(tenant_id, actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_action ON audit_log(tenant_id, action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_target ON audit_log(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_audit_trace ON audit_log(trace_id) WHERE trace_id IS NOT NULL;

ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS audit_log_tenant_isolation ON audit_log;
CREATE POLICY audit_log_tenant_isolation ON audit_log
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            current_setting('app.is_admin', true) = 'true'
            OR actor_id = current_setting('app.current_user_id', true)::UUID
            OR EXISTS (
                SELECT 1 FROM users
                WHERE id = current_setting('app.current_user_id', true)::UUID
                AND tenant_id = audit_log.tenant_id
                AND role IN ('owner', 'admin', 'manager')
                AND deleted_at IS NULL
            )
        )
    );

CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    token_id TEXT UNIQUE NOT NULL,           -- JTI for PASETO/JWT
    device_fingerprint TEXT,
    ip_address INET,
    user_agent TEXT,
    login_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN DEFAULT false,
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_active ON user_sessions(user_id) WHERE NOT revoked;

ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS user_sessions_tenant_isolation ON user_sessions;
CREATE POLICY user_sessions_tenant_isolation ON user_sessions
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            user_id = current_setting('app.current_user_id', true)::UUID
            OR current_setting('app.is_admin', true) = 'true'
            OR EXISTS (
                SELECT 1 FROM users
                WHERE id = current_setting('app.current_user_id', true)::UUID
                AND tenant_id = user_sessions.tenant_id
                AND role IN ('owner', 'admin')
                AND deleted_at IS NULL
            )
        )
    );
