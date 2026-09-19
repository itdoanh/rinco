# Landing Page + Facebook CAPI – Summary

## Overview

Phần 5 (`docs/05-landing-capi/README.md`, ~220 KB, 6949 dòng) thiết kế hệ thống Landing Page + Meta Conversions API (CAPI) Hybrid Dual-Tracking cho RINCO. Mục tiêu cốt lõi: **kế thừa 100 % nội dung từ thư mục legacy `chiase_cu/`** (dự án cũ viết bằng jQuery + Bootstrap + Tailwind CDN + AOS + Meta Pixel thuần), thay thế bằng stack hiện đại (Next.js 15 + Bun + Tailwind v4 + shadcn/Radix + Wasm + Go), đồng thời nâng cấp tracking lên hybrid client Pixel + server CAPI với HMAC chống giả mạo, bắn ngược conversion từ CRM về Meta để tối ưu quảng cáo, và chống bot bằng Wasm hardware attestation + Argon2 PoW.

Ba service chính: `landing-renderer` (Next.js), `landing-ingest` (Go + Huma), `meta-capi` (Go worker), cộng `bot-detection` (Rust Wasm). Acceptance gate 20 AC (Lighthouse ≥ 95, FCP < 0.4 s, CAPI success ≥ 99.5 %, EMQ ≥ 7, bot detection ≥ 99 %, RPO < 5 min, cost < $5/tenant/tháng).

## Nguyên tắc kế thừa `chiase_cu/`

- **Nội dung văn bản**: copy 100 % không thay đổi nghĩa. Cấu trúc DOM (các `<section>` theo thứ tự cũ) giữ nguyên thứ tự.
- **Hình ảnh**: copy sang MinIO, reference lại.
- **Branding**: logo, slogan giữ nguyên trừ khi tenant yêu cầu thay.
- **Bảng thay thế thư viện**: jQuery → Vanilla JS/Web Components, Bootstrap 4 → Tailwind v4 + shadcn, Font Awesome → Lucide Icons (SVG tree-shake), Slick → Embla Carousel, AOS → framer-motion, jQuery Validate → Zod + React Hook Form.
- **Performance target**: FCP < 0.4 s, LCP < 0.8 s, TTI < 1.2 s, CLS < 0.05, TBT < 100 ms.

> **Lưu ý**: README không mô tả tier hot SSD / cold HDD. Toàn bộ storage dùng MinIO bucket `rinco-tenant-assets/{tenant_id}/` (asset) và `rinco-backups/` (backup). Tiering SSD/HDD không có — tham chiếu ADR `0006-tiered-storage-minio.md` ở phần khác nếu cần.

## Landing Page Block System

Landing page là một chuỗi **block** có thể compose theo tenant config. Mỗi block có props riêng, lưu JSONB trong `landing_pages.content` của PostgreSQL. Cấu trúc chính (kế thừa `chiase_cu/index.html`, ~785 dòng):

| # | Block | Props chính | Mục đích |
|---|-------|-------------|----------|
| 1 | **TopBar** | `hotline`, `companyName`, `ctaTrigger` | Hotline + nút CTA trên cùng |
| 2 | **StickyHeader / Navbar** | logo, nav items, CTA button | Menu sticky khi scroll |
| 3 | **MobileMenu** | nav items | Drawer slide-in (Radix Dialog) |
| 4 | **Hero** | `headline.{line1,line2,subline}`, `subheadline`, `urgency`, `cta_text`, `heroImage`, `chips[]` | Headline + ảnh chuyên gia + form inline + 3 chip (TOP1, T+0, MXV) |
| 5 | **LogoCloud / Marquee** | `title`, `logos[]` | MXV, CQG, NYMEX, CBOT, LME — chứng nhận |
| 6 | **MarketContext** | `tag`, `headline`, `data_point`, `cards[4]` (4 trụ cột: T+0, 2 chiều, Margin, Quốc tế) | Bối cảnh thị trường 2026 |
| 7 | **SpeakerSection** | `name`, `title`, `company`, `bio_bullets[]`, `photo` | Card diễn giả Nguyễn Tuấn Anh |
| 8 | **TrustSection** | `banner_image`, `badges[4]` | Top 1 banner + 4 trust badge |
| 9 | **RegistrationForm (Multi-step)** | `steps[]`, `event_info.{date,format,gift}`, `channels[]` | 2-step: chọn kênh (Zalo/Phone) → nhập info |
| 10 | **RiskWarning** | `icon`, `title`, `text` | Cảnh báo rủi ro thị trường |
| 11 | **CompanyInfo** | `name`, `address`, `hotline`, `website` | Thông tin APEX Fintech |
| 12 | **Footer** | `copyright`, `links[]` | Multi-column footer |
| 13 | **StickyCTA** | `ctaText`, `formConfig`, `threshold` | Mobile + desktop bottom bar |
| 14 | **LeadCaptureModal** | `triggerLabel`, `triggerLocation`, `formConfig` | Modal popup (Radix Dialog) |
| 15 | **CookieConsent** | `tenantConfig` | GDPR/PDPA banner |
| 16 | **FAQAccordion** | `items[{q,a}]` | Hỏi đáp (Radix Accordion) |
| 17 | **TestimonialsCarousel** | `items[{avatar,name,role,quote}]` | Embla Carousel |
| 18 | **PricingTable** | `tiers[]` | Bảng giá (nếu áp dụng) |
| 19 | **VideoEmbed** | `url`, `poster`, `lazy` | Video lazy-load |
| 20 | **FloatingWhatsApp** | `phone`, `message` | Nút WhatsApp nổi |

