-- Migration 0002: email_logs + email_attachments + tenant_settings
-- Delivery log for every outbound email (status tracking, timing, error info).
-- Attachments stored in tiered S3 with hot/cold bucket support.
-- Per-tenant email settings and tier policy overrides.

CREATE TABLE IF NOT EXISTS email.email_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    msg_id TEXT NOT NULL UNIQUE,
    template_id UUID,
    from_addr TEXT NOT NULL,
    to_addrs JSONB NOT NULL DEFAULT '[]'::jsonb,
    cc JSONB NOT NULL DEFAULT '[]'::jsonb,
    bcc JSONB NOT NULL DEFAULT '[]'::jsonb,
    subject TEXT NOT NULL,
    body_rendered TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN (
            'queued', 'sending', 'sent', 'delivered',
            'opened', 'clicked', 'bounced', 'complained',
            'failed', 'retrying'
        )),
    driver TEXT NOT NULL DEFAULT 'console',
    driver_msg_id TEXT,
    priority TEXT NOT NULL DEFAULT 'normal' CHECK (priority IN ('high', 'normal', 'low')),
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    clicked_at TIMESTAMPTZ,
    bounced_at TIMESTAMPTZ,
    complained_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    scheduled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_logs_tenant ON email.email_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_email_logs_status ON email.email_logs(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_email_logs_template ON email.email_logs(template_id);
CREATE INDEX IF NOT EXISTS idx_email_logs_created ON email.email_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_email_logs_msg ON email.email_logs(msg_id);

ALTER TABLE email.email_logs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS email_logs_tenant ON email.email_logs;
CREATE POLICY email_logs_tenant ON email.email_logs
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        OR current_setting('app.is_admin', true) = 'true'
    );

-- Attachments: stored in S3, tracked in DB for cross-tier queries.
CREATE TABLE IF NOT EXISTS email.email_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    msg_id TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    s3_key TEXT NOT NULL,
    s3_bucket TEXT NOT NULL,
    tier TEXT NOT NULL DEFAULT 'hot' CHECK (tier IN ('hot', 'cold')),
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_attachments_msg ON email.email_attachments(msg_id);
CREATE INDEX IF NOT EXISTS idx_email_attachments_tenant ON email.email_attachments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_email_attachments_tier ON email.email_attachments(tier, uploaded_at);

-- Per-tenant email settings (default driver, from address, S3 tier policy).
CREATE TABLE IF NOT EXISTS email.tenant_settings (
    tenant_id UUID PRIMARY KEY,
    tier_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    driver TEXT NOT NULL DEFAULT 'console',
    default_from TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
