# Summary – 04 Dynamic Model Engine (Meta-Schema)

> Source: `docs/04-dynamic-model/README.md` (~180 KB, 5305 lines)
> Subsystem: Meta-Schema Engine — lets each tenant define its own CRM data model, workflow, validations, and forms at runtime, without code changes or restarts.

## 1. Overview & Goals

### 1.1 Objectives

| ID | Objective | Measurement |
|----|-----------|-------------|
| DM-1 | Create new entity with no service restart | Hot reload < 5 s |
| DM-2 | Per-tenant schema isolation | Multi-tenant JSONB |
| DM-3 | Runtime validation < 5 ms | CEL engine |
| DM-4 | Workflow automation | Rule engine |
| DM-5 | Optional code-gen | Auto Ent schemas |

### 1.2 Principles

- Schema is data — stored in PostgreSQL, never hardcoded.
- Backward compatible — adding fields does not break existing data.
- Strict validation — every record passes schema before write.
- Versioned — every schema has a version with automatic migration.
- Type-safe first — generate Go/TS code when possible; fall back to JSONB + runtime.

---

## 2. Meta-Schema Architecture (4 Tiers)

```
Tier 1: Meta-Schema Definition
  - entity types (Lead, Deal, Customer, …)
  - fields (text, number, enum, ref, …)
  - workflows (state machines)
  - validation rules
        ↓
Tier 2: Schema Storage (PostgreSQL)
  - tenants.meta_schema (JSON Schema)
  - tenants.workflow_defs
  - tenants.field_defs
        ↓
Tier 3: Runtime Engine
  - JSON Schema validator (gojsonschema)
  - CEL evaluator
  - Workflow runner
        ↓
Tier 4: Data Layer
  - PostgreSQL: relational + JSONB
  - MongoDB: optional for fully dynamic records
```

### 2.1 Hybrid Storage

- **Hardcoded columns:** for common fields (`name`, `email`, `phone`, `owner_id`, `tenant_id`, `status`, `created_at`).
- **JSONB `data` column:** for custom fields.
- **MongoDB option:** when schema drift is extreme (each record different structure).

---

## 3. Schema Definition Language

### 3.1 JSON Schema Draft 2020-12 (example)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://rinco.app/schemas/apex/lead.json",
  "title": "Lead",
  "type": "object",
  "properties": {
    "name":  { "type": "string", "minLength": 1, "maxLength": 255 },
    "phone": { "type": "string", "pattern": "^(\\+84|0)\\d{9,10}$" },
    "email": { "type": "string", "format": "email" },
    "source": { "type": "string",
      "enum": ["facebook", "tiktok", "google", "direct", "referral", "organic"] },
    "score": { "type": "integer", "minimum": 0, "maximum": 100 },
    "tags":  { "type": "array", "items": { "type": "string" } },
    "custom_fields": {
      "type": "object",
      "properties": {
        "budget": { "type": "number", "minimum": 0 },
        "interest": { "type": "string", "enum": ["buy", "rent", "invest"] }
      }
    }
  },
  "required": ["name", "phone", "source"],
  "additionalProperties": false
}
```

### 3.2 Field Types (Core)

`string`, `number`, `boolean`, `date`, `datetime`, `time`, `enum`, `array`, `object`, `reference`, `file`, `image`, `video`, `geo`, `phone`, `email`, `url`, `color`, `json`, `formula`.

### 3.3 Extended Field Types (33 total)

`string`, `text`, `rich_text` (Tiptap), `markdown` (CodeMirror), `number`, `integer`, `currency`, `percent`, `boolean`, `date`, `datetime`, `time`, `enum`, `multi_select`, `array`, `object`, `reference`, `file`, `files`, `image`, `images`, `video`, `geo`, `phone`, `email`, `url`, `color`, `json`, `formula`, `rollup`, `autonumber`, `barcode`, `signature`, `qrcode`.

### 3.4 Validation Primitives

- Required fields.
- Unique fields (e.g. email unique per tenant).
- Regex pattern.
- Range (min / max).
- Enum.
- Custom CEL expressions: `"score >= 70 || status == 'verified'"`.
- `oneOf` / `anyOf` / `allOf` for conditional schemas.
- `$rinco:reference` constraint: target entity, FK field, display field, filter, on-delete behaviour.

### 3.5 Formula Field DSL

```
Operators:    + - * / ^ ( ) &
Functions:    IF, AND, OR, NOT, IS_EMPTY, CONCAT, UPPER, LOWER, TRIM, LEN,
              ROUND, CEIL, FLOOR, MIN, MAX, AVG, TODAY, NOW, DATEADD,
              COUNTIF, SUMIF, LOOKUP
