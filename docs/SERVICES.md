# Services Index

Tổng hợp tất cả 17 microservices + 4 frontend apps trong RINCO Platform.

> **Cách dùng**: Mỗi service có `README.md` riêng với hướng dẫn API + run + test chi tiết. Click vào link để xem.

---

## Go (9 services)

| # | Service | HTTP Port | gRPC Port | Database | README |
|---|---------|-----------|-----------|----------|--------|
| 1 | [auth-service](services/auth-service/README.md) | 8081 | 9081 | PostgreSQL + Valkey | Xác thực: PASETO, FIDO2/WebAuthn, OAuth2, sessions, RBAC |
| 2 | [tenant-service](services/tenant-service/README.md) | 8082 | 9082 | PostgreSQL | Quản lý tenant, plans, quotas, isolated namespaces |
| 3 | [crm-service](services/crm-service/README.md) | 8083 | 9083 | PostgreSQL (LTREE) | CRM tree — lead/customer/deal phân cấp |
| 4 | [dynamic-model-service](services/dynamic-model-service/README.md) | 8084 | 9084 | PostgreSQL (JSONB) | Runtime schema + sinh API/UI động |
| 5 | [lead-service](services/lead-service/README.md) | 8085 | 9085 | PostgreSQL + MongoDB | Lead capture + ingestion cho scoring |
| 6 | [landing-service](services/landing-service/README.md) | 8086 | 9086 | MongoDB + ScyllaDB | Block renderer + tiered S3 + FB CAPI |
| 7 | [email-service](services/email-service/README.md) | 8087 | 9087 | PostgreSQL | Multi-driver SMTP, templates, tracking |
| 8 | [notification-service](services/notification-service/README.md) | 8088 | 9088 | PostgreSQL | Push / Email / SMS multi-channel + preferences |
| 9 | [observability-service](services/observability-service/README.md) | 8089 | 9089 | ClickHouse | Logs/metrics/traces aggregator + alerts + audit |

**Tech**: Go 1.23, Echo v4, pgx/v5, zap, OpenTelemetry, Prometheus client.

---

## Python (4 services)

| # | Service | HTTP Port | Database | README |
|---|---------|-----------|----------|--------|
| 1 | [lead-scoring](services/lead-scoring/README.md) | 8090 | PostgreSQL | XGBoost lead scoring với feature engineering + drift detection |
| 2 | [rag-chatbot](services/rag-chatbot/README.md) | 8091 | Qdrant + vLLM | RAG pipeline: ingest → embed → retrieve → generate |
| 3 | [ai-sre](services/ai-sre/README.md) | 8092 | ClickHouse + vLLM | Incident correlator + runbook + auto-hotfix generator |
| 4 | [stt-service](services/stt-service/README.md) | 8093 | - | Whisper transcription + diarization + alignment |

**Tech**: Python 3.12, FastAPI, SQLAlchemy, Celery, vLLM, Whisper, LangChain, scikit-learn, XGBoost.

---

## Rust (3 services)

| # | Service | HTTP Port | Database | README |
|---|---------|-----------|----------|--------|
| 1 | [chat-engine](services/chat-engine/README.md) | 8094 | ScyllaDB + Valkey | E2EE messenger (Signal Protocol), presence, devices |
| 2 | [webrtc-sfu](services/webrtc-sfu/README.md) | 8095 | ScyllaDB + MinIO | Selective Forwarding Unit (AV1/VP9 SVC), bandwidth adaptation |
| 3 | [recording-service](services/recording-service/README.md) | 8096 | MinIO | GPU NVENC recording, tiered SSD/HDD storage |

**Tech**: Rust 1.81, axum, tokio, scylla-rust-driver, libwebrtc, NVENC bindings, Signal Protocol.

---

## Frontend (4 apps)

| # | App | Port | README |
|---|-----|------|--------|
| 1 | [landing](frontend/landing/README.md) | 3000 | Marketing landing page (block renderer + FB CAPI + tracking) |
| 2 | [admin-portal](frontend/admin-portal/README.md) | 3001 | Super-admin portal (Dark Admin) |
| 3 | [tenant-site](frontend/tenant-site/README.md) | 3002 | Multi-tenant site renderer |
| 4 | [meeting-ui](frontend/meeting-ui/README.md) | 3003 | WebRTC meeting UI + chat |

**Tech**: Next.js 15 (App Router), React 19, TypeScript 5.6, Bun, Tailwind CSS, shadcn/ui.

---

## Service Dependency Graph (high-level)

```
                              ┌──────────────────┐
                              │   Frontend Apps  │
                              │ (landing/admin/  │
                              │  tenant/meeting) │
                              └────────┬─────────┘
                                       │ HTTPS / WSS
                                       ▼
                            ┌──────────────────────┐
                            │ Traefik Ingress +    │
                            │ cert-manager (TLS)   │
                            └──────────┬───────────┘
                                       │
       ┌───────────────────┬───────────┼───────────┬──────────────────┐
       ▼                   ▼           ▼           ▼                  ▼
┌─────────────┐    ┌─────────────┐  ┌──────┐ ┌─────────────┐  ┌─────────────┐
│ auth-service│◄───┤ crm-service │  │ chat │ │ landing-svc │  │ webrtc-sfu  │
└──────┬──────┘    └──────┬──────┘  │engine│ └──────┬──────┘  └──────┬──────┘
       │                  │         └──┬───┘        │             │
       │                  │            │            │             │
       ▼                  ▼            ▼            ▼             ▼
   PostgreSQL        PostgreSQL    ScyllaDB     MongoDB        ScyllaDB
   + Valkey          (LTREE)      + Valkey     + ScyllaDB     + MinIO
                                                   │
                                                   ▼
                                            Tiered S3 (MinIO)

       All services → NATS event bus → observability-service → ClickHouse
```

---

## Tổng quan

- **Tổng services**: 16 backend + 1 frontend-with-backend (meeting-ui) = **17**
- **Frontend apps riêng biệt**: 4
- **Ngôn ngữ backend**: Go, Python, Rust
- **Polyglot persistence**: 9 databases (PostgreSQL × 6 schemas, ScyllaDB, MongoDB, ClickHouse, Valkey, MinIO, Qdrant, Meilisearch, NATS)
- **Shared packages**: `packages/go/` (Go libs), `packages/frontend/ui/` (React component library)

---

## Xem thêm

- [docs/ARCHITECTURE.md](ARCHITECTURE.md) — Kiến trúc tổng thể
- [docs/DEVELOPMENT.md](DEVELOPMENT.md) — Hướng dẫn dev local
- [docs/DEPLOYMENT.md](DEPLOYMENT.md) — Production deploy trên K3s
