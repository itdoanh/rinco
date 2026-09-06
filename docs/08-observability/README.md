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

---

# PHẦN MỞ RỘNG – Audit, Edge Cases, Code Examples, Roadmap

> Phần này bổ sung cho tài liệu gốc, cung cấp implementation chi tiết cho observability stack.

---

## 8. Audit Report (Self-Audit)

### 8.1. Những gì đã đủ chi tiết ✓
- 4 tầng observability với triết lý rõ ràng.
- Schema ClickHouse, Prometheus metrics.
- Alert severity matrix.

### 8.2. Cần bổ sung ⚠️
- Code examples cho mỗi tầng.
- Sequence diagrams.
- Edge cases cụ thể cho mỗi loại lỗi.
- Testing strategy chi tiết.
- Disaster recovery runbook.
- Cost estimation per component.

### 8.3. Mâu thuẫn nội bộ ❌
- Hiện không có mâu thuẫn lớn nhưng cần cross-check với `11-ai-integration` (AI SRE overlap).

---

## 9. Edge Cases & Error Scenarios (≥ 30)

### 9.1. Logging Edge Cases
| # | Scenario | Triệu chứng | Xử lý |
|---|----------|------------|-------|
| 1 | Buffer log tràn do traffic spike | Log bị drop | Sampling theo rate, priority queue |
| 2 | Trace ID collision (cùng UUIDv7 ns) | Trace bị merge | Thêm entropy từ pid+random |
| 3 | PII leak do developer quên redact | Sentry báo | Lint rule, PII scanner trong CI |
| 4 | Log JSON parse fail | Vector drop | Fallback về raw text + alert |
| 5 | Clock skew giữa các service | Trace ordering sai | NTP, monotonic clock fallback |
| 6 | Disk đầy ở Vector agent | Log ship fail | Drain mode + buffer to memory |
| 7 | Async log block khi DB down | Goroutine leak | Circuit breaker + buffered write |

### 9.2. Metrics Edge Cases
| # | Scenario | Triệu chứng | Xử lý |
|---|----------|------------|-------|
| 8 | Prometheus scrape timeout | Metric bị miss | Increase timeout, chunked |
| 9 | High cardinality labels | Prometheus OOM | Drop unused labels, sampling |
| 10 | Counter reset do pod restart | Rate calculation sai | Use `rate()` over window |
| 11 | Histogram bucket miss | p99 sai | Auto-bucket tuning, native histograms |
| 12 | VictoriaMetrics WAL corruption | Data loss | WAL replication + snapshot |
| 13 | Recording rule conflict | Alert sai | Namespaced rules, audit |
| 14 | Multi-tenant metric leak | Tenant A thấy metric B | Force `tenant_id` label + Prometheus rule |

### 9.3. Tracing Edge Cases
| # | Scenario | Triệu chứng | Xử lý |
|---|----------|------------|-------|
| 15 | Trace context loss ở boundary | Orphan spans | W3C Trace Context + manual inject |
| 16 | Tail sampling fail | Important trace miss | Head sampling fallback |
| 17 | Jaeger indexer OOM | Search chậm | TTL ngắn, eslim snapshot |
| 18 | OTLP exporter backpressure | Span drop | Rate-limit sampling |
| 19 | Span limit exceeded | Sampling | Limit max spans per trace |
| 20 | Trace ID format mismatch (W3C vs B3) | Trace split | Configurable propagator |

### 9.4. AI SRE Edge Cases
| # | Scenario | Triệu chứng | Xử lý |
|---|----------|------------|-------|
| 21 | AI hallucination RCA | Fix sai | Human review gate, confidence score |
| 22 | Prompt injection từ log content | AI leak secret | Input sanitization |
| 23 | LLM API timeout | RCA fail | Local model fallback |
| 24 | Code-LLM suggest bad fix | Production break | Dry-run + canary deploy |
| 25 | Sentry rate limit | Error bị miss | Group + sample |
| 26 | Auto PR tạo quá nhiều | Spam | Limit per day, deduplicate |

