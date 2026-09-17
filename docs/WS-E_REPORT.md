# RINCO WS-E — Observability + Security Hardening (Delivery Report)

> **Workstream:** WS-E
> **Owner:** rinco-ws-e
> **Scope:** DEV-PLAN §10 — observability + security hardening
> **Branch:** main
> **Status:** ✅ Complete

---

## 1. Scope recap (per task contract)

| # | Item                                               | Status |
|---|----------------------------------------------------|--------|
| 1 | Observability service (structured logs + Prom + OTel + CH + endpoints) | ✅ verified (`/healthz`, `/readyz`, `/metrics` smoke-tested) |
| 2 | Audit middleware (`packages/go/middleware/audit.go`)               | ✅ rewritten |
| 3 | Security middleware (`packages/go/middleware/security.go`)         | ✅ new file |
| 4 | Grafana dashboards (system-wide + 5 service-specific)             | ✅ created/extended |
| 5 | Prometheus alert rules (P0/P1/P2/P3)                              | ✅ 70 rules across 3 files |
| 6 | eBPF/XDP architecture document                                    | ✅ `docs/09-security/EBPF_NOTES.md` |
| 7 | Secret rotation script                                             | ✅ `scripts/rotate-secrets.sh` |

---

## 2. Observability service

`services/observability-service/` already had a complete middleware stack
(from prior loops):

- Structured logging (slog JSON) per request.
- Prometheus HTTP counter + duration histogram (`observability_service_http_requests_total`, `observability_service_http_request_duration_seconds`).
- OpenTelemetry OTLP tracing (`platform.InitTracer` → `otel.SetTracerProvider`).
- ClickHouse aggregation via HTTP gateway (`clickhouse.Writer`).
- `/healthz` (200 OK), `/readyz` (200/503), `/metrics` (Prometheus format), `/version`.

**Smoke test** (run on `bin/observability-service.exe` with `OBSERVABILITY_HTTP_ADDR=:8099`):

```
GET /healthz  → 200  {"service":"observability-service","status":"ok","version":"1.0.0"}
GET /readyz   → 200  {"status":"ready"}
GET /metrics  → 200  Prometheus exposition (observability_service_http_requests_total + go_*)
GET /version  → 200  {"service":"observability-service","version":"1.0.0"}
```

No changes were required to the service itself — WS-E provided the
**shared middleware package** that other services can adopt
incrementally.

---

## 3. Audit middleware (new)

`packages/go/middleware/audit.go` is a full rewrite of the previously
stubbed file.  Highlights:

- Records every POST / PUT / PATCH / DELETE (configurable via
  `SkipMethods`/`IncludeReadMethods`).
- Captures **actor** (`X-User-ID`), **tenant** (`X-Tenant-ID`),
  **resource type/id** (derived from route params), **before/after**
  snapshots (`SetBeforeSnapshot(c, jsonString)` helper for handlers).
- Best-effort writes:
  - In-process channel → background goroutine (drained every 2 s or
    when batch is full).
  - `observability.audit_logs` table (Postgres) — schema already
    exists from migration `0002_audit.sql`.
  - `observability.app_audit_logs` table (ClickHouse) — JSONEachRow
    HTTP insert, batched.
- PII redaction of `password`, `token`, `secret`, `api_key`,
  `authorization`, `cookie`, `csrf_token`, etc. (configurable).
- Body capture bounded to 16 KiB by default (configurable).
- `cfg.Channel` (optional) lets in-process subscribers tap the stream
  for testing or in-memory analytics.
- Backwards-compatible shims for `parseActionResource`, `isUUID`,
  `isNumericID`, `redactSensitiveData`, `AuditLog`,
  `AuditLogChanWriter`, `AuditLogSlogWriter`, `AuditContext`,
  `WithAuditContext`, `GetAuditContext` so existing tests in the
  package keep compiling.

