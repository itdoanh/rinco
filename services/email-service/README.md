# Email Service

Service gửi transactional emails với template support.

## Tính năng

- **SMTP Sending**: Gửi qua SMTP servers (Gmail, SendGrid, Mailgun, AWS SES)
- **Template Engine**: Go templates với data binding
- **Multi-language**: Hỗ trợ i18n templates
- **Attachments**: Đính kèm files
- **HTML & Plain Text**: Hỗ trợ cả 2 format
- **Bulk Sending**: Gửi nhiều email cùng lúc
- **Tracking**: Open tracking, click tracking
- **Queue**: Background processing với retry
- **Bounce Handling**: Xử lý bounce complaints

## Công nghệ

- **Language**: Go 1.23+
- **Framework**: Echo v4
- **SMTP**: go-mail
- **Templates**: text/template, html/template
- **Queue**: In-memory (single instance) hoặc Redis (multi-instance)

## API Endpoints

```
POST   /v1/email/send              - Gửi email raw
POST   /v1/email/send/template     - Gửi email với template
POST   /v1/email/send/bulk         - Bulk send
GET    /v1/email/templates         - List templates
GET    /v1/email/templates/:name   - Get template
POST   /v1/email/templates         - Create/update template
GET    /v1/email/track/:id         - Track open/click
GET    /health                     - Health check
```

## Send Email Request

```json
POST /v1/email/send
{
  "to": ["user@example.com"],
  "cc": ["cc@example.com"],
  "bcc": ["bcc@example.com"],
  "from": "noreply@rinco.vn",
  "from_name": "RINCO",
  "subject": "Welcome to RINCO",
  "html_body": "<h1>Hello!</h1>",
  "text_body": "Hello!",
  "attachments": [
    {
      "filename": "invoice.pdf",
      "content_base64": "...",
      "content_type": "application/pdf"
    }
  ],
  "headers": {
    "X-Campaign-ID": "welcome-2026"
  },
  "tags": ["welcome", "onboarding"]
}
```

## Template Send

```json
POST /v1/email/send/template
{
  "to": ["user@example.com"],
  "template_name": "welcome",
  "language": "vi",
  "data": {
    "user_name": "Nguyễn Văn A",
    "activation_link": "https://..."
  }
}
```

## Template Example

```
Subject: Welcome {{.user_name}}!

Hi {{.user_name}},

Welcome to RINCO! Click the link below to activate your account:
{{.activation_link}}

Best regards,
RINCO Team
```

## Environment Variables

```bash
EMAIL_SERVICE_PORT=8090
SMTP_HOST=smtp.mailgun.org
SMTP_PORT=587
SMTP_USERNAME=postmaster@mg.rinco.vn
SMTP_PASSWORD=secret
SMTP_FROM=noreply@rinco.vn
SMTP_FROM_NAME=RINCO
TEMPLATES_DIR=/templates
TRACKING_DOMAIN=track.rinco.vn
TRACKING_ENABLED=true
MAX_RETRIES=3
RETRY_DELAY=5s
```

## Local Development

```bash
# Use MailHog
SMTP_HOST=localhost
SMTP_PORT=1025

# Or MailSlurper, smtp4dev, etc.
```

## Development

```bash
go build -o bin/email-service ./cmd/main.go
./bin/email-service
```

## Deliverability

- SPF, DKIM, DMARC configured
- Dedicated IPs for high volume
- Warm-up schedule cho domains mới
- List unsubscribe (RFC 8058)
