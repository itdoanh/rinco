# Summary – 03 CRM Tree (LTREE Hierarchical Organization)

> Source: `docs/03-crm-tree/README.md` (~78 KB, 2451 lines)
> Subsystem: CRM for employees inside each tenant — infinite-depth hierarchical org tree (Director → Manager → Lead → Employee) with PASETO invitation tokens, RBAC, and supporting CRM entities (leads, deals, activities, notes).

## 1. Overview & Goals

### 1.1 Objectives

| ID | Objective | Measurement |
|----|-----------|-------------|
| CRM-1 | Unlimited hierarchy depth | LTREE depth unbounded |
| CRM-2 | Lead distribution by branch | Each Lead has `owner_user_id` |
| CRM-3 | Fast invite via link | 1-click link generation |
| CRM-4 | Aggregated reports by branch | Rollup query < 100 ms |
| CRM-5 | Data isolation per tenant + branch | RLS enforced |

### 1.2 Guiding Principles

- Every user belongs to one node in the tree.
- Each node has exactly one role; roles can differ across nodes.
- Permission is inversely proportional to depth: higher in the tree = more power.
- Moving a branch = moving the entire subtree, atomically.
- Invitation links grant a role at most one level below the inviter.

### 1.3 Philosophy

"Cây Cổ Thụ" (Ancient Tree) — Roots (Admin) → Trunk (Managers) → Branches (Employees) → Fruit (Customers).

---

## 2. Tree Architecture (PostgreSQL LTREE)

### 2.1 Schema sketch

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  parent_id UUID REFERENCES users(id),
  path LTREE NOT NULL,             -- e.g. 'root.giamdoc.ql_kd.truong_nhom.nv01'
  role TEXT NOT NULL,
  depth INT NOT NULL,
  ...
);
CREATE INDEX idx_users_path ON users USING GIST(path);
```

### 2.2 Example tree

```
root (GiamDoc - GD)            path: root
├── QuanLy01 (QL)              path: root.quanly01
│   ├── TruongNhomA (TN)       path: root.quanly01.truongnhoma
│   │   ├── NV01 (NV)          path: root.quanly01.truongnhoma.nv01
│   │   └── NV02 (NV)          path: root.quanly01.truongnhoma.nv02
│   └── TruongNhomB (TN)       path: root.quanly01.truongnhomb
│       └── NV03 (NV)          path: root.quanly01.truongnhomb.nv03
└── QuanLy02 (QL)              path: root.quanly02
    └── NV04 (NV)              path: root.quanly02.nv04
```

### 2.3 Path Slug Rules

- Auto-generated from email (`nv01@apex.vn` → `nv01`).
- Slugs may repeat across branches; full path stays unique.
- Validation regex: `[a-z0-9-]{3,30}`.

### 2.4 Core Query Patterns

```sql
-- All descendants
SELECT * FROM users WHERE path <@ 'root.quanly01';

-- All ancestors
SELECT * FROM users WHERE path @> 'root.quanly01.truongnhoma.nv01';

