# Lead Service

Lead Service cho RINCO - quản lý lead capture, scoring, assignment, conversion với NATS event streaming.

## Features

### Lead Management
- **CRUD Leads**: Full create, read, update, delete với comprehensive fields
- **CSV Import/Export**: Bulk import từ CSV, export to CSV
- **Status Tracking**: new, contacted, qualified, proposal, won, lost, archived
- **UTM Tracking**: utm_source, utm_medium, utm_campaign, fbclid, gclid tracking
- **Custom Fields**: JSONB-based dynamic schema per tenant
- **Tags**: Array-based tag system

### Lead Operations
- **Assign Lead**: Assign to specific user hoặc tree subtree
- **Update Status**: Move between pipeline stages với history tracking
- **Lead Scoring**: Local scoring (basic) + async scoring service integration
- **Source Filter**: Filter leads by UTM source
- **Live Stream**: SSE endpoint cho real-time lead feed
- **Stats**: Aggregate statistics by stage, source, owner

### Lead Notes & Activities
- **Notes**: CRUD notes per lead với pin functionality
- **Timeline**: Activity history (calls, emails, meetings, status changes)
- **Auto Activity Logging**: Tự động log assignments, status changes, score updates

### Pipelines & Stages
- **CRUD Pipelines**: Multi-pipeline support per tenant
- **Default Pipeline**: Auto-set default flag
- **CRUD Stages**: Pipeline stages với probability, color, won/lost flags

### Integration
- **NATS Events**: Subscribe/publish lead events
  - `lead.created` (subscribe from landing-service)
  - `lead.scored` (from lead-scoring worker)
  - `lead.assigned` (echo)
  - `lead.converted` (publish)
  - `lead.updated` (publish)
  - `user.created` (subscribe for auto-assignment)
- **Connect-RPC**: Standard RPC handlers

## API Endpoints

### Leads
```
GET    /v1/leads                     - List leads
POST   /v1/leads                     - Create lead
GET    /v1/leads/:id                 - Get lead
PUT    /v1/leads/:id                 - Update lead
DELETE /v1/leads/:id                 - Delete lead
POST   /v1/leads/import              - CSV bulk import
GET    /v1/leads/export              - Export CSV
POST   /v1/leads/:id/assign          - Assign lead to user
POST   /v1/leads/:id/status          - Update status
POST   /v1/leads/:id/score           - Compute score
POST   /v1/leads/:id/convert         - Convert to contact + deal
GET    /v1/leads/:id/notes           - List notes
POST   /v1/leads/:id/notes           - Create note
GET    /v1/leads/:id/timeline        - Get timeline
GET    /v1/leads/by-source/:source   - Filter by UTM source
GET    /v1/leads/stream              - SSE live feed
GET    /v1/leads/stats               - Aggregate statistics
```

### Lead Sources (UTM Templates)
```
GET    /v1/lead-sources              - List UTM source templates
POST   /v1/lead-sources              - Create new source
```

### Pipelines
```
GET    /v1/pipelines                 - List pipelines
POST   /v1/pipelines                 - Create pipeline
```

### Stages
```
GET    /v1/stages                    - List pipeline stages
POST   /v1/stages                    - Create stage
```

## Database Schema

