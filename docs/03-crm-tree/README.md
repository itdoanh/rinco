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

**Mục lục mở rộng (phần bổ sung – v1.1)**

12. [Audit – Đánh giá nội dung hiện tại](#12-audit--đánh-giá-nội-dung-hiện-tại)
13. [Edge Cases & Error Scenarios chi tiết](#13-edge-cases--error-scenarios-chi-tiết)
14. [Code Examples chi tiết](#14-code-examples-chi-tiết)
15. [Implementation Roadmap chi tiết](#15-implementation-roadmap-chi-tiết)
16. [Testing Strategy](#16-testing-strategy)
17. [Migration Plan](#17-migration-plan)
18. [Disaster Recovery](#18-disaster-recovery)
19. [Cost Estimation](#19-cost-estimation)
20. [Open Questions / Cần user xác nhận](#20-open-questions--cần-user-xác-nhận)

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

### 8.7. Bảng `lead_activities` (MỚI – v1.1)
```sql
CREATE TABLE lead_activities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  lead_id UUID NOT NULL,
  actor_id UUID REFERENCES users(id),
  activity_type TEXT NOT NULL CHECK (activity_type IN ('CALL','EMAIL','MEETING','NOTE','STATUS_CHANGE','ASSIGN','TASK')),
  description TEXT,
  metadata JSONB,
  due_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_lead_activities_lead ON lead_activities(lead_id, created_at DESC);
CREATE INDEX idx_lead_activities_tenant_actor ON lead_activities(tenant_id, actor_id);
```

### 8.8. Bảng `leads` (Schema đầy đủ hơn – MỚI – v1.1)
```sql
CREATE TABLE leads (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  owner_user_id UUID REFERENCES users(id),
  full_name TEXT,
  email TEXT,
  phone TEXT,
  source TEXT,                -- 'facebook', 'website', 'referral', ...
  utm JSONB,                  -- {source, medium, campaign, term, content}
  fbclid TEXT,
  fbp TEXT,
  status TEXT NOT NULL DEFAULT 'NEW' CHECK (status IN ('NEW','CONTACTED','QUALIFIED','PROPOSAL','WON','LOST','ARCHIVED')),
  score DECIMAL(5,2),         -- AI score 0-100
  estimated_value DECIMAL(15,2),
  custom_fields JSONB DEFAULT '{}',   -- Dynamic schema
  tags TEXT[],
  next_followup_at TIMESTAMPTZ,
  last_contacted_at TIMESTAMPTZ,
  converted_at TIMESTAMPTZ,
  lost_reason TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_leads_tenant_owner ON leads(tenant_id, owner_user_id);
CREATE INDEX idx_leads_tenant_status ON leads(tenant_id, status);
CREATE INDEX idx_leads_score ON leads(tenant_id, score DESC) WHERE score IS NOT NULL;
CREATE INDEX idx_leads_next_followup ON leads(tenant_id, next_followup_at) WHERE next_followup_at IS NOT NULL;
CREATE INDEX idx_leads_custom_fields ON leads USING GIN(custom_fields);
```

### 8.9. Bảng `deals` (MỚI – v1.1)
```sql
CREATE TABLE deals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  lead_id UUID REFERENCES leads(id),
  owner_user_id UUID REFERENCES users(id),
  name TEXT NOT NULL,
  pipeline_stage TEXT NOT NULL DEFAULT 'PROSPECTING',
  amount DECIMAL(15,2),
  currency TEXT DEFAULT 'VND',
  probability INT CHECK (probability BETWEEN 0 AND 100),
  expected_close_date DATE,
  actual_close_date DATE,
  status TEXT NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','WON','LOST','ON_HOLD')),
  lost_reason TEXT,
  products JSONB,
  notes TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_deals_tenant_owner ON deals(tenant_id, owner_user_id);
CREATE INDEX idx_deals_tenant_stage ON deals(tenant_id, pipeline_stage);
CREATE INDEX idx_deals_tenant_status ON deals(tenant_id, status);
```

### 8.10. Bảng `pipelines` (MỚI – v1.1) – cho Dynamic Workflow
```sql
CREATE TABLE pipelines (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  name TEXT NOT NULL,
  stages JSONB NOT NULL,           -- [{name, color, probability, sla_hours}]
  is_default BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(tenant_id, name)
);
```

### 8.11. Bảng `user_sessions` (MỚI – v1.1) – cho Session Kill
```sql
CREATE TABLE user_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL,
  paseto_token_id TEXT UNIQUE NOT NULL,    -- JTI
  device_fingerprint TEXT,
  ip_address INET,
  user_agent TEXT,
  login_at TIMESTAMPTZ DEFAULT now(),
  last_active_at TIMESTAMPTZ DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  revoked BOOLEAN DEFAULT false,
  revoked_at TIMESTAMPTZ,
  revoked_reason TEXT
);
CREATE INDEX idx_user_sessions_user ON user_sessions(user_id) WHERE NOT revoked;
```

### 8.12. Bảng `user_notifications` (MỚI – v1.1) – cho thông báo nội bộ
```sql
CREATE TABLE user_notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  sender_user_id UUID REFERENCES users(id),     -- NULL nếu system notification
  type TEXT NOT NULL,                            -- 'broadcast', 'mention', 'lead_assigned', 'task_due', ...
  title TEXT NOT NULL,
  body TEXT,
  link TEXT,
  metadata JSONB,
  is_read BOOLEAN DEFAULT false,
  read_at TIMESTAMPTZ,
  priority TEXT DEFAULT 'NORMAL' CHECK (priority IN ('LOW','NORMAL','HIGH','URGENT')),
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_user_notif_user ON user_notifications(user_id, is_read, created_at DESC);
CREATE INDEX idx_user_notif_tenant_priority ON user_notifications(tenant_id, priority, created_at);
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

### 9.7. Lead & Deal Management (121-160) (MỚI – v1.1)
121. CRUD Lead (Create, Read, Update, Delete).
122. Lead duplicate detection.
123. Lead merge (vd: 2 Lead trùng email).
125. Lead qualification form.
126. Lead scoring (AI tự động).
127. Lead auto-assignment theo round-robin.
128. Lead pool (chưa có owner) – nhân viên Claim.
129. Lead reassign (manual).
130. Lead transfer (kèm lý do).
131. Bulk import Lead (CSV/Excel).
132. Bulk export Lead.
133. Lead filter/search nâng cao.
134. Lead tags.
135. Lead segmentation.
136. Lead custom fields (theo Dynamic Schema).
137. Lead timeline (activities log).
138. Lead notes (markdown).
139. Lead emails tracking.
140. Lead SMS tracking.
141. Lead call tracking (click-to-call).
142. Lead meeting scheduling.
143. Lead follow-up reminder.
144. Lead lost reason tracking.
145. Lead win/loss analytics.
146. CRUD Deal.
147. Deal pipeline view (Kanban).
148. Deal stage transition rules.
149. Deal probability calculation (AI).
150. Deal revenue forecast.
151. Deal products line items.
152. Deal close date tracking.
153. Deal notifications on stage change.
154. Deal comparison (same customer).
155. Bulk deal update.
156. Deal SLA tracking.
157. Deal won celebration animation.
158. Deal lost retro analysis.
159. Deal source tracking (UTM).
160. Deal attribution report.

### 9.8. Internal Notifications (161-170) (MỚI – v1.1)
161. Giám đốc broadcast thông báo toàn nhân viên.
162. Manager broadcast thông báo cho nhân viên trong nhánh.
163. Manager broadcast cho 1 phòng ban.
164. Manager broadcast cho 1 role cụ thể.
165. Notification theo tenant group.
166. Notification theo region.
167. Notification khẩn cấp bypass quiet hours.
168. Notification lên lịch gửi.
169. Notification digest (gộp theo ngày/tuần).
170. Notification A/B test.

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

### 10.6. Lead & Deal (MỚI – v1.1)
```
GET    /api/crm/v1/leads
POST   /api/crm/v1/leads
GET    /api/crm/v1/leads/:id
PATCH  /api/crm/v1/leads/:id
DELETE /api/crm/v1/leads/:id
POST   /api/crm/v1/leads/:id/assign
POST   /api/crm/v1/leads/:id/claim
POST   /api/crm/v1/leads/:id/merge
POST   /api/crm/v1/leads/import
GET    /api/crm/v1/leads/export

GET    /api/crm/v1/deals
POST   /api/crm/v1/deals
GET    /api/crm/v1/deals/:id
PATCH  /api/crm/v1/deals/:id
POST   /api/crm/v1/deals/:id/stage
POST   /api/crm/v1/deals/:id/win
POST   /api/crm/v1/deals/:id/lost
```

### 10.7. Internal Notifications (MỚI – v1.1)
```
POST   /api/crm/v1/notifications/broadcast       # Giám đốc / Manager gửi
GET    /api/crm/v1/notifications/inbox           # User xem
PATCH  /api/crm/v1/notifications/:id/read        # Mark as read
POST   /api/crm/v1/notifications/:id/dismiss     # Hide
GET    /api/crm/v1/notifications/preferences
PATCH  /api/crm/v1/notifications/preferences
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
- `/leads` – Lead management (MỚI).
- `/leads/:id` – Lead detail (MỚI).
- `/deals` – Deal pipeline (MỚI).
- `/notifications` – Notification inbox (MỚI).
- `/notifications/broadcast` – Compose broadcast (MỚI).

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
- **LeadPipeline** (Kanban) (MỚI).
- **LeadDetailSheet** (MỚI).
- **NotificationComposer** (MỚI).
- **NotificationInbox** (MỚI).

### 11.3. Mobile-first
- Responsive trên tablet/mobile.
- Native gesture (swipe để promote/demote).
- Push notification.

---

# PHẦN MỞ RỘNG (v1.1) – AUDIT, CODE EXAMPLES, EDGE CASES

## 12. Audit – Đánh giá nội dung hiện tại

### 12.1. Phần đã đủ chi tiết ✓

| Mục | Nội dung | Mức đủ |
|-----|---------|--------|
| 2 – Cấu trúc Cây | LTREE schema + query patterns | ✓ |
| 3 – RBAC | 6 roles + permission matrix đầy đủ | ✓ |
| 4 – Invitation Link | PASETO v4 + flow + lifecycle | ✓ |
| 5 – Move Subtree | Algorithm chi tiết + bulk move | ✓ |
| 8 – Database Schema | 6 bảng chính đầy đủ | ✓ (đã bổ sung thêm 6 bảng) |
| 9 – Tính năng | 120 tính năng chia 6 nhóm | ✓ (đã bổ sung 50 tính năng) |

### 12.2. Phần còn thiếu ⚠

| Mục | Vấn đề | Hướng bổ sung |
|-----|--------|---------------|
| 2.4 – Query Patterns | Chưa có materialized view cho subtree stats | Bổ sung MV + ClickHouse mirror |
| 4.4 – Signing Key | "Rotate mỗi 90 ngày" nhưng chưa có ceremony | Bổ sung key rotation procedure |
| 5.4 – Move Algorithm | Chưa có lead/deal migration logic | Bổ sung §14.4 |
| 7 – Phân quyền | Chưa có row-level merge cho cross-team Lead | Bổ sung team policy |
| 9 – Tính năng | Thiếu Lead/Deal (giờ có ở §9.7) | Bổ sung |
| 9.9 | Thiếu Notification nội bộ (giờ có §9.8) | Bổ sung |
| – Tổng thể | Thiếu performance test 100K user | Bổ sung §16.2 |
| – Tổng thể | Thiếu cơ chế backup/restore cây | Bổ sung §18 |

### 12.3. Mâu thuẫn nội bộ ✗

| Vị trí | Mâu thuẫn |
|--------|-----------|
| 3.4 – Department | "Mỗi user thuộc 1 hoặc nhiều phòng ban" nhưng schema `department TEXT[]` không rõ structure |
| 4.5 – UI Flow | "1 click tạo link" nhưng bước 2-3 yêu cầu user nhập nhiều config |
| 5.4 – Move Algorithm | SQL không handle case `old_path = new_path` (no-op) |
| 9.1 – User #17 | "Reset password" yêu cầu YubiKey nhưng user thường (không có YubiKey) không thể reset |
| 9.4 #82 | "Onboarding progress tracking" chưa rõ metrics nào |

### 12.4. Phần cần code example cụ thể 💡

| Mục | Cần code cho |
|-----|--------------|
| 4 – Invitation | PASETO generation + verify trong Go |
| 5 – Move Subtree | Go service + SQL transaction |
| 5 – Bulk Move | Batch processing với progress |
| 8 – Schema | sqlc.yaml + Ent schema |
| 9.1 – User | CRUD handler |
| 10 – API | Echo + Connect-RPC handler |
| 11 – UI | React OrgChart component |

## 13. Edge Cases & Error Scenarios chi tiết

### 13.1. Edge Cases – Tree Structure

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| T1 | Tạo user mà parent_id không tồn tại | FK constraint | 422 ValidationError |
| T2 | Path slug trùng trong cùng parent | UNIQUE constraint trên (tenant_id, parent_id, slug) | Tự động append số: `nv01-2` |
| T3 | User bị xóa nhưng còn con | ON DELETE RESTRICT | Cascade delete hoặc move con trước |
| T4 | Cycle trong cây (A là parent của B, B là parent A) | Pre-check trong Move | 422 InvalidParent |
| T5 | Move subtree tới node trong chính subtree đó | Pre-check (path <@ old_path) | 422 CircularMove |
| T6 | Depth vượt quá giới hạn (vd: max 10) | Check trước insert | 422 MaxDepthExceeded |
| T7 | Bulk move 10000 user gây lock long-running | Concurrent move | Use SKIP LOCKED hoặc queue |
| T8 | Tree khác tenant merge với nhau | Tenant isolation check | Block nếu cross-tenant |
| T9 | Slug chứa ký tự Unicode | Regex validation | 422 InvalidSlug |
| T10 | User có parent_id = NULL (orphan) | Data integrity | Cho phép (root tenant user) |

### 13.2. Edge Cases – Invitation & Onboarding

| # | Edge case | Xử lý |
|---|----------|-------|
| I1 | Token hết hạn trước khi user đăng ký | 410 Gone + link tạo mới |
| I2 | Token đã dùng hết max_uses | 410 Gone |
| I3 | Token bị revoke | Check Valkey blacklist |
| I4 | 2 user cùng click link cùng lúc | Atomic increment current_uses |
| I5 | Email đã tồn tại trong tenant | 409 EmailExists + suggest reset password |
| I6 | Token sign bằng key cũ sau rotation | Verify cả old + new key, support grace period |
| I7 | Link share công khai nhưng có limit thấp | UI warn admin |
| I8 | Manager xóa trong khi link chưa dùng | FK constraint cascade revoke |
| I9 | New user đăng ký nhưng parent đã PENDING | Allow nhưng mark subtree read-only |
| I10 | Welcome email bounce | Retry 3 lần với exponential backoff |

### 13.3. Edge Cases – RBAC

| # | Edge case | Xử lý |
|---|----------|-------|
| R1 | User có 2 roles (multi-role) | Union permissions |
| R2 | Role bị xóa nhưng user vẫn có assignment | ON DELETE SET NULL + audit |
| R3 | Custom role inherit từ role cũ bị xóa | Validate trước khi delete |
| R4 | Permission check race condition | Use row-level lock hoặc cache |
| R5 | Department leader bị thay đổi | Auto-update department.leader_id |
| R6 | Time-based permission (chỉ giờ HC) | Check timezone user |
| R7 | IP-based permission fail (user VPN) | Fallback hoặc block |
| R8 | Permission override conflict | Stack with clear precedence |
| R9 | User vượt quyền (SQL injection bypass RLS) | RLS + audit log + alert |
| R10 | Role bị edit khi user đang login | Force re-check next request |

### 13.4. Edge Cases – Dashboard & Realtime

| # | Edge case | Xử lý |
|---|----------|-------|
| D1 | SSE connection drop | Auto-reconnect với Last-Event-ID |
| D2 | User có subtree 10K user, query chật | Materialized view cache |
| D3 | Dashboard load cùng lúc 100 widget | Lazy load + virtual scroll |
| D4 | Real-time event 1M/giây | Server sampling + client filter |
| D5 | Chart render quá chậm | Pre-render PNG qua headless service |
| D6 | Export PDF timeout (dashboard quá lớn) | Async job + download link |
| D7 | Forecast AI model không load được | Fallback linear regression |
| D8 | Anomaly false positive | User feedback loop + tune threshold |
| D9 | Timezone user khác timezone server | Convert tất cả về user TZ |
| D10 | Date range quá lớn (5 năm) | Limit + suggest shorter |

### 13.5. Edge Cases – Lead/Deal (MỚI – v1.1)

| # | Edge case | Xử lý |
|---|----------|-------|
| L1 | Import CSV 100K Lead | Background job + progress |
| L2 | Lead email trùng với Lead đã có | Auto-merge hoặc flag for review |
| L3 | Lead assign cho user không có subtree access | 403 Forbidden |
| L4 | Deal stage transition skip rule | Validate transition graph |
| L5 | Won Deal nhưng không có close date | Auto-set to today + confirm |
| L6 | Bulk update 10K Lead | Background job |
| L7 | Custom field schema sai | Validation error per row |
| L8 | Lead có 2 owner khác nhau (race) | SELECT FOR UPDATE |
| L9 | Activity log bị truncate (>1000) | Pagination + archive |
| L10 | AI score chưa sẵn sàng | Fallback score = 50 (medium) |

### 13.6. Edge Cases – Internal Notifications (MỚI – v1.1)

| # | Edge case | Xử lý |
|---|----------|-------|
| N1 | Broadcast tới 10K user cùng lúc | Batch send 100/batch + retry |
| N2 | User offline khi gửi | Queue + push notification |
| N3 | Same notification gửi 2 lần (race) | Dedupe by (user_id, type, target_id) |
| N4 | Notification body quá dài (>4KB) | Truncate + "show more" |
| N5 | Link trong notification trỏ tới deleted entity | Show "deleted" message |
| N6 | Quiet hours override cho urgent | Allow urgent always |
| N7 | Digest job fail | Manual retry + alert |
| N8 | User block sender | Filter notifications |
| N9 | Notification spam (100/giờ) | Aggregate to digest |
| N10 | Translate missing locale | Fallback to default |

## 14. Code Examples chi tiết

### 14.1. PASETO Invitation Service (Go)

```go
// services/tree-org/internal/service/invitation.go
package service

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/o1egl/paseto"
    "golang.org/x/crypto/nacl/secretbox"

    "rinco/tree-org/internal/domain"
    "rinco/tree-org/internal/repository"
)

type InvitationService struct {
    pasetoKey    paseto.V4SymmetricKey
    invitationRepo repository.InvitationRepository
    userRepo     repository.UserRepository
    valkey       *ValkeyClient
}

type CreateInvitationInput struct {
    TenantID         string
    ParentUserID     string
    TargetRole       string
    MaxUses          int
    MaxSubtreeDepth  int
    AllowedDepartments []string
    ExpiresIn        time.Duration
    CreatedBy        string
}

func (s *InvitationService) Create(ctx context.Context, input CreateInvitationInput) (*domain.Invitation, string, error) {
    // 1. Validate parent user
    parent, err := s.userRepo.GetByID(ctx, input.TenantID, uuid.MustParse(input.ParentUserID))
    if err != nil {
        return nil, "", ErrParentNotFound
    }

    // 2. Validate target role (must be lower than parent's role)
    if !s.canInviteRole(parent.Role, input.TargetRole) {
        return nil, "", ErrInsufficientPrivilege
    }

    // 3. Generate PASETO token
    jti := uuid.NewV7()
    now := time.Now()
    expiresAt := now.Add(input.ExpiresIn)

    token := paseto.NewToken()
    token.SetIssuer("rinco-crm")
    token.SetSubject("invitation")
    token.SetJti(jti.String())
    token.SetIssuedAt(now)
    token.SetExpiration(expiresAt)
    token.SetString("tenant_id", input.TenantID)
    token.SetString("parent_user_id", input.ParentUserID)
    token.SetString("target_role", input.TargetRole)
    token.Set("max_uses", input.MaxUses)
    token.Set("current_uses", 0)
    token.Set("max_subtree_depth", input.MaxSubtreeDepth)
    token.Set("allowed_departments", input.AllowedDepartments)

    encrypted := token.Encrypt(s.pasetoKey)
    encodedToken := base64.URLEncoding.EncodeToString(encrypted)

    // 4. Save to DB
    invitation := &domain.Invitation{
        ID:                   uuid.New(),
        TenantID:             uuid.MustParse(input.TenantID),
        ParentUserID:         parent.ID,
        PasetoToken:          encodedToken,
        TargetRole:           input.TargetRole,
        MaxUses:              input.MaxUses,
        CurrentUses:          0,
        MaxSubtreeDepth:      input.MaxSubtreeDepth,
        AllowedDepartments:   input.AllowedDepartments,
        ExpiresAt:            expiresAt,
        Revoked:              false,
        CreatedBy:            uuid.MustParse(input.CreatedBy),
    }

    if err := s.invitationRepo.Create(ctx, invitation); err != nil {
        return nil, "", err
    }

    // 5. Generate URL
    baseURL := os.Getenv("FRONTEND_BASE_URL")
    url := fmt.Sprintf("%s/invite/accept?token=%s", baseURL, encodedToken)

    return invitation, url, nil
}

func (s *InvitationService) Verify(ctx context.Context, encodedToken string) (*VerificationResult, error) {
    // 1. Decode
    encrypted, err := base64.URLEncoding.DecodeString(encodedToken)
    if err != nil {
        return nil, ErrInvalidToken
    }

    // 2. Check Valkey blacklist (revoked tokens)
    jti, _ := s.valkey.Get(ctx, "invitation:revoked:"+extractJTI(encrypted))
    if jti != "" {
        return nil, ErrTokenRevoked
    }

    // 3. Decrypt PASETO
    var token paseto.Token
    if err := token.Decrypt(encrypted, s.pasetoKey); err != nil {
        return nil, ErrInvalidToken
    }

    // 4. Verify claims
    if token.IsExpired() {
        return nil, ErrTokenExpired
    }

    tenantID, _ := token.GetString("tenant_id")
    parentUserID, _ := token.GetString("parent_user_id")
    targetRole, _ := token.GetString("target_role")
    maxUses, _ := token.Get("max_uses")
    currentUses, _ := token.Get("current_uses")
    maxSubtreeDepth, _ := token.Get("max_subtree_depth")
    allowedDepts, _ := token.Get("allowed_departments")

    // 5. Check usage count
    if currentUses.(int64) >= maxUses.(int64) {
        return nil, ErrTokenMaxUsesReached
    }

    // 6. Check tenant status (must be ACTIVE)
    tenant, _ := s.tenantRepo.GetByID(ctx, uuid.MustParse(tenantID))
    if tenant.Status != "ACTIVE" {
        return nil, ErrTenantSuspended
    }

    return &VerificationResult{
        TenantID:            tenantID,
        ParentUserID:        parentUserID,
        TargetRole:          targetRole,
        MaxSubtreeDepth:     maxSubtreeDepth.(int64),
        RemainingUses:       maxUses.(int64) - currentUses.(int64),
        AllowedDepartments:  allowedDepts.([]string),
    }, nil
}

func (s *InvitationService) Accept(ctx context.Context, encodedToken string, input AcceptInvitationInput) (*domain.User, error) {
    // 1. Verify token
    verification, err := s.Verify(ctx, encodedToken)
    if err != nil {
        return nil, err
    }

    // 2. Atomic increment current_uses
    var newUser *domain.User
    err = s.invitationRepo.AtomicIncrementUsage(ctx, extractJTI([]byte(encodedToken)), func() error {
        // 3. Create user with parent
        parent, err := s.userRepo.GetByID(ctx, verification.TenantID, uuid.MustParse(verification.ParentUserID))
        if err != nil {
            return err
        }

        // Generate path slug
        slug, err := s.generateUniqueSlug(ctx, verification.TenantID, parent.ID, input.FullName, input.Email)
        if err != nil {
            return err
        }

        // Build path
        path := parent.Path
        if path != "root" {
            path = path + "." + slug
        } else {
            path = "root." + slug
        }

        // Hash password
        hashedPassword, _ := argon2id.CreateHash(input.Password, argon2id.DefaultParams)

        newUser = &domain.User{
            ID:           uuid.New(),
            TenantID:     uuid.MustParse(verification.TenantID),
            Email:        input.Email,
            Phone:        input.Phone,
            FullName:     input.FullName,
            PasswordHash: hashedPassword,
            ParentID:     &parent.ID,
            Path:         path,
            Depth:        nlevel(path),
            Role:         verification.TargetRole,
            Department:   input.Department,
            Status:       "ACTIVE",
        }

        if err := s.userRepo.Create(ctx, newUser); err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    // 4. Audit log
    s.auditLog.Record(ctx, AuditEntry{
        TenantID:   newUser.TenantID,
        ActorID:    &newUser.ID, // Self-registered
        Action:     "user.create",
        TargetType: "user",
        TargetID:   newUser.ID,
        Payload: map {
            "via": "invitation_link",
            "parent_user_id": verification.ParentUserID,
            "target_role": verification.TargetRole,
        },
    })

    return newUser, nil
}
```

### 14.2. Move Subtree Service (Go + SQL)

```go
// services/tree-org/internal/service/tree.go
package service

import (
    "context"
    "errors"
    "fmt"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"

    "rinco/tree-org/internal/domain"
)

type TreeService struct {
    db       *pgxpool.Pool
    userRepo repository.UserRepository
    auditLog AuditLogger
}

type MoveSubtreeInput struct {
    TenantID      string
    SourceUserID  string
    TargetParentID string
    MovedBy       string
    Reason        string
}

type MoveSubtreeResult struct {
    SourceUserID string
    NewPath      string
    AffectedUsers int
    AffectedLeads int
    AffectedDeals int
}

func (s *TreeService) MoveSubtree(ctx context.Context, input MoveSubtreeInput) (*MoveSubtreeResult, error) {
    // Validate
    if input.SourceUserID == input.TargetParentID {
        return nil, ErrSelfMove
    }

    tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)

    // 1. Get source user
    var source domain.User
    err = tx.QueryRow(ctx, `
        SELECT id, tenant_id, parent_id, path, depth, role, status
        FROM users
        WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
        FOR UPDATE
    `, input.SourceUserID, input.TenantID).Scan(&source.ID, &source.TenantID, &source.ParentID, &source.Path, &source.Depth, &source.Role, &source.Status)
    if err != nil {
        return nil, ErrSourceNotFound
    }

    // 2. Get target parent
    var targetParent domain.User
    err = tx.QueryRow(ctx, `
        SELECT id, tenant_id, parent_id, path, depth, role, status
        FROM users
        WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
        FOR UPDATE
    `, input.TargetParentID, input.TenantID).Scan(&targetParent.ID, &targetParent.TenantID, &targetParent.ParentID, &targetParent.Path, &targetParent.Depth, &targetParent.Role, &targetParent.Status)
    if err != nil {
        return nil, ErrTargetNotFound
    }

    // 3. Pre-check: target not in source's subtree (would create cycle)
    targetPath := targetParent.Path
    if targetPath == source.Path || targetPath.IsDescendantOf(source.Path) {
        return nil, ErrCircularMove
    }

    // 4. Pre-check: max depth
    newDepth := targetParent.Depth + 1
    var maxDepth int
    err = tx.QueryRow(ctx, `
        SELECT COALESCE(MAX(nlevel(path)), 0) FROM users
        WHERE tenant_id = $1 AND path <@ $2
    `, input.TenantID, source.Path).Scan(&maxDepth)

    finalDepth := newDepth + (maxDepth - source.Depth) - 1
    if finalDepth > 10 { // Max 10 levels
        return nil, ErrMaxDepthExceeded
    }

    // 5. Count affected entities
    var affectedUsers int
    err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND path <@ $2`,
        input.TenantID, source.Path).Scan(&affectedUsers)

    var affectedLeads int64
    err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM leads WHERE tenant_id = $1 AND owner_user_id IN (
        SELECT id FROM users WHERE path <@ $2
    )`, input.TenantID, source.Path).Scan(&affectedLeads)

    var affectedDeals int64
    err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM deals WHERE tenant_id = $1 AND owner_user_id IN (
        SELECT id FROM users WHERE path <@ $2
    )`, input.TenantID, source.Path).Scan(&affectedDeals)

    // 6. Move source user
    oldPath := source.Path
    var newSourcePath string
    if targetParent.Path == "root" {
        newSourcePath = "root." + extractLastSegment(oldPath)
    } else {
        newSourcePath = targetParent.Path + "." + extractLastSegment(oldPath)
    }

    _, err = tx.Exec(ctx, `
        UPDATE users SET
            parent_id = $1,
            path = $2::ltree,
            depth = nlevel($2::ltree)
        WHERE id = $3
    `, targetParent.ID, newSourcePath, source.ID)
    if err != nil {
        return nil, err
    }

    // 7. Update entire subtree
    _, err = tx.Exec(ctx, `
        UPDATE users
        SET path = $1::ltree || subpath(path, nlevel($2::ltree))
        WHERE path <@ $2::ltree AND id != $3
    `, newSourcePath, oldPath, source.ID)
    if err != nil {
        return nil, err
    }

    // 8. Commit
    if err := tx.Commit(ctx); err != nil {
        return nil, err
    }

    // 9. Audit log (after commit)
    s.auditLog.Record(ctx, AuditEntry{
        TenantID:   uuid.MustParse(input.TenantID),
        ActorID:    uuid.MustParse(input.MovedBy),
        Action:     "tree.move_subtree",
        TargetType: "user",
        TargetID:   source.ID,
        OldValue: map[string]any{
            "path": oldPath,
            "parent_id": source.ParentID,
        },
        NewValue: map[string]any{
            "path": newSourcePath,
            "parent_id": targetParent.ID,
            "reason": input.Reason,
        },
    })

    // 10. Invalidate caches
    s.valkey.Invalidate(ctx, fmt.Sprintf("user:subtree:%s", input.TenantID))

    return &MoveSubtreeResult{
        SourceUserID:  source.ID.String(),
        NewPath:       newSourcePath,
        AffectedUsers: affectedUsers,
        AffectedLeads: int(affectedLeads),
        AffectedDeals: int(affectedDeals),
    }, nil
}
```

### 14.3. RBAC Middleware

```go
// services/crm-core/internal/middleware/rbac.go
package middleware

import (
    "context"
    "errors"

    "github.com/labstack/echo/v4"

    "rinco/crm-core/internal/domain"
)

type Permission string

const (
    PermLeadRead   Permission = "lead.read"
    PermLeadCreate Permission = "lead.create"
    PermLeadUpdate Permission = "lead.update"
    PermLeadDelete Permission = "lead.delete"
    PermLeadAssign Permission = "lead.assign"
    PermUserRead   Permission = "user.read"
    PermUserCreate Permission = "user.create"
    PermUserUpdate Permission = "user.update"
    // ... thêm 200+ permissions
)

var DefaultRolePermissions = map[string][]Permission{
    "GD": { /* ALL */ },
    "PGD": { /* most except billing config */ },
    "QL": { PermLeadRead, PermLeadCreate, PermLeadUpdate, PermLeadDelete, PermLeadAssign, PermUserRead, PermUserCreate },
    "TN": { PermLeadRead, PermLeadCreate, PermLeadUpdate, PermLeadAssign, PermUserRead },
    "NV": { PermLeadRead, PermLeadCreate, PermLeadUpdate },
    "VIEW": { PermLeadRead, PermUserRead },
}

type RBACMiddleware struct {
    userRepo     repository.UserRepository
    customRoles  repository.CustomRoleRepository
}

func (m *RBACMiddleware) Require(perm Permission) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            ctx := c.Request().Context()

            user, ok := ctx.Value("user").(*domain.User)
            if !ok {
                return echo.NewHTTPError(401, "unauthenticated")
            }

            // 1. Check default role permissions
            if !m.checkDefaultRole(user.Role, perm) {
                // 2. Check custom role
                if !m.checkCustomRoles(ctx, user, perm) {
                    return echo.NewHTTPError(403, "permission denied")
                }
            }

            return next(c)
        }
    }
}

func (m *RBACMiddleware) checkDefaultRole(role string, perm Permission) bool {
    perms, ok := DefaultRolePermissions[role]
    if !ok {
        return false
    }
    for _, p := range perms {
        if p == perm {
            return true
        }
    }
    return false
}

func (m *RBACMiddleware) checkCustomRoles(ctx context.Context, user *domain.User, perm Permission) bool {
    customRoles, err := m.customRoles.GetByUser(ctx, user.ID)
    if err != nil {
        return false
    }
    for _, role := range customRoles {
        for _, p := range role.Permissions {
            if string(p) == string(perm) {
                return true
            }
        }
    }
    return false
}
```

### 14.4. Move Subtree Preview (TypeScript React)

```tsx
// apps/crm/components/MoveUserDialog.tsx
import React, { useState } from 'react';
import { AlertTriangle, Move, X } from 'lucide-react';

interface MoveUserDialogProps {
  userId: string;
  currentPath: string;
  affectedUsers: number;
  affectedLeads: number;
  affectedDeals: number;
  onClose: () => void;
  onConfirm: (targetParentId: string, reason: string) => Promise<void>;
}

export const MoveUserDialog: React.FC<MoveUserDialogProps> = ({
  userId, currentPath, affectedUsers, affectedLeads, affectedDeals, onClose, onConfirm
}) => {
  const [step, setStep] = useState<'select' | 'preview' | 'confirm' | 'processing'>('select');
  const [targetParentId, setTargetParentId] = useState<string>('');
  const [reason, setReason] = useState<string>('');

  return (
    <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center">
      <div className="bg-white dark:bg-slate-800 rounded-lg shadow-xl max-w-2xl w-full">
        <div className="p-6 border-b flex items-center justify-between">
          <h2 className="text-xl font-bold flex items-center gap-2">
            <Move className="w-5 h-5" />
            Di chuyển nhân sự
          </h2>
          <button onClick={onClose}><X /></button>
        </div>

        <div className="p-6">
          {step === 'select' && (
            <div>
              <label>Chọn parent mới:</label>
              <UserTreePicker
                excludeUserId={userId}
                onSelect={setTargetParentId}
              />
              <button
                className="mt-4 px-4 py-2 bg-blue-500 text-white rounded"
                onClick={() => setStep('preview')}
                disabled={!targetParentId}
              >
                Tiếp theo →
              </button>
            </div>
          )}

          {step === 'preview' && (
            <div className="space-y-4">
              <div className="bg-amber-50 border border-amber-200 p-4 rounded">
                <AlertTriangle className="inline w-5 h-5 text-amber-600" />
                <strong>Cảnh báo:</strong> Thao tác này sẽ ảnh hưởng:
                <ul className="mt-2 list-disc ml-6">
                  <li>{affectedUsers} nhân viên trong subtree</li>
                  <li>{affectedLeads} Lead đang được assign</li>
                  <li>{affectedDeals} Deal đang active</li>
                </ul>
              </div>

              <div>
                <label>Lý do di chuyển (bắt buộc, audit):</label>
                <textarea
                  value={reason}
                  onChange={e => setReason(e.target.value)}
                  className="w-full p-2 border rounded mt-1"
                  minLength={10}
                  maxLength={500}
                />
              </div>

              <div className="flex justify-between">
                <button onClick={() => setStep('select')}>← Quay lại</button>
                <button
                  className="px-4 py-2 bg-red-500 text-white rounded"
                  onClick={() => setStep('confirm')}
                  disabled={reason.length < 10}
                >
                  Tiếp tục →
                </button>
              </div>
            </div>
          )}

          {step === 'confirm' && (
            <div className="space-y-4">
              <p>Bạn chắc chắn muốn di chuyển?</p>
              <div className="bg-slate-100 p-3 rounded">
                <div>Từ: <code>{currentPath}</code></div>
                <div>Đến: <code>{targetParentId}</code></div>
                <div>Lý do: {reason}</div>
              </div>
              <div className="flex justify-between">
                <button onClick={() => setStep('preview')}>← Quay lại</button>
                <button
                  className="px-4 py-2 bg-red-600 text-white rounded font-bold"
                  onClick={async () => {
                    setStep('processing');
                    await onConfirm(targetParentId, reason);
                  }}
                >
                  XÁC NHẬN DI CHUYỂN
                </button>
              </div>
            </div>
          )}

          {step === 'processing' && (
            <div className="text-center">
              <div className="animate-spin w-8 h-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto" />
              <p className="mt-4">Đang thực hiện...</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
```

### 14.5. Internal Notification Service

```go
// services/crm-core/internal/service/notification.go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"

    "rinco/crm-core/internal/domain"
)

type NotificationService struct {
    notifRepo NotificationRepository
    userRepo  UserRepository
    pushSvc   PushNotificationService
    emailSvc  EmailService
    sseBroker SSEBroker
}

type BroadcastInput struct {
    TenantID    string
    SenderID    string          // User gửi (NULL = system)
    Title       string
    Body        string
    Link        string
    TargetFilter TargetFilter   // by role, dept, branch, ...
    Priority    string          // 'LOW', 'NORMAL', 'HIGH', 'URGENT'
    ScheduleAt  *time.Time      // NULL = immediate
}

type TargetFilter struct {
    Role            string   // 'NV', 'TN', 'QL', ...
    Department      string
    BranchPath      string   // subtree path
    UserIDs         []string // explicit list
    IncludeSender   bool     // có gửi cho chính sender không
}

func (s *NotificationService) Broadcast(ctx context.Context, input BroadcastInput) (*BroadcastResult, error) {
    // 1. Validate sender permission (must be GD or higher)
    sender, err := s.userRepo.GetByID(ctx, input.TenantID, uuid.MustParse(input.SenderID))
    if err != nil {
        return nil, ErrSenderNotFound
    }
    if !sender.CanBroadcast() {
        return nil, ErrNoPermission
    }

    // 2. Resolve recipients
    recipients, err := s.resolveRecipients(ctx, input.TenantID, input.TargetFilter)
    if err != nil {
        return nil, err
    }

    if len(recipients) == 0 {
        return &BroadcastResult{RecipientCount: 0}, nil
    }

    // 3. Handle schedule
    if input.ScheduleAt != nil && input.ScheduleAt.After(time.Now()) {
        // Queue for later
        return s.scheduleBroadcast(ctx, input, recipients)
    }

    // 4. Send to each recipient
    var successCount, failCount int
    var errors []string
    for _, recipient := range recipients {
        if !input.TargetFilter.IncludeSender && recipient.ID == sender.ID {
            continue
        }

        // Check recipient's quiet hours
        if input.Priority != "URGENT" && s.isInQuietHours(recipient) {
            s.queueForDigest(recipient, input)
            continue
        }

        // Send
        if err := s.sendToRecipient(ctx, recipient, input); err != nil {
            failCount++
            errors = append(errors, fmt.Sprintf("%s: %s", recipient.Email, err.Error()))
        } else {
            successCount++
        }
    }

    return &BroadcastResult{
        RecipientCount: len(recipients),
        SuccessCount:   successCount,
        FailCount:      failCount,
        Errors:         errors,
    }, nil
}

func (s *NotificationService) resolveRecipients(ctx context.Context, tenantID string, filter TargetFilter) ([]domain.User, error) {
    var users []domain.User

    // Build query based on filter
    if len(filter.UserIDs) > 0 {
        return s.userRepo.GetByIDs(ctx, tenantID, filter.UserIDs)
    }

    query := `SELECT id, tenant_id, email, full_name, role, path, timezone
              FROM users
              WHERE tenant_id = $1 AND status = 'ACTIVE' AND deleted_at IS NULL`
    args := []interface{}{tenantID}

    if filter.Role != "" {
        query += " AND role = $" + fmt.Sprint(len(args)+1)
        args = append(args, filter.Role)
    }

    if filter.Department != "" {
        query += " AND $" + fmt.Sprint(len(args)+1) + " = ANY(department)"
        args = append(args, filter.Department)
    }

    if filter.BranchPath != "" {
        query += " AND path <@ $" + fmt.Sprint(len(args)+1) + "::ltree"
        args = append(args, filter.BranchPath)
    }

    rows, err := s.db.Query(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var u domain.User
        if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.Role, &u.Path, &u.Timezone); err != nil {
            return nil, err
        }
        users = append(users, u)
    }

    return users, nil
}

func (s *NotificationService) sendToRecipient(ctx context.Context, recipient domain.User, input BroadcastInput) error {
    notification := domain.UserNotification{
        ID:            uuid.New(),
        TenantID:      uuid.MustParse(input.TenantID),
        UserID:        recipient.ID,
        SenderUserID:  uuidPtr(input.SenderID),
        Type:          "broadcast",
        Title:         input.Title,
        Body:          input.Body,
        Link:          input.Link,
        Priority:      input.Priority,
    }

    // 1. Persist
    if err := s.notifRepo.Create(ctx, &notification); err != nil {
        return err
    }

    // 2. Push via SSE (in-app real-time)
    s.sseBroker.Publish(recipient.ID.String(), map[string]any{
        "id":         notification.ID,
        "title":      input.Title,
        "body":       input.Body,
        "link":       input.Link,
        "priority":   input.Priority,
        "created_at": notification.CreatedAt,
    })

    // 3. Email notification (nếu user preference)
    if recipient.NotificationPrefs.Email {
        // queue email send
        s.emailSvc.QueueEmail(EmailInput{
            To:      recipient.Email,
            Subject: input.Title,
            Body:    input.Body,
        })
    }

    // 4. Push notification (nếu có device)
    if recipient.NotificationPrefs.Push {
        for _, device := range recipient.Devices {
            s.pushSvc.Send(device, input.Title, input.Body, input.Link)
        }
    }

    return nil
}

func uuidPtr(s string) *uuid.UUID {
    if s == "" {
        return nil
    }
    u := uuid.MustParse(s)
    return &u
}
```

### 14.6. Realtime Dashboard SSE Handler

```go
// services/crm-core/internal/api/dashboard_realtime.go
package api

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/labstack/echo/v4"

    "rinco/crm-core/internal/domain"
)

func (h *Handler) StreamDashboard(c echo.Context) error {
    ctx := c.Request().Context()
    user := getUserFromContext(ctx)
    tenantID := getTenantIDFromContext(ctx)

    c.Response().Header().Set("Content-Type", "text/event-stream")
    c.Response().Header().Set("Cache-Control", "no-cache")
    c.Response().Header().Set("Connection", "keep-alive")
    c.Response().Header().Set("X-Accel-Buffering", "no")

    // 1. Initial state
    initialState, err := h.getInitialDashboardState(ctx, tenantID, user)
    if err != nil {
        return err
    }
    writeSSE(c, "initial", initialState)

    // 2. Subscribe to NATS for updates
    sub, err := h.nats.Subscribe(fmt.Sprintf("dashboard.%s.%s", tenantID, user.ID), func(msg *nats.Msg) {
        var event DashboardEvent
        json.Unmarshal(msg.Data, &event)
        writeSSE(c, "update", event)
    })
    if err != nil {
        return err
    }
    defer sub.Unsubscribe()

    // 3. Heartbeat
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            fmt.Fprintf(c.Response(), ": heartbeat\n\n")
            c.Response().Flush()
        }
    }
}

func writeSSE(c echo.Context, event string, data interface{}) {
    jsonData, _ := json.Marshal(data)
    fmt.Fprintf(c.Response(), "event: %s\ndata: %s\n\n", event, jsonData)
    c.Response().Flush()
}

type DashboardEvent struct {
    Type     string      ` `json:"type"``
    Payload  interface{} ` `json:"payload"``
    TS       int64       ` `json:"ts"``
}
```

## 15. Implementation Roadmap chi tiết

### 15.1. Phase 1 – Core CRM (Tuần 1–6)

#### Tuần 1: Database Foundation
- [ ] Migration: users, invitations, user_roles, audit_actions, departments.
- [ ] sqlc.yaml config + generate.
- [ ] Indexes: GIST (path), B-tree (tenant_id, parent_id).

#### Tuần 2: User Service Skeleton
- [ ] Service `crm-core` (Go + Ent).
- [ ] CRUD users.
- [ ] Tenant isolation middleware.

#### Tuần 3: Tree Service
- [ ] Move subtree.
- [ ] Bulk move.
- [ ] Path utilities (ltree operations).

#### Tuần 4: Invitation Service
- [ ] PASETO generation.
- [ ] Verify + Accept.
- [ ] Token rotation.

#### Tuần 5: RBAC
- [ ] Default roles + permissions.
- [ ] Custom roles.
- [ ] Department.
- [ ] RBAC middleware.

#### Tuần 6: Audit + Dashboard
- [ ] Audit logging wrapper.
- [ ] Personal dashboard.
- [ ] Team dashboard.

**Acceptance Gate:**
- Tạo tenant → tạo user root → mời 2 manager → mời 4 nhân viên.
- Subtree query < 100ms trên 10K user.
- Move subtree 1000 user trong < 5s.

### 15.2. Phase 2 – Lead/Deal (Tuần 7–10)

#### Tuần 7: Lead Service
- [ ] CRUD Lead.
- [ ] Lead scoring integration (AI).
- [ ] Lead duplicate detection.

#### Tuần 8: Deal Pipeline
- [ ] Pipeline stages.
- [ ] Deal CRUD.
- [ ] Kanban view.

#### Tuần 9: Lead/Deal Activities
- [ ] Activity timeline.
- [ ] Notes + tasks.

#### Tuần 10: Lead Auto-Assignment
- [ ] Round-robin.
- [ ] Skill-based matching.
- [ ] Workload balancing.

**Acceptance Gate:**
- 100K Lead import < 10 phút.
- Kanban view lag < 100ms.
- Lead auto-assign < 50ms.

### 15.3. Phase 3 – Notification & Advanced (Tuần 11–14)

#### Tuần 11: Notification System
- [ ] User notifications table.
- [ ] Broadcast API.
- [ ] SSE inbox.

#### Tuần 12: Email + Push Integration
- [ ] Email service (SendGrid/SES).
- [ ] Push service (FCM/APNs).
- [ ] Quiet hours.

#### Tuần 13: Realtime Dashboard
- [ ] SSE endpoint.
- [ ] WebSocket fallback.
- [ ] Live metrics.

#### Tuần 14: Polish
- [ ] Performance tuning.
- [ ] Security audit.
- [ ] Load test.

**Acceptance Gate:**
- Broadcast 10K user < 30s.
- Dashboard real-time < 1s update.
- 100K concurrent users không lag.

### 15.4. Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-CRM-01 | Tạo link mời trong < 500ms | p95 |
| AC-CRM-02 | Verify token < 50ms | p95 |
| AC-CRM-03 | Subtree query < 100ms trên 100K user | ClickHouse |
| AC-CRM-04 | Move subtree 1000 user trong < 5s | Transaction |
| AC-CRM-05 | RLS chặn 100% cross-tenant access | Security test |
| AC-CRM-06 | Broadcast 10K user < 30s | p95 |
| AC-CRM-07 | Lead create < 50ms | p95 |
| AC-CRM-08 | Deal pipeline load < 200ms | p95 |
| AC-CRM-09 | Realtime dashboard update < 1s | p95 |
| AC-CRM-10 | Tree depth unlimited (test 20+) | Manual |

## 16. Testing Strategy

### 16.1. Unit Test Targets

| Module | Coverage |
|--------|----------|
| InvitationService | ≥ 90% |
| TreeService.Move | ≥ 85% |
| RBAC middleware | ≥ 95% |
| NotificationService | ≥ 85% |
| UserService | ≥ 85% |
| LeadService | ≥ 80% |
| DealService | ≥ 80% |

### 16.2. Integration Test (Critical Scenarios)

```go
// services/tree-org/test/integration/tree_test.go
package integration_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/require"

    "rinco/tree-org/test/helpers"
)

func TestMoveSubtree_1000Users_LessThan5Seconds(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping load test")
    }

    ctx := context.Background()
    h := helpers.NewTestHarness(t)
    defer h.Cleanup()

    // Setup: create 1 root + 5 children + each child has 200 grandchildren = 1006 users
    root := h.CreateUser(ctx, "root", "GD", nil)
    children := make([]*domain.User, 5)
    for i := range children {
        children[i] = h.CreateUser(ctx, fmt.Sprintf("ql%d", i), "QL", root)
    }
    for _, child := range children {
        for j := 0; j < 200; j++ {
            h.CreateUser(ctx, fmt.Sprintf("nv%d", j), "NV", child)
        }
    }

    // Move subtree: child[0] from root to child[1]
    start := time.Now()
    result, err := h.TreeService.MoveSubtree(ctx, MoveSubtreeInput{
        TenantID:       root.TenantID.String(),
        SourceUserID:   children[0].ID.String(),
        TargetParentID: children[1].ID.String(),
        MovedBy:        root.ID.String(),
        Reason:         "load test",
    })
    elapsed := time.Since(start)

    require.NoError(t, err)
    require.Equal(t, 201, result.AffectedUsers) // child[0] + 200 grandchildren
    require.Less(t, elapsed, 5*time.Second)

    // Verify path updated
    moved, _ := h.UserRepo.GetByID(ctx, root.TenantID, children[0].ID)
    expectedPath := children[1].Path + "." + extractLastSegment(children[0].Path)
    require.Equal(t, expectedPath, moved.Path)
}

