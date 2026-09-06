# Notification Service

> **Phân hệ #7 — Multi-channel notifications** · In-app / Email / SMS / Push / FCM / Slack / Discord / Telegram / Webhook.
> Priority matrix (high = in-app+email+push, normal = in-app+email, low = in-app only).
> User preferences (per-type enable/disable + quiet hours + daily digest).

## 1. Endpoints (10)

| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/health` / `/ready` / `/metrics` | Health / Prometheus |
| POST | `/v1/notifications/send` | Gửi cho 1 user, multi-channel |
| POST | `/v1/notifications/broadcast` | Broadcast theo audience |
| GET | `/v1/notifications` | List in-app store (filter `user_id`, `unread=1`) |
| GET | `/v1/notifications/:id` | Retrieve single |
| POST | `/v1/notifications/:id/read` | Mark as read |
| POST | `/v1/notifications/preferences/:user_id` | Set user prefs |
| GET | `/v1/notifications/preferences/:user_id` | Get user prefs |
| POST | `/v1/notifications/subscriptions` | Register push endpoint (web/fcm) |
| GET | `/v1/notifications/stats` | Counter snapshot |

## 2. Channels & Drivers

| Channel | Driver / Backend |
|---------|------------------|
| `in_app` | Internal store (proxy to chat-engine WS in production) |
| `email`  | HTTP delegate → email-service |
| `sms`    | Twilio (HTTP API, basic auth) |
| `push`   | Web Push (subscription endpoint) |
| `fcm`    | Firebase Cloud Messaging |
| `slack`  | Incoming webhook |
| `discord`| Incoming webhook |
| `telegram` | Bot API |
| `webhook` | Generic POST |

## 3. Priority matrix

| Priority | Default channels |
|----------|------------------|
| high   | in_app + email + push |
| normal | in_app + email |
| low    | in_app |

Có thể override qua `channels` array trong request.

## 4. User Preferences

```json
{
  "channels": {"in_app": true, "email": true, "sms": false, "push": true},
  "quiet_hours": [22, 7],
  "daily_digest": false,
  "types": {"lead_scored": true, "task_assigned": true}
}
```

- `quiet_hours` [start, end] hỗ trợ qua đêm (vd `[22, 7]`)
- Trong quiet hours, in-app vẫn được lưu; SMS/email/push bị skip
- `types` map cho phép per-type enable

## 5. ENV

| Var | Default | Purpose |
|-----|---------|---------|
| `PORT` | `8088` | HTTP port |
| `EMAIL_SERVICE_URL` | `http://email-service:8087` | Email upstream |
| `PUSH_SERVICE_URL` | `http://push-worker:9000/send` | Web Push |
| `TWILIO_ACCOUNT_SID` / `TWILIO_AUTH_TOKEN` / `TWILIO_FROM` | - | SMS |
| `FCM_SERVER_KEY` | - | Firebase Cloud Messaging |
| `TELEGRAM_BOT_TOKEN` | - | Telegram bot |
| `SLACK_DEFAULT_WEBHOOK` | - | Default Slack channel |

## 6. Run

```bash
cd services/notification-service
go build ./cmd && ./notification-service
```

## 7. Ví dụ

```bash
curl -X POST http://localhost:8088/v1/notifications/send \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id": "u-123",
    "type": "lead_scored",
    "title": "Lead mới nóng",
    "message": "Alice vừa đạt 92 điểm",
    "priority": "high",
    "data": {"lead_id": "L-001", "score": 92}
  }'
```
