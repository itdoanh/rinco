# ADR-0003: NATS JetStream as Event Bus

> **Status**: Accepted · **Date**: 2026-09-18 · **Authors**: RINCO Platform Team

## Context

RINCO có 20 microservices giao tiếp với nhau. Cần một **event bus** để:

1. **Decouple services** — producer không cần biết consumer.
2. **Async processing** — không block request (vd: AI scoring, CAPI forward).
3. **Fan-out** — 1 event → nhiều consumer (vd: `lead.created` → notification + analytics + lead-scoring).
4. **Replay** — replay events để recover state hoặc debug.
5. **Subject-based routing** — pattern matching (vd: `tenant.*.lead.created`).
6. **At-least-once delivery** — không mất event khi consumer down.
7. **Low latency** — p99 < 50ms cho publish-consume.

## Decision

Dùng **NATS JetStream** làm event bus.

### So sánh với alternatives

| Aspect | NATS JetStream | Kafka | RabbitMQ |
|--------|----------------|-------|----------|
| **Latency** | 🟢 sub-ms | 🟡 5-50ms | 🟡 5-30ms |
| **Throughput** | 🟢 10M+ msg/s | 🟢 1M+ msg/s | 🟡 100k msg/s |
| **Operational complexity** | 🟢 Đơn giản (single binary) | 🟡 Phức tạp (ZK/KRaft, partitions) | 🟡 Trung bình |
| **Subject routing** | 🟢 Native (wildcards) | 🟡 Phải qua topic naming | 🟡 Topic exchange |
| **Replay** | 🟢 Last 24h default | 🟢 Retention configurable | ❌ Không native |
| **Cluster size** | 🟢 3-5 nodes | 🟡 5+ nodes cho HA | 🟡 3 nodes cho HA |
| **Memory footprint** | 🟢 ~100MB | 🔴 ~2GB+ | 🟡 ~500MB |
| **Multi-tenancy** | 🟡 Account namespaces | 🟡 Topic prefix + ACL | 🟡 Vhost |
| **License** | 🟢 Apache 2.0 | 🟡 Confluent Community (modified) | 🟢 MPL 2.0 |

### Subject naming convention

```
<domain>.<entity>.<action>.<version>

Ví dụ:
  lead.created.v1
  lead.scored.v1
  crm.deal.stage_changed.v1
  crm.deal.won.v1
  chat.message.sent.v1
  chat.presence.changed.v1
  tenant.created.v1
  tenant.plan_changed.v1
  audit.user_action.v1
  audit.admin_action.v1
  notification.send.v1
  notification.delivered.v1
  system.health.v1
  system.deploy.v1
  billing.invoice.created.v1
  meta_capi.event.forwarded.v1
  analytics.cube.materialized.v1
  search.index.updated.v1
```

Wildcard subscriptions:

```
lead.*                 — Tất cả lead events
*.deal.won.v1          — Tất cả deal.won từ mọi domain
tenant.*.user.*        — Mọi user event trong tenant context
>                      — Tất cả (chỉ dùng cho debugging/audit)
```

### Stream design

```yaml
streams:
  - name: RINCO_EVENTS
    subjects: ["*.>"]
    retention: limits
    max_age: 168h       # 7 ngày
    max_bytes: 50GB
    storage: file
    replicas: 3

  - name: AUDIT_ARCHIVE
    subjects: ["audit.>"]
    retention: limits
    max_age: 8760h      # 1 năm
    storage: file
    replicas: 3
    # Mirror sang S3 sau 24h qua "nats-stream-backup"
```

### Delivery guarantees

- **At-least-once** mặc định.
- **Consumer ACK** sau khi xử lý thành công (DB insert + outbox marker).
- **Idempotency** qua `event_id` dedup table trong mỗi consumer.
- **Ordering**: per-subject + per-tenant (NATS partition by tenant_id).

### Cross-region / DR

- **Mirror** stream từ primary → secondary cluster.
- **Failover** manual (chưa auto, vì cross-region write latency cao).

### Client SDK

- **Go**: `github.com/nats-io/nats.go` (qua `packages/go/middleware`).
- **Rust**: `async-nats` crate.
- **Python**: `nats-py` (asyncio).

### Outbox pattern cho reliability

Đảm bảo mọi DB write đi kèm với event publish:

```
BEGIN TRANSACTION
  INSERT INTO lead.leads (...)
  INSERT INTO outbox.events (subject, payload, created_at)
COMMIT

-- Background worker
SELECT * FROM outbox.events WHERE published_at IS NULL
  → NATS publish → UPDATE published_at
```

Đảm bảo: nếu DB commit nhưng NATS publish fail, retry sau khi restart.

### Dead-letter queue

Subject `dlq.<original-subject>` cho events xử lý > 3 lần fail:

- Stream `DLQ` retention 30 ngày.
- Admin có thể replay hoặc discard.

## Consequences

### Positive

- **Đơn giản** — single binary, không cần ZooKeeper/KRaft.
- **Low latency** — sub-ms cho in-cluster publish.
- **Subject routing** — code consumer rất clean (`subscribe("lead.*.v1")`).
- **Replay** — debug hoặc rebuild state khi cần.
- **Multi-tenancy** qua Account namespaces.

### Negative

- **Single point of failure** nếu NATS cluster down → mọi event flow dừng (mitigate bằng 3-node cluster + replication).
- **Không có schema registry native** → schema phải enforce qua consumer code (mitigate bằng versioning `v1`, `v2`).
- **File storage** cần persistent volume đủ lớn.
- **Không có exactly-once** native (chỉ at-least-once + idempotency).

### Mitigations

- **3-node cluster** với quorum = 2.
- **Outbox pattern** cho mọi event (xem trên).
- **Schema versioning** mandatory (`.v1`, `.v2`).
- **Consumer health check** + alerting khi lag > threshold.

## Alternatives Considered

### A. Apache Kafka

- **Pro**: throughput cao, mature, ecosystem rộng.
- **Con**: phức tạp (partition rebalance, KRaft migration), tốn RAM, license.
- **Verdict**: ❌ Overkill cho 20 services / 50k msg/s peak.

### B. RabbitMQ

- **Pro**: mature, đơn giản hơn Kafka.
- **Con**: không có replay, throughput thấp hơn NATS.
- **Verdict**: ❌ Không đáp ứng yêu cầu replay.

### C. Redis Streams

- **Pro**: đơn giản, đã có Valkey.
- **Con**: persistence yếu, không có subject routing.
- **Verdict**: ❌ Không đủ cho production event bus.

### D. AWS SNS/SQS / GCP Pub/Sub

- **Pro**: managed, scale tự động.
- **Con**: vendor lock-in, latency cao hơn, cost cho high-volume.
- **Verdict**: ❌ Multi-cloud strategy cần self-hosted.

## References

- [ARCHITECTURE.md §6](../ARCHITECTURE.md#6-event-driven-architecture)
- [NATS documentation](https://docs.nats.io/)
- [services/lead-service](../../services/lead-service) — reference consumer
- [ADR-0001 Polyglot Persistence](0001-polyglot-persistence.md)
- [ADR-0007 Chat Engine Rust](0007-chat-engine-rust.md)