func TestMoveSubtree_CircularMove_Rejected(t *testing.T) {
    ctx := context.Background()
    h := helpers.NewTestHarness(t)
    defer h.Cleanup()

    root := h.CreateUser(ctx, "root", "GD", nil)
    ql := h.CreateUser(ctx, "ql01", "QL", root)
    nv := h.CreateUser(ctx, "nv01", "NV", ql)

    // Try to move root into nv (would create cycle)
    _, err := h.TreeService.MoveSubtree(ctx, MoveSubtreeInput{
        TenantID:       root.TenantID.String(),
        SourceUserID:   root.ID.String(),
        TargetParentID: nv.ID.String(),
        MovedBy:        root.ID.String(),
        Reason:         "should fail",
    })

    require.Error(t, err)
    require.Equal(t, ErrCircularMove, err)
}

func TestRBAC_BlockCrossTenant(t *testing.T) {
    ctx := context.Background()
    h := helpers.NewTestHarness(t)
    defer h.Cleanup()

    userA := h.CreateUserInTenant(ctx, "A", "GD")
    userB := h.CreateUserInTenant(ctx, "B", "GD")

    // userA tries to access user in tenant B
    _, err := h.UserService.GetByID(ctx, userB.TenantID, userB.ID, userA)
    require.Error(t, err)
    require.Equal(t, ErrForbidden, err)
}
```

### 16.3. E2E Test (Playwright)

```typescript
// apps/crm/e2e/invitation.spec.ts
import { test, expect } from '@playwright/test';

