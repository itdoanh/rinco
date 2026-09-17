# Runbooks Index

Operational runbooks cho RINCO Platform — hướng dẫn step-by-step để xử lý incidents và disaster recovery.

> **Quy tắc**: Khi incident xảy ra, mở runbook tương ứng, làm theo từng bước. Đừng improvise.

## Severity definitions

| Sev | Định nghĩa | Response time | Escalation |
|-----|-----------|---------------|------------|
| **P0** | Service down / data breach / critical impact | < 5 phút | Hotline + on-call page |
| **P1** | Degraded performance / partial outage | < 15 phút | Telegram on-call |
| **P2** | Bug / minor issue | < 4 giờ | Slack channel |
| **P3** | Cosmetic / nice-to-have | < 24 giờ | Ticket queue |

## Runbooks

### Incident response

| Runbook | Severity | Use when |
|---------|----------|----------|
| [incident-service-down.md](incident-service-down.md) | P0 | Bất kỳ backend service nào down |
| [incident-rls-bypass.md](incident-rls-bypass.md) | **P0 Critical** | Cross-tenant data leak / RLS bypass detected |
| [incident-db-failover.md](incident-db-failover.md) | P0 | Database primary fail / replication lag |
| [incident-sfu-overload.md](incident-sfu-overload.md) | P1 | webrtc-sfu quá tải (meeting lag/giật) |
| [incident-cost-spike.md](incident-cost-spike.md) | P1 | Chi phí đột biến (LLM tokens, bandwidth, storage) |

### Disaster recovery

| Runbook | Frequency | Use when |
|---------|-----------|----------|
| [dr-restore-drill.md](dr-restore-drill.md) | Quarterly | Test khôi phục toàn bộ hệ thống từ backup |

### Onboarding

| Runbook | Use when |
|---------|----------|
| [QUICKSTART.md](QUICKSTART.md) | Dev mới onboard (5-min setup) |

---

## Quy trình chung

### 1. Phát hiện incident

Có 3 nguồn chính:

1. **Prometheus alert** → Alertmanager → PagerDuty/Telegram.
2. **ai-sre** (port 8090) → tự động tạo incident `INC-XXX` + suggest remediation.
3. **User report** (qua support@rinco.vn hoặc in-app).

### 2. Acknowledge

```bash
# Qua ai-sre dashboard (admin-portal → System → Incidents)
# Hoặc qua API:
curl -X POST http://ai-sre:8090/v1/incidents/INC-42/acknowledge \
  -H "Authorization: Bearer <super-admin-token>"
```

Alert sẽ silence 15 phút.

### 3. Investigate

Mở runbook tương ứng. Đọc kỹ "First 5 minutes" section.

### 4. Communicate

```bash
# Update status page
curl -X POST http://statuspage/api/incidents \
  -d '{"name": "INC-42", "status": "investigating", "message": "..."}'

# Slack #incidents
# @channel INC-42: webrtc-sfu overload. Investigating.
```

### 5. Resolve

Apply fix → confirm metrics back to normal → close alert.

### 6. Postmortem

Trong vòng 48 giờ cho P0, 7 ngày cho P1.

Output: `docs/runbooks/postmortems/YYYY-MM-DD-INC-NNN.md`.

### 7. Action items

Track trong Linear/Jira. Review mỗi sprint retro.

---

## Tools & access

### kubectl shortcuts

```bash
# Xem tất cả services trong namespace rinco
kubectl get pods -n rinco

# Logs service cụ thể
kubectl logs -n rinco -l app=<service> --tail=200 -f

# Exec vào pod
kubectl exec -n rinco -it <pod-name> -- bash

# Port forward (để debug local)
kubectl port-forward -n rinco <pod-name> 8080:8080
```

### Prometheus queries

```promql
# Service down
up{job=~".*rinco.*"} == 0

# Error rate
sum by (service) (rate(http_requests_total{status=~"5.."}[5m]))
  / sum by (service) (rate(http_requests_total[5m]))

# Latency p95
histogram_quantile(0.95,
  sum by (service, le) (rate(http_request_duration_seconds_bucket[5m]))
)

# RLS defensive trigger
rate(rls_defensive_trigger_total[5m])
```

### Common env vars

```bash
export KUBECONFIG=~/.kube/rinco-prod.yaml
export TELEGRAM_TOKEN=<bot-token>
export SLACK_WEBHOOK=<webhook-url>
```

---

## On-call rotation

| Week | Primary | Secondary |
|------|---------|-----------|
| W1 | @alice | @bob |
| W2 | @bob | @charlie |
| W3 | @charlie | @diana |
| W4 | @diana | @alice |

PagerDuty schedule: `rinco-sre-primary`.

---

## References

- [ARCHITECTURE.md](../ARCHITECTURE.md) — system overview
- [ADR-0005 RLS Stickiness](../adr/0005-rls-stickness-mitigation.md) — security #1
- [SYSTEM_STATUS.md](../SYSTEM_STATUS.md) — realtime health
- [ADMIN_GUIDE.md §12](../ADMIN_GUIDE.md#12-incident-management-với-ai-sre) — ai-sre usage