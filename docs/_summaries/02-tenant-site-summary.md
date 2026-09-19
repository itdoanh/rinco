# Tenant-Site + Isolated VPS – Summary

## Overview

The Tenant-Site subsystem gives every RINCO tenant its own branded homepage (e.g. `apex.hanghoaphaisinh.net` or a fully custom domain like `landing.apexcorp.vn`), with the freedom to run either in a Shared Cluster or on an Isolated VPS that the tenant purchases and controls. Tenant VPS nodes join a peer-to-peer WireGuard mesh coordinated by a central Mesh Coordinator (Headscale + Go control plane), enabling full-mesh intra-tenant connectivity, automatic provisioning through Cloud APIs, and a Distributed Resource Sharing Scheduler that monetises idle CPU/RAM/GPU via a credit system. The site itself is a Next.js dynamic-routed app served via a TypeScript + Bun `tenant-site-bff`, cached in Valkey, with Valkey-backed domain resolution that returns a tenant in < 0.5 ms.

## Goals & Principles

| ID | Goal | Measurement |
|----|------|-------------|
| TN-1 | Every tenant has its own homepage | 100 % tenants have URL |
| TN-2 | Domain resolution < 0.5 ms | Valkey lookup |
| TN-3 | Isolated VPS self-managed | Tenant owns lifecycle |
| TN-4 | Secure mesh network | WireGuard mTLS |
| TN-5 | Utilise idle resources | ≥ 30 % utilisation |

Principles: tenant autonomy, absolute isolation (shared still uses RLS + per-tenant data dir), flat peer mesh, auto-scale on new VPS join.

## Two Deployment Modes

### A. Shared Cluster (default)
Central Control Plane + Edge GW (Rust/Go) + Go/Rust/Py services + PostgreSQL + ScyllaDB + ClickHouse. Tenants access only via Gateway; isolation enforced by RLS and per-tenant data directories.

### B. Isolated VPS (tenant-owned)
Single-node K3s on a tenant-provisioned VPS, running Edge GW + App Services + local PostgreSQL/ScyllaDB. The VPS joins the central WireGuard mesh so backups, replication, cross-tenant resource sharing and remote admin remain possible.

## Mesh Coordinator Services

- `mesh-controller` (Go + Headscale) — WireGuard keys, ACL, peer registry
- `resource-scheduler` (Go) — workload placement by load + credit
- `replication-manager` (Go) — cross-cluster backup sync
- `vps-provisioner` (Go + Terraform/Pulumi) — automated VPS creation via Cloud APIs

## Multi-Tenant URL Routing (3 Models)

### 3.1 SUBPATH — Shared Domain
URL: `hanghoaphaisinh.net/apexfintech`. All tenants share one root domain. The Edge Gateway parses the URL subpath and looks up `tenant_id` in Valkey.
- **Pros:** no DNS setup, fastest onboarding.
- **Cons:** weaker SEO, no separate brand.

### 3.2 SUBDOMAIN — Wildcard DNS
URL: `apex.hanghoaphaisinh.net`. A DNS wildcard `*.hanghoaphaisinh.net` points to the Edge Gateway. The Gateway parses the `Host` header and resolves the tenant.
- **Pros:** stronger brand, memorable.
- **Cons:** depends on the parent domain.

### 3.3 CUSTOM DOMAIN — Tenant-Owned
URL: `landing.apexcorp.vn` (set by tenant). Tenant creates a CNAME → `edge.rinco.app` or an A record → Edge IP. Let's Encrypt auto-issues SSL via ACME DNS-01 challenge.
- **Pros:** full brand independence.
- **Cons:** tenant must configure DNS.

### 3.4 Domain Map Cache (Valkey)
Key: `domain:apex.hanghoaphaisinh.net`
Value: `{ tenant_id, routing_type, ssl_status, expires_at }`
TTL: 3600 s — lookup p99 < 0.5 ms, replicated to every Gateway node, invalidated on tenant config change.

