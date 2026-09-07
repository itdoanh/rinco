# Admin Portal

Super Admin Portal for RINCO Multi-tenant SaaS Platform.

## Features

- **Dashboard**: Real-time KPIs, charts, and activity feed
- **Tenant Management**: CRUD operations, suspend/activate, domain management
- **User Management**: Global users list with role-based access
- **Analytics**: Leads, traffic, and conversion tracking
- **System Health**: Service status monitoring with latency and error rates
- **Log Viewer**: Centralized log aggregation with filtering
- **Audit Logs**: Complete audit trail of admin actions

## Tech Stack

- **Framework**: Next.js 15 (App Router)
- **Language**: TypeScript 5.6
- **Runtime**: Bun
- **Styling**: Tailwind CSS 4 + shadcn/ui
- **State Management**: Zustand
- **Server State**: TanStack Query v5
- **Auth**: NextAuth v5
- **Charts**: Recharts
- **Forms**: React Hook Form + Zod

## Getting Started

```bash
bun install
bun dev
```

Access at http://localhost:3001

## Pages

| Path | Description |
|------|-------------|
| `/login` | Admin login |
| `/dashboard` | Overview with KPIs |
| `/tenants` | Tenant management |
| `/users` | User management |
| `/analytics` | Analytics overview |
| `/analytics/leads` | Lead analytics |
| `/analytics/traffic` | Traffic analytics |
| `/system/health` | Service health grid |
| `/system/logs` | Log viewer |
| `/system/metrics` | System metrics |
| `/audit` | Audit logs |
| `/settings` | Platform settings |

## Environment Variables

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXTAUTH_SECRET=your-secret
NEXTAUTH_URL=http://localhost:3001
```
