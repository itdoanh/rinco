-- 0001_alerts.sql
-- Creates observability schema, alerts + alert_history, audit_logs and
-- service_health_cache.  Applied by the platform.RunMigrations helper.

CREATE SCHEMA IF NOT EXISTS observability;

CREATE TABLE IF NOT EXISTS observability.alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint TEXT UNIQUE,
    status TEXT NOT NULL DEFAULT 'firing' CHECK (status IN ('firing','resolved','ack')),
    severity TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('critical','warning','info')),
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    annotations JSONB NOT NULL DEFAULT '{}'::jsonb,
    service TEXT,
    tenant_id UUID,
    title TEXT NOT NULL,
    message TEXT,
    fired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    ack_by UUID,
    ack_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON observability.alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_service ON observability.alerts(service);
CREATE INDEX IF NOT EXISTS idx_alerts_tenant ON observability.alerts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_alerts_fired_at ON observability.alerts(fired_at DESC);

CREATE TABLE IF NOT EXISTS observability.alert_history (
    id BIGSERIAL PRIMARY KEY,
    alert_id UUID NOT NULL REFERENCES observability.alerts(id) ON DELETE CASCADE,
    event TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_alert_history_alert ON observability.alert_history(alert_id, ts DESC);

CREATE TABLE IF NOT EXISTS observability.audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    actor_user_id UUID,
    actor_ip TEXT,
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_tenant ON observability.audit_logs(tenant_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action ON observability.audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON observability.audit_logs(actor_user_id);

CREATE TABLE IF NOT EXISTS observability.service_health_cache (
    service TEXT PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'unknown',
    error_rate DOUBLE PRECISION,
    p99_latency DOUBLE PRECISION,
    active_alerts INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);