### Entity Relationship
```
┌────────────────┐         ┌──────────────────┐         ┌────────────────┐
│  lead_sources   │         │      leads       │         │   pipelines    │
│────────────────│         │──────────────────│         │────────────────│
│ id (PK)         │◄────────│ source_id (FK)   │         │ id (PK)        │
│ tenant_id       │         │ tenant_id        │         │ tenant_id      │
│ name            │         │ owner_user_id    │         │ name           │
│ utm_source      │         │ pipeline_id (FK) │────────►│ is_default     │
│ utm_medium      │         │ stage_id (FK)    │         │ color          │
│ utm_campaign    │         │ full_name        │         └────────┬───────┘
│ is_active       │         │ email            │                  │
└────────────────┘         │ phone            │                  ▼
                            │ company_name     │         ┌──────────────────┐
                            │ status           │         │ pipeline_stages  │
                            │ score            │         │──────────────────│
                            │ score_tier       │         │ id (PK)          │
                            │ utm (JSONB)      │         │ pipeline_id (FK) │
                            │ custom_fields    │         │ name             │
                            │ tags[]           │         │ display_order    │
                            │ next_followup_at │         │ probability      │
                            │ converted_at     │         │ is_won/is_lost   │
                            └──────────────────┘         └──────────────────┘
                                     │
                                     ▼
                            ┌──────────────────┐
                            │   lead_notes     │
                            │──────────────────│
                            │ id (PK)          │
                            │ lead_id (FK)     │
                            │ author_id        │
                            │ body             │
                            │ is_pinned        │
                            └──────────────────┘
                                     │
                                     ▼
                            ┌──────────────────┐
                            │ lead_activities  │
                            │──────────────────│
                            │ id (PK)          │
                            │ lead_id (FK)     │
                            │ type             │
                            │ actor_id         │
                            │ description      │
                            │ payload (JSONB)  │
                            └──────────────────┘

                            ┌──────────────────┐
                            │lead_assignments  │
                            │──────────────────│
                            │ id (PK)          │
                            │ lead_id (FK)     │
                            │ from_user_id     │
                            │ to_user_id       │
                            │ reason           │
                            └──────────────────┘

                            ┌──────────────────┐
                            │lead_stage_history│
                            │──────────────────│
                            │ id (PK)          │
                            │ lead_id (FK)     │
                            │ from_stage       │
                            │ to_stage         │
                            │ changed_by       │
                            └──────────────────┘
```

## NATS Event Flow

```
[landing-service]
     │
     ▼ (publish)
[lead.created] ───────────────────┐
                                 ▼
                          [lead-service]
                                 │
                  ┌──────────────┼──────────────┐
                  ▼              ▼              ▼
         [DB Insert]   [NATS publish]   [Activity log]
                              │
                              ▼ (publish to other services)
                          [lead.created echo]
                              │
                              ▼
                      [analytics-collector]
```

### Subscribe Subjects
- `lead.created` (from landing-service)
- `lead.scored` (from lead-scoring worker)
- `lead.assigned` (echo)
- `user.created` (for auto-assign logic)

### Publish Subjects
- `lead.created` (echo)
- `lead.assigned` (echo)
- `lead.converted`
- `lead.updated`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| LEAD_HTTP_ADDR | :8083 | HTTP server address |
| LEAD_DATABASE_URL | postgres://... | PostgreSQL connection |
| LEAD_VALKEY_URL | localhost:6379 | Redis/Valkey connection |
| LEAD_VALKEY_PASSWORD | rinco_dev_password | Redis password |
| LEAD_VALKEY_DB | 0 | Redis database number |
| LEAD_NATS_URL | - | NATS server URL (optional) |
| LEAD_SCORING_URL | - | Lead scoring service URL |
| LEAD_CAPI_WORKER_URL | - | Facebook CAPI worker URL |
| OTEL_EXPORTER_OTLP_ENDPOINT | - | OTLP endpoint for tracing |
| ENV | development | Environment (development/production) |

## Running

```bash
# Development
go run ./cmd/main.go

# Build
go build -o lead-service ./cmd/main.go

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
│  │ - TenantMW   │ │ - Leads     │ │ - CreateLead │  │
│  │ - CORS       │ │ - Sources   │ │ - AssignLead │  │
│  │ - Logging    │ │ - Pipelines │ │ - ConvertLead│  │
│  │ - Metrics    │ │ - Stats     │ │ - StreamLeads│  │
│  └─────────────┘ └─────────────┘ └─────────────┘  │
└─────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────┐
│              pgx/v5 + Redis + NATS                   │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐  │
│  │  Leads DB    │ │ Redis Cache │ │   NATS      │  │
│  │  Sources     │ │ - Sessions  │ │ - Subscribe │  │
│  │  Pipelines   │ │ - RateLimit │ │ - Publish   │  │
│  │  Activities  │ │             │ │             │  │
│  └─────────────┘ └─────────────┘ └─────────────┘  │
│                                                      │
│  Row Level Security (RLS) with tenant isolation     │
└─────────────────────────────────────────────────────┘
```

## Integration với AI Scoring Service

Lead service supports async AI scoring:
1. POST /v1/leads/:id/score triggers local fallback score computation
2. Asynchronously publishes to NATS subject `lead.score.requested`
3. External `lead-scoring` worker consumes event, computes advanced AI score
4. Worker publishes back via `lead.scored` event
5. Lead service consumes `lead.scored` to update lead record
