# Runbook: DR Restore Drill

> **Frequency**: Quarterly (mỗi 3 tháng) · **Owner**: SRE + DBA · **Duration**: 4-8 giờ

## Mục đích

Verify rằng hệ thống RINCO có thể **khôi phục từ disaster** trong thời gian RTO target (4 giờ) và đạt RPO target (1 giờ) thông qua việc thực hành định kỳ.

Drill bao gồm:
- Khôi phục PostgreSQL từ S3 snapshot + WAL replay.
- Khôi phục ScyllaDB từ snapshot.
- Khôi phục MinIO từ cross-region replica.
- Khôi phục ClickHouse.
- Verify RLS integrity (không có cross-tenant leak sau restore).
- Smoke test critical flows.

## Pre-requisites

- [ ] **Sandbox cluster** đã setup (region phụ, không phải production).
- [ ] **Backup verification** job chạy hàng ngày, tất cả backups gần nhất < 24h.
- [ ] **RTO/RPO targets** đã được leadership sign-off.
- [ ] **DR runbook** (file này) reviewed và tested trong vòng 6 tháng qua.
- [ ] **Communication plan** (Slack channel, status page, stakeholders list).

## Schedule

```
T-7 days:    Notify stakeholders (Slack #dr-drill, email leadership)
T-1 day:     Final prep, dry-run tabletop với team
T-0:         DR drill bắt đầu (window: 8 giờ, off-peak cho users)
T+0 to T+8:  Execute restore + verify
T+1 day:     Postmortem + report
T+7 days:    Action items tracked, sign-off từ VP Engineering
```

## Architecture overview

```
Primary Region (ap-southeast-1)
├── PostgreSQL (Patroni 3-node)
├── ScyllaDB (3-node)
├── MinIO (12 nodes, EC:4)
├── ClickHouse (3-shard)
└── → S3 snapshot every 6h + WAL streaming

Secondary Region (ap-southeast-2) — DR target
├── (empty cluster, ready for restore)
└── WireGuard mesh to primary (for WAL stream)
```

## Phase 1: Pre-drill validation (T-1 day)

### Verify backups exist

```bash
# List recent PostgreSQL backups
aws s3 ls s3://rinco-backup/postgres/ --recursive | tail -20

# Most recent full backup
aws s3 ls s3://rinco-backup/postgres/full/ --recursive | sort | tail -1

# WAL archive status
psql -c "SELECT pg_last_wal_replay_lsn();"
psql -c "SELECT last_archived_time FROM pg_stat_archiver;"
```

### Check backup integrity

```bash
# Download most recent backup
aws s3 cp s3://rinco-backup/postgres/full/2026-09-17/full.dump /tmp/

# Verify checksum
sha256sum /tmp/full.dump
# So sánh với checksum trong S3 metadata

# Test restore to scratch instance
pg_restore --list /tmp/full.dump | head -20
```

### Confirm DR sandbox ready

```bash
# SSH to DR region
ssh dr-sandbox-admin

# Verify cluster empty
kubectl get nodes
kubectl get pods -A | grep -v Running

# Verify WireGuard connectivity to primary
ping pg-primary-primary.ap-southeast-1.rinco.internal
```

## Phase 2: Initiate drill (T+0)

### T+0:00 — Declare drill

Trong Slack `#dr-drill`:

```
[DR DRILL START]
Started: 2026-09-18 14:00 UTC
Target RTO: 4 hours
Target RPO: 1 hour
Operator: @alice (lead), @bob (DBA), @charlie (SRE), @diana (networking)
Status page: https://status.rinco.internal (under maintenance)
Standby communications: Slack #dr-drill + Telegram @rinco_sre
```

### T+0:05 — Verify backups one more time

```bash
# Final check — latest backup
LATEST=$(aws s3 ls s3://rinco-backup/postgres/full/ | sort | tail -1 | awk '{print $4}')
echo "Latest backup: $LATEST"

# Verify checksum
aws s3 cp s3://rinco-backup/postgres/full/$LATEST /tmp/restore.dump
aws s3 cp s3://rinco-backup/postgres/full/$LATEST.sha256 /tmp/restore.sha256
sha256sum -c /tmp/restore.sha256
```

### T+0:15 — Start PostgreSQL restore

