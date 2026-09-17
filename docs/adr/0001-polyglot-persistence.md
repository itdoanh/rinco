# ADR-0001: Polyglot Persistence

> **Status**: Accepted · **Date**: 2026-09-18 · **Authors**: RINCO Platform Team

## Context

RINCO platform cần lưu trữ nhiều loại dữ liệu với đặc tính rất khác nhau:

- **Transactional data** (auth, tenant, CRM, billing, lead, dynamic-model): yêu cầu ACID, RLS, query phức tạp, JSONB flexible.
- **Real-time chat / signaling** (chat-engine, webrtc-sfu): write throughput cực cao (100k msg/s), latency sub-millisecond, schema không quan trọng.
- **Document / dynamic schema** (landing blocks, dynamic forms): schema thay đổi runtime.
- **OLAP / audit** (observability, analytics): query aggregate nhanh trên hàng tỷ rows.
- **Cache / session / rate-limit**: in-memory, low-latency.
- **Object storage** (recordings, uploads, invoice PDF): files lớn (GB), tiered storage.
- **Vector search** (RAG): nearest-neighbor trên embeddings 768–1536 dim.
- **Full-text search** (CRM, KB, messages): typo-tolerant, faceted.
- **Event bus**: pub/sub giữa services, không cần persistence lâu dài.

Một database duy nhất **không thể** đáp ứng tất cả — PostgreSQL tốt cho transactional nhưng không scale cho chat write; ScyllaDB tốt cho real-time nhưng không có RLS cho multi-tenant; ClickHouse tốt cho OLAP nhưng không phù hợp OLTP.

## Decision

Áp dụng **Polyglot Persistence**: mỗi workload chọn database phù hợp nhất.

| Database | Workload | Service sử dụng |
|----------|----------|------------------|
| **PostgreSQL 16** | Transactional + RLS + JSONB | auth, tenant, crm, dynamic-model, lead, landing, email, notification, lead-scoring, meta-capi, billing, search |
| **ScyllaDB 6.x** | Real-time writes | chat-engine, webrtc-sfu, recording-service (metadata) |
| **MongoDB 7.x** | Dynamic schema / documents | landing-service (blocks), analytics (cubes cache), lead (raw events) |
| **ClickHouse 24.x** | OLAP / audit | observability, ai-sre, analytics |
| **Valkey 7.x** | Cache / session / rate-limit / pub-sub | auth, chat (presence), all services |
| **MinIO** | Object storage + tiered | recording, landing, billing (PDF), all uploads |
| **Qdrant** | Vector search | rag-chatbot |
| **Meilisearch** | Full-text search | search-service |
| **NATS JetStream** | Event bus | all inter-service events |

### Nguyên tắc chọn database

1. **OLTP + strong consistency + RLS** → PostgreSQL.
2. **High write throughput (>10k req/s) + low latency** → ScyllaDB.
3. **Schema-less / document thay đổi liên tục** → MongoDB.
4. **OLAP / scan hàng tỷ rows / columnar compression** → ClickHouse.
5. **In-memory + TTL + ephemeral** → Valkey.
6. **Files > 10MB / immutable / lifecycle policy** → MinIO (S3).
7. **Vector similarity search** → Qdrant.
8. **Full-text + typo-tolerant + faceted** → Meilisearch.
9. **Pub/sub + at-least-once + replay** → NATS JetStream.

### Operational implications

- **9 operators/statefulsets** cần manage (PostgreSQL, Scylla, Mongo, ClickHouse, Valkey, MinIO, Qdrant, Meilisearch, NATS).
- **Backup strategy khác nhau** cho mỗi DB (xem [ADR-0006](0006-tiered-storage-minio.md)).
- **Migrations** mỗi DB có runner riêng (`packages/go/db/migrations` cho PG/CQL/Mongo/CH).
- **Observability** chuẩn hóa metrics/logs/traces cho mọi DB.
- **Connection pooling** quan trọng — mỗi service có pool riêng qua `packages/go/db`.

## Consequences

### Positive

- **Performance**: mỗi workload đạt được latency/throughput tốt nhất.
- **Cost-effective**: không over-provision một DB cho workload không phù hợp.
- **Independent scaling**: scale chat-engine mà không ảnh hưởng OLTP PG.
- **Polyglot team**: chuyên môn hoá — DBA team có thể quản lý cluster riêng.

### Negative

- **Operational complexity**: 9 DBs thay vì 1 → 9× monitoring, backup, upgrades.
- **Data consistency across DBs**: dùng event-driven (NATS) + idempotency keys + outbox pattern.
- **Developer onboarding**: cần hiểu 9 công nghệ, không phải ai cũng thành thạo.
- **License / vendor lock-in**: cần track license changes (PostgreSQL = BSD, MinIO = AGPL, …).
- **Local dev**: docker-compose có 9 services → tốn RAM; chạy selective mode.

### Mitigations

- **Internal training** cho mỗi database (doc + runbook).
- **Staging cluster** riêng để test upgrades.
- **Schema registry** để track schema changes (xem `services/<svc>/migrations/`).
- **Polyglot abstraction**: `packages/go/db` cung cấp interface thống nhất (dù thực tế vẫn khác nhau).

## Alternatives Considered

### A. PostgreSQL-only (single DB)

- **Pro**: ít moving parts, mature, RLS có sẵn.
- **Con**: không scale cho chat (write throughput), không có vector search native, OLAP chậm.
- **Verdict**: ❌ Rejected — không đáp ứng real-time + AI workloads.

### B. ScyllaDB-only

- **Pro**: write throughput tốt.
- **Con**: không có RLS, không có JSONB flexible, OLAP không tốt.
- **Verdict**: ❌ Rejected — multi-tenant CRM không an toàn.

### C. TiDB (NewSQL hybrid)

- **Pro**: HTAP, MySQL-compatible.
- **Con**: ít mature hơn PG/Scylla, không có RLS native, không có vector search.
- **Verdict**: ❌ Rejected — phù hợp general purpose, không tối ưu cho workload đặc thù.

## References

- [ARCHITECTURE.md §5](../ARCHITECTURE.md#5-polyglot-persistence--tại-sao)
- [SERVICES.md](../SERVICES.md)
- [ADR-0006 Tiered Storage MinIO](0006-tiered-storage-minio.md)
- [ADR-0003 NATS Event Bus](0003-nats-event-bus.md)