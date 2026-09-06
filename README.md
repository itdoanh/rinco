# RINCO Platform

> **Multi-tenant SaaS Platform** – Microservices architecture với Polyglot Persistence, Zero-Trust Security, AI Integration.

## 📚 Tài liệu thiết kế

Tất cả tài liệu thiết kế chi tiết được đặt trong `docs/`:

| File | Mô tả |
|------|-------|
| [`docs/00-master/README.md`](docs/00-master/README.md) | Tổng quan master design |
| [`docs/01-super-admin/README.md`](docs/01-super-admin/README.md) | Super Admin Portal + Dark Admin |
| [`docs/02-tenant-site/README.md`](docs/02-tenant-site/README.md) | Tenant Site + Mesh |
| [`docs/03-crm-tree/README.md`](docs/03-crm-tree/README.md) | CRM Tree với LTREE |
| [`docs/04-dynamic-model/README.md`](docs/04-dynamic-model/README.md) | Dynamic Model Engine |
| [`docs/05-landing-capi/README.md`](docs/05-landing-capi/README.md) | Landing Page + Facebook CAPI |
| [`docs/06-chat-engine/README.md`](docs/06-chat-engine/README.md) | Real-time Chat Engine |
| [`docs/07-webrtc-sfu/README.md`](docs/07-webrtc-sfu/README.md) | WebRTC SFU + Recording |
| [`docs/08-observability/README.md`](docs/08-observability/README.md) | 4-tier Observability |
| [`docs/09-security/README.md`](docs/09-security/README.md) | Multi-tenant Security |
| [`docs/10-database/README.md`](docs/10-database/README.md) | Polyglot Persistence |
| [`docs/11-ai-integration/README.md`](docs/11-ai-integration/README.md) | AI Integration |
| [`docs/DEV-PLAN.md`](docs/DEV-PLAN.md) | **Kế hoạch phát triển** |

**Tổng tài liệu:** ~38,000 dòng.

## 🏗️ Kiến trúc

### Tech Stack
- **Backend:** Go 1.23, Rust 1.81, Python 3.12, C++ (media)
- **Frontend:** Next.js 15 + React 19 + TypeScript 5.6 + Bun
- **Database:** PostgreSQL 16, ScyllaDB 6.x, ClickHouse 24.x, MongoDB 7.x, Valkey 7.x, MinIO, Qdrant, Meilisearch
- **Infrastructure:** K3s, ArgoCD, WireGuard, Prometheus, Grafana, Loki, Jaeger

### 12 Microservices

| Service | Ngôn ngữ | Database | Path |
|---------|----------|----------|------|
| Auth Service | Go | PostgreSQL | `services/auth-service/` |
| Tenant Service | Go | PostgreSQL | `services/tenant-service/` |
| CRM Service | Go | PostgreSQL | `services/crm-service/` |
| Dynamic Model Service | Go | PostgreSQL JSONB | `services/dynamic-model-service/` |
| Lead Service | Go | PostgreSQL + MongoDB | `services/lead-service/` |
| Landing Service | Go | MongoDB + ScyllaDB | `services/landing-service/` |
| Chat Engine | Rust | ScyllaDB + Valkey | `services/chat-engine/` |
| WebRTC SFU | Rust/C++ | ScyllaDB + MinIO | `services/webrtc-sfu/` |
| Recording Service | Rust | MinIO | `services/recording-service/` |
| Lead Scoring | Python | PostgreSQL | `services/lead-scoring/` |
| RAG Chatbot | Python | Qdrant + vLLM | `services/rag-chatbot/` |
| AI SRE | Python | ClickHouse + vLLM | `services/ai-sre/` |
| STT Service | Python | - | `services/stt-service/` |

## 🚀 Quick Start

### Prerequisites
- Docker 24+ & Docker Compose v2
- Go 1.23+
- Rust 1.81+
- Node.js 20+ & Bun 1.1+
- Python 3.12+
- Make

### Local Development (Docker Compose)

```bash
# 1. Clone repo
git clone https://github.com/itdoanh/rinco.git
cd rinco

# 2. Start infrastructure (9 databases + observability)
docker compose -f infra/docker-compose.yml up -d

# 3. Run database migrations
make migrate

# 4. Start services (terminal per service)
cd services/auth-service && go run cmd/main.go
cd services/lead-service && go run cmd/main.go
# ...

# 5. Start frontend
cd frontend/landing && bun dev
cd frontend/admin-portal && bun dev
```

### Production Deployment (K3s + ArgoCD)

```bash
# See docs/DEV-PLAN.md for full deployment guide
```

## 📂 Project Structure

```
.
├── docs/                      # Tài liệu thiết kế (37K+ lines)
├── services/                  # 16 microservices
│   ├── auth-service/         # Go
│   ├── chat-engine/          # Rust
│   └── ...
├── packages/                  # Shared libraries
│   └── go/                   # Go packages
│       ├── auth/             # PASETO + FIDO2
│       ├── db/               # PostgreSQL + RLS
│       ├── logger/           # Structured logging
│       └── ...
├── frontend/                  # Next.js apps
│   ├── landing/              # Landing page (chiase_cu clone)
│   ├── admin-portal/         # Super admin
│   ├── tenant-site/          # Tenant site renderer
│   └── meeting-ui/           # WebRTC meeting UI
├── infra/                     # Infrastructure configs
│   ├── docker-compose.yml    # Local dev stack
│   ├── postgres/             # Init SQL
│   ├── scylla/               # CQL schema
│   ├── clickhouse/           # CH schema
│   ├── prometheus/           # Metrics
│   └── grafana/              # Dashboards
├── migrations/               # DB migration scripts
│   ├── sql/                  # PostgreSQL
│   ├── cql/                  # ScyllaDB
│   └── ch/                   # ClickHouse
├── deployments/              # K8s manifests
│   ├── k8s/                  # Raw manifests
│   └── helm/                 # Helm charts
├── .skills/                  # AI agent skills
└── chiase_cu/                # Old landing page reference
```

## 🤖 AI Agent Skills

Project này có các skills cho Cursor AI ở `.skills/`:

| Skill | Path |
|-------|------|
| Primary (rinco-architect) | `.skills/SKILL.md` |
| Coding Standards | `.skills/rules/coding-standards.md` |
| Database | `.skills/rules/database.md` |
| Deployment | `.skills/rules/deployment.md` |
| Facebook CAPI | `.skills/rules/facebook-capi.md` |
| Multi-tenant | `.skills/rules/multi-tenant.md` |
| Observability | `.skills/rules/observability.md` |

## 📋 Development Status

Xem chi tiết từng chặng trong [`docs/DEV-PLAN.md`](docs/DEV-PLAN.md).

- ✅ **Chặng 1:** Foundation (databases, K3s) – In progress
- ⏳ **Chặng 2:** Identity & Tenancy
- ⏳ **Chặng 3:** CRM Core
- ⏳ **Chặng 4:** Dynamic Model
- ⏳ **Chặng 5:** Landing + CAPI
- ⏳ **Chặng 6:** Real-time Comm
- ⏳ **Chặng 7:** AI Integration
- ⏳ **Chặng 8:** Observability + Security
- ⏳ **Chặng 9:** Tenant Site + Mesh
- ⏳ **Chặng 10:** Super Admin
- ⏳ **Chặng 11:** Hardening

## 📄 License

Proprietary – © 2026 RINCO.

## 📞 Contact

- **Email:** dev@rinco.app
- **GitHub:** https://github.com/itdoanh/rinco
