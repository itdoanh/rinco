# RINCO Admin Portal

Super-admin console for the RINCO multi-tenant SaaS platform. Manage tenants, users, leads, system health and audit logs from one place.

> Port: **3001** • Admin-only • Sits behind NextAuth (Credentials + SSO providers)

## Stack

- **Next.js 15** App Router
- **React 19** + TypeScript
- **@rinco/ui** shared components
- **Tailwind CSS 3** (CSS variables)
- **TanStack Query** for API client
- **next-auth v5** for authentication (Credentials / GitHub / Google)
- **zustand** UI + tenant state
- **recharts** analytics charts
- **react-hook-form** + **zod** for forms
- **Playwright** for E2E

## Setup

```bash
pnpm install
cp .env.example .env.local      # edit NEXTAUTH_SECRET and API URL
```

## Dev workflow

```bash
pnpm dev            # next dev --port 3001 → http://localhost:3001
pnpm lint
pnpm typecheck
pnpm test:e2e
```

## Build

```bash
pnpm build
pnpm start          # production (port 3001)
```

## Project structure

```
admin-portal/
├── app/
│   ├── layout.tsx                          # Root + providers + auth
│   ├── globals.css                         # Tailwind tokens
│   ├── (auth)/                             # Public auth routes
│   │   ├── layout.tsx
│   │   └── login/page.tsx                  # Login form (Credentials)
│   └── (dashboard)/                        # Auth-required shell
│       ├── layout.tsx                      # Sidebar + Header + Breadcrumb
│       ├── dashboard/page.tsx              # KPI overview (tenants/leads/MRR)
│       ├── tenants/
│       │   ├── page.tsx                    # Tenant list + filters
│       │   └── [id]/page.tsx               # Tenant detail (contact, plan, stats)
│       ├── analytics/page.tsx              # Traffic + conversion chart
│       ├── audit/page.tsx                  # Searchable audit log
│       └── system/
│           ├── health/page.tsx             # Microservices health grid
│           ├── logs/page.tsx               # Live log viewer (LogViewer)
│           └── metrics/page.tsx            # Grafana embed
├── components/
│   ├── auth/AuthProvider.tsx               # next-auth SessionProvider
│   ├── layout/{Sidebar,Header,Breadcrumb}.tsx
│   ├── tenants/{TenantList,TenantForm}.tsx
│   ├── analytics/{MetricCard,Chart}.tsx
│   ├── system/{ServiceHealthGrid,LogViewer}.tsx
│   └── providers.tsx                       # QueryClient + UI providers
├── lib/
│   ├── api.ts                              # Typed apiClient + adminApi.*
│   ├── auth.ts                             # NextAuthOptions
│   └── schema.ts                           # zod schemas + types
├── store/index.ts                          # Zustand stores (admin, analytics, ui)
├── e2e/                                    # Playwright specs
└── playwright.config.ts
```

## Environment

See `.env.example`:

| Var                              | Purpose                              |
|----------------------------------|--------------------------------------|
| `NEXT_PUBLIC_API_URL`            | Backend API base                     |
| `NEXTAUTH_URL`                   | Public URL of portal (callback URL)  |
| `NEXTAUTH_SECRET`                | JWT signing secret (32+ chars)       |
| `GITHUB_CLIENT_ID/SECRET`        | Optional GitHub SSO                  |
| `GOOGLE_CLIENT_ID/SECRET`        | Optional Google SSO                  |
| `NEXT_PUBLIC_GRAFANA_URL`        | Grafana base URL (system/metrics)    |
| `NEXT_PUBLIC_GRAFANA_DASHBOARD_*`| Grafana dashboard IDs                |

## Deployment

```bash
pnpm build
pnpm start
```

Recommended production environment: Node 20+, set `NEXTAUTH_SECRET` to a 32+ char random string, terminate TLS upstream, restrict `/admin` via your network-layer ACL or VPN.
