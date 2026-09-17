# RINCO Demo Seed Expansion (Loop WS-B)

This folder contains idempotent expansion seeds that add extra demo data on top
of the base `migrations/seed/00_master.sql` run.

## Files

| File | Purpose | Tenant(s) |
| --- | --- | --- |
| `01_leads_extra.sql` | 550+ extra leads (250 Apex, 200 HCT, 100 Demo) with Vietnamese names, diverse UTMs, custom fields, predicted LTV | apex, hct, demo |
| `02_deals_activities_extra.sql` | 150 deals + 630 activities + 60 notes + 200 stage-history entries | apex, hct, demo |
| `03_users_crm_tree_extra.sql` | 18 users (Manager/Team Lead/Senior/Junior/Viewer/Intern) + 5 sub-departments + 15 companies + 360 legacy contacts | apex, hct, demo |
| `04_workflows_audit_extra.sql` | 14 workflows + 23 dynamic schema fields + 120+ workflow executions + 360 audit logs + 1000+ API request logs | apex, hct, demo |
| `05_notifications_chat_extra.sql` | 13 chat channels + 1050+ chat messages + 35 meeting rooms + 24 tags + 27 custom fields + 700+ notifications | apex, hct, demo |

## How to run

### Manual
```bash
PGPASSWORD=rinco_dev_password psql -h localhost -p 5433 -U rinco -d rinco \
  -v ON_ERROR_STOP=0 -f expansion/01_leads_extra.sql

PGPASSWORD=rinco_dev_password psql -h localhost -p 5433 -U rinco -d rinco \
  -v ON_ERROR_STOP=0 -f expansion/02_deals_activities_extra.sql
# ... etc
```

### Via master
`00_master.sql` now automatically includes all expansion files after section 99.
Run the master file once and you're done.

### Via Linux script
```bash
bash scripts/seed-all.sh
```
The orchestrator auto-detects the `expansion/` folder.

## Idempotency

Every `INSERT` uses `ON CONFLICT (id) DO NOTHING` so re-running is safe.
This is intentional — the files are designed to be runnable in CI or dev loops.

## Tenant IDs

```sql
'apexfintech'  -> 'aaaaaaaa-0000-0000-0000-000000000001'
'hct-consulting' -> 'bbbbbbbb-0000-0000-0000-000000000002'
'demo-company'   -> 'cccccccc-0000-0000-0000-000000000003'
```
