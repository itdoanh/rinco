# Phần 10 – Database Schema & Polyglot Persistence

> **Phân hệ:** Toàn bộ cơ sở dữ liệu của hệ thống RINCO.  
> **Mục tiêu:** Mỗi service chọn database tối ưu cho access pattern, đảm bảo performance, ACID, isolation.  
> **Triết lý:** Polyglot Persistence – không ép buộc 1 DB cho mọi use case.  
> **Phiên bản:** v1.2 (audit + mở rộng implementation roadmap, code examples, edge cases, DR, cost, open questions).

---

## Mục lục

1. [Tổng quan Polyglot Persistence](#1-tổng-quan-polyglot-persistence)
2. [PostgreSQL 17+ – CRM Core](#2-postgresql-17--crm-core)
3. [ScyllaDB – Chat & Events](#3-scylladb--chat--events)
4. [ClickHouse – Analytics](#4-clickhouse--analytics)
5. [MongoDB – Dynamic Forms](#5-mongodb--dynamic-forms)
6. [Valkey 9x – Cache & State](#6-valkey-9x--cache--state)
7. [MinIO/S3 – Object Storage](#7-minios3--object-storage)
8. [Qdrant + pgvector – Vector Search](#8-qdrant--pgvector--vector-search)
9. [Meilisearch – Full-text Search](#9-meilisearch--full-text-search)
10. [Migration Strategy](#10-migration-strategy)
11. [Backup & DR](#11-backup--dr)
12. [Connection Pooling](#12-connection-pooling)
13. [Audit Report](#13-audit-report)
14. [Edge Cases & Error Scenarios](#14-edge-cases--error-scenarios)
15. [Sequence Diagrams](#15-sequence-diagrams)
16. [Implementation Roadmap](#16-implementation-roadmap)
17. [Testing Strategy](#17-testing-strategy)
18. [Migration Plan](#18-migration-plan)
19. [Disaster Recovery](#19-disaster-recovery)
20. [Cost Estimation](#20-cost-estimation)
21. [Open Questions](#21-open-questions)

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

### 1.3. Ma trận khả năng mở rộng (theo phase)

| Database | Q1 (1K tenant) | Q2 (10K tenant) | Q3 (100K tenant) | Q4 (1M tenant) |
|----------|----------------|------------------|-------------------|------------------|
| PostgreSQL | 1 primary + 2 replicas (single region) | 1 primary + 4 replicas + PgCat | 3 shards + 6 replicas + PgBouncer | 6 shards per region + Citus |
| ScyllaDB | 3 node (RF=3) | 6 node | 12 node | 24 node multi-DC |
| ClickHouse | 1 shard × 2 replicas | 3 shard × 2 replicas | 6 shard × 2 replicas | Cloud-only |
| MongoDB | 3 node RS | 6 node sharded | 12 node | 24 node |
| Valkey | 3 master + 3 replica | 6 master + 6 replica | 12 + 12 | 24 + 24 |
| MinIO | 4 node EC:4 | 8 node EC:4 | 16 node EC:4 | Cross-region replication |
| Qdrant | 1 node | 3 node | 6 node | 12 node |
| Meilisearch | 1 node | 2 node | 4 node | 6 node |

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
wal_compression = zstd
synchronous_commit = on
checkpoint_completion_target = 0.9
max_wal_size = 4GB
min_wal_size = 1GB
log_min_duration_statement = 200ms
log_lock_waits = on
log_temp_files = 0
autovacuum_max_workers = 4
autovacuum_naptime = 30s
```

### 2.2. Extensions

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "ltree";           -- Tree structure
CREATE EXTENSION IF NOT EXISTS "pg_trgm";          -- Trigram fuzzy search
CREATE EXTENSION IF NOT EXISTS "pgvector";         -- Vector search
CREATE EXTENSION IF NOT EXISTS "vectorscale";      -- DiskANN index
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "postgis";          -- Geo
CREATE EXTENSION IF NOT EXISTS "pg_partman";       -- Partition management
CREATE EXTENSION IF NOT EXISTS "hstore";          -- key/value in row
CREATE EXTENSION IF NOT EXISTS "pg_cron";          -- Scheduled jobs
CREATE EXTENSION IF NOT EXISTS "pg_repack";        -- Online reorg
```

### 2.3. Schema Overview

#### Tenants & Identity (multi-tenancy)

```sql
-- Tenants
CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  plan TEXT NOT NULL,                    -- 'free','starter','business','enterprise'
  status TEXT NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE','SUSPENDED','ARCHIVED'
  isolation_mode TEXT DEFAULT 'SHARED', -- 'SHARED','ISOLATED_VPS'
  region TEXT NOT NULL,                  -- 'ap-southeast-1','us-east-1'
  vps_node_id UUID,
  settings JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_plan ON tenants(plan);

-- Domains (subpath/subdomain/custom)
CREATE TABLE tenant_domains (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  hostname TEXT UNIQUE NOT NULL,
  routing TEXT NOT NULL,                  -- 'subpath','subdomain','custom'
  verified BOOLEAN DEFAULT false,
  tls_cert_id UUID,
  created_at TIMESTAMPTZ DEFAULT now()
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
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_path ON users USING GIST(path);
CREATE INDEX idx_users_tenant_role ON users(tenant_id, role);
CREATE INDEX idx_users_email_lower ON users(LOWER(email));
CREATE INDEX idx_users_parent ON users(parent_id);
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
  gclid TEXT,
  status TEXT NOT NULL DEFAULT 'new',
  score INT,                                -- AI score 0-100
  quality_score TEXT,                       -- 'high','medium','low'
  predicted_ltv DECIMAL(18, 2),
  conversion_probability DECIMAL(5, 4),
  data JSONB DEFAULT '{}',                  -- dynamic fields
  consent_given BOOLEAN DEFAULT false,
  consent_at TIMESTAMPTZ,
  source_meta JSONB,                        -- IP, UA, referrer, fingerprint
  crm_synced BOOLEAN DEFAULT false,
  capi_sent BOOLEAN DEFAULT false,
  capi_event_id UUID,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_leads_tenant_owner_status ON leads(tenant_id, owner_user_id, status);
CREATE INDEX idx_leads_phone ON leads(tenant_id, phone);
CREATE INDEX idx_leads_email ON leads(tenant_id, email);
CREATE INDEX idx_leads_data_gin ON leads USING GIN(data);
CREATE INDEX idx_leads_created_at ON leads(tenant_id, created_at DESC);
CREATE INDEX idx_leads_score ON leads(tenant_id, score DESC) WHERE score IS NOT NULL;
CREATE INDEX idx_leads_status_active ON leads(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_leads_fbclid ON leads(tenant_id, fbclid) WHERE fbclid IS NOT NULL;
CREATE INDEX idx_leads_phone_trgm ON leads USING gin(phone gin_trgm_ops);
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
  probability INT,                          -- 0-100
  expected_close_date DATE,
  actual_close_date DATE,
  status TEXT NOT NULL DEFAULT 'OPEN',
  data JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_deals_tenant_stage ON deals(tenant_id, stage);
CREATE INDEX idx_deals_owner ON deals(tenant_id, owner_user_id);
CREATE INDEX idx_deals_pipeline ON deals(tenant_id, pipeline_id);
CREATE INDEX idx_deals_close_date ON deals(tenant_id, expected_close_date)
  WHERE status = 'OPEN';
CREATE INDEX idx_deals_data_gin ON deals USING GIN(data);
```

#### Activity Schema

```sql
CREATE TABLE activities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  user_id UUID REFERENCES users(id),
  lead_id UUID REFERENCES leads(id),
  deal_id UUID REFERENCES deals(id),
  type TEXT NOT NULL,                       -- 'call','email','meeting','note','task'
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
CREATE INDEX idx_activities_deal ON activities(deal_id);
CREATE INDEX idx_activities_pending ON activities(user_id, due_at)
  WHERE status = 'pending';
```

#### Dynamic Schema Tables

```sql
-- Schema definitions (admin-defined)
CREATE TABLE entity_definitions (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  entity_code TEXT NOT NULL,                -- 'real_estate','bds_property'
  display_name TEXT NOT NULL,
  icon TEXT,
  fields JSONB NOT NULL,                    -- JSON Schema array
  workflows JSONB,
  indexes JSONB,
  version INT DEFAULT 1,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(tenant_id, entity_code)
);

-- Dynamic records
CREATE TABLE dynamic_records (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  entity_code TEXT NOT NULL,
  owner_user_id UUID,
  data JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_dynamic_records_gin ON dynamic_records USING GIN(data);
CREATE INDEX idx_dynamic_records_tenant_entity ON dynamic_records(tenant_id, entity_code);
CREATE INDEX idx_dynamic_records_owner ON dynamic_records(owner_user_id);
```

#### Indexes chi tiết (B-tree, GIN, GIST, partial)

```sql
-- B-tree cho equality + range
CREATE INDEX idx_deals_value ON deals(tenant_id, value)
  WHERE status = 'OPEN';

-- GIN cho JSONB
CREATE INDEX idx_leads_data_phone_gin ON leads USING GIN((data->'phone'));
CREATE INDEX idx_users_settings_gin ON users USING GIN(settings);

-- GIST cho LTREE và range
CREATE INDEX idx_users_path_gist ON users USING GIST(path);
CREATE INDEX idx_activities_time_gist ON activities USING GIST(due_at);

-- Partial index cho hot path
CREATE INDEX idx_leads_hot ON leads(tenant_id, owner_user_id, created_at DESC)
  WHERE status IN ('new','contacted','qualified');

-- Composite
CREATE INDEX idx_leads_dashboard ON leads(tenant_id, status, owner_user_id, created_at DESC);

-- Covering index (INCLUDE)
CREATE INDEX idx_leads_owner_covering ON leads(tenant_id, owner_user_id)
  INCLUDE (status, score, created_at);
```

#### Views & Functions

```sql
-- View: dashboard lead funnel
CREATE OR REPLACE VIEW v_lead_funnel AS
SELECT
  tenant_id,
  DATE_TRUNC('day', created_at) AS day,
  source,
  utm_campaign,
  COUNT(*) FILTER (WHERE status = 'new') AS new_count,
  COUNT(*) FILTER (WHERE status = 'qualified') AS qualified_count,
  COUNT(*) FILTER (WHERE status = 'converted') AS converted_count,
  COUNT(*) FILTER (WHERE status = 'lost') AS lost_count,
  AVG(score) FILTER (WHERE score IS NOT NULL) AS avg_score
FROM leads
WHERE deleted_at IS NULL
GROUP BY tenant_id, DATE_TRUNC('day', created_at), source, utm_campaign;

-- View: team performance
CREATE OR REPLACE VIEW v_team_performance AS
SELECT
  u.tenant_id,
  u.path,
  u.id AS user_id,
  u.full_name,
  COUNT(DISTINCT l.id) AS leads_owned,
  COUNT(DISTINCT l.id) FILTER (WHERE l.status = 'won') AS leads_won,
  COALESCE(SUM(d.value) FILTER (WHERE d.status = 'WON'), 0) AS revenue_won,
  COALESCE(SUM(d.value) FILTER (WHERE d.status = 'OPEN'), 0) AS pipeline_value
FROM users u
LEFT JOIN leads l ON l.owner_user_id = u.id AND l.deleted_at IS NULL
LEFT JOIN deals d ON d.owner_user_id = u.id AND d.deleted_at IS NULL
WHERE u.deleted_at IS NULL
GROUP BY u.tenant_id, u.path, u.id, u.full_name;

-- Function: move subtree trong cây (transactional)
CREATE OR REPLACE FUNCTION move_subtree(
  p_user_id UUID,
  p_new_parent_id UUID
) RETURNS VOID AS $$
DECLARE
  v_old_path LTREE;
  v_new_parent_path LTREE;
  v_old_subtree LTREE;
BEGIN
  SELECT path INTO v_old_path FROM users WHERE id = p_user_id FOR UPDATE;
  SELECT path INTO v_new_parent_path FROM users WHERE id = p_new_parent_id FOR UPDATE;

  -- Cycle detection
  IF v_new_parent_path <@ v_old_path THEN
    RAISE EXCEPTION 'cycle_detected';
  END IF;

  v_old_subtree := v_old_path;
  UPDATE users
    SET path = v_new_parent_path || subpath(path, nlevel(v_old_path)),
        depth = depth + nlevel(v_new_parent_path) - nlevel(v_old_path) + 1,
        updated_at = now()
  WHERE path <@ v_old_subtree;
END;
$$ LANGUAGE plpgsql;

-- Function: get subtree users (optimized)
CREATE OR REPLACE FUNCTION get_subtree_users(p_user_id UUID)
RETURNS TABLE(id UUID, email TEXT, full_name TEXT, role TEXT, depth INT) AS $$
  SELECT id, email, full_name, role, depth
  FROM users
  WHERE path <@ (SELECT path FROM users WHERE id = p_user_id)
  ORDER BY path;
$$ LANGUAGE SQL STABLE;

-- Function: aggregate metrics per tenant (cached for dashboard)
CREATE OR REPLACE FUNCTION get_tenant_metrics(p_tenant_id UUID, p_days INT DEFAULT 30)
RETURNS JSONB AS $$
DECLARE
  result JSONB;
BEGIN
  SELECT jsonb_build_object(
    'total_leads', (SELECT COUNT(*) FROM leads WHERE tenant_id = p_tenant_id AND created_at > now() - (p_days || ' days')::INTERVAL),
    'converted_leads', (SELECT COUNT(*) FROM leads WHERE tenant_id = p_tenant_id AND status = 'converted' AND created_at > now() - (p_days || ' days')::INTERVAL),
    'total_deals_value', (SELECT COALESCE(SUM(value),0) FROM deals WHERE tenant_id = p_tenant_id AND status = 'WON' AND actual_close_date > now() - (p_days || ' days')::INTERVAL),
    'pipeline_value', (SELECT COALESCE(SUM(value),0) FROM deals WHERE tenant_id = p_tenant_id AND status = 'OPEN'),
    'avg_score', (SELECT COALESCE(AVG(score),0) FROM leads WHERE tenant_id = p_tenant_id AND score IS NOT NULL AND created_at > now() - (p_days || ' days')::INTERVAL)
  ) INTO result;
  RETURN result;
END;
$$ LANGUAGE plpgsql STABLE;
```

### 2.4. RLS Implementation

```sql
-- Enable RLS
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads FORCE ROW LEVEL SECURITY;

-- Tenant isolation
CREATE POLICY leads_tenant ON leads
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- User scope (own or subtree)
CREATE POLICY leads_user_scope ON leads
  USING (
    owner_user_id = current_setting('app.current_user_id', true)::UUID
    OR owner_user_id IN (
      SELECT id FROM users
      WHERE path <@ (SELECT path FROM users WHERE id = current_setting('app.current_user_id', true)::UUID)
    )
    OR current_setting('app.is_super_admin', true) = 'true'
  );

-- Deal table
ALTER TABLE deals ENABLE ROW LEVEL SECURITY;
ALTER TABLE deals FORCE ROW LEVEL SECURITY;
CREATE POLICY deals_tenant ON deals
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Activities
ALTER TABLE activities ENABLE ROW LEVEL SECURITY;
ALTER TABLE activities FORCE ROW LEVEL SECURITY;
CREATE POLICY activities_tenant ON activities
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
```

### 2.5. Repository Pattern (Go + sqlc)

```go
// internal/repository/lead_repository.go
package repository

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Lead struct {
    ID            string
    TenantID      string
    OwnerUserID   *string
    Name          string
    Phone         *string
    Email         *string
    Status        string
    Score         *int
    Data          map[string]any
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type LeadRepository struct {
    pool *pgxpool.Pool
}

func NewLeadRepository(pool *pgxpool.Pool) *LeadRepository {
    return &LeadRepository{pool: pool}
}

// WithTenantContext wraps a fn with tenant RLS context
func (r *LeadRepository) WithTenantContext(
    ctx context.Context,
    tenantID, userID string,
    fn func(tx pgx.Tx) error,
) error {
    tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // SET LOCAL chỉ có hiệu lực trong transaction này
    if _, err := tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID); err != nil {
        return err
    }
    if _, err := tx.Exec(ctx, "SET LOCAL app.current_user_id = $1", userID); err != nil {
        return err
    }

    if err := fn(tx); err != nil {
        return err
    }
    return tx.Commit(ctx)
}

func (r *LeadRepository) Create(ctx context.Context, l *Lead) error {
    return r.WithTenantContext(ctx, l.TenantID, *l.OwnerUserID, func(tx pgx.Tx) error {
        _, err := tx.Exec(ctx, `
            INSERT INTO leads (
                id, tenant_id, owner_user_id, name, phone, email,
                source, utm_source, utm_campaign, fbclid,
                status, score, data, created_at, updated_at
            ) VALUES (
                gen_random_uuid(), $1, $2, $3, $4, $5,
                $6, $7, $8, $9, $10, $11, $12, now(), now()
            )
        `, l.TenantID, l.OwnerUserID, l.Name, l.Phone, l.Email,
            nil, nil, nil, nil, l.Status, l.Score, l.Data)
        return err
    })
}

func (r *LeadRepository) ListByOwner(ctx context.Context, tenantID, ownerID string, limit int) ([]Lead, error) {
    var leads []Lead
    err := r.WithTenantContext(ctx, tenantID, ownerID, func(tx pgx.Tx) error {
        rows, err := tx.Query(ctx, `
            SELECT id, tenant_id, owner_user_id, name, phone, email, status, score,
                   data, created_at, updated_at
            FROM leads
            WHERE owner_user_id = $1 AND deleted_at IS NULL
            ORDER BY created_at DESC
            LIMIT $2
        `, ownerID, limit)
        if err != nil {
            return err
        }
        defer rows.Close()
        for rows.Next() {
            var l Lead
            if err := rows.Scan(&l.ID, &l.TenantID, &l.OwnerUserID, &l.Name,
                &l.Phone, &l.Email, &l.Status, &l.Score,
                &l.Data, &l.CreatedAt, &l.UpdatedAt); err != nil {
                return err
            }
            leads = append(leads, l)
        }
        return rows.Err()
    })
    return leads, err
}
```

### 2.6. Connection Pool (Go + pgx)

```go
// internal/db/postgres.go
package db

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jackc/pgx/v5/pgxpool/l logrusx"
)

type Config struct {
    DSN             string
    MaxConns        int32         // default 50
    MinConns        int32         // default 5
    MaxConnLifetime time.Duration // 1h
    MaxConnIdleTime time.Duration // 30m
    HealthCheck     time.Duration // 30s
}

func NewPostgresPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
    poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("parse dsn: %w", err)
    }

    poolCfg.MaxConns = cfg.MaxConns
    poolCfg.MinConns = cfg.MinConns
    poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
    poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
    poolCfg.HealthCheckPeriod = cfg.HealthCheck

    // Performance tuning
    poolCfg.ConnConfig.RuntimeParams["application_name"] = "rinco-crm-core"
    poolCfg.ConnConfig.RuntimeParams["statement_timeout"] = "5000"
    poolCfg.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "10000"

    pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
    if err != nil {
        return nil, fmt.Errorf("create pool: %w", err)
    }

    // Verify
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("ping: %w", err)
    }

    return pool, nil
}

// Health check
func PoolHealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    return pool.Ping(ctx)
}
```

### 2.7. Migration Tool (goose)

```sql
-- migrations/0001_create_tenants.sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  plan TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  isolation_mode TEXT DEFAULT 'SHARED',
  region TEXT NOT NULL,
  settings JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_tenants_status ON tenants(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tenants;
-- +goose StatementEnd
```

```sql
-- migrations/0002_create_users.sql
-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS ltree;

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  email TEXT NOT NULL,
  password_hash TEXT,
  full_name TEXT NOT NULL,
  parent_id UUID REFERENCES users(id),
  path LTREE NOT NULL,
  depth INT NOT NULL,
  role TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_path ON users USING GIST(path);
CREATE INDEX idx_users_tenant_role ON users(tenant_id, role);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
```

```go
// migrations/migrate.go
package main

import (
    "log"
    "os"

    "github.com/pressly/goose/v3"
    "github.com/jackc/pgx/v5/stdlib"
)

func main() {
    db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    goose.SetBaseFS(os.DirFS("migrations"))
    if err := goose.SetDialect("postgres"); err != nil {
        log.Fatal(err)
    }

    command := os.Args[1]
    switch command {
    case "up":
        if err := goose.Up(db, "."); err != nil {
            log.Fatal(err)
        }
    case "down":
        if err := goose.Down(db, "."); err != nil {
            log.Fatal(err)
        }
    case "status":
        if err := goose.Status(db, "."); err != nil {
            log.Fatal(err)
        }
    }
}
```

### 2.8. Connection Pooling (PgBouncer)

```ini
# /etc/pgbouncer/pgbouncer.ini
[databases]
rinco = host=postgres-primary port=5432 dbname=rinco
rinco_ro = host=postgres-replica port=5432 dbname=rinco
rinco_admin = host=postgres-primary port=5432 dbname=rinco_admin

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
auth_type = scram-sha-256
auth_file = /etc/pgbouncer/userlist.txt
admin_users = pgbouncer_admin
stats_users = pgbouncer_stats
pool_mode = transaction
max_client_conn = 5000
default_pool_size = 50
reserve_pool_size = 10
reserve_pool_timeout = 3
server_idle_timeout = 600
server_lifetime = 3600
server_connect_timeout = 15
client_login_timeout = 60
query_timeout = 30
query_wait_timeout = 30
client_idle_timeout = 0
tcp_keepalive = 1
tcp_keepidle = 30
tcp_keepintvl = 10
tcp_keepcnt = 5
log_connections = 1
log_disconnections = 1
log_pooler_errors = 1
```

```bash
# /etc/pgbouncer/userlist.txt
"rinco_app" "scram-sha-256=4096:xxx..."
"pgbouncer_admin" "scram-sha-256=4096:xxx..."
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
concurrent_compactors: 4
compaction_strategy: TimeWindowCompactionStrategy
compaction_window_size: 1
compaction_window_unit: DAYS
api_port: 10000
cql_port: 9042
rpc_address: 0.0.0.0
seed_provider:
  - class_name: org.apache.cassandra.locator.SimpleSeedProvider
    parameters:
      - seeds: "10.0.1.10,10.0.1.11,10.0.1.12"
endpoint_snitch: GossipingPropertyFileSnitch
auto_bootstrap: true
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

CREATE KEYSPACE rinco_leads
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
  AND compaction = {'class': 'TimeWindowCompactionStrategy', 'compaction_window_size': '1', 'compaction_window_unit': 'DAYS'}
  AND default_time_to_live = 7776000;  -- 90 days

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

CREATE TABLE rinco_chat.messages_by_reply (
  reply_to text,
  tenant_id text,
  event_time bigint,
  event_id uuid,
  text text,
  PRIMARY KEY ((reply_to, tenant_id), event_time)
);
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

CREATE TABLE rinco_chat.channel_unread (
  channel_id text,
  tenant_id text,
  user_id text,
  event_time bigint,
  PRIMARY KEY ((channel_id, tenant_id), user_id)
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
) WITH CLUSTERING ORDER BY (id DESC)
  AND default_time_to_live = 157680000;  -- 5 years
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
  retry_count int,
  PRIMARY KEY ((tenant_id, sent_at), event_id)
);

CREATE TABLE rinco_events.capi_failures (
  tenant_id text,
  failed_at timestamp,
  event_id uuid,
  payload text,
  retry_count int,
  next_retry_at timestamp,
  PRIMARY KEY ((tenant_id, failed_at), event_id)
);
```

#### Lead Raw (Landing Ingest)

```cql
CREATE TABLE rinco_leads.leads_raw (
  tenant_id text,
  event_time bigint,
  event_id uuid,
  idempotency_key text,
  full_name text,
  phone text,
  email text,
  utm_source text,
  utm_medium text,
  utm_campaign text,
  utm_content text,
  utm_term text,
  fbclid text,
  gclid text,
  ip_address text,
  user_agent text,
  source text,
  form_id text,
  extra map<text, text>,
  processed boolean,
  lead_id uuid,
  PRIMARY KEY ((tenant_id, event_time), event_id)
) WITH CLUSTERING ORDER BY (event_id DESC);

CREATE UNIQUE INDEX ON rinco_leads.leads_raw ((tenant_id), idempotency_key);
```

### 3.4. Compaction Strategy

- **TWCS (Time Window Compaction Strategy)** cho chat logs.
- Window = 1 day.
- Giảm write amplification, tối ưu cho time-series.
- **LCS (Leveled Compaction Strategy)** cho metadata (channels, users).
- **STCS (Size Tiered Compaction Strategy)** không dùng cho production.

### 3.5. Repository (Go + gocql)

```go
// internal/repository/scylla/message_repository.go
package scylla

import (
    "context"
    "time"

    "github.com/gocql/gocql"
)

type Message struct {
    ChannelID  string
    TenantID   string
    EventTime  time.Time
    EventID    gocql.UUID
    SenderID   string
    MessageType int8
    Text       string
}

type MessageRepository struct {
    session *gocql.Session
}

func NewMessageRepository(session *gocql.Session) *MessageRepository {
    return &MessageRepository{session: session}
}

func (r *MessageRepository) Insert(ctx context.Context, m *Message) error {
    return r.session.Query(`
        INSERT INTO rinco_chat.messages
            (channel_id, tenant_id, event_time, event_id, sender_id, message_type, text)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        USING TIMESTAMP ?
    `,
        m.ChannelID, m.TenantID, m.EventTime, m.EventID,
        m.SenderID, m.MessageType, m.Text, time.Now().UnixMicro(),
    ).WithContext(ctx).Exec()
}

func (r *MessageRepository) GetByChannel(ctx context.Context, channelID, tenantID string, limit int) ([]Message, error) {
    var msgs []Message
    iter := r.session.Query(`
        SELECT channel_id, tenant_id, event_time, event_id, sender_id, message_type, text
        FROM rinco_chat.messages
        WHERE channel_id = ? AND tenant_id = ?
        LIMIT ?
    `, channelID, tenantID, limit).
        WithContext(ctx).
        Iter()

    defer iter.Close()
    for iter.Scan(&msgs) {
        // ...
    }
    return msgs, iter.Close()
}
```

### 3.6. Connection Setup (gocql)

```go
// internal/db/scylla.go
package db

import (
    "time"

    "github.com/gocql/gocql"
)

func NewScyllaSession(hosts []string, keyspace string) (*gocql.Session, error) {
    cluster := gocql.NewCluster(hosts...)
    cluster.Keyspace = keyspace
    cluster.Consistency = gocql.Quorum
    cluster.ProtoVersion = 5
    cluster.Compression = &gocql.SnappyCompressor{}
    cluster.Timeout = 5 * time.Second
    cluster.ConnectTimeout = 10 * time.Second
    cluster.NumConns = 4
    cluster.Pool = gocql.NewSimplePool(hosts, &pool.Config{
        PoolSize:    20,
        MaxStreams:  100,
    })

    return gocql.WrapSession(cluster.CreateSession, nil, nil)
}
```

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
            <shard>
                <internal_replication>true</internal_replication>
                <replica><host>node3</host><port>9000</port></replica>
                <replica><host>node4</host><port>9000</port></replica>
            </shard>
        </rinco_cluster>
    </remote_servers>
    <macros>
        <shard>01</shard>
        <replica>node1</replica>
    </macros>
    <zookeeper>
        <node><host>zoo1</host><port>2181</port></node>
        <node><host>zoo2</host><port>2181</port></node>
        <node><host>zoo3</host><port>2181</port></node>
    </zookeeper>
</yandex>
```

### 4.2. Database & Tables

```sql
CREATE DATABASE IF NOT EXISTS rinco_analytics;

-- Events Raw
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

-- Aggregated
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

-- Funnel
CREATE TABLE rinco_analytics.funnel (
  event_date Date,
  tenant_id LowCardinality(String),
  campaign_id UUID,
  step LowCardinality(String),
  count UInt64,
  unique_users AggregateFunction(uniq, String)
) ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_date, campaign_id, step);

-- Lead Performance (per tenant funnel analytics)
CREATE TABLE rinco_analytics.lead_performance (
  event_date Date,
  tenant_id LowCardinality(String),
  owner_user_id UUID,
  source LowCardinality(String),
  status LowCardinality(String),
  count UInt64,
  total_value AggregateFunction(sum, Decimal(18, 4))
) ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_date, owner_user_id);
```

### 4.3. Materialized Views

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

-- Materialized View cho lead performance
CREATE MATERIALIZED VIEW rinco_analytics.lead_performance_mv
TO rinco_analytics.lead_performance AS
SELECT
  event_date,
  tenant_id,
  JSONExtractString(properties, 'owner_user_id') AS owner_user_id,
  source,
  JSONExtractString(properties, 'status') AS status,
  count() AS count,
  sumState(toDecimal64OrZero(JSONExtractFloat(properties, 'value'), 4)) AS total_value
FROM rinco_analytics.events_raw
WHERE event_name = 'lead.status_changed'
GROUP BY event_date, tenant_id, owner_user_id, source, status;
```

### 4.4. Per-Tenant Row Policy

```sql
-- User per tenant
CREATE USER tenant_apex IDENTIFIED BY 'xxx';

-- Grants
GRANT SELECT ON rinco_analytics.* TO tenant_apex;

-- Row policy
CREATE ROW POLICY tenant_apex_filter ON rinco_analytics.events_raw
  USING tenant_id = 'apex' TO tenant_apex;

CREATE ROW POLICY tenant_apex_agg_filter ON rinco_analytics.events_aggregated
  USING tenant_id = 'apex' TO tenant_apex;

-- Common queries
SELECT
  event_date,
  sum(count) AS total_events,
  sumMerge(unique_users) AS unique_users
FROM rinco_analytics.events_aggregated
WHERE tenant_id = 'apex'
  AND event_date BETWEEN '2026-09-01' AND '2026-09-07'
  AND event_name = 'lead.created'
GROUP BY event_date
ORDER BY event_date DESC;
```

### 4.5. Repository (Go + clickhouse-go)

```go
// internal/repository/clickhouse/event_repository.go
package clickhouse

import (
    "context"
    "time"

    "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Event struct {
    TenantID   string
    EventID    string
    SessionID  string
    EventName  string
    Timestamp  time.Time
    Properties map[string]any
}

type EventRepository struct {
    conn driver.Conn
}

func (r *EventRepository) Insert(ctx context.Context, e *Event) error {
    return r.conn.Exec(ctx, `
        INSERT INTO rinco_analytics.events_raw (
            timestamp, tenant_id, event_id, session_id, event_name, properties
        ) VALUES (?, ?, ?, ?, ?, ?)
    `, e.Timestamp, e.TenantID, e.EventID, e.SessionID, e.EventName, e.Properties)
}

func (r *EventRepository) Dashboard(ctx context.Context, tenantID string, days int) ([]map[string]any, error) {
    rows, err := r.conn.Query(ctx, `
        SELECT
          event_date,
          sum(count) AS events,
          sumMerge(unique_users) AS users
        FROM rinco_analytics.events_aggregated
        WHERE tenant_id = ? AND event_date >= today() - INTERVAL ? DAY
        GROUP BY event_date
        ORDER BY event_date DESC
    `, tenantID, days)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []map[string]any
    for rows.Next() {
        var (
            date   time.Time
            events uint64
            users  uint64
        )
        if err := rows.Scan(&date, &events, &users); err != nil {
            return nil, err
        }
        out = append(out, map[string]any{
            "date":   date,
            "events": events,
            "users":  users,
        })
    }
    return out, rows.Err()
}
```

### 4.6. Connection Setup (clickhouse-go)

```go
// internal/db/clickhouse.go
package db

import (
    "context"
    "fmt"
    "time"

    "github.com/ClickHouse/clickhouse-go/v2"
)

type ClickHouseConfig struct {
    Addr     string
    Database string
    Username string
    Password string
}

func NewClickHouse(cfg ClickHouseConfig) (driver.Conn, error) {
    conn, err := clickhouse.Open(&clickhouse.Options{
        Addr:        []string{cfg.Addr},
        Database:    cfg.Database,
        Username:    cfg.Username,
        Password:    cfg.Password,
        DialTimeout: 5 * time.Second,
        ReadTimeout: 30 * time.Second,
        Compression: &clickhouse.Compression{Method: clickhouse.CompressionZSTD},
        MaxOpenConns: 50,
        MaxIdleConns: 10,
    })
    if err != nil {
        return nil, fmt.Errorf("clickhouse open: %w", err)
    }
    return conn, nil
}
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
      journalCompressor: snappy
replication:
  replSetName: rinco-rs
sharding:
  clusterRole: shardsvr
net:
  port: 27017
  bindIp: 0.0.0.0
security:
  authorization: enabled
systemLog:
  destination: file
  path: /var/log/mongodb/mongod.log
```

### 5.3. Collections & Validators

```javascript
// Form submissions
db.createCollection("form_submissions", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["tenant_id", "form_id", "submitted_at", "data"],
      properties: {
        tenant_id: { bsonType: "string", pattern: "^[a-z0-9-]{3,50}$" },
        form_id: { bsonType: "string" },
        submitted_at: { bsonType: "date" },
        ip_address: { bsonType: "string" },
        user_agent: { bsonType: "string" },
        data: { bsonType: "object" },
        meta: { bsonType: "object" },
        status: { enum: ["new", "processed", "rejected"], bsonType: "string" }
      }
    }
  },
  validationLevel: "moderate",
  validationAction: "error"
});

db.form_submissions.createIndex({ tenant_id: 1, form_id: 1, submitted_at: -1 });
db.form_submissions.createIndex({ "data.email": 1 });
db.form_submissions.createIndex({ "data.phone": 1 });
db.form_submissions.createIndex({ tenant_id: 1, status: 1, submitted_at: -1 });

// Form Definitions
db.form_definitions.createIndex({ tenant_id: 1, slug: 1 }, { unique: true });
db.form_definitions.createIndex({ tenant_id: 1, is_active: 1 });

// Survey responses
db.createCollection("survey_responses", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["tenant_id", "survey_id", "respondent_id", "answers"],
      properties: {
        tenant_id: { bsonType: "string" },
        survey_id: { bsonType: "string" },
        respondent_id: { bsonType: "string" },
        answers: { bsonType: "array" }
      }
    }
  }
});
```

### 5.4. Sharding

```javascript
sh.enableSharding("rinco_forms");
sh.shardCollection("rinco_forms.form_submissions", { tenant_id: 1, submitted_at: 1 });
sh.shardCollection("rinco_forms.form_definitions", { tenant_id: 1 });
```

### 5.5. Repository (Go + mongo-driver)

```go
// internal/repository/mongo/form_repository.go
package mongo

import (
    "context"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
)

type FormSubmission struct {
    TenantID     string                 `bson:"tenant_id"`
    FormID       string                 `bson:"form_id"`
    SubmittedAt  time.Time              `bson:"submitted_at"`
    IPAddress    string                 `bson:"ip_address"`
    UserAgent    string                 `bson:"user_agent"`
    Data         map[string]any         `bson:"data"`
    Meta         map[string]any         `bson:"meta"`
    Status       string                 `bson:"status"`
}

type FormRepository struct {
    collection *mongo.Collection
}

func (r *FormRepository) Insert(ctx context.Context, fs *FormSubmission) error {
    _, err := r.collection.InsertOne(ctx, fs)
    return err
}

func (r *FormRepository) FindByTenant(ctx context.Context, tenantID string, limit int) ([]FormSubmission, error) {
    cur, err := r.collection.Find(ctx,
        bson.M{"tenant_id": tenantID},
        options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "submitted_at", Value: -1}}),
    )
    if err != nil {
        return nil, err
    }
    defer cur.Close(ctx)

    var out []FormSubmission
    if err := cur.All(ctx, &out); err != nil {
        return nil, err
    }
    return out, nil
}
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
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb

slowlog-log-slower-than 10000
slowlog-max-len 128

cluster-enabled yes
cluster-config-file nodes.conf
cluster-node-timeout 5000
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
| `tenant-scoped:ai:cache:{tid}:{prefix_hash}` | Tenant-isolated AI cache | 600s |
| `fingerprint:js:{ua_hash}` | Browser fingerprint | 86400s |

### 6.3. Cluster Setup

```yaml
# docker-compose.yml
valkey-cluster:
  image: valkey/valkey:9
  command: >
    valkey-server
    --cluster-enabled yes
    --cluster-config-file nodes.conf
    --cluster-node-timeout 5000
    --appendonly yes
    --maxmemory 8gb
    --maxmemory-policy allkeys-lru
  ports:
    - "6379:6379"
    - "16379:16379"
```

### 6.4. Streams (Lightweight Event Log)

```bash
# Publish
XADD rinco:events:lead * type "lead.created" tenant "apex" data "{...}"

# Subscribe
XREAD BLOCK 0 STREAMS rinco:events:lead $

# Consumer group
XGROUP CREATE rinco:events:lead crm-workers $ MKSTREAM
XREADGROUP GROUP crm-workers worker-1 COUNT 10 STREAMS rinco:events:lead >
```

### 6.5. Pub/Sub

```bash
# Publish
PUBLISH chat:room:c-123 "{...}"

# Subscribe
SUBSCRIBE chat:room:c-123
```

### 6.6. Repository (Go + redis-rs / fred)

```go
// internal/repository/valkey/cache_repository.go
package valkey

import (
    "context"
    "encoding/json"
    "time"

    "github.com/redis/go-redis/v9"
)

type CacheRepository struct {
    client *redis.Client
}

func (r *CacheRepository) GetTenantConfig(ctx context.Context, tenantID string) (map[string]any, error) {
    val, err := r.client.Get(ctx, "tenant:"+tenantID+":config").Result()
    if err != nil {
        return nil, err
    }
    var cfg map[string]any
    return cfg, json.Unmarshal([]byte(val), &cfg)
}

func (r *CacheRepository) SetTenantConfig(ctx context.Context, tenantID string, cfg map[string]any, ttl time.Duration) error {
    b, _ := json.Marshal(cfg)
    return r.client.Set(ctx, "tenant:"+tenantID+":config", b, ttl).Err()
}

func (r *CacheRepository) RateLimit(ctx context.Context, tenantID, action string, limit int, windowSec int) (bool, error) {
    key := "ratelimit:" + tenantID + ":" + action
    pipe := r.client.Pipeline()
    incr := pipe.Incr(ctx, key)
    pipe.Expire(ctx, key, time.Duration(windowSec)*time.Second)
    _, err := pipe.Exec(ctx)
    if err != nil {
        return false, err
    }
    return incr.Val() <= int64(limit), nil
}
```

---

## 7. MinIO/S3 – Object Storage

### 7.1. Setup

```yaml
# docker-compose.yml
minio:
  image: minio/minio:latest
  command: server /data --console-address ":9001"
  environment:
    MINIO_ROOT_USER: minio
    MINIO_ROOT_PASSWORD: xxx
  ports:
    - "9000:9000"
    - "9001:9001"
  volumes:
    - minio-data:/data
```

### 7.2. Buckets

| Bucket | Purpose | Retention |
|--------|---------|-----------|
| `rinco-tenant-assets` | Logo, favicon, custom CSS | Forever |
| `rinco-chat-files` | Chat attachments | 90 days |
| `rinco-user-avatars` | User profile pictures | Forever |
| `rinco-meeting-recordings` | Meeting recordings | Configurable |
| `rinco-ai-models` | ML model artifacts | Forever |
| `rinco-backups` | DB backups | 30 days hot, 1 year cold |
| `rinco-cold-archive` | Old data archive | 7 years |
| `rinco-form-uploads` | User uploaded files | Per form |

### 7.3. Bucket Policies

- Per-tenant prefix isolation.
- Lifecycle rules (auto-delete).
- Versioning enabled for critical buckets.
- Server-side encryption (SSE-KMS).

```bash
# Lifecycle policy
mc ilm add rinco/rinco-chat-files --expiry-days 90
mc ilm add rinco/rinco-cold-archive --expiry-days 2555

# Versioning
mc version enable rinco/rinco-tenant-assets

# Replication
mc replicate add rinco/rinco-backups --destination-bucket rinco-backups-archive
```

### 7.4. Presigned Upload (Go)

```go
// internal/storage/minio.go
package storage

import (
    "context"
    "time"

    "github.com/minio/minio-go/v7"
)

type Storage struct {
    client *minio.Client
}

func (s *Storage) GeneratePresignedURL(ctx context.Context, bucket, key string, size int64, contentType string) (string, error) {
    presignedURL, err := s.client.PresignedPutObject(ctx, bucket, key, 15*time.Minute)
    if err != nil {
        return "", err
    }
    return presignedURL.String(), nil
}

func (s *Storage) GeneratePresignedGet(ctx context.Context, bucket, key string) (string, error) {
    reqParams := url.Values{}
    presignedURL, err := s.client.PresignedGetObject(ctx, bucket, key, 15*time.Minute, reqParams)
    if err != nil {
        return "", err
    }
    return presignedURL.String(), nil
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

-- Tenant-scoped index
CREATE INDEX idx_knowledge_tenant ON knowledge_documents(tenant_id);
```

### 8.3. Qdrant Setup

```yaml
qdrant:
  image: qdrant/qdrant:latest
  environment:
    QDRANT__SERVICE__GRPC_PORT: 6334
    QDRANT__SERVICE__HTTP_PORT: 6333
    QDRANT__STORAGE__PERFORMANCE__INDEXING_THRESHOLD_KB: 20000
  volumes:
    - qdrant-data:/qdrant/storage
```

### 8.4. Collections

```python
# Qdrant collection per tenant (or sharded single collection)
from qdrant_client import QdrantClient
from qdrant_client.http import models

client = QdrantClient(host="qdrant", port=6333)

client.create_collection(
    collection_name="apex_kb",
    vectors_config={
        "size": 1024,
        "distance": "Cosine",
        "hnsw_config": {"m": 16, "ef_construct": 100}
    },
    quantization_config=models.ScalarQuantization(
        scalar=models.ScalarQuantizationConfig(
            type=models.ScalarType.INT8,
            quantile=0.99,
            always_ram=True
        )
    ),
    optimizers_config=models.OptimizersConfig(
        indexing_threshold=20000
    )
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
    limit=5,
    with_payload=True
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
    MEILI_DB_PATH: /meili_data
  volumes:
    - meili-data:/meili_data
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

// CRM contacts index
{
  "uid": "contacts",
  "primaryKey": "id",
  "searchableAttributes": ["full_name", "email", "phone", "company"],
  "filterableAttributes": ["tenant_id", "owner_user_id"],
  "sortableAttributes": ["created_at", "updated_at"]
}
```

### 9.3. Search API (Go)

```go
// internal/search/meili.go
package search

import (
    "github.com/meilisearch/meilisearch-go"
)

type MeiliClient struct {
    client *meilisearch.Client
}

func (m *MeiliClient) SearchLeads(tenantID, query string) (*meilisearch.SearchResponse, error) {
    req := &meilisearch.SearchRequest{
        Filter: "tenant_id = " + tenantID,
        Sort:   []string{"created_at:desc"},
        Limit:  20,
    }
    return m.client.Index("leads").Search(query, req)
}
```

---

## 10. Migration Strategy

### 10.1. Schema Versioning

```sql
-- PostgreSQL
CREATE TABLE schema_migrations (
  version TEXT PRIMARY KEY,
  description TEXT,
  applied_at TIMESTAMPTZ DEFAULT now()
);

-- ScyllaDB
CREATE TABLE rinco_admin.schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamp,
  description text
);

-- ClickHouse
CREATE TABLE rinco_analytics.schema_migrations (
  version String,
  applied_at DateTime DEFAULT now(),
  description String
) ENGINE = MergeTree ORDER BY version;
```

### 10.2. Migration Tool (Goose) đã trình bày ở §2.7

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
DO $$
DECLARE last_id UUID;
BEGIN
  LOOP
    UPDATE leads SET new_column = old_column
    WHERE id > COALESCE(last_id, '00000000-0000-0000-0000-000000000000'::UUID)
      AND new_column IS NULL
    LIMIT 1000
    RETURNING id INTO last_id;
    EXIT WHEN last_id IS NULL;
    COMMIT;
  END LOOP;
END $$;

-- Step 3: Code switch to new column
-- (deploy app reading from new_column)

-- Step 4: Drop old column
ALTER TABLE leads DROP COLUMN old_column;
```

---

## 11. Backup & DR

### 11.1. PostgreSQL Backup

```bash
#!/bin/bash
# scripts/backup-postgres.sh
set -e

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/postgres/$TIMESTAMP"

# Full backup using pg_basebackup (binary, supports PITR)
pg_basebackup -h postgres-primary -D $BACKUP_DIR/base \
  --checkpoint=fast --wal-method=stream

# Logical backup (optional, smaller)
pg_dump --format=custom --compress=9 --no-owner --no-privileges \
  --dbname=rinco > $BACKUP_DIR/rinco.dump

# Compress
tar czf $BACKUP_DIR.tar.gz $BACKUP_DIR
rm -rf $BACKUP_DIR

# Upload to MinIO
mc cp $BACKUP_DIR.tar.gz minio/backups/postgres/$TIMESTAMP.tar.gz

# Verify checksum
sha256sum $BACKUP_DIR.tar.gz > $BACKUP_DIR.tar.gz.sha256
mc cp $BACKUP_DIR.tar.gz.sha256 minio/backups/postgres/$TIMESTAMP.sha256

# Retain 30 days
mc rm --recursive --force --older-than 30d minio/backups/postgres/

echo "Backup completed: $TIMESTAMP"
```

```bash
# Continuous archiving (WAL)
# In postgresql.conf:
archive_command = 'mc cp %p minio/backups/postgres/wal/%f'
archive_timeout = '60s'
```

### 11.2. ScyllaDB Backup

```bash
#!/bin/bash
# scripts/backup-scylla.sh
set -e

KEYSPACE="rinco_chat"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/scylla/$TIMESTAMP"

# Snapshot per node
for node in 10.0.1.10 10.0.1.11 10.0.1.12; do
  ssh $node "nodetool snapshot $KEYSPACE -t $TIMESTAMP"
  scp -r $node:/var/lib/scylla/data/$KEYSPACE/snapshots/$TIMESTAMP \
    $BACKUP_DIR/$node/
done

# Compress and upload
tar czf $BACKUP_DIR.tar.gz $BACKUP_DIR
mc cp $BACKUP_DIR.tar.gz minio/backups/scylla/

# Cleanup
for node in 10.0.1.10 10.0.1.11 10.0.1.12; do
  ssh $node "nodetool clearsnapshot -t $TIMESTAMP"
done

echo "ScyllaDB backup completed: $TIMESTAMP"
```

### 11.3. ClickHouse Backup

```bash
# ClickHouse native backup
clickhouse-backup create \
  --config=/etc/clickhouse-backup/config.yml \
  --name=rinco_$(date +%Y%m%d_%H%M%S)

clickhouse-backup upload default \
  --config=/etc/clickhouse-backup/config.yml

# Config
cat > /etc/clickhouse-backup/config.yml <<EOF
general:
  remote_storage: s3
  max_concurrency: 4
clickhouse:
  host: localhost
  port: 9000
s3:
  access_key: minio
  secret_key: xxx
  bucket: rinco-backups
  endpoint: http://minio:9000
  region: us-east-1
  path: clickhouse/
EOF
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
- Use `redis-rs` hoặc `fred` (async Rust).

### 12.5. Database per Service

- **crm-core:** PostgreSQL.
- **chat-engine:** ScyllaDB + Valkey.
- **landing-ingest:** ScyllaDB + ClickHouse.
- **analytics:** ClickHouse.
- **auth:** PostgreSQL.
- **notification:** PostgreSQL + Valkey.
- **ai-services:** PostgreSQL + Qdrant + Valkey.

---

## 13. Audit Report

### 13.1. Phần đã đủ chi tiết ✓

| Mục | Nội dung | Đánh giá |
|-----|---------|-----------|
| §1 – Tổng quan | Ma trận quyết định + Read/Write path | ✓ Tốt |
| §2 – PostgreSQL | Schema, RLS, PgBouncer | ✓ Khá |
| §3 – ScyllaDB | Tables, compaction | ⚠ Cần thêm code Go |
| §4 – ClickHouse | MV, row policy | ✓ Khá |
| §5 – MongoDB | Validator, sharding | ⚠ Cần thêm |
| §6 – Valkey | Use cases, streams | ⚠ Cần key patterns chi tiết |

### 13.2. Phần còn thiếu ⚠

| Mục | Thiếu | Hướng bổ sung |
|-----|--------|---------------|
| Repository code | Thiếu pattern Go cho Scylla/ClickHouse/Mongo | Bổ sung §3.5, §4.5, §5.5 |
| Migration zero-downtime | Chỉ nói chung chung | Bổ sung pattern expand/contract |
| Indexes chi tiết | Chưa có GIN/GIST/partial examples | Bổ sung §2.3.4 |
| Multi-region DR | Thiếu runbook | Bổ sung §19 |
| Cost estimation | Không có | Bổ sung §20 |
| AI cache isolation | Không rõ | Bổ sung Valkey key cho tenant-scoped AI |

### 13.3. Mâu thuẫn nội bộ ✗

| Vị trí | Mâu thuẫn | Hướng xử lý |
|--------|-----------|--------------|
| §6.2 Key pattern `tenant:{tid}:config` vs §6 chưa có format | Format chưa thống nhất | Standardize `<scope>:<tid>:<resource>` |
| §2.3 LTREE `path` chưa index `tenant_id` trong composite | Query hot path sẽ scan lớn | Add composite index `(tenant_id, path)` |
| §3.3 Messages default_time_to_live 90 days vs retention policy 7 năm | Chưa align | Add tiered storage policy |

### 13.4. Phần cần code example cụ thể 💡

| Mục | Cần code |
|-----|---------|
| §2.5 | Repository cho Lead với sqlc |
| §3.5 | gocql session + insert |
| §4.5 | clickhouse-go insert |
| §11 | Backup script chạy được |

---

## 14. Edge Cases & Error Scenarios

### 14.1. PostgreSQL

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| **DB-01** | **Deadlock giữa 2 transactions update cùng lead** | Postgres log "deadlock detected" | Retry với exponential backoff (max 3 lần), return 503 |
| **DB-02** | **Connection pool exhausted** | pgxpool.Acquire timeout | Hystrix fallback; trả cached value; alert SRE |
| **DB-03** | **Postgres primary down, replica lag** | SELECT FROM replica trả > 5s lag | Failover, route read sang primary hoặc wait cho replica catch-up |
| **DB-04** | **RLS policy block legitimate query** | App get 0 row nhưng có data | Check `SET LOCAL app.current_tenant_id` đã set chưa |
| **DB-05** | **Foreign key violation** | Insert order fail | ON DELETE SET NULL hoặc transaction re-order |
| **DB-06** | **Long-running query (10s+)** | statement_timeout | Kill query, return 504; log slow query |
| **DB-07** | **Disk full** | Postgres shutdown | WAL archive fail → page SRE, manual cleanup |
| **DB-08** | **Sequence/UUID collision** | PK violation | UUIDv4/v7 thay vì SERIAL; probability ≈ 0 |
| **DB-09** | **JSONB malformed** | Insert fail | Validate JSON trước insert bằng Go struct |
| **DB-10** | **Index bloat** | pgstat, query chậm dần | pg_repack online |

### 14.2. ScyllaDB

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| **DB-11** | **Node failure trong cluster** | Driver trả NoHostAvailable | Retry với token-aware routing; tự động fail trong 5s |
| **DB-12** | **Hinted handoff queue đầy** | nodetool status | Giảm write rate hoặc tăng cluster size |
| **DB-13** | **Read repair storm** | CPU spike | Tăng read consistency; repair async |
| **DB-14** | **Tombstone compaction** | Disk usage tăng | Major compaction thủ công |
| **DB-15** | **Schema disagreement** | Driver query fail | Sync schema qua cụm; rolling restart |
| **DB-16** | **Batch write quá lớn** | Batch size > 100 | Chia batch, dùng logged batch cho critical |
| **DB-17** | **Partition hot (1 channel 10K msgs/s)** | Latency spike | Bucket theo hour; tách partition key |
| **DB-18** | **TTL data bị xóa sớm** | Query trả 0 | TTL đủ lớn, backup trước khi TTL hit |
| **DB-19** | **Consistency level quá cao** | Latency cao | LOCAL_QUORUM thay vì ALL |
| **DB-20** | **Materialized view stale** | Dashboard sai | Refresh async, alert nếu > 1h |

### 14.3. ClickHouse

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| **DB-21** | **Disk full trên 1 shard** | INSERT fail | Alert SRE; rotate shard; archive old partitions |
| **DB-22** | **Merge storm** | CPU > 90% | Optimize `parts_to_throw_insert`; manual merge |
| **DB-23** | **Replica lag > 5 phút** | Query distributed trả stale | Set `select_sequential_consistency=1` |
| **DB-24** | **ZooKeeper session expired** | Replication break | Restart ClickHouse, reconnect ZK |
| **DB-25** | **Distributed query trên 1 shard down** | Query timeout | SETTINGS `max_execution_time`; fallback to local shard |
| **DB-26** | **JSON property parse fail** | Default value 0 | `JSONExtractFloatOrZero` thay vì `JSONExtractFloat` |
| **DB-27** | **TTL xóa data quá sớm** | Query trả 0 | TTL ≥ retention requirement + buffer |
| **DB-28** | **Row policy bypass** | User xem cross-tenant data | Test row policy; audit log access |
| **DB-29** | **Mutation quá nhiều** | INSERT chậm | Gộp batch INSERT, dùng Buffer table |
| **DB-30** | **ClickHouse Keeper split-brain** | Một số node không replicate | Force leader election; kiểm tra mạng |

### 14.4. Valkey

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| **DB-31** | **OOM (max memory hit)** | Write trả OOM error | LRU eviction; giảm TTL; shard key |
| **DB-32** | **AOF file corrupt** | Restart fail | redis-check-aof --fix; backup last known good |
| **DB-33** | **Cluster split-brain** | Một node tự nhận master | Set cluster-node-timeout ngắn; min-replicas-to-write |
| **DB-34** | **Slow command (KEYS, SMEMBERS lớn)** | Latency spike | Dùng SCAN; cache kết quả; rename command |
| **DB-35** | **Hot key (1 key > 100K QPS)** | CPU spike trên 1 shard | Multi-key hash; replica read |
| **DB-36** | **Lua script timeout** | Eval error | Tăng lua-time-limit hoặc refactor script |
| **DB-37** | **Pub/Sub message lost** | Subscriber miss | Dùng Streams thay vì Pub/Sub |
| **DB-38** | **Streams consumer lag > 1 triệu** | Lag alarm | Tăng consumer; pending claim; PEL trim |
| **DB-39** | **Tenant leak qua cache key** | Cross-tenant data | Format `<scope>:<tid>:<resource>` enforced |
| **DB-40** | **Persistence bị disable** | Restart mất data | `appendonly yes` mandatory |

### 14.5. MongoDB

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| **DB-41** | **Schema validation fail** | Insert reject | Validate trước khi insert Go-side; relax validation level |
| **DB-42** | **Replica set election > 30s** | Write fail | Tăng heartbeat; priority adjustment |
| **DB-43** | **Sharded query scatter-gather** | Query chậm | Hashed shard key; covered query |
| **DB-44** | **Migration lock kéo dài** | Index build chậm | Dùng `db.collection.createIndex({...}, {background: true})` |
| **DB-45** | **WiredTiger cache miss** | Latency tăng | Tăng cacheSizeGB |
| **DB-46** | **Orphaned document (shard move)** | Cross-shard transaction | Retry; ref chưa chunk |
| **DB-47** | **Aggregation pipeline memory > 100MB** | Pipeline fail | `$out` hoặc `allowDiskUse: true` |
| **DB-48** | **BSON size > 16MB** | Insert fail | GridFS hoặc MinIO thay thế |
| **DB-49** | **Field name conflict (case-sensitive)** | Query miss | Standardize lowercase; schema validator |
| **DB-50** | **Backup quá lớn** | mongodump chậm | Incremental oplog backup |

### 14.6. MinIO

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| **DB-51** | **Drive fail trong EC** | Healing tự động | Đợi heal, alert nếu > 24h |
| **DB-52** | **Bucket policy lock sai** | Access denied | Admin unlock; audit policy changes |
| **DB-53** | **Presigned URL expire** | Upload fail sau 15 phút | Client refresh; tăng duration cho big file |
| **DB-54** | **Multipart upload incomplete** | Storage charge | Lifecycle rule abort incomplete |
| **DB-55** | **Encryption key bị mất** | Data unreadable | KMS rotation policy; backup key ở Vault |
| **DB-56** | **Versioning stack** | Disk đầy | Lifecycle expire old versions |
| **DB-57** | **Cross-region replication lag > 1h** | RPO vi phạm | Alert; check bandwidth |
| **DB-58** | **S3 API throttling** | 503 SlowDown | Tăng retry budget; queue upload |

### 14.7. Qdrant + pgvector

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| **DB-59** | **Index corruption** | Query trả NaN | Rebuild index từ raw data |
| **DB-60** | **Out-of-memory khi build HNSW** | OOM crash | Tăng RAM hoặc dùng DiskANN |
| **DB-61** | **Vector dimension mismatch** | Insert reject | Validate trước bằng embedding service |
| **DB-62** | **Payload filter quá phức tạp** | Query timeout | Denormalize payload hoặc pre-filter |
| **DB-63** | **Tenant leak trong Qdrant** | Cross-tenant data | Mandatory `tenant_id` filter check |
| **DB-64** | **Embedding model version mismatch** | Search sai | Re-embed all data khi model upgrade |

### 14.8. Migration & Schema

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| **DB-65** | **Migration stuck ở lock** | Postgres idle in transaction | Kill long transaction; alert |
| **DB-66** | **Backfill quá chậm** | Migration không xong trong window | Batch nhỏ hơn; parallelism |
| **DB-67** | **Forward + backward migration mismatch** | Rollback fail | Test cả Up/Down trên staging |
| **DB-68** | **Schema drift giữa staging và prod** | Production query fail | Migration trên prod identical staging |
| **DB-69** | **ALTER TABLE blocking long** | Postgres DDL lock | `CREATE INDEX CONCURRENTLY`; pg_repack |
| **DB-70** | **Cross-DB inconsistency** | Scylla có nhưng Postgres không | Outbox pattern + idempotent consumer |

### 14.9. Network & Infrastructure

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| **DB-71** | **WireGuard tunnel down** | Cluster split | Fallback SSH tunnel; auto reconnect |
| **DB-72** | **DNS resolution fail** | Driver fail | Cache IP trong Valkey; retry |
| **DB-73** | **TLS handshake slow** | Connection pool drain | TLS session resumption; pool warmup |
| **DB-74** | **Bandwidth giữa DC** | Replication lag | Compression; throttled batch |
| **DB-75** | **Storage IOPS exhausted** | Latency tăng | Tier storage; NVMe cache |

---

## 15. Sequence Diagrams

### 15.1. Lead Create Flow (PostgreSQL + ScyllaDB + ClickHouse)

```
[Client Form Submit]
        │  POST /api/ingest/v1/leads (HMAC signed)
        ▼
[api-gateway]
        │  Validate HMAC + tenant context
        ▼
[landing-ingest service]
        │
        ├──► [Valkey] Idempotency check (key: idem:{tid}:{key})
        │     OK → return existing lead_id
        │     MISS → continue
        │
        ├──► [ScyllaDB rinco_leads.leads_raw] INSERT (event_id, raw payload)
        │     Async write, fire-and-forget
        │
        ├──► [NATS JetStream] publish "tenant.lead.created"
        │     (event_id, payload, retry policy)
        │
        └──► Return 200 OK {lead_id, event_id}

[NATS Consumer: crm-core]
        │
        ├──► [PostgreSQL leads] INSERT (transactional, with RLS context)
        │     Set tenant_id, owner_user_id, RLS policy auto-filter
        │
        └──► [NATS] publish "lead.scored.request"

[NATS Consumer: ai-scoring]
        │
        ├──► Extract features (utm_source, time_on_site, ...)
        ├──► [ONNX Runtime] XGBoost inference (< 5ms)
        │
        └──► [PostgreSQL leads.score] UPDATE
              + [NATS] publish "lead.scored"

[NATS Consumer: meta-capi]
        │
        ├──► [Valkey] Check tenant quota + EMQ score
        ├──► [Facebook CAPI API] POST (with HMAC signature)
        │     Success → [ScyllaDB capi_events] write status
        │     Failure → retry with exp backoff
        │
        └──► [NATS] publish "capi.sent"

[NATS Consumer: clickhouse-pipeline]
        │
        └──► [ClickHouse events_raw] INSERT
              + auto-update Materialized Views (events_aggregated, funnel)
```

### 15.2. Message Send Flow (ScyllaDB + NATS)

```
[Client WebSocket]
        │  Binary FlatBuffer Message
        ▼
[chat-gateway (Rust + tokio-uring)]
        │
        ├──► [Valkey] Rate limit per user (token bucket)
        │     10 msg/s; overflow → 429
        │
        ├──► [Valkey] Singleflight coalesce per channel
        │     Merge duplicate reads in 10ms window
        │
        ├──► [ScyllaDB rinco_chat.messages] INSERT
        │     Partition: (channel_id, tenant_id), Clustering: event_time
        │
        ├──► [NATS JetStream] publish "chat.message.sent"
        │     (channel_id, message_id, sender, mentions)
        │
        └──► Broadcast to subscribers (WebSocket fan-out)

[Subscriber 1: chat-realtime]
        │
        ├──► [Valkey presence:channel:{cid}] update unread
        ├──► Push to connected WebSocket clients

[Subscriber 2: notification]
        │
        ├──► [Valkey notif:queue:{uid}] enqueue (offline users)
        └──► Send push notification (FCM/APNS)

[Subscriber 3: ai-conversation] (nếu mention AI bot)
        │
        ├──► [BGE-M3] embed message
        ├──► [Qdrant/pgvector] RAG search
        ├──► [vLLM Llama-3] response
        └──► Send reply to channel
```

### 15.3. Lead Update Flow (CRM)

```
[CRM Web UI] → [api-gateway] → [crm-core]
                                     │
                                     ├──► Tx begin
                                     ├──► SET LOCAL app.current_tenant_id
                                     ├──► SET LOCAL app.current_user_id
                                     ├──► UPDATE leads SET status='qualified' WHERE id=$1
                                     ├──► UPDATE deals SET stage='negotiation' WHERE lead_id=$1
                                     ├──► INSERT activities (type='status_change')
                                     ├──► Tx commit
                                     │
                                     └──► [NATS] publish "lead.updated" + "deal.updated"
                                           │
                                           ├──► [analytics-service] → ClickHouse
                                           ├──► [meta-capi-service] → FB CAPI (purchase event)
                                           ├──► [ai-scoring] → recompute score
                                           └──► [notification-service] → notify owner
```

---

## 16. Implementation Roadmap

### 16.1. Phase 1 (Tuần 1–2): PostgreSQL Setup + Schema

**Mục tiêu:** Cụm PostgreSQL 17 production-ready với RLS, backups, monitoring.

**Tasks:**
- [ ] Setup PostgreSQL 17 cluster 1 primary + 2 replicas
- [ ] Cấu hình postgresql.conf + tuning SSD
- [ ] Install extensions: pgvector, vectorscale, pg_partman, pg_cron
- [ ] Tạo schema tenants, users, leads, deals, activities
- [ ] RLS policies cho tất cả tables
- [ ] PgBouncer + PgCat setup
- [ ] Backup script pg_basebackup + WAL archive
- [ ] Monitoring (pg_exporter, pg_stat_statements)
- [ ] K3s manifest cho PostgreSQL + PgBouncer

**Acceptance Gate:**
- `pgbench -c 50 -j 4 -T 60` đạt > 10K TPS
- RLS test pass (cross-tenant không lộ data)
- PITR restore thành công về 5 phút trước
- Failover test: kill primary, replica promote trong < 30s

### 16.2. Phase 2 (Tuần 3–4): ScyllaDB Cluster

**Mục tiêu:** ScyllaDB 6-node cluster với replication 3.

**Tasks:**
- [ ] Setup ScyllaDB cluster 3 DC: 3 node × 2 DC = 6 node
- [ ] Cấu hình GossipingPropertyFileSnitch
- [ ] Tạo keyspaces rinco_chat, rinco_events, rinco_audit, rinco_leads
- [ ] Tables messages, channels, audit_log, capi_events, leads_raw
- [ ] TWCS compaction cho chat logs
- [ ] gocql Go driver integration
- [ ] Snapshot + restore procedure
- [ ] Monitoring (scylla_exporter)

**Acceptance Gate:**
- `cassandra-stress write -rate threads=200` đạt > 100K TPS
- Latency p99 < 10ms
- Node kill test: cluster vẫn serve được

### 16.3. Phase 3 (Tuần 5–6): ClickHouse Cluster

**Mục tiêu:** ClickHouse 4-node cluster với ZooKeeper.

**Tasks:**
- [ ] Setup ClickHouse 2 shard × 2 replica = 4 node
- [ ] ZooKeeper ensemble 3 node
- [ ] Database rinco_analytics + tables
- [ ] Materialized views (events_daily_mv, lead_performance_mv)
- [ ] Per-tenant row policy + user
- [ ] clickhouse-go driver
- [ ] clickhouse-backup integration
- [ ] Grafana datasource

**Acceptance Gate:**
- Insert 1B rows, query p99 < 1s
- Distributed query trên 2 shard hoạt động
- Backup + restore cycle < 1 giờ

### 16.4. Phase 4 (Tuần 7–8): Valkey, MinIO, MongoDB

**Mục tiêu:** Cache, Object Storage, Document DB production-ready.

**Tasks:**
- [ ] Valkey cluster 6 node (3 master + 3 replica)
- [ ] Key patterns theo §6.2
- [ ] Streams cho lightweight event log
- [ ] MinIO cluster 4 node với EC:4
- [ ] Lifecycle policies + versioning
- [ ] MongoDB replica set 3 node
- [ ] Sharding setup cho form_submissions
- [ ] JSON Schema validators
- [ ] Backup scripts

**Acceptance Gate:**
- Valkey 1M ops/s benchmark
- MinIO drive failure test pass
- MongoDB failover test < 30s

### 16.5. Phase 5 (Tuần 9–10): Qdrant, Meilisearch

**Mục tiêu:** Vector + full-text search.

**Tasks:**
- [ ] pgvector setup với HNSW index cho < 10M vectors
- [ ] Qdrant cluster 3 node cho > 10M vectors
- [ ] Tenant-scoped collection strategy
- [ ] Meilisearch cluster 2 node
- [ ] Indexes: leads, contacts, knowledge
- [ ] Reindex script cho model upgrade
- [ ] Backup strategy (Qdrant snapshots, Meili snapshots)

**Acceptance Gate:**
- Vector search p99 < 8ms (Qdrant), < 15ms (pgvector)
- Meili search p99 < 50ms trên 1M docs

### 16.6. Phase 6 (Tuần 11–12): Migration Tools, Backup Automation

**Mục tiêu:** Production deployment pipeline + DR automation.

**Tasks:**
- [ ] Goose migration tool integrated trong CI/CD
- [ ] Zero-downtime migration pattern (expand/contract)
- [ ] Cross-DB consistency check
- [ ] Backup automation (cron + monitoring)
- [ ] DR drill script (chaos test)
- [ ] Documentation + runbooks
- [ ] K8s Helm chart cho tất cả DB

**Acceptance Gate:**
- Migration forward + backward không downtime
- DR drill phục hồi 100% data
- Backup test pass tất cả DB

---

## 17. Testing Strategy

### 17.1. pgbench (PostgreSQL)

```bash
# Initialize test database
pgbench -i -s 100 rinco_test

# Run read-write test
pgbench -c 50 -j 4 -T 60 -M simple \
  -S -N -r rinco_test

# Custom script
cat > /tmp/lead_query.sql <<EOF
\set tenant_id random(1, 1000)
SELECT count(*) FROM leads
  WHERE tenant_id = :tenant_id
    AND status = 'new';
EOF

pgbench -c 50 -j 4 -T 60 -f /tmp/lead_query.sql rinco_test
```

**Targets:**
- Read p99 < 50ms
- Write p99 < 80ms
- TPS > 10K với 50 client

### 17.2. cassandra-stress (ScyllaDB)

```bash
cassandra-stress write \
  -schema "replication(factor=3)" \
  -mode native cql3 \
  -rate threads=200 \
  -col "n=100" \
  -pop "dist=uniform(1..1000000)"

cassandra-stress mixed \
  -schema "replication(factor=3)" \
  -ratio "write=1,read=3" \
  -rate threads=200
```

**Targets:**
- Write p99 < 10ms
- Mixed p99 < 15ms

### 17.3. clickhouse-benchmark (ClickHouse)

```bash
clickhouse-benchmark \
  --query "SELECT count() FROM rinco_analytics.events_raw WHERE tenant_id = 'apex' AND event_date >= today() - 7" \
  --iterations 1000 \
  --concurrency 10
```

**Targets:**
- OLAP query p99 < 1s trên 1B row

### 17.4. redis-benchmark (Valkey)

```bash
redis-benchmark -h valkey -p 6379 -n 1000000 -c 100 -P 10 -t set,get,lpush,lpop
```

**Targets:**
- SET p99 < 1ms
- GET p99 < 0.5ms

### 17.5. Failover Tests

```bash
# PostgreSQL: kill primary, promote replica
ssh postgres-primary "pg_ctl stop -m immediate"
ssh postgres-replica "pg_ctl promote"
# Verify app reconnects trong < 30s

# ScyllaDB: kill node
nodetool status  # verify remaining nodes
# Run cassandra-stress, verify no error

# ClickHouse: stop 1 replica
systemctl stop clickhouse-server
# Verify queries vẫn serve từ replica khác

# Valkey: kill master
valkey-cli -h valkey-node1 DEBUG SLEEP 60  # simulate hang
# Verify Sentinel promotes new master
```

### 17.6. RLS Penetration Test

```sql
-- Setup 2 tenants
SET app.current_tenant_id = 'tenant-a';
SET app.current_user_id = 'user-a';
INSERT INTO leads (tenant_id, name, email) VALUES ('tenant-a', 'Lead A', 'a@example.com');

SET app.current_tenant_id = 'tenant-b';
SET app.current_user_id = 'user-b';
INSERT INTO leads (tenant_id, name, email) VALUES ('tenant-b', 'Lead B', 'b@example.com');

-- Cross-tenant query should return 0
SET app.current_tenant_id = 'tenant-a';
SELECT * FROM leads WHERE email = 'b@example.com';  -- EXPECT: 0 rows
```

### 17.7. Backup Restore Test

```bash
# Quarterly DR drill
1. Tạo dummy tenant + 10K leads
2. Snapshot Postgres + ScyllaDB + ClickHouse
3. Kill primary Postgres
4. Promote replica
5. Verify data integrity (checksum)
6. Pass/Fail report
```

---

## 18. Migration Plan

### 18.1. Tenant Data Migration

Khi onboard khách hàng từ CRM khác:

```bash
# 1. Export CSV từ CRM cũ
psql old_crm -c "COPY leads TO '/tmp/leads.csv' CSV HEADER"

# 2. Transform
python3 scripts/transform_leads.py \
  --input /tmp/leads.csv \
  --output /tmp/leads_rinco.csv \
  --tenant-id apex

# 3. Import
psql rinco -c "
  COPY leads_tmp (name, phone, email, ...)
  FROM '/tmp/leads_rinco.csv' CSV HEADER;

  INSERT INTO leads
    (tenant_id, name, phone, email, data, created_at)
  SELECT
    'apex', name, phone, email,
    jsonb_build_object('migrated_from', 'old_crm', 'old_id', old_id),
    created_at
  FROM leads_tmp;

  DROP TABLE leads_tmp;
"

# 4. Re-index
psql rinco -c "REINDEX INDEX CONCURRENTLY idx_leads_phone;"
```

### 18.2. Schema Version Upgrade

**Pattern: Expand-Contract Migration**

```sql
-- Step 1 (Expand): Add column nullable + backfill
ALTER TABLE leads ADD COLUMN ai_score DECIMAL(5,2);

-- Backfill in batches (chạy async)
DO $$
DECLARE last_id UUID;
BEGIN
  LOOP
    UPDATE leads SET ai_score = 0
    WHERE id > COALESCE(last_id, '00000000-0000-0000-0000-000000000000'::UUID)
      AND ai_score IS NULL
    LIMIT 5000
    RETURNING id INTO last_id;
    EXIT WHEN last_id IS NULL;
    COMMIT;
  END LOOP;
END $$;

-- Step 2: Deploy code reads ai_score

-- Step 3 (Contract): Drop default
ALTER TABLE leads ALTER COLUMN ai_score SET NOT NULL;
ALTER TABLE leads ALTER COLUMN ai_score SET DEFAULT 0;
```

### 18.3. Cross-DB Migration

Khi cần chuyển data giữa các DB:

```python
# scripts/migrate_leads_scylla_to_postgres.py
import asyncio
from cassandra.cluster import Cluster
import asyncpg

async def migrate():
    # 1. Read từ ScyllaDB
    cluster = Cluster(['10.0.1.10', '10.0.1.11'])
    session = cluster.connect('rinco_leads')

    rows = session.execute("SELECT * FROM leads_raw WHERE tenant_id = 'apex'")

    # 2. Write vào PostgreSQL với batch
    conn = await asyncpg.connect(dsn='postgresql://...')
    batch = []
    for row in rows:
        batch.append((row.tenant_id, row.full_name, ...))
        if len(batch) >= 1000:
            await conn.executemany("INSERT INTO leads ...", batch)
            batch = []

    if batch:
        await conn.executemany("INSERT INTO leads ...", batch)

    # 3. Verify count
    scylla_count = session.execute("SELECT COUNT(*) FROM leads_raw WHERE tenant_id = 'apex'").one()[0]
    pg_count = await conn.fetchval("SELECT COUNT(*) FROM leads WHERE tenant_id = 'apex'")
    assert scylla_count == pg_count, "Migration count mismatch"

asyncio.run(migrate())
```

### 18.4. Migration Tracking Sheet

| Date | Service | Type | Risk | Status |
|------|---------|------|------|--------|
| 2026-10-01 | crm-core | Add ai_score column | Low | Pending |
| 2026-10-15 | crm-core | Change phone column | Medium | Pending |
| 2026-11-01 | chat-engine | Refactor partition key | High | Pending |
| 2026-11-15 | tree-org | Add LTREE composite index | Medium | Pending |
| 2026-12-01 | scylla | Add leads_raw keyspace | Low | Pending |

---

## 19. Disaster Recovery

### 19.1. PostgreSQL PITR (Point-in-Time Recovery)

```bash
# Restore to specific timestamp
pg_restore --target-time="2026-09-15 14:30:00" \
  --dbname=rinco_recovery \
  --format=custom \
  /backup/postgres/base/base.tar.gz

# Hoặc dùng pg_basebackup + WAL replay
# 1. Stop Postgres
# 2. Restore base backup
# 3. Configure recovery.conf với target time
# 4. Start Postgres in recovery mode
```

### 19.2. ScyllaDB Snapshot Restore

```bash
# 1. Stop ScyllaDB
systemctl stop scylla-server

# 2. Clear data dir
rm -rf /var/lib/scylla/data/*

# 3. Copy snapshot back
cp -r /backup/scylla/20260915_120000/node1/* /var/lib/scylla/data/

# 4. Start ScyllaDB
systemctl start scylla-server

# 5. Verify
nodetool status
nodetool repair
```

### 19.3. ClickHouse Backup Restore

```bash
# Download backup
clickhouse-backup download default_20260915_1200 \
  --config=/etc/clickhouse-backup/config.yml

# Restore
clickhouse-backup restore default_20260915_1200 \
  --config=/etc/clickhouse-backup/config.yml \
  --schema \
  --data

# Verify
clickhouse-client --query "SELECT COUNT(*) FROM rinco_analytics.events_raw"
```

### 19.4. Multi-Region DR Strategy

| Tier | Strategy | RPO | RTO |
|------|----------|-----|-----|
| **Tier 1: PostgreSQL** | Streaming replication + WAL archive to S3 | 5 min | 30 min |
| **Tier 2: ScyllaDB** | Multi-DC with `NetworkTopologyStrategy` | 1 min | 15 min |
| **Tier 3: ClickHouse** | Replicated + ZooKeeper ensemble | 1 hour | 1 hour |
| **Tier 4: Valkey** | Cluster mode + Sentinel failover | 0 | 30 sec |
| **Tier 5: MinIO** | Cross-region replication | 0 | 1 hour |

### 19.5. DR Drill Quarterly

```bash
# scripts/dr-drill.sh
#!/bin/bash
set -e

echo "=== DR Drill Q3 2026 ==="

# 1. Tạo dummy tenant với 1000 leads
psql rinco -c "INSERT INTO leads (tenant_id, name) SELECT 'dr-test', 'DR Test ' || g FROM generate_series(1, 1000) g;"

# 2. Snapshot tất cả DB
./scripts/backup-postgres.sh
./scripts/backup-scylla.sh
./scripts/backup-clickhouse.sh

# 3. Kill primary Postgres
ssh postgres-primary "pg_ctl stop -m immediate"

# 4. Promote replica
ssh postgres-replica "pg_ctl promote"

# 5. Verify
sleep 30
psql -h postgres-replica rinco -c "SELECT COUNT(*) FROM leads WHERE tenant_id = 'dr-test'"
# Should be 1000

# 6. Cleanup
psql -h postgres-replica rinco -c "DELETE FROM leads WHERE tenant_id = 'dr-test';"

echo "DR Drill completed"
```

---

## 20. Cost Estimation

### 20.1. Hardware Cost per Node (Self-Hosted Bare-Metal)

| Component | Spec | Unit Price | Monthly |
|-----------|------|------------|---------|
| PostgreSQL node | 8c/32GB/500GB NVMe | $200 | $200 |
| ScyllaDB node | 16c/64GB/1TB NVMe | $400 | $400 |
| ClickHouse node | 16c/64GB/2TB NVMe | $500 | $500 |
| Valkey node | 4c/16GB | $100 | $100 |
| MinIO node | 8c/16GB/4TB HDD | $200 | $200 |
| MongoDB node | 4c/16GB/200GB | $150 | $150 |
| Meilisearch node | 4c/8GB | $100 | $100 |
| Qdrant node | 8c/32GB | $250 | $250 |

### 20.2. Cost per Database (12-month Total)

| Database | Topology | Hardware/mo | Backup/mo | Total/mo | 12-month |
|----------|----------|-------------|-----------|----------|----------|
| **PostgreSQL** | 1P + 2R | $600 | $100 | $700 | $8,400 |
| **ScyllaDB** | 3 nodes | $1,200 | $200 | $1,400 | $16,800 |
| **ClickHouse** | 4 nodes | $2,000 | $300 | $2,300 | $27,600 |
| **Valkey** | 6 nodes | $600 | $50 | $650 | $7,800 |
| **MinIO** | 4 nodes | $800 | $200 (cross-region replication) | $1,000 | $12,000 |
| **MongoDB** | 3 nodes | $450 | $50 | $500 | $6,000 |
| **Meilisearch** | 2 nodes | $200 | $30 | $230 | $2,760 |
| **Qdrant** | 3 nodes | $750 | $50 | $800 | $9,600 |
| **Network/Bandwidth** | 10Gbps × 2 ISP | $400 | - | $400 | $4,800 |
| **Monitoring Stack** | VictoriaMetrics + Grafana + Loki | $300 | $50 | $350 | $4,200 |
| **Total** | - | - | - | **$8,330** | **$99,960** |

### 20.3. Storage Cost Breakdown

| DB | Data Volume | Storage Type | Cost/GB | Monthly |
|----|-------------|--------------|---------|---------|
| PostgreSQL | 500GB × 3 = 1.5TB | NVMe SSD | $0.20 | $300 |
| ScyllaDB | 1TB × 3 = 3TB | NVMe SSD | $0.20 | $600 |
| ClickHouse | 2TB × 4 = 8TB | NVMe SSD + HDD cold | $0.10 | $800 |
| MinIO | 4TB × 4 = 16TB | HDD | $0.04 | $640 |
| Backups | 5TB compressed | HDD + S3 Glacier | $0.01 | $50 |

### 20.4. Backup Cost (per DB)

| DB | Tool | Frequency | Storage/mo |
|----|------|-----------|------------|
| PostgreSQL | pg_basebackup + WAL | Continuous | 200GB × $0.10 = $20 |
| ScyllaDB | nodetool snapshot | Daily | 500GB × $0.10 = $50 |
| ClickHouse | clickhouse-backup | Daily | 1TB × $0.05 = $50 |
| MinIO | Cross-region replication | Real-time | $50 (bandwidth) |

### 20.5. Managed vs Self-Hosted Comparison

| Database | Self-Hosted | AWS RDS / Equivalent | Saving |
|----------|-------------|----------------------|--------|
| **PostgreSQL** | $700/mo | $2,500/mo (db.r6g.4xlarge) | 72% |
| **ScyllaDB** | $1,400/mo | $5,000/mo (Scylla Cloud 6-node) | 72% |
| **ClickHouse** | $2,300/mo | $3,500/mo (ClickHouse Cloud) | 34% |
| **Valkey** | $650/mo | $1,200/mo (ElastiCache) | 46% |
| **MongoDB** | $500/mo | $1,800/mo (Atlas M30) | 72% |
| **MinIO** | $1,000/mo | $800/mo (S3 Standard) | -25% (S3 rẻ hơn nếu ít data) |

**Recommendation:**
- Phase 1 (MVP): Self-host tất cả (tiết kiệm $4K/mo).
- Phase 2: Hybrid - Self-host ScyllaDB + PostgreSQL, Managed cho ClickHouse Cloud.
- Phase 3+: Eval lại theo workload.

### 20.6. ROI cho 10K tenants

- Tổng monthly cost: $8,330
- Per tenant: $0.83/month
- So với SaaS truyền thống (HubSpot + Salesforce + Intercom): ~$890/tenant/mo
- **Tiết kiệm: 99.9%** (per spec master doc)

---

## 21. Open Questions

### 21.1. Cần user xác nhận ngay ✋

| # | Câu hỏi | Options | Recommendation |
|---|---------|---------|----------------|
| **Q-DB-1** | **Managed (RDS, Scylla Cloud) hay self-host?** | (a) All self-host, (b) Hybrid, (c) All managed | (b) Hybrid: self-host DB chính, managed cho BI/Analytics |
| **Q-DB-2** | **Multi-region active-active hay active-passive?** | (a) Single region, (b) Active-passive, (c) Active-active | (b) cho Phase 1; (c) cho enterprise tier |
| **Q-DB-3** | **Compression strategy cho Postgres TOAST?** | (a) pglz (default), (b) LZ4, (c) ZSTD | (c) ZSTD tốt nhất 2026 |
| **Q-DB-4** | **Encryption at rest (LUKS vs TDE vs cloud-native)?** | (a) LUKS disk-level, (b) Postgres TDE, (c) Cloud KMS | (c) Cloud KMS (AWS RDS, GCP Cloud SQL) |
| **Q-DB-5** | **Postgres extension: pgvectorscale dùng cho production?** | (a) Yes ngay, (b) Phase 2, (c) Qdrant thay thế | (a) Yes, sẵn sàng scale |
| **Q-DB-6** | **ClickHouse: shared cluster hay per-tenant cluster?** | (a) Shared + row policy, (b) Per-tenant shard, (c) Hybrid | (a) Shared với row policy |
| **Q-DB-7** | **ScyllaDB: NetworkTopologyStrategy cần bao nhiêu DC?** | (a) 1 DC × 3 rack, (b) 2 DC × 3 rack | (a) cho Phase 1 |
| **Q-DB-8** | **Valkey persistence: AOF + RDB hay chỉ AOF?** | (a) Both, (b) Only AOF, (c) Only RDB | (a) Both |
| **Q-DB-9** | **MinIO EC ratio (EC:2 vs EC:4)?** | (a) EC:2 (cheap), (b) EC:4 (default), (c) EC:6 | (b) EC:4 balance |
| **Q-DB-10** | **MongoDB Atlas hay self-host?** | (a) Atlas, (b) Self-host, (c) DocumentDB (AWS) | (b) Self-host vì ít data |

### 21.2. Cần quyết định trong Phase tiếp theo 📋

| # | Câu hỏi | Impact | Owner |
|---|---------|--------|-------|
| Q-DB-11 | Tenant isolation: shared DB + RLS vs DB-per-tenant cho enterprise? | Cost vs isolation | Architecture |
| Q-DB-12 | Có cần Citus cho horizontal sharding? | Cost, complexity | Architecture |
| Q-DB-13 | ClickHouse: dùng Cloud hay self-host? | Cost, ops | Finance |
| Q-DB-14 | Backup retention: 30 ngày hot + 1 năm archive hay 7 năm? | Storage cost, compliance | Legal |
| Q-DB-15 | Có cần tách DB theo region (data residency)? | Compliance | Legal |
| Q-DB-16 | PII columns encryption (TDE) cho GDPR? | Compliance | Security |
| Q-DB-17 | ScyllaDB repair strategy (incremental vs full)? | Network overhead | DBA |
| Q-DB-18 | ClickHouse: Materialized View vs Projection? | Query performance | DBA |
| Q-DB-19 | Connection pool size per service? | Resource usage | SRE |
| Q-DB-20 | DB upgrade strategy (blue/green)? | Downtime | SRE |

### 21.3. TBD kỹ thuật ⏳

| # | Item | Status | Next step |
|---|------|--------|-----------|
| T-DB-1 | Chọn PgBouncer vs PgCat vs Odyssey | TBD | Benchmark Q4 2026 |
| T-DB-2 | ScyllaDB 6.x vs 7.x | TBD | Test 7.x alpha |
| T-DB-3 | ClickHouse Keeper vs ZooKeeper | TBD | ClickHouse 24.x dùng Keeper |
| T-DB-4 | MongoDB version (7.x vs 8.x) | TBD | MongoDB 7.x stable |
| T-DB-5 | Meilisearch vs Typesense vs Elasticsearch | TBD | Meilisearch Tiếng Việt tốt |
| T-DB-6 | pg_cron job schedules | TBD | Document all jobs |
| T-DB-7 | pg_partman partition strategy | TBD | Decide per table |
| T-DB-8 | pg_stat_statements sampling | TBD | 1/100 ratio |
| T-DB-9 | ScyllaDB monitoring (scylla_exporter vs node_exporter) | TBD | scylla_exporter |
| T-DB-10 | MinIO KMS key rotation | TBD | 90 days rotation |

### 21.4. Risk Register ⚠️

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| PostgreSQL RLS bypass qua subquery | Low | Critical | Audit + test cross-tenant |
| ScyllaDB write amplification vượt | Medium | High | TWCS + monitoring |
| ClickHouse merge storm | Medium | Medium | Tweak parts_to_throw_insert |
| Valkey OOM do hot key | Medium | High | Sharding + monitoring |
| MinIO drive fail | Low | High | EC + auto-heal |
| Cross-region latency > 200ms | Medium | Medium | Single region Phase 1 |
| Vendor lock-in (managed DB) | Medium | Medium | Self-host + open standard |
| Open-source CVE | High | Medium | Dependabot + patch weekly |

---

**Tiếp theo:** [`docs/11-ai-integration/README.md`](../11-ai-integration/README.md) – Tích hợp AI toàn hệ thống (đã được mở rộng tương ứng).