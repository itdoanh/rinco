# RINCO — Master Completion Plan

> **Date:** 2026-09-18  
> **Mode:** Docker Compose local stack, rich multi-tenant seed data, 3 parallel workstreams.

---

## 1. Bối cảnh & vì sao multi-day

Hệ thống RINCO đã có:
- 17 backend services (9 Go + 4 Python + 3 Rust + 1 meeting-ui gateway)
- 4 frontend apps (Next.js 15, App Router)
- 9 databases (PostgreSQL 6 schemas, ScyllaDB, MongoDB, ClickHouse, Valkey, MinIO, Qdrant, Meilisearch, NATS)
- 17 SQL seed files, 5 external seed bootstraps
- 12 design docs + DEV-PLAN tổng hợp (~37k dòng)

**Mục tiêu user yêu cầu:**
1. Hoàn thiện toàn hệ thống (cả backend, frontend, logic)
2. Xóa mock data trên frontend, thay bằng seed data thật vào mọi backend
3. Chạy tất cả backend + frontend + UI/UX đỉnh
4. Mọi tính năng hoạt động end-to-end

Đây là scope multi-day không thể 1 session xong — cần sub-agent chạy nền.

---

## 2. Workstreams song song (3 agents)

### WS-A: Backend services consolidation & E2E wiring
**Scope (priority HIGH):**
- Fix port mismatches giữa `SERVICES.md`, `docker-compose.services.yml`, và code thực tế
- Wire up cross-service auth (PASETO verification across services, không phải mỗi service tự validate)
- Hoàn thiện migration ordering + idempotent re-runs
- Test integration suite chạy được qua toàn bộ auth → tenant → crm → lead → scoring → notification flow
- Add dev script `make dev-all` hoặc `scripts/start-all.sh` để bring up toàn bộ stack
- Smoke test script `scripts/smoke.sh` verify health của 17 services + 4 frontends
- Tạo test data factories chung (multi-tenant, deterministic)

**Files to touch:**
- `infra/docker-compose.services.yml` — port sync
- `services/<each>/cmd/main.go` — port/env consistency
- `docs/SERVICES.md` — port table fix
- `scripts/dev.sh`, `scripts/smoke.sh` (new)
- `services/integration-tests/` — extend coverage
- `.env.example` — full env surface

### WS-B: Rich multi-tenant seed data
**Scope (priority HIGH):**
- Mở rộng 17 seed SQL files để có data phong phú cho 3 demo tenants đã có (`apexfintech`, `hct-consulting`, `demo-company`)
- Thêm tenants + users với LTREE hierarchy đầy đủ (Owner → Admin → Manager → Team Lead → Agent → Viewer)
- Tạo leads, deals, activities cho từng tenant (50+ leads, 10+ deals mỗi tenant)
- Activities, notes, tags, custom fields
- Workflows + audit logs
- Notifications
- Billing records
- External seeds: ClickHouse analytics rows, MinIO demo files, MongoDB enrichment docs, ScyllaDB chat/webrtc data, Valkey cache warmup
- Tạo `scripts/seed-all.sh` orchestrate tất cả
- Demo data cho AI services (lead-scoring features, RAG KB, STT transcripts)

**Files to touch:**
- `migrations/seed/00-15*.sql` — extend
- `seed/external/{clickhouse,minio,mongo,scylla,valkey}/` — extend
- `services/lead-scoring/seed_data.py`, `services/rag-chatbot/seed_data.py`, `services/ai-sre/seed_data.py`
- `services/stt-service/seed_data.py` (transcripts)
- `scripts/seed-all.sh` (new)
- `services/recording-service/seed_data.py` (mock S3 objects)

### WS-C: Frontend polish & real data integration
**Scope (priority HIGH):**
- Admin portal (3001): wire mọi page (analytics, audit, crm, dashboard, feature-flags, notifications, notification-templates, quorum, settings, system/{health,logs,metrics}, tenants, users) để gọi API thật, xóa hardcoded data
- Tenant site (3002): đa dạng trang per tenant, theme, i18n
- Landing (3000): form thật → POST /api/leads, CAPI + pixel tracking thật, multiple tenant routes
- Meeting UI (3003): WebRTC thật, screen share, chat side panel
- UI/UX polish: empty states, loading skeletons, error boundaries, dark mode, responsive
- Add E2E Playwright tests ở `frontend/e2e/` cho mỗi app
- Build verification toàn bộ: `bun run build` trên cả 4 apps

