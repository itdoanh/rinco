# Phần 10 — Database Schema & Polyglot Persistence (Tóm tắt)

> Tóm tắt tài liệu `docs/10-database/README.md` (v1.2) — Triết lý **Polyglot Persistence**: mỗi service chọn database tối ưu cho access pattern của mình, không ép một DB cho mọi use case.

---

## 1. Tổng quan — 9 Database Engines

RINCO sử dụng 9 engine khác nhau, mỗi engine phục vụ một nhóm use case:

| # | Engine | Use case chính | Lý do chọn |
|---|--------|----------------|------------|
| 1 | **PostgreSQL 17+** | CRM relational data (auth, tenant, crm, lead, deal, activity, dynamic_model) | ACID, JSONB, LTREE, pgvector, RLS |
| 2 | **ScyllaDB** | Chat message history, audit log, CAPI events, leads_raw, WebRTC/recording metadata | Shard-per-core, write > 1M/s, time-series |
| 3 | **MongoDB** | Landing page form submissions, survey responses, form definitions | Schema linh hoạt, document model |
| 4 | **ClickHouse** | Analytics, observability, audit warehouse, lead events | Column-oriented, OLAP, nén 10:1 |
| 5 | **Valkey 9.x** | Cache, sessions, presence, rate-limit, distributed locks, streams | In-memory, multi-threaded (fork Redis) |
| 6 | **MinIO/S3** | Object storage (assets, avatars, recordings, backups, cold archive) | Cheap, S3-compatible, EC erasure coding |
| 7 | **Qdrant** | Vector search (RAG > 10M vectors) | Rust-native HNSW, nhanh hơn pgvector 3-5× |
| 8 | **pgvector + pgvectorscale** | Vector search nhỏ (< 10M vectors) | Tận dụng cluster Postgres có sẵn |
| 9 | **Meilisearch** | Full-text search (leads, contacts, knowledge) | Inverted index, typo-tolerant, Tiếng Việt tốt |
| — | **PgBouncer** | Connection pooling cho Postgres | Transaction pooling, 5000 clients → 50 servers |
| — | **NATS JetStream** | Event bus giữa các services | Lightweight, durable, replay |

**Write path:** Landing Form → ScyllaDB (raw) → NATS → ClickHouse (async); CRM Update → PostgreSQL → NATS → many subscribers; Chat Send → ScyllaDB + NATS → broadcast.

**Read path:** Real-time → Valkey; Recent (7 days) → ScyllaDB; Historical → ClickHouse; CRM Detail → PostgreSQL; Search → Meilisearch; Vector → Qdrant/pgvector.

### Ma trận khả năng mở rộng theo phase

| Database | Q1 (1K tenant) | Q2 (10K tenant) | Q3 (100K tenant) | Q4 (1M tenant) |
|----------|----------------|------------------|-------------------|------------------|
| PostgreSQL | 1P + 2R | 1P + 4R + PgCat | 3 shards + 6R + PgBouncer | 6 shards/region + Citus |
| ScyllaDB | 3 node (RF=3) | 6 node | 12 node | 24 node multi-DC |
| ClickHouse | 1 shard × 2R | 3 × 2 | 6 × 2 | Cloud-only |
| MongoDB | 3 node RS | 6 node sharded | 12 node | 24 node |
| Valkey | 3M + 3R | 6M + 6R | 12+12 | 24+24 |
| MinIO | 4 node EC:4 | 8 node EC:4 | 16 node EC:4 | Cross-region replication |
| Qdrant | 1 node | 3 node | 6 node | 12 node |
| Meilisearch | 1 node | 2 node | 4 node | 6 node |

---

## 2. PostgreSQL — Schemas chính

PostgreSQL là **CRM Core**. Extensions: `uuid-ossp`, `pgcrypto`, `ltree`, `pg_trgm`, `pgvector`, `vectorscale` (DiskANN), `pg_stat_statements`, `postgis`, `pg_partman`, `hstore`, `pg_cron`, `pg_repack`. Tuning SSD: `shared_buffers=8GB`, `effective_cache_size=24GB`, `wal_compression=zstd`, `random_page_cost=1.1`.

### 2.1. Auth & Identity (`tenants`, `users`, `tenant_domains`)

