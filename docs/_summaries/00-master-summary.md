# RINCO Platform — Master Design Summary

> **Source:** `docs/00-master/README.md` (v1.1, ~1700 lines)
> **Scope:** Whole-system architecture spec covering 26 microservices, 9 databases, AI pipeline, security, observability, roadmap, DR, and SLOs.

---

## 1. Overview

RINCO is a hyper-scale, multi-tenant enterprise CRM + Landing Page + Chat + Meeting + AI platform built on a Vietnamese slogan *"Kết Nối Toàn Năng – Vững Vàng Quản Trị"* (Universal Connection – Solid Governance). Its target is **≥ 10,000 partner tenants / 1M MAU in year one**, with strict latency budgets: Landing FCP < 0.4s, lead ingest < 50ms, chat < 50ms, WebRTC voice < 50ms / video < 100ms. The architecture is a **6-layer polyglot stack** (Clients → Edge/Gateway → Business Microservices → Event Bus → Data Plane → Infrastructure), uses **kernel-bypass (Rust + io_uring + eBPF/XDP)** on the data path, **NATS JetStream + Valkey Streams** for eventing, **9 specialized databases** (Polyglot Persistence), and a **Zero-Trust Multi-Tenant** security model enforced at network, gateway, database (PostgreSQL RLS), admin (Dark Admin via WireGuard SPA), and kernel (eBPF RASP) layers. Core values R.I.N.C.O. = Responsibility, Integrity, Nurture, Cohesion, Optimization; org culture is "Treelike Governance" (root → managers → staff → community).

---

## 2. Microservices (26 services)

| # | Service | Language / Framework | Responsibility | Primary DB |
|---|---------|----------------------|----------------|-------------|
| 1 | **api-gateway** | Go (Echo) | Routing, Auth, Rate Limit, Tenant Resolver | Valkey |
| 2 | **edge-gateway** | Rust (tokio-uring) | io_uring Zero-Copy Gateway, Wasm Attestation challenge | Valkey |
| 3 | **landing-ingest** | Go (Huma) | Form intake, HMAC verify, anti-bot | ScyllaDB |
| 4 | **crm-core** | Go (Ent ORM) | Lead, Contact, Deal, Activity | PostgreSQL 17 |
| 5 | **tree-org** | Go (sqlc) | Hierarchical org tree, RBAC | PostgreSQL (LTREE) |
| 6 | **dynamic-schema** | Go (Huma) | Meta-Schema engine, JSON Schema CRUD | PostgreSQL |
| 7 | **auth-service** | Go | PASETO v4, RBAC, FIDO2/WebAuthn | PostgreSQL |
| 8 | **chat-engine** | Rust | WebSocket/WebTransport, FlatBuffers binary protocol | ScyllaDB + Valkey |
| 9 | **media-sfu** | Rust (str0m) | WebRTC SFU, AV1/VP9 SVC, eBPF/XDP UDP routing | Valkey (room state) |
| 10 | **recorder** | C++ (NVENC/CUDA) | GPU composite encode, chunked upload | MinIO |
| 11 | **ai-scoring** | Python (XGBoost + ONNX) | Lead scoring, pLTV, <5ms inference | PostgreSQL |
| 12 | **ai-conversation** | Python (vLLM + LangChain) | Chatbot + RAG, Llama-3-70B | Qdrant + Valkey |
| 13 | **ai-media** | Python (Whisper.cpp + faster-whisper) | STT, meeting summary, action items | PostgreSQL |
| 14 | **ai-sre** | Python (DeepSeek-Coder) | Auto RCA, hotfix-PR proposal | Vector DB + ClickHouse |
| 15 | **analytics** | Go | OLAP queries, real-time dashboard | ClickHouse |
| 16 | **meta-capi** | Go | Facebook CAPI gateway + Pixel/CAPI hybrid feedback | ScyllaDB + NATS |
| 17 | **notification** | Go | Email / Push / Telegram / SMS | PostgreSQL |
| 18 | **billing** | Go | Subscription, invoice, usage tracking, Stripe + VNPay | PostgreSQL |
| 19 | **tenant-manager** | Go | Create tenant, allocate VPS, deploy config | PostgreSQL + K3s API |
| 20 | **mesh-controller** | Go (Headscale wrapper) | WireGuard mesh between VPS tenants | KV store |
| 21 | **search-service** | Go (Meilisearch client) | Full-text search CRM (Vietnamese) | Meilisearch |
| 22 | **tenant-site** | TS (Next.js) | Partner company homepage (apex.hanghoaphaisinh.net or custom domain) | Static + API |
| 23 | **landing-renderer** | TS (Next.js 15 + Bun) | Landing Page SSR, render per tenant | ScyllaDB + MongoDB |
| 24 | **bff-admin** | TS (Bun) | Backend-for-Frontend for Admin Web | API aggregation |
| 25 | **bff-crm** | TS (Bun) | Backend-for-Frontend for CRM Web | API aggregation |
| 26 | **report-engine** | Go | Export reports, BI dashboard | ClickHouse |

