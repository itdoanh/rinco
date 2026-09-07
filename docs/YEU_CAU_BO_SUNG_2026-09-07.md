# Bổ sung yêu cầu mới (Sep 7, 2026)

## 1. Messenger mã hóa đầu cuối (E2EE)

**Yêu cầu:** Toàn bộ tin nhắn trong chat phải mã hóa end-to-end.

### Thiết kế:
- **Giao thức:** Signal Protocol (X3DH + Double Ratchet)
- **Library Go:** `github.com/libsignal/libsignal-go` hoặc `github.com/gotrueno/olric-e2ee`
- **Quản lý keys:**
  - Identity key pair (long-term, device-bound, stored encrypted)
  - Signed pre-key (rotates weekly)
  - One-time pre-keys (100 keys, replenish khi còn <20)
  - Ephemeral keys (per-session)
- **Key storage:** Server KHÔNG thấy plaintext; chỉ lưu ciphertext + metadata (sender, recipient, timestamp, key id)
- **Device sync:** Mỗi user có thể multi-device; pre-key bundle cho mỗi device; ciphertext replicated sang các device đã authorized
- **Forward secrecy:** Mỗi message dùng ephemeral key riêng → compromise 1 key không lộ lịch sử
- **Post-compromise security:** Double Ratchet tự heal sau khi attacker mất access

### Implementation:
- File: `services/chat-engine/src/crypto/` — Rust (libsodium) hoặc Go (libsignal-go)
- Storage: ScyllaDB lưu `(sender_device_id, recipient_device_id, ciphertext, ratchet_pub, msg_number, ts)`
- Server relay: chỉ forward ciphertext, không thấy plaintext; không thể sửa nội dung
- Group chat: Sender Keys protocol (mỗi member tạo sender key chain, distribute qua pairwise channel)
- Media: mã hóa riêng với AES-GCM-256, key wrap qua Signal session; upload encrypted blob lên S3

### Tài liệu tham chiếu:
- Signal Protocol: https://signal.org/docs/
- libsignal-go: https://github.com/libsignal/libsignal
- Olm/Megolm (Matrix): tham khảo (đơn giản hơn nhưng kém security hơn)

## 2. Tiered Object Storage (SSD cho file mới, HDD cho file cũ)

**Yêu cầu:** 
- File mới (trong khoảng thời gian nhất định) → S3 SSD (high-performance)
- File cũ → S3 HDD (low-cost)
- Tự động transition theo policy

### Thiết kế:

**2 buckets:**
- `rinco-hot-ssd` — MinIO NVMe tier, NVMe SSD backend
- `rinco-cold-hdd` — MinIO HDD tier, HDD/SATA backend
- Có thể dùng MinIO tiering config hoặc tách 2 deployment

**Lifecycle policy (default):**
- File age 0-30 ngày → hot SSD
- File age 30-90 ngày → warm (optional) HDD
- File age >90 ngày + không truy cập 30 ngày → cold HDD

**Implementation:**
- Mỗi file lưu metadata: `tier`, `created_at`, `last_access_at`, `access_count_30d`, `size_bytes`
- Background worker (cron mỗi ngày) chạy transition:
  - `tiered_storage_worker` service (Go) đọc metadata từ PostgreSQL
  - MinIO client: `mc cp` giữa buckets, set tier tag
  - Update metadata sau khi copy xong
  - Set retention policy (Glacier-like) nếu > 365 ngày
- Smart cache layer: khi client request URL → check metadata tier → proxy đến bucket tương ứng
- Presigned URL generation: chỉ trả URL cho bucket tương ứng

**Buckets config:**
```yaml
# infra/minio/buckets-hot.yaml
buckets:
  - name: rinco-hot-ssd
    quota: 2TB
    placement: nvme-pool
    version: enabled
    lifecycle:
      - id: transition-to-cold
        enabled: true
        days: 30

  - name: rinco-cold-hdd
    quota: 100TB
    placement: hdd-pool
    version: disabled
    lifecycle:
      - id: expire
        enabled: true
        days: 2555 # 7 years retention
```

**Per-tenant tier override:**
- Pro plan: hot tier 60 ngày
- Enterprise: hot tier 365 ngày
- Free: hot tier 7 ngày
- Configurable trong `tenant_settings.tier_policy` JSONB

**Cost estimation:**
- SSD tier: $0.023/GB/month (AWS S3 Standard equivalent)
- HDD tier: $0.004/GB/month (AWS S3 IA equivalent)
- Tiết kiệm ~83% cho data >30 ngày tuổi

### Tài liệu tham chiếu:
- MinIO tiering: https://min.io/docs/minio/linux/administration/object-management/object-lifecycle-management.html
- AWS S3 Intelligent-Tiering: https://aws.amazon.com/s3/storage-classes/intelligent-tiering/

## Trạng thái

Cả 2 yêu cầu cần được tích hợp vào:
- `services/chat-engine/` (E2EE)
- `services/landing-service/`, `services/recording-service/`, `services/email-service/` (tiered storage)
- `infra/minio/` (tier config)
- `docs/06-chat-engine/README.md` (E2EE section)
- `docs/02-tenant-site/README.md` (tiered storage section)
