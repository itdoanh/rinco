# Landing Page

Landing page application for RINCO multi-tenant platform. Built with Next.js 15, React 19, and TypeScript.

## Features

- **Block Rendering Engine**: Dynamic content blocks rendered from API schema
- **Multi-step Forms**: Lead capture with validation
- **Tracking Integration**: Meta Pixel + Facebook CAPI
- **UTM Capture**: Automatic UTM parameter tracking
- **SEO Optimized**: Server-side rendering with metadata
- **Responsive Design**: Mobile-first with Tailwind CSS

## Tech Stack

- **Framework**: Next.js 15 (App Router)
- **Language**: TypeScript 5.6
- **Runtime**: Bun
- **Styling**: Tailwind CSS 4 + shadcn/ui
- **Server State**: TanStack Query v5
- **Forms**: React Hook Form + Zod
- **Icons**: Lucide React
- **Animations**: Tailwind Animate

## Block Types

| Type | Component | Description |
|------|-----------|-------------|
| `hero` | Hero.tsx | Main hero section |
| `feature_grid` | FeatureGrid.tsx | Feature cards grid |
| `logo_cloud` | LogoCloud.tsx | Logo carousel |
| `speaker` | SpeakerSection.tsx | Speaker/profile section |
| `trust` | TrustSection.tsx | Trust badges section |
| `stats` | Stats.tsx | Statistics display |
| `faq` | FAQ.tsx | FAQ accordion |
| `cta` | CTA.tsx | Call to action |
| `testimonial` | Testimonial.tsx | Customer testimonials |
| `risk_warning` | RiskWarning.tsx | Risk disclaimer |
| `form` | FormBlock.tsx | Registration form |

## Form Schema Format

```typescript
{
  fields: [
    {
      name: "name",
      type: "text",
      label: "Họ và tên",
      placeholder: "Nhập họ và tên",
      required: true
    },
    {
      name: "phone",
      type: "phone",
      label: "Số điện thoại",
      required: true
    }
  ]
}
```

## Tracking Setup

### Meta Pixel

1. Set `NEXT_PUBLIC_FB_PIXEL_ID` in environment
2. Pixel is auto-initialized in Tracker component
3. Events tracked: PageView, Lead, ButtonClick, Custom events

### Facebook CAPI

1. Set `CAPI_ACCESS_TOKEN` and `CAPI_PIXEL_ID` in environment
2. Server-side events sent via `/api/capti` route
3. Events: Lead, PageView, custom conversions

### UTM Parameters

UTM parameters are automatically captured from URL and stored in localStorage, then sent with each form submission.

## Environment Variables

```env
# API
NEXT_PUBLIC_API_URL=http://localhost:8080

# Meta Pixel
NEXT_PUBLIC_FB_PIXEL_ID=your_pixel_id

# Facebook CAPI
CAPI_ACCESS_TOKEN=your_access_token
CAPI_PIXEL_ID=your_pixel_id

# Site
NEXT_PUBLIC_SITE_URL=https://your-domain.com
```

## Installation

```bash
bun install
bun dev
```

## Build

```bash
bun build
bun start
```

## Project Structure

```
landing/
├── app/
│   ├── api/
│   │   ├── leads/       # Lead submission proxy
│   │   ├── track/       # Event tracking proxy
│   │   └── capti/       # CAPI proxy
│   ├── globals.css
│   ├── layout.tsx
│   └── page.tsx
├── components/
│   ├── blocks/          # Block components
│   ├── form/            # Form components
│   ├── layout/           # Layout components
│   ├── tracking/         # Tracking components
│   └── seo/             # SEO components
├── lib/
│   ├── api.ts           # API client
│   ├── schema.ts        # Zod schemas
│   ├── tracking.ts      # Tracking utilities
│   ├── pixel.ts         # Meta Pixel wrapper
│   └── capti.ts         # CAPI utilities
├── hooks/
│   ├── useFormSubmit.ts
│   ├── useTracking.ts
│   ├── usePixel.ts
│   └── usePageView.ts
└── public/
    └── assets/
```
