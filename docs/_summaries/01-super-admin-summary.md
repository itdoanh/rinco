# Super-Admin Portal – Summary

## Overview

The Super-Admin Portal is the top-level control plane of RINCO, used by internal operators to monitor, configure, and govern every tenant, account, log, traffic stream, and revenue metric in near-real-time (1 s – 1 min latency). It is delivered as a Dark Admin — no public DNS, no public IP, accessed only through a WireGuard mesh + Single-Packet Authorization (SPA) handshake, then a browser SPA. The stack combines a Go + Echo `admin-gateway` hidden on port 8891, a TypeScript + Bun `admin-bff` for view aggregation, Go microservices (`tenant-manager`, `audit-log`, `realtime-dashboard`, `feature-flags-service`, `resource-allocator`, `billing-service`, `impersonation-service`), PostgreSQL for state, ScyllaDB for distributed audit logs, ClickHouse + Prometheus + Vector for telemetry, NATS JetStream as the realtime bus, and Valkey for caching. Authentication is PASETO + WebAuthn (FIDO2 / YubiKey), with a 2-of-3 Quorum signing ceremony for dangerous actions.

## Goals & Principles

| ID | Goal | Measurement |
|----|------|-------------|
| ADM-1 | Real-time monitoring of entire system | p99 update < 1 s |
| ADM-2 | Tenant CRUD + configuration | < 200 ms |
| ADM-3 | Early anomaly detection | < 30 s |
| ADM-4 | No public surface | No DNS A/CNAME pointing at admin |
| ADM-5 | Full audit trail | 100 % actions carry `trace_id` |

Principles: observe-then-act, multi-party for dangerous actions, dark-by-default, read-only by default, least privilege.

## Architecture & Ports

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

| Internal Port | Service | External |
|---------------|---------|----------|
| 8891 | admin-gateway | ❌ (WireGuard only) |
| 8892 | admin-bff | ❌ |
| 8893 | audit-query | ❌ |
| 5432 | postgres-admin | ❌ (private subnet) |
| 9090 | prometheus-admin | ❌ |

Realtime pipeline: ClickHouse + Prometheus + ScyllaDB + Vector + service health + K3s events + Valkey pub/sub → NATS JetStream topic `admin.metrics` → `realtime-dashboard` → SSE → Admin UI. SSE event types: `metric`, `alert`, `audit`, `tenant_status`. Backpressure: max 1000 events/s/connection, drop-oldest with audit.

## RBAC Matrix

| Role | Permissions |
|------|-------------|
| **OWNER** | Full powers incl. billing, FIDO2 management |
| **SRE_ADMIN** | Infrastructure monitoring, restart service, scale |
| **SECURITY_ADMIN** | Lock tenant, view audit, ban IP |
| **SUPPORT_ADMIN** | Impersonate tenant (max 24 h) |
| **FINANCE_ADMIN** | Billing, refund, subscription |
| **READONLY_VIEWER** | View dashboards, no edits |

### Multi-Party Authorization (Quorum)

- 2-of-3 YubiKey signing required for destructive actions (delete tenant, mutate global gateway, change DNS master).
- 5-minute signing window.
- Every action carries a YubiKey signature + mandatory reason.

### FIDO2 / YubiKey

- Every Super Admin must register ≥ 2 YubiKeys (primary + backup).
- WebAuthn challenges via PASETO + signed nonce.
- Failure handling: lock account 30 min after 5 fails, alert SECURITY_ADMIN, log every attempt.

## Admin Pages / Features

### 1. Dashboard Overview (live)
Card metrics (tenant count, user count, revenue, traffic), 1 s-refresh realtime chart, sparkline LiveMetricCards, top-tenants tables (by traffic, by revenue), system error heatmap, world map traffic, WebSocket connections count, DB connection pool status, cache hit/miss, p50/p95/p99 latency per endpoint, error rate per service, request rate per service, ScyllaDB write rate, ClickHouse query rate, NATS JetStream lag, active background jobs, queue depth.

### 2. Tenants List + Detail
DataTable with filter/search/bulk actions, tabs: Overview, Users, Domains, Billing, Resources, Audit. TenantDetailSheet slide-over.