### 9.5. Self-Healing Edge Cases
| # | Scenario | Triệu chứng | Xử lý |
|---|----------|------------|-------|
| 27 | Circuit breaker stuck open | Permanent fail | Half-open probe sau TTL |
| 28 | K3s restart loop (CrashLoopBackOff) | Pod không start | Exponential backoff, alert |
| 29 | Fallback to stale cache quá lâu | Data sai | Stale-while-revalidate + TTL |
| 30 | Rollback fail | Version cũ hỏng | Blue-green + version pinning |
| 31 | DB failover không graceful | Connection drop | PgBouncer + retry middleware |
| 32 | Liveness probe too strict | Restart liên tục | Tune thresholds |

---

## 10. Code Examples chi tiết

### 10.1. Go – Structured Logger hoàn chỉnh
```go
// pkg/logger/logger.go
package logger

import (
    "context"
    "os"
    "sync"
    "time"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

type ctxKey string

const (
    traceIDKey   ctxKey = "trace_id"
    tenantIDKey  ctxKey = "tenant_id"
    userIDKey    ctxKey = "user_id"
    spanIDKey    ctxKey = "span_id"
    requestIDKey ctxKey = "request_id"
)

var (
    baseLogger *zap.Logger
    once       sync.Once
)

// Init khởi tạo logger 1 lần với JSON output.
func Init(service, env, version string) {
    once.Do(func() {
        encoderCfg := zap.NewProductionEncoderConfig()
        encoderCfg.TimeKey = "timestamp"
        encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
        encoderCfg.MessageKey = "message"
        encoderCfg.LevelKey = "level"
        encoderCfg.CallerKey = "caller"

        core := zapcore.NewCore(
            zapcore.NewJSONEncoder(encoderCfg),
            zapcore.Lock(os.Stdout),
            zap.NewAtomicLevelAt(zap.InfoLevel),
        )

        baseLogger = zap.New(core,
            zap.AddCaller(),
            zap.AddCallerSkip(1),
            zap.Fields(
                zap.String("service", service),
                zap.String("env", env),
                zap.String("version", version),
                zap.String("host", hostname()),
            ),
        )
    })
}

// WithContext trích context fields rồi gắn vào logger.
func WithContext(ctx context.Context) *zap.Logger {
    l := baseLogger
    if v, ok := ctx.Value(traceIDKey).(string); ok && v != "" {
        l = l.With(zap.String("trace_id", v))
    }
    if v, ok := ctx.Value(tenantIDKey).(string); ok && v != "" {
        l = l.With(zap.String("tenant_id", v))
    }
    if v, ok := ctx.Value(userIDKey).(string); ok && v != "" {
        l = l.With(zap.String("user_id", v))
    }
    if v, ok := ctx.Value(spanIDKey).(string); ok && v != "" {
        l = l.With(zap.String("span_id", v))
    }
    if v, ok := ctx.Value(requestIDKey).(string); ok && v != "" {
        l = l.With(zap.String("request_id", v))
    }
    return l
}

// Info, Warn, Error shortcuts.
func Info(ctx context.Context, msg string, fields ...zap.Field) {
    WithContext(ctx).Info(msg, fields...)
}
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
    WithContext(ctx).Warn(msg, fields...)
}
func Error(ctx context.Context, msg string, err error, fields ...zap.Field) {
    all := append(fields,
        zap.Error(err),
        zap.String("error_type", errorType(err)),
    )
    WithContext(ctx).Error(msg, all...)
}

// Helper
func hostname() string {
    h, _ := os.Hostname()
    return h
}

func errorType(err error) string {
    if err == nil {
        return ""
    }
    // Lấy type name đầu tiên qua reflection-free path
    t := fmt.Sprintf("%T", err)
    return t
}
```

