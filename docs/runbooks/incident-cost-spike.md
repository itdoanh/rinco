# Runbook: Cost Spike

> **Severity**: P1 (P0 nếu spike > 5× baseline) · **Owner**: FinOps + SRE

## Symptoms

- Alert: `monthly_cost_estimate > 1.5× of month_to_date_budget`.
- Billing dashboard shows anomaly (vd: LLM token usage tăng 10×).
- Stripe / cloud invoice alert.
- User reports: "Dịch vụ chậm" (thường do rate-limit triggered).
- ai-sre incident: `INC-XXX — Cost anomaly detected`.

## Cost breakdown (baseline / tháng)

| Resource | Baseline cost | % of total |
|----------|---------------|------------|
| **LLM inference** (vLLM GPU) | $8,000 | 35% |
| **PostgreSQL + ScyllaDB** (managed) | $4,000 | 17% |
| **MinIO storage** (tiered) | $2,500 | 11% |
| **WebRTC bandwidth** (Cloudflare) | $3,000 | 13% |
| **K3s nodes** (compute) | $3,500 | 15% |
| **Other** (monitoring, NATS, etc.) | $2,000 | 9% |
| **Total** | **$23,000** | 100% |

> Chi phí thực tế phụ thuộc usage. Đây là estimate cho 50 tenants active.

## First 30 minutes — Identify spike

### 1. Cost dashboard (FinOps Grafana)

Vào `https://grafana.rinco.internal/d/cost-overview`:

```
┌─────────────────── Cost Breakdown (this hour) ───────────────────┐
│                                                                │
│  LLM inference   ████████████████████████████░░░░  $40/h (+5×) │
│  Bandwidth       ████████░░░░░░░░░░░░░░░░░░░░░░░  $5/h  (1.0x) │
│  Storage         ████░░░░░░░░░░░░░░░░░░░░░░░░░░░  $3/h  (1.0x) │
│  Database        ████░░░░░░░░░░░░░░░░░░░░░░░░░░░  $3/h  (1.0x) │
│  Compute         ██████░░░░░░░░░░░░░░░░░░░░░░░░░░  $4/h  (1.0x) │
│                                                                │
│  Total: $55/h (+3.5× baseline $16/h)                           │
└────────────────────────────────────────────────────────────────┘
```

### 2. Drill down

Click vào từng panel để xem top consumers:

```promql
# LLM tokens per service / model
sum by (service, model) (
  rate(llm_tokens_total[1h])
) * 24 * 30  # monthly estimate

# Top bandwidth consumers
sum by (service) (
  rate(network_bytes_total[1h])
) * 24 * 30
```

## Common causes & fixes

### A. LLM token explosion

**Symptom**: `llm_tokens_total` tăng 5-10× baseline.

**Root causes thường gặp**:

1. **Infinite retry loop**: RAG chatbot retry query vì hallucinated bad output.
2. **Prompt bloat**: ai-sre embed quá nhiều context vào prompt.
3. **Bug**: thiếu `max_tokens` cap → response dài 10k tokens thay vì 500.
4. **Abuse**: 1 tenant spam AI API.

**Mitigation ngay**:

```bash
# 1. Set rate limit per AI service (qua admin-portal)
curl -X POST http://api-gateway:8080/v1/admin/rate-limit \
  -H "Authorization: Bearer <admin-token>" \
  -d '{
    "service": "rag-chatbot",
    "limit_per_tenant": "1000/hour",
    "global_limit": "50000/hour"
  }'

# 2. Switch to cheaper model (vd: Mistral 7B thay vì Llama 70B)
kubectl set env deployment/rag-chatbot -n rinco \
  MODEL=mistral-7b-instruct \
  MAX_TOKENS=1024

# 3. Add response cache (Redis semantic cache)
kubectl set env deployment/rag-chatbot -n rinco \
  SEMANTIC_CACHE=enabled \
  CACHE_TTL=3601
```

**Long-term**:

- Implement **token budgeting** per tenant (alert ở 80%, throttle ở 100%).
- **Model routing**: small query → small model, big query → big model.
- **Output validation**: catch hallucinated output → retry cap = 2 (not infinite).

### B. Bandwidth spike (WebRTC)

**Symptom**: Cloudflare bandwidth charge tăng đột biến.

**Mitigations**:

```bash
# 1. Force codec down (VP9 → H264)
kubectl set env deployment/webrtc-sfu -n rinco \
  PREFERRED_CODEC=H264

# 2. Lower resolution cap
kubectl set env deployment/webrtc-sfu -n rinco \
  MAX_RESOLUTION=480p

# 3. Enable simulcast
kubectl set env deployment/webrtc-sfu -n rinco \
  SIMULCAST=enabled
```

