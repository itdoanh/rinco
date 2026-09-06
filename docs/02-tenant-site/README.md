# Phần 2 – Tenant Company Site + Isolated VPS

> **Phân hệ:** Trang web riêng cho mỗi công ty đối tác + Khả năng triển khai trên VPS riêng.  
> **Mục tiêu:** Cho phép mỗi tenant có homepage riêng (vd: `apex.hanghoaphaisinh.net` hoặc custom domain), có thể chạy shared cluster hoặc isolated VPS.  
> **Đặc thù:** WireGuard Mesh kết nối mọi VPS, Distributed Resource Sharing Scheduler tận dụng tài nguyên rảnh.

---

## Mục lục

1. [Mục tiêu & Nguyên tắc](#1-mục-tiêu--nguyên-tắc)
2. [Kiến trúc](#2-kiến-trúc)
3. [Định tuyến Domain (3 mô hình)](#3-định-tuyến-domain-3-mô-hình)
4. [Isolated VPS Deployment](#4-isolated-vps-deployment)
5. [WireGuard Mesh](#5-wireguard-mesh)
6. [Distributed Resource Sharing](#6-distributed-resource-sharing)
7. [Tenant Site Components](#7-tenant-site-components)
8. [Database Schema](#8-database-schema)
9. [Danh sách tính năng (≥ 100)](#9-danh-sách-tính-năng)
10. [API Surface](#10-api-surface)
11. [CI/CD & Deployment](#11-cicd--deployment)

**Mục lục mở rộng (phần bổ sung – v1.1)**

12. [Audit – Đánh giá nội dung hiện tại](#12-audit--đánh-giá-nội-dung-hiện-tại)
13. [Edge Cases & Error Scenarios chi tiết](#13-edge-cases--error-scenarios-chi-tiết)
14. [Code Examples chi tiết](#14-code-examples-chi-tiết)
15. [Implementation Roadmap chi tiết](#15-implementation-roadmap-chi-tiết)
16. [Testing Strategy](#16-testing-strategy)
17. [Migration Plan](#17-migration-plan)
18. [Disaster Recovery](#18-disaster-recovery)
19. [Cost Estimation](#19-cost-estimation)
20. [Open Questions / Cần user xác nhận](#20-open-questions--cần-user-xác-nhận)

---

## 1. Mục tiêu & Nguyên tắc

### 1.1. Mục tiêu
| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| TN-1 | Mỗi tenant có homepage riêng | 100% tenant có URL |
| TN-2 | Domain resolution < 0.5ms | Valkey lookup |
| TN-3 | Isolated VPS có thể tự quản | Self-managed |
| TN-4 | Mesh network an toàn | WireGuard mTLS |
| TN-5 | Tận dụng tài nguyên rảnh | ≥ 30% utilization |

### 1.2. Nguyên tắc
- **Tenant tự chủ:** Tenant admin có toàn quyền trong tenant của mình.
- **Cách ly tuyệt đối:** Shared cluster vẫn RLS + per-tenant data dir.
- **Mesh ngang hàng:** Mọi VPS ngang hàng, không có central bottleneck.
- **Tự động scale:** Thêm VPS mới tự động join mesh.

---

## 2. Kiến trúc

### 2.1. Hai chế độ triển khai

#### A. Shared Cluster Mode
```
┌──────────────────────────────────────────────────┐
│  SHARED CLUSTER (Central)                        │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐    │
│  │ Control    │ │ Edge GW    │ │ Services   │    │
│  │ Plane      │ │ Rust/Go    │ │ Go/Rust/Py │    │
│  └────────────┘ └────────────┘ └────────────┘    │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐    │
│  │ PostgreSQL │ │ ScyllaDB   │ │ ClickHouse │    │
│  └────────────┘ └────────────┘ └────────────┘    │
└──────────────────────────────────────────────────┘
       ▲         ▲         ▲
       │         │         │
   [Tenant A] [Tenant B] [Tenant C]   (chỉ truy cập qua Gateway)
```

#### B. Isolated VPS Mode
```
┌──────────────────────────────────────────────────┐
│  Tenant A's VPS                                  │
│  ┌────────────────────────────────────┐          │
│  │ K3s Single-node                    │          │
│  │ - Edge GW                          │          │
│  │ - App Services                     │          │
│  │ - Local PostgreSQL/ScyllaDB       │          │
│  └────────────────────────────────────┘          │
└──────────────────────────────────────────────────┘
       │
       │ WireGuard Mesh
       ▼
┌──────────────────────────────────────────────────┐
│  Mesh Coordinator (central)                      │
│  - Cluster registry                              │
│  - Resource Sharing Scheduler                    │
│  - Backup & Replication                          │
└──────────────────────────────────────────────────┘
```

### 2.2. Mesh Coordinator Services
- **`mesh-controller`** (Go + Headscale): Quản lý WireGuard keys, ACL.
- **`resource-scheduler`** (Go): Phân bổ workload dựa trên tải.
- **`replication-manager`** (Go): Đồng bộ backup giữa các cluster.
- **`vps-provisioner`** (Go + Terraform/Pulumi): Tự động tạo VPS qua Cloud API.

---

## 3. Định tuyến Domain (3 mô hình)

### 3.1. SUBPATH (Shared Domain)
```
URL: hanghoaphaisinh.net/apexfintech
```
- Tất cả tenant dùng chung domain gốc.
- Routing: Gateway parse subpath → lookup tenant_id trong Valkey.
- Ưu điểm: Không cần DNS, triển khai nhanh.
- Nhược: SEO kém hơn, không brand riêng.

### 3.2. SUBDOMAIN
```
URL: apex.hanghoaphaisinh.net
```
- DNS wildcard `*.hanghoaphaisinh.net` trỏ về Edge Gateway.
- Gateway parse Host header → lookup tenant.
- Ưu điểm: Brand tốt hơn, dễ nhớ.
- Nhược: Phụ thuộc domain gốc.

### 3.3. CUSTOM DOMAIN
```
URL: landing.apexcorp.vn (do tenant tự trỏ)
```
- Tenant tạo bản ghi CNAME → `edge.rinco.app` HOẶC A record → IP Edge.
- Let's Encrypt auto-issue SSL.
- ACME challenge tự động.
- Ưu điểm: Hoàn toàn brand riêng.
- Nhược: Cần setup DNS bên tenant.

### 3.4. Domain Map Cache (Valkey)
```
Key: domain:apex.hanghoaphaisinh.net
Value: { tenant_id: "apexfintech", routing_type: "subdomain", ssl_status: "active", expires_at: 1234567890 }
TTL: 3600s
```
- Lookup p99 < 0.5ms.
- Invalidation khi tenant update config.
- Replicated ra tất cả Gateway nodes.

### 3.5. SSL/TLS
- **Wildcard cert** cho subpath/subdomain: `*.hanghoaphaisinh.net`.
- **Per-tenant cert** cho custom domain qua Let's Encrypt DNS-01 challenge.
- Cert renewal tự động (cron 30 ngày trước expiry).
- OCSP Stapling bật.

---

## 4. Isolated VPS Deployment

### 4.1. K3s Single-node Cluster
- Cài K3s + WireGuard + Edge Gateway + local DB.
- Tất cả chạy trên 1 VPS ban đầu, scale khi cần.
- Backup mỗi ngày → Mesh Coordinator.

### 4.2. Hardware tối thiểu
| Profile | CPU | RAM | Disk | Bandwidth |
|---------|-----|-----|------|-----------|
| Small | 2 vCPU | 4GB | 80GB SSD | 100Mbps |
| Medium | 4 vCPU | 8GB | 160GB SSD | 500Mbps |
| Large | 8 vCPU | 16GB | 320GB NVMe | 1Gbps |
| XL | 16 vCPU | 32GB | 640GB NVMe | 2Gbps |

### 4.3. Bootstrap Script
```bash
#!/bin/bash
# bootstrap-vps.sh
# Chạy trên VPS mới do tenant mua
set -e

# 1. Install K3s
curl -sfL https://get.k3s.io | sh -s - \
  --node-name=$HOSTNAME \
  --write-kubeconfig-mode=644 \
  --disable=traefik \
  --disable=servicelb

# 2. Install WireGuard
apt install -y wireguard
wg genkey | tee /etc/wireguard/private.key | wg pubkey > /etc/wireguard/public.key

# 3. Join Mesh
PUBLIC_KEY=$(cat /etc/wireguard/public.key)
ENDPOINT=$(curl -s ifconfig.me)
curl -X POST https://mesh.rinco.app/v1/nodes/register \
  -H "Authorization: Bearer $REGISTRATION_TOKEN" \
  -d "{\"public_key\":\"$PUBLIC_KEY\",\"endpoint\":\"$ENDPOINT\",\"tenant_id\":\"$TENANT_ID\"}"

# 4. Apply tenant manifests
kubectl apply -f https://mesh.rinco.app/manifests/$TENANT_ID.yaml

# 5. Verify
sleep 30
curl https://localhost:8891/health
```

### 4.4. Multi-VPS Expansion
- Tenant có thể yêu cầu thêm VPS (vd: 1 cho DB, 1 cho App, 1 cho Media).
- Mesh Controller tự động setup WireGuard giữa các VPS.
- Service Mesh (Linkerd/Istio) để giao tiếp nội bộ.

---

## 5. WireGuard Mesh

### 5.1. Topology
```
                    ┌─────────────────────┐
                    │  Mesh Coordinator   │
                    │  (Central)          │
                    └──────────┬──────────┘
                               │
                  ┌────────────┼────────────┐
                  │            │            │
            ┌─────▼─────┐ ┌───▼─────┐ ┌────▼────┐
            │ VPS A    │ │ VPS B   │ │ VPS C   │
            │ Tenant 1 │ │ Tenant 2│ │ Tenant 3│
            └──────────┘ └─────────┘ └─────────┘
                  ▲            ▲           ▲
                  │            │           │
                  └────────────┴───────────┘
                       (full mesh - optional)
```

### 5.2. Implementation
- **Headscale** (open-source Tailscale control plane) làm coordinator.
- Mỗi VPS chạy `headscale` client.
- PreAuth Keys có TTL ngắn (10 phút).
- ACL giới hạn traffic: VPS A chỉ thấy Coordinator + các VPS cùng tenant.

### 5.3. Security
- WireGuard keys rotate mỗi 90 ngày.
- PSK (Pre-Shared Key) layer ngoài public key cho extra security.
- Audit log mọi lần handshake.

### 5.4. eBPF Mesh Acceleration
- eBPF program trong kernel đẩy gói tin mesh trực tiếp giữa các NIC.
- Bypass TCP/IP stack cho traffic internal mesh.
- Latency mesh: ~5ms giữa các VPS cùng region.

---

## 6. Distributed Resource Sharing

### 6.1. Scheduler Algorithm
```python
# resource-scheduler/main.py (pseudo-code)
def schedule_workload(workload):
    candidates = get_online_vps(workload.region)
    
    # Filter by capacity
    candidates = [v for v in candidates if v.cpu_free >= workload.cpu_required]
    candidates = [v for v in candidates if v.ram_free >= workload.ram_required]
    
    if not candidates:
        return reject(workload, "no_capacity")
    
    # Score by load (prefer less loaded)
    scores = []
    for v in candidates:
        cpu_util = v.cpu_used / v.cpu_total
        ram_util = v.ram_used / v.ram_total
        network = v.network_util
        score = (1 - cpu_util) * 0.4 + (1 - ram_util) * 0.4 + (1 - network) * 0.2
        scores.append((score, v))
    
    best = max(scores, key=lambda x: x[0])
    return assign(workload, best[1].id)
```

### 6.2. Workload Types có thể chia sẻ
| Workload | Chia sẻ được | Lý do |
|----------|-------------|-------|
| Render video meeting | ✅ | Stateless, CPU/GPU intensive |
| AI Inference (RAG) | ✅ | Stateless, GPU intensive |
| Backup & Replication | ✅ | Background job |
| Email/Push send | ✅ | Queue-based |
| Real-time chat | ❌ | Cần gần user |
| CRM core | ❌ | Sensitive data |

### 6.3. Trust Model
- **Sandbox:** Workload chạy trong container với seccomp + AppArmor.
- **No data access:** Workload chỉ nhận data đã được mask.
- **Audit:** Mọi job execution log vào Coordinator.
- **Billing:** Tenant A cho mượn tài nguyên → được credit.

---

## 7. Tenant Site Components

### 7.1. Tenant Site Service (Next.js)
- Mỗi tenant có 1 instance Next.js app (hoặc share qua dynamic routing).
- **Dynamic routing:** `/sites/[tenant]/[...slug]`.
- **Static Generation:** Cache HTML cho các page ít thay đổi.
- **ISR:** Incremental Static Regeneration mỗi 60s.

### 7.2. Sections có thể customize
1. **Hero Banner** – Hình ảnh + CTA.
2. **About Company** – Giới thiệu.
3. **Services/Products** – Danh sách dịch vụ.
4. **Team** – Đội ngũ.
5. **Testimonials** – Khách hàng nói gì.
6. **Contact Form** – Form liên hệ.
7. **News/Blog** – Tin tức.
8. **Pricing** – Bảng giá (nếu áp dụng).
9. **FAQ** – Câu hỏi thường gặp.
10. **Footer** – Liên hệ, MXH, copyright.
11. **Multi-language** – VN/EN/JP...
12. **Cookie consent banner.**
13. **Chat widget** (kết nối Chat Engine).
14. **Booking widget** (đặt lịch họp).
15. **Custom domain SSL badge.**

### 7.3. Theme Customization
- **Colors:** Primary, Secondary, Accent, Background.
- **Typography:** Font family, sizes.
- **Logo:** Upload SVG/PNG.
- **Favicon.**
- **Custom CSS** (cho advanced tenants).
- **Component variants** (rounded, sharp, pill buttons).

### 7.4. BFF (Backend-for-Frontend)
- **`tenant-site-bff`** (TS + Bun):
  - Aggregate API calls.
  - SSR data fetching.
  - Cache layer (Valkey).
  - Personalization (nếu user đã login).

---

## 8. Database Schema

### 8.1. Bảng `tenant_sites`
```sql
CREATE TABLE tenant_sites (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
  template TEXT NOT NULL DEFAULT 'default',
  sections JSONB NOT NULL DEFAULT '[]',
  theme JSONB NOT NULL DEFAULT '{}',
  custom_css TEXT,
  favicon_url TEXT,
  primary_locale TEXT DEFAULT 'vi',
  supported_locales TEXT[] DEFAULT '{vi}',
  seo_config JSONB DEFAULT '{}',
  analytics_config JSONB DEFAULT '{}',
  is_published BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_tenant_sites_tenant ON tenant_sites(tenant_id);
```

### 8.2. Bảng `tenant_vps_nodes`
```sql
CREATE TABLE tenant_vps_nodes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
  hostname TEXT NOT NULL,
  public_ip INET,
  private_ip INET,
  wg_public_key TEXT,
  wg_endpoint TEXT,
  region TEXT,
  tier TEXT CHECK (tier IN ('SHARED','ISOLATED_SMALL','ISOLATED_MEDIUM','ISOLATED_LARGE','ISOLATED_XL')),
  cpu_cores INT,
  ram_gb INT,
  disk_gb INT,
  bandwidth_mbps INT,
  status TEXT CHECK (status IN ('PROVISIONING','ONLINE','OFFLINE','MAINTENANCE')),
  k3s_version TEXT,
  last_heartbeat TIMESTAMPTZ,
  allow_resource_sharing BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT now()
);
```

### 8.3. Bảng `mesh_nodes`
```sql
CREATE TABLE mesh_nodes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vps_node_id UUID REFERENCES tenant_vps_nodes(id),
  wg_public_key TEXT UNIQUE NOT NULL,
  wg_ip INET NOT NULL,
  endpoint TEXT,
  allowed_peers UUID[],
  online BOOLEAN DEFAULT false,
  last_handshake TIMESTAMPTZ,
  bytes_sent BIGINT DEFAULT 0,
  bytes_received BIGINT DEFAULT 0
);
```

### 8.4. Bảng `resource_sharing_jobs`
```sql
CREATE TABLE resource_sharing_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_tenant_id UUID REFERENCES tenants(id),
  target_vps_id UUID REFERENCES tenant_vps_nodes(id),
  workload_type TEXT CHECK (workload_type IN ('VIDEO_RENDER','AI_INFERENCE','BACKUP','NOTIFICATION')),
  cpu_required INT,
  ram_required INT,
  gpu_required BOOLEAN DEFAULT false,
  status TEXT CHECK (status IN ('QUEUED','RUNNING','COMPLETED','FAILED')),
  result_url TEXT,
  credit_amount DECIMAL(10,2),
  created_at TIMESTAMPTZ DEFAULT now(),
  completed_at TIMESTAMPTZ
);
```

### 8.5. Valkey Keys
```
domain:<hostname> → JSON{tenant_id, routing_type, ssl_status}
tenant:<tenant_id>:site-cache → JSON sections
vps:<vps_id>:capacity → JSON{cpu_used, cpu_total, ram_used, ram_total}
mesh:peer:<wg_public_key> → JSON{tenant_id, vps_id, wg_ip}
```

### 8.6. Bảng `tenant_site_pages` (MỚI – v1.1) – cho Blog/News/FAQ
```sql
CREATE TABLE tenant_site_pages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
  slug TEXT NOT NULL,
  page_type TEXT NOT NULL CHECK (page_type IN ('BLOG','FAQ','LANDING','CUSTOM')),
  locale TEXT NOT NULL DEFAULT 'vi',
  title TEXT NOT NULL,
  excerpt TEXT,
  content TEXT,                 -- Markdown hoặc JSON cho block-based editor
  meta_title TEXT,
  meta_description TEXT,
  og_image_url TEXT,
  status TEXT NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','PUBLISHED','ARCHIVED')),
  published_at TIMESTAMPTZ,
  author_id UUID,
  tags TEXT[],
  category TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(tenant_id, slug, locale)
);
CREATE INDEX idx_tenant_site_pages_slug ON tenant_site_pages(tenant_id, slug);
CREATE INDEX idx_tenant_site_pages_status ON tenant_site_pages(tenant_id, status, published_at DESC);
CREATE INDEX idx_tenant_site_pages_tags ON tenant_site_pages USING GIN(tags);
```

### 8.7. Bảng `tenant_site_form_submissions` (MỚI – v1.1)
```sql
CREATE TABLE tenant_site_form_submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  site_id UUID REFERENCES tenant_sites(id) ON DELETE CASCADE,
  form_id TEXT NOT NULL,             -- Slug của form
  data JSONB NOT NULL,
  ip_address INET,
  user_agent TEXT,
  referer TEXT,
  utm JSONB,
  status TEXT DEFAULT 'NEW' CHECK (status IN ('NEW','PROCESSED','SPAM')),
  processed_by UUID,
  processed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_form_subs_tenant ON tenant_site_form_submissions(tenant_id, created_at DESC);
```

### 8.8. Bảng `mesh_resource_quotas` (MỚI – v1.1)
```sql
CREATE TABLE mesh_resource_quotas (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vps_node_id UUID REFERENCES tenant_vps_nodes(id) ON DELETE CASCADE,
  cpu_quota_cores INT,                -- Max CPU share được
  ram_quota_gb INT,                   -- Max RAM share được
  gpu_quota_hours_per_month INT,      -- GPU quota
  network_quota_gb_per_month INT,
  bandwidth_priority INT DEFAULT 100,
  enabled BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);
```

---

## 9. Danh sách tính năng (≥ 100)

### 9.1. Quản lý Site (1-25)
1. Tạo site cho tenant mới (chọn template).
2. Chọn template (5+ template có sẵn: BĐS, Tài chính, Bán lẻ, Giáo dục, Mặc định).
3. Custom theme (color, font, logo).
4. Upload logo + favicon.
5. Thêm/sửa/xóa sections.
6. Kéo thả sections (drag & drop).
7. Preview trước khi publish.
8. Publish / Unpublish.
9. Schedule publish (lên lịch đăng).
10. Multi-language support (VN, EN, JP, KR, ZH).
11. Translation management.
12. Custom domain setup wizard.
13. SSL auto-renew.
14. Subdomain tự động tạo khi tenant tạo.
15. Subpath routing config.
16. SEO config (meta title, description, og:image).
17. Sitemap.xml auto-generate.
18. Robots.txt customization.
19. 301 redirect management.
20. Custom 500 error page.
21. Maintenance mode page.
22. Cookie consent banner (GDPR/PDPA).
23. Page speed optimization (image lazy load, critical CSS).
24. AMP version (optional).
25. PWA support (manifest.json, service worker).

### 9.2. Nội dung (26-45)
26. Hero banner với CTA.
27. About section.
28. Services/Products grid.
29. Team member list.
30. Testimonials carousel.
31. News/Blog system (CRUD).
32. Blog categories & tags.
33. Blog comments.
34. Search trong blog.
35. RSS feed cho blog.
36. FAQ accordion.
37. Pricing table.
38. Contact form (multi-step).
39. Contact info (address, phone, email, MXH).
40. Google Maps embed.
42. Newsletter signup.
44. Video embed (YouTube, Vimeo).
45. Document download.

### 9.3. Domain & SSL (46-60)
46. Custom domain setup wizard.
48. CNAME verification.
49. A record verification.
50. Auto SSL Let's Encrypt.
51. Wildcard SSL cho subdomain.
52. SSL force HTTPS.
53. HSTS preload.
54. SSL expiration alerts.
55. Multi-domain trên 1 site.
56. Domain transfer giữa tenants.
57. Domain ownership verification.
58. CAA record config.
59. SSL transparency monitoring.
60. Domain health check.

### 9.4. Isolated VPS (61-80)
61. Yêu cầu VPS mới (form: tier, region, OS).
62. Auto-provision qua Cloud API (DO, AWS, GCP, Azure, Vultr).
63. K3s auto-install.
64. WireGuard auto-join mesh.
65. DNS auto-update.
66. Monitoring auto-attach.
67. Backup auto-config (daily).
68. Firewall rules auto-apply.
69. Failover to backup VPS.
70. Multi-VPS cluster setup.
71. Cross-VPS service mesh.
72. Resource sharing toggle (allow/disallow).
73. Resource sharing credit dashboard.
74. Manual scale up/down.
75. Auto-scale based on metrics.
76. Scheduled maintenance window.
77. SSH key management.
78. Update notification (K3s upgrade).
79. Disaster recovery drill.
80. VPS decommission (graceful).

### 9.5. Mesh Network (81-95)
81. View mesh topology (graph).
82. Add/remove peer.
83. Rotate WireGuard keys.
84. View bandwidth per peer.
85. View latency per peer.
86. ACL management.
87. Mesh health check.
88. Offline peer alert.
89. Reconnect peer tự động.
90. Bandwidth limit per peer.
91. Priority routing (VPN traffic).
92. Split-tunnel config.
93. Mesh event log.
94. Mesh capacity planning.
95. Mesh performance dashboard.

### 9.6. Resource Sharing (96-105)
96. Opt-in/opt-out cho tenant.
97. Workload whitelist (chỉ render, AI inference).
98. CPU limit khi share.
99. RAM limit khi share.
100. GPU sharing toggle.
101. Credit system (cho/nhận).
102. Audit log sharing job.
103. SLA monitoring.
104. Anomaly detection (job lạ).
105. Auto-revoke nếu abuse.

### 9.7. Nâng cao (106-120)
106. Multi-region active-active.
107. CDN integration (Cloudflare, BunnyCDN).
108. Edge workers (Cloudflare Workers / Vercel Edge).
109. A/B test trên site.
110. Heatmap (Hotjar tương đương).
111. Session recording.
112. Form submission analytics.
113. Conversion funnel.
114. Custom event tracking.
115. Real-time visitor counter.
116. Geo-blocking (chặn theo quốc gia).
117. IP whitelist/blacklist.
118. Rate limit per IP.
119. Bot protection (CAPTCHA, Wasm).
120. Audit log mọi thay đổi site.

---

## 10. API Surface

### 10.1. Tenant Site API
```
GET    /api/site/v1/:tenant_slug
GET    /api/site/v1/:tenant_slug/sections/:section_id
POST   /api/site/v1/:tenant_slug/contact        # Submit contact form
POST   /api/site/v1/:tenant_slug/newsletter
GET    /api/site/v1/:tenant_slug/blog
GET    /api/site/v1/:tenant_slug/blog/:slug
GET    /api/site/v1/:tenant_slug/locales/:locale.json
```

### 10.2. Admin (Tenant scope)
```
GET    /api/tenant-admin/v1/site
PATCH  /api/tenant-admin/v1/site
POST   /api/tenant-admin/v1/site/publish
POST   /api/tenant-admin/v1/site/unpublish
POST   /api/tenant-admin/v1/site/preview-token

GET    /api/tenant-admin/v1/domains
POST   /api/tenant-admin/v1/domains
DELETE /api/tenant-admin/v1/domains/:id
POST   /api/tenant-admin/v1/domains/:id/verify

GET    /api/tenant-admin/v1/vps
POST   /api/tenant-admin/v1/vps/request
GET    /api/tenant-admin/v1/vps/:id/metrics
POST   /api/tenant-admin/v1/vps/:id/maintenance
DELETE /api/tenant-admin/v1/vps/:id

GET    /api/tenant-admin/v1/mesh/peers
GET    /api/tenant-admin/v1/resource-sharing/jobs
PATCH  /api/tenant-admin/v1/resource-sharing/preferences
```

---

## 11. CI/CD & Deployment

### 11.1. GitOps Workflow
```
GitHub (tenant-template repo)
    ↓ (push)
ArgoCD (per tenant)
    ↓ (sync)
K3s Cluster (shared hoặc isolated)
```

### 11.2. Manifest Structure
```
manifests/
├── tenant-apexfintech/
│   ├── namespace.yaml
│   ├── site-deployment.yaml
│   ├── site-service.yaml
│   ├── site-ingress.yaml
│   ├── site-configmap.yaml
│   └── site-secret.yaml
├── tenant-hct/
│   └── ...
```

### 11.3. Tenant Provisioning Flow
```
1. Super Admin tạo tenant trong Admin Portal
   ↓
2. tenant-manager gọi K3s API
   ↓
3. Tạo namespace + service account
   ↓
4. Apply default manifests
   ↓
5. Cấu hình domain mapping vào Valkey
   ↓
6. Issue SSL cert (Let's Encrypt)
   ↓
7. Gửi welcome email cho tenant admin
   ↓
8. Tenant admin login → configure site
```

### 11.4. Rollback
- Mỗi deploy lưu Git revision.
- ArgoCD tự động sync nhưng giữ history.
- Manual rollback qua Admin UI: chọn revision → Sync.

---

# PHẦN MỞ RỘNG (v1.1) – AUDIT, CODE EXAMPLES, EDGE CASES

## 12. Audit – Đánh giá nội dung hiện tại

### 12.1. Phần đã đủ chi tiết ✓

| Mục | Nội dung | Mức đủ |
|-----|---------|--------|
| 3 – Domain Routing | 3 mô hình + Valkey cache + SSL | ✓ |
| 4 – Isolated VPS | Bootstrap script + hardware tier | ✓ |
| 5 – WireGuard Mesh | Topology + Headscale + ACL | ✓ |
| 6 – Resource Sharing | Algorithm + workload types + trust model | ✓ |
| 9 – Tính năng | 120 tính năng chia 7 nhóm | ✓ |

### 12.2. Phần còn thiếu ⚠

| Mục | Vấn đề | Hướng bổ sung |
|-----|--------|---------------|
| 3.5 – SSL | Chưa có staging flow cho ACME DNS-01 challenge | Bổ sung |
| 4 – Isolated VPS | Thiếu K3s manifest example chi tiết | Bổ sung §14.3 |
| 5 – WireGuard | Chưa có disaster recovery cho mesh | Bổ sung §18 |
| 6.1 – Scheduler | Chưa có fairness giữa tenants | Bổ sung weighted round-robin |
| 7.4 – BFF | Chưa có CDN integration | Bổ sung §14.4 |
| 9.2 – Nội dung | Thiếu versioning cho Blog posts | Bổ sung page history |
| 11 – CI/CD | Chưa có staging cluster cho tenant manifests | Bổ sung staging flow |

### 12.3. Mâu thuẫn nội bộ ✗

| Vị trí | Mâu thuẫn |
|--------|-----------|
| 4.3 Bootstrap Script | Dùng `disable=traefik` nhưng mesh traefik cần để expose site → conflict |
| 6.3 Trust Model | Workload "stateless, không có data" nhưng AI inference cần context của tenant khác |
| 9.1 #22 Cookie consent | Yêu cầu GDPR/PDPA nhưng §3.5 chưa đề cập region VN có ngoại lệ gì |
| 11.3 Provisioning | Bước 6 SSL cert được issue NGAY khi tạo tenant, nhưng domain có thể chưa được tenant trỏ DNS |

### 12.4. Phần cần code example cụ thể 💡

| Mục | Cần code cho |
|-----|--------------|
| 3 – Domain Routing | Edge gateway TenantResolver middleware |
| 4 – Isolated VPS | Kustomize manifest template |
| 5 – WireGuard | Mesh controller Handshake flow |
| 6 – Resource Sharing | Scheduler implementation chi tiết |
| 7 – Site Components | Next.js getServerSideProps với multi-tenant |
| 8 – Schema | sqlc.yaml + migrations |
| 10 – API | Echo handlers |
| 11 – CI/CD | ArgoCD ApplicationSet |

## 13. Edge Cases & Error Scenarios chi tiết

### 13.1. Edge Cases – Domain Routing

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| D1 | Domain không tồn tại trong Valkey cache | Cache miss | Fallback PostgreSQL lookup → 404 |
| D2 | Valkey down | Health check | Return cached last-known (in-process LRU) |
| D3 | Tenant bị FROZEN nhưng vẫn có domain mapping | Lookup thấy tenant_id → check status | Return 503 "tenant suspended" |
| D4 | Wildcard SSL hết hạn | Cert expiry monitor | Auto-renew 30 ngày trước |
| D5 | Custom domain verify fail | ACME challenge fail | Email tenant + retry queue |
| D6 | Tenant đổi routing_type (subpath → subdomain) | Valkey invalidation | Atomic update + brief 503 |
| D7 | 2 tenant cùng đăng ký 1 custom domain | UNIQUE constraint | 409 Conflict |
| D8 | CNAME trỏ tới domain khác (không phải Edge) | HTTP-01 challenge fail | Verify guide + manual approval |
| D9 | DNS propagation chưa xong khi user truy cập | First time | 30 min grace period + troubleshooting |
| D10 | HSTS được enable nhưng browser cũ không support | Browser compatibility | Force TLS 1.2+ |

### 13.2. Edge Cases – Isolated VPS

| # | Edge case | Xử lý |
|---|----------|-------|
| V1 | VPS provisioning fail giữa chừng | Rollback cloud API call + manual cleanup |
| V2 | WireGuard join mesh fail | Retry với exponential backoff + manual SSH access |
| V3 | Tenant chọn region không có cloud provider | Show error + suggest nearest |
| V4 | VPS đầy disk | Auto alert + cleanup logs |
| V5 | VPS bị DDoS | Cloud provider firewall + eBPF XDP_DROP |
| V6 | Tenant mua VPS rồi bỏ (no traffic 90 ngày) | Auto-shutdown + email warning |
| V7 | Multi-VPS nhưng wireguard key conflict | Regenerate key + re-handshake |
| V8 | K3s upgrade fail | Rollback + pin version |
| V9 | VPS bị thu hồi bởi cloud (payment fail) | Drain + alert + 7 ngày grace |
| V10 | Cross-region latency > 200ms | Block share job + suggest same-region |

### 13.3. Edge Cases – Resource Sharing

| # | Edge case | Xử lý |
|---|----------|-------|
| R1 | 2 tenant cùng request GPU cùng lúc | Queue ưu tiên theo credit balance |
| R2 | Tenant share VPS bị compromise | Auto-revoke all shares + alert |
| R3 | Workload chạy quá RAM quota | Cgroup OOM kill + log |
| R4 | Job stuck ở "RUNNING" > 1h | Auto-kill + alert |
| R5 | Source tenant yêu cầu stop job | SIGTERM graceful 30s |
| R6 | Network bị chia cắt giữa VPS | Fallback sang local execution |
| R7 | Credit calculation sai (off-by-one) | Daily reconciliation job |
| R8 | Tenant disable sharing nhưng có job đang chạy | Finish job rồi mới disable |
| R9 | GPU quota > physical GPU có sẵn | Reject quota request |
| R10 | Job tạo ra data cần upload về tenant | Direct WireGuard transfer (no public) |

### 13.4. Edge Cases – Tenant Site

| # | Edge case | Xử lý |
|---|----------|-------|
| S1 | Tenant upload logo quá lớn (50MB) | Resize client-side + max 5MB |
| S2 | Theme JSON malformed | Validate trước khi lưu + reject 422 |
| S3 | Custom CSS phá vỡ layout | Sandboxed iframe preview |
| S4 | Template mới phá vỡ backward compat | Migration script + warning |
| S5 | Site traffic spike 100x (viral) | Auto-scale BFF + CDN |
| S6 | Contact form bị spam | Wasm attestation + rate limit |
| S7 | User upload file > 100MB | Multipart upload + reject |
| S8 | Site publish nhưng CDN cache chưa purge | Purge API + wait 30s |
| S9 | i18n translation rỗng | Fallback về primary locale |
| S10 | Blog post có markdown chứa XSS | Sanitize DOMPurify server-side |

### 13.5. Edge Cases – WireGuard Mesh

| # | Edge case | Xử lý |
|---|----------|-------|
| M1 | Peer offline > 5 min | Alert + auto-reconnect |
| M2 | Endpoint thay đổi (DHCP lease) | Update endpoint + handshake |
| M3 | 2 peer cùng dùng IP trong mesh | UNIQUE constraint trên wg_ip |
| M4 | Key rotate fail ở 1 peer | Quarantine peer + manual fix |
| M5 | Mesh controller bị chia cắt | Local peer-to-peer mesh fallback |
| M6 | Bandwidth vượt quota | Throttle + alert |
| M7 | ACL config lỗi block critical traffic | Audit + rollback |
| M8 | eBPF program không load được (kernel version) | Fallback userspace WireGuard |
| M9 | Handshake fail liên tục | Investigate clock skew |
| M10 | PSK rotate đồng loạt nhiều peer | Stagger rotation theo region |

### 13.6. Edge Cases – Disaster Recovery

| # | Edge case | Xử lý |
|---|----------|-------|
| DR1 | Mesh coordinator mất hoàn toàn | 10 phút RTO, mesh tự recover bằng gossip |
| DR2 | Toàn bộ cluster region mất | DNS failover region khác |
| DR3 | VPS bị xóa nhầm | Restore từ snapshot MinIO (nếu có) |
| DR4 | Database bị corrupt | PITR (Point In Time Recovery) |
| DR5 | Backup bị cũ (>30 ngày) | Alert ngay khi sync fail |
| DR6 | TLS cert của Mesh Coordinator expire | Auto-renew qua ACME |

## 14. Code Examples chi tiết

### 14.1. Edge Gateway – Tenant Resolver (Rust + io_uring)

```rust
// services/edge-gateway/src/tenant_resolver.rs
use std::sync::Arc;
use ahash::AHashMap;
use dashmap::DashMap;
use moka::future::Cache;
use redis::AsyncCommands;

#[derive(Debug, Clone)]
pub struct TenantConfig {
    pub tenant_id: String,
    pub routing_type: RoutingType,
    pub ssl_status: SslStatus,
    pub expires_at: i64,
    pub isolation_mode: IsolationMode,
}

#[derive(Debug, Clone)]
pub enum RoutingType {
    Subpath(String),
    Subdomain,
    Custom,
}

#[derive(Debug, Clone)]
pub enum IsolationMode {
    Shared,
    Isolated { vps_node_id: String },
}

pub struct TenantResolver {
    valkey: redis::Client,
    cache: Cache<String, Arc<TenantConfig>>,
    fallback_static: DashMap<String, Arc<TenantConfig>>,
}

impl TenantResolver {
    pub fn new(valkey: redis::Client) -> Self {
        let cache = Cache::builder()
            .max_capacity(50_000)
            .time_to_live(std::time::Duration::from_secs(3600))
            .build();

        Self {
            valkey,
            cache,
            fallback_static: DashMap::new(),
        }
    }

    pub async fn resolve(&self, host: &str, path: &str) -> Result<Arc<TenantConfig>, ResolveError> {
        // 1. Parse routing type
        let cache_key = self.parse_cache_key(host, path);

        // 2. Try local cache first (<0.5ms)
        if let Some(cfg) = self.cache.get(&cache_key).await {
            return Ok(cfg);
        }

        // 3. Fallback to Valkey (<5ms)
        let mut conn = self.valkey.get_async_connection().await
            .map_err(|e| ResolveError::CacheUnavailable(e.to_string()))?;

        let cached: Option<String> = conn.get(&cache_key).await
            .map_err(|e| ResolveError::CacheLookup(e.to_string()))?;

        if let Some(json) = cached {
            let cfg: TenantConfig = serde_json::from_str(&json)?;
            let cfg = Arc::new(cfg);

            // Update local cache
            self.cache.insert(cache_key, cfg.clone()).await;

            return Ok(cfg);
        }

        // 4. Try static fallback (last-known-good)
        if let Some(cfg) = self.fallback_static.get(&cache_key) {
            tracing::warn!("Using static fallback for {}", cache_key);
            return Ok(cfg.clone());
        }

        Err(ResolveError::NotFound)
    }

    fn parse_cache_key(&self, host: &str, path: &str) -> String {
        // 1. Check subpath: hanghoaphaisinh.net/apexfintech
        if path.starts_with('/') {
            let segments: Vec<&str> = path.split('/').collect();
            if segments.len() >= 2 {
                return format!("domain:{}:{}", host, segments[1]);
            }
        }

        // 2. Subdomain: apex.hanghoaphaisinh.net
        let parts: Vec<&str> = host.split('.').collect();
        if parts.len() >= 3 && !host.ends_with("hanghoaphaisinh.net") || parts.len() >= 3 {
            let subdomain = parts[0];
            if subdomain != "www" && subdomain != "admin" {
                return format!("domain:{}:{}", host, subdomain);
            }
        }

        // 3. Custom domain
        format!("domain:{}", host)
    }

    pub async fn invalidate(&self, key: &str) {
        self.cache.invalidate(key).await;
    }
}

#[derive(Debug, thiserror::Error)]
pub enum ResolveError {
    #[error("tenant not found")]
    NotFound,

    #[error("cache unavailable: {0}")]
    CacheUnavailable(String),

    #[error("cache lookup failed: {0}")]
    CacheLookup(String),

    #[error("invalid config: {0}")]
    InvalidConfig(#[from] serde_json::Error),
}
```

### 14.2. WireGuard Mesh Controller (Go)

```go
// services/mesh-controller/internal/controller/handshake.go
package controller

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/juanfont/headscale"

    "rinco/mesh-controller/internal/domain"
)

type HandshakeService struct {
    hs          *headscale.Client
    nodeRepo    NodeRepository
    auditLogger AuditLogger
}

func (s *HandshakeService) RegisterPeer(ctx context.Context, req RegisterPeerRequest) (*PeerRegistration, error) {
    // 1. Verify registration token
    if !s.verifyToken(req.RegistrationToken) {
        s.auditLogger.Record(ctx, AuditEntry{
            Action:  "peer.register.denied",
            Payload: map[string]any{"reason": "invalid_token"},
            IPAddress: req.IP,
        })
        return nil, ErrInvalidToken
    }

    // 2. Generate unique IP in 100.64.0.0/10 (CGNAT range, reserved for WireGuard)
    ip, err := s.allocateIP(ctx, req.TenantID)
    if err != nil {
        return nil, fmt.Errorf("allocate IP: %w", err)
    }

    // 3. Register with Headscale
    preAuthKey, err := s.hs.CreatePreAuthKey(ctx, req.TenantID, 10*time.Minute, false, false)
    if err != nil {
        return nil, fmt.Errorf("create preauth key: %w", err)
    }

    // 4. Save to DB
    peer := &domain.MeshNode{
        ID:           uuid.New(),
        VPSNodeID:    req.VPSNodeID,
        WGPublicKey:  req.PublicKey,
        WGIP:         ip,
        Endpoint:     req.Endpoint,
        AllowedPeers: s.determineAllowedPeers(ctx, req.TenantID),
        Online:       true,
        LastHandshake: time.Now(),
    }

    if err := s.nodeRepo.Create(ctx, peer); err != nil {
        return nil, fmt.Errorf("save peer: %w", err)
    }

    // 5. Audit
    s.auditLogger.Record(ctx, AuditEntry{
        TenantID:   req.TenantID,
        Action:     "peer.register",
        TargetType: "vps_node",
        TargetID:   req.VPSNodeID,
        Payload: map[string]any{
            "wg_ip":    ip.String(),
            "endpoint": req.Endpoint,
        },
    })

    return &PeerRegistration{
        WGIP:           ip.String(),
        PreAuthKey:     preAuthKey.Key,
        AllowedPeers:   peer.AllowedPeers,
        MeshConfigYAML: s.generateMeshConfig(peer),
    }, nil
}

func (s *HandshakeService) determineAllowedPeers(ctx context.Context, tenantID string) []string {
    // Same tenant = full mesh
    // Cross tenant = only coordinator
    peers, _ := s.nodeRepo.FindByTenant(ctx, tenantID)
    var allowed []string
    for _, p := range peers {
        allowed = append(allowed, p.ID.String())
    }
    // Always allow coordinator
    allowed = append(allowed, "coordinator-id")
    return allowed
}

func (s *HandshakeService) generateMeshConfig(peer *domain.MeshNode) string {
    return fmt.Sprintf(`
# Generated by RINCO Mesh Controller
# Peer: %s
[Interface]
PrivateKey = <generated-on-vps>
Address = %s/32
[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = %s/32
PersistentKeepalive = 25
`, peer.ID, peer.WGIP, peer.WGPublicKey, peer.Endpoint, peer.WGIP)
}
```

### 14.3. Kustomize Manifest cho Isolated VPS

```yaml
# manifests/tenant-apexfintech/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: tenant-apexfintech

resources:
  - namespace.yaml
  - tenant-site-deployment.yaml
  - tenant-site-service.yaml
  - tenant-site-ingress.yaml
  - postgres-statefulset.yaml
  - valkey-deployment.yaml
  - secret.yaml

labels:
  - includeSelectors: true
    pairs:
      tenant: apexfintech
      managed-by: argocd
```

```yaml
# manifests/tenant-apexfintech/tenant-site-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tenant-site
  namespace: tenant-apexfintech
spec:
  replicas: 2
  selector:
    matchLabels:
      app: tenant-site
      tenant: apexfintech
  template:
    metadata:
      labels:
        app: tenant-site
        tenant: apexfintech
      annotations:
        sidecar.istio.io/inject: "true"
    spec:
      containers:
      - name: site
        image: ghcr.io/itdoanh/rinco/tenant-site:v1.0.0
        ports:
        - containerPort: 3000
        env:
        - name: TENANT_ID
          value: apexfintech
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: tenant-secrets
              key: database-url
        - name: VALKEY_URL
          valueFrom:
            secretKeyRef:
              name: tenant-secrets
              key: valkey-url
        - name: NATS_URL
          value: nats://nats.mesh:4222
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 1Gi
        livenessProbe:
          httpGet:
            path: /health/live
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5
        securityContext:
          runAsNonRoot: true
          runAsUser: 65534
          readOnlyRootFilesystem: true
          allowPrivilegeEscalation: false
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchLabels:
                  app: tenant-site
```

### 14.4. Resource Sharing Scheduler (Go)

```go
// services/mesh-controller/internal/scheduler/scheduler.go
package scheduler

import (
    "context"
    "sort"
    "sync"

    "rinco/mesh-controller/internal/domain"
)

type WorkloadRequest struct {
    ID             string
    SourceTenantID string
    Type           domain.WorkloadType
    CPURequired    int    // cores
    RAMRequired    int    // GB
    GPURequired    bool
    NetworkReq     int    // Mbps
    Region         string
    Priority       int    // 1-10
}

type VPSCapacity struct {
    VPSID       string
    TenantID    string
    Region      string
    CPUUsed     int
    CPUTotal    int
    RAMUsed     int
    RAMTotal    int
    GPUAvailable bool
    AllowSharing bool
    CreditRate  float64  // $/hour
}

type Scheduler struct {
    mu        sync.RWMutex
    vpsCache  map[string]*VPSCapacity
    quotaRepo QuotaRepository
    auditLog  AuditLogger
}

func (s *Scheduler) Schedule(ctx context.Context, req WorkloadRequest) (*VPSCapacity, error) {
    s.mu.RLock()
    candidates := make([]*VPSCapacity, 0, len(s.vpsCache))
    for _, vps := range s.vpsCache {
        candidates = append(candidates, vps)
    }
    s.mu.RUnlock()

    // 1. Filter by capability
    var feasible []*VPSCapacity
    for _, vps := range candidates {
        if !vps.AllowSharing {
            continue
        }
        if vps.Region != req.Region {
            continue
        }
        cpuFree := vps.CPUTotal - vps.CPUUsed
        ramFree := vps.RAMTotal - vps.RAMUsed
        if cpuFree < req.CPURequired {
            continue
        }
        if ramFree < req.RAMRequired {
            continue
        }
        if req.GPURequired && !vps.GPUAvailable {
            continue
        }
        feasible = append(feasible, vps)
    }

    if len(feasible) == 0 {
        return nil, ErrNoCapacity
    }

    // 2. Check quota for source tenant on each candidate
    type scoredCandidate struct {
        vps   *VPSCapacity
        score float64
        quota int
    }
    scored := make([]scoredCandidate, 0, len(feasible))
    for _, vps := range feasible {
        quota, _ := s.quotaRepo.GetQuota(ctx, vps.VPSID)
        if quota <= 0 {
            continue
        }

        // Scoring:
        // 0.4 weight: load (prefer less loaded)
        // 0.3 weight: credit (prefer lower rate for tenant)
        // 0.2 weight: regional proximity (lower latency)
        // 0.1 weight: historical reliability
        cpuUtil := float64(vps.CPUUsed) / float64(vps.CPUTotal)
        ramUtil := float64(vps.RAMUsed) / float64(vps.RAMTotal)
        loadScore := 1 - (cpuUtil+ramUtil)/2

        creditScore := 1 - vps.CreditRate/100 // Normalize
        if creditScore < 0 {
            creditScore = 0
        }

        reliabilityScore := 0.8 // TODO: từ historical data

        score := loadScore*0.4 + creditScore*0.3 + reliabilityScore*0.1 + 0.2

        scored = append(scored, scoredCandidate{
            vps:   vps,
            score: score,
            quota: quota,
        })
    }

    if len(scored) == 0 {
        return nil, ErrNoQuota
    }

    // 3. Sort by score descending
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].score > scored[j].score
    })

    best := scored[0]

    // 4. Reserve capacity
    s.mu.Lock()
    best.vps.CPUUsed += req.CPURequired
    best.vps.RAMUsed += req.RAMRequired
    s.mu.Unlock()

    // 5. Audit
    s.auditLog.Record(ctx, AuditLogEntry{
        Action: "scheduler.assign",
        Payload: map[string]any{
            "workload_id":  req.ID,
            "source":       req.SourceTenantID,
            "target_vps":   best.vps.VPSID,
            "score":        best.score,
            "cpu_request":  req.CPURequired,
            "ram_request":  req.RAMRequired,
        },
    })

    return best.vps, nil
}

func (s *Scheduler) Release(ctx context.Context, workloadID string, vpsID string, cpu int, ram int) {
    s.mu.Lock()
    defer s.mu.Unlock()

    vps, ok := s.vpsCache[vpsID]
    if !ok {
        return
    }

    vps.CPUUsed -= cpu
    vps.RAMUsed -= ram
    if vps.CPUUsed < 0 {
        vps.CPUUsed = 0
    }
    if vps.RAMUsed < 0 {
        vps.RAMUsed = 0
    }

    s.auditLog.Record(ctx, AuditLogEntry{
        Action: "scheduler.release",
        Payload: map[string]any{
            "workload_id": workloadID,
            "vps_id":      vpsID,
        },
    })
}
```

### 14.5. Next.js Tenant Site với Multi-Tenant SSR

```typescript
// apps/tenant-site/pages/[tenant]/[...slug].tsx
import type { GetServerSideProps, NextPage } from 'next';
import { Hero } from '@/components/sections/Hero';
import { About } from '@/components/sections/About';
import { ContactForm } from '@/components/forms/ContactForm';

type TenantSiteProps = {
  tenant: {
    id: string;
    name: string;
    theme: any;
    sections: any[];
    locale: string;
  };
  page?: any;
};

const TenantSite: NextPage<TenantSiteProps> = ({ tenant, page }) => {
  return (
    <div style={{ '--primary': tenant.theme.primary } as any}>
      <main>
        {tenant.sections.map((section) => {
          switch (section.type) {
            case 'hero':
              return <Hero key={section.id} config={section.config} />;
            case 'about':
              return <About key={section.id} config={section.config} />;
            case 'contact':
              return <ContactForm key={section.id} tenantId={tenant.id} />;
            // ...
            default:
              return null;
          }
        })}
      </main>
    </div>
  );
};

export const getServerSideProps: GetServerSideProps<TenantSiteProps> = async (ctx) => {
  const { tenant: tenantSlug, slug } = ctx.params!;

  // 1. Resolve tenant via Edge Gateway (server-side, < 50ms)
  const tenant = await resolveTenant(ctx.req.headers.host!, `/${tenantSlug}`);

  if (!tenant) {
    return { notFound: true };
  }

  // 2. Check tenant status
  if (tenant.status !== 'ACTIVE') {
    return {
      redirect: {
        destination: '/maintenance',
        permanent: false,
      },
    };
  }

  // 3. Load site config (cache via BFF)
  const site = await bff.getSiteConfig(tenant.id);

  // 4. Load page if slug provided
  let page = null;
  if (slug && slug.length > 0) {
    page = await bff.getPage(tenant.id, slug.join('/'), ctx.locale ?? 'vi');
  }

  // 5. Cache headers (CDN-friendly)
  ctx.res.setHeader(
    'Cache-Control',
    page ? 'public, s-maxage=60, stale-while-revalidate=300'
         : 'public, s-maxage=300, stale-while-revalidate=3600'
  );

  return {
    props: { tenant: site, page },
  };
};

export default TenantSite;
```

### 14.6. Bootstrap Script – Production Ready

```bash
#!/bin/bash
# bootstrap-vps.sh - Production hardened
set -euo pipefail

LOG_FILE="/var/log/rinco-bootstrap.log"
exec > >(tee -a "$LOG_FILE") 2>&1

echo "=== RINCO VPS Bootstrap Started at $(date -u) ==="

# Required env vars
: "${TENANT_ID:?TENANT_ID must be set}"
: "${REGISTRATION_TOKEN:?REGISTRATION_TOKEN must be set}"
: "${MESH_ENDPOINT:?MESH_ENDPOINT must be set}"

# 1. Validate environment
if [ "$(id -u)" != "0" ]; then
    echo "ERROR: Must run as root"
    exit 1
fi

# 2. Update system
apt-get update -y
apt-get upgrade -y

# 3. Install K3s with custom config
echo "Installing K3s..."
curl -sfL https://get.k3s.io | sh -s - \
  --node-name="$(hostname)" \
  --write-kubeconfig-mode=644 \
  --disable=traefik \
  --disable=servicelb \
  --kubelet-arg="max-pods=250" \
  --kubelet-arg="node-ip=$(hostname -I | awk '{print $1}')"

# Wait for K3s
echo "Waiting for K3s to be ready..."
timeout 60 bash -c 'until kubectl get nodes 2>/dev/null | grep -q " Ready "; do sleep 2; done'

# 4. Install WireGuard
echo "Installing WireGuard..."
apt-get install -y wireguard qrencode

# 5. Generate WireGuard keys
wg genkey | tee /etc/wireguard/private.key > /dev/null
wg pubkey < /etc/wireguard/private.key > /etc/wireguard/public.key
chmod 600 /etc/wireguard/private.key

# 6. Get public IP
PUBLIC_IP=$(curl -s --max-time 10 https://api.ipify.org)
if [ -z "$PUBLIC_IP" ]; then
    PUBLIC_IP=$(curl -s --max-time 10 ifconfig.me)
fi
if [ -z "$PUBLIC_IP" ]; then
    echo "ERROR: Cannot determine public IP"
    exit 1
fi

PUBLIC_KEY=$(cat /etc/wireguard/public.key)

# 7. Register with mesh controller
echo "Registering with mesh controller..."
REGISTRATION_RESPONSE=$(curl -sf --max-time 30 \
    -X POST "${MESH_ENDPOINT}/v1/nodes/register" \
    -H "Authorization: Bearer ${REGISTRATION_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
        \"public_key\": \"${PUBLIC_KEY}\",
        \"endpoint\": \"${PUBLIC_IP}:51820\",
        \"tenant_id\": \"${TENANT_ID}\",
        \"region\": \"${REGION:-vn-sg}\",
        \"hostname\": \"$(hostname)\",
        \"specs\": {
            \"cpu_cores\": $(nproc),
            \"ram_gb\": $(free -g | awk '/^Mem/ {print $2}'),
            \"disk_gb\": $(df -BG / | awk 'NR==2 {print $2}' | tr -d 'G')
        }
    }")

WIREGUARD_IP=$(echo "$REGISTRATION_RESPONSE" | jq -r '.wg_ip')
MESH_CONFIG=$(echo "$REGISTRATION_RESPONSE" | jq -r '.mesh_config')

if [ -z "$WIREGUARD_IP" ] || [ "$WIREGUARD_IP" == "null" ]; then
    echo "ERROR: Mesh registration failed"
    echo "Response: $REGISTRATION_RESPONSE"
    exit 1
fi

# 8. Configure WireGuard interface
echo "Configuring WireGuard interface..."
cat > /etc/wireguard/wg0.conf << EOF
[Interface]
PrivateKey = $(cat /etc/wireguard/private.key)
Address = ${WIREGUARD_IP}/32
Table = auto
${MESH_CONFIG}
EOF
chmod 600 /etc/wireguard/wg0.conf

# 9. Start WireGuard
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0

# 10. Apply tenant manifests
echo "Applying tenant manifests..."
MANIFESTS_URL="${MESH_ENDPOINT}/manifests/${TENANT_ID}.yaml"
kubectl apply -f "$MANIFESTS_URL" || {
    echo "WARN: Failed to apply manifests. Continuing..."
}

# 11. Install monitoring agents
echo "Installing Vector agent..."
curl -sfL https://packages.vector.dev/install.sh | bash -s -- -y
cat > /etc/vector/vector.toml << EOF
[sources.journal]
type = "journald"

[transforms.rinco_logs]
type = "remap"
inputs = ["journal"]
source = '''
.tenant_id = "${TENANT_ID}"
.environment = "production"
'''

[sinks.clickhouse]
type = "clickhouse"
inputs = ["rinco_logs"]
endpoint = "${CLICKHOUSE_ENDPOINT}"
database = "rinco_logs"
table = "events_raw"
EOF
systemctl enable vector
systemctl start vector

# 12. Install node_exporter
echo "Installing node_exporter..."
NODE_EXPORTER_VERSION="1.7.0"
curl -sfL "https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz" \
    | tar xz -C /opt/
ln -s /opt/node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64/node_exporter /usr/local/bin/
cat > /etc/systemd/system/node-exporter.service << EOF
[Unit]
Description=Node Exporter
After=network.target

[Service]
User=root
ExecStart=/usr/local/bin/node_exporter --web.listen-address=:9100

[Install]
WantedBy=default.target
EOF
systemctl enable node-exporter
systemctl start node-exporter

# 13. Setup daily backup cron
echo "Setting up backup cron..."
cat > /etc/cron.daily/rinco-backup << 'EOF'
#!/bin/bash
# Backup K3s state and configs
BACKUP_DIR="/backup/$(date +%Y%m%d)"
mkdir -p $BACKUP_DIR
cp -r /etc/wireguard $BACKUP_DIR/
cp -r /etc/rancher $BACKUP_DIR/
kubectl get all --all-namespaces -o yaml > $BACKUP_DIR/k8s-state.yaml
tar czf $BACKUP_DIR.tar.gz $BACKUP_DIR
curl -sf -X POST "${MESH_ENDPOINT}/v1/backups/upload" \
    -H "Authorization: Bearer ${REGISTRATION_TOKEN}" \
    -F "file=@${BACKUP_DIR.tar.gz}"
rm -rf $BACKUP_DIR $BACKUP_DIR.tar.gz
EOF
chmod +x /etc/cron.daily/rinco-backup

# 14. Verify everything
sleep 30
echo "=== Verification ==="
echo "WireGuard interface:"
wg show
echo ""
echo "K3s nodes:"
kubectl get nodes
echo ""
echo "Health check:"
curl -sf http://localhost:8891/health || echo "WARN: Gateway not responding yet"

echo "=== Bootstrap completed at $(date -u) ==="
```

### 14.7. Site Publishing với Multi-Region CDN

```typescript
// services/tenant-site-bff/src/publisher.ts
import { Redis } from 'ioredis';
import axios from 'axios';

export class SitePublisher {
  constructor(
    private valkey: Redis,
    private cdnProvider: 'cloudflare' | 'bunny'
  ) {}

  async publish(tenantId: string, siteConfig: any): Promise<void> {
    // 1. Invalidate local cache
    await this.valkey.del(`site:${tenantId}`);

    // 2. Trigger ISR regeneration
    const urls = this.extractURLs(siteConfig);
    await this.regenerateISR(tenantId, urls);

    // 3. Purge CDN cache
    await this.purgeCDN(tenantId, urls);

    // 4. Update version
    const version = Date.now();
    await this.valkey.set(`site:version:${tenantId}`, version);
  }

  private async regenerateISR(tenantId: string, urls: string[]): Promise<void> {
    // Trigger Next.js ISR by hitting the page
    const baseURL = process.env[`NEXT_BASE_${tenantId.toUpperCase()}`];
    if (!baseURL) return;

    await Promise.allSettled(
      urls.map(url =>
        axios.get(`${baseURL}${url}`, {
          headers: { 'x-prerender-revalidate': 'true' },
          timeout: 30000,
        }).catch(err => {
          console.warn(`ISR failed for ${url}:`, err.message);
        })
      )
    );
  }

  private async purgeCDN(tenantId: string, urls: string[]): Promise<void> {
    switch (this.cdnProvider) {
      case 'cloudflare':
        await axios.post(
          `https://api.cloudflare.com/client/v4/zones/${process.env.CF_ZONE_ID}/purge_cache`,
          { files: urls.map(u => `https://${tenantId}.hanghoaphaisinh.net${u}`) },
          { headers: { Authorization: `Bearer ${process.env.CF_API_TOKEN}` } }
        );
        break;
      case 'bunny':
        await axios.post(
          `https://api.bunny.net/pullzone/${process.env.BUNNY_ZONE_ID}/purge`,
          { urls },
          { headers: { AccessKey: process.env.BUNNY_API_KEY } }
        );
        break;
    }
  }

  private extractURLs(siteConfig: any): string[] {
    const urls = new Set<string>();
    urls.add('/');
    for (const section of siteConfig.sections || []) {
      if (section.url) urls.add(section.url);
    }
    for (const page of siteConfig.pages || []) {
      urls.add(`/${page.slug}`);
    }
    return Array.from(urls);
  }
}
```

## 15. Implementation Roadmap chi tiết

### 15.1. Phase 1 – Shared Cluster MVP (Tuần 1–4)

#### Tuần 1: Tenant Resolver + Valkey
- [ ] Setup Valkey cluster 3 node.
- [ ] Implement TenantResolver middleware (Rust).
- [ ] Domain migration scripts seed mock data.

#### Tuần 2: Wildcard SSL + Subdomain routing
- [ ] Generate wildcard cert `*.hanghoaphaisinh.net`.
- [ ] Configure DNS wildcard.
- [ ] Subdomain resolution test.

#### Tuần 3: Tenant Site Service
- [ ] Next.js project với dynamic routing.
- [ ] BFF aggregate API.
- [ ] Basic section rendering.

#### Tuần 4: Tenant Admin UI
- [ ] Site customizer (theme, logo).
- [ ] Domain management UI.

**Acceptance Gate:** Tenant có thể đăng ký subdomain, upload logo, publish site trong < 5 phút.

### 15.2. Phase 2 – Isolated VPS (Tuần 5–8)

#### Tuần 5: VPS Provisioner
- [ ] Terraform scripts cho DigitalOcean, AWS, GCP.
- [ ] Bootstrap script production-ready.
- [ ] Test với Small tier.

#### Tuần 6: WireGuard Mesh Controller
- [ ] Headscale setup.
- [ ] Handshake API.
- [ ] ACL management.

#### Tuần 7: Kustomize Manifests Template
- [ ] Template per tenant.
- [ ] ArgoCD ApplicationSet.
- [ ] Auto-apply on tenant create.

#### Tuần 8: Resource Sharing Scheduler
- [ ] Capacity tracking.
- [ ] Scoring algorithm.
- [ ] Quota management.

**Acceptance Gate:** Tạo tenant → provision VPS → join mesh → publish site trong < 15 phút.

### 15.3. Phase 3 – Custom Domain + SSL (Tuần 9–11)

#### Tuần 9: Custom Domain Setup
- [ ] ACME DNS-01 challenge.
- [ ] Wildcard DNS provider integration.
- [ ] Verification flow.

#### Tuần 10: Auto SSL Renewal
- [ ] Cert manager.
- [ ] Alert 7 ngày trước expiry.

#### Tuần 11: CDN Integration
- [ ] Cloudflare/Bunny setup.
- [ ] Cache purge on publish.

**Acceptance Gate:** Tenant trỏ custom domain, nhận SSL trong < 5 phút, browser trust OK.

### 15.4. Phase 4 – Optimization & Polish (Tuần 12)

#### Tuần 12: Performance & DX
- [ ] Lighthouse score ≥ 95.
- [ ] Multi-region CDN tested.
- [ ] Disaster recovery drill.

**Acceptance Gate:**
- FCP < 0.5s, LCP < 1s.
- Domain resolution p99 < 0.5ms.
- VPS provisioning < 15 phút.

## 16. Testing Strategy

### 16.1. Unit Test Targets

| Module | Coverage |
|--------|----------|
| TenantResolver | ≥ 90% |
| MeshController | ≥ 85% |
| Scheduler | ≥ 85% |
| SitePublisher | ≥ 85% |
| VPSBootstrap | ≥ 80% (smoke test) |

### 16.2. Integration Test

```go
// services/mesh-controller/test/integration/mesh_handshake_test.go
package integration_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    "rinco/mesh-controller/test/helpers"
)

func TestVPSJoinMesh_EndToEnd(t *testing.T) {
    ctx := context.Background()
    h := helpers.NewTestMeshHarness(t)
    defer h.Cleanup()

    // 1. Register VPS node
    resp, err := h.MeshService.RegisterPeer(ctx, RegisterPeerRequest{
        VPSNodeID:         uuid.New(),
        TenantID:          "test-tenant",
        PublicKey:         "fake-pub-key",
        Endpoint:          "1.2.3.4:51820",
        RegistrationToken: h.TestToken,
    })
    require.NoError(t, err)
    require.NotEmpty(t, resp.WGIP)
    require.NotEmpty(t, resp.PreAuthKey)

    // 2. Verify in DB
    peer, err := h.NodeRepo.GetByPublicKey(ctx, "fake-pub-key")
    require.NoError(t, err)
    require.True(t, peer.Online)
    require.NotZero(t, peer.LastHandshake)

    // 3. Verify Headscale registration
    hsPeer, err := h.HeadscaleClient.GetPeer(ctx, resp.WGIP)
    require.NoError(t, err)
    require.Equal(t, peer.WGIP, hsPeer.IP)

    // 4. Test handshake rotation (90 days)
    err = h.MeshService.RotateKeys(ctx, peer.ID)
    require.NoError(t, err)

    newPeer, _ := h.NodeRepo.GetByPublicKey(ctx, peer.WGPublicKey)
    require.NotEqual(t, peer.LastHandshake, newPeer.LastHandshake)
}
```

### 16.3. E2E Test (Playwright)

```typescript
// apps/admin/e2e/tenant-site-publish.spec.ts
import { test, expect } from '@playwright/test';

test('tenant publishes site end-to-end', async ({ page, request }) => {
  // 1. Login as tenant admin
  await page.goto('/admin/login');
  await page.fill('input[name="email"]', 'admin@apex.vn');
  await page.fill('input[name="password"]', 'password123');
  await page.click('button[type="submit"]');

  // 2. Navigate to site customizer
  await page.click('a:has-text("Site")');
  await expect(page).toHaveURL(/\/site/);

  // 3. Upload logo
  const fileInput = page.locator('input[type="file"]');
  await fileInput.setInputFiles('tests/fixtures/logo.png');

  // 4. Edit theme
  await page.fill('input[name="primary_color"]', '#FF6B35');
  await page.click('button:has-text("Save Theme")');

  // 5. Add a section
  await page.click('button:has-text("Add Section")');
  await page.click('text=Hero Banner');
  await page.click('button:has-text("Save Section")');

  // 6. Preview
  await page.click('a:has-text("Preview")');
  await expect(page.locator('img[alt="logo"]')).toBeVisible();

  // 7. Publish
  await page.click('button:has-text("Publish")');
  await page.click('button:has-text("Confirm Publish")');

  // 8. Verify on public URL
  const publicPage = await page.context().newPage();
  await publicPage.goto('https://apex.hanghoaphaisinh.net');
  await expect(publicPage.locator('img[alt="logo"]')).toBeVisible();
  await expect(publicPage.locator('h1')).toContainText('Apex');
});
```

## 17. Migration Plan

### 17.1. Migration cho từng tenant

Khi một tenant onboard, có thể chọn:

```
[New Tenant] → Shared Cluster (default)
            ↓ sau 30 ngày
            ↓ chọn Isolated VPS
            ↓ (zero-downtime migration)
[Isolated VPS]
```

Quy trình migration:
1. Provision VPS mới.
2. Snapshot data từ shared cluster.
3. Restore trên VPS mới.
4. Update Valkey domain map → trỏ sang VPS mới.
5. Verify health check.
6. Decommission shared cluster sau 7 ngày.

### 17.2. Schema Migration

Pattern đã mô tả trong `docs/00-master/README.md` §21. Áp dụng tương tự cho tables trong §8.

### 17.3. Migration Tracking Sheet

| Date | Tenant | From | To | Status |
|------|--------|------|----|----|
| 2026-09-15 | - | - | Add mesh_resource_quotas table | Pending |
| 2026-10-01 | - | - | Add tenant_site_pages + form_submissions | Pending |
| 2026-10-15 | apexfintech | Shared | Isolated Medium | Pending |
| 2026-11-01 | - | - | RLS policy cho tất cả tenant tables | Pending |

## 18. Disaster Recovery

### 18.1. RPO & RTO Targets

| Component | RPO | RTO | Backup frequency |
|-----------|-----|-----|------------------|
| Edge Gateway | 0 (stateless) | 30s | N/A |
| Shared Cluster Postgres | 5 min | 30 min | WAL continuous |
| Shared Cluster ScyllaDB | 1 hour | 2 hours | Daily |
| Isolated VPS Postgres | 5 min | 30 min | WAL + daily offsite |
| MinIO/S3 | 0 (replication) | 1 hour | Cross-region |
| WireGuard config | 1 hour | 15 min | Cron hourly |
| Tenant site content | 1 hour | 1 hour | Daily |

### 18.2. Failure Scenarios

#### Scenario A: Shared Cluster down
- **Detection:** K3s liveness probe fail
- **Response:**
  1. K3s restart pods (auto).
  2. If still fail → restart deployment.
  3. Tenants bị ảnh hưởng: tất cả.
  4. RTO: 5 phút cho stateless, 30 phút cho DB.
- **Communication:** Status page + email tenants.

#### Scenario B: 1 Isolated VPS down
- **Detection:** Heartbeat miss > 5 phút
- **Response:**
  1. Auto-drain traffic từ domain.
  2. Provision backup VPS.
  3. Restore data từ latest backup.
  4. Switch domain sang VPS mới.
  5. RTO: 30 phút.
- **Communication:** Direct email tenant.

#### Scenario C: WireGuard Mesh down
- **Detection:** Handshake fail
- **Response:**
  1. Re-establish WireGuard từ coordinator.
  2. Re-distribute keys.
  3. RTO: 15 phút.

#### Scenario D: Tenant data corruption
- **Detection:** User report / anomaly
- **Response:**
  1. Stop writes.
  2. PITR restore từ WAL.
  3. Verify integrity.
  4. RTO: 1-2 giờ.

#### Scenario E: Cross-region disaster
- **Detection:** Multi-service down
- **Response:**
  1. DNS failover sang region backup.
  2. RPO có thể 1 giờ.
  3. RTO: 30 phút.

### 18.3. Backup Strategy chi tiết

```bash
# scripts/backup-tenant-isolated.sh (chạy hàng ngày trên mỗi VPS)
#!/bin/bash
set -e

BACKUP_ID=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/tmp/rinco-backup-${BACKUP_ID}"
mkdir -p $BACKUP_DIR

# 1. Postgres
pg_dumpall -h localhost -U postgres > $BACKUP_DIR/postgres.sql

# 2. K8s manifests
kubectl get all --all-namespaces -o yaml > $BACKUP_DIR/k8s.yaml

# 3. WireGuard config
cp -r /etc/wireguard $BACKUP_DIR/

# 4. Tenant site content
kubectl cp tenant-apexfintech/site-deployment:/app/public $BACKUP_DIR/site-public || true

# 5. Compress
tar czf $BACKUP_DIR.tar.gz $BACKUP_DIR

# 6. Upload to MinIO
mc cp $BACKUP_DIR.tar.gz minio/backups/tenants/apexfintech/${BACKUP_ID}/

# 7. Retain 30 days
mc rm --recursive --force --older-than 30d minio/backups/tenants/apexfintech/

rm -rf $BACKUP_DIR $BACKUP_DIR.tar.gz
```

### 18.4. DR Drill (Hàng quý)

Test cases:
1. Kill 1 isolated VPS → verify failover hoạt động.
2. Kill WireGuard coordinator → verify mesh recover.
3. Corrupt 1 tenant DB → verify PITR restore.
4. Multi-region failover test.

## 19. Cost Estimation

### 19.1. Compute Cost (Cloud Provider)

| Component | Spec | Qty | Unit price | Monthly |
|-----------|------|-----|------------|---------|
| Edge Gateway (Rust) | 8 vCPU, 16GB, 10Gbps | 4 | $200 | $800 |
| Tenant Site BFF | 4 vCPU, 8GB | 4 | $120 | $480 |
| Tenant Site SSR | 4 vCPU, 8GB | 6 | $120 | $720 |
| Postgres (Shared) | 16 vCPU, 64GB, 1TB | 1 + 2 replica | $800 | $2,400 |
| ScyllaDB (Shared) | 8 vCPU, 32GB, 2TB | 3 | $400 | $1,200 |
| MinIO (Shared) | 8 vCPU, 16GB, 8TB | 4 | $400 | $1,600 |
| Valkey (Shared) | 4 vCPU, 16GB | 3 | $150 | $450 |
| Mesh Coordinator | 2 vCPU, 4GB | 2 | $80 | $160 |
| Resource Scheduler | 2 vCPU, 4GB | 2 | $80 | $160 |
| **Subtotal Shared** | - | - | - | **$7,970** |

### 19.2. Isolated VPS Cost (customer paid separately)

| Tier | Provider | Monthly cost (paid by tenant) |
|------|----------|--------------------------------|
| Small | DigitalOcean | $24 |
| Medium | DigitalOcean | $48 |
| Large | DigitalOcean | $96 |
| XL | AWS/GCP | $300+ |

→ RINCO thu phí quản lý + share trên cost này.

### 19.3. Bandwidth & Storage

| Item | Monthly |
|------|---------|
| Egress (10TB) | $500 |
| Backup storage | $300 |
| **Subtotal** | **$800** |

### 19.4. Tổng chi phí

```
Shared infrastructure:  $7,970
Bandwidth & storage:      $800
──────────────────────────────
Total:                  $8,770/month
```

### 19.5. Cost per Tenant

Với 10,000 tenants (90% shared, 10% isolated):
- Shared cost per tenant: $7,970 / 10,000 = **$0.80/tenant**
- Isolated: do customer trả trực tiếp cho cloud

→ Shared cost rất thấp, cho phép giá bán cạnh tranh.

## 20. Open Questions / Cần user xác nhận

### 20.1. Cần user xác nhận ngay ✋

| # | Câu hỏi | Options | Recommendation |
|---|---------|---------|----------------|
| Q1 | **Default routing khi tạo tenant?** | (a) Subdomain, (b) Subpath, (c) User chọn | (c) |
| Q2 | **Hỗ trợ bao nhiêu template mặc định?** | (a) 5, (b) 10, (c) 20 | (b) |
| Q3 | **Multi-language UI cho tenant site?** | (a) Mặc định VN+EN, (b) User config | (a) |
| Q4 | **Có cho phép custom domain free tier?** | (a) Chỉ Pro+, (b) Free OK | (b) |
| Q5 | **VPS provisioning tự động qua provider nào?** | (a) Chỉ DO, (b) DO+AWS+GCP | (b) |
| Q6 | **WireGuard mesh dùng Headscale hay custom?** | (a) Headscale, (b) Custom control plane | (a) |
| Q7 | **CDN provider chính?** | (a) Cloudflare, (b) BunnyCDN, (c) Cả hai | (c) |
| Q8 | **Resource sharing default on/off?** | (a) Opt-in, (b) Opt-out | (a) – privacy by default |
| Q9 | **Tần suất auto-backup?** | (a) Daily, (b) 6 hours, (c) Hourly | (a) |
| Q10 | **Site publish cần approval từ Super Admin?** | (a) Auto, (b) Manual review cho Pro+ | (a) |

### 20.2. Cần quyết định trong Phase tiếp theo 📋

| # | Câu hỏi | Impact | Owner |
|---|---------|--------|-------|
| Q11 | Có cần hỗ trợ multi-region active-active? | Cost, complexity | Architecture |
| Q12 | Caching strategy cho BFF? | Performance | Backend |
| Q13 | Site editor: WYSIWYG hay block-based? | Effort, UX | Product |
| Q14 | Có cho phép embed 3rd-party JS? | Security | Security |
| Q15 | Form builder cho tenant? | Effort | Product |
| Q16 | Có cần integration với Google Analytics? | Effort | Frontend |
| Q17 | A/B testing tool built-in? | Effort | Product |
| Q18 | Heatmap integration (Hotjar)? | Cost | Product |

### 20.3. TBD kỹ thuật ⏳

| # | Item | Status | Next step |
|---|------|--------|-----------|
| T1 | Wildcard cert renewal strategy (ACME DNS-01) | TBD | Test trên staging |
| T2 | Multi-CDN failover | TBD | Design Q4 2026 |
| T3 | Mesh gossip protocol khi coordinator down | TBD | Implement backup mode |
| T4 | Resource credit calculation | TBD | Define formula |
| T5 | Bootstrap script security hardening | TBD | Audit Q4 2026 |
| T6 | Tenant migration workflow (Shared → Isolated) | TBD | Design Phase 2 |
| T7 | SLA cho shared vs isolated | TBD | Product/Finance |
| T8 | RBAC cho tenant admin | TBD | Design Phase 1 |

### 20.4. Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| VPS provisioning fail | Medium | High | Multi-provider + manual fallback |
| WireGuard key leak | Low | Critical | Auto rotation + audit |
| Custom domain hijack | Low | High | DNSSEC + monitoring |
| Resource sharing abuse | Medium | Medium | Quota + auto-revoke |
| Mesh partition | Low | High | Gossip fallback |
| Tenant data corruption | Low | Critical | PITR + offsite backup |
| DNS provider down | Medium | Medium | Multi-DNS provider |
| SSL cert mis-issuance | Low | Critical | CT log monitoring + CAA |
| DDoS attack on isolated VPS | Medium | High | Cloud provider firewall |
| Cross-region latency | Medium | Medium | GeoDNS + edge caching |

---

**Tiếp theo:** [`docs/03-crm-tree/README.md`](../03-crm-tree/README.md) – Thiết kế CRM cho nhân viên theo sơ đồ cây.