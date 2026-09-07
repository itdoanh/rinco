package main

// migration0002: Sessions (refresh tokens + audit metadata).
// Sliding-window state is cached in Valkey/Redis; this table is the
// durable record that survives cache evictions.
const migration0002 = `
CREATE TABLE IF NOT EXISTS auth.sessions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    tenant_id             UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    refresh_token_hash    TEXT NOT NULL UNIQUE,
    access_token_jti      TEXT,
    ip                    INET,
    user_agent            TEXT,
    device_fingerprint    TEXT,
    metadata              JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_seen_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at            TIMESTAMPTZ NOT NULL,
    revoked_at            TIMESTAMPTZ,
    revoked_reason        TEXT
);

CREATE INDEX IF NOT EXISTS idx_sessions_user
    ON auth.sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_tenant
    ON auth.sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token_hash
    ON auth.sessions(refresh_token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_expires
    ON auth.sessions(expires_at);

ALTER TABLE auth.sessions
    ADD COLUMN IF NOT EXISTS previous_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL;
`
