---
name: rinco-deployment
description: Skill về Docker Compose local dev, K3s production, GitOps cho RINCO.
---

# RINCO Deployment Skill

## Local Dev (Docker Compose)

### File Structure
```
infra/docker/
├── docker-compose.yml          # Main
├── docker-compose.override.yml # Dev override
├── profiles/
│   ├── core.yml                # Core services only
│   ├── ai.yml                  # AI services
│   ├── media.yml               # WebRTC/Recording
│   └── observability.yml       # Monitoring stack
└── volumes/
```

### Start Commands
```bash
# All services
docker compose up -d

# Core only (start small)
docker compose --profile core up -d

# Add AI later
docker compose --profile core --profile ai up -d
```

## Production (K3s)

### Cluster Topology
- **Control Plane:** 3 nodes (HA).
- **Edge Nodes:** Auto-scale theo traffic.
- **GPU Pool:** 1+ node cho Recorder + AI.
- **WireGuard Mesh:** Tất cả nodes.

### GitOps Workflow
```
GitHub (rinco repo)
    ↓ (push)
ArgoCD (per cluster)
    ↓ (sync)
K3s Cluster
```

### Manifest Structure
```
infra/k8s/
├── argocd/
│   └── applicationset.yaml
├── tenants/
│   ├── tenant-apexfintech/
│   │   ├── namespace.yaml
│   │   ├── kustomization.yaml
│   │   └── *.yaml
├── services/
│   ├── crm-core/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── ingress.yaml
│   │   ├── configmap.yaml
│   │   ├── secret.yaml (gitignore)
│   │   └── hpa.yaml
└── infra/
    ├── postgres/
    ├── scylladb/
    └── ...
```

## Service Deployment Pattern

### Kubernetes Manifest
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: crm-core
  namespace: rinco
  labels:
    app: crm-core
    version: v1.0.0
spec:
  replicas: 3
  selector:
    matchLabels:
      app: crm-core
  template:
    metadata:
      labels:
        app: crm-core
    spec:
      containers:
      - name: crm-core
        image: ghcr.io/itdoanh/rinco/crm-core:v1.0.0
        ports:
        - containerPort: 8080
        - containerPort: 9090  # metrics
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: crm-core-secrets
              key: database-url
        - name: VALKEY_URL
          valueFrom:
            secretKeyRef:
              name: crm-core-secrets
              key: valkey-url
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "http://otel-collector:4317"
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        securityContext:
          runAsNonRoot: true
          runAsUser: 65534
          readOnlyRootFilesystem: true
          allowPrivilegeEscalation: false
          capabilities:
            drop: [ALL]
---
apiVersion: v1
kind: Service
metadata:
  name: crm-core
spec:
  selector:
    app: crm-core
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
```

### Auto-scaling (HPA)
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: crm-core
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: crm-core
  minReplicas: 3
  maxReplicas: 50
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Pods
    pods:
      metric:
        name: rinco_http_requests_per_second
      target:
        type: AverageValue
        averageValue: "1000"
```

## Database Deployment

### PostgreSQL (StatefulSet + PgBouncer)
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
spec:
  serviceName: postgres
  replicas: 1  # Single primary, replicas for read
  selector:
    matchLabels:
      app: postgres
  template:
    spec:
      containers:
      - name: postgres
        image: postgres:17
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
        - name: init
          mountPath: /docker-entrypoint-initdb.d
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ReadWriteOnce]
      storageClassName: fast-ssd
      resources:
        requests:
          storage: 500Gi
```

### ScyllaDB (DaemonSet cho shard-per-core)
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: scylladb
spec:
  serviceName: scylladb
  replicas: 3
  template:
    spec:
      containers:
      - name: scylladb
        image: scylladb/scylla:6
        resources:
          limits:
            cpu: 8
            memory: 32Gi
        volumeMounts:
        - name: data
          mountPath: /var/lib/scylla
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ReadWriteOnce]
      storageClassName: nvme
      resources:
        requests:
          storage: 1Ti
```

## CI/CD Pipeline

### GitHub Actions
```yaml
name: Build & Deploy

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: go test ./...
      - run: golangci-lint run

  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v5
        with:
          push: true
          tags: ghcr.io/itdoanh/rinco/${{ matrix.service }}:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - name: Update K8s manifests
        run: |
          kubectl set image deployment/${{ matrix.service }} \
            ${{ matrix.service }}=ghcr.io/itdoanh/rinco/${{ matrix.service }}:${{ github.sha }}
```

## Secrets Management

### Sealed Secrets
```bash
# Encrypt secret
kubeseal --format yaml < secret.yaml > sealed-secret.yaml

# Commit sealed-secret.yaml (safe)
git add sealed-secret.yaml
```

### External Secrets (Vault)
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: crm-core-secrets
spec:
  secretStoreRef:
    name: vault
    kind: ClusterSecretStore
  target:
    name: crm-core-secrets
  data:
  - secretKey: database-url
    remoteRef:
      key: secret/data/crm-core
      property: database-url
```

## Database Migrations

### Run migrations trên startup
```go
func RunMigrations() error {
    db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    
    goose.SetBaseFS(os.DirFS("migrations"))
    goose.SetDialect("postgres")
    
    return goose.Up(db, "migrations")
}
```

### Kubernetes Job cho migrations
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: crm-core-migrate
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: ghcr.io/itdoanh/rinco/crm-core:v1.0.0
        command: ["goose", "up"]
      restartPolicy: OnFailure
```

## Rollback Strategy

### ArgoCD Auto-Sync + Self-Heal
```yaml
spec:
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 10s
        factor: 2
        maxDuration: 5m
```

### Manual Rollback
```bash
# Via ArgoCD CLI
argocd app rollback crm-core

# Via kubectl
kubectl rollout undo deployment/crm-core
```

## Production Checklist

- [ ] TLS termination ở Ingress.
- [ ] Network Policies applied.
- [ ] Pod Security Standards: restricted.
- [ ] Resource limits set.
- [ ] Liveness/Readiness probes configured.
- [ ] HPA enabled.
- [ ] PDB (Pod Disruption Budget) set.
- [ ] Monitoring enabled.
- [ ] Backup verified.
- [ ] Secrets externalized.
- [ ] RBAC configured.
- [ ] Audit logging enabled.