# Changelog

Tất cả thay đổi đáng chú ý của RINCO Platform được ghi tại đây. Format theo [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [Unreleased]

### Added

#### Microservices (17 total)

**Go services (9):**
- **auth-service** — PASETO v4 + FIDO2/WebAuthn + OAuth2 + sessions + RBAC + API key + crypto_internal
- **tenant-service** — Multi-tenant CRUD, plans, quotas, isolated namespaces, migrations
- **crm-service** — CRM tree (LTREE) cho lead/customer/deal phân cấp, internal + migrations
- **dynamic-model-service** — Runtime JSON schema + sinh API động, internal + migrations
- **lead-service** — Lead capture + ingestion cho scoring (PostgreSQL + MongoDB), NATS events
- **landing-service** — Block renderer + tiered S3 (SSD/HDD) + Facebook Conversions API (CAPI) + tracking
- **email-service** — Multi-driver SMTP, templates, tracking, internal + migrations
- **notification-service** — Push/email/SMS multi-channel + user preferences
- **observability-service** — Logs/metrics/traces aggregator + alerts + audit (ClickHouse)

**Python services (4):**
- **lead-scoring** — XGBoost lead scoring với feature engineering + drift detection (FastAPI)
- **rag-chatbot** — RAG pipeline (chunker → embeddings → retrieval → generation), Qdrant + vLLM
- **ai-sre** — Incident correlator + runbook + auto-hotfix generator (ClickHouse + vLLM)
- **stt-service** — Whisper transcription + diarization + alignment (lead/score schemas)

**Rust services (3):**
- **chat-engine** — E2EE messenger (Signal Protocol — X3DH + Double Ratchet), presence, devices, channels, messages; ScyllaDB + Valkey
- **webrtc-sfu** — Selective Forwarding Unit với AV1/VP9 SVC, SRTP forwarder, bandwidth adaptation
- **recording-service** — GPU NVENC recording, tiered S3 SSD → HDD, STT pipeline integration

**Frontend service (1):**
- **meeting-ui** — WebRTC meeting UI + chat + signaling (Next.js 15 + WebSocket)

#### Frontend Apps (4)

- **landing** — Marketing landing page với block renderer, FB Pixel + CAPI, tracking attribution, Playwright E2E tests
- **admin-portal** — Super-admin (Dark Admin) cho platform owner
- **tenant-site** — Multi-tenant site renderer (dynamic blocks per tenant)
- **meeting-ui** — WebRTC meeting UI (cũng nằm trong `services/meeting-ui/`)

#### Shared Packages

- `packages/go/` (Go shared libs):
  - `auth/` — PASETO + FIDO2 + OAuth + RBAC + session + api_key + uuid_helper + crypto_internal
  - `db/` — PostgreSQL connection pool + RLS + migrations + repository + transaction helpers
  - `logger/` — Structured zap logger
  - `middleware/` — Echo middlewares (auth/tenant/tracing/metrics/audit)
  - `tracing/` — OpenTelemetry setup
  - `capi/` — Facebook Conversions API client
- `packages/frontend/ui/` — React component library:
  - 25+ shadcn/ui components (accordion, alert, alert-dialog, aspect-ratio, calendar, card, carousel, checkbox, collapsible, command, context-menu, drawer, form, hover-card, menubar, navigation-menu, pagination, popover, radio-group, resizable, scroll-area, separator, sheet, slider, sonner, switch, toggle)
  - Tailwind preset + theme tokens + globals.css
  - TypeScript types + barrel export (index.ts)

#### Local Dev Stack

- Full Docker Compose stack (9 databases + observability):
  - PostgreSQL 16 (6 schemas: auth, tenant, crm, dynamic_model, lead, landing, email, notification, ai_sre, lead_scoring, rag)
  - ScyllaDB 6.x (chat, recording_meta, webrtc_meta)
  - MongoDB 7.x (landing_pages, analytics)
  - ClickHouse 24.x (audit, observability, lead_events)
  - Valkey 7.x (Redis-compatible)
  - MinIO (S3-compatible, distributed-ready)
  - Qdrant (vector DB)
  - Meilisearch (full-text search)
  - NATS (event bus)
- Observability stack: Prometheus + Grafana + Loki + Jaeger + OTel Collector + Alertmanager
- K3s + Helm chart infrastructure
- WireGuard mesh (private networking giữa nodes)
- Migrations runner cho SQL + CQL + Mongo + ClickHouse

#### CI/CD

- **8 GitHub Actions workflows** (trong `.github/workflows/`):
  - `ci.yml` — Build + lint + unit test cho 17 services + 4 frontend apps
  - `cd.yml` — Build & push Docker images lên GHCR + bump Helm charts
  - `test-e2e.yml` — Playwright E2E tests + Python integration tests
  - `security.yml` — Trivy + gosec + cargo-audit + bandit weekly scans
- Makefile targets: install, dev, build, test, lint, migrate, deploy, clean, check, up, down, restart, health, logs
- Helper scripts: dev.sh, build.sh, deploy.sh, migrate.sh, lint.sh, test.sh
- PowerShell variants: setup.ps1, generate-dockerfiles.ps1, health-check.ps1, test-lead-flow.ps1

#### Documentation (~38K dòng)

- `docs/00-master/` — Tổng quan master design
- `docs/01-super-admin/` — Super Admin Portal + Dark Admin
- `docs/02-tenant-site/` — Tenant Site + Mesh
- `docs/03-crm-tree/` — CRM Tree với LTREE
- `docs/04-dynamic-model/` — Dynamic Model Engine
- `docs/05-landing-capi/` — Landing Page + Facebook CAPI
- `docs/06-chat-engine/` — Real-time Chat Engine
- `docs/07-webrtc-sfu/` — WebRTC SFU + Recording
- `docs/08-observability/` — 4-tier Observability
- `docs/09-security/` — Multi-tenant Security
- `docs/10-database/` — Polyglot Persistence
- `docs/11-ai-integration/` — AI Integration
- `docs/DEV-PLAN.md` — Kế hoạch phát triển tổng (~38K dòng)
- `docs/ARCHITECTURE.md` — Kiến trúc 6 tầng + service mesh + security model
- `docs/SERVICES.md` — Index 17 services + 4 frontend apps
- `docs/DEVELOPMENT.md` — Dev workflow (setup, conventions, testing, debug)
- `docs/DEPLOYMENT.md` — Production deployment trên K3s
- Service-level READMEs cho mỗi backend service + frontend app

#### Community Files

- `LICENSE` (MIT)
- `CHANGELOG.md` (file này)
- `CONTRIBUTING.md` — Conventional Commits + PR template + testing requirements
- `CODE_OF_CONDUCT.md` — Contributor Covenant
- `SECURITY.md` — Vulnerability reporting + supported versions

### Changed

- Refactored `.gitignore` cho đầy đủ các ngôn ngữ (Go/Rust/Python/Node) + IDE + OS + secrets
- Rewrote root `README.md` với title + tagline + tech stack badges + architecture diagram + service index
- Updated `.env.example` cho production-ready config (Auth, Tenant, Lead, CRM, Landing, Dynamic Model, Chat, WebRTC, AI, Frontend, Infrastructure, Cloud)
- Standardized Makefile với đầy đủ targets (install, dev, build, test, lint, migrate, deploy, clean, health)

### Fixed

- Docker Compose: removed pgbouncer, fixed MinIO command, removed YAML version directive
- Filled gaps in shared packages: tracing, capi, middleware, logger/redactor

---

## Tổng hợp theo commit (chronological)

Từ lần commit đầu đến hiện tại:

| Commit | Type | Mô tả |
|--------|------|-------|
| `d7142d7` | docs | Initial design system (master + 11 detailed sections + skills) |
| `ed21acf` | docs | Sections 00-03 audit + expand (implementation roadmap, code examples, edge cases) |
| `873f3d9` | docs | Sections 04-07 audit + expand |
| `bfdc7c6` | docs | Sections 04-05 audit + expand |
| `4878733` | docs | Sections 06-07 audit + expand |
| `adba5e2` | docs | Sections 10-11 audit + expand |
| `7c9921e` | docs | Sections 08-09 audit + expand (observability + security với DR, cost) |
| `6d0943b` | docs | DEV-PLAN: aggregated 12 design docs (37,114 lines) |
| `36eccf8` | feat | Chặng 1-3 foundation — docker compose, schemas, auth + crm services |
| `d51223a` | feat | Services: landing, dynamic-model, ai-sre, lead-scoring, rag-chatbot, stt, email, notification, observability + Next.js landing |
| `47fa3fe` | feat | Chặng 4-10 services + deployment configs + scripts + tests |
| `15f121f` | fix | Docker Compose: removed pgbouncer, fixed MinIO command, removed YAML version |
| `b437143` | feat | Complete missing pieces: Go modules, Rust source, frontend setup, migrations, CI/CD |
| `eb496bf` | feat | Full local-dev stack (9 DBs) + Prometheus/Grafana/Loki/Jaeger + K3s/Helm + migrations + WireGuard |
| `957ff6e` | feat | auth-service + tenant-service (full HTTP + Connect-RPC API) |
| `af49456` | feat | Packages: tracing, capi, middleware, logger/redactor |
| `1481a40` | docs | Capture new requirements (E2EE messenger + tiered S3 SSD/HDD) |
| `1481a40` | feat | CRM tree + lead-service (NATS events) |
| `dcfbb14` | feat | landing-service (tiered S3 + CAPI + tracking) + dynamic-model-service (JSON schema) |
| `7679882` | feat | email-service (multi-driver + tiered S3 + tracking) + notification-service (multi-channel + prefs) |
| `26650de` | feat | observability-service (logs/traces/metrics/alerts/audit aggregator) |
| `01c659c` | feat | chat-engine: E2EE messenger (Signal Protocol) + ScyllaDB chat engine + presence |
| `61b8050` | feat | webrtc-sfu: SFU với AV1/VP9 SVC, SRTP forwarder, bandwidth adaptation |
| `3a58f1b` | feat | recording-service: GPU-accelerated egress (NVENC), tiered S3 storage, STT pipeline |
| `1cf522e` | feat | Python AI services (lead-scoring, rag-chatbot, ai-sre, stt) |
| `8e34e44` | feat | Packages: tracing, capi, middleware, logger/redactor |
| `5c4b8bb` | feat | landing page (block renderer + FB CAPI + tracking) + shared @rinco/ui library (25 components) |
| `788eae5` | feat | tenant-site + meeting-ui + admin-portal pages |
| `a97179a` | feat | CI: GitHub Actions workflows + scripts + Makefile + .env.example |
| `a9c9bec` | test | Python tests cho lead-scoring + Playwright E2E tests cho frontend apps |
| `cc85133` | feat | Complete all 4 Next.js apps — landing blocks, admin portal, tenant-site, meeting-ui |

---

## Semantic Versioning

Project dùng [SemVer](https://semver.org/):
- **MAJOR** — breaking changes (API contract, schema migration cần manual)
- **MINOR** — new features, backward-compatible
- **PATCH** — bug fixes, backward-compatible

## Conventional Commits

Mọi commit theo format `<type>(<scope>): <subject>`:
- `feat:` — new feature
- `fix:` — bug fix
- `docs:` — documentation only
- `style:` — formatting (no code change)
- `refactor:` — code change (no feature/fix)
- `test:` — add/modify tests
- `chore:` — build/CI/tooling
- `perf:` — performance improvement
