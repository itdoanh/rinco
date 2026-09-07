-- Migration 0006: ClickHouse events metadata table
-- =============================================================================

CREATE TABLE IF NOT EXISTS clickhouse_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    lead_id UUID,
    user_id UUID,
    properties JSONB DEFAULT '{}',
    event_date DATE NOT NULL DEFAULT CURRENT_DATE,
    synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clickhouse_events_tenant ON clickhouse_events(tenant_id);
CREATE INDEX idx_clickhouse_events_lead ON clickhouse_events(lead_id);
CREATE INDEX idx_clickhouse_events_type ON clickhouse_events(tenant_id, event_type);
CREATE INDEX idx_clickhouse_events_date ON clickhouse_events(event_date);

ALTER TABLE clickhouse_events ENABLE ROW LEVEL SECURITY;

CREATE POLICY clickhouse_events_tenant_isolation ON clickhouse_events
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- MongoDB metadata reference table
CREATE TABLE IF NOT EXISTS mongodb_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    mongo_doc_id TEXT NOT NULL,
    collection_name TEXT NOT NULL DEFAULT 'lead_metadata',
    synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(lead_id, collection_name)
);

CREATE INDEX idx_mongodb_metadata_tenant ON mongodb_metadata(tenant_id);
CREATE INDEX idx_mongodb_metadata_lead ON mongodb_metadata(lead_id);
