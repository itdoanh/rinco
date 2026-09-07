# Tenant Service

Tenant, domain, branding, settings, and usage management for the RINCO platform.

## Highlights

- **CRUD tenants** with soft delete, suspend, activate, plan upgrades
- **Custom domains** with verification + SSL status tracking; cached in Valkey
- **Branding** (logo, colors, font, custom CSS) — both relational and JSONB snapshot
- **Per-tenant settings** (arbitrary key/value JSON)
- **Usage tracking** per period (users, leads, deals, storage, API calls, CCU)
- **Plan quotas** (starter/pro/business/enterprise) with feature flags
- **Audit log** of every lifecycle change in `tenant.tenant_audit`
- **Multi-tenant RLS** via `app.current_tenant_id` session variable on every read
- **Connect-RPC** for internal lookups (`GetTenantBySlug`, `ResolveDomain`, `GetQuota`)
- **Prometheus** metrics + **OpenTelemetry** tracing + **structured slog**
- **Graceful shutdown** (SIGINT/SIGTERM, 30 s drain)

## Tech

- Go 1.23+ • Echo v4 • pgx/v5 (pgxpool) • redis/go-redis/v9
- OpenTelemetry SDK + otelecho
- prometheus/client_golang

## Environment

| Variable | Default | Description |
|---|---|---|
| `TENANT_HTTP_ADDR` | `:8082` | HTTP listen address |
| `TENANT_DATABASE_URL` | _required_ | PostgreSQL DSN |
| `TENANT_VALKEY_URL` | `redis://localhost:6379/0` | Redis/Valkey for domain cache |
| `TENANT_ADMIN_API_KEY` | `dev_admin_key_change_me` | Required header `X-Admin-Key` for write endpoints |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | _empty_ | OTLP gRPC endpoint for traces |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `ENV` | `development` | Service environment label |

## Database

Migrations are embedded in `cmd/migrations/sql.go` and applied in order on startup. Tables (schema `tenant`):

- `tenants` (id, slug UNIQUE, name, status, plan, settings_json, branding_json, suspended_at, ...)
- `tenant_audit` (id, tenant_id, actor_id, actor_email, action, payload)
- `tenant_domains` (id, tenant_id, domain UNIQUE, type, verified_at, ssl_status, ssl_issuer, ssl_expires_at, is_primary)
- `tenant_settings` (id, tenant_id, key, value JSONB) with UNIQUE(tenant_id, key)
- `tenant_branding` (tenant_id PK, logo_url, favicon_url, primary_color, secondary_color, accent_color, font_family, custom_css, email_logo_url)
- `tenant_usage` (tenant_id, period, metric, value) with composite PK
- `plan_quotas` (plan PK, max_users, max_storage_mb, max_api_calls, max_ccu, features)

## HTTP API

Base URL: `http://localhost:8082`. All requests/responses are JSON.

### Public

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | Liveness |
| GET | `/readyz` | DB + Valkey ping |
| GET | `/metrics` | Prometheus |
| GET | `/v1/tenants/by-domain/:domain` | Resolve domain → tenant (cached 5 min) |

### Admin (header `X-Admin-Key: <TENANT_ADMIN_API_KEY>`)

| Method | Path | Description |
|---|---|---|
| POST | `/v1/tenants` | Create tenant `{slug, name, plan?, email?}` |
| GET | `/v1/tenants?status=&plan=&limit=&offset=` | List tenants |
| GET | `/v1/tenants/:id` | Get tenant |
| PUT | `/v1/tenants/:id` | Update `{name, plan, status, settings, branding}` |
| DELETE | `/v1/tenants/:id` | Soft delete (`status=deleted`, `deleted_at=now()`) |
| POST | `/v1/tenants/:id/suspend?reason=` | Suspend |
| POST | `/v1/tenants/:id/activate` | Activate |
| GET | `/v1/tenants/:id/stats` | Usage rollup for current period |
| GET | `/v1/tenants/:id/domains` | List domains (requires `X-Tenant-ID`) |
| POST | `/v1/tenants/:id/domains` | Add `{domain, type?, is_primary?}` |
| DELETE | `/v1/tenants/:id/domains/:domain_id` | Remove domain |
| GET | `/v1/tenants/:id/branding` | Get branding |
| POST | `/v1/tenants/:id/branding` | Upsert branding |
| DELETE | `/v1/tenants/:id/branding` | Reset branding |
| GET | `/v1/tenants/:id/usage?period=YYYY-MM` | Per-metric usage |
| POST | `/v1/tenants/:id/usage` | Record `{metric, value, period?}` (requires `X-Tenant-ID`) |

## Connect-RPC (internal)

`POST /rpc/tenant.v1.TenantService/<Method>` (JSON).

