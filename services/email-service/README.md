# Email Service

> **Phân hệ #6 — Email** · Multi-driver transactional email (SMTP / Resend / SendGrid / SES / Console).
> Template engine (Go `text/template`) + Markdown→HTML + tracking pixel + click redirector + webhook.

## 1. Endpoints (12+)

| Method | Path                                 | Mô tả |
|--------|--------------------------------------|-------|
| GET    | `/health` / `/ready` / `/metrics`    | Liveness / readiness / Prometheus |
| POST   | `/v1/email/send`                     | Send 1 email (text/html/markdown) |
| POST   | `/v1/email/batch`                    | Bulk send tối đa 500 |
| POST   | `/v1/email/templates`                | Create template |
| GET    | `/v1/email/templates`                | List |
| GET    | `/v1/email/templates/:id`            | Retrieve |
| PUT    | `/v1/email/templates/:id`            | Update |
| DELETE | `/v1/email/templates/:id`            | Delete |
| POST   | `/v1/email/templates/:id/render`     | Render with variables (no-send) |
| GET    | `/v1/email/logs`                     | Delivery log theo tenant |
| GET    | `/v1/email/logs/:id`                 | Chi tiết 1 log |
| GET    | `/v1/email/stats`                    | Tổng hợp open / click / bounce |
| POST   | `/v1/email/webhooks/:provider`       | Bounce / complaint / delivery |
| GET    | `/e/:msg_id.gif`                     | 1x1 tracking pixel |
| GET    | `/c/:msg_id`                         | Click → redirect |

## 2. Drivers

| Driver    | ENV                                            | Notes                                |
|-----------|------------------------------------------------|--------------------------------------|
| `console` | `EMAIL_DRIVER=console`                         | Print to stdout — best for dev       |
| `smtp`    | `SMTP_HOST/PORT/USER/PASSWORD/SMTP_FROM`       | Native smtp + gomail fallback        |
| `resend`  | `RESEND_API_KEY`                               | HTTP POST `api.resend.com/emails`    |
| `sendgrid`| `SENDGRID_API_KEY`                             | HTTP POST `api.sendgrid.com/v3/mail/send` |
| `ses`     | `AWS_REGION` + `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` | Minimal SigV4 stub (production cần aws-sdk-go-v2) |

Driver per request: `X-Tenant-Driver: <driver>` (nếu không → default từ `EMAIL_DRIVER`).

## 3. Template Engine

Go `text/template` + custom funcs:
- `upper`, `lower`, `escape`
- `dateFormat "2006-01-02"`
- `safeHTML "<...>"`
- Biến truyền qua `Variables map[string]string`

Ví dụ template body:
```
<p>Hi <strong>{{.name}}</strong>,</p>
<p>Your order #{{.order_id}} was placed on {{dateFormat "02 Jan 2006"}}.</p>
```

## 4. Markdown → HTML (built-in, no deps)

Supports: `# heading`, `**bold**`, `*italic*`, `[text](url)`.

## 5. Tracking

- 1×1 GIF pixel tại `/e/:msg_id.gif` (no-cache)
- Click rewriting: mọi `<a href="https://...">` được rewrite thành `/c/:msg_id?url=<encoded>`
- Bounce / complaint từ provider webhook gọi `/v1/email/webhooks/:provider`

## 6. ENV

| Var | Default | Purpose |
|-----|---------|---------|
| `PORT` | `8087` | HTTP port |
| `ENV` | `development` | - |
| `EMAIL_DRIVER` | `console` | default fallback driver |
| `EMAIL_TRACK_BASE` | `/` | base path cho tracking links |
| `SMTP_HOST/PORT/USER/PASSWORD/SMTP_FROM` | - | SMTP fallback |

## 7. Run

```bash
cd services/email-service
go build ./cmd && ./email-service
```

Docker:
```bash
docker build -t rinco/email-service -f Dockerfile .
```

## 8. Ví dụ

```bash
curl -X POST http://localhost:8087/v1/email/send \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-Driver: console' \
  -d '{
    "to": ["alice@example.com"],
    "subject": "Welcome",
    "body": "# Hello **Alice**\nWelcome to RINCO!",
    "body_type": "markdown"
  }'
```

```bash
# Render template without sending
curl -X POST http://localhost:8087/v1/email/templates/$ID/render \
  -H 'Content-Type: application/json' \
  -d '{"variables":{"name":"Alice"}}'
```
