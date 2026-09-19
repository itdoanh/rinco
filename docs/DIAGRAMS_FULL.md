# RINCO — Sơ Đồ Toàn Hệ Thống (Full System Diagrams)

> **Phạm vi**: Toàn bộ kiến trúc RINCO — Multi-Tenant SaaS Hyper-Scale Platform
> **Cập nhật**: 2026-09-20
> **Nguồn**: 12 docs master (`docs/00-master` → `docs/11-ai-integration`) + codebase thực tế
> **Mục đích**: Một tài liệu duy nhất chứa tất cả sơ đồ cần thiết để hiểu, triển khai, vận hành, mở rộng hệ thống.

---

## MỤC LỤC

1. [Sơ đồ 01 — File Tree (Cấu trúc thư mục)](#1-file-tree)
2. [Sơ đồ 02 — Kiến trúc tổng quan 6 lớp](#2-kiến-trúc-6-lớp)
3. [Sơ đồ 03 — Microservices Map (20 services)](#3-microservices-map)
4. [Sơ đồ 04 — Polyglot Persistence Matrix](#4-polyglot-persistence-matrix)
5. [Sơ đồ 05 — Frontend Apps Map](#5-frontend-apps)
6. [Sơ đồ 06 — Luồng dữ liệu End-to-End](#6-end-to-end-flow)
7. [Sơ đồ 07 — Multi-Tenant Routing](#7-multi-tenant-routing)
8. [Sơ đồ 08 — CRM Tree (LTREE)](#8-crm-tree)
9. [Sơ đồ 09 — Dynamic Model Engine](#9-dynamic-model)
10. [Sơ đồ 10 — Landing Page + Facebook CAPI](#10-landing-capi)
11. [Sơ đồ 11 — Chat Engine (E2EE)](#11-chat-engine)
12. [Sơ đồ 12 — WebRTC SFU + Recording](#12-webrtc-sfu)
13. [Sơ đồ 13 — Observability 4-tier](#13-observability)
14. [Sơ đồ 14 — Security Zero-Trust 6-layer](#14-security)
15. [Sơ đồ 15 — AI Integration](#15-ai)
16. [Sơ đồ 16 — Database Schema (PostgreSQL)](#16-db-schema)
17. [Sơ đồ 17 — NATS Event Bus](#17-nats)
18. [Sơ đồ 18 — RBAC Matrix](#18-rbac)
19. [Sơ đồ 19 — Super Admin Pages](#19-admin-pages)
20. [Sơ đồ 20 — Tenant Tree (sơ đồ cây tổ chức)](#20-tenant-tree)
21. [Sơ đồ 21 — Lead Lifecycle](#21-lead-lifecycle)
22. [Sơ đồ 22 — Deal Pipeline](#22-deal-pipeline)
23. [Sơ đồ 23 — Domain-Driven Feature Map (10+ cho mỗi module)](#23-feature-map)
24. [Sơ đồ 24 — CI/CD Pipeline](#24-cicd)
25. [Sơ đồ 25 — Deployment Topology (K3s)](#25-deployment)
26. [Sơ đồ 26 — DR/Backup Strategy](#26-dr)
27. [Sơ đồ 27 — Cost Model](#27-cost)
28. [Sơ đồ 28 — Roadmap Phases](#28-roadmap)
29. [Sơ đồ 29 — Logic quan trọng (chi tiết)](#29-logic)
30. [Sơ đồ 30 — Risks & Mitigations](#30-risks)

---

## 1. FILE TREE

```
RINCO/                                                  ← Workspace root (c:\code\RINCO)
│
├── .github/                       ← GitHub Actions workflows (CI/CD)
│   └── workflows/
│       ├── ci.yml                 ← Lint + build + unit-test tất cả services
│       ├── test-e2e.yml           ← Playwright E2E + Python integration
│       ├── cd.yml                 ← Build images + push GHCR + Helm bump
│       ├── security.yml           ← Trivy + gosec + cargo-audit + bandit
│       ├── rust-ci.yml            ← cargo build/test cho 3 Rust services
│       └── nightly.yml            ← Cron: nightly data refresh + audit
│
├── chiase_cu/                     ← Legacy landing page (BẮT BUỘC giữ 100% nội dung)
│   ├── admin/                     ← Old admin pages (HTML/PHP)
│   ├── anh/                       ← Image assets (webp, png, jpg)
│   ├── api/                       ← Old PHP API endpoints
│   ├── assets/                    ← CSS/JS files cũ (jQuery, Bootstrap cũ)
│   ├── cron/                      ← Old scheduled scripts
│   ├── includes/                  ← Shared includes
│   └── index.html                 ← Entry HTML (giữ nguyên branding)
│
├── deployments/                   ← Kustomize/Helm charts cho K3s prod
│   ├── argocd/                    ← ArgoCD Application manifests
│   ├── kustomize/                 ← Kustomize base + overlays
│   └── terraform/                 ← IaC cho cloud resources
│
├── docs/                          ← Tài liệu thiết kế hệ thống
│   ├── 00-master/                 ← Tổng quan 26 microservices + 9 databases
│   ├── 01-super-admin/            ← Dark Admin Portal (16+ pages)
│   ├── 02-tenant-site/            ← Multi-tenant + Isolated VPS + WireGuard mesh
│   ├── 03-crm-tree/               ← CRM cây (LTREE) + PASETO invitation
│   ├── 04-dynamic-model/          ← Meta-schema + dynamic API/UI gen
│   ├── 05-landing-capi/           ← Landing Page + Meta Pixel + Facebook CAPI
│   ├── 06-chat-engine/            ← E2EE chat (Signal Protocol) + FlatBuffers
│   ├── 07-webrtc-sfu/             ← SFU + GPU recording + Whisper + Llama-3
│   ├── 08-observability/          ← 4-tier observability + AI SRE
│   ├── 09-security/               ← Zero-Trust 6-layer + eBPF/XDP
│   ├── 10-database/               ← Polyglot persistence matrix
│   ├── 11-ai-integration/         ← AI SRE + Predictive CRM + vLLM + STT
│   ├── _summaries/                ← Tóm tắt từng doc (đã tạo trong Loop 2)
│   ├── adr/                       ← Architecture Decision Records (8 files)
│   ├── api/                       ← OpenAPI specs (auth, tenant, crm, lead)
│   ├── runbooks/                  ← DR/incident runbooks
│   ├── services/                  ← Per-service docs
│   ├── wst-issues/                ← Issue tracker files
│   ├── wst-reports/               ← Audit/regression reports
│   ├── ARCHITECTURE.md            ← Kiến trúc 6-lớp (top-level)
│   ├── DEPLOYMENT.md              ← K3s deployment guide
│   ├── DEVELOPMENT.md             ← Local dev setup
│   ├── DEV-PLAN.md                ← 12-phase dev plan
│   ├── DIAGRAMS.md                ← Sơ đồ ngắn (master)
│   ├── DIAGRAMS_FULL.md           ← File này
│   ├── MASTER_PLAN.md             ← Kế hoạch master
│   ├── SERVICES.md                ← Index 17 services
│   ├── SYSTEM_STATUS.md           ← Trạng thái tổng (đã qua 195 loops)
│   ├── USER_GUIDE.md              ← Hướng dẫn người dùng cuối
│   ├── ADMIN_GUIDE.md             ← Hướng dẫn admin
│   ├── CRM_ANALYSIS.md            ← Phân tích CRM
│   ├── CRM_LEAD_VERIFICATION.md   ← Verify lead lifecycle
│   ├── WS-D.md / WS-D-1.md / WS-D-DELIVERY.md  ← Workstream D reports
│   ├── WS-E_REPORT.md             ← Workstream E report
│   └── YEU_CAU_BO_SUNG_2026-09-07.md  ← Yêu cầu bổ sung
│
├── frontend/                      ← 4 Next.js 15 apps + shared packages
│   ├── admin-portal/              ← Super Admin (port 3001) — Dark Admin
│   ├── landing/                   ← Public landing page (port 3000)
│   ├── tenant-site/               ← Multi-tenant site renderer (port 3002)
│   ├── meeting-ui/                ← WebRTC meeting UI (port 3003)
│   └── e2e/                       ← Playwright shared config + specs
│
├── infra/                         ← Infrastructure configs
│   ├── docker-compose.core.yml    ← Core infra (postgres, valkey, nats...)
│   ├── docker-compose.realtime.yml← Realtime infra (scylla, minio)
│   ├── docker-compose.services.yml← All microservices + frontends
│   ├── docker-compose.yml         ← Unified stack (full)
│   ├── cert-manager/              ← Let's Encrypt
│   ├── clickhouse/                ← OLAP DB schemas + MV
│   ├── docker/                    ← Custom Dockerfiles
│   ├── external-dns/              ← Cloudflare/Route53 integration
│   ├── grafana/                   ← Dashboards + provisioning
│   ├── jaeger/                    ← Tracing UI
│   ├── k8s/                       ← K3s manifests (per-service)
│   ├── loki/                      ← Log aggregation
│   ├── meilisearch/               ← Full-text search
│   ├── minio/                     ← Object storage
│   ├── mongo/                     ← MongoDB init scripts
│   ├── mongodb/                   ← Alternative init
│   ├── nats/                      ← JetStream config
│   ├── otel/                      ← OpenTelemetry collector
│   ├── postgres/                  ← Init SQL + RLS + pg_hba
│   ├── prometheus/                ← Metrics + alerts + rules
│   ├── qdrant/                    ← Vector DB
│   ├── scylla/                    ← Wide-column store
│   ├── tempo/                     ← Distributed tracing
│   ├── traefik/                   ← Ingress + dynamic config
│   ├── valkey/                    ← Redis-compatible cache
│   ├── wireguard/                 ← Mesh VPN scripts
│   └── scripts/                   ← backup/cleanup/restore shell
│
├── logs/                          ← Runtime logs (gitignored)
│
├── migrations/                    ← Cross-service migrations (nếu có)
│
├── node_modules/                  ← Workspace deps (gitignored)
│
├── packages/                      ← Shared packages monorepo
│   ├── frontend/
│   │   └── ui/                    ← shadcn/ui + custom components
│   │       ├── components/ui/     ← Button, Input, Dialog, Table...
│   │       ├── lib/               ← utils, formatters, validators
│   │       └── hooks/             ← useDebounce, useToast...
│   └── go/
│       ├── apperrs/               ← Typed error codes
│       ├── auth/                  ← PASETO v4 + FIDO2 + Argon2 + RBAC
│       ├── capi/                  ← Meta CAPI client
│       ├── capifeedback/          ← Feedback loop publisher
│       ├── db/                    ← pgx/v5 + sqlc generated
│       ├── id/                    ← UUIDv7 generator
│       ├── logger/                ← slog JSON wrapper
│       ├── middleware/            ← Echo middleware (auth, ratelimit, RLS)
│       ├── pagination/            ← Cursor + offset pagination
│       ├── ratelimit/             ← Token-bucket per tenant
│       ├── tenant/                ← Tenant context + RLS helpers
│       ├── timex/                 ← Time utilities (RFC3339, ISO8601)
│       └── tracing/               ← OpenTelemetry traces
│
├── scripts/                       ← Shell + PowerShell scripts
│   ├── seed-all.sh                ← Master seed script (Linux)
│   ├── seed-all.ps1               ← Master seed script (Windows)
│   └── ...
│
├── seed/                          ← Seed data SQL/JSON/YAML
│   ├── 00_master.sql              ← Master tenant + admin + base
│   ├── tenants/                   ← Per-tenant seeds
│   ├── leads/                     ← 400+ leads across 5 tenants
│   ├── deals/                     ← 175 deals
│   ├── activities/                ← 1500 activities
│   ├── conversations/             ← 95 conversations 365-540 days
│   ├── events/                    ← ClickHouse events
│   └── ...
│
├── services/                      ← 17+ microservices (xem chi tiết ở §3)
│   ├── ai-sre/                    ← Python — incident correlator + hotfix
│   ├── analytics-service/         ← Go — ClickHouse aggregator
│   ├── auth-service/              ← Go — PASETO + FIDO2 + RBAC
│   ├── billing-service/           ← Go — Stripe integration
│   ├── chat-engine/               ← Rust — E2EE messenger
│   ├── crm-service/               ← Go — CRM tree (LTREE)
│   ├── dynamic-model-service/     ← Go — Meta-schema engine
│   ├── email-service/             ← Go — SMTP multi-driver + tiered S3
│   ├── integration-tests/         ← Cross-service tests
│   ├── landing-service/           ← Go — block renderer + CAPI
│   ├── lead-scoring/              ← Python — XGBoost lead scoring
│   ├── lead-service/              ← Go — Lead ingestion
│   ├── meeting-ui/                ← Frontend (xem frontend/meeting-ui)
│   ├── meta-capi-service/         ← Go — FB CAPI feedback loop
│   ├── notification-service/      ← Go — Push/email/SMS
│   ├── observability-service/     ← Go — Logs/metrics/traces
│   ├── rag-chatbot/               ← Python — RAG pipeline + Qdrant
│   ├── recording-service/         ← Rust — GPU NVENC recording
│   ├── search-service/            ← Go — Meilisearch wrapper
│   ├── stt-service/               ← Python — Whisper transcription
│   ├── tenant-service/            ← Go — Multi-tenant CRUD
│   └── webrtc-sfu/                ← Rust — SFU node
│
├── test-results/                  ← Test artifacts (gitignored)
│
├── .editorconfig                  ← Editor config
├── .env / .env.example            ← Environment variables
├── .gitattributes / .gitignore
├── BUILD_REPORT.md                ← Build status report
├── CHANGELOG.md                   ← Change history
├── CODE_OF_CONDUCT.md
├── CONTRIBUTING.md
├── docker-compose.yml             ← Master compose (full)
├── LICENSE                        ← MIT
├── Makefile                       ← Task runner
├── package.json / package-lock.json ← Root workspace
├── README.md                      ← Top-level README
├── SECURITY.md
├── start-auth.bat                 ← Windows start script
├── start-core-services.bat
├── SYSTEM_STATUS.md               ← Loop tracker (195 loops done)
└── yeucauthietke.md               ← Original Vietnamese requirements
```

---

## 2. KIẾN TRÚC 6 LỚP

```
┌────────────────────────────────────────────────────────────────────────────────────┐
│ TẦNG 1: CLIENT (Browsers / Mobile Apps)                                            │
│   - Landing (3000) — public marketing pages                                         │
│   - Admin-Portal (3001) — Dark Admin (không có DNS, không IP public)                │
│   - Tenant-Site (3002) — multi-tenant site renderer                                │
│   - Meeting-UI (3003) — WebRTC meeting + chat                                       │
└────────────────────────────────────┬───────────────────────────────────────────────┘
                                     │ HTTPS / WSS (mTLS)
                                     ▼
┌────────────────────────────────────────────────────────────────────────────────────┐
│ TẦNG 2: EDGE / GATEWAY (Traefik + cert-manager + eBPF/XDP)                         │
│   - Traefik ingress (built-in K3s)                                                  │
│   - cert-manager (Let's Encrypt + DNS-01)                                           │
│   - eBPF/XDP anti-DDoS ở NIC driver (token bucket + JA4+ fingerprint)              │
│   - Rate-limit per IP/tenant (token bucket)                                         │
│   - Dark Admin: SPA qua WireGuard + Single Packet Authorization (SPA)               │
└────────────────────────────────────┬───────────────────────────────────────────────┘
                                     │ Internal routing (service-mesh ready)
                                     ▼
┌────────────────────────────────────────────────────────────────────────────────────┐
│ TẦNG 3: APPLICATION SERVICES (17 microservices — polyglot)                         │
│   ┌──────────────────────┐  ┌──────────────────────┐  ┌──────────────────────┐     │
│   │ Go (13 services)     │  │ Python (4 services)  │  │ Rust (3 services)    │     │
│   │ - auth, tenant, crm  │  │ - lead-scoring       │  │ - chat-engine        │     │
│   │ - dynamic-model      │  │ - rag-chatbot        │  │ - webrtc-sfu         │     │
│   │ - lead, landing      │  │ - ai-sre             │  │ - recording-service  │     │
│   │ - email, notif       │  │ - stt-service        │  │                      │     │
│   │ - observ, billing    │  │                      │  │                      │     │
│   │ - search, analytics  │  │                      │  │                      │     │
│   │ - meta-capi          │  │                      │  │                      │     │
│   └──────────────────────┘  └──────────────────────┘  └──────────────────────┘     │
└────────────────────────────────────┬───────────────────────────────────────────────┘
                                     │ Connect-RPC / NATS / pgx/scylla-rust-driver
                                     ▼
┌────────────────────────────────────────────────────────────────────────────────────┐
│ TẦNG 4: POLYGLOT PERSISTENCE (9 databases)                                         │
│   PostgreSQL × 6 schemas | ScyllaDB | MongoDB | ClickHouse | Valkey                │
│   MinIO (S3) | Qdrant | Meilisearch | NATS JetStream                                │
└────────────────────────────────────┬───────────────────────────────────────────────┘
                                     │ Async workers / cron / schedulers
                                     ▼
┌────────────────────────────────────────────────────────────────────────────────────┐
│ TẦNG 5: OBSERVABILITY (4-tier)                                                     │
│   Logs: Vector → ClickHouse / Loki                                                  │
│   Metrics: VictoriaMetrics / Prometheus                                             │
│   Traces: OpenTelemetry → Jaeger / Tempo                                            │
│   Errors: Sentry / GlitchTip                                                        │
│   AI: DeepSeek-Coder RCA + Auto-Hotfix                                              │
└────────────────────────────────────┬───────────────────────────────────────────────┘
                                     │ Alertmanager + Telegram/Slack/PagerDuty
                                     ▼
┌────────────────────────────────────────────────────────────────────────────────────┐
│ TẦNG 6: SECURITY (Zero-Trust 6-layer)                                               │
│   Network: eBPF/XDP + Wasm attestation + Argon2 PoW                                 │
│   App: PASETO v4 + FIDO2 + 2-of-3 Quorum                                            │
│   Data: PostgreSQL RLS + LTREE subtree scope                                         │
│   Admin: Dark Admin + SPA + WireGuard mesh                                          │
│   Kernel: eBPF RASP chặn shell exec                                                 │
│   AI: Tenant KV-cache hash + GPU token pool admission                                │
└────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. MICROSERVICES MAP

| # | Service | Lang | Port (HTTP/gRPC) | Database(s) | Responsibilities |
|---|---------|------|------------------|-------------|------------------|
| 1 | **auth-service** | Go | 8081 / 9081 | PostgreSQL (`auth`) + Valkey | PASETO v4 tokens, FIDO2/WebAuthn, OAuth2, sessions, RBAC, Argon2 password hashing, 2FA, login throttling, session rotation |
| 2 | **tenant-service** | Go | 8082 / 9082 | PostgreSQL (`tenant`) | Multi-tenant CRUD, plans (Free/Pro/Enterprise), quotas, isolated VPS bootstrap, domain registration, certificate issuance, suspended/frozen states |
| 3 | **crm-service** | Go | 8083 / 9083 | PostgreSQL (`crm` LTREE) | CRM tree (Giám đốc → Quản lý → Trưởng nhóm → NV), contacts, deals, activities, notes, deal pipelines, lead scoring ingestion, RLS by tenant + subtree |
| 4 | **dynamic-model-service** | Go | 8084 / 9084 | PostgreSQL (`dynamic_model` JSONB) + MongoDB | Meta-schema CRUD, runtime JSON Schema validation, dynamic API/UI generation, workflow engine, ETag optimistic locking, drift detection |
| 5 | **lead-service** | Go | 8085 / 9085 | PostgreSQL (`lead`) + MongoDB | Lead capture, deduplication, scoring ingestion, AI score ingestion, lead lifecycle (new→contacted→qualified→won/lost), UTM tracking |
| 6 | **landing-service** | Go | 8086 / 9086 | MongoDB (`landing_pages`) + ScyllaDB | Block-renderer landing, tiered S3 (hot/cold), server-side Pixel + CAPI proxy, Wasm attestation, lead ingestion endpoint |
| 7 | **email-service** | Go | 8087 / 9087 | PostgreSQL (`email`) | Multi-driver SMTP (MailHog/SES/SendGrid/Postmark), tracking pixels, templates, tiered S3 for attachments, retry queue |
| 8 | **notification-service** | Go | 8088 / 9088 | PostgreSQL (`notification`) | Multi-channel push/email/SMS/in-app, templates, per-user preferences, batch digest, scheduled notifications |
| 9 | **observability-service** | Go | 8089 / 9089 | ClickHouse (`observability`) + PostgreSQL | Logs/metrics/traces aggregator, alert rules, dashboard data, SSE streaming, Sentry-compatible error endpoint |
| 10 | **search-service** | Go | 8090 | PostgreSQL + Meilisearch | Full-text search cho CMS content, leads, contacts; per-tenant index; faceted search; typo-tolerant |
| 11 | **billing-service** | Go | 8092 | PostgreSQL (`billing`) | Stripe integration, invoices, plans, webhooks, usage metering, prorated upgrades/downgrades |
| 12 | **analytics-service** | Go | 8093 | ClickHouse (`analytics`) | OLAP aggregates, materialized views, dashboards cho super-admin |
| 13 | **meta-capi-service** | Go | 8098 | PostgreSQL (state) | Facebook Conversions API server-side, HMAC signature, SHA-256 user data, gobreaker circuit breaker, retry queue, EMQ monitoring |
| 14 | **lead-scoring** | Python | 8094 | PostgreSQL (`lead_scoring`) | XGBoost model → ONNX, 32 features, predict 0-100, pLTV, batch scoring |
| 15 | **rag-chatbot** | Python | 8095 | Qdrant + PostgreSQL (`rag`) | RAG pipeline (chunk → embed → retrieve → rerank → LLM), multi-tenant vLLM, semantic router |
| 16 | **ai-sre** | Python | 8096 | ClickHouse (`ai_sre`) + NATS | DeepSeek-Coder RCA, auto-hotfix PR generator, runbook writer, incident correlator |
| 17 | **stt-service** | Python | 8097 | - | Whisper.cpp transcription, pyannote diarization, batch jobs |
| 18 | **chat-engine** | Rust | 8099 / 8094-WS | ScyllaDB + Valkey | E2EE messenger (Signal Protocol), FlatBuffers protocol, io_uring kernel bypass, presence, typing, channels, files, search |
| 19 | **webrtc-sfu** | Rust | 8100 + UDP 10000-10100 | ScyllaDB + MinIO | Selective Forwarding Unit, AV1/VP9 SVC, eBPF/XDP SRTP routing, meeting orchestrator, recording session control |
| 20 | **recording-service** | Rust | 8101 | MinIO (S3) + ScyllaDB | GPU NVENC composite recording, CUDA pipeline, HLS multi-bitrate, 200 concurrent/L4 GPU |

**Tổng**: 20 services (đã bao gồm cả billing, search, analytics, meta-capi — bổ sung sau so với docs ban đầu).

---

## 4. POLYGLOT PERSISTENCE MATRIX

```
┌─────────────────────┬──────────────┬──────────────────────────────────────┐
│ STORE               │ TECH         │ PURPOSE / USE CASES                  │
├─────────────────────┼──────────────┼──────────────────────────────────────┤
│ PostgreSQL 16       │ RDBMS        │ OLTP chính — auth/tenant/crm/        │
│                     │ + JSONB      │ dynamic_model/lead/landing/email/     │
│                     │ + LTREE      │ notification/ai_sre/lead_scoring/rag │
│                     │ + RLS        │                                      │
├─────────────────────┼──────────────┼──────────────────────────────────────┤
│ ScyllaDB 5.4        │ Wide-column  │ Chat messages, presence, recording   │
│                     │ Shard-per-   │ metadata, WebRTC session metadata,   │
│                     │ core         │ high-throughput event ingestion      │
├─────────────────────┼──────────────┼──────────────────────────────────────┤
│ MongoDB 7           │ Document     │ Landing pages content (flexible      │
│                     │              │ schema), analytics events            │
├─────────────────────┼──────────────┼──────────────────────────────────────┤
│ ClickHouse 24       │ Column-      │ Audit logs, observability aggs,      │
│                     │ oriented     │ lead_events OLAP, AI SRE incident    │
│                     │              │ storage, materialized views          │
├─────────────────────┼──────────────┼──────────────────────────────────────┤
│ Valkey 7 (Redis)    │ In-memory    │ Cache, sessions, presence, pub/sub,  │
│                     │              │ rate-limit, distributed lock, AI     │
│                     │              │ prompt cache, circuit breaker state  │
├─────────────────────┼──────────────┼──────────────────────────────────────┤
│ MinIO               │ Object store │ Recordings (mp4/webm), uploads,      │
│                     │ S3-compat    │ landing-assets (tiered SSD/HDD),     │
│                     │              │ tenant-files, backups                │
├─────────────────────┼──────────────┼──────────────────────────────────────┤
│ Qdrant              │ Vector DB    │ RAG embeddings (per-tenant or        │
│                     │ HNSW+Quant.  │ sharded), HNSW on-disk option        │
├─────────────────────┼──────────────┼──────────────────────────────────────┤
│ Meilisearch         │ Full-text    │ CMS search, leads search, instant    │
│                     │ typo-tolerant│ search-as-you-type                   │
├─────────────────────┼──────────────┼──────────────────────────────────────┤
│ NATS JetStream      │ Event bus    │ lead.*, crm.*, chat.*, audit.*,     │
│                     │              │ capi.*, notification.* subjects,     │
│                     │              │ 90-day retention, replay             │
└─────────────────────┴──────────────┴──────────────────────────────────────┘
```

---

## 5. FRONTEND APPS

| App | Port | Stack | Public? | Trang chính |
|-----|------|-------|---------|-------------|
| **landing** | 3000 | Next.js 15 + React 19 + Tailwind + shadcn/ui + Framer Motion + Lucide | ✅ Public | `/`, `/[tenant]`, `/[tenant]/[page]` |
| **admin-portal** | 3001 | Next.js 15 + NextAuth v5 + TanStack Query + shadcn/ui + Recharts | ❌ Dark Admin (SPA + WireGuard) | `/dashboard`, `/tenants`, `/admins`, `/audit`, `/quorum`, `/billing`, ... |
| **tenant-site** | 3002 | Next.js 15 + Tailwind + shadcn/ui + dynamic blocks | ✅ Public | `/[tenant_slug]`, `/[tenant_slug]/page/[page_slug]` |
| **meeting-ui** | 3003 | Next.js 15 + WebRTC APIs + shadcn/ui + sonner toast | ✅ Public (auth gated) | `/`, `/meeting/[room_id]` |

**Shared**: `packages/frontend/ui/` — 73+ components (Button, Input, Dialog, Table, Toast, Skeleton, Sheet, Tabs, Calendar, Command, etc.).

---

## 6. END-TO-END FLOW

```
[Khách hàng click FB Ads]
        │
        ▼
[Trình duyệt → Meta Pixel event_id UUIDv7 + fbclid]
        │
        ▼
[Landing Page UI (Next.js) — Next.js Server Component]
        │
        ├──► Meta Pixel (client-side): Lead event
        │
        ▼
[Go landing-service /api/v1/leads]
   ├─ Wasm verify (chống bot)
   ├─ Argon2 PoW (chống DDoS)
   ├─ HMAC SHA-256 signature
   ├─ SHA-256 normalize email/phone (Meta standard)
   └─ Ghi ScyllaDB (raw_leads) <10ms
        │
        ▼
[NATS subject: lead.created.{tenant_id}]
        │
        ├─► Python lead-scoring (XGBoost → ONNX) — predict 0-100
        │     └─► PostgreSQL lead_scoring.scores
        │
        ├─► meta-capi-service (Go) — POST Meta Graph API
        │     ├─ HMAC chống CAPI poisoning
        │     ├─ gobreaker circuit breaker
        │     └─ Meta trả EMQ ≥ 7
        │
        └─► crm-service (Go) — Tạo Contact + Deal trong PostgreSQL
              ├─ RLS: tenant_id + subtree scope
              └─ Audit log → ClickHouse
        │
        ▼
[Nhân viên CRM (Quản lý / Trưởng nhóm) login → admin-portal / tenant-site]
        │
        ├─► Xem lead mới (real-time SSE)
        ├─► Update status: new → contacted → qualified → won
        ├─► Move lead giữa các nhân viên (LTREE path update)
        └─► Khi chuyển → "won":
              └─► meta-capi-service POST Purchase event với deal_value
                    └─► FB tối ưu ads lookalike audience
```

---

## 7. MULTI-TENANT ROUTING

```
                          ┌─────────────────────────────────────┐
                          │ Traefik Ingress (TLS termination)    │
                          └────────────────┬────────────────────┘
                                           │
        ┌──────────────────────────────────┼──────────────────────────────────┐
        │                                  │                                  │
        ▼                                  ▼                                  ▼
   SUBPATH                          SUBDOMAIN                          CUSTOM DOMAIN
   hanghoaphaisinh.net/             apex.hanghoaphaisinh.net           landing.apexcorp.vn
   apexfintech                      /                                   (CNAME trỏ về IP)
        │                                  │                                  │
        │ Tenant resolver (Go middleware)  │                                  │
        │ path-prefix → tenant_slug        │ Host header → tenant_slug        │ Host header → tenant_slug
        │ (Valkey cache <0.5ms)            │ (Valkey cache <0.5ms)            │ (Valkey cache <0.5ms)
        └──────────────────────────────────┴──────────────────────────────────┘
                                           │
                                           ▼
                              ┌────────────────────────────┐
                              │ Tenant Context Middleware  │
                              │ - SET app.current_tenant   │
                              │ - RLS policies active      │
                              │ - Valkey namespace prefix  │
                              └────────────────────────────┘
```

**3 mô hình**:
1. **Subpath** (`hanghoaphaisinh.net/apexfintech`) — shared domain, path-based
2. **Subdomain** (`apex.hanghoaphaisinh.net`) — DNS wildcard
3. **Custom Domain** (`apex.hanghoaphaisinh.net` riêng) — CNAME trỏ về cluster, auto Let's Encrypt

**Isolated VPS** (mở rộng): tenant có thể có 1 VPS riêng, kết nối vào mesh qua WireGuard, được central orchestrator điều phối tài nguyên nhàn rỗi.

---

## 8. CRM TREE

```
[Giám đốc — root tenant]
        │
        ├──► [Quản lý Kinh doanh KV1]
        │       ├──► [Trưởng nhóm HN]
        │       │       ├──► [NV Tư vấn A]
        │       │       ├──► [NV Tư vấn B]
        │       │       └──► [NV Tư vấn C]
        │       └──► [Trưởng nhóm HCM]
        │               ├──► [NV Tư vấn D]
        │               └──► [NV Tư vấn E]
        │
        └──► [Quản lý Kinh doanh KV2]
                ├──► [Trưởng nhóm ĐN]
                │       └──► [NV Tư vấn F]
                └──► [Trưởng nhóm HUE]
                        └──► [NV Tư vấn G]

Path (LTREE):
  gd.qldkv1.tnhn.nva
  gd.qldkv1.tnhn.nvb
  gd.qldkv1.tnhcm.nvd
  gd.qldkv2.tndn.nvf
  gd.qldkv2.tnhue.nvg
```

**RLS subtree scope**:
```sql
CREATE POLICY crm_visible_to_subtree ON users
  USING (
    tenant_id = current_setting('app.current_tenant_id')::UUID
    AND path <@ current_setting('app.current_user_subtree_path')::LTREE
  );
```

**Operations**:
- Promote/Demote: cập nhật role_id, không thay đổi path
- Move with subtree: cập nhật path của user + tất cả descendants (1 transaction)
- Invitation: PASETO v4 token với parent_user_id + target_role_id + expire_time
- Cycle prevention: khi move, check path overlap (path <@ old_path hoặc path @> old_path → reject)

---

## 9. DYNAMIC MODEL

```
[Admin / Chủ DN vào admin-portal → /dynamic-models]
        │
        ▼
[JSON Schema Editor (UI động — render từ meta-schema)]
        │
        ├─► Define fields (33 types: text, number, date, select, multi-select,
        │                  relation, formula, rollup, file, signature, json...)
        ├─► Define validations (regex, min/max, CEL expression)
        ├─► Define workflows (state machine: draft → review → approved → published)
        ├─► Define views (kanban, calendar, gallery, gantt)
        └─► Define forms (multi-step, conditional fields)
        │
        ▼
[POST /api/v1/dynamic-models/schemas]
        │
        ├─► JSON Schema 2020-12 validate
        ├─► ETag hash (optimistic locking)
        ├─► PostgreSQL INSERT JSONB
        ├─► NATS publish: dynamic_model.schema.updated
        │
        ▼
[Code Generator — Ent (Go) + ts-proto (TS)]
        │
        ├─► Generate Go entity + handler + route
        ├─► Generate TS types + Zod schema + React form
        └─► Canary 5% → 100% rollout
        │
        ▼
[Dynamic CRUD endpoint live]
        ├─► /api/v1/dynamic/{model_name}
        ├─► Validation pipeline (Type → JSON Schema → CEL → DB Unique → Cross-entity)
        └─► Storage: PostgreSQL JSONB + GIN index; auto-promote to MongoDB khi drift
```

---

## 10. LANDING + CAPI

```
[Khách click FB ad]
        │   (event_id UUIDv7, fbclid, fbp, IP, UA, server-side via /api/capi)
        ▼
[Browser / Next.js]
   ├─► Meta Pixel: fbq('track', 'Lead', {...})
   ├─► Wasm attestation token (8 signals: webdriver, swiftshader, canvas...)
   ├─► Argon2 PoW (adaptive difficulty, 20-50ms browser time)
   └─► Form submit → POST /api/v1/leads
        │
        ▼
[Go landing-service]
   ├─► Verify HMAC SHA-256 signature
   ├─► Verify Wasm token
   ├─► Verify Argon2 PoW
   ├─► SHA-256 normalize user data (email, phone prepend "84" VN, name, city, zip, country, external_id)
   ├─► ScyllaDB write (raw_leads) — <10ms p99
   └─► NATS publish: lead.created.{tenant_id}
        │
        ▼
[NATS subscribers]
   ├─► lead-scoring: XGBoost predict 0-100
   ├─► meta-capi: POST Meta Graph API (HMAC chống CAPI poisoning, gobreaker CB)
   └─► crm-service: Tạo Contact + Deal
        │
        ▼
[CRM updates → CRM Conversion Event]
        └─► meta-capi: POST Purchase với deal_value → Meta tối ưu ads

Event taxonomy: Lead, QualifiedLead, Schedule, Purchase, Subscribe,
                CompleteRegistration, Custom_High_Value, Custom_SQL_Qualified,
                Custom_Audience_Upload, Custom_Conversion
```

**Landing Page Block System** (20 blocks):
- TopBar, StickyHeader, MobileMenu, Hero, LogoCloud, MarketContext
- SpeakerSection, TrustSection, RegistrationForm (multi-step), RiskWarning
- CompanyInfo, Footer, StickyCTA, LeadCaptureModal, CookieConsent
- FAQAccordion, TestimonialsCarousel, PricingTable, VideoEmbed, FloatingWhatsApp

**Tiered S3 storage** (MinIO):
- Hot bucket (NVMe SSD): files < 30 ngày, accessed thường xuyên
- Cold bucket (HDD): files > 30 ngày, ít truy cập → transition tự động
- Bucket: `rinco-tenant-assets/{tenant_id}/`, `rinco-backups/`

---

## 11. CHAT ENGINE

```
[Client A — WebTransport/Zero-Copy WebSocket]
        │
        ▼
[Rust chat-engine gateway — tokio-uring + eBPF/XDP]
   ├─► FlatBuffers protocol (21 frame types, zero-parse overhead)
   ├─► Singleflight coalescing (50K clients querying → 1 DB query)
   └─► Per-tenant channel router (consistent hashing)
        │
        ├─► Valkey: presence + typing + read receipts
        └─► ScyllaDB: messages (shard-per-core, TWCS compaction)
                ├─► Keyspace `rinco_chat.messages_by_channel`
                ├─► Keyspace `rinco_chat.messages_by_user`
                └─► Keyspace `rinco_chat.attachments`

[E2EE — Signal Protocol]
   ├─► Per-channel symmetric key (rotated mỗi 7 ngày)
   ├─► X3DH key agreement (initial contact)
   ├─► Double Ratchet (forward secrecy + post-compromise security)
   └─► Server KHÔNG thấy plaintext

[Channel types]
   ├─► 1-1 DM
   ├─► Group channel
   ├─► Broadcast channel
   └─► Threaded channel (reply-to)

[Features: 152]
   - Reactions (emoji), polls, location, contact card, file attachment,
     voice message, video message, message editing, deletion (own only),
     forwarded messages, replies, mentions (@user, @channel, @here),
     pinned messages, search (full-text + semantic), mute/unmute,
     archive, block, report, typing indicators, read receipts,
     last-seen, online status, custom themes, custom emoji (tenant-level),
     scheduled messages, drafts, message requests (DM filter),
     channel categories, member roles (owner/admin/member/guest),
     join links, invite codes, audit log (admin only)...
```

---

## 12. WEBRTC SFU

```
[Publisher (WebRTC client)]
   │ AV1/VP9 SVC stream (4 spatial × 4 temporal layers = 16 layers)
   ▼
[Rust webrtc-sfu node]
   ├─► Zero-transcoding (chỉ forward packet, không decode)
   ├─► eBPF/XDP SRTP routing (<100ns)
   ├─► Coturn TURN (HMAC-SHA1 time-limited credentials, multi-region GeoDNS)
   └─► Dynamic layer switching theo bandwidth từng subscriber
        │
        ├─► Subscriber High (full 1080p60)
        ├─► Subscriber Med (720p30, temporal layer 1)
        └─► Subscriber Low (360p15, spatial layer 0)

[Recording — recording-service]
   ├─► Zero-Chromium (Rust worker join RTP binary stream)
   ├─► CUDA composite kernel (mix video grid in VRAM)
   ├─► NVENC H.264 encode (200 concurrent/L4 GPU)
   ├─► Direct chunked stream → MinIO S3
   └─► HLS multi-bitrate playback

[STT — stt-service]
   ├─► Audio extracted from RTP
   ├─► Whisper.cpp + TensorRT GPU
   └─► pyannote diarization (speaker ID)

[LLM Summary — ai-sre / custom]
   ├─► Llama-3 70B Vietnamese prompt-engineered
   ├─► JSON structured output: summary + action_items + decisions
   └─► Auto-create CRM tasks cho từng action_item
```

**Meeting controls**: mute/unmute, video on/off, screen share (high-priority RTCP flag), hand raise, chat side-panel, participant grid 1/4/9/16/25, picture-in-picture, virtual backgrounds, noise suppression (RNNoise), recording indicator, waiting room, breakout rooms, polls, emoji reactions, end-to-end encrypted 1-1 messages in meeting.

---

## 13. OBSERVABILITY (4-tier)

```
┌─────────────────────────────────────────────────────────────────┐
│ TIER 1: APPLICATION LEVEL                                        │
│   - UUIDv7 trace_id (chronological)                              │
│   - slog JSON (Go) / tracing JSON (Rust)                         │
│   - Error wrapping (fmt.Errorf("...: %w", err))                  │
│   - OpenTelemetry spans                                          │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ TIER 2: TELEMETRY TRIAD                                          │
│   Logs    → Vector (Rust agent) → ClickHouse + Loki             │
│   Metrics → Prometheus / VictoriaMetrics                         │
│   Traces  → OpenTelemetry Collector → Jaeger / Tempo            │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ TIER 3: ALERTING + AI RCA                                        │
│   - Sentry/GlitchTip error aggregation                           │
│   - Alertmanager (Prometheus)                                    │
│   - AI SRE: DeepSeek-Coder-V2 RCA < 3s                          │
│   - Auto-PR hotfix (confidence threshold)                       │
│   - Multi-channel: Telegram / Slack / PagerDuty / Twilio SMS    │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ TIER 4: SELF-HEALING                                             │
│   - Circuit Breaker (gobreaker: closed → open → half-open)       │
│   - K3s Liveness/Readiness probes (mỗi 2s)                      │
│   - Auto-restart OOM containers (<2s)                            │
│   - Traffic shift sang VPS dự phòng                              │
│   - Graceful degradation (ClickHouse down → Valkey buffer)      │
└─────────────────────────────────────────────────────────────────┘
```

---

## 14. SECURITY (Zero-Trust 6-layer)

```
┌─────────────────────────────────────────────────────────────────┐
│ Layer 1: NETWORK (eBPF/XDP ở NIC)                                │
│   - XDP_DRV mode (chạy trước kernel TCP/IP stack)               │
│   - Token bucket rate limit                                      │
│   - JA4+ TLS fingerprint                                         │
│   - BGP blackhole cho IP độc hại                                 │
└─────────────────────────────────────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 2: APPLICATION (Wasm + Argon2)                             │
│   - Wasm hardware attestation (chống Selenium/Puppeteer)         │
│   - Argon2 PoW adaptive difficulty (chống botnet)                │
│   - HMAC SHA-256 chống CAPI poisoning                            │
└─────────────────────────────────────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 3: AUTH (PASETO + FIDO2)                                   │
│   - PASETO v4 (thay JWT, chống alg=none attack)                  │
│   - FIDO2/WebAuthn (YubiKey)                                     │
│   - Argon2id password hashing                                    │
│   - 2-of-3 Quorum (multi-party authorization)                    │
│   - Session rotation                                             │
└─────────────────────────────────────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 4: DATA (RLS + LTREE)                                      │
│   - PostgreSQL FORCE ROW LEVEL SECURITY                          │
│   - LTREE subtree scope                                          │
│   - Tenant-scoped KV-cache hashing (PROMPTPEEK defense)          │
│   - GPU Token Pool Admission Control                             │
│   - PII redaction trước khi gửi cho AI                           │
└─────────────────────────────────────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 5: ADMIN (Dark Admin)                                      │
│   - WireGuard mesh (không DNS, không IP public)                  │
│   - Single Packet Authorization (SPA)                            │
│   - Không thấy Open Port nếu SPA sai                             │
│   - Just-in-Time access (60 phút TTL)                            │
└─────────────────────────────────────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 6: KERNEL (eBPF RASP)                                      │
│   - Chặn execve syscall từ tiến trình bị khai thác               │
│   - Chặn ptrace (chống debug injection)                          │
│   - Chặn network connect đến IP lạ                               │
│   - SIGKILL tiến trình trong 0.1ms                               │
└─────────────────────────────────────────────────────────────────┘
```

---

## 15. AI INTEGRATION

```
┌─────────────────────────────────────────────────────────────────┐
│ AI SRE & Code Intelligence                                       │
│   - DeepSeek-Coder-V2 (on-premise)                               │
│   - AST parsing + Code RAG (Vector DB = source code embeddings)  │
│   - RCA < 3s                                                     │
│   - Auto-PR hotfix (confidence > 0.85)                          │
│   - Runbook writer (tự sinh runbook mới từ incident)            │
└─────────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ Predictive CRM & Ads Optimizer                                   │
│   - XGBoost → ONNX Runtime (32 features, <5ms p99)               │
│   - Lead Scoring (0-100)                                         │
│   - pLTV (predicted lifetime value)                              │
│   - Threshold: ≥90 = High Value → Meta CAPI Custom_High_Value    │
│   - Threshold: ≥70 = SQL Qualified                               │
│   - <70 = nurture campaign                                       │
│   - Feedback loop: conversion → retrain nightly                  │
└─────────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ Conversational & Media AI                                        │
│   - vLLM (Llama-3-70B-AWQ) + PagedAttention + Prefix Caching     │
│   - Tenant-Scoped KV-Cache: SHA256(tenant_id + prompt_prefix)    │
│   - Jitter 0-50ms chống timing side-channel                      │
│   - Whisper.cpp + TensorRT STT (1/10 realtime)                   │
│   - Semantic Router (phân loại intent → route LLM phù hợp)      │
└─────────────────────────────────────────────────────────────────┘
```

**Defense chống PROMPTPEEK**:
1. Tenant-Scoped Cache Key Hashing
2. GPU Token Pool Admission Control (per-tenant λ, χ, r)
3. Cache key isolation (cache hit chỉ trong cùng tenant)

---

## 16. DATABASE SCHEMA (PostgreSQL — major tables)

### Schema `auth`
- `users(id, tenant_id, email, password_hash_argon2id, fido2_credential_id, totp_secret, status, created_at, updated_at, deleted_at)`
- `user_roles(id, tenant_id, name, permissions_jsonb)`
- `user_role_assignments(user_id, role_id, scope_path ltree, granted_at, granted_by)`
- `sessions(id, user_id, tenant_id, refresh_token_hash, ip, user_agent, expires_at, created_at, rotated_at)`
- `fido2_credentials(id, user_id, credential_id, public_key, counter, transports[], created_at)`
- `oauth_clients(id, tenant_id, client_id_hash, client_secret_hash, redirect_uris, scopes, type)`
- `oauth_tokens(id, client_id, user_id, scope, expires_at, revoked_at)`
- `invitations(id, tenant_id, parent_user_id, target_role_id, paseto_token_hash, expires_at, used_at, created_by)`

### Schema `tenant`
- `tenants(id, slug, name, plan, status, created_at, suspended_at, frozen_at)`
- `tenant_domains(id, tenant_id, domain, type[subpath|subdomain|custom], verified_at, ssl_cert, ssl_key)`
- `tenant_vps(id, tenant_id, vps_ip, region, tier, wg_pubkey, mesh_status, last_heartbeat)`
- `tenant_quotas(id, tenant_id, resource, limit_value, current_value, period)`
- `tenant_plans(id, name, price_usd, features_jsonb, max_users, max_storage_gb)`

### Schema `crm`
- `users(id, tenant_id, parent_id, path LTREE, role_id, status, ...)` — hierarchical
- `contacts(id, tenant_id, owner_user_id, first_name, last_name, email, phone, company, status, score, custom_fields JSONB, ...)`
- `deals(id, tenant_id, owner_user_id, contact_id, pipeline_id, stage, amount, currency, expected_close_date, custom_fields JSONB, ...)`
- `activities(id, tenant_id, owner_user_id, type, subject, due_date, completed_at, contact_id, deal_id, custom_fields JSONB)`
- `notes(id, tenant_id, author_id, contact_id, deal_id, content, mentioned_user_ids[])`
- `pipelines(id, tenant_id, name, stages JSONB)`
- `deal_stage_history(id, deal_id, from_stage, to_stage, changed_by, changed_at)`
- `notification_preferences(user_id, channel, enabled, schedule)`

### Schema `dynamic_model`
- `schemas(id, tenant_id, name, version, json_schema JSONB, status[draft|published], etag, created_by, created_at, published_at)`
- `workflows(id, tenant_id, schema_id, state_machine JSONB, triggers JSONB)`
- `views(id, tenant_id, schema_id, type[kanban|calendar|gallery|gantt], config JSONB)`
- `forms(id, tenant_id, schema_id, steps JSONB, conditional_rules JSONB)`
- `dynamic_records(id, tenant_id, schema_id, data JSONB, created_by, created_at, updated_at, version)` — GIN index trên data
- `dynamic_record_history(id, record_id, version, diff JSONB, changed_by, changed_at)`

### Schema `lead`
- `leads(id, tenant_id, contact_id, source, campaign, utm JSONB, fbclid, fbp, fbc, status, score, pLTV, custom_fields JSONB, created_at, qualified_at, closed_at)`
- `lead_activities(id, lead_id, type, payload JSONB, ts)`

### Schema `landing`
- `pages(id, tenant_id, slug, blocks JSONB, published_version, draft_version, status)`
- `page_versions(id, page_id, version, blocks JSONB, etag, created_by, created_at)`
- `assets(id, tenant_id, type, url, mime, size, tier[hot|cold], last_accessed_at)`

### Schema `email`
- `templates(id, tenant_id, name, subject, html, text, variables JSONB)`
- `messages(id, tenant_id, to, cc, bcc, subject, body, status[sent|failed|queued], sent_at, opened_at, clicked_at)`
- `tracking_events(id, message_id, type[open|click|bounce|complaint], ip, ua, ts)`

### Schema `notification`
- `notifications(id, tenant_id, user_id, type, channel, title, body, payload JSONB, read_at, created_at)`
- `notification_preferences(user_id, type, channel, enabled, schedule)`
- `notification_templates(id, tenant_id, name, type, channel, template)`

### Schema `ai_sre`
- `incidents(id, tenant_id, severity, status, title, summary, root_cause, runbook_id, started_at, resolved_at, rca_trace_id)`
- `hotfixes(id, incident_id, pr_url, diff, confidence, status[proposed|merged|rejected])`
- `runbooks(id, tenant_id, name, steps JSONB, version)`

### Schema `lead_scoring`
- `models(id, tenant_id, version, framework[onnx|xgboost], accuracy, trained_at)`
- `scores(lead_id, model_id, score, pLTV, features JSONB, scored_at)`

### Schema `rag`
- `collections(id, tenant_id, name, embedding_model, chunk_size, chunk_overlap)`
- `documents(id, collection_id, source, url, mime, status, indexed_at)`
- `chunks(id, document_id, ordinal, content, embedding_id, metadata JSONB)`
- `conversations(id, tenant_id, user_id, title, created_at)`
- `messages(id, conversation_id, role, content, sources JSONB, token_count, latency_ms)`

---

## 17. NATS EVENT BUS

```
Subjects:
  auth.events.{tenant_id}              ← login/logout/session_rotated
  tenant.events.{tenant_id}           ← tenant.created/suspended/frozen
  crm.events.{tenant_id}              ← contact.created, deal.updated, ...
  crm.conversions.{tenant_id}         ← Purchase event for CAPI feedback
  lead.events.{tenant_id}             ← lead.created, lead.scored
  lead.created.{tenant_id}            ← from landing-service
  lead.scored.{tenant_id}             ← from lead-scoring
  dynamic_model.schema.updated        ← global (cross-tenant)
  chat.events.{channel_id}            ← message.created, presence.changed
  chat.conversation.{tenant_id}       ← for CRM integration
  capi.send.{tenant_id}               ← for meta-capi-service worker
  notification.send.{tenant_id}       ← for notification-service
  audit.{tenant_id}                   ← any state change → ClickHouse
  ai_sre.incident.{tenant_id}         ← for AI SRE to process
  ai_sre.hotfix.proposed              ← auto-PR notification
```

**JetStream config**:
- 90-day retention
- Replay support
- Per-subject rate limit
- Cluster mode (3 nodes min)

---

## 18. RBAC MATRIX

### Super Admin (Dark Admin)
| Role | Permissions |
|------|-------------|
| **OWNER** | All — including 2-of-3 Quorum approval |
| **SRE_ADMIN** | Infra + service mgmt + observability + runbooks |
| **SECURITY_ADMIN** | Audit + security events + quarantine |
| **SUPPORT_ADMIN** | Tenant impersonation (JIT 60min) + read-only |
| **FINANCE_ADMIN** | Billing + invoices + plans |
| **READONLY_VIEWER** | Read-all, write-none |

### Tenant Roles (CRM)
| Role | Path Scope | Permissions |
|------|-----------|-------------|
| **GĐ** (Director) | root subtree | All + invite Manager |
| **QL** (Manager) | self + descendants | All trong KV + invite Lead |
| **TN** (Team Lead) | self + descendants | All trong team + invite Employee |
| **NV** (Employee) | own | CRUD own contacts/deals, read team |
| **VIEW** | own | Read-only own |

---

## 19. SUPER ADMIN PAGES

1. **Dashboard** — KPI tổng (MRR, tenants, MAU, errors p99)
2. **Tenants** — list + CRUD + suspend/freeze/impersonate
3. **Admins** — admin users + role assignments
4. **Logs** — log search (ClickHouse + Loki)
5. **Traces** — Jaeger UI embed
6. **Metrics** — Grafana embed
7. **Alerts & AI SRE** — active alerts + AI hotfixes
8. **Audit** — audit log search + filter + export
9. **Quorum** — pending 2-of-3 approvals
10. **Billing** — invoices + Stripe sync
11. **Resources** — VPS + cluster node status
12. **System** — version + feature flags + maintenance mode
13. **Settings** — global config
14. **Notification Center** — admin notifications
15. **Impersonation** — JIT access (60min TTL)
16. **Broadcast** — gửi thông báo cho tất cả tenants (theo plan/tier)
17. **GDPR** — data subject requests
18. **Data Residency** — per-tenant region pinning
19. **Webhooks** — outbound webhook configs
20. **API Keys** — programmatic access
21. **Maintenance** — scheduled maintenance windows
22. **Bot Integration** — Telegram/Slack bot status
23. **Templates** — email/notification templates

---

## 20. TENANT TREE (Sơ đồ tổ chức doanh nghiệp)

```
Tenant: Apex Fintech (apex-fintech)
├── Giám đốc Điều hành (CEO)
│   ├── Phó Giám đốc Kinh doanh (VP Sales)
│   │   ├── Quản lý KV Miền Bắc
│   │   │   ├── Trưởng nhóm HN-1
│   │   │   │   ├── NV Tư vấn 1
│   │   │   │   ├── NV Tư vấn 2
│   │   │   │   └── NV Tư vấn 3
│   │   │   └── Trưởng nhóm HN-2
│   │   │       └── (empty)
│   │   ├── Quản lý KV Miền Nam
│   │   │   ├── Trưởng nhóm HCM-1
│   │   │   │   └── NV Tư vấn 4
│   │   │   └── Trưởng nhóm HCM-2
│   │   │       ├── NV Tư vấn 5
│   │   │       └── NV Tư vấn 6
│   │   └── Quản lý KV Miền Trung
│   │       └── Trưởng nhóm ĐN
│   │           └── NV Tư vấn 7
│   └── Phó Giám đốc Marketing (VP Marketing)
│       └── Quản lý Marketing
│           └── NV Marketing
└── (root) ← Admin tạo CEO
```

**Path examples** (LTREE):
- `gd.pgd_kd.qlkv_bac.tn_hn1.nv_1`
- `gd.pgd_mkt.qlmkt.nvmkt`

---

## 21. LEAD LIFECYCLE

```
[Khách truy cập Landing Page]
        │
        ▼
[Submit Form] → POST /api/v1/leads
        │
        ▼
[Status: NEW] ────────────────────────┐
        │                              │
        ▼                              │
[CRM Contact tạo, Lead auto-created]  │
        │                              │
        ▼                              │
[Status: CONTACTED]                   │
   (NV gọi điện, ghi note)            │
        │                              │
        ▼                              │
[Status: QUALIFIED]                   │
   (đủ budget, đúng ICP)              │
        │                              │
        ▼                              │
[Status: PROPOSAL]                    │
   (gửi báo giá)                      │
        │                              │
        ├─► [Status: WON] ────────────┤
       │    → Tạo Deal won             │
       │    → meta-capi Purchase event │
       │                               │
       └─► [Status: LOST] ────────────┤
            (ghi lý do, archive)        │
                                         │
[Auto-archive sau 90 ngày inactive] ────┘
```

**Side states**:
- DUPLICATE (dedup theo email/phone hash)
- INVALID (email bounce, phone invalid)
- BLOCKED (NV block thủ công)
- IMPORTED (từ CSV bulk import)

---

## 22. DEAL PIPELINE

```
[Pipeline: Sales mặc định]
├── Stage 1: Lead In (auto from contact)
├── Stage 2: Qualified
├── Stage 3: Proposal Sent
├── Stage 4: Negotiation
├── Stage 5: Closed Won ← (terminal, → Meta Purchase event)
└── Stage 6: Closed Lost ← (terminal)

[Pipeline: Enterprise Sales]
├── Stage 1: Discovery
├── Stage 2: Demo Scheduled
├── Stage 3: POC
├── Stage 4: Stakeholder Buy-in
├── Stage 5: Contract Sent
├── Stage 6: Closed Won
└── Stage 7: Closed Lost

[Pipeline: Customer Success]
├── Stage 1: Onboarding
├── Stage 2: Adoption
├── Stage 3: Expansion
├── Stage 4: Renewal
└── Stage 5: Churn Risk
```

**Deal fields**:
- amount, currency, expected_close_date, probability, owner_user_id
- contact_id, company_id
- custom_fields (per industry template)
- next_step, last_activity_at

**Deal stage history**: mỗi lần chuyển stage → audit + notification + Slack/Telegram alert.

---

## 23. FEATURE MAP (10+ cho mỗi module)

### Super Admin (16+ pages, 130 features)
1. Dashboard KPI real-time
2. Tenant CRUD + suspend/freeze/impersonate
3. Admin CRUD + role mgmt
4. Log search với filter nâng cao
5. Trace viewer (Jaeger)
6. Metric explorer
7. Alert rules CRUD
8. Audit log search + export CSV/JSON
9. 2-of-3 Quorum approval workflow
10. Billing dashboard + invoice mgmt
11. VPS cluster status + resource graphs
12. System config + feature flags + maintenance
13. Notification templates CRUD
14. Impersonation (JIT 60min)
15. Broadcast to tenants
16. GDPR data export/delete
17. Data residency per-tenant
18. Webhook outbound CRUD
19. API key mgmt với scopes
20. Bot integration (Telegram/Slack status)
21. Maintenance window scheduling
22. Email template editor

### Tenant Site (15+ feature groups, ~80 features)
1. Site CRUD + theme + branding
2. Page builder (block-based)
3. Domain mgmt (subpath/subdomain/custom + SSL)
4. Isolated VPS provisioning
5. WireGuard mesh join/leave
6. Resource sharing scheduler config
7. Custom CSS/JS injection
8. i18n (vi/en/ja/zh)
9. SEO meta + sitemap + robots.txt
10. Analytics integration (GA4, Meta Pixel, TikTok Pixel)
11. Custom DNS records
12. Email forwarding
13. CDN config
14. Backup/restore self-service
15. Audit log view (own scope)

### CRM Tree (12 modules, 170+ features)
**User & Org**:
1. User CRUD
2. Org tree (LTREE)
3. Role assignment (scope: own/subtree/global)
4. PASETO invitation
5. Promotion/demotion
6. Subtree move
7. Department mgmt
8. Profile + avatar
9. Password change
10. 2FA setup
11. Active sessions view
12. Audit (own actions)

**Lead**:
13. Lead CRUD
14. Lead import CSV
15. Lead dedup
16. Lead scoring (auto từ AI)
17. Lead assignment (auto round-robin / manual)
18. Lead status workflow
19. Lead capture (form webhook)
20. UTM tracking
21. Lead merge
22. Lead archive

**Contact**:
23. Contact CRUD
24. Contact enrichment (API)
25. Contact merge
26. Custom fields
27. Tags
28. Segments (saved filters)

**Deal**:
29. Deal CRUD
30. Pipeline CRUD
31. Stage transition
32. Probability auto-calc
33. Forecast
34. Win/loss reasons
35. Deal templates

**Activity**:
36. Activity CRUD (call/email/meeting/task)
37. Calendar view
38. Activity reminders
39. Activity templates
40. Bulk activity

**Notes**:
41. Note CRUD
42. @mention
43. Note-to-deal/contact link

**Notification**:
44. In-app notifications
45. Email digest
46. Push (web push)
47. Per-type preferences

### Dynamic Model (15+ features per category, ~100 total)
1. Schema CRUD (JSON Schema 2020-12)
2. Field type: 33 types
3. Validation: regex, CEL
4. Workflow state machine
5. View types: kanban, calendar, gallery, gantt
6. Form: multi-step, conditional
7. Rollup fields
8. Formula fields
9. Hot reload (no restart)
10. Versioning + ETag
11. Templates: Real Estate, Finance, Education, E-commerce, Agency
12. Import/export CSV/JSON
13. Audit per record
14. Code-gen (Go + TS)
15. Drift detection → MongoDB promote
16. Per-tenant RBAC on schema

### Landing Page + CAPI (60+ features)
1. Block system (20 blocks)
2. Per-block visibility rules
3. A/B testing (z-test)
4. Meta Pixel (22 events)
5. Facebook CAPI (HMAC signed)
6. SHA-256 user data normalize
7. Wasm attestation (8 signals)
8. Argon2 PoW adaptive
9. Lead scoring routing (≥90/≥70/<70)
10. Lead magnet download
11. Multi-step form
12. Modal form (sticky CTA)
13. Cookie consent (GDPR/PDPA)
14. i18n (4 languages)
15. Lighthouse CI gate
16. Playwright visual regression
17. Hot/cold S3 tiering
18. Custom domain + SSL auto
19. Conversion API feedback loop
20. EMQ monitor + dashboard
21. Idempotency key
22. GDPR data export/delete
23. Testcontainers (local dev)
24. Backup daily + 30-day retention

### Chat Engine (152 features) — xem `06-chat-engine-summary.md`

### WebRTC SFU + Recording (135 features) — xem `07-webrtc-sfu-summary.md`

### Observability (131 features) — xem `08-observability-summary.md`

### Security (125 features) — xem `09-security-summary.md`

### Database (40 features) — xem `10-database-summary.md`

### AI Integration (135 features) — xem `11-ai-integration-summary.md`

---

## 24. CI/CD PIPELINE

```
[git push to main]
        │
        ▼
[GitHub Actions — ci.yml]
   ├─► Go: build + lint (golangci-lint) + test cho 13 services
   ├─► Python: ruff + black + pytest cho 4 services
   ├─► Rust: cargo build + clippy + test cho 3 services
   ├─► TypeScript: eslint + tsc + vitest cho 4 apps
   ├─► Trivy scan (container images)
   └─► Codecov upload
        │
        ▼ (if green)
[cd.yml]
   ├─► Build Docker images (multi-stage, distroless final)
   ├─► Push to GHCR (ghcr.io/itdoanh/rinco/*)
   ├─► Update Helm chart versions
   └─► Commit version bump to repo
        │
        ▼
[ArgoCD watches GHCR + Helm repo]
   └─► Sync to K3s cluster (canary 10% → 100% in 30 min)
        │
        ▼
[nightly.yml — cron 02:00 UTC]
   ├─► Refresh seed data
   ├─► Retrain lead-scoring model
   ├─► Run security audit
   └─► Send report to admin Telegram

[test-e2e.yml — on PR]
   ├─► Spin up infra (postgres, valkey, nats, mongo, scylla, minio)
   ├─► Migrate + seed
   ├─► Start 4 frontend dev servers
   ├─► Playwright E2E (33 scenarios)
   └─► Lighthouse CI (>90 perf score gate)
```

---

## 25. DEPLOYMENT TOPOLOGY (K3s)

```
[Cloud Provider — bare metal or VMs]
   │
   ├─► Node 1 (control plane + worker)
   │    ├─► Traefik ingress (80/443)
   │    ├─► PostgreSQL (StatefulSet, 3 replicas)
   │    ├─► MinIO (4 nodes, EC:4)
   │    └─► K3s control plane
   │
   ├─► Node 2-4 (workers, application services)
   │    ├─► auth, tenant, crm, dynamic-model
   │    ├─► lead, landing, email, notification
   │    ├─► billing, search, analytics, observ, meta-capi
   │    └─► 4 frontend Next.js apps (SSR + ISR)
   │
   ├─► Node 5-7 (AI workers — GPU)
   │    ├─► lead-scoring (CPU OK)
   │    ├─► rag-chatbot (vLLM — A100 80GB)
   │    ├─► ai-sre (CPU OK, model loaded on-demand)
   │    ├─► stt-service (Whisper.cpp — T4 GPU)
   │    └─► recording-service (NVENC — A10/A30)
   │
   ├─► Node 8-10 (realtime — Rust)
   │    ├─► chat-engine (scaled 3-10 replicas)
   │    └─► webrtc-sfu (scaled by region)
   │
   └─► Tenant VPS (optional, isolated)
        └─► WireGuard mesh (auto-join, 90-day key rotation)
```

**HPA (Horizontal Pod Autoscaler)**:
- auth-service: CPU 60%, min 2 max 10
- crm-service: CPU 70%, min 3 max 20
- chat-engine: connection count, min 3 max 30
- webrtc-sfu: CPU + bandwidth, min 2 max 50
- frontend: request latency p95, min 2 max 8

**PDB (Pod Disruption Budget)**: min 1 available cho mỗi critical service.

---

## 26. DR/BACKUP

### Backup schedule
- **PostgreSQL**: pg_basebackup mỗi 6h, WAL archive continuous. Retention 30 ngày local + 1 năm S3.
- **MongoDB**: mongodump daily. Retention 30 ngày.
- **ScyllaDB**: snapshot mỗi 12h. Retention 7 ngày.
- **ClickHouse**: backup to S3 mỗi 24h. Retention 90 ngày.
- **MinIO**: replication cross-region erasure coding.
- **Valkey**: RDB snapshot mỗi 15 phút + AOF. Retention 7 ngày.
- **Qdrant**: snapshot daily to S3.

### RPO / RTO
| Tier | Service | RPO | RTO |
|------|---------|-----|-----|
| 0 (Critical) | auth, crm, billing | 5 phút | 30 phút |
| 1 (Important) | lead, landing, dynamic-model | 15 phút | 2 giờ |
| 2 (Standard) | email, notification, search | 1 giờ | 4 giờ |
| 3 (Best-effort) | analytics, observability | 4 giờ | 24 giờ |

### DR drill
- Quarterly full restore test (xem `docs/runbooks/dr-restore-drill.md`)
- Auto-failover test với chaos mesh (mỗi tháng)

---

## 27. COST MODEL (estimated monthly)

| Resource | Spec | Cost USD |
|----------|------|----------|
| K3s cluster (10 nodes bare metal) | 32 vCPU / 128GB / 2TB NVMe | ~$1,500 |
| GPU nodes (3× A100 80GB) | rented | ~$4,500 |
| Object storage (10TB) | MinIO on 4× HDD nodes | ~$200 |
| Bandwidth (10TB egress) | cloud provider | ~$900 |
| Domain + SSL | Let's Encrypt (free) | $0 |
| SaaS | Sentry (OSS self-host), Grafana Cloud free tier | $0-100 |
| Email (transactional) | SES (pay per use) | ~$50 |
| SMS | Twilio (pay per use) | ~$30 |
| **Total infra** | | **~$7,280** |
| Plus dev/SRE/PM (5 FTE × $5K) | | $25,000 |
| **Total monthly** | | **~$32,000** |

Per-tenant cost (1000 tenants): ~$32 / tenant / month.

---

## 28. ROADMAP

### Phase 1 — Foundation (✅ DONE)
- 13 Go services + 4 Python + 3 Rust
- 9 databases
- 4 frontend apps
- Basic multi-tenant
- Basic RBAC + RLS

### Phase 2 — Advanced CRM + Dynamic Model (✅ DONE)
- LTREE CRM tree
- PASETO invitation
- Dynamic schema + API gen
- 33 field types

### Phase 3 — Real-time + Media (✅ DONE)
- E2EE chat (Signal Protocol)
- WebRTC SFU (AV1/VP9 SVC)
- GPU recording (NVENC)
- Whisper STT

### Phase 4 — AI + CAPI (🚧 IN PROGRESS)
- Lead scoring (XGBoost → ONNX)
- RAG chatbot (vLLM)
- Meta CAPI feedback loop
- AI SRE (RCA + hotfix)
- 60 features done, 30 remaining

### Phase 5 — Hyper-Scale Optimization (📅 NEXT)
- io_uring gateway (Rust)
- eBPF/XDP in production
- Singleflight coalescing
- GPU token pool admission
- Multi-region active-active

### Phase 6 — Ecosystem (📅 FUTURE)
- Public API + marketplace
- Mobile apps (React Native)
- Zapier/Make.com integration
- White-label SDK
- 3rd-party developer portal

---

## 29. LOGIC QUAN TRỌNG (chi tiết)

### Logic 1: Lead Capture → CRM (End-to-End)

```typescript
// Landing Page (Next.js Client Component)
async function submitLeadForm(formData: FormData) {
  const eventId = crypto.randomUUID() // UUIDv7
  const fbclid = getCookie('fbclid')
  const fbp = getCookie('_fbp')

  // 1. Wasm attestation
  const attestationToken = await getWasmAttestation()

  // 2. Argon2 PoW (nếu cần)
  const powToken = await solveArgon2PoW()

  // 3. Meta Pixel (client-side, fire-and-forget)
  fbq('track', 'Lead', {
    eventID: eventId,
    content_name: 'Apex Fintech Registration',
    value: 0,
    currency: 'VND',
  })

  // 4. POST to landing-service
  const res = await fetch('/api/v1/leads', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Trace-ID': getTraceId(),
      'X-Wasm-Token': attestationToken,
      'X-PoW-Token': powToken,
      'X-Event-ID': eventId,
    },
    body: JSON.stringify({
      tenant_slug: 'apex-fintech',
      form_data: {
        first_name: formData.get('first_name'),
        last_name: formData.get('last_name'),
        email: formData.get('email'),
        phone: formData.get('phone'),
      },
      utm: { source: 'fb', medium: 'cpc', campaign: 'apex-q3' },
      fbclid, fbp,
      client_event_id: eventId,
    }),
  })

  if (res.ok) {
    showSuccessModal()
  }
}
```

```go
// Go landing-service handler
func (h *Handler) CreateLead(c echo.Context) error {
    ctx := c.Request().Context()
    traceID := middleware.GetTraceID(ctx)
    
    var req CreateLeadRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(400, "invalid_request")
    }

    // 1. Verify Wasm token (HMAC + expiry < 1h)
    if !h.wasmVerifier.Verify(req.WasmToken, c.RealIP(), c.Request().UserAgent()) {
        return echo.NewHTTPError(403, "wasm_verification_failed")
    }

    // 2. Verify Argon2 PoW
    if !h.powVerifier.Verify(req.PoWToken, c.RealIP()) {
        return echo.NewHTTPError(429, "pow_failed")
    }

    // 3. SHA-256 normalize user data (Meta standard)
    normalized := capi.NormalizeUserData(capi.UserData{
        Email:   req.FormData.Email,
        Phone:   prepend84VN(req.FormData.Phone),
        FirstName: strings.ToLower(strings.TrimSpace(req.FormData.FirstName)),
        LastName:  strings.ToLower(strings.TrimSpace(req.FormData.LastName)),
        ClientIP: c.RealIP(),
        UserAgent: c.Request().UserAgent(),
        FBP: req.FBP,
        FBC: req.FBC,
        ExternalID: req.ClientEventID,
    })

    // 4. HMAC SHA-256 signature (chống CAPI poisoning)
    signature := capi.ComputeLeadSignature(capi.LeadSignatureInput{
        LeadID:    uuid.NewV7(),
        FBCLID:    req.FBCLID,
        Timestamp: time.Now().Unix(),
        Payload:   normalized,
    }, h.config.CAPISecret)

    // 5. ScyllaDB write (raw_leads) — <10ms
    leadID, err := h.scyllaRepo.InsertRawLead(ctx, RawLead{
        TenantSlug:  req.TenantSlug,
        EventID:     req.ClientEventID,
        FBCLID:      req.FBCLID,
        FBP:         req.FBP,
        UserData:    normalized,
        Signature:   signature,
        UTM:         req.UTM,
        TraceID:     traceID,
        ReceivedAt:  time.Now(),
    })
    if err != nil {
        slog.Error("scylla_insert_failed", "error", err, "trace_id", traceID)
        // Fallback: write to Valkey queue
        h.valkeyRepo.PushRawLead(ctx, leadID, normalized)
    }

    // 6. NATS publish (async fanout)
    h.nats.PublishAsync(fmt.Sprintf("lead.created.%s", req.TenantSlug), NatsLeadCreated{
        LeadID: leadID,
        TenantSlug: req.TenantSlug,
        EventID: req.ClientEventID,
        UserData: normalized,
        Signature: signature,
        ReceivedAt: time.Now(),
    })

    // 7. Return immediately (don't wait for downstream)
    return c.JSON(202, AcceptedResponse{
        LeadID: leadID,
        EventID: req.ClientEventID,
        StatusURL: fmt.Sprintf("/api/v1/leads/%s/status", leadID),
    })
}
```

```python
# Python lead-scoring consumer
@app.post("/v1/score")
async def score_lead(lead: LeadFeatures) -> ScoreResponse:
    # ONNX inference < 5ms p99
    features = lead.to_array()  # 32 features
    score = onnx_session.run(None, {"features": features})[0][0]
    pLTV = predict_ltv_model(features)

    # Persist score
    db.execute(
        "INSERT INTO lead_scoring.scores (lead_id, score, pLTV, scored_at) VALUES ($1, $2, $3, NOW())",
        lead.lead_id, score, pLTV
    )

    # Notify CRM service
    nats.publish(f"lead.scored.{tenant_id}", {
        "lead_id": lead.lead_id,
        "score": score,
        "pLTV": pLTV,
    })

    return ScoreResponse(score=score, pLTV=pLTV)
```

```go
// meta-capi-service consumer (with gobreaker)
func (w *CAPISender) HandleLeadScored(msg NatsLeadScored) error {
    // Circuit breaker check
    if !w.cb.Execute(func() (interface{}, error) {
        // Build CAPI event
        event := capi.Event{
            EventName: "Lead",
            EventTime: msg.ReceivedAt.Unix(),
            EventID:   msg.EventID,
            ActionSource: "website",
            UserData: msg.UserData,
            CustomData: map[string]interface{}{
                "score": msg.Score,
                "pLTV":  msg.PLTV,
            },
        }

        // Threshold routing
        var capiEventName string
        switch {
        case msg.Score >= 90:
            capiEventName = "Custom_High_Value"
        case msg.Score >= 70:
            capiEventName = "Custom_SQL_Qualified"
        default:
            capiEventName = "Lead"  // Don't send high-cost custom events
        }
        event.EventName = capiEventName

        // POST to Meta Graph API
        return nil, w.metaClient.SendEvent(event)
    }) {
        // Circuit open — push to retry queue
        return w.retryQueue.Push(msg)
    }
    return nil
}
```

### Logic 2: CRM Subtree Move (atomic)

```go
// crm-service: Move user X (and all descendants) to new parent Y
func (s *CRMService) MoveUserSubtree(ctx context.Context, userID, newParentID uuid.UUID, actorID uuid.UUID) error {
    // Authorization: actor must be ancestor of newParent
    actorSubtree := s.getUserSubtree(ctx, actorID)
    newParentPath := s.getUserPath(ctx, newParentID)
    if !actorSubtree.Contains(newParentPath) {
        return ErrForbidden
    }

    // Cycle check: newParent must NOT be descendant of userID
    userPath := s.getUserPath(ctx, userID)
    if newParentPath.IsDescendantOf(userPath) {
        return ErrCycleDetected
    }

    // Max depth check
    if newParentPath.Depth()+s.getSubtreeDepth(ctx, userID) > MAX_TREE_DEPTH {
        return ErrMaxDepthExceeded
    }

    // Atomic move (single transaction)
    tx, err := s.db.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)

    oldPath := s.getUserPath(ctx, userID)
    newPath := newParentPath.Append(userPath.Suffix())  // gd.qldkv1.tn_hn1.nv_a → gd.qldkv2.tn_dn.nv_a

    // 1. Update user
    _, err = tx.Exec(ctx, `
        UPDATE users SET path = $1, parent_id = $2, updated_at = NOW()
        WHERE id = $3
    `, newPath, newParentID, userID)

    // 2. Update all descendants
    _, err = tx.Exec(ctx, `
        UPDATE users SET path = $1 || subpath(path, nlevel($2))
        WHERE path <@ $2
    `, newPath, oldPath)

    // 3. Reassign leads owned by user + descendants
    _, err = tx.Exec(ctx, `
        UPDATE leads SET owner_path = $1 || subpath(owner_path, nlevel($2))
        WHERE owner_path <@ $2
    `, newPath, oldPath)

    // 4. Audit
    _, err = tx.Exec(ctx, `
        INSERT INTO audit_actions (tenant_id, actor_id, action, target_user_id, old_path, new_path, ts)
        VALUES ($1, $2, 'move_subtree', $3, $4, $5, NOW())
    `, tenantID, actorID, userID, oldPath, newPath)

    return tx.Commit(ctx)
}
```

### Logic 3: Chat E2EE (Signal Protocol Simplified)

```rust
// chat-engine: Send encrypted message
async fn send_message(channel_id: Uuid, plaintext: &[u8]) -> Result<()> {
    // 1. Get channel symmetric key (from Valkey cache, decrypted with tenant master key)
    let channel_key = get_channel_key(channel_id).await?;
    
    // 2. Generate ephemeral key pair (Double Ratchet step)
    let ephemeral_priv = x25519::EphemeralSecret::random();
    let ephemeral_pub = ephemeral_priv.public_key();
    
    // 3. Derive message key (HKDF)
    let shared_secret = ephemeral_priv.diffie_hellman(&channel_key.public);
    let message_key = hkdf_sha256(&shared_secret, b"message-key", 32);
    
    // 4. Encrypt (AES-256-GCM)
    let ciphertext = aes_gcm_encrypt(message_key, plaintext);
    
    // 5. Build FlatBuffers frame
    let frame = MessageFrame {
        channel_id,
        sender_id: my_user_id,
        ephemeral_pub: ephemeral_pub.to_bytes(),
        ciphertext,
        nonce: random_16_bytes(),
        timestamp: now_ms(),
    };
    
    // 6. Write to ScyllaDB
    let message_id = scylla.insert_message(channel_id, frame).await?;
    
    // 7. Publish via NATS for fanout (other replicas of chat-engine)
    nats.publish(format!("chat.events.{}", channel_id), &frame).await?;
    
    // 8. Update Valkey last-message cursor
    valkey.set_ex(format!("chat:last:{}", channel_id), message_id.to_string(), 86400).await?;
    
    Ok(())
}
```

### Logic 4: 2-of-3 Quorum

```typescript
// admin-portal: Quorum approval page
async function approveAction(actionId: string, signature: string) {
  const action = await fetch(`/api/v1/quorum/${actionId}`).then(r => r.json())
  
  if (action.status !== 'pending') {
    toast.error('Action already resolved')
    return
  }
  
  // YubiKey signature verification
  const isValid = await verifyYubiKeySignature(
    action.payload_hash,
    signature,
    action.challenge
  )
  
  if (!isValid) {
    toast.error('Invalid signature')
    return
  }
  
  await fetch(`/api/v1/quorum/${actionId}/approve`, {
    method: 'POST',
    headers: {
      'X-Admin-ID': currentAdmin.id,
      'X-Signature': signature,
    },
  })
}

// Backend: Count approvals
async function resolveAction(actionId: string) {
  const approvals = await db.query(
    'SELECT admin_id, signature FROM quorum_approvals WHERE action_id = $1',
    [actionId]
  )
  
  const validSigs = approvals.filter(a => verifyYubiKey(a.signature, actionId))
  
  if (validSigs.length >= 2) {
    // Execute action
    await executeAction(actionId)
    await db.query(
      'UPDATE quorum_actions SET status = $1, resolved_at = NOW() WHERE id = $2',
      ['approved', actionId]
    )
    // Notify all 3 admins via Telegram
    notifyAdmins(`Action ${actionId} approved by ${validSigs.length} admins`)
  }
}
```

### Logic 5: AI Lead Scoring Real-time

```python
# lead-scoring: Feature engineering
def featurize_lead(lead: Lead) -> LeadFeatures:
    return LeadFeatures(
        # Engagement (5 features)
        page_views_30d=count_page_views(lead.id, days=30),
        time_on_site_avg=avg_time_on_site(lead.id),
        scroll_depth_max=max_scroll_depth(lead.id),
        video_watch_pct=video_watch_percentage(lead.id),
        cta_clicks=count_cta_clicks(lead.id),
        # Source (5 features)
        utm_source=one_hot(lead.utm.source),
        utm_medium=one_hot(lead.utm.medium),
        utm_campaign_hash=hash(lead.utm.campaign) % 1000,
        device_type=one_hot(lead.device.type),
        country=one_hot(lead.geo.country),
        # Form (8 features)
        has_email=int(bool(lead.email)),
        has_phone=int(bool(lead.phone)),
        email_business=int(is_business_email(lead.email)),
        phone_vn_mobile=int(is_vn_mobile(lead.phone)),
        company_size_log=log1p(lead.company.size or 0),
        job_title_seniority=ordinal(lead.job.seniority),
        industry_relevance=fetch_industry_score(lead.industry),
        form_completion_time_s=lead.form_completion_time.total_seconds(),
        # Behavioral (10 features)
        visit_count_7d=count_visits(lead.id, days=7),
        return_visitor=int(lead.visit_count > 1),
        weekend_visit=int(lead.first_visit.weekday() >= 5),
        business_hours=int(9 <= lead.first_visit.hour <= 17),
        referral_present=int(bool(lead.referral)),
        landing_page_score=score_landing_page(lead.landing_page_slug),
        ab_test_variant_hash=hash(lead.ab_variant) % 100,
        social_proof_viewed=int(lead.viewed_social_proof),
        competitor_mention=int(lead.mentioned_competitor),
        pricing_page_visited=int(lead.visited_pricing),
        # Demographic (4 features)
        age_bucket=ordinal(bucket_age(lead.dob)),
        income_bucket=ordinal(bucket_income(lead.income)),
        education_level=ordinal(lead.education.level),
        marital_status=one_hot(lead.marital),
    )

# Total: 32 features. ONNX model predict 0-100.
```

---

## 30. RISKS & MITIGATIONS

| # | Risk | Impact | Mitigation |
|---|------|--------|------------|
| 1 | Go GC xung đột io_uring | Memory corruption | Gateway Rust thay Go cho L7 I/O critical path |
| 2 | eBPF không handle SRTP state | Mất gói WebRTC | eBPF chỉ L4 routing, SFU vẫn user-space |
| 3 | NVENC session limit (200/concurrent) | Không record được | Cluster Egress worker, passthrough khi không GPU |
| 4 | RLS + LTREE overhead CPU | Slow queries | Cache subtree scope lên Valkey, materialize path |
| 5 | PROMPTPEEK attack trên vLLM | Tenant data leak | Tenant-scoped KV-cache hash + jitter + admission control |
| 6 | Indirect Prompt Injection | AI thực thi admin command | Tách agent, zero-trust permission, sandbox |
| 7 | ScyllaDB shard imbalance | Hot shard | Tweak vnode + compaction strategy |
| 8 | NATS JetStream overflow | Lost events | Auto-scale streams, alert at 70% |
| 9 | MinIO single-region | DR risk | Cross-region replication, RPO < 5min |
| 10 | Lead scoring model drift | Decreasing accuracy | Nightly retrain, A/B test new model |
| 11 | Frontend hydration mismatch | Page flicker | Suspense boundary + ISR + retry-on-miss |
| 12 | Migration order conflict (FK to future tables) | Migration fail | Defer FK to later migration (Loop 1 fix) |

---

**Tổng kết**:
- 20 microservices + 4 frontend apps + 9 databases
- 9 docs master (~1.4 MB design specs)
- ~70% implemented (~46K LOC backend, 4K LOC frontend)
- 195+ iteration loops done, tiếp tục loop 4+ để hoàn thiện

Xem chi tiết từng module trong `docs/_summaries/` (12 file tóm tắt đã tạo ở Loop 2).
