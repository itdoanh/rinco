# Observability 4 Tầng + AI SRE — Tóm tắt

> Tài liệu tóm tắt `docs/08-observability/README.md`. Triết lý cốt lõi: **"Lỗi chắc chắn xảy ra → phải phát hiện sớm, xử lý nhanh, tự phục hồi"**. Mục tiêu là biến hàng trăm VPS/Cluster thành một hệ thống có thể nhìn thấu, gỡ lỗi trong vài giây, và tự chữa lành.

---

## 1. Kiến trúc 4 tầng

Quan sát toàn hệ thống được tổ chức thành bốn tầng xếp tầng, mỗi tầng giải quyết một câu hỏi:

```
Tầng 1: Bắt lỗi tại code (App-level)
        ↓
Tầng 2: Gom về trung tâm (Telemetry Triad)
        ↓
Tầng 3: Báo cáo tự động + AI RCA
        ↓
Tầng 4: Tự phục hồi (Self-healing)
```

| Tầng | Câu hỏi trả lời | Công nghệ chính |
|------|----------------|----------------|
| 1 – Application | "Đoạn code nào đang lỗi, do ai, tại dòng nào?" | Zap (Go), `tracing` (Rust), OpenTelemetry SDK |
| 2 – Telemetry Triad | "Hệ thống có bao nhiêu log/metric/trace, ở đâu?" | Vector, ClickHouse/Loki, VictoriaMetrics, Jaeger/Tempo |
| 3 – Alerting + AI RCA | "Lỗi này nghĩa là gì, ai xử lý, root cause là gì?" | Sentry/GlitchTip, DeepSeek-Coder, Telegram/Slack/PagerDuty |
| 4 – Self-Healing | "Hệ thống tự hồi phục được không, trong bao lâu?" | gobreaker, K3s probes, ArgoCD auto-rollback |

**Acceptance criteria (mức dịch vụ):**

| Mã | Tiêu chí | Đo lường |
|----|---------|---------|
| OBS-1 | 100% request có `trace_id` | Audit |
| OBS-2 | Log search < 100ms trên 1 tỷ dòng | ClickHouse benchmark |
| OBS-3 | Alert trigger < 30s | E2E test |
| OBS-4 | AI RCA < 3s | Benchmark |
| OBS-5 | Self-healing < 2s | Chaos test |

---

## 2. Tầng 1 — Application Level

### 2.1. Trace ID Generation & Propagation

Mọi request khi vào hệ thống được gắn một `trace_id` UUIDv7 (timestamp-sortable). ID này đi theo request xuyên suốt: HTTP header `X-Trace-ID`, NATS header, OpenTelemetry span, log record, Sentry tag, audit log.

```go
// UUIDv7 có timestamp → sort được theo thời gian
func NewTraceID() string {
    id, _ := uuid.NewV7()
    return id.String()
}
```

Middleware HTTP:
1. Lấy W3C Trace Context từ header (nếu có) bằng `otel.GetTextMapPropagator()`.
2. Nếu chưa có, sinh UUIDv7 mới.
3. Inject vào `context.Context` (logger/OTel/middleware đều đọc từ đây).
4. Start một root span với kind = `SpanKindServer`, attach attribute (method, target, status code, peer IP, user agent).
5. Response header `X-Trace-ID` trả về cho client.

Cross-service qua NATS: header `X-Trace-ID` được publisher đính kèm, consumer extract và gắn lại vào span + log.

### 2.2. Structured Logging

