# observability-service

Unified self-hosted observability aggregator.  Proxies Loki / Prometheus /
Jaeger / Tempo, stores application audit (Postgres + ClickHouse rollups) and
manages AlertManager alerts.

## Endpoints

| Method | Path                                | Backend |
|--------|------------------------------------|--------|
| GET    | `/healthz`                         | local |
| GET    | `/readyz`                          | local + pool ping |
| GET    | `/metrics`                         | Prometheus exposition |
| GET    | `/v1/logs`                         | Loki `query_range` |
| GET    | `/v1/logs/aggregate`               | Loki LogQL aggregate |
| GET    | `/v1/traces/:trace_id`             | Jaeger `/api/traces/{id}` |
| GET    | `/v1/traces`                       | Jaeger search |
| GET    | `/v1/metrics`                      | PromQL instant query |
| GET    | `/v1/metrics/range`                | PromQL range query |
| GET    | `/v1/services`                     | Prom `up{}` + alert count |
| GET    | `/v1/services/:service/health`     | Prom summary |
| GET    | `/v1/alerts/active`                | Postgres `observability.alerts` |
| GET    | `/v1/alerts`                       | Postgres history |
| POST   | `/v1/alerts/:id/ack`               | Postgres |
| GET    | `/v1/audit/logs`                   | Postgres `observability.audit_logs` |
| POST   | `/v1/webhook/alertmanager`         | AlertManager v4 receiver |
| POST   | `/internal/observability.v1.*`     | Connect-RPC adapter |

## Configuration

| Env var                          | Default                       | Purpose |
|----------------------------------|-------------------------------|---------|
| `OBSERVABILITY_HTTP_ADDR`        | `:8099`                       | HTTP listen addr |
| `OBSERVABILITY_DATABASE_URL`     | —                             | alerts + audit storage |
| `OBSERVABILITY_CLICKHOUSE_URL`   | `http://localhost:8123`       | audit rollups (HTTP gateway) |
| `OBSERVABILITY_PROMETHEUS_URL`   | `http://localhost:9090`       | metrics |
| `OBSERVABILITY_LOKI_URL`         | `http://localhost:3100`       | logs |
| `OBSERVABILITY_JAEGER_URL`       | `http://localhost:16686`      | traces |
| `OBSERVABILITY_TEMPO_URL`        | `http://localhost:3200`       | trace search fallback |
| `OBSERVABILITY_FANOUT_WEBHOOK`   | —                             | where to POST alert payloads (Slack/…) |
| `OBSERVABILITY_RATE_LIMIT`       | `600`                         | per-tenant per-route req/min |

## Data model

Postgres:

* `observability.alerts(id, fingerprint, status, severity, labels, annotations,
  service, tenant_id, title, message, fired_at, resolved_at, ack_by, ack_at)`
* `observability.alert_history(alert_id, event, payload, ts)`
* `observability.audit_logs(tenant_id, actor_user_id, actor_ip, action,
  resource_type, resource_id, payload, ts)`
* `observability.service_health_cache(service, status, error_rate, p99_latency,
  active_alerts, updated_at)`

ClickHouse (`observability` database, applied on startup when reachable):

* `app_audit_logs` (MergeTree PARTITION BY toYYYYMM(ts))
* `app_incidents`  (MergeTree PARTITION BY toYYYYMM(started_at))
* `app_metric_rollup` (MergeTree PARTITION BY toYYYYMM(ts))

## Build

```bash
go build ./...
go vet ./...
```

Multi-stage Docker:

```bash
docker build -f services/observability-service/Dockerfile -t rinco/observability-service .
```