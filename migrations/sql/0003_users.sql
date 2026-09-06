-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0003 — users + CRM hierarchy (LTREE)
-- ============================================================

CREATE TABLE IF NOT EXISTS auth.users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenant.tenants(id) ON DELETE CASCADE,
    email               CITEXT NOT NULL,
    password_hash       TEXT,
    full_name           TEXT NOT NULL,
    avatar_url          TEXT,
    phone               TEXT,
    parent_id           UUID REFERENCES auth.users(id),
    path                LTREE NOT NULL DEFAULT 'root'::ltree,
    depth               INT   NOT NULL DEFAULT 0,
    role                TEXT NOT NULL DEFAULT 'member'
                        CHECK (role IN ('super_admin','tenant_admin','manager','member','guest')),
    roles               TEXT[] NOT NULL DEFAULT ARRAY['member']::TEXT[],
    permissions         JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_super_admin      BOOLEAN NOT NULL DEFAULT false,
    fido2_credentials   JSONB NOT NULL DEFAULT '[]'::jsonb,
    mfa_enabled         BOOLEAN NOT NULL DEFAULT false,
    last_login_at       TIMESTAMPTZ,
    failed_login_count  INT NOT NULL DEFAULT 0,
    locked_until        TIMESTAMPTZ,
    metadata            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    UNIQUE(tenant_id, email)
);

CREATE INDEX IF NOT EXISTS idx_users_tenant       ON auth.users(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_path_gist    ON auth.users USING GIST(path);
CREATE INDEX IF NOT EXISTS idx_users_path_btree   ON auth.users USING BTREE(path);
CREATE INDEX IF NOT EXISTS idx_users_email_trgm   ON auth.users USING GIN(email gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_super_admin  ON auth.users(is_super_admin) WHERE is_super_admin = true;
CREATE INDEX IF NOT EXISTS idx_users_parent       ON auth.users(parent_id);

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- Refresh tokens
CREATE TABLE IF NOT EXISTS auth.refresh_tokens (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token_hash          TEXT NOT NULL,
    device_fingerprint  TEXT,
    ip_address          INET,
    user_agent          TEXT,
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user        ON auth.refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash  ON auth.refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires     ON auth.refresh_tokens(expires_at);

-- FIDO2 challenges (anti-replay)
CREATE TABLE IF NOT EXISTS auth.fido2_challenges (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    challenge    TEXT NOT NULL,
    type         TEXT NOT NULL CHECK (type IN ('reg','auth')),
    expires_at   TIMESTAMPTZ NOT NULL,
    consumed_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_fido2_challenges_user    ON auth.fido2_challenges(user_id);
CREATE INDEX IF NOT EXISTS idx_fido2_challenges_expires ON auth.fido2_challenges(expires_at);

-- Passkey credentials (long-term)
CREATE TABLE IF NOT EXISTS auth.fido2_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    credential_id   TEXT UNIQUE NOT NULL,
    public_key      TEXT NOT NULL,
    counter         BIGINT NOT NULL DEFAULT 0,
    transports      TEXT[],
    aaguid          TEXT,
    name            TEXT,
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_fido2_credentials_user ON auth.fido2_credentials(user_id);

-- ============================================================
-- RLS
-- ============================================================
ALTER TABLE auth.users              ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.users              FORCE  ROW LEVEL SECURITY;
ALTER TABLE auth.refresh_tokens     ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.refresh_tokens     FORCE  ROW LEVEL SECURITY;
ALTER TABLE auth.fido2_credentials  ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.fido2_credentials  FORCE  ROW LEVEL SECURITY;
ALTER TABLE auth.fido2_challenges   ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.fido2_challenges   FORCE  ROW LEVEL SECURITY;

-- Tenant isolation (every query must match tenant or be super-admin)
CREATE POLICY users_tenant ON auth.users
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

-- Hierarchy: see self, all descendants (LTREE <@), or be super-admin
CREATE POLICY users_subtree ON auth.users
    FOR SELECT
    USING (
        app.bypass_rls_check()
        OR id = app.current_user_id()
        OR path <@ (SELECT path FROM auth.users WHERE id = app.current_user_id())
    );

CREATE POLICY refresh_tokens_own ON auth.refresh_tokens
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR user_id = app.current_user_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR user_id = app.current_user_id()
    );

CREATE POLICY fido2_credentials_own ON auth.fido2_credentials
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR user_id = app.current_user_id()
    )
    WITH CHECK (
        app.bypass_rls_check()
        OR user_id = app.current_user_id()
    );

CREATE POLICY fido2_challenges_own ON auth.fido2_challenges
    FOR ALL
    USING (
        app.bypass_rls_check()
        OR user_id = app.current_user_id()
        OR user_id IS NULL
    );

-- Helper function: list subtree users
CREATE OR REPLACE FUNCTION auth.get_subtree_users(p_user_id UUID)
RETURNS TABLE(id UUID, email CITEXT, full_name TEXT, role TEXT, depth INT) AS $$
    SELECT id, email, full_name, role, depth
    FROM auth.users
    WHERE deleted_at IS NULL
      AND path <@ (SELECT path FROM auth.users WHERE id = p_user_id)
    ORDER BY path;
$$ LANGUAGE SQL STABLE;

-- Helper function: move subtree (transactional, cycle-detected)
CREATE OR REPLACE FUNCTION auth.move_subtree(
  p_user_id        UUID,
  p_new_parent_id  UUID
) RETURNS VOID AS $$
DECLARE
  v_old_path       LTREE;
  v_new_parent     LTREE;
  v_old_subtree    LTREE;
BEGIN
  SELECT path INTO v_old_path FROM auth.users WHERE id = p_user_id FOR UPDATE;
  SELECT path INTO v_new_parent FROM auth.users WHERE id = p_new_parent_id FOR UPDATE;

  IF v_new_parent <@ v_old_path THEN
    RAISE EXCEPTION 'cycle_detected';
  END IF;

  v_old_subtree := v_old_path;
  UPDATE auth.users
    SET path   = v_new_parent || subpath(path, nlevel(v_old_path)),
        depth  = depth + nlevel(v_new_parent) - nlevel(v_old_path),
        updated_at = NOW()
  WHERE path <@ v_old_subtree AND deleted_at IS NULL;
END;
$$ LANGUAGE plpgsql;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS auth.fido2_credentials CASCADE;
DROP TABLE IF EXISTS auth.fido2_challenges  CASCADE;
DROP TABLE IF EXISTS auth.refresh_tokens    CASCADE;
DROP TABLE IF EXISTS auth.users             CASCADE;
DROP FUNCTION IF EXISTS auth.move_subtree;
DROP FUNCTION IF EXISTS auth.get_subtree_users(UUID);
-- +goose StatementEnd