Schema JSON chuẩn (mọi service Go/Rust/Python đều dùng chung):

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
  "caller": {"file": "user_repository.go", "line": 142, "function": "CreateLead"},
  "message": "failed to insert lead to postgres",
  "error": {"type": "*pq.Error", "message": "pq: duplicate key...", "stack": "..."},
  "context": {"lead_id": "uuid...", "phone": "+84xxx", "source": "facebook"}
}
```

Field bắt buộc cho mọi log record: `timestamp` (ISO8601 với nanosecond), `level`, `service`, `version`, `trace_id`, `tenant_id`, `user_id`, `request_id`, `caller` (file:line), `message`. Field mở rộng: `error`, `context`, `attributes`.

Cú pháp:
- **Go:** `zap` với `zap.AddCallerSkip(1)`, encoder JSON, các helper `logger.WithContext(ctx)` tự động attach trace/tenant/user.
- **Rust:** `tracing` với `#[instrument(skip_all, fields(...))]`, format JSON.
- **Python:** `structlog` để có cùng schema.

### 2.3. Error Wrapping & PII Redaction

- **Wrap theo layer:** repository → service → handler, mỗi tầng `fmt.Errorf("Layer.Method: tenant=%s: %w", err)` để chain vẫn truy ngược được.
- **PII redaction** áp dụng trước khi log: phone, email, SSN, credit_card được thay bằng mask trước khi ghi log hoặc gửi Sentry. Lint rule + PII scanner chạy trong CI phát hiện leak.

---

## 3. Tầng 2 — Telemetry Triad (Logs / Metrics / Traces)

### 3.1. Logs Center (Vector → ClickHouse / Loki)

```
[Code trên các VPS]
       │
       ├─► (Logs) ───────► [Vector / FluentBit] ──► [ClickHouse / Grafana Loki]
```

**Vector** thu log từ mọi node (`/var/log/rinco/*.log`), parse JSON, normalize schema, đẩy sang **ClickHouse** cho analytics hoặc **Grafana Loki** cho log aggregation. ClickHouse được chọn vì query full-text trên 1 tỷ dòng dưới 100ms.

Schema ClickHouse:
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

Query pattern: tra cứu theo `trace_id`, lọc theo `service` + `level` + khoảng thời gian, count error per service per hour. TTL 90 ngày, sau đó archive S3 Glacier.

### 3.2. Metrics Center (VictoriaMetrics / Prometheus)

Mỗi service Go/Rust expose `/metrics` Prometheus. **VictoriaMetrics** (drop-in Prometheus, tiết kiệm storage 5–10x) scrape mỗi 15 giây.

Metric chuẩn toàn hệ thống (đặt prefix `rinco_`):

| Metric | Type | Labels |
|--------|------|--------|
| `rinco_http_requests_total` | Counter | service, method, path, status |
| `rinco_http_request_duration_seconds` | Histogram | service, method, path |
| `rinco_db_query_duration_seconds` | Histogram | db, query_type |
| `rinco_db_connections_active` | Gauge | db |
| `rinco_nats_publish_duration_seconds` | Histogram | topic |
| `rinco_grpc_calls_total` | Counter | service, method, status |
| `rinco_websocket_connections` | Gauge | service |
| `rinco_active_meetings` | Gauge | tenant |
| `rinco_active_chat_channels` | Gauge | tenant |
| `rinco_ai_inference_duration_seconds` | Histogram | model |
| `rinco_circuit_breaker_state` | Gauge | name (0=closed, 1=half-open, 2=open) |

Middleware Go tự động đo duration, inflight, status code cho mọi HTTP request. Recording rules + alert rules + dashboard auto-provisioning qua Grafana datasource. Cardinality label được giới hạn (drop unused labels) để tránh Prometheus OOM.

### 3.3. Traces Center (OpenTelemetry → Jaeger / Tempo)

Mọi service Go/Rust instrument bằng **OpenTelemetry SDK**, xuất OTLP sang **OTLP Collector** rồi sang **Jaeger** (hoặc Tempo). Trace ID W3C chuẩn, propagate qua HTTP, gRPC, NATS, Kafka. Span attribute: `tenant.id`, `lead.source`, `db.statement`, `http.status_code`. Span event cho mốc quan trọng (publish event, commit DB). Tail-based sampling cho trace có lỗi hoặc latency cao.