Block system được drive bởi `landing_pages` table:

```sql
CREATE TABLE landing_pages (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  slug TEXT NOT NULL,
  template TEXT NOT NULL,
  content JSONB NOT NULL,       -- các block + props
  form_config JSONB NOT NULL,   -- LeadFormConfig riêng
  seo_config JSONB,
  is_published BOOLEAN DEFAULT false,
  UNIQUE(tenant_id, slug)
);
```

`LeadFormConfig` (props của form) gồm: `tenant_id`, `campaign_id`, `fields[]`, `submit_endpoint`, `enable_pixel`, `enable_capi`, `enable_wasm_attestation`, `enable_argon2_pow`, `phone_required`, `email_required`, `consent_required`. `FormField` có `code`, `label`, `type` (`text|phone|email|select|textarea|hidden`), `required`, `placeholder`, `options`, `validation`.

## Tiered S3 Storage

README **không mô tả cụ thể tiering hot SSD / cold HDD**. Toàn bộ storage assets dùng **MinIO (S3-compatible)** với hai bucket chính:

- **Tiering**: README không mô tả cụ thể tier hot SSD / cold HDD. Toàn bộ storage dùng **MinIO (S3-compatible)**:
  - Bucket `rinco-tenant-assets/{tenant_id}/` — asset tenant (logo, ảnh chuyên gia, banner), image resize on-the-fly qua `imageLoader` (`?w=&q=&fmt=webp`). Format: webp + avif. Sizes gợi ý: logo [16, 32, 64, 128, 256], speaker [400, 800, 1200], banner [800, 1600].
  - Bucket `rinco-backups/` — backup dump hàng ngày (PostgreSQL, ScyllaDB snapshot, ClickHouse, MinIO mirror).

Backup RPO/RTO (chi tiết trong §39): landing pages (PostgreSQL) RPO 5 min, raw leads (ScyllaDB) RPO 1 giờ, asset files (MinIO) RPO 0 (cross-region replication), HMAC secrets lưu HashiCorp Vault. Nếu cần tiering SSD/HDD thực sự phải tham chiếu ADR `0006-tiered-storage-minio.md` ngoài doc này.

## Lead Ingestion Flow (User Form → CRM)

```
[Browser] Load page → init tracking, scroll/time-on-page → Fill form (Zod realtime + honeypot)
   ↓ Submit
   a. fbq('track','Lead',{...}, { eventID })  ← Meta Pixel client
   b. POST /api/ingest/v1/leads  (Headers: X-RINCO-Attestation + X-PoW-* nếu cần)
[Go landing-ingest] Middleware verify HMAC + Wasm attestation + Argon2 PoW
   → Rate limit per IP (Valkey Lua, 100 req/min) → Validate (Zod)
   → Idempotency check (Valkey, key = tenant_id:phone:event_id)
   → Generate HMAC signature cho CAPI worker
   → Write ScyllaDB `raw_leads` (partition tenant_id, cluster created_at) async
   → Emit NATS event `lead.created.{tenant_id}` (subject per-tenant)
[NATS JetStream] Subscribers: analytics-collector (ClickHouse) / AI Scoring worker / meta-capi worker / Notification
[CRM Service] Lead record tạo, trigger conversion events ngược về Meta
```

Edge case: duplicate submit (idempotency), bot submit (ghi analytics, không ghi CAPI), failed validation (422), rate limit (429 + Retry-After), system down (fallback Valkey queue).

## Meta Pixel Client-side Tracking

`RincoTracker` (TypeScript singleton) capture đầy đủ context gửi về server:

- **Identifiers**: `event_id` (UUIDv7), `session_id` (cookie `__rinco_session`, 30 ngày), `anonymous_id` (cookie `__rinco_anon`, 1 năm), `user_id` (nếu login).
- **Source params**: `utm_source/medium/campaign/content/term`, `fbclid`, `fbc` (Facebook Click ID cookie hoặc synthesize `fb.1.{ts}.{fbclid}`), `fbp` (Facebook Browser ID), `gclid`, `ttclid`.
- **Device**: `user_agent`, `screen_resolution`, `viewport_size`, `language`, `timezone`.
- **Page**: `referrer`, `landing_page`, `current_page`, `page_title`.
- **Engagement**: `scroll_depth` (25/50/75/100 %), `time_on_page` (30/60/120/300 s), `click_count`.
- **Meta**: `timestamp`, `tenant_id`, `campaign_id`, `trace_id`.

