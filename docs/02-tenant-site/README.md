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
43. Social feed embed (Facebook, Instagram).
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

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-TN-01 | Domain resolution < 0.5ms | Valkey lookup p99 |
| AC-TN-02 | Site load < 1s | Lighthouse FCP < 1s |
| AC-TN-03 | VPS provisioning < 10 phút | Time to ready |
| AC-TN-04 | WireGuard rekey tự động mỗi 90 ngày | Cron |
| AC-TN-05 | Resource sharing không ảnh hưởng source tenant | SLA |

---

**Tiếp theo:** [`docs/03-crm-tree/README.md`](../03-crm-tree/README.md) – Thiết kế CRM cho nhân viên theo sơ đồ cây.