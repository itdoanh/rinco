# Runbook: RLS Bypass / Cross-Tenant Data Leak

> **Severity**: P0 (Critical Issue #1) · **On-call response**: < 5 phút · **Owner**: Security + DBA
>
> ⚠️ **Đây là incident bảo mật nghiêm trọng nhất. Mọi cross-tenant query phải treated as data breach cho đến khi scope được xác định.**

## Symptoms

- Alert: `cross_tenant_attempt` (rows returned > baseline × 5).
- Alert: `rls_policy_bypass` (query executed by role with `BYPASSRLS`).
- Log: query với `app.tenant_id` = empty string (RLS defensive check).
- User report: "Tôi thấy data của tenant khác".
- ai-sre incident: `INC-XXX — Possible cross-tenant data leak`.

## First 5 minutes — STOP THE BLEED

### Immediate actions (< 1 phút)

```bash
# 1. Set tenant to maintenance mode (refuse all API traffic)
# Dùng feature flag override (qua admin-portal hoặc API):
curl -X POST http://auth-service:8081/v1/admin/emergency-mode \
  -H "Authorization: Bearer <super-admin-token>" \
  -d '{"mode": "lockdown", "reason": "RLS incident INC-XXX"}'

# 2. Page security team
# Telegram: @rinco_security (P0 hotline)
```

Trong khi đó:

- **KHÔNG** rollback code ngay (mất forensic).
- **KHÔNG** drop table / truncate.
- **BẬT** audit log verbose (capture mọi query sau đó).
- **Capture** toàn bộ logs từ start-of-incident đến now.

### Confirm scope (< 5 phút)

```sql
-- Tìm query với empty tenant_id (defensive policy trigger)
SELECT
    application_name,
    client_addr,
    query,
    query_start,
    state
FROM pg_stat_activity
WHERE
    -- empty setting = RLS policy returns FALSE = should never have result
    query LIKE '%app.tenant_id%'
    OR current_setting('app.tenant_id', true) = ''
LIMIT 100;

-- Check connection pool "stickiness"
-- (connection that was used by tenant X and then tenant Y)
SELECT * FROM auth.audit_log
WHERE action IN ('rls_defensive_trigger', 'cross_tenant_attempt')
ORDER BY created_at DESC
LIMIT 100;
```

### Identify affected tenants

```sql
-- Tenant nào đã query bị leak?
SELECT DISTINCT
    tenant_id,
    user_id,
    count(*) AS query_count
FROM auth.audit_log
WHERE
    created_at > now() - interval '1 hour'
    AND action IN ('cross_tenant_attempt', 'rls_violation')
GROUP BY tenant_id, user_id;
```

## Common causes & fixes

### A. RLS Stickiness (Connection pool không reset)

**Root cause**: connection trong pool vẫn giữ `app.tenant_id` của request trước.

**Fix**:

```go
// packages/go/db/pool.go — verify AfterRelease hook
pool, _ := pgxpool.New(ctx, dsn, pgxpool.WithAfterRelease(
    func(conn *pgx.Conn) bool {
        // Defensive reset
        _, _ = conn.Exec(ctx, "SELECT set_config('app.tenant_id', '', false)")
        _, _ = conn.Exec(ctx, "DISCARD ALL")
        return true
    },
))
```

Verify bằng canary test:

```bash
# Run canary (xem tests/integration/rls_stickness_test.go)
cd services/auth-service
go test -run TestRLSStickiness -v
```

Nếu test fail → không rollback production cho đến khi fix verified.

### B. Service accidentally dùng `BYPASSRLS` role

**Root cause**: code chạy với role có `BYPASSRLS` privilege (vd: `rinco_admin` cho migrations).

**Fix**:

```sql
-- Xác nhận roles của mọi user
SELECT rolname, rolsuper, rolbypassrls FROM pg_roles;

-- Revoke BYPASSRLS từ roles không cần
REVOKE BYPASSRLS FROM rinco_app;  -- example role

-- Application user CHỈ được dùng role không có BYPASSRLS
-- Đảm bảo app DSN dùng user `rinco_app`, không phải `rinco_admin`
```

### C. Missing RLS policy trên table mới

**Root cause**: dev tạo table mới, quên enable RLS.

```sql
-- List tables không có RLS
SELECT
    schemaname || '.' || tablename AS table_name,
    rowsecurity AS rls_enabled
FROM pg_tables
WHERE
    schemaname NOT IN ('pg_catalog', 'information_schema')
    AND schemaname IN (
        'auth', 'tenant', 'crm', 'lead', 'dynamic_model',
        'email', 'notification', 'lead_scoring', 'meta_capi',
        'billing', 'search'
    )
ORDER BY rls_enabled, table_name;

-- Nếu rls_enabled = false → ENABLE ngay
ALTER TABLE <schema>.<table> ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON <schema>.<table>
    USING (
        current_setting('app.tenant_id', true) != ''
        AND tenant_id = current_setting('app.tenant_id')::UUID
    );
```

### D. `WITH CHECK` policy bị thiếu

**Symptom**: SELECT filter OK, nhưng INSERT/UPDATE không block.

```sql
-- Verify policy có cả USING và WITH CHECK
SELECT
    schemaname, tablename, policyname,
    cmd, qual, with_check
FROM pg_policies
WHERE schemaname = 'crm';
```

Fix: thêm `WITH CHECK` cho mọi INSERT/UPDATE/DELETE policy.

## After containment (< 30 phút)

### Notify affected tenants

```bash
# Gửi notification qua notification-service (internal incident template)
curl -X POST http://notification-service:8088/v1/internal/incident-notify \
  -H "Authorization: Bearer <super-admin-token>" \
  -d '{
    "severity": "critical",
    "title": "Security incident — investigation in progress",
    "affected_tenants": [...],
    "body": "We are investigating a potential security incident. Your data integrity is our priority. Update within 24 hours."
  }'
```

**Tuân thủ**: tùy thuộc vào quy định địa phương (GDPR, NĐ 13/2023 về bảo vệ dữ liệu VN), có thể phải notify regulator trong 72h.

### Forensic collection

```bash
# Snapshot database state
pg_dump --schema-only --no-owner > forensic-schema-$(date +%s).sql
pg_dump --data-only --table=auth.audit_log --no-owner > forensic-audit.sql

# Capture connections snapshot
psql -c "SELECT pg_stat_activity" > forensic-pgstat-$(date +%s).txt

# Snapshot NATS messages
nats stream backup RINCO_EVENTS > forensic-nats-$(date +%s).tar.gz

# Snapshot logs
kubectl logs -n rinco -l app=<service> --since=2h > forensic-app-$(date +%s).log
```

Lưu vào S3 bucket `rinco-forensic` (encrypted, access log).

### Patch và verify

1. Identify exact code/config causing leak.
2. Patch (ví dụ: add RLS policy, fix `WithTenant` wrapper usage).
3. Canary test pass 100% trong CI trước khi merge.
4. Deploy với **progressive rollout** (1% → 10% → 100%) + monitoring.
5. Verify trong production: alert `cross_tenant_attempt` không fire.

## Disclosure

| Audience | When | Channel |
|----------|------|---------|
| Internal eng team | < 15 phút | Slack `#incidents` |
| Affected tenants | < 24 giờ | Email + in-app banner |
| Public (nếu > 100 tenants affected) | < 72 giờ | Blog post + status page |
| Regulator (NĐ 13/2023 VN, GDPR nếu EU) | < 72 giờ | Formal report |

## Postmortem (within 7 ngày)

- Root cause analysis (5 whys).
- Action items với owner + due date.
- Update [ADR-0005 RLS Stickiness](../adr/0005-rls-stickness-mitigation.md) nếu có thêm layer mới.
- Share lessons learned với team.

## Prevention checklist

- [ ] CI linter reject `BYPASSRLS` role trong app DSN.
- [ ] Canary test chạy trên mỗi build (mọi PR).
- [ ] Alert `rls_defensive_trigger` → page on-call (không chỉ log).
- [ ] Quarterly red-team exercise: cố tình exploit → verify alert fire.
- [ ] Bounty program cho security researcher.

## References

- [ADR-0005 RLS Stickiness Mitigation](../adr/0005-rls-stickness-mitigation.md) — chi tiết 5 layers
- [PostgreSQL RLS docs](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [incident-service-down.md](incident-service-down.md)
- [GDPR Art. 33 — breach notification](https://gdpr-info.eu/art-33-gdpr/)