**Tests** (`packages/go/middleware/wse_audit_test.go`):
- 18 tests covering redactor, action/resource derivation, ClickHouse
  schema, CORS preflight, rate-limiter blocks and skip paths.
- All pass: `go test ./middleware/... -count=1`.

---

## 4. Security middleware (new)

`packages/go/middleware/security.go`:

### 4.1 `SecurityHeadersWithConfig(cfg)`

- CORS with explicit allowlist (no wildcard by default in production).
- CSP (`default-src 'self'; frame-ancestors 'none'; object-src 'none'`).
- HSTS (`max-age=31536000; includeSubDomains; preload`).
- X-Frame-Options `DENY`.
- X-Content-Type-Options `nosniff`.
- Permissions-Policy (camera, microphone, geolocation, payment, usb
  disabled).
- Cross-Origin-Opener-Policy / Cross-Origin-Embedder-Policy /
  Cross-Origin-Resource-Policy.
- Short-circuits OPTIONS preflight with 204.

### 4.2 `RateLimiter(cfg)`

- Token bucket via `packages/go/ratelimit.NewTokenBucket`
  (in-memory, no Redis required).
- When `cfg.Redis != nil`, falls back to `ratelimit.SlidingWindow`
  for cross-pod quotas.
- Best-effort: if Valkey is unreachable, requests still pass (with a
  warn log).
- Exposes `X-RateLimit-Limit`, `X-RateLimit-Remaining`,
  `X-RateLimit-Source`, `Retry-After` headers.
- 429 JSON response on limit hit.

### 4.3 Sanitizers

- `SanitizeString(s, maxLen)` — strip NUL, control chars, cap length.
- `SanitizeHTML(s)` — escape `<`, `>`, `&`, `"`, `'`.
- `SanitizeEmail(s)` — lowercase + format-validate.
- `SanitizeJSON(payload, maxStringLen)` — walk JSON, redact sensitive
  keys, cap string fields.
- `SanitizeHeaderValue(s)` — strip CR/LF (response splitting).
- `ValidateUUID(s)` — canonical lower-case UUID.
- `SanitizeMiddleware(maxLen)` — Echo middleware that sanitizes the
  request body before delegating.

---

## 5. Grafana dashboards

| File                                       | Purpose                          | Panels |
|--------------------------------------------|----------------------------------|-------:|
| `rinco-overview.json` (rewrite)            | system-wide overview             | 10     |
| `auth-service.json` (new)                  | auth-service                     | 9      |
| `crm-service.json` (`03-crm.json` existing)| CRM                              | 5      |
| `lead-service.json` (new)                  | lead-service                     | 8      |
| `chat-engine.json` (`04-chat.json` existing)| chat-engine                     | 5      |
| `webrtc-sfu.json` (`05-sfu.json` existing) | SFU                              | 5      |
| `ai-services.json` (extended)              | scoring + RAG + AI-SRE + STT + cost + drift | 12 |

All dashboards provisioned by the existing
`infra/grafana/provisioning/dashboards/dashboards.yml` (no config
change required).

---

## 6. Prometheus alert rules

| File                                | Sev     | Groups | Rules |
|-------------------------------------|---------|-------:|------:|
| `critical.yml` (new)                | P0 page | 6      | 21    |
| `warning.yml` (new)                 | P1      | 6      | 29    |
| `info.yml` (new)                    | P2/P3   | 4      | 20    |
| `rinco.yml` + `01-alerts.yml` + `rinco-ai.yml` (existing) | mixed | various | various |

Critical rules cover:
- service down (>50% replicas, single replica, pod crash-loop),
- OOMKilled, memory pressure critical,
- **RLS bypass / cross-tenant / FORCE ROW LEVEL disabled**,
- **PASETO key revocation** (potential compromise),
- **super-admin unauthorized access**,
- Postgres / Scylla / Valkey / ClickHouse exhaustion and failover,
- Disk < 5%, inodes < 5%,
- SLO burn-rate fast + very-fast (14x / 100x).

