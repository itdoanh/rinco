---
name: rinco-facebook-capi
description: Skill về Facebook Conversions API, Hybrid Dual-Tracking, CAPI Poisoning defense, EMQ optimization.
---

# RINCO Facebook CAPI Skill

## Nguyên tắc

### Hybrid Dual-Tracking
- Client-side: Meta Pixel.
- Server-side: Conversions API.
- **Cùng `event_id`** → Meta tự deduplicate.

### CAPI Poisoning Defense
- Mỗi event bắt buộc có HMAC signature.
- Server verify trước khi gửi Meta.
- Reject nếu signature không hợp lệ.

### EMQ (Event Match Quality)
- Target: ≥ 7/10.
- Gửi đầy đủ user_data fields.
- Track EMQ theo tenant, alert nếu < 6.

## SHA-256 Normalization (Meta Standard)

```go
import (
    "crypto/sha256"
    "strings"
    "regexp"
)

func NormalizeEmail(email string) string {
    return sha256Hash(strings.ToLower(strings.TrimSpace(email)))
}

func NormalizePhone(phone string) string {
    digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")
    if !strings.HasPrefix(digits, "84") {
        digits = strings.TrimPrefix(digits, "0")
        digits = "84" + digits
    }
    return sha256Hash(digits)
}

func NormalizeName(name string) string {
    return sha256Hash(strings.ToLower(strings.TrimSpace(name)))
}

func sha256Hash(s string) string {
    h := sha256.Sum256([]byte(s))
    return hex.EncodeToString(h[:])
}
```

## User Data Payload

```go
type UserData struct {
    Email         []string `json:"em"`              // SHA-256
    Phone         []string `json:"ph"`              // SHA-256
    FirstName     []string `json:"fn"`              // SHA-256
    LastName      []string `json:"ln"`              // SHA-256
    City          []string `json:"ct"`              // SHA-256
    State         []string `json:"st"`              // SHA-256
    Country       []string `json:"country"`         // ISO 3166-1 alpha-2, SHA-256
    ZipCode       []string `json:"zp"`              // SHA-256
    ExternalID    []string `json:"external_id"`     // RINCO user_id, SHA-256
    ClientIP      string   `json:"client_ip_address"`
    ClientUA      string   `json:"client_user_agent"`
    FBC           string   `json:"fbc"`
    FBP           string   `json:"fbp"`
    SubscriptionID string  `json:"subscription_id"`
}
```

## HMAC Signature

```go
import (
    "crypto/hmac"
    "crypto/sha256"
)

func GenerateLeadSignature(leadID, fbclid, timestamp string, payload []byte, secretKey string) string {
    h := hmac.New(sha256.New, []byte(secretKey))
    h.Write([]byte(leadID))
    h.Write([]byte(fbclid))
    h.Write([]byte(timestamp))
    h.Write(payload)
    return hex.EncodeToString(h.Sum(nil))
}

func VerifyLeadSignature(leadID, fbclid, timestamp string, payload []byte, signature, secretKey string) bool {
    expected := GenerateLeadSignature(leadID, fbclid, timestamp, payload, secretKey)
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

## CAPI Worker

```go
type CAPIWorker struct {
    metaClient *meta.Client
    secretKey  string
    cb         *gobreaker.CircuitBreaker
}

func (w *CAPIWorker) Process(event CAPIEvent) error {
    // Verify HMAC
    if !VerifyLeadSignature(event.LeadID, event.FBC, event.Timestamp, event.Payload, event.Signature, w.secretKey) {
        return ErrInvalidSignature
    }
    
    return w.cb.Execute(func() error {
        return w.metaClient.SendEvent(event)
    })
}
```

## Event Types

| Event | Khi nào | Value |
|-------|---------|-------|
| `Lead` | Form submit | 0 |
| `QualifiedLead` | AI score ≥ 70 | 0 |
| `Schedule` | Đặt lịch họp | 0 |
| `Purchase` | Won deal | Deal value |
| `Subscribe` | Subscription active | Plan price |

## Server-Side Flow

```
[Lead Form Submit] → [Go Ingestion API]
                          ↓
                   [Normalize user_data]
                          ↓
                   [Generate HMAC]
                          ↓
                   [NATS Queue]
                          ↓
                   [FB CAPI Worker]
                          ↓
                   [Verify HMAC]
                          ↓
                   [Send to Meta Graph API]
                          ↓
                   [Log response]
                          ↓
                   [Update ClickHouse metric]
```

## Client-Side (Browser)

```typescript
// On form submit
const eventID = uuidv7();

fbq('track', 'Lead', {
  content_name: 'Apex Fintech Landing',
  value: 0,
  currency: 'VND',
}, {
  eventID: eventID,  // Cùng server-side
});

// Sau đó POST form data với cùng eventID
fetch('/api/ingest/v1/leads', {
  method: 'POST',
  body: JSON.stringify({
    ...formData,
    event_id: eventID,
  }),
});
```

## Anti-Bot

### Wasm Attestation
- Client chạy Wasm module verify browser là real (không Headless).
- Pass → cấp `X-RINCO-Attestation` token.

### Argon2 PoW
- Khi traffic spike → Gateway issue challenge.
- Client compute Argon2 trong ~20ms.
- Bot không có JS → fail.

## Best Practices

1. **Luôn gửi đầy đủ user_data** để tăng EMQ.
2. **Dùng event_id** matching giữa client và server.
3. **Retry với exponential backoff** (3 lần).
4. **Circuit breaker** khi Meta lỗi liên tục.
5. **Log đầy đủ response** để debug.
6. **Monitor EMQ** theo tenant.
7. **Test Event Code** khi dev.
8. **Domain Verification** phải pass trước khi go-live.