# Architecture Overview

Kiến trúc tổng thể của RINCO Platform — 6 tầng (layers) với service mesh, polyglot persistence, và zero-trust security model.

---

## 1. 6-Layer Architecture

```
┌────────────────────────────────────────────────────────────────────────────┐
│  Layer 1: CLIENT                                                            │
│  Browsers, Mobile apps, 3rd-party integrations (webhooks, OAuth clients)    │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Layer 2: EDGE / GATEWAY                                                    │
│  Traefik (ingress, mTLS, rate-limit, JWT/PASETO validation)                │
│  Cloudflare (DDoS, WAF, CDN)                                               │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Layer 3: APPLICATION (17 microservices)                                   │
│  Go services: auth, tenant, crm, dynamic-model, lead, landing,             │
│               email, notification, observability                            │
│  Python services: lead-scoring, rag-chatbot, ai-sre, stt-service           │
│  Rust services: chat-engine, webrtc-sfu, recording-service                  │
│  Frontend services: meeting-ui                                              │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Layer 4: DATA PLANE (Polyglot Persistence + Event Bus)                    │
│  PostgreSQL × 6 • ScyllaDB • MongoDB • ClickHouse • Valkey                │
│  MinIO (S3) • Qdrant (vectors) • Meilisearch • NATS (event bus)            │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Layer 5: OBSERVABILITY & OPERATIONS                                        │
│  Prometheus (metrics) • Grafana • Loki (logs) • Jaeger/Tempo (traces)      │
│  OpenTelemetry Collector • Alertmanager • Velero (backups)                  │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Layer 6: PLATFORM / INFRA                                                 │
│  K3s (Kubernetes) • Helm • ArgoCD (GitOps) • cert-manager • external-dns   │
│  WireGuard (private mesh) • Terraform / Ansible (provisioning)             │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Service Mesh

### 2.1 Inter-service communication

| Pattern | Công nghệ | Use case |
|---------|-----------|----------|
| **Synchronous HTTP** | Echo (Go) / axum (Rust) / FastAPI (Python) + Connect-RPC | Request/response CRUD |
| **Streaming** | WebSocket / gRPC streaming | Chat, presence, signaling |
| **Async events** | NATS JetStream | Pub/sub giữa services |
| **Internal RPC** | Connect-RPC (HTTP/2 + protobuf) | Type-safe service-to-service |

### 2.2 Routing

```
Client → Traefik (TLS termination)
            │
            ├─→ /api/auth/*        → auth-service
            ├─→ /api/tenants/*     → tenant-service
            ├─→ /api/crm/*         → crm-service
            ├─→ /api/models/*      → dynamic-model-service
            ├─→ /api/leads/*       → lead-service
            ├─→ /api/landing/*     → landing-service
            ├─→ /api/email/*       → email-service
            ├─→ /api/notifications/* → notification-service
            ├─→ /api/observability/* → observability-service
            ├─→ /api/scoring/*     → lead-scoring
            ├─→ /api/rag/*         → rag-chatbot
            ├─→ /api/ai-sre/*      → ai-sre
            ├─→ /api/stt/*         → stt-service
            ├─→ /ws/chat           → chat-engine
            ├─→ /ws/meeting        → webrtc-sfu
            └─→ /static/*          → MinIO (CDN-style)
```

### 2.3 Service discovery

- **Trong cluster**: K8s DNS (`<service>.<namespace>.svc.cluster.local`).
- **Cross-cluster**: NATS subject-based routing.

---

## 3. Data Flow

### 3.1 Write path (typical user action: tạo Lead)

```
1. User fills landing form (frontend/landing)
   POST /api/leads
       │
       ▼
2. Traefik → lead-service (Go)
   - Validates JWT
   - Tenant resolution (X-Tenant-Id)
   - RLS policy applies (tenant_id filter)
       │
       ├──→ INSERT INTO lead.leads (PostgreSQL)
       │
       └──→ Publish NATS event: lead.created
                │
                ▼
       3. lead-scoring (Python) consumes
          - Fetch features
          - Run XGBoost model
          - INSERT INTO lead_scoring.scores
          - Publish lead.scored
                │
                ▼
       4. notification-service consumes lead.scored
          - If score > threshold → send Slack/Email
                │
                ▼
       5. observability-service consumes all
          - Aggregates → ClickHouse (audit)
```

### 3.2 Read path (chat messages)

```
1. User connects via WebSocket /ws/chat
       │
       ▼
2. chat-engine (Rust) accepts connection
   - Validates PASETO token
   - Looks up device pubkey
       │
       ├──→ Valkey: presence.<user_id> = {device_id, online}
       │
       └──→ ScyllaDB: SELECT * FROM chat.messages
                       WHERE channel_id = ? AND tenant_id = ?
                       LIMIT 50 (paginated)
```

### 3.3 Real-time path (WebRTC meeting)

```
1. meeting-ui establishes WebSocket → webrtc-sfu
       │
       ▼
2. SDP offer/answer exchange
       │
       ▼
3. webrtc-sfu allocates media pipeline (AV1/VP9 SVC)
       │
       ├──→ Forwards SRTP between peers
       │
       └──→ If recording enabled:
              recording-service spawns NVENC encoder
              → MinIO (tiered SSD → HDD after 30 days)
```

---

## 4. Security Model (Zero-Trust)

### 4.1 Authentication

| Layer | Method | Implementation |
|-------|--------|----------------|
| **User** | PASETO v4.public + FIDO2/WebAuthn | auth-service + `packages/go/auth` |
| **Service-to-service** | mTLS (K8s cert-manager) + JWT service-account | Envoy sidecar (Linkerd roadmap) |
| **API client** | API key + HMAC signature | auth-service |
| **Webhook receiver** | HMAC SHA256 + replay protection | Per-service config |

### 4.2 Authorization (RBAC + ABAC)

```
┌──────────────────────────────────────────────────┐
│ Super-Admin (platform owner)                     │
│   └─ Tenant Owner (1 per tenant)                 │
│        ├─ Admin (full tenant access)             │
│        ├─ Manager (read + write assigned scope)  │
│        ├─ Agent (read + write own records)       │
│        └─ Viewer (read-only)                     │
└──────────────────────────────────────────────────┘
```

Mỗi API request flow:
1. JWT/PASETO validation → user_id + tenant_id
2. RBAC check → role vs resource
3. ABAC check → policy engine (custom DSL) → allow/deny
4. PostgreSQL RLS policy → row-level filter by tenant_id

### 4.3 Encryption

- **In transit**: TLS 1.3 cho tất cả public endpoints; mTLS nội bộ (K8s).
- **At rest**:
  - PostgreSQL: pgcrypto cho sensitive columns + WAL archiving encrypted.
  - MinIO: SSE-KMS với key rotate 90 ngày.
  - ScyllaDB: TDE (transparent data encryption).
  - Backups: AES-256-GCM trước khi upload S3.
- **E2EE**: chat-engine dùng Signal Protocol (X3DH + Double Ratchet) — server chỉ lưu ciphertext.

### 4.4 Network segmentation

```
┌────────────────────────────────────────────────────────┐
│ Public subnet (DMZ)                                    │
│   Traefik ingress • Cloudflare CDN                     │
└────────────────────────────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────────────────────┐
│ Application subnet (private)                           │
│   17 services • NATS                                   │
│   NetworkPolicy: chỉ ingress được từ Traefik           │
└────────────────────────────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────────────────────┐
│ Data subnet (private, không ingress từ internet)       │
│   9 databases • MinIO • Qdrant • Meilisearch           │
│   NetworkPolicy: chỉ services trong app subnet access  │
└────────────────────────────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────────────────────┐
│ Observability subnet (private)                         │
│   Prometheus • Grafana • Loki • Jaeger                 │
│   Access qua VPN hoặc SSO portal                       │
└────────────────────────────────────────────────────────┘
```

### 4.5 Threat model (tóm tắt)

| Threat | Mitigation |
|--------|-----------|
| Credential stuffing | Rate-limit + FIDO2 + captcha |
| XSS / CSRF | CSP strict + SameSite cookies + CSRF tokens |
| SQL injection | Parameterized queries (pgx) + input validation |
| Tenant data leak | RLS + tenant_id middleware + audit log |
| E2EE bypass | Signal Protocol với forward secrecy |
| Supply chain attack | Image signing (cosign) + SBOM + Trivy scans |
| Insider threat | Audit log toàn bộ admin actions (ClickHouse) |
| DDoS | Cloudflare proxy + rate-limit at Traefik |
| Ransomware | Air-gapped backups + immutable S3 bucket |

---

## 5. Polyglot Persistence — tại sao?

| Database | Chọn vì | Service sử dụng |
|----------|---------|------------------|
| **PostgreSQL** | ACID, RLS, mature, JSONB flexible | auth, tenant, crm, dynamic-model, lead, landing, email, notification, lead-scoring |
| **ScyllaDB** | Sub-ms writes ở scale, CQL = SQL-ish | chat-engine, webrtc-sfu, recording-service |
| **MongoDB** | Document schema thay đổi thường xuyên, sharding tự động | landing-service (block content), analytics |
| **ClickHouse** | OLAP nhanh, nén tốt, columnar | observability-service, audit logs |
| **Valkey** | In-memory, pub/sub, low latency | sessions, presence, rate-limit, cache |
| **MinIO** | S3-compatible, cost-effective, on-prem friendly | recording-service, landing-service, uploads |
| **Qdrant** | Vector search tốc độ cao, payload filter | rag-chatbot |
| **Meilisearch** | Full-text search typo-tolerant, lightweight | CMS / global search |
| **NATS** | Event-driven, lightweight, subject-based routing | Toàn bộ inter-service events |

**Không** dùng 1 DB cho tất cả — mỗi workload chọn tool phù hợp (polyglot principle).

---

## 6. Event-driven Architecture

### 6.1 NATS subjects (chính)

```
lead.*          → lead.created, lead.updated, lead.scored
crm.*           → crm.deal.created, crm.deal.stage_changed
chat.*          → chat.message.sent, chat.presence.changed
tenant.*        → tenant.created, tenant.plan_changed
audit.*         → audit.user_action, audit.admin_action
notification.*  → notification.send, notification.delivered
system.*        → system.health, system.deploy
```

### 6.2 Event flow ví dụ

```
[crm-service]   ── publish crm.deal.stage_changed ──▶ [NATS]
                                                          │
   ┌──────────────────────────────────────────────────────┼─────────┐
   ▼                                                      ▼         ▼
[notification-service]                            [observability] [audit-service]
   - Gửi Slack cho owner                             - Aggregate metrics  - Log to CH
```

### 6.3 Guarantees

- **At-least-once** delivery mặc định (JetStream acks).
- **Idempotency**: consumer xử lý duplicate qua `event_id` dedup table.
- **Ordering**: per-subject + per-tenant ordering.

---

## 7. Observability Architecture

### 7.1 4 Pillars + Audit

| Pillar | Tool | Use |
|--------|------|-----|
| **Metrics** | Prometheus | Business + system metrics |
| **Logs** | Loki (structured JSON) | All app logs |
| **Traces** | Jaeger / Tempo | Distributed tracing (OpenTelemetry) |
| **Audit** | ClickHouse | Long-term immutable audit trail |

### 7.2 Trace propagation

```
HTTP request
    │
    ▼
Traefik (injects traceparent header)
    │
    ▼
auth-service (root span: "auth.login")
    │
    ├─▶ lead-service (child span: "lead.create")
    │       │
    │       └─▶ lead-scoring (child span: "model.predict")
    │
    └─▶ observability-service (sink span)
```

Mọi span có tags: `tenant_id`, `user_id`, `service.name`, `http.status_code`.

### 7.3 SLI / SLO

| Service | SLI | SLO |
|---------|-----|-----|
| auth-service | Login success rate | ≥ 99.9% |
| chat-engine | Message delivery p99 latency | < 200ms |
| webrtc-sfu | Concurrent meetings supported | > 100 per pod |
| landing | Time to first byte p95 | < 300ms |
| lead-scoring | Scoring latency p95 | < 2s |

---

## 8. Deployment Topology

```
                                Internet
                                   │
                                   ▼
                          ┌────────────────┐
                          │   Cloudflare   │
                          │  (DDoS/WAF/CDN)│
                          └────────┬───────┘
                                   │
                                   ▼
                    ┌──────────────────────────┐
                    │  K3s cluster (3 nodes)   │
                    │  ┌────────────────────┐  │
                    │  │ Traefik ingress    │  │
                    │  └─────────┬──────────┘  │
                    │            │             │
                    │  ┌─────────▼──────────┐  │
                    │  │ 17 app services    │  │
                    │  │ (Helm chart)       │  │
                    │  └─────────┬──────────┘  │
                    │            │             │
                    │  ┌─────────▼──────────┐  │
                    │  │ 9 databases        │  │
                    │  │ (Operators/Helm)   │  │
                    │  └────────────────────┘  │
                    │                          │
                    │  + observability stack   │
                    └──────────────────────────┘
                                   │
                          ┌────────┴─────────┐
                          ▼                  ▼
                   S3 backups         WireGuard mesh
                                       (cross-region)
```

---

## 9. Design Principles

1. **Polyglot but pragmatic** — chọn tool phù hợp workload, không over-engineer.
2. **Zero-Trust by default** — mTLS + JWT + RLS + audit log cho mọi request.
3. **Event-driven first** — services giao tiếp qua events khi có thể.
4. **Observability built-in** — không ship service nào không có metrics/logs/traces.
5. **API-first** — mọi capability exposed qua HTTP/gRPC API.
6. **Schema evolution** — migrations additive, backward-compatible.
7. **Cost-aware** — tiered storage (SSD → HDD), auto-scaling, spot instances cho batch.
8. **Open standards** — OpenTelemetry, Prometheus, S3, Kubernetes — không vendor lock-in.

---

## 10. Roadmap

| Quarter | Initiative |
|---------|------------|
| Q4 2026 | Service mesh (Linkerd), policy engine (OPA) |
| Q1 2027 | Multi-region active-active, edge compute (Cloudflare Workers) |
| Q2 2027 | AI cost optimization (model distillation, caching layer) |
| Q3 2027 | Zero-downtime migration tooling cho tenants |

---

## Xem thêm

- [docs/SERVICES.md](SERVICES.md) — Service catalog
- [docs/DEPLOYMENT.md](DEPLOYMENT.md) — Production deployment
- [docs/DEVELOPMENT.md](DEVELOPMENT.md) — Dev workflow
- [SECURITY.md](../SECURITY.md) — Security policy
- [docs/DEV-PLAN.md](DEV-PLAN.md) — Kế hoạch phát triển chi tiết