```bash
# 1. Stop DR sandbox PostgreSQL (if any)
kubectl scale statefulset/postgres --replicas=0 -n rinco-dr

# 2. Restore full backup
kubectl run pg-restore -n rinco-dr --rm -it --restart=Never \
  --image=postgres:16 \
  --env="PGPASSWORD=$DB_PASS" \
  -- bash -c "
    pg_restore -h postgres-scratch -U rinco -d rinco /tmp/restore.dump
  "

# 3. Configure as standby, replay WAL up to point-in-time
kubectl exec -n rinco-dr postgres-0 -- bash -c "
  cat > /etc/postgresql/recovery.signal << EOF
EOF
  echo 'recovery_target_time = \"2026-09-18 13:30:00\"' >> /etc/postgresql/postgresql.conf
  echo 'restore_command = \"wal-g wal-fetch %f %p\"' >> /etc/postgresql/postgresql.conf
  pg_ctl restart
"
```

### T+0:30 — Start ScyllaDB restore

```bash
# 1. Stop DR sandbox ScyllaDB
kubectl scale statefulset/scylla --replicas=0 -n rinco-dr

# 2. Download snapshot from S3
kubectl run scylla-restore -n rinco-dr --rm -it --restart=Never \
  --image=scylladb/scylla:6 \
  -- bash -c "
    aws s3 sync s3://rinco-backup/scylla/2026-09-17/ /var/lib/scylla/data/
  "

# 3. Start ScyllaDB and run repair
kubectl scale statefulset/scylla --replicas=3 -n rinco-dr
sleep 60  # wait for ready

kubectl exec -n rinco-dr scylla-0 -- nodetool repair -pr
```

### T+0:45 — Restore MinIO

```bash
# Option A: Restore from cross-region replica (preferred)
mc alias set dr-source https://s3-ap-southeast-1.rinco.internal $SOURCE_KEY $SOURCE_SECRET
mc mirror dr-source/rinco-data dr-target/rinco-data

# Option B: Restore from S3 backup
aws s3 sync s3://rinco-backup-minio/rinco-data/ /mnt/minio-data/ \
  --delete
```

### T+1:00 — Restore ClickHouse

```bash
# 1. Restore from backup
kubectl exec -n rinco-dr clickhouse-0 -- bash -c "
  clickhouse-backup restore --config=/etc/clickhouse-backup/config.yml \
    --backup=2026-09-17-daily \
    --schema \
    --data
"

# 2. Verify
kubectl exec -n rinco-dr clickhouse-0 -- clickhouse-client -q "
  SELECT count() FROM audit.events WHERE created_at > now() - INTERVAL 1 DAY;
"
```

### T+1:30 — Start other services

```bash
# Bring up NATS, Valkey, Qdrant, Meilisearch, MinIO
kubectl apply -f deployments/helm/rinco/templates/ -n rinco-dr

# Wait for all pods ready
kubectl wait --for=condition=Ready pods --all -n rinco-dr --timeout=600s
```

### T+2:00 — Start application services

```bash
# Apply Helm chart (với sandbox-specific config)
helm upgrade --install rinco deployments/helm/rinco/ \
  --namespace rinco-dr \
  --values deployments/helm/rinco/values-dr.yaml \
  --set image.tag=v1.1.0-dr-drill

# Wait for rollout
kubectl rollout status deployment -n rinco-dr --timeout=600s
```

## Phase 3: Verification (T+2:00 to T+4:00)

### T+2:00 — RLS integrity test (CRITICAL)

```sql
-- Verify tenant isolation vẫn hoạt động sau restore
\c rinco

-- Test 1: Empty tenant_id → 0 rows
SET app.tenant_id = '';
SELECT count(*) FROM crm.nodes;  -- MUST be 0

-- Test 2: Tenant X chỉ thấy data của X
SET app.tenant_id = '00000000-0000-0000-0000-000000000001';
SELECT count(*) FROM crm.nodes;
-- So sánh với count từ primary (phải match)

-- Test 3: Tenant Y thấy 0 rows của X
SET app.tenant_id = '00000000-0000-0000-0000-000000000002';
SELECT count(*) FROM crm.nodes WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
-- MUST be 0
```

### T+2:30 — Smoke test critical flows

```bash
# 1. Auth flow
curl -X POST http://auth-service:8081/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@rinco.vn","password":"test"}'
# Expect: 200 + PASETO token

# 2. Lead capture flow
curl -X POST http://landing-service:8086/v1/leads \
  -H "Content-Type: application/json" \
  -d '{"name":"DR Drill","email":"dr@rinco.vn","phone":"+84900000000"}'
# Expect: 201 + lead_id

# 3. CRM query
TOKEN="..."
curl http://crm-service:8083/v1/deals \
  -H "Authorization: Bearer $TOKEN"
# Expect: 200 + list of deals

# 4. AI scoring (asynchronous, check ClickHouse after 30s)
sleep 30
clickhouse-client -q "SELECT count() FROM lead_events WHERE event_type='lead.scored' AND created_at > now() - INTERVAL 5 MINUTE"
# Expect: > 0
```