### 10.2. Go – Trace Middleware (HTTP)
```go
// pkg/middleware/trace.go
package middleware

import (
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
)

const traceHeader = "X-Trace-ID"

func Trace() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            req := c.Request()
            ctx := req.Context()

            // 1. Lấy trace context từ W3C Trace Context (nếu có)
            prop := otel.GetTextMapPropagator()
            ctx = prop.Extract(ctx, propagation.HeaderCarrier(req.Header))

            // 2. Sinh trace_id mới nếu chưa có
            sc := trace.SpanContextFromContext(ctx)
            var traceID string
            if sc.HasTraceID() {
                traceID = sc.TraceID().String()
            } else {
                traceID = uuid.NewV7().String()
            }

            // 3. Set vào context custom (logger đọc từ đây)
            ctx = context.WithValue(ctx, traceIDKey, traceID)
            ctx = context.WithValue(ctx, requestIDKey, uuid.NewV7().String())

            // 4. Start root span
            tracer := otel.Tracer("rinco")
            spanName := req.Method + " " + c.Path()
            ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
            defer span.End()

            span.SetAttributes(
                attribute.String("http.method", req.Method),
                attribute.String("http.target", req.RequestURI),
                attribute.String("http.scheme", req.URL.Scheme),
                attribute.String("net.peer.ip", c.RealIP()),
                attribute.String("user_agent.original", req.UserAgent()),
            )

            // 5. Set header response
            c.Response().Header().Set(traceHeader, traceID)

            // 6. Tiếp tục chain
            err := next(c)

            // 7. Gắn status code
            status := c.Response().Status
            span.SetAttributes(attribute.Int("http.status_code", status))

            if err != nil {
                span.RecordError(err)
            }
            if status >= 500 {
                span.SetStatus(codes.Error, "5xx response")
            }

            return err
        }
    }
}
```

### 10.3. Go – Prometheus Metrics Middleware
```go
// pkg/middleware/metrics.go
package middleware

import (
    "strconv"
    "time"

    "github.com/labstack/echo/v4"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rinco_http_requests_total",
            Help: "Total HTTP requests processed",
        },
        []string{"service", "method", "route", "status"},
    )
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "rinco_http_request_duration_seconds",
            Help:    "HTTP request duration",
            Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
        },
        []string{"service", "method", "route"},
    )
    httpInflight = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "rinco_http_inflight_requests",
            Help: "Number of inflight HTTP requests",
        },
        []string{"service"},
    )
)

func Metrics(serviceName string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            req := c.Request()
            start := time.Now()
            inflight := httpInflight.WithLabelValues(serviceName)
            inflight.Inc()
            defer inflight.Dec()

            err := next(c)

            route := c.Path()
            status := strconv.Itoa(c.Response().Status)
            duration := time.Since(start).Seconds()

            httpRequestsTotal.WithLabelValues(serviceName, req.Method, route, status).Inc()
            httpRequestDuration.WithLabelValues(serviceName, req.Method, route).Observe(duration)

            return err
        }
    }
}
```

### 10.4. Rust – tracing Setup cho service
```rust
// src/observability.rs
use opentelemetry::global;
use opentelemetry::propagation::Extractor;
use opentelemetry::trace::TracerProvider as _;
use opentelemetry::KeyValue;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::propagation::TraceContextPropagator;
use opentelemetry_sdk::trace::SdkTracerProvider;
use opentelemetry_sdk::Resource;
use tracing_subscriber::layer::SubscriberExt;
use tracing_subscriber::util::SubscriberInitExt;
use tracing_subscriber::EnvFilter;

pub fn init(service: &str, env: &str, otlp: &str) -> anyhow::Result<()> {
    global::set_text_map_propagator(TraceContextPropagator::new());

    let exporter = opentelemetry_otlp::SpanExporter::builder()
        .with_http()
        .with_endpoint(otlp)
        .build()?;

    let provider = SdkTracerProvider::builder()
        .with_resource(Resource::new(vec![
            KeyValue::new("service.name", service.to_string()),
            KeyValue::new("deployment.environment", env.to_string()),
        ]))
        .with_batch_exporter(exporter)
        .build();

    global::set_tracer_provider(provider.clone());
    let tracer = provider.tracer("rinco");

    let telemetry = tracing_opentelemetry::layer().with_tracer(tracer);

    let fmt_layer = tracing_subscriber::fmt::layer()
        .json()
        .with_current_span(true)
        .with_span_list(false)
        .with_target(true)
        .with_file(true)
        .with_line_number(true);

    tracing_subscriber::registry()
        .with(EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")))
        .with(fmt_layer)
        .with(telemetry)
        .init();

    Ok(())
}

#[derive(Debug)]
pub struct HeaderExtractor<'a>(pub &'a axum::http::HeaderMap);

impl<'a> Extractor for HeaderExtractor<'a> {
    fn get(&self, key: &str) -> Option<&str> {
        self.0.get(key).and_then(|v| v.to_str().ok())
    }
    fn keys(&self) -> Vec<&str> {
        self.0.keys().map(|k| k.as_str()).collect()
    }
}
```

