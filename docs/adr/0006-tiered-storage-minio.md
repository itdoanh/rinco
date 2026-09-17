# ADR-0006: Tiered Storage on MinIO

> **Status**: Accepted · **Date**: 2026-09-18 · **Authors**: RINCO Platform Team

## Context

RINCO lưu trữ nhiều loại object trên MinIO (S3-compatible):

| Object | Size | Access pattern | Volume |
|--------|------|----------------|--------|
| **Recording video** (WebRTC) | 100MB–2GB/recording | Hiếm sau 30 ngày | ~5TB/tháng |
| **Landing assets** (images, fonts) | 10KB–5MB | Hot (CDN cache) | ~500GB/tháng |
| **Invoice PDF** (billing) | 50KB–200KB | Hiếm sau khi gửi | ~10GB/tháng |
| **User upload** (chat, profile) | 100KB–50MB | Tùy user | ~1TB/tháng |
| **AI artifacts** (transcripts, summary) | 10KB–500KB | Thường | ~100GB/tháng |
| **Backup snapshot** (DB dump) | 10GB–100GB | Cực hiếm (DR) | ~5TB/tháng |

**Cost issue**: lưu tất cả trên SSD = đắt. Lưu tất cả trên HDD = lâu (latency cao cho hot data).

Giải pháp: **tiered storage** tự động chuyển data từ SSD → HDD dựa trên age + access pattern.

## Decision

Dùng **MinIO với erasure coding + ILM (Information Lifecycle Management) + 2 tiers**:

```
┌─────────────────────────────────────────────────────────┐
│  Bucket: rinco-data                                     │
│                                                         │
│  ┌─────────────────────┐  ┌─────────────────────────┐   │
│  │  TIER 1: SSD (hot)  │  │  TIER 2: HDD (cold)     │   │
│  │  NVMe 4TB × 12      │  │  SATA 8TB × 12          │   │
│  │  ~30 TB raw         │  │  ~80 TB raw             │   │
│  │  ~10 TB usable (EC) │  │  ~40 TB usable (EC)     │   │
│  │                     │  │                         │   │
│  │  Latency < 5ms      │  │  Latency ~30ms          │   │
│  │  Cost: $$$          │  │  Cost: $                │   │
│  └─────────────────────┘  └─────────────────────────┘   │
│                                                         │
│  ILM rules:                                             │
│  - 0–30 ngày: TIER 1 (SSD)                              │
│  - 30+ ngày: chuyển sang TIER 2 (HDD)                   │
│  - 365 ngày: xóa (trừ backup bucket)                    │
└─────────────────────────────────────────────────────────┘
```

### MinIO setup

**Erasure coding**: EC:4 (4 data shards + 4 parity shards) → có thể chịu 4 disk fail đồng thời mà không mất data.

**Server pools**: 2 pools (tier1 SSD + tier2 HDD) trong cùng 1 MinIO deployment.

```yaml
# docker-compose snippet
services:
  minio:
    image: minio/minio:RELEASE.2024-09-13T20-26-02Z
    command: server /data{1...12} /hdd{1...12}
    environment:
      MINIO_STORAGE_CLASS_STANDARD: "EC:4"
      MINIO_BROWSER_REDIRECT_URL: https://s3.rinco.vn
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - ssd-data:/data1
      ...
      - hdd-data:/hdd1
      ...
```

### Bucket layout

```
rinco-data/                            # Hot bucket
├── recordings/{tenant_id}/{year}/{month}/{recording_id}.mp4
├── landing/{tenant_id}/assets/...
├── uploads/{tenant_id}/{user_id}/{file_id}
├── invoices/{tenant_id}/{year}/{invoice_id}.pdf
└── artifacts/{tenant_id}/{artifact_type}/{id}

rinco-backup/                          # Cold bucket
├── postgres/{db}/{date}.dump
├── scylla/{date}/...
├── mongo/{date}/...
└── minio-snapshot/{date}/...          # Cross-region replica
```

### ILM rules (MinIO mc client)

```bash
# Tier transition: SSD → HDD sau 30 ngày
mc ilm rule add rinco/rinco-data \
  --transition-tier TIER_HDD \
  --transition-days 30

# Expiry: xóa sau 365 ngày (không áp dụng cho backup bucket)
mc ilm rule add rinco/rinco-data \
  --expiry-days 365
```

Riêng `rinco-backup`: không expire (giữ 5 năm cho compliance).

### Cross-region replication

`rinco-backup` replicate sang region thứ 2 (vd: `ap-southeast-2` AUS):

