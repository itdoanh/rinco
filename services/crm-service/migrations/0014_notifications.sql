-- =============================================================================
-- Migration 0014: Internal notifications + departments
-- =============================================================================
-- Loop 203 — CRM Tree enterprise completion.
-- Phục vụ "Giám đốc broadcast toàn nhân viên", quản lý phòng ban.
-- =============================================================================

CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    leader_id UUID REFERENCES users(id) ON DELETE SET NULL,
    parent_department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_departments_tenant ON departments(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE departments ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS departments_tenant_isolation ON departments;
CREATE POLICY departments_tenant_isolation ON departments
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Member mapping
CREATE TABLE IF NOT EXISTS department_members (
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    role TEXT DEFAULT 'member' CHECK (role IN ('leader', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (department_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_dept_members_user ON department_members(user_id);
CREATE INDEX IF NOT EXISTS idx_dept_members_tenant ON department_members(tenant_id);

ALTER TABLE department_members ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS department_members_tenant_isolation ON department_members;
CREATE POLICY department_members_tenant_isolation ON department_members
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sender_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    type TEXT NOT NULL CHECK (type IN (
        'broadcast', 'mention', 'lead_assigned', 'task_due',
        'workflow', 'invite', 'system'
    )),
    title TEXT NOT NULL,
    body TEXT,
    link TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMPTZ,
    priority TEXT NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('LOW','NORMAL','HIGH','URGENT')),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notif_user_inbox ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_tenant_priority ON notifications(tenant_id, priority, created_at);

ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS notifications_tenant_isolation ON notifications;
CREATE POLICY notifications_tenant_isolation ON notifications
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            user_id = current_setting('app.current_user_id', true)::UUID
            OR current_setting('app.is_admin', true) = 'true'
        )
    );

-- Preference per user (JSON bitmask: email/push/sms)
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    email_enabled BOOLEAN DEFAULT true,
    push_enabled BOOLEAN DEFAULT true,
    inapp_enabled BOOLEAN DEFAULT true,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    digest_mode TEXT DEFAULT 'off' CHECK (digest_mode IN ('off','daily','weekly')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS notif_prefs_tenant_isolation ON notification_preferences;
CREATE POLICY notif_prefs_tenant_isolation ON notification_preferences
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND user_id = current_setting('app.current_user_id', true)::UUID
    );