12 dashboard Grafana đã dựng sẵn:

1. **System Overview** – High-level metrics.
2. **API Gateway** – Request rate, latency, errors.
3. **Database Performance** – Query duration, connections.
4. **Service Map** – Topology với traffic giữa các service.
5. **Resource Usage** – CPU, RAM, disk, network per node.
6. **WebRTC SFU** – Active meetings, packet loss.
7. **Chat Engine** – WS connections, message throughput.
8. **AI Inference** – Model latency, GPU usage.
9. **ClickHouse Analytics** – Query rate, OLAP performance.
10. **Tenant Health** – Per-tenant SLA.
11. **Cost Analysis** – Resource cost per tenant.
12. **Error Heatmap** – Errors by service/time.

---

## 4. Tầng 3 — Realtime Alerting + AI RCA

### 4.1. Sentry / GlitchTip — Error Aggregation

Mọi exception được capture bằng **Sentry SDK** (Go, Rust, Frontend JS). Sentry tự group các error giống nhau thành "Issue", gắn tag `tenant_id`, `trace_id`, `release`, `environment`. Trước khi gửi, `beforeSend` hook strip PII (email, phone, IP). Self-host GlitchTip thay Sentry SaaS để kiểm soát data residency.

### 4.2. AI SRE — Auto Root Cause Analysis

Khi Sentry mở issue mới hoặc error rate vượt ngưỡng, webhook kích hoạt **AI SRE Worker** (Python service):

```
[Error detected]
    ↓
[Sentry groups errors into Issue]
    ↓
[Webhook → AI SRE Worker]
    ↓
[Worker receives: trace_id, error_message, stack_trace]
    ↓
1. Query ClickHouse: get all logs with trace_id
2. Query Jaeger: get trace spans
3. Read source code from Git repo (find file:line)
4. Use Code-LLM (DeepSeek-Coder) to analyze
5. Generate report → Telegram/Slack
6. (Optional) Auto-create PR with patch
```

Prompt gửi LLM: error message + stack trace + recent logs (cùng trace_id) + source code xung quanh file:line. LLM trả về JSON với `root_cause`, `why`, `suggested_fix` (code snippet), `prevention`, `confidence` (0–1).

Nếu `confidence > 0.8` và severity ≥ P1, worker tự tạo branch `hotfix/ai-{incident_id}`, commit patch, mở PR. PR được review bởi con người trước khi merge (human-in-the-loop gate).

### 4.3. Anomaly Detection & Capacity Planning

- **AI anomaly detection:** Prophet/LSTM forecast dựa trên 1 tuần lịch sử, nếu deviation > 3σ → alert `anomaly_detected`.
- **AI capacity planning:** dự đoán 30 ngày tới, đề xuất scale up CPU/storage trước khi đầy.
- **AI cost optimization**, **AI regression detection**, **AI log pattern clustering**, **AI dependency analysis**, **AI weekly report**, **AI post-mortem generator**, **AI runbook generation**, **AI alert noise reduction**, **AI service map generation**, **AI on-call suggestion**.

### 4.4. Alert Severity & Multi-Channel Routing

| Level | Phản hồi |
|-------|----------|
| **P0 / Critical** | Service down / data loss → Page on-call + SMS + Phone call (PagerDuty) |
| **P1 / High** | Feature chính hỏng → Telegram + Slack + Email |
| **P2 / Medium** | Degraded nhưng vẫn chạy → Slack channel + dashboard |
| **P3 / Low** | Warning → Dashboard only |

Kênh alert: **Alertmanager**, **Telegram bot** (gửi Markdown với link Grafana), **Slack** (rich attachment), **Discord**, **PagerDuty** (page on-call, escalation policy), **Opsgenie**, **SMS qua Twilio**, **Phone call**. Alert có grouping, deduplication, silence (bảo trì), inhibition (khi upstream alert đã cover), acknowledgment, resolution tracking.

