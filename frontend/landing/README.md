# RINCO Landing — Public Marketing Site

Next.js 15 marketing site for RINCO. Public, tenant-aware, conversion-optimized landing pages with full tracking (FB Pixel, Google Ads, UTM, click events) and lead capture.

> Port: **3000** • Public • Tenant routes at `/[tenant]` and `/[tenant]/[page]`

## Stack

- **Next.js 15** App Router + TypeScript
- **React 19** with **Server Components**
- **@rinco/ui** shared component library
- **Tailwind CSS 3** (CSS variables, design tokens)
- **shadcn-style** UI primitives (Radix UI under the hood)
- **TanStack Query** for client-side data fetching
- **react-hook-form** + **zod** for validated forms
- **Playwright** for E2E

## Setup

```bash
# From repo root
pnpm install

# Or from this app
cd frontend/landing
pnpm install

# Optional: copy and edit env
cp .env.example .env.local
```

## Dev workflow

```bash
pnpm dev              # next dev (http://localhost:3000)
pnpm lint             # next lint
pnpm typecheck        # tsc --noEmit
pnpm test:e2e         # playwright test
```

## Build

```bash
pnpm build
pnpm start            # production (http://localhost:3000)
```

## Project structure

```
landing/
├── app/                          # App Router
│   ├── layout.tsx                # Root, font + tracker + providers
│   ├── page.tsx                  # Default homepage
│   ├── [tenant]/page.tsx         # Tenant homepage
│   ├── [tenant]/[page]/page.tsx  # Tenant sub-pages
│   ├── globals.css               # Tailwind layer + design tokens
│   └── api/
│       ├── leads/route.ts        # POST: capture lead
│       ├── track/route.ts        # POST: track page-view / event
│       ├── capi/route.ts         # POST: FB CAPI server-side
│       └── capti/route.ts        # POST: captcha verify
├── components/
│   ├── blocks/                   # 12+ landing-page blocks
│   │   ├── Hero.tsx
│   │   ├── FeatureGrid.tsx
│   │   ├── LogoCloud.tsx
│   │   ├── Testimonial.tsx
│   │   ├── FAQ.tsx
│   │   ├── FormBlock.tsx
│   │   ├── CTA.tsx
│   │   ├── PricingTable.tsx
│   │   ├── Stats.tsx
│   │   ├── SpeakerSection.tsx
│   │   ├── TrustSection.tsx
│   │   ├── RiskWarning.tsx
│   │   └── BlockRenderer.tsx     # switch-case render by block type
│   ├── layout/                   # Header, Footer, LeadModal, StickyCTA
│   ├── form/                     # DynamicForm.tsx + FormBlock.tsx
│   ├── tracking/                 # Tracker, UTMCapture, PixelInit, ClickTracker
│   ├── seo/                      # SEOHead + SchemaMarkup
│   ├── providers.tsx             # QueryClient + Tooltip + Toaster
│   └── ui/                       # App-local wrappers (alias)
├── hooks/                        # usePageView, useFormSubmit, usePixel, useLeadModal
├── lib/                          # api.ts, schema.ts, tracking.ts, pixel.ts, capti.ts
├── e2e/                          # Playwright specs
├── playwright.config.ts
├── tailwind.config.ts
├── next.config.js                # next-intl + image domains
├── tsconfig.json
└── package.json
```

## Environment variables

See `.env.example`.

| Var                              | Purpose                                  |
|----------------------------------|------------------------------------------|
| `NEXT_PUBLIC_API_BASE_URL`       | Backend API base (lead-svc)              |
| `NEXT_PUBLIC_TENANT_DEFAULT`     | Default tenant slug when `/` is hit      |
| `NEXT_PUBLIC_SITE_URL`           | Public site URL (used by SEO + OG tags)  |
| `NEXT_PUBLIC_FB_PIXEL_ID`        | Facebook Pixel ID                        |
| `NEXT_PUBLIC_GOOGLE_ADS_ID`      | Google Ads conversion ID                 |
| `NEXT_PUBLIC_GTM_ID`             | Google Tag Manager container ID          |
| `NEXT_PUBLIC_CAPTCHA_SITE_KEY`   | Captcha site key                         |
| `CAPTCHA_SECRET`                 | Captcha secret (server)                  |

## Deployment

Any Node host (Vercel, Render, Fly.io, DigitalOcean) works.

```bash
pnpm build
pnpm start          # listens on :3000
```

**Docker-ready:** add a 3-line Dockerfile (node:20-alpine, build, start).
