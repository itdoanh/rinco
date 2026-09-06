# Phần 10 – Database Schema & Polyglot Persistence

> **Phân hệ:** Toàn bộ cơ sở dữ liệu của hệ thống RINCO.  
> **Mục tiêu:** Mỗi service chọn database tối ưu cho access pattern, đảm bảo performance, ACID, isolation.  
> **Triết lý:** Polyglot Persistence – không ép buộc 1 DB cho mọi use case.

---

## Mục lục
1. [Tổng quan Polyglot Persistence](#1-tổng-quan-polyglot-persistence)
2. [PostgreSQL 17+ – CRM Core](#2-postgresql-17--crm-core)
3. [ScyllaDB – Chat & Events](#3-scylladb--chat--events)
4. [ClickHouse – Analytics](#4-clickhouse--analytics)
5. [MongoDB – Dynamic Forms](#5-mongodb--dynamic-forms)
6. [Valkey 9.x – Cache & State](#6-valkey-9x--cache--state)
7. [MinIO/S3 – Object Storage](#7-minios3--object-storage)
8. [Qdrant + pgvector – Vector Search](#8-qdrant--pgvector--vector-search)
9. [Meilisearch – Full-text Search](#9-meilisearch--full-text-search)
10. [Migration Strategy](#10-migration-strategy)
11. [Backup & DR](#11-backup--dr)
12. [Connection Pooling](#12-connection-pooling)

---

## 1. Tổng quan Polyglot Persistence

### 1.1. Ma trận quyết định

| Use case | Database | Lý do |
|----------|----------|-------|
| **CRM relational data** | PostgreSQL 17+ | ACID, JSONB, LTREE, pgvector |
| **Chat message history** | ScyllaDB | Shard-per-core, write > 1M/s |
| **Analytics & clickstream** | ClickHouse | Column-oriented, OLAP, nén 10:1 |
| **Dynamic form data** | MongoDB | Schema linh hoạt |
| **Cache & session** | Valkey 9.x | In-memory, multi-threaded |
| **Media files** | MinIO/S3 | Object storage, cheap |
| **Vector RAG** | Qdrant / pgvector | HNSW index |
| **Full-text search** | Meilisearch | Inverted index, typo-tolerant |
| **Audit log** | ScyllaDB | Append-only, time-series |
| **Tenant metadata** | PostgreSQL | Strong consistency |
| **Notification queue** | Valkey Streams | Lightweight log |
| **Meeting state (hot)** | Valkey | Real-time presence |

### 1.2. Read/Write Pattern

```
┌─────────────────────────────────────────────────────────┐
│                    WRITE PATH                            │
│  Landing Form → ScyllaDB (raw) → NATS → ClickHouse (async)│
│  CRM Update → PostgreSQL → NATS → Many subscribers       │
│  Chat Send → ScyllaDB + NATS → Broadcast                 │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    READ PATH                              │
│  Real-time → Valkey                                       │
│  Recent (7 days) → ScyllaDB                              │
│  Historical → ClickHouse                                 │
│  CRM Detail → PostgreSQL                                  │
│  Search → Meilisearch                                    │
│  Vector search → Qdrant / pgvector                       │
└─────────────────────────────────────────────────────────┘
```

---

## 2. PostgreSQL 17+ – CRM Core

### 2.1. Setup
```yaml
# postgresql.conf highlights
shared_buffers = 8GB
effective_cache_size = 24GB
work_mem = 64MB
maintenance_work_mem = 1GB
max_connections = 200  # Use PgBouncer in front
wal_level = replica
max_wal_senders = 10
random_page_cost = 1.1  # SSD
```

### 2.2. Extensions
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "ltree";           -- Tree structure
CREATE EXTENSION IF NOT EXISTS "pg_trgm";          -- Trigram fuzzy search
CREATE EXTENSION IF NOT EXISTS "pgvector";         -- Vector search
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "postgis";          -- Geo
CREATE EXTENSION IF NOT EXISTS "pg_partman";       -- Partition management
```

### 2.3. Schema Overview

#### Multi-tenancy Schema
```sql
-- Tenants
CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  plan TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  isolation_mode TEXT DEFAULT 'SHARED',
  region TEXT NOT NULL,
  vps_node_id UUID,
  settings JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

-- Users (CRM Org Tree)
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  email TEXT NOT NULL,
  phone TEXT,
  password_hash TEXT,
  full_name TEXT NOT NULL,
  avatar_url TEXT,
  parent_id UUID REFERENCES users(id),
  path LTREE NOT NULL,
  depth INT NOT NULL,
  role TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  mfa_enabled BOOLEAN DEFAULT false,
  last_login_at TIMESTAMPTZ,
  failed_login_count INT DEFAULT 0,
  locked_until TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_path ON users USING GIST(path);
CREATE INDEX idx_users_tenant_role ON users(tenant_id, role);
```

#### Lead Schema
```sql
CREATE TABLE leads (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  owner_user_id UUID REFERENCES users(id),
  name TEXT NOT NULL,
  phone TEXT,
  email TEXT,
  source TEXT,
  campaign_id UUID,
  utm_source TEXT,
  utm_medium TEXT,
  utm_campaign TEXT,
  utm_content TEXT,
  utm_term TEXT,
  fbclid TEXT,
  status TEXT NOT NULL DEFAULT 'new',
  score INT,
  quality_score TEXT,    -- 'high','medium','low'
  data JSONB DEFAULT '{}',
  consent_given BOOLEAN DEFAULT false,
  consent_at TIMESTAMPTZ,
  source_meta JSONB,     -- IP, UA, etc.
  crm_synced BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_leads_tenant_owner_status ON leads(tenant_id, owner_user_id, status);
CREATE INDEX idx_leads_phone ON leads(tenant_id, phone);
CREATE INDEX idx_leads_email ON leads(tenant_id, email);
CREATE INDEX idx_leads_data_gin ON leads USING GIN(data);
CREATE INDEX idx_leads_created_at ON leads(tenant_id, created_at DESC);
CREATE INDEX idx_leads_score ON leads(tenant_id, score DESC) WHERE score IS NOT NULL;
```

#### Deal Schema
```sql
CREATE TABLE deals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  lead_id UUID REFERENCES leads(id),
  owner_user_id UUID REFERENCES users(id),
  team_id UUID,
  pipeline_id UUID,
  stage TEXT NOT NULL DEFAULT 'new',
  title TEXT NOT NULL,
  value DECIMAL(18, 2),
  currency TEXT DEFAULT 'VND',
  probability INT,
  expected_close_date DATE,
  actual_close_date DATE,
  status TEXT NOT NULL DEFAULT 'OPEN',
  data JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_deals_tenant_stage ON deals(tenant_id, stage);
CREATE INDEX idx_deals_owner ON deals(tenant_id, owner_user_id);
CREATE INDEX idx_deals_pipeline ON deals(tenant_id, pipeline_id);
```

#### Activity Schema
```sql
CREATE TABLE activities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  user_id UUID REFERENCES users(id),
  lead_id UUID REFERENCES leads(id),
  deal_id UUID REFERENCES deals(id),
  type TEXT NOT NULL,    -- 'call','email','meeting','note','task'
  title TEXT,
  description TEXT,
  due_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  status TEXT DEFAULT 'pending',
  data JSONB,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_activities_user_due ON activities(user_id, due_at);
CREATE INDEX idx_activities_lead ON activities(lead_id);
```

#### Dynamic Schema Tables
```sql
-- Schema definitions
CREATE TABLE entity_definitions (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  entity_code TEXT NOT NULL,
  display_name TEXT NOT NULL,
  icon TEXT,
  fields JSONB NOT NULL,
  workflows JSONB,
  indexes JSONB,
  version INT DEFAULT 1,
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(tenant_id, entity_code)
);

-- Dynamic records (per tenant opt-in)
CREATE TABLE dynamic_records (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  entity_code TEXT NOT NULL,
  owner_user_id UUID,
  data JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_dynamic_records_gin ON dynamic_records USING GIN(data);
```

### 2.4. RLS Implementation
```sql
-- Enable RLS
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads FORCE ROW LEVEL SECURITY;

-- Tenant isolation
CREATE POLICY leads_tenant ON leads
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- User scope
CREATE POLICY leads_user_scope ON leads
  USING (
    owner_user_id = current_setting('app.current_user_id', true)::UUID
    OR owner_user_id IN (
      SELECT id FROM users
      WHERE path <@ (SELECT path FROM users WHERE id = current_setting('app.current_user_id', true)::UUID)
    )
    OR current_setting('app.is_super_admin', true) = 'true'
  );
```

### 2.5. Connection Pooling (PgBouncer)
```ini
[databases]
rinco = host=postgres port=5432 dbname=rinco
rinro_admin = host=postgres port=5432 dbname=rinco_admin

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
auth_type = scram-sha-256
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 5000
default_pool_size = 50
reserve_pool_size = 10
server_idle_timeout = 600
```

---

## 3. ScyllaDB – Chat & Events

### 3.1. Setup
```yaml
# scylla.yaml highlights
cluster_name: rinco_chat
num_tokens: 256
commitlog_total_space_in_mb: 4096
read_request_timeout_in_ms: 5000
write_request_timeout_in_ms: 2000
compaction_throughput_mb_per_sec: 200
```

### 3.2. Keyspaces
```cql
CREATE KEYSPACE rinco_chat 
WITH replication = {
  'class': 'NetworkTopologyStrategy',
  'datacenter1': 3
} AND durable_writes = true;

CREATE KEYSPACE rinco_events
WITH replication = {
  'class': 'NetworkTopologyStrategy',
  'datacenter1': 3
} AND durable_writes = true;

CREATE KEYSPACE rinco_audit
WITH replication = {
  'class': 'NetworkTopologyStrategy',
  'datacenter1': 3
} AND durable_writes = true;
```

### 3.3. Tables

#### Messages
```cql
CREATE TABLE rinco_chat.messages (
  channel_id text,
  tenant_id text,
  event_time bigint,
  event_id uuid,
  sender_id text,
  message_type tinyint,
  text text,
  metadata map<text, text>,
  attachments list<frozen<attachment>>,
  reactions map<text, frozen<reaction>>,
  reply_to text,
  mentions list<text>,
  pinned boolean,
  deleted boolean,
  edited_at bigint,
  PRIMARY KEY ((channel_id, tenant_id), event_time, event_id)
) WITH CLUSTERING ORDER BY (event_time DESC, event_id DESC)
  AND compaction = {'class': 'TimeWindowCompactionStrategy', 'compaction_window_size': '1', 'compaction_window_unit': 'DAYS'};

CREATE TABLE rinco_chat.messages_by_user (
  user_id text,
  tenant_id text,
  channel_id text,
  event_time bigint,
  event_id uuid,
  message_type tinyint,
  text text,
  PRIMARY KEY ((user_id, tenant_id), event_time, event_id)
) WITH CLUSTERING ORDER BY (event_time DESC);
```

#### Channels
```cql
CREATE TABLE rinco_chat.channels (
  channel_id text,
  tenant_id text,
  name text,
  type tinyint,
  description text,
  avatar_url text,
  members set<text>,
  admins set<text>,
  created_by text,
  created_at bigint,
  archived boolean,
  PRIMARY KEY ((channel_id, tenant_id))
);

CREATE TABLE rinco_chat.user_channels (
  user_id text,
  tenant_id text,
  channel_id text,
  joined_at bigint,
  last_read_event_time bigint,
  unread_count int,
  muted boolean,
  pinned boolean,
  PRIMARY KEY ((user_id, tenant_id), channel_id)
);
```

#### Audit Log
```cql
CREATE TABLE rinco_audit.audit_log (
  tenant_id text,
  created_at timestamp,
  id uuid,
  actor_id text,
  actor_email text,
  actor_type text,
  action text,
  resource_type text,
  resource_id text,
  ip_address inet,
  trace_id uuid,
  payload text,
  signature text,
  PRIMARY KEY ((tenant_id, created_at), id)
) WITH CLUSTERING ORDER BY (id DESC);
```

#### CAPI Events
```cql
CREATE TABLE rinco_events.capi_events (
  tenant_id text,
  sent_at timestamp,
  event_id uuid,
  fb_event_name text,
  status_code int,
  events_received int,
  fbtrace_id text,
  error_message text,
  PRIMARY KEY ((tenant_id, sent_at), event_id)
);
```

### 3.4. Compaction Strategy
- **TWCS (Time Window Compaction Strategy)** cho chat logs.
- Window = 1 day.
- Giảm write amplification.

---

## 4. ClickHouse – Analytics

### 4.1. Setup
```xml
<yandex>
    <listen_host>0.0.0.0</listen_host>
    <tcp_port>9000</tcp_port>
    <http_port>8123</http_port>
    <max_connections>4096</max_connections>
    <keep_alive_timeout>3</keep_alive_timeout>
    <max_concurrent_queries>100</max_concurrent_queries>
    <uncompressed_cache_size>8589934592</uncompressed_cache_size>
    <mark_cache_size>8589934592</mark_cache_size>
    <path>/var/lib/clickhouse/</path>
    <timezone>Asia/Ho_Chi_Minh</timezone>
    <remote_servers>
        <rinco_cluster>
            <shard>
                <internal_replication>true</internal_replication>
                <replica><host>node1</host><port>9000</port></replica>
                <replica><host>node2</host><port>9000</port></replica>
            </shard>
        </rinco_cluster>
    </remote_servers>
</yandex>
```

### 4.2. Database
```sql
CREATE DATABASE IF NOT EXISTS rinco_analytics;
```

### 4.3. Tables

#### Events Raw
```sql
CREATE TABLE rinco_analytics.events_raw (
  event_date Date MATERIALIZED toDate(timestamp),
  timestamp DateTime64(9),
  tenant_id LowCardinality(String),
  event_id UUID,
  session_id UUID,
  anonymous_id String,
  user_id String,
  event_name LowCardinality(String),
  source LowCardinality(String),
  utm_source LowCardinality(String),
  utm_campaign LowCardinality(String),
  utm_content String,
  utm_term String,
  fbclid String,
  gclid String,
  country LowCardinality(String),
  city String,
  device_type LowCardinality(String),
  browser LowCardinality(String),
  os LowCardinality(String),
  ip_address IPv4,
  user_agent String,
  referrer String,
  page_url String,
  properties JSON
) ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_date, event_name)
TTL event_date + INTERVAL 90 DAY;
```

#### Aggregated Events
```sql
CREATE TABLE rinco_analytics.events_aggregated (
  event_date Date,
  tenant_id LowCardinality(String),
  event_name LowCardinality(String),
  source LowCardinality(String),
  utm_source LowCardinality(String),
  utm_campaign LowCardinality(String),
  country LowCardinality(String),
  device_type LowCardinality(String),
  count UInt64,
  unique_users AggregateFunction(uniq, String),
  unique_sessions AggregateFunction(uniq, String),
  total_value AggregateFunction(sum, Decimal(18, 4))
) ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_date, event_name);
```

#### Funnel
```sql
CREATE TABLE rinco_analytics.funnel (
  event_date Date,
  tenant_id LowCardinality(String),
  campaign_id UUID,
  step LowCardinality(String),  -- 'page_view','cta_click','form_submit','qualified'
  count UInt64,
  unique_users AggregateFunction(uniq, String)
) ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_date, campaign_id, step);
```

### 4.4. Materialized Views
```sql
CREATE MATERIALIZED VIEW rinco_analytics.events_daily_mv
TO rinco_analytics.events_aggregated AS
SELECT 
  event_date,
  tenant_id,
  event_name,
  source,
  utm_source,
  utm_campaign,
  country,
  device_type,
  count() AS count,
  uniqState(user_id) AS unique_users,
  uniqState(session_id) AS unique_sessions,
  sumState(toDecimal64OrZero(JSONExtractFloat(properties, 'value'), 4)) AS total_value
FROM rinco_analytics.events_raw
GROUP BY event_date, tenant_id, event_name, source, utm_source, utm_campaign, country, device_type;
```

### 4.5. Per-Tenant User Setup
```sql
-- User per tenant
CREATE USER tenant_apex IDENTIFIED BY 'xxx';

-- Grants
GRANT SELECT ON rinco_analytics.* TO tenant_apex;

-- Row policy (filter by tenant_id automatically)
CREATE ROW POLICY tenant_apex_filter ON rinco_analytics.events_raw
  USING tenant_id = 'apex' TO tenant_apex;

CREATE ROW POLICY tenant_apex_agg_filter ON rinco_analytics.events_aggregated
  USING tenant_id = 'apex' TO tenant_apex;
```

---

## 5. MongoDB – Dynamic Forms

### 5.1. Use Cases
- Landing page forms (cấu trúc thay đổi liên tục).
- Survey responses.
- User-generated content (comments, descriptions).
- Logs có cấu trúc không cố định.

### 5.2. Setup
```yaml
# mongod.conf
storage:
  wiredTiger:
    engineConfig:
      cacheSizeGB: 4
replication:
  replSetName: rinco-rs
sharding:
  clusterRole: shardsvr
```

### 5.3. Collections

#### Form Submissions
```javascript
db.createCollection("form_submissions", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["tenant_id", "form_id", "submitted_at", "data"],
      properties: {
        tenant_id: { bsonType: "string" },
        form_id: { bsonType: "string" },
        submitted_at: { bsonType: "date" },
        ip_address: { bsonType: "string" },
        user_agent: { bsonType: "string" },
        data: { bsonType: "object" },
        meta: { bsonType: "object" }
      }
    }
  }
});

db.form_submissions.createIndex({ tenant_id: 1, form_id: 1, submitted_at: -1 });
db.form_submissions.createIndex({ "data.email": 1 });
db.form_submissions.createIndex({ "data.phone": 1 });
```

#### Form Definitions
```javascript
db.form_definitions.createIndex({ tenant_id: 1, slug: 1 }, { unique: true });
```

### 5.4. Sharding
```javascript
sh.enableSharding("rinco_forms");
sh.shardCollection("rinco_forms.form_submissions", { tenant_id: 1, submitted_at: 1 });
```

---

## 6. Valkey 9.x – Cache & State

### 6.1. Setup
```conf
# valkey.conf
port 6379
bind 0.0.0.0
protected-mode yes
requirepass xxx

maxmemory 8gb
maxmemory-policy allkeys-lru
io-threads 8
io-threads-do-reads yes

appendonly yes
appendfsync everysec
```

### 6.2. Use Cases & Keys

| Key pattern | Purpose | TTL |
|-------------|---------|-----|
| `domain:{hostname}` | Tenant resolution | 3600s |
| `tenant:{tid}:config` | Tenant settings | 600s |
| `user:{uid}:session` | Session data | 24h |
| `presence:user:{uid}` | Online status | 60s |
| `presence:channel:{cid}` | Online users in channel | 300s |
| `ratelimit:{tid}:{action}` | Rate limit counter | 60s |
| `singleflight:{key}` | Coalescing lock | 10s |
| `distlock:{name}` | Distributed lock | 30s |
| `meetsfu:room:{room_code}` | Meeting room state | 24h |
| `capi:emq:{tid}` | CAPI EMQ score | 3600s |
| `notif:queue:{uid}` | Notification pending | 7 days |
| `dashboard:cache:{tid}:{query}` | Dashboard query cache | 60s |

### 6.3. Cluster Setup
```yaml
# docker-compose
valkey-cluster:
  image: valkey/valkey:9
  command: valkey-server --cluster-enabled yes --cluster-config-file nodes.conf --appendonly yes
```

### 6.4. Streams (Lightweight Event Log)
```
# Publish
XADD rinco:events:lead * type "lead.created" data "{...}"

# Subscribe
XREAD BLOCK 0 STREAMS rinco:events:lead $
```

### 6.5. Pub/Sub
```
# Publish
PUBLISH chat:room:c-123 "{...}"

# Subscribe
SUBSCRIBE chat:room:c-123
```

---

## 7. MinIO/S3 – Object Storage

### 7.1. Setup
```yaml
# docker-compose
minio:
  image: minio/minio:latest
  command: server /data --console-address ":9001"
  environment:
    MINIO_ROOT_USER: minio
    MINIO_ROOT_PASSWORD: xxx
```

### 7.2. Buckets

| Bucket | Purpose | Retention |
|--------|---------|-----------|
| `rinco-tenant-assets` | Logo, favicon, custom CSS | Forever |
| `rinco-chat-files` | Chat attachments | 90 days |
| `rinco-user-avatars` | User profile pictures | Forever |
| `rinco-meeting-recings` | Meeting recordings | Configurable |
| `rinco-ai-models` | ML model artifacts | Forever |
| `rinco-backups` | DB backups | 30 days hot, 1 year cold |
| `rinco-cold-archive` | Old data archive | 7 years |

### 7.3. Bucket Policies
- Per-tenant prefix isolation.
- Lifecycle rules (auto-delete).
- Versioning enabled for critical buckets.

### 7.4. Presigned Upload
```go
func GeneratePresignedURL(bucket, key string, size int64, contentType string) (string, error) {
    req, _ := s3Client.PutObjectRequest(&s3.PutObjectInput{
        Bucket:      aws.String(bucket),
        Key:         aws.String(key),
        ContentType: aws.String(contentType),
        ContentLength: aws.Int64(size),
    })
    url, err := req.Presign(15 * time.Minute)
    return url, err
}
```

### 7.5. Erasure Coding
- MinIO default: EC:4 (4 data, 4 parity).
- Cost-optimal: EC:2.
- Performance: no EC (replicated).

---

## 8. Qdrant + pgvector – Vector Search

### 8.1. Khi nào dùng gì
| Quy mô | Dùng | Lý do |
|--------|------|-------|
| < 10M vectors | **pgvector + pgvectorscale** | Tận dụng cluster Postgres có sẵn |
| > 10M vectors | **Qdrant** | Rust-native, nhanh hơn 3-5x |

### 8.2. pgvector Setup
```sql
-- pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- pgvectorscale for DiskANN
CREATE EXTENSION IF NOT EXISTS vectorscale;

-- Documents table for RAG
CREATE TABLE knowledge_documents (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  collection TEXT NOT NULL,
  title TEXT,
  content TEXT,
  embedding vector(1024),  -- BGE-M3 dimension
  metadata JSONB,
  created_at TIMESTAMPTZ DEFAULT now()
);

-- HNSW index
CREATE INDEX idx_knowledge_hnsw ON knowledge_documents 
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- DiskANN for scale (pgvectorscale)
-- CREATE INDEX idx_knowledge_diskann ON knowledge_documents 
-- USING diskann (embedding vector_cosine_ops);
```

### 8.3. Qdrant Setup
```yaml
qdrant:
  image: qdrant/qdrant:latest
  environment:
    QDRANT__SERVICE__GRPC_PORT: 6334
  volumes:
    - qdrant-data:/qdrant/storage
```

### 8.4. Collections
```python
# Qdrant collection per tenant (or sharded single collection)
client.create_collection(
    collection_name="apex_kb",
    vectors_config={
        "size": 1024,
        "distance": "Cosine",
        "hnsw_config": {"m": 16, "ef_construct": 100}
    },
    quantization_config={"scalar": {"type": "int8"}},
    optimizers_config={"indexing_threshold": 20000}
)
```

### 8.5. Tenant Isolation
- One collection per tenant (small tenants).
- Sharded single collection (large tenants) with tenant_id payload filter.

```python
# Search with tenant filter
hits = client.search(
    collection_name="rinco_kb",
    query_vector=embedding,
    query_filter={
        "must": [{"key": "tenant_id", "match": {"value": "apex"}}]
    },
    limit=5
)
```

---

## 9. Meilisearch – Full-text Search

### 9.1. Setup
```yaml
meilisearch:
  image: getmeili/meilisearch:latest
  environment:
    MEILI_MASTER_KEY: xxx
    MEILI_ENV: production
    MEILI_NO_ANALYTICS: true
```

### 9.2. Indexes

```javascript
// Leads index
{
  "uid": "leads",
  "primaryKey": "id",
  "searchableAttributes": ["name", "phone", "email", "data.*"],
  "filterableAttributes": ["tenant_id", "status", "owner_user_id", "source"],
  "sortableAttributes": ["created_at", "score", "updated_at"]
}
```

### 9.3. Search API
```go
results, err := meili.Index("leads").Search("Nguyễn Văn A", &meilisearch.SearchRequest{
    Filter: "tenant_id = apex AND status = new",
    Sort:   []string{"created_at:desc"},
    Limit:  20,
})
```

---

## 10. Migration Strategy

### 10.1. Schema Versioning
```sql
CREATE TABLE schema_migrations (
  version TEXT PRIMARY KEY,
  description TEXT,
  applied_at TIMESTAMPTZ DEFAULT now()
);
```

### 10.2. Migration Tool (Goose)
```sql
-- +goose Up
CREATE TABLE leads (...);

-- +goose Down
DROP TABLE leads;
```

### 10.3. Workflow
```
1. Dev viết migration SQL
2. Review PR
3. Apply lên staging → test
4. Apply lên prod qua tool zero-downtime
5. Rollback nếu fail (forward fix preferred)
```

### 10.4. Zero-downtime Migration Pattern
```sql
-- Step 1: Add new column nullable
ALTER TABLE leads ADD COLUMN new_column TEXT;

-- Step 2: Backfill in batches
UPDATE leads SET new_column = old_column WHERE id IN (...);

-- Step 3: Code switch to new column
-- (deploy app reading from new_column)

-- Step 4: Drop old column
ALTER TABLE leads DROP COLUMN old_column;
```

---

## 11. Backup & DR

### 11.1. PostgreSQL Backup
```bash
# Full backup
pg_dump --format=custom --compress=9 rinco > backup.dump

# Continuous archiving (WAL)
archive_command = 'cp %p /backup/wal/%f'

# Backup to S3
pg_dump --format=custom | aws s3 cp - s3://rinco-backups/pg/$(date +%Y%m%d).dump
```

### 11.2. ScyllaDB Backup
```bash
# Snapshot
nodetool snapshot rinco_chat

# Upload to S3
sstableloader -d /path/to/snapshot s3://rinco-backups/scylla/
```

### 11.3. ClickHouse Backup
```bash
# Native backup
clickhouse-backup create --config=/etc/clickhouse-backup/config.yml

# Upload to S3
clickhouse-backup upload default --config=/etc/clickhouse-backup/config.yml
```

### 11.4. RPO/RTO Targets
| Database | RPO | RTO |
|----------|-----|-----|
| PostgreSQL | 5 phút | 30 phút |
| ScyllaDB | 1 phút | 15 phút |
| ClickHouse | 1 giờ | 1 giờ |
| Valkey | 0 (no data loss) | 30 giây |
| MinIO | 0 (replication) | 1 phút |

### 11.5. Disaster Recovery Drill
- Mỗi quý 1 lần.
- Restore to isolated environment.
- Run smoke tests.
- Document timing.

---

## 12. Connection Pooling

### 12.1. PgBouncer
- Pool mode: **transaction** (default).
- Max client conn: 5000.
- Default pool size: 50.
- Auth: SCRAM-SHA-256.

### 12.2. ScyllaDB Driver
- Token-aware load balancing.
- Per-host connection pool (Local: 1, Remote: 1).

### 12.3. ClickHouse HTTP Pool
- HTTP keep-alive.
- Pool size: 100 connections.

### 12.4. Valkey
- Connection pool per service.
- Use `redis-rs` or `fred` (async Rust).

### 12.5. Database per Service
- **crm-core:** PostgreSQL.
- **chat-engine:** ScyllaDB + Valkey.
- **landing-ingest:** ScyllaDB + ClickHouse.
- **analytics:** ClickHouse.
- **auth:** PostgreSQL.
- **notification:** PostgreSQL + Valkey.
- **ai-services:** PostgreSQL + Qdrant + Valkey.

---

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-DB-01 | PostgreSQL query p99 < 50ms | pgbench |
| AC-DB-02 | ScyllaDB write p99 < 10ms | cassandra-stress |
| AC-DB-03 | ClickHouse OLAP p99 < 1s trên 1B row | clickhouse-benchmark |
| AC-DB-04 | Valkey ops p99 < 1ms | redis-benchmark |
| AC-DB-05 | RPO < 5 phút | Backup test |

---

**Tiếp theo:** [`docs/11-ai-integration/README.md`](../11-ai-integration/README.md) – Tích hợp AI toàn hệ thống.