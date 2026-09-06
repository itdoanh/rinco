# Phần 4 – Trình Sinh Mô Hình Doanh Nghiệp Động (Dynamic Model Engine)

> **Phân hệ:** Meta-Schema Engine cho phép tạo vô số mô hình CRM khác nhau theo ngành nghề.
> **Mục tiêu:** Admin/tenant có thể tự định nghĩa cấu trúc dữ liệu, workflow, trạng thái, validation rules cho CRM của riêng mình.
> **Đặc thù:** JSON Schema + Ent Code-gen + Runtime Validation. Không cần restart service.

---

## Mục lục

1. [Mục tiêu & Nguyên tắc](#1-mục-tiêu--nguyên-tắc)
2. [Kiến trúc Meta-Schema](#2-kiến-trúc-meta-schema)
3. [Schema Definition Language](#3-schema-definition-language)
4. [Entity Builder](#4-entity-builder)
5. [Workflow Engine](#5-workflow-engine)
6. [Validation Engine](#6-validation-engine)
7. [Code Generation](#7-code-generation)
8. [Database Strategy](#8-database-strategy)
9. [Danh sách tính năng (≥ 100)](#9-danh-sách-tính-năng)
10. [API Surface](#10-api-surface)
11. [UI/UX Schema Builder](#11-uiux-schema-builder)
12. [Industry Templates](#12-industry-templates)
13. **MỤC LỤC MỚI (AUDIT + MỞ RỘNG)**
13. [Audit Report](#13-audit-report)
14. [Type System Mở Rộng](#14-type-system-mở-rộng)
15. [Workflow Engine Nâng Cao](#15-workflow-engine-nâng-cao)
16. [Cross-field & Cross-entity Validation](#16-cross-field--cross-entity-validation)
17. [Hot Reload & Schema Versioning](#17-hot-reload--schema-versioning)
18. [Code Generation Pipeline chi tiết](#18-code-generation-pipeline-chi-tiết)
19. [Database Strategy nâng cao](#19-database-strategy-nâng-cao)
20. [Industry Templates chi tiết](#20-industry-templates-chi-tiết)
21. [Migration & Backfill](#21-migration--backfill)
22. [Performance & Benchmark](#22-performance--benchmark)
23. [Security & Permission Matrix](#23-security--permission-matrix)
24. [Disaster Recovery](#24-disaster-recovery)
25. [Cost Estimation](#25-cost-estimation)
26. [Testing Strategy](#26-testing-strategy)
27. [Implementation Roadmap chi tiết](#27-implementation-roadmap-chi-tiết)
28. [Open Questions / Cần user xác nhận](#28-open-questions--cần-user-xác-nhận)

---

## 1. Mục tiêu & Nguyên tắc

### 1.1. Mục tiêu
| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| DM-1 | Tạo entity mới không cần restart | Hot reload < 5s |
| DM-2 | Mỗi tenant có schema riêng | Multi-tenant JSONB |
| DM-3 | Validation runtime < 5ms | CEL engine |
| DM-4 | Workflow automation | Rule engine |
| DM-5 | Code-gen optional | Auto Ent schema |

### 1.2. Nguyên tắc
- **Schema is data:** Schema lưu trong PostgreSQL, không hardcode.
- **Backward compat:** Thêm field mới không phá field cũ.
- **Validation strict:** Data phải pass schema trước khi lưu.
- **Versioning:** Mỗi schema có version, migration tự động.
- **Type-safe ưu tiên:** Nếu có thể generate code Go → dùng code; nếu không → JSONB + runtime.

---

## 2. Kiến trúc Meta-Schema

### 2.1. Các tầng
```
┌──────────────────────────────────────────────────┐
│  Tier 1: Meta-Schema Definition                  │
│  - Loại entity (Lead, Deal, Customer, ...)        │
│  - Fields (text, number, enum, ref, ...)         │
│  - Workflows (state machine)                     │
│  - Validation Rules                              │
└──────────────────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────┐
│  Tier 2: Schema Storage (PostgreSQL)             │
│  - tenants.meta_schema (JSON Schema)             │
│  - tenants.workflow_defs (JSON)                  │
│  - tenants.field_defs (JSONB)                    │
└──────────────────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────┐
│  Tier 3: Runtime Engine                          │
│  - JSON Schema validator (gojsonschema)          │
│  - CEL evaluator                                  │
│  - Workflow runner                                │
└──────────────────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────┐
│  Tier 4: Data Layer                              │
│  - PostgreSQL: relational + JSONB                │
│  - MongoDB: fully dynamic                         │
└──────────────────────────────────────────────────┘
```

### 2.2. Hybrid Storage
- **Cột hardcode:** Cho field phổ biến (name, email, phone, owner_id, tenant_id, status, created_at).
- **JSONB column `data`:** Cho custom fields.
- **MongoDB option:** Nếu schema quá dynamic (mỗi record khác structure).

---

## 3. Schema Definition Language

### 3.1. JSON Schema Draft 2020-12
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://rinco.app/schemas/apex/lead.json",
  "title": "Lead",
  "type": "object",
  "properties": {
    "name": { "type": "string", "minLength": 1, "maxLength": 255 },
    "phone": { "type": "string", "pattern": "^(\\+84|0)\\d{9,10}$" },
    "email": { "type": "string", "format": "email" },
    "source": {
      "type": "string",
      "enum": ["facebook", "tiktok", "google", "direct", "referral", "organic"]
    },
    "score": { "type": "integer", "minimum": 0, "maximum": 100 },
    "tags": { "type": "array", "items": { "type": "string" } },
    "custom_fields": {
      "type": "object",
      "properties": {
        "budget": { "type": "number", "minimum": 0 },
        "interest": { "type": "string", "enum": ["buy", "rent", "invest"] },
        "preferred_location": { "type": "string" }
      }
    }
  },
  "required": ["name", "phone", "source"],
  "additionalProperties": false
}
```

### 3.2. Field Types hỗ trợ
- `string` (text, textarea, rich text, markdown)
- `number` (integer, float, currency, percentage)
- `boolean`
- `date`, `datetime`, `time`
- `enum` (dropdown, radio, multi-select)
- `array`
- `object` (nested)
- `reference` (foreign key)
- `file` (upload)
- `image`
- `video`
- `geo` (lat/lng)
- `phone`
- `email`
- `url`
- `color`
- `json`
- `formula` (computed)

### 3.3. Validation
- **Required fields**
- **Unique fields** (vd: email unique per tenant)
- **Pattern** (regex)
- **Range** (min/max)
- **Enum**
- **Custom CEL expressions:** `"score >= 70 || status == 'verified'"`

---

## 4. Entity Builder

### 4.1. Entity Structure
```json
{
  "entity_code": "lead",
  "display_name": "Khách hàng tiềm năng",
  "icon": "lucide:user-plus",
  "color": "#10b981",
  "is_system": false,
  "fields": [
    {
      "code": "name",
      "label": "Họ tên",
      "type": "string",
      "required": true,
      "validation": { "minLength": 1, "maxLength": 255 },
      "ui": { "order": 1, "width": "full", "show_in_list": true }
    },
    ...
  ],
  "indexes": [
    { "fields": ["phone"], "unique": true },
    { "fields": ["status", "owner_id"] }
  ],
  "relations": [
    {
      "type": "has_many",
      "target_entity": "activity",
      "foreign_key": "lead_id"
    },
    {
      "type": "belongs_to",
      "target_entity": "user",
      "foreign_key": "owner_id"
    }
  ]
}
```

### 4.2. CRUD Operations
- **Create:** Insert với validation + audit.
- **Read:** Filter + pagination + sort.
- **Update:** Partial update + diff + audit.
- **Delete:** Soft delete + archive.
- **Bulk:** Import/Export CSV.
- **Search:** Full-text qua Meilisearch.
- **Aggregate:** Count, sum, avg, group-by.

---

## 5. Workflow Engine

### 5.1. State Machine
```
[NEW] ─contact──► [CONTACTED] ─qualified──► [QUALIFIED] ─won──► [WON]
                      │                     │
                      └──not_interested──► [LOST]
```

### 5.2. Workflow Definition
```json
{
  "workflow_code": "lead_lifecycle",
  "entity": "lead",
  "states": [
    { "code": "new", "label": "Mới", "color": "#3b82f6" },
    { "code": "contacted", "label": "Đã liên hệ", "color": "#8b5cf6" },
    { "code": "qualified", "label": "Đủ điều kiện", "color": "#10b981" },
    { "code": "won", "label": "Thắng", "color": "#22c55e" },
    { "code": "lost", "label": "Thua", "color": "#ef4444" }
  ],
  "transitions": [
    {
      "from": "new",
      "to": "contacted",
      "label": "Liên hệ",
      "permission": "lead.contact",
      "triggers": ["send_notification", "create_activity"]
    },
    {
      "from": "contacted",
      "to": "qualified",
      "label": "Đánh giá đủ ĐK",
      "permission": "lead.qualify",
      "condition": "score >= 70"
    },
    {
      "from": "*",
      "to": "lost",
      "label": "Đánh dấu thua",
      "permission": "lead.mark_lost"
    }
  ]
}
```

### 5.3. Workflow Triggers (Automation)
- **send_notification:** Gửi email/push.
- **create_activity:** Tạo activity log.
- **assign_owner:** Tự động phân Lead.
- **update_score:** Cập nhật AI score.
- **webhook:** Gọi external API.
- **custom_function:** Chạy script (sandboxed).

### 5.4. Workflow Execution
```
1. User thực hiện action → trigger transition
2. Validate permission + condition
3. Update state + audit log
4. Run triggers (sync hoặc queue)
5. Notify subscribers (WebSocket)
```

---

## 6. Validation Engine

### 6.1. Layers
```
┌─────────────────────────────────────────┐
│ Layer 1: Type check (built-in)          │
│ Layer 2: JSON Schema validation         │
│ Layer 3: CEL expression                 │
│ Layer 4: Uniqueness constraint (DB)     │
│ Layer 5: Cross-entity rules             │
└─────────────────────────────────────────┘
```

### 6.2. CEL Examples
```cel
// Lead score must be set before qualify
state == "qualified" && score != null

// Phone must match VN format
phone.matches("^(\\+84|0)\\d{9,10}$")

// Won leads must have deal_value
state == "won" && deal_value > 0

// Owner must be in same subtree
owner_id in subtree_of(current_user_id)
```

### 6.3. Performance
- Schema cache in Valkey.
- CEL compiled once, cached.
- Validation parallel cho multi-field.

---

## 7. Code Generation

### 7.1. Go Ent Code-gen (Hot Path)
- Khi schema stable, generate Ent schema Go.
- Build thành binary riêng → deploy.
- Hot reload không cần restart cho JSONB fields.

### 7.2. TypeScript Type-gen
- Generate TS types từ JSON Schema.
- Frontend dùng để validate form + autocomplete.

### 7.3. Migration Tool
- So sánh schema version → generate SQL migration.
- Backup data trước khi apply.
- Rollback nếu fail.

---

## 8. Database Strategy

### 8.1. Hybrid Storage
```sql
-- Bảng lead (fixed columns + JSONB)
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

CREATE INDEX idx_leads_data_gin ON leads USING GIN (data);
CREATE INDEX idx_leads_status_owner ON leads(tenant_id, status, owner_id);
```

### 8.2. JSONB Indexing (cho custom fields hot)
```sql
-- Nếu field "budget" hay query
CREATE INDEX idx_leads_budget ON leads ((data->>'budget')::numeric);

-- Nếu field "city" hay filter
CREATE INDEX idx_leads_city ON leads ((data->>'city'));
```

### 8.3. Full Schema Mode (MongoDB Option)
- Nếu mỗi Lead có structure khác nhau → chuyển sang MongoDB.
- Auto-detect: nếu số field khác nhau > 50% → MongoDB.

---

## 9. Danh sách tính năng (≥ 100)

### 9.1. Schema Builder (1-30)
1. Tạo entity mới.
2. Tạo field mới trong entity.
3. Sửa field.
4. Xóa field (soft – archive).
6. Clone field từ entity khác.
7. Reorder fields (drag-drop).
8. Field validation rule builder.
9. Required field toggle.
10. Unique field toggle.
11. Default value.
12. Conditional show/hide (formula).
13. Field help text.
14. Field placeholder.
15. Field i18n (multi-language label).
16. Field permission (ai được sửa).
17. Field history tracking.
18. Field mask (PII).
19. Field encryption (sensitive).
20. Custom field for relationship.
21. Reference field (entity khác).
22. Lookup field (auto-fill từ entity khác).
23. Computed/formula field.
24. Rollup field (sum/count từ child entity).
25. Approval field (cần duyệt).
26. Read-only field.
27. Calculated field on create.
28. Auto-number field.
29. Version field.
30. Deleted field recovery.

### 9.2. Workflow Builder (31-55)
31. Tạo workflow cho entity.
32. Định nghĩa states.
33. Định nghĩa transitions.
34. Visual workflow editor (React Flow).
35. Transition permission.
36. Transition condition (CEL).
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
47. Export/import workflow.
48. A/B test workflow.
49. Workflow analytics.
50. Workflow performance (avg time per state).
51. Workflow bottleneck alert.
52. Multi-workflow per entity.
53. Sub-workflow.
54. Workflow rollback.
55. Workflow approval chain.

### 9.3. Custom Views (56-75)
56. List view config (columns, filters, sort).
57. Kanban view (theo state).
58. Calendar view.
59. Timeline view (Gantt).
60. Map view (geo fields).
61. Gallery view (image fields).
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

### 9.4. Forms & Pages (76-95)
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
91. Form CAPTCHA integration.
92. Form file upload.
93. Form signature.
94. Form auto-respond email.
95. Form GDPR consent.

### 9.5. Automation (96-115)
96. Time-based trigger (cron).
97. Event-based trigger (create/update/delete).
98. Manual trigger (button).
99. Webhook trigger (external).
100. Email trigger (incoming).
101. Scheduled job.
102. Action: send email.
103. Action: send SMS.
104. Action: send notification.
105. Action: create record.
106. Action: update record.
107. Action: assign owner.
108. Action: call webhook.
109. Action: run function (sandbox).
110. Action: branch (if/else).
111. Action: loop.
112. Action: delay.
113. Action: parallel.
114. Action: human approval.
115. Action: AI agent.

### 9.6. Industry Templates (116-130)
116. Template: Bất động sản.
117. Template: Tài chính/Tín dụng.
118. Template: Bán lẻ/E-commerce.
119. Template: Giáo dục/Đào tạo.
120. Template: Du lịch.
121. Template: Y tế/Phòng khám.
122. Template: Spa/Beauty.
123. Template: F&B (nhà hàng).
124. Template: Logistic.
125. Template: B2B Services.
126. Template: Agency/Marketing.
127. Template: Insurance.
128. Template: Telecom.
129. Template: Real Estate Rental.
130. Template: Custom Builder Wizard.

---

## 10. API Surface

### 10.1. Schema Management
```
GET    /api/dm/v1/entities
POST   /api/dm/v1/entities
GET    /api/dm/v1/entities/:code
PATCH  /api/dm/v1/entities/:code
DELETE /api/dm/v1/entities/:code
POST   /api/dm/v1/entities/:code/clone
POST   /api/dm/v1/entities/:code/export
POST   /api/dm/v1/entities/import

GET    /api/dm/v1/entities/:code/fields
POST   /api/dm/v1/entities/:code/fields
PATCH  /api/dm/v1/entities/:code/fields/:code
DELETE /api/dm/v1/entities/:code/fields/:code
POST   /api/dm/v1/entities/:code/fields/reorder

POST   /api/dm/v1/entities/:code/migrate
GET    /api/dm/v1/entities/:code/migrations
```

### 10.2. Dynamic CRUD
```
# Generic CRUD theo entity code
GET    /api/crm/v1/:entity                    # List
POST   /api/crm/v1/:entity                    # Create
GET    /api/crm/v1/:entity/:id                # Read
PATCH  /api/crm/v1/:entity/:id                # Update
DELETE /api/crm/v1/:entity/:id                # Delete
POST   /api/crm/v1/:entity/bulk               # Bulk operation
POST   /api/crm/v1/:entity/search             # Full-text search
POST   /api/crm/v1/:entity/aggregate          # Aggregate
```

### 10.3. Workflow
```
GET    /api/dm/v1/workflows
POST   /api/dm/v1/workflows
PATCH  /api/dm/v1/workflows/:code
POST   /api/dm/v1/workflows/:code/simulate
GET    /api/dm/v1/workflows/:code/history

POST   /api/crm/v1/:entity/:id/transition     # Trigger transition
GET    /api/crm/v1/:entity/:id/workflow-state
```

### 10.4. Views
```
GET    /api/dm/v1/views
POST   /api/dm/v1/views
PATCH  /api/dm/v1/views/:id
DELETE /api/dm/v1/views/:id
POST   /api/dm/v1/views/:id/share
```

---

## 11. UI/UX Schema Builder

### 11.1. Schema Builder UI
```
┌────────────────────────────────────────────────┐
│ [Entity: Lead ▼]  [+ Field] [Settings] [Export]│
├────────────────────────────────────────────────┤
│ Sidebar        │ Main Canvas                    │
│                │                                 │
│ Entities       │  ┌──────────────┐              │
│ ▸ Lead ✓       │  │ name [string]│              │
│   ▸ name       │  │ required     │              │
│   ▸ phone      │  └──────────────┘              │
│   ▸ email      │  ┌──────────────┐              │
│ ▸ Deal         │  │ phone[string]│              │
│ ▸ Activity     │  │ pattern: VN  │              │
│ ▸ Product      │  └──────────────┘              │
│                │  [+ Add Field]                 │
└────────────────────────────────────────────────┘
```

### 11.2. Visual Workflow Editor (React Flow)
- Drag-drop states.
- Draw transitions with conditions.
- Triggers on hover.
- Real-time validation.

### 11.3. Tech
- **React 18** + **TypeScript**.
- **React Flow** cho workflow editor.
- **dnd-kit** cho drag-drop fields.
- **shadcn/ui** components.
- **Monaco Editor** cho CEL expressions.
- **Zod** cho validation runtime.

---

## 12. Industry Templates

### 12.1. Bất động sản (Real Estate)
**Lead fields:**
- name, phone, email, source
- budget (range), area (number), location (geo)
- property_type (enum: apartment, house, land, shophouse)
- bedrooms, bathrooms
- interest (buy, rent, invest)
- move_in_date
- financing_needed (boolean)
- agent_assigned

**Workflow states:**
- new → contacted → viewing_scheduled → viewed → negotiating → won/lost

**Sample entities:**
- Lead, Viewing, Deal, Property, Contract

### 12.2. Tài chính (Finance)
**Lead fields:**
- name, phone, email, income, employment, loan_amount, loan_purpose, credit_score, collateral

**Workflow:**
- new → pre_qualified → docs_collected → approved/rejected → disbursed

### 12.3. Giáo dục (Education)
**Lead fields:**
- student_name, parent_name, parent_phone, course_interest, current_school, grade_level

**Workflow:**
- new → contacted → trial_scheduled → trialed → enrolled/lost

### 12.4. Bán lẻ (E-commerce)
**Lead/Customer:**
- name, phone, email, total_orders, lifetime_value, last_purchase_date, preferred_category

### 12.5. Marketing Agency
**Lead:**
- name, company, industry, services_interested, budget_range, decision_maker

### 12.6. Template Storage
- PostgreSQL: `tenant.templates` table.
- JSON format mỗi template có 5-15 entities.

---

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-DM-01 | Tạo field mới không cần restart | Hot reload < 5s |
| AC-DM-02 | Validate 1000 field trong < 50ms | CEL benchmark |
| AC-DM-03 | CRUD dynamic entity < 100ms | p95 |
| AC-DM-04 | Workflow transition < 200ms | p95 |
| AC-DM-05 | JSONB query < 50ms với GIN index | p95 |

---

# PHẦN MỞ RỘNG (AUDIT + DEEPEN)

> Phần này bổ sung theo yêu cầu audit & mở rộng. Giữ nguyên 100% nội dung cũ ở trên.

---

## 13. Audit Report

### 13.1. Phần đã đủ chi tiết
- ✅ Tầng kiến trúc 4 tầng (Tier 1-4).
- ✅ Hybrid Storage (PostgreSQL JSONB + MongoDB option).
- ✅ Field Types 18 loại.
- ✅ Validation 6 tầng (Layer 1-6).
- ✅ State machine + transition triggers.
- ✅ Code-gen Go Ent + TS.
- ✅ Industry Templates 5 ví dụ.
- ✅ Acceptance Criteria 5 chỉ số.

### 13.2. Phần còn thiếu (gap)
| Gap ID | Mô tả | Mức độ | Giải pháp |
|--------|--------|--------|-----------|
| GAP-1 | Chưa có Sequence Diagram cho Workflow Trigger (sync/async) | Medium | Thêm ở §15 |
| GAP-2 | CEL Sandbox chưa nói rõ resource limit (CPU/MEM/timeout) | High | Thêm ở §6 và §16 |
| GAP-3 | Chưa có Index Recommendation Engine (auto-suggest GIN/B-tree) | Medium | Thêm ở §19 |
| GAP-4 | Migration zero-downtime chưa nói batch size, lock_timeout | High | Thêm ở §21 |
| GAP-5 | Industry Templates chỉ có field list, thiếu workflow.json + sample_data.csv | Medium | Thêm ở §20 |
| GAP-6 | Concurrent edit conflict (2 admin cùng edit 1 schema) chưa giải quyết | High | Thêm §17 với ETag/optimistic locking |
| GAP-7 | Audit log của schema change chưa tách riêng (chỉ nói chung với audit log) | Medium | Thêm bảng `schema_change_log` |
| GAP-8 | Chưa có Cost Estimation cho từng storage mode (Postgres vs Mongo) | Medium | Thêm ở §25 |
| GAP-9 | Workflow Rollback chưa giải thích rõ undo stack | Low | Thêm ở §15 |
| GAP-10 | Formula field chưa có DSL syntax đầy đủ (chỉ nói chung chung) | Medium | Thêm ở §14 |
| GAP-11 | Sub-workflow chưa có giới hạn độ sâu (infinite recursion risk) | High | Thêm ở §15 |
| GAP-12 | Code-gen cho schema "stable" chưa có tiêu chí "stable" | Medium | Thêm ở §18 |
| GAP-13 | JSON Schema Draft 2020-12 chưa khai thác `oneOf/anyOf/allOf` | Medium | Thêm ở §14 |
| GAP-14 | Multi-tenant isolation trên schema definition: super admin có quyền override | Medium | Thêm ở §23 |
| GAP-15 | Disaster Recovery chưa có RPO/RTO cho schema storage | High | Thêm ở §24 |

### 13.3. Phần có mâu thuẫn nội bộ
- **§2.2 vs §8.3:** Mâu thuẫn "Hybrid Storage" vs "Full Schema Mode MongoDB". Thiếu trigger để chuyển đổi giữa 2 mode.
  → Fix: Thêm §19.4 với algorithm "Schema Drift Score" tự động đề xuất migration.
- **§3.3 vs §6.1:** Validation layers liệt kê 5 layer (Type/JSON/CEL/UniqueDB/Cross-entity) nhưng §3.3 chỉ nói 6 loại validate rule. Cần thống nhất.
- **§8.2 vs §10.2:** Index `idx_leads_data_gin` đã cover mọi JSONB query, nhưng cũng tạo thêm `idx_leads_city` expression index. Trùng lặp → cần ghi rõ "GIN trước, expression index cho hot field".

### 13.4. Phần cần code example cụ thể
- Hàm `validate_record(entity_code, payload)` chạy cả 5 validation layer.
- Hàm `apply_migration(schema_v1, schema_v2)` với backoff + rollback.
- Workflow runner với retry + dead-letter queue.
- Code-gen pipeline trigger tự động khi schema "stable".

---

## 14. Type System Mở Rộng

### 14.1. Extended Field Types
Bổ sung các type mới cho `dynamic-model-engine`:

| Type code | Ý nghĩa | Storage | UI Renderer |
|-----------|---------|---------|-------------|
| `string` | text đơn | TEXT | `<Input />` |
| `text` | multi-line | TEXT | `<Textarea />` |
| `rich_text` | WYSIWYG | TEXT (HTML/MD) | Tiptap editor |
| `markdown` | Markdown | TEXT | CodeMirror preview |
| `number` | số | NUMERIC | `<Input type="number" />` |
| `integer` | số nguyên | BIGINT | `<Input type="number" />` |
| `currency` | tiền tệ | NUMERIC | CurrencyInput + symbol |
| `percent` | phần trăm | NUMERIC | 0-100 slider |
| `boolean` | true/false | BOOLEAN | Switch |
| `date` | ngày | DATE | date-picker |
| `datetime` | ngày giờ | TIMESTAMPTZ | date-time picker |
| `time` | giờ | TIME | time picker |
| `enum` | lựa chọn đơn | TEXT | Select dropdown |
| `multi_select` | nhiều lựa chọn | TEXT[] | Combobox multi |
| `array` | mảng generic | JSONB | Tags input |
| `object` | nested object | JSONB | Nested form (recursive) |
| `reference` | FK sang entity khác | UUID | Lookup picker |
| `file` | upload 1 file | TEXT (storage URL) | File picker |
| `files` | upload nhiều | TEXT[] | Multi-file picker |
| `image` | 1 ảnh | TEXT | Image picker + crop |
| `images` | nhiều ảnh | TEXT[] | Gallery uploader |
| `video` | video | TEXT | Video upload + thumbnail |
| `geo` | lat/lng | JSONB {lat, lng} | Map picker |
| `phone` | SĐT | TEXT | PhoneInput có country code |
| `email` | email | TEXT | EmailInput có verify |
| `url` | URL | TEXT | URL preview |
| `color` | màu | TEXT | Color picker |
| `json` | free JSON | JSONB | Monaco JSON editor |
| `formula` | computed | GENERATED column hoặc runtime | Read-only |
| `rollup` | tổng hợp từ child | Runtime query | Read-only |
| `autonumber` | auto-increment | BIGINT + sequence | Read-only |
| `barcode` | scan barcode | TEXT | Barcode scanner |
| `signature` | chữ ký tay | TEXT (base64 PNG) | Canvas signature pad |
| `qrcode` | QR generator | TEXT (URL) | Generate on the fly |

### 14.2. JSON Schema Extensions (mở rộng §3.1)

#### 14.2.1. Conditional Schema với `oneOf`/`anyOf`/`allOf`
```json
{
  "type": "object",
  "title": "Lead",
  "oneOf": [
    {
      "properties": {
        "source": { "const": "facebook" },
        "fbclid": { "type": "string", "pattern": "^fb\\..*" }
      },
      "required": ["fbclid"]
    },
    {
      "properties": {
        "source": { "const": "tiktok" },
        "ttclid": { "type": "string", "pattern": "^tt\\..*" }
      },
      "required": ["ttclid"]
    }
  ],
  "allOf": [
    {
      "if": {
        "properties": { "score": { "type": "integer" } },
        "required": ["score"]
      },
      "then": {
        "properties": {
          "score": { "minimum": 0, "maximum": 100 }
        }
      }
    }
  ]
}
```

#### 14.2.2. Reference Constraint
```json
{
  "type": "object",
  "properties": {
    "owner_id": {
      "type": "string",
      "format": "uuid",
      "$rinco:reference": {
        "entity": "user",
        "fk_field": "id",
        "display_field": "full_name",
        "filter": { "status": "active" },
        "on_delete": "restrict"
      }
    }
  }
}
```

### 14.3. Formula Field DSL

Cú pháp formula field (tương thích Microsoft Excel + Airtable):

```
Formula Syntax:
  - Field reference: {field_code}
  - Arithmetic: + - * / ^ ( )
  - String concat: & hoặc CONCAT(a, b)
  - Functions:
      IF(condition, true_value, false_value)
      AND(a, b, c)
      OR(a, b)
      NOT(a)
      IS_EMPTY(field)
      CONCAT(a, b, c)
      UPPER(s), LOWER(s), TRIM(s), LEN(s)
      ROUND(n, decimals), CEIL(n), FLOOR(n)
      MIN(a, b), MAX(a, b), AVG(a, b)
      TODAY(), NOW(), DATEADD(date, days)
      COUNTIF(range, criteria)
      SUMIF(range, criteria)
      LOOKUP(entity, code_field, return_field, key_value)
```

Ví dụ:
```
total = {quantity} * {unit_price}
display_name = CONCAT({first_name}, " ", {last_name})
is_hot = IF({score} >= 70, true, false)
days_since_contact = DATEDIFF(NOW(), {last_contacted_at}, "day")
```

### 14.4. Rollup Field

```yaml
rollup_field:
  code: total_deal_value
  label: "Tổng giá trị deal"
  target_entity: deal
  relation_field: lead_id
  target_field: value
  aggregation: sum   # sum | count | avg | min | max
  filter: "status = 'won'"
  refresh: realtime  # realtime | hourly | daily
```

### 14.5. Computed Field Runtime vs Stored

| Mode | Khi nào dùng | Trade-off |
|------|-------------|-----------|
| **Generated column** (PostgreSQL `GENERATED ALWAYS AS`) | Công thức đơn giản, deterministic | Auto update, nhưng không reference field ở bảng khác |
| **Trigger-based** | Công thức phức tạp, có side-effect | Linh hoạt, nhưng tốn trigger overhead |
| **On-read** | Formula phụ thuộc external (AI score) | Linh hoạt nhất, nhưng chậm nếu list page |
| **Materialized column** + cron refresh | Dashboard/report | Pre-compute, dashboard load nhanh |

Default cho RINCO: **on-read** với cache Valkey TTL 60s.

---

## 15. Workflow Engine Nâng Cao

### 15.1. Trigger Execution Mode

```
Trigger Mode Selection:
  - sync:  Block transition cho tới khi trigger xong (max 2s, fail → rollback)
  - async_queue: Đẩy NATS event, transition thành công ngay
  - async_deferred: Trigger chạy sau 5-60s (batch)
```

### 15.2. Sequence Diagram – Workflow Transition

```
User       CRM Service    Workflow Engine    Trigger Worker     NATS       External API
 │              │                │                  │               │              │
 │ PATCH lead   │                │                  │               │              │
 │ {status:     │                │                  │               │              │
 │  "won"}      │                │                  │               │              │
 │─────────────►│                │                  │               │              │
 │              │ validate body  │                  │               │              │
 │              │ (JSON Schema)  │                  │               │              │
 │              │───────────────►│                  │               │              │
 │              │                │ Check permission │               │              │
 │              │                │ (RBAC)           │               │              │
 │              │                │ Check CEL        │               │              │
 │              │                │ condition        │               │              │
 │              │                │ (score >= 70)    │               │              │
 │              │                │                  │               │              │
 │              │                │ Begin transaction                 │              │
 │              │                │ ───────────────►│               │              │
 │              │                │ UPDATE status   │               │              │
 │              │                │ INSERT audit    │               │              │
 │              │                │                  │               │              │
 │              │                │ Enqueue trigger │               │              │
 │              │                │ (async mode)    │               │              │
 │              │                │─────────────────►               │              │
 │              │                │                  │ Publish event │              │
 │              │                │                  │──────────────►│              │
 │              │                │ Commit                          │              │
 │              │                │ ───────────────►                │              │
 │              │ Return 200     │                  │               │              │
 │              │◄──────────────│                  │               │              │
 │◄─────────────│                │                  │               │              │
 │              │                │                  │ Consume event │              │
 │              │                │                  │──────────────►│              │
 │              │                │                  │               │ POST webhook│
 │              │                │                  │               │────────────►│
 │              │                │                  │               │ 200 OK      │
 │              │                │                  │               │◄────────────│
```

### 15.3. Workflow Engine Implementation (Go)

```go
// services/workflow-engine/internal/executor/executor.go
package executor

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/sony/gobreaker"

    "github.com/itdoanh/rinco/workflow-engine/internal/cel"
    "github.com/itdoanh/rinco/workflow-engine/internal/events"
    "github.com/itdoanh/rinco/workflow-engine/internal/repository"
    "github.com/itdoanh/rinco/workflow-engine/internal/rbac"
)

type Executor struct {
    repo      *repository.WorkflowRepo
    celEnv    *cel.Env
    rbac      *rbac.Service
    publisher *events.Publisher
    cb        *gobreaker.CircuitBreaker
}

type TransitionRequest struct {
    TenantID    string                 `json:"tenant_id"`
    EntityCode  string                 `json:"entity_code"`
    RecordID    uuid.UUID              `json:"record_id"`
    FromState   string                 `json:"from_state"`
    ToState     string                 `json:"to_state"`
    ActorUserID uuid.UUID              `json:"actor_user_id"`
    Payload     map[string]interface{} `json:"payload"`
    TraceID     string                 `json:"trace_id"`
}

type TransitionResult struct {
    Success     bool          `json:"success"`
    NewState    string        `json:"new_state"`
    TriggerIDs  []uuid.UUID   `json:"trigger_ids"`
    DurationMS  int64         `json:"duration_ms"`
    Error       string        `json:"error,omitempty"`
}

func (e *Executor) Execute(ctx context.Context, req TransitionRequest) (*TransitionResult, error) {
    traceID := req.TraceID
    log := log.With("trace_id", traceID, "tenant_id", req.TenantID, "entity", req.EntityCode, "record_id", req.RecordID)

    // 1. Load workflow definition
    wf, err := e.repo.GetActiveWorkflow(ctx, req.TenantID, req.EntityCode)
    if err != nil {
        log.Error("failed to load workflow", "error", err)
        return nil, fmt.Errorf("workflow not found: %w", err)
    }

    // 2. Find matching transition
    transition, err := wf.FindTransition(req.FromState, req.ToState)
    if err != nil {
        log.Warn("transition not allowed", "from", req.FromState, "to", req.ToState)
        return nil, fmt.Errorf("transition %s -> %s not defined", req.FromState, req.ToState)
    }

    // 3. Check permission
    if transition.Permission != "" {
        ok, err := e.rbac.HasPermission(ctx, req.ActorUserID, req.TenantID, transition.Permission)
        if err != nil {
            return nil, fmt.Errorf("rbac check: %w", err)
        }
        if !ok {
            return nil, ErrPermissionDenied
        }
    }

    // 4. Check CEL condition
    if transition.Condition != "" {
        ok, err := e.celEnv.EvaluateBool(ctx, transition.Condition, req.Payload)
        if err != nil {
            log.Error("CEL eval failed", "expression", transition.Condition, "error", err)
            return nil, fmt.Errorf("CEL eval: %w", err)
        }
        if !ok {
            return nil, ErrConditionNotMet
        }
    }

    // 5. Begin DB transaction
    tx, err := e.repo.BeginTx(ctx)
    if err != nil {
        return nil, fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback()

    // 6. Update state
    if err := e.repo.UpdateState(ctx, tx, req.EntityCode, req.RecordID, req.ToState); err != nil {
        return nil, fmt.Errorf("update state: %w", err)
    }

    // 7. Insert audit log
    auditID := uuid.NewV7()
    if err := e.repo.InsertAuditLog(ctx, tx, auditID, req, transition); err != nil {
        return nil, fmt.Errorf("insert audit: %w", err)
    }

    // 8. Enqueue triggers (async via NATS)
    var triggerIDs []uuid.UUID
    for _, trig := range transition.Triggers {
        tid := uuid.NewV7()
        triggerIDs = append(triggerIDs, tid)

        event := events.TriggerEvent{
            TriggerID:   tid,
            TenantID:    req.TenantID,
            EntityCode:  req.EntityCode,
            RecordID:    req.RecordID,
            TriggerType: trig.Type,
            Config:      trig.Config,
            TraceID:     traceID,
            EnqueuedAt:  time.Now().UnixMilli(),
        }
        // Sử dụng circuit breaker cho publisher
        if err := e.cb.Execute(func() error {
            return e.publisher.PublishTrigger(ctx, event)
        }); err != nil {
            log.Warn("trigger enqueue failed, will retry async", "trigger_id", tid, "error", err)
            // Không fail transition vì trigger là async
            go e.retryPublish(event)
        }
    }

    // 9. Commit
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit: %w", err)
    }

    return &TransitionResult{
        Success:    true,
        NewState:   req.ToState,
        TriggerIDs: triggerIDs,
    }, nil
}

// retryPublish retry với exponential backoff khi circuit breaker mở
func (e *Executor) retryPublish(event events.TriggerEvent) {
    backoff := 1 * time.Second
    for i := 0; i < 5; i++ {
        time.Sleep(backoff)
        backoff *= 2
        if err := e.publisher.PublishTrigger(context.Background(), event); err == nil {
            return
        }
    }
    // Sau 5 lần retry → đẩy vào dead-letter queue
    _ = e.publisher.PublishToDLQ(event)
}
```

### 15.4. Workflow Undo / Rollback

Workflow rollback dùng "compensating transition" - mỗi transition có thể khai báo 1 `compensating_transition` ngược lại:

```json
{
  "from": "new",
  "to": "contacted",
  "compensating": "undo_to_new",
  "compensating_permission": "lead.undo_contact"
}
```

Rollback giới hạn:
- Chỉ rollback transition trong **24h**.
- Không rollback nếu có child record đã được tạo.
- Rollback audit log riêng `rollback_audit_log`.

### 15.5. Sub-workflow giới hạn

```
MAX_WORKFLOW_DEPTH = 5  # Recursion limit
MAX_TRIGGERS_PER_TRANSITION = 10
MAX_TIME_WORKFLOW_INSTANCE = 90 days  # Auto expire nếu stuck
```

### 15.6. Workflow Simulation Mode

```bash
# CLI: workflow simulate
$ rincoctl workflow simulate --tenant apex-fintech --entity lead \
    --record-id abc-def \
    --to-state qualified \
    --actor user:manager-001

# Output:
✓ Permission check: PASS (lead.qualify)
✓ CEL condition: PASS (score=75 >= 70)
✓ Triggers would fire:
  - send_notification (channel: email)
  - create_activity (type: qualification)
  - webhook (url: https://crm.apex.vn/hooks/qualified)
✓ Would update last 3 records in 250ms
```

---

## 16. Cross-field & Cross-entity Validation

### 16.1. CEL Sandbox Configuration

```go
// services/validation-engine/internal/cel/sandbox.go
package cel

import (
    "time"
    "github.com/google/cel-go/cel"
    "github.com/google/cel-go/common/types"
)

type SandboxConfig struct {
    MaxExpressionLen int           // 4096 chars
    MaxEvalTime      time.Duration // 100ms per expression
    MaxRecursionDepth int          // 10
    MaxAllocBytes    int           // 64KB per eval
    EnableLoops      bool          // false (chống DoS)
}

var DefaultSandboxConfig = SandboxConfig{
    MaxExpressionLen:  4096,
    MaxEvalTime:       100 * time.Millisecond,
    MaxRecursionDepth: 10,
    MaxAllocBytes:     65536,
    EnableLoops:       false,
}

// Sử dụng cel-go với custom cost tracker
func NewSandboxedEnv(cfg SandboxConfig) (*cel.Env, error) {
    envOpts := []cel.EnvOption{
        cel.Variable("record", cel.DynType),
        cel.Variable("user", cel.DynType),
        cel.Variable("tenant", cel.DynType),
        cel.Variable("now", cel.TimestampType),
        cel.OptionalTypes(),
        cel.CrossTypeNumericComparisons(true),
    }
    // Thêm custom functions
    envOpts = append(envOpts,
        cel.Function("subtree_of",
            cel.Overload("string_subtree_of_string",
                []*cel.Type{cel.StringType, cel.StringType},
                cel.BoolType)),
        cel.Function("today_vn",
            cel.Overload("today_vn", []*cel.Type{}, cel.StringType)),
    )

    return cel.NewEnv(envOpts...)
}
```

### 16.2. Cross-entity Rule Engine

Rule có thể reference entity khác, ví dụ:
```
"won_deal_value_must_match_quote":
  when: state == "won"
  check: |
    related_deal.actual_value == related_quote.total_amount
  error_message: "Giá trị deal thắng phải khớp với báo giá"
```

```go
type CrossEntityRule struct {
    Code         string                 `json:"code"`
    EntityCode   string                 `json:"entity_code"`
    Trigger      string                 `json:"trigger"`     // "create" | "update" | "transition"
    Condition    string                 `json:"condition"`   // CEL
    References   []EntityReference      `json:"references"`
    ErrorMessage string                 `json:"error_message"`
    Severity     string                 `json:"severity"`    // "error" | "warning"
}

type EntityReference struct {
    EntityCode    string `json:"entity_code"`
    RelationField string `json:"relation_field"`  // field FK trên entity hiện tại
    Fields        []string `json:"fields"`         // fields cần load
}

// Engine load reference qua repository, cache trong Valkey TTL 30s
func (e *Engine) EvaluateCrossRules(ctx context.Context, recordID uuid.UUID, payload map[string]interface{}) ([]ValidationError, error) {
    var errors []ValidationError
    for _, rule := range e.rules {
        // Load referenced entities (cached)
        env := map[string]interface{}{
            "record": payload,
            "user":   ctxUser(ctx),
            "tenant": ctxTenant(ctx),
            "now":    time.Now(),
        }
        for _, ref := range rule.References {
            refRecord, err := e.repo.LoadRelated(ctx, ref.EntityCode, ref.RelationField, recordID, ref.Fields)
            if err != nil {
                return nil, fmt.Errorf("load %s: %w", ref.EntityCode, err)
            }
            env[ref.EntityCode] = refRecord
        }
        // Evaluate
        ok, err := e.cel.EvaluateBool(ctx, rule.Condition, env)
        if err != nil {
            return nil, fmt.Errorf("rule %s: %w", rule.Code, err)
        }
        if !ok {
            errors = append(errors, ValidationError{
                Rule:    rule.Code,
                Field:   "_root",
                Message: rule.ErrorMessage,
                Severity: rule.Severity,
            })
        }
    }
    return errors, nil
}
```

### 16.3. Validation Pipeline Order

```go
func (s *ValidationService) Validate(ctx context.Context, req ValidateRequest) (*ValidateResult, error) {
    // Layer 1: Schema-level
    schema, err := s.repo.LoadSchema(ctx, req.TenantID, req.EntityCode)
    if err != nil {
        return nil, fmt.Errorf("load schema: %w", err)
    }
    if err := s.jsonschema.Validate(ctx, schema.JSONSchema, req.Payload); err != nil {
        return &ValidateResult{Ok: false, Errors: toFieldErrors(err)}, nil
    }

    // Layer 2: Field-level CEL
    var warnings []ValidationError
    for _, field := range schema.Fields {
        if field.CELValidation != "" {
            ok, err := s.cel.EvaluateBool(ctx, field.CELValidation, req.Payload)
            if err != nil {
                return nil, fmt.Errorf("field %s cel: %w", field.Code, err)
            }
            if !ok {
                warnings = append(warnings, ValidationError{
                    Field: field.Code, Message: field.ValidationMessage, Severity: "warning",
                })
            }
        }
    }

    // Layer 3: Record-level CEL
    for _, rule := range schema.RecordRules {
        ok, err := s.cel.EvaluateBool(ctx, rule.Condition, req.Payload)
        if err != nil {
            return nil, fmt.Errorf("record rule %s: %w", rule.Code, err)
        }
        if !ok {
            return &ValidateResult{Ok: false, Errors: []ValidationError{{
                Rule: rule.Code, Message: rule.ErrorMessage, Severity: rule.Severity,
            }}}, nil
        }
    }

    // Layer 4: Uniqueness (DB)
    for _, field := range schema.Fields {
        if field.Unique {
            exists, err := s.repo.CheckUnique(ctx, req.TenantID, req.EntityCode, field.Code, req.Payload[field.Code])
            if err != nil {
                return nil, fmt.Errorf("unique check %s: %w", field.Code, err)
            }
            if exists {
                return &ValidateResult{Ok: false, Errors: []ValidationError{{
                    Field: field.Code, Message: fmt.Sprintf("%s đã tồn tại", field.Label),
                }}}, nil
            }
        }
    }

    // Layer 5: Cross-entity
    crossErrors, err := s.crossRules.EvaluateCrossRules(ctx, req.RecordID, req.Payload)
    if err != nil {
        return nil, fmt.Errorf("cross rules: %w", err)
    }
    for _, e := range crossErrors {
        if e.Severity == "error" {
            return &ValidateResult{Ok: false, Errors: []ValidationError{e}}, nil
        }
        warnings = append(warnings, e)
    }

    return &ValidateResult{Ok: true, Warnings: warnings}, nil
}
```

### 16.4. Validation Result Format

```json
{
  "ok": false,
  "errors": [
    {
      "rule": "phone_format",
      "field": "phone",
      "message": "Số điện thoại không đúng định dạng Việt Nam",
      "severity": "error",
      "constraint": "pattern",
      "expected": "^(\\+84|0)\\d{9,10}$"
    }
  ],
  "warnings": [
    {
      "field": "score",
      "message": "Score thấp, khả năng chuyển đổi dưới 30%",
      "severity": "warning"
    }
  ]
}
```

---

## 17. Hot Reload & Schema Versioning

### 17.1. Concurrent Edit Resolution (Optimistic Locking)

Mỗi schema entity có field `version` (BIGINT) và `etag` (UUID v7).

```http
PATCH /api/dm/v1/entities/lead
If-Match: "0190a5b3-7c1e-7000-8000-123456789abc"  # etag của version hiện tại

{
  "fields": [...],
  "expected_version": 12
}
```

Nếu server thấy version ≠ expected_version → trả 409 Conflict + thông tin version hiện tại:

```json
{
  "error": "conflict",
  "code": "SCHEMA_VERSION_CONFLICT",
  "current_version": 13,
  "your_version": 12,
  "diff_summary": "fields.email.validation đã thay đổi"
}
```

Client phải fetch lại schema, merge changes manually, rồi retry.

### 17.2. Schema Change Audit

```sql
CREATE TABLE schema_change_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  entity_code TEXT NOT NULL,
  schema_version INT NOT NULL,
  change_type TEXT NOT NULL,  -- 'create' | 'update_field' | 'delete_field' | 'migrate'
  changed_fields JSONB,       -- chi tiết thay đổi
  diff JSONB,                 -- full diff vs previous version
  actor_user_id UUID NOT NULL,
  trace_id UUID NOT NULL,
  applied_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_schema_change_tenant ON schema_change_log(tenant_id, entity_code, applied_at DESC);
```

### 17.3. Hot Reload via NATS Pub/Sub

Khi admin save schema mới:
```
[dynamic-model service]
        │
        ├──► UPDATE schema SET version = 13, ...
        ├──► INSERT schema_change_log
        │
        └──► PUBLISH NATS:
            Subject: schema.{tenant_id}.{entity_code}.updated
            Payload: { version: 13, etag: "...", changed_fields: [...] }
                    │
                    ▼
        [crm-core service]      [chat-engine service]     [landing-ingest service]
        Invalidate cache        Reload entity rules        Reload form schema
        Reload workflow         ...
```

### 17.4. Cache Invalidation Strategy

```go
type SchemaCache struct {
    valkey *valkey.Client
    local *ristretto.Cache  // L1 cache in-process
    ttl   time.Duration
}

func (c *SchemaCache) Get(ctx context.Context, tenantID, entityCode string) (*Schema, error) {
    // L1 cache
    if v, ok := c.local.Get(fmt.Sprintf("%s:%s", tenantID, entityCode)); ok {
        return v.(*Schema), nil
    }
    // L2 valkey
    key := fmt.Sprintf("schema:%s:%s", tenantID, entityCode)
    raw, err := c.valkey.Get(ctx, key).Result()
    if err == nil {
        var s Schema
        json.Unmarshal([]byte(raw), &s)
        c.local.Set(key, &s, 1)
        return &s, nil
    }
    // L3 database
    s, err := c.repo.LoadSchema(ctx, tenantID, entityCode)
    if err != nil {
        return nil, err
    }
    bytes, _ := json.Marshal(s)
    c.valkey.Set(ctx, key, bytes, c.ttl)
    c.local.Set(key, s, 1)
    return s, nil
}
```

---

## 18. Code Generation Pipeline chi tiết

### 18.1. Khi nào trigger Code-gen?

Tiêu chí schema "stable":
```
STABLE_CRITERIA = all([
    schema_age > 7 days,
    no_changes_in_last_3_days,
    total_changes_in_lifetime > 5,  # đã mature
    daily_record_count > 1000,     # traffic đủ lớn
    field_count > 8,                # đủ phức tạp để worth gen
    error_rate < 0.1%               # schema không bị validation error nhiều
])
```

### 18.2. Pipeline Stages

```
[1] Detect stable schema (cron hourly)
        │
        ▼
[2] Snapshot schema JSON → Git repo (tagged commit)
        │
        ▼
[3] Trigger CI/CD pipeline "schema-codegen-{entity_code}"
        │
        ├──► Run sqlc generate
        ├──► Run ent generate
        ├──► Run protoc → ts-proto
        │
        ▼
[4] Build new binary: crm-core-{version}-ent-{entity}
        │
        ▼
[5] Push to ghcr.io/itdoanh/rinco/crm-core:{version}-ent-{entity}
        │
        ▼
[6] Canary deploy: 5% traffic 30 phút
        │
        ▼
[7] Nếu metrics OK → Rollout 100%
```

### 18.3. Ent Code-gen Output Example

Cho entity `lead`:
```go
// services/crm-core/ent/schema/lead.go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type Lead struct {
    ent.Schema
}

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
        field.Time("created_at").Default(time.Now).Immutable(),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
    }
}

func (Lead) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("tenant_id", "phone").Unique(),
        index.Fields("tenant_id", "status", "owner_id"),
        index.Fields("tenant_id", "score").StorageKey("idx_leads_score"),
    }
}
```

### 18.4. TypeScript Type Generation

```typescript
// services/crm-core/ui/types/lead.ts
// Auto-generated từ schema
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
  custom_fields?: {
    budget?: number;
    interest?: 'buy' | 'rent' | 'invest';
    preferred_location?: string;
  };
  created_at: string;  // ISO 8601
  updated_at: string;
}

export const LeadValidation = {
  name: { required: true, minLength: 1, maxLength: 255 },
  phone: { pattern: /^(\+84|0)\d{9,10}$/ },
  email: { format: 'email' },
  score: { min: 0, max: 100 },
} as const;
```

---

## 19. Database Strategy nâng cao

### 19.1. Auto Index Recommendation Engine

```sql
-- Track slow queries
CREATE TABLE slow_query_log (
  id BIGSERIAL PRIMARY KEY,
  tenant_id UUID NOT NULL,
  query_hash TEXT NOT NULL,
  query_text TEXT NOT NULL,
  execution_ms INT NOT NULL,
  rows_examined BIGINT,
  occurred_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_sq_tenant_hash ON slow_query_log(tenant_id, query_hash, occurred_at DESC);

-- Recommendation worker (cron hourly)
```

```python
# services/admin/recommender/index_recommender.py
import hashlib
import re
from collections import defaultdict

class IndexRecommender:
    def __init__(self, db, valkey):
        self.db = db
        self.valkey = valkey
        self.min_calls = 50        # chỉ recommend nếu query chạy > 50 lần/giờ
        self.threshold_ms = 100    # và latency > 100ms

    def recommend(self, tenant_id):
        rows = self.db.query("""
            SELECT query_hash, query_text, AVG(execution_ms) as avg_ms, COUNT(*) as cnt
            FROM slow_query_log
            WHERE tenant_id = %s AND occurred_at > NOW() - INTERVAL '1 hour'
            GROUP BY query_hash, query_text
            HAVING COUNT(*) > %s AND AVG(execution_ms) > %s
        """, (tenant_id, self.min_calls, self.threshold_ms))

        recommendations = []
        for r in rows:
            # Extract WHERE columns từ SQL
            cols = self.extract_columns(r.query_text, "WHERE")
            if not cols:
                continue
            rec = {
                "query_hash": r.query_hash,
                "avg_ms": r.avg_ms,
                "calls_per_hour": r.cnt,
                "suggested_index": f"CREATE INDEX CONCURRENTLY idx_{r.query_hash[:8]} ON {self.extract_table(r.query_text)} ({','.join(cols)})",
                "estimated_speedup": "3-10x",
            }
            recommendations.append(rec)
        return recommendations

    def extract_columns(self, sql, clause):
        # Parse SQL cơ bản - dùng pgsql parser cho chính xác
        pattern = rf"{clause}\s+(.*?)(?:\s+(?:AND|ORDER BY|GROUP BY|LIMIT|$))"
        match = re.search(pattern, sql, re.IGNORECASE | re.DOTALL)
        if not match:
            return []
        cols = re.findall(r"(\w+)\s*[=<>]", match.group(1))
        return list(set(cols))
```

Recommendation hiển thị trong Admin Dashboard:
> ⚠️ Query `SELECT * FROM leads WHERE tenant_id = ? AND custom_fields->>'budget' > ?` chạy 230 lần/giờ, avg 145ms. Đề xuất: `CREATE INDEX CONCURRENTLY idx_leads_budget ON leads ((data->>'budget')::numeric)`. Expected speedup: 5x.

### 19.2. JSONB Indexing Patterns

```sql
-- 1. GIN full-text (mặc định)
CREATE INDEX idx_leads_data_gin ON leads USING GIN (data);

-- 2. Expression index cho 1 hot field
CREATE INDEX idx_leads_budget ON leads ((data->>'budget')::numeric);

-- 3. B-tree cho enum hot
CREATE INDEX idx_leads_city ON leads ((data->>'city')) WHERE data->>'city' IS NOT NULL;

-- 4. Composite JSONB
CREATE INDEX idx_leads_composite ON leads (
  tenant_id,
  (data->>'city'),
  (data->>'budget')::numeric
);

-- 5. Partial index cho specific value
CREATE INDEX idx_leads_hot ON leads (tenant_id, score)
  WHERE (data->>'is_hot')::boolean = true;
```

### 19.3. Storage Mode Decision Tree

```
                    ┌─ Khai báo schema
                    │
                    ▼
        Schema "stable" với 100% fields hot?
                    │
        ┌───────────┴───────────┐
        │ YES                   │ NO
        ▼                       ▼
   PostgreSQL              PostgreSQL JSONB
   relational columns      + JSONB column 'data'
   + indexes              + GIN index
        │                       │
        │ < 50,000 records      │ > 50,000 records
        │ same structure        │ schema drift > 30%
        ▼                       ▼
   Stable mode             Drift detection ON
                          │
                          ▼
                  Drift > 50% sustained 30 ngày?
                          │
                          ▼
                  Migrate to MongoDB
                  (background process)
```

### 19.4. Schema Drift Detection

```python
# services/dynamic-model/internal/drift/detector.py
class DriftDetector:
    def __init__(self, postgres_conn, mongo_conn, valkey):
        self.pg = postgres_conn
        self.mongo = mongo_conn
        self.valkey = valkey

    def compute_drift_score(self, tenant_id, entity_code, window_days=30):
        # Lấy sample 1000 records gần nhất
        samples = self.pg.query("""
            SELECT jsonb_object_keys(data) as field_name
            FROM leads
            WHERE tenant_id = %s
              AND created_at > NOW() - INTERVAL '%s days'
        """, (tenant_id, window_days))

        # Count frequency mỗi field
        field_freq = defaultdict(int)
        total = 0
        for row in samples:
            for field in row['data'].keys() if row['data'] else []:
                field_freq[field] += 1
            total += 1

        # Drift score = (số field xuất hiện < 50% records) / tổng số field
        if not field_freq:
            return 0.0
        rare_fields = sum(1 for c in field_freq.values() if c < total * 0.5)
        drift = rare_fields / len(field_freq)
        return drift

    def recommend_migration(self, tenant_id, entity_code):
        score = self.compute_drift_score(tenant_id, entity_code)
        if score > 0.5:
            return {
                "action": "MIGRATE_TO_MONGODB",
                "reason": f"Drift score {score:.2%} > 50%",
                "estimate_records": self.pg.query_scalar(
                    "SELECT COUNT(*) FROM leads WHERE tenant_id=%s", (tenant_id,)),
                "downtime_estimate_minutes": "30-60",
            }
        return None
```

---

## 20. Industry Templates chi tiết

Mỗi template được lưu trong file JSON riêng trong repo `services/dynamic-model/templates/{industry}.json` và seed vào database khi user chọn.

### 20.1. Template: Bất động sản

```json
{
  "template_code": "real_estate",
  "display_name": "Bất động sản",
  "description": "CRM cho sàn BĐS, môi giới, dự án",
  "icon": "lucide:home",
  "color": "#0ea5e9",
  "entities": [
    {
      "code": "lead",
      "display_name": "Khách hàng quan tâm",
      "icon": "lucide:user-plus",
      "fields": [
        { "code": "name", "label": "Họ tên", "type": "string", "required": true },
        { "code": "phone", "label": "Số điện thoại", "type": "phone", "required": true, "unique": true },
        { "code": "email", "label": "Email", "type": "email" },
        { "code": "budget_min", "label": "Ngân sách tối thiểu", "type": "currency" },
        { "code": "budget_max", "label": "Ngân sách tối đa", "type": "currency" },
        { "code": "preferred_locations", "label": "Khu vực quan tâm", "type": "multi_select", "options": [] },
        { "code": "property_type", "label": "Loại BĐS", "type": "enum", "options": ["apartment", "house", "land", "shophouse", "villa"] },
        { "code": "bedrooms", "label": "Số phòng ngủ", "type": "integer", "min": 0, "max": 10 },
        { "code": "bathrooms", "label": "Số phòng tắm", "type": "integer", "min": 0, "max": 10 },
        { "code": "interest", "label": "Mục đích", "type": "enum", "options": ["buy", "rent", "invest"] },
        { "code": "move_in_date", "label": "Ngày dự kiến dọn vào", "type": "date" },
        { "code": "financing_needed", "label": "Cần hỗ trợ tài chính", "type": "boolean" }
      ]
    },
    {
      "code": "viewing",
      "display_name": "Lịch xem nhà",
      "fields": [
        { "code": "lead_id", "type": "reference", "ref_entity": "lead" },
        { "code": "property_id", "type": "reference", "ref_entity": "property" },
        { "code": "scheduled_at", "type": "datetime", "required": true },
        { "code": "agent_id", "type": "reference", "ref_entity": "user" },
        { "code": "notes", "type": "text" }
      ]
    },
    {
      "code": "property",
      "display_name": "Bất động sản",
      "fields": [
        { "code": "title", "label": "Tiêu đề", "type": "string", "required": true },
        { "code": "address", "type": "string", "required": true },
        { "code": "lat_lng", "type": "geo" },
        { "code": "area_m2", "type": "number" },
        { "code": "price", "type": "currency", "required": true },
        { "code": "photos", "type": "images" }
      ]
    },
    {
      "code": "deal",
      "display_name": "Giao dịch",
      "fields": [
        { "code": "lead_id", "type": "reference", "ref_entity": "lead" },
        { "code": "property_id", "type": "reference", "ref_entity": "property" },
        { "code": "deal_value", "type": "currency", "required": true },
        { "code": "commission", "type": "currency" },
        { "code": "signed_at", "type": "date" }
      ]
    }
  ],
  "workflows": [
    {
      "code": "lead_lifecycle_re",
      "entity": "lead",
      "states": ["new", "contacted", "viewing_scheduled", "viewed", "negotiating", "won", "lost"],
      "transitions": [
        { "from": "new", "to": "contacted", "trigger": "create_activity" },
        { "from": "contacted", "to": "viewing_scheduled", "trigger": "send_notification" },
        { "from": "viewing_scheduled", "to": "viewed", "trigger": "create_activity" },
        { "from": "viewed", "to": "negotiating", "trigger": "send_notification" },
        { "from": "negotiating", "to": "won", "condition": "deal_value > 0", "trigger": "notify_accounting" }
      ]
    }
  ],
  "sample_data_csv": "templates/real_estate/sample_leads.csv"
}
```

### 20.2. Template: Tài chính / Tín dụng

Workflow có approval chain đặc biệt:
```
new → pre_qualified (auto) → docs_collected → under_review →
   approved → contract_signed → disbursed
                  ↓
                rejected (terminate)
```

Field đặc biệt:
- `credit_score` (lookup từ CIC API)
- `kyc_status` (enum: pending, verified, rejected)
- `collateral_value` (currency)
- `monthly_income` (currency)
- `debt_to_income_ratio` (formula)

### 20.3. Template: Giáo dục

Entity `student` có sub-entity `enrollment` (n-n với `course`).
Workflow:
```
new → contacted → trial_booked → trialed → enrolled
                       ↓
                  no_show → nurture (CRM sequence 30 ngày)
```

### 20.4. Template: Bán lẻ / E-commerce

Tích hợp với Shopify/WooCommerce:
- `order_count` (rollup từ bảng orders)
- `lifetime_value` (rollup sum)
- `last_purchase_date` (rollup max)
- `preferred_category` (machine learning predicted)

Workflow:
```
visitor → lead → first_purchase → repeat_customer → vip
```

### 20.5. Template: Marketing Agency

Entity `client` (công ty khách hàng của agency):
- `monthly_retainer` (currency)
- `services_subscribed` (multi_select)
- `health_score` (computed: weighted from activity)
- `churn_risk` (formula based on last contact + NPS)

### 20.6. Template Import Flow

```bash
# CLI install template
$ rincoctl template install real_estate --tenant apex-fintech

# Output:
✓ Loaded template from templates/real_estate.json (4 entities, 1 workflow, 5 relations)
✓ Created entity: lead (12 fields)
✓ Created entity: viewing (5 fields)
✓ Created entity: property (6 fields)
✓ Created entity: deal (5 fields)
✓ Created workflow: lead_lifecycle_re (7 states, 6 transitions)
✓ Imported 250 sample records from templates/real_estate/sample_leads.csv
✓ Custom views created: "Hot Leads", "This Week's Viewings"

Done in 4.2s.
```

---

## 21. Migration & Backfill

### 21.1. Zero-downtime Schema Migration

```sql
-- Phase 1: Add nullable column (no lock)
ALTER TABLE leads ADD COLUMN new_phone_format TEXT;

-- Phase 2: Backfill in batches với progress tracking
DO $$
DECLARE
    last_id UUID := NULL;
    batch_size INT := 5000;
    rows_updated INT;
    total_updated INT := 0;
BEGIN
    LOOP
        UPDATE leads
        SET new_phone_format = regexp_replace(phone, '^0', '+84')
        WHERE id > COALESCE(last_id, '00000000-0000-0000-0000-000000000000'::UUID)
          AND new_phone_format IS NULL
        ORDER BY id
        LIMIT batch_size
        RETURNING id INTO last_id;

        GET DIAGNOSTICS rows_updated = ROW_COUNT;
        total_updated := total_updated + rows_updated;

        -- Log progress
        RAISE NOTICE 'Backfilled % rows, total %', rows_updated, total_updated;

        EXIT WHEN rows_updated = 0;

        -- Yield to other queries
        PERFORM pg_sleep(0.05);

        -- Commit batch
        COMMIT;
    END LOOP;
END $$;

-- Phase 3: Application reads new column (deploy code)
-- (Đợi 24-48h cho cache invalidate + monitoring)

-- Phase 4: Drop old column (lock ngắn ~ms)
ALTER TABLE leads DROP COLUMN phone;

-- Phase 5: Rename new column
ALTER TABLE leads RENAME COLUMN new_phone_format TO phone;
```

### 21.2. Migration Lock Timeout

```sql
-- Trước mỗi migration DDL, set timeout
SET lock_timeout = '5s';
SET statement_timeout = '30s';
SET idle_in_transaction_session_timeout = '60s';
```

### 21.3. Rollback Strategy

Mỗi migration phải có file `down.sql`:

```sql
-- migrations/20260907_001_add_phone_format/up.sql
ALTER TABLE leads ADD COLUMN new_phone_format TEXT;
-- (backfill chạy ở Go code, không trong SQL file)

-- migrations/20260907_001_add_phone_format/down.sql
ALTER TABLE leads DROP COLUMN IF EXISTS new_phone_format;
```

### 21.4. Goose Migration Tool Configuration

```go
// services/crm-core/migrations/main.go
package main

import (
    "database/sql"
    "embed"
    "log"

    "github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedMigrations embed.FS

func RunMigrations(db *sql.DB) error {
    goose.SetBaseFS(embedMigrations)
    goose.SetLogger(goose.NopLogger())  // dùng slog thay thế
    if err := goose.SetDialect("postgres"); err != nil {
        return err
    }
    return goose.Up(db, ".")
}
```

### 21.5. Auto-Migration Service

Service riêng `dynamic-model-migrator` (Python + Celery) chạy background:
```python
# services/dynamic-model-migrator/migrator.py
class SchemaMigrator:
    def detect_drift(self, tenant_id):
        """Phát hiện schema thay đổi từ dynamic-model"""
        current = self.pg.get_schema(tenant_id)
        applied = self.valkey.get(f"applied_schema:{tenant_id}")
        if current.version > applied.version:
            return self.generate_migration(current, applied)

    def generate_migration(self, new_schema, old_schema):
        changes = self.diff(old_schema, new_schema)
        return {
            "tenant_id": new_schema.tenant_id,
            "from_version": old_schema.version,
            "to_version": new_schema.version,
            "sql_up": self.generate_sql_up(changes),
            "sql_down": self.generate_sql_down(changes),
            "data_backfill": self.generate_backfill(changes),
            "estimated_duration_minutes": self.estimate_duration(changes),
            "risk_level": self.compute_risk(changes),
        }

    def apply_migration(self, migration):
        if migration.risk_level == "HIGH":
            # Yêu cầu admin approval
            self.notify_admin(migration)
            return "PENDING_APPROVAL"

        # Run backfill in batches
        for batch in migration.data_backfill:
            self.apply_batch(batch)
        # Apply DDL
        self.apply_ddl(migration.sql_up)
        # Mark applied
        self.valkey.set(f"applied_schema:{migration.tenant_id}", migration.to_version)
```

---

## 22. Performance & Benchmark

### 22.1. Benchmark Targets

| Operation | Target p50 | Target p95 | Target p99 |
|-----------|------------|------------|------------|
| Validate 1 record (10 fields) | < 1ms | < 5ms | < 10ms |
| Validate 1 record (50 fields + 3 CEL rules) | < 5ms | < 20ms | < 50ms |
| Dynamic CRUD create | < 10ms | < 50ms | < 100ms |
| Dynamic CRUD list page (50 records) | < 30ms | < 100ms | < 200ms |
| Workflow transition (sync part) | < 20ms | < 100ms | < 200ms |
| Schema save + cache invalidate | < 50ms | < 200ms | < 500ms |
| Code-gen pipeline (full) | < 5 min | < 10 min | < 30 min |

### 22.2. Validation Performance Test

```go
// services/validation-engine/internal/cel/benchmark_test.go
package cel_test

import (
    "testing"
    "time"

    "github.com/itdoanh/rinco/validation-engine/internal/cel"
)

func BenchmarkCELValidation_Simple(b *testing.B) {
    env, _ := cel.NewSandboxedEnv(cel.DefaultSandboxConfig)
    ast, _ := env.Compile(`score >= 70`)
    program, _ := env.Program(ast)

    payload := map[string]interface{}{
        "score": float64(75),
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := program.Eval(payload)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkCELValidation_Complex(b *testing.B) {
    env, _ := cel.NewSandboxedEnv(cel.DefaultSandboxConfig)
    ast, _ := env.Compile(`
        phone.matches("^(\\+84|0)\\d{9,10}$") &&
        score >= 70 &&
        state == "qualified" &&
        owner_id in subtree_of("user-root-id")
    `)
    program, _ := env.Program(ast)

    payload := map[string]interface{}{
        "phone":   "0987654321",
        "score":   float64(75),
        "state":   "qualified",
        "owner_id": "user-123",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := program.Eval(payload)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Run: go test -bench=. -benchmem -count=3
// Expected:
// BenchmarkCELValidation_Simple-8     5000000    250 ns/op    48 B/op    1 allocs/op
// BenchmarkCELValidation_Complex-8   500000     2800 ns/op   256 B/op    4 allocs/op
```

### 22.3. Load Test with Vegeta

```bash
# Target: 1000 RPS validate API
echo "POST http://localhost:8080/api/crm/v1/lead/validate
Content-Type: application/json
X-Tenant-ID: apex-fintech
X-Trace-ID: 0190a5b3-7c1e-7000-8000-loadtest

{\"name\":\"Test\",\"phone\":\"0987654321\",\"source\":\"facebook\",\"score\":75}
" | vegeta attack -duration=60s -rate=1000 -output=results.bin

vegeta report results.bin
# Expected: p99 < 50ms, success 99.9%
```

### 22.4. Profiling (pprof)

```go
import _ "net/http/pprof"

// Trong main.go
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// Capture CPU profile 30s
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

// Heap
go tool pprof http://localhost:6060/debug/pprof/heap
```

---

## 23. Security & Permission Matrix

### 23.1. Schema Permission Matrix

| Role | View Schema | Edit Schema | Delete Schema | Activate Template |
|------|------------|-------------|---------------|-------------------|
| **Super Admin** | All tenants | Any | Any | Any |
| **Tenant Admin** | Own tenant | Own tenant | Own tenant | Own tenant |
| **Tenant Manager** | Own tenant | Yes (own entities) | No | No |
| **Tenant Staff** | Own tenant | No | No | No |
| **External Auditor** | Own tenant | No | No | No |

### 23.2. Field-Level Permission

```json
{
  "field_code": "phone",
  "permissions": {
    "view": ["owner", "manager", "admin", "tenant_admin"],
    "create": ["owner", "manager", "admin"],
    "update": ["owner", "manager", "admin"],
    "export": ["manager", "admin", "tenant_admin"],  // GDPR: limit export
    "mask_in_list": ["staff"],  // xem được nhưng masked
    "encrypted_at_rest": true,
    "audit_log": true
  }
}
```

### 23.3. Encryption (PII Fields)

- **At rest:** PostgreSQL TDE (Transparent Data Encryption) hoặc column-level với `pgcrypto`.
- **In transit:** TLS 1.3 only.
- **In application:** Mask trước khi log, dùng Redact helper.

```go
// services/crm-core/internal/redact/redact.go
package redact

var piiFields = []string{"password", "phone", "email", "ssn", "credit_card", "id_number"}

func Redact(data map[string]any) map[string]any {
    for _, k := range piiFields {
        if v, ok := data[k]; ok {
            data[k] = "[REDACTED]"
            _ = v
        }
    }
    return data
}

func RedactString(s string) string {
    if len(s) <= 4 {
        return "****"
    }
    return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}
```

### 23.4. Schema Definition Isolation (RLS)

```sql
ALTER TABLE entity_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE entity_definitions FORCE ROW LEVEL SECURITY;

CREATE POLICY entity_def_tenant ON entity_definitions
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Super admin bypass:
CREATE POLICY entity_def_superadmin ON entity_definitions
  USING (current_setting('app.is_super_admin', true) = 'true');
```

### 23.5. GDPR / Right to be Forgotten

```go
func (s *SchemaService) HardDeleteRecord(ctx context.Context, tenantID string, entityCode string, recordID uuid.UUID) error {
    tx, _ := s.db.BeginTx(ctx, nil)
    defer tx.Rollback()
    tx.Exec("SET LOCAL app.current_tenant_id = $1", tenantID)

    // Soft delete trước (30 ngày grace)
    _, err := tx.Exec(`
        UPDATE ${entity_code}
        SET deleted_at = NOW(), anonymized = false
        WHERE id = $1 AND tenant_id = $2
    ``, recordID, tenantID)
    if err != nil {
        return err
    }
    // Schedule hard delete job
    s.scheduler.ScheduleIn(30*24*time.Hour, "hard_delete_record", map[string]interface{}{
        "tenant_id": tenantID, "entity_code": entityCode, "record_id": recordID,
    })
    return tx.Commit()
}

func (s *SchemaService) HardDeleteJob(ctx context.Context, payload map[string]interface{}) error {
    // Anonymize PII thay vì xóa hẳn để giữ foreign key integrity
    tx, _ := s.db.BeginTx(ctx, nil)
    defer tx.Rollback()
    tx.Exec(`
        UPDATE ${payload["entity_code"]}
        SET name = 'ANONYMIZED-' || id,
            phone = NULL,
            email = NULL,
            data = '{}'::jsonb,
            anonymized = true,
            anonymized_at = NOW()
        WHERE id = $1 AND tenant_id = $2
    `, payload["record_id"], payload["tenant_id"])
    return tx.Commit()
}
```

---

## 24. Disaster Recovery

### 24.1. RPO / RTO Targets

| Resource | RPO (data loss) | RTO (recovery time) | Method |
|----------|----------------|---------------------|--------|
| Schema definitions (PostgreSQL) | 5 min (WAL continuous) | 30 min | WAL-G + PITR |
| Workflow definitions | 5 min | 30 min | Same as above |
| Custom views | 1 hour | 30 min | Daily snapshot |
| Industry templates (git) | 0 (git) | 5 min | Git clone |
| Code-gen artifacts | 0 (CI/CD) | 10 min | Re-run pipeline |

### 24.2. Backup Strategy

```yaml
# infra/k8s/services/postgres/backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
spec:
  schedule: "0 2 * * *"  # 2 AM daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: wal-g
            image: ghcr.io/wal-g/wal-g:latest
            command: ["wal-g", "backup-push", "/var/lib/postgresql/data"]
            env:
            - name: WALG_S3_PREFIX
              value: "s3://rinco-backups/postgres"
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef: { name: aws-creds, key: id }
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef: { name: aws-creds, key: secret }
```

### 24.3. Disaster Recovery Drill

Quarterly DR drill:
1. Tạo postgres-replica trong region mới.
2. Stop primary (simulate outage).
3. Promote replica → primary.
4. Verify schema validation hoạt động.
5. RTO measured, log to incident postmortem.

### 24.4. Schema Corruption Recovery

```bash
# Detect corruption
$ rincoctl schema verify --tenant apex-fintech
✗ Entity "lead": foreign key inconsistency
  - references "user.owner_id" but record owner_id="missing-uuid"
✗ Entity "deal": index checksum mismatch

# Auto-repair
$ rincoctl schema repair --tenant apex-fintech --entity lead --auto
✓ Backed up to s3://rinco-backups/repair/20260907-...
✓ Rebuilt foreign key constraints
✓ Recomputed indexes

# Verify
$ rincoctl schema verify --tenant apex-fintech
✓ All 12 entities verified
```

---

## 25. Cost Estimation

### 25.1. Storage Cost per 1,000 Tenants

Giả định:
- 1,000 tenants active
- Trung bình 5 entities / tenant
- Trung bình 50 fields / entity
- Trung bình 100,000 records / entity

**PostgreSQL:**
- Schema definitions: 1000 × 5 × 50 × 1KB = 250 MB (negligible)
- Records: 1000 × 5 × 100,000 × 5KB (with JSONB) = 2.5 TB
- Indexes (10x): 25 TB
- Total: ~28 TB
- AWS RDS db.r6g.4xlarge (500 GB SSD): $1,500/mo × 56 nodes = **$84,000/mo** (over-provisioned for shared)

Nếu shared cluster:
- Single db.r6g.16xlarge (4 TB NVMe): $5,000/mo → effective **$5/tenant/mo**

**ScyllaDB (cho events/audit):**
- Audit log: 1000 tenants × 1M events/mo × 1KB = 1 TB/mo
- ScyllaDB i4i.4xlarge (1.2 TB NVMe): $2,000/mo × 1 → **$2/tenant/mo (shared)**

**ClickHouse (analytics):**
- Aggregated data: 100 GB/mo
- ClickHouse 8-core: $800/mo → **$0.8/tenant/mo (shared)**

**MongoDB (cho schema-drift entities):**
- 20% tenants có drift: 200 tenants × 10GB = 2 TB
- MongoDB M50: $1,200/mo → **$6/tenant/mo (drift tenants only)**

**Valkey (cache):**
- 100 GB RAM: $1,500/mo → **$1.5/tenant/mo (shared)**

**Tổng chi phí storage / 1000 tenants / month:** ~$15/tenant (shared mode).

### 25.2. Compute Cost

| Service | Per-tenant EC2 equiv | Monthly | Cost per tenant |
|---------|---------------------|---------|-----------------|
| dynamic-model (Go) | shared, 4 vCPU | $400 | $0.4 |
| validation-engine (Go) | shared, 8 vCPU | $800 | $0.8 |
| workflow-engine (Go) | shared, 4 vCPU | $400 | $0.4 |
| migration-worker (Python) | shared, 2 vCPU | $200 | $0.2 |

**Total compute:** ~$1.8/tenant/mo (shared).

### 25.3. Cost Optimization Tips

1. **Tiered storage:**
   - Active records → NVMe SSD
   - Archived (>1 year) → S3 IA
   - Cold (>3 years) → S3 Glacier
2. **Index pruning:** Drop indexes không dùng sau 30 ngày.
3. **Code-gen caching:** Cache binary trong ECR, không rebuild.
4. **Reserved Instances:** Commit 1-year cho predictable workloads.
5. **Spot Instances** cho migration workers.

---

## 26. Testing Strategy

### 26.1. Unit Test Pyramid

```
        E2E (10%)
       /          \
      / Integration \
     /    (20%)      \
    /                  \
   /    Unit (70%)      \
  ───────────────────────
```

### 26.2. Unit Test Example

```go
// services/dynamic-model/internal/service/entity_test.go
package service_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "github.com/itdoanh/rinco/dynamic-model/internal/domain"
    "github.com/itdoanh/rinco/dynamic-model/internal/repository/mock"
)

func TestEntityService_Create_ValidPayload(t *testing.T) {
    ctx := context.Background()
    repo := &mock.EntityRepoMock{}
    service := NewEntityService(repo)

    schema := &domain.EntityDefinition{
        Code: "lead",
        TenantID: "apex-fintech",
        Fields: []domain.Field{
            {Code: "name", Type: "string", Required: true},
            {Code: "phone", Type: "phone", Required: true, Validation: map[string]any{
                "pattern": `^(\+84|0)\d{9,10}$`,
            }},
        },
    }
    repo.On("GetEntityByCode", ctx, "apex-fintech", "lead").Return(schema, nil)
    repo.On("InsertRecord", ctx, mock.Anything).Return(uuid.New(), nil)

    payload := map[string]any{
        "name":  "Nguyen Van A",
        "phone": "0987654321",
    }

    id, err := service.Create(ctx, "apex-fintech", "lead", payload)
    assert.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, id)
    repo.AssertExpectations(t)
}

func TestEntityService_Create_InvalidPhone(t *testing.T) {
    ctx := context.Background()
    repo := &mock.EntityRepoMock{}
    service := NewEntityService(repo)

    schema := &domain.EntityDefinition{
        Code: "lead", TenantID: "apex-fintech",
        Fields: []domain.Field{{Code: "phone", Type: "phone", Required: true}},
    }
    repo.On("GetEntityByCode", ctx, "apex-fintech", "lead").Return(schema, nil)

    payload := map[string]any{"phone": "abc-not-phone"}

    id, err := service.Create(ctx, "apex-fintech", "lead", payload)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "phone")
    assert.Equal(t, uuid.Nil, id)
}
```

### 26.3. Integration Test (with testcontainers)

```go
//go:build integration

package service_test

import (
    "context"
    "testing"

    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/stretchr/testify/require"

    "github.com/itdoanh/rinco/dynamic-model/internal/service"
)

func TestEntityService_IntegrationWithPostgres(t *testing.T) {
    ctx := context.Background()

    pgC, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:17"),
        postgres.WithDatabase("rinco_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    require.NoError(t, err)
    defer pgC.Terminate(ctx)

    connStr, _ := pgC.ConnectionString(ctx)
    repo := repository.NewPostgresRepo(connStr)
    service := service.NewEntityService(repo)

    // Run migrations
    require.NoError(t, repo.Migrate())

    // Test full CRUD cycle
    t.Run("create_entity", func(t *testing.T) {
        err := service.CreateEntity(ctx, &domain.EntityDefinition{
            Code: "lead", TenantID: "test-tenant",
            Fields: []domain.Field{{Code: "name", Type: "string"}},
        })
        require.NoError(t, err)
    })

    t.Run("insert_record", func(t *testing.T) {
        id, err := service.Create(ctx, "test-tenant", "lead", map[string]any{"name": "Test"})
        require.NoError(t, err)
        require.NotEqual(t, uuid.Nil, id)
    })
}
```

### 26.4. E2E Test (Playwright)

```typescript
// apps/crm-admin/e2e/schema-builder.spec.ts
import { test, expect } from '@playwright/test';

test('admin creates new entity with custom field', async ({ page, request }) => {
  // Login
  await page.goto('https://admin.rinco.local/login');
  await page.fill('[data-testid=email]', 'admin@apex.vn');
  await page.fill('[data-testid=password]', 'test-password');
  await page.click('[data-testid=submit]');
  await expect(page).toHaveURL(/\/dashboard/);

  // Navigate to Schema Builder
  await page.click('[data-testid=nav-schema]');
  await expect(page.locator('h1')).toContainText('Schema Builder');

  // Create entity
  await page.click('[data-testid=create-entity]');
  await page.fill('[data-testid=entity-code]', 'custom_lead_test');
  await page.fill('[data-testid=entity-label]', 'Custom Lead Test');
  await page.click('[data-testid=entity-save]');

  // Add field
  await page.click('[data-testid=add-field]');
  await page.fill('[data-testid=field-code]', 'custom_field_1');
  await page.fill('[data-testid=field-label]', 'Custom Field 1');
  await page.selectOption('[data-testid=field-type]', 'string');
  await page.click('[data-testid=field-save]');

  // Verify schema saved
  await expect(page.locator('[data-testid=field-custom_field_1]')).toBeVisible();

  // Verify via API
  const res = await request.get('/api/dm/v1/entities/custom_lead_test', {
    headers: { 'X-Tenant-ID': 'apex-fintech' }
  });
  expect(res.ok()).toBeTruthy();
  const json = await res.json();
  expect(json.fields).toHaveLength(2);
});
```

### 26.5. Load Test (k6)

```javascript
// tests/load/dynamic-model.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

export const options = {
  stages: [
    { duration: '30s', target: 100 },   // ramp up
    { duration: '2m', target: 1000 },   // sustain 1000 RPS
    { duration: '30s', target: 5000 },  // spike
    { duration: '1m', target: 1000 },   // recover
    { duration: '30s', target: 0 },     // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(99)<200'],  // p99 < 200ms
    http_req_failed: ['rate<0.001'],   // < 0.1% errors
  },
};

const errors = new Counter('validation_errors');

export default function () {
  const payload = JSON.stringify({
    name: `Test Lead ${__VU}-${__ITER}`,
    phone: `0987${String(__VU).padStart(6, '0')}`,
    source: 'facebook',
    score: Math.floor(Math.random() * 100),
  });

  const res = http.post('http://localhost:8080/api/crm/v1/lead', payload, {
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': 'apex-fintech',
      'Authorization': 'Bearer ' + __ENV.TOKEN,
    },
  });

  check(res, {
    'status is 201': (r) => r.status === 201,
    'response time < 200ms': (r) => r.timings.duration < 200,
  });

  if (res.status >= 400) errors.add(1);
  sleep(0.1);
}
```

Run:
```bash
k6 run --out json=results.json tests/load/dynamic-model.js
```

---

## 27. Implementation Roadmap chi tiết

Roadmap 16 tuần (4 tháng) chia thành 4 phase. Mỗi task có owner, estimate, dependency rõ ràng.

### Phase 1: Foundation (Tuần 1-4)

#### Tuần 1: Setup & Schema Definition
- [ ] **Day 1-2:** Init repo `services/dynamic-model/` với Go module + Echo + sqlc.
  - Owner: Backend Lead
  - Setup: `go mod init github.com/itdoanh/rinco/dynamic-model`
  - Add deps: echo, huma, sqlc, pgx, gobreaker, cel-go, slog, zap.
- [ ] **Day 2-3:** Setup PostgreSQL migration với Goose.
  - Tạo bảng `entity_definitions`, `workflow_defs`, `field_defs`, `schema_change_log`.
- [ ] **Day 3-4:** Implement Schema Definition Language parser (JSON Schema 2020-12).
  - Sử dụng `github.com/santhosh-tekuri/jsonschema/v5`.
  - Unit test: 100+ test cases cho draft 2020-12 compliance.
- [ ] **Day 4-5:** CRUD API cơ bản cho entity/field definitions.
  - Endpoint: §10.1.

#### Tuần 2: Validation Engine
- [ ] **Day 1-2:** CEL environment setup + sandbox config.
  - Implement §16.1 với cost tracker.
- [ ] **Day 2-3:** Layer 1-2 validation (Type + JSON Schema).
- [ ] **Day 3-4:** Layer 3 (CEL field-level).
- [ ] **Day 4-5:** Layer 4 (Unique DB check).
  - Sử dụng `INSERT ... ON CONFLICT` của PostgreSQL.

#### Tuần 3: Dynamic CRUD & Database
- [ ] **Day 1-2:** Hybrid storage layer (PostgreSQL JSONB).
  - Implement dynamic INSERT/UPDATE với reflection-free template generation.
- [ ] **Day 2-3:** GIN + expression index generation từ schema.
- [ ] **Day 3-4:** Generic CRUD endpoint `GET/POST/PATCH/DELETE /api/crm/v1/:entity`.
- [ ] **Day 4-5:** Pagination + sort + filter generic.

#### Tuần 4: Workflow Engine cơ bản
- [ ] **Day 1-2:** Workflow definition storage + load.
- [ ] **Day 2-3:** State machine executor (§15.3).
- [ ] **Day 3-4:** Trigger queue với NATS.
- [ ] **Day 4-5:** Workflow simulator CLI (`rincoctl workflow simulate`).

### Phase 2: Hardening (Tuần 5-8)

#### Tuần 5: Validation nâng cao
- [ ] **Day 1-2:** Cross-entity rule engine (§16.2).
- [ ] **Day 2-3:** Formula DSL parser + evaluator.
- [ ] **Day 3-4:** Rollup field với cache.
- [ ] **Day 4-5:** Realtime validation performance benchmark.

#### Tuần 6: Schema Versioning & Hot Reload
- [ ] **Day 1-2:** Optimistic locking với ETag (§17.1).
- [ ] **Day 2-3:** Schema change audit log.
- [ ] **Day 3-4:** NATS pub/sub cho invalidation.
- [ ] **Day 4-5:** L1 + L2 cache (ristretto + Valkey).

#### Tuần 7: UI Schema Builder MVP
- [ ] **Day 1-3:** Next.js scaffold + entity list UI.
- [ ] **Day 3-5:** Field editor (add/edit/delete field).
- [ ] **Day 5:** Workflow editor với React Flow (simplified).

#### Tuần 8: Code Generation Pipeline
- [ ] **Day 1-2:** Stable schema detector (§18.1).
- [ ] **Day 2-3:** Git snapshot + CI/CD trigger.
- [ ] **Day 3-4:** Ent code-gen test cho 1 entity mẫu.
- [ ] **Day 4-5:** TypeScript type-gen test.

### Phase 3: Production Readiness (Tuần 9-12)

#### Tuần 9: Industry Templates
- [ ] **Day 1-2:** Real Estate template (§20.1).
- [ ] **Day 2-3:** Finance template (§20.2).
- [ ] **Day 3:** Education template (§20.3).
- [ ] **Day 4:** E-commerce template (§20.4).
- [ ] **Day 5:** Agency template (§20.5).

#### Tuần 10: Migration Tooling
- [ ] **Day 1-2:** Zero-downtime migration runner (§21.1).
- [ ] **Day 2-3:** Goose integration + rollback.
- [ ] **Day 3-4:** Auto-migrator service Python.
- [ ] **Day 4-5:** Drift detector + Mongo migration recommendation.

#### Tuần 11: Security & Performance
- [ ] **Day 1-2:** Field-level permission (§23.2).
- [ ] **Day 2-3:** Encryption + Redact helper.
- [ ] **Day 3-4:** Load test k6 + benchmark report.
- [ ] **Day 4-5:** Performance optimization dựa trên profiling.

#### Tuần 12: Disaster Recovery & Observability
- [ ] **Day 1-2:** Backup + PITR setup.
- [ ] **Day 2-3:** DR drill scripted.
- [ ] **Day 3-4:** Schema corruption auto-repair.
- [ ] **Day 4-5:** Grafana dashboard cho dynamic-model.

### Phase 4: Scale & Polish (Tuần 13-16)

#### Tuần 13-14: Multi-region & Scaling
- [ ] Cross-region replication cho PostgreSQL.
- [ ] Read replicas với read-write split.
- [ ] Cache warming on failover.

#### Tuần 15: Documentation & Training
- [ ] API docs từ Huma OpenAPI.
- [ ] User guide cho admin (Schema Builder).
- [ ] Video tutorial cho workflow builder.

#### Tuần 16: GA Launch
- [ ] Security audit (external).
- [ ] Penetration test.
- [ ] Canary deploy.
- [ ] Documentation final.

### Deliverable Timeline Summary

| Week | Milestone |
|------|-----------|
| 4 | Dynamic CRUD + Validation basic work |
| 8 | Workflow + Hot Reload + UI MVP |
| 12 | Industry Templates + Migration + Security |
| 16 | GA: Multi-region + DR + Docs |

---

## 28. Open Questions / Cần user xác nhận

Các câu hỏi cần user quyết định trước khi implement:

1. **Formula DSL:** Có cần support nested formula phức tạp kiểu Excel không? Hay chỉ cần arithmetic + function cơ bản? → Ảnh hưởng tới complexity của DSL parser.

2. **Code-gen trigger:** Có nên auto-trigger khi schema stable, hay cần admin manual click "Generate Code"? → Ảnh hưởng tới deployment flow.

3. **MongoDB migration:** Có cần auto-migrate sang MongoDB khi drift > 50%, hay chỉ recommend và để admin quyết? → Ảnh hưởng tới operational complexity.

4. **Industry templates seed data:** Có cần nhúng sample data CSV vào template, hay chỉ schema? → Sample data giúp demo nhưng tăng bundle size.

5. **Custom function trigger:** "Run function" cho workflow trigger - nên support JavaScript (Wasm), Python, hay chỉ Go plugin? → JS Wasm linh hoạt nhất nhưng cần sandbox.

6. **Workflow Approval Chain:** Approval chain cho transition (multi-step) - cần support bao nhiêu level tối đa? 3, 5, unlimited?

7. **A/B test workflow:** Tenant có cần A/B test workflow không? Nếu có, traffic split % ở level user_id hay organization?

8. **Custom view embed iframe:** Cho phép embed view ra public site không? Nếu có, security policy thế nào (CSP, origin check)?

9. **Formula field language:** Có cần multi-language formula (Tiếng Việt) hay chỉ English? → "IF" vs "NẾU".

10. **Versioning depth:** Schema version giữ tối đa bao nhiêu version? 10, 50, unlimited?

11. **Industry templates bundle size:** 5 templates hiện tại + 10 sắp tới → bundle size có cần split lazy-load?

12. **Cross-tenant schema sharing:** Admin super có cần share schema giữa các tenant không (vd: tenant A tạo schema, gán cho tenant B)? → Multi-tenant data isolation implications.

13. **Schema export format:** Export schema ra format nào? JSON, YAML, SQL DDL, hay cả 3?

14. **Workflow time-based trigger:** Trigger tự động theo cron - tenant có cần config cron expression, hay dùng preset (daily, weekly, monthly)?

15. **Formula field dependency:** Formula A reference Formula B. Có cần explicit dependency declaration (để detect circular ref), hay implicit?

16. **Rollup field freshness:** Realtime, hourly, daily - default là gì? Realtime tốn query nhưng user-friendly; daily rẻ nhưng stale data.

17. **Mobile offline schema:** Khi mobile offline, có cần cache schema locally không? Nếu có, sync mechanism thế nào?

18. **Schema deprecation:** Field cũ sẽ bị xóa sau bao lâu? 30 ngày, 90 ngày, hay giữ mãi mãi (chỉ ẩn khỏi UI)?

19. **Custom error message i18n:** Error message có cần multi-language (Tiếng Việt, English, Japanese)?

20. **Industry template override:** Tenant customize 1 industry template rồi, update template version mới từ admin có auto-merge hay replace?

21. **Schema import conflict resolution:** Khi import schema từ file, gặp field trùng tên → skip, rename, hay error?

22. **Auto code-gen rollback:** Nếu code-gen pipeline fail ở bước canary, có cần auto-rollback hay manual?

23. **Browser compatibility:** Schema Builder UI support tới browser nào? Chrome 90+, Safari 14+, Edge 90+? Hay cần IE11 (legacy)?

24. **Quota per tenant:** Có cần quota field/entity count per tenant (free tier: 50 fields, pro: unlimited)? → Ảnh hưởng tới subscription model.

25. **Webhook signing:** Workflow webhook trigger có cần HMAC signing để receiver verify không?

---

**Tiếp theo:** [`docs/05-landing-capi/README.md`](../05-landing-capi/README.md) – Landing Page + Facebook CAPI chi tiết (đã được mở rộng với chiase_cu migration).
