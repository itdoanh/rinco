# ADR-0008: Frontend Multi-app Strategy

> **Status**: Accepted · **Date**: 2026-09-18 · **Authors**: RINCO Platform Team

## Context

RINCO có **4 frontend apps** với yêu cầu rất khác nhau:

1. **landing** (port 3000) — Marketing landing page, block renderer, SEO critical, FB tracking.
2. **admin-portal** (port 3001) — Super admin (Dark Admin), high security, internal-only.
3. **tenant-site** (port 3002) — Multi-tenant site renderer (dynamic blocks per tenant).
4. **meeting-ui** (port 3003) — WebRTC meeting UI, real-time, heavy client-side state.

Yêu cầu:

- **Independent deploy** — landing không bị block khi admin-portal fail.
- **Different auth context** — admin có YubiKey 2FA, user có email/password, meeting có JWT ngắn hạn.
- **Different CDN strategy** — landing qua Cloudflare, admin chỉ qua WireGuard.
- **Different build optimization** — landing prioritize SEO, meeting prioritize realtime.
- **Shared component library** — UI components (Button, Card, Modal) dùng chung.
- **Code reuse** — không duplicate logic (auth helper, API client, theme tokens).

### Lựa chọn: Monorepo hay Multi-repo?

| Approach | Pros | Cons |
|----------|------|------|
| **Monorepo (Turborepo / Nx)** | Shared code dễ, atomic changes, 1 PR cross-app | Tooling phức tạp, build time lâu nếu scale |
| **Multi-repo** | Independent ownership | Code reuse khó, version drift, nhiều CI configs |
| **Polyrepo + shared packages** (npm) | Vừa shared vừa independent | Một số trade-off giữa 2 cách trên |

### Lựa chọn framework: Next.js vs Vite SPA vs Remix?

| Approach | Pros | Cons |
|----------|------|------|
| **Next.js App Router** | SSR/SSG, file routing, Edge support, mature | Build chậm, opinionated |
| **Vite SPA + React Router** | Build nhanh, đơn giản | SEO kém, không có SSR |
| **Remix** | Data loading tốt | Ecosystem nhỏ hơn Next.js |

## Decision

### Repository layout: **Monorepo với 4 apps + 2 shared packages**

```
frontend/
├── apps/
│   ├── landing/         # port 3000 (Next.js 15)
│   ├── admin-portal/    # port 3001 (Next.js 15, internal-only)
│   ├── tenant-site/     # port 3002 (Next.js 15, dynamic routes)
│   └── meeting-ui/      # port 3003 (Next.js 15, WebRTC-heavy)
├── packages/
│   └── ui/              # React component library (shadcn/ui)
└── e2e/                 # Playwright shared
```

Dùng **Bun workspace** (file `package.json` ở root) thay vì Turborepo / Nx — đơn giản, nhanh.

### Framework: Next.js 15 (App Router) cho tất cả 4 apps

Lý do:
- **SSR/SSG** cho landing (SEO quan trọng).
- **Middleware.ts** cho route protection (vd: admin-portal kiểm tra NextAuth session).
- **API routes** cho BFF pattern (vd: landing forward request tới backend với proper headers).
- **React Server Components** giảm JS bundle size.
- **Single ecosystem** — 1 framework, 4 apps → ít context switch cho dev.

Trade-off: meeting-ui có thể dùng Vite SPA sẽ nhẹ hơn — nhưng Next.js vẫn đủ nhanh và unified stack tốt hơn.

### Shared packages

#### `packages/ui/` — Component library

- 25+ components từ shadcn/ui (Button, Card, Dialog, Form, Table, …).
- Tailwind preset + theme tokens (`bg-primary`, `text-foreground`, …).
- `globals.css` (CSS variables cho dark/light mode).
- TypeScript types + barrel export (`index.ts`).
- Build: `tsup` → output ESM + CJS + `.d.ts`.

Mỗi app import:

```ts
// app/landing/app/page.tsx
import { Button, Card } from "@rinco/ui";
import { API_BASE_URL } from "@rinco/ui/config";
```

#### (Future) `packages/api-client/` — API client generator

- OpenAPI specs (`docs/api/*.yaml`) → generate TypeScript client.
- Per-service: `authClient`, `crmClient`, `chatClient`.
- Singleton instance per app.

### Authentication per app

| App | Auth strategy |
|-----|---------------|
| **landing** | Public (no auth, optional lead capture) |
| **admin-portal** | NextAuth v5 + YubiKey (FIDO2) + IP whitelist |
| **tenant-site** | Email/password + 2FA TOTP + session cookie |
| **meeting-ui** | Short-lived JWT (15 min) from auth-service, refresh in-meeting |

### Build optimization per app

| App | Strategy |
|-----|----------|
| **landing** | Static export cho marketing pages; dynamic cho `/[tenant]/[page]`. ISR với `revalidate=60`. |
| **admin-portal** | SSR + cache `no-store`. Bundle splitting lớn (charts, tables). |
| **tenant-site** | SSR + tenant-specific bundle (không có tenant → 404). |
| **meeting-ui** | SPA mode (CSR), preload WebRTC adapter, lazy load chat panel. |