Tracking event taxonomy (22 event): `page_view`, `view_content`, `scroll_depth`, `time_on_page`, `cta_click`, `form_start`, `form_field_focus`, `form_field_complete`, `form_submit`, `form_submit_success`, `form_submit_error`, `lead_qualified`, `video_play`/`pause`/`complete`, `outbound_click`, `chat_open`, `phone_click`, `email_click`, `social_click`, `share`, `exit_intent`.

Pixel fire kèm `eventID` match với server-side CAPI để Meta deduplicate trong vòng 48 h. Capture: client JS SDK + server `__rinco_session` cookie + IP header, hybrid (critical event gửi cả client + server redundancy). Storage: ScyllaDB `raw_events` 90 ngày, ClickHouse `analytics` aggregate 2 năm.

## Facebook CAPI Server-side

Cấu trúc `CAPIEvent` (`event_name`, `event_time`, `event_id` match client, `event_source_url`, `action_source="website"`, `user_data`, `custom_data`). `UserData` SHA-256 normalized cho `em`/`ph`/`fn`/`ln`/`ct`/`st`/`country`/`zp`/`external_id`; pass-through un-hashed cho `client_ip_address`, `client_user_agent`, `fbc`, `fbp`, `subscription_id` (tăng EMQ).

**SHA-256 Normalization (Meta spec)**: `NormalizeEmail` TrimSpace → Lowercase → SHA-256; `NormalizePhone` Strip non-digits → drop leading 0 → prepend "84" nếu VN → SHA-256; `NormalizeName/City/State/Zip/Country/ExternalID` TrimSpace → Lowercase → SHA-256.

**HMAC Signature (chống CAPI Poisoning)** — `GenerateLeadSignature(leadID, fbclid, timestamp, payload) = hex(SHA256_HMAC(secretKey, leadID || fbclid || timestamp || canonical_payload_json))`. Canonical = JSON sorted keys alphabetically.

**CAPI Worker flow** (`services/meta-capi/internal/worker/worker.go`): (1) Subscribe NATS `lead.events` + `crm.conversion`; (2) Verify HMAC (reject + log nếu sai); (3) Load tenant pixel config (pixel_id, capi_token_encrypted, test_event_code, emq_target); (4) Normalize user data theo Meta standard; (5) Build CAPIEvent với `event_id` match client; (6) POST `graph.facebook.com/v18.0/{pixel_id}/events` qua **sony/gobreaker circuit breaker** (open sau 5 consecutive failures hoặc failure ratio ≥ 0.5; half-open sau 60 s); (7) Retry exponential backoff (1 m → 5 m → 30 m); 4xx (except 429) không retry; rate limit 429 backoff 60 s; (8) Log response `events_received`, `fbtrace_id` vào ScyllaDB `capi_events`; update ClickHouse metric; (9) Event Match Quality (EMQ) target ≥ 7/10, alert nếu < 6.

## Feedback Loop (CRM → Meta Conversion)

```
[CRM crm-core] Deal Won → Publish NATS "crm.conversion" → meta-capi worker
  → POST graph.facebook.com events event_name=Purchase, value=deal_value → Meta Ads Algorithm
```

**Conversion Events (CRM → Meta)**: `Lead Created (AI score ≥ 70)` → `QualifiedLead` + `predicted_ltv`; `Lead Created (< 70)` → `Lead`; `Appointment scheduled` → `Schedule`; `Deal Won` → `Purchase` + value + currency; `Subscription activated` → `Subscribe` + value + predicted_ltv; `Score ≥ 90` → `Custom_High_Value`; `SQL Qualified` → `Custom_SQL_Qualified`. Server-side only (PII nhiều, không qua Pixel): `Custom_Audience_Upload`, `Custom_Conversion` offline.

## Wasm Hardware Attestation (Anti-Bot)

Rust Wasm module (`crates/bot-detection-wasm`) chạy trong browser, kiểm tra nhiều signal để phân biệt human vs bot/Selenium/Puppeteer: (1) `navigator.webdriver === true` → score -= 0.3; (2) UA chứa `HeadlessChrome`/`PhantomJS`/`selenium` → -= 0.5; (3) WebGL renderer `swiftshader`/`llvmpipe` → -= 0.3-0.4; (4) Canvas fingerprint SHA-256 hash; (5) Mouse movement non-linear (score < 0.3 → -= 0.3); (6) Mobile touch events (`maxTouchPoints=0` → -= 0.2); (7) AudioContext; (8) Timing check (real browser 100-500 ms; < 10 ms GPU bot hoặc > 5 s → block).