### 4.5. On-Call Schedule

PostgreSQL lưu `oncall_schedules` (primary + secondary user, khoảng thời gian), rotation tự động. Lịch sử alert (`alert_history`) kèm `ai_rca` JSONB để họp post-mortem.

---

## 5. Tầng 4 — Self-Healing

### 5.1. Circuit Breaker Pattern

Dùng `sony/gobreaker` cho mọi external dependency (Meta CAPI, Stripe, OpenAI, SendGrid):

```go
var fbCB = gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name: "facebook-capi",
    Timeout: 60 * time.Second,
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        return counts.ConsecutiveFailures > 5 || counts.FailureRatio() > 0.5
    },
    OnStateChange: func(name string, from, to gobreaker.State) {
        if to == gobreaker.StateOpen { alert.Send("circuit_open", name) }
    },
})
```

State: **closed** → bình thường, **open** → fail fast, **half-open** → probe. Metric `rinco_circuit_breaker_state` được export, alert rule `rinco_circuit_breaker_state == 2` → P1.

### 5.2. K3s Self-Healing

Mỗi Deployment có 3 probe:

- **Liveness probe** `GET /health/live` (process alive) → 3 fail → restart pod.
- **Readiness probe** `GET /health/ready` (sẵn sàng nhận traffic, check DB/cache) → gỡ khỏi Service khi fail.
- **Startup probe** `GET /health/startup` → dành cho service khởi động chậm.

K3s tự restart pod chết, tự rebalance khi node chết, HPA scale theo CPU/RAM/custom metric.

### 5.3. Graceful Degradation

Mỗi dependency có fallback chain:

```python
def get_analytics(tenant_id, query):
    try: return clickhouse_query(query)
    except Exception as e:
        return postgres_query(fallback_query(query))   # slower but works
```

Stale-while-revalidate cho cache: khi DB chết, serve cache cũ kèm flag `stale=true`. Retry middleware với exponential backoff cho mọi call.

### 5.4. Auto-Rollback

**ArgoCD Application** với `syncPolicy.automated.selfHeal: true`: nếu manifest khác Git → sync lại. Khi deploy health check fail (error rate tăng đột biến trong 2 phút), ArgoCD rollback về revision trước. Blue-green + version pinning đảm bảo luôn có phiên bản cũ để quay lại.

### 5.5. Database Self-Healing

| DB | Cơ chế |
|----|-------|
| PostgreSQL | PgBouncer + auto-restart on connection failure |
| ScyllaDB | Auto-rebalance shards |
| ClickHouse | Keeper election tự động |
| Valkey | Sentinel hoặc Cluster mode với failover |

---

## 6. Công cụ & Ma trận tích hợp

| Tầng | Công nghệ | Tích hợp |
|------|-----------|---------|
| Code Level | Zap (Go) / `tracing` (Rust) + OpenTelemetry | Auto-instrument |
| Log Center | Vector → ClickHouse | Native client |
| Metrics | VictoriaMetrics + Grafana | Prometheus |
| Traces | OpenTelemetry → Jaeger | OTLP |
| Error Tracking | Sentry / GlitchTip | SDK |
| Alerting | Alertmanager + Telegram + PagerDuty | Webhook |
| AI RCA | DeepSeek-Coder / Llama-3 + OpenTelemetry | Custom |
| Resilience | gobreaker + K3s Probe | Built-in |

Local dev: profile `obs` trong docker-compose bao gồm ClickHouse + Vector + VictoriaMetrics + Jaeger + Grafana + Sentry trên một node.

---

## 7. Database Schema tóm tắt

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
  enabled BOOLEAN DEFAULT true
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
  ai_rca JSONB
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

## 8. API Surface (Observability)

