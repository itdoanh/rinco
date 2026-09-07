# Tenant Service

Multi-tenant CRUD: tenants, custom domains, branding, settings, usage counters, subscription state.

## Overview

The tenant-service is the **source of truth for tenants in the RINCO
platform**.  Every other service that needs to know "which tenant does
this request belong to?" reads from `tenant.tenants` (a schema exposed
to every database by the gateway / Connect-RPC interceptor).  This
service owns:

- Tenant lifecycle (`active`, `suspended`, `cancelled`).
- Per-tenant settings (`theme`, `locale`, `currency`, `timezone`, …).
- Custom domains + SSL provisioning hooks.
- Branding assets (logo, favicon, palette) → uploaded to the tiered
  S3-compatible store via the `landing-service` SDK.
- Usage counters (api_calls, storage_gb, seats) read by the billing
  service.
- Subscription / plan state (`starter`, `pro`, `enterprise`).

## Architecture

```
                    ┌──────────────────────────┐
   admin-portal ───▶│ /v1/admin/tenants        │
                    │ /v1/admin/tenants/:id    │──┐
   tenant-site  ───▶│ /v1/tenant/:slug         │  │
                    │ /v1/tenant/:slug/branding│  │
                    └──────────┬───────────────┘  │
                               │                  │
                               ▼                  ▼
                     ┌────────────────────────────┐
                     │     PostgreSQL (RLS)       │
                     │  schema `tenant`           │
                     │  tables: tenants, domains, │
                     │         settings, usage,   │
                     │         subscriptions      │
                     └────────┬───────────────────┘
                              │ publish
                              ▼
                     ┌─────────────────────┐
                     │   NATS              │
                     │ tenant.created      │
                     │ tenant.updated      │
                     │ tenant.suspended    │
                     └─────────────────────┘
```

**Tech:** Go 1.23 • Echo v4 • pgx/v5 • redis/go-redis/v9 • Prometheus • OpenTelemetry OTLP/gRPC • Connect-RPC (JSON / proto).

## Quick Start

```bash
# 1. Run Postgres + Valkey (docker compose in infra/dev/)
docker compose up -d postgres valkey

# 2. Set required env
export TENANT_DATABASE_URL=postgres://rinco:rinco_dev_password@localhost:5432/rinco?sslmode=disable
export TENANT_VALKEY_URL=redis://localhost:6379/0
export TENANT_ADMIN_API_KEY=dev_admin_key_change_me
export OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317

# 3. Build + run
go build -o bin/tenant-service ./cmd
./bin/tenant-service

# 4. Smoke
curl -fsS localhost:8082/healthz
curl -fsS localhost:8082/readyz
curl -fsS localhost:8082/metrics | head -8
```

## API Reference

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/healthz` | – | Liveness |
| GET | `/readyz` | – | Postgres + Valkey ping |
| GET | `/metrics` | – | Prometheus exposition |
| GET | `/version` | – | Service + version |
| POST | `/v1/admin/tenants` | admin | Create tenant |
| GET | `/v1/admin/tenants` | admin | List (filter by status, plan, search) |
| GET | `/v1/admin/tenants/:id` | admin | Get by id |
| PATCH | `/v1/admin/tenants/:id` | admin | Update mutable fields |
| POST | `/v1/admin/tenants/:id/suspend` | admin | Suspend |
| POST | `/v1/admin/tenants/:id/reactivate` | admin | Reactivate |
| DELETE | `/v1/admin/tenants/:id` | admin | Soft-delete |
| GET | `/v1/tenant/:slug` | – | Public profile |
| PUT | `/v1/tenant/:slug/settings` | tenant | Update settings |
| GET | `/v1/tenant/:slug/branding` | – | Branding payload |
| PUT | `/v1/tenant/:slug/branding` | tenant | Replace branding |
| POST | `/v1/tenant/:slug/domains` | tenant | Attach custom domain |
| DELETE | `/v1/tenant/:slug/domains/:domain` | tenant | Detach |
| GET | `/v1/tenant/:slug/usage` | tenant | Usage counters |
| POST | `/v1/internal/tenants/lookup` | rpc | Slug → id (Connect-RPC) |

Full payload examples live in `examples/curl.sh` and the Postman collection.

## Database Schema

```
tenants
  id           uuid PK
  slug         text UNIQUE NOT NULL
  name         text NOT NULL
  plan         text NOT NULL DEFAULT 'starter'   -- starter|pro|enterprise
  status       text NOT NULL DEFAULT 'active'    -- active|suspended|cancelled
  trial_ends_at timestamptz
  created_at   timestamptz NOT NULL DEFAULT now()
  updated_at   timestamptz NOT NULL DEFAULT now()
  deleted_at   timestamptz