**Communication rules:**
- **Public API (Web/App → Gateway):** REST (Huma + Echo) + WebTransport/WebSocket.
- **Internal RPC (Service ↔ Service):** Connect-RPC (gRPC over HTTP/1.1, browser-friendly).
- **Async Event:** NATS JetStream (primary) + Valkey Streams (lightweight log) + Kafka (heavy).
- **Realtime:** WebSocket + FlatBuffers binary (zero-parse).

---

## 3. Polyglot Persistence Matrix

| Subsystem | Database | Model | Why |
|-----------|----------|-------|-----|
| Chat history & event streams | **ScyllaDB 6.x** | Wide-column, shard-per-core | Write > 1M/s, p99 < 10ms |
| Real-time analytics & clickstream | **ClickHouse 24.x** | Columnar + SIMD | 10:1 compression, fast OLAP |
| CRM core & business logic | **PostgreSQL 17+** | Relational, MVCC | ACID, JSONB, LTREE, pgvector |
| State management, presence, cache | **Valkey 9.x** | In-memory | 8–20% faster than Redis, BSD |
| Dynamic form data | **MongoDB 7.x** | Document | Schema flexibility |
| Media files (video/audio) | **MinIO / S3** | Object + erasure coding | Cheap, unlimited scale |
| Vector search (> 50M) | **Qdrant** (Rust) | HNSW | Multi-tenant vector isolation |
| Vector search (< 10M) | **pgvector** | HNSW in Postgres | No extra service |
| Full-text search | **Meilisearch** | Inverted index | Best Vietnamese tokenization |
| Cache layer (session, rate limit, coalescing) | **Valkey 9.x** | In-memory | Token bucket, singleflight |

**Selection rule:** Heavy write real-time → ScyllaDB • Relational ACID → Postgres • Flexible schema → MongoDB • OLAP → ClickHouse • Session/lock → Valkey • Large binary → MinIO • Full-text → Meilisearch • Vectors → Qdrant or pgvector.

---

## 4. Key Architectural Decisions (8 ADRs / Principles)

- **AP1 Polyglot Persistence** – each service picks the best DB for its access pattern.
- **AP2 Zero-Copy & Kernel Bypass** – Rust + io_uring + eBPF/XDP on the hot path (Edge Gateway, Chat, SFU).
- **AP3 Event-First** – every business change publishes to NATS JetStream; consumers use Watermill abstraction (NATS/Kafka/Valkey).
- **AP4 Zero-Trust Multi-Tenant** – RLS + tenant context (`SET LOCAL app.current_tenant_id`) at every query.
- **AP5 Observability by Default** – UUIDv7 `trace_id`, structured JSON logs, Prometheus metrics on every service.
- **AP6 Graceful Degradation** – partial failure does not collapse the system (circuit breakers, fallbacks).
- **AP7 Self-Healing** – K3s liveness probes (< 2s restart), `gobreaker`, auto-rollback.
- **AP8 Boring Tech Where Possible** – proven tech unless perf forces otherwise.
- **Languages:** Go 1.26+ (core API), Rust 1.85+ (low-latency + SFU), C++23 (NVENC/CUDA/eBPF), TypeScript on Bun + Next.js 15 (frontend + BFF), Python 3.13 (AI/ML).