Pass → sinh `X-RINCO-Attestation` JWT-like token (base64url `header.payload.signature`, HS256, TTL 1 h, chứa `challenge`, `timestamp`, `score`, `signals`, `exp`). Score ≥ 0.6 coi là human. Fail → form ẩn + show CAPTCHA fallback (hCaptcha hoặc Cloudflare Turnstile). Build: `cargo build --target wasm32-wasi --release` → `wasm-opt -O3` → mục tiêu < 50 KB gzipped. Wasm load qua fetch + `cache: force-cache`, SHA-256 verify integrity, instantiate với memory exports.

Server-side (`landing-ingest` middleware) verify token HMAC, check expiry, gắn `visitor_id` vào context. Nếu `!IsHuman` → slog.Warn + 403 (silent drop).

## Argon2 PoW Challenge Protocol

Khi traffic vượt threshold hoặc IP đáng ngờ, server yêu cầu client compute proof-of-work: `POST /api/ingest/v1/leads` → server return 202 + `X-PoW-Challenge` (`challenge_id`, `salt`, `ts`, `difficulty` bits) → client compute Argon2id(salt|ts|nonce) cho tới khi `hash[0:difficulty/8] == 0` (~20-50 ms trên browser thật) → retry POST với headers `X-PoW-Challenge-Id`, `X-PoW-Timestamp`, `X-PoW-Nonce` → server verify + accept. Adaptive difficulty tăng khi traffic spike. Challenge TTL 60 s, ±2 min timestamp window, single-use. Real browser ~20 ms; GPU bot < 10 ms (block); mobile chậm bất thường > 1 s (block).

## Lead Scoring & ATT/ITP Bypass Strategy

**Lead Scoring**: NATS pipeline có `AI Scoring worker` subscribe `lead.events`, output 0-100. Score ≥ 90 → publish `Custom_High_Value` với `predicted_ltv`; ≥ 70 → publish `QualifiedLead` với `predicted_ltv`; < 70 → publish `Lead`. High score trigger Notification (Slack/Telegram) cho sales. Score lưu vào `leads.ai_score`, dùng cho value optimization khi chưa có deal thực. Bot scoring dashboard riêng trong ClickHouse cho `bot_events`.

**ATT/ITP Bypass**: Apple ITP chặn 30 % third-party cookie, AdBlocker chặn Pixel script. Bypass: (1) Hybrid Dual-Tracking — server-side CAPI là primary, client Pixel supplement, cùng `eventID` để Meta dedup; (2) First-party cookie `__rinco_session`, `__rinco_anon`, `__rinco_consent`, `__rinco_locale` set `SameSite=Lax` không bị ITP xóa trong 7 ngày; (3) Server-side fbc synthesize `fb.1.{ts}.{fbclid}` từ URL `fbclid` khi `_fbc` cookie bị clear; (4) Fallback server-side only khi disable third-party / iOS ITP → skip Pixel; (5) Pass-through un-hashed fields `client_ip_address`, `client_user_agent`, `fbc`, `fbp`, `subscription_id` để tăng EMQ; (6) UTM persistence 30 ngày first-party cookie stitch session across ITP resets; (7) Aggregated Event Measurement giảm phụ thuộc pixel-level data.

## Database Schema

**ScyllaDB** (raw, partition by tenant_id+timestamp): `rinco.raw_leads` (payload JSON, user_data JSON, source, campaign_id, event_name; PK `((tenant_id, created_at), event_id)`); `rinco.bot_events` (score, reasons list); `rinco.capi_events` (sent_at, response_code, events_received, fbtrace_id, error_message).

**ClickHouse** (aggregated, 2 năm retention): `rinco.analytics` (`AggregatingMergeTree` PARTITION BY `toYYYYMM(event_date)`, ORDER BY `(tenant_id, event_date, event_hour, event_name)`; unique users via `AggregateFunction(uniq, String)`); `rinco.funnel` (`MergeTree` cho page_view → cta_click → form_start → form_submit → qualified).

**PostgreSQL** (metadata): `tenant_pixels` (pixel_id, capi_token_encrypted AES, test_event_code, emq_target INT DEFAULT 7); `landing_pages` (slug, template, content JSONB, form_config JSONB, seo_config JSONB); `ab_tests`, `ab_experiments`, `ab_variants`, `ab_assignments`, `ab_conversions` (A/B framework); `ingest_logs` cho landing-ingest service.

## API Surface

**Public Landing**:
- `GET /l/:tenant/:slug` — render landing page.
- `GET /api/public/lng/:tenant/:slug/config` — get form config.
- `POST /api/ingest/v1/leads` — submit lead.
- `POST /api/ingest/v1/events` — submit any event (analytics).
- `GET /api/ingest/v1/pow-challenge` — get PoW challenge nếu cần.

**Tenant Admin (`/api/lp-admin/v1/`)**:
- `GET/POST/PATCH/DELETE /pages`, `POST /pages/:id/publish`, `POST /pages/:id/ab-test`.
- `GET/POST/PATCH /pixels`, `/forms`.
- `GET /analytics/overview|funnel|campaigns`.
- `GET /capi-log`, `POST /test-event` (Meta Test Events).