### Deploy

- Mỗi app = 1 Docker image riêng (`landing`, `admin-portal`, …).
- Helm chart `frontend/<app>` trong `deployments/helm/rinco/`.
- Independent K8s Deployment + Service + Ingress.
- CI build song song 4 apps (matrix strategy).

```
frontend/
├── landing/Dockerfile       → GHCR: ghcr.io/itdoanh/rinco-landing:1.1.0
├── admin-portal/Dockerfile  → GHCR: ghcr.io/itdoanh/rinco-admin:1.1.0
├── tenant-site/Dockerfile   → GHCR: ghcr.io/itdoanh/rinco-tenant:1.1.0
└── meeting-ui/Dockerfile    → GHCR: ghcr.io/itdoanh/rinco-meeting:1.1.0
```

### Routing isolation

Mỗi app có domain riêng:

| App | Production URL |
|-----|----------------|
| landing | `https://rinco.vn` (and tenant custom domains `https://{tenant}.rinco.vn`) |
| admin-portal | `https://admin.rinco.internal:3001` (WireGuard only) |
| tenant-site | `https://{tenant}.rinco.app` |
| meeting-ui | `https://meet.rinco.vn` |

### Shared dependencies versions

- React 19, Next.js 15, TypeScript 5.6.
- Tailwind CSS 3.4 với preset shared.
- shadcn/ui CLI dùng config ở `packages/ui/components.json`.
- ESLint + Prettier config ở root (extends Next.js + shared rules).

### Why not Turborepo / Nx?

- **Tooling complexity**: Turborepo cần `turbo.json`, pipelines, caching strategy.
- **Bun workspace** đã đủ: `bun install` ở root → install mọi package.
- **Bun's built-in features**: workspaces, test runner, script runner.
- **Build cache**: nếu cần, dùng `bunfig.toml` + manual cache.

Nếu scale lên 10+ apps thì cân nhắc Turborepo. Hiện tại 4 apps → Bun đủ.

### Why not Vite SPA cho meeting-ui?

- WebRTC adapter ~50KB, large state.
- Vite SPA build nhanh hơn ~30% so với Next.js.
- Nhưng Next.js cung cấp API routes (BFF) → forward auth check trước khi join meeting.
- Trade-off: tăng 100KB bundle để có auth check ở edge.

## Consequences

### Positive

- **Single framework** — 1 stack, 4 apps.
- **Code reuse** qua `packages/ui` (~30% code reuse).
- **Independent deploy** — meeting-ui fail không ảnh hưởng landing.
- **Different auth** — không lẫn lộn admin session với user session.
- **Different CDN/security** — admin chỉ qua WireGuard, landing qua Cloudflare.
- **Type safety** — TypeScript end-to-end.
- **SEO tốt** cho landing/tenant-site qua SSR/SSG.
- **Atomic changes** — fix bug trong shared component → tất cả apps update.

### Negative

- **Build time** Next.js chậm hơn Vite.
- **Next.js opinionated** — đôi khi gặp giới hạn (vd: WebSocket route).
- **Bundle size** Next.js runtime ~80KB baseline.
- **Lock-in**: khó migrate sang framework khác.
- **4 apps × N versions** = version management phức tạp.

### Mitigations

- **Incremental Static Regeneration** (ISR) cho landing — build nhanh, revalidate sau.
- **Standalone output mode** (`output: 'standalone'`) → Docker image nhỏ.
- **Bundle analyzer** `@next/bundle-analyzer` CI step.
- **Renovate bot** auto-update dependencies, batch PR mỗi tuần.

## Alternatives Considered

### A. Monolith Next.js app (1 codebase, 4 sub-routes)

- **Pro**: shared state, shared deps, đơn giản.
- **Con**: 1 app fail = tất cả fail; bundle lớn (mọi route load chung).
- **Verdict**: ❌ Rejected — security boundary giữa admin/user không được tách.

### B. Multi-repo (4 Git repos)

- **Pro**: ownership rõ ràng.
- **Con**: code reuse khó, version drift, nhiều CI.
- **Verdict**: ❌ Rejected — overhead không đáng.

### C. Vite SPA + React Router cho tất cả

- **Pro**: build nhanh, simple.
- **Con**: SEO kém cho landing (đây là ưu tiên số 1).
- **Verdict**: ❌ Rejected.

### D. Remix framework

- **Pro**: data loading tốt.
- **Con**: ecosystem nhỏ hơn Next.js, ít dev quen.
- **Verdict**: 🟡 Cân nhắc — nhưng Next.js đủ tốt.

## References

- [Next.js 15 documentation](https://nextjs.org/docs)
- [Bun workspaces](https://bun.sh/docs/install/workspaces)
- [services/meeting-ui](../../services/meeting-ui)
- [packages/frontend/ui](../../packages/frontend/ui) (or `frontend/packages/ui`)
- [ARCHITECTURE.md §2.1](../ARCHITECTURE.md#21-inter-service-communication)
- [ADR-0002 PASETO](0002-paseto-vs-jwt.md)