### 3. Tenant Lifecycle Management
- Create tenant (name, slug, plan, region, template)
- Lock (soft — read-only for users)
- Freeze (no user access)
- Hard delete after 30-day grace
- Migrate between clusters (zero-downtime, idempotency key)
- Backup & restore config + DB
- Clone tenant as template
- Update plan (Free / Pro / Business / Enterprise)
- Per-tenant feature flags, limits (max_users, max_storage_gb, max_requests_per_sec)
- Domain mapping (subpath / subdomain / custom)
- SSL/TLS auto-renew Let's Encrypt
- IP whitelist per tenant
- Retention config (logs, video, recording)
- GDPR data export
- Tenant notes (internal)
- Custom branding (logo, colors)

### 4. Super-Admins Management
Create (email + YubiKey invitation), disable, reset password (YubiKey confirmation), session kill, personal audit log, role assignment, register new YubiKey, backup YubiKey, PASETO key rotation, WebAuthn challenge cache management, active sessions list, device fingerprint, abnormal-IP login alert, periodic password change enforcement, 2FA backup codes.

### 5. Live Logs
xterm-style LogStream with filter by level, full-text search via ClickHouse, saved per-admin searches, PII redaction, download log archive by day, log-shipping audit (who viewed which log), alert rules from log patterns, log correlation grouping by `trace_id`.

### 6. Traces Explorer
Jaeger UI embed, flamegraph, span details, P95 per trace.

### 7. Metrics
Grafana-style query builder, per-endpoint SLO dashboard.

### 8. Alerts & AI SRE
- Alert rules (CPU, RAM, error rate, latency)
- Alert channels (Telegram, Slack, SMS, Email)
- On-call rotation (P0/P1/P2)
- Escalation policy
- Alert silence (mute window)
- **AI RCA** from alert (Code-LLM auto-investigate, target < 3 s)
- **AI Hotfix Proposal** (auto-PR generation)
- **AI weekly incident report**
- **AI capacity planning**
- **AI anomaly detection** (traffic / error spikes)
- **AI security threat scoring** per IP / per request
- **AI cost optimization** (scale down / right-size)
- **AI tenant health score**
- Alert dedup & grouping (Sentry-style)

### 9. Audit Log
Filterable virtual-scroll table, query builder (actor, action, target, time range), CSV/JSON export for compliance, retention 5 years for P0 actions, ScyllaDB Merkle hash chain for tamper-evidence.

### 10. Quorum Requests
Pending / Approved / Rejected lists, QuorumDialog multi-step with YubiKey signature.

### 11. Billing
Subscription list, trial period (14 days), auto-renewal, manual refund, usage tracking (API calls, storage, AI calls), invoice generation, tax config (VAT by country), discount codes, payment methods (Stripe / VNPay / Momo), billing webhooks.

### 12. Resources / Cluster Manager
Cluster / node list, rolling restart, manual + rule-based scaling, node drain / cordon, GitOps/ArgoCD config deploy, rollback deployment, K3s cluster health, WireGuard mesh peer status (online/offline), distributed resource sharing dashboard (idle VPS).

### 13. System Status
Cluster health, mesh status, internal status page (private).

### 14. Settings
Global config, branding, notification preferences.

### 15. Notification Center (v1.1)
In-app inbox, NotificationComposer, templates CRUD, A/B test content, scheduled sends, open/click rate analytics, per-role/region/tenant targeting, bypass quiet-hours for P0, digest mode (daily/weekly).

### 16. Tenant Impersonation
SUPPORT_ADMIN starts session (max 24 h hard expiry), red ImpersonationBanner warning top bar, full audit of actions performed, forced logout after expiry.

### 17. Mass Broadcast Notification
Notify all tenants (e.g. maintenance notice), targeting by role / region / tenant group.

### 18. GDPR Right-to-be-forgotten
Erase end-user data per tenant.

### 19. Data Residency
Choose region (VN, SG, US) for tenant data storage.

### 20. Webhooks
Outbound (admin subscribes to events) + Incoming (from external systems).

