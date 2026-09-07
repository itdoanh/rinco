-- Migration 0006: Tags table
-- =============================================================================

CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    color TEXT DEFAULT '#6366f1',
    description TEXT,
    usage_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_tags_tenant ON tags(tenant_id);
CREATE INDEX idx_tags_name ON tags(tenant_id, name);

ALTER TABLE tags ENABLE ROW LEVEL SECURITY;

CREATE POLICY tags_tenant_isolation ON tags
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Contact tags junction table
CREATE TABLE IF NOT EXISTS contact_tags (
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (contact_id, tag_id)
);

CREATE INDEX idx_contact_tags_tag ON contact_tags(tag_id);
