# Email Service

> **Phân hệ #6 — Email** · Multi-driver transactional email (SMTP / Resend / SendGrid / AWS SES / Console).
> Built-in template engine (Go `text/template`) with date format, i18n (vi/en), markdown → HTML,
> URL-safe encoding, click tracking, 1×1 tracking pixel, tiered S3 attachments, webhook ingest.

## 1. Endpoints (12+)

| Method | Path | Mô tả |
|--------|------|-------|
| GET    | `/healthz` / `/readyz` / `/metrics` / `/version` | Liveness / readiness / Prometheus / version |
| POST   | `/v1/email/send` | Send a single email (text / html / markdown) |
| POST   | `/v1/email/batch` | Bulk send, fans out per recipient |
| POST   | `/v1/email/templates` | Create template |
| GET    | `/v1/email/templates` | List templates (paginated) |
| GET    | `/v1/email/templates/:id` | Retrieve a template |
| PUT    | `/v1/email/templates/:id` | Update template (bumps version) |
| DELETE | `/v1/email/templates/:id` | Soft delete template |
| POST   | `/v1/email/templates/:id/render` | Render with variables (no send) |
| GET    | `/v1/email/templates/:id/preview` | Render with sample data |
| GET    | `/v1/email/logs` | Filter by `status`, `from`, `to` |
| GET    | `/v1/email/logs/:id` | Retrieve one delivery log |
| GET    | `/v1/email/stats` | Counts by status / driver / day |
| POST   | `/v1/email/webhooks/:provider` | Receive bounce / complaint / delivery |
| GET    | `/e/:msg_id.gif` | 1×1 transparent GIF tracking pixel |
| GET    | `/c/:msg_id?url=...` | Click → redirect (with tracking) |
| GET    | `/u/:msg_id` | One-click unsubscribe |

Connect-RPC (Buf/Connect over HTTP/JSON):

```
POST /internal/email.v1.EmailService/SendEmail
POST /internal/email.v1.EmailService/BatchSend
POST /internal/email.v1.EmailService/RenderTemplate
POST /internal/email.v1.EmailService/GetDeliveryLog
POST /internal/email.v1.EmailService/ListTemplates
POST /internal/email.v1.EmailService/GetTemplateStats
```

## 2. Drivers

| Driver    | Env | Notes |
|-----------|-----|-------|
| `console` | `EMAIL_DRIVER=console` | Logs to stdout — default for dev |
| `smtp`    | `EMAIL_SMTP_HOST/PORT/USER/PASS` | `net/smtp` built-in |
| `resend`  | `EMAIL_RESEND_API_KEY` | `POST https://api.resend.com/emails` |
| `sendgrid`| `EMAIL_SENDGRID_API_KEY` | `POST https://api.sendgrid.com/v3/mail/send` |
| `ses`     | `EMAIL_AWS_REGION/ACCESS_KEY/SECRET` | SigV4 signed POST to `email.<region>.amazonaws.com` |

Each backend implements:

```go
type Driver interface {
    Name() string
    Send(ctx context.Context, msg Message) (Result, error)
    Close() error
}
```

## 3. Template Engine

`internal/render/template.go` exposes a Go `text/template` engine with custom funcs:

| Func | Example | Description |
|------|---------|-------------|
| `upper` `lower` `title` `trim` | `{{ upper .Name }}` | Standard string helpers |
| `default` | `{{ default "—" .Nickname }}` | Default value for missing keys |
| `dateFormat` | `{{ dateFormat "02 Jan 2006" .CreatedAt }}` | Format RFC3339 / `time.Time` |
| `i18n` | `{{ i18n "hello" "vi" }}` | Vietnamese / English dictionary |
| `markdown` | `{{ markdown .Body }}` | Built-in Markdown → HTML |
| `urlSafe` | `{{ urlSafe .Url }}` | `url.QueryEscape` wrapper |

Built-in Markdown supports headings, bold, italic, links, inline code.

Tracking helpers (also in `internal/render`):

* `InjectTrackingPixel(html, pixelURL)` — inserts `<img>` before `</body>`.
* `RewriteClickLinks(html, redirectBase)` — rewrites every `<a href>` through the redirector.
* `BuildUnsubscribeHeader(oneClickURL, mailto)` — `List-Unsubscribe` + `List-Unsubscribe-Post`.

## 4. Tiered S3 Storage

Attachments live in MinIO/S3 with a daily background goroutine that promotes
old objects from the hot SSD bucket to the cold HDD bucket.  Per-tenant policy
overrides are stored in `email.tenant_settings.tier_policy`.

* Hot → Cold threshold: `EMAIL_S3_HOT_DAYS` (default 30).
* Worker runs every 6 hours, plus an immediate `UPDATE` against `email.email_attachments`.
* Buckets: `EMAIL_S3_BUCKET_HOT` / `EMAIL_S3_BUCKET_COLD`.

