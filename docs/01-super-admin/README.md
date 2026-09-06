# Phần 1 – Super Admin Portal

> **Phân hệ:** Quản trị tối cao của RINCO.  
> **Mục tiêu:** Admin giám sát toàn bộ tenants, tài khoản, log, traffic, doanh thu, mọi thứ trong thời gian thực (độ trễ 1 giây – 1 phút).  
> **Đặc thù:** Dark Admin (không DNS, không IP public), kết nối qua WireGuard Mesh + SPA.

---

## Mục lục
1. [Mục tiêu & Nguyên tắc](#1-mục-tiêu--nguyên-tắc)
2. [Kiến trúc](#2-kiến-trúc)
3. [Phân quyền & RBAC](#3-phân-quyền--rbac)
4. [Danh sách tính năng (≥ 100)](#4-danh-sách-tính-năng)
5. [Database Schema](#5-database-schema)
6. [API Surface](#6-api-surface)
7. [UI/UX & Components](#7-uiux--components)
8. [Realtime Pipeline](#8-realtime-pipeline)
9. [Observability cho Admin](#9-observability-cho-admin)
10. [Bảo mật Dark Admin](#10-bảo-mật-dark-admin)

---

## 1. Mục tiêu & Nguyên tắc

### 1.1. Mục tiêu
| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| ADM-1 | Giám sát real-time toàn hệ thống | p99 update < 1s |
| ADM-2 | CRUD tenants + cấu hình | Hoàn tất < 200ms |
| ADM-3 | Phát hiện bất thường sớm | Anomaly detected < 30s |
| ADM-4 | Không lộ public surface | Không DNS A/CNAME trỏ vào admin |
| ADM-5 | Audit log đầy đủ | 100% action có trace_id |

### 1.2. Nguyên tắc
- **Quan sát trước, hành động sau:** Mọi thay đổi phải có diff + audit log.
- **Multi-party cho thao tác nguy hiểm:** Xóa tenant, can thiệp gateway → cần 2-of-3 YubiKey.
- **Dark by default:** Không có URL truy cập trực tiếp, chỉ vào qua VPN.
- **Read-only by default:** Mỗi action đều có quyền tối thiểu (least privilege).

---

## 2. Kiến trúc

### 2.1. Sơ đồ thành phần
```
[Admin Browser] ──WireGuard Mesh + SPA──► [Admin Gateway :8891 (hidden)]
                                              │
                                              ├─► Auth (PASETO + FIDO2)
                                              ├─► Tenant Manager
                                              ├─► Feature Flags Service
                                              ├─► Resource Allocator (K3s API)
                                              ├─► Billing Service
                                              ├─► Audit Log Service
                                              ├─► Realtime Dashboard (SSE)
                                              └─► Impersonation Service
```

### 2.2. Service chính
- **`admin-gateway`** (Go + Echo): Hidden trên port 8891, chỉ bind trên WireGuard interface.
- **`admin-bff`** (TS + Bun): Tổng hợp dữ liệu cho Admin UI.
- **`tenant-manager`** (Go): CRUD tenant, provisioning.
- **`audit-log`** (Go + ScyllaDB): Ghi lại mọi thao tác admin.
- **`realtime-dashboard`** (Go + SSE): Push cập nhật mỗi 1s.

### 2.3. Cổng kết nối
| Port (internal) | Service | External |
|----------------|---------|----------|
| 8891 | admin-gateway | ❌ (chỉ WireGuard) |
| 8892 | admin-bff | ❌ |
| 8893 | audit-query | ❌ |
| 5432 | postgres-admin (PostgreSQL) | ❌ (private subnet) |
| 9090 | prometheus-admin | ❌ |

---

## 3. Phân quyền & RBAC

### 3.1. Vai trò Super Admin
| Role | Quyền |
|------|-------|
| **OWNER** | Toàn quyền, bao gồm billing, FIDO2 quản lý |
| **SRE_ADMIN** | Giám sát hạ tầng, restart service, scale |
| **SECURITY_ADMIN** | Khóa tenant, xem audit, ban IP |
| **SUPPORT_ADMIN** | Impersonate tenant (giới hạn 24h) |
| **FINANCE_ADMIN** | Billing, refund, subscription |
| **READONLY_VIEWER** | Xem dashboard, không sửa |

### 3.2. Multi-Party Authorization
- **2-of-3 Quorum:** Cho action nguy hiểm (xóa tenant, can thiệp gateway global, thay đổi DNS master).
- **Time Window:** 5 phút để 2 admin ký.
- **Audit:** Tất cả action có chữ ký + lý do bắt buộc.

### 3.3. FIDO2/YubiKey
- Mỗi Super Admin phải đăng ký ≥ 2 YubiKey (1 chính + 1 backup).
- WebAuthn challenge qua PASETO + signed nonce.

---

## 4. Danh sách tính năng (≥ 100)

### 4.1. Quản lý Tenant (1-15)
1. Tạo tenant mới (form: tên, slug, plan, region, template).
2. Khóa tenant (soft lock – vẫn cho user truy cập đọc).
3. Đóng băng tenant (freeze – không cho user truy cập).
4. Xóa tenant (hard delete sau grace period 30 ngày).
5. Di chuyển tenant giữa các cluster (zero-downtime).
6. Backup & Restore tenant config + DB.
7. Clone tenant (dùng làm template).
8. Cập nhật plan tenant (Free/Pro/Business/Enterprise).
9. Cấu hình feature flags theo tenant.
10. Cấu hình giới hạn (max user, max storage, max request/s).
11. Quản lý domain mapping (subpath, subdomain, custom domain).
12. Cấu hình SSL/TLS cho custom domain (auto-renew Let's Encrypt).
13. Whitelist IP cho admin tenant.
14. Cấu hình retention (số ngày giữ log, video, recording).
15. Export toàn bộ data tenant (GDPR).

### 4.2. Quản lý Tài khoản (16-30)
16. Tạo Super Admin (mời qua email + YubiKey).
17. Vô hiệu hóa Super Admin.
18. Reset password Super Admin (cần YubiKey xác nhận).
19. Khóa phiên (session kill) Super Admin.
20. Xem audit log cá nhân.
21. Phân quyền cho Super Admin mới.
22. Đăng ký YubiKey mới cho Super Admin.
23. Backup YubiKey.
24. Quản lý PASETO key rotation.
25. Quản lý WebAuthn challenge cache.
26. Danh sách phiên đang hoạt động.
27. Thiết bị đã đăng nhập (device fingerprint).
28. Cảnh báo đăng nhập từ IP/VPS lạ.
29. Yêu cầu thay đổi mật khẩu định kỳ.
30. Two-Factor backup codes.

### 4.3. Dashboard Realtime (31-55)
31. Tổng quan số liệu (tenant count, user count, revenue, traffic).
32. Graph traffic theo thời gian thực (1s refresh).
33. Top tenants theo traffic.
34. Top tenants theo doanh thu.
35. Heatmap lỗi toàn hệ thống.
36. World map traffic.
37. WebSocket connections count.
38. DB connection pool status.
39. Cache hit/miss ratio.
40. Disk usage per service.
41. RAM/CPU usage per node.
42. Network I/O per node.
43. GPU usage (nếu có AI workload).
44. p50/p95/p99 latency per endpoint.
45. Error rate per service (5xx, 4xx).
46. Request rate per second per service.
47. Active WebRTC meetings count.
48. Active chat channels count.
49. Chat message throughput.
50. Storage usage MinIO/S3.
51. ScyllaDB write rate.
52. ClickHouse query rate.
53. NATS JetStream lag.
54. Active background jobs.
55. Queue depth per service.

### 4.4. Logs & Traces (56-65)
56. Live tail log từng service (filter theo level).
57. Search log full-text (ClickHouse).
58. Saved searches (mỗi admin có query riêng).
59. Trace explorer (Jaeger UI embed).
60. Log retention config.
61. PII redaction trong log view.
62. Download log archive theo ngày.
63. Log shipping audit (ai xem log nào).
64. Alert rule từ log pattern.
65. Log correlation (group log theo trace_id).

### 4.5. Alerts & AI SRE (66-80)
66. Quản lý alert rules (CPU, RAM, error rate, latency).
67. Quản lý alert channels (Telegram, Slack, SMS, Email).
68. On-call schedule (P0, P1, P2 rotation).
69. Escalation policy.
71. Alert silence (mute trong khoảng thời gian).
72. AI RCA từ alert (Code-LLM auto-investigate).
73. AI Hotfix Proposal (tạo PR tự động).
74. AI weekly incident report.
75. AI capacity planning recommendation.
76. AI anomaly detection (bất thường traffic, error spike).
77. AI security threat scoring (per IP, per request).
78. AI cost optimization (gợi ý scale down/right-size).
79. AI tenant health score (per tenant).
80. Alert dedup & grouping (Sentry-style).

### 4.6. Resource & Deployment (81-90)
81. Xem danh sách cluster/node.
82. Restart service (rolling restart).
83. Scale service (manual hoặc theo rule).
84. Drain node (cho maintenance).
85. Cordon node.
86. Deploy config mới (GitOps/ArgoCD).
87. Rollback deployment.
88. K3s cluster health check.
89. WireGuard mesh status (peer online/offline).
90. Distributed resource sharing dashboard (VPS rảnh).

### 4.7. Billing & Subscription (91-100)
91. Tạo subscription cho tenant.
92. Trial period (14 ngày free).
93. Auto-renewal.
94. Manual refund.
95. Usage tracking (API calls, storage, AI calls).
96. Invoice generation.
97. Tax config (VAT theo quốc gia).
98. Discount codes.
99. Payment method (Stripe, VNPay, Momo).
100. Webhook billing event.

### 4.8. Nâng cao (101-120)
101. **Tenant Impersonation:** Đăng nhập hộ tenant (giới hạn 24h, audit đầy đủ).
102. **Mass Broadcast Notification:** Gửi thông báo tới mọi tenant (vd: bảo trì hệ thống).
103. **Audit Log Query Builder:** Tìm kiếm nâng cao theo actor, action, target, time range.
104. **Audit Log Export:** CSV/JSON cho compliance.
105. **GDPR Right-to-be-forgotten:** Xóa dữ liệu user cuối theo tenant.
106. **Data Residency:** Chọn region lưu trữ data (VN, SG, US).
107. **Webhook Outbound:** Admin đăng ký webhook nhận event.
108. **Webhook Incoming:** Nhận event từ hệ thống bên ngoài.
109. **API Key Management:** Cho phép tenant lấy API key.
110. **Rate Limit per Tenant API key.**
111. **Custom Branding:** Logo, màu sắc cho trang admin của từng tenant.
112. **Tenant Notes:** Ghi chú nội bộ về tenant.
114. **Status Page Internal:** Trang status private cho admin.
115. **Maintenance Mode:** Đặt hệ thống vào chế độ bảo trì.
116. **Read-Only Mode:** Khóa mọi write operation.
117. **Maintenance Window Scheduler.**
118. **Change Request Workflow:** Tạo CR cho thay đổi lớn.
119. **Approval Chain:** Multi-level approval cho action nguy hiểm.
120. **Slack/Teams/Telegram Bot Integration.**

### 4.9. Thông báo & Truyền thông nội bộ (121-130)
121. **Hệ thống thông báo tối ưu:** Push notification qua WebSocket + Email + Telegram Bot.
122. **Phân loại thông báo:** Critical, Warning, Info, Marketing.
123. **Template thông báo** có thể chỉnh sửa.
124. **Lên lịch gửi thông báo.**
125. **A/B test nội dung thông báo.**
126. **Đo lường tỷ lệ đọc/click thông báo.**
127. **Thông báo theo role/region/tenant group.**
128. **Thông báo khẩn cấp bypass quiet hours.**
129. **Digest mode:** Gộp thông báo theo ngày/tuần.
130. **In-app notification center** cho Admin UI.

---

## 5. Database Schema

### 5.1. Bảng `super_admins`
```sql
CREATE TABLE super_admins (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('OWNER','SRE_ADMIN','SECURITY_ADMIN','SUPPORT_ADMIN','FINANCE_ADMIN','READONLY_VIEWER')),
  status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','DISABLED','LOCKED')),
  mfa_enabled BOOLEAN DEFAULT false,
  last_login_at TIMESTAMPTZ,
  last_login_ip INET,
  failed_login_count INT DEFAULT 0,
  locked_until TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);
```

### 5.2. Bảng `super_admin_webauthn`
```sql
CREATE TABLE super_admin_webauthn (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  super_admin_id UUID REFERENCES super_admins(id) ON DELETE CASCADE,
  credential_id TEXT UNIQUE NOT NULL,
  public_key BYTEA NOT NULL,
  counter BIGINT NOT NULL DEFAULT 0,
  transports TEXT[],
  aaguid UUID,
  created_at TIMESTAMPTZ DEFAULT now(),
  last_used_at TIMESTAMPTZ
);
```

### 5.3. Bảng `tenants` (Master)
```sql
CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  plan TEXT NOT NULL CHECK (plan IN ('FREE','PRO','BUSINESS','ENTERPRISE')),
  status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','LOCKED','FROZEN','DELETING')),
  region TEXT NOT NULL,
  cluster_id TEXT,
  vps_node_id TEXT,
  isolation_mode TEXT CHECK (isolation_mode IN ('SHARED','ISOLATED')),
  max_users INT,
  max_storage_gb INT,
  max_requests_per_sec INT,
  feature_flags JSONB DEFAULT '{}',
  retention_days INT DEFAULT 90,
  custom_branding JSONB,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status);
```

### 5.4. Bảng `tenant_domains`
```sql
CREATE TABLE tenant_domains (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
  domain TEXT NOT NULL,
  routing_type TEXT CHECK (routing_type IN ('SUBPATH','SUBDOMAIN','CUSTOM')),
  target_path TEXT,
  ssl_status TEXT CHECK (ssl_status IN ('PENDING','ACTIVE','FAILED')),
  ssl_expires_at TIMESTAMPTZ,
  verified BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_tenant_domains ON tenant_domains(domain);
```

### 5.5. Bảng `audit_log` (phân tán sang ScyllaDB)
```
TABLE audit_log (
  id UUID,
  trace_id UUID,
  actor_id UUID,
  actor_email TEXT,
  actor_role TEXT,
  tenant_id UUID,
  action TEXT,             -- 'tenant.create', 'tenant.lock', ...
  target_type TEXT,
  target_id UUID,
  payload JSON,
  ip_address INET,
  user_agent TEXT,
  signature TEXT,          -- Chữ ký số từ YubiKey nếu có
  timestamp TIMESTAMP,
  PRIMARY KEY ((tenant_id), timestamp, id)
) WITH CLUSTERING ORDER BY (timestamp DESC);
```

### 5.6. Bảng `feature_flags`
```sql
CREATE TABLE feature_flags (
  flag_key TEXT PRIMARY KEY,
  description TEXT,
  default_enabled BOOLEAN DEFAULT false,
  rollout_percentage INT DEFAULT 0,
  tenant_overrides JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);
```

### 5.7. Bảng `quorum_requests`
```sql
CREATE TABLE quorum_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  action TEXT NOT NULL,
  payload JSONB NOT NULL,
  initiator_id UUID REFERENCES super_admins(id),
  required_signatures INT DEFAULT 2,
  collected_signatures JSONB DEFAULT '[]',
  status TEXT CHECK (status IN ('PENDING','APPROVED','REJECTED','EXPIRED')),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);
```

---

## 6. API Surface

### 6.1. Admin API (REST + Connect-RPC)
```
POST   /api/admin/v1/auth/login          # PASETO challenge
POST   /api/admin/v1/auth/webauthn/init
POST   /api/admin/v1/auth/webauthn/finalize
POST   /api/admin/v1/auth/logout
GET    /api/admin/v1/auth/sessions

GET    /api/admin/v1/tenants
POST   /api/admin/v1/tenants
GET    /api/admin/v1/tenants/:id
PATCH  /api/admin/v1/tenants/:id
DELETE /api/admin/v1/tenants/:id
POST   /api/admin/v1/tenants/:id/lock
POST   /api/admin/v1/tenants/:id/unlock
POST   /api/admin/v1/tenants/:id/freeze
POST   /api/admin/v1/tenants/:id/clone
POST   /api/admin/v1/tenants/:id/migrate

GET    /api/admin/v1/domains
POST   /api/admin/v1/tenants/:id/domains
DELETE /api/admin/v1/domains/:id
POST   /api/admin/v1/domains/:id/verify

GET    /api/admin/v1/admins
POST   /api/admin/v1/admins
PATCH  /api/admin/v1/admins/:id
POST   /api/admin/v1/admins/:id/disable
POST   /api/admin/v1/admins/:id/reset-password

GET    /api/admin/v1/dashboard/overview
GET    /api/admin/v1/dashboard/realtime      # SSE
GET    /api/admin/v1/dashboard/timeseries
GET    /api/admin/v1/dashboard/top-tenants

GET    /api/admin/v1/logs                    # Search
GET    /api/admin/v1/traces/:trace_id
GET    /api/admin/v1/metrics
GET    /api/admin/v1/alerts
POST   /api/admin/v1/alerts/:id/silence

POST   /api/admin/v1/quorum                  # Tạo yêu cầu quorum
POST   /api/admin/v1/quorum/:id/sign         # Ký duyệt
GET    /api/admin/v1/quorum/:id

POST   /api/admin/v1/impersonate/:tenant_id  # Bắt đầu impersonate
POST   /api/admin/v1/impersonate/end

GET    /api/admin/v1/audit
POST   /api/admin/v1/audit/export

GET    /api/admin/v1/billing/subscriptions
POST   /api/admin/v1/billing/refund
GET    /api/admin/v1/billing/usage
```

### 6.2. WebSocket/SSE Endpoints
- `/ws/admin/realtime` – Server-Sent Events cho dashboard.
- `/ws/admin/logs` – Live tail logs.

---

## 7. UI/UX & Components

### 7.1. Layout
```
┌──────────────────────────────────────────────────────┐
│  Top Bar: Logo │ Search │ Alerts │ Profile │ Logout  │
├──────────┬───────────────────────────────────────────┤
│ Sidebar  │  Main Content                              │
│ ▸ Tenants │  ┌─────────┬─────────┬─────────┐         │
│ ▸ Admins │  │ Card 1  │ Card 2  │ Card 3  │         │
│ ▸ Dash   │  └─────────┴─────────┴─────────┘         │
│ ▸ Logs   │  ┌─────────────────────────────────┐     │
│ ▸ Alerts │  │ Realtime Chart                  │     │
│ ▸ Audit  │  └─────────────────────────────────┘     │
│ ▸ Billing│  ┌────────────────┬───────────────┐     │
│ ▸ System │  │ Table         │   │     │
│ ▸ Quorum │  └────────────────┴───────────────┘     │
└──────────┴───────────────────────────────────────────┘
```

### 7.2. Tech Stack Frontend
- **React 18** + **Vite** + **TypeScript**.
- **shadcn/ui** + **Radix UI** primitives.
- **TanStack Query** cho server state.
- **Zustand** cho client state.
- **Recharts** + **Tremor** cho chart.
- **Tailwind CSS v4**.
- **Lucide Icons**.

### 7.3. Component Library (shadcn/ui style)
- DataTable (sortable, filterable, pagination, bulk action).
- ChartCard (Line, Bar, Area, Pie, Heatmap).
- LiveMetricCard (with sparkline).
- TenantDetailSheet (slide-over).
- AuditLogViewer (virtual scroll).
- ImpersonationBanner (warning top bar).
- QuorumDialog (multi-step với YubiKey).
- FeatureFlagMatrix.
- LogStream (xterm-style).
- AlertRuleBuilder.
- ResourceGauges.
- WorldMapTraffic.

### 7.4. Trang chính
1. **Dashboard Overview** – Card metrics + realtime chart.
2. **Tenants List** – Table với filter, search, bulk action.
3. **Tenant Detail** – Tab: Overview, Users, Domains, Billing, Resources, Audit.
4. **Admins List** – Table với role badges.
5. **Admin Detail** – Profile + Sessions + WebAuthn.
6. **Live Logs** – Terminal-style viewer.
7. **Traces Explorer** – Flamegraph + Span details.
8. **Metrics** – Grafana-style query builder.
9. **Alerts** – Active alerts + Rules.
10. **Audit Log** – Filterable table.
11. **Quorum Requests** – Pending + Approved + Rejected.
12. **Billing** – Subscriptions + Invoices + Usage.
13. **Resources** – Cluster + Nodes + Services.
14. **System** – Cluster health + Mesh status.
15. **Settings** – Global config, Branding, Notifications.

---

## 8. Realtime Pipeline

### 8.1. Data Flow
```
[ClickHouse metrics] ──┐
[Prometheus] ──────────┤
[ScyllaDB audit_log] ──┤
[Vector log agent] ───┤
                       ├──► [NATS JetStream topic: admin.metrics] ──► [realtime-dashboard] ──► [SSE] ──► [Admin UI]
[Service health] ──────┤
[K3s events] ──────────┤
[Valkey pub/sub] ──────┘
```

### 8.2. SSE Event Format
```typescript
type AdminEvent = 
  | { type: 'metric', service: string, name: string, value: number, ts: number }
  | { type: 'alert', severity: 'P0'|'P1'|'P2', message: string, link: string, ts: number }
  | { type: 'audit', actor: string, action: string, target: string, ts: number }
  | { type: 'tenant_status', tenant_id: string, status: string, ts: number };
```

### 8.3. Backpressure
- Max 1000 events/s per admin connection.
- Drop oldest if overflow, log to audit.
- Client có thể subscribe filter (chỉ nhận metric nhất định).

---

## 9. Observability cho Admin

### 9.1. Admin Health Endpoint
```
GET /admin/health
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": 3600,
  "db": "ok",
  "cache": "ok",
  "queue": "ok",
  "mesh": "ok",
  "deps": {
    "postgres": "ok",
    "clickhouse": "ok",
    "scylladb": "ok",
    "valkey": "ok",
    "nats": "ok"
  }
}
```

### 9.2. Metrics Export
- `/metrics` endpoint (Prometheus format).
- Metrics: `admin_requests_total`, `admin_audit_writes_total`, `admin_quorum_pending`, etc.

---

## 10. Bảo mật Dark Admin

### 10.1. Network Isolation
- **Không DNS record** cho admin subdomain.
- **Không IP public** – chỉ bind trên WireGuard interface (`wg0`).
- **Single Packet Authorization (SPA):** WireGuard phải handshake trước khi mở kết nối TCP.

### 10.2. Auth Flow
```
1. Admin mở SPA Tool → gửi gói tin SPA đến Gateway → nhận ephemeral port.
2. WireGuard tunnel lên ephemeral port.
3. PASETO challenge từ admin-gateway.
4. WebAuthn (YubiKey) → response signed.
6. Verify chữ ký → cấp session PASETO.
7. Cookie httpOnly + SameSite=Strict + Secure.
```

### 10.3. Audit
- Mọi request admin đều log vào ScyllaDB `audit_log`.
- Retention 5 năm cho P0 actions.
- Tìm kiếm nhanh theo actor, time range.

### 10.4. Rate Limiting
- 60 req/phút per IP.
- 1000 req/giờ per session.
- Auto-lock sau 5 lần fail WebAuthn.

---

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-ADM-01 | Tạo tenant mới trong < 5s | p95 |
| AC-ADM-02 | Realtime dashboard cập nhật < 1s | p95 |
| AC-ADM-03 | Tìm kiếm log < 100ms trên 1B row | ClickHouse |
| AC-ADM-04 | Audit log ghi đầy đủ 100% action | 100% |
| AC-ADM-05 | Không có DNS/IP public cho admin | Security scan |
| AC-ADM-06 | Quorum 2-of-3 hoạt động | Test scenario |

---

**Tiếp theo:** [`docs/02-tenant-site/README.md`](../02-tenant-site/README.md) – Thiết kế Tenant Company Site + Isolated VPS.