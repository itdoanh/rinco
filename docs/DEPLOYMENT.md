# Deployment Guide

Hướng dẫn triển khai RINCO Platform lên production (K3s cluster on bare-metal / VMs).

---

## 1. Prerequisites

### 1.1 Hardware (recommended cho 100K MAU)

| Role | Count | CPU | RAM | Disk | Network |
|------|-------|-----|-----|------|---------|
| **K3s control plane** | 3 | 4 vCPU | 8 GB | 100 GB SSD | 1 Gbps |
| **K3s worker (general)** | 3+ | 8 vCPU | 16 GB | 200 GB SSD | 1 Gbps |
| **K3s worker (media)** | 2+ | 16 vCPU + GPU | 32 GB | 500 GB SSD + 2 TB HDD | 10 Gbps |
| **MinIO node** | 4 (distributed) | 4 vCPU | 8 GB | 4 TB HDD each | 10 Gbps |
| **PostgreSQL primary** | 1 | 4 vCPU | 16 GB | 200 GB SSD | 1 Gbps |
| **PostgreSQL replica** | 2 | 4 vCPU | 16 GB | 200 GB SSD | 1 Gbps |
| **ScyllaDB node** | 3+ | 8 vCPU | 32 GB | 500 GB NVMe | 10 Gbps |

### 1.2 Software

- **K3s** v1.30+ trên tất cả nodes (xem `deployments/k3s/setup-k3s.sh`).
- **kubectl** + **helm** 3.x.
- **ArgoCD** (optional) cho GitOps.
- **cert-manager** v1.14+ với Let's Encrypt cluster issuers.
- **WireGuard** (optional) cho private mesh giữa các region.

### 1.3 Databases (9 cần thiết)

| Database | Phiên bản | HA mode | Lưu ý |
|----------|-----------|---------|-------|
| PostgreSQL | 16 | Primary + 2 streaming replicas + PgBouncer | RLS enabled |
| ScyllaDB | 6.x | 3-node cluster, replication factor 3 | Tune shard count = vCPU |
| MongoDB | 7.x | ReplicaSet 3 nodes | WiredTiger cache = 60% RAM |
| ClickHouse | 24.x | Replicated + Distributed tables | Kafka engine optional |
| Valkey (Redis) | 7.x | Sentinel (1 master + 2 replicas) | AOF + RDB |
| MinIO | latest | Distributed (4 nodes, erasure coding EC:2) | TLS terminator at LB |
| Qdrant | latest | Cluster mode | Snapshot backups daily |
| Meilisearch | latest | Single instance (hoặc 2-node) | Không cần HA mạnh |
| NATS | latest | 3-node cluster với JetStream | Subject-level RLS |

### 1.4 Observability stack

- Prometheus (HA: 2 replicas + Thanos sidecar)
- Grafana (single + daily DB dump)
- Loki (3-node distributed mode + S3 backend)
- Jaeger hoặc Tempo (collector mode + S3 backend)
- OpenTelemetry Collector (DaemonSet)
- Alertmanager (cluster mode)

---

## 2. Deployment Steps

### 2.1 Chuẩn bị K3s cluster

```bash
# Trên control plane nodes
curl -sfL https://get.k3s.io | sh -s - \
  --disable=traefik \
  --write-kubeconfig-mode=644 \
  --cluster-cidr=10.42.0.0/16 \
  --service-cidr=10.43.0.0/16

# Lấy kubeconfig
cat /etc/rancher/k3s/k3s.yaml

# Cài Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Verify
kubectl get nodes
```

### 2.2 Cài infra dependencies (Helm)