-- Subtree at depth N
SELECT * FROM users WHERE nlevel(path) <= 4;
```

---

## 3. RBAC – Roles & Permissions

### 3.1 Default Roles

| Role | Code | Depth | Permission |
|------|------|-------|-----------|
| Giám đốc (Director) | GD | 1 | Entire tenant |
| Phó Giám đốc (Deputy Director) | PGD | 2 | Own subtree |
| Quản lý (Manager) | QL | 2-3 | Subtree |
| Trưởng nhóm (Lead) | TN | 3-5 | Subtree |
| Nhân viên (Employee) | NV | 4+ | Own data only |
| Viewer | VIEW | any | Read-only subtree |

### 3.2 Permission Matrix (excerpt)

| Action | GD | PGD | QL | TN | NV | VIEW |
|--------|----|----|----|----|----|------|
| View subtree leads | ✅ | ✅ | ✅ | ✅ | own | ✅ |
| Create Lead | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Edit Lead | ✅ | ✅ | ✅ | ✅ | own | ❌ |
| Delete Lead | ✅ | ✅ | ✅ | own | ❌ | ❌ |
| Assign Lead downward | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| View subtree reports | ✅ | ✅ | ✅ | ✅ | own | ✅ |
| Create user | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| Edit user in subtree | ✅ | ✅ | ✅ | own | ❌ | ❌ |
| Configure workflow | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| View billing | ✅ | own | own | own | own | own |

### 3.3 Custom Roles

- Super Admin / Tenant Admin can create custom roles.
- Permission sets are picked from predefined actions.

### 3.4 Departments

- A user belongs to one or more departments.
- Departments have their own leaders (cross-tree).
- Reports can be sliced by department instead of by tree.

---

## 4. PASETO v4 Invitation Links

### 4.1 Flow

```
Manager → choose role + limits → generate link → share
                                              ↓
New user clicks → registration page → fills form → submits
                                              ↓
Server verifies PASETO token → checks expiry → creates user with parent = Manager
```

### 4.2 Token Payload

```json
{
  "v4": { "purpose": "local", "payload": {
    "iss": "rinco-crm",
    "sub": "invitation",
    "tenant_id": "apexfintech",
    "parent_user_id": "uuid...",
    "target_role": "NV",
    "max_subtree_depth": 5,
    "max_uses": 5,
    "current_uses": 0,
    "allowed_departments": ["sales"],
    "exp": "2026-09-13T10:00:00Z",
    "iat": "2026-09-06T10:00:00Z",
    "jti": "uuid-v7..."
  }}
}
```

### 4.3 Lifecycle

- **Default expiry:** 7 days.
- **Max uses:** configurable (1 or N).
- **Revocation:** via Valkey blacklist on the JTI.
- **Rotation:** after max uses, manager generates a new link.

### 4.4 Signing Keys

- Each tenant has its own PASETO key (stored encrypted in PostgreSQL).
- Rotate every 90 days, keep a backup key for recovery.

### 4.5 UI Flow

1. Manager → Settings → Team → "Invite member".
2. Pick role (NV / TN / QL / …) and limits.
3. Click "Generate link" → receive link + QR code.
4. Copy / share via chat / email.
5. Track usage, revoke if needed.

---

## 5. Promote / Demote & Move Subtree

### 5.1 Promote

```
NV (parent=TN)  ─promote─►  TN (parent=QL)
```

- Update role and (optionally) parent.
- Trigger webhook + notification.
- Audit log.

### 5.2 Demote

- Only users within own subtree can be demoted.
- Role decreases, parent may stay or change.

### 5.3 Move Subtree

- Atomic update of the entire subtree path.
- Implemented via LTREE functions (`subpath`, `<@`, `||`).
- **Caveat:** All leads/deals owned by the moved users follow them to the new branch.
- **Two-step confirm:** preview, then commit.

### 5.4 Move Algorithm

```sql
BEGIN;
SELECT path INTO old_path FROM users WHERE id = $source_id;
SELECT path INTO new_parent_path FROM users WHERE id = $new_parent_id;
new_path := new_parent_path || subpath(old_path, -1);

UPDATE users SET path = new_path, parent_id = $new_parent_id WHERE id = $source_id;

UPDATE users
SET path = new_path || subpath(path, nlevel(old_path))
WHERE path <@ old_path AND id != $source_id;

