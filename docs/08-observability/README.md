# Phần 8 – Observability 4 Tầng + AI SRE

> **Phân hệ:** "Đôi mắt thần" nhìn thấu toàn bộ hệ thống từ code đến hạ tầng.  
> **Mục tiêu:** Phát hiện lỗi trong vài giây ở hàng trăm VPS/Cluster, tìm đúng dòng code gây lỗi.  
> **Triết lý:** Lỗi chắc chắn xảy ra → phải phát hiện sớm, xử lý nhanh, tự phục hồi.

---

## Mục lục
1. [Mục tiêu & Triết lý](#1-mục-tiêu--triết-lý)
2. [Tầng 1 – Application Level](#2-tầng-1--application-level)
3. [Tầng 2 – Telemetry Triad](#3-tầng-2--telemetry-triad)
4. [Tầng 3 – Realtime Alerting + AI RCA](#4-tầng-3--realtime-alerting--ai-rca)
5. [Tầng 4 – Self-Healing](#5-tầng-4--self-healing)
6. [Công cụ cụ thể](#6-công-cụ-cụ-thể)
7. [Danh sách tính năng (≥ 100)](#7-danh-sách-tính-năng)
8. [Database Schema](#8-database-schema)
9. [API Surface](#9-api-surface)

---

## 1. Mục tiêu & Triết lý

### 1.1. Mục tiêu
| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| OBS-1 | Mọi request có trace_id | 100% |
| OBS-2 | Log search < 100ms (1B row) | ClickHouse |
| OBS-3 | Alert trigger < 30s | Time to alert |
| OBS-4 | AI RCA < 3s | Time to root cause |
| OBS-5 | Self-healing < 2s | K3s restart |

### 1.2. Triết lý "4 Tầng"
```
Tầng 1: Bắt lỗi tại code (App-level)
        ↓
Tầng 2: Gom về trung tâm (Telemetry Triad)
        ↓
Tầng 3: Báo cáo tự động + AI RCA
        ↓
Tầng 4: Tự phục hồi (Self-healing)
```

---

## 2. Tầng 1 – Application Level

### 2.1. Context Logging & Trace ID

#### Trace ID Generation
```go
import "github.com/google/uuid"

func NewTraceID() string {
    // UUIDv7 có timestamp → sort được
    id, _ := uuid.NewV7()
    return id.String()
}
```

#### Trace ID Propagation
```
[Client Request]
    ↓
[API Gateway]
    ├─► if !traceID in header: generate
    ├─► inject vào context.Context
    └─► X-Trace-ID header
        ↓
[Service A]
    ├─► Đọc X-Trace-ID từ header
    ├─► Gắn vào log (mọi log)
    ├─► Gắn vào span (OpenTelemetry)
    └─► Forward qua NATS:
         NATS Header: X-Trace-ID
        ↓
[Service B nhận NATS]
    ├─► Extract X-Trace-ID từ header
    ├─► Gắn vào log + span
    └─► ...
```

#### Implementation trong Go
```go
func (s *Service) HandleRequest(ctx context.Context, req *Request) error {
    traceID := getTraceID(ctx)  // Từ context
    
    s.log.Info("handling request",
        "trace_id", traceID,
        "tenant_id", req.TenantID,
        "user_id", req.UserID,
        "method", req.Method,
        "endpoint", req.Endpoint,
    )
    
    // Start OpenTelemetry span
    ctx, span := otel.Tracer("service").Start(ctx, "HandleRequest")
    defer span.End()
    
    // ...
    return err
}
```

#### Implementation trong Rust
```rust
use tracing::{info_span, Instrument};

#[tracing::instrument(skip(req), fields(trace_id = %get_trace_id()))]
async fn handle_request(req: Request) -> Result<Response, Error> {
    info!("handling request");
    // ...
    Ok(response)
}
```

### 2.2. Structured Logging

#### Standard JSON Schema
```json
{
  "timestamp": "2026-09-06T18:42:00.123Z",
  "level": "ERROR",
  "service": "crm-core",
  "version": "1.2.3",
  "trace_id": "018f3a9b-7c1e-7000-8000-123456789abc",
  "span_id": "abc123",
  "tenant_id": "apexfintech",
  "user_id": "uuid...",
  "request_id": "uuid...",
  "caller": {
    "file": "user_repository.go",
    "line": 142,
    "function": "CreateLead"
  },
  "message": "failed to insert lead to postgres",
  "error": {
    "type": "*pq.Error",
    "message": "pq: duplicate key value violates unique constraint",
    "stack": "..."
  },
  "context": {
    "lead_id": "uuid...",
    "phone": "+84xxx",
    "source": "facebook"
  }
}
```

#### Go (Zap)
```go
import "go.uber.org/zap"

var logger *zap.Logger

func InitLogger() {
    cfg := zap.NewProductionConfig()
    cfg.EncoderConfig.TimeKey = "timestamp"
    cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    logger, _ = cfg.Build(zap.AddCallerSkip(1))
}

func LogError(ctx context.Context, msg string, err error) {
    logger.Error(msg,
        zap.String("trace_id", getTraceID(ctx)),
        zap.String("tenant_id", getTenantID(ctx)),
        zap.Error(err),
        zap.Stack("stack"),
    )
}
```

#### Rust (tracing)
```rust
use tracing::{error, instrument};

#[instrument(skip_all, fields(trace_id = %trace_id))]
async fn create_lead(trace_id: String, lead: Lead) -> Result<LeadID, Error> {
    match db.insert(&lead).await {
        Ok(id) => Ok(id),
        Err(e) => {
            error!(error = %e, stack = ?std::backtrace::Backtrace::capture(), "failed to insert lead");
            Err(e.into())
        }
    }
}
```

### 2.3. Error Wrapping
```go
// Go: wrap error với context
func (r *LeadRepo) Create(ctx context.Context, lead *Lead) error {
    if err := r.db.Insert(lead); err != nil {
        return fmt.Errorf("LeadRepo.Create: tenant=%s phone=%s: %w", 
            lead.TenantID, lead.Phone, err)
    }
    return nil
}

// Ở tầng trên, tiếp tục wrap
func (s *LeadService) Create(ctx context.Context, lead *Lead) error {
    if err := s.repo.Create(ctx, lead); err != nil {
        return fmt.Errorf("LeadService.Create: %w", err)
    }
    return nil
}
```

### 2.4. PII Redaction
```go
func redactPII(data map[string]any) map[string]any {
    sensitive := []string{"phone", "email", "ssn", "credit_card"}
    for _, k := range sensitive {
        if v, ok := data[k]; ok {
            data[k] = maskString(v.(string))
        }
    }
    return data
}
```

---

## 3. Tầng 2 – Telemetry Triad

### 3.1. Sơ đồ
```
[Code trên các VPS]
       │
       ├─► (Logs) ───────► [Vector / FluentBit] ──► [ClickHouse / Grafana Loki]
       ├─► (Metrics) ─────► [Prometheus / VictoriaMetrics]
       └─► (Traces) ──────► [OpenTelemetry Collector] ──► [Jaeger / Tempo]
                                                       │
                                                       ▼
                                              [Bảng Điều Khiển Grafana]
```

### 3.2. Logs Center (ClickHouse)

#### Vector Config
```yaml
# vector.toml
[sources.app_logs]
type = "file"
include = ["/var/log/rinco/*.log"]

[transforms.parse_json]
type = "remap"
inputs = ["app_logs"]
source = '''
. = parse_json!(.message)
'''

[sinks.clickhouse]
type = "clickhouse"
inputs = ["parse_json"]
endpoint = "http://clickhouse:8123"
database = "rinco_logs"
table = "app_logs"
```

#### ClickHouse Schema
```sql
CREATE TABLE rinco_logs.app_logs (
  timestamp DateTime64(9),
  level LowCardinality(String),
  service LowCardinality(String),
  trace_id String,
  span_id String,
  tenant_id LowCardinality(String),
  user_id String,
  message String,
  error_message String,
  error_type String,
  caller_file LowCardinality(String),
  caller_line UInt32,
  attributes JSON,
  INDEX idx_trace trace_id TYPE bloom_filter(0.01) GRANULARITY 3,
  INDEX idx_message message TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 3
) ENGINE = MergeTree
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (service, timestamp)
TTL timestamp + INTERVAL 90 DAY;
```

#### Query Patterns
```sql
-- Search by trace
SELECT * FROM app_logs WHERE trace_id = '...' ORDER BY timestamp;

-- Errors in last 5 min for service X
SELECT * FROM app_logs 
WHERE service = 'crm-core' 
  AND level = 'ERROR' 
  AND timestamp > now() - INTERVAL 5 MINUTE
ORDER BY timestamp DESC LIMIT 100;

-- Count errors per service per hour
SELECT service, count(), min(timestamp), max(timestamp)
FROM app_logs
WHERE level = 'ERROR' AND timestamp > now() - INTERVAL 24 HOUR
GROUP BY service;
```

### 3.3. Metrics Center (VictoriaMetrics)

#### Go Prometheus Metrics
```go
import "github.com/prometheus/client_golang/prometheus"

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rinco_http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"service", "method", "path", "status"},
    )
    
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "rinco_http_request_duration_seconds",
            Help: "HTTP request duration",
            Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
        },
        []string{"service", "method", "path"},
    )
)

func Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        ww := &responseWriter{ResponseWriter: w, status: 200}
        next.ServeHTTP(ww, r)
        
        httpRequestsTotal.WithLabelValues(...).Inc()
        httpRequestDuration.WithLabelValues(...).Observe(time.Since(start).Seconds())
    })
}
```

#### Standard Metrics
- `rinco_http_requests_total{service,method,path,status}` – Counter.
- `rinco_http_request_duration_seconds{service,method,path}` – Histogram.
- `rinco_db_query_duration_seconds{db,query_type}` – Histogram.
- `rinco_db_connections_active{db}` – Gauge.
- `rinco_nats_publish_duration_seconds{topic}` – Histogram.
- `rinco_grpc_calls_total{service,method,status}` – Counter.
- `rinco_websocket_connections{service}` – Gauge.
- `rinco_active_meetings{tenant}` – Gauge.
- `rinco_active_chat_channels{tenant}` – Gauge.
- `rinco_ai_inference_duration_seconds{model}` – Histogram.
- `rinco_circuit_breaker_state{name}` – Gauge (0:closed, 1:half-open, 2:open).

#### Scrape Config (Prometheus)
```yaml
scrape_configs:
  - job_name: 'rinco-services'
    scrape_interval: 15s
    static_configs:
      - targets:
        - 'api-gateway:9090'
        - 'crm-core:9090'
        - 'chat-gateway:9090'
        # ...
```

### 3.4. Traces Center (OpenTelemetry)

#### Go Instrumentation
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitTracing() {
    exporter, _ := otlp.NewExporter(otlp.WithInsecure())
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.ServiceNameKey.String("crm-core"),
        )),
    )
    otel.SetTracerProvider(tp)
}

// Span
ctx, span := otel.Tracer("crm-core").Start(ctx, "CreateLead")
defer span.End()

span.SetAttributes(
    attribute.String("tenant.id", lead.TenantID),
    attribute.String("lead.source", lead.Source),
)

// Cross-service
span.AddEvent("publishing NATS event")
publisher.Publish(ctx, "lead.created", lead)
```

#### OTLP Collector Config
```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 5s
    send_batch_size: 1000

exporters:
  otlp/jaeger:
    endpoint: jaeger:4317
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/jaeger]
```

### 3.5. Grafana Dashboards

#### Pre-built Dashboards
1. **System Overview** – High-level metrics.
2. **API Gateway** – Request rate, latency, errors.
3. **Database Performance** – Query duration, connections.
4. **Service Map** – Topology with traffic.
5. **Resource Usage** – CPU, RAM, disk, network per node.
6. **WebRTC SFU** – Active meetings, packet loss.
7. **Chat Engine** – WS connections, message throughput.
8. **AI Inference** – Model latency, GPU usage.
9. **ClickHouse Analytics** – Query rate, OLAP performance.
10. **Tenant Health** – Per-tenant SLA.
11. **Cost Analysis** – Resource cost per tenant.
12. **Error Heatmap** – Errors by service/time.

---

## 4. Tầng 3 – Realtime Alerting + AI RCA

### 4.1. Sentry/GlitchTip Integration

#### Go Sentry
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
    hub := sentry.GetHubFromContext(ctx)
    hub.CaptureException(err)
}

// Capture với tags
sentry.CaptureException(err, 
    sentry.WithTags("tenant_id", tenantID),
    sentry.WithExtra("trace_id", traceID),
)
```

#### Frontend Sentry
```typescript
import * as Sentry from "@sentry/react";

Sentry.init({
  dsn: "https://...@sentry.io/...",
  integrations: [
    new Sentry.BrowserTracing(),
  ],
  tracesSampleRate: 0.1,
  beforeSend(event) {
    // Strip PII
    if (event.user) {
      delete event.user.email;
      delete event.user.phone;
    }
    return event;
  }
});
```

### 4.2. AI SRE – Auto Root Cause Analysis

#### Flow
```
[Error detected] 
    ↓
[Sentry groups errors into Issue]
    ↓
[Webhook → AI SRE Worker]
    ↓
[AI Worker receives: trace_id, error_message, stack_trace]
    ↓
1. Query ClickHouse: get all logs with trace_id
2. Query Jaeger: get trace spans
3. Read source code from Git repo (find file:line)
4. Use Code-LLM (DeepSeek-Coder) to analyze:
   - What's the error?
   - Why did it happen?
   - What's the fix?
5. Generate report → send to Telegram/Slack
6. (Optional) Auto-create PR with patch
```

#### AI SRE Prompt
```python
PROMPT = """
You are an expert SRE analyzing a production incident.

Error: {error_message}
Stack trace:
{stack_trace}

Recent logs from this trace_id:
{logs}

Source code context:
{code_snippet}

Provide:
1. Root cause (1-2 sentences).
2. Why this happened.
3. Suggested fix (code snippet).
4. Prevention strategy.

Be concise. Output as JSON.
"""
```

### 4.3. Alert Channels

#### Severity Levels
| Level | Description | Response |
|-------|-------------|----------|
| **P0/Critical** | Service down, data loss | Page on-call + SMS + Phone call |
| **P1/High** | Major feature broken | Telegram + Slack + Email |
| **P2/Medium** | Degraded but functional | Slack channel + dashboard |
| **P3/Low** | Warning | Dashboard only |

#### Telegram Bot
```go
func SendTelegramAlert(alert Alert) {
    msg := fmt.Sprintf("🚨 *%s*\nService: %s\nError: %s\nTrace: `%s`\n[Open in Grafana](%s)",
        alert.Severity, alert.Service, alert.Message, alert.TraceID, alert.Link)
    
    http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken), "application/json", 
        strings.NewReader(fmt.Sprintf(`{"chat_id":"%s","text":%q,"parse_mode":"Markdown"}`, chatID, msg)))
}
```

#### PagerDuty Integration
```go
func PageOnCall(alert Alert) {
    payload := map[string]any{
        "routing_key": pdKey,
        "event_action": "trigger",
        "payload": map[string]any{
            "summary": alert.Title,
            "severity": alert.Severity,
            "source": alert.Service,
            "custom_details": alert.Details,
        },
    }
    // POST to events.pagerduty.com
}
```

### 4.4. Anomaly Detection (AI)
```python
# AI detect traffic anomaly
def detect_anomaly(tenant_id, metric_name, window=300):
    history = get_metric(tenant_id, metric_name, hours=168)  # 1 week
    current = get_metric(tenant_id, metric_name, minutes=5)
    
    # Prophet / LSTM forecast
    forecast = model.predict(history)
    
    # Check deviation
    deviation = abs(current.mean() - forecast.yhat.mean()) / forecast.yhat.std()
    
    if deviation > 3:
        alert("anomaly_detected", deviation=deviation)
```

### 4.5. Capacity Planning AI
```python
def plan_capacity(tenant_id):
    usage_30d = get_usage(tenant_id, days=30)
    forecast = lstm.predict(usage_30d, days_ahead=30)
    
    if forecast.cpu > 0.8:
        recommend("scale_up", new_size="XL", eta_days=7)
    if forecast.storage > 0.9:
        recommend("add_storage", new_size="200GB")
```

---

## 5. Tầng 4 – Self-Healing

### 5.1. Circuit Breaker

```go
import "github.com/sony/gobreaker"

var fbCB = gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name: "facebook-capi",
    Timeout: 60 * time.Second,
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        return counts.ConsecutiveFailures > 5 || 
               counts.FailureRatio() > 0.5
    },
    OnStateChange: func(name string, from, to gobreaker.State) {
        log.Warn("circuit_breaker_state_change",
            "name", name, "from", from, "to", to)
        if to == gobreaker.StateOpen {
            alert.Send("circuit_open", name)
        }
    },
})

func SendCAPIEvent(event CAPIEvent) error {
    return fbCB.Execute(func() error {
        return sendToMeta(event)
    })
}
```

### 5.2. K3s Self-Healing

#### Liveness Probe
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: crm-core
spec:
  template:
    spec:
      containers:
      - name: app
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8890
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 3
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8890
          initialDelaySeconds: 5
          periodSeconds: 5
```

#### Health Endpoints
- `/health/live` – Process alive (return 200).
- `/health/ready` – Ready to serve traffic (check DB, cache).
- `/health/startup` – Initial startup check.

### 5.3. Graceful Degradation

#### Fallback Strategy
```python
def get_analytics(tenant_id, query):
    try:
        return clickhouse_query(query)
    except Exception as e:
        log.error("clickhouse_failed", error=e)
        # Fallback to PostgreSQL slow query
        return postgres_query(fallback_query(query))
```

#### Caching Fallback
```go
// If DB down, serve stale cache
func GetLead(id string) (*Lead, error) {
    if val, ok := cache.Get("lead:" + id); ok {
        if dbHealthy() {
            return db.Get(id)
        }
        // Serve stale
        return val.(*Lead), nil
    }
    return db.Get(id)
}
```

### 5.4. Auto-Rollback
```yaml
# ArgoCD Application
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: crm-core
spec:
  source:
    repoURL: https://github.com/itdoanh/rinco
    path: services/crm-core
    targetRevision: HEAD
  destination:
    server: https://kubernetes.default.svc
    namespace: rinco
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    retry:
      limit: 5
      backoff:
        duration: 10s
        factor: 2
        maxDuration: 5m
```

### 5.5. Database Self-Healing
- **PostgreSQL:** PgBouncer + auto-restart on connection failure.
- **ScyllaDB:** Auto-rebalance shards.
- **ClickHouse:** Keeper election tự động.
- **Valkey:** Sentinel hoặc Cluster mode với failover.

---

## 6. Công cụ cụ thể

### 6.1. Ma trận công cụ

| Phân tầng | Công nghệ | Tích hợp |
|-----------|-----------|---------|
| **Code Level** | Zap (Go) / tracing (Rust) + OpenTelemetry | Auto-instrument |
| **Log Center** | Vector → ClickHouse | Native client |
| **Metrics** | VictoriaMetrics + Grafana | Prometheus |
| **Traces** | OpenTelemetry → Jaeger | OTLP |
| **Error Tracking** | Sentry / GlitchTip | SDK |
| **Alerting** | Alertmanager + Telegram + PagerDuty | Webhook |
| **AI RCA** | DeepSeek-Coder / Llama-3 + OpenTelemetry | Custom |
| **Resilience** | gobreaker + K3s Probe | Built-in |

### 6.2. Local Dev (docker-compose)
- ClickHouse + Vector + VictoriaMetrics + Jaeger + Grafana + Sentry.
- All in single node.
- Profile `obs` in docker-compose.

---

## 7. Danh sách tính năng (≥ 100)

### 7.1. Logging (1-25)
1. JSON structured log.
2. Log level (DEBUG/INFO/WARN/ERROR/FATAL).
3. Trace ID tự động.
4. Caller info (file:line).
5. PII redaction.
6. Log sampling (rate).
7. Log buffering.
8. Log compression.
9. Async log write.
10. Log rotation.
11. Log archive (S3 cold storage).
12. Log search (full-text).
13. Log filter (level, service, trace).
14. Saved searches.
15. Share log query qua link.
16. Live tail log.
17. Log highlight (regex).
18. Context log (extra fields).
19. Audit log riêng (không mix).
20. Log retention policy (per service).
21. Log ship qua Vector.
22. Log streaming qua WebSocket.
23. Log viewer mobile-friendly.
24. Log export (CSV/JSON).
25. Log correlation (group theo trace).

### 7.2. Metrics (26-50)
26. Prometheus metrics endpoint.
27. Custom metric registration.
28. Histogram với custom buckets.
29. Counter, Gauge, Summary.
30. HTTP request duration.
31. DB query duration.
32. NATS publish duration.
34. Cache hit/miss.
35. Active connections gauge.
36. Queue depth gauge.
37. Memory usage per service.
38. Goroutine count.
39. Goroutine leak detection.
40. Custom business metrics (Lead created/min).
41. Per-tenant metric.
42. Aggregated metric per region.
43. Metric label cardinality limit.
44. Push gateway (batch metrics).
45. Metric retention.
46. Recording rules.
47. Alert rules.
48. Dashboard auto-provisioning.
49. Metric export Prometheus format.
50. OpenMetrics format support.

### 7.3. Tracing (51-75)
51. OpenTelemetry SDK.
52. Auto-instrumentation (HTTP, DB, gRPC).
53. Manual span creation.
54. Span attributes.
55. Span events.
56. Span links (causal relationship).
57. Trace context propagation (W3C).
58. Trace sampling (head/tail).
59. Trace tail-based sampling.
60. Span error tracking.
61. Trace export OTLP.
62. Trace export Jaeger.
63. Trace export Zipkin.
64. Trace ID in logs.
65. Trace in errors.
66. Trace UI (Jaeger/Tempo).
67. Trace search.
68. Trace comparison.
69. Trace flamegraph.
70. Trace service map.
71. Trace latency breakdown.
72. Trace export.
73. Trace-based metrics.
74. Trace exemplar.
75. Trace-based alerting.

### 7.4. Alerting (76-95)
76. Prometheus Alertmanager.
77. Multi-channel alert.
78. Alert grouping.
79. Alert deduplication.
80. Alert silence.
81. Alert inhibition.
82. Alert escalation.
83. On-call schedule.
84. On-call rotation.
85. Alert webhook.
86. Alert to Telegram.
87. Alert to Slack.
88. Alert to Discord.
89. Alert to PagerDuty.
90. Alert to Opsgenie.
91. Alert via SMS (Twilio).
92. Alert via Phone call.
93. Alert severity routing.
94. Alert acknowledgment.
95. Alert resolution tracking.

### 7.5. AI SRE (96-115)
96. Auto error grouping.
97. Auto RCA (Root Cause Analysis).
98. Auto Hotfix Proposal (PR).
99. Auto issue create (GitHub).
100. AI capacity planning.
101. AI cost optimization.
102. AI anomaly detection.
103. AI performance regression detection.
104. AI log pattern clustering.
105. AI dependency analysis.
106. AI change impact analysis.
107. AI weekly report.
108. AI incident post-mortem.
109. AI runbook generation.
110. AI alert noise reduction.
111. AI service map generation.
112. AI on-call suggestion.
113. AI runbook execution.
114. AI Playbook automation.
115. AI knowledge base search.

### 7.6. Self-Healing (116-130)
116. Circuit Breaker pattern.
117. Auto-restart (K3s).
118. Health check endpoint.
119. Liveness probe.
120. Readiness probe.
121. Startup probe.
122. Auto-scaling (HPA).
123. Cluster auto-scaling.
124. Node auto-scaling.
125. Auto-rollback deploy.
126. Graceful shutdown.
127. Connection draining.
128. Resource limit (CPU/RAM).
129. Pod disruption budget.
130. Disaster recovery drill.

---

## 8. Database Schema

### 8.1. PostgreSQL (Alert Rules, On-call)
```sql
CREATE TABLE alert_rules (
  id UUID PRIMARY KEY,
  tenant_id UUID,
  name TEXT NOT NULL,
  query TEXT NOT NULL,           -- PromQL
  condition TEXT NOT NULL,        -- '>','<','=='
  threshold DOUBLE PRECISION,
  duration_seconds INT,
  severity TEXT,                  -- 'P0','P1','P2','P3'
  channels TEXT[],                -- ['telegram','slack','pagerduty']
  enabled BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE oncall_schedules (
  id UUID PRIMARY KEY,
  team TEXT,
  primary_user_id UUID,
  secondary_user_id UUID,
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ
);

CREATE TABLE alert_history (
  id UUID PRIMARY KEY,
  alert_rule_id UUID,
  fired_at TIMESTAMPTZ,
  resolved_at TIMESTAMPTZ,
  severity TEXT,
  acknowledged_by UUID,
  notes TEXT,
  ai_rca JSONB                   -- AI's root cause analysis
);

CREATE TABLE silences (
  id UUID PRIMARY KEY,
  matchers JSONB,
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ,
  created_by UUID,
  comment TEXT
);
```

---

## 9. API Surface

### 9.1. Internal Observability
```
GET    /metrics                              # Prometheus
GET    /health/live
GET    /health/ready
GET    /health/startup

POST   /api/obs/v1/logs/search
GET    /api/obs/v1/logs/:trace_id
GET    /api/obs/v1/traces/:trace_id
GET    /api/obs/v1/metrics/query              # PromQL
GET    /api/obs/v1/dashboards

GET    /api/obs/v1/alerts/active
POST   /api/obs/v1/alerts/:id/acknowledge
POST   /api/obs/v1/alerts/:id/resolve
POST   /api/obs/v1/silences

GET    /api/obs/v1/ai/anomalies
GET    /api/obs/v1/ai/rca/:incident_id
POST   /api/obs/v1/ai/capacity-plan
```

---

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-OBS-01 | 100% request có trace_id | Audit |
| AC-OBS-02 | Log search < 100ms trên 1B row | ClickHouse |
| AC-OBS-03 | Alert < 30s | E2E test |
| AC-OBS-04 | AI RCA < 3s | Benchmark |
| AC-OBS-05 | Self-healing < 2s | Chaos test |

---

**Tiếp theo:** [`docs/09-security/README.md`](../09-security/README.md) – Multi-Tenant Security Zero-Trust.