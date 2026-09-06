# Phần 3 – CRM cho Nhân viên (Sơ Đồ Cây Phân Cấp)

> **Phân hệ:** CRM dành cho nhân viên trong mỗi tenant.  
> **Mục tiêu:** Tổ chức nhân sự theo sơ đồ cây vô hạn cấp (Giám đốc → Quản lý → Trưởng nhóm → Nhân viên). Hỗ trợ tạo link mời theo cấp, thăng/hạ chức, di chuyển nhánh.  
> **Triết lý:** "Cây Cổ Thụ" – Rễ (Admin) → Thân (Quản lý) → Cành (Nhân viên) → Trái (Khách hàng).

---

## Mục lục
1. [Mục tiêu & Nguyên tắc](#1-mục-tiêu--nguyên-tắc)
2. [Cấu trúc Cây](#2-cấu-trúc-cây)
3. [Phân quyền RBAC theo Vai trò](#3-phân-quyền-rbac-theo-vai-trò)
4. [Tokenized Invitation Link (PASETO v4)](#4-tokenized-invitation-link-paseto-v4)
5. [Thăng/Hạ chức & Di chuyển Nhánh](#5-thănghạ-chức--di-chuyển-nhánh)
6. [Dashboard Nhân viên](#6-dashboard-nhân-viên)
7. [Phân quyền Dữ liệu](#7-phân-quyền-dữ-liệu)
8. [Database Schema](#8-database-schema)
9. [Danh sách tính năng (≥ 100)](#9-danh-sách-tính-năng)
10. [API Surface](#10-api-surface)
11. [UI/UX](#11-uiux)

---

## 1. Mục tiêu & Nguyên tắc

### 1.1. Mục tiêu
| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| CRM-1 | Phân cấp vô hạn cấp | LTREE depth không giới hạn |
| CRM-2 | Lead phân bổ theo nhánh | Mỗi Lead có owner_user_id |
| CRM-3 | Mời nhanh qua link | 1 click tạo link |
| CRM-4 | Báo cáo tổng hợp theo nhánh | Rollup query < 100ms |
| CRM-5 | Cách ly dữ liệu theo tenant + nhánh | RLS |

### 1.2. Nguyên tắc
- **Mỗi user thuộc về một node** trong cây.
- **Mỗi node có 1 role** (vai trò) – có thể khác nhau.
- **Quyền hạn tỉ lệ nghịch với độ sâu:** Càng cao càng có nhiều quyền.
- **Di chuyển nhánh = di chuyển cả subtree:** Atomic transaction.
- **Mời qua link = cấp tối đa bằng cấp của người mời - 1.**

---

## 2. Cấu trúc Cây

### 2.1. PostgreSQL LTREE
```sql
-- Bảng users có cột path
CREATE TABLE users (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  parent_id UUID REFERENCES users(id),
  path LTREE NOT NULL,        -- VD: 'root.giamdoc.ql_kd.truong_nhom.nhanvien01'
  role TEXT NOT NULL,
  depth INT NOT NULL,
  ...
);

-- Index để query nhanh
CREATE INDEX idx_users_path ON users USING GIST (path);
CREATE INDEX idx_users_tenant ON users(tenant_id);
```

### 2.2. Minh họa cây
```
root (GiamDoc - GD)         path: root
├── QuanLy01 (QL)            path: root.quanly01
│   ├── TruongNhomA (TN)     path: root.quanly01.truongnhoma
│   │   ├── NV01 (NV)        path: root.quanly01.truongnhoma.nv01
│   │   └── NV02 (NV)        path: root.quanly01.truongnhoma.nv02
│   └── TruongNhomB (TN)     path: root.quanly01.truongnhomb
│       └── NV03 (NV)        path: root.quanly01.truongnhomb.nv03
└── QuanLy02 (QL)            path: root.quanly02
    └── NV04 (NV)            path: root.quanly02.nv04
```

### 2.3. Path Slug Rules
- Auto-generate từ email: `nv01@apex.vn` → `nv01`.
- Cho phép trùng slug trong các nhánh khác nhau (path duy nhất).
- Validate slug: `[a-z0-9-]{3,30}`.

### 2.4. Query Patterns
```sql
-- Tất cả cấp dưới của user X
SELECT * FROM users WHERE path <@ 'root.quanly01';

-- Tất cả cấp trên của user X
SELECT * FROM users WHERE path @> 'root.quanly01.truongnhoma.nv01';

-- Cây con depth N
SELECT * FROM users WHERE nlevel(path) <= 4;

-- Subtree của user X
SELECT * FROM users WHERE path <@ 'root.quanly01' ORDER BY path;
```

---

## 3. Phân quyền RBAC theo Vai trò

### 3.1. Vai trò mặc định
| Role | Code | Depth | Quyền |
|------|------|-------|-------|
| **Giám đốc** | GD | 1 | Toàn tenant |
| **Phó Giám đốc** | PGD | 2 | Subtree của mình |
| **Quản lý** | QL | 2-3 | Subtree |
| **Trưởng nhóm** | TN | 3-5 | Subtree |
| **Nhân viên** | NV | 4+ | Data của mình |
| **Viewer** | VIEW | any | Read only subtree |

### 3.2. Permission Matrix
| Action | GD | PGD | QL | TN | NV | VIEW |
|--------|----|----|----|----|----|------|
| Xem Lead subtree | ✅ | ✅ | ✅ | ✅ | own | ✅ |
| Tạo Lead | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Sửa Lead | ✅ | ✅ | ✅ | ✅ | own | ❌ |
| Xóa Lead | ✅ | ✅ | ✅ | own | ❌ | ❌ |
| Phân Lead cho cấp dưới | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| Xem báo cáo subtree | ✅ | ✅ | ✅ | ✅ | own | ✅ |
| Tạo user mới | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| Sửa user subtree | ✅ | ✅ | ✅ | own | ❌ | ❌ |
| Cấu hình workflow | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Xem billing | ✅ | own | own | own | own | own |

### 3.3. Custom Role
- Super Admin / Tenant Admin có thể tạo custom role.
- Permission set chọn từ predefined actions.

### 3.4. Department (Phòng ban)
- Mỗi user thuộc 1 hoặc nhiều phòng ban.
- Phòng ban có leader riêng (cross-tree).
- Báo cáo có thể theo phòng ban thay vì cây.

---

## 4. Tokenized Invitation Link (PASETO v4)

### 4.1. Flow
```
[Manager A] → [Chọn role + Giới hạn] → [Tạo link] → [Share]
                                                            │
                                                            ▼
[New User click] → [Trang đăng ký] → [Điền form] → [Submit]
                                                            │
                                                            ▼
[Server verify PASETO token] → [Validate expire] → [Tạo user với parent = Manager A]
```

### 4.2. PASETO Token Structure
```json
{
  "v4": {
    "purpose": "local",
    "payload": {
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
    }
  }
}
```

### 4.3. Token Lifecycle
- **Default expiry:** 7 ngày.
- **Max uses:** Có thể giới hạn 1 lần hoặc N lần.
- **Revoke:** PASETO có thể revoke bằng cách thêm vào blacklist (Valkey).
- **Rotation:** Sau khi dùng hết, manager tạo link mới.

### 4.4. Signing Key
- Mỗi tenant có PASETO key riêng (lưu trong PostgreSQL encrypted).
- Rotate mỗi 90 ngày.
- Backup key cho recovery.

### 4.5. UI Flow
```
1. Manager vào Settings → Team → "Mời thành viên"
2. Chọn role (NV/TN/QL/...) và giới hạn
3. Click "Tạo link" → nhận link + QR code
4. Copy link hoặc share qua chat
5. Theo dõi số lượt dùng, revoke nếu cần
```

---

## 5. Thăng/Hạ chức & Di chuyển Nhánh

### 5.1. Thăng chức
```
NV (TN parent) ─thăng─► TN (QL parent)
```
- Update role, parent_id (có thể giữ hoặc đổi).
- Trigger webhook/notification.
- Audit log.

### 5.2. Hạ chức
```
TN (QL parent) ─hạ─► NV (cùng parent hoặc parent khác)
```
- Giống thăng nhưng role giảm.
- **Quy tắc:** Chỉ hạ được người trong subtree của mình.

### 5.3. Di chuyển nhánh (Move Subtree)
```
NV01 (TN_A) ─move─► NV01 (TN_B)
```
- Atomic update subtree path.
- Transaction: Update path của cả subtree (sử dụng `ltree` functions).
- **Cảnh báo:** Lead/Deal của NV01 sẽ thuộc về nhánh mới.
- **2 bước confirm:** Step 1 preview, Step 2 commit.

### 5.4. Move Algorithm
```sql
BEGIN;
-- Lấy path cũ và mới
SELECT path INTO old_path FROM users WHERE id = $source_id;
SELECT path INTO new_parent_path FROM users WHERE id = $new_parent_id;
new_path := new_parent_path || subpath(old_path, -1);

-- Update user
UPDATE users SET path = new_path, parent_id = $new_parent_id WHERE id = $source_id;

-- Update toàn bộ subtree
UPDATE users 
SET path = new_path || subpath(path, nlevel(old_path))
WHERE path <@ old_path AND id != $source_id;

COMMIT;
```

### 5.5. Bulk Move
- Move cả 1 nhánh sang vị trí khác.
- Preview trước khi commit.
- Rollback nếu có lỗi.

---

## 6. Dashboard Nhân viên

### 6.1. Layout
```
┌────────────────────────────────────────────────┐
│ Welcome back, NV01!                            │
├────────────┬───────────────────────────────────┤
│ Sidebar    │ Stats Cards:                      │
│ ▸ Dashboard│ ┌──────┬──────┬──────┬──────┐    │
│ ▸ Leads    │ │ 23   │ 5    │ 12   │ $5M  │    │
│ ▸ Deals    │ │ New  │ Hot  │ Deals│ Rev  │    │
│ ▸ Calendar │ └──────┴──────┴──────┴──────┘    │
│ ▸ Reports  │ Charts: Lead Conversion          │
│ ▸ Chat    │ My Tasks                           │
│ ▸ Meeting │ Recent Activities                  │
└────────────┴───────────────────────────────────┘
```

### 6.2. Stats Cards
- **Lead mới hôm nay.**
- **Lead đang xử lý.**
- **Deal đang active.**
- **Doanh thu tháng này.**
- **Conversion rate.**
- **Avg response time.**

### 6.3. Charts
- Lead conversion funnel.
- Daily/Weekly/Monthly activity.
- Performance vs team.
- Revenue trend.

### 6.4. Tasks
- Today tasks.
- Overdue tasks.
- Upcoming follow-ups.

### 6.5. Manager Dashboard (subtree view)
- Tổng quan toàn nhánh.
- Drill-down theo từng nhân viên.
- So sánh performance.
- Lead distribution map.

### 6.6. Giám đốc Dashboard
- Toàn tenant.
- Phân bổ Lead theo nhánh.
- Revenue by branch.
- Top performers.
- Risk alerts.

---

## 7. Phân quyền Dữ liệu

### 7.1. Lead/Deal Ownership
- Mỗi Lead có `owner_user_id`.
- Mỗi Deal có `owner_user_id` + `team_id` (optional).

### 7.2. RLS Policy
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

### 7.3. Application-Level Filter
- Middleware set `SET LOCAL app.current_tenant_id` + `app.current_user_id`.
- Mọi query đều qua RLS tự động filter.

### 7.4. Shared Lead (Pool)
- Lead không có owner (cho cả team).
- Nhân viên có thể "Claim" → owner = current user.
- Hoặc Manager assign trực tiếp.

---

## 8. Database Schema

### 8.1. Bảng `users`
```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  email TEXT NOT NULL,
  phone TEXT,
  password_hash TEXT,
  full_name TEXT NOT NULL,
  avatar_url TEXT,
  parent_id UUID REFERENCES users(id),
  path LTREE NOT NULL,
  depth INT NOT NULL,
  role TEXT NOT NULL,
  department TEXT[],
  status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE','LOCKED','PENDING')),
  invitation_token_id UUID,
  last_login_at TIMESTAMPTZ,
  last_login_ip INET,
  mfa_enabled BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(tenant_id, email)
);
CREATE INDEX idx_users_path ON users USING GIST(path);
CREATE INDEX idx_users_tenant_parent ON users(tenant_id, parent_id);
```

### 8.2. Bảng `invitations`
```sql
CREATE TABLE invitations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  parent_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  paseto_token TEXT NOT NULL,
  target_role TEXT NOT NULL,
  max_uses INT DEFAULT 1,
  current_uses INT DEFAULT 0,
  max_subtree_depth INT,
  allowed_departments TEXT[],
  expires_at TIMESTAMPTZ NOT NULL,
  revoked BOOLEAN DEFAULT false,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_invitations_token ON invitations(paseto_token);
CREATE INDEX idx_invitations_parent ON invitations(parent_user_id);
```

### 8.3. Bảng `user_roles` (custom roles)
```sql
CREATE TABLE user_roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  role_code TEXT NOT NULL,
  role_name TEXT NOT NULL,
  permissions TEXT[] NOT NULL,    -- ['lead.read.subtree','lead.create', ...]
  inherits_from TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(tenant_id, role_code)
);
```

### 8.4. Bảng `user_role_assignments`
```sql
CREATE TABLE user_role_assignments (
  user_id UUID REFERENCES users(id),
  role_id UUID REFERENCES user_roles(id),
  assigned_by UUID REFERENCES users(id),
  assigned_at TIMESTAMPTZ DEFAULT now(),
  PRIMARY KEY (user_id, role_id)
);
```

### 8.5. Bảng `audit_actions`
```sql
CREATE TABLE audit_actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  actor_id UUID REFERENCES users(id),
  actor_email TEXT,
  action TEXT NOT NULL,    -- 'user.create','user.promote','lead.move', ...
  target_type TEXT,
  target_id UUID,
  old_value JSONB,
  new_value JSONB,
  ip_address INET,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_audit_tenant_actor ON audit_actions(tenant_id, actor_id);
CREATE INDEX idx_audit_target ON audit_actions(target_type, target_id);
```

### 8.6. Bảng `departments`
```sql
CREATE TABLE departments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  name TEXT NOT NULL,
  description TEXT,
  leader_id UUID REFERENCES users(id),
  parent_department_id UUID REFERENCES departments(id),
  created_at TIMESTAMPTZ DEFAULT now()
);
```

---

## 9. Danh sách tính năng (≥ 100)

### 9.1. Quản lý User (1-25)
1. Tạo user mới (form).
2. Mời qua link PASETO.
3. Mời qua email trực tiếp.
4. Bulk invite (CSV upload).
5. Import user từ hệ thống cũ.
6. Export danh sách user.
7. Tìm kiếm user (full-text).
8. Filter theo role, status, branch.
9. Xem profile user.
10. Sửa thông tin user (tên, email, phone, avatar).
11. Đổi role user.
12. Đổi parent (di chuyển subtree).
13. Khóa user (login bị chặn).
14. Mở khóa user.
15. Reset password.
16. Force logout (kill all sessions).
17. Xem session đang active.
19. Xóa user (soft delete).
20. Khôi phục user đã xóa.
21. Merge 2 user (vd: trùng email).
22. Cấu hình user permissions custom.
23. Gán department cho user.
24. Gán label/tag cho user.
25. User activity timeline.

### 9.2. Cây tổ chức (26-45)
26. Xem cây dạng Org Chart (zoomable, drag).
27. Xem cây dạng List (table).
28. Xem cây dạng Tree (collapsible).
29. Drag & drop để sắp xếp (có thể tắt).
31. Search trong cây.
32. Highlight subtree của user hiện tại.
33. Filter theo role.
34. Filter theo department.
35. Xem số lượng user trong subtree.
36. Xem doanh thu subtree.
37. Xem Lead count subtree.
38. Export cây ra OrgChart JSON.
39. So sánh 2 nhánh (performance).
40. Bulk move user.
41. Bulk promote/demote.
42. Bulk deactivate.
43. Bulk delete.
44. Bulk re-assign parent.
45. Org chart print/export PDF.

### 9.3. Phân quyền & Role (46-65)
46. Tạo custom role.
47. Sửa role (thêm/xóa permissions).
48. Xóa role.
49. Clone role.
50. Permission list (predefined 200+ actions).
51. Permission group (vd: Lead, Deal, Report, ...).
52. Role hierarchy (inherits).
53. Override permission per user.
54. Effective permissions viewer.
55. Test permission (giả lập user role).
56. Permission audit log.
57. Department permission.
58. Conditional permission (theo data).
59. Time-based permission (chỉ trong giờ HC).
60. IP-based permission (chỉ từ IP cty).
61. Permission delegation (tạm thời).
62. Permission request workflow.
63. Permission approval (manager duyệt).
64. Role assignment history.
65. Role conflict detection.

### 9.4. Mời & Onboarding (66-85)
66. Tạo link mời NV/TN/QL (chọn role).
67. Cấu hình link (max uses, expiry, allowed dept).
68. QR code cho link.
69. Share link qua Email/Chat/SMS.
70. Theo dõi link status (used/unused).
71. Revoke link.
72. Extend expiry.
73. Auto-reminder trước khi hết hạn.
74. Welcome email template.
75. Welcome message trong app.
76. Onboarding checklist cho NV mới.
77. First-time login wizard.
78. Tour guide trong app.
79. Gán trainer cho NV mới.
80. Sample data cho NV mới (Lead mẫu).
81. Bulk onboarding (CSV).
82. Onboarding progress tracking.
83. Auto deactivate nếu không login 30 ngày.
84. Re-activation link.
85. Offboarding checklist (khi NV nghỉ).

### 9.5. Dashboard & Báo cáo (86-100)
86. Personal dashboard (stats cá nhân).
87. Team dashboard (subtree view).
88. Manager dashboard.
89. Director dashboard (toàn tenant).
90. Real-time stats (SSE, cập nhật mỗi giây).
91. Custom dashboard builder.
92. Drag-drop widgets.
93. Saved views.
94. Share dashboard qua link.
95. Export dashboard (PDF/PNG).
96. Scheduled report (email hàng tuần).
97. Anomaly alert (doanh thu giảm bất thường).
98. Forecast (AI dự đoán doanh thu tháng).
99. Comparison (tháng này vs tháng trước).
100. Cohort analysis.

### 9.6. Profile & Personalization (101-120)
101. Xem profile cá nhân.
102. Sửa profile.
103. Đổi avatar.
104. Đổi password.
105. Cấu hình 2FA (TOTP, WebAuthn).
106. Backup codes cho 2FA.
107. Notification preferences (email, push, SMS).
108. Theme (light/dark/system).
109. Language.
110. Timezone.
111. Working hours config.
112. Auto-reply khi offline.
114. Custom fields cho user profile.
115. Calendar integration (Google/Outlook).
116. Vacation mode (auto redirect Lead).
117. Status (Available/Busy/In meeting/Offline).
118. Custom status message.
119. Avatar frame (gamification).
120. Profile completion score.

---

## 10. API Surface

### 10.1. User Management
```
GET    /api/crm/v1/users
POST   /api/crm/v1/users
GET    /api/crm/v1/users/:id
PATCH  /api/crm/v1/users/:id
DELETE /api/crm/v1/users/:id
POST   /api/crm/v1/users/:id/lock
POST   /api/crm/v1/users/:id/unlock
POST   /api/crm/v1/users/:id/reset-password
POST   /api/crm/v1/users/:id/force-logout
POST   /api/crm/v1/users/:id/promote
POST   /api/crm/v1/users/:id/demote
POST   /api/crm/v1/users/:id/move
```

### 10.2. Org Tree
```
GET    /api/crm/v1/tree/me
GET    /api/crm/v1/tree/subtree/:user_id
GET    /api/crm/v1/tree/ancestors/:user_id
GET    /api/crm/v1/tree/descendants/:user_id
POST   /api/crm/v1/tree/move-subtree
POST   /api/crm/v1/tree/bulk-move
```

### 10.3. Invitations
```
POST   /api/crm/v1/invitations
GET    /api/crm/v1/invitations
DELETE /api/crm/v1/invitations/:id
POST   /api/crm/v1/invitations/:id/revoke
GET    /api/crm/v1/invitations/verify/:token    # Public
POST   /api/crm/v1/invitations/accept            # Public, register form
```

### 10.4. Roles & Permissions
```
GET    /api/crm/v1/roles
POST   /api/crm/v1/roles
PATCH  /api/crm/v1/roles/:id
DELETE /api/crm/v1/roles/:id
GET    /api/crm/v1/permissions
POST   /api/crm/v1/users/:id/permissions
GET    /api/crm/v1/users/:id/effective-permissions
```

### 10.5. Audit & Dashboard
```
GET    /api/crm/v1/audit
GET    /api/crm/v1/dashboard/personal
GET    /api/crm/v1/dashboard/team
GET    /api/crm/v1/dashboard/director
WS     /api/crm/v1/dashboard/realtime
```

---

## 11. UI/UX

### 11.1. Pages
- `/team` – Danh sách user (table view).
- `/team/tree` – Org Chart view.
- `/team/invite` – Tạo link mời.
- `/team/invitations` – Quản lý link đã tạo.
- `/team/roles` – Quản lý roles.
- `/team/departments` – Quản lý phòng ban.
- `/dashboard` – Personal dashboard.
- `/team-dashboard` – Team dashboard (subtree).
- `/director-dashboard` – Director dashboard.
- `/profile` – Cá nhân.
- `/profile/security` – 2FA, sessions.
- `/profile/preferences` – Cấu hình cá nhân.

### 11.2. Components
- **OrgChart** (D3.js + react-d3-tree).
- **UserCard** (avatar + info).
- **TreeNode** (collapsible).
- **InvitationLinkGenerator** (form + QR).
- **PermissionMatrix** (table).
- **RoleBuilder** (drag-drop permissions).
- **AuditTimeline** (vertical timeline).
- **PersonalStats** (cards + chart).
- **TeamPerformanceChart** (bar/line).
- **ActivityFeed** (real-time).
- **BulkActionBar.**
- **MoveUserDialog** (2-step confirm).

### 11.3. Mobile-first
- Responsive trên tablet/mobile.
- Native gesture (swipe để promote/demote).
- Push notification.

---

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-CRM-01 | Tạo link mời trong < 500ms | p95 |
| AC-CRM-02 | Verify token < 50ms | p95 |
| AC-CRM-03 | Subtree query < 100ms trên 100K user | ClickHouse |
| AC-CRM-04 | Move subtree 1000 user trong < 5s | Transaction |
| AC-CRM-05 | RLS chặn 100% cross-tenant access | Security test |

---

**Tiếp theo:** [`docs/04-dynamic-model/README.md`](../04-dynamic-model/README.md) – Trình sinh mô hình doanh nghiệp động (Meta-Schema Engine).