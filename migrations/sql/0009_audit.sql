-- +goose Up
-- +goose StatementBegin
-- ============================================================
-- Migration 0009 — audit log + login history
-- ============================================================

CREATE TABLE IF NOT EXISTS audit.audit_logs (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       UUID REFERENCES tenant.tenants(id),
    actor_user_id   UUID REFERENCES auth.users(id),
    actor_ip        INET,
    action          TEXT NOT NULL,
    entity_type     TEXT NOT NULL,
    entity_id       TEXT,
    before          JSONB,
    after           JSONB,
    request_id      TEXT,
    trace_id        TEXT,
    user_agent      TEXT,
    extra           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_time   ON audit.audit_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_entity        ON audit.audit_logs(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_actor         ON audit.audit_logs(actor_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action        ON audit.audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_tenant_action ON audit.audit_logs(tenant_id, action, created_at DESC);

-- Login history (separate table for tight retention)
CREATE TABLE IF NOT EXISTS audit.login_history (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       UUID,
    user_id         UUID REFERENCES auth.users(id),
    email           CITEXT,
    success         BOOLEAN NOT NULL,
    failure_reason  TEXT,
    ip_address      INET,
    user_agent      TEXT,
    mfa_used        BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_login_history_user     ON audit.login_history(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_login_history_tenant   ON audit.login_history(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_login_history_failures ON audit.login_history(created_at DESC) WHERE success = false;

-- API request log (for rate-limit / cost analysis)
CREATE TABLE IF NOT EXISTS audit.api_requests (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       UUID,
    user_id         UUID REFERENCES auth.users(id),
    service         TEXT NOT NULL,
    method          TEXT NOT NULL,
    route           TEXT NOT NULL,
    status          INT  NOT NULL,
    duration_ms     INT  NOT NULL,
    bytes_in        INT,
    bytes_out       INT,
    ip_address      INET,
    user_agent      TEXT,
    trace_id        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_api_req_tenant_time ON audit.api_requests(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_req_user_time   ON audit.api_requests(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_req_status      ON audit.api_requests(status, created_at DESC);

-- ============================================================
-- RLS
-- ============================================================
-- Audit logs are ONLY readable by super-admin. Services bypass RLS
-- via SET LOCAL app.is_super_admin = 'true' for writes.
ALTER TABLE audit.audit_logs      ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit.audit_logs      FORCE  ROW LEVEL SECURITY;
ALTER TABLE audit.login_history   ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit.login_history   FORCE  ROW LEVEL SECURITY;
ALTER TABLE audit.api_requests    ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit.api_requests    FORCE  ROW LEVEL SECURITY;

CREATE POLICY audit_super_admin ON audit.audit_logs
    FOR ALL
    USING (app.bypass_rls_check())
    WITH CHECK (app.bypass_rls_check());

-- Tenants may request their own audit log via a service-mediated call
CREATE POLICY audit_tenant_read ON audit.audit_logs
    FOR SELECT
    USING (
        app.bypass_rls_check()
        OR tenant_id = app.current_tenant_id()
    );

CREATE POLICY login_history_super ON audit.login_history
    FOR ALL
    USING (app.bypass_rls_check())
    WITH CHECK (app.bypass_rls_check());

CREATE POLICY api_requests_super ON audit.api_requests
    FOR ALL
    USING (app.bypass_rls_check())
    WITH CHECK (app.bypass_rls_check());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit.api_requests     CASCADE;
DROP TABLE IF EXISTS audit.login_history    CASCADE;
DROP TABLE IF EXISTS audit.audit_logs       CASCADE;
-- +goose StatementEnd
