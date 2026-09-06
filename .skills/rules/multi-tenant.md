---
name: rinco-multi-tenant
description: Skill về Multi-Tenant isolation, RLS, tenant context, và security patterns trong RINCO.
---

# RINCO Multi-Tenant Skill

## Core Principle
**Mọi query, mọi cache key, mọi event đều phải có `tenant_id`.** Nếu thiếu → bug ng hiểm trọng.

## Database (PostgreSQL)

### Row-Level Security Pattern
```sql
-- 1. Enable RLS
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads FORCE ROW LEVEL SECURITY;

-- 2. Policy
CREATE POLICY leads_tenant_isolation ON leads
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY leads_user_scope ON leads
  USING (
    owner_user_id = current_setting('app.current_user_id', true)::UUID
    OR owner_user_id IN (
      SELECT id FROM users
      WHERE path <@ (
        SELECT path FROM users 
        WHERE id = current_setting('app.current_user_id', true)::UUID
      )
    )
    OR current_setting('app.is_super_admin', true) = 'true'
  );
```

### Application Code
```go
// PHẢI set context trước khi query
func (s *LeadService) ListLeads(ctx context.Context) ([]Lead, error) {
    tenantID := getTenantID(ctx)
    userID := getUserID(ctx)
    
    tx, _ := s.db.BeginTx(ctx, nil)
    defer tx.Rollback()
    
    // Set session variables for RLS
    tx.Exec("SET LOCAL app.current_tenant_id = $1", tenantID)
    tx.Exec("SET LOCAL app.current_user_id = $1", userID)
    
    return tx.QueryContext(ctx, "SELECT * FROM leads WHERE ...")
}
```

## Valkey Cache Keys

### Format
```
<scope>:<tenant_id>:<resource>

Ví dụ:
  domain:apex-fintech:tenant-config
  presence:user:{apex-fintech}:uuid
  ratelimit:apex-fintech:api-call
```

### Multi-tenant Token Bucket
```go
key := fmt.Sprintf("ratelimit:%s:%s", tenantID, action)
valkey.Eval(`
    local current = redis.call('INCR', KEYS[1])
    if current == 1 then
        redis.call('EXPIRE', KEYS[1], ARGV[2])
    end
    return current <= tonumber(ARGV[1])
`, []string{key}, limit, windowSeconds)
```

## ScyllaDB

### Partition Key phải có tenant_id
```cql
CREATE TABLE rinco_chat.messages (
  channel_id text,
  tenant_id text,           -- PHẢI có
  event_time bigint,
  ...
  PRIMARY KEY ((channel_id, tenant_id), event_time, event_id)
);

-- Mọi query PHẢI có tenant_id trong WHERE
SELECT * FROM messages WHERE channel_id = ? AND tenant_id = ?;
```

## ClickHouse

### Per-tenant Row Policy
```sql
CREATE USER tenant_apex IDENTIFIED BY 'xxx';
CREATE ROW POLICY tenant_apex_filter ON rinco_analytics.events_raw
  USING tenant_id = 'apex' TO tenant_apex;
```

## MinIO/S3

### Bucket Prefix
```
rinco-chat-files/{tenant_id}/{channel_id}/{file_id}
rinco-user-avatars/{tenant_id}/{user_id}/{filename}
```

## NATS Topics

### Format
```
{tenant_scope}.{resource}.{action}

Ví dụ:
  apex-fintech.lead.created
  apex-fintech.deal.updated
  system.admin.alert
```

## Tenant Resolution

### Từ Domain
```go
func ResolveTenant(host string) (string, error) {
    val, err := valkey.Get("domain:" + host).Result()
    if err != nil {
        // Fallback: query PostgreSQL
        return queryTenantByDomain(host)
    }
    var config TenantConfig
    json.Unmarshal([]byte(val), &config)
    return config.TenantID, nil
}
```

### Từ Subdomain
```go
// apex.hanghoaphaisinh.net → apex-fintech
parts := strings.Split(host, ".")
if len(parts) >= 3 {
    subdomain := parts[0]
    return lookupTenant(subdomain)
}
```

## Audit

### Mọi tenant-related action phải log
```go
auditLog.Record(ctx, AuditEntry{
    TenantID:   tenantID,
    ActorID:    userID,
    Action:     "lead.delete",
    TargetID:   leadID,
    IPAddress:  getIP(ctx),
    TraceID:    getTraceID(ctx),
})
```

## Testing

### Cross-tenant test bắt buộc
```go
func TestCrossTenantIsolation(t *testing.T) {
    // Setup 2 tenants
    ctxA := context.WithTenant(ctx, "tenant-a")
    ctxB := context.WithTenant(ctx, "tenant-b")
    
    // Create lead in A
    leadID := createLead(ctxA, ...)
    
    // Try to access from B → phải fail
    _, err := getLead(ctxB, leadID)
    require.Error(t, err)
    require.Equal(t, ErrNotFound, err)
}
```

## Common Mistakes

❌ **KHÔNG BAO GIỜ** query mà không có tenant_id filter.
❌ **KHÔNG BAO GIỜ** cache key mà không có tenant_id.
❌ **KHÔNG BAO GIỜ** emit event mà không có tenant_id.
❌ **KHÔNG BAO GIỜ** dùng chung connection pool giữa tenants mà không có RLS.