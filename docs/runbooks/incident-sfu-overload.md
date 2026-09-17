# Runbook: WebRTC SFU Overload

> **Severity**: P1 · **On-call response**: < 10 phút · **Owner**: Realtime SRE

## Symptoms

- Alert: `webrtc_sfu_cpu_usage > 90%` for > 2m.
- Alert: `webrtc_sfu_active_meetings > 80% of capacity`.
- User reports: "Video bị giật / lag / mất kết nối".
- Sfu-upstream bandwidth > 80% capacity.
- ai-sre incident: `INC-XXX — webrtc-sfu overload`.

## First 5 minutes — Stabilize

### 1. Verify load

```bash
# Check current capacity
kubectl exec -n rinco deploy/webrtc-sfu -- \
  curl -s http://localhost:8102/metrics | grep -E "active_meetings|active_participants|cpu"

# Per-pod breakdown
kubectl top pod -n rinco -l app=webrtc-sfu --containers
```

### 2. Scale horizontally (fastest relief)

```bash
# Manual scale (nếu HPA chưa kịp)
kubectl scale deployment/webrtc-sfu -n rinco --replicas=10

# Hoặc trigger HPA ngay
kubectl patch hpa webrtc-sfu -n rinco -p \
  '{"spec":{"maxReplicas":20,"targetCPUUtilizationPercentage":70}}'

# Verify new pods ready
kubectl get pods -n rinco -l app=webrtc-sfu -w
```

### 3. Drain heaviest meetings (nếu cần)

```bash
# Xem meetings nào đang nhiều participants
kubectl exec -n rinco deploy/webrtc-sfu -- \
  curl -s http://localhost:8102/admin/meetings | jq '.meetings | sort_by(-.participants) | .[0:5]'

# Force-end 1 meeting (last resort)
curl -X POST http://webrtc-sfu:8102/admin/meetings/{id}/end \
  -H "Authorization: Bearer <admin-token>"
```

## Common causes & fixes

### A. Single large meeting (ví dụ: company all-hands 500 người)

**Symptom**: 1 meeting chiếm 60% capacity.

**Mitigation ngay**:

```bash
# 1. Enable simulcast fallback (giảm bandwidth)
curl -X POST http://webrtc-sfu:8102/admin/config \
  -H "Authorization: Bearer <admin-token>" \
  -d '{"simulcast":"on","max_resolution":"480p"}'

# 2. Notify host: "Meeting này có thể bị giảm chất lượng, consider dùng recording + replay"

# 3. Schedule: với > 200 người → chia thành breakout rooms
```

**Long-term fix**: add tier "webinar mode" (1-to-many broadcast, không SFU).

### B. Misconfigured codec (VP9 không hardware encode được)

```bash
# Check codec distribution
kubectl logs -n rinco -l app=webrtc-sfu --tail=100 | grep -E "codec|VP9|AV1"

# Force AV1 (GPU NVENC supported)
kubectl set env deployment/webrtc-sfu -n rinco \
  PREFERRED_CODEC=AV1 \
  HW_ACCEL=NVENC
```

### C. Network bandwidth saturation (upstream provider)

```bash
# Check per-node bandwidth
kubectl exec -n rinco deploy/webrtc-sfu -- \
  iftop -P -n -t -s 30

# If Cloudflare Stream quota hit → enable regional fallback
kubectl set env deployment/webrtc-sfu -n rinco \
  REGIONAL_FAILOVER=on \
  FALLBACK_REGION=ap-southeast-2
```

### D. Memory leak trong SFU

**Symptom**: Memory tăng dần, không giảm sau meeting end.

```bash
# Check RSS
kubectl top pod -n rinco -l app=webrtc-sfu --containers

# Force restart pods (graceful — drain meetings first)
kubectl rollout restart deployment/webrtc-sfu -n rinco \
  --grace-period=120
```

Meetings đang active sẽ tự động reconnect sang pod mới (qua Traefik routing).

### E. Recording service down → SFU buffer overflow

```bash
# Check recording-service
kubectl get pods -n rinco -l app=recording-service

# Nếu down → disable recording trong SFU
kubectl set env deployment/webrtc-sfu -n rinco \
  RECORDING_ENABLED=false
```

## Capacity planning

### Per-pod capacity

| Resolution | Participants / pod | CPU / pod | Bandwidth / pod |
|------------|-------------------|-----------|-----------------|
| 360p | 200 | 4 cores | 500 Mbps |
| 480p | 100 | 4 cores | 800 Mbps |
| 720p | 50 | 4 cores | 1.2 Gbps |
| 1080p | 25 | 4 cores | 2 Gbps |

Total cluster capacity (10 pods @ 4 cores):

- ~500 meetings @ 720p (5,000 participants)
- ~1,000 meetings @ 480p (10,000 participants)

### HPA config

```yaml
# deployments/helm/rinco/templates/webrtc-sfu-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: webrtc-sfu
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: webrtc-sfu
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Pods
    pods:
      metric:
        name: webrtc_sfu_active_meetings
      target:
        type: AverageValue
        averageValue: "30"  # 30 meetings per pod target
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 30
    scaleDown:
      stabilizationWindowSeconds: 300
```

### Node capacity

- 3 dedicated nodes (label `workload=realtime`) với 16 cores / 32GB RAM mỗi node.
- 10 pods per node = 160 cores / 320 GB.
- Taints + tolerations để realtime pods không bị schedule chung với batch.

## Monitoring metrics

| Metric | Threshold | Alert |
|--------|-----------|-------|
| `webrtc_sfu_active_meetings` | > 80% capacity | Warning |
| `webrtc_sfu_active_participants` | > 80% capacity | Warning |
| `webrtc_sfu_cpu_usage` | > 85% | Warning |
| `webrtc_sfu_memory_usage` | > 80% | Warning |
| `webrtc_sfu_network_rx_errors` | > 0.1% | Warning |
| `webrtc_sfu_meeting_join_failure_rate` | > 5% | Critical |
| `webrtc_sfu_packet_loss_p95` | > 2% | Critical |

## After resolution

1. **Verify metrics back to normal** (Grafana dashboard).
2. **Capacity review** — if load > 80% sustained → consider node scale-out.
3. **Postmortem** cho mỗi P0/P1 incident.
4. **Action items**:
   - [ ] Tune HPA threshold.
   - [ ] Add new region (nếu 1 region thường xuyên saturate).
   - [ ] Improve codec selection algorithm.

## Prevention

- [ ] **Test load** với k6 / Artillery trước mỗi release.
- [ ] **Auto-scaling** đã tuned (HPA + cluster autoscaler).
- [ ] **Multi-region deployment** — us-east, ap-southeast-1, eu-west-1.
- [ ] **Recording offload** — không block SFU khi recording fail.
- [ ] **Network QoS** — ưu tiên WebRTC traffic qua router QoS rules.

## References

- [services/webrtc-sfu](../../services/webrtc-sfu)
- [services/recording-service](../../services/recording-service)
- [ARCHITECTURE.md §3.3](../ARCHITECTURE.md#33-real-time-path-webrtc-meeting)
- [ADR-0007 Chat Engine Rust](../adr/0007-chat-engine-rust.md) — same realtime patterns
- [incident-service-down.md](incident-service-down.md)
- [WebRTC scaling guide](https://webrtc.org/getting-started/scaling)