## Frontend Stack & Performance

- **Next.js 15** App Router + React Server Components + Streaming SSR + Bun runtime.
- **Tailwind CSS v4** + CSS Modules; Lucide Icons; Heroicons backup; Custom components với React Aria.
- **Framer Motion** cho entrance/exit; CSS transitions hover/focus; `prefers-reduced-motion` respected.
- **React Hook Form + Zod** validation; server-side validation luôn (defense in depth).
- **`next/image`** AVIF/WebP auto; self-host Inter font, `font-display: swap`; preload critical; code splitting automatic; Edge SSR.
- **Performance budget**: Initial HTML < 50 KB, critical CSS inline < 20 KB, JS initial chunk < 150 KB gzipped, total page < 500 KB, requests < 30, LCP image < 200 KB AVIF. CI Lighthouse fail nếu vượt.
- **Custom components**: Hero (image/video/animated variants), FeaturesGrid, TestimonialsCarousel (Embla), FAQAccordion (Radix), ContactForm, CookieConsent (PDPA), StickyHeader, MobileMenu, FloatingCTA, VideoEmbed, ImageOptimized.

## Tất cả Block Types trong chi tiết

1. **HeroSection** (§14.1): eyebrow (webinar tag), `headline.line1/line2/subline` (`text-navy-900`, `gradient-text`), `subheadline`, `UrgencyBanner` (red, ⚡ icon, seatsLeft), inline `LeadForm`, hero image 3 floating chips (TOP1/T+0/MXV), float-shape decorative.
2. **LeadForm** (§14.2): variant `inline|modal|section`, `react-hook-form` + `zodResolver`, honeypot field `website` (`position: absolute; left: -9999px`), consent checkbox, GDPR link, trust badge (Shield icon), event info chips (📅 + 💻).
3. **MultiStepForm** (§14.3): 2 step, Step 1 chọn `channel` enum `zalo|phone` (button ChannelOption), Step 2 nhập `name`, `phone` (regex `/^(\+84|0)\d{9,10}$/`), consent, back button, StepIndicator component (active/completed state).
4. **ModalLeadCapture** (§14.4): Radix Dialog + framer-motion backdrop + spring animation content, modal badge "Số lượng vé miễn phí có giới hạn!", modal header với Dialog.Title/Description, focus trap + ESC close.
5. **LogoCloud**: marquee logo MXV/CQG/NYMEX/CBOT/LME, pause on hover.
6. **MarketContext**: 4 cards với num (01-04), icon emoji, title, description — trụ cột T+0/2 chiều/Margin/Quốc tế.
7. **SpeakerSection**: photo + name + title + company + bio_bullets (4 items).
8. **TrustSection**: banner_image top + 4 TrustBadge (icon + title + desc): TOP1, Pháp Lý Minh Bạch, CQG, Real-time Data.
9. **RegistrationForm (multi-step)**: composite block chứa MultiStepForm + event_info card.
10. **RiskWarning**: amber alert, ⚠ icon, title + text disclaimer.
11. **CompanyInfo**: tên, địa chỉ B-TT8-3 Him Lam Vạn Phúc Hà Đông, hotline 0984386538, website `https://hanghoaphaisinh.net/`.
12. **Footer**: copyright "© 2026 hanghoaphaisinh.net" + 3 links (Điều khoản, Chính sách bảo mật, Cảnh báo rủi ro).
13. **StickyCTA**: bottom bar mobile + desktop, scroll threshold > 600 px.
14. **CookieConsent**: GDPR/PDPA banner, 3 modes (essential/all/custom detail), `__rinco_consent` cookie 180 ngày.
15. **FAQAccordion**: items q/a, Radix Accordion.
16. **TestimonialsCarousel**: Embla Carousel, items avatar/name/role/quote.
17. **PricingTable**: tiers[].
18. **VideoEmbed**: lazy load, poster image.
19. **FloatingWhatsApp**: `wa.me/{phone}?text=...` hoặc WhatsApp Business API.
20. **TopBar / StickyHeader / MobileMenu**: navigation chrome.

## Tất cả Conversion Events

**Client-side (Pixel)**: `PageView`, `ViewContent`, `Lead`, `VideoPlay`, `VideoComplete`, `CompleteRegistration`, custom `WebinarRegistration`.

**Server-side (CAPI)**: `Lead`, `QualifiedLead` (AI score ≥ 70), `Schedule` (đặt lịch), `Purchase` (Deal Won, value = deal_value), `Subscribe` (subscription active), `Custom_High_Value` (score ≥ 90), `Custom_SQL_Qualified`, `Custom_Audience_Upload`, `Custom_Conversion` (offline), `CompleteRegistration`.

