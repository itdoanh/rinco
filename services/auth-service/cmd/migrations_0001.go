package main

// migration0001: Initial bootstrap — tenant.tenants, auth.users,
// auth.password_resets, auth.email_verifications. Idempotent CREATE IF NOT
// EXISTS so it can run on top of the platform-wide schema.
const migration0001 = `
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS tenant;

CREATE TABLE IF NOT EXISTS tenant.tenants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    display_name    TEXT,
    status          TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','suspended','trial','expired','pending')),
    plan            TEXT NOT NULL DEFAULT 'starter'
                        CHECK (plan IN ('starter','pro','enterprise','custom')),
    settings_json   JSONB NOT NULL DEFAULT '{}'::jsonb,
    branding_json   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    suspended_at    TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug
    ON tenant.tenants(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_status
    ON tenant.tenants(status) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS auth.users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    email               TEXT NOT NULL,
    password_hash       TEXT,
    full_name           TEXT NOT NULL DEFAULT '',
    avatar_url          TEXT,
    phone               TEXT,
    email_verified      BOOLEAN NOT NULL DEFAULT FALSE,
    status              TEXT NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active','pending','suspended','locked','deleted')),
    is_super_admin      BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_enabled         BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_secret          TEXT,
    last_login_at       TIMESTAMPTZ,
    last_login_ip       INET,
    failed_login_count  INTEGER NOT NULL DEFAULT 0,
    locked_until        TIMESTAMPTZ,
    metadata            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, email)
);

CREATE INDEX IF NOT EXISTS idx_users_email ON auth.users(LOWER(email));
CREATE INDEX IF NOT EXISTS idx_users_tenant ON auth.users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_status ON auth.users(status);

CREATE TABLE IF NOT EXISTS auth.password_resets (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    used_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_resets_user
    ON auth.password_resets(user_id);
CREATE INDEX IF NOT EXISTS idx_password_resets_expires
    ON auth.password_resets(expires_at);

CREATE TABLE IF NOT EXISTS auth.email_verifications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    email        TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    consumed_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_verif_user
    ON auth.email_verifications(user_id);
`