- **`tenants`** — multi-tenancy root: `id`, `slug UNIQUE`, `plan` (free/starter/business/enterprise), `status` (ACTIVE/SUSPENDED/ARCHIVED), `isolation_mode` (SHARED/ISOLATED_VPS), `region`, `vps_node_id`, `settings JSONB`, `created_at`, `deleted_at`. Indexes: `idx_tenants_status WHERE deleted_at IS NULL`, `idx_tenants_plan`.
- **`tenant_domains`** — hostname routing: `hostname UNIQUE`, `routing` (subpath/subdomain/custom), `verified`, `tls_cert_id`. Self-reference `tenants(id) ON DELETE CASCADE`.
- **`users`** — CRM org tree dùng **LTREE**:
  - `path LTREE NOT NULL`, `depth INT NOT NULL`, `parent_id REFERENCES users(id)` (self-FK).
  - `email`, `phone`, `password_hash`, `full_name`, `avatar_url`, `role`, `status`.
  - `mfa_enabled`, `last_login_at`, `failed_login_count`, `locked_until` (account lockout).
  - `UNIQUE(tenant_id, email)`.
  - **Indexes:** `idx_users_path GIST(path)`, `idx_users_tenant_role`, `idx_users_email_lower(LOWER(email))`, `idx_users_parent`.
  - Function `move_subtree(p_user_id, p_new_parent_id)` chuyển nhánh cây với cycle detection (`v_new_parent_path <@ v_old_path` → RAISE 'cycle_detected'). Function `get_subtree_users(p_user_id)` lấy cây con dùng `path <@ subquery`. Function `get_tenant_metrics(p_tenant_id, p_days)` aggregate dashboard.

### 2.2. CRM Core (`leads`, `deals`, `activities`)

- **`leads`** — bảng trung tâm:
  - Identifiers: `id`, `tenant_id`, `owner_user_id REFERENCES users(id)`.
  - Thông tin: `name`, `phone`, `email`, `source`, `campaign_id`.
  - UTM tracking: `utm_source`, `utm_medium`, `utm_campaign`, `utm_content`, `utm_term`, `fbclid`, `gclid`.
  - AI score: `score INT 0-100`, `quality_score TEXT` (high/medium/low), `predicted_ltv DECIMAL(18,2)`, `conversion_probability DECIMAL(5,4)`.
  - Dynamic: `data JSONB DEFAULT '{}'`, `source_meta JSONB` (IP, UA, referrer, fingerprint).
  - Consent: `consent_given`, `consent_at`, `crm_synced`, `capi_sent`, `capi_event_id`.
  - **Indexes:** `idx_leads_tenant_owner_status`, `idx_leads_phone`, `idx_leads_email`, `idx_leads_data_gin GIN(data)`, `idx_leads_created_at DESC`, `idx_leads_score DESC WHERE NOT NULL`, `idx_leads_status_active WHERE deleted_at IS NULL`, `idx_leads_fbclid WHERE NOT NULL`, `idx_leads_phone_trgm gin_trgm_ops`, **partial hot-path** `idx_leads_hot (tenant_id, owner_user_id, created_at DESC) WHERE status IN ('new','contacted','qualified')`, **covering** `idx_leads_owner_covering INCLUDE (status, score, created_at)`.
- **`deals`** — pipeline sales: `lead_id`, `owner_user_id`, `team_id`, `pipeline_id`, `stage`, `title`, `value DECIMAL(18,2)`, `currency DEFAULT 'VND'`, `probability 0-100`, `expected_close_date`, `actual_close_date`, `status (OPEN/WON/LOST)`, `data JSONB`. Index `idx_deals_close_date WHERE status='OPEN'`.
- **`activities`** — log tương tác: `user_id`, `lead_id`, `deal_id`, `type` (call/email/meeting/note/task), `title`, `description`, `due_at`, `completed_at`, `status`, `data JSONB`. Partial index `idx_activities_pending WHERE status='pending'`.

### 2.3. Dynamic Model (`entity_definitions`, `dynamic_records`)

- **`entity_definitions`** — schema do admin định nghĩa: `entity_code` (real_estate, bds_property...), `display_name`, `icon`, **`fields JSONB`** (JSON Schema array), `workflows JSONB`, `indexes JSONB`, `version`, `is_active`. `UNIQUE(tenant_id, entity_code)`.
- **`dynamic_records`** — bản ghi động: `entity_code`, `owner_user_id`, **`data JSONB NOT NULL`** (toàn bộ field values), `created_at`, `deleted_at`. Indexes: `idx_dynamic_records_gin GIN(data)`, `idx_dynamic_records_tenant_entity`.

### 2.4. Lead Schema (xem §2.2 — bảng `leads` đầy đủ UTM + AI score + CAPI state).