**Internal tracking events** (không gửi Meta nhưng ghi analytics): `page_view`, `view_content`, `scroll_depth`, `time_on_page`, `cta_click`, `form_start`, `form_field_focus`, `form_field_complete`, `form_submit`, `form_submit_success`, `form_submit_error`, `lead_qualified`, `video_play`, `video_pause`, `video_complete`, `outbound_click`, `chat_open`, `phone_click`, `email_click`, `social_click`, `share`, `exit_intent`, `modal_open`, `ab_assignment`, `experiment_exposed`.

## ≥ 30 Distinct Features

1. Hybrid Dual-Tracking (Pixel + CAPI cùng eventID).
2. HMAC-SHA256 chống CAPI Poisoning trên từng lead.
3. SHA-256 user data normalization theo Meta spec (email, phone, name, city, zip, country, external_id).
4. Meta Graph API client với retry exponential backoff.
5. sony/gobreaker circuit breaker (open sau 5 fail hoặc ratio ≥ 0.5).
6. EMQ monitoring & alert khi score < 6.
7. Wasm hardware attestation (Rust → wasm32-wasi, target < 50 KB gzipped).
8. Argon2id proof-of-work adaptive challenge (memory 16 KB, time ~20 ms).
9. eBPF/XDP L4 filtering (drop known bot IP, geo-block).
10. Fingerprinting: Canvas, AudioContext, WebGL, Font enumeration.
11. Honeypot field `website` ẩn tuyệt đối + tabIndex -1.
12. CAPTCHA fallback (hCaptcha / Cloudflare Turnstile) chỉ khi bot score > threshold.
13. Rate limit per IP + per fingerprint (Valkey Lua, 100 req/min).
14. Idempotency key check (Valkey 24 h TTL) chống double-click.
15. RINCO custom tracking SDK (`RincoTracker` singleton, UUIDv7).
16. Scroll depth (25/50/75/100 %), time-on-page (30/60/120/300 s) milestones.
17. Exit-intent popup tracking (`mouseleave` khi `clientY <= 0`).
18. Multi-step form (2 bước: chọn kênh → nhập info).
19. Modal lead capture với Radix Dialog (focus trap + ESC + spring animation).
20. Sticky CTA bar sau 600 px scroll.
21. Cookie consent GDPR/PDPA với 3 modes (essential/all/custom).
22. Server-side fbc synthesize từ fbclid query param (`fb.1.{ts}.{fbclid}`).
23. Pass-through un-hashed fields cho Meta EMQ (client_ip, client_ua, fbc, fbp, subscription_id).
24. Lead scoring AI worker (0-100), threshold 70 / 90.
25. predicted_ltv value optimization cho Conversion API.
26. Per-tenant pixel config với RLS + AES-encrypted CAPI token.
27. Conversion API token rotation, Meta Test Events integration.
28. Aggregated Event Measurement + offline conversion upload tool.
29. Custom audience creation API, domain verification helper.
30. A/B testing framework (PostgreSQL + deterministic SHA-256 hash assignment).
31. Statistical significance calculator (two-proportion z-test, p-value).
32. Multi-language landing (vi/en/ja/ko) với locale switcher + cookie persist 30 ngày.
33. Right to be Forgotten API (`DELETE /api/gdpr/v1/me` — anonymize PII, soft delete 30 ngày).
34. MinIO asset storage với image on-the-fly resize (`?w=&q=&fmt=webp`) + AVIF/WebP.
35. Multi-tenant pixel per landing page, webhook cho tenant (HMAC signed).
36. Lighthouse CI với budget assertion (Perf ≥ 95, FCP < 400 ms, LCP < 800 ms, CLS < 0.05, TBT < 100 ms).
37. Visual regression test (Playwright screenshot + maxDiffPixels 100).
38. E2E test Playwright (form submit, fbq queue assert, modal open).
39. Integration test Testcontainers (NATS + ScyllaDB + mock Meta server).
40. Disaster Recovery drill quarterly (snapshot/restore test).
41. ScyllaDB daily snapshot to MinIO (`aws s3 sync`), backup SHA-256 verify.
42. RPO/RTO documented per resource (PostgreSQL WAL 5 min, ScyllaDB 1 h, MinIO replicated).
43. Cost estimation per tenant (~$3.42/tháng, vs SaaS $37+ tiết kiệm 90 %).
44. Edge case matrix 60 scenarios (E1-E60) chia theo Pixel/CAPI/Form/Tracking/Bot/Conversion trong §34.
45. Tracking queue overflow protection (drop oldest > 100 events).
46. SendBeacon fallback khi `visibilityState === 'hidden'`.
47. Offline mode buffer (queue 100 events, flush khi online).
48. AdBlock detection (`navigator.sab` test) → skip pixel, force CAPI.
49. Slack/Telegram notification cho high-score lead + webhook retry 100 %.
50. Risk Warning block cho finance landing (chuẩn compliance VN).
51. Custom domain routing (subpath/subdomain/custom), Let's Encrypt/Caddy.
52. URL params auto-fill utm_* vào form hidden fields.
53. SEO schema.org JSON-LD (Event/Webinar) + Open Graph + Twitter Card.
54. Critical CSS inline + PurgeCSS, next/image lazy load.
55. ClickHouse AggregatingMergeTree cho real-time funnel + cohort analysis + ROI calculator.
56. OpenTelemetry tracing + Sentry integration + Grafana dashboard.
57. GDPR consent mode v2, CCPA "Do Not Sell" link (open question §41).
58. Email lead magnet auto-send ebook PDF sau submit, PWA install prompt.
59. Form pre-fill nếu user submit trước, multi-step resume mid-flow.
60. Edge function personalization (theo IP/cookie), push notification opt-in.

