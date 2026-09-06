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

**Tiếp theo:** [`docs/06-chat-engine/README.md`](../06-chat-engine/README.md) – Chat Real-time Engine.