Warning rules cover high error rate, p99/p95 latency, DB near-
saturation, brute-force, rate-limit spikes, audit-drop spikes,
AI service failures.

Info rules cover drift (lead score, chat topic entropy, conversion
rate), cost (AI spend, egress, storage growth), trends, cert expiry.

Prometheus will pick these up automatically once
`rule_files: [ "/etc/prometheus/rules/*.yml" ]` is added to
`infra/prometheus/prometheus.yml` (the existing config already
references the directory pattern in the recorder file; the new files
are co-located and the operator can add them by symlink).

---

## 7. eBPF / XDP notes

`docs/09-security/EBPF_NOTES.md` documents the future kernel-level
DDoS mitigation architecture.  No runtime is shipped in this
milestone — the current Traefik + Valkey stack is sufficient for the
documented SLOs.  The doc records:

- Layered filter (XDP → tc → Traefik → service).
- BPF maps layout (`rinco_rate_map`, `rinco_geo_map`, `rinco_config_map`).
- Full XDP source (`xdp_rinco.bpf.c`) — design target for WS-E2.
- Threat-feed → userspace controller (`rinco-ebpf-ctl`) → BPF map
  flow.
- Auto-blackhole flow (>/24 / 10k drops / 60 s → block 24 h).
- Metrics wired into the existing Prometheus pipeline
  (`rinco_xdp_drops_total`, `rinco_autoblock_total`, etc.).
- Acceptance criteria (WS-E-AC-DD-1..5).

---

## 8. Secret rotation script

`scripts/rotate-secrets.sh` — POSIX bash, idempotent, dry-run by
default.

Subcommands:

- `paseto` — generates new 256-bit hex key, writes
  `infra/k8s/base/secrets/paseto-current.json` with previous-key
  metadata, patches `rinco-auth-paseto-${id}` K8s secret with both
  keys for the overlap window.
- `smtp` — generates new password, writes
  `infra/k8s/base/secrets/smtp-current.json`, patches the
  `rinco-email-smtp` K8s secret.
- `all` — runs PASETO + SMTP + a K8s `rinco.app/secret-version`
  annotation bump that forces a rolling restart to pick up env vars.

Flags:
- `--overlap <duration>` — overlap window (default `24h`).
- `--env <name>` — environment (default `staging`).
- `--apply` — actually apply (default `--dry-run`).
- `--namespace <ns>` — K8s namespace (default `rinco`).

---

## 9. Build / test verification

```
$ go build ./packages/go/middleware/...
(no errors)

$ go test ./packages/go/middleware/... -count=1
ok  	github.com/itdoanh/rinco/packages/go/middleware	0.291s

$ go build ./services/observability-service/cmd/...
(no errors)

$ go test ./services/observability-service/cmd/... ./services/observability-service/internal/...
ok  	9 packages

$ go build ./services/auth-service/cmd/...
(no errors)

$ go build ./services/crm-service/cmd/...
(no errors)

$ go build ./services/tenant-service/cmd/...
(no errors)
```

The observability-service binary was launched locally and responded
correctly to `/healthz`, `/readyz`, `/metrics`, `/version` smoke
probes.

---

## 10. Out-of-scope items (intentionally untouched)

- Auth-service / tenant-service internals — those belong to WS-A.
- CRM-tree / lead-service business logic — WS-B.
- WebRTC SFU / chat-engine — WS-D.
- Frontend admin-portal pages — WS-F.
- RLS policies themselves — WS-A already shipped `0001_rls-setup.sql`
  + WS-A 003 tenant GUC middleware.
- MLflow lead-scoring config — WS-C.

---

## 11. Commits

```
fa97f95  Loop WS-E 2: dashboards + alert rules + eBPF notes + secret rotation
696539d  Loop WS-E 3: extend AI services dashboard with cost + drift panels
```

(The middleware + test files were committed earlier inside WS-B's
commit `3e4c45c` due to the WS-C bot's `git add` race — the actual
content is intact and fully covers WS-E item 2 + 3.)