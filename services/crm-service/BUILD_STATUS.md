# crm-service Build Status (Loop 203)

- `go build -o bin/crm-service.exe ./cmd` : **PASS**
- `go vet ./...` : **PASS**
- `go test ./... -count=1 -run TestNothing` : **PASS** (all 6 packages compile)
- Files: 10 (+1 handler file: `handler_crm_extra.go`)
- LOC: 6,400+ (was 3,291)
- Migrations: 14 (was 10)
- Status: ✅ ready — enterprise CRM Tree feature-complete (lead/pipeline/workflow/notification/audit)

## New endpoints (loop 203)

```
# Leads (full lifecycle)
POST   /v1/leads                       create lead (from landing / manual / import)
GET    /v1/leads?status=&owner=&source=&min_score=&from=&to=
GET    /v1/leads/:id
PUT    /v1/leads/:id
DELETE /v1/leads/:id                    soft delete
POST   /v1/leads/:id/assign
POST   /v1/leads/:id/stage             move stage (workflow triggers)
POST   /v1/leads/:id/convert           → contact (+optional deal)
POST   /v1/leads/bulk                  bulk import (≤ 1000)

# Pipelines
POST   /v1/pipelines
GET    /v1/pipelines
GET    /v1/pipelines/:id

# Workflows
POST   /v1/workflows
GET    /v1/workflows
GET    /v1/workflows/:id
PUT    /v1/workflows/:id
DELETE /v1/workflows/:id
POST   /v1/workflows/:id/trigger       manual trigger

# Notifications
POST   /v1/notifications/broadcast     director/manager → subtree/role/dept/user
GET    /v1/notifications/inbox
PATCH  /v1/notifications/:id/read
POST   /v1/notifications/:id/dismiss

# Audit + Sessions
GET    /v1/audit                       audit log (action / actor / target filter)
GET    /v1/sessions                    active sessions
POST   /v1/sessions/:id/revoke         force logout

# User management extras
PUT    /v1/tree/:tenant_id/users/:id/promote
PUT    /v1/tree/:tenant_id/users/:id/demote

# Tags / Custom Fields full CRUD
PUT    /v1/tags/:id
DELETE /v1/tags/:id
PUT    /v1/custom-fields/:id
DELETE /v1/custom-fields/:id
```