### 3.5 SSL / TLS
- Wildcard cert for subpath/subdomain (`*.hanghoaphaisinh.net`).
- Per-tenant Let's Encrypt via DNS-01 for custom domains.
- Auto-renewal 30 days before expiry.
- OCSP Stapling enabled.

## Isolated VPS Strategy

### Hardware Tiers

| Profile | CPU | RAM | Disk | Bandwidth |
|---------|-----|-----|------|-----------|
| Small | 2 vCPU | 4 GB | 80 GB SSD | 100 Mbps |
| Medium | 4 vCPU | 8 GB | 160 GB SSD | 500 Mbps |
| Large | 8 vCPU | 16 GB | 320 GB NVMe | 1 Gbps |
| XL | 16 vCPU | 32 GB | 640 GB NVMe | 2 Gbps |

### Bootstrap Script (production)
1. Install K3s with `--disable=traefik --disable=servicelb`.
2. Install WireGuard + generate keypair.
3. Register public key with `mesh.rinco.app/v1/nodes/register` using a short-lived bearer token.
4. Receive assigned WireGuard IP in `100.64.0.0/10` (CGNAT range) + mesh config.
5. Pull & apply tenant manifests from `mesh.rinco.app/manifests/<TENANT_ID>.yaml`.
6. Install Vector agent → ClickHouse log shipping.
7. Install Prometheus `node_exporter`.
8. Install daily backup cron → push to MinIO.

### Multi-VPS Expansion
Tenant can request additional VPSes (DB / App / Media). Mesh Controller auto-wires WireGuard between them; Linkerd/Istio service mesh enables mTLS internal traffic.

## WireGuard Mesh Setup

### Topology
Coordinator sits at the top; each tenant VPS connects to the Coordinator and optionally to peer VPSes of the same tenant. Same-tenant peers may form a full mesh; cross-tenant traffic is limited to the Coordinator only.

### Implementation
- **Headscale** (open-source Tailscale control plane) as the coordinator.
- Each VPS runs the `headscale` client.
- Pre-Auth Keys have a 10-minute TTL.
- ACL: VPS A sees only the Coordinator + same-tenant peers.

### Security
- WireGuard keys rotate every 90 days.
- PSK (Pre-Shared Key) layer atop the public key.
- Every handshake is audit-logged.

### eBPF Mesh Acceleration
eBPF program in kernel pushes mesh packets directly between NICs, bypassing the TCP/IP stack for intra-mesh traffic. Inter-region mesh latency drops to ~5 ms.

## Distributed Resource Sharing

### Scheduler Algorithm
```
1. candidates = online VPS in workload.region
2. filter by capacity (cpu_free ≥ req, ram_free ≥ req)
3. score by load: score = (1-cpu_util)*0.4 + (1-ram_util)*0.4 + (1-net)*0.2
4. pick max-score VPS, reserve capacity
```

Score weighting: load 0.4, credit rate 0.3, regional proximity 0.2, reliability 0.1.

### Workload Eligibility

| Workload | Shareable | Why |
|----------|-----------|-----|
| Video meeting render | ✅ | Stateless, CPU/GPU heavy |
| AI inference (RAG) | ✅ | Stateless, GPU heavy |
| Backup & replication | ✅ | Background job |
| Email/Push send | ✅ | Queue-based |
| Real-time chat | ❌ | Latency-sensitive |
| CRM core | ❌ | Sensitive data |

### Trust Model
- Sandboxed container (seccomp + AppArmor).
- Workload receives masked data only — no raw data access.
- Every execution logged to Coordinator.
- Tenant A lends resources → earns credit.

## Tenant-Site Features (15+)

### Site Management
1. Create site per new tenant (template selection).
2. 10 built-in templates (Real Estate, Finance, Retail, Education, Default, …).
3. Theme customisation (primary / secondary / accent / background colors, font family & sizes).
4. Logo + favicon upload.
5. Add / edit / delete sections.
6. Drag-and-drop section ordering.
7. Live preview before publish.
8. Publish / unpublish + scheduled publish.
9. Multi-language (VN / EN / JP / KR / ZH) with translation management.
10. Custom-domain setup wizard + auto-renew SSL.
11. Subdomain auto-provisioned at tenant creation.
12. Subpath routing configuration.
13. SEO config (meta title, description, og:image).
14. Auto-generated `sitemap.xml` + custom `robots.txt`.
15. 301 redirect management.
16. Custom 500 error page.
17. Maintenance mode page.
18. Cookie consent banner (GDPR / PDPA compliant).
19. Page-speed optimisation (image lazy-load, critical CSS).
20. Optional AMP version.
21. PWA support (`manifest.json`, service worker).

