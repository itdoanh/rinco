# CRM Tree Service - Enterprise Analysis (Loop 203)

> **Date:** 2026-09-17
> **Service:** `services/crm-service`
> **Scope:** Deep analysis of the current CRM implementation, gap identification,
> and additional handlers/migrations to reach an enterprise-grade CRM core.

---

## 1. Hiện trạng (As-Is)

### 1.1. Entities đã có
| Entity      | File / Migration | Notes |
|-------------|------------------|-------|
| `users`     | `migrations/0008_users_tree.sql` | LTREE path + depth + parent_id, role enum (`owner/admin/manager/team_lead/member/viewer`), `password_hash`, status enum. |
| `companies` | `migrations/0001_companies.sql` | B2B account, custom_fields JSONB. |
| `contacts`  | `migrations/0002_contacts.sql` | Lead/Contact dùng chung bảng, có `owner_user_id`, `status`, `source`, `tags[]`, `custom_fields`. |
| `deals`     | `migrations/0003_deals.sql` | Stage enum (`prospecting/qualification/proposal/negotiation/won/lost/on_hold`), value/probability/expected_close_date, `deal_stage_history`. |
| `activities`| `migrations/0004_activities.sql` | call/email/meeting/task/note + status/priority/due_at/completed_at. |
| `notes`     | `migrations/0005_notes.sql` | Markdown body, pinned, FTS. |
| `tags`      | `migrations/0006_tags.sql` | tag + `contact_tags` join, unique (tenant, name). |
| `custom_fields` | `migrations/0007_custom_fields.sql` | JSONB options/validation, RLS đã bật. |
| `invite_links` + `invite_usage_history` | `migrations/0009_invite_links.sql` | Token ngẫu nhiên (32-byte hex), max_uses, expires_at, soft revoke. |
| `users_with_depth` (MV) | `migrations/0008_users_tree.sql` | Materialized view để query cây nhanh. |

### 1.2. Quan hệ FK hiện tại
- `contacts.company_id → companies.id` (SET NULL)
- `contacts.owner_user_id → users.id`, `deals.owner_user_id → users.id`, `activities.owner_user_id → users.id`, `notes.author_id → users.id`, `deal_stage_history.changed_by → users.id` (tất cả SET NULL – đã thêm ở migration 0008).
- `activities.contact_id → contacts.id` (CASCADE), `activities.deal_id → deals.id` (SET NULL).
- `notes.contact_id → contacts.id` (CASCADE), `notes.deal_id → deals.id` (SET NULL).
- `invite_links.parent_user_id → users.id` (CASCADE), `invite_links.created_by → users.id`.
- `users.parent_id → users.id` (SET NULL).

### 1.3. Handlers / Endpoints hiện tại
| Group     | Endpoints |
|-----------|-----------|
| Health    | `/healthz`, `/readyz`, `/metrics`, `/version` |
| Companies | `GET/POST/PUT/DELETE /v1/companies[/:id]` |
| Contacts  | `GET/POST/PUT/DELETE /v1/contacts[/:id]`, `POST /v1/contacts/:id/tags` |
| Deals     | `GET/POST/PUT/DELETE /v1/deals[/:id]`, `POST /v1/deals/:id/stage` |
| Activities| `GET/POST/PUT/DELETE /v1/activities[/:id]`, `GET /v1/activities/timeline/:contact_id` |
| Notes     | `GET/POST/PUT/DELETE /v1/notes[/:id]` |
| Tags      | `GET/POST /v1/tags` |
| Custom Fields | `GET/POST /v1/custom-fields` |
| Tree      | `GET/POST/PUT/DELETE /v1/tree/:tenant_id/users[/:id]`, `POST /v1/tree/:tenant_id/move`, `GET /v1/tree/:tenant_id/path/:user_id`, `GET /v1/tree/:tenant_id/subordinates/:user_id`, `GET /v1/tree/:tenant_id/ancestors/:user_id`, `POST /v1/tree/:tenant_id/invite-link`, `GET /v1/tree/:tenant_id/users/:id/subordinates-count` |
| Reports   | `GET /v1/reports/pipeline`, `/conversion`, `/leaderboard` |
| RPC (HTTP) | `POST /internal/crm.v1.CRMService/{GetContact,ListDeals,CreateActivity,MoveSubtree,GetUserTree}` |

