-- 0002_domains_settings.sql — custom domain mapping & per-tenant settings

CREATE TABLE IF NOT EXISTS tenant.tenant_domains (
    id          UUID PRIMARY KEY,
    tenant_id   UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    domain      TEXT NOT NULL UNIQUE,
    type        TEXT NOT NULL DEFAULT 'custom'
                CHECK (type IN ('custom', 'subdomain', 'alias')),
    verified_at TIMESTAMPTZ,
    ssl_status  TEXT NOT NULL DEFAULT 'pending'
                CHECK (ssl_status IN ('pending', 'active', 'failed', 'expired')),
    ssl_issuer  TEXT,
    ssl_expires_at TIMESTAMPTZ,
    is_primary  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenant_domains_tenant ON tenant.tenant_domains(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_domains_domain ON tenant.tenant_domains(domain);

CREATE TABLE IF NOT EXISTS tenant.tenant_settings (
    id          UUID PRIMARY KEY,
    tenant_id   UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    value       JSONB NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, key)
);

CREATE INDEX IF NOT EXISTS idx_tenant_settings_tenant ON tenant.tenant_settings(tenant_id);

-- Branding kept both as JSONB on tenants (snapshot) and structured here for query flexibility
CREATE TABLE IF NOT EXISTS tenant.tenant_branding (
    tenant_id   UUID PRIMARY KEY REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    logo_url    TEXT,
    favicon_url TEXT,
    primary_color   TEXT,
    secondary_color TEXT,
    accent_color    TEXT,
    font_family     TEXT,
    custom_css      TEXT,
    email_logo_url  TEXT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