test('Manager invites Employee via link', async ({ page, context }) => {
  // 1. Login as Manager
  await page.goto('/login');
  await page.fill('input[name=email]', 'manager@apex.vn');
  await page.fill('input[name=password]', 'password123');
  await page.click('button[type=submit]');

  // 2. Navigate to invite page
  await page.click('a:has-text("Team")');
  await page.click('button:has-text("Mời thành viên")');

  // 3. Fill form
  await page.selectOption('select[name=role]', 'NV');
  await page.fill('input[name=max_uses]', '5');
  await page.fill('input[name=expires_in_days]', '7');
  await page.click('button:has-text("Tạo link")');

  // 4. Capture link
  const inviteURL = await page.locator('[data-testid="invitation-url"]').textContent();
  expect(inviteURL).toContain('/invite/accept?token=');

  // 5. Open in new context (simulate new user)
  const newUserContext = await context.browser()!.newContext();
  const newUserPage = await newUserContext.newPage();
  await newUserPage.goto(inviteURL!);

  // 6. Verify token + form shown
  await expect(newUserPage.locator('text=Manager đã mời bạn')).toBeVisible();
  await expect(newUserPage.locator('input[name=email]')).toBeVisible();

  // 7. Register
  await newUserPage.fill('input[name=email]', 'newuser@apex.vn');
  await newUserPage.fill('input[name=full_name]', 'Nguyen Van Moi');
  await newUserPage.fill('input[name=phone]', '0909123456');
  await newUserPage.fill('input[name=password]', 'Password123!');
  await newUserPage.click('button:has-text("Đăng ký")');

  // 8. Verify success
  await expect(newUserPage).toHaveURL(/\/dashboard/);

  // 9. Verify user appears in manager's team list
  await page.reload();
  await expect(page.locator('text=Nguyen Van Moi')).toBeVisible();
});
```

## 17. Migration Plan

### 17.1. Initial Schema Migration

```bash
# Run migrations
cd services/tree-org
goose -dir migrations postgres "$DATABASE_URL" up