| Method | Body | Response |
|---|---|---|
| `GetTenantBySlug` | `{slug}` | tenant object |
| `ResolveDomain` | `{domain}` | `{tenant_id}` |
| `GetQuota` | `{tenant_id, metric}` | `{tenant_id, metric, limit, used, features}` |

## Curl examples

```bash
# Health
curl -s localhost:8082/healthz

# Create
curl -s -X POST localhost:8082/v1/tenants \
  -H 'Content-Type: application/json' \
  -H 'X-Admin-Key: dev_admin_key_change_me' \
  -d '{"slug":"acme","name":"Acme Inc","plan":"pro","email":"admin@acme.com"}'

# List
curl -s "localhost:8082/v1/tenants?status=active&limit=20" \
  -H 'X-Admin-Key: dev_admin_key_change_me'

# Get
curl -s localhost:8082/v1/tenants/<id> -H 'X-Admin-Key: dev_admin_key_change_me'

# Update
curl -s -X PUT localhost:8082/v1/tenants/<id> \
  -H 'Content-Type: application/json' \
  -H 'X-Admin-Key: dev_admin_key_change_me' \
  -d '{"plan":"business","settings":{"locale":"vi-VN"}}'

# Suspend / activate
curl -s -X POST "localhost:8082/v1/tenants/<id>/suspend?reason=non_payment" \
  -H 'X-Admin-Key: dev_admin_key_change_me'
curl -s -X POST localhost:8082/v1/tenants/<id>/activate \
  -H 'X-Admin-Key: dev_admin_key_change_me'

# Delete (soft)
curl -s -X DELETE localhost:8082/v1/tenants/<id> \
  -H 'X-Admin-Key: dev_admin_key_change_me'

# Add domain
curl -s -X POST localhost:8082/v1/tenants/<id>/domains \
  -H 'Content-Type: application/json' \
  -H 'X-Admin-Key: dev_admin_key_change_me' \
  -d '{"domain":"shop.acme.com","type":"subdomain","is_primary":true}'

# Resolve domain
curl -s localhost:8082/v1/tenants/by-domain/shop.acme.com

# Stats
curl -s localhost:8082/v1/tenants/<id>/stats -H 'X-Admin-Key: dev_admin_key_change_me'

# Branding
curl -s localhost:8082/v1/tenants/<id>/branding
curl -s -X POST localhost:8082/v1/tenants/<id>/branding \
  -H 'Content-Type: application/json' \
  -H 'X-Admin-Key: dev_admin_key_change_me' \
  -d '{"logo_url":"https://cdn.acme.com/logo.svg","primary_color":"#0ea5e9","font_family":"Inter"}'

# Record / get usage
curl -s -X POST localhost:8082/v1/tenants/<id>/usage \
  -H 'Content-Type: application/json' -H 'X-Tenant-ID: <id>' \
  -d '{"metric":"api_calls","value":42}'
curl -s localhost:8082/v1/tenants/<id>/usage -H 'X-Admin-Key: dev_admin_key_change_me'

# Connect-RPC (called by other services)
curl -s -X POST localhost:8082/rpc/tenant.v1.TenantService/GetTenantBySlug \
  -H 'Content-Type: application/json' -d '{"slug":"acme"}'
curl -s -X POST localhost:8082/rpc/tenant.v1.TenantService/ResolveDomain \
  -H 'Content-Type: application/json' -d '{"domain":"shop.acme.com"}'
curl -s -X POST localhost:8082/rpc/tenant.v1.TenantService/GetQuota \
  -H 'Content-Type: application/json' -d '{"tenant_id":"<id>","metric":"users"}'
```

## Observability

- All logs JSON via `slog` with `service=tenant-service`, `trace_id`, `span_id`.
- Every request gets a `X-Trace-ID` header (generated if absent).
- `/metrics` exposes `http_requests_total`, `http_request_duration_seconds`, plus Go process metrics.
- Tracing is OTLP gRPC; if `OTEL_EXPORTER_OTLP_ENDPOINT` is unset, tracing is a no-op (cheap spans still attach).

## Development

```bash
# Tidy
go mod tidy

# Build
go build -o bin/tenant-service ./cmd/main.go

# Run locally
TENANT_DATABASE_URL=postgres://postgres:postgres@localhost:5432/rinco?sslmode=disable \
TENANT_ADMIN_API_KEY=dev_admin_key_change_me \
./bin/tenant-service

# Vet / test
go vet ./...
go test ./...
```

## Docker

```bash
docker build -t rinco/tenant-service:latest .
docker run --rm -p 8082:8082 \
  -e TENANT_DATABASE_URL=postgres://... \
  -e TENANT_ADMIN_API_KEY=... \
  rinco/tenant-service:latest
```
