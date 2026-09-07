-- 0003_tracking.sql
-- Tracking events and conversion goals

CREATE TABLE landing.tracking_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    session_id TEXT,
    page_slug TEXT,
    properties JSONB DEFAULT '{}',
    ip INET,
    ua TEXT,
    referer TEXT,
    utm JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tracking_events_tenant ON landing.tracking_events(tenant_id, created_at DESC);
CREATE INDEX idx_tracking_events_session ON landing.tracking_events(session_id);
CREATE INDEX idx_tracking_events_type ON landing.tracking_events(tenant_id, event_type);

CREATE TABLE landing.conversion_goals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    event_name TEXT NOT NULL,
    value DECIMAL(10,2) DEFAULT 0,
    conditions JSONB DEFAULT '{}',
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversion_goals_tenant ON landing.conversion_goals(tenant_id);
