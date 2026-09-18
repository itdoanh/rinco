# WS-T Issue: Tenant site `/demo` page could not be submitted to backend

## Repro steps

1. Navigate to `http://localhost:3002/demo`
2. Click "Xem demo" on any tenant card (e.g. Apex Fintech)
3. Tenant page loads with mock data

## Expected
- Real backend fetch via `tenantApi.getPage()` returns API data when available
- When backend not running, falls back to mock data

## Actual
- ✅ Mock fallback works correctly
- ✅ Page renders with full content
- ❌ Backend is not running, so all tenant pages serve from `frontend/tenant-site/lib/mock-data.ts`

## Root cause
The `mock-data.ts` file IS intentional graceful fallback, but with no backend running, all tenant pages are mocked.

## Fix recommendation
1. Start backend services: `docker compose -f infra/docker-compose.yml up -d && docker compose -f infra/docker-compose.services.yml up -d`
2. Run `make migrate` to seed tenant data
3. WS-T will then verify real API responses

## Impact
- **Demo flow works**: users can browse tenant pages
- **Production not impacted**: real backend will serve real data
- **WS-T scope**: documented as expected behaviour

## Discovered by
- WS-T subagent, 2026-09-18
- Severity: **P3** (cosmetic / documentation issue)
- Effort: N/A — wait for backend to come up