### 2.5. Landing Page Submissions (`leads_raw` mirror)

Trong khi MongoDB lưu form submissions ở tầng document, PostgreSQL `leads` mirror sau khi NATS consumer `crm-core` xử lý từ ScyllaDB raw. Idempotency: `capi_event_id UUID UNIQUE`.

### 2.6. Email (`email_messages`, `email_templates`)

Email transactional: bảng `email_messages` lưu outbound queue với `to`, `cc`, `subject`, `body_html`, `body_text`, `template_id`, `template_vars JSONB`, `status` (queued/sent/failed/bounced), `provider_message_id`, `sent_at`, `opened_at`, `clicked_at`. Bảng `email_templates` với `tenant_id`, `slug`, `subject`, `body_html`, `variables JSONB` cho template engine.

### 2.7. Notification (`notifications`, `notification_preferences`)

- **`notifications`** — `tenant_id`, `user_id`, `channel` (in_app/email/sms/push), `type`, `title`, `body`, `data JSONB`, `read_at`, `delivered_at`, `status` (pending/delivered/failed).
- **`notification_preferences`** — `user_id`, `channel`, `event_type`, `enabled`, `quiet_hours_start/end`. Index `idx_notifications_user_unread (user_id, created_at) WHERE read_at IS NULL`.

### 2.8. AI SRE (`ai_sre_incidents`, `ai_audit_log`)

- **`ai_sre_incidents`** — `trace_id`, `tenant_id`, `error_message TEXT`, `stack_trace JSONB`, `rca JSONB` (root cause, why, fix, severity, confidence), `severity`, `confidence DECIMAL(3,2)`, `pr_url`, `status` (pending/analyzed/hotfix_created/resolved), `resolved_at`.
- **`ai_audit_log`** — `tenant_id`, `service`, `model`, `input_tokens INT`, `output_tokens INT`, `cost DECIMAL(10,6)`, `latency_ms INT`, `trace_id UUID`. RLS bật trên cả hai bảng (`FORCE ROW LEVEL SECURITY` + policy `tenant_id = current_setting('app.current_tenant_id')::UUID`).

### 2.9. Lead Scoring (`lead_scores`, `ai_models`, `lead_training_data`)

- **`lead_scores`** — cache nhanh: `lead_id PK`, `tenant_id`, `score INT`, `p_ltv DECIMAL(18,2)`, `conversion_probability DECIMAL(5,4)`, `churn_risk`, `model_version`, `scored_at`. RLS enabled.
- **`ai_models`** — registry: `model_code UNIQUE`, `model_type` (scoring/rag/stt/summary), `base_model`, `tenant_id NULL` (= system model), `version`, `metadata JSONB`, `is_active`.
- **`lead_training_data`** — historical features + label `converted` để train XGBoost per tenant.

### 2.10. RAG Knowledge (`knowledge_documents`, `ai_conversations`)

- **`knowledge_documents`** — `tenant_id`, `collection TEXT`, `title`, `content`, **`embedding vector(1024)`** (BGE-M3 dimension), `metadata JSONB`. **HNSW index** `idx_knowledge_hnsw USING hnsw (embedding vector_cosine_ops) WITH (m=16, ef_construction=64)`. Hoặc DiskANN qua pgvectorscale cho scale > 10M. Tenant filter `idx_knowledge_tenant`.
- **`ai_conversations`** — `tenant_id`, `user_id`, `session_id`, `messages JSONB`, `token_used INT`, `cost DECIMAL(10,6)`. RLS enabled.
- **`meeting_ai_summary`** — `meeting_id PK`, `tenant_id`, `transcript JSONB`, `summary TEXT`, `action_items JSONB`, `decisions JSONB`, `key_points JSONB`, `sentiment JSONB`, `model_version`.

### 2.11. Row-Level Security (RLS)

Tất cả bảng tenant-scoped đều `ENABLE + FORCE ROW LEVEL SECURITY` với policy pattern:

