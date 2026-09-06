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

---

## 1. Mục tiêu & Nguyên tắc

### 1.1. Mục tiêu
| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| DM-1 | Tạo entity mới không cần restart | Hot reload |
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

**Tiếp theo:** [`docs/05-landing-capi/README.md`](../05-landing-capi/README.md) – Landing Page + Facebook CAPI chi tiết.