**Files to touch:**
- `frontend/admin-portal/app/(dashboard)/*/page.tsx` — real API calls
- `frontend/tenant-site/app/**` — multi-tenant rendering
- `frontend/landing/app/**` — real form submission + tracking
- `frontend/meeting-ui/app/**` — WebRTC integration
- `frontend/e2e/**` — Playwright tests
- `packages/frontend/ui/` — shared component polish

---

## 3. Chuẩn hoà cross-cutting (cả 3 workstreams)

- **Port table** (canonical):
  | service | port |
  |---|---|
  | auth-service | 8081 |
  | tenant-service | 8082 |
  | crm-service | 8083 |
  | dynamic-model-service | 8084 |
  | lead-service | 8085 |
  | landing-service | 8086 |
  | email-service | 8087 |
  | notification-service | 8088 |
  | billing-service | 8089 |
  | observability-service | 8090 |
  | search-service | 8091 |
  | ai-sre | 8092 |
  | stt-service | 8093 |
  | rag-chatbot | 8094 |
  | chat-engine | 8101 |
  | webrtc-sfu | 8102 |
  | recording-service | 8103 |
  | meta-capi-service | 8104 |
  | analytics-service | 8105 |
  | landing frontend | 3000 |
  | admin-portal | 3001 |
  | tenant-site | 3002 |
  | meeting-ui | 3003 |

- **Demo tenants**: `apexfintech`, `hct-consulting`, `demo-company` (đã có trong seed)
- **Auth bypass cho dev**: PASETO_KEY_DEV để verify cross-service
- **Test DBs**: `*_test` suffix cho mỗi service
- **Commit convention**: `Loop NNN: <workstream> <what>`

---

## 4. Definition of Done (DOD) — Phase 1 (sau khi 3 workstreams xong)

- [ ] `docker compose -f infra/docker-compose.yml up -d` → 17 services + 9 DBs healthy
- [ ] `bun run build` → 4 frontends OK
- [ ] `go test ./...` → all Go services PASS
- [ ] `pytest` → all Python services PASS
- [ ] `cargo test` → Rust services PASS
- [ ] `scripts/smoke.sh` → all 17 endpoints 200 OK
- [ ] `scripts/seed-all.sh` → 3 tenants with rich data
- [ ] Playwright E2E cho mỗi frontend → pass
- [ ] Tất cả commit đẩy lên `main` với message rõ ràng

---

## 5. Phase 2 (sau khi Phase 1 done) — bám sát DEV-PLAN

- Fix RLS stickiness (Critical #1)
- PASETO v4 cho invite links
- Workflow CEL evaluator
- Notification SSE streaming
- Lead bulk import worker
- Audit log background goroutine
- Tiered storage MinIO worker
- E2EE Signal Protocol trong chat-engine
- Chaos test drill scripts
- Cost optimization reports

---

## 6. Sub-agent contract

Mỗi sub-agent phải:
1. Work trong branch riêng (`ws-a-...`, `ws-b-...`, `ws-c-...`) hoặc commit trực tiếp lên main với prefix rõ ràng
2. Mỗi 30 phút commit 1 lần với message có loop number + workstream prefix
3. Trước khi commit, chạy `git pull --rebase` để tránh conflict
4. Cuối mỗi phase, update `docs/SYSTEM_STATUS.md` với tiến độ
5. Không động vào file của workstream khác (dùng port/tenant/file scoping)
6. Verify: `go build` cho Go touched, `bun run build` cho frontend touched, `pytest` cho Python touched

---

## 7. Conflict avoidance matrix

| Resource | WS-A owns | WS-B owns | WS-C owns |
|---|---|---|---|
| `infra/docker-compose.services.yml` | ✅ | | |
| `services/*/cmd/main.go` port | ✅ | | |
| `migrations/seed/**` | | ✅ | |
| `seed/external/**` | | ✅ | |
| `frontend/**` | | | ✅ |
| `services/integration-tests/**` | ✅ | | |
| `scripts/dev.sh`, `scripts/smoke.sh`, `scripts/seed-all.sh` | ✅ co-own | ✅ co-own | |
| `docs/SERVICES.md` (port table) | ✅ | | |
| `docs/SYSTEM_STATUS.md` (status) | ✅ | ✅ | ✅ |
| `docs/USER_GUIDE.md`, `docs/ADMIN_GUIDE.md` | | | ✅ |

Nếu workstream nào cần touch file của workstream khác → dừng lại, hỏi user trước khi sửa.

---

## 8. Tracking & reporting

Mỗi sub-agent báo cáo:
- `[Loop NNN] <workstream>: <one-line status>` mỗi ~30 min
- Phase done → final report với: files changed, lines added/removed, test results, git commit SHAs

Tổng hợp vào `docs/SYSTEM_STATUS.md` mỗi phase.