### 10.5. Python – AI SRE Worker (rút gọn)
```python
# services/ai-sre/main.py
import asyncio
import json
import os
from datetime import datetime
from typing import Any

import httpx
from fastapi import FastAPI, BackgroundTasks
from pydantic import BaseModel
import structlog

logger = structlog.get_logger(__name__)
app = FastAPI(title="AI SRE Worker")


class Incident(BaseModel):
    incident_id: str
    service: str
    trace_id: str | None
    error: str
    stack: str | None = None
    severity: str = "P2"
    sentry_url: str | None = None


class RCAResult(BaseModel):
    incident_id: str
    root_cause: str
    why: str
    suggested_fix: str
    prevention: str
    confidence: float
    file: str | None = None
    line: int | None = None
    pr_url: str | None = None


async def fetch_logs(trace_id: str) -> list[dict[str, Any]]:
    """ClickHouse query."""
    async with httpx.AsyncClient() as client:
        r = await client.get(
            f"http://clickhouse:8123/?query=SELECT+*+FROM+rinco_logs.app_logs+WHERE+trace_id='{trace_id}'+LIMIT+50",
            headers={"X-ClickHouse-User": "readonly"},
        )
        r.raise_for_status()
        return json.loads(r.text)


async def fetch_source(file: str, line: int) -> str:
    """Read source via GitHub API."""
    repo = os.getenv("GITHUB_REPO", "itdoanh/rinco")
    ref = os.getenv("GITHUB_REF", "main")
    token = os.getenv("GITHUB_TOKEN", "")
    async with httpx.AsyncClient() as client:
        r = await client.get(
            f"https://api.github.com/repos/{repo}/contents/{file}?ref={ref}",
            headers={"Authorization": f"Bearer {token}"},
        )
        r.raise_for_status()
        return base64.b64decode(r.json()["content"]).decode()


def build_prompt(error: str, stack: str, logs: list, source: str) -> str:
    log_lines = "\n".join(
        f"[{l.get('level')}] {l.get('message')} ({l.get('caller_file', '')}:{l.get('caller_line', '')})"
        for l in logs
    )
    return f"""You are an expert SRE AI.

Error: {error}
Stack:
{stack or '(no stack)'}

Recent logs:
{log_lines[:3000]}

Source (file:line):
```
{source[:5000]}
```

Provide a JSON with keys: root_cause, why, suggested_fix (code), prevention, file, line, confidence (0-1).
Be concise. No preamble."""


async def call_llm(prompt: str) -> dict:
    """Gọi vLLM local."""
    async with httpx.AsyncClient(timeout=60.0) as client:
        r = await client.post(
            "http://vllm:8000/v1/chat/completions",
            json={
                "model": "deepseek-coder-v2-lite-instruct",
                "messages": [{"role": "user", "content": prompt}],
                "max_tokens": 2000,
                "temperature": 0.1,
            },
        )
        r.raise_for_status()
        return r.json()["choices"][0]["message"]["content"]


@app.post("/analyze")
async def analyze(incident: Incident, bg: BackgroundTasks) -> dict:
    """Trigger RCA ngay."""
    logger.info("incident.received", incident_id=incident.incident_id, severity=incident.severity)

    logs = []
    if incident.trace_id:
        try:
            logs = await fetch_logs(incident.trace_id)
        except Exception as e:
            logger.warning("logs.fetch_failed", error=str(e))

    file, line, source = "(unknown)", 0, ""
    if incident.stack:
        # Parse stack đơn giản: lấy frame đầu tiên có file.go:line
        # ...
        file = "pkg/repo/lead.go"
        line = 142
    if file != "(unknown)":
        try:
            source = await fetch_source(file, line)
        except Exception as e:
            logger.warning("source.fetch_failed", error=str(e))

    prompt = build_prompt(incident.error, incident.stack or "", logs, source)
    result_text = await call_llm(prompt)

    # Parse JSON
    try:
        rca = json.loads(result_text)
    except Exception:
        # Fallback: regex
        rca = {"raw": result_text, "root_cause": "unparseable", "confidence": 0.0}

    rca["incident_id"] = incident.incident_id

    # Send Telegram alert
    bg.add_task(notify_telegram, incident, rca)

    # Auto-create PR if confidence high and severity >= P1
    if rca.get("confidence", 0) > 0.8 and incident.severity in ("P0", "P1"):
        bg.add_task(auto_create_pr, rca, source)

    return rca


async def notify_telegram(incident: Incident, rca: dict):
    token = os.getenv("TELEGRAM_BOT_TOKEN", "")
    chat_id = os.getenv("TELEGRAM_CHAT_ID", "")
    if not token:
        return
    msg = f"""🚨 *{incident.severity}* – `{incident.service}`

*Error*: `{incident.error[:200]}`

*RCA*: {rca.get('root_cause', '?')[:500]}

*Fix*: 
```
{rca.get('suggested_fix', '?')[:1000]}
```

*Confidence*: {rca.get('confidence', 0):.0%}

Trace: `{incident.trace_id}`
[Open Sentry]({incident.sentry_url or '#'})
"""
    async with httpx.AsyncClient() as client:
        await client.post(
            f"https://api.telegram.org/bot{token}/sendMessage",
            json={"chat_id": chat_id, "text": msg, "parse_mode": "Markdown"},
        )


async def auto_create_pr(rca: dict, source: str):
    repo = os.getenv("GITHUB_REPO", "itdoanh/rinco")
    token = os.getenv("GITHUB_TOKEN", "")
    branch = f"hotfix/ai-{rca['incident_id'][:8]}"
    # ... use PyGithub or raw API ...
```