# Seed sample data
psql "$DATABASE_URL" < scripts/seed_dev.sql
```

### 17.2. Zero-Downtime Schema Changes

Pattern đã mô tả trong `docs/00-master/README.md` §21. Áp dụng cho tất cả tables trong §8.

### 17.3. Migration Tracking Sheet

| Date | Description | Status |
|------|-------------|--------|
| 2026-09-15 | Add leads, deals tables | Pending |
| 2026-09-20 | Add pipelines table | Pending |
| 2026-10-01 | Add user_sessions table | Pending |
| 2026-10-05 | Add user_notifications table | Pending |
| 2026-10-10 | Add lead_activities table | Pending |
| 2026-10-15 | RLS policies cho tất cả tables | Pending |
| 2026-11-01 | Add Materialized View cho subtree stats | Pending |

### 17.4. Backfill Strategy

Khi cần backfill data lớn (vd: tính lại AI score cho 1M Lead):

```sql
-- Batched update
DO $$
DECLARE
  last_id UUID;
BEGIN
  LOOP
    UPDATE leads SET score = ai_scoring_service.calculate(...)
    WHERE id > COALESCE(last_id, '00000000-0000-0000-0000-000000000000'::UUID)
      AND score IS NULL
    LIMIT 1000
    RETURNING id INTO last_id;
    EXIT WHEN last_id IS NULL;
    COMMIT;
    PERFORM pg_sleep(0.1); -- Throttled
  END LOOP;