```sql
CREATE POLICY leads_tenant ON leads
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

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

App phải `SET LOCAL app.current_tenant_id` và `app.current_user_id` trong transaction qua pgx (`tx.Exec("SET LOCAL app.current_tenant_id = $1", tenantID)`).

### 2.12. Views & Functions tiêu biểu

- **View** `v_lead_funnel` — dashboard funnel theo ngày × source × campaign, dùng `COUNT(*) FILTER (WHERE status='new')`.
- **View** `v_team_performance` — leads_owned, leads_won, revenue_won, pipeline_value theo user subtree.
- **Function** `get_tenant_metrics(p_tenant_id, p_days)` — aggregate JSONB cho dashboard.

### 2.13. Indexes chi tiết (B-tree / GIN / GIST / partial / covering)

B-tree equality+range, GIN cho JSONB (`USING GIN((data->'phone'))`), GIST cho LTREE (`USING GIST(path)`) và range (`USING GIST(due_at)`), partial indexes cho hot path, covering index với `INCLUDE`.

### 2.14. Connection Pool (pgx)

```go
poolCfg.MaxConns = 50        // default
poolCfg.MinConns = 5
poolCfg.MaxConnLifetime = 1h
poolCfg.HealthCheckPeriod = 30s
poolCfg.ConnConfig.RuntimeParams["statement_timeout"] = "5000"
poolCfg.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "10000"
```

Migration tool: **goose** với `-- +goose Up/Down` directives.

---

## 3. ScyllaDB — Keyspaces & Tables

ScyllaDB phục vụ **chat engine, recording metadata, WebRTC metadata, audit log, CAPI events, leads_raw**. Compaction: **TWCS** (Time Window Compaction Strategy) cho chat logs (window = 1 day), **LCS** cho metadata, **không dùng STCS** cho production.

### 3.1. Keyspace `rinco_chat`

- **`messages`** — partition `(channel_id, tenant_id)`, clustering `(event_time DESC, event_id DESC)`. Columns: `sender_id`, `message_type tinyint`, `text`, `metadata map<text,text>`, `attachments list<frozen<attachment>>`, `reactions map<text,frozen<reaction>>`, `reply_to`, `mentions list<text>`, `pinned`, `deleted`, `edited_at`. TTL 90 ngày (`default_time_to_live = 7776000`).
- **`messages_by_user`** — partition `(user_id, tenant_id)`, clustering `event_time DESC`. Hỗ trợ "tin nhắn của tôi".
- **`messages_by_reply`** — partition `(reply_to, tenant_id)`, clustering `event_time`. Thread view.
- **`channels`** — partition `(channel_id, tenant_id)`. Columns: `name`, `type tinyint`, `description`, `avatar_url`, `members set<text>`, `admins set<text>`, `created_by`, `created_at`, `archived`.
- **`user_channels`** — partition `(user_id, tenant_id)`, clustering `channel_id`. Columns: `joined_at`, `last_read_event_time`, `unread_count`, `muted`, `pinned`.
- **`channel_unread`** — partition `(channel_id, tenant_id)`, clustering `user_id`. Columns: `event_time` (last activity).

### 3.2. Keyspace `rinco_events` (CAPI events)

- **`capi_events`** — partition `(tenant_id, sent_at)`, clustering `event_id`. Columns: `fb_event_name`, `status_code`, `events_received`, `fbtrace_id`, `error_message`, `retry_count`. Track gửi CAPI Facebook.
- **`capi_failures`** — partition `(tenant_id, failed_at)`, clustering `event_id`. Columns: `payload TEXT`, `retry_count`, `next_retry_at`. Retry queue.

### 3.3. Keyspace `rinco_audit`

- **`audit_log`** — partition `(tenant_id, created_at)`, clustering `id DESC`. Columns: `actor_id`, `actor_email`, `actor_type`, `action`, `resource_type`, `resource_id`, `ip_address inet`, `trace_id uuid`, `payload TEXT`, **`signature TEXT`** (cryptographic chain). TTL 5 năm (`default_time_to_live = 157680000`).

### 3.4. Keyspace `rinco_leads` (Landing Ingest)

- **`leads_raw`** — partition `(tenant_id, event_time)`, clustering `event_id DESC`. Columns: `idempotency_key`, `full_name`, `phone`, `email`, UTM đầy đủ, `fbclid`, `gclid`, `ip_address`, `user_agent`, `source`, `form_id`, `extra map<text,text>`, `processed boolean`, `lead_id uuid`. Unique index `(tenant_id), idempotency_key` cho idempotent ingestion.

### 3.5. Connection Setup (gocql)

`Consistency = Quorum`, `ProtoVersion = 5`, `Compression = Snappy`, `Timeout = 5s`, `ConnectTimeout = 10s`, `NumConns = 4`, `PoolSize = 20`, `MaxStreams = 100`.

---

## 4. MongoDB — Databases

MongoDB dùng cho **dynamic form data** với schema linh hoạt. WiredTiger cache 4GB, journalCompressor snappy, replica set `rinco-rs`, sharded cluster role.

### 4.1. Database `rinco_forms` (Landing Pages)

- **`form_submissions`** — bắt buộc fields: `tenant_id` (regex `^[a-z0-9-]{3,50}$`), `form_id`, `submitted_at`, `data object`. Tuỳ chọn: `ip_address`, `user_agent`, `meta`, `status enum [new, processed, rejected]`. `$jsonSchema` validator ở `validationLevel: "moderate"`, `validationAction: "error"`. Sharded theo `{tenant_id: 1, submitted_at: 1}`. Indexes: `(tenant_id, form_id, submitted_at -1)`, `data.email`, `data.phone`, `(tenant_id, status, submitted_at -1)`.
- **`form_definitions`** — JSON Schema definitions của forms. Unique index `(tenant_id, slug)`. Sharded theo `{tenant_id: 1}`.
- **`survey_responses`** — `survey_id`, `respondent_id`, `answers array`. Validator yêu cầu `answers` là array.
- **`page_variants`** (A/B test landing page) — `tenant_id`, `page_id`, `variant`, `weight`, `html_content`, `css`, `metadata`.
- **`page_analytics`** — page view event raw từ landing trước khi đẩy ClickHouse.

### 4.2. Database `rinco_analytics` (event raw landing)

Event landing page trước khi NATS ship sang ClickHouse.

### 4.3. Sharding

```javascript
sh.enableSharding("rinco_forms");
sh.shardCollection("rinco_forms.form_submissions", { tenant_id: 1, submitted_at: 1 });
sh.shardCollection("rinco_forms.form_definitions", { tenant_id: 1 });
```

---

## 5. ClickHouse — Databases

ClickHouse là **OLAP warehouse**. Cluster 2 shard × 2 replica với ZooKeeper ensemble 3 node. Compression ZSTD, timezone Asia/Ho_Chi_Minh.

### 5.1. Database `rinco_audit` (long-term audit)

Lưu trữ audit log lâu dài, nén cao, query slow-scan.

### 5.2. Database `rinco_observability` (system observability)

Application logs, traces, metrics aggregated từ eBPF signals. Phục vụ AI SRE query logs by `trace_id`.

### 5.3. Database `rinco_analytics` (lead events + business analytics)

- **`rinco_analytics.events_raw`** — MergeTree, `PARTITION BY toYYYYMM(event_date)`, `ORDER BY (tenant_id, event_date, event_name)`, TTL 90 ngày. Columns: `event_date MATERIALIZED toDate(timestamp)`, `timestamp DateTime64(9)`, `tenant_id LowCardinality(String)`, `event_id UUID`, `session_id UUID`, `anonymous_id`, `user_id`, `event_name LowCardinality(String)`, UTM fields, `country`, `city`, `device_type`, `browser`, `os`, `ip_address IPv4`, `user_agent`, `referrer`, `page_url`, `properties JSON`.
- **`events_aggregated`** — AggregatingMergeTree với `AggregateFunction(uniq, String)` cho unique_users/sessions và `sum, Decimal(18,4)` cho total_value.
- **`funnel`** — AggregatingMergeTree, partition theo `toYYYYMM(event_date)`, order `(tenant_id, event_date, campaign_id, step)`.
- **`lead_performance`** — AggregatingMergeTree theo `(tenant_id, event_date, owner_user_id)`. Track conversion funnel per sales rep.
- **Materialized views:** `events_daily_mv TO events_aggregated`, `lead_performance_mv TO lead_performance`.
- **Per-tenant row policy:**
  ```sql
  CREATE ROW POLICY tenant_apex_filter ON rinco_analytics.events_raw
    USING tenant_id = 'apex' TO tenant_apex;
  ```

### 5.4. Database `rinco_lead_events`

Lead lifecycle events riêng (lead.created, lead.scored, lead.converted) với partitioning theo `event_date`.

---

## 6. Valkey — Usage Patterns

Valkey 9.x (Redis fork) phục vụ 5 use case chính: **cache, sessions, presence, rate-limit, distributed lock, streams, pub/sub**. Setup: `maxmemory 8GB`, `maxmemory-policy allkeys-lru`, `io-threads 8`, `appendonly yes` (AOF + RDB both), cluster mode enabled.

### 6.1. Key Patterns theo use case

| Key pattern | Purpose | TTL |
|-------------|---------|-----|
| `domain:{hostname}` | Tenant resolution (subdomain/subpath) | 3600s |
| `tenant:{tid}:config` | Tenant settings cache | 600s |
| `user:{uid}:session` | Session data | 24h |
| `presence:user:{uid}` | Online status (heartbeat refresh) | 60s |
| `presence:channel:{cid}` | Online users in channel (set) | 300s |
| `ratelimit:{tid}:{action}` | Rate limit counter (token bucket) | 60s |
| `singleflight:{key}` | Coalescing lock cho duplicate reads | 10s |
| `distlock:{name}` | Distributed lock (Redlock pattern) | 30s |
| `meetsfu:room:{room_code}` | Meeting room state | 24h |
| `capi:emq:{tid}` | CAPI EMQ score | 3600s |
| `notif:queue:{uid}` | Notification pending list | 7 days |
| `dashboard:cache:{tid}:{query}` | Dashboard query cache | 60s |
| `tenant-scoped:ai:cache:{tid}:{prefix_hash}` | **Tenant-isolated AI cache** (PROMPTPEEK defense) | 600s |
| `fingerprint:js:{ua_hash}` | Browser fingerprint | 86400s |
| `idem:{tid}:{key}` | Idempotency key cho landing ingest | 24h |

### 6.2. Streams (lightweight event log)

```bash
XADD rinco:events:lead * type "lead.created" tenant "apex" data "{...}"
XREAD BLOCK 0 STREAMS rinco:events:lead $
XGROUP CREATE rinco:events:lead crm-workers $ MKSTREAM
```

Streams thay thế Pub/Sub khi cần durability + replay + consumer groups.

### 6.3. Pub/Sub (chat broadcast)

```bash
PUBLISH chat:room:c-123 "{...}"
SUBSCRIBE chat:room:c-123
```

### 6.4. Cluster

6 node (3 master + 3 replica), hash-slot sharding. Driver: `redis-rs` hoặc `fred` (async).

---

## 7. MinIO — Buckets & Tiering

MinIO cluster 4 node EC:4 (4 data + 4 parity), S3-compatible, server-side encryption SSE-KMS.

| Bucket | Purpose | Retention / Tiering |
|--------|---------|---------------------|
| `rinco-tenant-assets` | Logo, favicon, custom CSS per tenant | Forever + versioning |
| `rinco-chat-files` | Chat attachments | 90 days (auto-expire) |
| `rinco-user-avatars` | User profile pictures | Forever |
| `rinco-meeting-recordings` | Meeting recordings (mp4) | Configurable per tenant |
| `rinco-ai-models` | ML model artifacts (XGBoost ONNX, Llama-3 AWQ) | Forever + versioning |
| `rinco-backups` | PostgreSQL/Scylla/ClickHouse backups | 30 days hot + 1 năm cold |
| `rinco-cold-archive` | Old data archive | 7 năm (Glacier tier) |
| `rinco-form-uploads` | User uploaded files (form fields) | Per form retention |

### 7.1. Bucket Policies & Tiering

- Per-tenant prefix isolation (`tenant_id/...`).
- Lifecycle rules: `mc ilm add rinco/rinco-chat-files --expiry-days 90`, `mc ilm add rinco/rinco-cold-archive --expiry-days 2555`.
- Versioning enabled cho critical buckets (`mc version enable rinco/rinco-tenant-assets`).
- Cross-region replication cho `rinco-backups` và `rinco-cold-archive`.
- SSE-KMS encryption; KMS key rotation 90 ngày.

### 7.2. Erasure Coding

- Default EC:4 (4 data + 4 parity).
- Cost-optimal EC:2 (ít quan trọng).
- Performance: no EC (replicated, ít dùng).
- Drive fail tự động heal; alert nếu > 24h.

### 7.3. Presigned URL

`PresignedPutObject` 15 phút cho upload, `PresignedGetObject` 15 phút cho download.

---

## 8. Qdrant — Vector Collections

Qdrant dùng cho **vector search > 10M vectors**. Rust-native HNSW + quantization (INT8 scalar), HNSW config `m=16`, `ef_construct=100`, `indexing_threshold=20000KB`.

### 8.1. Collections

- **`kb_{tenant_id}`** — knowledge base per tenant (RAG documents). Vectors 1024-dim (BGE-M3), Cosine distance, ScalarQuantization INT8 always_ram.
- **`documents_{tenant_id}`** — document embeddings riêng.
- **`rinco_kb`** (sharded) — large tenants dùng chung collection với `tenant_id` payload filter thay vì per-tenant collection.

### 8.2. Tenant Isolation Strategy

```python
hits = client.search(
    collection_name="rinco_kb",
    query_vector=embedding,
    query_filter={"must": [{"key": "tenant_id", "match": {"value": "apex"}}]},
    limit=5,
    with_payload=True,
)
```

Mandatory `tenant_id` filter; audit log mọi search request.

### 8.3. Routing

`< 10M vectors → pgvector + pgvectorscale`; `> 10M vectors → Qdrant`. Vector dimension BGE-M3 = 1024.

---

## 9. pgvector (alternative cho Qdrant)

Khi tenant có < 10M vectors, dùng **`vector` extension** + **`vectorscale` (DiskANN)** trong chính cluster PostgreSQL → tiết kiệm ops cost.

```sql
CREATE TABLE knowledge_documents (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  collection TEXT NOT NULL,
  embedding vector(1024),  -- BGE-M3
  metadata JSONB
);
CREATE INDEX idx_knowledge_hnsw ON knowledge_documents
  USING hnsw (embedding vector_cosine_ops) WITH (m=16, ef_construction=64);