### Content Modules
- Hero banner with CTA
- About company
- Services / Products grid
- Team member list
- Testimonials carousel
- News / Blog system (CRUD, categories, tags, comments, search, RSS feed)
- FAQ accordion
- Pricing table
- Multi-step contact form
- Contact info (address, phone, email, socials)
- Google Maps embed
- Newsletter signup
- Video embed (YouTube, Vimeo)
- Document download
- Blog version history

### Domain & SSL
- Custom-domain wizard
- CNAME / A record verification
- Auto Let's Encrypt via DNS-01
- Wildcard SSL for `*.hanghoaphaisinh.net`
- Force HTTPS + HSTS preload
- SSL expiry alerts
- Multi-domain on a single site
- Domain transfer between tenants
- Domain ownership verification
- CAA record config
- SSL transparency monitoring (CT log)
- Domain health check

### Isolated VPS
- Request new VPS (tier / region / OS)
- Auto-provision via DigitalOcean / AWS / GCP / Azure / Vultr APIs
- K3s auto-install
- WireGuard auto-join mesh
- DNS auto-update
- Monitoring auto-attach (Vector, node_exporter)
- Backup auto-config (daily)
- Firewall rules auto-apply
- Failover to backup VPS
- Multi-VPS cluster setup with service mesh
- Resource sharing toggle (allow / disallow)
- Resource-sharing credit dashboard
- Manual scale up/down
- Auto-scale by metric
- Scheduled maintenance window
- SSH key management
- K3s upgrade notifications
- Disaster-recovery drill
- Graceful VPS decommission

### WireGuard Mesh
- Topology graph view
- Add / remove peer
- WireGuard key rotation
- Per-peer bandwidth view
- Per-peer latency view
- ACL management
- Mesh health check
- Offline peer alerts
- Automatic peer reconnect
- Bandwidth limit per peer
- Priority routing (VPN traffic)
- Split-tunnel config
- Mesh event log
- Mesh capacity planning
- Mesh performance dashboard

### Resource Sharing
- Opt-in / opt-out per tenant
- Workload whitelist (render / AI inference)
- CPU/RAM/GPU quotas via `mesh_resource_quotas` table
- Credit system (lender earns credits)
- Audit log of every shared job
- SLA monitoring
- Anomaly detection on shared jobs
- Auto-revoke on abuse

### Advanced
- Multi-region active-active
- CDN integration (Cloudflare + BunnyCDN)
- Edge workers (Cloudflare Workers / Vercel Edge)
- A/B test on site
- Heatmap (Hotjar-style)
- Session recording
- Form submission analytics
- Conversion funnel
- Custom event tracking
- Real-time visitor counter
- Geo-blocking (per country)
- IP whitelist / blacklist
- Per-IP rate limit
- Bot protection (CAPTCHA + Wasm)
- Audit log of every site change

## Database Schema Highlights

