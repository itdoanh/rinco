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

**Mục lục mở rộng (phần bổ sung – v1.1)**

11. [Audit – Đánh giá nội dung hiện tại](#11-audit--đánh-giá-nội-dung-hiện-tại)
12. [Edge Cases & Error Scenarios chi tiết](#12-edge-cases--error-scenarios-chi-tiết)
13. [Code Examples chi tiết](#13-code-examples-chi-tiết)
14. [Implementation Roadmap chi tiết](#14-implementation-roadmap-chi-tiết)
15. [Testing Strategy](#15-testing-strategy)
16. [Migration Plan](#16-migration-plan)
17. [Disaster Recovery](#17-disaster-recovery)
18. [Cost Estimation](#18-cost-estimation)
19. [Open Questions / Cần user xác nhận](#19-open-questions--cần-user-xác-nhận)

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

### 5.8. Bảng `notification_templates` (MỚI – v1.1)
```sql
CREATE TABLE notification_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code TEXT UNIQUE NOT NULL,            -- 'maintenance.scheduled', 'billing.invoice_failed'
  category TEXT NOT NULL CHECK (category IN ('CRITICAL','WARNING','INFO','MARKETING')),
  channel TEXT NOT NULL CHECK (channel IN ('IN_APP','EMAIL','TELEGRAM','PUSH','SMS')),
  subject TEXT NOT NULL,
  body_template TEXT NOT NULL,          -- Go text/template
  variables JSONB NOT NULL DEFAULT '[]', -- Schema: [{name, type, required}]
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_notification_templates_code ON notification_templates(code);
```

### 5.9. Bảng `notification_dispatch_log` (MỚI – v1.1)
```sql
CREATE TABLE notification_dispatch_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id UUID REFERENCES notification_templates(id),
  recipient_type TEXT NOT NULL CHECK (recipient_type IN ('SUPER_ADMIN','TENANT','USER')),
  recipient_id UUID,
  channel TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('PENDING','SENT','FAILED','BOUNCED')),
  error_message TEXT,
  trace_id UUID,
  sent_at TIMESTAMPTZ DEFAULT now(),
  delivered_at TIMESTAMPTZ,
  opened_at TIMESTAMPTZ,
  clicked_at TIMESTAMPTZ
);
CREATE INDEX idx_dispatch_recipient ON notification_dispatch_log(recipient_type, recipient_id, sent_at DESC);
```

### 5.10. Bảng `impersonation_sessions` (MỚI – v1.1)
```sql
CREATE TABLE impersonation_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  admin_id UUID REFERENCES super_admins(id),
  tenant_id UUID REFERENCES tenants(id),
  reason TEXT NOT NULL,                -- Lý do impersonation (audit)
  started_at TIMESTAMPTZ DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,     -- Mặc định +24h
  ended_at TIMESTAMPTZ,
  actions_performed JSONB DEFAULT '[]' -- Mọi action trong session
);
CREATE INDEX idx_impers_admin ON impersonation_sessions(admin_id);
CREATE INDEX idx_impers_tenant ON impersonation_sessions(tenant_id);
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

### 6.3. Notification API (MỚI – v1.1)
```
GET    /api/admin/v1/notifications/templates
POST   /api/admin/v1/notifications/templates
PATCH  /api/admin/v1/notifications/templates/:id
DELETE /api/admin/v1/notifications/templates/:id

POST   /api/admin/v1/notifications/broadcast      # Gửi tới tất cả
POST   /api/admin/v1/notifications/targeted        # Gửi tới role/region
GET    /api/admin/v1/notifications/dispatch-log   # Tracking
GET    /api/admin/v1/notifications/stats           # Open/click rates
```

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
16. **Notification Center** – In-app notifications + Templates (MỚI – v1.1).

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

# PHẦN MỞ RỘNG (v1.1) – AUDIT, CODE EXAMPLES, EDGE CASES

## 11. Audit – Đánh giá nội dung hiện tại

### 11.1. Phần đã đủ chi tiết ✓

| Mục | Nội dung | Mức đủ |
|-----|---------|--------|
| 4 – 130 tính năng | Chia rõ 9 nhóm, mỗi nhóm ≥ 10 tính năng | ✓ |
| 5 – Database Schema | 7 bảng + 2 bảng bổ sung | ✓ |
| 6 – API Surface | REST + WebSocket/SSE + Connect-RPC | ✓ |
| 7 – UI/UX Components | 12 components được định nghĩa | ✓ |
| 10 – Bảo mật Dark Admin | SPA + YubiKey + Audit | ✓ |

### 11.2. Phần còn thiếu ⚠

| Mục | Vấn đề | Hướng bổ sung |
|-----|--------|---------------|
| 5.5 – audit_log | Lưu ScyllaDB nhưng chưa có RLS/partition key đầy đủ | Bổ sung retention policy |
| 6 – API | Thiếu error response format chuẩn | Bổ sung envelope chuẩn |
| 7.3 – Components | Thiếu `NotificationComposer` (4.9 mới thêm) | Bổ sung vào UI |
| 8.3 – Backpressure | "Drop oldest" có thể mất alert P0 | Bổ sung priority queue |
| 9 – Observability | Chưa có SLO/SLA dashboard riêng | Bổ sung §9.3 SLO Dashboard |
| 10 – Dark Admin | Chưa mô tả rõ key rotation mechanism | Bổ sung key ceremony |
| – Tổng thể | Thiếu **team ownership matrix** | Bổ sung §20 |
| – Tổng thể | Thiếu **Disaster Recovery** chi tiết | Bổ sung §17 |

### 11.3. Mâu thuẫn nội bộ ✗

| Vị trí | Mâu thuẫn |
|--------|-----------|
| 6 – API §6.1 (admin) | Có route `/api/admin/v1/*` nhưng §2.1 sơ đồ chỉ có 1 cổng 8891 → cần làm rõ ingress |
| 5.3 – tenants.status | Có `DELETING` nhưng §4.1 (item 4) lại ghi "hard delete" → không khớp |
| 4.7 – Billing | "Stripe, VNPay, Momo" nhưng §3 không có FINANCE_ADMIN permission cho việc config payment provider |
| 10.2 – Auth Flow | Bước "1. Mở SPA Tool" chưa nói rõ SPA tool chạy ở đâu (admin laptop?) |

### 11.4. Phần cần code example cụ thể 💡

| Mục | Cần code cho |
|-----|--------------|
| 2 – Kiến trúc | `admin-gateway` Go service skeleton |
| 5 – Schema | sqlc.yaml + migration file |
| 6 – API | Echo handler + Huma schema cho login flow |
| 8 – Realtime | SSE handler + NATS consumer |
| 10 – Dark Admin | WebAuthn registration + YubiKey challenge |

## 12. Edge Cases & Error Scenarios chi tiết

### 12.1. Edge Cases – Authentication

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| A1 | Admin quên password | Reset flow cần YubiKey | Force WebAuthn re-register |
| A2 | YubiKey mất → backup YubiKey | Khi login fail | Allow dùng backup key (đã đăng ký) |
| A3 | YubiKey cả 2 mất | Không vào được | Recovery key dạng paper (sinh lúc setup) |
| A4 | Brute force WebAuthn challenge | 5 lần fail | Lock account 30 phút + alert |
| A5 | YubiKey device lạ (không đăng ký) | WebAuthn reject | Log + alert + lock |
| A6 | PASETO token bị replay | Signature + nonce | Verify timestamp window 5min |
| A7 | Concurrent login 2 thiết bị | Session count > N | Force logout session cũ nhất |
| A8 | Admin login từ IP bất thường | Geo anomaly | Email + Telegram alert |
| A9 | WireGuard key bị compromise | Handshake pattern detect | Force rekey tất cả node |
| A10 | SPA packet bị sniff (khó) | Challenge-response one-time | Reject, log |

### 12.2. Edge Cases – Tenant Management

| # | Edge case | Xử lý |
|---|----------|-------|
| T1 | Tạo tenant với slug trùng | UNIQUE constraint → 409 Conflict |
| T2 | Xóa tenant có hơn 1000 user | Cascade delete background job + grace period |
| T3 | Tenant bị khóa nhưng WebSocket connections vẫn live | Force disconnect tại gateway + 30s timeout |
| T4 | Move tenant từ cluster A → B mà fail giữa chừng | Rollback + retry, idempotency_key |
| T5 | Admin clone tenant nhưng DB size 100GB | Background async job + progress UI |
| T6 | 2 admin cùng edit tenant plan | Optimistic locking với `version` column |
| T7 | Tenant có hơn 100 domains | List query phải paginate |
| T8 | SSL cert renewal fail 3 lần liên tiếp | Alert + fallback cert (manual upload) |
| T9 | Custom domain chưa verify mà user truy cập | 521 SSL handshake fail + troubleshooting page |
| T10 | Tenant bị FROZEN mà vẫn có scheduled job | Cancel all NATS subscriptions |

### 12.3. Edge Cases – Realtime Dashboard

| # | Edge case | Xử lý |
|---|----------|-------|
| R1 | SSE connection bị drop giữa chừng | Client auto-reconnect với Last-Event-ID |
| R2 | Quá nhiều admin connect cùng lúc | Limit 100 concurrent, dùng pub/sub backpressure |
| R3 | ClickHouse chậm → dashboard lag | Cache metric trong Valkey 1s |
| R4 | Grafana query timeout | Materialized view refresh mỗi 30s |
| R5 | Client subscribe filter lỗi | Default subscribe tất cả + log warning |
| R6 | Disk full trên admin node | Auto cleanup old logs + alert |
| R7 | Admin mở 10 tab cùng lúc | Tối đa 3 session active, các tab sau bị kick |
| R8 | Server time skew giữa các metric service | NTP enforce, log warning nếu skew > 1s |

### 12.4. Edge Cases – Quorum & Multi-Party

| # | Edge case | Xử lý |
|---|----------|-------|
| Q1 | Quorum request hết hạn giữa 2 admin đang ký | Auto cancel, log |
| Q2 | 2 admin ký cùng lúc (race condition) | Atomic update collected_signatures |
| Q3 | Admin tạo quorum cho chính mình | Block, require initiator ≠ signer |
| Q4 | YubiKey fail khi đang ký quorum | Cho phép retry trong 5 phút |
| Q5 | Quorum approved nhưng action fail | Auto-rollback + notify admin |
| Q6 | Audit log của quorum bị tamper | ScyllaDB Merkle hash chain (immutable log) |

### 12.5. Edge Cases – Notification System (MỚI – v1.1)

| # | Edge case | Xử lý |
|---|----------|-------|
| N1 | Telegram bot bị rate-limit | Queue lại, retry với exponential backoff |
| N2 | Email bị bounce (invalid address) | Auto disable notification cho user đó |
| N3 | SMS provider (Twilio) downtime | Fallback qua VNPay SMS gateway |
| N4 | Thông báo P0 nhưng admin offline | SMS + phone call (PagerDuty) |
| N5 | Thông báo marketing nhưng user quiet hours | Defer sang 7:00 sáng hôm sau |
| N6 | Template biến thiếu (variable không match) | Send "unknown" placeholder + alert dev |
| N7 | Notification queue quá tải (>1M pending) | Bulk thành batch digest (1 email/ngày) |
| N8 | A/B test 2 variant mà data skew | Auto-revert về variant A |
| N9 | User unsubscribe khỏi category | Honor immediately |
| N10 | Tracking pixel block bởi email client | Open rate ≈ 20% thực → adjust metric |

### 12.6. Edge Cases – Impersonation

| # | Edge case | Xử lý |
|---|----------|-------|
| I1 | Admin impersonate xong quên end session | 24h hard expiry, force logout |
| I2 | Admin impersonate nhưng mất mạng | Session persist, khi reconnect vẫn impersonate |
| I3 | Trong lúc impersonate, có 2 admin cùng impersonate cùng tenant | Cho phép 2 session parallel (audit đầy đủ) |
| I4 | Impersonate để xóa data, sau đó khiếu nại | Audit log đầy đủ lưu ScyllaDB |
| I5 | Support admin impersonate vượt quyền | RBAC vẫn áp dụng + alert SECURITY_ADMIN |

## 13. Code Examples chi tiết

### 13.1. Service `admin-gateway` – Skeleton (Go + Echo + Huma)

```
services/admin-gateway/
├── cmd/
│   └── main.go
├── internal/
│   ├── api/
│   │   ├── auth.go
│   │   ├── tenants.go
│   │   ├── dashboard.go
│   │   └── audit.go
│   ├── domain/
│   │   ├── admin.go
│   │   ├── tenant.go
│   │   └── audit.go
│   ├── repository/
│   │   ├── admin_repo.go
│   │   ├── tenant_repo.go
│   │   └── audit_repo.go
│   ├── service/
│   │   ├── auth_service.go
│   │   └── tenant_service.go
│   ├── middleware/
│   │   ├── trace.go
│   │   ├── auth.go
│   │   └── ratelimit.go
│   └── config/
│       └── config.go
├── migrations/
├── Dockerfile
└── go.mod
```

**main.go:**
```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/adapters/echoadaptor"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

    "rinco/admin-gateway/internal/api"
    "rinco/admin-gateway/internal/config"
    "rinco/admin-gateway/internal/middleware/trace"
)

func main() {
    cfg := config.Load()
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    e := echo.New()
    e.HideBanner = true

    // Middleware
    e.Use(trace.TraceMiddleware())
    e.Use(middleware.Recover())
    e.Use(middleware.RequestID())

    // Health endpoints
    e.GET("/health/live", func(c echo.Context) error { return c.JSON(200, map[string]string{"status": "ok"}) })
    e.GET("/health/ready", api.ReadinessHandler)

    // Huma API
    api := huma.NewAPI(echoadaptor.New(e, huma.DefaultConfig("RINCO Admin API", "1.0.0")))
    api.UseMiddleware(trace.HumaMiddleware())

    // Register endpoints
    api.RegisterRoutes()

    srv := &http.Server{
        Addr:              cfg.ListenAddr, // ":8891"
        Handler:           e,
        ReadHeaderTimeout: 5 * time.Second,
    }

    // Graceful shutdown
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("server failed", "err", err)
            os.Exit(1)
        }
    }()

    slog.Info("admin-gateway started", "addr", cfg.ListenAddr)

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        slog.Error("shutdown failed", "err", err)
    }
}
```

### 13.2. WebAuthn Login Flow

**auth_service.go (Go):**
```go
package service

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "time"

    "github.com/go-webauthn/webauthn/protocol"
    "github.com/go-webauthn/webauthn/webauthn"
    "github.com/google/uuid"
    "github.com/o1egl/paseto"

    "rinco/admin-gateway/internal/domain"
)

type AuthService struct {
    repo       AdminRepository
    webauthn   *webauthn.WebAuthn
    pasetoKey  paseto.V4SymmetricKey
    valkey     RedisClient
    challengeCache map[string]ChallengeData // In-memory cache với TTL
}

type ChallengeData struct {
    Challenge   string
    UserID      string
    ExpiresAt   time.Time
}

func (s *AuthService) BeginWebAuthnLogin(ctx context.Context, email string) (*protocol.CredentialAssertion, error) {
    admin, err := s.repo.FindByEmail(ctx, email)
    if err != nil {
        return nil, ErrInvalidCredentials // Generic error để chống enumeration
    }
    if admin.Status != "ACTIVE" {
        return nil, ErrAccountDisabled
    }
    if admin.FailedLoginCount >= 5 {
        if admin.LockedUntil.After(time.Now()) {
            return nil, ErrAccountLocked
        }
    }

    options, sessionData, err := s.webauthn.BeginLogin(admin)
    if err != nil {
        return nil, err
    }

    // Cache challenge với TTL 5 phút
    s.challengeCache[admin.ID.String()] = ChallengeData{
        Challenge: sessionData.Challenge,
        UserID:    admin.ID.String(),
        ExpiresAt: time.Now().Add(5 * time.Minute),
    }

    return options, nil
}

func (s *AuthService) FinishWebAuthnLogin(ctx context.Context, email string, response *protocol.ParsedCredentialAssertionData) (string, error) {
    admin, err := s.repo.FindByEmail(ctx, email)
    if err != nil {
        return nil, ErrInvalidCredentials
    }

    cached, ok := s.challengeCache[admin.ID.String()]
    if !ok || cached.ExpiresAt.Before(time.Now()) {
        return nil, ErrChallengeExpired
    }
    delete(s.challengeCache, admin.ID.String())

    credential, err := s.webauthn.ValidateLogin(admin, *cached, response)
    if err != nil {
        // Increment failed count
        admin.FailedLoginCount++
        if admin.FailedLoginCount >= 5 {
            admin.LockedUntil = time.Now().Add(30 * time.Minute)
        }
        s.repo.Update(ctx, admin)

        // Alert SECURITY_ADMIN
        s.notifyFailedLogin(ctx, admin, err)

        return "", ErrInvalidCredentials
    }

    // Update counter
    for _, cred := range admin.Credentials {
        if string(cred.CredentialID) == string(credential.ID) {
            cred.Counter = credential.Authenticator.Count
        }
    }
    admin.FailedLoginCount = 0
    admin.LastLoginAt = time.Now()
    s.repo.Update(ctx, admin)

    // Issue PASETO token
    token := s.issuePASETO(admin)

    // Audit
    s.auditLog.Record(ctx, domain.AuditEntry{
        ActorID:   admin.ID,
        Action:    "admin.login",
        IPAddress: getIPFromContext(ctx),
        TraceID:   getTraceID(ctx),
    })

    return token, nil
}

func (s *AuthService) issuePASETO(admin *domain.SuperAdmin) (string, error) {
    token := paseto.NewToken()
    token.SetIssuer("rinco-admin")
    token.SetSubject(admin.ID.String())
    token.SetExpiration(time.Now().Add(8 * time.Hour))
    token.Set("email", admin.Email)
    token.Set("role", admin.Role)

    encrypted := token.Encrypt(s.pasetoKey)
    return encrypted, nil
}
```

### 13.3. Tenant CRUD với Audit Log (sqlc + Ent hybrid)

**tenants.go:**
```go
package api

import (
    "context"
    "net/http"
    "time"

    "github.com/danielgtaylor/huma/v2"
    "github.com/google/uuid"

    "rinco/admin-gateway/internal/domain"
    "rinco/admin-gateway/internal/service"
)

type CreateTenantRequest struct {
    Body struct {
        Name         string `json:"name" minLength:"3" maxLength:"100" required:"true"`
        Slug         string `json:"slug" pattern:"^[a-z0-9-]{3,30}$" required:"true"`
        Plan         string `json:"plan" enum:"FREE,PRO,BUSINESS,ENTERPRISE" required:"true"`
        Region       string `json:"region" required:"true"`
        Template     string `json:"template" default:"default"`
        AdminEmail   string `json:"admin_email" format:"email" required:"true"`
    }
}

type CreateTenantResponse struct {
    Body struct {
        ID         string    `json:"id"`
        Slug       string    `json:"slug"`
        CreatedAt  time.Time `json:"created_at"`
        AdminURL   string    `json:"admin_url"`
    }
}

func (h *Handler) CreateTenant(ctx context.Context, input *CreateTenantRequest) (*CreateTenantResponse, error) {
    actor := getActorFromContext(ctx) // From middleware

    // 1. Validate slug uniqueness
    exists, err := h.tenantSvc.ExistsBySlug(ctx, input.Body.Slug)
    if err != nil {
        return nil, huma.Error500InternalServerError("failed to check slug", err)
    }
    if exists {
        return nil, huma.Error409Conflict("slug already exists", nil)
    }

    // 2. Create tenant transaction
    tenant, adminUser, err := h.tenantSvc.CreateWithAdmin(ctx, domain.TenantCreateInput{
        Name:       input.Body.Name,
        Slug:       input.Body.Slug,
        Plan:       input.Body.Plan,
        Region:     input.Body.Region,
        Template:   input.Body.Template,
        AdminEmail: input.Body.AdminEmail,
        ActorID:    actor.ID,
    })
    if err != nil {
        return nil, huma.Error500InternalServerError("failed to create tenant", err)
    }

    // 3. Audit log
    h.auditSvc.Record(ctx, domain.AuditEntry{
        TraceID:    getTraceID(ctx),
        ActorID:    actor.ID,
        ActorEmail: actor.Email,
        TenantID:   tenant.ID,
        Action:     "tenant.create",
        TargetType: "tenant",
        TargetID:   tenant.ID,
        Payload: map[string]any{
            "plan":   tenant.Plan,
            "region": tenant.Region,
        },
        IPAddress: getIPFromContext(ctx),
    })

    return &CreateTenantResponse{
        Body: struct {
            ID         string    `json:"id"`
            Slug       string    `json:"slug"`
            CreatedAt  time.Time `json:"created_at"`
            AdminURL   string    `json:"admin_url"`
        }{
            ID:        tenant.ID.String(),
            Slug:      tenant.Slug,
            CreatedAt: tenant.CreatedAt,
            AdminURL:  fmt.Sprintf("https://%s.hanghoaphaisinh.net", tenant.Slug),
        },
    }, nil
}
```

### 13.4. SSE Realtime Dashboard

**dashboard.go:**
```go
package api

import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "github.com/labstack/echo/v4"
    "github.com/nats-io/nats.go"

    "rinco/admin-gateway/internal/metrics"
)

type DashboardEvent struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
    TS      int64           `json:"ts"`
}

func (h *Handler) StreamDashboard(c echo.Context) error {
    ctx := c.Request().Context()

    // Set SSE headers
    c.Response().Header().Set("Content-Type", "text/event-stream")
    c.Response().Header().Set("Cache-Control", "no-cache")
    c.Response().Header().Set("Connection", "keep-alive")
    c.Response().Header().Set("X-Accel-Buffering", "no")

    // Subscribe to NATS topic
    sub, err := h.nats.Subscribe("admin.metrics.>", func(msg *nats.Msg) {
        event := parseMetricEvent(msg.Data)
        data, _ := json.Marshal(event)
        c.Response().Write([]byte("data: "))
        c.Response().Write(data)
        c.Response().Write([]byte("\n\n"))
        c.Response().Flush()
    })
    if err != nil {
        return err
    }
    defer sub.Unsubscribe()

    // Heartbeat every 15s
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            c.Response().Write([]byte(": heartbeat\n\n"))
            c.Response().Flush()
        }
    }
}
```

### 13.5. Notification System (MỚI – v1.1)

**notification_service.go:**
```go
package service

import (
    "context"
    "fmt"
    "sync"
    "text/template"
    "time"

    "rinco/admin-gateway/internal/domain"
)

type NotificationDispatcher struct {
    templates map[string]domain.NotificationTemplate
    channels  map[string]ChannelSender // email, telegram, push, sms
    repo      NotificationRepository
    rateLimiter *RateLimiter
    mu        sync.RWMutex
}

type ChannelSender interface {
    Send(ctx context.Context, recipient string, subject, body string) error
    Channel() string
}

func (d *NotificationDispatcher) Broadcast(ctx context.Context, code string, vars map[string]any, target TargetFilter) error {
    tmpl, ok := d.templates[code]
    if !ok {
        return ErrTemplateNotFound
    }

    // Check quiet hours (nếu category != CRITICAL)
    if tmpl.Category != "CRITICAL" {
        if isQuietHours(time.Now(), vars) {
            // Schedule for next 7am
            return d.scheduleDigest(ctx, tmpl, vars, target)
        }
    }

    // Resolve recipients
    recipients, err := d.resolveRecipients(ctx, target)
    if err != nil {
        return err
    }

    // Render template
    subject, body := renderTemplate(tmpl, vars)

    // Dispatch in parallel
    var wg sync.WaitGroup
    errCh := make(chan error, len(recipients))
    for _, r := range recipients {
        wg.Add(1)
        go func(r domain.Recipient) {
            defer wg.Done()
            if d.rateLimiter.Allow(r.ID, tmpl.Channel) {
                err := d.channels[tmpl.Channel].Send(ctx, r.Address, subject, body)
                d.logDispatch(ctx, tmpl, r, err)
                if err != nil {
                    errCh <- err
                }
            }
        }(r)
    }
    wg.Wait()
    close(errCh)

    return nil
}

func renderTemplate(t domain.NotificationTemplate, vars map[string]any) (string, string) {
    subjTmpl, _ := template.New("subj").Parse(t.Subject)
    bodyTmpl, _ := template.New("body").Parse(t.BodyTemplate)

    var subj, body strings.Builder
    subjTmpl.Execute(&subj, vars)
    bodyTmpl.Execute(&body, vars)

    return subj.String(), body.String()
}

// Quiet hours detection (default 22:00-07:00 theo timezone user)
func isQuietHours(now time.Time, vars map[string]any) bool {
    tz, ok := vars["timezone"].(string)
    if !ok {
        tz = "UTC"
    }
    loc, err := time.LoadLocation(tz)
    if err != nil {
        return false
    }
    hour := now.In(loc).Hour()
    return hour >= 22 || hour < 7
}
```

### 13.6. Quorum 2-of-3 cho Xóa Tenant

**quorum_handler.go:**
```go
package api

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "time"

    "github.com/danielgtaylor/huma/v2"
    "github.com/google/uuid"
)

type QuorumCreateRequest struct {
    Body struct {
        Action  string         `json:"action" required:"true"` // "tenant.delete", "gateway.global_config"
        Payload map[string]any `json:"payload" required:"true"`
        Reason  string         `json:"reason" minLength:"10" maxLength:"500" required:"true"`
    }
}

type QuorumCreateResponse struct {
    Body struct {
        QuorumID  string    `json:"quorum_id"`
        ExpiresAt time.Time `json:"expires_at"`
        SignURL   string    `json:"sign_url"`
    }
}

func (h *Handler) CreateQuorum(ctx context.Context, input *QuorumCreateRequest) (*QuorumCreateResponse, error) {
    actor := getActorFromContext(ctx)

    quorum := &domain.QuorumRequest{
        ID:               uuid.New(),
        Action:           input.Body.Action,
        Payload:          input.Body.Payload,
        InitiatorID:      actor.ID,
        Reason:           input.Body.Reason,
        RequiredSigs:      2,
        CollectedSigs:    []domain.Signature{},
        Status:           "PENDING",
        ExpiresAt:        time.Now().Add(5 * time.Minute),
    }

    if err := h.quorumRepo.Create(ctx, quorum); err != nil {
        return nil, huma.Error500InternalServerError("failed to create quorum", err)
    }

    // Notify other OWNER + SRE_ADMIN
    h.notifyAdmins(ctx, []string{"OWNER", "SRE_ADMIN"}, "quorum.pending", map[string]any{
        "quorum_id": quorum.ID.String(),
        "action":    quorum.Action,
        "initiator": actor.Email,
        "reason":    quorum.Reason,
        "expires_at": quorum.ExpiresAt,
    })

    return &QuorumCreateResponse{
        Body: struct {
            QuorumID  string    `json:"quorum_id"`
            ExpiresAt time.Time `json:"expires_at"`
            SignURL   string    `json:"sign_url"`
        }{
            QuorumID:  quorum.ID.String(),
            ExpiresAt: quorum.ExpiresAt,
            SignURL:   fmt.Sprintf("https://admin.rinco.local/quorum/%s/sign", quorum.ID),
        },
    }, nil
}

type QuorumSignRequest struct {
    Body struct {
        Signature string `json:"signature" required:"true"` // YubiKey signed nonce
    }
}

func (h *Handler) SignQuorum(ctx context.Context, input *QuorumSignRequest, quorumID string) (*struct{}, error) {
    actor := getActorFromContext(ctx)

    quorum, err := h.quorumRepo.GetByID(ctx, uuid.MustParse(quorumID))
    if err != nil {
        return nil, huma.Error404NotFound("quorum not found", err)
    }

    // Verify state
    if quorum.Status != "PENDING" {
        return nil, huma.Error409Conflict("quorum already finalized", nil)
    }
    if quorum.ExpiresAt.Before(time.Now()) {
        h.quorumRepo.UpdateStatus(ctx, quorum.ID, "EXPIRED")
        return nil, huma.Error410Gone("quorum expired", nil)
    }
    if quorum.InitiatorID == actor.ID {
        return nil, huma.Error403Forbidden("cannot sign own quorum", nil)
    }

    // Verify YubiKey signature (challenge was the quorum ID + nonce)
    expectedPayload := fmt.Sprintf("%s:%s", quorum.ID, actor.ID)
    expectedSig := h.signWithYubiKey(expectedPayload)
    if !hmac.Equal([]byte(input.Body.Signature), []byte(expectedSig)) {
        return nil, huma.Error401Unauthorized("invalid signature", nil)
    }

    // Add signature (atomic)
    err = h.quorumRepo.AppendSignature(ctx, quorum.ID, domain.Signature{
        SignerID: actor.ID,
        SignedAt: time.Now(),
        Signature: input.Body.Signature,
    })
    if err != nil {
        return nil, huma.Error500InternalServerError("failed to record signature", err)
    }

    // Check if quorum reached
    quorum, _ = h.quorumRepo.GetByID(ctx, quorum.ID)
    if len(quorum.CollectedSigs) >= quorum.RequiredSigs {
        h.quorumRepo.UpdateStatus(ctx, quorum.ID, "APPROVED")
        // Execute the action
        h.executeQuorumAction(ctx, quorum)
    }

    return &struct{}{}, nil
}

func (h *Handler) executeQuorumAction(ctx context.Context, q *domain.QuorumRequest) {
    switch q.Action {
    case "tenant.delete":
        tenantID := q.Payload["tenant_id"].(string)
        err := h.tenantSvc.HardDelete(ctx, uuid.MustParse(tenantID))
        if err != nil {
            h.auditSvc.Record(ctx, domain.AuditEntry{
                Action: "quorum.execute.failed",
                Payload: map[string]any{
                    "quorum_id": q.ID,
                    "action":    q.Action,
                    "error":     err.Error(),
                },
            })
        }
    // ...
    }
}
```

### 13.7. sqlc Migration cho Tenants

**migrations/0001_create_tenants.sql:**
```sql
-- +goose Up
-- +goose StatementBegin
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
    version INT NOT NULL DEFAULT 1,  -- Optimistic locking
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tenants_slug ON tenants(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_region ON tenants(region) WHERE deleted_at IS NULL;

-- Trigger update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tenants_updated_at
BEFORE UPDATE ON tenants
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TABLE IF EXISTS tenants;
DROP FUNCTION IF EXISTS update_updated_at_column();
-- +goose StatementEnd
```

## 14. Implementation Roadmap chi tiết

### 14.1. Phase 1 – Admin MVP (Tuần 1–4)

#### Tuần 1: Skeleton + Auth
- [ ] Tạo `services/admin-gateway/` Go project.
- [ ] Setup Echo + Huma + sqlc.
- [ ] WireGuard container dev.
- [ ] Migrations: `super_admins`, `super_admin_webauthn`.
- [ ] Health endpoints.

#### Tuần 2: WebAuthn Login
- [ ] WebAuthn registration endpoint.
- [ ] WebAuthn login flow.
- [ ] PASETO token issue.
- [ ] Rate limiting middleware.

#### Tuần 3: Tenant CRUD
- [ ] Tenant create/list/get/update.
- [ ] Slug validation.
- [ ] Audit log cho mọi action.
- [ ] Optimistic locking.

#### Tuần 4: Dashboard + Realtime
- [ ] SSE endpoint.
- [ ] NATS consumer.
- [ ] ClickHouse aggregation.
- [ ] Basic Grafana dashboard embed.

**Acceptance Gate Phase 1:**
- [ ] Admin login được với YubiKey.
- [ ] Tạo tenant mới trong < 5s.
- [ ] SSE stream cập nhật < 1s.

### 14.2. Phase 2 – Tính năng nâng cao (Tuần 5–8)

#### Tuần 5: Quorum + Audit
- [ ] Quorum create/sign flow.
- [ ] 2-of-3 signing.
- [ ] Auto execute action.

#### Tuần 6: Impersonation + Impersonation Banner
- [ ] Start impersonation (giới hạn 24h).
- [ ] End impersonation.
- [ ] Audit log đầy đủ.

#### Tuần 7: Notification System
- [ ] Templates CRUD.
- [ ] Telegram/Email/SMS channels.
- [ ] Broadcast API.
- [ ] Dispatch log + stats.

#### Tuần 8: Resource Manager
- [ ] K3s API client.
- [ ] Cluster health view.
- [ ] WireGuard mesh status.

**Acceptance Gate Phase 2:**
- [ ] Xóa tenant cần 2 YubiKey.
- [ ] Impersonate tenant hoạt động với audit trail.
- [ ] Gửi broadcast notification đến 1000 admin < 30s.

### 14.3. Phase 3 – AI & Observability (Tuần 9–12)

#### Tuần 9: AI SRE Hook
- [ ] Sentry webhook integration.
- [ ] AI RCA generation.
- [ ] Telegram notification.

#### Tuần 10: Anomaly Detection
- [ ] Traffic anomaly baseline.
- [ ] Error spike detection.
- [ ] Alert AI confidence scoring.

#### Tuần 11: Capacity Planning
- [ ] Historical metrics aggregation.
- [ ] Right-sizing recommendation.

#### Tuần 12: Polish
- [ ] Performance tuning.
- [ ] E2E test full flow.
- [ ] Documentation.

**Acceptance Gate Phase 3:**
- [ ] AI RCA < 3s.
- [ ] Anomaly detected < 30s.
- [ ] Load test: 100 concurrent admins không lag.

### 14.4. Acceptance Criteria cuối Phase

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-ADM-01 | Tạo tenant mới trong < 5s | p95 |
| AC-ADM-02 | Realtime dashboard cập nhật < 1s | p95 |
| AC-ADM-03 | Tìm kiếm log < 100ms trên 1B row | ClickHouse |
| AC-ADM-04 | Audit log ghi đầy đủ 100% action | 100% |
| AC-ADM-05 | Không có DNS/IP public cho admin | Security scan |
| AC-ADM-06 | Quorum 2-of-3 hoạt động | Test scenario |
| AC-ADM-07 | WebAuthn login < 500ms | p95 |
| AC-ADM-08 | Notification broadcast < 30s cho 1000 recipients | p95 |
| AC-ADM-09 | Impersonation audit đầy đủ | 100% actions tracked |
| AC-ADM-10 | AI RCA trong < 3s | p95 |

## 15. Testing Strategy

### 15.1. Unit Test Targets

| Module | Coverage |
|--------|----------|
| Auth Service | ≥ 90% |
| Tenant Service | ≥ 85% |
| Quorum Handler | ≥ 90% |
| Notification Dispatcher | ≥ 85% |
| Audit Logger | ≥ 90% |
| Impersonation Service | ≥ 90% |

### 15.2. Integration Tests

```go
// services/admin-gateway/test/integration/tenant_lifecycle_test.go
package integration_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    "rinco/admin-gateway/test/helpers"
)

func TestTenantLifecycle_CreateLockDelete(t *testing.T) {
    ctx := context.Background()
    h := helpers.NewTestHarness(t)
    defer h.Cleanup()

    // 1. Create tenant
    actor := h.CreateAdmin(ctx, helpers.AdminOpts{Role: "OWNER"})

    tenant, err := h.TenantService.Create(ctx, domain.TenantCreateInput{
        Name:       "Apex Fintech Test",
        Slug:       "apexfintech-test",
        Plan:       "PRO",
        Region:     "vn-sg",
        AdminEmail: "admin@apex.vn",
    }, actor)
    require.NoError(t, err)

    // 2. Lock tenant
    err = h.TenantService.Lock(ctx, tenant.ID, actor)
    require.NoError(t, err)

    locked, _ := h.TenantService.Get(ctx, tenant.ID)
    require.Equal(t, "LOCKED", locked.Status)

    // 3. Quorum delete (need 2 admins)
    admin2 := h.CreateAdmin(ctx, helpers.AdminOpts{Role: "OWNER"})
    quorum, err := h.QuorumService.Create(ctx, domain.QuorumCreateInput{
        Action:  "tenant.delete",
        Payload: map[string]any{"tenant_id": tenant.ID.String()},
        Reason:  "GDPR request from customer",
    }, actor)
    require.NoError(t, err)

    err = h.QuorumService.Sign(ctx, quorum.ID, admin2, helpers.MockYubiKeySignature())
    require.NoError(t, err)

    // 4. Verify audit log
    logs, err := h.AuditRepo.FindByTenant(ctx, tenant.ID)
    require.NoError(t, err)
    require.GreaterOrEqual(t, len(logs), 3) // create, lock, delete

    // 5. Verify tenant soft-deleted
    deleted, _ := h.TenantService.Get(ctx, tenant.ID)
    require.Equal(t, "DELETING", deleted.Status)
}
```

### 15.3. E2E Test (Playwright)

```typescript
// apps/admin/e2e/login.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Admin Login', () => {
  test('successful YubiKey login', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="email"]', 'owner@rinco.app');

    // Click login - this triggers WebAuthn ceremony
    await page.click('button:has-text("Login with YubiKey")');

    // Mock YubiKey response (in real test would use hardware)
    await page.evaluate(() => {
      (window as any).mockWebAuthnAssertion();
    });

    await expect(page).toHaveURL(/\/dashboard/);
    await expect(page.locator('text=Welcome')).toBeVisible();
  });

  test('reject when YubiKey fails', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="email"]', 'owner@rinco.app');
    await page.click('button:has-text("Login with YubiKey")');

    await page.evaluate(() => {
      (window as any).mockWebAuthnFailure();
    });

    await expect(page.locator('text=Authentication failed')).toBeVisible();
  });
});
```

## 16. Migration Plan

### 16.1. Initial Setup
```bash
# 1. Tạo admin database
createdb -h localhost -U postgres rinco_admin

# 2. Run migrations
cd services/admin-gateway
goose -dir migrations postgres "postgres://postgres@localhost:5432/rinco_admin?sslmode=disable" up

# 3. Seed first OWNER
psql -h localhost -U postgres rinco_admin < scripts/seed_owner.sql
```

### 16.2. Zero-downtime Schema Changes

Pattern đã được mô tả chi tiết trong `docs/00-master/README.md` §21.3.

### 16.3. Migration Tracking

| Version | Date | Description | Status |
|---------|------|-------------|--------|
| 0001 | T0 | super_admins, webauthn | Done |
| 0002 | T1 | tenants, domains | Done |
| 0003 | T2 | audit_log (Postgres) → migrate to Scylla | Pending |
| 0004 | T3 | feature_flags, quorum_requests | Done |
| 0005 | T4 | notification_templates, dispatch_log | Done |
| 0006 | T5 | impersonation_sessions | Done |
| 0007 | T6 | RLS policies trên tenants | Pending |
| 0008 | T7 | Index cho audit (timestamp DESC, tenant_id) | Pending |

## 17. Disaster Recovery

### 17.1. RPO & RTO

| Component | RPO | RTO |
|-----------|-----|-----|
| admin-gateway | 0 (stateless) | 30s (K3s restart) |
| postgres-admin | 5 min (WAL) | 30 min (restore from backup) |
| audit-log ScyllaDB | 1 hour | 2 hours |
| WireGuard config | 1 hour | 15 min |

### 17.2. Failure Scenarios

#### Scenario A: admin-gateway down
- **Detection:** K3s liveness probe fail
- **Response:** K3s restart < 2s
- **Recovery:** Stateless, no data loss

#### Scenario B: postgres-admin corruption
- **Detection:** Smoke test fail
- **Response:**
  1. Stop writes (set admin-gateway to read-only).
  2. Restore from latest backup.
  3. Apply WAL logs since backup.
  4. Smoke test.
  5. Resume writes.
- **RTO:** 30 min

#### Scenario C: WireGuard mesh down
- **Detection:** Handshake fail alert
- **Response:** SRE manual re-establish WireGuard + redistribute keys
- **RTO:** 15 min

#### Scenario D: Audit log loss (ScyllaDB)
- **Detection:** Audit count gap
- **Response:** Restore from daily snapshot
- **RPO:** Max 1 hour

### 17.3. Backup Strategy

```bash
# scripts/backup-admin-db.sh (chạy mỗi giờ)
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
pg_basebackup -h postgres-admin-primary -D /backup/admin/$TIMESTAMP \
  --checkpoint=fast --wal-method=stream
tar czf /backup/admin/$TIMESTAMP.tar.gz /backup/admin/$TIMESTAMP
mc cp /backup/admin/$TIMESTAMP.tar.gz minio/backups/admin/

# Retain 30 days
mc rm --older-than 30d minio/backups/admin/
```

### 17.4. DR Drill (Hàng quý)

Test case:
1. Snapshot admin DB.
2. Kill primary Postgres.
3. Promote replica.
4. Verify admin-gateway vẫn hoạt động.
5. Verify không mất audit log.

## 18. Cost Estimation

### 18.1. Compute Cost

| Component | Spec | Qty | Unit price | Monthly |
|-----------|------|-----|------------|---------|
| admin-gateway | 4 vCPU, 8GB | 2 | $120 | $240 |
| admin-bff (Next.js) | 2 vCPU, 4GB | 2 | $80 | $160 |
| postgres-admin | 4 vCPU, 16GB, 100GB NVMe | 1 + 1 replica | $300 | $600 |
| scylla audit | 4 vCPU, 16GB, 200GB | 1 | $200 | $200 |
| WireGuard bastion | 2 vCPU, 4GB | 2 | $80 | $160 |
| Grafana | 2 vCPU, 4GB | 1 | $80 | $80 |
| **Subtotal Admin** | - | - | - | **$1,440** |

### 18.2. Storage Cost

| Item | Size | Cost/GB | Monthly |
|------|------|---------|---------|
| Postgres backup (30d) | 1TB | $0.023 | $23 |
| ScyllaDB backup | 500GB | $0.023 | $11.5 |
| Audit log archive | 5TB | $0.023 | $115 |
| **Subtotal Storage** | - | - | **$150** |

### 18.3. External Services

| Service | Cost |
|---------|------|
| Sentry Team (1 seat) | $26 |
| PagerDuty (1 user) | $21 |
| Telegram Bot API | Free |
| **Subtotal External** | **$47** |

### 18.4. Tổng Admin

```
Compute:  $1,440
Storage:  $150
External: $47
──────────────
Total:    ~$1,637/month
```

### 18.5. Cost per Tenant

Với 10,000 tenants:
- Cost per tenant for admin overhead: **$0.16/tenant/month**
- Rất thấp so với giá trị giám sát mang lại.

## 19. Open Questions / Cần user xác nhận

### 19.1. Cần user xác nhận ngay ✋

| # | Câu hỏi | Options | Recommendation |
|---|---------|---------|----------------|
| Q1 | **WebAuthn support trên Safari/Firefox?** | (a) Chỉ hỗ trợ Chromium-based, (b) Full support | (b) cho UX tốt |
| Q2 | **Số lượng Super Admin tối đa?** | (a) 3 OWNER, (b) 5, (c) Unlimited | (a) – minimize attack surface |
| Q3 | **Quorum default là 2-of-3 hay 3-of-5?** | (a) 2-of-3, (b) 3-of-5 | (a) – balance speed & security |
| Q4 | **Notification channels ưu tiên?** | (a) Email only, (b) + Telegram, (c) + SMS | (b) – cost/UX balance |
| Q5 | **Có cần AI Auto-PR cho hotfix không?** | (a) Suggest only, (b) Auto PR + human approve | (a) – safe by default |
| Q6 | **Impersonation max duration?** | (a) 1h, (b) 8h, (c) 24h | (c) theo thiết kế hiện tại |
| Q7 | **Audit log retention ở production?** | (a) 1 năm, (b) 5 năm, (c) 7 năm | (b) theo thiết kế |
| Q8 | **Có cần rate limit per Super Admin?** | (a) Shared, (b) Per-admin | (b) – prevent abuse |
| Q9 | **Admin role nào được quyền impersonate?** | (a) Chỉ OWNER, (b) + SUPPORT_ADMIN | (b) theo thiết kế |
| Q10 | **Có cần hỗ trợ LDAP/SSO cho Super Admin?** | (a) Yes, (b) No | (b) Phase 1 – giữ đơn giản |

### 19.2. Cần quyết định trong Phase tiếp theo 📋

| # | Câu hỏi | Impact | Owner |
|---|---------|--------|-------|
| Q11 | Tenant deletion grace period bao lâu? | Storage cost | Product |
| Q12 | Có cho phép Super Admin reset MFA của tenant user? | Security vs support | Security |
| Q13 | Sentry self-host hay cloud? | Cost, ops | DevOps |
| Q14 | Grafana self-host hay cloud? | Cost, ops | DevOps |
| Q15 | Có cần mobile app cho Admin? | Effort | Product |
| Q16 | i18n cho Admin UI? | Effort | Frontend |
| Q17 | Có cần SSO cho Admin? | Effort | Security |
| Q18 | Backup cross-region cho admin DB? | Cost | DevOps |

### 19.3. TBD kỹ thuật ⏳

| # | Item | Status | Next step |
|---|------|--------|-----------|
| T1 | PASETO key rotation mechanism | TBD | Design Q4 2026 |
| T2 | Sentry SDK cho Admin | TBD | Tích hợp Phase 3 |
| T3 | AI SRE prompt template | TBD | Optimize qua feedback |
| T4 | Grafana datasource cho audit_log | TBD | Setup PostgreSQL source |
| T5 | Backup encryption (KMS) | TBD | AWS KMS / Vault |
| T6 | WebAuthn backup code format | TBD | BIP39 vs custom |
| T7 | Real-time channel backpressure (SSE) | TBD | Test với 1000 concurrent |
| T8 | Notification template editor UI | TBD | Design Q4 2026 |

### 19.4. Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| YubiKey supply chain | Low | Critical | Backup key + recovery codes |
| WebAuthn browser regression | Low | High | Test trên tất cả browser mỗi release |
| Audit log loss | Low | Critical | ScyllaDB replication + offsite backup |
| AI hallucination (RCA) | Medium | High | Human-in-the-loop + sandbox test PR |
| Telegram bot banned | Low | Medium | Fallback Slack + Email |
| Notification spam (DoS admin) | Medium | Medium | Rate limit + per-admin filter |
| Mass impersonate abuse | Low | Critical | 24h expiry + audit + alert |
| Dark Admin bị leak IP | Low | Critical | WireGuard only + bind trên `wg0` |

---

**Tiếp theo:** [`docs/02-tenant-site/README.md`](../02-tenant-site/README.md) – Thiết kế Tenant Company Site + Isolated VPS.