### T+3:00 — Load test (optional, nếu time cho phép)

```bash
# 100 concurrent users, 5 phút
k6 run --vus 100 --duration 5m loadtest/basic.js

# Verify:
# - p95 latency < 500ms
# - error rate < 0.1%
# - no connection pool exhaustion
```

### T+3:30 — Document metrics

Điền vào bảng dưới và archive:

| Metric | Target | Actual | Pass? |
|--------|--------|--------|-------|
| **RTO achieved** (full restore) | < 4 giờ | ___ | ☐ |
| **RPO achieved** (data loss) | < 1 giờ | ___ | ☐ |
| **PostgreSQL restore time** | < 1 giờ | ___ | ☐ |
| **ScyllaDB restore time** | < 1 giờ | ___ | ☐ |
| **MinIO restore time** | < 1 giờ | ___ | ☐ |
| **ClickHouse restore time** | < 30 phút | ___ | ☐ |
| **RLS integrity test** | Pass | ___ | ☐ |
| **Smoke tests** | All pass | ___ | ☐ |
| **Load test p95** | < 500ms | ___ | ☐ |

## Phase 4: Cleanup & declare success (T+4:00)

### T+4:00 — Tear down DR sandbox

```bash
# Scale down all services
kubectl scale deployment --all --replicas=0 -n rinco-dr
kubectl scale statefulset --all --replicas=0 -n rinco-dr

# Delete sandbox namespace (optional, hoặc giữ để investigate)
# kubectl delete namespace rinco-dr
```

### T+4:00 — Announce completion

```
[DR DRILL COMPLETE]
Started: 2026-09-18 14:00 UTC
Ended: 2026-09-18 18:00 UTC
RTO achieved: 3h 45min (target: 4h) ✅
RPO achieved: 23 min (target: 1h) ✅
RLS integrity: PASS ✅
Smoke tests: 4/4 PASS ✅

Lessons learned:
- [TBD từ postmortem]

Action items:
- [TBD]
```

## Phase 5: Postmortem (T+1 day)

Schedule meeting trong vòng 24 giờ với:

- SRE team
- DBA team
- Engineering leads
- VP Engineering (review + sign-off)

Topics:

1. **Timeline** — chi tiết từng phase, ai làm gì lúc nào.
2. **Issues encountered** — bugs, gaps, unclear runbook steps.
3. **RTO/RPO actual vs target**.
4. **Improvements** — tooling, runbook, automation.
5. **Action items** — owner, due date, severity.

Output: `docs/runbooks/postmortems/2026-09-18-dr-drill.md`

## Common issues & fixes

### Issue: WAL replay quá chậm

**Fix**: tăng `wal_compress` + parallel replay workers.

### Issue: ScyllaDB repair không converge

**Fix**: dùng `--full` flag cho first repair, sau đó incremental.

### Issue: MinIO mirror không complete

**Fix**: run lại với `--overwrite` flag, verify md5sum.

### Issue: ClickHouse schema version mismatch

**Fix**: replay schema migrations từ `migrations/clickhouse/`.

### Issue: RLS test fail

**CRITICAL** — investigate ngay. Có thể RLS policy bị drop trong restore process.

```sql
-- Verify policies
SELECT schemaname, tablename, policyname FROM pg_policies
WHERE schemaname IN ('auth','tenant','crm','lead','dynamic_model','email','notification','lead_scoring','meta_capi','billing','search')
ORDER BY schemaname, tablename;

-- Re-create missing policies
-- (xem scripts/rls-policies.sql trong repo)
```

## Automation (future)

Roadmap items:

- [ ] **Ansible playbook** tự động restore từ S3.
- [ ] **Cron** schedule quarterly drill (currently manual).
- [ ] **DR test dashboard** show metrics real-time.
- [ ] **Slack bot** announce progress updates.

## References

- [ADR-0001 Polyglot Persistence](../adr/0001-polyglot-persistence.md)
- [ADR-0006 Tiered Storage MinIO](../adr/0006-tiered-storage-minio.md)
- [runbook: incident-db-failover.md](incident-db-failover.md)
- [PostgreSQL PITR docs](https://www.postgresql.org/docs/current/continuous-archiving.html)
- [ScyllaDB backup docs](https://docs.scylladb.com/operating-scylla/backup-restore/)
- [ClickHouse backup docs](https://clickhouse.com/docs/en/operations/backup)