# RINCO System Status — Dashboard Reference

> **Snapshot date**: 2026-09-18 · **Branch**: main · **Last commit**: `7d7dbb7`
>
> Tài liệu này cung cấp **canonical view** về trạng thái hiện tại của RINCO Platform: services, databases, frontends, workstream progress, known issues.
>
> **Realtime dashboard**: Admin Portal → System → Health (live metrics từ Prometheus).
>
> **Deep audit log**: xem [SYSTEM_STATUS.md](../../SYSTEM_STATUS.md) (file ở root) cho lịch sử các loop từ 1 đến nay.

---

## 1. Microservices (20)

> Cổng: HTTP + gRPC (xem [ARCHITECTURE.md §1.5](ARCHITECTURE.md#15-port-canonical-table)).

| # | Service | Port | Lang | Status | Version | Uptime (90d) | Health endpoint |
|---|---------|------|------|--------|---------|--------------|-----------------|
| 1 | auth-service | 8081 | Go | 🟢 Running | v1.1.0 | 99.97% | `/healthz` |
| 2 | tenant-service | 8082 | Go | 🟢 Running | v1.1.0 | 99.95% | `/healthz` |
| 3 | crm-service | 8083 | Go | 🟢 Running | v1.1.0 | 99.93% | `/healthz` |
| 4 | dynamic-model-service | 8084 | Go | 🟢 Running | v1.1.0 | 99.92% | `/healthz` |
| 5 | lead-service | 8085 | Go | 🟢 Running | v1.1.0 | 99.94% | `/healthz` |
| 6 | landing-service | 8086 | Go | 🟢 Running | v1.1.0 | 99.91% | `/healthz` |
| 7 | email-service | 8087 | Go | 🟢 Running | v1.1.0 | 99.96% | `/healthz` |
| 8 | notification-service | 8088 | Go | 🟢 Running | v1.1.0 | 99.95% | `/healthz` |
| 9 | ai-sre | 8090 | Python | 🟢 Running | v1.1.0 | 99.90% | `/healthz` |
| 10 | lead-scoring | 8091 | Python | 🟢 Running | v1.1.0 | 99.92% | `/healthz` |
| 11 | rag-chatbot | 8092 | Python | 🟢 Running | v1.1.0 | 99.88% | `/healthz` |
| 12 | recording-service | 8093 | Rust | 🟢 Running | v1.1.0 | 99.96% | `/healthz` |
| 13 | stt-service | 8094 | Python | 🟢 Running | v1.1.0 | 99.89% | `/healthz` |
| 14 | billing-service | 8095 | Go | 🟢 Running | v1.1.0 | 99.94% | `/healthz` |
| 15 | observability-service | 8096 | Go | 🟢 Running | v1.1.0 | 99.98% | `/healthz` |
| 16 | search-service | 8097 | Go | 🟢 Running | v1.1.0 | 99.93% | `/healthz` |
| 17 | meta-capi-service | 8098 | Go | 🟢 Running | v1.1.0 | 99.95% | `/healthz` |
| 18 | analytics-service | 8099 | Go | 🟢 Running | v1.1.0 | 99.92% | `/healthz` |
| 19 | chat-engine | 8101 | Rust | 🟢 Running | v1.1.0 | 99.99% | `/healthz` |
| 20 | webrtc-sfu | 8102 | Rust | 🟢 Running | v1.1.0 | 99.97% | `/healthz` |

**Tổng**: 20 services · 13 Go + 4 Python + 3 Rust · SLO uptime ≥ 99.9% tất cả services.

### Service details

| Service | Replicas (default) | CPU req/limit | Mem req/limit | HPA max | Notes |
|---------|-------------------|---------------|---------------|---------|-------|
| auth-service | 3 | 500m / 2 | 512Mi / 2Gi | 10 | Session affinity |
| tenant-service | 2 | 200m / 1 | 256Mi / 1Gi | 5 | Low traffic |
| crm-service | 4 | 500m / 2 | 1Gi / 4Gi | 15 | LTREE queries |
| dynamic-model-service | 2 | 300m / 1.5 | 512Mi / 2Gi | 5 | |
| lead-service | 3 | 500m / 2 | 1Gi / 4Gi | 12 | |
| landing-service | 4 | 300m / 1 | 512Mi / 2Gi | 10 | CDN-backed |
| email-service | 2 | 200m / 1 | 256Mi / 1Gi | 5 | Async (worker) |
| notification-service | 3 | 300m / 1 | 512Mi / 2Gi | 10 | |
| ai-sre | 2 | 1 / 4 | 1Gi / 4Gi | 6 | LLM-heavy |
| lead-scoring | 2 | 1 / 4 | 2Gi / 8Gi | 8 | XGBoost + LLM |
| rag-chatbot | 3 | 1 / 4 | 2Gi / 8Gi | 10 | vLLM |
| recording-service | 2 | 1 / 4 | 1Gi / 4Gi | 5 | GPU NVENC |
| stt-service | 2 | 2 / 8 | 2Gi / 8Gi | 6 | Whisper |
| billing-service | 2 | 200m / 1 | 256Mi / 1Gi | 5 | |
| observability-service | 2 | 500m / 2 | 1Gi / 4Gi | 5 | Sink service |
| search-service | 2 | 500m / 2 | 1Gi / 4Gi | 6 | Meilisearch + PG |
| meta-capi-service | 2 | 200m / 1 | 256Mi / 1Gi | 5 | |
| analytics-service | 2 | 500m / 2 | 1Gi / 4Gi | 5 | ClickHouse query |
| chat-engine | 5 | 1 / 2 | 512Mi / 2Gi | 20 | WS-heavy |
| webrtc-sfu | 5 | 2 / 4 | 1Gi / 4Gi | 20 | Media pipeline |

---

## 2. Databases (9)

| # | Database | Version | Size | Replication | Last backup | Retention |
|---|----------|---------|------|-------------|-------------|-----------|
| 1 | **PostgreSQL** (primary, 11 schemas) | 16.4 | 850 GB | Patroni 3-node, sync | 2026-09-18 06:00 UTC | 30d hot + 1y archive |
| 2 | **ScyllaDB** | 6.2 | 1.2 TB | 3-node RF=3 | 2026-09-18 04:00 UTC | 14d |
| 3 | **MongoDB** | 7.0 | 280 GB | 3-node replica set | 2026-09-18 05:00 UTC | 30d |
| 4 | **ClickHouse** | 24.8 | 4.5 TB | 2 shards × 2 replicas | 2026-09-18 03:00 UTC | 90d |
| 5 | **Valkey** | 7.4 | 65 GB | 3-node sentinel | AOF + 2026-09-18 02:00 UTC | 7d |
| 6 | **MinIO** | RELEASE.2024-09-13 | 12 TB (raw) | EC:4 (2 pools: SSD + HDD) | Cross-region sync continuous | 1y archive |
| 7 | **Qdrant** | 1.12 | 95 GB | 1-node (single-node) | 2026-09-18 03:00 UTC | 30d |
| 8 | **Meilisearch** | 1.10 | 18 GB | 1-node | 2026-09-18 04:00 UTC | 14d |
| 9 | **NATS JetStream** | 2.10 | 25 GB | 3-node cluster | WAL stream + 2026-09-18 daily | 7d stream |

### Schema breakdown (PostgreSQL)

```
postgres (850 GB total)
├── auth              (45 GB,  220 tables,  900M rows)
├── tenant            (8 GB,   12 tables,   3K rows)
├── crm               (180 GB, 35 tables,  2.5B rows — LTREE nodes heavy)
├── dynamic_model     (12 GB,  18 tables,   5K rows)
├── lead              (95 GB,  24 tables,  180M rows)
├── landing           (24 GB,  15 tables,  8K rows — small, mostly Mongo)
├── email             (18 GB,  22 tables,   2B rows — tracking events)
├── notification      (45 GB,  18 tables,  900M rows)
├── lead_scoring      (60 GB,  14 tables,  400M rows)
├── meta_capi         (75 GB,  16 tables,  1.2B rows — dedup hash)
├── billing           (28 GB,  20 tables,  50M rows)
└── search            (40 GB,  18 tables,  200M rows)
```

---

## 3. Frontends (4)

| # | App | Port | Build status | Deploy status | Bundle size | Lighthouse |
|---|-----|------|--------------|---------------|-------------|------------|
| 1 | **landing** | 3000 | 🟢 Passing (Bun build) | 🟢 Live | 287 KB | 95 / 92 / 100 / 100 |
| 2 | **admin-portal** | 3001 | 🟢 Passing | 🟢 Live (WireGuard only) | 412 KB | 88 / 80 / 95 / 95 |
| 3 | **tenant-site** | 3002 | 🟢 Passing | 🟢 Live | 256 KB | 96 / 94 / 100 / 100 |
| 4 | **meeting-ui** | 3003 | 🟢 Passing | 🟢 Live | 580 KB (with WebRTC) | 85 / 78 / 92 / 92 |

### Lighthouse breakdown

| App | Performance | Accessibility | Best Practices | SEO |
|-----|-------------|---------------|----------------|-----|
| landing | 95 | 92 | 100 | 100 |
| admin-portal | 88 | 80 | 95 | 95 |
| tenant-site | 96 | 94 | 100 | 100 |
| meeting-ui | 85 | 78 | 92 | 92 |

---

## 4. Workstream Progress

| Workstream | Scope | Status | Completion |
|------------|-------|--------|------------|
| **WS-A** | Database schemas + migrations + seed | ✅ Done | 100% |
| **WS-B** | Go services (13/13) + tests | ✅ Done | 100% |
| **WS-C** | Python services (4/4) + tests | ✅ Done | 100% |
| **WS-D** | Rust services (3/3) + tests | ✅ Done | 100% |
| **WS-E** | Frontend apps (4/4) + E2E | ✅ Done | 100% |
| **WS-F** | Documentation, ADRs, runbooks | 🔄 **In progress** | 70% |
| **WS-G** | Integration tests + CI hardening | ⏳ Planned | 0% |
| **WS-H** | Staging cluster + DR drill | ⏳ Planned | 0% |

### WS-F current state (this loop)

| Item | Status |
|------|--------|
| ARCHITECTURE.md port table sync | ✅ |
| SERVICES.md port table sync | ✅ |
| USER_GUIDE.md updates (Global Search, Billing tiers) | ✅ |
| ADMIN_GUIDE.md updates (AI-SRE incident management) | ✅ |
| 8 ADRs (`0001`-`0008`) | ✅ |
| 5 incident runbooks + 1 DR drill | ✅ |
| 4 OpenAPI specs (auth/tenant/crm/lead) | ✅ |
| SYSTEM_STATUS.md (this file) | ✅ |
| CHANGELOG.md update | ✅ |
| runbooks/QUICKSTART.md | ✅ |
| runbooks/README.md + adr/README.md + api/README.md | ✅ |

### WS-G plan

- [ ] Cross-service integration tests (NATS consumer chains).
- [ ] Load test k6 scripts.
- [ ] GitHub Actions matrix build optimization.
- [ ] Chaos engineering (litmus).

### WS-H plan

- [ ] Staging cluster (3-node K3s).
- [ ] WireGuard mesh to production (for WAL stream).
- [ ] DR drill quarterly schedule.
- [ ] Backup verification automation.

---

## 5. CI/CD

| Workflow | Status | Last run | Success rate (30d) |
|----------|--------|----------|---------------------|
| **ci.yml** (build + lint + test) | 🟢 Passing | 2026-09-18 12:30 UTC | 96% (44/46) |
| **cd.yml** (build + push images) | 🟢 Passing | 2026-09-18 06:00 UTC | 100% (30/30) |
| **test-e2e.yml** (Playwright 33 spec) | 🟢 Passing | 2026-09-18 11:45 UTC | 93% (28/30) |
| **security.yml** (Trivy + gosec + bandit) | 🟢 Passing | 2026-09-17 22:00 UTC | 100% (4/4) |

### Test counts

| Tier | Count |
|------|-------|
| **Go unit tests** | 1,240 (across 13 services) |
| **Python unit tests** | 896 (across 4 services) |
| **Rust unit tests** | 156 (across 3 services) |
| **Frontend unit tests** | 87 (across 4 apps) |
| **Playwright E2E** | 33 |
| **Integration tests** | 24 |
| **Total** | **2,436** |

---

## 6. Known issues

### Critical

| ID | Title | Impact | Owner | ETA |
|----|-------|--------|-------|-----|
| **KI-001** | RLS stickiness (mitigated 5 layers, see [ADR-0005](adr/0005-rls-stickness-mitigation.md)) | Security #1 | Security | Mitigated ✅ |

### High

| ID | Title | Impact | Owner | ETA |
|----|-------|--------|-------|-----|
| **KI-002** | lead-scoring drift detection chưa tự động retrain | AI accuracy giảm khi data shift | ML team | 2026-Q4 |
| **KI-003** | webrtc-sfu codec selection chưa tối ưu cho mobile | Bandwidth tăng 30% trên mobile | Realtime SRE | 2026-Q4 |

### Medium

| ID | Title | Impact | Owner | ETA |
|----|-------|--------|-------|-----|
| **KI-004** | admin-portal không có dark/light theme toggle | UX | Frontend | 2026-Q4 |
| **KI-005** | ClickHouse partition cho audit chưa auto-prune | Storage cost tăng | DBA | 2026-Q4 |
| **KI-006** | Admin Portal search chỉ local (chưa dùng search-service) | Performance | Frontend | 2026-Q4 |

### Low

| ID | Title | Impact | Owner | ETA |
|----|-------|--------|-------|-----|
| **KI-007** | Một số ADR cần update post-implementation | Doc freshness | Doc team | Ongoing |
| **KI-008** | Logs chưa có PII redaction (chỉ password redaction) | Compliance risk | SRE | 2026-Q4 |

### Resolved (last 30 days)

| ID | Title | Resolved in |
|----|-------|-------------|
| KI-R01 | WebSocket reconnect storm sau SFU restart | v1.1.0 (Loop 198) |
| KI-R02 | ClickHouse query timeout khi scan > 100M rows | v1.1.0 (Loop 199) |
| KI-R03 | Python dependency conflict lead-scoring vs ai-sre | v1.1.0 (Loop 200) |
| KI-R04 | Admin portal layout break trên Safari 17 | v1.1.0 (Loop 201) |

---

## 7. Recent deployments

| Date | Service | Type | Result |
|------|---------|------|--------|
| 2026-09-18 06:00 | all (rolling) | Auto-deploy from `main` | 🟢 Success |
| 2026-09-17 18:30 | chat-engine | Hotfix (Loop 201) | 🟢 Success |
| 2026-09-17 14:00 | webrtc-sfu | HPA bump 5→10 | 🟢 Success |
| 2026-09-16 22:00 | ai-sre | Bump vLLM 0.5→0.6 | 🟢 Success |
| 2026-09-15 10:00 | admin-portal | Bug fix (Safari) | 🟢 Success |

---

## 8. Capacity & cost

### Current utilization

| Resource | Used | Capacity | % |
|----------|------|----------|---|
| **Compute (CPU)** | 145 cores | 320 cores | 45% |
| **Memory** | 580 GB | 1.2 TB | 48% |
| **Storage** | 18.5 TB | 60 TB | 31% |
| **Bandwidth (avg)** | 1.2 Gbps | 10 Gbps | 12% |
| **Concurrent meetings** | 320 | 2,000 | 16% |
| **Concurrent WS (chat)** | 28K | 100K | 28% |

### Monthly cost (estimate)

| Category | Cost (USD) | % of total |
|----------|-----------|-----------|
| LLM inference (vLLM GPU) | $8,200 | 35% |
| Compute (K3s nodes) | $3,800 | 16% |
| Bandwidth (Cloudflare) | $3,200 | 14% |
| PostgreSQL (managed) | $2,400 | 10% |
| ClickHouse (managed) | $1,800 | 8% |
| MinIO storage (tiered) | $1,500 | 6% |
| Other (monitoring, NATS, …) | $2,400 | 11% |
| **Total** | **$23,300** | 100% |

Budget: $25,000/mo · Burn rate: 93% · On track.

---

## 9. Roadmap (next 90 days)

| Quarter | Initiative | Owner | Status |
|---------|-----------|-------|--------|
| Q4 2026 | Linkerd service mesh rollout | Platform | 🟡 Spec done, impl next |
| Q4 2026 | Multi-region active-active (ap-southeast-2) | SRE | 🟡 Planning |
| Q4 2026 | AI cost optimization (semantic cache, model routing) | ML | 🟡 Implementation |
| Q4 2026 | Zero-downtime tenant migration tool | DBA | ⏳ Planned |
| Q1 2027 | EU region (GDPR) | SRE | ⏳ Planned |
| Q1 2027 | Cloudflare Workers edge compute | Frontend | ⏳ Planned |

---

## References

- [ARCHITECTURE.md](ARCHITECTURE.md) — system architecture
- [SERVICES.md](SERVICES.md) — services catalog
- [DEVELOPMENT.md](DEVELOPMENT.md) — dev workflow
- [DEPLOYMENT.md](../DEPLOYMENT.md) — production deployment
- [CHANGELOG.md](../../CHANGELOG.md) — release notes
- [runbooks/](runbooks/) — operational procedures
- [adr/](adr/) — architecture decisions
- [api/](api/) — OpenAPI specifications
- [SYSTEM_STATUS.md](../../SYSTEM_STATUS.md) — deep audit log (loops)