### C. Storage growth (recordings, backups)

**Symptom**: MinIO usage tăng 2× baseline.

```bash
# Check bucket size
mc du rinco/rinco-data
mc du --depth=1 rinco/rinco-data

# Identify hot buckets
mc stat rinco/rinco-data/recordings

# Aggressive tier migration (off-peak only)
mc ilm edit rinco/rinco-data \
  --transition-days 14  # move to HDD after 14 days instead of 30
```

### D. Database storage spike

```sql
-- Find largest tables
SELECT
    schemaname || '.' || tablename AS table_name,
    pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename)) AS size,
    pg_total_relation_size(schemaname || '.' || tablename) AS size_bytes
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY size_bytes DESC
LIMIT 20;

-- Identify bloat
SELECT
    schemaname || '.' || tablename AS table_name,
    pg_size_pretty(pg_relation_size(schemaname || '.' || tablename)) AS table_size,
    pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename) - pg_relation_size(schemaname || '.' || tablename)) AS bloat_size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname || '.' || tablename) DESC
LIMIT 20;
```

**Mitigation**:

```sql
-- VACUUM FULL (off-peak only, requires downtime)
VACUUM FULL ANALYZE <table>;

-- Drop unused indexes
DROP INDEX CONCURRENTLY <unused_index>;

-- Partition large tables
-- (xem ADR chi tiết về partitioning)
```

### E. Compute spike (idle pods)

```bash
# Find pods với low utilization
kubectl top pods -n rinco --sort-by=cpu | head -20

# Find oversized HPA targets
kubectl get hpa -n rinco

# Reduce replicas (off-peak)
kubectl scale deployment/<service> -n rinco --replicas=2
```

## Long-term controls

### 1. Budgets & alerts

```yaml
# budgets.yaml (FinOps config)
budgets:
  - name: monthly_total
    amount: 25000
    period: monthly
    alerts:
      - threshold: 50%
        severity: info
      - threshold: 80%
        severity: warning
        channel: slack
      - threshold: 100%
        severity: critical
        channel: pager
  - name: llm_tokens
    amount: 9000
    period: monthly
    alerts:
      - threshold: 90%
        action: throttle
  - name: bandwidth
    amount: 3500
    period: monthly
```

### 2. Per-tenant cost attribution

- Tag mọi metric với `tenant_id`.
- Dashboard per-tenant cost breakdown.
- Email Owner khi vượt 80% quota.

### 3. Spot instances cho batch

- AI training jobs → spot instances (giảm 70% cost).
- ETL jobs → spot.
- Web traffic → on-demand.

### 4. Caching strategy

- RAG: semantic cache (Redis).
- Static assets: CDN cache (Cloudflare).
- Database queries: prepared statement cache.

## After resolution

1. **Verify** cost trend về baseline trong vòng 24h.
2. **Update** budget thresholds nếu cần.
3. **Identify** root cause + implement prevention.
4. **Communicate** tới stakeholders nếu cost > budget (chargeback internal).

## Cost anomaly detection

Dùng [ai-sre](../../services/ai-sre) (port 8090) với LLM-based anomaly detection:

```
[FinOps dashboard] → click "Investigate"
                   ↓
        ai-sre queries ClickHouse:
          - llm_tokens_total (1h, 24h, 7d)
          - cost_by_tenant
          - cost_by_service
        ↓
        LLM suggests root cause:
          - "Tenant apex-fintech đột biến 8× bình thường.
             90% là rag-chatbot. Có thể do user spam."
        ↓
        Admin click "Throttle tenant apex-fintech"
        ↓
        Notification gửi tới Owner
```

## Prevention

- [ ] **Budget alerts** configured cho mọi dimension (service, tenant, region).
- [ ] **Rate limits** strict per tenant.
- [ ] **Auto-shutoff** cho non-prod environments (off business hours).
- [ ] **Quarterly cost review** với leadership.
- [ ] **Showback report** monthly cho từng tenant (charge cho usage).

## References

- [ADR-0001 Polyglot Persistence](../adr/0001-polyglot-persistence.md)
- [ADR-0006 Tiered Storage MinIO](../adr/0006-tiered-storage-minio.md)
- [services/ai-sre](../../services/ai-sre)
- [incident-sfu-overload.md](incident-sfu-overload.md)
- [Grafana FinOps dashboards](https://grafana.com/grafana/dashboards/)