```
GET    /metrics                                 # Prometheus
GET    /health/live
GET    /health/ready
GET    /health/startup

POST   /api/obs/v1/logs/search
GET    /api/obs/v1/logs/:trace_id
GET    /api/obs/v1/traces/:trace_id
GET    /api/obs/v1/metrics/query                # PromQL
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

## 9. Danh sách ≥ 25+ tính năng Observability

### Logging (1–25)
1. JSON structured log (schema chuẩn toàn hệ thống)
2. Log level DEBUG/INFO/WARN/ERROR/FATAL
3. Trace ID tự động (UUIDv7)
4. Caller info file:line
5. PII redaction (phone, email, SSN, credit_card)
6. Log sampling theo rate
7. Log buffering (giảm syscall)
8. Log compression (zstd)
9. Async log write
10. Log rotation (size/time-based)
11. Log archive (S3 cold storage / Glacier)
12. Full-text log search (tokenbf_v1)
13. Log filter (level, service, trace)
14. Saved searches
15. Chia sẻ log query qua link
16. Live tail log
17. Log highlight theo regex
18. Context log (extra fields)
19. Audit log riêng (không trộn)
20. Per-service retention policy
21. Log ship qua Vector / FluentBit
22. Log streaming qua WebSocket
23. Mobile-friendly log viewer
24. Log export CSV/JSON
25. Log correlation (group theo trace_id)

### Metrics (26–50)
26. `/metrics` Prometheus endpoint
27. Custom metric registration
28. Histogram với custom buckets
29. Counter, Gauge, Summary
30. HTTP request duration
31. DB query duration
32. NATS publish duration
33. Cache hit/miss ratio
34. Active connections gauge
35. Queue depth gauge
36. Memory per service
37. Goroutine count
38. Goroutine leak detection
39. Custom business metric (Lead created/min)
40. Per-tenant metric
41. Per-region aggregated metric
42. Label cardinality limit
43. Push gateway (batch metrics)
44. Metric retention config
45. Recording rules
46. Alert rules
47. Dashboard auto-provisioning
48. OpenMetrics format
49. Native histograms
50. Metric exemplars (link sang trace)

### Tracing (51–75)
51. OpenTelemetry SDK cho Go/Rust/Python
52. Auto-instrumentation HTTP/DB/gRPC
53. Manual span creation
54. Span attributes (tenant.id, lead.source)
55. Span events (publish, commit)
56. Span links (causal relationship)
57. W3C Trace Context propagation
58. Head-based sampling
59. Tail-based sampling (lấy mẫu error)
60. Span error tracking
61. OTLP export
62. Jaeger export
63. Zipkin export
64. Trace ID trong logs
65. Trace trong errors (Sentry tag)
66. Trace UI (Jaeger/Tempo)
67. Trace search
68. Trace comparison
69. Trace flamegraph
70. Trace service map
71. Trace latency breakdown
72. Trace export
73. Trace-based metrics
74. Trace exemplar
75. Trace-based alerting

### Alerting (76–95)
76. Prometheus Alertmanager
77. Multi-channel alert
78. Alert grouping
79. Alert deduplication
80. Alert silence
81. Alert inhibition
82. Alert escalation
83. On-call schedule
84. On-call rotation
85. Alert webhook
86. Telegram bot alert
87. Slack alert
88. Discord alert
89. PagerDuty alert
90. Opsgenie alert
91. SMS alert (Twilio)
92. Phone call alert
93. Severity routing (P0–P3)
94. Alert acknowledgment
95. Alert resolution tracking

### AI SRE (96–115)
96. Auto error grouping (Sentry)
97. Auto Root Cause Analysis
98. Auto Hotfix Proposal (PR)
99. Auto issue create (GitHub)
100. AI capacity planning
101. AI cost optimization
102. AI anomaly detection (Prophet/LSTM)
103. AI performance regression detection
104. AI log pattern clustering
105. AI dependency analysis
106. AI change impact analysis
107. AI weekly report
108. AI incident post-mortem
109. AI runbook generation
110. AI alert noise reduction
111. AI service map generation
112. AI on-call suggestion
113. AI runbook execution
114. AI Playbook automation
115. AI knowledge base search

### Self-Healing (116–130)
116. Circuit Breaker pattern (gobreaker)
117. K3s auto-restart
118. Health check endpoint
119. Liveness probe
120. Readiness probe
121. Startup probe
122. HPA auto-scaling
123. Cluster auto-scaling
124. Node auto-scaling
125. Auto-rollback deploy (ArgoCD)
126. Graceful shutdown
127. Connection draining
128. Resource limit (CPU/RAM requests/limits)
129. Pod disruption budget
130. Disaster recovery drill (Chaos Mesh)

---

## 10. Cost Estimation (≈ $2,420 / tháng)

| Component | Spec | USD/tháng |
|-----------|------|-----------|
| ClickHouse (3 nodes) | 8 vCPU, 32GB, 1TB NVMe | $450 |
| VictoriaMetrics (1 node) | 4 vCPU, 16GB, 500GB | $120 |
| Grafana Cloud Pro | 10K series | $290 |
| Jaeger (3 nodes) | 4 vCPU, 16GB | $240 |
| Sentry (self-hosted) | 4 vCPU, 16GB | $120 |
| Vector agents | Per pod (lightweight) | included |
| vLLM (GPU) cho AI SRE | 1× A100 | $1,200 |

---

## 11. Edge cases & Disaster Recovery

Đã cover 32 edge case trong tài liệu gốc:

- **Logging:** buffer tràn, trace ID collision, PII leak, JSON parse fail, clock skew, disk đầy, async log block.
- **Metrics:** scrape timeout, high cardinality, counter reset, histogram bucket miss, WAL corruption, recording rule conflict, multi-tenant metric leak.
- **Tracing:** trace context loss ở boundary, tail sampling fail, Jaeger indexer OOM, OTLP backpressure, span limit, W3C vs B3 mismatch.
- **AI SRE:** hallucination RCA, prompt injection từ log, LLM timeout, code-LLM suggest fix sai, Sentry rate limit, auto PR spam.
- **Self-Healing:** circuit breaker stuck open, K3s restart loop, fallback stale cache quá lâu, rollback fail, DB failover không graceful, liveness probe too strict.

DR plan:
- **ClickHouse:** `clickhouse-backup` snapshot daily lên S3, restore từ snapshot.
- **Prometheus:** WAL recovery + snapshot restart.
- **Jaeger:** Elasticsearch backend, snapshot daily.
- **Sentry:** self-host GlitchTip, `pg_dump` daily.

---

## 12. Acceptance Criteria bổ sung

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-OBS-07 | AI SRE RCA accuracy ≥ 70% | Manual review |
| AC-OBS-08 | Self-healing MTTR < 30s | Chaos test |
| AC-OBS-09 | Log PII leak = 0 | Scanner CI |
| AC-OBS-10 | Prometheus uptime ≥ 99.95% | Uptime check |
| AC-OBS-11 | Alert false positive < 5% | Audit |
| AC-OBS-12 | Cost per service < $200/month | Billing |

---

**Tóm lại:** Observability của RINCO là một hệ thống quan sát 4 tầng (Application → Telemetry Triad → Alerting + AI SRE → Self-Healing), với mục tiêu SLA rất khắt khe (log search < 100ms, alert < 30s, AI RCA < 3s, self-healing < 2s). AI SRE không chỉ tóm tắt lỗi mà còn đề xuất patch và mở PR tự động với human-in-the-loop gate. Stack công nghệ chuẩn open-source-first: Vector, ClickHouse, VictoriaMetrics, Jaeger, OpenTelemetry, gobreaker, K3s, ArgoCD — không phụ thuộc SaaS đắt tiền, có thể tự host hoàn toàn.