## Legacy `chiase_cu/` Content phải giữ 100 % verbatim

File gốc `chiase_cu/index.html` (785 dòng), phân tích chi tiết trong §13.3. **Tất cả text dưới đây phải copy nguyên văn vào các block tương ứng**:

| File / Lines | Block mới | Nội dung bắt buộc verbatim |
|--------------|-----------|---------------------------|
| `chiase_cu/index.html` lines 84-99 | TopBar | Hotline + CTA "GIỮ VÉ ZOOM" |
| `chiase_cu/index.html` lines 100-122 | StickyHeader | Logo + nav + CTA mobile |
| `chiase_cu/index.html` lines 123-146 | MobileMenu | Slide-in drawer |
| `chiase_cu/index.html` lines 147-256 | HeroSection | Headline "Tối Ưu Dòng Tiền 2026 Với" / "Kênh Đầu Tư Hàng Hóa Phái Sinh" / "(Chính Thống - Quản Lý Bởi Bộ Công Thương & MXV)", subheadline "Giải mã cơ chế giao dịch T+0 & sinh lời 2 chiều linh hoạt...", urgency "⚡ Số lượng vé đăng ký có hạn...", CTA "GIỮ VÉ ZOOM & NHẬN EBOOK MIỄN PHÍ", hero_image `anh/anhchandung_chuyengia_nguyentuananh.webp`, 3 chips TOP1/T+0/MXV |
| `chiase_cu/index.html` lines 257-283 | LogoCloud | "Được cấp phép chính thức và liên thông giao dịch quốc tế" + logos MXV/CQG/NYMEX/CBOT/LME |
| `chiase_cu/index.html` lines 284-353 | MarketContext | "BỐI CẢNH THỊ TRƯỜNG 2026" / "TẠI SAO HÀNG HÓA PHÁI SINH ĐANG LÀ KÊNH ĐẦU TƯ BÙNG NỔ?" + data_point "Thanh khoản bình quân 7.500 tỷ – 17.000 tỷ đồng/ngày" + 4 cards 01-04 |
| `chiase_cu/index.html` lines 354-394 | SpeakerSection | "NGUYỄN TUẤN ANH", "Giám đốc Phát triển Thị trường", "APEX Fintech", 4 bio_bullets, photo `anh/anhchandung_chuyengia_nguyentuananh.webp` |
| `chiase_cu/index.html` lines 395-444 | TrustSection | banner `anh/apex_top1_thiphan_quy2_2026.webp` + 4 badges TOP1 / Pháp Lý Minh Bạch / CQG / Real-time Data |
| `chiase_cu/index.html` lines 445-565 | RegistrationForm (multi-step) | Step 1 "Bạn muốn nhận suất qua kênh nào?" với 2 channel Zalo/Phone; Step 2 "Nhập thông tin để Hệ thống Gửi Vé Zoom & Ebook Miễn Phí" với name/phone; event_info date "20:00 – 22:00 | 07/09/2026", format "Zoom Online", gift "Tặng Bộ 10 Ebook + Tư vấn 1:1" |
| `chiase_cu/index.html` lines 566-606 | RiskWarning + CompanyInfo | "CẢNH BÁO RỦI RO THỊ TRƯỜNG" + "CÔNG TY CỔ PHẦN CÔNG NGHỆ TÀI CHÍNH APEX" / "B-TT8-3 Him Lam, Vạn Phúc, Phường Hà Đông, Thành phố Hà Nội" / hotline "0984386538" / website `https://hanghoaphaisinh.net/` |
| `chiase_cu/index.html` lines 607-622 | Footer | "© 2026 hanghoaphaisinh.net" + 3 links "Điều khoản dịch vụ", "Chính sách bảo mật", "Cảnh báo rủi ro" |
| `chiase_cu/index.html` lines 623-633 | StickyCTA | Mobile + desktop bottom bar |
| `chiase_cu/index.html` lines 634-770 | LeadCaptureModal | Modal popup lead capture |
| `chiase_cu/index.html` lines 771-785 | Scripts | main.js, tracker.js, AOS init |

**Assets phải upload sang MinIO/S3** (giữ tên file gốc, thay path):