-- DiskANN thay thế khi scale
-- CREATE INDEX idx_knowledge_diskann ON knowledge_documents USING diskann (embedding vector_cosine_ops);
```

---

## 10. Meilisearch — Indexes

Meilisearch cluster 2-6 node, typo-tolerant, inverted index, Tiếng Việt tốt.

### 10.1. Indexes

- **`leads`** — `primaryKey: id`. `searchableAttributes: [name, phone, email, data.*]`. `filterableAttributes: [tenant_id, status, owner_user_id, source]`. `sortableAttributes: [created_at, score, updated_at]`.
- **`contacts`** — `searchableAttributes: [full_name, email, phone, company]`. Filter + sort theo tenant.
- **`knowledge`** — full-text knowledge base articles.
- **`conversations`** — search lịch sử chat AI.

Search API Go:

```go
req := &meilisearch.SearchRequest{
  Filter: "tenant_id = " + tenantID,
  Sort:   []string{"created_at:desc"},
  Limit:  20,
}
m.client.Index("leads").Search(query, req)
```

---

## 11. NATS JetStream — Subjects

NATS JetStream làm **event bus** giữa các services, durable + replay.

### 11.1. Subjects chính

```
tenant.lead.created       # landing-ingest → crm-core, ai-scoring, meta-capi, clickhouse-pipeline
tenant.lead.updated       # crm-core → analytics, meta-capi, ai-scoring, notification
tenant.deal.updated       # crm-core → analytics, meta-capi, ai-scoring
chat.message.sent         # chat-engine → chat-realtime, notification, ai-conversation
meeting.recording.uploaded # egress-worker → ai-media-stt-worker
meeting.transcript.ready  # ai-media-stt → ai-media-summarizer
ai.rca.completed          # ai-sre-worker → telegram-bot, ai-sre-incidents
lead.scored               # ai-scoring → crm-core, meta-capi
capi.sent                 # meta-capi → analytics, ai-feedback-loop
```

### 11.2. Sequence flow Lead Create

`Landing Form → api-gateway → landing-ingest → Valkey (idempotency) → ScyllaDB rinco_leads.leads_raw → NATS tenant.lead.created → [crm-core → PostgreSQL leads INSERT + RLS] → [ai-scoring → XGBoost ONNX → PostgreSQL UPDATE leads.score + NATS lead.scored] → [meta-capi → FB CAPI + ScyllaDB capi_events] → [clickhouse-pipeline → events_raw + MV updates]`.

---

## 12. PgBouncer — Connection Pooling

PgBouncer đặt trước PostgreSQL, transaction pooling.

```ini
[pgbouncer]
listen_port = 6432
auth_type = scram-sha-256
pool_mode = transaction
max_client_conn = 5000
default_pool_size = 50
reserve_pool_size = 10
reserve_pool_timeout = 3
server_idle_timeout = 600
server_lifetime = 3600
query_timeout = 30
query_wait_timeout = 30
tcp_keepalive = 1
log_connections = 1
log_disconnections = 1
```

3 databases: `rinco` (primary), `rinco_ro` (replica read-only), `rinco_admin`. Auth SCRAM-SHA-256 qua `userlist.txt`.

Service-to-database mapping: **crm-core → PostgreSQL**; **chat-engine → ScyllaDB + Valkey**; **landing-ingest → ScyllaDB + ClickHouse**; **analytics → ClickHouse**; **auth → PostgreSQL**; **notification → PostgreSQL + Valkey**; **ai-services → PostgreSQL + Qdrant + Valkey**.

---

## 13. Backup & DR

| Database | RPO | RTO |
|----------|-----|-----|
| PostgreSQL | 5 phút (WAL archive continuous) | 30 phút (PgBouncer failover) |
| ScyllaDB | 1 phút (multi-DC) | 15 phút (NetworkTopologyStrategy) |
| ClickHouse | 1 giờ | 1 giờ |
| Valkey | 0 (AOF + replica) | 30 giây (Sentinel failover) |
| MinIO | 0 (cross-region replication) | 1 phút |

Backup scripts:
- **PostgreSQL:** `pg_basebackup --checkpoint=fast --wal-method=stream` → MinIO `rinco-backups/postgres/` với SHA256 checksum. WAL archive liên tục qua `archive_command = 'mc cp %p minio/backups/postgres/wal/%f'`.
- **ScyllaDB:** `nodetool snapshot` per node → tar → MinIO. Retain 30 ngày.
- **ClickHouse:** `clickhouse-backup create → upload s3 → MinIO rinco-backups/clickhouse/`.

DR drill quarterly: snapshot → kill primary → promote replica → verify data integrity (checksum) → cleanup.

---

## 14. Danh sách ≥ 25 Database Features

1. **Polyglot persistence** — 9 engines, mỗi engine cho 1 access pattern.
2. **PostgreSQL 17+** với JSONB, LTREE, pgvector, vectorscale, pg_partman, pg_cron.
3. **Row-Level Security (RLS)** tenant isolation với `SET LOCAL app.current_tenant_id`.
4. **LTREE org tree** cho users với GIST index + cycle detection trong `move_subtree`.
5. **JSONB GIN indexes** cho dynamic fields (`idx_leads_data_gin`, `idx_leads_data_phone_gin`).
6. **Partial indexes** cho hot path (`idx_leads_hot WHERE status IN (...)`).
7. **Covering indexes** với `INCLUDE` clause.
8. **Trigram fuzzy search** `gin_trgm_ops` cho phone matching.
9. **Materialized views** kết hợp AggregatingMergeTree trong ClickHouse.
10. **Per-tenant row policy** ClickHouse cho tenant isolation ở OLAP.
11. **Time-Window Compaction Strategy (TWCS)** cho chat messages (window = 1 ngày).
12. **Idempotency unique index** trên `(tenant_id, idempotency_key)` ở ScyllaDB leads_raw.
13. **TTL tự động** — chat 90 ngày, audit 5 năm, events 90 ngày.
14. **Scylla NetworkTopologyStrategy** multi-DC replication (DC1=3).
15. **gocql token-aware routing** cho Scylla driver.
16. **PgBouncer transaction pooling** 5000 → 50 servers.
17. **pgxpool tuning** với `statement_timeout=5s`, `idle_in_transaction=10s`.
18. **goose migration tool** với Up/Down + expand-contract pattern.
19. **Valkey Streams + Consumer Groups** thay thế Pub/Sub khi cần durability.
20. **Valkey Pub/Sub** cho chat realtime broadcast.
21. **Valkey distributed lock + singleflight coalescing** chống duplicate work.
22. **MinIO EC:4 erasure coding** (4 data + 4 parity) cho storage hiệu quả.
23. **MinIO lifecycle rules** + cross-region replication + versioning.
24. **Qdrant HNSW + INT8 ScalarQuantization** cho vector search scale.
25. **Qdrant tenant filter** mandatory trong mọi search request.
26. **pgvector DiskANN** (pgvectorscale) cho vector < 10M.
27. **Meilisearch typo-tolerant inverted index** cho Tiếng Việt.
28. **NATS JetStream** durable messaging giữa services.
29. **Outbox pattern + idempotent consumer** chống cross-DB inconsistency.
30. **Cross-DB consistency check** trong DR drill.
31. **Continuous WAL archiving** PostgreSQL sang MinIO để PITR.
32. **ScyllaDB snapshot per-node backup** distributed.
33. **clickhouse-backup** S3-native export.
34. **Quarterly DR drill** với restore time tracking.
35. **Managed vs self-hosted cost comparison** (PostgreSQL 72% saving vs RDS).
36. **Migration tracking sheet** với risk classification (Low/Medium/High).
37. **Database-per-service** mapping enforced (crm-core → PG, chat-engine → Scylla + Valkey).
38. **Connection pool warmup** + DNS caching qua Valkey.
39. **Database sharding roadmap** Q4 → 6 shards/region + Citus.
40. **Edge case catalog** (DB-01 → DB-75) với detection + handling cho mỗi engine.