END $$;
```

## 18. Disaster Recovery

### 18.1. RPO & RTO

| Component | RPO | RTO |
 |-----------|-----|-----|
| tree-org Postgres | 5 min | 30 min |
| Lead/Deal data | 5 min | 30 min |
| Audit log | 1 hour | 2 hours |
| Notification log | 1 hour | 2 hours |

### 18.2. Failure Scenarios

#### Scenario A: tree-org service down
- **Detection:** K3s liveness probe fail
- **Response:** Auto-restart < 2s
- **Recovery:** Stateless, no data loss

#### Scenario B: Postgres primary fail
- **Response:** Promote replica, redirect writes
- **RTO:** 30 min
- **RPO:** 5 min

#### Scenario C: Tree data corruption
- **Response:** PITR restore from latest backup
- **RTO:** 1 hour

#### Scenario D: LTREE index corrupt
- **Response:** REINDEX CONCURRENTLY
- **RTO:** 15 min

### 18.3. Backup Strategy

```bash
# scripts/backup-tree-org-db.sh (chạy mỗi 6 giờ)
#!/bin/bash
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/tree-org/${TIMESTAMP}"
mkdir -p $BACKUP_DIR

# 1. Schema only (small)
pg_dump --schema-only -h postgres -U rinco rinco_crm > $BACKUP_DIR/schema.sql

