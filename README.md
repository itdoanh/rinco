# RINCO Platform

> **Multi-tenant SaaS Platform** — Polyglot microservices với Zero-Trust Security, AI Integration, và real-time collaboration.

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Rust](https://img.shields.io/badge/Rust-1.81-000000?logo=rust&logoColor=white)](https://www.rust-lang.org/)
[![Python](https://img.shields.io/badge/Python-3.12-3776AB?logo=python&logoColor=white)](https://www.python.org/)
[![Next.js](https://img.shields.io/badge/Next.js-15-000000?logo=next.js&logoColor=white)](https://nextjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.6-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Bun](https://img.shields.io/badge/Bun-1.x-F9F1E1?logo=bun&logoColor=black)](https://bun.sh/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![ScyllaDB](https://img.shields.io/badge/ScyllaDB-6.x-621773?logo=scylladb&logoColor=white)](https://www.scylladb.com/)
[![ClickHouse](https://img.shields.io/badge/ClickHouse-24.x-FFCC01?logo=clickhouse&logoColor=black)](https://clickhouse.com/)
[![MongoDB](https://img.shields.io/badge/MongoDB-7.x-47A248?logo=mongodb&logoColor=white)](https://www.mongodb.com/)
[![K3s](https://img.shields.io/badge/K3s-latest-FFC61C?logo=k3s&logoColor=black)](https://k3s.io/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 📖 Mô tả dự án

**RINCO** là một nền tảng SaaS đa-tenant (multi-tenant) full-stack, được thiết kế cho các doanh nghiệp vừa và nhỏ cần:

- 🎯 **Quản lý CRM** dạng cây (LTREE) cho phép tổ chức leads / customers / deals theo cấu trúc phân cấp tùy ý.
- 📝 **Dynamic Model Engine** — admin định nghĩa schema JSON, hệ thống sinh API + UI động.
- 🌐 **Landing Page Builder** với block renderer + Facebook Conversions API (CAPI) + tracking attribution chính xác.
- 💬 **Real-time Chat Engine** end-to-end encrypted (Signal Protocol) trên ScyllaDB + WebSocket.
- 📹 **WebRTC SFU** cho meetings (AV1/VP9 SVC, NVENC recording, bandwidth adaptation).
- 🤖 **AI Suite**: Lead Scoring (XGBoost), RAG Chatbot (Qdrant), AI SRE (incident auto-hotfix), STT (Whisper).
- 🔐 **Zero-Trust Security**: PASETO tokens, FIDO2/WebAuthn, Row-Level Security (RLS), mTLS service mesh.
- 📊 **4-Tier Observability**: Prometheus + Grafana + Loki + Jaeger + ClickHouse + OpenTelemetry.

Nền tảng phù hợp với **~10K–1M MAU**, scale theo chiều ngang nhờ microservices + polyglot persistence.

---

## 🛠️ Tech Stack

### Languages
| Layer | Language | Phiên bản |
|-------|----------|-----------|
| Backend (Core APIs) | Go | 1.23+ |
| Backend (Real-time / Media) | Rust | 1.81+ |
| Backend (AI / ML) | Python | 3.12+ |
| Frontend | TypeScript | 5.6+ |
| Runtime (JS) | Bun / Node.js | 1.x / 20+ |

### Frameworks
- **Go**: Echo v4, pgx/v5, paseto, zap, OpenTelemetry SDK
- **Rust**: axum, tokio, scylla-rust-driver, libwebrtc, NVENC bindings
- **Python**: FastAPI, SQLAlchemy, Celery, vLLM, Whisper, LangChain
- **Frontend**: Next.js 15 (App Router), React 19, Tailwind CSS, shadcn/ui

### Data Stores
| Store | Phiên bản | Mục đích |
|-------|-----------|----------|
| PostgreSQL | 16 | OLTP chính (auth, CRM, leads, tenant, dynamic model, landing) — 6 schemas |
| ScyllaDB | 6.x | Chat messages, presence, recording metadata |
| MongoDB | 7.x | Landing analytics events, landing pages content |
| ClickHouse | 24.x | Audit logs, observability aggregates, lead events |
| Valkey (Redis-compatible) | 7.x | Cache, sessions, presence, rate-limit |
| MinIO | latest | S3-compatible object storage (recordings, uploads) |
| Qdrant | latest | Vector embeddings cho RAG chatbot |
| Meilisearch | latest | Full-text search (CMS) |
| NATS | latest | Event bus giữa các service |

---

## 🏗️ Architecture Diagram

```
┌────────────────────────────────────────────────────────────────────────────┐
│                          Client Layer (Browsers / Mobile)                   │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│   │  landing     │  │ admin-portal │  │ tenant-site  │  │ meeting-ui   │   │
│   │ (Next.js 15) │  │ (Next.js 15) │  │ (Next.js 15) │  │ (Next.js 15) │   │
│   └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │
└──────────┼─────────────────┼─────────────────┼─────────────────┼──────────┘
           │                 │                 │                 │
           ▼                 ▼                 ▼                 ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                  Edge / Gateway (Traefik + cert-manager)                   │
│                  mTLS, rate-limit, JWT/PASETO validation                   │
└────────────────────────────────────────────────────────────────────────────┘
           │
           ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                       Application Services (17)                            │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐        │
│  │ auth-service │ │ tenant-svc   │ │ crm-service  │ │ dynamic-model│        │
│  │ (Go)         │ │ (Go)         │ │ (Go)         │ │ (Go)         │        │
│  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘        │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐        │
│  │ lead-service │ │ landing-svc  │ │ chat-engine  │ │ webrtc-sfu   │        │
│  │ (Go)         │ │ (Go)         │ │ (Rust)       │ │ (Rust/C++)   │        │
│  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘        │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐        │
│  │ recording-svc│ │ email-svc    │ │ notif-svc    │ │ observ-svc   │        │
│  │ (Rust)       │ │ (Go)         │ │ (Go)         │ │ (Go)         │        │
│  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘        │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐        │
│  │ lead-scoring │ │ rag-chatbot  │ │ ai-sre       │ │ stt-service  │        │
│  │ (Python)     │ │ (Python)     │ │ (Python)     │ │ (Python)     │        │
│  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘        │
└────────────────────────────────────────────────────────────────────────────┘
           │
           ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                       Polyglot Persistence (9 stores)                      │
│   PostgreSQL × 6 schemas • ScyllaDB • MongoDB • ClickHouse • Valkey        │
│   MinIO (S3) • Qdrant (vectors) • Meilisearch • NATS (event bus)           │
└────────────────────────────────────────────────────────────────────────────┘
           │
           ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                  Observability Stack (4-tier)                               │
│   Prometheus (metrics) • Grafana • Loki (logs) • Jaeger/Tempo (traces)      │
│   OTel Collector • Alertmanager • ClickHouse (long-term audit)              │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 🚀 Quick Start

### Prerequisites

| Tool | Phiên bản | Mục đích |
|------|-----------|----------|
| Docker + Compose | 24+ / v2 | Local infrastructure |
| Go | 1.23+ | Backend Go services |
| Rust | 1.81+ | Rust real-time services |
| Python | 3.12+ | AI services |
| Node.js | 20+ | Frontend tooling |
| Bun | 1.x | Frontend package manager |
| Make | latest | Task runner |
| K3s (optional) | latest | Production-like K8s |

### Local Development (Docker Compose)

```bash
# 1. Clone
git clone https://github.com/itdoanh/rinco.git
cd rinco

# 2. Copy env
cp .env.example .env

# 3. Install deps
make install

# 4. Start infrastructure (9 DBs + observability)
docker compose -f infra/docker-compose.yml up -d

# 5. Migrate databases
make migrate

# 6. Start all services
make dev

# 7. Start frontend apps (separate terminals)
cd frontend/landing && bun dev          # → http://localhost:3000
cd frontend/admin-portal && bun dev      # → http://localhost:3001
cd frontend/tenant-site && bun dev       # → http://localhost:3002
cd frontend/meeting-ui && bun dev        # → http://localhost:3003
```

### Production Deployment (K3s + Helm + ArgoCD)

```bash
# Xem chi tiết trong docs/DEPLOYMENT.md
make deploy
```

---

## 📦 Services (17 microservices)

### Go Services (9)

| Service | Port (HTTP/gRPC) | Database | Mô tả | README |
|---------|------------------|----------|-------|--------|
| **auth-service** | 8081 / 9081 | PostgreSQL + Valkey | PASETO, FIDO2, OAuth2, sessions, RBAC | [link](services/auth-service/README.md) |
| **tenant-service** | 8082 / 9082 | PostgreSQL | Multi-tenant CRUD, plans, quotas | [link](services/tenant-service/README.md) |
| **crm-service** | 8083 / 9083 | PostgreSQL (LTREE) | CRM tree (lead/customer/deal) | [link](services/crm-service/README.md) |
| **dynamic-model-service** | 8084 / 9084 | PostgreSQL (JSONB) | Runtime schema + dynamic API | [link](services/dynamic-model-service/README.md) |
| **lead-service** | 8085 / 9085 | PostgreSQL + MongoDB | Lead capture, scoring ingestion | [link](services/lead-service/README.md) |
| **landing-service** | 8086 / 9086 | MongoDB + ScyllaDB | Block-renderer landing, tiered S3, CAPI | [link](services/landing-service/README.md) |
| **email-service** | 8087 / 9087 | PostgreSQL | Multi-driver SMTP, tracking, templates | [link](services/email-service/README.md) |
| **notification-service** | 8088 / 9088 | PostgreSQL | Push/email/SMS multi-channel | [link](services/notification-service/README.md) |
| **observability-service** | 8089 / 9089 | ClickHouse | Logs/metrics/traces aggregator, alerts | [link](services/observability-service/README.md) |

### Python Services (4)

| Service | Port | Database | Mô tả | README |
|---------|------|----------|-------|--------|
| **lead-scoring** | 8090 | PostgreSQL | XGBoost lead scoring | [link](services/lead-scoring/README.md) |
| **rag-chatbot** | 8091 | Qdrant + vLLM | RAG pipeline, embeddings, retrieval | [link](services/rag-chatbot/README.md) |
| **ai-sre** | 8092 | ClickHouse + vLLM | Incident correlator + auto-hotfix | [link](services/ai-sre/README.md) |
| **stt-service** | 8093 | - | Whisper transcription + diarization | [link](services/stt-service/README.md) |

### Rust Services (3)

| Service | Port | Database | Mô tả | README |
|---------|------|----------|-------|--------|
| **chat-engine** | 8094 | ScyllaDB + Valkey | E2EE messenger (Signal Protocol), presence | [link](services/chat-engine/README.md) |
| **webrtc-sfu** | 8095 | ScyllaDB + MinIO | Selective Forwarding Unit (AV1/VP9 SVC) | [link](services/webrtc-sfu/README.md) |
| **recording-service** | 8096 | MinIO | GPU NVENC recording, tiered SSD/HDD | [link](services/recording-service/README.md) |

### Frontend Apps (4) — Next.js 15 + React 19

| App | Port | Mô tả |
|-----|------|-------|
| **landing** | 3000 | Marketing landing pages (block renderer + FB CAPI) |
| **admin-portal** | 3001 | Super-admin (Dark Admin) |
| **tenant-site** | 3002 | Multi-tenant site renderer |
| **meeting-ui** | 3003 | WebRTC meeting UI + chat |

---

## 🗄️ Databases (9)

| Database | Cổng | Schema/DBs | Mục đích |
|----------|------|------------|----------|
| PostgreSQL | 5432 | `auth`, `tenant`, `crm`, `dynamic_model`, `lead`, `landing`, `email`, `notification`, `ai_sre`, `lead_scoring`, `rag` | OLTP chính, RLS multi-tenant |
| ScyllaDB | 9042 | `chat_engine`, `recording_meta`, `webrtc_meta` | Real-time chat, metadata |
| MongoDB | 27017 | `landing_pages`, `analytics` | Document store, analytics events |
| ClickHouse | 8123 | `audit`, `observability`, `lead_events` | OLAP, audit log |
| Valkey (Redis) | 6379 | - | Cache, sessions, presence, pub/sub |
| MinIO | 9000 / 9001 | buckets: `recordings`, `uploads`, `landing-assets`, `tenant-files` | Object storage |
| Qdrant | 6333 | collections: `rinco_docs` | Vector embeddings |
| Meilisearch | 7700 | indexes: `cms`, `leads` | Full-text search |
| NATS | 4222 | subjects: `lead.*`, `crm.*`, `chat.*`, `audit.*` | Event bus |

---

## 🏗️ Infrastructure

- **Container Orchestration**: K3s (lightweight Kubernetes, ~50MB binary) trên bare-metal / VMs.
- **Ingress Controller**: Traefik (built-in với K3s) — auto-TLS via cert-manager + Let's Encrypt.
- **Service Mesh** *(roadmap)*: Linkerd hoặc Istio — mTLS, retries, traffic split.
- **External DNS**: cloudflare / route53 integration.
- **Object Storage**: MinIO cluster (distributed mode, erasure coding).
- **Secrets**: Sealed Secrets + SOPS — không commit raw `.env`.
- **Networking**: WireGuard mesh giữa các K3s nodes (off-road deployments).

### Observability Stack

| Component | Cổng | Mục đích |
|-----------|------|----------|
| Prometheus | 9090 | Metrics scraping |
| Grafana | 3000 | Dashboards (pre-loaded cho mỗi service) |
| Loki | 3100 | Log aggregation |
| Jaeger / Tempo | 16686 | Distributed tracing |
| OTel Collector | 4317 / 4318 | OTLP ingestion |
| Alertmanager | 9093 | Alert routing (Slack, PagerDuty) |

---

## 🔄 CI/CD

GitHub Actions workflows (xem `.github/workflows/`):

- `ci.yml` — Build + lint + unit test cho tất cả 17 services + 4 frontend apps.
- `test-e2e.yml` — Playwright E2E tests cho frontend + Python integration tests.
- `cd.yml` — Build & push Docker images lên GHCR + update Helm charts.
- `security.yml` — Trivy + gosec + cargo-audit + bandit weekly scans.

CD flow: `push to main → CI green → Docker build → Helm chart bump → ArgoCD sync → K3s rollout (canary 10% → 100%)`.

---

## 🤝 Contributing

Xem [CONTRIBUTING.md](CONTRIBUTING.md) để biết:

- Fork + branch workflow
- Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`)
- Code style cho từng ngôn ngữ
- PR template + review checklist
- Testing requirements

---

## 📜 License

Dự án phát hành dưới **MIT License** — xem [LICENSE](LICENSE) để biết chi tiết.

---

## 📞 Liên hệ

- **Email**: dev@rinco.app
- **GitHub**: https://github.com/itdoanh/rinco
- **Issues**: https://github.com/itdoanh/rinco/issues

---

## 🔗 Tham khảo

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — Kiến trúc tổng thể (6 tầng)
- [docs/SERVICES.md](docs/SERVICES.md) — Index chi tiết 17 services
- [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) — Hướng dẫn dev local
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) — Production deploy trên K3s
- [docs/DEV-PLAN.md](docs/DEV-PLAN.md) — Kế hoạch phát triển tổng (~38K dòng)
- [CHANGELOG.md](CHANGELOG.md) — Lịch sử thay đổi
- [SECURITY.md](SECURITY.md) — Báo cáo lỗ hổng bảo mật
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) — Quy tắc cộng đồng
