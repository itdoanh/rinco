# Landing Service

Service xử lý landing pages, lead submissions và Facebook CAPI events.

## Tính năng

- **Static File Serving**: Serve landing page content
- **Lead Submission API**: Form submission với validation
- **HMAC Validation**: Anti-tampering cho lead data
- **Idempotency**: Duplicate submission protection
- **Dynamic Model Validation**: Validate fields theo tenant schema
- **Multi-DB Persistence**: ScyllaDB (raw) + PostgreSQL (rich)
- **Facebook CAPI**: Server-side conversion tracking
- **Browser Event Tracking**: Page views, clicks, conversions
- **Static Asset Hosting**: CDN-friendly cache headers

## Công nghệ

- **Language**: Go 1.23+
- **Framework**: Echo v4
- **Databases**: ScyllaDB + PostgreSQL
- **Cache**: Valkey/Redis
- **External APIs**: Facebook Conversions API

## API Endpoints

```
GET    /                                        - Serve landing page
GET    /static/*                                - Static assets
POST   /api/v1/leads                            - Submit lead
POST   /api/v1/events                           - Track browser events
POST   /api/v1/track                            - Track custom event
POST   /api/v1/capi/send                        - Manual CAPI send (internal)
GET    /api/v1/health                           - Health check
```

## Lead Submission Flow

```
1. Client submits form
   ↓
2. Validate HMAC signature
   ↓
3. Check idempotency key (Redis)
   ↓
4. Validate against Dynamic Model schema
   ↓
5. Save to ScyllaDB (raw, fast)
   ↓
6. Save to PostgreSQL (rich, queryable)
   ↓
7. Trigger Facebook CAPI
   ↓
8. Trigger workflow automation
   ↓
9. Return success
```

## HMAC Validation

```go
func validateHMAC(payload []byte, signature string, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

Client-side:
```javascript
const payload = JSON.stringify(data);
const signature = CryptoJS.HmacSHA256(payload, SECRET).toString();
fetch('/api/v1/leads', {
  method: 'POST',
  headers: { 'X-HMAC-Signature': signature },
  body: payload
});
```

## Facebook CAPI

```go
type CAPIEvent struct {
    EventName      string    `json:"event_name"`
    EventTime      int64     `json:"event_time"`
    UserData       UserData  `json:"user_data"`
    CustomData     CustomData `json:"custom_data"`
    EventSourceURL string    `json:"event_source_url"`
    ActionSource   string    `json:"action_source"`
}
```

Events sent:
- `Lead` - Lead submission
- `CompleteRegistration` - Account creation
- `Purchase` - Conversion
- `Contact` - Form submission

## Environment Variables

```bash
LANDING_SERVICE_PORT=8085
DATABASE_URL=postgres://postgres:postgres@localhost:5432/rinco?sslmode=disable
SCYLLA_SEEDS=scylla1,scylla2,scylla3
REDIS_URL=redis://valkey:6379
HMAC_SECRET=your-hmac-secret
FB_PIXEL_ID=your-pixel-id
FB_ACCESS_TOKEN=your-access-token
FB_CAPI_ENABLED=true
DYNAMIC_MODEL_SERVICE_URL=http://dynamic-model-service:8086
```

## Development

```bash
go build -o bin/landing-service ./cmd/main.go
./bin/landing-service
```

## Performance

- **Lead submission latency**: < 200ms p99
- **Throughput**: 1000 leads/sec/instance
- **CAPI success rate**: > 99%
- **HMAC validation**: < 5ms
