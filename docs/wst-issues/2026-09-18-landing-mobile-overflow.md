# WS-T Issue: Landing page horizontal overflow on mobile/tablet

## Repro steps

1. Start dev servers:
   ```
   cd C:\code\RINCO\frontend
   cd landing && npm run dev &
   cd tenant-site && npm run dev &
   ```

2. Open `http://localhost:3000/` in Chrome
3. Open DevTools → toggle device toolbar → iPhone SE (375x667)
4. Scroll horizontally — body scrolls ~100px to the right
5. Switch to iPad (768x1024) — body scrolls ~103px

## Expected
- No horizontal scroll on any standard viewport (320px – 2560px wide)
- Hero block responsive

## Actual
- 100px horizontal overflow on iPhone SE (375x667)
- 103px horizontal overflow on iPad (768x1024)
- Desktop 1440x900 OK

## Root cause (suspected)
- Block content (likely the wide `defaultBlocks` data in `frontend/landing/app/page.tsx`)
- The `feature_grid` block with 4 columns of icon+text may be forcing min-width
- Speaker block image may be unconstrained

## Fix recommendation
```tsx
// frontend/landing/app/page.tsx
<main className="min-h-screen overflow-x-hidden">
  ...
</main>

// OR per-block:
<div className="overflow-x-hidden">
  {heroBlock && <BlockRenderer blocks={...} />}
</div>
```

## Screenshots
- `frontend/e2e/screenshots/responsive/iphone-se-landing.png`
- `frontend/e2e/screenshots/responsive/ipad-landing.png`

## Discovered by
- WS-T subagent, 2026-09-18
- Test: `tests/responsive.spec.ts` — "Responsive mobile (iPhone SE) › landing homepage renders"
- Severity: **P2** (UX issue, no functional impact)
- Effort: ~30 min CSS fix