### 21. API Key Management
Per-tenant API keys with rate limit.

### 22. Maintenance Mode
Global read-only / maintenance window scheduler, change-request workflow, multi-level approval chain.

### 23. Slack / Teams / Telegram Bot Integration
Operational chatops from the same role gating.

### 24. Live Notification Templates (v1.1)
Categories: CRITICAL / WARNING / INFO / MARKETING. Channels: IN_APP / EMAIL / TELEGRAM / PUSH / SMS. `text/template` body with typed `variables` schema. `notification_dispatch_log` tracks pending / sent / failed / bounced / opened / clicked.

## Highlighted Capability Set

- **FIDO2 / WebAuthn** login flow (`BeginWebAuthnLogin`, `FinishWebAuthnLogin`), PASETO v4 8 h tokens, 5-min challenge TTL, lockout after 5 fails, `last_login_ip` INET column.
- **Quorum** 2-of-3 signing: `quorum_requests` table with `collected_signatures` JSONB, atomic append, initiator ≠ signer rule, auto-execute on threshold.
- **Tenants**: 4-state lifecycle (`ACTIVE` / `LOCKED` / `FROZEN` / `DELETING`), isolation mode (`SHARED` / `ISOLATED`), `version` column for optimistic locking.
- **Audit**: ScyllaDB table partitioned by `tenant_id`, clustering by `timestamp DESC`, `signature` column from YubiKey, 5-year retention for P0.
- **Notifications**: Template engine + multi-channel dispatcher with quiet-hours (default 22:00 – 07:00 user TZ), exponential backoff, A/B testing with auto-revert on skew, Telegram / Email / Push / SMS / PagerDuty fallback chain.
- **Realtime**: SSE on `/ws/admin/realtime`, NATS JetStream consumer, 1000 events/s backpressure with priority queue, Last-Event-ID resume.
- **Dark Admin**: SPA → ephemeral WireGuard port → PASETO challenge → WebAuthn → session cookie (httpOnly + SameSite=Strict + Secure), bind only on `wg0`, no DNS A/CNAME, rate limit 60 req/min/IP + 1000 req/h/session.
- **AI SRE**: Code-LLM RCA, hotfix PR auto-propose (human-in-the-loop), capacity planning, anomaly baseline, threat scoring, tenant health scoring.

## Database Schema (core tables)

- `super_admins` — id, email UNIQUE, password_hash, role CHECK, status, mfa_enabled, last_login_at, last_login_ip INET, failed_login_count, locked_until.
- `super_admin_webauthn` — credential_id UNIQUE, public_key BYTEA, counter, transports TEXT[], aaguid.
- `tenants` — slug UNIQUE, plan CHECK, status, region, cluster_id, vps_node_id, isolation_mode CHECK, max_users, max_storage_gb, max_requests_per_sec, feature_flags JSONB, retention_days, custom_branding JSONB.
- `tenant_domains` — routing_type (`SUBPATH` / `SUBDOMAIN` / `CUSTOM`), target_path, ssl_status (`PENDING` / `ACTIVE` / `FAILED`), verified.
- `audit_log` — ScyllaDB table, primary key `(tenant_id, timestamp, id)`, signature column, payload JSON.
- `feature_flags` — flag_key PK, default_enabled, rollout_percentage, tenant_overrides JSONB.
- `quorum_requests` — required_signatures, collected_signatures JSONB, status (`PENDING` / `APPROVED` / `REJECTED` / `EXPIRED`), expires_at.
- `notification_templates` (v1.1) — code UNIQUE, category, channel, subject, body_template (Go `text/template`), variables JSONB.
- `notification_dispatch_log` (v1.1) — recipient_type, channel, status (`PENDING` / `SENT` / `FAILED` / `BOUNCED`), opened_at, clicked_at.
- `impersonation_sessions` (v1.1) — admin_id, tenant_id, reason, started_at, expires_at (+24 h), actions_performed JSONB.

## API Surface (REST + Connect-RPC)

