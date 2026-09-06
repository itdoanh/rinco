# Observability Service

Centralized health check và observability aggregation service.

## Tính năng

- **Health Aggregation**: Check health của tất cả services
- **Service Registry**: Biết được services nào đang chạy
- **Metrics Aggregation**: Aggregate metrics từ nhiều services
- **Status Dashboard**: Single endpoint cho overall status
- **Dependency Tracking**: Map service dependencies
- **Custom Checks**: Health checks với custom logic
- **SLA Tracking**: Uptime %, response time
- **Incident Correlation**: Liên kết related incidents

## Công nghệ

- **Language**: Go 1.23+
- **Framework**: Echo v4
- **HTTP Client**: net/http với timeouts
- **Caching**: In-memory với TTL

## API Endpoints

```
GET    /v1/observability/health/all         - Health của tất cả services
GET    /v1/observability/health/:service    - Health của service cụ thể
GET    /v1/observability/services           - List registered services
POST   /v1/observability/services/register  - Register service
DELETE /v1/observability/services/:id       - Deregister
GET    /v1/observability/metrics            - Aggregated metrics
GET    /v1/observability/dependencies       - Service dependency graph
GET    /v1/observability/sla                - SLA report
GET    /health                              - Service self-health
```

## Health Check

```json
GET /v1/observability/health/all

{
  "status": "healthy",  // healthy, degraded, unhealthy
  "timestamp": "2026-09-07T10:30:00Z",
  "services": {
    "auth-service": {
      "status": "healthy",
      "response_time_ms": 12,
      "last_checked": "2026-09-07T10:30:00Z",
      "version": "0.1.0",
      "uptime_seconds": 86400
    },
    "tenant-service": {
      "status": "healthy",
      "response_time_ms": 8,
      ...
    },
    "postgresql": {
      "status": "healthy",
      "response_time_ms": 3,
      "connections": 45
    },
    "scylladb": {
      "status": "healthy",
      "response_time_ms": 5
    }
  },
  "summary": {
    "total_services": 16,
    "healthy": 16,
    "degraded": 0,
    "unhealthy": 0
  }
}
```

## Service Registration

```json
POST /v1/observability/services/register
{
  "service_name": "auth-service",
  "version": "0.1.0",
  "health_endpoint": "http://auth-service:8081/health",
  "metrics_endpoint": "http://auth-service:8081/metrics",
  "dependencies": ["postgresql", "redis"],
  "tags": ["core", "auth"]
}
```

## Dependency Graph

```json
GET /v1/observability/dependencies

{
  "nodes": [
    {"id": "auth-service", "type": "service"},
    {"id": "crm-service", "type": "service"},
    {"id": "postgresql", "type": "database"},
    {"id": "scylladb", "type": "database"}
  ],
  "edges": [
    {"from": "auth-service", "to": "postgresql"},
    {"from": "crm-service", "to": "postgresql"},
    {"from": "chat-engine", "to": "scylladb"}
  ]
}
```

## SLA Report

```json
GET /v1/observability/sla?period=30d

{
  "period": "30d",
  "services": {
    "auth-service": {
      "uptime_percent": 99.99,
      "avg_response_time_ms": 15,
      "p99_response_time_ms": 45,
      "total_requests": 5000000,
      "error_rate": 0.01
    }
  },
  "overall_uptime": 99.95
}
```

## Environment Variables

```bash
OBSERVABILITY_SERVICE_PORT=8092
HEALTH_CHECK_INTERVAL=30     # seconds
HEALTH_CHECK_TIMEOUT=5       # seconds
CACHE_TTL=10                 # seconds
SERVICES_CONFIG=/config/services.json
```

## Development

```bash
go build -o bin/observability-service ./cmd/main.go
./bin/observability-service
```

## Architecture

```
┌─────────────────┐
│ Observability   │
│ Service         │
│                 │      ┌──────────┐
│  - Aggregator  │ ←──→ │ Service A │
│  - Cache        │      └──────────┘
│  - Registry     │      ┌──────────┐
│                 │ ←──→ │ Service B │
└─────────────────┘      └──────────┘
       │                  ┌──────────┐
       └─────────────────→│ Service C│
                          └──────────┘
```
