# WS-D — Real Data & Backend Integration

Mission: Replace placeholder seed data with **production-realistic** Vietnamese seed data so every frontend screen shows real-feeling content. Iterate in atomic Conventional Commits on `main`.

## Deliverables

| File | Purpose |
|------|---------|
| `logs/wsd-inventory.txt` | Phase 1: complete seed file inventory |
| `logs/wsd-consistency.txt` | Phase 3: cross-backend consistency report |
| `logs/wsd-frontend-data.txt` | Phase 4: frontend data verification |
| `logs/wsd-blocked.txt` | (if applicable) blocked backend log |
| `migrations/seed/expansion/06_ws_d_tenants.sql` | 2 new tenants (Vinamilk Dist, VNG Corp) |
| `migrations/seed/expansion/07_ws_d_users_extra.sql` | 36 users (Apex+HCT+Demo+Vinamilk) |
| `migrations/seed/expansion/07b_ws_d_vng_users.sql` | 35 VNG users |
| `migrations/seed/expansion/08_ws_d_leads_18mo.sql` | 400 leads spanning 540 days |
| `migrations/seed/expansion/09_ws_d_activities_extra.sql` | 175 deals + 1500 activities |
| `seed/external/clickhouse/expansion/09_ws_d_events_18mo.sql` | 12500 events + 1000 sessions + 500 conversions |
| `seed/external/scylla/expansion/09_ws_d_chat_extra.sql` | 95 conversations across 5 tenants |
| `seed/external/mongo/expansion/09_ws_d_landing_extra.js` | 6 Vietnamese landing pages + 200 enrichment cache + 8 forms |
| `scripts/wsd-refresh.ps1` | Nightly refresh (5 leads, 2 deals, 50 activities per tenant) |
| `scripts/wsd-schedule.ps1` | Windows Task Scheduler registration |

## Quick start

```powershell
# Apply WS-D seeds
psql -h localhost -U rinco -d rinco -f migrations/seed/00_master.sql

# Schedule nightly refresh
pwsh scripts/wsd-schedule.ps1

# Run a refresh now (all tenants, dry run first)
pwsh scripts/wsd-refresh.ps1 -DryRun
pwsh scripts/wsd-refresh.ps1 -TenantSlug vng-corp
```

## Realism principles

1. **Names** — Real Vietnamese names (Nguyễn Văn An, Trần Thị Bình). Built from 29 surname × 20 middle × 67 given arrays.
2. **Phone** — `+84 9X XXXX XXX` for mobile, `+84 24/28 XXX XXXX` for landline.
3. **Emails** — Personal `gmail.com/outlook.com/yahoo.com` or work `@<company-domain>.vn` (vinamilk.com.vn, vng.com.vn, vpbank.com.vn, etc.).
4. **Companies** — Vinamilk, FPT, VNG, VPBank, Techcombank, Masan, Vietjet, Tiki, MoMo, Hòa Phát, Sun Group, Vingroup, CellphoneS, Highlands, Phúc Long, Bách Hóa Xanh, etc.
5. **Addresses** — `Số 12 Nguyễn Huệ, Phường Bến Nghé, Quận 1, TP.HCM` style.
6. **Currency** — VND with realistic amounts (15M-50M for SaaS, 2B-50B for BDS, 50M-1B for enterprise).
7. **Timestamps** — Spread across 540 days (18 months) with seasonal spikes: Tết (Feb), mid-year (Jun), Black Friday (Nov), year-end (Dec).
8. **Status mix** — 30% New, 25% Contacted, 20% Qualified, 15% Proposal, 10% Won (per lead); similar for deals.

## Tenant coverage

| Slug | Industry | Users | Leads | Deals | Activities | Created |
|------|----------|-------|-------|-------|------------|---------|
| apexfintech | Fintech | 27 | 165 | 54 | 1035 | 120d |
| hct-consulting | Real Estate | 30 | 145 | 44 | 940 | 60d |
| demo-company | Multi | 30 | 110 | 20 | 200 | 10d |
| vinamilk-dist | FMCG Distribution | 30 | 80 | 35 | 250 | 540d |
| vng-corp | Internet/Software | 35 | 90 | 50 | 400 | 365d |

## Notes

- All INSERTs idempotent (ON CONFLICT DO NOTHING / upsert).
- All expansion files wired into `migrations/seed/00_master.sql`.
- Never delete existing data; only add.
- All commits are atomic + Conventional Commits on `main`.