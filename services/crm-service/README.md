# CRM Service

CRM Service cho RINCO - quản lý khách hàng, công ty, giao dịch, hoạt động và cây tổ chức LTREE.

## Features

### Core CRM
- **Contacts**: CRUD contacts với custom fields, tags, timeline
- **Companies**: CRUD companies với thông tin doanh nghiệp
- **Deals**: CRUD deals với pipeline stages (Kanban), probability tracking
- **Activities**: CRUD activities (call, email, meeting, task, note)
- **Notes**: CRUD notes với pin feature
- **Tags**: CRUD tags với usage counting
- **Custom Fields**: Dynamic schema per tenant (text, number, select, date, etc.)

### Organizational Tree (LTREE)
- **Hierarchical User Management**: Unlimited depth tree structure
- **Move Subtree**: Atomic subtree movement với LTREE operations
- **Path Queries**: Get user path, subordinates, ancestors
- **Invite Links**: Secure invitation links với role-based access

### Reports
- **Pipeline Report**: Deal values by stage
- **Conversion Report**: Contact qualification rates
- **Leaderboard**: User performance ranking

## API Endpoints

### Companies
```
GET    /v1/companies           - List companies
POST   /v1/companies           - Create company
GET    /v1/companies/:id       - Get company
PUT    /v1/companies/:id       - Update company
DELETE /v1/companies/:id       - Delete company
```

### Contacts
```
GET    /v1/contacts            - List contacts
POST   /v1/contacts           - Create contact
GET    /v1/contacts/:id        - Get contact
PUT    /v1/contacts/:id        - Update contact
DELETE /v1/contacts/:id        - Delete contact
POST   /v1/contacts/:id/tags  - Add tag to contact
```

### Deals
```
GET    /v1/deals               - List deals
POST   /v1/deals               - Create deal
GET    /v1/deals/:id           - Get deal
PUT    /v1/deals/:id           - Update deal
DELETE /v1/deals/:id           - Delete deal
POST   /v1/deals/:id/stage     - Move deal to new stage
```

### Activities
```
GET    /v1/activities                - List activities
POST   /v1/activities               - Create activity
GET    /v1/activities/:id           - Get activity
PUT    /v1/activities/:id           - Update activity
DELETE /v1/activities/:id           - Delete activity
GET    /v1/activities/timeline/:id  - Get contact timeline
```

### Notes
```
GET    /v1/notes            - List notes
POST   /v1/notes            - Create note
GET    /v1/notes/:id        - Get note
PUT    /v1/notes/:id        - Update note
DELETE /v1/notes/:id        - Delete note
```

### Tags
```
GET    /v1/tags              - List tags
POST   /v1/tags              - Create tag
```

### Custom Fields
```
GET    /v1/custom-fields                 - List custom fields (filter by entity_type)
POST   /v1/custom-fields                 - Create custom field
```

### Tree (LTREE)
```
GET    /v1/tree/:tenant_id/users                    - List all users in tree
POST   /v1/tree/:tenant_id/users                    - Create user in tree
GET    /v1/tree/:tenant_id/users/:id                - Get user details
PUT    /v1/tree/:tenant_id/users/:id                - Update user
DELETE /v1/tree/:tenant_id/users/:id                - Delete user
POST   /v1/tree/:tenant_id/move                     - Move subtree
GET    /v1/tree/:tenant_id/path/:user_id            - Get user path
GET    /v1/tree/:tenant_id/subordinates/:user_id   - Get subordinates
GET    /v1/tree/:tenant_id/ancestors/:user_id       - Get ancestors
POST   /v1/tree/:tenant_id/invite-link             - Create invite link
GET    /v1/tree/:tenant_id/users/:id/subordinates-count - Get subordinates count
```

### Reports
```
GET    /v1/reports/pipeline    - Pipeline report
GET    /v1/reports/conversion  - Conversion report
GET    /v1/reports/leaderboard - Leaderboard
```

## Database Schema

