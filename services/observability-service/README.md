# Observability Service

> **Phân hệ #8 — Self-hosted Observability API** · Aggregates logs (Loki),
> traces (Jaeger/Tempo), metrics (Prometheus), audit (ClickHouse) vào 1
> unified REST surface.  Built-in dashboards (Grafana JSON), alert
> aggregation từ AlertManager webhook.

## 1. Endpoints (13)

| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/health` / `/ready` / `/metrics` | Self health |
| GET | `/v1/observability/logs` | Query logs (Loki passthrough) |
| GET | `/v1/observability/traces/:trace_id` | Query trace (Jaeger) |
| GET | `/v1/observability/metrics` | Query PromQL (`?query=...`) |
| GET | `/v1/observability/services` | List all services' health (probes `/health` trên 15 service) |
| GET | `/v1/observability/services/:service` | Health check 1 service |
| GET | `/v1/observability/alerts/active` | Active alerts |
| POST | `/v1/observability/alerts/ack/:id` | Ack 1 alert |
| GET | `/v1/observability/audit/logs` | ClickHouse audit query |
| POST | `/v1/observability/webhook/alertmanager` | AlertManager webhook receiver |
| GET | `/v1/observability/dashboards` | List auto-generated Grafana dashboards |
| GET | `/v1/observability/dashboards/:name` | Dashboard JSON (`overview`, `service-latency`, `ai-sre`, ...) |

## 2. Upstream backends

| Backend   | URL env var           | Default                  |
|-----------|-----------------------|--------------------------|
| Loki      | `LOKI_URL`            | `http://loki:3100`       |
| Jaeger    | `JAEGER_URL`          | `http://jaeger:16686`    |
| Prometheus| `PROM_URL`            | `http://prometheus:9090` |
| ClickHouse| `CLICKHOUSE_URL`      | `http://clickhouse:8123` |

Service URL overrides: `AUTH_URL`, `CRM_URL`, `DMS_URL`, ...

## 3. AlertManager Integration

Register AlertManager webhook tới `/v1/observability/webhook/alertmanager` →
service sẽ tự dedupe theo `service + alertname` và emit "firing/resolved"
events.  POST `/v1/observability/alerts/ack/:id` để ack.

## 4. Examples

### Logs
```bash
curl 'http://localhost:8089/v1/observability/logs?service=auth-service&limit=50&filter=login' | jq
```

### Trace
```bash
curl 'http://localhost:8089/v1/observability/traces/018f3a9b-7c1e-7000-...'
```

### Metric (PromQL)
```bash
curl 'http://localhost:8089/v1/observability/metrics?query=up' | jq .data.result
```

### Services
```bash
curl http://localhost:8089/v1/observability/services | jq
```

### Alert ack
```bash
curl -X POST http://localhost:8089/v1/observability/alerts/ack/auth-service%2FHighErrorRate \
  -H 'Content-Type: application/json' -d '{"user_id":"u-1"}'
```

### Grafana Dashboard JSON
```bash
curl http://localhost:8089/v1/observability/dashboards/service-latency > /tmp/dash.json
# import vào Grafana qua HTTP API hoặc file mount.
```

## 5. ENV

| Var | Default |
|-----|---------|
| `PORT` | `8089` |
| `LOKI_URL` / `JAEGER_URL` / `PROM_URL` / `CLICKHOUSE_URL` | Docker service names |
| `AUTH_URL` / `CRM_URL` / `DMS_URL` / `LANDING_URL` / `EMAIL_URL` / `NOTIF_URL` / `LEAD_SCORE_URL` / `AI_SRE_URL` / `RAG_URL` / `STT_URL` / `REC_URL` / `CHAT_URL` / `SFU_URL` / `TENANT_URL` | per-service probe URLs |

## 6. Run

```bash
cd services/observability-service
go build ./cmd && ./observability-service
```
