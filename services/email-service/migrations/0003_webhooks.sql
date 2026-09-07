-- Migration 0003: email_webhooks
-- Stores raw webhook payloads from providers (SES, SendGrid, Resend, etc.)
-- for replay, audit, and debugging.
CREATE TABLE IF NOT EXISTS email.email_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    provider TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    processed_at TIMESTAMPTZ,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_webhooks_tenant ON email.email_webhooks(tenant_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_email_webhooks_provider ON email.email_webhooks(provider, event_type);