---

## 5. Major Feature Areas / Modules

### 5.1 Super Admin Portal (`docs/01-super-admin/`)
Create/lock tenants, manage Super Admin accounts (Dark Admin — no public DNS), realtime dashboard (tenants/users/revenue/traffic/errors), VPS resource allocation, billing & subscription mgmt, feature flags per tenant, tenant impersonation, multi-region control.

### 5.2 Tenant Company Site + Isolated VPS (`docs/02-tenant-site/`)
Each partner gets its own site (`apex.hanghoaphaisinh.net` or custom domain). Two deployment modes: **Shared Cluster** (cost-efficient) or **Isolated VPS** (customer-purchased, joined via WireGuard). All VPSes connect via eBPF/WireGuard mesh; a Distributed Resource Sharing Scheduler lets idle VPSes render/AI for other tenants.

### 5.3 CRM for Staff — Tree Org (`docs/03-crm-tree/`)
- PostgreSQL LTREE hierarchical structure (Director → Manager → Lead → Staff).
- PASETO v4 tokenized invitation link per branch.
- Promote/demote, move subtree, RBAC by role.
- Lead dashboard, KPI, commission tracking.

### 5.4 Dynamic Enterprise Schema Engine (`docs/04-dynamic-model/`)
Per-industry CRM models (Real Estate / Finance / Retail / B2B SaaS). Components:
- Meta-Schema Engine (JSON Schema storage).
- Field Customizer (add/edit/delete dynamic fields).
- Workflow Builder (status state machines).
- Validation Engine (runtime JSON-Schema validation).

### 5.5 Landing Page & Facebook CAPI (`docs/05-landing-capi/`)
- 100% content inherited from `chiase_cu/`, libs swapped to Tailwind + Lucide + Native Web Components.
- Extended tracking: UTM, fbclid, fbp, fbc, IP, UA, referrer, device fingerprint, scroll depth, time-on-page, heatmap.
- **Hybrid Dual-Tracking:** Meta Pixel (browser) + Conversions API (server).
- Anti-spoofing flow: `User → Landing Form → Go Ingest API → SHA-256 normalize → HMAC-SHA256(lead_id+fbclid+timestamp, Secret) → NATS queue → FB CAPI Worker → Meta Graph API`. Every event has UUIDv7 `event_id`. CRM state changes fire `Purchase`/`Custom_SQL_Event` back to FB.

### 5.6 Chat Real-time Engine (`docs/06-chat-engine/`)
- Stack: `Web/App → WebTransport/Zero-Copy WS → Kernel-Bypass Gateway (Rust + io_uring + eBPF/XDP) → FlatBuffers binary + Singleflight coalescing → Valkey 9.x (presence) + ScyllaDB (logs)`.
- Features: 1-1 / group / company channels, file via **Presigned S3 Direct Upload** (bypasses backend), reactions, threads, mentions, replies, custom emoji/stickers, typing indicator, read receipt, history search via Meilisearch, voice messages, offline push notifications.

### 5.7 WebRTC SFU & Recording (`docs/07-webrtc-sfu/`)
- **SFU:** Rust/C++, **Zero-Transcoding** with AV1/VP9 SVC, eBPF/XDP routing UDP/SRTP, BBR-WebRTC congestion control.
- **Recorder:** C++ + CUDA Egress Worker composites on GPU VRAM, NVENC direct encode → MinIO.
- **AI post:** Whisper.cpp STT + Llama-3 summarize.
- **Meeting features:** voice/video 1-1 and group, screen share 4K@60fps, background blur/virtual, cloud + local recording, live real-time caption, AI-extracted action items, calendar integration, waiting room, knock-to-join, 4-hour auto-end cap.