Refs:         {field_code}
Example:      total = {quantity} * {unit_price}
              days_since_contact = DATEDIFF(NOW(), {last_contacted_at}, "day")
```

### 3.6 Rollup Field

```yaml
rollup_field:
  code: total_deal_value
  target_entity: deal
  relation_field: lead_id
  target_field: value
  aggregation: sum        # sum | count | avg | min | max
  filter: "status = 'won'"
  refresh: realtime       # realtime | hourly | daily
```

---

## 4. Entity Builder

### 4.1 Entity Structure

```json
{
  "entity_code": "lead",
  "display_name": "Khách hàng tiềm năng",
  "icon": "lucide:user-plus",
  "color": "#10b981",
  "is_system": false,
  "fields": [
    {
      "code": "name", "label": "Họ tên", "type": "string",
      "required": true,
      "validation": { "minLength": 1, "maxLength": 255 },
      "ui": { "order": 1, "width": "full", "show_in_list": true }
    }
  ],
  "indexes": [
    { "fields": ["phone"], "unique": true },
    { "fields": ["status", "owner_id"] }
  ],
  "relations": [
    { "type": "has_many",   "target_entity": "activity", "foreign_key": "lead_id" },
    { "type": "belongs_to", "target_entity": "user",     "foreign_key": "owner_id" }
  ]
}
```

### 4.2 CRUD Operations (Dynamic)

- **Create** — validation + audit.
- **Read** — filter + pagination + sort.
- **Update** — partial update + diff + audit.
- **Delete** — soft delete + archive.
- **Bulk** — import / export CSV.
- **Search** — full-text via Meilisearch.
- **Aggregate** — count, sum, avg, group-by.

---

## 5. Workflow Engine

### 5.1 State Machine

```
[NEW] ─contact──► [CONTACTED] ─qualified──► [QUALIFIED] ─won──► [WON]
                      │                       │
                      └──not_interested────► [LOST]
```

### 5.2 Workflow Definition

```json
{
  "workflow_code": "lead_lifecycle",
  "entity": "lead",
  "states": [
    { "code": "new",       "label": "Mới",        "color": "#3b82f6" },
    { "code": "contacted", "label": "Đã liên hệ", "color": "#8b5cf6" },
    { "code": "qualified", "label": "Đủ ĐK",      "color": "#10b981" },
    { "code": "won",       "label": "Thắng",      "color": "#22c55e" },
    { "code": "lost",      "label": "Thua",       "color": "#ef4444" }
  ],
  "transitions": [
    { "from": "new", "to": "contacted", "label": "Liên hệ",
      "permission": "lead.contact", "triggers": ["send_notification", "create_activity"] },
    { "from": "contacted", "to": "qualified",
      "permission": "lead.qualify", "condition": "score >= 70" },
    { "from": "*", "to": "lost", "permission": "lead.mark_lost" }
  ]
}
```

### 5.3 Workflow Triggers (Automation)

- `send_notification` — email / push.
- `create_activity` — activity log entry.
- `assign_owner` — auto-assign Lead.
- `update_score` — refresh AI score.
- `webhook` — call external API.
- `custom_function` — sandboxed script.

### 5.4 Execution Modes

- `sync` — block transition until trigger done (max 2 s; fail → rollback).
- `async_queue` — push to NATS, transition succeeds immediately.
- `async_deferred` — trigger runs after 5-60 s in batch.

### 5.5 Compensation & Limits

- Each transition can declare a `compensating_transition` (rollback within 24 h).
- `MAX_WORKFLOW_DEPTH = 5`, `MAX_TRIGGERS_PER_TRANSITION = 10`, `MAX_TIME_WORKFLOW_INSTANCE = 90 days`.
- Retry with exponential backoff + dead-letter queue.

### 5.6 Simulation CLI

```bash
rincoctl workflow simulate --tenant apex-fintech --entity lead \
  --record-id abc-def --to-state qualified --actor user:manager-001
