# Hướng Dẫn Sử Dụng RINCO

> **Phiên bản:** 1.0 · **Cập nhật:** Sep 2026
>
> Hướng dẫn này dành cho **người dùng cuối** (Giám đốc, Quản lý, Nhân viên) và **Chủ doanh nghiệp** sử dụng nền tảng RINCO để quản lý Landing Page, CRM, Chat và Cuộc họp.

---

## Mục Lục

1. [Giới thiệu](#1-giới-thiệu)
2. [Đăng ký & Đăng nhập](#2-đăng-ký--đăng-nhập)
3. [Dashboard](#3-dashboard)
4. [Quản lý Tenant (Doanh nghiệp)](#4-quản-lý-tenant-doanh-nghiệp)
5. [Landing Page Builder](#5-landing-page-builder)
6. [CRM Tree – Cây tổ chức](#6-crm-tree--cây-tổ-chức)
7. [Quản lý Lead](#7-quản-lý-lead)
8. [Quản lý Deal](#8-quản-lý-deal)
9. [Chat & Meeting](#9-chat--meeting)
10. [AI Features](#10-ai-features)
11. [Analytics & Báo cáo](#11-analytics--báo-cáo)
12. [Settings & Billing](#12-settings--billing)
13. [FAQ – Câu hỏi thường gặp](#13-faq--câu-hỏi-thường-gặp)

---

## 1. Giới thiệu

RINCO là nền tảng **MarTech + SalesTech + AdTech** đa tenant, giúp doanh nghiệp Việt Nam:

- 🏗️ **Xây dựng Landing Page** chuyên nghiệp với tracking đầy đủ
- 👥 **Quản lý CRM** theo sơ đồ cây tổ chức (Giám đốc → Quản lý → Trưởng nhóm → Nhân viên)
- 💬 **Chat real-time** với mã hóa đầu cuối
- 📞 **Gọi thoại & Video call** chất lượng cao
- 🤖 **AI Scoring** tự động chấm điểm tiềm năng của Lead
- 📊 **Analytics** thời gian thực với ClickHouse OLAP
- 🔗 **Tích hợp Facebook CAPI** để tối ưu quảng cáo

### Triết lý RINCO

> *"Kết Nối Toàn Năng – Vững Vàng Quản Trị"*

RINCO bình dân hóa công nghệ đỉnh cao, giúp mọi doanh nghiệp (dù là startup hay tập đoàn) tiếp cận hạ tầng quản trị chất lượng cao với chi phí tối ưu.

### Bộ giá trị RINCO

| Ký tự | Ý nghĩa |
|-------|---------|
| **R** – Responsibility | Trách nhiệm phụng sự |
| **I** – Integrity | Trung thực & bảo mật tuyệt đối |
| **N** – Nurture | Nuôi dưỡng & đồng hành |
| **C** – Cohesion | Gắn kết hoàn hảo |
| **O** – Optimization | Tối ưu hóa vì cộng đồng |

---

## 2. Đăng ký & Đăng nhập

### 2.1. Đăng ký tài khoản mới

> 📷 **Screenshot placeholder:** Màn hình đăng ký với form Email / Mật khẩu / Họ tên

**Bước 1:** Truy cập [https://app.rinco.vn/register](https://app.rinco.vn/register)

**Bước 2:** Điền thông tin:
- **Email** (dùng email công ty)
- **Mật khẩu** (tối thiểu 8 ký tự, bao gồm cả chữ và số)
- **Họ và tên**
- **Mã mời** *(nếu có – do quản lý cấp trên gửi)*

**Bước 3:** Click **Đăng ký**

> 💡 **Mẹo:** Nếu bạn nhận được link mời từ quản lý (ví dụ: `https://app.rinco.vn/invite/abc123`), click vào link đó — hệ thống sẽ tự động điền mã mời và đặt bạn vào đúng vị trí trong cây tổ chức.

### 2.2. Đăng nhập

**Bước 1:** Truy cập [https://app.rinco.vn/login](https://app.rinco.vn/login)

**Bước 2:** Nhập **Email** + **Mật khẩu**

**Bước 3:** *(Tùy chọn, bảo mật cao)* Nếu tài khoản của bạn bật **Xác thực 2 yếu tố (2FA)** bằng YubiKey, hệ thống sẽ yêu cầu cắm YubiKey và chạm vào nút.

**Bước 4:** Click **Đăng nhập**

> ⚠️ **Quên mật khẩu?** Click "Quên mật khẩu" ở cuối form, nhập email, hệ thống sẽ gửi link đặt lại mật khẩu vào email của bạn.

### 2.3. Đăng xuất

Click vào **Avatar** (góc trên bên phải) → **Đăng xuất**.

---

## 3. Dashboard

> 📷 **Screenshot placeholder:** Dashboard tổng quan với 4 ô KPI + biểu đồ conversion

Sau khi đăng nhập, bạn sẽ thấy **Dashboard** hiển thị:

### 3.1. KPI tổng quan (4 ô phía trên)

| KPI | Mô tả | Cập nhật |
|-----|-------|----------|
| **Tổng Lead** | Số Lead trong 30 ngày qua | Realtime |
| **Tổng Deal** | Số Deal đang mở | Realtime |
| **Doanh thu** | Tổng giá trị Deal đã chốt | Realtime |
| **Tỷ lệ chuyển đổi** | % Lead → Deal | Realtime |

### 3.2. Biểu đồ

- **Conversion Funnel:** Lead → Contact → Deal → Won (cập nhật mỗi 5 giây)
- **Revenue Trend:** Doanh thu theo ngày/tuần/tháng
- **Top Performers:** Nhân viên có nhiều Deal nhất

### 3.3. Recent Activities

Danh sách các hoạt động gần đây:
- 🆕 Lead mới được tạo
- 📞 Cuộc gọi vừa kết thúc
- 💬 Tin nhắn mới
- ✅ Deal chuyển stage

### 3.4. System Health (chỉ hiển thị với Admin)

22 microservices + 9 databases hiển thị trạng thái realtime (xanh = OK, vàng = cảnh báo, đỏ = lỗi).

---

## 4. Quản lý Tenant (Doanh nghiệp)

> 📷 **Screenshot placeholder:** Trang quản lý tenant với danh sách + nút "+ Tạo mới"

### 4.1. Xem danh sách Tenant

Vào **Settings → Tenants** để xem tất cả doanh nghiệp đang hoạt động trong hệ thống.

Mỗi tenant hiển thị:
- **Logo** (nếu có)
- **Tên** + **Slug** (slug dùng cho URL)
- **Plan** (Starter / Pro / Enterprise)
- **Trạng thái** (Active / Suspended)
- **Số user** đang hoạt động
- **Số Lead** trong tháng

### 4.2. Tạo Tenant mới

**Bước 1:** Click **+ Tạo Tenant mới**

**Bước 2:** Điền form:

| Trường | Mô tả | Ví dụ |
|--------|-------|-------|
| **Slug** | URL-friendly identifier (chỉ chữ thường, số, dấu gạch ngang) | `apex-fintech` |
| **Tên** | Tên đầy đủ của doanh nghiệp | `Công ty CP Tài chính Apex` |
| **Plan** | Starter / Pro / Enterprise | `Pro` |
| **Custom domain** *(tùy chọn)* | Domain riêng (vd: `apex.vn`) | `apex.vn` |
| **Email chủ sở hữu** | Email sẽ trở thành Admin gốc | `giámđốc@apex.vn` |

**Bước 3:** Click **Tạo**.

### 4.3. Tạm dừng (Suspend) / Khôi phục (Restore)

Chọn tenant → Click **Hành động → Tạm dừng**.

Khi bị tạm dừng:
- ❌ User không thể đăng nhập
- ❌ Landing Page ngừng nhận Lead
- ❌ API trả về 503
- ✅ Dữ liệu được bảo toàn

Để khôi phục: **Hành động → Khôi phục**.

### 4.4. Xóa Tenant (yêu cầu 2-of-3 Quorum)

> ⚠️ **Thao tác nguy hiểm!** Yêu cầu 2 trong 3 Admin ký duyệt.

**Bước 1:** Chọn tenant → **Hành động → Yêu cầu xóa**

**Bước 2:** Hệ thống tạo một **Quorum Request** với ID và thời hạn 5 phút.

**Bước 3:** Chia sẻ ID Quorum với 2 Admin khác. Mỗi Admin vào trang **Quorum → Ký duyệt** và ký bằng YubiKey.

**Bước 4:** Khi đủ 2/3 chữ ký, hệ thống tự động thực thi xóa.

---

## 5. Landing Page Builder

> 📷 **Screenshot placeholder:** Giao diện kéo-thả block với sidebar trái là palette

### 5.1. Tạo Landing Page mới

**Bước 1:** Vào **Landing Pages → + Tạo mới**

**Bước 2:** Chọn template:
- 🎯 **Fintech / Tài chính** – Tối ưu cho vay tín dụng, bảo hiểm
- 🏠 **Bất động sản** – Dự án, mẫu nhà
- 🎓 **Giáo dục** – Khóa học, tuyển sinh
- 🛒 **Bán lẻ / E-commerce** – Sản phẩm, khuyến mãi
- 📋 **Chung** – Tùy biến từ đầu

**Bước 3:** Đặt tên + Slug (slug sẽ là đường dẫn URL, vd: `apex-fintech.vn/khuyen-mai-he`)

**Bước 4:** Click **Tạo**.

### 5.2. Kéo-thả Block

Sidebar trái có các **block**:
- **Hero** – Tiêu đề + mô tả + nút CTA
- **Form** – Form thu thập Lead
- **Stats** – Số liệu ấn tượng
- **Testimonial** – Đánh giá khách hàng
- **FAQ** – Câu hỏi thường gặp
- **Pricing** – Bảng giá
- **Logo Cloud** – Logo đối tác
- **Footer** – Chân trang

Kéo block từ sidebar vào canvas. Mỗi block có thể chỉnh sửa nội dung trực tiếp.

### 5.3. Cấu hình Form thu thập Lead

Click vào block **Form** để cấu hình:

| Trường | Mô tả |
|--------|-------|
| **Tên** *(required)* | Họ và tên |
| **Email** *(required)* | Email |
| **Số điện thoại** *(required)* | Format VN: +84… |
| **Trường tùy chỉnh** | Tự thêm (vd: "Số tiền vay mong muốn") |

Khi submit form, hệ thống sẽ:
1. ✅ Ghi vào **CRM** (lead-service)
2. ✅ Ghi raw vào **ScyllaDB**
3. ✅ Đẩy sự kiện sang **NATS**
4. ✅ Bắn **Facebook CAPI** (với HMAC chống giả mạo)
5. ✅ Trigger **AI Scoring**

### 5.4. Tracking chi tiết

Mỗi Landing Page tự động tracking:
- 📍 UTM parameters (`utm_source`, `utm_medium`, `utm_campaign`)
- 🔗 `fbclid`, `fbp`, `fbc` (Facebook identifiers)
- 🌍 IP, User Agent, Referrer
- 🖱️ Scroll depth, time on page, click heatmap
- 📊 Conversion events

### 5.5. Xuất bản

Click **Xuất bản** → chọn **Domain** (subdomain hoặc custom domain) → **Xuất bản**.

Landing Page được serve qua Cloudflare CDN, FCP < 0.4s.

---

## 6. CRM Tree – Cây tổ chức

### 6.1. Cây tổ chức là gì?

Mỗi tenant có một **cây tổ chức** (organisational tree). User ở root là **Admin (Giám đốc)**, các user dưới là **Manager** / **Team Lead** / **Agent**.

```
Apex Fintech (root)
├── Nguyễn Văn A (Manager - Miền Bắc)
│   ├── Trần Thị B (Team Lead - Hà Nội)
│   │   ├── Lê Văn C (Agent)
│   │   └── Phạm Thị D (Agent)
│   └── Hoàng Văn E (Team Lead - Hải Phòng)
│       └── Vũ Thị F (Agent)
└── Phạm Văn G (Manager - Miền Nam)
    └── Trần Văn H (Team Lead - HCM)
        └── Ngô Thị I (Agent)
```

### 6.2. Tạo User mới

**Cách 1: Tạo trực tiếp**

**Bước 1:** Vào **Users → + Add User**

**Bước 2:** Điền:
- Email
- Họ tên
- Số điện thoại *(tùy chọn)*
- **Role**: Admin / Manager / Team Lead / Agent
- **Department** *(tùy chọn)*

**Bước 3:** Click **Lưu**.

Hệ thống tự động gửi email mời cho user mới (link đăng ký có hiệu lực 7 ngày).

**Cách 2: Tạo link mời (Invite Link)** — *Khuyến nghị*

**Bước 1:** Vào **Users → Generate Invite Link**

**Bước 2:** Chọn:
- **Cấp bậc**: Manager / Team Lead / Agent
- **Hạn sử dụng**: 24h / 7 ngày / 30 ngày

**Bước 3:** Click **Generate**.

**Bước 4:** Copy link → Gửi cho người được mời qua Zalo / Email / SMS.

Họ click link → đăng ký → **tự động** được đặt vào đúng vị trí trong cây dưới quyền bạn.

### 6.3. Phân quyền (RBAC)

| Role | Xem | Sửa | Xóa |
|------|-----|------|-----|
| **Super Admin** (hệ thống) | Tất cả tenants | Tất cả | Tất cả |
| **Tenant Admin** (root) | Tất cả users trong tenant | Tất cả | Tất cả |
| **Manager** | Users trong subtree của mình | Subtree của mình | Subtree của mình |
| **Team Lead** | Users thuộc team mình | Team mình | Team mình |
| **Agent** | Chỉ data của mình | Data của mình | Không |

### 6.4. Thăng / hạ chức / Di chuyển nhánh

Chọn user → **Hành động**:

- **Đổi role**: Admin / Manager / Team Lead / Agent
- **Di chuyển sang nhánh khác**: Kéo thả hoặc chọn "Move to…"
- **Tạm khóa**: User không thể đăng nhập nhưng data được giữ
- **Xóa**: Soft-delete (giữ data 90 ngày)

Khi di chuyển user, **toàn bộ Lead/Deal/Contact** của user đó cũng được di chuyển theo.

---

## 7. Quản lý Lead

> 📷 **Screenshot placeholder:** Bảng Lead với filter sidebar và CTA hành động hàng loạt

### 7.1. Lead là gì?

Lead là **khách hàng tiềm năng** được thu thập từ Landing Page hoặc nhập tay. Mỗi Lead có:
- Thông tin liên hệ (tên, email, SĐT)
- Nguồn (Facebook Ads, Google, Zalo,…)
- Trạng thái (Mới / Đang chăm / Đã chốt / Thất bại)
- **Điểm AI Score** (0–100, tự động)
- **Owner** (nhân viên phụ trách)
- **Band** (cold / warm / hot — dựa trên AI Score)

### 7.2. Xem danh sách Lead

Vào **CRM → Leads**.

Có thể filter:
- Theo **Owner** (chỉ xem Lead của mình / team mình)
- Theo **Trạng thái**
- Theo **Nguồn**
- Theo **AI Band** (Hot / Warm / Cold)
- Theo **Khoảng thời gian**

### 7.3. Phân Lead cho nhân viên

**Auto-assign (theo AI):**

Vào **Settings → Lead Distribution**:
- **Strategy**: `round-robin` / `least-busy` / `top-performer` / `by-region`
- Bật **Auto-assign on creation**

→ Mỗi Lead mới sẽ tự động được phân cho nhân viên phù hợp nhất.

**Manual assign:**

Chọn 1 hoặc nhiều Lead → **Actions → Assign to** → Chọn nhân viên.

### 7.4. Chấm điểm AI

Hệ thống tự động chấm điểm 0–100 dựa trên:
- Hành vi trên Landing Page (thời gian, scroll depth)
- Nguồn (Facebook Ads có EMQ cao → điểm cao hơn)
- Thông tin điền (email công ty, SĐT hợp lệ)
- Lịch sử tương tác

| Band | Score | Hành động khuyến nghị |
|------|-------|----------------------|
| 🔥 Hot | 80–100 | Gọi trong 5 phút |
| 🌡️ Warm | 50–79 | Gọi trong 24h |
| ❄️ Cold | 0–49 | Email tự động nuôi |

### 7.5. Nhập Lead từ file Excel

**Bước 1:** Vào **CRM → Leads → Import**

**Bước 2:** Tải file **Mẫu Excel** về → điền Lead → Upload.

**Bước 3:** Hệ thống hiển thị preview → sửa các lỗi mapping cột → **Import**.

Hỗ trợ định dạng `.xlsx`, `.csv` (tối đa 10,000 Lead/lần).

---

## 8. Quản lý Deal

### 8.1. Pipeline (Quy trình bán hàng)

Mặc định, RINCO cung cấp pipeline chuẩn cho BĐS / Tài chính:

```
Prospecting → Qualification → Proposal → Negotiation → Won / Lost
```

Mỗi tenant có thể tùy biến pipeline riêng (Settings → Pipelines).

### 8.2. Tạo Deal

**Bước 1:** Vào **CRM → Deals → + New Deal**

**Bước 2:** Điền:
- **Tên Deal** (vd: "Hợp đồng vay 500tr - Nguyễn Văn A")
- **Contact** (liên kết với Lead đã có)
- **Giá trị** (VNĐ)
- **Ngày dự kiến chốt**
- **Owner** (mặc định: người tạo)
- **Stage**: mặc định `Prospecting`

### 8.3. Chuyển Stage

Trong trang chi tiết Deal, kéo thả Deal card từ cột này sang cột khác, hoặc click **Move to** → chọn stage.

Khi Deal chuyển sang `Won`:
- ✅ Deal được đánh dấu "Đã chốt"
- ✅ Hệ thống **tự động bắn Facebook CAPI** event `Purchase` (với monetary value)
- ✅ Meta Ads tối ưu lại targeting dựa trên conversion thật
- ✅ Doanh thu được cộng vào Dashboard

---

## 9. Chat & Meeting

### 9.1. Chat real-time

> 📷 **Screenshot placeholder:** Giao diện chat 3-pane (sidebar / conversation / detail)

**Tính năng:**
- 💬 Chat 1-1, nhóm, channel công ty
- 🔐 **Mã hóa đầu cuối** (End-to-end encryption) bằng Signal Protocol
- 📎 Gửi file (qua Presigned S3 Direct Upload — không qua backend)
- 😀 Reactions, threads, mentions
- 🔍 Tìm kiếm lịch sử (Meilisearch)
- 🎙️ Voice message
- 📱 Push notification khi offline

**Cách dùng:**

Vào **Messages** → Chọn user/channel bên trái → Nhập tin nhắn bên phải → Enter.

### 9.2. Gọi thoại & Video call

**Bắt đầu cuộc gọi 1-1:**

Vào **Chat →** chọn user → Click **📞 Voice** hoặc **📹 Video**.

**Tạo Meeting nhóm:**

Vào **Meetings → + New Meeting** → Đặt tên → Mời user → Click **Start**.

Tính năng Meeting:
- 🎥 Video HD (AV1/VP9 codec, tiết kiệm bandwidth)
- 🖥️ Share màn hình 4K@60fps
- 🎙️ Voice call
- 📝 Live caption (Whisper STT, real-time)
- 🤖 AI tự động tóm tắt cuộc họp + Action Items sau khi kết thúc
- ⏺️ Recording (lưu MinIO, có thể tua và tải về)

---

## 10. AI Features

### 10.1. AI Lead Scoring

Tự động. Mỗi Lead mới được chấm điểm trong vòng 1 giây. Xem chi tiết tại [§7.4](#74-chấm-điểm-ai).

### 10.2. AI Chatbot (RAG)

**Bước 1:** Vào **AI Assistant**

**Bước 2:** Hỏi bất kỳ câu hỏi nào về sản phẩm, CRM data, hoặc tài liệu nội bộ.

AI sẽ trả lời dựa trên:
- Dữ liệu CRM của tenant bạn (Lead, Deal, Contact)
- Tài liệu nội bộ đã upload
- Knowledge base công ty

### 10.3. AI Meeting Summary

Sau mỗi cuộc họp được ghi âm, hệ thống tự động:
1. Bóc tách giọng nói thành văn bản (Whisper.cpp)
2. Tóm tắt cuộc họp (Llama-3 70B)
3. Trích xuất **Action Items** (công việc cần làm)
4. Lưu vào CRM, liên kết với Deal/Contact liên quan

Vào **Meetings → [cuộc họp đã ghi] → Summary** để xem.

---

## 11. Analytics & Báo cáo

### 11.1. Dashboard Realtime

Vào **Analytics → Overview**.

Cập nhật mỗi 1–5 giây (ClickHouse OLAP).

### 11.2. Báo cáo chi tiết

**Lead Performance:**
- Lead theo nguồn (Facebook, Google, Zalo,…)
- Tỷ lệ chuyển đổi theo landing page
- Chi phí mỗi Lead (CPL)
- AI Score distribution

**Sales Performance:**
- Doanh thu theo nhân viên / team / khu vực
- Win rate
- Average deal size
- Sales velocity

**Marketing Performance:**
- Facebook Ads → CRM → Revenue (feedback loop)
- ROI từng chiến dịch
- EMQ (Event Match Quality)

### 11.3. Xuất báo cáo

Click **Export** → chọn định dạng (PDF / Excel / CSV) → **Tải về**.

---

## 12. Settings & Billing

### 12.1. Settings (Cài đặt)

| Mục | Mô tả |
|-----|-------|
| **Profile** | Đổi tên, avatar, SĐT |
| **Password** | Đổi mật khẩu (yêu cầu mật khẩu cũ) |
| **2FA** | Bật/tắt YubiKey / TOTP |
| **Notifications** | Chọn kênh nhận thông báo (Email / Push / Telegram / SMS) |
| **API Keys** | Tạo key cho webhook / tích hợp bên thứ ba |
| **Team** | Quản lý user, role, invite link |
| **Pipelines** | Tùy biến quy trình bán hàng |
| **Custom Fields** | Thêm trường tùy chỉnh cho Lead/Deal/Contact |

### 12.2. Billing (Thanh toán)

Vào **Settings → Billing** để xem:
- **Plan hiện tại** (Starter / Pro / Enterprise)
- **Hóa đơn** (invoice) hàng tháng
- **Payment method** (thẻ / chuyển khoản / VNPay)
- **Usage** (số Lead / số user / storage)

**Nâng cấp plan:** Click **Upgrade** → chọn plan → thanh toán.

---

## 13. FAQ – Câu hỏi thường gặp

### Q1: Tôi quên mật khẩu, làm sao?

**A:** Tại trang đăng nhập, click "Quên mật khẩu" → nhập email → kiểm tra hộp thư (kể cả spam) → click link đặt lại.

### Q2: Tôi muốn thêm nhân viên mới vào team của tôi?

**A:** Vào **Users → Generate Invite Link** → chọn cấp bậc (Manager / Team Lead / Agent) → copy link → gửi cho nhân viên mới. Họ click link → đăng ký → tự động vào team của bạn.

### Q3: Landing Page của tôi không nhận được Lead?

**A:** Kiểm tra:
1. Landing Page đã được **xuất bản** chưa (Status = Published)?
2. URL có đang dùng HTTPS không?
3. DNS đã trỏ về RINCO chưa? (nếu dùng custom domain)
4. Vào **System Health** xem có service nào đỏ không.
5. Liên hệ support kèm `trace_id` nếu có.

### Q4: Làm sao để xem báo cáo doanh thu?

**A:** Vào **Analytics → Sales Performance**. Có thể filter theo nhân viên, team, khu vực, thời gian.

### Q5: AI Scoring có chính xác không?

**A:** AI được train trên dữ liệu lịch sử của chính tenant bạn (cold-start: dùng model generic). Sau 1000 Lead, model bắt đầu "học" pattern riêng. Độ chính xác thường đạt 75–85% AUC sau 3 tháng sử dụng.

### Q6: Tôi muốn tích hợp RINCO với phần mềm khác (vd: kế toán)?

**A:** Vào **Settings → API Keys** → tạo key mới → dùng key đó gọi RINCO REST API (xem [API Docs](https://docs.rinco.vn/api)).

### Q7: Hệ thống có hỗ trợ tiếng Anh không?

**A:** Hiện tại RINCO UI đang hỗ trợ **Tiếng Việt** (mặc định) và **Tiếng Anh**. Đổi ngôn ngữ: Click avatar → **Language → English**.

### Q8: Lead của tôi bị đánh dấu "Bot", tại sao?

**A:** Hệ thống có 3 lớp chống bot:
1. Wasm attestation (kiểm tra trình duyệt thật)
2. Argon2 PoW (yêu cầu tính toán)
3. Behavioral analysis (hành vi giống bot)

Lead bị nghi ngờ bot sẽ được đánh dấu nhưng vẫn lưu — bạn có thể xem lại trong **CRM → Leads → Filter: Source = suspected_bot**.

### Q9: Tôi cần hỗ trợ, liên hệ ai?

**A:**
- 📧 Email: support@rinco.vn
- 💬 Live chat trong app (góc phải)
- 📞 Hotline: 1900-xxxx (8:00–22:00)
- 📚 Docs: https://docs.rinco.vn

### Q10: Dữ liệu của tôi có được backup không?

**A:** Có. RINCO tự động backup:
- **PostgreSQL:** WAL streaming + daily snapshot (giữ 30 ngày)
- **ScyllaDB:** Daily snapshot
- **MinIO/S3:** Cross-region replication
- **ClickHouse:** Daily + 90 ngày retention

Khôi phục: liên hệ support với yêu cầu restore.

---

## Phụ lục: Phím tắt

| Phím tắt | Hành động |
|-----------|-----------|
| `Ctrl + K` | Mở command palette (tìm kiếm nhanh) |
| `Ctrl + N` | Tạo Lead mới |
| `Ctrl + D` | Tạo Deal mới |
| `Ctrl + /` | Toggle sidebar |
| `Ctrl + Shift + L` | Đăng xuất |
| `Esc` | Đóng modal |

---

**Liên hệ hỗ trợ:** support@rinco.vn · https://rinco.vn