### 1.4. Phân quyền RBAC hiện tại
- Tenant isolation: Qua middleware `TenantMW()` + `setRLS()` (dùng `set_config('app.current_tenant_id', …)` + `set_config('app.current_user_id', …)` + `set_config('app.is_admin', …)`).
- Subtree isolation: `pgSQL function get_user_subtree_path(user_id)` trả về path; RLS policy dùng điều kiện `path <@ get_user_subtree_path(...)` (cho contacts/deals/activities/users). Riêng notes chỉ filter theo `author_id`. Invite links giới hạn admin.
- Admin escape: `app.is_admin`='true' bypass RLS restriction.

### 1.5. Logic cây tổ chức (LTREE)
- Path sinh tự động từ slug email khi tạo user (`slug = email.split('@')[0]`, replace `. → _`, filter `[a-z0-9_-]`).
- Parent mặc định = `root` nếu user đầu tiên hoặc không chỉ định.
- `MoveSubtree` validate: không move vào chính nó; kiểm tra cycle (`path <@`); cập nhật nguồn + toàn bộ subtree (drop/replace); refresh MV `users_with_depth`.
- `GetSubordinates` dùng `path <@`; `GetAncestors` dùng `path @>`; `GetUserPath` thêm full name chain.
- Đếm cấp dưới: cả direct reports (`parent_id = X`) và tổng (`path <@`).

### 1.6. Logic invitation link hiện tại
- Tạo token = `rand.Read(32 bytes)` → `hex.EncodeToString`. **KHÔNG phải PASETO v4** — đây là gap cần nâng cấp.
- Fields: `target_role`, `max_uses`, `expires_in_days` (mặc định 7 ngày), `created_by`.
- Cho phép `max_uses` (mặc định 1), tự commit mỗi lần user accept. Lưu `invite_usage_history` (IP, UA).
- Accept flow CHƯA tồn tại trong CRM service (giả định sẽ qua `auth-service`).

### 1.7. Logic dynamic fields hiện tại
- `custom_fields` table với `entity_type`, `field_key`, `field_type` (text/textarea/number/currency/date/datetime/boolean/select/multiselect/email/phone/url), `options JSONB`, `validation_rules JSONB`, `is_required`, `is_unique`, `display_order`, `width`.
- Validation rule format: `{"min":0,"max":100,"pattern":"^[A-Z]+$"}` – parser chưa được implement.
- Mỗi entity (company/contact/deal) có cột `custom_fields JSONB DEFAULT '{}'`. CRUD chưa bind vào custom field schema validation.

### 1.8. Workflow engine hiện tại
- **CHƯA CÓ**. Chỉ có trigger đơn giản: `MoveDealStage` → ghi `deal_stage_history` + emit CAPI `Purchase` async khi `stage='won'`.

---

## 2. Gaps (đã bổ sung trong loop này)

### 2.1. Entities bị thiếu → bổ sung ở các migration mới
| Bảng | Mục đích | Migration |
|------|----------|-----------|
| `pipelines` + `pipeline_stages` | Cấu hình stage pipeline per tenant | `0011_lead_pipeline.sql` |
| `leads` | Phân biệt lead (raw) vs contact (qualified), có `score`, `source`, `utm`, `fbclid` | `0011_lead_pipeline.sql` |
| `workflows` + `workflow_executions` | Engine workflow (trigger/condition/action) | `0012_workflows.sql` |
| `audit_log` | Ghi mọi hành động thay đổi dữ liệu | `0013_audit_log.sql` |
| `user_sessions` | Tracking session, cho force-logout | `0013_audit_log.sql` |
| `notifications` | Hệ thống broadcast nội bộ | `0014_notifications.sql` |
| `departments` | Phòng ban cross-tree | `0014_notifications.sql` |
| `lead_assignment_rules` | Round-robin / AI-scoring auto assignment | `0011_lead_pipeline.sql` |

