# Landing Service

> **Subsystem #2/#5/#02 — Landing pages + form ingestion + Facebook CAPI + tiered S3**
>
> Renders dynamic landing pages from PostgreSQL `landing.landing_pages`
> (with optional MongoDB override), accepts lead-form submissions,
> publishes server-side tracking events, delivers Conversions API events
> to Meta, and stores media in a hot-SSD / cold-HDD tiered bucket layout.

## 1. Stack

- Go 1.23 + Echo v4 + Connect-RPC-style `/internal/landing.v1.LandingService/*` paths
- PostgreSQL 17 (JSONB design schema, dedup via unique indexes)
- MongoDB (optional) for `dynamic_pages` and `form_payloads` collections
- Valkey (Redis-compatible) for dedup cache
- MinIO / S3 (`rinco-hot-ssd`, `rinco-cold-hdd`) with age-based lifecycle
- slog JSON + OTLP tracing + Prometheus client

## 2. Endpoints

### Page rendering
- `GET /v1/pages/:tenant_slug/:page_slug` — render the active page
- `GET /v1/pages/:tenant_slug/index` — alias for `index` slug
- `GET /v1/preview/:page_id` — preview draft (requires `X-User-ID` or `Authorization`)
- `POST /v1/pages` — create or update page (`X-User-ID` required)
- `GET /v1/pages/:id` — fetch page by ID

### Forms
- `POST /v1/forms/:form_slug/submit` — public submission, no auth
- `POST /v1/forms/:form_slug/submit/batch` — bulk submit (max 100 items)
- `GET /v1/forms/:form_slug/schema` — public form schema

### Tracking
- `POST /v1/track/pageview`
- `POST /v1/track/event`
- `POST /v1/track/conversion`
- `GET /v1/track/pixel/:tenant_slug/p.gif` — 1×1 GIF pixel + auto event
- `GET /v1/track/redirect/:tenant_slug/:click_id?url=...` — click redirector

### CAPI
- `POST /v1/capi/send` — manual CAPI event (signed dedup_id required upstream)
- `GET /v1/capi/status` — configuration + queue size
- `POST /v1/capi/test` — test event (requires editor auth)

### Health / runtime
- `GET /healthz`, `GET /readyz`, `GET /metrics`, `GET /version`

### Connect-RPC
- `POST /internal/landing.v1.LandingService/GetPageConfig`
- `POST /internal/landing.v1.LandingService/SubmitLead`
- `POST /internal/landing.v1.LandingService/TrackEvent`
- `POST /internal/landing.v1.LandingService/SendCAPIEvent`
- `POST /internal/landing.v1.LandingService/GetFBPixelConfig`

## 3. Facebook CAPI flow

```
Public form ─► POST /v1/forms/:slug/submit
              │
              ├─ SHA-256(idempotency_key + LANDING_TRACKING_SALT) → event_id
              ├─ Insert landing.form_submissions (UNIQUE tenant+idempotency)
              └─ enqueue capiEvent ─► background goroutine ─► POST graph.facebook.com/v18.0/<pixel>/events
                                                                                  ├─ HMAC SHA-256 (in production)
                                                                                  └─ dedup_event_id for Meta dedup
```

## 4. Tiered S3 storage

`internal/storage/tiered.go` exposes `TieredStore` with two buckets:

| Bucket                | Use case                                | Default policy                       |
|----------------------|------------------------------------------|--------------------------------------|
| `rinco-hot-ssd`      | Newly uploaded assets (< 30 days)        | `MaxHotAge = 30 * 24h`               |
| `rinco-cold-hdd`     | Long-term archive                        | `cold` flag / auto migration worker  |

Lifecycle worker runs every 6 hours, scans the hot bucket, and moves any
object older than `MaxHotAge` to the cold bucket, deleting the original.
Both buckets must exist on startup (`EnsureBuckets`) when credentials are
provided. If `LANDING_S3_ENDPOINT` is empty the store degrades gracefully
(no-op uploader) so the service still runs in dev clusters without MinIO.

## 5. Migrations

| File                                | Purpose                                            |
|-------------------------------------|----------------------------------------------------|
| `migrations/0001_pages.sql`         | `landing.landing_pages`                            |
| `migrations/0002_forms.sql`         | `landing.form_definitions` + `form_submissions`    |
| `migrations/0003_tracking.sql`      | `landing.tracking_events` + `conversion_goals`     |
| `migrations/0004_pixel.sql`         | `landing.fb_pixel_configs` + `tracking_clicks`    |
| `migrations/0005_rls.sql`           | Row Level Security policies                        |

Set `LANDING_AUTOMIGRATE=true` to have the embedded bootstrap SQL run on
start-up; otherwise run `psql -f migrations/000*.sql` from your CD
pipeline.

## 6. Env vars

| Variable                          | Default                | Purpose                                |
|----------------------------------|------------------------|----------------------------------------|
| `LANDING_HTTP_ADDR`              | `:8086`                | HTTP listen port                       |
| `LANDING_DATABASE_URL`           | local dev DSN          | PostgreSQL                             |
| `LANDING_MONGO_URL`              | *(unset)*              | MongoDB                                |
| `LANDING_VALKEY_URL`             | `localhost:6379`       | Cache                                  |
| `LANDING_NATS_URL`               | *(unset)*              | Optional NATS subscribe (`lead.created`) |
| `LANDING_S3_ENDPOINT`            | *(unset)*              | MinIO/S3 endpoint                      |
| `LANDING_S3_ACCESS_KEY`          | *(unset)*              | S3 access key                          |
| `LANDING_S3_SECRET_KEY`          | *(unset)*              | S3 secret key                          |
| `LANDING_S3_BUCKET_HOT`          | `rinco-hot-ssd`        | Hot bucket                             |
| `LANDING_S3_BUCKET_COLD`         | `rinco-cold-hdd`       | Cold bucket                            |
| `LANDING_S3_SSL`                 | `false`                | `https://` for endpoint                |
| `LANDING_FB_PIXEL_ID`            | *(unset)*              | Default pixel id                       |
| `LANDING_FB_APP_SECRET`          | *(unset)*              | App secret (used as access token)      |
| `LANDING_TRACKING_SALT`          | `rinco-tracking-salt`  | Deterministic event_id salt            |
| `OTEL_EXPORTER_OTLP_ENDPOINT`    | *(unset)*              | OTLP gRPC collector                    |

## 7. Run

```bash
cd services/landing-service
go build ./cmd
LANDING_AUTOMIGRATE=true ./landing-service
```

Docker:

```bash
docker build -t rinco/landing-service -f Dockerfile .
```

## 8. Observability

- Prometheus: `GET /metrics`
  - `landing_service_http_requests_total`
  - `landing_service_http_request_duration_seconds`
- Structured logs (slog JSON) include `service`, `env`, `version`, `trace_id`
- OTLP gRPC exporter when `OTEL_EXPORTER_OTLP_ENDPOINT` is set
