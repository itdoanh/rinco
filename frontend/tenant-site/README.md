# RINCO Tenant Site

Dynamic, multi-tenant customer websites. Each tenant gets its own branding (colors, logo, font) and pages rendered from server-defined blocks.

> Port: **3002** • Per-tenant dynamic routes `/[tenant]` and `/[tenant]/[page]`

## Stack

- **Next.js 15** App Router (server components for SSR pages)
- **React 19** + TypeScript
- **@rinco/ui** shared components
- **Tailwind CSS 3**
- **TanStack Query** (client-side interactivity)
- **react-hook-form** + **zod** validation
- **Playwright** for E2E

## Setup

```bash
pnpm install
cp .env.example .env.local
```

## Dev workflow

```bash
pnpm dev            # next dev --port 3002 → http://localhost:3002
pnpm lint
pnpm typecheck
pnpm test:e2e
```

## Build

```bash
pnpm build
pnpm start          # production (port 3002)
```

## Project structure

```
tenant-site/
├── app/
│   ├── layout.tsx                         # Root + Providers
│   ├── globals.css                        # Tailwind tokens
│   ├── [tenant]/page.tsx                  # Tenant homepage (renders blocks)
│   ├── [tenant]/[page]/page.tsx           # Tenant sub-page
│   └── api/leads/route.ts                 # POST: capture lead (tenant-scoped)
├── components/
│   ├── branding/{Header,Footer}.tsx       # Tenant-themed chrome
│   ├── renderer/
│   │   ├── PageRenderer.tsx               # Maps blocks → components
│   │   └── ContactForm.tsx                # Per-tenant contact form
│   ├── providers.tsx                      # QueryClient + Tooltip
│   └── ui/                                # Local wrappers
└── lib/
    ├── api.ts                             # tenantApi (typed)
    ├── branding.ts                        # useBranding hook + hex→hsl
    └── schema.ts                          # zod schemas + Tenant/Page/Block types
```

## Environment

See `.env.example`.

| Var                          | Purpose                              |
|------------------------------|--------------------------------------|
| `NEXT_PUBLIC_API_URL`        | Backend API base (tenant + leads)     |
| `NEXT_PUBLIC_SITE_URL`       | Public site URL (default 3002)       |
| `NEXT_PUBLIC_DEFAULT_TENANT` | Default slug when none provided      |

## Tenant branding

Each tenant can override:

- `primary`, `secondary`, `accent` — color palette (hex)
- `logoUrl`, `favicon` — assets
- `fontFamily` — typography

`useBranding()` injects these as CSS variables on mount, so child components
read `var(--brand-primary)` etc. to remain visual-consistent.

## Deployment

```bash
pnpm build
pnpm start
```

In production, the site is typically behind a wildcard host router that
maps `*.rinco.app` → this app, with `host`-based slug extraction in
middleware (not included in this scaffold).
