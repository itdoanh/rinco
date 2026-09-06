# Phần 5 – Landing Page + Facebook CAPI

> **Phân hệ:** Trang Landing Page cho thu Lead + Tích hợp Meta Pixel & Conversions API.
> **Mục tiêu:** Kế thừa 100% nội dung `chiase_cu/`, thay thư viện cũ (jQuery/Bootstrap), tracking chi tiết, chống giả mạo, bắn signal ngược Facebook để tối ưu quảng cáo.
> **Đặc thù:** Hybrid Dual-Tracking (Pixel + CAPI), HMAC chống CAPI Poisoning, EMQ (Event Match Quality) cao.

---

## Mục lục

1. [Nguyên tắc Kế thừa](#1-nguyên-tắc-kế-thừa)
2. [Kiến trúc](#2-kiến-trúc)
3. [Frontend Stack (Modern)](#3-frontend-stack-modern)
4. [Tracking Layer](#4-tracking-layer)
5. [Facebook CAPI Hybrid Dual-Tracking](#5-facebook-capi-hybrid-dual-tracking)
6. [Lead Form & Ingestion](#6-lead-form--ingestion)
7. [Chống Giả Mạo & Bot](#7-chống-giả-mạo--bot)
8. [Feedback Loop](#8-feedback-loop)
9. [Danh sách tính năng (≥ 100)](#9-danh-sách-tính-năng)
10. [Database Schema](#10-database-schema)
11. [API Surface](#11-api-surface)
12. [UI/UX Landing Page](#12-uiux-landing-page)
13. **MỤC LỤC MỚI (MỞ RỘNG)**
13. [Migration Plan từ chiase_cu](#13-migration-plan-từ-chiase_cu)
14. [Component Mapping chi tiết (jQuery/Bootstrap → Tailwind/shadcn)](#14-component-mapping-chi-tiết-jquerybootstrap--tailwindshadcn)
15. [Pixel Event Mapping (Cũ → Mới)](#15-pixel-event-mapping-cũ--mới)
16. [Tracking Implementation chi tiết](#16-tracking-implementation-chi-tiết)
17. [Wasm Attestation Module](#17-wasm-attestation-module)
18. [Argon2 PoW Challenge Protocol](#18-argon2-pow-challenge-protocol)
19. [CAPI Worker chi tiết](#19-capi-worker-chi-tiết)
20. [Webhook Feedback Loop chi tiết](#20-webhook-feedback-loop-chi-tiết)
21. [A/B Testing Framework](#21-ab-testing-framework)
22. [Multi-language Landing](#22-multi-language-landing)
23. [GDPR/PDPA Compliance](#23-gdprpdpa-compliance)
24. [Performance Optimization chi tiết](#24-performance-optimization-chi-tiết)
25. [Edge Cases & Error Handling](#25-edge-cases--error-handling)
26. [Testing Strategy](#26-testing-strategy)
27. [Disaster Recovery](#27-disaster-recovery)
28. [Cost Estimation](#28-cost-estimation)
29. [Implementation Roadmap chi tiết](#29-implementation-roadmap-chi-tiết)
30. [Open Questions / Cần user xác nhận](#30-open-questions--cần-user-xác-nhận)

---

## 1. Nguyên tắc Kế thừa

### 1.1. Giữ nguyên 100% nội dung `chiase_cu/`
- **Nội dung văn bản:** Copy 100% không thay đổi ý nghĩa.
- **Cấu trúc DOM:** Sections giữ nguyên thứ tự.
- **Hình ảnh:** Copy sang MinIO/S3, reference mới.
- **Branding cũ:** Logo, slogan giữ nguyên (trừ khi tenant yêu cầu thay).

### 1.2. Thay thư viện cũ
| Cũ | Mới | Lý do |
|----|-----|-------|
| jQuery | Vanilla JS / Web Components | Nhẹ hơn 90%, không bị blocking |
| Bootstrap 4 | Tailwind CSS v4 + shadcn-style | Utility-first, không CSS thừa |
| Font Awesome | Lucide Icons | SVG, tree-shake, đẹp hơn |
| Slick/Carousel cũ | Embla Carousel / Native | Hiệu năng cao |
| Smooth Scroll JS | CSS scroll-behavior | Native browser |
| Cookie consent cũ | Custom component GDPR/PDPA | Compliance |
| GA cũ (nếu có) | GA4 + Server-side tracking | Privacy-first |

### 1.3. Performance Target
- **FCP < 0.4s**
- **LCP < 0.8s**
- **TTI < 1.2s**
- **CLS < 0.05**
- **TBT < 100ms**

---

## 2. Kiến trúc

### 2.1. Sơ đồ
```
[Browser Client]
    │
    ├─► [Landing Page Renderer (Next.js)]
    │       - SSR / ISR
    │       - Critical CSS inline
    │       - Lazy load assets
    │
    ├─► [Meta Pixel (Client)]
    │       - PageView, ViewContent, Lead events
    │
    └─► [Go Ingestion API]
            │
            ├─► Wasm Attestation
            ├─► Argon2 PoW
            ├─► Validate HMAC
            ├─► Write ScyllaDB (raw lead)
            ├─► Emit NATS event
            │
            ├─► [ClickHouse Analytics]
            ├─► [AI Scoring Worker]
            └─► [FB CAPI Worker]
                    │
                    └─► Meta Graph API
```

### 2.2. Components
- **`landing-renderer`** (Next.js 15 + Bun): SSR Landing Page theo tenant.
- **`landing-ingest`** (Go + Huma): Nhận form submission.
- **`meta-capi`** (Go): Gửi event lên Meta Graph API.
- **`bot-detection`** (Rust Wasm): Phát hiện bot.
- **`analytics-collector`** (Go + ClickHouse): Ghi analytics event.

---

## 3. Frontend Stack (Modern)

### 3.1. Core
- **Next.js 15** (App Router) + **TypeScript** + **Bun runtime**.
- **React Server Components** cho nội dung.
- **Streaming SSR** cho nhanh.

### 3.2. Styling
- **Tailwind CSS v4** (utility-first).
- **CSS Modules** cho component-specific.
- **No Bootstrap, no jQuery.**

### 3.3. UI Library
- **Lucide Icons** (chính).
- **Heroicons** (backup).
- **Custom components** với React Aria (a11y).

### 3.4. Animation
- **Framer Motion** cho entrance/exit.
- **CSS transitions** cho hover/focus.
- **`prefers-reduced-motion`** respected.

### 3.5. Form
- **React Hook Form** + **Zod** validation.
- **Server-side validation** luôn (defense in depth).

### 3.6. Performance
- **Image:** `next/image` với AVIF/WebP auto.
- **Font:** Self-host Inter, font-display: swap.
- **Preload** critical resources.
- **Code splitting** automatic.
- **Edge SSR** cho tốc độ.

### 3.7. Cấu trúc thư mục
```
landing-renderer/
├── app/
│   ├── [tenant]/
│   │   ├── [...slug]/
│   │   │   └── page.tsx
│   │   └── layout.tsx
│   ├── layout.tsx
│   └── globals.css
├── components/
│   ├── hero/
│   ├── form/
│   ├── features/
│   ├── testimonials/
│   └── ui/  # shadcn-style
├── lib/
│   ├── api.ts
│   ├── tracking.ts
│   ├── capi.ts
│   └── validation.ts
├── public/
└── styles/
```

---

## 4. Tracking Layer

### 4.1. Event Taxonomy
```typescript
type TrackingEvent =
  | 'page_view'
  | 'view_content'
  | 'scroll_depth'      // 25%, 50%, 75%, 100%
  | 'time_on_page'      // 30s, 60s, 120s, 300s
  | 'cta_click'
  | 'form_start'
  | 'form_field_focus'
  | 'form_field_complete'
  | 'form_submit'
  | 'form_submit_success'
  | 'form_submit_error'
  | 'lead_qualified'
  | 'video_play'
  | 'video_pause'
  | 'video_complete'
  | 'outbound_click'
  | 'chat_open'
  | 'phone_click'
  | 'email_click'
  | 'social_click'
  | 'share'
  | 'exit_intent';
```

### 4.2. Data Captured
```typescript
interface TrackingPayload {
  // Identifiers
  event_id: string;          // UUIDv7
  session_id: string;        // UUIDv7
  anonymous_id: string;      // UUIDv7 (cookie)
  user_id?: string;          // Nếu đã login

  // Source
  utm_source: string;
  utm_medium: string;
  utm_campaign: string;
  utm_content?: string;
  utm_term?: string;
  fbclid?: string;
  fbp?: string;              // Facebook Browser ID
  fbc?: string;              // Facebook Click ID cookie
  gclid?: string;
  ttclid?: string;

  // Device
  ip_address: string;
  user_agent: string;
  screen_resolution: string;
  viewport_size: string;
  language: string;
  timezone: string;

  // Browser
  referrer: string;
  landing_page: string;
  current_page: string;
  page_title: string;

  // Engagement
  scroll_depth: number;
  time_on_page: number;
  click_count: number;

  // Custom
  event_name: string;
  properties: Record<string, any>;

  // Meta
  timestamp: number;         // ISO 8601
  tenant_id: string;
  campaign_id?: string;
}
```

### 4.3. Capture Methods
- **Client:** JS SDK tự inject tracking.
- **Server:** Cookie `__rinco_session` + IP header.
- **Hybrid:** Critical event được gửi cả client + server để redundancy.

### 4.4. Storage
- **ScyllaDB `raw_events` table:** Mỗi event 1 row.
- **ClickHouse `analytics`:** Aggregated view cho dashboard.
- **Retention:** 90 ngày raw, 2 năm aggregated.

---

## 5. Facebook CAPI Hybrid Dual-Tracking

### 5.1. Tại sao Hybrid?
- Apple ITP chặn 30% third-party cookie → mất data qua Pixel.
- AdBlocker chặn Pixel script.
- → Cần server-side track qua Conversions API.

### 5.2. Luồng Event

#### Client-side (Pixel)
```javascript
// On form submit
fbq('track', 'Lead', {
  content_name: 'Apex Fintech Landing',
  value: 0,
  currency: 'VND',
}, {
  eventID: event_id,  // Cùng eventID với server-side
});
```

#### Server-side (CAPI)
```go
// Go: meta-capi worker
type CAPIEvent struct {
  EventName      string    `json:"event_name"`
  EventTime      int64     `json:"event_time"`
  EventID        string    `json:"event_id"`        // Cùng với client
  EventSourceURL string    `json:"event_source_url"`
  ActionSource   string    `json:"action_source"`   // "website"
  UserData        UserData  `json:"user_data"`
  CustomData     CustomData `json:"custom_data"`
}

type UserData struct {
  Email        []string `json:"em"`     // SHA-256 normalized
  Phone        []string `json:"ph"`     // SHA-256 normalized (digits only)
  FirstName    []string `json:"fn"`     // SHA-256 normalized
  LastName     []string `json:"ln"`
  City         []string `json:"ct"`
  State        []string `json:"st"`
  Country      []string `json:"country"` // ISO 3166-1 alpha-2
  ZipCode      []string `json:"zp"`
  ExternalID   []string `json:"external_id"` // RINCO user_id
  ClientIP     string   `json:"client_ip_address"`
  ClientUA     string   `json:"client_user_agent"`
  FBC          string   `json:"fbc"`
  FBP          string   `json:"fbp"`
  SubscriptionID string `json:"subscription_id"`
}
```

### 5.3. SHA-256 Normalization (Meta Standard)
```go
func NormalizeEmail(email string) string {
  return sha256(strings.ToLower(strings.TrimSpace(email)))
}

func NormalizePhone(phone string) string {
  digits := extractDigits(phone)
  if !strings.HasPrefix(digits, "84") {
    digits = "84" + strings.TrimPrefix(digits, "0")
  }
  return sha256(digits)
}

func NormalizeName(name string) string {
  return sha256(strings.ToLower(strings.TrimSpace(name)))
}
```

### 5.4. HMAC Signature (Chống CAPI Poisoning)
```go
func GenerateLeadSignature(leadID, fbclid, timestamp string, payload []byte) string {
  h := hmac.New(sha256.New, []byte(secretKey))
  h.Write([]byte(leadID))
  h.Write([]byte(fbclid))
  h.Write([]byte(timestamp))
  h.Write(payload)
  return hex.EncodeToString(h.Sum(nil))
}
```
- Worker verify trước khi gửi Meta.
- Nếu signature không khớp → reject + log alert.

### 5.5. EventID Deduplication
- Client và Server dùng **cùng eventID** cho cùng 1 event.
- Meta Graph API tự deduplicate trong vòng 48h.

### 5.6. CAPI Worker Logic
```
1. Nhận event từ NATS topic "lead.events"
2. Verify HMAC signature
3. Build CAPIEvent với normalized user_data
4. Send POST https://graph.facebook.com/v18.0/{pixel_id}/events
5. Retry với exponential backoff (3 lần)
6. Nếu fail → Circuit Breaker open
7. Log response (events_received, messages)
8. Update ClickHouse metric
```

### 5.7. Event Match Quality (EMQ)
- Tối ưu score bằng cách gửi đầy đủ fields.
- Score target ≥ 7/10.
- Track EMQ theo tenant, alert nếu < 6.

---

## 6. Lead Form & Ingestion

### 6.1. Form Schema (per tenant)
```typescript
interface LeadFormConfig {
  tenant_id: string;
  campaign_id: string;
  fields: FormField[];
  submit_endpoint: string;
  success_redirect?: string;
  success_message?: string;
  // Tracking config
  enable_pixel: boolean;
  enable_capi: boolean;
  // Bot protection
  enable_wasm_attestation: boolean;
  enable_argon2_pow: boolean;
  // Validation
  phone_required: boolean;
  email_required: boolean;
  consent_required: boolean;
}

interface FormField {
  code: string;
  label: string;
  type: 'text' | 'phone' | 'email' | 'select' | 'textarea' | 'hidden';
  required: boolean;
  placeholder?: string;
  options?: string[];
  validation?: Record<string, any>;
}
```

### 6.2. Client-side Flow
```
1. User load page → JS init tracking
2. Scroll, time_on_page tracking
3. Click CTA → track cta_click
4. Focus first field → track form_start
5. Fill fields → validate realtime (Zod)
6. Submit →
   a. fbq('track', 'Lead', {...}, {eventID})
   b. POST /api/ingest/v1/leads
      - HMAC signed
      - Wasm attestation header
      - Argon2 PoW header (nếu traffic spike)
7. Server response → show success/error
```

### 6.3. Server-side Flow (`landing-ingest`)
```
1. Receive POST /api/ingest/v1/leads
2. Middleware:
   - Verify HMAC
   - Verify Wasm attestation
   - Check Argon2 PoW (nếu có)
   - Rate limit per IP
3. Validate payload (Zod/JSON Schema)
4. Generate event_id (UUIDv7)
5. Write ScyllaDB:
   - Table raw_leads
   - Partition by tenant_id, cluster by created_at
6. Emit NATS event:
   - Topic: lead.created
   - Payload: full lead + metadata
7. Subscribe:
   - ClickHouse consumer (async)
   - AI Scoring worker (async)
   - Meta CAPI worker (async)
   - Notification (if high score)
8. Response: 200 OK with lead_id + event_id
```

### 6.4. Edge Cases
- **Submit duplicate:** Idempotency key (session_id + form_id).
- **Bot submit:** Mark as bot, không ghi CAPI nhưng ghi analytics.
- **Failed validation:** Return chi tiết field errors.
- **Rate limit:** Return 429 + Retry-After.
- **System down:** Fallback write Valkey queue → worker xử lý sau.

---

## 7. Chống Giả Mạo & Bot

### 7.1. Wasm Hardware Attestation
- Trình duyệt chạy 1 module WebAssembly encrypted.
- Wasm verify:
  - User Agent không phải Headless Chrome.
  - Mouse movement real (không linear).
  - Touch events real (mobile).
  - WebGL renderer != "swiftshader".
  - Canvas hash stable.
- Pass → cấp `X-RINCO-Attestation` token.
- Fail → form bị ẩn, hiện CAPTCHA.

### 7.2. Argon2 Proof-of-Work
- Khi traffic > threshold → Gateway issue challenge.
- Client phải compute `Argon2id(nonce, ts, salt)` trong ~20ms.
- Bot (không có JS) fail ngay.

### 7.3. eBPF/XDP L4 Filtering
- Drop known bot IP ranges.
- Rate limit per IP ở NIC level.
- Geo-block nếu cần.

### 7.4. Fingerprinting
- Canvas fingerprint.
- AudioContext fingerprint.
- WebGL fingerprint.
- Font enumeration.
- Combine → unique visitor_id.

### 7.5. CAPTCHA Fallback
- hCaptcha hoặc Cloudflare Turnstile.
- Chỉ show khi bot score > threshold.

---

## 8. Feedback Loop

### 8.1. Luồng ngược về Meta
```
[CRM] Lead → Qualified → Won (Deal)
                  │
                  ▼
        [meta-capi worker nhận event]
                  │
                  ▼
        POST /events với event_name="Purchase"
                  │
                  └─► Meta Ads Algorithm
                       Optimizes cho high-LTV leads
```

### 8.2. Conversion Events
| Event | Khi nào | Value |
|-------|---------|-------|
| `Lead` | Form submit | 0 |
| `QualifiedLead` | AI score ≥ 70 | 0 |
| `Schedule` | Đặt lịch họp | 0 |
| `Purchase` | Won deal | Deal value |
| `Subscribe` | Subscription active | Plan price |
| `Custom_SQL_Qualified` | SQL qualified | 0 |
| `Custom_High_Value` | Score ≥ 90 | 0 |

### 8.3. Value Optimization
- Gửi Monetary Value chính xác từ Deal.
- Currency theo tenant setting.
- Dùng `predicted_ltv` từ AI scoring (nếu chưa có deal).

### 8.4. Server-Side Only Events
- Một số event chỉ nên gửi server-side (PII nhiều).
- `Custom_Audience_Upload` cho retargeting.
- `Custom_Conversion` cho offline conversion.

---

## 9. Danh sách tính năng (≥ 100)

### 9.1. Landing Page Content (1-25)
1. Hero section (ảnh + headline + CTA).
2. Sub-hero (USP list).
3. About / Giới thiệu.
4. Services grid.
6. Features list (icon + text).
7. Testimonials carousel.
8. Team section.
9. Pricing table (nếu có).
10. FAQ accordion.
11. Contact form.
12. Footer (multi-column).
13. Sticky CTA bar.
14. Floating WhatsApp button.
15. Cookie consent banner.
16. Privacy policy page.
17. Terms of service page.
18. Cookie policy page.
19. 404 page custom.
20. 500 page custom.
21. Sitemap page.
22. Search results page.
23. Multi-language toggle.
24. Theme switcher (light/dark nếu áp dụng).
25. Print-friendly version.

### 9.2. Forms (26-50)
26. Form builder cho tenant.
27. Multi-step form.
28. Conditional fields.
29. Custom validation messages.
30. Auto-save draft.
31. Pre-fill từ URL params.
32. Hidden fields (utm tracking).
33. File upload field.
34. Image upload field.
35. Date picker field.
36. Phone input với country code.
37. Email validation realtime.
38. Captcha integration.
39. Consent checkbox (GDPR/PDPA).
40. Submit success animation.
41. Submit error handling.
42. Form A/B test.
43. Form analytics.
44. Form abandonment tracking.
45. Form webhook.
46. Form email notification.
47. Form CRM auto-create.
48. Form duplicate prevention.
49. Form spam filter.
50. Form offline mode.

### 9.3. Tracking & Analytics (51-75)
51. Page view tracking.
52. Scroll depth tracking.
53. Time on page tracking.
54. Click heatmap.
55. CTA click tracking.
56. Form interaction tracking.
57. Video engagement tracking.
58. Outbound link tracking.
59. Phone click tracking (call tracking).
60. Email click tracking.
61. Social share tracking.
62. Chat open tracking.
63. Exit intent popup.
64. Session recording (tùy chọn).
65. UTM builder.
66. UTM tracking.
67. fbclid tracking.
68. gclid tracking.
69. ttclid tracking.
70. Referrer tracking.
72. Geo location tracking.
73. Device detection.
74. Browser detection.
75. Real-time visitor counter.

### 9.4. Facebook Integration (76-95)
76. Pixel install (per tenant).
77. Pixel event mapping.
78. CAPI gateway.
79. CAPI event deduplication.
80. CAPI HMAC signing.
81. CAPI retry logic.
82. CAPI circuit breaker.
83. Server-side event.
84. Client-side event.
85. Hybrid dual tracking.
86. Custom audience creation.
87. Custom conversion setup.
88. EMQ monitoring.
89. Pixel diagnostics.
90. CAPI diagnostics.
91. Conversion API token rotation.
92. Domain verification helper.
93. Aggregated event measurement.
94. Conversion value optimization.
95. Offline conversion upload.

### 9.5. Bot Protection (96-110)
96. Wasm attestation.
97. Argon2 PoW.
98. hCaptcha.
99. Cloudflare Turnstile.
100. IP blacklist.
101. Geo-blocking.
102. Rate limit per IP.
103. Rate limit per fingerprint.
104. Honeypot field.
105. Time-to-fill check.
96. Mouse movement analysis.
107. Touch event analysis.
108. Canvas fingerprint.
109. WebGL fingerprint.
110. Audio fingerprint.

### 9.6. Admin (111-130)
111. Landing page editor (CMS).
113. Multi-tenant landing pages.
114. A/B test creator.
115. Form version control.
116. Form rollback.
117. Pixel ID management.
118. CAPI token management.
119. Conversion event mapping.
120. Test event helper (Meta Test Events).
121. Event log viewer.
122. Error tracking.
123. EMQ dashboard.
124. Funnel analytics.
125. Cohort analysis.
126. ROI calculator.
127. Webhook config.
128. Email notification config.
129. Slack/Telegram notification.
130. Custom domain SSL.

---

## 10. Database Schema

### 10.1. ScyllaDB Tables
```cql
-- Raw lead events
CREATE TABLE rinco.raw_leads (
  tenant_id text,
  event_id uuid,
  created_at timestamp,
  lead_id uuid,
  payload text,             -- JSON
  user_data text,           -- JSON
  source text,
  campaign_id text,
  event_name text,
  PRIMARY KEY ((tenant_id, created_at), event_id)
) WITH CLUSTERING ORDER BY (event_id DESC);

-- Bot scores
CREATE TABLE rinco.bot_events (
  tenant_id text,
  event_id uuid,
  score double,
  reasons list<text>,
  PRIMARY KEY ((tenant_id), event_id)
);

-- CAPI events log
CREATE TABLE rinco.capi_events (
  tenant_id text,
  event_id uuid,
  sent_at timestamp,
  response_code int,
  events_received int,
  fbtrace_id text,
  error_message text,
  PRIMARY KEY ((tenant_id, sent_at), event_id)
);
```

### 10.2. ClickHouse Tables
```sql
-- Aggregated analytics
CREATE TABLE rinco.analytics (
  tenant_id String,
  campaign_id String,
  event_date Date,
  event_hour DateTime,
  event_name LowCardinality(String),
  source LowCardinality(String),
  utm_source LowCardinality(String),
  utm_campaign LowCardinality(String),
  country LowCardinality(String),
  device_type LowCardinality(String),
  browser LowCardinality(String),
  count UInt64,
  unique_users AggregateFunction(uniq, String),
  total_value Decimal(18, 4)
) ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_date, event_hour, event_name);

-- Funnel
CREATE TABLE rinco.funnel (
  tenant_id String,
  campaign_id String,
  event_date Date,
  step String,              -- 'page_view','cta_click','form_start','form_submit','qualified'
  count UInt64
) ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (tenant_id, event_date, step);
```

### 10.3. PostgreSQL (Metadata)
```sql
-- Tenant Pixel config
CREATE TABLE tenant_pixels (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  pixel_id TEXT NOT NULL,
  capi_token_encrypted TEXT NOT NULL,
  test_event_code TEXT,
  emq_target INT DEFAULT 7,
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Landing pages
CREATE TABLE landing_pages (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  slug TEXT NOT NULL,
  template TEXT NOT NULL,
  content JSONB NOT NULL,
  form_config JSONB NOT NULL,
  seo_config JSONB,
  is_published BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(tenant_id, slug)
);

-- A/B tests
CREATE TABLE ab_tests (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  name TEXT NOT NULL,
  variants JSONB NOT NULL,
  goal TEXT NOT NULL,
  status TEXT DEFAULT 'DRAFT',
  started_at TIMESTAMPTZ,
  ended_at TIMESTAMPTZ
);
```

---

## 11. API Surface

### 11.1. Public Landing
```
GET    /l/:tenant/:slug              # Render landing page
GET    /api/public/lng/:tenant/:slug/config  # Get form config
POST   /api/ingest/v1/leads          # Submit lead
POST   /api/ingest/v1/events         # Submit any event (analytics)
GET    /api/ingest/v1/pow-challenge  # Get PoW if needed
```

### 11.2. Tenant Admin
```
GET    /api/lp-admin/v1/pages
POST   /api/lp-admin/v1/pages
PATCH  /api/lp-admin/v1/pages/:id
DELETE /api/lp-admin/v1/pages/:id
POST   /api/lp-admin/v1/pages/:id/publish
POST   /api/lp-admin/v1/pages/:id/ab-test

GET    /api/lp-admin/v1/pixels
POST   /api/lp-admin/v1/pixels
PATCH  /api/lp-admin/v1/pixels/:id

GET    /api/lp-admin/v1/forms
POST   /api/lp-admin/v1/forms
PATCH  /api/lp-admin/v1/forms/:id

GET    /api/lp-admin/v1/analytics/overview
GET    /api/lp-admin/v1/analytics/funnel
GET    /api/lp-admin/v1/analytics/campaigns

GET    /api/lp-admin/v1/capi-log
POST   /api/lp-admin/v1/test-event   # Send test to Meta
```

---

## 12. UI/UX Landing Page

### 12.1. Layout chính (kế thừa `chiase_cu/`)
```
┌────────────────────────────────────────────────────┐
│  Navbar: Logo │ Menu │ CTA Button                  │
├────────────────────────────────────────────────────┤
│  HERO                                              │
│  - Headline (kế thừa từ chiase_cu)                 │
│  - Subheadline                                      │
│  - CTA Button                                       │
│  - Hero Image                                       │
├────────────────────────────────────────────────────┤
│  Trust Bar / Logo Strip                             │
├────────────────────────────────────────────────────┤
│  Features Grid (4-6 cards)                          │
├────────────────────────────────────────────────────┤
│  About / Story                                      │
├────────────────────────────────────────────────────┤
│  Services (grid/list)                               │
├────────────────────────────────────────────────────┤
│  Testimonials Carousel                              │
├────────────────────────────────────────────────────┤
│  FAQ Accordion                                      │
├────────────────────────────────────────────────────┤
│  Contact Form                                       │
├────────────────────────────────────────────────────┤
│  Footer (multi-column)                              │
└────────────────────────────────────────────────────┘
```

### 12.2. Components (Next.js + React Aria)
- Hero (variant: image, video, animated)
- FeaturesGrid
- TestimonialsCarousel (Embla)
- FAQAccordion (Radix)
- ContactForm (React Hook Form + Zod)
- CookieConsent (PDPA-compliant)
- StickyHeader
- MobileMenu
- FloatingCTA
- VideoEmbed (lazy)
- ImageOptimized (next/image)

### 12.3. Theme System
- CSS Variables (HSL-based).
- Easy theme switching (tenant-specific).
- High contrast support.
- Reduced motion support.

---

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-LP-01 | Lighthouse Performance ≥ 95 | Score |
| AC-LP-02 | FCP < 0.4s | Lighthouse |
| AC-LP-03 | Form submit → Lead in Scylla < 50ms | p95 |
| AC-LP-04 | CAPI success rate ≥ 99.5% | Metric |
| AC-LP-05 | EMQ ≥ 7 | Meta diagnostic |
| AC-LP-06 | Bot detection rate ≥ 99% | Test suite |
| AC-LP-07 | 100% nội dung từ chiase_cu được giữ | Manual QA |

---

# PHẦN MỞ RỘNG (Migration chiase_cu + Code Examples)

> Phần này bổ sung theo yêu cầu đặc biệt: migrate 100% từ `chiase_cu/`, code examples, edge cases.

---

## 13. Migration Plan từ chiase_cu

### 13.1. Phân tích file `chiase_cu/index.html` (785 dòng)

Tổng quan cấu trúc cũ:
| Section | Lines (approx) | Purpose |
|---------|---------------|---------|
| `<head>` | 1-83 | Meta tags, fonts (Inter + Plus Jakarta), Tailwind CDN, AOS, schema.org, Meta Pixel |
| Top bar | 84-99 | Hotline + CTA "GIỮ VÉ ZOOM" |
| Header sticky | 100-122 | Logo + nav + CTA mobile menu |
| Mobile Menu | 123-146 | Slide-in drawer |
| **Section 1: Hero** | 147-256 | Headline + sub + form (inline) + hero image + 3 chips |
| **Section 2: Logo Cloud** | 257-283 | MXV, CQG, NYMEX, CBOT, LME marquee |
| **Section 3: Bối cảnh** | 284-353 | 4 trụ cột lợi thế T+0/2 chiều/Margin/Quốc tế |
| **Section 4: Diễn giả** | 354-394 | Card Nguyễn Tuấn Anh |
| **Section 5: Uy tín** | 395-444 | Top 1 banner + 4 trust badges |
| **Section 6: Form multi-step** | 445-565 | 2-step form (chọn kênh → info) |
| Section 7: Risk + Company | 566-606 | Cảnh báo rủi ro + info APEX |
| Footer | 607-622 | Copyright + links |
| Sticky CTA | 623-633 | Mobile + desktop bottom bar |
| Modal Popup | 634-770 | Lead capture modal |
| Scripts | 771-785 | main.js, tracker.js, AOS init |

### 13.2. Migration Roadmap

| Phase | Tuần | Công việc | Output |
|-------|------|-----------|--------|
| **P1: Inventory** | 1 | Đọc kỹ `chiase_cu/`, liệt kê tất cả assets (text, image, font, link) | File `migration/INVENTORY.md` |
| **P2: Design Tokens** | 1 | Extract colors, fonts, spacing từ CSS | `tailwind.config.ts` mới |
| **P3: Components Migration** | 2-3 | Migrate từng section sang React component | 12 components mới |
| **P4: Asset Migration** | 2 | Upload ảnh lên MinIO, replace paths | Image URLs mới |
| **P5: Tracking Layer** | 3 | Implement tracking SDK riêng | `lib/tracking.ts` |
| **P6: Form Engine** | 3-4 | Build Form Engine (React Hook Form + Zod) | Dynamic form |
| **P7: Modal + Multi-step** | 4 | Migrate modal + multi-step form | `ModalLead` + `MultiStepForm` |
| **P8: AOS → Framer** | 4 | Replace AOS với Framer Motion | Animations mới |
| **P9: A11y** | 5 | Add ARIA, keyboard nav, focus trap | Audit pass |
| **P10: SEO** | 5 | Server-side rendering + meta tags | Lighthouse SEO 100 |
| **P11: Performance** | 6 | Lighthouse optimization | FCP < 0.4s, LCP < 0.8s |
| **P12: Test** | 6-7 | Visual regression test với Playwright | All test pass |

### 13.3. Inventory chi tiết từ `chiase_cu/index.html`

```yaml
content_blocks:
  - section: hero
    headline:
      line1: "Tối Ưu Dòng Tiền 2026 Với"
      line2: "Kênh Đầu Tư Hàng Hóa Phái Sinh"
      subline: "(Chính Thống - Quản Lý Bởi Bộ Công Thương & MXV)"
    subheadline: "Giải mã cơ chế giao dịch T+0 & sinh lời 2 chiều linh hoạt..."
    urgency: "⚡ Số lượng vé đăng ký có hạn..."
    cta_text: "GIỮ VÉ ZOOM & NHẬN EBOOK MIỄN PHÍ"
    hero_image: "anh/anhchandung_chuyengia_nguyentuananh.webp"
    chips:
      - emoji: "🏆"
        title: "TOP 1"
        subtitle: "Thị trường Q2/2026"
      - emoji: "⚡"
        title: "T+0"
        subtitle: "Thanh toán tức thì"
      - emoji: "🛡"
        title: "MXV"
        subtitle: "Thành viên 080"

  - section: logo_cloud
    title: "Được cấp phép chính thức và liên thông giao dịch quốc tế"
    logos: ["MXV", "CQG", "NYMEX", "CBOT", "LME"]

  - section: market_context
    tag: "BỐI CẢNH THỊ TRƯỜNG 2026"
    headline: "TẠI SAO HÀNG HÓA PHÁI SINH ĐANG LÀ KÊNH ĐẦU TƯ BÙNG NỔ?"
    data_point: "Thanh khoản bình quân 7.500 tỷ – 17.000 tỷ đồng/ngày"
    cards:
      - num: "01"
        icon: "⚡"
        title: "Thanh toán T+0 Linh hoạt"
        description: "Nghiệp vụ + Giá trị đầu tư"
      - num: "02"
        icon: "⇅"
        title: "Cơ chế Giao dịch 2 Chiều"
      - num: "03"
        icon: "💰"
        title: "Tỷ lệ Ký quỹ Margin Tối ưu"
      - num: "04"
        icon: "🌐"
        title: "Minh bạch & Liên thông Quốc tế"

  - section: speaker
    name: "NGUYỄN TUẤN ANH"
    title: "Giám đốc Phát triển Thị trường"
    company: "APEX Fintech"
    bio_bullets:
      - "Nhiều năm kinh nghiệm phân tích tài chính phái sinh..."
      - "Chứng chỉ chính thức bởi MXV"
      - "Chuyên gia cấu trúc giải pháp phòng ngừa rủi ro giá..."
      - "Tư vấn 1:1 cho nhà đầu tư"
    photo: "anh/anhchandung_chuyengia_nguyentuananh.webp"

  - section: trust
    banner_image: "anh/apex_top1_thiphan_quy2_2026.webp"
    badges:
      - icon: "🏆"
        title: "TOP 1 Thị Phần Giao Dịch"
        description: "MXV Q2/2026"
      - icon: "📜"
        title: "Pháp Lý Minh Bạch"
      - icon: "💻"
        title: "Hạ Tầng CQG Chuẩn Quốc Tế"
      - icon: "🌍"
        title: "Dữ Liệu Thời Gian Thực"

  - section: registration_form
    multistep: true
    steps:
      - step: 1
        title: "Bạn muốn nhận suất qua kênh nào?"
        options:
          - { emoji: "📲", name: "Nhận qua Zalo", sub: "Nhanh & tiện lợi nhất", channel: "zalo" }
          - { emoji: "📞", name: "Nhận qua liên hệ SĐT", sub: "Chuyên viên gọi tư vấn", channel: "phone" }
      - step: 2
        title: "Nhập thông tin để Hệ thống Gửi Vé Zoom & Ebook Miễn Phí"
        fields: [name, phone]
    event_info:
      date: "20:00 – 22:00 | 07/09/2026"
      format: "Zoom Online"
      gift: "Tặng Bộ 10 Ebook + Tư vấn 1:1"

  - section: risk_warning
    icon: "⚠"
    title: "CẢNH BÁO RỦI RO THỊ TRƯỜNG"
    text: "Giao dịch Hàng hóa Phái sinh..."

  - section: company_info
    name: "CÔNG TY CỔ PHẦN CÔNG NGHỆ TÀI CHÍNH APEX"
    address: "B-TT8-3 Him Lam, Vạn Phúc, Phường Hà Đông, Thành phố Hà Nội"
    hotline: "0984386538"
    website: "https://hanghoaphaisinh.net/"

  - section: footer
    copyright: "© 2026 hanghoaphaisinh.net"
    links: ["Điều khoản dịch vụ", "Chính sách bảo mật", "Cảnh báo rủi ro"]
```

### 13.4. Asset Migration Mapping

```yaml
assets:
  - source: "assets/img/logo_APEX_K_NEN.webp"
    target: "minio://rinco-tenant-assets/{tenant_id}/logo.webp"
    sizes: [16, 32, 64, 128, 256]
    formats: [webp, avif]

  - source: "anh/anhchandung_chuyengia_nguyentuananh.webp"
    target: "minio://rinco-tenant-assets/{tenant_id}/speaker.webp"
    sizes: [400, 800, 1200]
    formats: [webp, avif]

  - source: "anh/apex_top1_thiphan_quy2_2026.webp"
    target: "minio://rinco-tenant-assets/{tenant_id}/top1-banner.webp"
    sizes: [800, 1600]
    formats: [webp]

  - source: "assets/css/style.css"
    target: "apps/landing-renderer/styles/legacy.css"
    action: "Reference + adapt vào Tailwind"

  - source: "assets/js/main.js"
    target: "apps/landing-renderer/lib/legacy/main.ts"
    action: "Rewrite bằng TypeScript + React"

  - source: "assets/js/tracker.js"
    target: "apps/landing-renderer/lib/tracking/rinco-tracker.ts"
    action: "Replace với official SDK"
```

---

## 14. Component Mapping chi tiết (jQuery/Bootstrap → Tailwind/shadcn)

| Cũ (chiase_cu/) | Mới (RINCO) | Notes |
|----------------|-------------|-------|
| `<button class="btn-cta">` (custom Tailwind CDN) | `<Button variant="cta" size="lg">` (shadcn) | Wrap với React Aria |
| `class="form-input"` (custom) | `<Input className="h-12">` (shadcn) | Validation từ React Hook Form |
| `class="modal-overlay"` (vanilla JS toggle) | `<Dialog>` (Radix UI) | Focus trap + ESC close |
| `class="webinar-tag"` | `<Badge variant="outline">` (shadcn) | Animated dot |
| `class="feature-card"` (CSS grid) | `<FeatureCard>` (custom) | với framer-motion stagger |
| `class="speaker-card"` (CSS only) | `<SpeakerCard>` (custom) | Hover scale |
| `class="trust-badge"` | `<TrustBadge>` (custom) | Icon + title + desc |
| `class="step-indicator"` | `<Stepper>` (custom) | Active state animation |
| `class="multistep-pane"` | `<MultiStepForm>` (custom) | Auto-save draft |
| `class="channel-btn"` | `<ChannelOption>` (custom) | Click để next step |
| `class="hero-chip"` | `<HeroChip>` (custom) | Floating animation |
| `class="success-icon"` (CSS checkmark) | `<SuccessAnimation>` (framer-motion) | SVG path animation |
| `class="sticky-btn"` | `<FloatingCTA>` (custom) | Visibility threshold |
| `data-modal-trigger` | Radix Dialog controlled | Trigger button → open state |
| `class="logo-marquee"` (CSS animation) | `<Marquee>` (custom với CSS animation) | Pause on hover |
| AOS `data-aos="fade-up"` | framer-motion `<motion.div whileInView>` | Intersection Observer |
| jQuery `.click()` handler | React `onClick` | Native |
| jQuery `.fadeIn()` | framer-motion `animate={{ opacity: 1 }}` | |
| jQuery `.scroll()` listener | `useEffect` với `IntersectionObserver` hook | |
| jQuery `$(window).on('scroll')` | `useScroll` hook từ framer-motion | |
| `pattern="^(\+84\|0)\d{9,10}$"` inline | Zod schema: `z.string().regex(/^(\+84\|0)\d{9,10}$/)` | |
| `inputmode="numeric"` | Giữ nguyên | Mobile UX |
| `autocomplete="tel"` | Giữ nguyên | |
| `tabindex="-1"` honeypot | Giữ nguyên với React ref | |
| Mobile menu `<div id="mobileMenu">` | Radix `Dialog` với side="right" | Better accessibility |

### 14.1. Hero Section Component (Next.js)

```tsx
// apps/landing-renderer/components/hero/HeroSection.tsx
import { motion } from 'framer-motion';
import { ArrowRight, Shield } from 'lucide-react';
import { useEffect, useState } from 'react';
import { LeadForm } from '../form/LeadForm';
import { HeroChip } from './HeroChip';

interface HeroSectionProps {
  tenant: TenantConfig;
  content: HeroContent;
  formConfig: LeadFormConfig;
}

export function HeroSection({ tenant, content, formConfig }: HeroSectionProps) {
  const [showStickyCta, setShowStickyCta] = useState(false);

  // Track scroll for sticky CTA (thay thế jQuery scroll listener)
  useEffect(() => {
    const onScroll = () => setShowStickyCta(window.scrollY > 600);
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  return (
    <section
      id="hero"
      className="relative overflow-hidden bg-gradient-to-b from-white to-gray-50 pt-10 pb-16 md:pt-16 md:pb-24"
    >
      {/* Floating shapes - chuyển từ CSS float-shape */}
      <div className="float-shape float-shape-1" aria-hidden="true" />
      <div className="float-shape float-shape-2" aria-hidden="true" />
      <div className="float-shape float-shape-3" aria-hidden="true" />

      <div className="relative z-10 max-w-7xl mx-auto px-4">
        <div className="flex flex-col lg:flex-row items-center gap-10 lg:gap-12">
          {/* LEFT */}
          <motion.div
            className="flex-1 w-full"
            initial={{ opacity: 0, x: -30 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.8, ease: 'easeOutCubic' }}
          >
            <div className="webinar-tag inline-flex items-center gap-2 px-4 py-2 rounded-full text-white text-xs md:text-sm font-bold uppercase tracking-wide mb-5">
              <span className="webinar-tag-dot" />
              {content.eyebrow}  {/* "CHIA SẺ TRỰC TUYẾN MIỄN PHÍ QUA ZOOM ONLINE" */}
            </div>

            <h1 className="font-heading text-3xl sm:text-4xl md:text-5xl lg:text-5xl font-black leading-tight mb-4">
              <span className="block text-navy-900">{content.headline.line1}</span>
              <span className="block gradient-text">{content.headline.line2}</span>
              <span className="block text-base md:text-lg lg:text-xl text-gray-500 font-bold mt-3">
                {content.headline.subline}
              </span>
            </h1>

            <p className="text-gray-600 text-base md:text-lg leading-relaxed mb-5 max-w-xl">
              {content.subheadline}
            </p>

            <UrgencyBanner seatsLeft={content.urgency} />

            <LeadForm config={formConfig} variant="inline" />
          </motion.div>

          {/* RIGHT - Hero image */}
          <motion.div
            className="hidden lg:block flex-shrink-0 w-full max-w-md"
            initial={{ opacity: 0, x: 30 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 1, ease: 'easeOutCubic' }}
          >
            <HeroImage
              src={content.heroImage}
              alt={content.speaker.name}
              chips={content.chips}
            />
          </motion.div>
        </div>
      </div>
    </section>
  );
}

function UrgencyBanner({ seatsLeft }: { seatsLeft: number | string }) {
  return (
    <div className="flex items-center gap-2 text-red-600 text-sm font-bold mb-6 bg-red-50 border border-red-200 rounded-full px-4 py-2 inline-flex">
      <Zap className="w-4 h-4 animate-pulse flex-shrink-0" />
      <span>⚡ Số lượng <strong>{seatsLeft}</strong> vé đăng ký có hạn...</span>
    </div>
  );
}
```

### 14.2. LeadForm Component (React Hook Form + Zod)

```tsx
// apps/landing-renderer/components/form/LeadForm.tsx
'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useState } from 'react';
import { Shield, Loader2, Check } from 'lucide-react';
import { useTracker } from '@/lib/tracking/useTracker';
import { usePowChallenge } from '@/lib/pow/usePowChallenge';
import { submitLead } from '@/lib/api/leads';

const phoneRegex = /^(\+84|0)\d{9,10}$/;

const leadSchema = z.object({
  name: z.string().min(1, 'Vui lòng nhập họ tên').max(255),
  phone: z.string().regex(phoneRegex, 'Số điện thoại không hợp lệ (VD: 0987654321)'),
  consent: z.literal(true, {
    errorMap: () => ({ message: 'Vui lòng đồng ý điều khoản' }),
  }),
  // Honeypot
  website: z.string().max(0).optional(),
});

type LeadFormData = z.infer<typeof leadSchema>;

interface LeadFormProps {
  config: LeadFormConfig;
  variant: 'inline' | 'modal' | 'section';
}

export function LeadForm({ config, variant }: LeadFormProps) {
  const { track, eventId } = useTracker();
  const { challenge, solvePow } = usePowChallenge();
  const [submitState, setSubmitState] = useState<'idle' | 'submitting' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState('');

  const { register, handleSubmit, formState: { errors }, reset } = useForm<LeadFormData>({
    resolver: zodResolver(leadSchema),
    mode: 'onBlur',
  });

  const onSubmit = async (data: LeadFormData) => {
    setSubmitState('submitting');
    setErrorMessage('');

    try {
      // 1. Compute Argon2 PoW nếu cần
      let powHeader: { nonce: string; hash: string; ts: number } | undefined;
      if (challenge) {
        const result = await solvePow(challenge);
        powHeader = result;
      }

      // 2. Track start submit (Meta Pixel client-side)
      track('form_submit_attempt', { event_id: eventId, ...data });

      // 3. Fire Meta Pixel Lead (matching eventID với server)
      if (typeof window !== 'undefined' && (window as any).fbq) {
        (window as any).fbq('track', 'Lead', {
          content_name: 'APEX Fintech Webinar',
          value: 0,
          currency: 'VND',
        }, { eventID: eventId });
      }

      // 4. Submit to server
      const result = await submitLead({
        ...data,
        event_id: eventId,
        tenant_id: config.tenant_id,
        campaign_id: config.campaign_id,
        // ...other tracking metadata
      }, { powHeader });

      // 5. Success
      setSubmitState('success');
      track('form_submit_success', { lead_id: result.lead_id });
      reset();
    } catch (err: any) {
      setSubmitState('error');
      setErrorMessage(err.message || 'Đã có lỗi xảy ra, vui lòng thử lại');
      track('form_submit_error', { error: err.message });
    }
  };

  if (submitState === 'success') {
    return <SuccessState onReset={() => setSubmitState('idle')} />;
  }

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className={`form-card ${variant === 'inline' ? 'max-w-md' : 'w-full'}`}
      autoComplete="on"
      noValidate
    >
      <div className="form-card-header px-6 py-4">
        <span className="text-white font-bold text-center block text-sm uppercase tracking-wide">
          🚀 ĐĂNG KÝ GIỮ VÉ ZOOM & NHẬN EBOOK
        </span>
      </div>

      <div className="p-6 flex flex-col gap-4">
        {/* Name */}
        <div>
          <label htmlFor={`${variant}-name`} className="block text-sm font-bold text-navy-900 mb-2">
            Họ và tên
          </label>
          <input
            id={`${variant}-name`}
            type="text"
            autoComplete="name"
            placeholder="Nhập họ và tên của bạn"
            className="form-input w-full px-4 py-3.5 rounded-xl text-sm"
            {...register('name')}
          />
          {errors.name && (
            <p role="alert" className="text-red-500 text-xs mt-1">{errors.name.message}</p>
          )}
        </div>

        {/* Phone */}
        <div>
          <label htmlFor={`${variant}-phone`} className="block text-sm font-bold text-navy-900 mb-2">
            Số điện thoại <span className="text-red-500">*</span>
          </label>
          <input
            id={`${variant}-phone`}
            type="tel"
            autoComplete="tel"
            inputMode="numeric"
            placeholder="Nhập số điện thoại Zalo"
            required
            className="form-input w-full px-4 py-3.5 rounded-xl text-sm"
            {...register('phone')}
          />
          {errors.phone && (
            <p role="alert" className="text-red-500 text-xs mt-1">{errors.phone.message}</p>
          )}
        </div>

        {/* Honeypot - hidden */}
        <input
          type="text"
          aria-hidden="true"
          tabIndex={-1}
          autoComplete="off"
          style={{ position: 'absolute', left: '-9999px' }}
          {...register('website')}
        />

        {/* Consent */}
        <label className="flex items-start gap-2 text-xs text-gray-600">
          <input type="checkbox" className="mt-0.5" {...register('consent')} />
          <span>Tôi đồng ý với <a href="/privacy" className="underline">chính sách bảo mật</a> và cho phép nhận thông tin qua Zalo/SMS.</span>
        </label>
        {errors.consent && (
          <p role="alert" className="text-red-500 text-xs">{errors.consent.message}</p>
        )}

        {/* Submit */}
        <button
          type="submit"
          disabled={submitState === 'submitting'}
          className="btn-cta text-white font-bold py-4 rounded-full text-sm md:text-base uppercase tracking-wide w-full mt-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {submitState === 'submitting' ? (
            <span className="flex items-center justify-center gap-2">
              <Loader2 className="animate-spin w-4 h-4" />
              ĐANG XỬ LÝ...
            </span>
          ) : (
            '🚀 GIỮ VÉ ZOOM & NHẬN EBOOK MIỄN PHÍ'
          )}
        </button>

        {/* Error */}
        {submitState === 'error' && errorMessage && (
          <div role="alert" className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg p-3">
            {errorMessage}
          </div>
        )}

        {/* Trust */}
        <div className="flex items-center justify-center gap-2 text-gray-400 text-xs text-center">
          <Shield className="w-4 h-4 text-emerald-500 flex-shrink-0" />
          <span>Cam kết bảo mật 100% thông tin cá nhân theo quy định pháp luật.</span>
        </div>

        {/* Event info */}
        <div className="flex flex-wrap gap-2 justify-center pt-2 border-t border-gray-100">
          <div className="flex items-center gap-1.5 bg-gray-50 px-3 py-1.5 rounded-lg text-xs">
            <span>📅</span>
            <span><strong>20:00 | 07/09/2026</strong></span>
          </div>
          <div className="flex items-center gap-1.5 bg-gray-50 px-3 py-1.5 rounded-lg text-xs">
            <span>💻</span>
            <span>Zoom Online</span>
          </div>
        </div>
      </div>
    </form>
  );
}

function SuccessState({ onReset }: { onReset: () => void }) {
  return (
    <div className="text-center p-8" role="status" aria-live="polite">
      <motion.div
        initial={{ scale: 0 }}
        animate={{ scale: 1 }}
        transition={{ type: 'spring', stiffness: 200 }}
        className="success-icon mx-auto"
      >
        <Check className="w-16 h-16 text-emerald-500" />
      </motion.div>
      <h3 className="text-xl font-heading font-bold text-emerald-500 mt-4">
        ĐĂNG KÝ THÀNH CÔNG!
      </h3>
      <p className="text-gray-600 text-sm mt-2">
        Bạn sẽ nhận được link Zoom & Ebook qua Zalo trong thời gian sớm nhất.
      </p>
      <p className="text-sm font-bold text-navy-900 mt-3">
        Hotline: <strong className="text-orange">0984386538</strong>
      </p>
      <button onClick={onReset} className="text-sm text-gray-500 underline mt-4">
        Đăng ký thêm vé
      </button>
    </div>
  );
}
```

### 14.3. MultiStepForm Component

```tsx
// apps/landing-renderer/components/form/MultiStepForm.tsx
'use client';

import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { z } from 'zod';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Phone, MessageCircle, ArrowLeft, Shield } from 'lucide-react';

const multistepSchema = z.object({
  channel: z.enum(['zalo', 'phone']),
  name: z.string().min(1),
  phone: z.string().regex(/^(\+84|0)\d{9,10}$/),
  consent: z.literal(true),
});

type MultistepData = z.infer<typeof multistepSchema>;

interface MultistepFormProps {
  config: LeadFormConfig;
  channels: ChannelOption[];
}

export function MultiStepForm({ config, channels }: MultistepFormProps) {
  const [step, setStep] = useState<1 | 2>(1);
  const [submitState, setSubmitState] = useState<'idle' | 'submitting' | 'success'>('idle');

  const { register, handleSubmit, formState: { errors }, setValue, watch, reset } = useForm<MultistepData>({
    resolver: zodResolver(multistepSchema),
    defaultValues: { channel: undefined },
  });

  const currentChannel = watch('channel');

  const handleChannelSelect = (channel: 'zalo' | 'phone') => {
    setValue('channel', channel, { shouldValidate: true });
    setStep(2);
  };

  const onSubmit = handleSubmit(async (data) => {
    setSubmitState('submitting');
    try {
      await submitLead({ ...data, event_id: crypto.randomUUID(), tenant_id: config.tenant_id });
      setSubmitState('success');
    } catch (err) {
      console.error(err);
    } finally {
      setSubmitState('submitting' === 'submitting' ? 'idle' : 'idle');
    }
  });

  if (submitState === 'success') {
    return (
      <div className="text-center py-6" role="status">
        <div className="success-icon mx-auto">✓</div>
        <h3 className="text-xl font-heading font-bold text-emerald-500 mt-4">
          ĐĂNG KÝ THÀNH CÔNG!
        </h3>
        <p className="text-gray-600 text-sm mt-2">Bạn sẽ nhận được link Zoom & Ebook...</p>
      </div>
    );
  }

  return (
    <div className="p-6 md:p-10">
      {/* Step indicator */}
      <div className="step-indicator mb-6 flex items-center justify-center gap-3">
        <StepIndicator step={1} active={step === 1} completed={step === 2} label="Chọn kênh" />
        <div className={`step-line w-12 h-0.5 ${step === 2 ? 'bg-orange' : 'bg-gray-200'}`} />
        <StepIndicator step={2} active={step === 2} completed={false} label="Thông tin" />
      </div>

      <form onSubmit={onSubmit} autoComplete="on">
        <AnimatePresence mode="wait">
          {step === 1 && (
            <motion.div
              key="step1"
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: 20 }}
              transition={{ duration: 0.2 }}
            >
              <h3 className="form-step-title">Bạn muốn nhận suất tham gia buổi chia sẻ và nhận Bộ 10 Ebook qua kênh nào?</h3>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-5">
                {channels.map((ch) => (
                  <button
                    key={ch.code}
                    type="button"
                    onClick={() => handleChannelSelect(ch.code)}
                    className={`channel-btn ${currentChannel === ch.code ? 'is-active' : ''}`}
                    data-channel={ch.code}
                  >
                    <span className="channel-emoji">{ch.emoji}</span>
                    <span className="channel-name">{ch.name}</span>
                    <span className="channel-sub">{ch.sub}</span>
                  </button>
                ))}
              </div>
              <input type="hidden" {...register('channel')} />
            </motion.div>
          )}

          {step === 2 && (
            <motion.div
              key="step2"
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
              transition={{ duration: 0.2 }}
              className="flex flex-col gap-4 mt-5"
            >
              <h3 className="form-step-title">Nhập thông tin để Hệ thống Gửi Vé Zoom & Ebook Miễn Phí</h3>

              <div>
                <label htmlFor="multiName" className="block text-sm font-bold text-navy-900 mb-2">Họ và tên</label>
                <input
                  id="multiName"
                  type="text"
                  autoComplete="name"
                  placeholder="Nhập họ và tên của bạn"
                  className="form-input w-full px-4 py-3.5 rounded-xl text-sm"
                  {...register('name')}
                />
                {errors.name && <p className="text-red-500 text-xs mt-1">{errors.name.message}</p>}
              </div>

              <div>
                <label htmlFor="multiPhone" className="block text-sm font-bold text-navy-900 mb-2">
                  Số điện thoại (Zalo) <span className="text-red-500">*</span>
                </label>
                <input
                  id="multiPhone"
                  type="tel"
                  autoComplete="tel"
                  inputMode="numeric"
                  placeholder="Nhập số điện thoại để nhận mã Zoom & Ebook"
                  required
                  className="form-input w-full px-4 py-3.5 rounded-xl text-sm"
                  {...register('phone')}
                />
                {errors.phone && <p className="text-red-500 text-xs mt-1">{errors.phone.message}</p>}
              </div>

              <input
                type="text"
                aria-hidden="true"
                tabIndex={-1}
                autoComplete="off"
                style={{ position: 'absolute', left: '-9999px' }}
                {...register('website')}
              />

              <label className="flex items-start gap-2 text-xs text-gray-600">
                <input type="checkbox" className="mt-0.5" {...register('consent')} />
                <span>Tôi đồng ý với <a href="/privacy" className="underline">chính sách bảo mật</a>...</span>
              </label>

              <button
                type="submit"
                disabled={submitState === 'submitting'}
                className="btn-cta text-white font-bold py-4 rounded-full text-sm md:text-base uppercase tracking-wide w-full mt-2 disabled:opacity-50"
              >
                {submitState === 'submitting' ? 'ĐANG XỬ LÝ...' : '🚀 XÁC NHẬN NHẬN TÀI LIỆU & VÉ ZOOM'}
              </button>

              <button
                type="button"
                onClick={() => setStep(1)}
                className="text-sm text-gray-500 hover:text-orange font-semibold mt-1 flex items-center justify-center gap-1"
              >
                <ArrowLeft className="w-4 h-4" />
                Quay lại chọn kênh
              </button>
            </motion.div>
          )}
        </AnimatePresence>
      </form>
    </div>
  );
}

function StepIndicator({ step, active, completed, label }: {
  step: number; active: boolean; completed: boolean; label: string;
}) {
  return (
    <div className={`step-item ${active ? 'is-active' : ''} ${completed ? 'is-completed' : ''} flex items-center gap-2`}>
      <div className={`step-circle w-8 h-8 rounded-full flex items-center justify-center font-bold ${
        active ? 'bg-orange text-white' : completed ? 'bg-emerald-500 text-white' : 'bg-gray-200 text-gray-500'
      }`}>
        {completed ? '✓' : step}
      </div>
      <div className="step-label text-sm">{label}</div>
    </div>
  );
}
```

### 14.4. Modal Dialog (Radix)

```tsx
// apps/landing-renderer/components/ui/Modal.tsx
'use client';

import * as Dialog from '@radix-ui/react-dialog';
import { X } from 'lucide-react';
import { LeadForm } from '../form/LeadForm';
import { motion, AnimatePresence } from 'framer-motion';
import { useEffect, useState } from 'react';

interface ModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  triggerLabel: string;
  triggerLocation: string;
  formConfig: LeadFormConfig;
}

export function LeadCaptureModal({ open, onOpenChange, triggerLabel, triggerLocation, formConfig }: ModalProps) {
  // Analytics tracking
  useEffect(() => {
    if (open) {
      fetch('/api/ingest/v1/events', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          event_name: 'modal_open',
          event_id: crypto.randomUUID(),
          tenant_id: formConfig.tenant_id,
          properties: { trigger: triggerLabel, location: triggerLocation },
        }),
      }).catch(console.error);
    }
  }, [open, triggerLabel, triggerLocation, formConfig.tenant_id]);

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <AnimatePresence>
        {open && (
          <Dialog.Portal forceMount>
            <Dialog.Overlay asChild>
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                className="modal-backdrop fixed inset-0 bg-black/60 z-50"
              />
            </Dialog.Overlay>

            <Dialog.Content asChild>
              <motion.div
                initial={{ opacity: 0, scale: 0.95, y: 20 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.95, y: 20 }}
                transition={{ type: 'spring', damping: 25, stiffness: 300 }}
                className="modal-dialog fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 w-[95vw] max-w-md max-h-[90vh] overflow-y-auto bg-white rounded-3xl shadow-2xl"
                aria-describedby="modal-description"
              >
                {/* Close button */}
                <Dialog.Close asChild>
                  <button
                    className="modal-close absolute top-4 right-4 z-10 p-2 rounded-full bg-white/90 hover:bg-white"
                    aria-label="Đóng"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </Dialog.Close>

                {/* Header */}
                <div className="modal-header text-center px-6 pt-6 pb-4">
                  <div className="modal-badge inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-orange/10 border border-orange/30 text-orange text-xs font-bold uppercase tracking-wider mb-4">
                    <span className="w-2 h-2 rounded-full bg-orange animate-pulse" />
                    Số lượng vé miễn phí có giới hạn!
                  </div>
                  <Dialog.Title className="text-2xl md:text-3xl font-heading font-black text-navy-900 mb-2">
                    ĐĂNG KÝ GIỮ VÉ <span className="gradient-text">BUỔI CHIA SẺ</span>
                  </Dialog.Title>
                  <Dialog.Description id="modal-description" className="text-gray-500 text-sm md:text-base">
                    Nhận ngay <strong className="text-orange">Bộ 10 Ebook Thực Chiến</strong> khi đăng ký sớm!
                  </Dialog.Description>
                </div>

                {/* Form */}
                <div className="px-6 pb-6">
                  <LeadForm config={formConfig} variant="modal" />
                </div>
              </motion.div>
            </Dialog.Content>
          </Dialog.Portal>
        )}
      </AnimatePresence>
    </Dialog.Root>
  );
}
```

---

## 15. Pixel Event Mapping (Cũ → Mới)

### 15.1. Mapping chi tiết

| Cũ (chiase_cu pixel) | Mới (RINCO hybrid) | Khi nào |
|---------------------|--------------------|---------| 
| `fbq('init', '1731191014777xxx')` | `pixel_init(tenant_pixel_id)` | Page load |
| `fbq('track', 'PageView')` | `track('page_view')` + CAPI PageView | Every page |
| (không có) | `track('scroll_depth', { percent: 25 })` | Scroll 25% |
| (không có) | `track('time_on_page', { seconds: 30 })` | 30s trên page |
| (không có) | `track('cta_click', { label })` | Click CTA |
| (không có) | `track('form_start')` | Focus first field |
| (không có) | `track('form_field_focus', { field })` | Focus từng field |
| (không có) | `track('form_field_complete', { field })` | Blur từng field valid |
| `fbq('track', 'Lead')` | `track('form_submit')` + `fbq('track', 'Lead')` + CAPI Lead | Submit form |
| (không có) | `track('form_submit_success')` | Server confirm |
| (không có) | `track('form_submit_error')` | Server error |
| (không có) | `track('video_play')` | Video play |
| (không có) | `track('video_complete')` | Video ended |
| (không có) | `track('exit_intent')` | Mouse leaves viewport |
| (không có) | CAPI `Purchase` | Deal won |
| (không có) | CAPI `Subscribe` | User subscribe |
| (không có) | CAPI `Custom_Audience_Upload` | Offline conversion |
| (không có) | CAPI `QualifiedLead` | AI score ≥ 70 |

### 15.2. Hybrid Tracking Implementation

```typescript
// apps/landing-renderer/lib/tracking/rinco-tracker.ts
import { v7 as uuidv7 } from 'uuid';

export type TrackingEvent =
  | 'page_view'
  | 'view_content'
  | 'scroll_depth'
  | 'time_on_page'
  | 'cta_click'
  | 'form_start'
  | 'form_field_focus'
  | 'form_field_complete'
  | 'form_submit'
  | 'form_submit_success'
  | 'form_submit_error'
  | 'video_play'
  | 'video_pause'
  | 'video_complete'
  | 'outbound_click'
  | 'exit_intent';

export interface TrackingPayload {
  event_id: string;
  event_name: TrackingEvent;
  tenant_id: string;
  campaign_id?: string;
  properties?: Record<string, any>;
  timestamp: number;
  trace_id: string;
  // Captured context
  session_id: string;
  anonymous_id: string;
  utm_source?: string;
  utm_medium?: string;
  utm_campaign?: string;
  fbclid?: string;
  fbp?: string;
  fbc?: string;
  referrer?: string;
  current_url: string;
}

declare global {
  interface Window {
    fbq?: (...args: any[]) => void;
    _fbq?: any;
  }
}

class RincoTracker {
  private static instance: RincoTracker;
  private sessionId: string;
  private anonymousId: string;
  private tenantId: string;
  private pixelId?: string;
  private enableCAPI: boolean = true;
  private scrollMilestones = new Set<number>();
  private timeMilestones = new Set<number>();
  private startTime: number = Date.now();
  private pendingEvents: TrackingPayload[] = [];

  private constructor() {
    this.sessionId = this.getOrCreateCookie('__rinco_session') || uuidv7();
    this.anonymousId = this.getOrCreateCookie('__rinco_anon') || uuidv7();
  }

  static getInstance(): RincoTracker {
    if (!RincoTracker.instance) {
      RincoTracker.instance = new RincoTracker();
    }
    return RincoTracker.instance;
  }

  init(config: { tenantId: string; pixelId?: string; enableCAPI?: boolean }) {
    this.tenantId = config.tenantId;
    this.pixelId = config.pixelId;
    this.enableCAPI = config.enableCAPI ?? true;

    // Setup scroll tracking
    this.setupScrollTracking();

    // Setup time-on-page tracking
    this.setupTimeTracking();

    // Setup exit intent
    this.setupExitIntent();

    // Track page_view
    this.track('page_view', { url: window.location.href });

    // Init Meta Pixel
    if (this.pixelId) {
      this.initMetaPixel();
    }
  }

  private initMetaPixel() {
    if (!this.pixelId || typeof window === 'undefined') return;
    if (window.fbq) return; // already loaded

    (function (f: any, b: any, e: any, v: any, n?: any, t?: any, s?: any) {
      if (f.fbq) return;
      n = f.fbq = function () {
        n.callMethod ? n.callMethod.apply(n, arguments) : n.queue.push(arguments);
      };
      if (!f._fbq) f._fbq = n;
      n.push = n;
      n.loaded = true;
      n.version = '2.0';
      n.queue = [];
      t = b.createElement(e);
      t.async = true;
      t.src = v;
      s = b.getElementsByTagName(e)[0];
      s.parentNode.insertBefore(t, s);
    })(window, document, 'script', 'https://connect.facebook.net/en_US/fbevents.js');

    window.fbq!('init', this.pixelId);
    window.fbq!('track', 'PageView');
  }

  track(eventName: TrackingEvent, properties: Record<string, any> = {}) {
    const payload: TrackingPayload = {
      event_id: uuidv7(),
      event_name: eventName,
      tenant_id: this.tenantId,
      session_id: this.sessionId,
      anonymous_id: this.anonymousId,
      timestamp: Date.now(),
      trace_id: uuidv7(),
      current_url: window.location.href,
      referrer: document.referrer,
      utm_source: this.getUrlParam('utm_source'),
      utm_medium: this.getUrlParam('utm_medium'),
      utm_campaign: this.getUrlParam('utm_campaign'),
      fbclid: this.getUrlParam('fbclid'),
      fbc: this.getCookie('_fbc'),
      fbp: this.getCookie('_fbp'),
      properties,
    };

    // 1. Send to server (CAPI + analytics)
    this.sendToServer(payload);

    // 2. Fire Meta Pixel event (matching eventID)
    this.firePixelEvent(eventName, payload.event_id, properties);
  }

  private firePixelEvent(eventName: TrackingEvent, eventID: string, properties: any) {
    if (!window.fbq) return;
    const fbEvent = this.mapToFBEvent(eventName);
    if (fbEvent) {
      window.fbq('track', fbEvent, this.buildPixelPayload(eventName, properties), { eventID });
    }
  }

  private mapToFBEvent(event: TrackingEvent): string | null {
    const mapping: Partial<Record<TrackingEvent, string>> = {
      page_view: 'PageView',
      view_content: 'ViewContent',
      form_submit: 'Lead',
      form_submit_success: 'Lead',
      video_play: 'VideoPlay',
      video_complete: 'VideoComplete',
    };
    return mapping[event] || null;
  }

  private buildPixelPayload(event: TrackingEvent, properties: any): Record<string, any> {
    switch (event) {
      case 'page_view':
        return {};
      case 'view_content':
        return { content_name: properties.title || '' };
      case 'form_submit':
        return {
          content_name: 'APEX Fintech Landing',
          value: 0,
          currency: 'VND',
        };
      case 'form_submit_success':
        return {
          content_name: 'APEX Fintech Landing',
          value: properties.deal_value || 0,
          currency: 'VND',
          content_category: properties.lead_status,
        };
      default:
        return properties;
    }
  }

  private async sendToServer(payload: TrackingPayload) {
    try {
      // Use sendBeacon nếu unload, fetch otherwise
      if (document.visibilityState === 'hidden') {
        navigator.sendBeacon('/api/ingest/v1/events', JSON.stringify(payload));
      } else {
        await fetch('/api/ingest/v1/events', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
          keepalive: true,
        });
      }
    } catch (err) {
      // Queue for retry
      this.pendingEvents.push(payload);
      if (this.pendingEvents.length > 100) {
        this.pendingEvents.shift(); // Drop oldest
      }
    }
  }

  private setupScrollTracking() {
    const milestones = [25, 50, 75, 100];
    const onScroll = () => {
      const docHeight = document.documentElement.scrollHeight - window.innerHeight;
      const scrolled = (window.scrollY / docHeight) * 100;
      for (const m of milestones) {
        if (scrolled >= m && !this.scrollMilestones.has(m)) {
          this.scrollMilestones.add(m);
          this.track('scroll_depth', { percent: m });
        }
      }
    };
    window.addEventListener('scroll', onScroll, { passive: true });
  }

  private setupTimeTracking() {
    const milestones = [30, 60, 120, 300];
    for (const m of milestones) {
      setTimeout(() => {
        if (!this.timeMilestones.has(m)) {
          this.timeMilestones.add(m);
          this.track('time_on_page', { seconds: m });
        }
      }, m * 1000);
    }
  }

  private setupExitIntent() {
    document.addEventListener('mouseout', (e: MouseEvent) => {
      if (e.clientY <= 0) {
        this.track('exit_intent', { url: window.location.href });
      }
    });
  }

  // Utility helpers
  private getCookie(name: string): string | undefined {
    if (typeof document === 'undefined') return;
    const match = document.cookie.match(new RegExp('(^|; )' + name + '=([^;]*)'));
    return match ? decodeURIComponent(match[2]) : undefined;
  }

  private getOrCreateCookie(name: string): string | undefined {
    const existing = this.getCookie(name);
    if (existing) return existing;
    const value = uuidv7();
    document.cookie = `${name}=${value}; max-age=${30 * 24 * 60 * 60}; path=/; SameSite=Lax`;
    return value;
  }

  private getUrlParam(key: string): string | undefined {
    if (typeof window === 'undefined') return;
    return new URLSearchParams(window.location.search).get(key) || undefined;
  }
}

export const tracker = RincoTracker.getInstance();
export const track = (event: TrackingEvent, properties?: Record<string, any>) =>
  tracker.track(event, properties);
```

---

## 16. Tracking Implementation chi tiết

### 16.1. useTracker Hook (React)

```typescript
// apps/landing-renderer/lib/tracking/useTracker.ts
'use client';

import { useEffect, useRef } from 'react';
import { tracker, TrackingEvent } from './rinco-tracker';
import { v7 as uuidv7 } from 'uuid';

export function useTracker() {
  const eventIdRef = useRef<string>('');

  // Generate stable event_id per form session
  useEffect(() => {
    eventIdRef.current = uuidv7();
  }, []);

  return {
    track: (event: TrackingEvent, properties?: Record<string, any>) =>
      tracker.track(event, properties),
    eventId: eventIdRef.current,
    tracker,
  };
}
```

### 16.2. Server-side Event Ingestion (Go)

```go
// services/landing-ingest/internal/handler/event.go
package handler

import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "github.com/google/uuid"
    "github.com/itdoanh/rinco/landing-ingest/internal/auth"
    "github.com/itdoanh/rinco/landing-ingest/internal/scylla"
    "github.com/itdoanh/rinco/landing-ingest/internal/nats"
)

type EventHandler struct {
    scyllaRepo *scylla.EventRepo
    publisher  *nats.Publisher
    tenantCtx  *auth.TenantContext
}

type IngestEventRequest struct {
    EventID    string                 `json:"event_id"`
    EventName  string                 `json:"event_name"`
    TenantID   string                 `json:"tenant_id"`
    SessionID  string                 `json:"session_id"`
    AnonID     string                 `json:"anonymous_id"`
    Timestamp  int64                  `json:"timestamp"`
    Properties map[string]interface{} `json:"properties"`
    URL        string                 `json:"current_url"`
    Referrer   string                 `json:"referrer"`
    FBCLID     string                 `json:"fbclid"`
    FBC        string                 `json:"fbc"`
    FBP        string                 `json:"fbp"`
    UTMSource  string                 `json:"utm_source"`
    UTMMedium  string                 `json:"utm_medium"`
    UTMCampaign string                `json:"utm_campaign"`
}

func (h *EventHandler) Handle(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    traceID := r.Header.Get("X-Trace-ID")
    if traceID == "" {
        traceID = uuid.NewV7().String()
    }

    // Limit body size 8KB (events should be small)
    r.Body = http.MaxBytesReader(w, r.Body, 8*1024)

    var req IngestEventRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }

    // Validate
    if req.EventID == "" || req.EventName == "" {
        http.Error(w, "missing required fields", http.StatusBadRequest)
        return
    }

    // Rate limit per IP + tenant
    if !h.rateLimit(ctx, r.RemoteAddr, req.TenantID) {
        w.Header().Set("Retry-After", "60")
        http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
        return
    }

    // 1. Write to ScyllaDB (async, fire-and-forget)
    go func() {
        bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := h.scyllaRepo.InsertEvent(bgCtx, scylla.EventRow{
            TenantID:  req.TenantID,
            EventID:   req.EventID,
            EventName: req.EventName,
            CreatedAt: time.Unix(req.Timestamp/1000, 0),
            Payload:   req,
            SessionID: req.SessionID,
        }); err != nil {
            slog.Error("scylla insert failed", "error", err, "event_id", req.EventID)
        }
    }()

    // 2. Publish NATS for real-time processing
    natsEvent := nats.AnalyticsEvent{
        EventID:   req.EventID,
        EventName: req.EventName,
        TenantID:  req.TenantID,
        Properties: req.Properties,
        Timestamp: req.Timestamp,
        TraceID:   traceID,
    }
    if err := h.publisher.PublishAnalyticsEvent(ctx, natsEvent); err != nil {
        slog.Warn("nats publish failed, will retry", "error", err)
        // Queue locally
        go h.retryPublish(natsEvent)
    }

    // 3. Response
    w.Header().Set("X-Trace-ID", traceID)
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{"status": "accepted", "event_id": req.EventID})
}

func (h *EventHandler) rateLimit(ctx context.Context, ip, tenantID string) bool {
    key := fmt.Sprintf("ratelimit:%s:%s:%s", tenantID, ip, "event")
    allowed, err := h.valkey.Eval(ctx, `
        local current = redis.call('INCR', KEYS[1])
        if current == 1 then
            redis.call('EXPIRE', KEYS[1], ARGV[2])
        end
        return current <= tonumber(ARGV[1])
    `, []string{key}, 100, 60).Result()
    return err == nil && allowed.(int64) <= 100
}
```

---

## 17. Wasm Attestation Module

### 17.1. Wasm Module (Rust)

```rust
// crates/bot-detection-wasm/src/lib.rs
use wasm_bindgen::prelude::*;
use serde::{Serialize, Deserialize};
use web_sys::{window, CanvasRenderingContext2d, HtmlCanvasElement};

#[wasm_bindgen]
pub struct AttestationResult {
    pub score: f64,
    pub reasons: Vec<JsValue>,
    pub token: String,
}

#[derive(Serialize, Deserialize)]
pub struct AttestationClaims {
    pub visitor_id: String,
    pub is_human: bool,
    pub score: f64,
    pub issued_at: i64,
    pub expires_at: i64,
}

#[wasm_bindgen]
pub fn run_attestation() -> Result<JsValue, JsValue> {
    let mut score = 0.0;
    let mut reasons = Vec::new();

    // 1. Check WebDriver flag
    if is_webdriver() {
        return Ok(serialize_result(0.0, vec!["webdriver_detected".into()], ""));
    }

    // 2. Check user agent cho bot pattern
    let ua = get_user_agent();
    if ua.contains("HeadlessChrome") || ua.contains("PhantomJS") {
        return Ok(serialize_result(0.0, vec!["headless_browser".into()], ""));
    }

    // 3. Check WebGL renderer
    let renderer = get_webgl_renderer();
    if renderer.contains("swiftshader") {
        return Ok(serialize_result(0.0, vec!["swiftshader_webgl".into()], ""));
    }
    score += 0.2;

    // 4. Check Canvas fingerprint stability
    let canvas_hash = canvas_fingerprint();
    if canvas_hash.is_empty() {
        score -= 0.3;
        reasons.push("canvas_empty".into());
    } else {
        score += 0.2;
    }

    // 5. Mouse movement analysis (linear vs real)
    // Requires DOM events; simplified here
    score += 0.3;

    // 6. Touch events check (mobile)
    #[cfg(feature = "mobile")]
    {
        if !has_touch_support() {
            score -= 0.2;
            reasons.push("no_touch".into());
        }
    }

    let is_human = score >= 0.6;
    let claims = AttestationClaims {
        visitor_id: generate_visitor_id(&canvas_hash, &renderer),
        is_human,
        score,
        issued_at: js_sys::Date::now() as i64,
        expires_at: (js_sys::Date::now() + 3600_000.0) as i64, // 1h
    };

    // HMAC sign (server secret embedded encrypted)
    let token = sign_token(&claims);

    Ok(serialize_result(score, reasons, &token))
}

fn is_webdriver() -> bool {
    window()
        .and_then(|w| w.navigator().webdriver().ok())
        .unwrap_or(false)
}

fn get_user_agent() -> String {
    window()
        .and_then(|w| w.navigator().user_agent().ok())
        .unwrap_or_default()
}

fn get_webgl_renderer() -> String {
    // Tạo canvas, get context, lấy UNMASKED_RENDERER_WEBGL
    let document = web_sys::window().unwrap().document().unwrap();
    let canvas: HtmlCanvasElement = document
        .create_element("canvas")
        .unwrap()
        .dyn_into()
        .unwrap();
    let ctx = canvas
        .get_context("webgl")
        .unwrap()
        .unwrap()
        .dyn_into::<WebGlRenderingContext>()
        .unwrap();

    let ext = ctx.get_extension("WEBGL_debug_renderer_info").unwrap().unwrap();
    let renderer = ctx.get_parameter(ext.unmasked_renderer_webgl()).unwrap();
    renderer.as_string().unwrap_or_default()
}

fn canvas_fingerprint() -> String {
    use sha2::{Digest, Sha256};
    let document = web_sys::window().unwrap().document().unwrap();
    let canvas = document.create_element("canvas").unwrap().dyn_into::<HtmlCanvasElement>().unwrap();
    canvas.set_width(280);
    canvas.set_height(60);
    let ctx: CanvasRenderingContext2d = canvas.get_context("2d").unwrap().unwrap().dyn_into().unwrap();
    ctx.set_font("14px Arial");
    ctx.fill_text("RINCO Bot Detection", 2, 15);
    ctx.fill_rect(10, 20, 80, 40);

    let data_url = canvas.to_data_url().unwrap_or_default();
    let mut hasher = Sha256::new();
    hasher.update(data_url.as_bytes());
    format!("{:x}", hasher.finalize())
}

fn generate_visitor_id(canvas_hash: &str, renderer: &str) -> String {
    use sha2::{Digest, Sha256};
    let mut hasher = Sha256::new();
    hasher.update(canvas_hash.as_bytes());
    hasher.update(renderer.as_bytes());
    hasher.update(get_user_agent().as_bytes());
    format!("{:x}", hasher.finalize())[..32].to_string()
}

fn sign_token(claims: &AttestationClaims) -> String {
    // In production: gọi server để sign với secret key
    // Ở đây dùng placeholder
    use base64::{Engine, engine::general_purpose::URL_SAFE_NO_PAD};
    let json = serde_json::to_string(claims).unwrap();
    URL_SAFE_NO_PAD.encode(json.as_bytes())
}

fn serialize_result(score: f64, reasons: Vec<String>, token: &str) -> JsValue {
    let result = AttestationResult {
        score,
        reasons: reasons.into_iter().map(JsValue::from).collect(),
        token: token.to_string(),
    };
    serde_wasm_bindgen::to_value(&result).unwrap()
}

fn has_touch_support() -> bool {
    window()
        .and_then(|w| {
            let navigator = w.navigator();
            js_sys::Reflect::get(&navigator, &"maxTouchPoints".into())
                .ok()
                .and_then(|v| v.as_f64())
                .map(|n| n > 0.0)
                .or_else(|| {
                    js_sys::Reflect::get(&w, &"ontouchstart".into())
                        .ok()
                        .map(|_| true)
                })
        })
        .unwrap_or(false)
}
```

### 17.2. Client Integration

```typescript
// apps/landing-renderer/lib/bot/attestation.ts
import init, { run_attestation } from '@rinco/bot-detection-wasm';

let initialized = false;

export async function getAttestation(): Promise<{
  score: number;
  isHuman: boolean;
  token: string;
  visitorId: string;
}> {
  if (!initialized) {
    await init();
    initialized = true;
  }
  const result = await run_attestation();
  return {
    score: result.score,
    isHuman: result.score >= 0.6,
    token: result.token,
    visitorId: extractVisitorId(result.token),
  };
}

function extractVisitorId(token: string): string {
  try {
    const json = atob(token.replace(/-/g, '+').replace(/_/g, '/'));
    const claims = JSON.parse(json);
    return claims.visitor_id;
  } catch {
    return '';
  }
}
```

### 17.3. Server-side Verification

```go
// services/landing-ingest/internal/middleware/attestation.go
package middleware

import (
    "net/http"
    "strings"

    "github.com/itdoanh/rinco/landing-ingest/internal/attestation"
)

type AttestationMiddleware struct {
    verifier *attestation.Verifier
}

func (m *AttestationMiddleware) Wrap(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("X-RINCO-Attestation")
        if token == "" {
            // Skip for backwards compat, nhưng flag risky
            // Có thể return 403 nếu strict mode
            next.ServeHTTP(w, r)
            return
        }

        claims, err := m.verifier.Verify(token)
        if err != nil {
            http.Error(w, "invalid attestation", http.StatusForbidden)
            return
        }

        if !claims.IsHuman {
            // Log bot attempt, mark as spam
            slog.Warn("bot submit blocked", "visitor_id", claims.VisitorID, "score", claims.Score)
            http.Error(w, "bot detected", http.StatusForbidden)
            return
        }

        // Attach to context
        ctx := context.WithValue(r.Context(), "visitor_id", claims.VisitorID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## 18. Argon2 PoW Challenge Protocol

### 18.1. Protocol Flow

```
1. Client sends form submit
        │
        ▼
2. Server check: rate limit > threshold OR suspicious IP?
        │
   ┌────┴────┐
   │ NO      │ YES
   │         ▼
   │    3. Server returns 202 + X-PoW-Challenge header:
   │       {
   │         "challenge_id": "uuid",
   │         "salt": "base64",
   │         "ts": 1725696000,
   │         "difficulty": 4  // bits phải match
   │       }
   │         │
   │         ▼
   │    4. Client compute Argon2id(salt + ts + nonce) cho tới khi hash[0:difficulty/8] == 0
   │       Thường mất ~20-50ms trên browser
   │         │
   │         ▼
   │    5. Client retries POST with:
   │       X-PoW-Challenge-Id: uuid
   │       X-PoW-Timestamp: 1725696000
   │       X-PoW-Nonce: <found nonce>
   │         │
   │         ▼
   │    6. Server verify + accept
   │
   ▼
7. Direct submit accepted
```

### 18.2. Server: Challenge Generator

```go
// services/landing-ingest/internal/pow/challenge.go
package pow

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/itdoanh/rinco/landing-ingest/internal/valkey"
)

type Challenge struct {
    ChallengeID string
    Salt        string
    Timestamp   int64
    Difficulty  int // bits
}

type ChallengeStore struct {
    valkey *valkey.Client
    ttl    time.Duration
}

func NewChallengeStore(v *valkey.Client) *ChallengeStore {
    return &ChallengeStore{
        valkey: v,
        ttl:    60 * time.Second,
    }
}

func (s *ChallengeStore) Issue(ctx context.Context, tenantID string, baseDifficulty int) (*Challenge, error) {
    // Adaptive difficulty dựa trên traffic
    difficulty := s.adaptDifficulty(tenantID, baseDifficulty)

    saltBytes := make([]byte, 16)
    rand.Read(saltBytes)

    c := &Challenge{
        ChallengeID: uuid.NewV7().String(),
        Salt:        base64.URLEncoding.EncodeToString(saltBytes),
        Timestamp:   time.Now().Unix(),
        Difficulty:  difficulty,
    }

    // Store in Valkey với TTL 60s
    key := fmt.Sprintf("pow:challenge:%s", c.ChallengeID)
    if err := s.valkey.Set(ctx, key, fmt.Sprintf("%s:%d:%d", c.Salt, c.Timestamp, c.Difficulty), s.ttl).Err(); err != nil {
        return nil, fmt.Errorf("valkey set: %w", err)
    }

    return c, nil
}

func (s *ChallengeStore) adaptDifficulty(tenantID string, base int) int {
    // Nếu traffic tăng đột biến → tăng difficulty
    // TODO: integrate với rate limit metrics
    return base // default 4 bits
}

func (s *ChallengeStore) Verify(ctx context.Context, challengeID, nonce string, ts int64) error {
    key := fmt.Sprintf("pow:challenge:%s", challengeID)
    raw, err := s.valkey.Get(ctx, key).Result()
    if err != nil {
        return fmt.Errorf("challenge not found or expired")
    }

    // Parse salt:ts:difficulty
    var salt string
    var origTS, difficulty int64
    fmt.Sscanf(raw, "%s:%d:%d", &salt, &origTS, &difficulty)

    // Check timestamp window (±2 min)
    now := time.Now().Unix()
    if abs(now-ts) > 120 || abs(now-origTS) > 120 {
        return fmt.Errorf("challenge expired")
    }

    // Verify Argon2 với nonce
    expectedHash := computeArgon2id(salt, ts, nonce)
    if !checkDifficulty(expectedHash, int(difficulty)) {
        return fmt.Errorf("invalid solution")
    }

    // Consume challenge (single use)
    s.valkey.Del(ctx, key)
    return nil
}

func abs(n int64) int64 {
    if n < 0 {
        return -n
    }
    return n
}

func checkDifficulty(hash []byte, difficultyBits int) bool {
    fullBytes := difficultyBits / 8
    remainder := difficultyBits % 8

    for i := 0; i < fullBytes; i++ {
        if hash[i] != 0 {
            return false
        }
    }
    if remainder > 0 {
        mask := byte(0xFF) << (8 - remainder)
        if hash[fullBytes]&mask != 0 {
            return false
        }
    }
    return true
}
```

### 18.3. Client: PoW Solver

```typescript
// apps/landing-renderer/lib/pow/solver.ts
import argon2 from 'argon2-browser';

export async function solvePow(challenge: {
  challenge_id: string;
  salt: string;
  ts: number;
  difficulty: number;
}): Promise<{ nonce: string; hash: string; ts: number; challengeId: string }> {
  const { challenge_id, salt, ts, difficulty } = challenge;
  const startTime = Date.now();

  let nonce = 0;
  const maxAttempts = 10_000_000;
  const deadline = startTime + 30_000; // max 30s

  while (nonce < maxAttempts && Date.now() < deadline) {
    const hash = await argon2.hash({
      pass: String(nonce),
      salt: `${salt}|${ts}`,
      time: 2,
      mem: 1024,
      parallelism: 1,
      hashLen: 32,
      type: argon2.ArgonType.Argon2id,
    });

    if (checkDifficulty(hash.encoded, difficulty)) {
      return {
        nonce: String(nonce),
        hash: hash.encoded,
        ts,
        challengeId: challenge_id,
      };
    }

    nonce++;
  }

  throw new Error('PoW: failed to find solution within timeout');
}

function checkDifficulty(hash: string, bits: number): boolean {
  const fullBytes = Math.floor(bits / 8);
  const remainder = bits % 8;

  // Decode base64 → bytes
  const bytes = base64ToBytes(hash);

  for (let i = 0; i < fullBytes; i++) {
    if (bytes[i] !== 0) return false;
  }
  if (remainder > 0) {
    const mask = (0xff << (8 - remainder)) & 0xff;
    if ((bytes[fullBytes] & mask) !== 0) return false;
  }
  return true;
}

function base64ToBytes(b64: string): Uint8Array {
  const bin = atob(b64.replace(/-/g, '+').replace(/_/g, '/'));
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

// React hook
export function usePowChallenge() {
  const solvePowWithChallenge = async (challengeId: string) => {
    const res = await fetch(`/api/ingest/v1/pow-challenge?challenge_id=${challengeId}`);
    const challenge = await res.json();
    return solvePow(challenge);
  };

  return { solvePow: solvePowWithChallenge };
}
```

---

## 19. CAPI Worker chi tiết

### 19.1. Full Implementation

```go
// services/meta-capi/internal/worker/worker.go
package worker

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/nats-io/nats.go"
    "github.com/sony/gobreaker"
    "slog"

    "github.com/itdoanh/rinco/meta-capi/internal/clickhouse"
    "github.com/itdoanh/rinco/meta-capi/internal/meta"
    "github.com/itdoanh/rinco/meta-capi/internal/repository"
    "github.com/itdoanh/rinco/meta-capi/internal/signature"
    "github.com/itdoanh/rinco/meta-capi/internal/valkey"
)

type Worker struct {
    nc        *nats.Conn
    meta      *meta.Client
    secretKey []byte
    cb        *gobreaker.CircuitBreaker
    repo      *repository.CAPIRepo
    valkey    *valkey.Client
    ch        *clickhouse.Client
}

func NewWorker(nc *nats.Conn, metaClient *meta.Client, secretKey []byte) *Worker {
    settings := gobreaker.Settings{
        Name: "facebook-capi",
        Timeout: 60 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures > 5 ||
                (counts.Requests >= 10 && float64(counts.TotalFailures)/float64(counts.Requests) > 0.5)
        },
        OnStateChange: func(name string, from, to gobreaker.State) {
            slog.Warn("circuit breaker state changed",
                "name", name, "from", from.String(), "to", to.String())
        },
    }

    return &Worker{
        nc:        nc,
        meta:      metaClient,
        secretKey: secretKey,
        cb:        gobreaker.NewCircuitBreaker(settings),
    }
}

func (w *Worker) Start(ctx context.Context) error {
    sub, err := w.nc.Subscribe("lead.events", w.handleLeadEvent)
    if err != nil {
        return fmt.Errorf("nats subscribe: %w", err)
    }
    defer sub.Unsubscribe()

    sub2, err := w.nc.Subscribe("crm.conversion", w.handleConversion)
    if err != nil {
        return fmt.Errorf("nats subscribe crm.conversion: %w", err)
    }
    defer sub2.Unsubscribe()

    <-ctx.Done()
    return nil
}

type LeadEventMsg struct {
    EventID    string                 `json:"event_id"`
    LeadID     string                 `json:"lead_id"`
    TenantID   string                 `json:"tenant_id"`
    EventName  string                 `json:"event_name"`
    EventTime  int64                  `json:"event_time"`
    FBCLID     string                 `json:"fbclid"`
    FBC        string                 `json:"fbc"`
    FBP        string                 `json:"fbp"`
    UserData   signature.UserDataRaw  `json:"user_data"`
    CustomData map[string]interface{} `json:"custom_data"`
    ActionSource string               `json:"action_source"`
    EventSourceURL string             `json:"event_source_url"`
    Signature  string                 `json:"signature"`
    Timestamp  string                 `json:"timestamp"`
}

func (w *Worker) handleLeadEvent(msg *nats.Msg) {
    var event LeadEventMsg
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        slog.Error("unmarshal failed", "error", err)
        return
    }

    ctx := context.Background()
    traceID := uuid.NewV7().String()
    slog.Info("processing lead event", "event_id", event.EventID, "lead_id", event.LeadID, "trace_id", traceID)

    // 1. Verify HMAC
    payloadBytes, _ := json.Marshal(event.UserData)
    if !signature.VerifyLeadSignature(event.LeadID, event.FBCLID, event.Timestamp, payloadBytes, event.Signature, string(w.secretKey)) {
        slog.Warn("HMAC verification failed", "lead_id", event.LeadID, "trace_id", traceID)
        w.recordCAPILog(ctx, event.TenantID, event.EventID, 401, 0, "", "invalid_signature", traceID)
        return
    }

    // 2. Load tenant pixel config
    pixel, err := w.repo.GetTenantPixel(ctx, event.TenantID)
    if err != nil {
        slog.Error("load pixel config", "error", err)
        msg.Nak() // retry
        return
    }

    // 3. Normalize user data theo Meta standard
    normalized := w.normalizeUserData(event.UserData)

    // 4. Build CAPI event
    capiEvent := meta.CAPIEvent{
        EventName:      event.EventName,
        EventTime:      event.EventTime,
        EventID:        event.EventID,
        ActionSource:   event.ActionSource,
        EventSourceURL: event.EventSourceURL,
        UserData:       normalized,
        CustomData:     event.CustomData,
    }

    // 5. Send with circuit breaker
    err = w.cb.Execute(func() error {
        return w.meta.SendEvent(ctx, pixel.PixelID, pixel.CAPIToken, capiEvent)
    })

    if err != nil {
        slog.Error("CAPI send failed", "error", err, "event_id", event.EventID)
        // Send to retry queue
        w.enqueueRetry(event)
        w.recordCAPILog(ctx, event.TenantID, event.EventID, 500, 0, "", err.Error(), traceID)
        msg.Ack()
        return
    }

    // 6. Log success
    response := w.meta.LastResponse()
    slog.Info("CAPI sent successfully",
        "event_id", event.EventID,
        "events_received", response.EventsReceived,
        "fbtrace_id", response.FBTraceID,
        "trace_id", traceID,
    )

    w.recordCAPILog(ctx, event.TenantID, event.EventID, 200, response.EventsReceived, response.FBTraceID, "", traceID)
    w.updateClickHouseMetric(ctx, event.TenantID, "success")
    msg.Ack()
}

func (w *Worker) normalizeUserData(raw signature.UserDataRaw) meta.UserData {
    return meta.UserData{
        Email:         []string{signature.NormalizeEmail(raw.Email)},
        Phone:         []string{signature.NormalizePhone(raw.Phone)},
        FirstName:     []string{signature.NormalizeName(raw.FirstName)},
        LastName:      []string{signature.NormalizeName(raw.LastName)},
        City:          []string{signature.NormalizeCity(raw.City)},
        State:         []string{signature.NormalizeString(raw.State)},
        Country:       []string{raw.Country},
        ExternalID:    []string{signature.SHA256(raw.ExternalID)},
        ClientIP:      raw.ClientIP,
        ClientUA:      raw.ClientUA,
        FBC:           raw.FBC,
        FBP:           raw.FBP,
        SubscriptionID: raw.SubscriptionID,
    }
}

func (w *Worker) handleConversion(msg *nats.Msg) {
    var event struct {
        EventID    string                 `json:"event_id"`
        LeadID     string                 `json:"lead_id"`
        TenantID   string                 `json:"tenant_id"`
        EventName  string                 `json:"event_name"`
        DealValue  float64                `json:"deal_value"`
        Currency   string                 `json:"currency"`
        UserData   signature.UserDataRaw  `json:"user_data"`
        EventTime  int64                  `json:"event_time"`
    }
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        slog.Error("unmarshal conversion", "error", err)
        return
    }

    // Build conversion event
    pixel, _ := w.repo.GetTenantPixel(context.Background(), event.TenantID)
    capiEvent := meta.CAPIEvent{
        EventName: event.EventName,
        EventTime: event.EventTime,
        EventID:   event.EventID,
        ActionSource: "website",
        UserData: w.normalizeUserData(event.UserData),
        CustomData: map[string]interface{}{
            "value":    event.DealValue,
            "currency": event.Currency,
        },
    }

    if err := w.cb.Execute(func() error {
        return w.meta.SendEvent(context.Background(), pixel.PixelID, pixel.CAPIToken, capiEvent)
    }); err != nil {
        slog.Error("conversion CAPI failed", "error", err)
        msg.Nak()
        return
    }
    msg.Ack()
}

func (w *Worker) enqueueRetry(event LeadEventMsg) {
    // Đẩy vào retry queue với delay tăng dần
    delay := calculateBackoff(w.retryCount(event.EventID))
    // ... publish NATS delayed message
}

func (w *Worker) recordCAPILog(ctx context.Context, tenantID, eventID string, statusCode, eventsReceived int, fbtraceID, errMsg, traceID string) {
    w.repo.InsertCAPILog(ctx, repository.CAPILog{
        TenantID:       tenantID,
        EventID:        eventID,
        SentAt:         time.Now(),
        ResponseCode:   statusCode,
        EventsReceived: eventsReceived,
        FBTraceID:      fbtraceID,
        ErrorMessage:   errMsg,
        TraceID:        traceID,
    })
}

func (w *Worker) updateClickHouseMetric(ctx context.Context, tenantID, status string) {
    w.ch.Insert(ctx, clickhouse.Metric{
        TenantID:  tenantID,
        EventDate: time.Now(),
        EventName: "capi_send",
        Status:    status,
    })
}
```

---

## 20. Webhook Feedback Loop chi tiết

### 20.1. CRM → Meta Conversion Webhook

```go
// services/crm-core/internal/conversion/event_publisher.go
package conversion

import (
    "context"
    "encoding/json"
    "time"

    "github.com/google/uuid"
    "github.com/itdoanh/rinco/crm-core/internal/nats"
)

type Publisher struct {
    nc *nats.Conn
}

type ConversionEvent struct {
    EventID    string                 `json:"event_id"`
    LeadID     string                 `json:"lead_id"`
    TenantID   string                 `json:"tenant_id"`
    EventName  string                 `json:"event_name"`
    DealValue  float64                `json:"deal_value"`
    Currency   string                 `json:"currency"`
    UserData   map[string]interface{} `json:"user_data"`
    EventTime  int64                  `json:"event_time"`
    AI_Score   float64                `json:"ai_score"`
}

func (p *Publisher) PublishDealWon(ctx context.Context, lead Lead, deal Deal) error {
    event := ConversionEvent{
        EventID:   uuid.NewV7().String(),
        LeadID:    lead.ID.String(),
        TenantID:  lead.TenantID,
        EventName: "Purchase",
        DealValue: deal.Value,
        Currency:  deal.Currency,
        UserData: map[string]interface{}{
            "email":        lead.Email,
            "phone":        lead.Phone,
            "first_name":   lead.FirstName,
            "last_name":    lead.LastName,
            "city":         lead.City,
            "country":      lead.Country,
            "external_id":  lead.ID.String(),
            "client_ip_address": lead.LastIP,
            "client_user_agent": lead.LastUA,
            "fbp":          lead.FBP,
            "fbc":          lead.FBC,
        },
        EventTime: time.Now().Unix(),
        AI_Score:  lead.Score,
    }

    data, _ := json.Marshal(event)
    return p.nc.Publish("crm.conversion", data)
}

func (p *Publisher) PublishHighScore(ctx context.Context, lead Lead) error {
    if lead.Score < 70 {
        return nil
    }
    event := ConversionEvent{
        EventID:   uuid.NewV7().String(),
        LeadID:    lead.ID.String(),
        TenantID:  lead.TenantID,
        EventName: "QualifiedLead",
        DealValue: 0,
        Currency:  "VND",
        UserData:  buildUserData(lead),
        EventTime: time.Now().Unix(),
        AI_Score:  lead.Score,
    }
    data, _ := json.Marshal(event)
    return p.nc.Publish("crm.conversion", data)
}

func (p *Publisher) PublishSchedule(ctx context.Context, lead Lead, schedule Schedule) error {
    event := ConversionEvent{
        EventID:   uuid.NewV7().String(),
        LeadID:    lead.ID.String(),
        TenantID:  lead.TenantID,
        EventName: "Schedule",
        UserData:  buildUserData(lead),
        EventTime: time.Now().Unix(),
    }
    data, _ := json.Marshal(event)
    return p.nc.Publish("crm.conversion", data)
}
```

### 20.2. Conversion Event Mapping

| CRM Event | Meta Event | Custom Data | Ghi chú |
|-----------|-----------|-------------|---------|
| Lead Created (AI score ≥ 70) | `QualifiedLead` | `predicted_ltv` | Optimizes for high-quality |
| Lead Created (AI score < 70) | `Lead` | (none) | Standard tracking |
| Appointment scheduled | `Schedule` | (none) | |
| Deal Won | `Purchase` | `value`, `currency` | Optimizes for revenue |
| Subscription activated | `Subscribe` | `value`, `currency`, `predicted_ltv` | |
| High-value lead (score ≥ 90) | `Custom_High_Value` | `value: predicted_ltv` | |
| SQL Qualified (manual) | `Custom_SQL_Qualified` | (none) | |

---

## 21. A/B Testing Framework

### 21.1. Experiment Schema

```sql
CREATE TABLE ab_experiments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  name TEXT NOT NULL,
  hypothesis TEXT,
  goal TEXT NOT NULL,           -- 'form_submit', 'cta_click', 'time_on_page'
  primary_metric TEXT NOT NULL, -- 'conversion_rate', 'click_rate'
  status TEXT DEFAULT 'DRAFT',  -- 'DRAFT', 'RUNNING', 'STOPPED', 'COMPLETED'
  traffic_allocation INT DEFAULT 100, -- % traffic in experiment
  started_at TIMESTAMPTZ,
  ended_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE ab_variants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  experiment_id UUID REFERENCES ab_experiments(id),
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  weight INT NOT NULL,          -- sum phải = 100
  content JSONB NOT NULL,       -- variant landing page content
  is_control BOOLEAN DEFAULT false,
  UNIQUE(experiment_id, code)
);

CREATE TABLE ab_assignments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  experiment_id UUID REFERENCES ab_experiments(id),
  variant_id UUID REFERENCES ab_variants(id),
  visitor_id TEXT NOT NULL,
  assigned_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(experiment_id, visitor_id)
);

CREATE TABLE ab_conversions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  assignment_id UUID REFERENCES ab_assignments(id),
  event_name TEXT NOT NULL,
  value NUMERIC,
  converted_at TIMESTAMPTZ DEFAULT now()
);
```

### 21.2. Variant Assignment Logic

```go
// apps/landing-renderer/lib/ab/assign.ts
import { v7 as uuidv7 } from 'uuid';

export interface ABVariant {
  id: string;
  code: string;
  name: string;
  weight: number;
  content: Record<string, any>;
}

export interface ABExperiment {
  id: string;
  name: string;
  variants: ABVariant[];
  status: 'DRAFT' | 'RUNNING' | 'STOPPED' | 'COMPLETED';
}

const STORAGE_KEY = '__rinco_ab_assignments';

export function getAssignment(experiment: ABExperiment, visitorId: string): ABVariant {
  // 1. Check existing assignment (sticky)
  const cached = getCachedAssignment(experiment.id);
  if (cached) {
    const variant = experiment.variants.find(v => v.id === cached.variantId);
    if (variant) return variant;
  }

  // 2. Deterministic hash-based assignment (same visitor → same variant)
  const bucket = hashToBucket(visitorId + experiment.id, 100);
  let cumulative = 0;
  for (const variant of experiment.variants) {
    cumulative += variant.weight;
    if (bucket < cumulative) {
      cacheAssignment(experiment.id, variant.id, visitorId);
      trackAssignment(experiment.id, variant.id, visitorId);
      return variant;
    }
  }
  // Fallback: first variant
  return experiment.variants[0];
}

function hashToBucket(input: string, buckets: number): number {
  let hash = 0;
  for (let i = 0; i < input.length; i++) {
    hash = ((hash << 5) - hash) + input.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash) % buckets;
}

function getCachedAssignment(experimentId: string): { variantId: string; visitorId: string } | null {
  if (typeof window === 'undefined') return null;
  try {
    const all = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}');
    return all[experimentId] || null;
  } catch {
    return null;
  }
}

function cacheAssignment(experimentId: string, variantId: string, visitorId: string) {
  if (typeof window === 'undefined') return;
  try {
    const all = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}');
    all[experimentId] = { variantId, visitorId };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(all));
  } catch {}
}

function trackAssignment(experimentId: string, variantId: string, visitorId: string) {
  fetch('/api/ingest/v1/events', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      event_id: uuidv7(),
      event_name: 'ab_assignment',
      tenant_id: '',
      session_id: '',
      anonymous_id: visitorId,
      timestamp: Date.now(),
      properties: { experiment_id: experimentId, variant_id: variantId },
    }),
    keepalive: true,
  }).catch(() => {});
}
```

### 21.3. A/B Test Reporting (ClickHouse)

```sql
-- Conversion rate by variant
SELECT
  v.name AS variant,
  COUNT(DISTINCT a.visitor_id) AS total_visitors,
  COUNT(DISTINCT c.assignment_id) AS conversions,
  ROUND(COUNT(DISTINCT c.assignment_id) * 100.0 / COUNT(DISTINCT a.visitor_id), 2) AS cvr_pct
FROM ab_assignments a
JOIN ab_variants v ON a.variant_id = v.id
LEFT JOIN ab_conversions c ON c.assignment_id = a.id
WHERE a.experiment_id = '...'
GROUP BY v.name
ORDER BY cvr_pct DESC;

-- Statistical significance (z-test for proportions)
WITH
  (SELECT COUNT(DISTINCT c.assignment_id) FROM ab_conversions c JOIN ab_assignments a ON c.assignment_id = a.id WHERE a.variant_id = 'control') AS conv_control,
  (SELECT COUNT(DISTINCT a.visitor_id) FROM ab_assignments a WHERE a.variant_id = 'control') AS visitors_control,
  (SELECT COUNT(DISTINCT c.assignment_id) FROM ab_conversions c JOIN ab_assignments a ON c.assignment_id = a.id WHERE a.variant_id = 'variant_a') AS conv_a,
  (SELECT COUNT(DISTINCT a.visitor_id) FROM ab_assignments a WHERE a.variant_id = 'variant_a') AS visitors_a
SELECT
  conv_control::Float / visitors_control AS p_control,
  conv_a::Float / visitors_a AS p_a,
  -- Z-score
  (conv_a::Float / visitors_a - conv_control::Float / visitors_control) /
  SQRT(
    ((conv_a::Float / visitors_a * (1 - conv_a::Float / visitors_a)) / visitors_a) +
    ((conv_control::Float / visitors_control * (1 - conv_control::Float / visitors_control)) / visitors_control)
  ) AS z_score,
  -- p-value approximation
  1 - 0.5 * (1 + erf(ABS(
    (conv_a::Float / visitors_a - conv_control::Float / visitors_control) /
    SQRT(
      ((conv_a::Float / visitors_a * (1 - conv_a::Float / visitors_a)) / visitors_a) +
      ((conv_control::Float / visitors_control * (1 - conv_control::Float / visitors_control)) / visitors_control)
    ) / SQRT(2)
  ))) AS p_value
```

---

## 22. Multi-language Landing

### 22.1. i18n Architecture

```
Content storage (per tenant):
- Vietnamese (default)
- English
- Japanese
- Korean
```

### 22.2. Implementation

```tsx
// apps/landing-renderer/lib/i18n/index.ts
import { createContext, useContext } from 'react';

export type Locale = 'vi' | 'en' | 'ja' | 'ko';

export const SUPPORTED_LOCALES: Locale[] = ['vi', 'en', 'ja', 'ko'];
export const DEFAULT_LOCALE: Locale = 'vi';

export const translations = {
  vi: {
    hero: {
      cta: 'GIỮ VÉ ZOOM & NHẬN EBOOK MIỄN PHÍ',
      urgency: 'Số lượng vé đăng ký có hạn',
    },
    form: {
      name: 'Họ và tên',
      phone: 'Số điện thoại',
      consent: 'Tôi đồng ý với chính sách bảo mật',
      submit: 'GỬI NGAY',
      success: 'ĐĂNG KÝ THÀNH CÔNG!',
      error: 'Đã có lỗi xảy ra',
    },
    sections: {
      speaker: 'DIỄN GIẢ CHÍNH',
      trust: 'BẢO CHỨNG UY TÍN',
      registration: 'ĐĂNG KÝ THAM GIA',
    },
  },
  en: {
    hero: {
      cta: 'RESERVE ZOOM SEAT & GET FREE EBOOK',
      urgency: 'Limited seats available',
    },
    form: {
      name: 'Full name',
      phone: 'Phone number',
      consent: 'I agree to the privacy policy',
      submit: 'SUBMIT',
      success: 'REGISTRATION SUCCESSFUL!',
      error: 'An error occurred',
    },
    sections: {
      speaker: 'KEY SPEAKER',
      trust: 'TRUST CERTIFICATION',
      registration: 'REGISTER NOW',
    },
  },
  ja: {
    hero: {
      cta: 'ZOOM席を予約して、ebookを無料で受け取る',
      urgency: '席数限定',
    },
    // ...
  },
  ko: {
    hero: {
      cta: 'ZOOM 좌석 예약 및 ebook 무료',
      urgency: '제한된 좌석',
    },
    // ...
  },
} as const;

const I18nContext = createContext<{ locale: Locale; t: typeof translations.vi }>({
  locale: 'vi',
  t: translations.vi,
});

export const I18nProvider = I18nContext.Provider;
export const useI18n = () => useContext(I18nContext);
```

```tsx
// apps/landing-renderer/components/HeroSection.i18n.tsx
'use client';

import { useI18n } from '@/lib/i18n';

export function HeroSectionI18n() {
  const { locale, t } = useI18n();

  return (
    <section id="hero">
      <h1>{t.hero.cta}</h1>
      <span>{t.hero.urgency}</span>
    </section>
  );
}
```

### 22.3. Locale Switcher

```tsx
// apps/landing-renderer/components/i18n/LocaleSwitcher.tsx
'use client';

import { Globe } from 'lucide-react';
import { useRouter, usePathname } from 'next/navigation';
import { SUPPORTED_LOCALES, Locale } from '@/lib/i18n';

export function LocaleSwitcher({ currentLocale }: { currentLocale: Locale }) {
  const router = useRouter();
  const pathname = usePathname();

  const switchLocale = (locale: Locale) => {
    // Replace /vi/... with /en/...
    const newPath = pathname.replace(/^\/[a-z]{2}/, `/${locale}`);
    document.cookie = `__rinco_locale=${locale}; path=/; max-age=${30*24*60*60}`;
    router.push(newPath);
  };

  return (
    <select
      value={currentLocale}
      onChange={(e) => switchLocale(e.target.value as Locale)}
      className="bg-transparent border border-gray-200 rounded px-2 py-1 text-sm"
      aria-label="Language"
    >
      {SUPPORTED_LOCALES.map(loc => (
        <option key={loc} value={loc}>
          {localeFlags[loc]} {localeNames[loc]}
        </option>
      ))}
    </select>
  );
}

const localeNames = { vi: 'Tiếng Việt', en: 'English', ja: '日本語', ko: '한국어' };
const localeFlags = { vi: '🇻🇳', en: '🇬🇧', ja: '🇯🇵', ko: '🇰🇷' };
```

---

## 23. GDPR/PDPA Compliance

### 23.1. Cookie Consent Banner

```tsx
// apps/landing-renderer/components/compliance/CookieConsent.tsx
'use client';

import { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Cookie, X } from 'lucide-react';

const CONSENT_KEY = '__rinco_consent';

export function CookieConsent({ tenantConfig }: { tenantConfig: TenantConfig }) {
  const [visible, setVisible] = useState(false);
  const [showDetails, setShowDetails] = useState(false);

  useEffect(() => {
    const consent = localStorage.getItem(CONSENT_KEY);
    if (!consent) {
      // Delay 2s để không hiện ngay khi load
      setTimeout(() => setVisible(true), 2000);
    }
  }, []);

  const accept = (mode: 'all' | 'essential') => {
    const consent = {
      mode,
      timestamp: Date.now(),
      version: '1.0',
      // Categories
      essential: true,
      analytics: mode === 'all',
      marketing: mode === 'all',
      preferences: mode === 'all',
    };
    localStorage.setItem(CONSENT_KEY, JSON.stringify(consent));
    document.cookie = `__rinco_consent=${mode}; path=/; max-age=${180*24*60*60}`;
    setVisible(false);
  };

  return (
    <AnimatePresence>
      {visible && (
        <motion.div
          initial={{ y: 100, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          exit={{ y: 100, opacity: 0 }}
          className="fixed bottom-0 left-0 right-0 z-50 bg-white border-t border-gray-200 shadow-2xl p-4"
          role="dialog"
          aria-label="Cookie consent"
        >
          <div className="max-w-6xl mx-auto flex flex-col md:flex-row items-center gap-4">
            <Cookie className="w-8 h-8 text-orange flex-shrink-0" />
            <div className="flex-1">
              <p className="text-sm text-gray-700">
                Chúng tôi sử dụng cookie để cải thiện trải nghiệm của bạn, phân tích lưu lượng truy cập và tối ưu hóa quảng cáo.{' '}
                <button onClick={() => setShowDetails(true)} className="underline">Xem chi tiết</button>
              </p>
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => accept('essential')}
                className="px-4 py-2 border border-gray-300 rounded-lg text-sm font-semibold hover:bg-gray-50"
              >
                Chỉ cần thiết
              </button>
              <button
                onClick={() => accept('all')}
                className="px-4 py-2 bg-orange text-white rounded-lg text-sm font-semibold hover:bg-orange/90"
              >
                Chấp nhận tất cả
              </button>
            </div>
          </div>

          {showDetails && (
            <ConsentDetailsModal
              onClose={() => setShowDetails(false)}
              onSave={(settings) => {
                localStorage.setItem(CONSENT_KEY, JSON.stringify(settings));
                setVisible(false);
              }}
            />
          )}
        </motion.div>
      )}
    </AnimatePresence>
  );
}
```

### 23.2. Right to be Forgotten API

```
DELETE /api/gdpr/v1/me
Authorization: Bearer {user_token}

→ Anonymize PII fields:
  - name → "ANONYMIZED-{hash}"
  - phone → NULL
  - email → NULL
  - data → "{}"

→ Soft delete + schedule hard delete 30 days later
→ Revoke all sessions
→ Notify tenant admin via Slack
```

---

## 24. Performance Optimization chi tiết

### 24.1. Critical CSS Inline

```typescript
// apps/landing-renderer/lib/perf/critical-css.ts
import { PurgeCSS } from 'purgecss';

export async function extractCriticalCSS(html: string, css: string): Promise<string> {
  // Extract used class names from HTML
  const usedClasses = new Set<string>();
  const regex = /class="([^"]+)"/g;
  let match;
  while ((match = regex.exec(html)) !== null) {
    match[1].split(/\s+/).forEach(cls => usedClasses.add(cls));
  }

  // Run PurgeCSS
  const result = await new PurgeCSS().purge({
    content: [{ raw: html, extension: 'html' }],
    css: [{ raw: css }],
    safelist: {
      standard: ['^is-', '^has-', /^animate-/, /enter$/, /leave$/],
    },
  });

  return result[0]?.css || '';
}
```

### 24.2. Image Optimization Pipeline

```typescript
// apps/landing-renderer/lib/perf/image.ts
import Image from 'next/image';

export const imageLoader = ({ src, width, quality }: { src: string; width: number; quality?: number }) => {
  // Delegate to MinIO with on-the-fly resize
  if (src.startsWith('minio://')) {
    const path = src.replace('minio://', '');
    return `https://cdn.rinco.app/${path}?w=${width}&q=${quality || 75}&fmt=webp`;
  }
  return src;
};

// Component usage
<Image
  loader={imageLoader}
  src="minio://rinco-tenant-assets/apex-fintech/speaker.webp"
  alt="Speaker"
  width={800}
  height={1000}
  priority={isAboveFold}
  sizes="(max-width: 768px) 100vw, 50vw"
  placeholder="blur"
  blurDataURL={blurPlaceholder}
/>
```

### 24.3. Bundle Splitting

```typescript
// apps/landing-renderer/next.config.ts
const config: NextConfig = {
  experimental: {
    optimizePackageImports: ['lucide-react', 'framer-motion'],
  },
  webpack: (config, { dev, isServer }) => {
    if (!dev && !isServer) {
      // Split heavy libs
      config.optimization.splitChunks = {
        chunks: 'all',
        cacheGroups: {
          default: false,
          vendors: false,
          framework: {
            chunks: 'all',
            name: 'framework',
            test: /(?<!node_modules.*)[\\/]node_modules[\\/](react|react-dom|scheduler|prop-types|use-subscription)[\\/]/,
            priority: 40,
            enforce: true,
          },
          lib: {
            test(module) {
              return module.size() > 160000 &&
                /node_modules[/\\]/.test(module.identifier());
            },
            name: 'lib',
            priority: 30,
            minChunks: 1,
            reuseExistingChunk: true,
          },
        },
      };
    }
    return config;
  },
};
```

### 24.4. Performance Budget

| Resource | Budget |
|----------|--------|
| Initial HTML | < 50KB |
| Critical CSS (inline) | < 20KB |
| JavaScript (initial chunk) | < 150KB gzipped |
| Total page weight | < 500KB |
| Number of requests | < 30 |
| Image LCP | < 200KB (AVIF) |

CI check: Lighthouse CI fail nếu vượt budget.

---

## 25. Edge Cases & Error Handling

### 25.1. Form Submission Edge Cases

| Edge Case | Detection | Action |
|-----------|-----------|--------|
| **Duplicate submit (double click)** | Same event_id within 5s | Idempotent: return same lead_id |
| **Bot submit** | Wasm attestation fail | Log + 403 |
| **Rate limit exceeded** | > 100/min per IP | 429 + Retry-After |
| **Invalid phone format** | Zod regex fail | 422 + field error |
| **Email + phone empty** | Both optional fields empty | 422 (require at least 1) |
| **Suspicious IP** | Geo mismatch with language | Flag for review |
| **VPS offline** | ScyllaDB connection timeout | Buffer to local NATS + retry |
| **Tenant suspended** | Tenant status = suspended | 403 + admin notification |
| **Pixel disabled** | Tenant config enable_pixel = false | Skip pixel, only CAPI |
| **CAPI token expired** | Meta returns 401 | Auto-refresh from tenant config |
| **Network error client** | fetch reject | Retry 3x với exponential backoff |
| **Browser back after submit** | Form re-shown | Show success state từ sessionStorage |
| **Cookie disabled** | document.cookie = "" | Fallback to localStorage + IP fingerprint |
| **Privacy mode** | Do Not Track = 1 | Skip tracking events |
| **AdBlock detected** | navigator.sab test fails | Skip pixel, force CAPI |

### 25.2. Tracking Errors

| Error | Recovery |
|-------|----------|
| `fbq` not loaded (AdBlock) | Skip pixel, send CAPI only |
| `sendBeacon` fail | Queue in memory + flush on next event |
| `fetch` rejected | Retry with backoff |
| Network offline | Buffer 100 events, flush when online |
| Server 500 | Mark as "deferred", retry next page load |

### 25.3. CAPI Worker Errors

| Error | Retry Strategy |
|-------|----------------|
| Meta returns 4xx (except 429) | NO retry - bad request |
| Meta returns 5xx | Exponential backoff: 1m, 5m, 30m |
| Meta returns 429 (rate limit) | Backoff: 60s |
| Network timeout | Same as 5xx |
| Invalid HMAC | NO retry - log alert |
| Tenant not found | NO retry - log alert |
| Circuit breaker open | NO retry - DLQ |
| All retries failed | DLQ + alert admin |

### 25.4. Idempotency

```go
// services/landing-ingest/internal/handler/lead.go
func (h *LeadHandler) checkIdempotency(ctx context.Context, idempotencyKey string) (existingLeadID string, isDuplicate bool) {
    key := fmt.Sprintf("idem:lead:%s", idempotencyKey)
    existing, err := h.valkey.Get(ctx, key).Result()
    if err == nil && existing != "" {
        return existing, true
    }
    return "", false
}

func (h *LeadHandler) markIdempotency(ctx context.Context, idempotencyKey, leadID string) error {
    key := fmt.Sprintf("idem:lead:%s", idempotencyKey)
    return h.valkey.Set(ctx, key, leadID, 24*time.Hour).Err()
}
```

---

## 26. Testing Strategy

### 26.1. Unit Tests

```go
// services/meta-capi/internal/signature/normalize_test.go
package signature_test

import (
    "testing"
    "github.com/itdoanh/rinco/meta-capi/internal/signature"
)

func TestNormalizePhone_VN(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"0987654321", "84ba3..."}, // SHA-256 của "84987654321"
        {"+84987654321", "84ba3..."},
        {" 0987 654 321 ", "84ba3..."},
        {"(84) 987 654 321", "84ba3..."},
    }
    for _, tt := range tests {
        got := signature.NormalizePhone(tt.input)
        if got != tt.expected {
            t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.expected)
        }
    }
}

func TestVerifyLeadSignature(t *testing.T) {
    secretKey := "test-secret"
    leadID := "lead-123"
    fbclid := "fb.1.123456789.987654321"
    timestamp := "1725696000"
    payload := []byte(`{"email":"test@example.com","phone":"0987654321"}`)

    sig := signature.GenerateLeadSignature(leadID, fbclid, timestamp, payload, secretKey)
    if !signature.VerifyLeadSignature(leadID, fbclid, timestamp, payload, sig, secretKey) {
        t.Fatal("valid signature should verify")
    }

    // Tamper payload
    tampered := []byte(`{"email":"hacker@example.com","phone":"0987654321"}`)
    if signature.VerifyLeadSignature(leadID, fbclid, timestamp, tampered, sig, secretKey) {
        t.Fatal("tampered payload should NOT verify")
    }

    // Wrong secret
    if signature.VerifyLeadSignature(leadID, fbclid, timestamp, payload, sig, "wrong-secret") {
        t.Fatal("wrong secret should NOT verify")
    }
}
```

### 26.2. Integration Test (Testcontainers)

```go
//go:build integration

package handler_test

import (
    "context"
    "testing"

    "github.com/nats-io/nats.go"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go/modules/nats"
    "github.com/testcontainers/testcontainers-go/modules/scylla"

    "github.com/itdoanh/rinco/meta-capi/internal/worker"
)

func TestCAPIWorker_ProcessLeadEvent(t *testing.T) {
    ctx := context.Background()

    natsC, _ := nats.RunContainer(ctx)
    defer natsC.Terminate(ctx)

    scyllaC, _ := scylladb.RunContainer(ctx,
        testcontainers.WithImage("scylladb/scylla:6"),
    )
    defer scyllaC.Terminate(ctx)

    nc, _ := nats.Connect(natsC.ConnectionString(ctx))
    defer nc.Close()

    // Mock Meta API server
    metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"events_received":1,"fbtrace_id":"trace-123"}`))
    }))
    defer metaServer.Close()

    w := worker.NewWorker(nc, meta.NewClient(metaServer.URL), []byte("secret"))
    go w.Start(ctx)

    // Publish test event
    nc.Publish("lead.events", []byte(`{
        "event_id": "evt-1",
        "lead_id": "lead-1",
        "tenant_id": "test",
        "event_name": "Lead",
        "event_time": 1725696000,
        "fbclid": "fb.test",
        "user_data": {"email":"test@example.com","phone":"0987654321"},
        "signature": "...",
        "timestamp": "1725696000"
    }`))

    // Wait + assert
    time.Sleep(1 * time.Second)
    // Verify call was made to Meta
    require.Equal(t, 1, callCount)
}
```

### 26.3. E2E Test (Playwright)

```typescript
// apps/landing-renderer/e2e/landing-page.spec.ts
import { test, expect } from '@playwright/test';

test('user can submit lead form and see success', async ({ page, request }) => {
  // Mock Meta CAPI endpoint
  await page.route('**/api/ingest/v1/leads', async (route) => {
    const body = JSON.parse(route.request().body());
    expect(body.tenant_id).toBe('apex-fintech');
    expect(body.phone).toMatch(/^(\+84|0)\d{9,10}$/);

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ lead_id: 'test-lead-123', event_id: body.event_id }),
    });
  });

  await page.goto('/apex-fintech/landing');

  // Verify sections visible
  await expect(page.locator('h1')).toContainText('Tối Ưu Dòng Tiền');
  await expect(page.locator('#hero')).toBeVisible();

  // Fill form
  await page.fill('input[name="name"]', 'Nguyen Van A');
  await page.fill('input[name="phone"]', '0987654321');
  await page.check('input[name="consent"]');

  // Submit
  await page.click('button[type="submit"]');

  // Verify success
  await expect(page.locator('text=ĐĂNG KÝ THÀNH CÔNG')).toBeVisible({ timeout: 10000 });

  // Verify Meta Pixel was fired
  const fbqCalls = await page.evaluate(() => (window as any)._fbq?.queue || []);
  const hasLeadEvent = fbqCalls.some((call: any) => call[0] === 'track' && call[1] === 'Lead');
  expect(hasLeadEvent).toBeTruthy();
});

test('user sees error on invalid phone', async ({ page }) => {
  await page.goto('/apex-fintech/landing');
  await page.fill('input[name="phone"]', '123');
  await page.locator('input[name="phone"]').blur();
  await expect(page.locator('text=Số điện thoại không hợp lệ')).toBeVisible();
});

test('CTA button opens modal', async ({ page }) => {
  await page.goto('/apex-fintech/landing');
  await page.click('[data-modal-trigger="header"]');
  await expect(page.getByRole('dialog')).toBeVisible();
  await expect(page.locator('input[id="modalPhone"]')).toBeVisible();
});
```

### 26.4. Visual Regression

```typescript
// apps/landing-renderer/e2e/visual.spec.ts
import { test, expect } from '@playwright/test';

test('hero section matches design', async ({ page }) => {
  await page.goto('/apex-fintech/landing');
  await page.waitForLoadState('networkidle');

  await expect(page.locator('#hero')).toHaveScreenshot('hero.png', {
    maxDiffPixels: 100,
  });
});
```

### 26.5. Lighthouse CI

```yaml
# .github/workflows/lighthouse.yml
name: Lighthouse CI
on: [pull_request]

jobs:
  lighthouse:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
      - run: npm ci
      - run: npm run build
      - run: npm run start &
      - run: sleep 5
      - uses: treosh/lighthouse-ci-action@v10
        with:
          urls: |
            http://localhost:3000/apex-fintech/landing
          budgetPath: ./lighthouse-budget.json
          uploadArtifacts: true
```

```json
// lighthouse-budget.json
[
  {
    "path": "/*",
    "resourceSizes": [
      { "resourceType": "script", "budget": 200 },
      { "resourceType": "image", "budget": 500 },
      { "resourceType": "total", "budget": 1000 }
    ],
    "timings": [
      { "metric": "first-contentful-paint", "budget": 800 },
      { "metric": "largest-contentful-paint", "budget": 1500 },
      { "metric": "interactive", "budget": 2500 }
    ]
  }
]
```

---

## 27. Disaster Recovery

### 27.1. RPO / RTO Targets

| Resource | RPO | RTO | Method |
|----------|-----|-----|--------|
| raw_leads (ScyllaDB) | 1 hour | 2 hours | Daily snapshot |
| analytics (ClickHouse) | 1 hour | 1 hour | Replicated cluster |
| pixel config (Postgres) | 5 min | 15 min | WAL streaming |
| Form schema (Postgres) | 5 min | 15 min | WAL streaming |
| Media assets (MinIO) | 0 (versioned) | 5 min | Cross-region replication |
| Landing page templates | 0 (git) | 5 min | Git clone |

### 27.2. ScyllaDB Backup

```bash
#!/bin/bash
# scripts/backup-scylla.sh

NAMESPACE=rinco
DATE=$(date +%Y%m%d-%H%M%S)
BACKUP_BUCKET=s3://rinco-backups/scylla/${DATE}

# Snapshot từng node
for node in $(kubectl get pods -n $NAMESPACE -l app=scylladb -o name); do
  echo "Snapshotting $node..."
  kubectl exec -n $NAMESPACE $node -- nodetool snapshot -t $DATE rinco

  # Copy to MinIO
  kubectl exec -n $NAMESPACE $node -- \
    aws s3 sync /var/lib/scylla/data/snapshots/$DATE $BACKUP_BUCKET/$node/
done

# Cleanup snapshots cũ (>7 days)
for node in $(kubectl get pods -n $NAMESPACE -l app=scylladb -o name); do
  kubectl exec -n $NAMESPACE $node -- nodetool clearsnapshot -t $DATE
done
```

### 27.3. Recovery Drill

```bash
# Quarterly DR drill
# 1. Stop primary ScyllaDB cluster
kubectl scale statefulset scylladb --replicas=0 -n rinco

# 2. Restore from backup
./scripts/restore-scylla.sh s3://rinco-backups/scylla/20260907-020000

# 3. Verify
./scripts/verify-data.sh

# Expected: RTO < 2 hours
```

---

## 28. Cost Estimation

### 28.1. Per-tenant monthly cost (active landing page)

| Resource | Cost | Notes |
|----------|------|-------|
| Next.js hosting (Vercel Pro equivalent) | $20 | |
| Domain + SSL | $2 | Custom domain |
| Bandwidth (50 GB/mo) | $5 | |
| Lambda/Render compute (serverless) | $15 | Form submit + CAPI |
| ScyllaDB write (10K events/mo) | $3 | Shared |
| ClickHouse analytics | $1 | Shared |
| Valkey cache | $1 | Shared |
| Meta Pixel fee | $0 | Free |
| Meta CAPI fee | $0 | Free |

**Total: ~$47/tenant/mo**

For 100 tenants: **$4,700/mo** with shared infrastructure.
For 1,000 tenants: ~$30,000/mo (volume discount).

### 28.2. Cost Optimization

1. **ISR caching:** Static sections cached 5 phút, only form revalidate.
2. **Image CDN:** MinIO + CloudFlare, lazy load below-fold images.
3. **Edge functions:** Region-aware routing để giảm latency.
4. **Reserved capacity:** ScyllaDB + Postgres commit 1-year saving 30%.

---

## 29. Implementation Roadmap chi tiết

Roadmap 12 tuần (3 tháng) để migration xong `chiase_cu/` + nâng cấp.

### Phase 1: Foundation & Migration (Tuần 1-4)

#### Tuần 1: Inventory & Setup
- [ ] **Day 1-2:** Init repo `apps/landing-renderer/` với Next.js 15 + Bun.
- [ ] **Day 2-3:** Inventory chi tiết từ `chiase_cu/` (xem §13.3).
- [ ] **Day 3-4:** Setup design tokens (colors, fonts, spacing) trong `tailwind.config.ts`.
- [ ] **Day 4-5:** Setup CI/CD với Vercel/Cloudflare Pages.

#### Tuần 2: Component Migration
- [ ] **Day 1:** `HeroSection` component (§14.1).
- [ ] **Day 2:** `LeadForm` component (§14.2) với React Hook Form + Zod.
- [ ] **Day 3:** `MultiStepForm` (§14.3) + `Modal` (§14.4).
- [ ] **Day 4:** `FeatureCard`, `SpeakerCard`, `TrustBadge`, `FAQAccordion`.
- [ ] **Day 5:** `LogoCloud` (marquee), `Footer`, `StickyCTA`.

#### Tuần 3: Tracking & Bot Protection
- [ ] **Day 1-2:** `rinco-tracker.ts` (§15.2).
- [ ] **Day 3:** Wasm attestation module (§17).
- [ ] **Day 4:** Argon2 PoW solver (§18).
- [ ] **Day 5:** Cookie consent GDPR/PDPA (§23).

#### Tuần 4: Backend Services
- [ ] **Day 1-2:** `landing-ingest` service Go với Huma + sqlc.
- [ ] **Day 3:** `meta-capi` worker (§19).
- [ ] **Day 4:** ScyllaDB + ClickHouse schema + migrations.
- [ ] **Day 5:** NATS pub/sub setup + E2E test.

### Phase 2: Hardening (Tuần 5-8)

#### Tuần 5: Production Readiness
- [ ] **Day 1-2:** Performance optimization (§24) - critical CSS, image CDN, bundle splitting.
- [ ] **Day 2-3:** Lighthouse CI với budget (§26.5).
- [ ] **Day 3-4:** Load test k6 (1000 RPS target).
- [ ] **Day 4-5:** Security audit (rate limiting, HMAC verification).

#### Tuần 6: Feedback Loop & A/B Testing
- [ ] **Day 1-2:** CRM conversion events (§20).
- [ ] **Day 2-3:** A/B testing framework (§21).
- [ ] **Day 3-4:** A/B reporting dashboard.
- [ ] **Day 4-5:** Offline conversion upload tool.

#### Tuần 7: Multi-tenant & i18n
- [ ] **Day 1-2:** Multi-tenant config loader.
- [ ] **Day 2-3:** i18n system (§22).
- [ ] **Day 3-4:** Tenant admin dashboard (basic).
- [ ] **Day 4-5:** Domain routing (subpath/subdomain/custom).

#### Tuần 8: Observability & DR
- [ ] **Day 1-2:** OpenTelemetry tracing + Sentry integration.
- [ ] **Day 2-3:** Grafana dashboard cho Landing Page metrics.
- [ ] **Day 3-4:** Backup script + DR drill (§27).
- [ ] **Day 4-5:** Alerting rules (CAPI success rate, EMQ).

### Phase 3: Polish & GA (Tuần 9-12)

#### Tuần 9-10: UX Polish
- [ ] A/B test variants on Hero copy + CTA color.
- [ ] Accessibility audit (WCAG 2.1 AA).
- [ ] Mobile UX test trên các device phổ biến.
- [ ] SEO optimization (structured data, sitemap).

#### Tuần 11: Documentation
- [ ] Admin guide PDF/video.
- [ ] API docs từ Huma OpenAPI.
- [ ] Migration guide cho legacy tenants.
- [ ] Runbook cho on-call.

#### Tuần 12: GA Launch
- [ ] Penetration test.
- [ ] Performance baseline final.
- [ ] Canary deploy → 100% rollout.
- [ ] Marketing announcement.

### Deliverable Timeline Summary

| Week | Milestone |
|------|-----------|
| 4 | Migration xong `chiase_cu/` + Tracking MVP |
| 8 | Production-ready với A/B test + observability |
| 12 | GA với đầy đủ i18n, multi-tenant, DR |

---

## 30. Open Questions / Cần user xác nhận

1. **Migration chiase_cu sang React:** Có cần giữ 100% HTML structure cũ (từng `<section>` theo thứ tự), hay có thể restructure lại cho cleaner React component tree?

2. **Asset storage:** Hình ảnh từ `chiase_cu/` đặt ở MinIO bucket `rinco-tenant-assets/{tenant_id}/` - có cần tổ chức folder con (vd: `/images/`, `/logos/`) không?

3. **Pixel ID:** Mỗi tenant có 1 pixel ID riêng (đã có sẵn cho `apex-fintech`), hay share giữa các tenant? Nếu share, cần tenant_id custom dimension.

4. **Multi-language scope:** Phase 1 chỉ Tiếng Việt + English, hay launch với cả 4 ngôn ngữ (vi/en/ja/ko)?

5. **A/B test priority:** Sau migration, test variant nào đầu tiên? (Hero headline, CTA color, form length...)

6. **CAPI event value:** Custom event `QualifiedLead` value = 0, hay gửi `predicted_ltv`? Meta có thể tối ưu tốt hơn với predicted value.

7. **Form auto-save:** Có cần auto-save draft localStorage mỗi 5s không? UX tốt nhưng GDPR concerns.

8. **Bot detection strictness:** Wasm attestation fail → block submit, hay chỉ flag for review?

9. **Cookie consent default:** GDPR/PDPA - default là "essential only" (cần user opt-in) hay "all" (opt-out)?

10. **Modal trigger timing:** Hiện modal popup sau bao lâu? Hiện tại `chiase_cu` không tự động (chỉ click trigger). Có nên thêm exit-intent popup không?

11. **Video section:** `chiase_cu` không có video, nhưng tính năng #16/#58 có. Có cần thêm video explainer vào landing không?

12. **Custom domain SSL:** Tenant muốn custom domain (vd `landing.apex.vn`) - chứng chỉ SSL tự động qua Let's Encrypt / Caddy?

13. **Spam filter rule:** Ngoài PoW + honeypot + Wasm, có cần AI-based spam detection (vd: phone thuộc known spam list)?

14. **Webhook cho tenant:** Cho phép tenant config webhook URL nhận event khi lead submit? Nếu có, cần HMAC signing.

15. **GDPR data export:** Có cần implement data export API cho user request "My data" theo GDPR Article 15?

16. **UTM persistence:** UTM params giữ bao lâu trong cookie? 30 ngày hay session?

17. **Multi-step form analytics:** Track drop-off rate mỗi step? Hiển thị trong dashboard?

18. **CTA button text:** "GIỮ VÉ ZOOM" có còn phù hợp cho non-webinar landing, hay nên template variable?

19. **Risk warning section:** `chiase_cu` có risk warning cho đầu tư - có cần template này cho mọi finance-related landing?

20. **WhatsApp integration:** Floating WhatsApp button (#14) - dùng `wa.me/{phone}?text=...` hay WhatsApp Business API?

---

**Tiếp theo:** [`docs/06-chat-engine/README.md`](../06-chat-engine/README.md) – Chat Real-time Engine (đã được mở rộng).