- `tenant_sites` — template, sections JSONB, theme JSONB, custom_css, favicon_url, primary_locale, supported_locales TEXT[], seo_config JSONB, analytics_config JSONB, is_published.
- `tenant_vps_nodes` — tenant_id, hostname, public_ip INET, private_ip INET, wg_public_key, wg_endpoint, region, tier (`SHARED` / `ISOLATED_SMALL` / `_MEDIUM` / `_LARGE` / `_XL`), cpu_cores, ram_gb, disk_gb, bandwidth_mbps, status (`PROVISIONING` / `ONLINE` / `OFFLINE` / `MAINTENANCE`), k3s_version, last_heartbeat, allow_resource_sharing.
- `mesh_nodes` — wg_public_key UNIQUE, wg_ip INET, endpoint, allowed_peers UUID[], online, last_handshake, bytes_sent, bytes_received.
- `resource_sharing_jobs` — source_tenant_id, target_vps_id, workload_type (`VIDEO_RENDER` / `AI_INFERENCE` / `BACKUP` / `NOTIFICATION`), cpu_required, ram_required, gpu_required, status (`QUEUED` / `RUNNING` / `COMPLETED` / `FAILED`), result_url, credit_amount DECIMAL.
- `tenant_site_pages` (v1.1) — slug, page_type (`BLOG` / `FAQ` / `LANDING` / `CUSTOM`), locale, title, excerpt, content (Markdown or JSON blocks), meta_title, meta_description, og_image_url, status (`DRAFT` / `PUBLISHED` / `ARCHIVED`), tags TEXT[], category, UNIQUE(tenant_id, slug, locale) + GIN(tags).
- `tenant_site_form_submissions` (v1.1) — form_id, data JSONB, ip_address INET, user_agent, referer, utm JSONB, status (`NEW` / `PROCESSED` / `SPAM`).
- `mesh_resource_quotas` (v1.1) — cpu_quota_cores, ram_quota_gb, gpu_quota_hours_per_month, network_quota_gb_per_month, bandwidth_priority, enabled.

### Valkey Keys
```
domain:<hostname>                  → {tenant_id, routing_type, ssl_status}
tenant:<tenant_id>:site-cache      → sections JSON
vps:<vps_id>:capacity              → {cpu_used, cpu_total, ram_used, ram_total}
mesh:peer:<wg_public_key>          → {tenant_id, vps_id, wg_ip}
```

## API Surface

### Tenant Site (public)
```
GET    /api/site/v1/:tenant_slug
GET    /api/site/v1/:tenant_slug/sections/:section_id
POST   /api/site/v1/:tenant_slug/contact
POST   /api/site/v1/:tenant_slug/newsletter
GET    /api/site/v1/:tenant_slug/blog
GET    /api/site/v1/:tenant_slug/blog/:slug
GET    /api/site/v1/:tenant_slug/locales/:locale.json
```

### Tenant Admin (per-tenant scope)
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

## Edge Gateway – Tenant Resolver (Rust + io_uring)

The resolver flow:
1. Parse `host` + `path` into a cache key (subpath → `domain:<host>:<seg>`; subdomain → `domain:<host>:<sub>`; custom → `domain:<host>`).
2. Local Moka cache (< 0.5 ms).
3. Valkey fallback (< 5 ms).
4. In-process LRU static fallback.
5. Update local cache on hit.
6. `invalidate()` on tenant config change.

## CI/CD & Deployment

- GitOps: GitHub → ArgoCD (per tenant ApplicationSet) → K3s cluster.
- Manifest structure: `manifests/tenant-<slug>/{namespace,deployment,service,ingress,configmap,secret}.yaml`.
- Provisioning flow: Super Admin creates tenant → `tenant-manager` calls K3s API → namespace + service account → apply default manifests → Valkey domain mapping → Let's Encrypt cert → welcome email.
- Rollback: every deploy stores Git revision; ArgoCD keeps history; manual rollback via Admin UI.

## DR & Cost

RPO/RTO: Edge GW 0/30 s; shared Postgres 5 min/30 min; shared ScyllaDB 1 h/2 h; isolated Postgres 5 min/30 min + daily offsite; MinIO cross-region replication 0/1 h; WireGuard config 1 h/15 min; site content 1 h/1 h.

Scenarios: shared cluster down, isolated VPS down, WireGuard mesh down, tenant data corruption (PITR), cross-region disaster (DNS failover).

Backup script `backup-tenant-isolated.sh` ships daily Postgres dump + K8s manifests + WireGuard config + site content to MinIO with 30-day retention.

Cost: shared infra ~$8,770 / month → ≈ $0.80 / shared tenant at 10 k tenants. Isolated VPS tier is paid by tenant ($24 small → $300+ XL).
