-- 0002_forms.sql
-- Form definitions and submissions

CREATE TABLE landing.form_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    fields JSONB NOT NULL DEFAULT '[]',
    submit_action JSONB DEFAULT '{"type":"lead"}',
    success_message TEXT DEFAULT 'Cảm ơn bạn! Chúng tôi sẽ liên hệ sớm nhất.',
    redirect_url TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_form_definitions_tenant ON landing.form_definitions(tenant_id);

CREATE TABLE landing.form_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_id UUID NOT NULL REFERENCES landing.form_definitions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    lead_id UUID,
    ip INET,
    ua TEXT,
    utm JSONB DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'synced', 'error')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_form_submissions_tenant ON landing.form_submissions(tenant_id, created_at DESC);
CREATE INDEX idx_form_submissions_form ON landing.form_submissions(form_id);
