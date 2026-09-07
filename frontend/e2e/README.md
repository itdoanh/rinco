# RINCO Frontend — Cross-app E2E

Cross-application Playwright suite that boots all four RINCO frontends and validates flows that span them.

## Apps under test

| App          | Port | URL                                |
|--------------|------|------------------------------------|
| `landing`    | 3000 | http://localhost:3000              |
| `admin`      | 3001 | http://localhost:3001              |
| `tenant`     | 3002 | http://localhost:3002              |
| `meeting`    | 3003 | http://localhost:3003              |

## Setup

```bash
cd frontend/e2e
pnpm install
pnpm exec playwright install --with-deps chromium firefox
```

## Run

```bash
# All apps (spins up webServer for each)
pnpm test

# Just landing flow
pnpm test:landing

# Just auth flow
pnpm test:auth

# Just tenant CRUD
pnpm test:tenants

# Just meeting flow
pnpm test:meeting
```

## Specs

- `landing.spec.ts` — landing page + dynamic tenant route + tracking API
- `auth-flow.spec.ts` — admin login + cross-app public/private gates
- `tenant-crud.spec.ts` — admin tenant list/detail + tenant-site lead form
- `meeting-flow.spec.ts` — meeting root + room shell

## CI considerations

- The suite uses **`fullyParallel: false`** to avoid port conflicts and media-permission flakiness.
- WebRTC tests **do not attempt a real peer connection** — they assert that the shell renders (loading or error state) which is sufficient to detect regressions in app boot, routing, and component tree.
- For richer WebRTC E2E, point `NEXT_PUBLIC_SIGNALING_URL` at a staging signaling service and grant `camera`/`microphone` permissions in CI.