COMMIT;
```

### 5.5 Bulk Move

- Move an entire branch in one transaction.
- Preview before commit.
- Automatic rollback on error.

---

## 6. Dashboards

### 6.1 Personal Dashboard

- Welcome banner.
- Stats cards: New leads today, Leads in progress, Active deals, Revenue MTD, Conversion, Avg response time.
- Charts: Lead conversion funnel, Daily/Weekly/Monthly activity, Performance vs team, Revenue trend.
- Task lists: today, overdue, upcoming follow-ups.

### 6.2 Manager Dashboard (subtree view)

- Whole-branch overview.
- Drill-down per employee.
- Performance comparison.
- Lead distribution map.

### 6.3 Director Dashboard

- Tenant-wide.
- Lead distribution by branch.
- Revenue by branch.
- Top performers.
- Risk alerts.

---

## 7. Data-Level Permissions

### 7.1 Ownership

- Every Lead has `owner_user_id`.
- Every Deal has `owner_user_id` plus optional `team_id`.

### 7.2 Row-Level Security

```sql
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;

CREATE POLICY leads_tenant_isolation ON leads
  USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY leads_subtree_access ON leads
  USING (
    owner_user_id = current_setting('app.current_user_id')::UUID
    OR owner_user_id IN (
      SELECT id FROM users WHERE path <@ (
        SELECT path FROM users WHERE id = current_setting('app.current_user_id')::UUID
      )
    )
  );
