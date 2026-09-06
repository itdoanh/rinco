# Tenant Service

Service quản lý tenants (doanh nghiệp khách hàng) cho nền tảng multi-tenant RINCO.

## Tính năng

- **Tenant CRUD**: Tạo, đọc, cập nhật, xóa tenants
- **Tenant Sites**: Quản lý domain/subdomain cho mỗi tenant
- **Tier Management**: 3 tiers (starter, pro, enterprise)
- **Status Management**: active, suspended, trial, expired
- **Quota Tracking**: max_users, max_leads_per_month
- **Settings & Metadata**: JSONB-based configuration
- **API Key Authentication**: Cho super admin operations
- **Slug Generation**: Auto-generate URL-friendly slugs

## Công nghệ

- **Language**: Go 1.23+
- **Framework**: Echo v4
- **Database**: PostgreSQL với Row-Level Security
- **Multi-tenancy**: RLS policies + SET LOCAL app.current_tenant_id

## API Endpoints

```
POST   /v1/tenants                       - Tạo tenant mới (admin only)
GET    /v1/tenants                       - List tenants (admin only)
GET    /v1/tenants/:id                   - Lấy thông tin tenant
GET    /v1/tenants/slug/:slug            - Lấy tenant theo slug
PATCH  /v1/tenants/:id                   - Cập nhật tenant
POST   /v1/tenants/:id/suspend           - Tạm dừng tenant
POST   /v1/tenants/:id/activate          - Kích hoạt lại tenant
DELETE /v1/tenants/:id                   - Xóa tenant (soft delete)

POST   /v1/tenants/:id/sites             - Tạo site cho tenant
GET    /v1/tenants/:id/sites             - List sites của tenant
GET    /v1/tenants/sites/:site_id        - Lấy thông tin site
DELETE /v1/tenants/sites/:site_id        - Xóa site
```

## Environment Variables

```bash
TENANT_SERVICE_PORT=8082
DATABASE_URL=postgres://postgres:postgres@localhost:5432/rinco?sslmode=disable
SUPER_ADMIN_API_KEY=your-secret-api-key
```

## Tenant Tiers

| Tier | Users | Leads/month | Features |
|------|-------|-------------|----------|
| Starter | 10 | 1,000 | Basic CRM |
| Pro | 100 | 50,000 | + AI, Workflow |
| Enterprise | Unlimited | Unlimited | + Custom, SLA |

## Tenant Lifecycle

```
trial → active → (suspended) → active
                 ↓
                 expired → (renew) → active
                 ↓
                 deleted (soft delete)
```

## Development

```bash
go build -o bin/tenant-service ./cmd/main.go
./bin/tenant-service
```
