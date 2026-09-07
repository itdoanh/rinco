# Notification Service

> **Phân hệ #7 — Multi-channel notifications** · In-app / Email / SMS / Push / FCM / Slack / Discord / Telegram.
> Per-user preferences with quiet hours & digest mode. Priority matrix (high = in-app + email + push,
> normal = in-app + email, low = in-app only). Daily aggregates for analytics.

## 1. Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| GET    | `/healthz` / `/readyz` / `/metrics` / `/version` | Liveness / readiness / Prometheus / version |
| POST   | `/v1/notifications/send` | Fan-out to a single user |
| POST   | `/v1/notifications/broadcast` | Broadcast by `user_ids` / `role` / `department` |
| GET    | `/v1/notifications` | List (filter `user_id`, `status`, `type`, `from`, `cursor`) |
| GET    | `/v1/notifications/:id` | Retrieve single |
| POST   | `/v1/notifications/:id/read` | Mark as read |
| POST   | `/v1/preferences/:user_id` | Set per-type channel preferences |
| GET    | `/v1/preferences/:user_id` | Get preferences |
| POST   | `/v1/subscriptions/webpush` | Register VAPID push endpoint |
| DELETE | `/v1/subscriptions/webpush/:id` | Delete push subscription |
| POST   | `/v1/subscriptions/fcm` | Register FCM device token |
| GET    | `/v1/stats` | Counts by status / channel / top types |

Connect-RPC adapter:

```
POST /internal/notification.v1.NotificationService/SendNotification
POST /internal/notification.v1.NotificationService/BroadcastNotification
POST /internal/notification.v1.NotificationService/MarkAsRead
POST /internal/notification.v1.NotificationService/GetUserPreferences
POST /internal/notification.v1.NotificationService/SetUserPreferences
POST /internal/notification.v1.NotificationService/RegisterPushSubscription
POST /internal/notification.v1.NotificationService/GetNotificationFeed
```

## 2. Channels & Drivers

Each channel is a self-contained `channels.Driver`:

```go
type Driver interface {
    Name() string
    Send(ctx context.Context, n Notification) (DeliveryResult, error)
}
```

| Channel | Backend | Implementation |
|---------|---------|----------------|
| `in_app` | NATS pub/sub → chat-engine WS | `internal/channels/inapp.go` |
| `email`  | HTTP delegate → email-service | `internal/channels/email.go` |
| `sms`    | Twilio HTTP API | `internal/channels/sms.go` |
| `push`   | Web Push (VAPID) | `internal/channels/push.go` |
| `fcm`    | Firebase Cloud Messaging | `internal/channels/push.go` |
| `slack`  | Incoming webhook | `internal/channels/slack.go` |
| `discord`| Incoming webhook | `internal/channels/slack.go` |
| `telegram` | Bot API | `internal/channels/telegram.go` |

## 3. Priority Matrix

| Priority | Default channels |
|----------|------------------|
| `high`   | `in_app` + `email` + `push` |
| `normal` | `in_app` + `email` |
| `low`    | `in_app` only |

User preferences can override (per `(user, type, channel)`). When quiet hours are
in effect (configured by `[start_hour, end_hour]`, supports overnight like `[22, 7]`),
all channels except `in_app` are skipped.

```go
channels = req.Channels
if len(channels) == 0 {
    channels = preferences.DefaultPriorityChannels[priority]
}
channels = resolver.Resolve(priority, req.Type, req.UserID)
```

## 4. User Preferences

```json
{
    "channels": { "in_app": true, "email": true, "sms": false, "push": true },
    "quiet_hours": [22, 7],
    "digest_mode": "none",
    "types": { "lead_scored": true, "task_assigned": true }
}
```

Stored in `notification.notification_preferences` keyed by `(user_id, notif_type, channel)`.

## 5. NATS Subjects

The service subscribes to:

```
notification.broadcast
user.notification
lead.scored
lead.assigned
task.created
```

And publishes in-app notifications on:

```
user.{user_id}.notification
```

## 6. Database schema

| Table | Purpose |
|-------|---------|
| `notification.notifications` | Per-user notification records + RLS |
| `notification.notification_preferences` | Per-user per-type channel settings |
| `notification.push_subscriptions` | VAPID push endpoints |
| `notification.fcm_subscriptions` | FCM device tokens |
| `notification.notification_delivery_logs` | Per-channel delivery audit |
| `notification.daily_aggregates` | Counts by (date, tenant, type, channel) |

## 7. ENV

| Var | Default | Purpose |
|-----|---------|---------|
| `NOTIF_HTTP_ADDR` | `:8088` | HTTP listen address |
| `NOTIF_DATABASE_URL` | - | Postgres DSN |
| `NOTIF_VALKEY_URL` | `localhost:6379` | Redis/Valkey |
| `NOTIF_NATS_URL` | - | NATS server |
| `NOTIF_EMAIL_RPC_URL` | `http://email-service:8087` | email-service base |
| `NOTIF_TWILIO_SID/TOKEN/FROM` | - | SMS provider |
| `NOTIF_FCM_PROJECT_ID` | - | FCM |
| `NOTIF_FCM_CREDENTIALS_FILE` | - | Service-account JSON |
| `NOTIF_VAPID_PUBLIC/PRIVATE/SUBJECT` | - | Web Push VAPID |
| `NOTIF_WEB_BASE_URL` | `https://app.rinco.app` | Web base URL |
| `NOTIF_SLACK_WEBHOOK` | - | Default Slack webhook |
| `NOTIF_TELEGRAM_BOT_TOKEN` | - | Telegram bot |

## 8. Run

```bash
cd services/notification-service
go build ./cmd && ./notification-service
```

Docker:

```bash
docker build -t rinco/notification-service -f services/notification-service/Dockerfile .
```

## 9. Curl examples

```bash
# Send a notification (default priority = normal → in_app + email)
curl -X POST http://localhost:8088/v1/notifications/send \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: t-001' \
  -d '{
    "user_id": "u-123",
    "type": "lead_scored",
    "title": "Lead mới nóng",
    "body": "Alice vừa đạt 92 điểm",
    "priority": "high",
    "data": { "lead_id": "L-001", "score": 92 },
    "email": "alice@example.com"
  }'

# Broadcast to a department
curl -X POST http://localhost:8088/v1/notifications/broadcast \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: t-001' \
  -d '{
    "type": "announcement",
    "title": "All-hands meeting",
    "body": "Tomorrow at 10am",
    "priority": "normal",
    "audience": { "department": "engineering" }
  }'

# List notifications (cursor pagination)
curl "http://localhost:8088/v1/notifications?user_id=u-123&limit=20" \
  -H 'X-Tenant-ID: t-001'

# Set preferences
curl -X POST http://localhost:8088/v1/preferences/u-123 \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: t-001' \
  -d '{
    "channels": { "in_app": true, "email": true, "sms": false, "push": true },
    "quiet_hours": [22, 7],
    "digest_mode": "none"
  }'

# Register a Web Push subscription
curl -X POST http://localhost:8088/v1/subscriptions/webpush \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: t-001' \
  -d '{
    "user_id": "u-123",
    "endpoint": "https://fcm.googleapis.com/...",
    "p256dh": "...",
    "auth": "..."
  }'

# Register an FCM device token
curl -X POST http://localhost:8088/v1/subscriptions/fcm \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: t-001' \
  -d '{
    "user_id": "u-123",
    "device_token": "...",
    "platform": "android",
    "app_version": "1.0.0"
  }'

# Stats
curl http://localhost:8088/v1/stats -H 'X-Tenant-ID: t-001'
```