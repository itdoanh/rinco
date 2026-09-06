---
name: rinco-coding-standards
description: Skill chi tiết về coding standards, naming conventions, file structure cho RINCO.
---

# RINCO Coding Standards

## Naming Conventions
- **Service name:** `kebab-case` (vd: `crm-core`, `landing-ingest`).
- **Database name:** `snake_case` (vd: `rinco_crm`, `rinco_chat`).
- **NATS topic:** `domain.action` (vd: `lead.created`, `deal.updated`).
- **Trace ID:** `UUIDv7`.
- **Tenant ID:** `kebab-case` slug (vd: `apex-fintech`).

## Go Standards

### Project Layout
```
services/<service-name>/
├── cmd/
│   └── main.go
├── internal/
│   ├── api/
│   ├── domain/
│   ├── repository/
│   ├── service/
│   └── config/
├── migrations/
├── Dockerfile
├── go.mod
└── README.md
```

### Linting
```bash
gofumpt -l -w .
golangci-lint run ./...
```

### Error Handling
```go
// LUÔN wrap error với context
if err != nil {
    return fmt.Errorf("LeadRepo.Create: tenant=%s: %w", tenantID, err)
}
```

### Logging
```go
import "go.uber.org/zap"
slog.Info("user created",
    "trace_id", traceID,
    "tenant_id", tenantID,
    "user_id", userID,
)
```

## Rust Standards

### Project Layout
```
services/<service-name>/
├── src/
│   ├── main.rs
│   ├── lib.rs
│   ├── api/
│   ├── domain/
│   └── infra/
├── Cargo.toml
└── Dockerfile
```

### Linting
```bash
cargo fmt
cargo clippy --all-targets -- -D warnings
```

### Async
- Dùng `tokio` runtime.
- Không block event loop.

## TypeScript Standards

### Project Layout
```
apps/<app-name>/
├── app/             # Next.js App Router
├── components/
├── lib/
├── public/
├── styles/
├── package.json
└── tsconfig.json
```

### Linting
```bash
eslint .
prettier --check .
tsc --noEmit --strict
```

## Database Conventions

### PostgreSQL
- Table name: `snake_case`, số ít (vd: `lead`, `user`, `deal`).
- Column: `snake_case`.
- Primary key: `id UUID`.
- Foreign key: `<table>_id`.
- Timestamp: `<action>_at` (vd: `created_at`, `updated_at`).
- Soft delete: `deleted_at TIMESTAMPTZ`.

### ScyllaDB / Cassandra
- Table name: `snake_case`.
- Partition key prefix: `tenant_id` cho mọi table.
- Clustering order: thời gian DESC cho log.

### ClickHouse
- Database: `rinco_analytics`.
- Engine: MergeTree hoặc AggregatingMergeTree.

## Git Workflow

### Branches
- `main` – code production-ready.
- `develop` – integration branch.
- `feature/<scope>-<desc>` – feature mới.
- `hotfix/<scope>-<desc>` – bug fix urgent.
- `release/<version>` – chuẩn bị release.

### Commit Message
```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`.

### Ví dụ
```
feat(crm-core): add lead scoring API endpoint

- POST /api/crm/v1/leads/:id/score
- Integrate with ai-scoring service via NATS
- Cache score in Valkey for 1 hour

Refs: docs/11-ai-integration/README.md#3-predictive-crm
```

## Testing Standards

### Unit Test Coverage Target
- Go: ≥ 80%.
- Rust: ≥ 80%.
- TypeScript: ≥ 70%.

### Test Pyramid
- Unit tests (70%).
- Integration tests (20%).
- E2E tests (10%).

### Tools
- Go: `testify`, `gomock`, `dockertest`.
- Rust: built-in `cargo test`, `mockall`.
- TS: `vitest`, `@testing-library/react`.

## API Standards

### REST Conventions
- Resource-based URL: `/api/<version>/<resource>`.
- HTTP methods: GET (read), POST (create), PUT (replace), PATCH (partial), DELETE.
- Status codes: 200, 201, 204, 400, 401, 403, 404, 409, 422, 500.
- Pagination: `?page=1&limit=20` hoặc cursor-based.
- Filtering: `?status=active&owner_id=uuid`.
- Sorting: `?sort=created_at:desc`.

### Connect-RPC Conventions
- Service name: PascalCase.
- Method name: PascalCase.
- Field name: snake_case (proto).

## Security Standards

### Authentication
- PASETO v4 cho tokens.
- FIDO2 / YubiKey cho admin.
- Argon2id cho password hashing.

### Authorization
- RBAC mặc định.
- Row-Level Security ở database.
- Tenant context middleware LUÔN được set.

### Secrets
- KHÔNG BAO GIỜ commit secrets.
- Dùng Vault hoặc Kubernetes Secrets.
- Rotate mỗi 90 ngày.

## Documentation Standards

### Code Comments
- Public function phải có comment.
- Complex logic phải giải thích "tại sao", không phải "là gì".
- TODO phải có ticket number.

### README mỗi service
- Overview.
- API endpoints.
- Environment variables.
- Local dev setup.
- Deployment notes.

## Performance Standards

### Latency Targets
- API: p99 < 50ms.
- DB query: p99 < 20ms.
- Realtime: end-to-end < 100ms.

### Throughput Targets
- API Gateway: 100K req/s per node.
- Chat Gateway: 100K WS connections.
- DB write: 100K/s per shard.