| File cũ | Target MinIO | Sizes |
|---------|--------------|-------|
| `chiase_cu/assets/img/logo_APEX_K_NEN.webp` | `rinco-tenant-assets/{tenant_id}/logo.webp` | [16, 32, 64, 128, 256] webp+avif |
| `chiase_cu/anh/anhchandung_chuyengia_nguyentuananh.webp` | `rinco-tenant-assets/{tenant_id}/speaker.webp` | [400, 800, 1200] webp+avif |
| `chiase_cu/anh/apex_top1_thiphan_quy2_2026.webp` | `rinco-tenant-assets/{tenant_id}/top1-banner.webp` | [800, 1600] webp |
| `chiase_cu/assets/css/style.css` | `apps/landing-renderer/styles/legacy.css` | Reference + adapt Tailwind |
| `chiase_cu/assets/js/main.js` | `apps/landing-renderer/lib/legacy/main.ts` | Rewrite TypeScript + React |
| `chiase_cu/assets/js/tracker.js` | `apps/landing-renderer/lib/tracking/rinco-tracker.ts` | Replace bằng official SDK |

**Branding giữ nguyên 100 %**: logo APEX, slogan, hotline `0984386538`, tên công ty "CÔNG TY CỔ PHẦN CÔNG NGHỆ TÀI CHÍNH APEX", website `hanghoaphaisinh.net`. Acceptance `AC-LP-10` yêu cầu manual QA xác nhận 100 % nội dung từ `chiase_cu/` được giữ.

## Acceptance Criteria (20 AC)

| AC | Tiêu chí | Đo |
|----|---------|-----|
| AC-LP-01 | Lighthouse Performance ≥ 95 | Score |
| AC-LP-02 | FCP < 0.4 s | Lighthouse |
| AC-LP-03 | LCP < 0.8 s | Lighthouse |
| AC-LP-04 | CLS < 0.05 | Lighthouse |
| AC-LP-05 | TBT < 100 ms | Lighthouse |
| AC-LP-06 | Form submit → ScyllaDB < 50 ms p95 | Custom metric |
| AC-LP-07 | CAPI success rate ≥ 99.5 % | Metric |
| AC-LP-08 | EMQ ≥ 7 | Meta diagnostic |
| AC-LP-09 | Bot detection ≥ 99 % | Test suite |
| AC-LP-10 | 100 % nội dung `chiase_cu/` được giữ | Manual QA |
| AC-LP-11 | GDPR/PDPA cookie consent | Audit |
| AC-LP-12 | Hybrid Dual-Tracking 100 % match | Conversion validator |
| AC-LP-13 | CAPI call < 500 ms p95 | Custom metric |
| AC-LP-14 | A/B traffic allocation exact | Stats validator |
| AC-LP-15 | Webhook delivery 100 % retry | Delivery report |
| AC-LP-16 | Form submission rate +20 % vs cũ | A/B comparison |
| AC-LP-17 | Cost per tenant < $5 / tháng | Billing |
| AC-LP-18 | RPO < 5 min | DR drill |
| AC-LP-19 | 10 tenants onboard / tuần | Onboarding metric |
| AC-LP-20 | Zero critical security issues | Security scan |

## Implementation Roadmap (12 tuần × 5 FTE = 60 person-weeks)

- **Phase 1 (Tuần 1-4) — Foundation & Migration**: Setup Next.js 15 + Bun + Tailwind v4, migrate 100 % các section từ `chiase_cu/`, tracking SDK MVP, `landing-ingest` Go + Huma + ScyllaDB + NATS, `meta-capi` worker.
- **Phase 2 (Tuần 5-8) — Hardening**: Performance optimization, Lighthouse CI, load test k6 1000 RPS, security audit, A/B framework, i18n (vi/en/ja/ko), OpenTelemetry + Sentry, backup script + DR drill.
- **Phase 3 (Tuần 9-12) — Polish & GA**: A/B variants on Hero/CTA, WCAG 2.1 AA, SEO structured data, penetration test, canary deploy → 100 % rollout.

Team: 1 Frontend Lead, 1 Frontend Dev, 1 Backend Lead, 1 Backend Dev, 1 Bot/Security, 0.5 DevOps, 0.5 QA = 5 FTE.

**Risk Register** (10 rủi ro chính, chọn lọc): Pixel ID leak → RLS + audit; Meta rate-limit → queue + circuit breaker; Wasm quá nặng → code-split; Argon2 PoW mobile chậm → adaptive; GDPR/PDPA → default essential consent; Asset migration lost → SHA-256 verify; Form spam vượt PoW → hCaptcha fallback; EMQ < 6 → audit user_data fields; CAPI Poisoning → HMAC + IP allowlist; Lighthouse regression → CI Lighthouse check.

## Kết luận

Landing Page + CAPI là mặt tiền của RINCO. Khi GA expected: Lighthouse 95+ trên mọi tenant, EMQ ≥ 7 → Meta ads optimization tốt hơn 30 %, bot detection 99%+ → giảm 80 % spam leads, conversion rate +20 % vs landing cũ, cost per tenant < $5 / tháng (vs $37+ SaaS truyền thống).
