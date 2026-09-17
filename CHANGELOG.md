# Changelog

Tất cả thay đổi đáng chú ý của RINCO Platform được ghi tại đây. Format theo [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [Unreleased]

### Added (Loop WS-F — documentation completion)

#### Documentation overhaul
- **`docs/ARCHITECTURE.md`** — synced canonical port table (20 backend services), added Layer 3 count, expanded polyglot persistence table (added billing, meta-capi, analytics, search).
- **`docs/SERVICES.md`** — rewrote with full port table, 4 new services (billing 8095, search 8097, meta-capi 8098, analytics 8099), updated dependency graph, 20 + 4 = 24 processes total.
- **`docs/USER_GUIDE.md`** — bumped to v1.1: added Global Search section (Meilisearch-powered, Ctrl+K), expanded Billing (3 plans: Starter/Pro/Enterprise with VND pricing + quota tables + usage charts + auto-topup), ASCII System Health diagram showing 20 services + 9 databases.
- **`docs/ADMIN_GUIDE.md`** — bumped to v1.1: added §12 Incident Management with ai-sre, ASCII incident dashboard, AI auto-hotfix flow, runbook library.

#### Architecture Decision Records (8 ADRs in `docs/adr/`)
- **`0001-polyglot-persistence.md`** — why 9 databases (PG, Scylla, Mongo, ClickHouse, Valkey, MinIO, Qdrant, Meilisearch, NATS).
- **`0002-paseto-vs-jwt.md`** — PASETO v4.public + FIDO2/WebAuthn (no `alg:none` attack).
- **`0003-nats-event-bus.md`** — NATS JetStream (sub-ms latency, subject-based routing, replay, 3-node cluster).
- **`0004-ltree-crm.md`** — PostgreSQL LTREE for organizational tree (GIST index, cycle prevention, materialized view for RBAC).
- **`0005-rls-stickness-mitigation.md`** ⚠️ **CRITICAL ISSUE #1** — 6 layers of defense against cross-tenant data leak.
- **`0006-tiered-storage-minio.md`** — MinIO SSD→HDD tier transition (saves 60-70% cost), SSE-KMS, cross-region replica.
- **`0007-chat-engine-rust.md`** — Rust choice (100k+ WS connections, <2KB/conn, official Signal Protocol library).
- **`0008-frontend-multi-app-strategy.md`** — Monorepo 4 apps (landing/admin-portal/tenant-site/meeting-ui) + Next.js 15 + Bun workspaces.
- **`docs/adr/README.md`** — index with critical issues registry.

#### Runbooks (6 in `docs/runbooks/`)
- **`incident-service-down.md`** — generic P0 service-down playbook.
- **`incident-rls-bypass.md`** — CRITICAL: cross-tenant leak response (forensic, contain, notify, postmortem).
- **`incident-db-failover.md`** — PostgreSQL Patroni failover + ScyllaDB repair + RTO/RPO targets.
- **`incident-sfu-overload.md`** — WebRTC SFU scaling + per-pod capacity table + HPA tuning.
- **`incident-cost-spike.md`** — FinOps incident (LLM token explosion, bandwidth spike, storage growth).
- **`dr-restore-drill.md`** — quarterly DR drill runbook (5 phases, RTO/RPO measurement table).
- **`QUICKSTART.md`** — 5-min dev onboarding.
- **`README.md`** — index with severity definitions + on-call rotation.

#### OpenAPI specifications (4 in `docs/api/`)
- **`auth-service.yaml`** — auth/login, FIDO2/WebAuthn, OAuth2, sessions, RBAC, API keys, quorum.
- **`tenant-service.yaml`** — tenant CRUD, plans, quotas, usage metrics.
- **`crm-service.yaml`** — CRM tree nodes (LTREE), leads, contacts, deals, pipelines.
- **`lead-service.yaml`** — lead capture (idempotent), AI scoring, bulk import, sources, attribution.
- **`README.md`** — conventions (error format, pagination, auth, idempotency, rate limits, versioning) + CI integration.

#### System status
- **`docs/SYSTEM_STATUS.md`** — comprehensive health snapshot: 20 services (port, lang, version, uptime, replicas, resources), 9 databases (size, backup, replication, retention), 4 frontends (build, deploy, bundle, Lighthouse), 6 workstreams progress, known issues (Critical/High/Medium/Low + Resolved), CI/CD metrics, capacity, cost, roadmap.

### Changed (Loops 112–199, since `52fa650`)

#### New Go services (4)
- **billing-service** (port 8095) — Stripe/VNPay integration, invoice generation, usage metering, proration.
- **meta-capi-service** (port 8098) — Meta Conversions API client v18.0, event dedup (sha256), feedback loop from Meta.
- **analytics-service** (port 8099) — ClickHouse OLAP cubes, materialized views, dashboard data API.
- **search-service** (port 8097) — Meilisearch indexer (5 indices: leads/deals/contacts/kb/users), RLS-aware, NATS consumer.

#### New test coverage
- `services/billing-service/` — 87 unit tests
- `services/meta-capi-service/` — 42 unit tests
- `services/analytics-service/` — 56 unit tests
- `services/search-service/` — 71 unit tests
- Total Go tests now 1,240 (across 13 services)

#### Database additions
- 2 new PostgreSQL schemas: `meta_capi`, `billing`, `search`.
- 4 indexes added to PostgreSQL for query performance.
- ClickHouse: 2 new databases (`analytics`, `meta_capi_events`).

#### Documentation
- `MASTER_PLAN.md` added (master completion plan with 6 workstreams).

### Fixed (Loops 112–199)

- Docker Compose port corrections (frontend apps).
- CI: GO_VERSION 1.26, PYTHON_VERSION 3.12.
- Playwright CI matrix consolidated into single comprehensive spec (33 tests).
- Multiple cross-tenant safety bugfixes.
- WebRTC reconnect storm hotfix (Loop 201).
- ClickHouse query timeout fix for scans > 100M rows (Loop 199).

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
