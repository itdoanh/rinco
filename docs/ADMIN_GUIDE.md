# Admin Portal Guide (Super Admin)

> **Phiên bản:** 1.0 · **Cập nhật:** Sep 2026
>
> Hướng dẫn này dành cho **Super Admin** của nền tảng RINCO — những người có quyền cao nhất, quản lý toàn bộ hệ thống, tenants, và thực hiện các thao tác P0 (delete tenant, rotate keys).

---

## Mục Lục

1. [Đăng nhập Admin Portal](#1-đăng-nhập-admin-portal)
2. [Quản lý Tenants](#2-quản-lý-tenants)
3. [2-of-3 Quorum](#3-2-of-3-quorum)
4. [Feature Flags](#4-feature-flags)
5. [System Health](#5-system-health)
6. [Audit Log](#6-audit-log)
7. [Notifications](#7-notifications)
8. [Thông báo trong Tenant](#8-thông-báo-trong-tenant)
9. [Bảo mật](#9-bảo-mật)
10. [Disaster Recovery](#10-disaster-recovery)
11. [Văn hóa & Quy trình](#11-văn-hóa--quy-trình)

---

## 1. Đăng nhập Admin Portal

### URL truy cập

> ⚠️ **Quan trọng:** Admin Portal **KHÔNG** có DNS record công khai (Dark Admin). Bạn chỉ có thể truy cập qua mạng WireGuard nội bộ.

```
https://admin.rinco.internal:3001
```

### Tài khoản mặc định (seed)

| Email | Password | Role |
|-------|----------|------|
| `superadmin@rinco.vn` | `Admin@123` | Super Admin |

> 🔒 Bắt buộc đổi mật khẩu ngay lần đăng nhập đầu tiên. Kích hoạt **YubiKey 2FA** ngay sau đó.

### Kết nối WireGuard

Trước khi truy cập Admin Portal, đảm bảo bạn đã kết nối vào WireGuard mesh:

```bash
# Linux / macOS
sudo wg-quick up rinco-admin

# Windows (sử dụng WireGuard app)
# Import file: rinco-admin.conf → Activate
```

---

## 2. Quản lý Tenants

### 2.1. Xem danh sách Tenants

Vào **Tenants** ở sidebar.

> 📷 **Screenshot placeholder:** Bảng tenants với filter theo plan/status/region

Mỗi tenant hiển thị:
- Logo + Tên + Slug
- Plan (Starter / Pro / Enterprise)
- Trạng thái (Active / Suspended / Pending Verification)
- Region (ap-southeast-1 / eu-west-1 / …)
- Owner email
- Số user / Số Lead trong tháng
- Doanh thu tháng
- Ngày tạo

### 2.2. Tạo Tenant mới

**Bước 1:** Click **+ New Tenant**

**Bước 2:** Form **Create Tenant**:

| Trường | Bắt buộc | Mô tả |
|--------|----------|-------|
| **Slug** | ✅ | URL identifier (lowercase, kebab-case, 2–40 chars) |
| **Name** | ✅ | Tên đầy đủ |
| **Plan** | ✅ | Starter / Pro / Enterprise |
| **Region** | ✅ | Region đặt server (closest to customer) |
| **Custom Domain** | ❌ | Domain riêng (vd: `apex.vn`) |
| **Subdomain** | ❌ | Subdomain under rinco.app (vd: `apex.rinco.app`) |
| **Template** | ✅ | Fintech / Edu / Real Estate / Retail / Custom |
| **Owner Email** | ✅ | Sẽ nhận email kích hoạt |
| **Quota (Lead/month)** | ❌ | Default: 10,000 (Starter) / 100,000 (Pro) |
| **Quota (User)** | ❌ | Default: 5 (Starter) / 50 (Pro) |

**Bước 3:** Chọn **Template** — hệ thống sẽ tự động sinh schema CRM phù hợp:
- **Fintech:** Lead có trường `loan_amount`, `loan_term`, `income`, …
- **Real Estate:** Lead có trường `property_type`, `budget`, `location`, …
- **Edu:** Lead có trường `course_interest`, `education_level`, …
- **Retail:** Lead có trường `product_interest`, `purchase_timeline`, …

**Bước 4:** Click **Create**.

Hệ thống sẽ:
1. Tạo database schema riêng cho tenant
2. Seed dữ liệu mẫu (nếu template được chọn)
3. Gửi email kích hoạt cho Owner
4. Tạo NATS subject prefix riêng: `tenant.{slug}.*`

### 2.3. Suspend (Tạm dừng) Tenant

**Bước 1:** Chọn tenant → **Actions → Suspend**

**Bước 2:** Nhập **lý do** (sẽ log vào audit).

**Bước 3:** Click **Confirm**.

**Hậu quả:**
- ❌ User không thể đăng nhập
- ❌ Landing Page trả về 503
- ❌ API trả về 403
- ✅ Dữ liệu được bảo toàn 100%
- ✅ Vẫn tính tiền (nếu vi phạm thanh toán)

### 2.4. Restore Tenant

**Bước 1:** Chọn tenant (status = Suspended) → **Actions → Restore**

**Bước 2:** Click **Confirm**.

Hệ thống tự động unsuspend trong vòng 5 giây.

### 2.5. Xóa Tenant (yêu cầu 2-of-3 Quorum)

> ⚠️ **Thao tác P0 — KHÔNG THỂ HOÀN TÁC!**

Xem chi tiết tại [§3. 2-of-3 Quorum](#3-2-of-3-quorum).

---

## 3. 2-of-3 Quorum

### 3.1. Quorum là gì?

Để ngăn chặn **insider attack** (admin lạm quyền), mọi thao tác P0 yêu cầu **2 trong 3 Super Admin** ký duyệt.

### 3.2. Các thao tác P0

| Hành động | Mô tả |
|-----------|-------|
| **Delete tenant** | Xóa hoàn toàn tenant + tất cả data |
| **Restore tenant đã xóa** | Khôi phục từ backup (nếu có) |
| **Rotate master keys** | Xoay PASETO / JWT keys |
| **Quorum admin change** | Thêm/xóa admin khỏi nhóm 3 |
| **Domain takeover** | Chuyển domain giữa tenants |
| **Cross-tenant data access** | Truy cập data của tenant khác (debug) |

### 3.3. 3 Admin mặc định

| # | Admin | Email | YubiKey Serial |
|---|-------|-------|----------------|
| 1 | Doanh Nguyen | doanh@rinco.vn | YK-XXXXX-001 |
| 2 | Tuan Anh | tuananh@rinco.vn | YK-XXXXX-002 |
| 3 | Van Toan | vantoan@rinco.vn | YK-XXXXX-003 |

### 3.4. Quy trình thực hiện

**Ví dụ: Xóa tenant "apex-fintech"**

**Bước 1 (Admin #1):** Vào **Tenants → apex-fintech → Actions → Request Deletion**

**Bước 2:** Hệ thống tạo **Quorum Request** với ID `q-abc-123`, hết hạn trong 5 phút.

**Bước 3 (Admin #1):** Click **Sign with YubiKey** → cắm YubiKey → chạm → ký thành công (1/2).

**Bước 4:** Chia sẻ `q-abc-123` qua **secure channel** (Signal / Telegram encrypted) với Admin #2 hoặc Admin #3.

> ⚠️ **KHÔNG** chia sẻ qua email / Zalo thường!

**Bước 5 (Admin #2):** Vào **Quorum → Pending → q-abc-123 → Sign** → cắm YubiKey → chạm (2/2).

**Bước 6:** Hệ thống tự động thực thi:
- ✅ Database của tenant được DROP
- ✅ NATS streams bị xóa
- ✅ S3 prefix bị xóa
- ✅ Audit log: ghi lại cả 2 chữ ký + timestamp + IP

### 3.5. Quorum States

| State | Ý nghĩa |
|-------|---------|
| `pending` | Đang chờ chữ ký (hết hạn sau 5 phút) |
| `partial` | Có 1 chữ ký, chờ thêm |
| `executed` | Đã thực thi thành công |
| `expired` | Hết 5 phút, cần tạo lại |
| `rejected` | Admin từ chối ký |

---

## 4. Feature Flags

### 4.1. Xem danh sách Feature Flags

Vào **Settings → Feature Flags**.

Mỗi flag có:
- **Key** (vd: `enable_capi`)
- **Tên hiển thị**
- **Mô tả**
- **Default value** (on/off)
- **Override theo tenant** (nếu có)

### 4.2. Các Feature Flags chính

| Flag | Mô tả | Default |
|------|-------|---------|
| `enable_capi` | Facebook Conversions API | ✅ On |
| `enable_ai_scoring` | AI chấm điểm Lead | ✅ On |
| `enable_ai_chatbot` | AI Chatbot RAG | ✅ On |
| `enable_meeting` | WebRTC Meeting | ✅ On |
| `enable_meeting_recording` | Ghi âm cuộc họp | ✅ On |
| `enable_custom_domain` | Tenant dùng custom domain | ❌ Off (Enterprise only) |
| `enable_dedicated_vps` | Tenant có VPS riêng | ❌ Off (Enterprise only) |
| `enable_advanced_analytics` | ClickHouse + custom reports | ❌ Off (Pro+) |
| `enable_webhooks` | Webhook outbound | ✅ On |
| `enable_sso` | SAML / OIDC SSO | ❌ Off (Enterprise only) |

### 4.3. Toggle Flag cho Tenant

**Bước 1:** Vào **Settings → Feature Flags → [Flag] → Tenant Overrides**

**Bước 2:** Click **+ Add Tenant Override**

**Bước 3:** Chọn tenant → bật/tắt → **Save**.

**Ví dụ:** Tắt `enable_meeting_recording` cho tenant `apex-fintech` (vì policy compliance).

---

## 5. System Health

### 5.1. Dashboard System Health

Vào **System → Health**.

> 📷 **Screenshot placeholder:** Grid 22 services + 9 databases với màu xanh/vàng/đỏ

### 5.2. Trạng thái

| Màu | Ý nghĩa | Hành động |
|-----|----------|-----------|
| 🟢 Green | Healthy | Không cần làm gì |
| 🟡 Yellow | Degraded | Kiểm tra log |
| 🔴 Red | Down | Alert ngay |

### 5.3. Metrics per Service

Click vào 1 service để xem:
- **CPU / Memory / Network** (realtime)
- **Request rate** (req/s)
- **Error rate** (5xx %)
- **Latency P50 / P95 / P99**
- **Active connections**
- **Last 100 errors** với `trace_id`

### 5.4. Alerts

Cấu hình **Alert Rules**:
- **Error rate > 1%** trong 5 phút → Alert Telegram
- **Latency P99 > 500ms** trong 10 phút → Alert Telegram
- **CPU > 90%** trong 15 phút → Alert
- **Disk usage > 80%** → Alert
- **Service down** > 30s → Critical Alert (gọi điện)

---

## 6. Audit Log

### 6.1. Xem Audit Log

Vào **System → Audit Log**.

Mọi hành động đều được log:
- **Timestamp** (UTC)
- **Admin** (user thực hiện)
- **Action** (vd: `tenant.create`, `feature_flag.toggle`)
- **Target** (resource bị ảnh hưởng)
- **IP** + **User Agent**
- **Result** (success / failure)
- **Trace ID** (correlate với logs)
- **Diff** (thay đổi gì)

### 6.2. Filter

- Theo **Admin**
- Theo **Action type**
- Theo **Tenant**
- Theo **Khoảng thời gian**
- Theo **IP**

### 6.3. Export

Click **Export** → CSV / JSON → Tải về.

Dữ liệu audit log **KHÔNG** bao giờ bị xóa (giữ 7 năm theo quy định).

---

## 7. Notifications (Thông báo Admin)

### 7.1. Gửi thông báo tới toàn hệ thống

**Bước 1:** Vào **Notifications → + New Broadcast**

**Bước 2:** Chọn **Audience**:
- **All Admins** (tất cả Super Admin)
- **Tenant Admins** (tất cả chủ doanh nghiệp)
- **Custom segment** (filter theo plan / region / status)

**Bước 3:** Soạn nội dung:
- **Title** (ngắn gọn, ≤ 80 chars)
- **Body** (Markdown hỗ trợ)
- **Severity**: Info / Warning / Critical
- **Action URL** *(tùy chọn)* — nút CTA

**Bước 4:** Chọn kênh:
- 🔔 In-app notification
- 📧 Email
- 💬 Telegram (nếu user đã link)
- 📱 Push (mobile app)
- 📞 SMS (Critical only)

**Bước 5:** Click **Send**.

### 7.2. Xem lịch sử thông báo

Vào **Notifications → History** để xem các broadcast đã gửi, tỷ lệ đọc, click-through rate.

---

## 8. Thông báo trong Tenant

> 📷 **Screenshot placeholder:** Composer soạn thông báo tenant với target audience picker

Đây là hệ thống thông báo nội bộ tenant (do Admin tenant hoặc Giám đốc gửi tới nhân viên của mình).

### 8.1. Ai có quyền gửi?

| Role | Quyền gửi |
|------|-----------|
| **Tenant Admin** (Giám đốc) | ✅ Toàn tenant |
| **Manager** | ✅ Subtree của mình |
| **Team Lead** | ✅ Team mình |
| **Agent** | ❌ Không |

### 8.2. Cách gửi

**Bước 1:** Vào **Tenant → Communications → + New Announcement**

**Bước 2:** Soạn:
- **Tiêu đề**
- **Nội dung** (Markdown / rich text)
- **Loại thông báo**:
  - 📢 **Announcement** — quan trọng, hiển thị banner
  - 📋 **Update** — thông tin thường
  - 🎉 **Celebration** — chúc mừng, khen thưởng
  - ⚠️ **Alert** — cảnh báo cần chú ý
- **Audience**:
  - Toàn tenant
  - Theo role
  - Theo department
  - Theo region
  - Custom (chọn từng user)
- **Kênh gửi**: in-app / email / push / SMS (chọn)
- **Schedule**: gửi ngay / lên lịch

**Bước 3:** Click **Send**.

### 8.3. Thông báo tự động (hệ thống tự gửi)

Hệ thống TỰ ĐỘNG gửi thông báo trong các trường hợp:

| Sự kiện | Thông báo tới |
|---------|---------------|
| 🆕 Lead mới được assign cho bạn | Owner |
| 📞 Cuộc gọi nhỡ | Owner |
| 💬 Tin nhắn mới (chat) | Người nhận |
| 📅 Deal sắp đến hạn đóng (trong 3 ngày) | Owner + Manager |
| 🎯 Deal chuyển sang Won | Owner + Manager + Admin |
| ⚠️ Deal chuyển sang Lost | Owner + Manager |
| 🎂 Sinh nhật nhân viên | Toàn team |
| 📊 Đạt KPI tháng | Owner + Manager |
| 📈 Vượt target | Admin |

### 8.4. Xem lịch sử

Vào **Communications → Sent History** để xem:
- Thông báo đã gửi
- Tỷ lệ đọc (% người nhận đã mở)
- Tỷ lệ phản hồi (nếu có CTA)

---

## 9. Bảo mật

### 9.1. Quản lý Super Admin

Vào **Security → Super Admins**.

- Xem danh sách 3 Super Admin
- Thay đổi YubiKey (cần quorum)
- Tạm khóa admin (cần quorum)
- Xem audit log của admin

### 9.2. IP Whitelist

Chỉ cho phép IP trong whitelist truy cập Admin Portal:

Vào **Security → IP Whitelist**:
- Thêm IP CIDR (vd: `203.0.113.0/24`)
- Bật/tắt whitelist
- Yêu cầu Quorum để tắt whitelist (tránh lockout)

### 9.3. Session Policy

- **Idle timeout**: 15 phút
- **Absolute timeout**: 8 giờ (bắt buộc login lại)
- **Concurrent sessions**: tối đa 2
- **Force logout** cho tất cả sessions: yêu cầu Quorum

---

## 10. Disaster Recovery

### 10.1. Backup

Hệ thống tự động backup theo lịch:

| Resource | Tần suất | Retention |
|----------|----------|-----------|
| **PostgreSQL** (CRM, Auth, Tenant) | WAL streaming + daily full | 30 ngày hot + 1 năm archive |
| **ScyllaDB** (Chat, Landing) | Daily snapshot | 14 ngày |
| **MongoDB** (Dynamic Form) | Daily snapshot | 30 ngày |
| **ClickHouse** (Analytics) | Daily | 90 ngày |
| **MinIO** (Media) | Cross-region replication | 1 năm |
| **Valkey** (Cache) | AOF + daily | 7 ngày |

### 10.2. Khôi phục dữ liệu cho Tenant

**Bước 1:** Vào **Tenants → [tenant] → Actions → Request Restore**

**Bước 2:** Chọn **thời điểm** cần khôi phục (point-in-time recovery cho PostgreSQL).

**Bước 3:** Hệ thống tạo **restore job** chạy async, thông báo khi hoàn tất.

> ⚠️ Restore sẽ **ghi đè** dữ liệu hiện tại. Cân nhắc dùng "Clone to sandbox" trước.

### 10.3. DR Drill (Test khôi phục)

Nên chạy DR Drill **mỗi quý**:
- Tạo sandbox cluster
- Restore backup
- Verify tính toàn vẹn dữ liệu (checksum)
- Đo thời gian RTO

---

## 11. Văn hóa & Quy trình

### 11.1. Triết lý "Cây Cổ Thụ" (Treelike Governance)

Lấy cảm hứng từ sơ đồ phân cấp hình cây:

| Tầng | Vai trò |
|------|---------|
| **🌳 Rễ cây** (Ban Lãnh đạo) | Chiến lược, hạ tầng, công cụ — nuôi dưỡng toàn hệ thống |
| **🌲 Thân cây** (Quản lý) | Kênh truyền dẫn thông tin hai chiều |
| **🌿 Cành lá** (Nhân sự) | Tiếp xúc khách hàng, xử lý công việc |
| **🍎 Trái ngọt** (Khách hàng) | Sự hài lòng, tăng trưởng, giá trị cho cộng đồng |

### 11.2. Văn hóa "Không Khoảng Cách"

- Đề xuất của nhân viên mới được lắng nghe ngang hàng với quản lý lâu năm
- Sử dụng Slack/Discord để trao đổi trực tiếp
- Monthly "Open Tree Day" — mọi cấp ngồi lại đối thoại

### 11.3. Văn hóa "Meritocracy"

- Thăng tiến theo kết quả thực tế, không theo seniority
- Quarterly "RINCO Heroes" — vinh danh người sống trọn vẹn giá trị R-I-N-C-O

### 11.4. Rituals

| Ritual | Tần suất |
|--------|----------|
| 🩺 **Pulse Check** | 10 phút đầu ngày |
| 🌳 **Open Tree Day** | Hàng tháng |
| 🏆 **RINCO Heroes** | Hàng quý |
| 🌍 **Community Impact Day** | Hàng năm |

---

## Phụ lục: Liên hệ khẩn cấp

| Mức độ | Kênh | Phản hồi |
|--------|------|----------|
| **P0 (Critical)** | 📞 Hotline 1900-xxxx | < 15 phút |
| **P1 (High)** | 💬 Telegram `@rinco_sre` | < 1 giờ |
| **P2 (Medium)** | 📧 support@rinco.vn | < 4 giờ |
| **P3 (Low)** | 🎫 Ticket portal | < 24 giờ |

---

**Tài liệu này được cập nhật thường xuyên. Phiên bản mới nhất luôn ở https://docs.rinco.vn/admin**