### 10.6. YAML – Prometheus Alert Rules
```yaml
# infra/prometheus/rules/rinco.yml
groups:
- name: rinco.core
  rules:
  - alert: HighErrorRate
    expr: |
      sum(rate(rinco_http_requests_total{status=~"5.."}[5m])) by (service, route)
      /
      sum(rate(rinco_http_requests_total[5m])) by (service, route)
      > 0.01
    for: 1m
    labels:
      severity: P1
      team: sre
    annotations:
      summary: "Error rate > 1% on {{ $labels.service }} {{ $labels.route }}"
      runbook: "https://wiki.rinco.app/runbooks/high-error-rate"
      dashboard: "https://grafana.rinco.app/d/{{ $labels.service }}"

  - alert: HighLatencyP99
    expr: |
      histogram_quantile(0.99, sum(rate(rinco_http_request_duration_seconds_bucket[5m])) by (service, route, le))
      > 0.5
    for: 2m
    labels:
      severity: P1
    annotations:
      summary: "P99 > 500ms on {{ $labels.service }}/{{ $labels.route }}"

  - alert: DiskSpaceFilling
    expr: |
      (node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) < 0.1
    for: 5m
    labels:
      severity: P2
    annotations:
      summary: "Disk < 10% on {{ $labels.instance }}"

  - alert: PostgresConnectionsHigh
    expr: |
      pg_stat_activity_count > 80
    for: 1m
    labels:
      severity: P2

  - alert: ScyllaWriteTimeout
    expr: |
      rate(scylla_write_timeouts_total[5m]) > 0.1
    for: 1m
    labels:
      severity: P1

  - alert: CIRCUIT_OPEN
    expr: rinco_circuit_breaker_state == 2
    for: 0m
    labels:
      severity: P1
    annotations:
      summary: "Circuit breaker OPEN for {{ $labels.name }}"

- name: rinco.tenant
  rules:
  - alert: TenantStorageQuotaNear
    expr: rinco_tenant_storage_bytes / rinco_tenant_storage_quota_bytes > 0.9
    for: 10m
    labels:
      severity: P2
    annotations:
      summary: "Tenant {{ $labels.tenant_id }} near storage quota"

  - alert: TenantAPIErrors
    expr: |
      sum(rate(rinco_http_requests_total{tenant_id=~".+",status=~"5.."}[10m])) by (tenant_id)
      > 5
    for: 1m
    labels:
      severity: P2
```

---

## 11. Implementation Roadmap (chi tiết theo tuần)

### Tuần 1-2: Application Logging
- [ ] Setup `pkg/logger` Go với Zap.
- [ ] Setup `observability.rs` Rust với tracing.
- [ ] Trace ID middleware cho Echo + Axum.
- [ ] PII redaction helper + lint rule.
- [ ] Unit test logger.
- **Acceptance:** Mọi HTTP request có trace_id, PII được redact.

