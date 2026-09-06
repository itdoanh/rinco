# KẾ HOẠCH PHÁT TRIỂN DỰ ÁN RINCO (Development Plan)

> **Mục tiêu:** Tổng hợp toàn bộ kế hoạch phát triển dự án RINCO từ 12 tài liệu thiết kế chi tiết, sắp xếp theo trình tự logic triển khai (không bao gồm timeline cụ thể, chỉ thứ tự các bước).
>
> **Phạm vi:** 12 microservices chính + 9 databases + 7 skill files + 12 tài liệu thiết kế (~37,114 dòng).
>
> **Nguyên tắc:** Mỗi bước phải hoàn thành trước khi bước tiếp theo bắt đầu. Không parallelize những phần có dependency.

---

## MỤC LỤC

1. [Nguyên tắc & Phương pháp](#1-nguyên-tắc--phương-pháp)
2. [Tổng quan dự án](#2-tổng-quan-dự-án)
3. [Chặng 1: Foundation (Hạ tầng nền tảng)](#3-chặng-1-foundation-hạ-tầng-nền-tảng)
4. [Chặng 2: Identity & Tenancy](#4-chặng-2-identity--tenancy)
5. [Chặng 3: CRM Core](#5-chặng-3-crm-core)
6. [Chặng 4: Dynamic Model Engine](#6-chặng-4-dynamic-model-engine)
7. [Chặng 5: Landing Page & CAPI](#7-chặng-5-landing-page--capi)
8. [Chặng 6: Real-time Communication](#8-chặng-6-real-time-communication)
9. [Chặng 7: AI Integration](#9-chặng-7-ai-integration)
10. [Chặng 8: Observability & Security](#10-chặng-8-observability--security)
11. [Chặng 9: Tenant Site & Mesh](#11-chặng-9-tenant-site--mesh)
12. [Chặng 10: Super Admin](#12-chặng-10-super-admin)
13. [Chặng 11: Hardening & Polish](#13-chặng-11-hardening--polish)
14. [Acceptance Criteria tổng](#14-acceptance-criteria-tổng)
15. [Risks & Mitigations](#15-risks--mitigations)
16. [Final Checklist](#16-final-checklist)

---

## 1. NGUYÊN TẮC & PHƯƠNG PHÁP

### 1.1. Nguyên tắc phát triển
- **Build → Verify → Validate:** Mỗi bước đều có test pass trước khi next.
- **No silver bullet:** Mỗi component đều có phần riêng để đảm bảo chất lượng.
- **Production-ready from day 1:** Mọi code đều có observability, security, multi-tenant isolation.
- **Documentation-driven:** Mọi feature đều có doc cập nhật trước/song song với code.

### 1.2. Phương pháp triển khai
- **Vertical slice:** Triển khai 1 luồng end-to-end trước (e.g. tenant → user → lead → CAPI), rồi mới expand.
- **Progressive enhancement:** MVP trước, advanced features sau.
- **Trunk-based development:** Branch life time < 1 ngày, CI/CD tự động merge.
- **GitOps:** ArgoCD sync mọi state từ Git.

### 1.3. Definition of Done (DoD) cho mỗi phase
- [ ] Code merged vào main với full tests pass.
- [ ] Documentation cập nhật.
- [ ] Staging deployment thành công.
- [ ] Smoke test pass.
- [ ] Monitoring + alerting configured.
- [ ] Security review passed.
- [ ] Cost within budget.

---

## 2. TỔNG QUAN DỰ ÁN

### 2.1. 12 Microservices (theo docs)
| # | Service | Ngôn ngữ | Database | Doc |
|---|---------|----------|----------|-----|
| 1 | Master Design | - | - | `docs/00-master` |
| 2 | Super Admin Portal | Go + TypeScript | PostgreSQL | `docs/01-super-admin` |
| 3 | Tenant Site Service | Next.js + Rust | PostgreSQL | `docs/02-tenant-site` |
| 4 | CRM Tree Service | Go | PostgreSQL LTREE | `docs/03-crm-tree` |
| 5 | Dynamic Model Engine | Go + React Flow | PostgreSQL JSONB | `docs/04-dynamic-model` |
| 6 | Landing + CAPI | Next.js + Go | MongoDB + ScyllaDB | `docs/05-landing-capi` |
| 7 | Chat Engine | Rust (io_uring) | ScyllaDB + Valkey | `docs/06-chat-engine` |
| 8 | WebRTC SFU + Recording | Rust/C++ | ScyllaDB + MinIO | `docs/07-webrtc-sfu` |
| 9 | Observability Stack | Go + Rust + Python | ClickHouse + Prometheus | `docs/08-observability` |
| 10 | Security Stack | Go + Rust + eBPF | PostgreSQL RLS | `docs/09-security` |
| 11 | Database Layer | Multi | Polyglot | `docs/10-database` |
| 12 | AI Integration | Python + vLLM | Qdrant + MLflow | `docs/11-ai-integration` |

### 2.2. Tech Matrix
| Layer | Tech |
|-------|------|
| API Gateway | Envoy + custom Go plugin |
| Service Mesh | Linkerd (lightweight) |
| Backend | Go 1.23, Rust 1.81, Python 3.12, C++ (chỉ cho media) |
| Frontend | Next.js 15 (Bun), React 19, TypeScript 5.6 |
| Database | PostgreSQL 16, ScyllaDB 6.x, ClickHouse 24.x, MongoDB 7.x, Valkey 7.x, MinIO, Qdrant, Meilisearch |
| Cache/Queue | Valkey, NATS JetStream |
| Observability | Prometheus + VictoriaMetrics, Grafana, Loki/Vector → ClickHouse, Jaeger, Sentry |
| Security | eBPF/XDP, Wasm, Argon2, PASETO v4, PostgreSQL RLS, WireGuard |
| CI/CD | GitHub Actions, ArgoCD, K3s |
| AI | vLLM (Qwen2.5, DeepSeek-Coder, Whisper), MLflow, Qdrant |

---

## 3. CHẶNG 1: FOUNDATION (HẠ TẦNG NỀN TẢNG)

> **Mục tiêu:** Có được 1 cluster K3s + GitOps + 9 databases chạy ổn định, sẵn sàng cho services.

### Bước 1.1: Repository & Tooling Setup
- [ ] Tạo GitHub repo `itdoanh/rinco` (private).
- [ ] Setup `.gitignore`, `.editorconfig`, `.gitattributes`.
- [ ] Setup GitHub Actions với caching.
- [ ] Setup pre-commit hooks (golangci-lint, rustfmt, eslint).
- [ ] Setup CODEOWNERS.
- [ ] Tạo `.skills/` với SKILL.md + 6 rule files.
- [ ] Setup branch protection rules (require PR review, status checks).

### Bước 1.2: K3s Cluster
- [ ] Provision 3 master + 5 worker nodes (Hetzner/OVH).
- [ ] Install K3s v1.31+.
- [ ] Setup MetalLB cho LoadBalancer.
- [ ] Setup Longhorn cho storage.
- [ ] Setup cert-manager + Let's Encrypt.
- [ ] Setup Traefik ingress.
- [ ] Setup ArgoCD.
- [ ] Setup sealed-secrets.

### Bước 1.3: Database Cluster (theo `docs/10-database/`)
- [ ] **PostgreSQL:** Deploy 3 nodes với Patroni + PgBouncer.
  - Extensions: pgcrypto, ltree, pg_trgm, postgis.
  - Init schemas: tenants, users, leads, audit_logs, tenant_sites, workflows.
  - Configure RLS policies (sẽ apply đầy đủ ở Chặng 8).
- [ ] **ScyllaDB:** Deploy 5 nodes (RF=3).
  - Keyspaces: chat, webrtc, leads_raw, presence.
  - Tables: messages_by_conversation, conversations, presence_by_user, webrtc_sessions, leads_raw.
- [ ] **ClickHouse:** Deploy 3 nodes + Keeper (3 nodes).
  - Databases: rinco_logs, rinco_metrics, rinco_audit.
  - Tables: app_logs, audit_logs, metrics_samples, traces_spans.
  - Materialized views cho hourly aggregates.
- [ ] **MongoDB:** Deploy 3 nodes replica set.
  - Collections: leads (rich), enrichment_cache, fb_pixel_events.
  - Indexes: tenant_id + score, stage + score, enrichment.company.
- [ ] **Valkey:** Deploy 3 nodes cluster.
  - Use cases: presence, rate-limit, AI cache, session.
- [ ] **MinIO:** Deploy 4 nodes distributed.
  - Buckets: chat-attachments, recordings, landing-assets, model-artifacts.
- [ ] **Qdrant:** Deploy 2 nodes với replication.
  - Collections per tenant: `{tenant_id}_kb`.
- [ ] **Meilisearch:** Deploy 2 nodes.
  - Indexes: leads_search, kb_search.
- [ ] **NATS JetStream:** Deploy 3 nodes.
  - Streams: chat.events, lead.events, audit.events, crm.events.

### Bước 1.4: Backup & DR
- [ ] Setup WAL-G cho PostgreSQL → S3.
- [ ] Setup clickhouse-backup → S3.
- [ ] Setup ScyllaDB snapshots → S3.
- [ ] Setup MongoDB oplog → S3.
- [ ] Test restore từ backup (weekly drill).
- [ ] Document RPO/RTO cho mỗi DB.

### Bước 1.5: Domain & DNS
- [ ] Mua `rinco.app` + các subdomain.
- [ ] Setup DNS records: api, app, cdn, mail.
- [ ] Setup Cloudflare proxy + DDoS.
- [ ] Setup wildcard TLS.

**✅ Acceptance Chặng 1:**
- Cluster healthy, ArgoCD sync OK.
- Tất cả 9 databases up, có thể insert/query test record.
- Backup & restore verified cho mỗi DB.
- DNS resolve đúng, TLS OK.

---

## 4. CHẶNG 2: IDENTITY & TENANCY

> **Mục tiêu:** Có thể tạo tenant, user, đăng nhập an toàn với PASETO v4.

### Bước 2.1: Tenant Schema & Migration (PostgreSQL)
- [ ] Implement schema: tenants, users (cơ bản, chưa có LTREE path), audit_logs.
- [ ] Setup sqlc cho type-safe queries.
- [ ] Migration tooling: Atlas.
- [ ] Seed data: 1 super admin, 2 demo tenants.

### Bước 2.2: Auth Service (Go)
- [ ] Implement `pkg/auth/paseto.go` (đầy đủ với KeyRing rotation).
- [ ] Implement `pkg/auth/webauthn.go` (FIDO2 cho admin).
- [ ] Implement `pkg/auth/argon2.go` (PoW challenge).
- [ ] Endpoint `POST /auth/login` (email + password + MFA).
- [ ] Endpoint `POST /auth/refresh`.
- [ ] Endpoint `POST /auth/webauthn/begin` + `/finish`.
- [ ] Token revocation list (Valkey).
- [ ] Rate limiting (Valkey token bucket).

### Bước 2.3: Tenant Resolution Middleware
- [ ] Header `X-Tenant-ID` validation.
- [ ] Subdomain → tenant slug mapping.
- [ ] Custom domain → tenant_id (lookup cached in Valkey).
- [ ] Set PostgreSQL session vars (RLS ready).
- [ ] Audit log mọi request.

### Bước 2.4: User Management API (Go)
- [ ] CRUD users (chưa có LTREE, flat).
- [ ] Invite user (email + PASETO token).
- [ ] Verify invite, tạo user.
- [ ] Suspend / restore user.
- [ ] FIDO2 enrollment.

**✅ Acceptance Chặng 2:**
- Tạo được tenant mới.
- Login → nhận PASETO token.
- Refresh token hoạt động.
- Logout → token revoke.
- FIDO2 login cho admin.

---

## 5. CHẶNG 3: CRM CORE

> **Mục tiêu:** CRM với employee hierarchy (LTREE) + dynamic fields cơ bản.

### Bước 3.1: CRM Tree Service (theo `docs/03-crm-tree/`)
- [ ] Schema: thêm `path LTREE` cho users.
- [ ] Migration từ flat → tree.
- [ ] API: `POST /crm/users` (với parent).
- [ ] API: `GET /crm/users/{id}/subtree`.
- [ ] API: `POST /crm/users/{id}/promote`.
- [ ] API: `POST /crm/users/{id}/move` (move subtree).
- [ ] API: `GET /crm/users/{id}/path`.
- [ ] Invitation flow với PASETO.
- [ ] LTREE index (GIST + BTREE).

### Bước 3.2: Owner in subtree helper
- [ ] PostgreSQL function `owner_in_subtree(uuid)`.
- [ ] RLS policy dùng function.
- [ ] Test cross-tenant access (must fail).

### Bước 3.3: Lead Schema
- [ ] Schema: leads (cơ bản, custom_fields JSONB).
- [ ] Indexes: tenant_id, owner_user_id, score, stage.
- [ ] SQL function cho score band.
- [ ] CRUD leads API.

**✅ Acceptance Chặng 3:**
- Tạo được user với parent → đúng LTREE path.
- Promote / demote không break subtree.
- Move subtree atomically.
- Owner chỉ thấy lead của mình + subtree.

---

## 6. CHẶNG 4: DYNAMIC MODEL ENGINE

> **Mục tiêu:** Tenant có thể tự định nghĩa custom fields cho leads mà không cần restart service.

### Bước 4.1: Meta-Schema Service (theo `docs/04-dynamic-model/`)
- [ ] Schema: entity_field_defs, workflows, dynamic_validators.
- [ ] API: CRUD field definitions.
- [ ] API: Workflow definitions (CEL/JSONLogic).
- [ ] JSON Schema validator runtime.
- [ ] Type generation (TypeScript + Go).

### Bước 4.2: Dynamic Table Migration Tool
- [ ] Service detect column changes → generate migration.
- [ ] Apply migration online (zero-downtime).
- [ ] Rollback support.
- [ ] Backup trước khi migrate.

### Bước 4.3: Visual Designer UI (React Flow)
- [ ] Drag-and-drop field editor.
- [ ] Form preview.
- [ ] Workflow designer.
- [ ] Validation rule editor (CEL).
- [ ] Export → deploy.

### Bước 4.4: Workflow Engine
- [ ] Trigger detection (PostgreSQL CDC → NATS).
- [ ] Condition evaluation (CEL).
- [ ] Action execution (webhook, email, update).
- [ ] Error handling + retry.
- [ ] Audit log.

**✅ Acceptance Chặng 4:**
- Tạo được custom field mới → xuất hiện trong form ngay.
- Validation rules hoạt động.
- Workflow trigger khi điều kiện match.
- Migration zero-downtime (verified).

---

## 7. CHẶNG 5: LANDING PAGE & CAPI

> **Mục tiêu:** Migrate `chiase_cu/index.html` sang Next.js + thêm Facebook CAPI với Hybrid Dual-Tracking.

### Bước 5.1: Migration từ `chiase_cu` (theo `docs/05-landing-capi/`)
- [ ] Parse `chiase_cu/index.html`, list asset dependencies.
- [ ] Migrate DOM structure → Next.js page.
- [ ] Migrate inline JS → React components.
- [ ] Migrate inline CSS → Tailwind + CSS modules.
- [ ] Preserve 100% visual content.
- [ ] Setup Next.js 15 với Bun.
- [ ] Static export fallback.

### Bưỏi 5.2: Form Builder
- [ ] React Hook Form + Zod schema.
- [ ] Multi-step form.
- [ ] Anti-bot (Honeypot + Timing).
- [ ] Submission API → MongoDB + ScyllaDB.

### Bước 5.3: Tracking SDK
- [ ] Browser: pixel + GA4 + custom events.
- [ ] First-party cookies (subdomain).
- [ ] SHA-256 normalization (email, phone).
- [ ] Click ID capture (fbclid, gclid, ttclid).

### Bước 5.4: Ingestion API (Go)
- [ ] Endpoint `POST /v1/track` (server-side).
- [ ] HMAC verification.
- [ ] Idempotency key.
- [ ] Validate schema.
- [ ] Write to ScyllaDB `leads_raw` + MongoDB enrichment queue.

### Bước 5.5: CAPI Worker
- [ ] Đọc từ ScyllaDB `leads_raw` stream.
- [ ] Map event → Facebook CAPI format.
- [ ] HMAC-SHA256 cho user_data.
- [ ] Send batch tới Facebook.
- [ ] Circuit breaker cho Facebook API.
- [ ] Retry với exponential backoff.
- [ ] Deduplication logic.

### Bước 5.6: Wasm Attestation
- [ ] Compile `attestation/src/lib.rs` → WASM.
- [ ] Embed vào landing page.
- [ ] Server verify attestation.
- [ ] Argon2 PoW cho suspicious requests.

### Bước 5.7: Lead Scoring (basic)
- [ ] Rule-based scoring đầu tiên.
- [ ] Lưu score vào `leads.score`.
- [ ] Update score_band.

**✅ Acceptance Chặng 5:**
- Landing page render giống 100% `chiase_cu`.
- Form submit → lead tạo trong DB.
- Pixel + CAPI match events (dedup test).
- Wasm attestation block headless Chrome.
- Lighthouse score ≥ 95.

---

## 8. CHẶNG 6: REAL-TIME COMMUNICATION

> **Mục tiêu:** Chat engine + WebRTC SFU + recording.

### Bước 6.1: Chat Engine (theo `docs/06-chat-engine/`)
- [ ] Rust service với io_uring + FlatBuffers.
- [ ] WebSocket gateway.
- [ ] ScyllaDB schema: messages_by_conversation.
- [ ] Valkey presence.
- [ ] NATS cho fan-out.
- [ ] Typing indicator, read receipts.
- [ ] File upload (MinIO).
- [ ] E2E encryption (Signal Protocol - optional MVP).

### Bước 6.2: WebRTC SFU (theo `docs/07-webrtc-sfu/`)
- [ ] Rust SFU với mediasoup.
- [ ] AV1/VP9 SVC.
- [ ] Coturn TURN server.
- [ ] DTLS-SRTP.
- [ ] ICE negotiation.
- [ ] Bandwidth estimation.

### Bước 6.3: Recording Service
- [ ] GPU node với NVIDIA NVENC.
- [ ] FFmpeg + CUDA cho composite.
- [ ] Upload tới MinIO.
- [ ] Playback API.

### Bước 6.4: Meeting UI (Next.js)
- [ ] Prejoin screen.
- [ ] Media controls.
- [ ] Chat sidebar.
- [ ] Screen share.
- [ ] Recording indicator.

**✅ Acceptance Chặng 6:**
- 2 người chat real-time.
- Group chat 10 người.
- 1-1 video call thành công.
- Group meeting 5 người ổn định.
- Recording play lại đúng.

---

## 9. CHẶNG 7: AI INTEGRATION

> **Mục tiêu:** Lead Scoring ML + RAG chatbot + AI SRE + STT/TTS.

### Bước 7.1: AI Infrastructure (theo `docs/11-ai-integration/`)
- [ ] Setup GPU nodes (Hetzner + Lambda).
- [ ] Deploy vLLM cluster.
- [ ] Deploy Qdrant (vector DB).
- [ ] Setup MLflow tracking server.

### Bước 7.2: Lead Scoring (ML)
- [ ] Train default model (LightGBM) với historical data.
- [ ] Deploy Go service `/score`.
- [ ] Periodic retrain (weekly).
- [ ] Drift detection.

### Bước 7.3: AI Gateway (Go)
- [ ] Rate limiting per tenant.
- [ ] Cost tracking per request.
- [ ] Budget enforcement.
- [ ] Multi-model fallback.

### Bước 7.4: RAG Chatbot
- [ ] Embedding service (multilingual-e5-large).
- [ ] Ingest KB documents → Qdrant.
- [ ] Retrieve + Generate pipeline.
- [ ] Source citation.
- [ ] Conversation history.

### Bước 7.5: AI SRE
- [ ] ClickHouse query helper.
- [ ] GitHub API integration.
- [ ] Auto-PR generation.
- [ ] Telegram bot cho alert.
- [ ] RCA report format.

### Bước 7.6: STT/TTS
- [ ] faster-whisper service (Vietnamese).
- [ ] VITS Vietnamese TTS.
- [ ] Real-time streaming.

### Bước 7.7: CAPI Optimization AI
- [ ] Analyze pixel + CAPI match rate.
- [ ] Suggest better event mapping.
- [ ] Auto-tune event deduplication window.

**✅ Acceptance Chặng 7:**
- Lead scoring AUC > 0.85.
- RAG answer relevant, cite source.
- AI SRE RCA < 3s.
- Whisper WER < 10% (Vietnamese).
- Cost per AI request < $0.01.

---

## 10. CHẶNG 8: OBSERVABILITY & SECURITY

> **Mục tiêu:** Zero-Trust security + 4-tier observability + self-healing.

### Bước 8.1: Observability (theo `docs/08-observability/`)
- [ ] Structured logging cho mọi service (Zap/tracing).
- [ ] Prometheus + VictoriaMetrics.
- [ ] OpenTelemetry tracing end-to-end.
- [ ] ClickHouse cho logs/audit.
- [ ] Sentry / GlitchTip.
- [ ] Grafana dashboards (5 dashboard per service).
- [ ] Alertmanager rules (P0/P1/P2).

### Bước 8.2: Security Hardening (theo `docs/09-security/`)
- [ ] PostgreSQL RLS policies (force cho tất cả tables).
- [ ] PASETO rotation script.
- [ ] Argon2 PoW server.
- [ ] Wasm attestation integration.
- [ ] Rate limiting middleware.
- [ ] CORS, CSP, HSTS.

### Bước 8.3: DDoS Protection
- [ ] Compile + load eBPF/XDP program.
- [ ] Userspace controller (Go).
- [ ] Test với 1M pps.
- [ ] Per-IP blacklist / whitelist.

### Bước 8.4: eBPF RASP
- [ ] Syscall filter (block reverse shell).
- [ ] File integrity monitoring.
- [ ] Process whitelist.

### Bước 8.5: Dark Admin
- [ ] WireGuard mesh deployment.
- [ ] Admin SPA (Go + Wails hoặc Tauri).
- [ ] No public DNS cho admin.
- [ ] FIDO2 only login.

### Bước 8.6: Circuit Breaker + Self-healing
- [ ] sony/gobreaker cho mỗi external call.
- [ ] K3s liveness/readiness probes.
- [ ] Auto-rollback Argo Rollouts.
- [ ] Graceful degradation.

**✅ Acceptance Chặng 8:**
- 100% cross-tenant RLS block.
- DDoS 10Gbps block < 50% CPU.
- Self-healing MTTR < 30s.
- Alert < 30s cho P1.
- AI SRE RCA accuracy > 70%.

---

## 11. CHẶNG 9: TENANT SITE & MESH

> **Mục tiểu:** Tenant sites deployable isolated (VPS), communicate qua WireGuard mesh.

### Bước 9.1: Tenant Site Service (theo `docs/02-tenant-site/`)
- [ ] Next.js multi-tenant routing (subpath + subdomain + custom domain).
- [ ] Theme + branding per tenant.
- [ ] CMS cho pages.
- [ ] SEO settings.

### Bước 9.2: Mesh Coordinator (Rust)
- [ ] WireGuard peer management.
- [ ] Health check giữa các VPS.
- [ ] Resource scheduler.
- [ ] Auto-failover.

### Bước 9.3: Distributed Resource Sharing (Python)
- [ ] Predict tenant load.
- [ ] Auto-scale VPS.
- [ ] Cost optimization.

### Bước 9.4: VPS Provisioning
- [ ] Ansible playbook cho VPS setup.
- [ ] Terraform cho cloud resources.
- [ ] Auto-register vào mesh.

**✅ Acceptance Chặng 9:**
- Tenant site render với custom domain.
- 1 tenant có thể chạy dedicated VPS.
- WireGuard mesh auto-heal.
- Resource scheduler balance.

---

## 12. CHẶNG 10: SUPER ADMIN

> **Mục tiêu:** Super admin portal oversee tất cả tenants + system metrics + billing.

### Bước 10.1: Super Admin Backend (theo `docs/01-super-admin/`)
- [ ] Admin Gateway (Go) chạy trên WireGuard interface only.
- [ ] Tenant CRUD.
- [ ] Resource quota management.
- [ ] Billing engine.
- [ ] System health dashboard.

### Bước 10.2: Super Admin Frontend (Next.js)
- [ ] Admin dashboard.
- [ ] Tenant list với filters.
- [ ] Metrics visualization.
- [ ] Billing reports.

### Bước 10.3: Dark Admin Tool (Tauri + Go)
- [ ] Desktop app cho super admin.
- [ ] Local-first.
- [ ] WireGuard auto-connect.
- [ ] FIDO2 authentication.

### Bước 10.4: 2-of-3 Quorum
- [ ] Implement quorum service.
- [ ] Critical actions cần 2 admin ký.
- [ ] Time-bound approval.
- [ ] Audit log.

**✅ Acceptance Chặng 10:**
- Super admin login với FIDO2 + WireGuard.
- Tenant CRUD với audit log.
- Critical action (e.g. delete tenant) cần 2 admin.
- Dark Admin không accessible từ public IP.

---

## 13. CHẶNG 11: HARDENING & POLISH

> **Mục tiêu:** Production-ready với chaos engineering, DR drill, optimization.

### Bước 11.1: Chaos Engineering
- [ ] Deploy Chaos Mesh.
- [ ] Game day drills hàng tuần.
- [ ] Test failover cho từng service.
- [ ] Document runbooks.

### Bước 11.2: DR Drill
- [ ] Restore từ backup → verify data integrity.
- [ ] Cross-region failover test.
- [ ] RPO/RTO measurement.
- [ ] Document incident response.

### Bước 11.3: Performance Optimization
- [ ] Lighthouse score ≥ 95 cho landing.
- [ ] API p99 < 200ms.
- [ ] DB query optimization.
- [ ] CDN setup cho static assets.

### Bước 11.4: Compliance
- [ ] GDPR right-to-be-forgotten flow.
- [ ] Data export flow.
- [ ] Audit log retention policy.
- [ ] Penetration test (annual).

### Bước 11.5: Documentation Finalization
- [ ] API documentation (OpenAPI).
- [ ] Runbooks cho mỗi service.
- [ ] Architecture decision records (ADR).
- [ ] Onboarding guide.

### Bước 11.6: Cost Optimization
- [ ] Reserved instance cho stable workloads.
- [ ] Spot instance cho AI training.
- [ ] Storage tiering (hot/cold).
- [ ] Bandwidth optimization.

**✅ Acceptance Chặng 11:**
- Pass 5 chaos game day drills.
- RPO/RTO đạt target.
- Lighthouse 95+.
- Penetration test no P0/P1.
- Cost trong budget.

---

## 14. ACCEPTANCE CRITERIA TỔNG

### 14.1. Functional (theo từng doc)

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-OVERALL-01 | 12 services deploy và healthy | K3s status |
| AC-OVERALL-02 | 100% cross-tenant isolation | Pen test |
| AC-OVERALL-03 | 100% features theo `yeucauthietke.md` | Manual review |

### 14.2. Performance

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-PERF-01 | API p99 < 200ms | Load test |
| AC-PERF-02 | Chat latency < 50ms | E2E test |
| AC-PERF-03 | WebRTC latency < 200ms | WebRTC stats |
| AC-PERF-04 | Landing Lighthouse ≥ 95 | Lighthouse CI |
| AC-PERF-05 | Lead scoring < 5ms | p99 |
| AC-PERF-06 | RAG first token < 200ms | p95 |
| AC-PERF-07 | AI SRE RCA < 3s | p95 |
| AC-PERF-08 | Whisper STT < 200ms | p99 |

### 14.3. Reliability

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-REL-01 | Uptime ≥ 99.9% | Uptime check |
| AC-REL-02 | RPO < 5 phút | DR drill |
| AC-REL-03 | RTO < 30 phút | DR drill |
| AC-REL-04 | Self-healing MTTR < 30s | Chaos test |

### 14.4. Security

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-SEC-01 | 0 P0/P1 vulnerability | Pen test |
| AC-SEC-02 | DDoS resistance 10Gbps | Load test |
| AC-SEC-03 | FIDO2 admin only | E2E |
| AC-SEC-04 | eBPF block reverse shell | Chaos test |
| AC-SEC-05 | Dark admin nmap clean | Security scan |

---

## 15. RISKS & MITIGATIONS

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **GPU shortage** | Medium | High | Spot instances + multi-cloud |
| **LLM API cost overrun** | High | High | Self-host + budget alert |
| **ScyllaDB operation complexity** | Medium | High | Training + monitoring + DR |
| **eBPF kernel compatibility** | Low | High | Version matrix + fallback |
| **PASETO rotation bug** | Low | High | Overlap period + rollback |
| **RLS bypass** | Low | Critical | Audit + test cross-tenant |
| **Model drift** | High | Medium | Daily retrain + A/B test |
| **ClickHouse merge storm** | Medium | Medium | Parts threshold tuning |
| **Wasm browser compat** | Low | Medium | Polyfill + fallback |
| **TURN bandwidth cost** | High | High | SFU + selective forwarding |

---

## 16. FINAL CHECKLIST

### Trước khi go-live

- [ ] Tất cả 12 docs có version mới nhất.
- [ ] Tất cả 11 chặng đã hoàn thành.
- [ ] Tất cả acceptance criteria đạt.
- [ ] DR drill pass.
- [ ] Security review pass.
- [ ] Cost budget OK.
- [ ] Team trained.
- [ ] On-call rotation setup.
- [ ] Runbooks documented.
- [ ] Stakeholder demo done.

### Sau khi go-live (week 1)

- [ ] Monitor error rate.
- [ ] Monitor cost.
- [ ] Monitor latency.
- [ ] Daily standup.
- [ ] Bug bash.
- [ ] User feedback collection.

### Sau khi go-live (month 1)

- [ ] Performance review.
- [ ] Cost optimization.
- [ ] Feature roadmap planning.
- [ ] Retrospective.

---

## PHỤ LỤC: LIÊN KẾT TÀI LIỆU

| Doc | Path | Lines |
|-----|------|-------|
| Master | `docs/00-master/README.md` | 1385 |
| Super Admin | `docs/01-super-admin/README.md` | 1614 |
| Tenant Site | `docs/02-tenant-site/README.md` | 1838 |
| CRM Tree | `docs/03-crm-tree/README.md` | 2089 |
| Dynamic Model | `docs/04-dynamic-model/README.md` | 4508 |
| Landing + CAPI | `docs/05-landing-capi/README.md` | 5888 |
| Chat Engine | `docs/06-chat-engine/README.md` | 5400 |
| WebRTC SFU | `docs/07-webrtc-sfu/README.md` | 5789 |
| Observability | `docs/08-observability/README.md` | 1534 |
| Security | `docs/09-security/README.md` | 1516 |
| Database | `docs/10-database/README.md` | 2573 |
| AI Integration | `docs/11-ai-integration/README.md` | 2780 |
| **Dev Plan** | **`docs/DEV-PLAN.md`** | **This file** |

**Tổng tài liệu:** ~37,114 dòng.

---

## PHỤ LỤC: TECH STACK FINAL

### Backend
- Go 1.23 (services chính: API, ingestion, CRM, dynamic model, admin, security)
- Rust 1.81 (chat engine, WebRTC SFU, mesh coordinator, eBPF tooling)
- Python 3.12 (AI/ML pipelines, RAG, STT/TTS, AI SRE)
- C++ (chỉ cho media processing, NVENC binding)

### Frontend
- Next.js 15 (App Router, RSC) + Bun runtime
- React 19, TypeScript 5.6
- Tailwind CSS + shadcn/ui
- React Flow (cho dynamic model designer)

### Database (Polyglot)
- PostgreSQL 16 (RLS, LTREE, JSONB)
- ScyllaDB 6.x (chat, leads_raw)
- ClickHouse 24.x (logs, traces, audit OLAP)
- MongoDB 7.x (lead enrichment, flexible schema)
- Valkey 7.x (cache, presence, rate limit, AI cache)
- MinIO (S3-compatible, attachments, recordings)
- Qdrant (vector DB cho RAG)
- Meilisearch (full-text search tiếng Việt)

### Infrastructure
- K3s (Kubernetes distribution)
- ArgoCD (GitOps)
- Longhorn (storage)
- MetalLB (LoadBalancer)
- cert-manager (TLS)
- Cloudflare (CDN + DDoS)
- WireGuard + Headscale (mesh VPN)

### Observability
- Prometheus + VictoriaMetrics (metrics)
- Grafana (visualization)
- Vector → ClickHouse (logs)
- OpenTelemetry + Jaeger (tracing)
- Sentry / GlitchTip (errors)
- Alertmanager + Telegram + PagerDuty (alerting)
- vLLM (AI SRE LLM)

### AI/ML
- vLLM (LLM serving)
- faster-whisper (STT)
- VITS (TTS)
- MLflow (experiment tracking)
- LightGBM (lead scoring)
- Qdrant (vector store)

---

## KẾT LUẬN

Tài liệu này tổng hợp toàn bộ kế hoạch phát triển dự án RINCO từ 12 tài liệu thiết kế chi tiết. Thứ tự các bước đã được sắp xếp để đảm bảo:

1. **Foundation trước, features sau.** Không build feature khi infra chưa sẵn sàng.
2. **Identity & tenancy trước, CRM sau.** Multi-tenant là core, phải có từ đầu.
3. **Core features trước, advanced features sau.** MVP trước, polish sau.
4. **Observability + Security song song.** Zero-Trust + 4-tier observability apply từ Chặng 2.
5. **Hardening cuối cùng.** Chaos + DR + optimization là last mile.

**Mỗi chặng đều có acceptance criteria riêng, không skip được.**

Khi đã sẵn sàng, đội ngũ phát triển sẽ đi theo thứ tự này, commit code hàng ngày, demo cuối mỗi chặng.