## 5. Tracking

* Open tracking: every HTML body gets `<img src="https://track.rinco.app/e/{msg_id}.gif">`.
* Click tracking: every `<a href>` rewritten to `/c/{msg_id}?url=<original>`.
* Unsubscribe: `List-Unsubscribe` and `List-Unsubscribe-Post: List-Unsubscribe=One-Click` headers.
* Pixel endpoint: `GET /e/:msg_id.gif` returns a 1×1 GIF, logs the open.

## 6. NATS Queue

```
Subject: email.send   (queue group: email-workers)
Subject: email.sent
Subject: email.failed
Subject: email.dlq
```

Retry schedule (exponential): `1m → 5m → 30m → 2h → 12h`. After 5 attempts
the message is published to `email.dlq`.

## 7. Webhooks (Provider-Agnostic)

`POST /v1/email/webhooks/:provider` accepts any payload. We extract:

* `msg_id` (or `MessageID`, `message_id`, `id`)
* `event_type` (or `event`, `type`, `notificationType`)

And update the corresponding log row:

| Event | DB column updated |
|-------|-------------------|
| `delivered` | `delivered_at` |
| `opened` | `opened_at` |
| `clicked` | `clicked_at` |
| `bounced` | `bounced_at` |
| `complained` | `complained_at` |
| `failed` | `failed_at` |

The raw payload is also stored in `email.email_webhooks` for audit.

## 8. ENV

| Var | Default | Purpose |
|-----|---------|---------|
| `EMAIL_HTTP_ADDR` | `:8087` | HTTP listen address |
| `EMAIL_DATABASE_URL` | - | Postgres DSN |
| `EMAIL_VALKEY_URL` | `localhost:6379` | Redis/Valkey for rate limit + cache |
| `EMAIL_NATS_URL` | - | NATS server (optional) |
| `EMAIL_DRIVER` | `console` | Default driver |
| `EMAIL_SMTP_HOST/PORT/USER/PASS` | - | SMTP config |
| `EMAIL_RESEND_API_KEY` | - | Resend API key |
| `EMAIL_SENDGRID_API_KEY` | - | SendGrid API key |
| `EMAIL_AWS_REGION/ACCESS_KEY/SECRET` | - | SES config |
| `EMAIL_FROM_DEFAULT` | `no-reply@rinco.app` | Default from address |
| `EMAIL_WEB_BASE_URL` | `https://track.rinco.app` | Tracking pixel / click base URL |
| `EMAIL_S3_ENDPOINT` / `EMAIL_S3_ACCESS_KEY` / `EMAIL_S3_SECRET_KEY` | - | S3 / MinIO |
| `EMAIL_S3_BUCKET_HOT` / `EMAIL_S3_BUCKET_COLD` | `rinco-email-hot` / `rinco-email-cold` | Tier buckets |
| `EMAIL_S3_HOT_DAYS` | `30` | Hot → cold threshold |

## 9. Run

```bash
cd services/email-service
go build ./cmd && ./email-service
```

Docker:

```bash
docker build -t rinco/email-service -f services/email-service/Dockerfile .
```

## 10. Curl examples

```bash
# Send a single HTML email
curl -X POST http://localhost:8087/v1/email/send \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: t-001' \
  -d '{
    "to": ["alice@example.com"],
    "subject": "Welcome",
    "body": "# Hello **Alice**\nWelcome to RINCO!",
    "body_type": "markdown",
    "priority": "normal"
  }'

# Batch send
curl -X POST http://localhost:8087/v1/email/batch \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: t-001' \
  -d '{
    "default": { "subject": "Newsletter", "body": "<p>Hi {{.name}}</p>", "body_type": "html" },
    "items": [
      { "to": ["alice@example.com"] },
      { "to": ["bob@example.com"] }
    ]
  }'

# Create a template
curl -X POST http://localhost:8087/v1/email/templates \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: t-001' \
  -d '{
    "name": "welcome",
    "subject": "Welcome {{.name}}",
    "body": "<p>{{ i18n \"hello\" \"en\" }} {{ .name }}!</p>",
    "body_type": "html"
  }'

# Render without sending
curl -X POST http://localhost:8087/v1/email/templates/$ID/render \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: t-001' \
  -d '{ "data": { "name": "Alice" } }'

# Stats
curl http://localhost:8087/v1/email/stats -H 'X-Tenant-ID: t-001'
```

## 11. Database schema

Tables (created idempotently on startup):

| Table | Purpose |
|-------|---------|
| `email.email_templates` | Reusable templates with version history + RLS |
| `email.email_logs` | Delivery log (status, driver, timing, error) + RLS |
| `email.email_attachments` | Tier-aware S3 attachment registry |
| `email.email_webhooks` | Provider webhook audit log |
| `email.tenant_settings` | Per-tenant driver / from / tier policy |