```

---

## 6. Validation Engine (5 Layers)

```
1. Type check (built-in)
2. JSON Schema validation
3. CEL expression
4. Uniqueness constraint (DB)
5. Cross-entity rules
```

### 6.1 CEL Sandbox

```go
SandboxConfig{
  MaxExpressionLen:  4096,
  MaxEvalTime:       100 * time.Millisecond,
  MaxRecursionDepth: 10,
  MaxAllocBytes:     65536,
  EnableLoops:       false,   // DoS protection
}
```

Custom functions: `subtree_of`, `today_vn`, `exists_in_db`, `related_entity`, `geoip`, etc.

### 6.2 CEL Examples

```cel
state == "qualified" && score != null
phone.matches("^(\\+84|0)\\d{9,10}$")
state == "won" && deal_value > 0
owner_id in subtree_of(current_user_id)
```

### 6.3 Cross-Entity Rules

```yaml
won_deal_value_must_match_quote:
  when: state == "won"
  check: related_deal.actual_value == related_quote.total_amount
  error_message: "Giá trị deal thắng phải khớp với báo giá"
```

### 6.4 Performance Optimisations

- Schemas cached in Valkey.
- CEL compiled once, cached.
- Parallel evaluation across fields.

---

## 7. Code Generation Pipeline

### 7.1 When Code-gen Triggers

A schema is considered "stable" when **all** of:

```
schema_age > 7 days
no_changes_in_last_3_days
total_changes_in_lifetime > 5
daily_record_count > 1000
field_count > 8
error_rate < 0.1%
```

### 7.2 Pipeline Stages

```
[1] Detect stable schema (cron hourly)
        ↓
[2] Snapshot schema JSON → Git (tagged commit)
        ↓
[3] Trigger CI/CD "schema-codegen-{entity_code}"
        ├── sqlc generate
        ├── ent generate
        └── protoc → ts-proto
        ↓
[4] Build binary crm-core-{version}-ent-{entity}
        ↓
[5] Push to ghcr.io/itdoanh/rinco/crm-core:{version}-ent-{entity}
        ↓
[6] Canary 5 % traffic for 30 min
        ↓
[7] Rollout 100 % if metrics OK
```

### 7.3 Go Ent Output Example

```go
type Lead struct { ent.Schema }
func (Lead) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
        field.UUID("tenant_id", uuid.UUID{}).Immutable(),
        field.UUID("owner_id", uuid.UUID{}),
        field.String("name").MaxLen(255).NotEmpty(),
        field.String("phone").Optional().MaxLen(20),
        field.String("email").Optional(),
        field.String("source").NotEmpty(),
        field.Int("score").Optional().Range(0, 100),
        field.String("status").Default("new"),
        field.JSON("data", map[string]interface{}{}).Optional(),
        ...
    }
}
```

### 7.4 TypeScript Type Generation

```typescript
export interface Lead {
  id: string;
  tenant_id: string;
  owner_id: string;
  name: string;
  phone?: string;
  email?: string;
  source: 'facebook' | 'tiktok' | 'google' | 'direct' | 'referral' | 'organic';
  score?: number;
  status: 'new' | 'contacted' | 'qualified' | 'won' | 'lost';
  data: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}