```bash
mc replicate add source/bucket \
  --destination https://s3-ap-southeast-2.rinco.vn \
  --replicate "delete,delete-marker,replica-delete-marker"
```

### Access patterns qua SDK

Code dùng MinIO SDK (Go: `minio-go`, Rust: `rust-s3`, Python: `boto3`):

```go
// Upload — luôn vào SSD tier
client.PutObject(ctx, "rinco-data", key, reader, size, minio.PutObjectOptions{
    StorageClass: "STANDARD",  // SSD
    Metadata: map[string]string{"tenant_id": tenantID},
})

// Download — tự động từ tier nào
client.GetObject(ctx, "rinco-data", key, minio.GetObjectOptions{})
```

### Lifecycle policies chi tiết

| Object type | SSD (0-30d) | HDD (30-365d) | Sau 365d |
|-------------|-------------|---------------|----------|
| Recording | ✅ | ✅ | Delete (trừ nếu user lưu trữ) |
| Landing asset | ✅ | ✅ | Delete (CDN cache vẫn serve từ edge) |
| Invoice PDF | ✅ | ✅ (compliance 7 năm) | Move sang Glacier |
| User upload | ✅ | ✅ | Delete |
| AI artifact | ✅ | ✅ | Delete |
| Backup | ❌ (luôn HDD) | ✅ | Move sang Glacier sau 1 năm |

### Presigned URL

Cho direct upload từ client (tránh qua backend):

```go
url, _ := client.PresignedPutObject(ctx, "rinco-data", key, 15*time.Minute)
// Client PUT trực tiếp → giảm tải backend
```

### Encryption

- **SSE-KMS** với key rotation 90 ngày.
- Key lưu trong HashiCorp Vault.
- Bucket policy: enforce SSE-KMS, deny unencrypted upload.

### Monitoring

| Metric | Alert threshold |
|--------|-----------------|
| `minio_disk_usage{role=tier1}` > 80% | Warning |
| `minio_disk_usage{role=tier1}` > 90% | Critical — manual migrate |
| `minio_disk_usage{role=tier2}` > 85% | Warning |
| `minio_tier_transition_lag` > 1 day | Warning |
| `minio_request_errors` > 0.1% | Warning |
| `minio_replication_lag` > 1 hour | Critical |

## Consequences

### Positive

- **Cost giảm 60-70%** so với all-SSD (5TB SSD/tháng vs 2TB SSD + 3TB HDD).
- **Hot data latency vẫn nhanh** (< 5ms cho recording trong 30 ngày đầu).
- **Cross-region DR** — disaster recovery tự động.
- **SSE-KMS** — encryption tự động cho mọi object.

### Negative

- **Tier transition overhead** — job chạy hàng ngày, tốn IO.
- **HDD slower** — nếu user mở recording cũ → 30ms latency.
- **Multi-region replication lag** — nếu internet chậm → backup delay.
- **Operational complexity** — phải monitor 2 pools.

### Mitigations

- **Tier transition** chạy off-peak (02:00 ICT).
- **Pre-fetch** — nếu user mở recording cũ → trigger upgrade lên SSD trong 24h.
- **Replication lag** monitor + alert; manual failover nếu lag > 1h.
- **Helm chart** auto-configure MinIO với 2 pools.

## Alternatives Considered

### A. AWS S3 with Intelligent-Tiering

- **Pro**: managed, automatic tiering.
- **Con**: vendor lock-in, data egress cost, latency Internet.
- **Verdict**: ❌ Rejected — multi-cloud strategy cần self-hosted.

### B. Single tier (all-SSD)

- **Pro**: latency đồng nhất.
- **Con**: cost × 3-5 cho HDD.
- **Verdict**: ❌ Rejected — cost không bền vững.

### C. Ceph RADOS

- **Pro**: scale rất lớn.
- **Con**: phức tạp hơn MinIO, ít tính năng S3-compatible.
- **Verdict**: ❌ Rejected — overkill cho 50TB total.

### D. Local filesystem + rsync

- **Pro**: đơn giản nhất.
- **Con**: không có replication, không có S3 API.
- **Verdict**: ❌ Rejected — không đáp ứng S3 API cho landing/chat.

## References

- [MinIO documentation](https://min.io/docs/)
- [ARCHITECTURE.md §5](../ARCHITECTURE.md#5-polyglot-persistence--tại-sao)
- [services/recording-service](../../services/recording-service) — primary consumer
- [runbook: incident-cost-spike](../runbooks/incident-cost-spike.md)
- [ADR-0001 Polyglot Persistence](0001-polyglot-persistence.md)