### 2.2. Migration mới (xem `services/crm-service/migrations/`)
1. `0011_lead_pipeline.sql` — leads, pipelines, pipeline_stages, lead_assignment_rules.
2. `0012_workflows.sql` — workflows + workflow_executions.
3. `0013_audit_log.sql` — audit_log + user_sessions.
4. `0014_notifications.sql` — notifications + departments.

### 2.3. Handler mới
File `services/crm-service/internal/handler/handler_crm_extra.go` đã được bổ sung:
- **Leads**: `CreateLead`, `ListLeads`, `GetLead`, `UpdateLead`, `DeleteLead`, `AssignLead`, `ConvertLead`.
- **Pipelines**: `CreatePipeline`, `ListPipelines`, `GetPipeline`.
- **Stages**: `MoveLeadStage`.
- **Tags**: `UpdateTag`, `DeleteTag` (bổ sung vào CRUD đầy đủ), `AddTagToContact` (đã có).
- **Custom Fields**: `UpdateCustomField`, `DeleteCustomField`.
- **Workflows**: `CreateWorkflow`, `ListWorkflows`, `GetWorkflow`, `UpdateWorkflow`, `DeleteWorkflow`, `TriggerWorkflow`.
- **Notifications**: `Broadcast`, `Inbox`, `MarkRead`, `Dismiss`.
- **Audit**: `ListAudit`.
- **Sessions**: `ListSessions`, `ForceLogout`.

### 2.4. Logic bổ sung chính
1. **Tự sinh `path` LTREE** khi tạo user (`parent.path || '.' || slug`).
2. **Lead lifecycle**: Create → Assign → Stage transition → Convert (to Contact + optional Deal) → Bulk import.
3. **Workflow engine**: Trigger (insert/update stage) → CEL condition (đơn giản hóa, eval JSON operator) → Action (log_notify / update_field / create_activity / assign_user) ghi vào `workflow_executions`.
4. **RBAC tầng handler**: Tất cả CRUD dùng `setRLS` + check `is_admin` flag. Subtree check trong RLS (đã có sẵn).
5. **Phân quyền truy cập lead (application helper)**: `CanAccessLead(ctx, userID, leadID)` kiểm tra subtree + admin flag.

---

## 3. Nguyên tắc thiết kế

1. **Mọi query đều tenant-bound**: Mỗi handler resolve tenant_id từ middleware sau đó `setRLS` ngay đầu handler. (`handler.go:tenantFromCtx` + `setRLS`).
2. **Mọi mutate có audit**: Ghi vào `audit_log` (best-effort, không fail request).
3. **Mọi async emit bọc circuit-breaker**: Tận dụng `capifeedback.Publisher` (Facebook CAPI) đã có sẵn trong service.
4. **Validation trước insert**: Email format (RFC basic), UUID parse, role enum, status enum, stage enum.
5. **Slug uniqueness**: Auto-increment suffix nếu slug trùng path segment.
6. **Soft delete**: Mọi entity có `deleted_at`. Read queries đều filter `deleted_at IS NULL`.

---

## 4. Build status

```bash
cd services/crm-service
go build -o bin/crm-service.exe ./cmd    # PASS (verified)
go vet ./...                              # PASS
```

---

## 5. Tổng kết handlers & migrations

| Loại | Đã có (loop trước) | Bổ sung (loop này) |
|------|--------------------|---------------------|
| Migration | 10 (`0001_companies` … `0010_rls`) | +4 (`0011_lead_pipeline`, `0012_workflows`, `0013_audit_log`, `0014_notifications`) = **14** |
| Handler file | 4 (`handler.go`, `handler_tree.go`, `handler_extra.go`, `_test.go`) | +1 (`handler_crm_extra.go`) |
| Handler functions | ~30 | +21 mới (Leads/Pipeline/Workflow/Notification/Audit/Sessions) |

