# Runbook: Service Down

> **Severity**: P0 · **On-call response**: < 5 phút · **Owner**: SRE team

## Symptoms

- Prometheus alert: `service_up{service="<name>"} == 0` for > 1m.
- Health check endpoint `/healthz` trả 5xx.
- Grafana dashboard "Service Health" hiển thị service màu đỏ.
- `ai-sre` (port 8090) tự động tạo incident `INC-XXX` với severity P0.
- User reports: "Tôi không vào được / API trả 500".

## First 5 minutes — triages

### 1. Verify scope

```bash
# Check service is actually down (không phải monitor false-positive)
kubectl get deployment -n rinco <service-name>
kubectl get pods -n rinco -l app=<service-name>

# Check health endpoint
kubectl exec -n rinco deploy/<service-name> -- \
  curl -fsS http://localhost:8081/healthz || echo "DOWN"
```

### 2. Check recent events

```bash
# Recent deployments / config changes
kubectl rollout history deployment/<service-name> -n rinco

# Check NATS — events có đang được publish/consume?
curl -s http://nats:8222/connz | jq '.connections'

# Check database connection
kubectl logs -n rinco deploy/<service-name> --tail=200 | grep -i "database\|connection refused"
```

### 3. Check dependencies

Service có thể down vì upstream (database, NATS, Valkey) down. Kiểm tra:

- [ ] Database healthy? (xem runbook [incident-db-failover.md](incident-db-failover.md))
- [ ] NATS healthy? (`curl http://nats:8222/healthz`)
- [ ] Valkey healthy? (`redis-cli -h valkey ping`)
- [ ] ClickHouse healthy? (`curl http://clickhouse:8123/ping`)

### 4. Acquire trong ai-sre dashboard

Vào Admin Portal → System → Incidents → `INC-XXX` → **Acknowledge**.

`ai-sre` sẽ silence alert 15 phút và suggest remediation.

## Common causes & fixes

### A. Crash loop sau deploy (exit code 1)

**Symptom**: `kubectl get pods` shows `CrashLoopBackOff`, `kubectl logs` shows panic / startup error.

```bash
# Check logs of crashed container
kubectl logs -n rinco <pod-name> --previous

# Rollback nếu vừa deploy
kubectl rollout undo deployment/<service-name> -n rinco
kubectl rollout status deployment/<service-name> -n rinco
```

Sau rollback → xác nhận service healthy, root-cause investigation sau.

### B. OOMKilled

**Symptom**: `kubectl describe pod` shows `Last State: Terminated, Reason: OOMKilled`.

```bash
# Check memory usage trước khi crash
kubectl top pod -n rinco <pod-name> --containers

# Tăng memory limit (nếu memory leak chưa rõ root cause)
kubectl set resources deployment/<service-name> -n rinco \
  --limits=memory=2Gi --requests=memory=1Gi

# Restart pod để clear memory
kubectl rollout restart deployment/<service-name> -n rinco
```

Nếu OOM lặp lại → memory leak → escalate tới dev team.

### C. Database connection exhausted

**Symptom**: Logs show `pq: remaining connection slots are reserved`, hoặc `connection refused`.

```bash
# Check active connections
PGPASSWORD=$DB_PASS psql -h postgres -U rinco -c \
  "SELECT count(*) FROM pg_stat_activity;"

# Check pgBouncer stats
curl http://pgbouncer:9090/metrics | grep pg_pool_connections

# Kill idle connections
PGPASSWORD=$DB_PASS psql -h postgres -U rinco -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
   WHERE state = 'idle' AND query_start < now() - interval '5 minutes';"
```

### D. Network policy / DNS issue

**Symptom**: Logs show `no such host`, `i/o timeout`, hoặc `connection refused` đến upstream service.

```bash
# Test DNS resolution
kubectl exec -n rinco deploy/<service-name> -- \
  nslookup auth-service.rinco.svc.cluster.local

# Test connectivity
kubectl exec -n rinco deploy/<service-name> -- \
  nc -zv auth-service 8081

# Check NetworkPolicy
kubectl get networkpolicy -n rinco -o yaml
```

### E. Image pull failure

**Symptom**: `kubectl describe pod` shows `ErrImagePull` hoặc `ImagePullBackOff`.

```bash
# Check image exists
docker manifest inspect ghcr.io/itdoanh/rinco-<service>:<tag>

# Check image pull secret
kubectl get secret -n rinco regcred -o yaml

# Force re-pull
kubectl rollout restart deployment/<service-name> -n rinco
```

## Escalation

| Situation | Contact |
|-----------|---------|
| Service down > 15 phút | Telegram `@rinco_sre` |
| Suspected security breach | Hotline 1900-xxxx (24/7) |
| Database / infra down | Cloud provider support |
| Code bug | Dev team lead của service |

## After resolution

1. **Verify recovery** — Grafana shows service xanh, alert cleared.
2. **Postmortem** — file PR với template `docs/runbooks/postmortem-template.md` trong vòng 48 giờ.
3. **Update runbook** nếu phát hiện gap.
4. **Action items** — assign owners, due dates.
5. **Close incident** trong ai-sre dashboard.

## Prevention

- [ ] **Health check strict** — `/healthz` phải check DB + NATS + Valkey (deep health).
- [ ] **Resource limits** đặt đúng (CPU + memory requests = limits → QoS Guaranteed).
- [ ] **PDB (PodDisruptionBudget)** để tránh voluntary disruption.
- [ ] **HPA** scale theo CPU/RPS, không scale theo memory (tránh flapping).
- [ ] **Smoke test sau deploy** — tự động test critical endpoint sau mỗi release.

## References

- [ai-sre service](../../services/ai-sre)
- [ADR-0005 RLS](adr/0005-rls-stickness-mitigation.md)
- [incident-db-failover.md](incident-db-failover.md)
- [K8s docs: Debug Pod](https://kubernetes.io/docs/tasks/debug/debug-application/debug-pods/)