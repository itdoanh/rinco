package main

// migration0003: WebAuthn credentials, OAuth account linkage, and
// challenge/state persistence (used by the begin/finish ceremonies).
const migration0003 = `
CREATE TABLE IF NOT EXISTS auth.webauthn_credentials (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    tenant_id         UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    credential_id     BYTEA NOT NULL,
    public_key        BYTEA NOT NULL,
    aaguid            BYTEA,
    counter           BIGINT NOT NULL DEFAULT 0,
    transports        TEXT[] NOT NULL DEFAULT '{}',
    backup_eligible   BOOLEAN NOT NULL DEFAULT FALSE,
    backup_state      BOOLEAN NOT NULL DEFAULT FALSE,
    user_present      BOOLEAN NOT NULL DEFAULT TRUE,
    user_verified     BOOLEAN NOT NULL DEFAULT FALSE,
    name              TEXT NOT NULL DEFAULT '',
    last_used_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, credential_id)
);

CREATE INDEX IF NOT EXISTS idx_webauthn_user
    ON auth.webauthn_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_tenant
    ON auth.webauthn_credentials(tenant_id);

CREATE TABLE IF NOT EXISTS auth.oauth_accounts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    provider            TEXT NOT NULL CHECK (provider IN ('google','facebook','microsoft','apple','line')),
    provider_user_id    TEXT NOT NULL,
    provider_email      TEXT,
    access_token_hash   TEXT,
    refresh_token_hash  TEXT,
    scopes              TEXT[] NOT NULL DEFAULT '{}',
    raw_profile         JSONB NOT NULL DEFAULT '{}'::jsonb,
    linked_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at        TIMESTAMPTZ,
    UNIQUE (provider, provider_user_id)
);

CREATE INDEX IF NOT EXISTS idx_oauth_user
    ON auth.oauth_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_provider
    ON auth.oauth_accounts(provider, provider_user_id);

CREATE TABLE IF NOT EXISTS auth.webauthn_challenges (
    key              TEXT PRIMARY KEY,
    flow             TEXT NOT NULL CHECK (flow IN ('registration','login')),
    user_id          UUID,
    tenant_id        UUID,
    challenge        BYTEA NOT NULL,
    allowed_creds    BYTEA[] NOT NULL DEFAULT '{}',
    user_handle      BYTEA,
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_expires
    ON auth.webauthn_challenges(expires_at);

CREATE TABLE IF NOT EXISTS auth.oauth_states (
    state            TEXT PRIMARY KEY,
    provider         TEXT NOT NULL,
    user_id          UUID,
    tenant_id        UUID,
    pkce_verifier    TEXT,
    redirect_after   TEXT,
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_states_expires
    ON auth.oauth_states(expires_at);
`
