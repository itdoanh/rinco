-- Migration 0001: email_templates
-- Stores reusable email templates per tenant with version history support.
CREATE TABLE IF NOT EXISTS email.email_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    body_type TEXT NOT NULL DEFAULT 'html' CHECK (body_type IN ('text', 'html', 'markdown')),
    vars JSONB NOT NULL DEFAULT '{}'::jsonb,
    type TEXT NOT NULL DEFAULT 'transactional',
    is_active BOOLEAN NOT NULL DEFAULT true,
    version INT NOT NULL DEFAULT 1,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_email_templates_tenant ON email.email_templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_email_templates_tenant_name ON email.email_templates(tenant_id, name) WHERE deleted_at IS NULL;

ALTER TABLE email.email_templates ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS email_templates_tenant ON email.email_templates;
CREATE POLICY email_templates_tenant ON email.email_templates
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        OR current_setting('app.is_admin', true) = 'true'
    );