```
POST   /api/admin/v1/auth/login            # PASETO challenge
POST   /api/admin/v1/auth/webauthn/init
POST   /api/admin/v1/auth/webauthn/finalize
POST   /api/admin/v1/auth/logout
GET    /api/admin/v1/auth/sessions

GET    /api/admin/v1/tenants
POST   /api/admin/v1/tenants
GET    /api/admin/v1/tenants/:id
PATCH  /api/admin/v1/tenants/:id
DELETE /api/admin/v1/tenants/:id           # → triggers Quorum
POST   /api/admin/v1/tenants/:id/{lock,unlock,freeze,clone,migrate}

GET    /api/admin/v1/domains
POST   /api/admin/v1/tenants/:id/domains
DELETE /api/admin/v1/domains/:id
POST   /api/admin/v1/domains/:id/verify

GET    /api/admin/v1/admins
POST   /api/admin/v1/admins
PATCH  /api/admin/v1/admins/:id
POST   /api/admin/v1/admins/:id/{disable,reset-password}

GET    /api/admin/v1/dashboard/overview
GET    /api/admin/v1/dashboard/realtime    # SSE
GET    /api/admin/v1/dashboard/timeseries
GET    /api/admin/v1/dashboard/top-tenants

GET    /api/admin/v1/logs                  # ClickHouse search
GET    /api/admin/v1/traces/:trace_id
GET    /api/admin/v1/metrics
GET    /api/admin/v1/alerts
POST   /api/admin/v1/alerts/:id/silence

POST   /api/admin/v1/quorum                # Create request
POST   /api/admin/v1/quorum/:id/sign       # YubiKey sign
GET    /api/admin/v1/quorum/:id

POST   /api/admin/v1/impersonate/:tenant_id
POST   /api/admin/v1/impersonate/end

GET    /api/admin/v1/audit
POST   /api/admin/v1/audit/export

GET    /api/admin/v1/billing/subscriptions
POST   /api/admin/v1/billing/refund
GET    /api/admin/v1/billing/usage

# Notifications (v1.1)
GET    /api/admin/v1/notifications/templates
POST   /api/admin/v1/notifications/templates
PATCH  /api/admin/v1/notifications/templates/:id
DELETE /api/admin/v1/notifications/templates/:id
POST   /api/admin/v1/notifications/broadcast
POST   /api/admin/v1/notifications/targeted
GET    /api/admin/v1/notifications/dispatch-log
GET    /api/admin/v1/notifications/stats
```

WebSocket / SSE endpoints: `/ws/admin/realtime`, `/ws/admin/logs`.

## Frontend Stack

- React 18 + Vite + TypeScript
- shadcn/ui + Radix UI primitives
- TanStack Query (server state), Zustand (client state)
- Recharts + Tremor (charts), Tailwind CSS v4, Lucide Icons

Component library: DataTable, ChartCard, LiveMetricCard, TenantDetailSheet, AuditLogViewer, ImpersonationBanner, QuorumDialog, FeatureFlagMatrix, LogStream (xterm-style), AlertRuleBuilder, ResourceGauges, WorldMapTraffic, NotificationComposer.

## Acceptance Criteria (Phase 1–3)

- AC-ADM-01 Create tenant < 5 s (p95)
- AC-ADM-02 Realtime dashboard update < 1 s (p95)
- AC-ADM-03 Log search < 100 ms on 1 B rows (ClickHouse)
- AC-ADM-04 Audit covers 100 % of actions
- AC-ADM-05 No DNS/IP public for admin
- AC-ADM-06 Quorum 2-of-3 works end-to-end
- AC-ADM-07 WebAuthn login < 500 ms (p95)
- AC-ADM-08 Broadcast notification < 30 s for 1000 recipients
- AC-ADM-09 Impersonation audit 100 %
- AC-ADM-10 AI RCA < 3 s

## Disaster Recovery & Cost

RPO/RTO: admin-gateway 0/30 s, postgres-admin 5 min/30 min, audit ScyllaDB 1 h/2 h, WireGuard config 1 h/15 min. Quarterly DR drills.

Estimated infra ~$1,637 / month → ≈ $0.16 per tenant at 10 k tenants.