# 2. Data dump (compressed)
pg_dump -h postgres -U rinco rinco_crm | gzip > $BACKUP_DIR/data.sql.gz

# 3. Specific high-importance tables
pg_dump -h postgres -U rinco -t users -t invitations -t audit_actions \
    rinco_crm | gzip > $BACKUP_DIR/critical.sql.gz

# 4. Upload to MinIO
mc cp $BACKUP_DIR/data.sql.gz minio/backups/tree-org/data/${TIMESTAMP}.sql.gz
mc cp $BACKUP_DIR/critical.sql.gz minio/backups/tree-org/critical/${TIMESTAMP}.sql.gz

# 5. Retain 90 days
mc rm --recursive --force --older-than 90d minio/backups/tree-org/

rm -rf $BACKUP_DIR
```

### 18.4. DR Drill (Hàng quý)

Test cases:
1. Move subtree 10K user rồi kill service → verify recovery.
2. Corrupt 1 user record → verify restore.
3. Test parallel moves → verify no race condition.

## 19. Cost Estimation

### 19.1. Compute Cost

| Component | Spec | Qty | Monthly |
|-----------|------|-----|---------|
| crm-core | 4 vCPU, 8GB | 6 | $720 |
| tree-org | 2 vCPU, 4GB | 4 | $320 |
| Postgres (CRM) | 8 vCPU, 32GB, 500GB | 1 + 1 replica | $1,000 |
| ScyllaDB (notification log) | 4 vCPU, 16GB, 100GB | 1 | $200 |
| Valkey (cache, session) | 4 vCPU, 16GB | 2 | $300 |
| ClickHouse (analytics mirror) | 8 vCPU, 32GB, 500GB | 1 | $400 |
| **Subtotal** | - | - | **$2,940** |

### 19.2. Storage Cost

| Item | Size | Monthly |
|------|------|---------|
| Postgres backup (90d) | 2TB | $46 |
| Notification archive | 500GB | $11.5 |
| **Subtotal** | - | **$57.5** |

### 19.3. External Services

| Service | Monthly |
|---------|---------|
| Email (SendGrid) - 100K emails | $100 |
| Push (FCM + APNs) | Free |
| **Subtotal** | **$100** |

### 19.4. Tổng chi phí CRM

```
Compute:  $2,940
Storage:  $57.5
External: $100
──────────────
Total:    ~$3,097/month
```

### 19.5. Cost per User

Với 10,000 tenants, mỗi tenant trung bình 50 user = 500K users:
- Cost per user: $3,097 / 500,000 = **$0.006/user/month**

## 20. Open Questions / Cần user xác nhận

### 20.1. Cần user xác nhận ngay ✋

| # | Câu hỏi | Options | Recommendation |
|---|---------|---------|----------------|
| Q1 | **Max depth cây tổ chức?** | (a) 10, (b) 20, (c) Unlimited | (a) - giới hạn để dễ quản |
| Q2 | **Default target role khi mời?** | (a) NV, (b) User chọn | (a) – an toàn hơn |
| Q3 | **Onboarding checklist ai tạo?** | (a) Tenant admin, (b) System template | (b) – consistency |
| Q4 | **Lead duplicate detection threshold?** | (a) Exact match, (b) Fuzzy 80%, (c) AI | (b) – balance |
| Q5 | **Deal pipeline default stages?** | (a) 5 stages (Prospect→Won), (b) User config | (b) |
| Q6 | **Notification broadcast max recipients?** | (a) 1K, (b) 10K, (c) Unlimited | (b) – protect infra |
| Q7 | **Có cho phép Giám đốc xóa cây con không?** | (a) Có, audit, (b) Cần Quorum | (b) – safety |
| Q8 | **Lead assignment: AI hay manual?** | (a) Manual, (b) AI suggested, (c) Auto | (b) – assist, not replace |
| Q9 | **MFA bắt buộc cho user?** | (a) Cho QL+, (b) Cho GD only, (c) All | (a) |
| Q10 | **Bulk move có cần confirm từng user không?** | (a) Bulk confirm, (b) Per-user notify | (a) – UX |

### 20.2. Cần quyết định trong Phase tiếp theo 📋

| # | Câu hỏi | Impact | Owner |
|---|---------|--------|-------|
| Q11 | Có cần custom role cho viewer? | Effort | Product |
| Q12 | Department có phải multi-tenant không? | Schema | Architecture |
| Q13 | Notification digest frequency? | UX | Product |
| Q14 | Lead scoring AI model nào? | Effort | AI |
| Q15 | Deal close date required hay optional? | UX | Product |
| Q16 | Permission approval workflow nặng hay nhẹ? | UX | Product |
| Q17 | Có cần permission theo IP không? | Security | Security |
| Q18 | Subtree stats update real-time hay batch? | Perf | Backend |

### 20.3. TBD kỹ thuật ⏳

| # | Item | Status | Next step |
|---|------|--------|-----------|
| T1 | PASETO key rotation ceremony | TBD | Design Q4 2026 |
| T2 | Subtree materialized view refresh strategy | TBD | Test performance |
| T3 | Notification digest job scheduler | TBD | Use NATS delayed |
| T4 | Lead scoring AI integration (sync vs async) | TBD | Test async |
| T5 | Email template engine | TBD | Use MJML |
| T6 | OrgChart rendering library | TBD | Compare react-d3-tree vs GoJS |
| T7 | Bulk import CSV parsing library | TBD | Compare encoding/csv |
| T8 | Realtime dashboard framework | TBD | SSE vs WebSocket decision |

### 20.4. Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| LTREE index corrupt | Low | High | REINDEX CONCURRENTLY + monitoring |
| Path slug collision | Medium | Low | Auto-increment suffix |
| Cycle in tree | Low | Critical | Pre-check trước move |
| Lead bulk import DoS | Medium | Medium | Rate limit + background job |
| AI scoring bias | Medium | High | A/B test + manual override |
| Notification spam | Medium | Medium | Rate limit + digest |
| Subtree query slow (100K users) | Medium | Medium | Materialized view cache |
| Cross-tenant data leak | Low | Critical | RLS + audit + e2e test |
| PASETO key compromise | Low | Critical | Auto-rotation + black list old |
| Department schema ambiguity | Medium | Low | Clarify schema |

---

**Tiếp theo:** [`docs/04-dynamic-model/README.md`](../04-dynamic-model/README.md) – Trình sinh mô hình doanh nghiệp động (Meta-Schema Engine).