### 5.8 Observability 4 Layers + AI SRE (`docs/08-observability/`)
- **Layer 1 App-level:** UUIDv7 `trace_id`, structured JSON logs (Zap / `tracing`), error wrapping with context.
- **Layer 2 Telemetry Triad:** Logs via Vector → ClickHouse/Loki; Metrics via Prometheus/VictoriaMetrics; Traces via OpenTelemetry → Jaeger/Tempo; unified Grafana dashboard.
- **Layer 3 Realtime Alerting:** Sentry/GlitchTip aggregation, AI SRE (Code-LLM) RCA, multi-channel alerts (Telegram, Slack, PagerDuty, Twilio).
- **Layer 4 Self-Healing:** `gobreaker` circuit breakers, K3s liveness probe (< 2s restart), Valkey/SQLite local fallback when ClickHouse is down.

### 5.9 Multi-Tenant Security — Zero-Trust (`docs/09-security/`)
- **Network:** eBPF/XDP Anti-DDoS, JA4+ TLS fingerprint, Rate-Limit Map.
- **Frontend/Ingest:** Wasm Hardware Attestation (anti-headless), Argon2 Proof-of-Work.
- **Database:** PostgreSQL Row-Level Security, `SET LOCAL app.current_tenant_id`, Valkey ACL cache.
- **Admin:** **Dark Admin** (no DNS, no public IP), Single Packet Authorization via WireGuard, FIDO2/YubiKey + 2-of-3 Quorum for dangerous ops.
- **Kernel:** eBPF RASP blocks anomalous syscalls; auto BGP blackhole for attacker IPs.

### 5.10 Database Schema (`docs/10-database/`)
Multi-DB: Postgres 17 (CRM), ScyllaDB (chat/leads), ClickHouse (analytics), MongoDB (dynamic forms), Valkey (cache/presence), Qdrant + pgvector (RAG), Meilisearch (full-text), MinIO (media). Uses `goose` + `sqlc` for versioned SQL migrations; Ent for graph schemas; LTREE for tree paths.

### 5.11 AI Integration (`docs/11-ai-integration/`)
Three AI zones:
- **AI SRE & Code Guard:** DeepSeek-Coder / Llama-3-Code — auto RCA + hotfix-PR proposal (< 3s).
- **Predictive Analytics:** XGBoost / LightGBM / scikit-learn — lead scoring, pLTV, FB Optimizer (< 5ms).
- **Conversational & Media:** Whisper.cpp + Llama-3-70B + vLLM + LangChain + LlamaIndex + BGE-M3 embeddings — chatbot RAG, STT, meeting summary (first-token < 200ms).
- **Multi-Tenant AI Isolation:** tenant-scoped cache key `SHA256(tenant_id + prompt_prefix)`, GPU token-bucket admission control, Zero-Trust agent permission model.

---

## 6. Infrastructure

- **Local Dev:** Docker Compose profiles `core`, `ai`, `media`, `observability` — config at `infra/docker/docker-compose.yml`.
- **Production:** **K3s** cluster — 3-node HA control plane, auto-scaling edge nodes, **WireGuard overlay mesh** (Headscale), GPU pool (NVENC/TensorRT) for Recorder + AI.
- **Network VLANs:** Public (Edge + Anti-DDoS) • Private Service (microservices) • Data (DB/cache/storage) • Admin Mesh (WireGuard, Super Admin only).

---

## 7. Roadmap Phases

| Phase | Timeline | Scope (✅ done / ⬜ future) |
|-------|----------|-----------------------------|
| **Phase 1 — MVP** | Month 1–3 | ✅ Master design (this doc); ⬜ api-gateway, auth, tenant-manager; Landing Renderer (migrate from `chiase_cu/`); Landing Ingest + ScyllaDB; CRM Core (Lead/Contact/Deal); Super Admin skeleton; Tree-Org + PASETO invite; Docker Compose local |
| **Phase 2 — Core Features** | Month 4–6 | ⬜ Chat Engine (Rust + Scylla); WebRTC SFU + Recorder; FB CAPI Hybrid Dual-Tracking; AI Scoring (XGBoost); Notification; Analytics Dashboard (ClickHouse) |
| **Phase 3 — Full Multi-Tenant** | Month 7–9 | ⬜ Dynamic Schema Engine; Isolated VPS + WireGuard Mesh; Dark Admin + FIDO2; PostgreSQL RLS complete; Billing & Subscription |
| **Phase 4 — AI & Optimization** | Month 10–12 | ⬜ AI Conversation (vLLM + RAG); AI SRE (Code-LLM RCA); Whisper STT for meetings; GPU Composite Recording; advanced eBPF |

