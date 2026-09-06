---
name: rinco-database
description: Skill về chọn database, schema design, migration cho RINCO.
---

# RINCO Database Skill

## Decision Matrix

| Use case | Database | Lý do |
|----------|----------|-------|
| **CRM relational** | PostgreSQL 17 | ACID, JSONB, LTREE, pgvector |
| **Chat logs** | ScyllaDB | Shard-per-core, write > 1M/s |
| **Analytics OLAP** | ClickHouse | Column-oriented, nén 10:1 |
| **Dynamic forms** | MongoDB | Schema linh hoảng |
| **Cache/Session** | Valkey 9.x | In-memory, multi-threaded |
| **Files** | MinIO/S3 | Object storage |
| **Vector < 10M** | pgvector | Tận dụng Postgres |
| **Vector > 10M** | Qdrant | Rust-native, nhanh |
| **Search** | Meilisearch | Full-text Tiếng Việt |

## PostgreSQL Conventions

### Naming
- Table: `snake_case`, số ít.
- Column: `snake_case`.
- Primary key: `id UUID`.
- Foreign key: `<table>_id`.
- Timestamp: `<action>_at`.

### Standard Columns (MỌI table)
```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
tenant_id UUID NOT NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
deleted_at TIMESTAMPTZ  -- Soft delete
```

### Indexes Strategy
```sql
-- B-Tree cho equality + range
CREATE INDEX idx_table_column ON table(column);

-- GIN cho JSONB
CREATE INDEX idx_table_data_gin ON table USING GIN(data);

-- GIST cho LTREE
CREATE INDEX idx_users_path ON users USING GIST(path);

-- Partial index cho filter phổ biến
CREATE INDEX idx_leads_score ON leads(tenant_id, score DESC) WHERE score IS NOT NULL;

-- Composite index cho query phổ biến
CREATE INDEX idx_leads_tenant_owner_status ON leads(tenant_id, owner_user_id, status);
```

### Multi-tenancy
```sql
-- LUÔN filter theo tenant_id
SELECT * FROM leads WHERE tenant_id = $1 AND status = 'new';

-- RLS bắt buộc
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads FORCE ROW LEVEL SECURITY;

CREATE POLICY leads_tenant ON leads
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
```

### Migrations (Goose)
```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE leads (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  -- ...
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE leads;
```

### Zero-Downtime Migration
```sql
-- Step 1: Add nullable
ALTER TABLE leads ADD COLUMN new_field TEXT;

-- Step 2: Backfill in batches
DO $$
DECLARE
  last_id UUID;
BEGIN
  LOOP
    UPDATE leads SET new_field = old_field
    WHERE id > COALESCE(last_id, '00000000-0000-0000-0000-000000000000'::UUID)
      AND new_field IS NULL
    LIMIT 1000
    RETURNING id INTO last_id;
    EXIT WHEN last_id IS NULL;
    COMMIT;
  END LOOP;
END $$;

-- Step 3: Deploy code đọc từ new_field

-- Step 4: Drop old column
ALTER TABLE leads DROP COLUMN old_field;
```

## ScyllaDB Conventions

### Partition Key = Tenant Scope
```cql
CREATE TABLE messages (
  channel_id text,
  tenant_id text,        -- BẮT BUỘC
  event_time bigint,
  ...
  PRIMARY KEY ((channel_id, tenant_id), event_time, event_id)
);
```

### Compaction
- TWCS (Time Window Compaction Strategy) cho logs.
- Window = 1 day.

### Query Patterns
- Luôn có `tenant_id` trong WHERE.
- Tránh ALLOW FILTERING.

## ClickHouse Conventions

### MergeTree cho raw events
```sql
CREATE TABLE events_raw (
  timestamp DateTime64(9),
  event_date Date MATERIALIZED toDate(timestamp),
  tenant_id LowCardinality(String),
  ...
) ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_date, event_name);
```

### AggregatingMergeTree cho aggregated
```sql
CREATE TABLE events_aggregated (
  ...
  count UInt64,
  unique_users AggregateFunction(uniq, String),
) ENGINE = AggregatingMergeTree
ORDER BY (tenant_id, event_date);
```

### Per-tenant Row Policy
```sql
CREATE USER tenant_apex IDENTIFIED BY 'xxx';
CREATE ROW POLICY tenant_apex_filter ON events_raw
  USING tenant_id = 'apex' TO tenant_apex;
```

## Valkey Conventions

### Key Format
```
<scope>:<tenant_id>:<resource>:<id>

Ví dụ:
  domain:apex-fintech:config
  presence:user:apex-fintech:uuid-xxx
  ratelimit:apex-fintech:api-call
```

### TTL Strategy
- Session: 24h.
- Token bucket: 60s.
- Tenant config: 600s.
- Domain map: 3600s.

### Streams for Events
```bash
# Publish
XADD rinco:events:lead * type "lead.created" data "{...}"

# Subscribe
XREAD BLOCK 0 STREAMS rinco:events:lead $
```

## MongoDB Conventions

### Validator
```javascript
db.createCollection("form_submissions", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["tenant_id", "form_id", "data"],
      properties: {
        tenant_id: { bsonType: "string" },
        // ...
      }
    }
  }
});
```

### Sharding
```javascript
sh.enableSharding("rinco_forms");
sh.shardCollection("rinco_forms.form_submissions", { tenant_id: 1, submitted_at: 1 });
```

## Backup Strategy

| DB | Frequency | RPO | Storage |
|----|-----------|-----|---------|
| PostgreSQL | Daily + WAL continuous | 5 min | MinIO backup bucket |
| ScyllaDB | Daily snapshot | 1 hour | MinIO backup bucket |
| ClickHouse | Daily | 1 hour | MinIO backup bucket |
| Valkey | AOF continuous | 0 | Local + MinIO |
| MinIO | Replication | 0 | Cross-region |

## Performance Tips

1. **Index everything** trong WHERE, JOIN, ORDER BY.
2. **Connection pooling** qua PgBouncer.
3. **Read replicas** cho PostgreSQL khi read-heavy.
4. **Materialized views** cho ClickHouse aggregated queries.
5. **Pagination** mọi list API (cursor-based tốt hơn offset).
6. **Batch operations** thay vì N+1 queries.
7. **Async** cho non-blocking I/O.
8. **Cache hot data** ở Valkey với TTL phù hợp.