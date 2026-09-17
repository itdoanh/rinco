# ADR-0005: RLS Stickiness Mitigation (Critical Issue #1)

> **Status**: Accepted · **Date**: 2026-09-18 · **Authors**: RINCO Platform Team
> **Severity**: **Critical** — đây là security issue #1 của hệ thống.

## Context

RINCO là **multi-tenant SaaS** — mỗi tenant có dữ liệu riêng và **TUYỆT ĐỐI** không được lẫn vào nhau.

Mọi bảng có `tenant_id NOT NULL` + **Row-Level Security (RLS)** policy:

```sql
CREATE POLICY tenant_isolation ON crm.nodes
    USING (tenant_id = current_setting('app.tenant_id')::UUID);
```

**Critical Issue #1 (RLS Stickiness)**: nếu connection pool **không reset** `app.tenant_id` giữa các request, request sau có thể dùng nhầm `tenant_id` của request trước → **tenant data leak**.

### Ví dụ bug

```
T1: Request A (tenant X) sets app.tenant_id='X' → returns connection to pool
T2: Request B (tenant Y) checks out SAME connection → app.tenant_id still='X'
T3: Request B executes SELECT * FROM crm.nodes → RLS uses 'X' → returns X's data
T4: Request B logs "selected 1000 rows for tenant X" → BUG!
```

Đây là **silent data leak** — không có error, không có log cảnh báo.

## Decision

Áp dụng **5 lớp defense** để mitigate RLS stickiness:

### Layer 1: Connection-scoped tenant context (NOT session-scoped)

PostgreSQL có 3 cách set GUC:

| Cách | Persistent? | Risk |
|------|-------------|------|
| `SET app.tenant_id = 'X'` | Session (connection) | 🟡 Vẫn sticky nếu pool reuse |
| `SELECT set_config('app.tenant_id', 'X', false)` | Transaction | 🟢 An toàn nhất |
| `SELECT set_config('app.tenant_id', 'X', true)` | Session | 🔴 Sticky |

**Quyết định**: dùng `set_config(..., false)` — transaction-scoped. Sau `COMMIT` → reset.

### Layer 2: Middleware enforce set tenant trong transaction

Mọi service (Go/Rust/Python) phải:

1. **BEGIN TRANSACTION**.
2. `SELECT set_config('app.tenant_id', $tenantID, false)`.
3. Execute query.
4. **COMMIT** (hoặc ROLLBACK).

Code wrapper trong `packages/go/db`:

```go
func (db *DB) WithTenant(ctx context.Context, tenantID string, fn func(tx Tx) error) error {
    tx, err := db.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback()

    // CRITICAL: false = transaction-scoped (NOT session-scoped)
    _, err = tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, false)", tenantID)
    if err != nil { return err }

    if err := fn(tx); err != nil { return err }
    return tx.Commit()
}
```

**FORBIDDEN pattern** (sẽ bị CI fail):

```go
// ❌ WRONG: setting at session level
db.Exec("SET app.tenant_id = $1", tenantID)
// ❌ WRONG: setting before BEGIN
db.Exec("SELECT set_config('app.tenant_id', $1, true)", tenantID)
```

### Layer 3: Pool hygiene — `db.ResetSession` giữa checkout/checkin

Dùng `pgxpool` với custom `AfterRelease` hook:

```go
pool, _ := pgxpool.New(ctx, dsn, pgxpool.WithAfterRelease(func(conn *pgx.Conn) bool {
    // Defensive reset — nếu có bug ở Layer 1, đây là safety net
    _, _ = conn.Exec(ctx, "SELECT set_config('app.tenant_id', '', false)")
    _, _ = conn.Exec(ctx, "DISCARD ALL")
    return true
}))
```

### Layer 4: RLS policy với `current_setting` có fallback

```sql
CREATE POLICY tenant_isolation ON crm.nodes
    USING (
        current_setting('app.tenant_id', true) != ''
        AND tenant_id = current_setting('app.tenant_id')::UUID
    );
```