Detailed week-by-week plan with acceptance gates in `§19` of the master doc.

---

## 8. Cross-Cutting Concerns

### 8.1 Multi-Tenancy
- Isolation at 5 layers: eBPF Mesh, Gateway, Database (Postgres RLS), Admin Mesh, Kernel eBPF RASP.
- `tenant_id` is a `kebab-case` slug (e.g. `apex-fintech`); propagated via `trace_id` UUIDv7 and `SET LOCAL app.current_tenant_id`.
- Optional: shared cluster OR isolated VPS (customer-owned, joined via WireGuard mesh).
- Tenant-scoped AI cache keys + GPU token-bucket admission control prevent cross-tenant leakage.

### 8.2 Security (Zero-Trust)
- DDoS: eBPF/XDP at NIC, JA4+ fingerprint.
- Bot: Wasm Hardware Attestation + Argon2 PoW.
- Auth: PASETO v4 (stateless tokens), FIDO2/WebAuthn, AGE encryption.
- Admin: Dark Admin (no DNS/IP), WireGuard SPA, 2-of-3 Quorum + YubiKey for dangerous ops.
- DB: parameterized queries + Postgres RLS (defense in depth even against SQLi).
- Storage: MinIO presigned URLs include tenant prefix.
- Audit: 100% audit-log coverage requirement per phase gate.
- Insider threat: anomaly-detection AI on admin actions.

### 8.3 Observability
- 4-layer model: App → Telemetry Triad (Logs/Metrics/Traces) → Realtime Alerting → Self-Healing.
- Stack: Vector (logs) → ClickHouse/Loki; Prometheus/VictoriaMetrics; OpenTelemetry → Jaeger/Tempo; Grafana; Sentry/GlitchTip.
- Every request: UUIDv7 `trace_id` + structured JSON log + Prometheus metric.
- AI SRE auto-RCA + sandbox-tested auto-PR generation.

### 8.4 AI Integration
- 3 zones: AI SRE (DeepSeek-Coder), Predictive (XGBoost/LightGBM), Conversational+Media (vLLM + Whisper.cpp + Llama-3 + BGE-M3 + Qdrant).
- LLM serving on self-hosted A100 GPU pool (Llama-3 self-host preferred for cost & privacy).
- Whisper model: `medium` (best balance per Q5).
- Pipeline guardrails: strip system prompts, sandbox-test AI SRE PRs before merge, GPU token-bucket per tenant.

### 8.5 Testing & Quality
- **Pyramid:** 70% unit (testify / cargo / pytest) / 20% integration (Testcontainers / dockertest) / 10% E2E (Playwright + Cypress).
- **Coverage targets:** api-gateway/auth/landing-ingest ≥ 85–90%, chat-engine ≥ 80%, SFU/recorder ≥ 70%, ai-conversation ≥ 70%.
- **Load test:** k6 staging ramp 1K → 10K → 50K RPS, p99 < 50ms threshold.
- **Security scan:** OWASP ZAP baseline, sqlmap level-5 risk-3, `trivy` container scan.

### 8.6 Migration & DR
- Migration rules: backward-compatible, zero-downtime (< 30s), reversible (`up.sql` + `down.sql`), staging smoke 30 min, canary 10%→50%→100%.
- **RPO/RTO:** Postgres (RPO 5min / RTO 30min, WAL continuous), ScyllaDB (1h/2h, daily snapshot), ClickHouse (1h/4h), MinIO (0/1h cross-region replica), Valkey (0/5min AOF).
- 6 disaster scenarios: service crash, node fail, DB primary down, region fail, ransomware, bad migration — each with documented automated response.
- Quarterly DR drill (`tests/dr/drill-job.yaml`).

