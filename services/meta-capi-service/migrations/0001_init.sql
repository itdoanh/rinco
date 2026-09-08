-- Meta CAPI Service: PostgreSQL schema
-- Run: psql $DATABASE_URL -f 0001_init.sql

BEGIN;

-- CAPI Configurations per tenant
CREATE SCHEMA IF NOT EXISTS meta_capi;

CREATE TABLE IF NOT EXISTS meta_capi.configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL UNIQUE,
    pixel_id        TEXT NOT NULL,
    access_token    TEXT NOT NULL, -- encrypted at rest
    test_event_code TEXT,
    is_enabled      BOOLEAN NOT NULL DEFAULT true,
    event_types     TEXT[] NOT NULL DEFAULT ARRAY['Lead', 'Purchase', 'Contact', 'ViewContent'],
    sample_rate     NUMERIC(5,3) NOT NULL DEFAULT 1.0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_configs_tenant ON meta_capi.configs(tenant_id);

-- CAPI Events (outbound to Facebook)
CREATE TABLE IF NOT EXISTS meta_capi.events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    event_id        TEXT NOT NULL, -- unique deduplication ID
    event_name      TEXT NOT NULL,
    event_time      TIMESTAMPTZ NOT NULL,
    event_source    TEXT NOT NULL, -- web, app, offline, crm
    email           TEXT,
    phone           TEXT,
    ip_address      INET,
    user_agent      TEXT,
    country         TEXT,
    fbp_id          TEXT,
    fbc_id          TEXT,
    lead_id         TEXT,
    deal_id         TEXT,
    order_value     NUMERIC(15,2),
    currency        TEXT DEFAULT 'USD',
    custom_data     JSONB,
    status          TEXT NOT NULL DEFAULT 'pending',
    fb_event_id     TEXT,
    error_message   TEXT,
    retry_count     INT NOT NULL DEFAULT 0,
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, event_id)
);
CREATE INDEX idx_events_tenant_status ON meta_capi.events(tenant_id, status);
CREATE INDEX idx_events_created ON meta_capi.events(created_at);

-- Feedback Events (inbound from Facebook)
CREATE TABLE IF NOT EXISTS meta_capi.feedback_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    capi_event_id   UUID,
    fb_event_id     TEXT NOT NULL,
    fb_event_name   TEXT,
    fb_event_time   TIMESTAMPTZ,
    fb_partner_name TEXT,
    fb_partner_id   TEXT,
    fb_artist_id    TEXT,
    fb_claim_code   TEXT,
    fb_disaggregate TEXT,
    processed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(fb_event_id)
);
CREATE INDEX idx_feedback_tenant ON meta_capi.feedback_events(tenant_id);

-- Conversion Mappings (CRM events → CAPI events)
CREATE TABLE IF NOT EXISTS meta_capi.mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    crm_event_type  TEXT NOT NULL,
    capi_event_name TEXT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    value_field     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, crm_event_type)
);
CREATE INDEX idx_mappings_tenant ON meta_capi.mappings(tenant_id);

COMMIT;
