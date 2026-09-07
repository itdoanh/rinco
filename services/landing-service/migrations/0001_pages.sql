-- 0001_init.sql
-- Landing pages schema

CREATE SCHEMA IF NOT EXISTS landing;

CREATE TABLE landing.landing_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    meta JSONB DEFAULT '{}',
    design_schema JSONB NOT NULL DEFAULT '{"blocks":[]}',
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
    published_at TIMESTAMPTZ,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_landing_pages_tenant ON landing.landing_pages(tenant_id);
CREATE INDEX idx_landing_pages_slug ON landing.landing_pages(tenant_id, slug);
CREATE INDEX idx_landing_pages_status ON landing.landing_pages(tenant_id, status);