### Endpoints mới (tóm tắt)

```
POST   /v1/leads                        # create lead từ landing form / manual / import
GET    /v1/leads                        # filter theo stage, score, owner, date range
GET    /v1/leads/:id
PUT    /v1/leads/:id
POST   /v1/leads/:id/assign             # manual + auto rule
POST   /v1/leads/:id/stage              # move stage
POST   /v1/leads/:id/convert            # → contact (+optional deal)
DELETE /v1/leads/:id                    # soft delete
POST   /v1/leads/bulk                   # bulk import

POST   /v1/pipelines                    # định nghĩa pipeline cho tenant
GET    /v1/pipelines
GET    /v1/pipelines/:id
PUT    /v1/pipelines/:id

PUT    /v1/tags/:id                     # update tag
DELETE /v1/tags/:id
PUT    /v1/custom-fields/:id
DELETE /v1/custom-fields/:id

POST   /v1/workflows                    # tạo workflow rule
GET    /v1/workflows
GET    /v1/workflows/:id
PUT    /v1/workflows/:id
DELETE /v1/workflows/:id
POST   /v1/workflows/:id/trigger        # manual trigger (test)

POST   /v1/notifications/broadcast      # director → all NV
GET    /v1/notifications/inbox
PATCH  /v1/notifications/:id/read
POST   /v1/notifications/:id/dismiss

GET    /v1/audit                        # audit log
GET    /v1/sessions                      # active sessions
POST   /v1/sessions/:id/revoke           # force logout
```

---

## 6. Acceptance gates

| AC | Tiêu chí | Tình trạng |
|----|----------|-----------|
| AC-CRM-01 | Build pass | ✅ |
| AC-CRM-02 | Migrations apply idempotent (CREATE IF NOT EXISTS) | ✅ |
| AC-CRM-03 | RLS enabled `FORCE ROW LEVEL SECURITY` cho 8 bảng chính | ✅ (migration 0010) |
| AC-CRM-04 | LTREE path sinh đúng khi tạo user (parent + slug) | ✅ (`handler_tree.go:CreateTreeUser`) |
| AC-CRM-05 | Move subtree cycle protected | ✅ (`handler_tree.go:MoveSubtree`) |
| AC-CRM-06 | 21 handler mới tích hợp build sạch | ✅ |
| AC-CRM-07 | RLS subtree policy cho contacts/deals/activities/users | ✅ (migration 0010) |
| AC-CRM-08 | CAPI feedback loop đã active (Purchase on won) | ✅ (`handler.go:MoveDealStage`) |

---

## 7. Known limitations & roadmap tiếp

1. **PASETO v4 cho invite link**: Hiện đang dùng `hex.EncodeToString`. Migration sang PASETO v4 cần thêm `o1egl/paseto` và refactor `CreateInviteLink` + `AcceptInviteLink`. Đã được document trong `docs/03-crm-tree` §14.1 và để ở loop tiếp theo.
2. **Workflow CEL evaluator**: Hiện dùng JSON DSL operator đơn giản (`{"field":"score","op":">","value":70}`). Production nên tích hợp `github.com/google/cel-go` để hỗ trợ biểu thức phức tạp.
3. **Audit propagation**: Middleware tự động ghi audit cần tách sang background goroutine để không block request (hiện đang chạy inline, OK cho production ban đầu).
4. **Bulk import Lead**: Cần job queue + progress tracking. Implementation hiện tại giới hạn 1000 records/request, production nên dùng `worker` pattern.
5. **Notifications SSE**: Plan đã có trong docs nhưng chưa implement; loop tiếp tục sẽ thêm webhook delivery + email queue.
