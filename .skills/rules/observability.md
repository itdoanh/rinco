---
name: rinco-observability
description: Skill về logging, tracing, metrics, alerting, AI SRE trong RINCO.
---

# RINCO Observability Skill

## Triết lý 4 tầng
1. **Application Level:** Structured log + trace_id.
2. **Telemetry Triad:** Logs + Metrics + Traces → Trung tâm.
3. **AI Alerting:** Sentry + AI RCA + Telegram.
4. **Self-Healing:** Circuit Breaker + K3s restart + Fallback.

## Trace ID

### Sinh UUIDv7
```go
import "github.com/google/uuid"
traceID, _ := uuid.NewV7()
// → "018f3a9b-7c1e-7000-8000-123456789abc"
```

### Propagation
```
[Client] → [Gateway] → [Service A] → [NATS] → [Service B] → [Database]
   X-Trace-ID header truyền suốt
```

### Go Middleware
```go
func TraceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        traceID := r.Header.Get("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.NewV7().String()
        }
        
        ctx := context.WithValue(r.Context(), "trace_id", traceID)
        w.Header().Set("X-Trace-ID", traceID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Structured Log

### Format (BẮT BUỘC JSON)
```json
{
  "timestamp": "2026-09-06T18:42:00.123Z",
  "level": "INFO",
  "service": "crm-core",
  "trace_id": "018f3a9b-...",
  "tenant_id": "apex-fintech",
  "user_id": "uuid...",
  "caller": {"file": "lead.go", "line": 42, "function": "Create"},
  "message": "lead created",
  "context": {...}
}
```

### Helper
```go
func LogInfo(ctx context.Context, msg string, fields ...zap.Field) {
    logger.Info(msg, append(
        []zap.Field{
            zap.String("trace_id", getTraceID(ctx)),
            zap.String("tenant_id", getTenantID(ctx)),
        },
        fields...,
    )...)
}
```

### PII Redaction
```go
func Redact(sensitiveFields []string, data map[string]any) map[string]any {
    for _, k := range sensitiveFields {
        if _, ok := data[k]; ok {
            data[k] = "[REDACTED]"
        }
    }
    return data
}
// sensitiveFields = ["password", "phone", "email", "ssn", "credit_card"]
```

## Metrics

### Standard Metrics (PHẢI có)
```go
var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "rinco_http_requests_total"},
        []string{"service", "method", "path", "status"},
    )
    httpDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "rinco_http_duration_seconds",
            Buckets: []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1, 5},
        },
        []string{"service", "method", "path"},
    )
)
```

### Custom Metrics
- Business: `rinco_leads_created_total{tenant_id}`, `rinco_chat_messages_total{tenant_id}`.
- Infrastructure: `rinco_db_connections{db}`, `rinco_cache_hit_total{cache}`.

## Distributed Tracing

### OpenTelemetry Setup
```go
import "go.opentelemetry.io/otel"

func InitTracing(serviceName string) {
    exporter, _ := otlp.NewExporter(otlp.WithInsecure())
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.ServiceNameKey.String(serviceName),
        )),
    )
    otel.SetTracerProvider(tp)
}

func Span(ctx context.Context, name string) (context.Context, trace.Span) {
    return otel.Tracer("service").Start(ctx, name)
}
```

### Span Attributes
```go
span.SetAttributes(
    attribute.String("tenant.id", tenantID),
    attribute.String("user.id", userID),
    attribute.String("db.system", "postgresql"),
    attribute.String("db.statement", sql),
)
```

## Error Tracking (Sentry)

### Go
```go
import "github.com/getsentry/sentry-go"

func InitSentry() {
    sentry.Init(sentry.ClientOptions{
        Dsn: os.Getenv("SENTRY_DSN"),
        Environment: os.Getenv("ENV"),
        Release: os.Getenv("RELEASE"),
        TracesSampleRate: 0.1,
    })
}

func ReportError(ctx context.Context, err error) {
    sentry.GetHubFromContext(ctx).CaptureException(err)
}
```

### Frontend
```typescript
Sentry.init({
  dsn: "...",
  integrations: [new Sentry.BrowserTracing()],
  tracesSampleRate: 0.1,
  beforeSend(event) {
    // Strip PII
    return event;
  }
});
```

## Alerting

### Severity Levels
| Level | Response |
|-------|----------|
| P0 Critical | Page + SMS + Phone call |
| P1 High | Telegram + Slack |
| P2 Medium | Slack channel |
| P3 Low | Dashboard |

### Alert Rules
```yaml
groups:
- name: rinco
  rules:
  - alert: HighErrorRate
    expr: |
      sum(rate(rinco_http_requests_total{status=~"5.."}[5m]))
      /
      sum(rate(rinco_http_requests_total[5m]))
      > 0.01
    for: 1m
    labels:
      severity: P1
    annotations:
      summary: "Error rate > 1% on {{ $labels.service }}"
```

## Circuit Breaker Pattern
```go
import "github.com/sony/gobreaker"

cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name: "facebook-capi",
    Timeout: 60 * time.Second,
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        return counts.ConsecutiveFailures > 5
    },
})

err := cb.Execute(func() error {
    return callFacebookCAPI()
})
```

## Self-Healing

### K3s Probes
```yaml
livenessProbe:
  httpGet: {path: /health/live, port: 8890}
  initialDelaySeconds: 30
  periodSeconds: 10
readinessProbe:
  httpGet: {path: /health/ready, port: 8890}
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Health Endpoints
- `/health/live` – Process alive.
- `/health/ready` – Ready to serve (check DB, cache).
- `/health/startup` – Initial startup check.

## AI SRE

### Auto RCA Flow
```
[Error detected] 
    → [Sentry groups into Issue]
    → [Webhook → AI SRE Worker]
    → [Query ClickHouse logs by trace_id]
    → [Query Jaeger trace spans]
    → [Read source code via Git API]
    → [Code-LLM (DeepSeek-Coder) analysis]
    → [Generate RCA + Hotfix]
    → [Telegram notification]
    → [Optional: Auto PR]
```

### AI SRE Prompt Template
```
Analyze this production incident.

Error: {error}
Stack: {stack}

Logs:
{recent_logs}

Source (file:line):
{source_code}

Provide JSON:
{
  "root_cause": "...",
  "why": "...",
  "fix": "code snippet",
  "prevention": "...",
  "severity": "P0|P1|P2|P3"
}
```

## Grafana Dashboard

### Pre-built Dashboards
1. System Overview
2. API Gateway
3. Database Performance
4. Service Map
5. Resource Usage
6. WebRTC SFU
7. Chat Engine
8. AI Inference
9. Tenant Health
10. Cost Analysis

## Best Practices

1. **Mọi request có trace_id** (100%).
2. **Log levels: DEBUG < INFO < WARN < ERROR < FATAL**.
3. **Không log PII** (password, phone, email, SSN).
4. **Sampling:** 100% errors, 10% info.
5. **Retention:** Logs 90 ngày, Metrics 1 năm, Traces 30 ngày.
6. **Alert < 30s từ khi lỗi xảy ra.**
7. **RCA < 3 giây.**