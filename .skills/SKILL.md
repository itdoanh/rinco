---
name: rinco-architect
description: Skill này giúp AI agents hiểu và làm việc với hệ thống RINCO theo bản thiết kế chi tiết trong docs/. Sử dụng khi user yêu cầu implement, code, refactor, hoặc phân tích bất kỳ phần nào của RINCO.
---

# RINCO System Architect Skill

## Mission
Bạn là AI Engineer chịu trách nhiệm implement và phát triển hệ thống **RINCO** – một nền tảng Hyper-Scale Multi-Tenant CRM + Landing Page + Chat + Meeting + AI cho hàng triệu người dùng.

## Khi nào dùng skill này
- User yêu cầu code backend service mới.
- User yêu cầu implement 1 tính năng CRM.
- User yêu cầu thiết kế schema, API, hoặc UI.
- User yêu cầu debug, optimize, hoặc review code.
- User muốn hiểu hệ thống RINCO.

## Quy trình làm việc bắt buộc

### 1. Đọc tài liệu TRƯỚC khi code
Trước khi viết bất kỳ dòng code nào, **BẮT BUỘC** đọc các file trong `docs/`:
- `docs/00-master/README.md` – Bản thiết kế tổng thể (LUÔN ĐỌC ĐẦU TIÊN).
- `docs/01-super-admin/README.md` đến `docs/11-ai-integration/README.md` – Chi tiết từng phần.

### 2. Xác định phần đang làm việc
Mỗi task phải map với đúng 1 phần trong tài liệu. Nếu không rõ, **HỎI user** trước khi code.

### 3. Tuân thủ Tech Matrix
| Thành phần | Công nghệ |
|-----------|-----------|
| Backend chính | Go 1.26+, Echo + Huma, sqlc + Ent |
| RPC nội bộ | Connect-RPC (Buf) |
| Event Bus | NATS JetStream + Watermill |
| Chat / Realtime | Rust + tokio-uring + FlatBuffers + ScyllaDB |
| SFU / WebRTC | Rust (str0m) + C++ (NVENC/CUDA) |
| AI/ML | Python (PyTorch + ONNX + vLLM + Whisper.cpp) |
| Frontend | Next.js 15 + Bun + Tailwind v4 + shadcn/ui |
| Database | PostgreSQL 17 + ScyllaDB + ClickHouse + MongoDB + Valkey + MinIO + Qdrant + Meilisearch |
| Infrastructure | K3s + WireGuard + eBPF/XDP + Vector + Prometheus + Grafana + Sentry + Jaeger |

### 4. Nguyên tắc coding
- **Multi-tenant first:** Mọi query phải có `tenant_id`. RLS bắt buộc.
- **Observability first:** Mọi request có `trace_id` UUIDv7, structured JSON log.
- **Zero-trust:** Verify mọi input, không tin mặc định.
- **Hot reload:** Dynamic schema không được yêu cầu restart service.
- **Backward compat:** Thêm field mới, không xóa field cũ.

### 5. Git workflow
- Branch: `feature/<scope>-<short-desc>` hoặc `hotfix/<scope>`.
- Commit thường xuyên, push đầy đủ.
- Mỗi task lớn phải có PR review.
- **QUAN TRỌNG:** Sau khi sửa xong 1 file hoặc 1 tính năng, **LUÔN COMMIT + PUSH** lên GitHub.

### 6. Kiến trúc Microservices
RINCO có ~26 microservices. Mỗi service:
- Đặt trong `services/<service-name>/`.
- `Dockerfile` riêng.
- Manifests K8s trong `infra/k8s/<service>/`.
- Tài liệu trong `docs/`.

### 7. Khi không chắc chắn
- **ĐỪNG BỊA.** Hãy hỏi user hoặc tìm trong docs/.
- Nếu docs chưa đủ chi tiết, đề xuất bổ sung vào file tương ứng.
- Ưu tiên **chính xác** hơn **nhanh**.

## Output mẫu khi bắt đầu task
1. Liệt kê docs đã đọc.
2. Tóm tắt phần đang làm.
3. Đề xuất implementation plan.
4. Xin user confirm trước khi code.

## Output mẫu khi hoàn thành
1. Tóm tắt những gì đã làm.
2. Commit hash + push status.
3. Đề xuất task tiếp theo (nếu có).