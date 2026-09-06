-- ============================================================
-- Events — append-only audit / business event log
-- ============================================================

USE events;

-- Generic event stream, partitioned by (tenant, day bucket) so
-- partitions stay bounded. Append-only.
CREATE TABLE IF NOT EXISTS events.event_log (
    tenant_id         text,
    event_date        date,
    event_id          uuid,
    event_time        timestamp,
    event_name        text,            -- 'lead.created','deal.won',...
    actor_user_id     uuid,
    actor_type        text,            -- 'user','system','integration'
    entity_type       text,
    entity_id         uuid,
    correlation_id    uuid,
    payload           text,            -- JSON-encoded
    source_service    text,
    PRIMARY KEY ((tenant_id, event_date), event_time, event_id)
) WITH CLUSTERING ORDER BY (event_time DESC, event_id DESC)
  AND default_time_to_live = 31536000  -- 365 days
  AND compaction = {
    'class': 'TimeWindowCompactionStrategy',
    'compaction_window_size': '1',
    'compaction_window_unit': 'DAYS'
  }
  AND comment = 'Append-only event log, 365-day retention.';

-- Lead-ingress raw stream (landing page submissions).
CREATE TABLE IF NOT EXISTS events.landing_submissions (
    tenant_id        text,
    submit_date      date,
    event_id         uuid,
    event_time       timestamp,
    full_name        text,
    phone            text,
    email            text,
    utm              map<text, text>,
    fbclid           text,
    gclid            text,
    idempotency_key  text,
    raw_json         text,
    processed        boolean,
    lead_id          uuid,
    PRIMARY KEY ((tenant_id, submit_date), event_time, event_id)
) WITH CLUSTERING ORDER BY (event_time DESC, event_id DESC)
  AND comment = 'Landing page submissions, append-only.';

-- Webhook delivery log (for fan-out debugging).
CREATE TABLE IF NOT EXISTS events.webhook_deliveries (
    tenant_id        text,
    delivery_date    date,
    delivery_id      uuid,
    webhook_id       uuid,
    target_url       text,
    event_name       text,
    request_body     text,
    response_status  int,
    response_body    text,
    attempt_count    int,
    delivered_at     timestamp,
    PRIMARY KEY ((tenant_id, delivery_date), delivered_at, delivery_id)
) WITH CLUSTERING ORDER BY (delivered_at DESC, delivery_id DESC)
  AND comment = 'Webhook delivery attempts (debug + dedup).';
