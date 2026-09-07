-- Migration 0007: Custom Fields table
-- =============================================================================

CREATE TABLE IF NOT EXISTS custom_fields (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('contact', 'company', 'deal', 'lead', 'activity')),
    name TEXT NOT NULL,
    field_key TEXT NOT NULL,
    field_type TEXT NOT NULL CHECK (field_type IN (
        'text', 'textarea', 'number', 'currency', 'date', 'datetime',
        'boolean', 'select', 'multiselect', 'email', 'phone', 'url'
    )),
    description TEXT,
    options JSONB DEFAULT '[]', -- For select/multiselect types
    default_value JSONB,
    validation_rules JSONB DEFAULT '{}',
    is_required BOOLEAN DEFAULT false,
    is_unique BOOLEAN DEFAULT false,
    display_order INT DEFAULT 0,
    width TEXT DEFAULT 'full' CHECK (width IN ('full', 'half', 'third')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, entity_type, field_key)
);

CREATE INDEX idx_custom_fields_tenant ON custom_fields(tenant_id);
CREATE INDEX idx_custom_fields_entity ON custom_fields(tenant_id, entity_type);
CREATE INDEX idx_custom_fields_active ON custom_fields(tenant_id, is_active) WHERE is_active = true;

ALTER TABLE custom_fields ENABLE ROW LEVEL SECURITY;

CREATE POLICY custom_fields_tenant_isolation ON custom_fields
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
