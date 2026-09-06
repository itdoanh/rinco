# CRM Service

Service quản lý CRM core: phân cấp nhân sự (LTREE), leads và dynamic schemas.

## Tính năng

- **User Hierarchy với LTREE**: PostgreSQL ltree extension cho cây phân cấp
- **Department Management**: Quản lý phòng ban
- **Lead Management**: CRUD leads với custom fields
- **Lead Assignment**: Gán lead cho user theo hierarchy
- **Lead Scoring**: Lưu trữ điểm AI
- **Dynamic Schema**: JSON Schema generation cho tenant-specific fields
- **pLTV Calculation**: Predicted lifetime value
- **Multi-tenancy**: RLS enforced

## Công nghệ

- **Language**: Go 1.23+
- **Framework**: Echo v4
- **Database**: PostgreSQL 16 với ltree, pgcrypto extensions
- **Hierarchy**: LTREE cho tree operations (subtree, ancestors, descendants)
- **Dynamic Model**: JSON Schema + validation

## API Endpoints

### User Hierarchy
```
POST   /v1/crm/hierarchy/users                    - Tạo user trong hierarchy
GET    /v1/crm/hierarchy/users/:user_id           - Lấy thông tin
GET    /v1/crm/hierarchy/users/:user_id/subtree   - Lấy subtree (LTREE)
PATCH  /v1/crm/hierarchy/users/:user_id/promote   - Thăng chức
PATCH  /v1/crm/hierarchy/users/:user_id/demote    - Hạ chức
PATCH  /v1/crm/hierarchy/users/:user_id/move      - Chuyển sang manager khác
DELETE /v1/crm/hierarchy/users/:user_id           - Xóa khỏi hierarchy
```

### Departments
```
POST   /v1/crm/departments                        - Tạo phòng ban
GET    /v1/crm/departments                        - List departments
GET    /v1/crm/departments/:id                    - Chi tiết
PATCH  /v1/crm/departments/:id                    - Cập nhật
DELETE /v1/crm/departments/:id                    - Xóa
```

### Leads
```
POST   /v1/crm/leads                              - Tạo lead
GET    /v1/crm/leads                              - List leads (filterable)
GET    /v1/crm/leads/:id                          - Chi tiết lead
PATCH  /v1/crm/leads/:id                          - Cập nhật
POST   /v1/crm/leads/:id/assign                   - Gán cho user
POST   /v1/crm/leads/:id/score                    - Cập nhật điểm AI
POST   /v1/crm/leads/:id/convert                  - Chuyển đổi thành customer
DELETE /v1/crm/leads/:id                          - Xóa (soft)
```

## LTREE Path Examples

```
CEO
├── CTO (path: cto)
│   ├── Backend Lead (path: cto.backend_lead)
│   │   ├── Backend Dev 1 (path: cto.backend_lead.dev1)
│   │   └── Backend Dev 2 (path: cto.backend_lead.dev2)
│   └── Frontend Lead (path: cto.frontend_lead)
└── CFO (path: cfo)
```

Queries:
- `WHERE path <@ 'cto'` - Get all descendants of CTO
- `WHERE path @> 'cto.backend_lead'` - Get all ancestors of Backend Lead
- `WHERE nlevel(path) = 2` - Get all direct reports of CEO

## Environment Variables

```bash
CRM_SERVICE_PORT=8083
DATABASE_URL=postgres://postgres:postgres@localhost:5432/rinco?sslmode=disable
LEAD_SCORING_SERVICE_URL=http://lead-scoring:8084
```

## Development

```bash
go build -o bin/crm-service ./cmd/main.go
./bin/crm-service
```
