# Notification Service

Multi-channel notification service: Telegram, Slack, Discord, Webhook, SMS, Push.

## Tính năng

- **Telegram**: Gửi tin nhắn qua Telegram Bot API
- **Slack**: Slack Webhook URLs
- **Discord**: Discord Webhooks
- **Webhook**: Custom HTTP webhooks
- **SMS**: Twilio, Vonage, MessageBird
- **Push**: FCM (Firebase Cloud Messaging) cho mobile
- **Email**: Delegate tới email-service
- **Templates**: Multi-language template support
- **Routing**: Route theo user preferences
- **Priority**: High/normal/low với different delivery
- **Retry**: Auto-retry với exponential backoff

## Công nghệ

- **Language**: Go 1.23+
- **Framework**: Echo v4
- **HTTP Client**: net/http với connection pooling
- **Templates**: Go templates

## API Endpoints

```
POST   /v1/notify                        - Generic notify
POST   /v1/notify/telegram               - Telegram
POST   /v1/notify/slack                  - Slack
POST   /v1/notify/discord                - Discord
POST   /v1/notify/webhook                - Webhook
POST   /v1/notify/sms                    - SMS
POST   /v1/notify/push                   - Push notification
POST   /v1/notify/email                  - Email (proxy)
GET    /v1/notify/preferences/:user_id   - Get user preferences
PATCH  /v1/notify/preferences/:user_id   - Update preferences
GET    /health                           - Health check
```

## Generic Notify

```json
POST /v1/notify
{
  "user_id": "uuid",
  "tenant_id": "uuid",
  "channel": "telegram",  // telegram|slack|discord|webhook|sms|push|email
  "template": "lead_assigned",
  "data": {
    "lead_name": "Nguyễn Văn A",
    "assigned_by": "Manager X"
  },
  "priority": "high"
}
```

## Telegram Example

```json
POST /v1/notify/telegram
{
  "chat_id": "123456789",
  "text": "🎉 Lead mới: Nguyễn Văn A (điểm: 92/100)",
  "parse_mode": "HTML",
  "reply_markup": {
    "inline_keyboard": [
      [
        {"text": "Xem chi tiết", "url": "https://..."}
      ]
    ]
  }
}
```

## Slack Example

```json
POST /v1/notify/slack
{
  "webhook_url": "https://hooks.slack.com/...",
  "channel": "#sales",
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "*New High-Value Lead*\nNguyễn Văn A - 100M VND"
      }
    }
  ]
}
```

## Push Notification

```json
POST /v1/notify/push
{
  "user_id": "uuid",
  "title": "Lead mới",
  "body": "Nguyễn Văn A vừa đăng ký",
  "data": {
    "lead_id": "uuid",
    "deep_link": "rinco://leads/uuid"
  },
  "badge": 5,
  "sound": "default"
}
```

## Environment Variables

```bash
NOTIFICATION_SERVICE_PORT=8091
TELEGRAM_BOT_TOKEN=...
SLACK_DEFAULT_WEBHOOK=...
TWILIO_ACCOUNT_SID=...
TWILIO_AUTH_TOKEN=...
TWILIO_FROM_NUMBER=...
FCM_SERVER_KEY=...
MAX_RETRIES=3
RETRY_BACKOFF=exponential
QUEUE_SIZE=10000
WORKERS=10
```

## Development

```bash
go build -o bin/notification-service ./cmd/main.go
./bin/notification-service
```

## User Preferences

Users có thể config:
- Kênh nào enable/disable
- Quiet hours
- Priority filters
- Language

## Routing Logic

1. User receives notification
2. Check user preferences
3. Route to enabled channels
4. Apply template + data
5. Send via channel API
6. Retry on failure
7. Track delivery status
