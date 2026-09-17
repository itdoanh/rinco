# OpenAPI Specifications

OpenAPI 3.1 specs cho RINCO Platform services. Mỗi spec tương ứng 1 service.

> **Mục đích**:
> - Generate client SDK (TypeScript, Go, Python).
> - Generate server stubs.
> - Reference contract cho frontend ↔ backend.
> - Triggers CI check (Breaking change detection).

## Cấu trúc mỗi spec

```yaml
openapi: 3.1.0
info:
  title: <service> API
  version: <semver>
  description: ...
servers:
  - url: http://localhost:<port>
  - url: https://api.rinco.vn/v1
tags:
  - name: <group>
paths:
  /v1/...:
    get / post / patch / delete:
      ...
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: PASETO
  schemas:
    <SchemaName>: ...
```

## Danh sách specs

| Service | File | Port | Status |
|---------|------|------|--------|
| **auth-service** | [auth-service.yaml](auth-service.yaml) | 8081 | ✅ Complete |
| **tenant-service** | [tenant-service.yaml](tenant-service.yaml) | 8082 | ✅ Complete |
| **crm-service** | [crm-service.yaml](crm-service.yaml) | 8083 | ✅ Complete |
| **lead-service** | [lead-service.yaml](lead-service.yaml) | 8085 | ✅ Complete |

### Planned (this PR covers 4 specs; remaining 16 in subsequent loops)

| Service | File | Port | Note |
|---------|------|------|------|
| dynamic-model-service | dynamic-model-service.yaml | 8084 | |
| landing-service | landing-service.yaml | 8086 | |
| email-service | email-service.yaml | 8087 | |
| notification-service | notification-service.yaml | 8088 | |
| ai-sre | ai-sre.yaml | 8090 | |
| lead-scoring | lead-scoring.yaml | 8091 | |
| rag-chatbot | rag-chatbot.yaml | 8092 | |
| recording-service | recording-service.yaml | 8093 | |
| stt-service | stt-service.yaml | 8094 | |
| billing-service | billing-service.yaml | 8095 | |
| observability-service | observability-service.yaml | 8096 | |
| search-service | search-service.yaml | 8097 | |
| meta-capi-service | meta-capi-service.yaml | 8098 | |
| analytics-service | analytics-service.yaml | 8099 | |
| chat-engine | chat-engine.yaml | 8101 | (WS-heavy, partial) |
| webrtc-sfu | webrtc-sfu.yaml | 8102 | (WS-heavy, partial) |

## Conventions

### Error response

Mọi error response tuân theo format:

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "User abc-123 not found",
    "trace_id": "01HXYZ..."
  }
}
```

### Pagination

Mọi list endpoint dùng **cursor-based** pagination:

```
GET /v1/leads?limit=50&cursor=eyJpZCI6IjAxSEFZWi4uLiJ9

Response:
{
  "data": [...],
  "next_cursor": "eyJpZCI6IjAxSEFZWi4uLiJ9" | null
}
```

- `limit`: default 50, max 200.
- Cursor: opaque base64-encoded JSON.

### Authentication

Mọi endpoint (trừ public) yêu cầu `Authorization: Bearer <PASETO>`.

WebSocket endpoints (chat-engine, webrtc-sfu) nhận PASETO qua:
- `Sec-WebSocket-Protocol: bearer.<token>` (subprotocol), hoặc
- `?token=<token>` query param (cho browser compatibility).

### Idempotency

`POST /v1/leads` và các mutation POST hỗ trợ `Idempotency-Key` header:

```
POST /v1/leads
Idempotency-Key: lead-2026-09-18-abc123
```

Server lưu key 24h → duplicate request trả về cùng response.

### Rate limits

| Service | Default limit |
|---------|---------------|
| auth-service | 100 req/min per IP |
| crm-service | 1000 req/min per tenant |
| lead-service | 500 req/min per tenant |
| landing-service | 100 req/sec per IP (public) |
| chat-engine | unlimited (WS) |
| webrtc-sfu | 10 concurrent meetings per user |

429 response kèm `Retry-After` header.

### Versioning

- Mọi endpoint bắt đầu `/v1/`.
- Backward-compatible changes (thêm field, thêm endpoint): bump PATCH.
- Breaking changes (xóa field, đổi type): bump MAJOR + version mới (`/v2/`).

## Tools

### Validate

```bash
# Validate 1 file
npx @redocly/cli lint docs/api/auth-service.yaml

# Validate all
for f in docs/api/*.yaml; do
  npx @redocly/cli lint "$f"
done
```

### Generate TypeScript client

```bash
npx openapi-typescript-codegen \
  --input docs/api/auth-service.yaml \
  --output frontend/packages/api-client/src/auth \
  --client axios
```

### Generate server stub (Go)

```bash
oapi-codegen -package auth \
  -generate types,gin \
  docs/api/auth-service.yaml > services/auth-service/internal/api/openapi.gen.go
```

### ReDoc UI

```bash
npx @redocly/cli preview-docs docs/api/auth-service.yaml
# Mở http://localhost:8080 xem docs
```

## CI Integration

GitHub Actions CI chạy:

```yaml
- name: Validate OpenAPI specs
  run: |
    for f in docs/api/*.yaml; do
      npx @redocly/cli lint "$f" --skip-rule operation-operationId
    done

- name: Breaking change detection
  run: |
    npx @redocly/cli diff docs/api/auth-service.yaml \
      origin/main --docs/api/auth-service.yaml \
      --skip-rule operation-operationId
```

Breaking change → block merge vào `main`.

## References

- [OpenAPI 3.1 spec](https://spec.openapis.org/oas/v3.1.0)
- [Redocly CLI](https://redocly.com/docs/cli/)
- [ARCHITECTURE.md §2](../ARCHITECTURE.md#2-service-mesh)
- [SERVICES.md](../SERVICES.md)