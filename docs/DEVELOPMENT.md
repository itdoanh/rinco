# Development Guide

Hướng dẫn phát triển local cho RINCO Platform. Tài liệu này dành cho engineers onboard vào dự án.

---

## 1. Setup

### 1.1 Yêu cầu công cụ

| Tool | Phiên bản | Cài đặt |
|------|-----------|---------|
| **Go** | 1.23+ | [go.dev/dl](https://go.dev/dl/) |
| **Rust** | 1.81+ | [rustup.rs](https://rustup.rs/) |
| **Python** | 3.12+ | [python.org](https://www.python.org/downloads/) |
| **Bun** | 1.x | [bun.sh](https://bun.sh/) |
| **Node.js** | 20+ (fallback) | [nodejs.org](https://nodejs.org/) |
| **Docker + Compose** | 24+ / v2 | [docker.com](https://www.docker.com/) |
| **Make** | latest | (Linux/macOS có sẵn; Windows: dùng WSL hoặc `choco install make`) |
| **k3d / K3s** *(optional)* | latest | Cho integration tests trên K8s |
| **psql / mongosh / cqlsh** | latest | DB clients để debug |

### 1.2 Clone & cấu hình

```bash
git clone https://github.com/itdoanh/rinco.git
cd rinco
cp .env.example .env
# Sửa .env theo môi trường local của bạn (đặc biệt PASETO key, JWT secret)
```

### 1.3 Khởi động local stack

```bash
# Cài deps cho tất cả services
make install

# Bật 9 databases + observability stack (Prometheus, Grafana, Loki, Jaeger)
docker compose -f infra/docker-compose.yml up -d

# Chạy migrations cho tất cả schemas
make migrate

# Verify health
make health
```

### 1.4 Chạy services

#### Tất cả services (via script)
```bash
make dev
# script dev.sh sẽ chạy mỗi service trong 1 background process (xem logs/ dir)
```

#### Từng service riêng (recommended khi dev)
Mỗi terminal một service:

```bash
# Go
cd services/auth-service && go run ./cmd/server
cd services/tenant-service && go run ./cmd/server
# ...

# Rust
cd services/chat-engine && cargo run --release
cd services/webrtc-sfu && cargo run --release

# Python
cd services/lead-scoring && uvicorn app.main:app --reload --port 8090
cd services/rag-chatbot && uvicorn app.main:app --reload --port 8091
```

#### Frontend apps

```bash
cd frontend/landing      && bun dev   # → :3000
cd frontend/admin-portal && bun dev   # → :3001
cd frontend/tenant-site  && bun dev   # → :3002
cd frontend/meeting-ui   && bun dev   # → :3003
```

---

## 2. Quy ước (Conventions)

### 2.1 Git workflow

- **Branch naming**: `<type>/<scope>-<short-desc>`
  - `feat/auth-add-fido2`
  - `fix/crm-tree-query`
  - `chore/bump-deps`
  - `docs/architecture-update`
- **Commit messages**: [Conventional Commits](https://www.conventionalcommits.org/)
  - `feat: thêm webhook delivery cho lead-service`
  - `fix(crm): sửa off-by-one khi tính lead score`
  - `docs: cập nhật ARCHITECTURE.md`
- **PRs**: 1 PR = 1 concern. Title theo convention. Body mô tả: why, what, how to test, screenshots nếu UI.

### 2.2 Code style

| Language | Style | Tool |
|----------|-------|------|
| Go | `gofmt` + `go vet` + `golangci-lint` | `make lint` |
| Rust | `cargo fmt` + `cargo clippy` | `make lint` |
| Python | `black` + `isort` + `ruff` + `mypy` | `make lint` |
| TypeScript | `prettier` + `eslint` | `make lint` |
| SQL | `pgFormatter` | (manual) |

### 2.3 Folder structure (Go service)

```
services/<name>/
├── cmd/
│   └── server/main.go       # Entry point
├── internal/
│   ├── api/                 # HTTP handlers
│   ├── grpc/                # gRPC handlers (nếu có)
│   ├── domain/              # Business entities + interfaces
│   ├── repository/          # DB access
│   ├── service/             # Use cases
│   ├── middleware/          # Auth, tenant, tracing
│   └── config/              # Env config
├── migrations/              # SQL migrations
├── Dockerfile
├── README.md
└── go.mod
```

### 2.4 Folder structure (Rust service)

```
services/<name>/
├── src/
│   ├── api/                 # HTTP / WS handlers
│   ├── crypto/              # Encryption, signal protocol
│   ├── db/                  # Scylla / Redis / S3
│   ├── handlers/
│   ├── media/               # Codec, recording
│   ├── config.rs
│   ├── error.rs
│   ├── lib.rs
│   ├── main.rs
│   ├── nats.rs
│   └── telemetry.rs
├── tests/
├── Cargo.toml
├── Dockerfile
└── README.md
```

### 2.5 Folder structure (Python service)

```
services/<name>/
├── app/
│   ├── api/                 # FastAPI routers
│   ├── core/                # config, logging, metrics, tracing
│   ├── schemas/             # Pydantic models
│   ├── services/            # Business logic
│   └── main.py
├── tests/
├── pyproject.toml
├── Dockerfile
└── README.md
```

### 2.6 Multi-tenant isolation

- Mọi bảng có cột `tenant_id` (NOT NULL) + RLS policy trên PostgreSQL.
- Mọi query **phải** filter theo `tenant_id` (middleware auto-inject từ JWT).
- Cross-tenant queries bị cấm — chỉ super-admin mới có quyền.

### 2.7 Observability mặc định

Mọi service **phải** export:
- **Metrics** (Prometheus format, `/metrics`)
- **Logs** (structured JSON, có `trace_id` + `tenant_id`)
- **Traces** (OpenTelemetry OTLP)

Dùng `packages/go/middleware/tracing` cho Go, `opentelemetry` crate cho Rust, `opentelemetry-python` cho Python.

### 2.8 Error handling

- Trả HTTP status code chính xác (4xx cho client error, 5xx cho server error).
- Response body theo format:
  ```json
  { "error": { "code": "RESOURCE_NOT_FOUND", "message": "...", "trace_id": "..." } }
  ```
- Không leak stack traces ra client — chỉ log server-side.

### 2.9 Secrets

- **KHÔNG** commit `.env`, `*.pem`, `*.key`.
- Dev: dùng `.env.local` (gitignored).
- Prod: dùng K8s Secrets hoặc Sealed Secrets.
- Rotation: mỗi service có policy rotate secret mỗi 90 ngày.

---

## 3. Testing

### 3.1 Test pyramid

```
        ╱╲
       ╱  ╲         E2E (Playwright) — frontend/integration
      ╱────╲
     ╱      ╲       Integration tests — service + DB
    ╱────────╲
   ╱          ╲     Unit tests — logic thuần
  ╱────────────╲
```

### 3.2 Chạy tests

```bash
# Tất cả
make test

# Theo ngôn ngữ
make go-test
make rust-test
make python-test
make frontend-test

# E2E (Playwright)
make e2e
```

### 3.3 Test database

Mỗi test phải dùng DB riêng (không touch dev DB). Setup qua:
```bash
make db-test-reset   # Tạo DB `*_test`, chạy migrations, seed fixtures
```

### 3.4 Coverage

- Target: **≥ 80%** cho mỗi service.
- Coverage report HTML ở `coverage/<service>/index.html`.
- CI fail nếu coverage giảm > 2% so với main.

---

## 4. Debug

### 4.1 Logs

```bash
# Tất cả services (Docker)
make logs

# Một service cụ thể
docker compose -f infra/docker-compose.services.yml logs -f auth-service

# Structured query (Loki + Grafana)
# Truy cập http://localhost:3000 → Explore → Loki
```

### 4.2 Tracing

- Truy cập Jaeger UI: http://localhost:16686
- Mỗi request có `trace_id` được log + gửi response header `X-Trace-Id`.

### 4.3 Database debug

```bash
# PostgreSQL
make shell-postgres

# Redis / Valkey
make shell-redis

# MongoDB
docker compose -f infra/docker-compose.yml exec mongodb mongosh

# ScyllaDB
docker compose -f infra/docker-compose.yml exec scylla cqlsh

# ClickHouse
docker compose -f infra/docker-compose.yml exec clickhouse clickhouse-client
```

### 4.4 Common issues

| Issue | Solution |
|-------|----------|
| Port already in use | `lsof -i :8081` → kill process |
| DB connection refused | Check `.env` + chạy `docker compose ps` |
| Migration fail | `make db-reset` (chỉ dev!) |
| Frontend build fail | `rm -rf .next node_modules && bun install` |
| Rust build chậm | `cargo clean` rồi `cargo build` lại (lần đầu lâu) |

---

## 5. Common tasks

### 5.1 Thêm service mới

1. Tạo folder `services/<name>/` theo convention trên.
2. Copy `Dockerfile` + `README.md` template từ service khác.
3. Đăng ký trong `infra/docker-compose.services.yml`.
4. Thêm Helm chart trong `deployments/helm/rinco/templates/<name>.yaml`.
5. Update `docs/SERVICES.md`.
6. Thêm CI workflow step trong `.github/workflows/ci.yml`.

### 5.2 Thêm shared Go package

1. Tạo `packages/go/<name>/`.
2. Implement + viết test (`pkg_test.go`).
3. Import từ service qua `replace` directive trong `go.mod` của service.

### 5.3 Update database schema

1. Tạo migration file mới trong `services/<name>/migrations/` (timestamp prefix).
2. **Không** sửa file migration cũ — chỉ thêm mới.
3. Chạy `make migrate` để apply.
4. Update ERD trong `docs/10-database/`.

### 5.4 Add CI workflow

1. Tạo file `.github/workflows/<name>.yml`.
2. Follow pattern trong `ci.yml` / `cd.yml`.
3. Verify bằng cách push branch + check Actions tab.

---

## 6. Xem thêm

- [docs/ARCHITECTURE.md](ARCHITECTURE.md) — Kiến trúc tổng thể
- [docs/DEPLOYMENT.md](DEPLOYMENT.md) — Production deploy
- [docs/SERVICES.md](SERVICES.md) — Index services
- [CONTRIBUTING.md](../CONTRIBUTING.md) — Workflow đóng góp
- [docs/DEV-PLAN.md](DEV-PLAN.md) — Kế hoạch phát triển tổng