```

---

## 8. Database Strategy

### 8.1 Hybrid Storage

```sql
CREATE TABLE leads (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  owner_id UUID NOT NULL,
  name TEXT NOT NULL,
  phone TEXT,
  email TEXT,
  source TEXT,
  status TEXT NOT NULL DEFAULT 'new',
  score INT,
  data JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_leads_data_gin ON leads USING GIN(data);
CREATE INDEX idx_leads_status_owner ON leads(tenant_id, status, owner_id);
```

### 8.2 JSONB Index Patterns

- `GIN` on `data` (general).
- Expression index for hot numeric field: `((data->>'budget')::numeric)`.
- B-tree for hot enum: `(data->>'city') WHERE data->>'city' IS NOT NULL`.
- Composite JSONB index for hot combinations.
- Partial index for boolean hot values.

### 8.3 Storage Mode Decision Tree

```
Schema "stable" with 100 % fields hot? ─yes─► Stable mode (relational columns)
                                       ─no──► PostgreSQL JSONB + GIN
                                                ↓ drift > 50 % for 30 days?
                                                └─yes─► Migrate to MongoDB (background)
```

### 8.4 Drift Detection

```python
drift_score = rare_fields / total_fields
  where rare = fields appearing in < 50 % of last 1000 records
```

If score > 0.5 for 30 days → recommend MongoDB migration.

---

## 9. Feature Catalog (≥ 25 selected)

### 9.1 Schema Builder (1–30)

1. Create new entity.
2. Create field.
3. Edit field.
5. Soft-delete (archive) field.
6. Clone field from another entity.
7. Reorder fields (drag-drop).
8. Field validation rule builder.
9. Required toggle.
10. Unique toggle.
11. Default value.
12. Conditional show/hide (formula).
13. Field help text.
14. Field placeholder.
15. Field i18n.
16. Field-level permission (who can edit).
17. Field history tracking.
18. Field mask (PII).
19. Field encryption.
20. Custom relationship field.
21. Reference field.
22. Lookup field (auto-fill).
23. Computed / formula field.
24. Rollup field (sum / count from child).
25. Approval field.
26. Read-only field.
27. Calculated on create.
28. Auto-number field.
29. Version field.
30. Deleted field recovery.

### 9.2 Workflow Builder (31–55)

31. Create workflow for entity.
32. Define states.
33. Define transitions.
34. Visual workflow editor (React Flow).
35. Transition permission.
36. Transition CEL condition.
37. Transition triggers.
38. Auto-assign on transition.
39. Auto-notify on transition.
40. Webhook on transition.
41. Time-based transition (SLA).
42. Bulk transition.
43. Transition history log.
44. Workflow simulation.
45. Workflow version.
46. Clone workflow.
47. Export / import workflow.
48. A/B test workflow.
49. Workflow analytics.
50. Performance per state.
51. Bottleneck alert.
52. Multi-workflow per entity.
53. Sub-workflow.
54. Workflow rollback.
55. Workflow approval chain.

### 9.3 Custom Views (56–75)

56. List view config (columns, filters, sort).
57. Kanban view (by state).
58. Calendar view.
59. Timeline (Gantt) view.
60. Map view (geo fields).
61. Gallery view (images).
62. Form view (input).
63. Detail view (read).
64. Saved views per user.
65. Shared views per team.
66. Public views (link share).
67. Embedded views (iframe).
68. View permissions.
69. View export.
70. View print.
71. Custom filters builder.
72. Filter presets.
73. Advanced search.
74. Search history.
75. Saved searches.

### 9.4 Forms & Pages (76–95)

76. Custom form builder (drag-drop).
77. Form sections (tabs, accordions).
78. Form conditional fields.
79. Form validation rules.
80. Form submit action.
81. Form success page.
82. Multi-step form.
83. Form A/B test.
84. Form analytics.
85. Form embed code.
86. Form webhook.
87. Form auto-save.
88. Form draft mode.
89. Form pre-fill (URL params).
90. Form localization.
91. CAPTCHA integration.
92. Form file upload.
93. Form signature.
94. Form auto-respond email.
95. Form GDPR consent.

### 9.5 Automation (96–115)

96. Time-based (cron) trigger.
97. Event-based (create / update / delete) trigger.
98. Manual (button) trigger.
99. Webhook trigger (incoming).
100. Email trigger (incoming).
101. Scheduled job.
102. Action: send email.
103. Action: send SMS.
104. Action: send notification.
105. Action: create record.
106. Action: update record.
107. Action: assign owner.
108. Action: call webhook.
109. Action: run sandboxed function.
110. Action: branch (if / else).
111. Action: loop.
112. Action: delay.
113. Action: parallel.
114. Action: human approval.
115. Action: AI agent.

### 9.6 Industry Templates (116–130)

116. Real Estate.
117. Finance / Credit.
118. Retail / E-commerce.
119. Education / Training.
120. Travel.
121. Healthcare / Clinic.
122. Spa / Beauty.
123. F&B / Restaurant.
124. Logistics.
125. B2B Services.
126. Marketing Agency.
127. Insurance.
128. Telecom.
129. Real Estate Rental.
130. Custom Builder Wizard.

---

## 10. API Surface (selection)

### 10.1 Schema Management

```
GET    /api/dm/v1/entities
POST   /api/dm/v1/entities
GET    /api/dm/v1/entities/:code
PATCH  /api/dm/v1/entities/:code        (If-Match: etag)
DELETE /api/dm/v1/entities/:code
POST   /api/dm/v1/entities/:code/{clone,export,import}

GET    /api/dm/v1/entities/:code/fields
POST   /api/dm/v1/entities/:code/fields
PATCH  /api/dm/v1/entities/:code/fields/:code
DELETE /api/dm/v1/entities/:code/fields/:code
POST   /api/dm/v1/entities/:code/fields/reorder

POST   /api/dm/v1/entities/:code/migrate
GET    /api/dm/v1/entities/:code/migrations
```

### 10.2 Dynamic CRUD (generic)

```
GET    /api/crm/v1/:entity
POST   /api/crm/v1/:entity
GET    /api/crm/v1/:entity/:id
PATCH  /api/crm/v1/:entity/:id
DELETE /api/crm/v1/:entity/:id
POST   /api/crm/v1/:entity/bulk
POST   /api/crm/v1/:entity/search
POST   /api/crm/v1/:entity/aggregate
```

### 10.3 Workflow

```
GET    /api/dm/v1/workflows
POST   /api/dm/v1/workflows
PATCH  /api/dm/v1/workflows/:code
POST   /api/dm/v1/workflows/:code/simulate
GET    /api/dm/v1/workflows/:code/history

POST   /api/crm/v1/:entity/:id/transition
GET    /api/crm/v1/:entity/:id/workflow-state
```

### 10.4 Views

```
GET    /api/dm/v1/views
POST   /api/dm/v1/views
PATCH  /api/dm/v1/views/:id
DELETE /api/dm/v1/views/:id
POST   /api/dm/v1/views/:id/share
```

---

## 11. Schema Builder UI

### 11.1 Visual Layout

```
[Entity: Lead ▼]  [+ + Field + Settings + Export]
┌─────────────────────────────────────────────┐
│ Sidebar          │ Main Canvas              │
│ Entities         │                          │
│  ▸ Lead ✓        │   ┌──────────────┐       │
│   ▸ name         │   │ name [string]│       │
│   ▸ phone        │   │ required     │       │
│   ▸ email        │   └──────────────┘       │
│  ▸ Deal          │   ┌──────────────┐       │
│  ▸ Activity      │   │ phone[string]│       │
│  ▸ Product       │   │ pattern: VN  │       │
└─────────────────────────────────────────────┘
```

### 11.2 Visual Workflow Editor

- React Flow + dnd-kit.
- Drag-drop states.
- Draw transitions with conditions.
- Triggers shown on hover.
- Real-time validation.

### 11.3 Tech Stack

- React 18 + TypeScript.
- React Flow for workflow editor.
- dnd-kit for field drag-drop.
- shadcn/ui components.
- Monaco editor for CEL.
- Zod for runtime validation.

---

## 12. Industry Templates (5+ examples)

### 12.1 Real Estate

- Entities: `lead` (12 fields incl. budget range, property type, bedrooms, interest), `viewing`, `property`, `deal`.
- Workflow: `new → contacted → viewing_scheduled → viewed → negotiating → won/lost`.

### 12.2 Finance / Credit

- Entities: `loan_application` (income, loan amount, purpose, credit_score, KYC, collateral, DTI), `loan_decision`.
- Workflow: `new → pre_qualified → docs_collected → under_review → approved/rejected → contract_signed → disbursed`.
- Approval chain: 3 levels (credit officer / branch manager / CEO) based on amount.

### 12.3 Education

- Entities: `student` (parent info, current school, grade level, schedule), `enrollment`, `course`.
- Workflow: `new → contacted → trial_booked → trialed → enrolled / nurture / lost`.

### 12.4 E-commerce

- Entities: `customer` (rollup LTV, total_orders, last_purchase_date), `order`, `product`.
- Workflow: `visitor → lead → first_purchase → repeat_customer → vip`.
- Integration: Shopify / WooCommerce.

### 12.5 Marketing Agency

- Entities: `client` (retainer, health_score, churn_risk), `campaign`.
- Workflow: `prospect → negotiating → contract_signed → onboarding → active → paused → churned`.

### 12.6 Template Install CLI

```bash
rincoctl template install real_estate --tenant apex-fintech
✓ Loaded template (4 entities, 1 workflow, 5 relations)
✓ Created entity: lead (12 fields)
✓ Created workflow: lead_lifecycle_re (7 states, 6 transitions)
✓ Imported 250 sample records
Done in 4.2 s
```

---

## 13. Hot Reload & Versioning

### 13.1 Optimistic Locking

- Every entity has `version BIGINT` and `etag UUIDv7`.
- `PATCH` requires `If-Match: "etag"` and `expected_version` in body.
- On mismatch → `409 Conflict` with current version + diff summary.

### 13.2 Schema Change Audit Log

```sql
CREATE TABLE schema_change_log (
  id, tenant_id, entity_code, schema_version,
  change_type,   -- 'create' | 'update_field' | 'delete_field' | 'migrate'
  changed_fields JSONB, diff JSONB,
  actor_user_id, trace_id, applied_at
);
```

### 13.3 Hot Reload via NATS Pub-Sub

When admin saves a schema:

```
dynamic-model-service
  ├─ UPDATE schema SET version = 13, ...
  ├─ INSERT schema_change_log
  └─ PUBLISH NATS: schema.{tenant_id}.{entity_code}.updated
         ↓
crm-core (invalidate cache, reload workflow)
chat-engine (reload entity rules)
landing-ingest (reload form schema)
```

### 13.4 Cache Strategy (3 Levels)

1. **L1 – Ristretto** in-process (per service).
2. **L2 – Valkey** shared cache.
3. **L3 – PostgreSQL** source of truth.

---

## 14. Database Schema (key tables)

### 14.1 `entity_definitions`

```sql
CREATE TABLE entity_definitions (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  code TEXT NOT NULL,                 -- '^[a-z][a-z0-9_]{2,49}$'
  display_name, description, icon, color,
  is_system BOOL,
  fields JSONB, workflows JSONB, indexes JSONB,
  relations JSONB, permissions JSONB,
  json_schema JSONB,
  version BIGINT, etag TEXT,
  created_at, updated_at, created_by, updated_by,
  deleted_at,
  UNIQUE (tenant_id, code) WHERE deleted_at IS NULL
);
```

Trigger auto-bumps `version` + rotates `etag`. RLS policy isolates by `app.current_tenant_id`; super-admin bypass via `app.is_super_admin`.

### 14.2 `dynamic_records`

Generic dynamic row:

- Fixed columns: `id`, `tenant_id`, `entity_code`, `owner_id`, `name`, `status`, `data JSONB`, `search_text TSVECTOR`.
- Indexes: `(tenant_id, entity_code)`, `(owner_id)`, GIN on `data`, GIN on `search_text`, `(tenant_id, entity_code, status)`.
- Trigger keeps `search_text` up to date for full-text search.

---

## 15. Edge Cases & Error Scenarios (≥ 30 covered)

### 15.1 Schema Definition Conflicts (E1–10)

Two admins editing simultaneously, duplicate field codes, type conflicts, dropping required fields, deleting referenced entities, invalid CEL length, circular formula refs, missing formula targets, index name collisions.

### 15.2 Migration Failures (E11–20)

Backfill timeout, lock timeout, disk full, connection drop, concurrent migration, tenant deleted mid-migration, version conflict, bad SQL, memory exhaustion, replication lag.

### 15.3 JSONB Query Performance (E21–30)

Full table scan, deep nesting, N+1, large IN clause, multi-OR, JSONB→numeric cast fail, Unicode keys, large TOAST, concurrent key update, uncommitted read during backfill.

### 15.4 Workflow Engine (E31–40)

Trigger timeout, infinite sub-workflow recursion, webhook receiver down, SMTP fail, CEL panic, invalid workflow JSON, missing permission, deadlock cycle, record quota, instance > 90 days.

### 15.5 Code-Gen (E41–50)

Ent generation fail, TS type cycle, git push conflict, binary build fail, canary regression, SAST security issue, binary > 100 MB, old binary still running, pipeline stuck, spam codegen triggers.

---

## 16. Security & Permissions

### 16.1 Schema Permission Matrix

| Role | View | Edit | Delete | Activate Template |
|------|------|------|--------|-------------------|
| Super Admin | All tenants | Any | Any | Any |
| Tenant Admin | Own | Own | Own | Own |
| Tenant Manager | Own | Own entities | ❌ | ❌ |
| Tenant Staff | Own | ❌ | ❌ | ❌ |
| External Auditor | Own | ❌ | ❌ | ❌ |

### 16.2 Field-Level Permission

```json
{
  "field_code": "phone",
  "permissions": {
    "view":   ["owner", "manager", "admin", "tenant_admin"],
    "create": ["owner", "manager", "admin"],
    "update": ["owner", "manager", "admin"],
    "export": ["manager", "admin", "tenant_admin"],
    "mask_in_list": ["staff"],
    "encrypted_at_rest": true,
    "audit_log": true
  }
}
```

### 16.3 Encryption & Redaction

- At rest: PostgreSQL TDE or column-level pgcrypto.
- In transit: TLS 1.3 only.
- In app: `Redact` helper masks PII (`password`, `phone`, `email`, `ssn`, `credit_card`, `id_number`).

### 16.4 GDPR / Right to be Forgotten

- Soft delete with 30-day grace.
- Scheduled hard-delete job anonymizes PII (`name = 'ANONYMIZED-<id>'`, `phone/email = NULL`, `data = '{}'`).

---

## 17. Disaster Recovery

### 17.1 RPO / RTO

| Resource | RPO | RTO | Method |
|----------|-----|-----|--------|
| Schema definitions | 5 min | 30 min | WAL-G + PITR |
| Workflow definitions | 5 min | 30 min | Same as above |
| Custom views | 1 h | 30 min | Daily snapshot |
| Industry templates | 0 | 5 min | Git clone |
| Code-gen artifacts | 0 | 10 min | Re-run pipeline |

### 17.2 Schema Corruption Auto-Repair

```bash
rincoctl schema verify --tenant apex-fintech
rincoctl schema repair  --tenant apex-fintech --auto --backup
# → backup to s3://rinco-backups/repair/...
# → clean orphan records, rebuild indexes, verify
```

---

## 18. Cost Estimation

### 18.1 Per 1,000 Tenants

- PostgreSQL (shared `db.r6g.16xlarge` 4 TB): ~$5 / tenant / mo.
- ScyllaDB for events / audit: ~$2 / tenant / mo.
- ClickHouse analytics: ~$0.8 / tenant / mo.
- MongoDB for drift entities only: ~$6 / tenant / mo.
- Valkey cache: ~$1.5 / tenant / mo.
- **Total storage: ~$15 / tenant / month (shared mode).**

### 18.2 Compute

- `dynamic-model-service`: $0.4 / tenant.
- `validation-engine`: $0.8 / tenant.
- `workflow-engine`: $0.4 / tenant.
- `migration-worker`: $0.2 / tenant.
- **Total compute: ~$1.8 / tenant / month.**

### 18.3 Optimization Tips

- Tiered storage: NVMe → S3 IA → S3 Glacier.
- Index pruning after 30 days unused.
- Code-gen binary caching in ECR.
- Reserved Instances + Spot for migration workers.

---

## 19. Service Architecture (8 services)

| Service | Language | Responsibility |
|---------|----------|----------------|
| `dynamic-model-service` | Go (Echo + Huma) | Entity / field CRUD |
| `validation-engine` | Go (cel-go) | Runtime validation pipeline |
| `workflow-engine` | Go | State machine + triggers |
| `schema-migrator` | Python (Celery) | Drift detection + migration |
| `codegen-pipeline` | Go + Bash | Generate Ent + TS types |
| `formula-evaluator` | Rust (wasmtime) | High-perf formula eval |
| `meta-schema-ui` | Next.js | Visual Schema Builder |
| `dynamic-schema-bff` | TS (Bun) | BFF aggregation for UI |

Inter-service: HTTPS → BFF → gRPC + Connect-RPC → backend services, all wired via NATS for invalidation.

---

## 20. Implementation Roadmap (16 weeks / 4 phases)

- **Phase 1 — Foundation (W1–4):** Setup, schema definition parser, validation engine, dynamic CRUD + DB, basic workflow engine.
- **Phase 2 — Hardening (W5–8):** Cross-entity rules, formula DSL, rollup, optimistic locking, NATS pub-sub, Schema Builder UI MVP, code-gen pipeline.
- **Phase 3 — Production Readiness (W9–12):** 5 industry templates, zero-downtime migration tooling, drift detection, field-level permission + encryption, load test, DR + observability.
- **Phase 4 — Scale & Polish (W13–16):** Multi-region replication, read replicas, cache warming, docs, training, GA with canary deploy + pen test.

Total: 9 people × 16 weeks ≈ 144 person-weeks.

---

## 21. Acceptance Criteria (15 ACs)

| AC | Criterion | Target |
|----|-----------|--------|
| AC-DM-01 | New field without restart | Hot reload < 5 s |
| AC-DM-02 | Validate 1000 fields | < 50 ms |
| AC-DM-03 | Dynamic CRUD | < 100 ms p95 |
| AC-DM-04 | Workflow transition | < 200 ms p95 |
| AC-DM-05 | JSONB query with GIN | < 50 ms p95 |
| AC-DM-06 | Schema save invalidates cache | < 1 s |
| AC-DM-07 | Multi-tenant isolation | 100 % |
| AC-DM-08 | Code-gen pipeline success | > 95 % |
| AC-DM-09 | Zero-downtime migration | DR drill |
| AC-DM-10 | Industry template install | < 10 s |
| AC-DM-11 | Formula evaluation | < 5 ms |
| AC-DM-12 | CEL sandbox resource limit | Fuzz test |
| AC-DM-13 | Concurrent edit conflict | Concurrency test |
| AC-DM-14 | Audit log 100 % schema changes | Verification |
| AC-DM-15 | Disaster recovery RTO | < 30 min |

---

## 22. Highlights — Why It Matters

1. Each tenant owns its schema without forking code.
2. Validation runtime stays under 5 ms via CEL sandbox.
3. Workflow automation deploys without re-deploy.
4. Optimistic locking + ETag + audit log guarantee safe concurrent edits.
5. Auto code-gen on stable schemas trades flexibility for raw performance.
6. 15+ industry templates enable fast onboarding.
7. Drift detection automatically suggests MongoDB migration when structures diverge.