```

### 7.3 Application Filter

- Middleware sets `SET LOCAL app.current_tenant_id` and `app.current_user_id`.
- Every query passes through RLS automatically.

### 7.4 Shared Lead Pool

- Leads can exist with no owner (team pool).
- Employees can claim them.
- Managers can also assign directly.

---

## 8. Database Schema (key tables)

### 8.1 `users`

- `id`, `tenant_id`, `email`, `phone`, `password_hash`, `full_name`, `avatar_url`
- `parent_id`, `path LTREE`, `depth INT`, `role TEXT`, `department TEXT[]`
- `status` (`ACTIVE` / `INACTIVE` / `LOCKED` / `PENDING`)
- `invitation_token_id`, `last_login_at`, `last_login_ip`, `mfa_enabled`
- `created_at`, `updated_at`, UNIQUE (`tenant_id`, `email`)
- Indexes: GIST on `path`, B-tree on (`tenant_id`, `parent_id`).

### 8.2 `invitations`

- `paseto_token`, `parent_user_id`, `target_role`, `max_uses`, `current_uses`
- `max_subtree_depth`, `allowed_departments TEXT[]`
- `expires_at`, `revoked`, `revoked_at`, `created_at`.

### 8.3 `user_roles` (custom roles)

- `role_code`, `role_name`, `permissions TEXT[]`, `inherits_from`.

### 8.4 `user_role_assignments`

- `user_id`, `role_id`, `assigned_by`, `assigned_at`.

### 8.5 `audit_actions`

- `actor_id`, `actor_email`, `action` (e.g. `user.create`, `lead.move`)
- `target_type`, `target_id`, `old_value JSONB`, `new_value JSONB`, `ip_address`.

### 8.6 `departments`

- `name`, `description`, `leader_id`, `parent_department_id`.

### 8.7 `lead_activities`

- `lead_id`, `actor_id`, `activity_type` (`CALL` / `EMAIL` / `MEETING` / `NOTE` / `STATUS_CHANGE` / `ASSIGN` / `TASK`)
- `description`, `metadata JSONB`, `due_at`, `completed_at`.

### 8.8 `leads`

- `owner_user_id`, `full_name`, `email`, `phone`
- `source`, `utm JSONB`, `fbclid`, `fbp`
- `status` (`NEW` / `CONTACTED` / `QUALIFIED` / `PROPOSAL` / `WON` / `LOST` / `ARCHIVED`)
- `score` (AI 0-100), `estimated_value`
- `custom_fields JSONB`, `tags TEXT[]`
- `next_followup_at`, `last_contacted_at`, `converted_at`, `lost_reason`
- Indexes: GIN on `custom_fields`, partial on score and follow-up.

### 8.9 `deals`

- `lead_id`, `owner_user_id`, `name`, `pipeline_stage`, `amount`, `currency`
- `probability`, `expected_close_date`, `actual_close_date`
- `status` (`OPEN` / `WON` / `LOST` / `ON_HOLD`), `lost_reason`, `products JSONB`, `notes`.

### 8.10 `pipelines`

- Per-tenant pipeline definitions: `name`, `stages JSONB` (each with name, color, probability, SLA hours), `is_default`.

### 8.11 `user_sessions`

- `paseto_token_id` (JTI), `device_fingerprint`, `ip_address`, `user_agent`
- `login_at`, `last_active_at`, `expires_at`, `revoked`, `revoked_reason`.

### 8.12 `user_notifications`

- `user_id`, `sender_user_id` (NULL = system), `type` (`broadcast` / `mention` / `lead_assigned` / `task_due` …)
- `title`, `body`, `link`, `metadata JSONB`, `is_read`, `read_at`
- `priority` (`LOW` / `NORMAL` / `HIGH` / `URGENT`).

---

## 9. Main CRM Entities (Fields)

### 9.1 Contacts / Leads

- **Identity:** `id`, `tenant_id`, `owner_user_id`.
- **Personal:** `full_name`, `email`, `phone`.
- **Acquisition:** `source` (facebook, tiktok, google, direct, referral, organic), `utm` JSONB `{source, medium, campaign, term, content}`, `fbclid`, `fbp`.
- **Status:** `NEW` → `CONTACTED` → `QUALIFIED` → `PROPOSAL` → `WON` / `LOST` / `ARCHIVED`.
- **Scoring:** `score` (AI 0-100), `estimated_value`.
- **Follow-up:** `next_followup_at`, `last_contacted_at`, `converted_at`, `lost_reason`.
- **Customization:** `custom_fields JSONB`, `tags TEXT[]`.

### 9.2 Companies (implied reference)

- Each Lead can be linked to a Company via custom fields or future entities.

### 9.3 Deals

- `name`, `lead_id`, `pipeline_stage`, `amount`, `currency` (default VND), `probability` (0-100).
- `expected_close_date`, `actual_close_date`.
- `status` (`OPEN` / `WON` / `LOST` / `ON_HOLD`).
- `lost_reason`, `products JSONB`, `notes`.
- SLA via `pipelines.stages[].sla_hours`.

### 9.4 Activities (`lead_activities`)

- Type: `CALL`, `EMAIL`, `MEETING`, `NOTE`, `STATUS_CHANGE`, `ASSIGN`, `TASK`.
- `actor_id`, `description`, `metadata JSONB`.
- `due_at`, `completed_at` for tasks.

### 9.5 Notes

- Stored as `lead_activities` rows of type `NOTE` with markdown content in `description` and optional attachments in `metadata`.

### 9.6 Notifications (`user_notifications`)

- `type`, `title`, `body`, `link`, `metadata`.
- `priority`, `is_read`, `read_at`, `sender_user_id`.

---

## 10. Feature Catalog (selected 35+)

### 10.1 User Management (1–25)

1. Create user via form.
2. Invite via PASETO link.
3. Invite via direct email.
4. Bulk invite (CSV upload).
5. Import users from legacy system.
6. Export user list.
7. Search users (full-text).
8. Filter by role / status / branch.
9. View user profile.
10. Edit user info (name, email, phone, avatar).
11. Change user role.
12. Change parent (move subtree).
13. Lock user (login blocked).
14. Unlock user.
15. Reset password.
16. Force logout (kill all sessions).
17. View active sessions.
18. Soft-delete user.
19. Restore deleted user.
20. Merge two users (e.g. duplicate email).
21. Custom user permissions.
22. Assign department.
23. Assign label/tag to user.
24. User activity timeline.
25. Bulk assign / bulk move / bulk promote.

### 10.2 Org Tree (26–45)

26. Org Chart view (zoomable, drag).
27. List view.
28. Collapsible tree view.
29. Drag-and-drop reordering (optional).
30. Search within tree.
31. Highlight current user's subtree.
32. Filter by role.
33. Filter by department.
34. Show user count in subtree.
35. Show subtree revenue.
36. Show subtree lead count.
37. Export tree as OrgChart JSON.
38. Compare two branches (performance).
39. Bulk move users.
40. Bulk promote/demote.
41. Bulk deactivate.
42. Bulk re-assign parent.
43. Org chart print / PDF export.

### 10.3 Roles & Permissions (46–65)

44. Create custom role.
45. Edit role permissions.
46. Clone / delete role.
47. Predefined permission catalog (200+ actions).
48. Permission groups (Lead, Deal, Report…).
49. Role inheritance.
50. Per-user permission override.
51. Effective permissions viewer.
52. Permission testing (impersonate).
53. Permission audit log.
54. Department-scoped permissions.
55. Conditional permissions (data-driven).
56. Time-based permissions (working hours).
57. IP-based permissions.
58. Permission delegation (temporary).
59. Permission request workflow.
60. Permission approval.

### 10.4 Invite & Onboarding (66–85)

61. Create invite link with role selector.
62. Configure link (max uses, expiry, allowed dept).
63. QR code for link.
64. Share link via email / chat / SMS.
65. Track link status.
66. Revoke link.
67. Extend expiry.
68. Auto-reminder before expiry.
69. Welcome email template.
70. Welcome in-app message.
71. Onboarding checklist.
72. First-login wizard.
73. In-app tour guide.
74. Assign trainer.
75. Sample data for new employee.
76. Bulk onboarding (CSV).
77. Onboarding progress tracking.
78. Auto-deactivate after 30 days no login.
79. Re-activation link.
80. Offboarding checklist.

### 10.5 Dashboards & Reports (86–100)

81. Personal dashboard.
82. Team (subtree) dashboard.
83. Manager dashboard.
84. Director dashboard (tenant-wide).
85. Real-time stats via SSE.
86. Custom dashboard builder.
87. Drag-drop widgets.
88. Saved views.
89. Share dashboard via link.
90. Export dashboard (PDF / PNG).
91. Scheduled email reports.
92. Anomaly alerts.
93. AI forecast.
94. Period comparison.
95. Cohort analysis.

### 10.6 Profile & Personalization (101–120)

96. View / edit profile.
97. Avatar change.
98. Password change.
99. 2FA setup (TOTP, WebAuthn).
100. Backup codes for 2FA.
101. Notification preferences (email, push, SMS).
102. Theme (light / dark / system).
103. Language.
104. Timezone.
105. Working hours config.
106. Auto-reply when offline.
107. Custom fields on profile.
108. Calendar integration (Google / Outlook).
109. Vacation mode (auto-redirect leads).
110. Status (Available / Busy / In meeting / Offline).
114. Custom status message.
115. Gamification avatar frame.
116. Profile completion score.

### 10.7 Lead & Deal Management (121–160)

117. CRUD Lead.
118. Lead duplicate detection.
119. Lead merge.
120. Lead qualification form.
121. AI lead scoring.
122. Round-robin auto-assignment.
123. Lead pool (claim mechanism).
124. Manual lead reassign.
126. Lead transfer with reason.
127. Bulk import Leads (CSV / Excel).
128. Bulk export Leads.
129. Advanced filter / search.
130. Lead tags.
131. Lead segmentation.
132. Lead custom fields (Dynamic Schema).
133. Lead timeline (activity log).
134. Lead notes (markdown).
135. Lead email tracking.
136. Lead SMS tracking.
137. Lead call tracking (click-to-call).
138. Lead meeting scheduling.
139. Follow-up reminder.
140. Lost-reason tracking.
141. Win/loss analytics.
142. CRUD Deal.
143. Deal pipeline (Kanban).
144. Deal stage transition rules.
145. Deal probability calculation.
146. Revenue forecast.
147. Deal product line items.
148. Close-date tracking.
149. Stage-change notifications.
150. Same-customer deal comparison.
151. Bulk deal update.
152. Deal SLA tracking.
153. Won celebration animation.
154. Lost retro analysis.
155. Source tracking (UTM).
156. Deal attribution report.

### 10.8 Internal Notifications (161–170)

157. Director broadcasts tenant-wide.
158. Manager broadcasts to branch.
159. Manager broadcasts to department.
160. Manager broadcasts to role.
161. Tenant-group notification.
162. Region notification.
163. Urgent bypasses quiet hours.
164. Scheduled send.
165. Daily / weekly digest.
166. A/B testing of notifications.

---

## 11. API Surface (selection)

### 11.1 User Management

```
GET    /api/crm/v1/users
POST   /api/crm/v1/users
GET    /api/crm/v1/users/:id
PATCH  /api/crm/v1/users/:id
DELETE /api/crm/v1/users/:id
POST   /api/crm/v1/users/:id/{lock,unlock,reset-password,force-logout,promote,demote,move}
```

### 11.2 Org Tree

```
GET    /api/crm/v1/tree/{me,subtree/:user_id,ancestors/:user_id,descendants/:user_id}
POST   /api/crm/v1/tree/{move-subtree,bulk-move}
```

### 11.3 Invitations

```
POST   /api/crm/v1/invitations
GET    /api/crm/v1/invitations
DELETE /api/crm/v1/invitations/:id
POST   /api/crm/v1/invitations/:id/revoke
GET    /api/crm/v1/invitations/verify/:token   (public)
POST   /api/crm/v1/invitations/accept          (public)
```

### 11.4 Roles & Permissions

```
GET    /api/crm/v1/roles
POST   /api/crm/v1/roles
PATCH  /api/crm/v1/roles/:id
DELETE /api/crm/v1/roles/:id
GET    /api/crm/v1/permissions
POST   /api/crm/v1/users/:id/permissions
GET    /api/crm/v1/users/:id/effective-permissions
```

### 11.5 Audit & Dashboard

```
GET    /api/crm/v1/audit
GET    /api/crm/v1/dashboard/{personal,team,director}
WS     /api/crm/v1/dashboard/realtime
```

### 11.6 Lead & Deal

```
GET    /api/crm/v1/leads
POST   /api/crm/v1/leads
GET    /api/crm/v1/leads/:id
PATCH  /api/crm/v1/leads/:id
DELETE /api/crm/v1/leads/:id
POST   /api/crm/v1/leads/:id/{assign,claim,merge,import}
GET    /api/crm/v1/leads/export