### Tuần 3-4: Metrics + Prometheus
- [ ] Metrics middleware Go + Rust.
- [ ] Custom business metrics (leads_created_total, chat_messages_total).
- [ ] Prometheus scrape config + service discovery.
- [ ] Grafana datasource + 5 dashboard cơ bản.
- **Acceptance:** Metrics xuất hiện ở `/metrics`, Prometheus scrape thành công.

### Tuần 5-6: Distributed Tracing
- [ ] OpenTelemetry SDK setup Go + Rust + Python.
- [ ] OTLP Collector deployment.
- [ ] Jaeger hoặc Tempo deployment.
- [ ] Auto-instrumentation cho HTTP, DB, gRPC.
- **Acceptance:** Trace end-to-end xuất hiện trong Jaeger UI.

### Tuần 7-8: Error Tracking + Alerting
- [ ] Sentry SDK Go + Frontend.
- [ ] Alertmanager deployment.
- [ ] Alert rules cho critical metrics.
- [ ] Telegram bot + on-call rotation.
- **Acceptance:** Lỗi xuất hiện trong Sentry, alert gửi qua Telegram.

### Tuần 9-10: AI SRE
- [ ] vLLM deployment với DeepSeek-Coder.
- [ ] AI SRE worker (Python).
- [ ] ClickHouse + GitHub integration.
- [ ] Auto PR creation.
- **Acceptance:** Sentry webhook → RCA report → Telegram.

### Tuần 11-12: Chaos + Polish
- [ ] Chaos Mesh deployment.
- [ ] Game day drills.
- [ ] Runbook cho mỗi alert.
- [ ] Cost optimization.
- **Acceptance:** Self-healing trong < 2s cho 5 scenarios.

---

## 12. Disaster Recovery

### 12.1. ClickHouse Recovery
```bash
# Backup
clickhouse-backup create --config config.yml
clickhouse-backup upload default

# Restore
clickhouse-backup download default
clickhouse-backup restore default
```

### 12.2. Prometheus Recovery
```bash
# WAL recovery
docker exec prometheus promtool tsdb recover /prometheus

# Restart với snapshot
docker-compose restart prometheus
```

### 12.3. Jaeger Recovery
- Storage backend: Elasticsearch hoặc Cassandra.
- Snapshot mỗi ngày.
- Restore từ snapshot → reindex nếu cần.

### 12.4. Sentry Recovery
- Self-hosted GlitchTip.
- Postgres backend.
- Backup daily qua pg_dump.

---

## 13. Cost Estimation

| Component | Spec | Cost/month (USD) |
|-----------|------|-----------------|
| ClickHouse (3 nodes) | 8 vCPU, 32GB, 1TB NVMe | $450 |
| VictoriaMetrics (1 node) | 4 vCPU, 16GB, 500GB | $120 |
| Grafana Cloud Pro | 10K series | $290 |
| Jaeger (3 nodes) | 4 vCPU, 16GB | $240 |
| Sentry (self-hosted) | 4 vCPU, 16GB | $120 |
| Vector agents | Per pod (lightweight) | included in cluster |
| vLLM (GPU) | 1x A100 | $1,200 |
| **Total** | | **~$2,420** |

---

## 14. Open Questions / Cần user xác nhận

1. **Self-hosted Sentry hay cloud?** Ảnh hưởng cost + PII control.
2. **LLM cho AI SRE: DeepSeek-Coder hay Llama-3-Code?** Trade-off cost vs quality.
3. **Retention bao lâu cho log/trace?** Ảnh hưởng storage cost.
4. **Multi-region observability?** Tăng cost 2-3x.
5. **Sentry sampling rate cho production?** 10% traces default OK?
6. **Alert group strategy:** group by service hay by tenant?
7. **Auto-PR workflow có bật cho production không?** Risk vs benefit.
8. **Chaos engineering schedule (weekly drill)?**
9. **OpenTelemetry SDK version pinning?**
10. **Cost budget cho observability stack per month?**

---

## 15. Acceptance Criteria bổ sung

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-OBS-07 | AI SRE RCA accuracy ≥ 70% | Manual review |
| AC-OBS-08 | Self-healing MTTR < 30s | Chaos test |
| AC-OBS-09 | Log PII leak = 0 | Scan |
| AC-OBS-10 | Prometheus uptime ≥ 99.95% | Uptime check |
| AC-OBS-11 | Alert false positive < 5% | Audit |
| AC-OBS-12 | Cost per service < $200/month | Billing |