### 8.7 Cost & SLOs
- **Estimated cost:** ~$26,982/month (infra $21,440 + AI $4,642 + external $900) for 10K tenants / 1M MAU → ~$2.7/tenant/month, **90%+ cheaper** than HubSpot/Salesforce/Intercom+Zoom bundles.
- **Key SLOs:** api-gateway p99 < 20ms / 99.95% / error < 0.1%; landing-ingest p99 < 50ms, write > 100K/s; crm-core p99 read < 30ms, write < 50ms; chat-engine p99 < 50ms with > 100K concurrent per node; media-sfu p99 first-frame < 200ms, > 500 streams per node; ai-scoring p99 < 5ms; ai-conversation first-token < 200ms.
- **Frontend budget:** HTML < 50KB, CSS < 30KB, JS < 100KB, above-fold images < 200KB, total < 500KB; FCP < 0.4s / LCP < 0.8s / TTI < 1.5s / CLS < 0.05.
- **Error budget policy:** exceeding monthly budget → freeze deploys, hotfix-only.

### 8.8 Naming & Conventions
- Services `kebab-case` (`crm-core`); DB names `snake_case` (`rinco_crm`); NATS topics `domain.action` (`lead.created`); trace IDs UUIDv7; tenant IDs `kebab-case` slug.
- Git workflow: `main` / `develop` / `feature/<scope>-<short-desc>` / `hotfix/<scope>-<short-desc>`; mandatory commit+push.
- Code style: `gofumpt` + `golangci-lint` (Go) • `rustfmt` + `clippy` (Rust) • `eslint` + `prettier` + `tsc --strict` (TS).

### 8.9 Open Questions / Risks
- **Pending user decisions (Q1–Q10):** initial budget, ScyllaDB bare-metal vs cloud, Go 1.26 confirmed, hybrid cloud/on-prem, Whisper `medium`, self-hosted Llama-3, 90-day hot + 1y archive backup, LDAP/SSO in Phase 2, Flutter mobile, basic email marketing.
- **Architectural TBDs (Q11–Q20):** RLS vs DB-per-tenant, per-tenant rate limit, custom branding depth, CockroachDB vs Postgres, i18n, GDPR/PDPA level, notification priority, usage billing formula, Cloudflare Workers edge, VN data residency.
- **Technical TBDs (T1–T10):** HAProxy vs Envoy, ClickHouse cluster size, NATS JetStream storage backend, Meilisearch chosen over Typesense (Vietnamese), Headscale over Tailscale, VictoriaMetrics chosen, frontend monorepo (Turborepo/Nx) once apps > 3.
- **Top risks:** io_uring + Go GC conflict (mitigation: Rust gateway only); AI vendor lock-in (Llama-3 fallback); ScyllaDB operational complexity (hire DBA); dependency CVEs (Dependabot + Renovate weekly).

---

## 9. Companion Documents (deeper detail)

| # | Doc | Path |
|---|-----|------|
| 01 | Super Admin Portal | `docs/01-super-admin/README.md` |
| 02 | Tenant Company Site + VPS | `docs/02-tenant-site/README.md` |
| 03 | CRM Tree (hierarchy) | `docs/03-crm-tree/README.md` |
| 04 | Dynamic Model Engine | `docs/04-dynamic-model/README.md` |
| 05 | Landing Page + CAPI | `docs/05-landing-capi/README.md` |
| 06 | Chat Real-time Engine | `docs/06-chat-engine/README.md` |
| 07 | WebRTC SFU + Recording | `docs/07-webrtc-sfu/README.md` |
| 08 | Observability + AI SRE | `docs/08-observability/README.md` |
| 09 | Multi-Tenant Security | `docs/09-security/README.md` |
| 10 | Database Schema | `docs/10-database/README.md` |
| 11 | AI Integration | `docs/11-ai-integration/README.md` |

Master doc also catalogs **65 edge cases** (`§18`), **test pyramid + Go/TS load-test patterns** (`§20`), **goose up/down migration patterns + Strangler Fig decomposition** (`§21`), **quarterly DR drill job spec** (`§22.4`), **break-even analysis table** (`§23.6`), and **8-row risk register** (`§25.4`).

---

*Generated from `docs/00-master/README.md` v1.1 — comprehensive but concise summary for quick reference.*