```bash
# Traefik ingress
helm repo add traefik https://traefik.github.io/charts
helm install traefik traefik/traefik -n traefik --create-namespace

# cert-manager
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  -n cert-manager --create-namespace \
  --set installCRDs=true

# Apply ClusterIssuers
kubectl apply -f deployments/k3s/cert-issuers.yaml

# ExternalDNS (Cloudflare / Route53)
helm install external-dns ... # theo cloud provider

# MinIO (Operator)
helm install minio-operator minio/minio-operator -n minio --create-namespace

# Apply observability stack
kubectl apply -f deployments/helm/rinco/charts/observability/

# Apply 9 databases (qua Operators hoặc Helm sub-charts)
kubectl apply -f deployments/helm/rinco/charts/databases/
```

### 2.3 Build & push Docker images

```bash
# Login registry
echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USER --password-stdin

# Build tất cả services
./scripts/build.sh

# Hoặc build từng service
docker build -t ghcr.io/itdoanh/rinco/auth-service:v1.0.0      services/auth-service/
docker build -t ghcr.io/itdoanh/rinco/chat-engine:v1.0.0       services/chat-engine/
docker build -t ghcr.io/itdoanh/rinco/landing:v1.0.0           frontend/landing/
# ... (17 backend + 4 frontend)

# Push
docker push ghcr.io/itdoanh/rinco/auth-service:v1.0.0
# ...
```

Hoặc dùng CI/CD:
```bash
git tag v1.0.0
git push origin v1.0.0
# → GitHub Actions sẽ build + push tự động (xem .github/workflows/cd.yml)
```

### 2.4 Apply database migrations

```bash
# Từ máy có quyền truy cập DB nội bộ (hoặc kubectl exec vào bastion)
kubectl run migrate --rm -i --tty \
  --image=ghcr.io/itdoanh/rinco/auth-service:v1.0.0 \
  --restart=Never -- /app/migrate

# Hoặc
./scripts/migrate.sh production
```

### 2.5 Seed dữ liệu (optional, lần đầu)

```bash
# Tạo super-admin tenant + admin user
kubectl exec -it deployment/auth-service -- /app/seed-super-admin \
  --email=admin@rinco.app \
  --password=<secure-password>

# Seed demo data (chỉ dev/staging)
./scripts/seed-demo.sh
```

### 2.6 Deploy tất cả services qua Helm

```bash
# Cập nhật values.yaml với production config
cp deployments/helm/rinco/values.example.yaml deployments/helm/rinco/values-prod.yaml
# Sửa values-prod.yaml: registry, replicas, secrets, ...

# Install
helm install rinco deployments/helm/rinco/ \
  -n rinco --create-namespace \
  -f deployments/helm/rinco/values-prod.yaml

# Verify
kubectl get pods -n rinco
kubectl get svc -n rinco
kubectl get ingress -n rinco
```

### 2.7 DNS + TLS

cert-manager + Let's Encrypt tự động cấp cert khi Ingress được apply.

Verify:
```bash
kubectl get certificate -n rinco
# → READY = True cho mỗi domain

# Test TLS
curl -vI https://api.rinco.app/health
```

---

## 3. Post-deployment

### 3.1 Health checks

```bash
# Smoke test tất cả services
./scripts/health-check.sh production

# Verify metrics
curl https://api.rinco.app/metrics | head -20

# Verify traces
# Truy cập Jaeger UI → đảm bảo có spans mới trong 5 phút gần nhất
```

### 3.2 Backup

| Component | Backup | Frequency | Retention |
|-----------|--------|-----------|-----------|
| PostgreSQL | `pg_basebackup` + WAL archiving (S3) | Continuous (WAL) + daily (full) | 30 days |
| ScyllaDB | `nodetool snapshot` + S3 | Daily | 14 days |
| MongoDB | `mongodump` + S3 | Daily | 14 days |
| ClickHouse | S3 native (table-level) | Daily | 30 days |
| MinIO | Cross-region replication | Continuous | 90 days |
| Qdrant | Snapshot to S3 | Daily | 14 days |
| Etcd (K3s) | `k3s etcd-snapshot` | Hourly | 7 days |