- `current_setting('app.tenant_id', true)` — `true` = missing_ok, trả về empty string thay vì lỗi.
- Nếu setting rỗng → policy return FALSE → không trả về row nào (defensive).
- KHÔNG bao giờ "fallback" về 1 default tenant — đó là anti-pattern.

### Layer 5: Audit & canary tests

- **Canary test**: mỗi build chạy test tự động:
  1. Mở connection pool (10 conns).
  2. Set `app.tenant_id='X'` trên conn #1, commit.
  3. Check-out conn #1 mới, SELECT count(*) — phải = 0 (vì không có X).
  4. Tương tự cho mọi conn trong pool.

- **Audit log**: mỗi query bất thường (rows > baseline × 5) → log `audit.cross_tenant_attempt` + alert.

### Layer 6 (Bonus): Connection routing theo tenant

Dùng PgBouncer với `pool_mode = transaction` + **per-tenant routing** qua `pgbouncer.ini`:

```
[databases]
rinco_auth = host=pg-auth dbname=rinco
rinco_crm  = host=pg-crm  dbname=rinco
...
```

→ Mỗi service → 1 DB → 1 pool → tenant_id rõ ràng.

Trade-off: thêm PgBouncer dependency.

## Consequences

### Positive

- **5 lớp defense** — bug ở 1 layer vẫn an toàn.
- **Transaction-scoped setting** — tự nhiên reset.
- **Test canary tự động** — phát hiện bug ngay trong CI.
- **Defensive policy** — không có fallback nguy hiểm.

### Negative

- **Performance**: mỗi transaction phải `set_config` — overhead ~0.5ms.
- **Code discipline**: dev phải nhớ dùng `WithTenant` wrapper.
- **PgBouncer routing** tăng operational complexity (mitigate bằng Helm chart auto-config).
- **Canary test** false-positive có thể xảy ra nếu pool bị rebuild.

### Mitigations

- **Linter rule** trong CI: cảnh báo nếu thấy `SET app.tenant_id` (không phải `set_config`).
- **Wrapper force**: `WithTenant` là **only** exported API; raw `db.Exec` không nên dùng ngoài package.
- **Monthly red-team exercise**: tạo tenant X, tenant Y, cố tình exploit bug → verify không leak.

## Alternatives Considered

### A. Database-per-tenant

- **Pro**: isolation tuyệt đối, không có RLS risk.
- **Con**: cost cao (hàng trăm DBs), migration nightmare, monitoring phức tạp.
- **Verdict**: ❌ Rejected — không scale cho 1000+ tenants.

### B. Schema-per-tenant

- **Pro**: tốt hơn DB-per-tenant về cost.
- **Con**: migration vẫn phức tạp (multi-schema), connection routing vẫn cần.
- **Verdict**: ❌ Rejected — RLS hiệu quả hơn.

### C. Application-layer filter only (no RLS)

- **Pro**: đơn giản, không cần RLS.
- **Con**: 1 bug query = tenant leak. Không defense-in-depth.
- **Verdict**: ❌ Rejected — không đủ an toàn cho SaaS B2B.

### D. Column encryption per tenant

- **Pro**: data at rest cũng tách biệt.
- **Con**: query gần như không thể (index không hoạt động).
- **Verdict**: ❌ Rejected — không thực tế.

## Monitoring & Alerts

| Alert | Condition | Severity |
|-------|-----------|----------|
| `tenant_id_empty_query` | Query thực thi khi `app.tenant_id` = '' | P1 |
| `cross_tenant_attempt` | Row count > baseline × 5 từ 1 endpoint | P0 |
| `rls_policy_bypass` | Query với `BYPASSRLS` role | P0 |
| `canary_test_fail` | CI canary test fail | P0 (block merge) |

## References

- [PostgreSQL RLS docs](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [services/auth-service internal/db](../../services/auth-service/internal/db) — implementation
- [packages/go/db](../../packages/go/db)
- [runbook: incident-rls-bypass](../runbooks/incident-rls-bypass.md)
- [ADR-0004 LTREE CRM](0004-ltree-crm.md)