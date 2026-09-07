-- 0004_pixel.sql
-- Facebook Pixel and tracking click configs

CREATE TABLE landing.fb_pixel_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    pixel_id TEXT NOT NULL,
    access_token TEXT,
    test_event_code TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fb_pixel_configs_tenant ON landing.fb_pixel_configs(tenant_id);

CREATE TABLE landing.tracking_clicks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    target_url TEXT NOT NULL,
    utm JSONB DEFAULT '{}',
    clicks INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tracking_clicks_tenant ON landing.tracking_clicks(tenant_id);
