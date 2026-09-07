-- Migration 0002: Contacts table
-- =============================================================================

CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    job_title TEXT,
    department TEXT,
    owner_user_id UUID, -- Will reference users table after migration 0008
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'lead')),
    source TEXT CHECK (source IN ('organic', 'referral', 'social', 'ads', 'direct', 'other')),
    custom_fields JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    last_contacted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_contacts_tenant ON contacts(tenant_id);
CREATE INDEX idx_contacts_company ON contacts(tenant_id, company_id);
CREATE INDEX idx_contacts_owner ON contacts(tenant_id, owner_user_id);
CREATE INDEX idx_contacts_email ON contacts(tenant_id, email) WHERE email IS NOT NULL;
CREATE INDEX idx_contacts_name ON contacts USING gin(to_tsvector('simple', first_name || ' ' || last_name));
CREATE INDEX idx_contacts_status ON contacts(tenant_id, status);
CREATE INDEX idx_contacts_custom_fields ON contacts USING GIN (custom_fields);

ALTER TABLE contacts ENABLE ROW LEVEL SECURITY;

CREATE POLICY contacts_tenant_isolation ON contacts
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