GET    /api/crm/v1/deals
POST   /api/crm/v1/deals
GET    /api/crm/v1/deals/:id
PATCH  /api/crm/v1/deals/:id
POST   /api/crm/v1/deals/:id/{stage,win,lost}
```

### 11.7 Internal Notifications

```
POST   /api/crm/v1/notifications/broadcast
GET    /api/crm/v1/notifications/inbox
PATCH  /api/crm/v1/notifications/:id/{read,dismiss}
GET    /api/crm/v1/notifications/preferences
PATCH  /api/crm/v1/notifications/preferences
```

---

## 12. UI / UX Pages

- `/team` – user list.
- `/team/tree` – Org Chart view.
- `/team/invite` – create invite link.
- `/team/invitations` – manage active links.
- `/team/roles` – role management.
- `/team/departments` – department management.
- `/dashboard` – personal dashboard.
- `/team-dashboard` – subtree dashboard.
- `/director-dashboard` – tenant dashboard.
- `/profile`, `/profile/security`, `/profile/preferences`.
- `/leads`, `/leads/:id`, `/deals`.
- `/notifications`, `/notifications/broadcast`.

### 12.1 Components

OrgChart (D3.js / react-d3-tree), UserCard, TreeNode, InvitationLinkGenerator, PermissionMatrix, RoleBuilder, AuditTimeline, PersonalStats, TeamPerformanceChart, ActivityFeed, BulkActionBar, MoveUserDialog (2-step confirm), LeadPipeline (Kanban), LeadDetailSheet, NotificationComposer, NotificationInbox.

### 12.2 Mobile

Responsive design, swipe gestures for promote / demote, push notifications.

---

## 13. Edge Cases (highlights)

- **Tree T1–T10:** missing parent FK, slug collisions, orphans, cycles, max-depth, Unicode slugs, bulk-lock contention, cross-tenant merges.
- **Invite I1–I10:** expiry during registration, max-uses reached, token revoked, race conditions on accept, key rotation during sign.
- **RBAC R1–R10:** multi-role union, deleted-role dangling, timezone-sensitive time-perms, role edits during active session.
- **Dashboard D1–D10:** SSE reconnect, 10K subtree, lazy widget loading, real-time 1M events/s, forecast fallback.
- **Lead/Deal L1–L10:** bulk CSV import, duplicate detection, stage-skip rules, race conditions, AI score fallback.
- **Notifications N1–N10:** 10K-user broadcast batching, offline queue, dedupe, length truncation, broken links, quiet hours, translation fallback.

---

## 14. Code Examples (highlights)

- **PASETO Invitation Service (Go):** full `Create`, `Verify`, `Accept` flow with argon2id password hashing, slug auto-generation, atomic usage increment, audit log.
- **Move Subtree Service (Go + pgx):** serializable transaction, cycle prevention, max-depth check, affected-row counts for preview, audit + cache invalidation.
- **RBAC Middleware:** `Require(perm)` decorator merging default-role permissions with custom roles.
- **MoveUserDialog (TSX):** 4-step wizard (select → preview → confirm → processing) showing affected users, leads, deals.
- **NotificationService:** `Broadcast` with target filters (role, department, branch path), quiet hours check, urgent override, dedupe by `(user_id, type, target_id)`.
- **SSE Dashboard Stream:** initial snapshot, NATS subscribe, 15-second heartbeat, `Last-Event-ID` resume.

---

## 15. Roadmap (4 phases / 14 weeks)

- **Phase 1 (W1–6):** Core CRM — DB, user/tree service, invitation, RBAC, audit, dashboards.
- **Phase 2 (W7–10):** Lead / Deal — CRUD, scoring, Kanban, activities, auto-assignment.
- **Phase 3 (W11–14):** Notifications — table, broadcast API, SSE inbox, email + push integrations, real-time dashboards.

### 15.1 Acceptance Criteria

| AC | Criterion | Target |
|----|-----------|--------|
| AC-CRM-01 | Invite link created | < 500 ms p95 |
| AC-CRM-02 | Token verify | < 50 ms p95 |
| AC-CRM-03 | Subtree query on 100K users | < 100 ms |
| AC-CRM-04 | Move 1,000-user subtree | < 5 s |
| AC-CRM-05 | RLS blocks cross-tenant access | 100 % |
| AC-CRM-06 | Broadcast to 10K users | < 30 s |
| AC-CRM-07 | Lead create | < 50 ms |
| AC-CRM-08 | Deal pipeline load | < 200 ms |
| AC-CRM-09 | Realtime dashboard update | < 1 s |
| AC-CRM-10 | Unlimited tree depth | Tested up to 20 |

---

## 16. Open Questions

1. Max tree depth (10 / 20 / unlimited — recommended 10).
2. Default target role on invite (recommended NV).
3. Who authors the onboarding checklist (recommended system template).
4. Lead duplicate threshold (exact / fuzzy 80 % / AI — recommended fuzzy 80 %).
5. Default deal pipeline stages (user-configurable).
6. Max broadcast recipients (recommended 10K).
7. Can Director delete subtree (recommended Quorum).
8. Lead assignment mode (manual / AI suggested / auto — recommended AI suggested).
10. Bulk move confirmation (recommended bulk confirm).

(See source doc §20 for full list including Q11–Q18 and TBD items T1–T8.)