Tất cả backup dùng [Velero](https://velero.io/) cho K8s resources + cron job cho DBs.

### 3.3 Monitoring & alerts

- Truy cập Grafana: `https://grafana.rinco.app` (admin auto-provisioned).
- Pre-loaded dashboards cho mỗi service (xem `infra/grafana/dashboards/`).
- Alerts đã cấu hình trong `infra/prometheus/alerts/`:
  - Service down > 1 phút → Slack #ops-alerts
  - DB connection > 80% pool → Slack #db-alerts
  - Disk usage > 85% → Slack + email
  - Latency p99 > 1s → Slack #perf-alerts
  - Error rate > 5% → PagerDuty

---

## 4. Scaling

### 4.1 Horizontal Pod Autoscaler (HPA)

Mỗi service có HPA mặc định:

| Service | Min replicas | Max replicas | Target metric |
|---------|--------------|--------------|---------------|
| auth-service | 2 | 10 | CPU 70% |
| chat-engine | 2 | 20 | CPU 60% (WebSocket-heavy) |
| webrtc-sfu | 2 | 10 | CPU 70% (bandwidth-bound) |
| lead-scoring | 1 | 5 | CPU 80% (batch) |
| (others) | 2 | 6 | CPU 70% |

### 4.2 Cluster autoscaler

- K3s nodes tự động scale qua `cluster-autoscaler` (Helm chart).
- Trigger: pending pods > 30s.
- Cooldown: 5 phút scale-down, tức thì scale-up.

### 4.3 Database scaling

- **PostgreSQL**: Vertical scale trước, sau đó partition theo `tenant_id`.
- **ScyllaDB**: Thêm node — tự động rebalance trong vài giờ.
- **MongoDB**: Shard theo `tenant_id` khi vượt 500 GB.
- **MinIO**: Thêm node distributed — EC tự rebalance.

---

## 5. Disaster Recovery

### 5.1 RTO / RPO

| Tier | Service | RTO | RPO |
|------|---------|-----|-----|
| Critical | auth, tenant, crm | 15 phút | 0 (sync replicas) |
| High | chat-engine, webrtc-sfu | 30 phút | 5 phút |
| Medium | landing, dynamic-model | 1 giờ | 1 giờ |
| Low | ai-sre, lead-scoring | 4 giờ | 24 giờ |

### 5.2 Runbook

- DB fail → chạy failover tự động (Patroni / Scylla native).
- Region fail → restore từ cross-region backup trong `s3://rinco-backups-dr/`.
- Ransomware → restore từ offline backup (air-gapped S3 bucket).

Xem chi tiết trong `docs/09-security/`.

---

## 6. CI/CD Flow (production)

```
   Developer push to main
            │
            ▼
   ┌─────────────────┐
   │ GitHub Actions  │
   │   ci.yml        │  → lint + test + build
   └────────┬────────┘
            ▼
   ┌─────────────────┐
   │  Build images   │  → 17 backend + 4 frontend
   │  Push GHCR      │  → tag: sha-<short>, latest, semver
   └────────┬────────┘
            ▼
   ┌─────────────────┐
   │  Update Helm    │  → bump Chart.yaml + values.yaml
   │  chart          │  → commit to deployments/helm-charts/
   └────────┬────────┘
            ▼
   ┌─────────────────┐
   │   ArgoCD        │  → sync (canary 10% → 50% → 100%)
   │   (GitOps)      │  → auto-rollback nếu error rate tăng
   └─────────────────┘
```

---

## 7. Rollback

```bash
# Rollback Helm release
helm rollback rinco -n rinco

# Hoặc qua ArgoCD UI
# → Applications → rinco → History → Rollback

# Rollback DB migration (cẩn thận — chỉ khi migration DOWN tồn tại)
./scripts/migrate.sh production down
```

---

## 8. Xem thêm

- [docs/DEVELOPMENT.md](DEVELOPMENT.md) — Dev workflow
- [docs/ARCHITECTURE.md](ARCHITECTURE.md) — Kiến trúc
- [docs/SERVICES.md](SERVICES.md) — Service catalog
- [SECURITY.md](../SECURITY.md) — Security policy
