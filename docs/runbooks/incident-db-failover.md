# Runbook: Database Failover

> **Severity**: P0 · **On-call response**: < 10 phút · **Owner**: DBA + SRE

## Symptoms

- Alert: `postgres_up{role="primary"} == 0` for > 30s.
- Alert: `replication_lag_seconds > 60` (primary write nhưng replica lag).
- Multiple services down với error `connection refused` hoặc `FATAL: could not write to file`.
- ai-sre: `INC-XXX — Database primary unavailable`.

## First 5 minutes — Triage

### 1. Verify primary status

```bash
# Check Patroni / HA cluster status
patronictl -c /etc/patroni.yml list

# Hoặc qua kubectl
kubectl exec -n rinco postgres-0 -- pg_isready

# Check failover state
patronictl -c /etc/patroni.yml history
```

Output mẫu:

```
+ Cluster: rinco-pg (7180425420312013801) -------+---------+----+-----------+
| Member       | Host         | Role    | State     | TL | Lag in MB |
+--------------+--------------+---------+-----------+----+-----------+
| postgres-0   | 10.0.1.10    | Leader  | running   |  3 |           |
| postgres-1   | 10.0.1.11    | Replica | streaming |  3 |         0 |
| postgres-2   | 10.0.1.12    | Replica | streaming |  3 |         0 |
+--------------+--------------+---------+-----------+----+-----------+
```

### 2. Check service impact

```bash
# Which services down?
kubectl get pods -n rinco -o wide | grep -v Running | grep -v Completed

# Connection error rate
curl -s http://prometheus:9090/api/v1/query?query=rate\(pg_connection_errors_total\[5m\]\)
```

## Common scenarios

### A. Primary pod crash (auto-recovery)

**Patroni** sẽ tự động promote 1 replica trong 10-30 giây.

**Verify**:

```bash
# Patroni auto-failover
patronictl -c /etc/patroni.yml list

# New leader là postgres-1 hoặc postgres-2 (tùy priority)

# Verify services reconnect
kubectl logs -n rinco -l app=auth-service --tail=50 | grep -i "database\|reconnect"
```

**Recovery time**: ~30 giây — ~2 phút (phụ thuộc vào client reconnect logic).

### B. Replica lag quá lớn

**Symptom**: read queries trên replica trả về data cũ.

```bash
# Check lag
psql -h postgres-1 -c "SELECT now() - pg_last_xact_replay_timestamp() AS replication_lag;"

# Lag > 60s → read traffic tự switch về primary (qua HAProxy)
```

**Fix**: HAProxy routing tự động:

```ini
# /etc/haproxy/haproxy.cfg
backend postgres_write
    option httpchk GET /primary
    http-check expect status 200
    default-server inter 3s fall 3 rise 2

backend postgres_read
    balance roundrobin
    option httpchk GET /replica
    http-check expect status 200
    default-server inter 3s fall 3 rise 2 on-marked-down shutdown-sessions
```

### C. Disk full trên primary

**Symptom**: `ERROR: could not extend file`, `No space left on device`.

```bash
# Check disk usage
df -h /var/lib/postgresql
du -sh /var/lib/postgresql/wal/

# Purge old WAL
psql -c "SELECT pg_switch_wal();"

# Trigger checkpoint to flush WAL
psql -c "CHECKPOINT;"

# If still full → manual intervention
```

### D. Network partition (split brain)

**Symptom**: 2 nodes đều claim là primary.

**Patroni xử lý**: TTL + witness giúp detect. Nếu split brain xảy ra:

```bash
# Manually fence old primary
kubectl cordon postgres-0 -n rinco
kubectl drain postgres-0 -n rinco --ignore-daemonsets

# Verify new primary stable
patronictl -c /etc/patroni.yml list
```

### E. ScyllaDB node down

ScyllaDB cluster tự repair (gossip + hinted handoff). Verify:

```bash
# Check ScyllaDB status
nodetool status

# Check load
nodetool info

# Check repair status
nodetool repair -pr
```

## Manual failover (khi Patroni fail)

**⚠️ CHỈ thực hiện khi Patroni không thể tự failover (consensus không đạt được).**

```bash
# 1. Stop Patroni trên node hiện tại (nếu còn chạy)
ssh postgres-0 'sudo systemctl stop patroni'

# 2. Promote replica thủ công
ssh postgres-1
sudo -u postgres pg_ctl promote -D /var/lib/postgresql/data

# 3. Verify
psql -c "SELECT pg_is_in_recovery();"  # Should return 'f'

# 4. Update HAProxy / pgbouncer routing
haproxy -W -db -f /etc/haproxy/haproxy.cfg

# 5. Restart Patroni trên node cũ (nó sẽ tự join as replica)
ssh postgres-0 'sudo systemctl start patroni'
```

## Communication

Trong failover (> 1 phút downtime), thông báo trên status page:

```
[Investigating] Database failover in progress.
Impact: API requests may return 503 for ~2 minutes.
Started: 2026-09-18 10:23 UTC
Updated: 2026-09-18 10:25 UTC — New primary elected, services recovering
Resolved: 2026-09-18 10:27 UTC
```

## After failover

### Verify integrity

```bash
# Check data consistency
psql -c "SELECT count(*) FROM <table>;"

# Check RLS still enforced
psql -c "SET app.tenant_id = ''; SELECT count(*) FROM <table>;"  # should return 0

# Check replication re-established
patronictl -c /etc/patroni.yml list  # all nodes 'streaming'
```

### Postmortem

Trong vòng 48 giờ:

- Timeline (alert → detection → action → resolution).
- Root cause (disk? network? bug?).
- Recovery time (RTO achieved vs target).
- Action items:
  - [ ] Update HAProxy config nếu cần.
  - [ ] Tune Postgres checkpoint / WAL config.
  - [ ] Update Prometheus alert threshold.
  - [ ] DR drill quarterly.

## RTO/RPO targets

| Scenario | RTO (Recovery Time Objective) | RPO (Recovery Point Objective) |
|----------|------------------------------|--------------------------------|
| **Single primary fail** (auto-failover) | < 2 phút | < 5 giây (synchronous replication) |
| **Replica fail** | < 1 phút (no impact) | 0 (async, lag thường < 1s) |
| **Whole cluster fail** | < 30 phút (manual + restore from S3) | < 15 phút (last S3 snapshot + WAL replay) |
| **Region fail** | < 4 giờ (manual restore ở region khác) | < 1 giờ (last cross-region WAL) |

## Prevention

- [ ] **Patroni + 3-node cluster** (consensus = 2).
- [ ] **Synchronous replication** cho tenant/crm/billing DBs (RPO = 0).
- [ ] **Asynchronous replication** cho analytics/sessions (RPO có thể vài giây).
- [ ] **Disk monitoring** — alert khi disk > 80%.
- [ ] **Quarterly DR drill** — test failover sang region thứ 2.

## References

- [Patroni documentation](https://patroni.readthedocs.io/)
- [PostgreSQL HA](https://www.postgresql.org/docs/current/high-availability.html)
- [ADR-0001 Polyglot Persistence](../adr/0001-polyglot-persistence.md)
- [incident-service-down.md](incident-service-down.md)
- [dr-restore-drill.md](dr-restore-drill.md)