tenant_settings
  tenant_id    uuid PK REFERENCES tenants(id)
  theme        jsonb NOT NULL DEFAULT '{}'::jsonb
  locale       text NOT NULL DEFAULT 'en'
  currency     text NOT NULL DEFAULT 'USD'
  timezone     text NOT NULL DEFAULT 'UTC'

tenant_domains
  tenant_id    uuid REFERENCES tenants(id)
  domain       text PRIMARY KEY
  verified_at  timestamptz
  ssl_status   text DEFAULT 'pending'

tenant_usage
  tenant_id    uuid REFERENCES tenants(id)
  period       text  -- YYYY-MM
  api_calls    bigint NOT NULL DEFAULT 0
  storage_gb   numeric(12,3) NOT NULL DEFAULT 0
  seats        int  NOT NULL DEFAULT 0
  PRIMARY KEY (tenant_id, period)
```

Row-level security is enabled on every table with a policy of
`USING (id::text = current_setting('app.current_tenant_id', true))`.

## Configuration

See `config.example.yaml` for the full list.  Required:

| Var | Default | Description |
|---|---|---|
| `TENANT_DATABASE_URL` | – | PostgreSQL DSN |
| `TENANT_VALKEY_URL` | `redis://localhost:6379/0` | Redis-compatible URL |
| `TENANT_HTTP_ADDR` | `:8082` | Listen address |
| `TENANT_ADMIN_API_KEY` | – | Bearer for `/v1/admin/*` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | – | OTLP gRPC collector |

Optional:

| Var | Default | Description |
|---|---|---|
| `TENANT_PUBLISH_NATS` | `true` | Publish lifecycle events |
| `TENANT_NATS_URL` | `nats://nats:4222` | NATS URL |
| `TENANT_ENABLE_RLS_GUC` | `true` | Set `app.current_tenant_id` per request |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |

## Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata: { name: tenant-service }
spec:
  replicas: 2
  selector: { matchLabels: { app: tenant-service } }
  template:
    metadata: { labels: { app: tenant-service } }
    spec:
      containers:
        - name: tenant
          image: rinco/tenant-service:1.0.0
          ports: [{ containerPort: 8082 }]
          envFrom:
            - configMapRef: { name: tenant-service }
            - secretRef:   { name: tenant-service }
          readinessProbe:
            httpGet: { path: /readyz, port: 8082 }
            periodSeconds: 5
          livenessProbe:
            httpGet: { path: /healthz, port: 8082 }
            periodSeconds: 10
          resources:
            requests: { cpu: 100m, memory: 128Mi }
            limits:   { cpu: 500m, memory: 512Mi }
```

## Observability

- **Metrics**: `tenant_http_requests_total{route,status}`,
  `tenant_http_request_duration_seconds_bucket{route}`, Go runtime.
- **Logs**: `slog` JSON with `service`, `trace_id`, `tenant_id`, `request_id`.
- **Traces**: OTLP/gRPC, sampler = parent-based(AlwaysOn).  Every request
  emits a server span; downstream DB calls share the trace context.

## Development

```bash
go mod tidy
go vet ./...
go test ./...
docker build -t rinco/tenant-service:dev .
```
