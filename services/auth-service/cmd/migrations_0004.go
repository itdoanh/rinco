package main

// migration0004: API keys + audit_logs + RLS enable.
const migration0004 = `
CREATE TABLE IF NOT EXISTS auth.api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    prefix          TEXT NOT NULL,
    hash            TEXT NOT NULL UNIQUE,
    last_4          TEXT NOT NULL,
    environment     TEXT NOT NULL DEFAULT 'live' CHECK (environment IN ('live','test')),
    scopes          TEXT[] NOT NULL DEFAULT '{}',
    rate_limit      INTEGER NOT NULL DEFAULT 1000,
    expires_at      TIMESTAMPTZ,
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ,
    revoked_reason  TEXT
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant
    ON auth.api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_user
    ON auth.api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix
    ON auth.api_keys(prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash
    ON auth.api_keys(hash);

CREATE TABLE IF NOT EXISTS auth.audit_logs (
    id                BIGSERIAL PRIMARY KEY,
    tenant_id         UUID,
    user_id           UUID,
    actor_ip          INET,
    actor_user_agent  TEXT,
    event             TEXT NOT NULL,
    outcome           TEXT NOT NULL,
    resource          TEXT,
    metadata          JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant
    ON auth.audit_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user
    ON auth.audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_event
    ON auth.audit_logs(event, created_at DESC);

DO $$
BEGIN
    BEGIN
        ALTER TABLE tenant.tenants ENABLE ROW LEVEL SECURITY;
    EXCEPTION WHEN duplicate_object THEN NULL; END;
    BEGIN
        ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
    EXCEPTION WHEN duplicate_object THEN NULL; END;
    BEGIN
        ALTER TABLE auth.sessions ENABLE ROW LEVEL SECURITY;
    EXCEPTION WHEN duplicate_object THEN NULL; END;
    BEGIN
        ALTER TABLE auth.webauthn_credentials ENABLE ROW LEVEL SECURITY;
    EXCEPTION WHEN duplicate_object THEN NULL; END;
    BEGIN
        ALTER TABLE auth.oauth_accounts ENABLE ROW LEVEL SECURITY;
    EXCEPTION WHEN duplicate_object THEN NULL; END;
    BEGIN
        ALTER TABLE auth.api_keys ENABLE ROW LEVEL SECURITY;
    EXCEPTION WHEN duplicate_object THEN NULL; END;
END $$;
`
