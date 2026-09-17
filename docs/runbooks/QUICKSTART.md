# QUICKSTART — 5 phút onboard

> Mục tiêu: chạy được **1 frontend app + 1 backend service** local trong 5 phút.

## Bước 0 — Yêu cầu (cần cài trước, ~10 phút)

| Tool | Lệnh kiểm tra | Cài |
|------|---------------|-----|
| **Docker + Compose** | `docker --version` | [docker.com](https://docker.com) |
| **Bun** | `bun --version` | `curl -fsSL https://bun.sh/install \| bash` |
| **Go 1.23+** (chỉ cần nếu dev Go service) | `go version` | [go.dev](https://go.dev) |
| **Git** | `git --version` | (có sẵn) |

Nếu dùng Windows: dùng **WSL2** hoặc **PowerShell + Docker Desktop**.

## Bước 1 — Clone repo (30 giây)

```bash
git clone https://github.com/itdoanh/rinco.git
cd rinco
cp .env.example .env
```

## Bước 2 — Bật databases (1 phút)

```bash
docker compose -f infra/docker-compose.yml up -d
```

Lệnh này bật 9 databases + observability:

- PostgreSQL 16 (auth, tenant, crm, lead, dynamic_model, email, notification, lead_scoring, meta_capi, billing, search)
- ScyllaDB 6
- MongoDB 7
- ClickHouse 24
- Valkey 7
- MinIO
- Qdrant
- Meilisearch
- NATS JetStream
- Prometheus + Grafana + Loki + Jaeger

Verify:

```bash
docker compose -f infra/docker-compose.yml ps
# Tất cả status = Up / healthy
```

## Bước 3 — Migrations (1 phút)

```bash
make migrate
# hoặc:
./scripts/migrate.sh
```

Output mẫu:

```
[auth-service]   ✓ 12 migrations applied
[tenant-service] ✓ 8 migrations applied
[crm-service]    ✓ 15 migrations applied
[lead-service]   ✓ 10 migrations applied
...
```

## Bước 4 — Seed data (optional, 30 giây)

```bash
make seed
# Tạo 3 demo tenants + 1 super-admin
```

Tài khoản super-admin:

```
Email:    superadmin@rinco.vn
Password: Admin@123
```

## Bước 5 — Chạy 1 service + 1 frontend (1 phút)

### Terminal 1: backend service

```bash
# VD: auth-service (Go)
cd services/auth-service
go run ./cmd/server
# Listening on http://localhost:8081
```

### Terminal 2: frontend app

```bash
cd frontend/admin-portal
bun install  # lần đầu tiên, sau đó cache
bun dev
# Listening on http://localhost:3001
```

### Terminal 3: verify

```bash
# Backend health
curl http://localhost:8081/healthz
# {"status":"ok","service":"auth-service","version":"1.1.0"}

# Frontend
curl http://localhost:3001
# <html>...</html>
```

## Bước 6 — Đăng nhập & explore (30 giây)

Mở browser: **http://localhost:3001**

Đăng nhập với `superadmin@rinco.vn` / `Admin@123`.

Bạn sẽ thấy dashboard với:

- 20 services status (1 đang chạy: auth-service; 19 còn lại sẽ là red nếu chưa bật).
- 9 databases status.
- Quick links tới Tenants / Audit / Incidents.

## Bước 7 — Bật thêm service (optional)

```bash
# Mở thêm terminal
cd services/crm-service && go run ./cmd/server    # port 8083
cd services/lead-service && go run ./cmd/server   # port 8085
cd services/landing-service && go run ./cmd/server # port 8086

# Frontend khác
cd frontend/landing && bun dev        # port 3000
cd frontend/tenant-site && bun dev    # port 3002
```

Xem [SERVICES.md](../SERVICES.md) để biết port nào cho service nào.

## Common pitfalls

| Lỗi | Fix |
|-----|-----|
| `port 8081 already in use` | `lsof -i :8081` → kill process hoặc đổi port trong `.env` |
| `database connection refused` | Check `docker compose ps` — Postgres có Up không? |
| `migration failed` | `make db-reset` (chỉ dev!) rồi `make migrate` lại |
| `frontend build fail` | `rm -rf .next node_modules && bun install` |
| `permission denied` trên Linux | `sudo usermod -aG docker $USER` rồi logout/login |

## Tắt stack

```bash
# Tắt services (Ctrl+C trong terminal)
# Tắt databases
docker compose -f infra/docker-compose.yml down

# Xóa data (CẢNH BÁO: mất hết data)
docker compose -f infra/docker-compose.yml down -v
```

## Next steps

- [DEVELOPMENT.md](../DEVELOPMENT.md) — workflow chi tiết
- [ARCHITECTURE.md](../ARCHITECTURE.md) — hiểu kiến trúc
- [SERVICES.md](../SERVICES.md) — danh sách services
- [runbook/incident-service-down.md](incident-service-down.md) — nếu có service down
- [adr/0001-polyglot-persistence.md](../adr/0001-polyglot-persistence.md) — tại sao nhiều database?

## Trợ giúp

- Slack: `#engineering`
- Telegram: `@rinco_dev`
- Email: dev-support@rinco.vn