### ER Diagram
```
┌─────────────┐       ┌──────────────┐       ┌─────────┐
│  companies   │       │   contacts   │       │  deals  │
│─────────────│       │──────────────│       │─────────│
│ id (PK)     │──┐    │ id (PK)     │──┐    │ id (PK) │
│ tenant_id   │  │    │ tenant_id   │  │    │ tenant  │
│ name        │  │    │ company_id  │  │    │contact_id│
│ industry    │  │    │ first_name  │  │    │owner_id │
│ size        │  │    │ last_name   │  │    │ name    │
│ website     │  └───►│ email       │  └───►│ value   │
│ ...         │       │ phone       │       │ stage   │
└─────────────┘       │ owner_id    │       │ ...     │
                      │ status      │       └─────────┘
                      │ custom_fields│            │
                      │ tags        │            ▼
                      └──────────────┘       ┌──────────────┐
                            │               │ deal_stage_  │
                            │               │   history    │
                            ▼               └──────────────┘
                      ┌──────────────┐
                      │  activities  │
                      │──────────────│
                      │ id (PK)      │
                      │ tenant_id    │
                      │ contact_id   │
                      │ deal_id      │
                      │ type         │
                      │ subject      │
                      │ body         │
                      │ due_at       │
                      │ status       │
                      │ priority     │
                      └──────────────┘
                            │
                            ▼
                      ┌──────────────┐
                      │    notes      │
                      │──────────────│
                      │ id (PK)      │
                      │ tenant_id    │
                      │ contact_id   │
                      │ deal_id      │
                      │ body         │
                      │ author_id    │
                      │ is_pinned    │
                      └──────────────┘

┌─────────────┐       ┌──────────────┐
│    tags     │◄──────│ contact_tags │
│─────────────│       └──────────────┘
│ id (PK)     │
│ tenant_id   │
│ name        │
│ color       │
│ usage_count │
└─────────────┘

┌─────────────────┐
│  custom_fields  │
│─────────────────│
│ id (PK)         │
│ tenant_id       │
│ entity_type     │
│ name            │
│ field_key       │
│ field_type      │
│ options         │
│ is_required     │
└─────────────────┘

┌─────────────┐     ┌──────────────┐
│   users     │     │ invite_links │
│─────────────│     │──────────────│
│ id (PK)     │◄────│parent_user_id│
│ tenant_id   │     │ target_role  │
│ email       │     │ token        │
│ full_name   │     │ max_uses     │
│ path (LTREE)│     │ expires_at   │
│ depth       │     └──────────────┘
│ parent_id   │
│ role        │
│ status      │
└─────────────┘
```

## LTREE Examples

### Query Patterns
```sql
-- Get all users in a subtree
SELECT * FROM users WHERE path <@ 'root.sales.lead';

-- Get all ancestors of a user
SELECT * FROM users WHERE path @> 'root.sales.lead.nv01';

-- Get direct children
SELECT * FROM users WHERE path ~ 'root.sales.*{1}';

-- Get path depth
SELECT nlevel(path) FROM users WHERE id = 'user-uuid';

-- Move subtree (atomic)
UPDATE users SET path = 'root.sales.manager' || subpath(path, nlevel('root.sales.lead'))
WHERE path <@ 'root.sales.lead';
```

### Path Structure
```
root (depth 0)
├── admin (depth 1) path: root.admin
│   └── manager_a (depth 2) path: root.admin.manager_a
│       ├── lead_1 (depth 3) path: root.admin.manager_a.lead_1
│       │   └── nv_01 (depth 4) path: root.admin.manager_a.lead_1.nv_01
│       └── lead_2 (depth 3) path: root.admin.manager_a.lead_2
└── manager_b (depth 2) path: root.admin.manager_b
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| CRM_HTTP_ADDR | :8082 | HTTP server address |
| CRM_DATABASE_URL | postgres://... | PostgreSQL connection |
| CRM_VALKEY_URL | localhost:6379 | Redis/Valkey connection |
| CRM_VALKEY_PASSWORD | rinco_dev_password | Redis password |
| CRM_VALKEY_DB | 0 | Redis database number |
| OTEL_EXPORTER_OTLP_ENDPOINT | - | OTLP endpoint for tracing |
| ENV | development | Environment (development/production) |

## Running

```bash
# Development
go run ./cmd/main.go

# Build
go build -o crm-service ./cmd/main.go

# Test
go test ./...

# Lint
go vet ./...
```

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Echo HTTP Server                  │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐  │
│  │ Middleware   │ │ Handlers    │ │ Connect-RPC  │  │
│  │ - TenantMW   │ │ - Companies │ │ - GetContact │  │
│  │ - AuthMW     │ │ - Contacts  │ │ - ListDeals  │  │
│  │ - MetricsMW  │ │ - Deals     │ │ - MoveSubtree│  │
│  │ - CORSMW     │ │ - Tree      │ │ - GetUserTree│  │
│  └─────────────┘ │ - Reports   │ │             │  │
│                   └─────────────┘ └─────────────┘  │
└─────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────┐
│                    pgx/v5 Pool                       │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │
│  │ Companies   │ │ Contacts    │ │   Users     │   │
│  │ Deals       │ │ Activities  │ │  (LTREE)   │   │
│  │ Notes       │ │ Tags        │ │             │   │
│  │ CustomFields│ │ InviteLinks │ │             │   │
│  └─────────────┘ └─────────────┘ └─────────────┘   │
│                                                      │
│  Row Level Security (RLS) with tenant isolation     │
└─────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────┐
│              Redis (Valkey)                          │
│  - Session cache                                     │
│  - Rate limiting                                     │
│  - Tree materialized view cache                      │
└─────────────────────────────────────────────────────┘
```
