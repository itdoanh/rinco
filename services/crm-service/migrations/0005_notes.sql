-- Migration 0005: Notes table
-- =============================================================================

CREATE TABLE IF NOT EXISTS notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    contact_id UUID REFERENCES contacts(id) ON DELETE CASCADE,
    deal_id UUID REFERENCES deals(id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    author_id UUID, -- Will reference users table after migration 0008
    is_pinned BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_notes_tenant ON notes(tenant_id);
CREATE INDEX idx_notes_contact ON notes(tenant_id, contact_id);
CREATE INDEX idx_notes_deal ON notes(tenant_id, deal_id);
CREATE INDEX idx_notes_author ON notes(tenant_id, author_id);
CREATE INDEX idx_notes_body ON notes USING gin(to_tsvector('simple', body));

ALTER TABLE notes ENABLE ROW LEVEL SECURITY;

CREATE POLICY notes_tenant_isolation ON notes
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
