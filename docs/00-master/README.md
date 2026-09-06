# RINCO – Bản Thiết Kế Hệ Thống Tổng Thể (Master Design)

> **Phiên bản:** v1.1 (mở rộng)  
> **Tên dự án:** RINCO  
> **Slogan:** *Kết Nối Toàn Năng – Vững Vàng Quản Trị*  
> **Tầm nhìn:** Nền tảng Quản trị & Kết nối Doanh nghiệp Quốc dân.  
> **Triết lý:** Bình dân hóa công nghệ đỉnh cao – Phụng sự sự tăng trưởng của cộng đồng doanh nghiệp.  
> **Bộ giá trị cốt lõi:** R – Responsibility, I – Integrity, N – Nurture, C – Cohesion, O – Optimization.

---

## Mục Lục

1. [Bối cảnh & Bài toán](#1-bối-cảnh--bài-toán)
2. [Mục tiêu sản phẩm](#2-mục-tiêu-sản-phẩm)
3. [Kiến trúc tổng quan (High-Level)](#3-kiến-trúc-tổng-quan-high-level)
4. [Ma trận công nghệ (Tech Matrix)](#4-ma-trận-công-nghệ-tech-matrix)
5. [Cấu trúc Microservices](#5-cấu-trúc-microservices)
6. [Polyglot Persistence – Chiến lược đa cơ sở dữ liệu](#6-polyglot-persistence--chiến-lược-đa-cơ-sở-dữ-liệu)
7. [Bốn phân hệ CRM (4 Phần Chính)](#7-bốn-phân-hệ-crm-4-phần-chính)
8. [Landing Page & Facebook CAPI](#8-landing-page--facebook-capi)
9. [Chat Real-time Engine](#9-chat-real-time-engine)
10. [WebRTC SFU & Recording](#10-webrtc-sfu--recording)
11. [Observability 4 tầng + AI SRE](#11-observability-4-tầng--ai-sre)
12. [Bảo mật Multi-Tenant Zero-Trust](#12-bảo-mật-multi-tenant-zero-trust)
13. [Tích hợp AI toàn hệ thống](#13-tích-hợp-ai-toàn-hệ-thống)
14. [Cơ sở hạ tầng đề xuất](#14-cơ-sở-hạ-tầng-đề-xuất)
15. [Lộ trình triển khai](#15-lộ-trình-triển-khai)
16. [Danh sách tài liệu chi tiết](#16-danh-sách-tài-liệu-chi-tiết)

**Mục lục mở rộng (phần bổ sung – v1.1)**

17. [Audit – Đánh giá nội dung hiện tại](#17-audit--đánh-giá-nội-dung-hiện-tại)
18. [Edge Cases & Error Scenarios chi tiết](#18-edge-cases--error-scenarios-chi-tiết)
19. [Implementation Roadmap chi tiết](#19-implementation-roadmap-chi-tiết)
20. [Testing Strategy End-to-End](#20-testing-strategy-end-to-end)
21. [Migration Plan cho Schema & Service](#21-migration-plan-cho-schema--service)
22. [Disaster Recovery](#22-disaster-recovery)
23. [Cost Estimation (rough numbers)](#23-cost-estimation-rough-numbers)
24. [Performance Budget & SLO Matrix](#24-performance-budget--slo-matrix)
25. [Concrete Open Questions / TBD](#25-concrete-open-questions--tbd)

---

## 1. Bối cảnh & Bài toán

### 1.1. Bối cảnh
RINCO được sinh ra để giải quyết bài toán **"Quản trị đa doanh nghiệp – Đa cấp – Đa kênh"** với 3 đặc thù:
- **Quy mô siêu lớn (Hyper-Scale):** Hàng triệu CCU (Concurrent Users), hàng triệu request/s từ Landing Page.
- **Đa đối tượng:** Admin hệ thống, nhiều doanh nghiệp đối tác, nhân viên theo sơ đồ cây, khách hàng cuối.
- **Đa dịch vụ:** Landing Page, CRM, Chat, Call/Meeting, AI, Analytics, Marketing Automation.

### 1.2. Bài toán cốt lõi
| # | Bài toán | Yêu cầu bắt buộc |
|---|----------|------------------|
| 1 | **Thu thập Lead** từ Landing Page đạt tốc độ cực cao, chính xác, chống giả mạo | FCP < 0.4s, ghi trăm nghìn form/s |
| 2 | **Đồng bộ Lead ↔ CRM ↔ Facebook CAPI** theo thời gian thực, có HMAC chống giả mạo | Phản hồi < 50ms |
| 3 | **Quản lý nhiều doanh nghiệp (Multi-Tenant)** trên cùng hạ tầng, cách ly tuyệt đối | RLS + eBPF + Mesh Network |
| 4 | **Phân cấp nhân sự hình cây** với khả năng tạo link mời theo cấp | PostgreSQL LTREE + PASETO Token |
| 5 | **Trình sinh mô hình doanh nghiệp động** – mỗi ngành có schema khác nhau | Meta-Schema + JSON Schema |
| 6 | **Chat real-time** triệu user với độ trễ < 50ms | FlatBuffers + io_uring + ScyllaDB |
| 7 | **Gọi thoại / Video / Meeting** Sub-50ms | Rust SFU + AV1 SVC + eBPF/XDP |
| 8 | **Ghi âm cuộc họp** chi phí thấp, AI tóm tắt | GPU NVENC + Whisper.cpp |
| 9 | **Phát hiện & xử lý lỗi** trong vài giây ở hàng trăm VPS | Observability 4 tầng |
| 10 | **An ninh đa tầng** chống DDoS, Anti-Bot, Rò rỉ dữ liệu chéo | Zero-Trust + Dark Admin |

---

## 2. Mục tiêu sản phẩm

### 2.1. Mục tiêu chiến lược (Business Goals)
- **OG1:** Cung cấp nền tảng CRM + Landing Page + Chat + Meeting cho **≥ 10.000 doanh nghiệp** trong năm đầu tiên.
- **OG2:** Giảm **≥ 80% chi phí hạ tầng** so với mô hình SaaS truyền thống nhờ Kernel Bypass + GPU Offload.
- **OG3:** Đảm bảo **độ trễ cuối cùng**:
  - Landing Page Load: < 0.4s (FCP), < 0.8s (LCP).
  - Lead Ingestion → CRM: < 50ms.
  - Chat gửi/nhận: < 50ms.
  - WebRTC gọi thoại: < 50ms, video: < 100ms.

### 2.2. Mục tiêu kỹ thuật (Engineering Goals)
- **EG1:** Mọi microservice có **p99 latency < 50ms** ở 80% các đường dẫn nóng.
- **EG2:** Khả năng **scale ngang không giới hạn** bằng Shard-per-Core (ScyllaDB) + K8s HPA.
- **EG3:** **Self-Healing** – Container crash phục hồi < 2s, mất node tự động reroute < 30s.
- **EG4:** **Zero-Trust Multi-Tenant** – Không bao giờ để tenant này đọc/ghi dữ liệu tenant khác dù có lỗ hổng SQLi.
- **EG5:** **Observability First** – Mọi request đều có `trace_id` UUIDv7, có log JSON có cấu trúc, có metric Prometheus.

### 2.3. Mục tiêu văn hóa (Culture Goals – RINCO Code of Conduct)
- **CG1:** Triết lý "Cây Cổ Thụ" (Treelike Governance) – Rễ (Admin), Thân (Quản lý), Cành (Nhân viên), Trái (Cộng đồng).
- **CG2:** Văn hóa "Không Khoảng Cách" – Đề xuất của nhân viên mới ngang hàng với quản lý lâu năm nếu giải pháp tốt hơn.
- **CG3:** Meritocracy – Thăng tiến tự động theo số liệu thực tế.
- **CG4:** Ritual: Pulse Check (10 phút/ngày), Open Tree Day (1 lần/tháng), RINCO Heroes (1 lần/quý), Community Impact Day (1 lần/năm).

---

## 3. Kiến trúc tổng quan (High-Level)

### 3.1. Sơ đồ tổng quan 6 tầng

```
┌─────────────────────────────────────────────────────────────────────────┐
│  TẦNG 6 — CLIENTS                                                        │
│  Landing Page (Next.js + Tailwind + Lucide) │ Admin Web (React/Vite)     │
│  CRM Web (React) │ Mobile App (React Native/Flutter) │ Desktop (Electron)│
└─────────────────────────────────────────────────────────────────────────┘
                                    │ HTTPS / WSS / WebTransport
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  TẦNG 5 — EDGE & GATEWAY (Rust + Go + eBPF)                              │
│  • eBPF/XDP Anti-DDoS + JA4+ Fingerprint                                 │
│  • io_uring Zero-Copy Gateway (Rust – tokio-uring)                       │
│  • Echo/Huma REST + Connect-RPC Gateway (Go)                            │
│  • Valkey Domain Map (<0.5ms tenant resolution)                          │
│  • Bẫy Headless: Wasm Attestation + Argon2 PoW                           │
└─────────────────────────────────────────────────────────────────────────┘
                                    │ FlatBuffers / Connect-RPC / REST
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  TẦNG 4 — BUSINESS MICROSERVICES (Go 1.26 + Rust + TS BFF + Python)      │
│  Landing Ingest │ CRM Core │ Auth │ Tree Org │ Notification │ Media SFU │
│  Recorder │ AI Scoring │ Chat Engine │ Meeting Orchestrator │ BFF       │
└─────────────────────────────────────────────────────────────────────────┘
                                    │ NATS JetStream + Valkey Streams
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  TẦNG 3 — EVENT BUS & MESSAGING                                          │
│  NATS JetStream (primary) │ Valkey Streams (event log) │ Kafka (heavy)  │
│  Watermill abstraction (Publisher/Subscriber)                            │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  TẦNG 2 — DATA PLANE (Polyglot Persistence)                              │
│  ScyllaDB (Chat & Event) │ PostgreSQL 17 (CRM Core)                     │
│  MongoDB (Dynamic Form) │ ClickHouse (Analytics) │ Valkey (Cache/State)  │
│  Qdrant / pgvector (Vector RAG) │ MinIO/S3 (Media) │ Meilisearch (Search)│
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  TẦNG 1 — INFRASTRUCTURE & ORCHESTRATION                                │
│  K3s (Orchestrator) │ WireGuard Mesh (Headscale) │ eBPF/XDP (Network)  │
│  GPU Pool (NVENC/TensorRT) │ Vector Agent (Logs) │ Prometheus Exporter  │
│  S3/MinIO │ Backup Vault │ Anti-DDoS Appliance                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.2. Nguyên tắc kiến trúc (Architectural Principles)
| # | Nguyên tắc | Áp dụng |
|---|-----------|---------|
| AP1 | **Polyglot Persistence** | Mỗi service chọn DB tối ưu cho access pattern |
| AP2 | **Zero-Copy & Kernel Bypass** | Rust + io_uring cho data path nóng |
| AP3 | **Event-First** | Mọi thay đổi nghiệp vụ phát sự kiện qua NATS |
| AP4 | **Zero-Trust Multi-Tenant** | RLS + Tenant Context ở mọi query |
| AP5 | **Observability by Default** | trace_id + structured log + metric ở mọi service |
| AP6 | **Graceful Degradation** | Lỗi một phần không sập toàn hệ thống |
| AP7 | **Self-Healing** | K3s Probe + Circuit Breaker + Auto-Rollback |
| AP8 | **Boring Tech Where Possible** | Dùng công nghệ đã chứng minh; chỉ dùng "tân tiến" khi cần |

---

## 4. Ma trận công nghệ (Tech Matrix)

### 4.1. Ngôn ngữ chính & phụ

| Ngôn ngữ | Vai trò | Lý do chọn | Bài toán giải quyết |
|----------|---------|-----------|---------------------|
| **Go 1.26+** | Cốt lõi Backend (API, BFF, Service) | Goroutines nhẹ, static binary, ecosystem chín muồi | API Gateway, CRM Core, Auth, Notification, Ingestion |
| **Rust 1.85+** | Low-latency, Media SFU, Edge | Zero-cost abstraction, ownership an toàn, không GC | WebRTC SFU, Egress Recorder, Chat Gateway (io_uring) |
| **C++23** | Phần cứng-tầng (NVENC, CUDA, eBPF) | Hiệu năng tiệm cận C, hỗ trợ GPU trực tiếp | Egress Worker, eBPF filters, Custom Protocol |
| **TypeScript (Bun + Next.js)** | Frontend + BFF | Tái sử dụng Type, ecosystem frontend | Landing Page, Admin Web, CRM Web, BFF |
| **Python 3.13** | AI/ML Pipeline | Ecosystem AI/ML phong phú | Predictive Scoring, RAG, Whisper orchestrator |

### 4.2. Thư viện chuẩn cho mỗi ngôn ngữ

#### 4.2.1. Go Ecosystem

| Phân loại | Thư viện | Mục đích |
|-----------|---------|---------|
| API Framework (REST/HTTP) | **Echo** + **Huma** | Routing nhanh + Type-safe OpenAPI |
| RPC nội bộ | **Connect-RPC** (Buf) | gRPC + HTTP/1.1, browser-friendly |
| Database SQL | **sqlc** | Type-safe từ SQL thuần, không runtime overhead |
| Database ORM | **Ent** (Facebook) | Graph-based schema, không reflection |
| Dependency Injection | **Uber FX** | Lifecycle + IoC tự động |
| Event-Driven | **Watermill** | Pub/Sub chuẩn hóa cho NATS/Kafka/Valkey |
| Validation | **protovalidate** | CEL trong file `.proto` |
| Logging | **log/slog** + **Zap** | Structured JSON |
| Tracing | **OpenTelemetry Go SDK** | Distributed Tracing |
| Migrations | **goose** + **sqlc** | Versioned SQL migration |
| Config | **Viper** hoặc **envconfig** | Config 12-factor |
| Crypto | **PASETO v4** + **AGE** | Token stateless + encryption |
| Circuit Breaker | **gobreaker** + **sony/gobreaker** | Cách ly lỗi downstream |
| WebSocket | **nhooyr/websocket** | Realtime gateway |
| UUID | **google/uuid** (v7) | Trace ID có timestamp |

#### 4.2.2. Rust Ecosystem

| Phân loại | Thư viện | Mục đích |
|-----------|---------|---------|
| Async runtime | **Tokio** + **glommio** | CPU-pinned async |
| io_uring | **tokio-uring** | Zero-copy I/O |
| WebSocket | **tokio-tungstenite** + **fastwebsockets** | Realtime |
| WebRTC | **str0m** + **webrtc-rs** | Native WebRTC stack |
| Media codec | **rav1e** + **vpx-rs** + **dav1d-rs** | AV1/VP9 codec |
| GPU | **cudarc** + **ash** (Vulkan) | CUDA + Vulkan bindings |
| Serialization | **flatbuffers** + **rkyv** | Zero-parse |
| Logging | **tracing** + **tracing-subscriber** | Structured log + span |
| eBPF | **aya** | eBPF program loader |
| Tonic gRPC | **tonic** | gRPC cho internal |
| Redis/Valkey | **redis-rs** + **fred** | Cache client |
| Scylla | **scylla-rust-driver** | CQL client |
| Postgres | **sqlx** | Async SQL |

#### 4.2.3. Frontend (TypeScript) Ecosystem

| Phân loại | Thư viện | Mục đích |
|-----------|---------|---------|
| Framework | **Next.js 15** + **Bun runtime** | SSR + RSC |
| Styling | **Tailwind CSS v4** | Utility-first |
| Icons | **Lucide Icons** | SVG icons |
| Animation | **Framer Motion** + **Motion One** | 60fps animation |
| Component | **shadcn/ui** + **Radix UI** | Accessible primitives |
| State | **Zustand** + **Jotai** | Lightweight reactive |
| Server State | **TanStack Query** | Cache + sync |
| Form | **React Hook Form** + **Zod** | Validation |
| Realtime | **WebTransport** polyfill + **WebSocket** | Realtime |
| WebRTC | **mediasoup-client** | Client WebRTC |
| Chart | **Recharts** + **Tremor** | Dashboard chart |
| Crypto | **libsodium-wrappers** (PASETO v4) | Token verify client-side |

#### 4.2.4. Python Ecosystem (AI)

| Phân loại | Thư viện | Mục đích |
|-----------|---------|---------|
| ML Framework | **PyTorch** + **ONNX Runtime** | Train & inference |
| Classical ML | **XGBoost** + **LightGBM** + **scikit-learn** | Tabular scoring |
| LLM Serving | **vLLM** + **TensorRT-LLM** | High-throughput LLM |
| Speech-to-Text | **Whisper.cpp** + **faster-whisper** | STT |
| LLM Client | **LangChain** + **LlamaIndex** | RAG orchestration |
| Embedding | **sentence-transformers** + **BGE-M3** | Vector embedding |
| MLOps | **MLflow** + **DVC** | Experiment tracking |

---

## 5. Cấu trúc Microservices

### 5.1. Danh sách Microservices (MVP)

| # | Service | Ngôn ngữ | Trách nhiệm | DB chính |
|---|---------|----------|-------------|----------|
| 1 | **api-gateway** | Go (Echo) | Routing, Auth, Rate Limit, Tenant Resolver | Valkey |
| 2 | **edge-gateway** | Rust (tokio-uring) | io_uring Zero-Copy, Wasm Challenge | Valkey |
| 3 | **landing-ingest** | Go (Huma) | Nhận form, HMAC, anti-bot | ScyllaDB |
| 4 | **crm-core** | Go (Ent) | Lead, Contact, Deal, Activity | PostgreSQL |
| 5 | **tree-org** | Go (sqlc) | Cây tổ chức, phân quyền | PostgreSQL (LTREE) |
| 6 | **dynamic-schema** | Go (Huma) | Meta-Schema Engine, JSON Schema | PostgreSQL |
| 7 | **auth-service** | Go | PASETO, RBAC, FIDO2/WebAuthn | PostgreSQL |
| 8 | **chat-engine** | Rust | WebSocket/WebTransport, FlatBuffers | ScyllaDB + Valkey |
| 9 | **media-sfu** | Rust (str0m) | WebRTC SFU, eBPF/XDP | Valkey (room state) |
| 10 | **recorder** | C++ (NVENC) | GPU Composite + Chunked Upload | MinIO |
| 11 | **ai-scoring** | Python (XGBoost) | Lead Scoring, pLTV | PostgreSQL |
| 12 | **ai-conversation** | Python (vLLM) | Chatbot + RAG | Qdrant + Valkey |
| 13 | **ai-media** | Python (Whisper.cpp) | STT, Summarize, Action Items | PostgreSQL |
| 14 | **ai-sre** | Python (DeepSeek-Coder) | RCA, Hotfix Proposal | Vector DB + ClickHouse |
| 15 | **analytics** | Go | OLAP Query, Real-time Dashboard | ClickHouse |
| 16 | **meta-capi** | Go | Facebook CAPI Gateway + Feedback | ScyllaDB + NATS |
| 17 | **notification** | Go | Email/Push/Telegram/SMS | PostgreSQL |
| 18 | **billing** | Go | Subscription, Invoice, Usage tracking | PostgreSQL |
| 19 | **tenant-manager** | Go | Tạo tenant, gán VPS, deploy config | PostgreSQL + K3s API |
| 20 | **mesh-controller** | Go (Headscale) | WireGuard mesh network | KV store |
| 21 | **search-service** | Go (Meilisearch client) | Full-text search CRM | Meilisearch |
| 22 | **tenant-site** | TS (Next.js) | Trang chủ công ty đối tác | (Static + API) |
| 23 | **landing-renderer** | TS (Next.js) | Landing Page renderer | ScyllaDB + MongoDB |
| 24 | **bff-admin** | TS (Bun) | Backend-for-Frontend cho Admin | (API aggregation) |
| 25 | **bff-crm** | TS (Bun) | Backend-for-Frontend cho CRM | (API aggregation) |
| 26 | **report-engine** | Go | Xuất báo cáo, BI dashboard | ClickHouse |

### 5.2. Sơ đồ Microservices

```
[CLIENTS] → [API GATEWAY (Go)] → [EDGE GATEWAY (Rust)]
                                       │
       ┌───────────────┬───────────────┼───────────────┬──────────────┐
       ▼               ▼               ▼               ▼              ▼
[Landing]       [CRM]          [Tree-Org]      [Chat]         [Media SFU]
  Ingest         Core                                          
  │              │                                              │
  ▼              ▼                                              ▼
[FB CAPI]    [Notification]                              [Recorder]
                                                              │
                                                              ▼
                                                    [AI Scoring/Conv/Media/SRE]
                                                              │
                                                              ▼
                                                        [Analytics + ClickHouse]
```

### 5.3. Nguyên tắc giao tiếp
- **Public API (Web/App → Gateway):** REST (Huma + Echo) + WebTransport/WebSocket.
- **Internal RPC (Service ↔ Service):** Connect-RPC (gRPC + HTTP/1.1) — trình duyệt có thể gọi trực tiếp.
- **Async Event:** NATS JetStream (primary) + Valkey Streams (lightweight log).
- **Realtime:** WebSocket + FlatBuffers binary.

---

## 6. Polyglot Persistence – Chiến lược đa cơ sở dữ liệu

### 6.1. Ma trận cơ sở dữ liệu

| Phân hệ | Database | Mô hình | Lý do |
|---------|----------|---------|-------|
| **Chat History & Event Streams** | **ScyllaDB 6.x** | Wide-column / Shard-per-core | Ghi > 1M/s, p99 < 10ms |
| **Real-time Analytics & Clickstream** | **ClickHouse 24.x** | Column-oriented / SIMD | Nén 10:1, OLAP tốc độ cao |
| **CRM Core & Business Logic** | **PostgreSQL 17+** | Relational / MVCC | ACID, JSONB, LTREE, pgvector |
| **State Management & Presence** | **Valkey 9.x** | In-memory | Nhanh hơn Redis 8-20%, BSD |
| **Dynamic Form Data** | **MongoDB 7.x** | Document | Schema linh hoạt |
| **Media Files** | **MinIO / S3** | Object Storage + Erasure Coding | Chi phí thấp, mở rộng vô hạn |
| **Vector Search** | **Qdrant** (Rust) + **pgvector** | HNSW | Tách tenant > 50M vectors |
| **Search** | **Meilisearch** | Inverted index | Full-text Tiếng Việt |
| **Caching Layer** | **Valkey 9.x** | In-memory | Session, Rate Limit, Coalescing |

### 6.2. Quy tắc lựa chọn DB
1. **Dữ liệu ghi nặng, real-time?** → ScyllaDB
2. **Dữ liệu quan hệ, cần JOIN, ACID?** → PostgreSQL
3. **Dữ liệu không cấu trúc, schema thay đổi liên tục?** → MongoDB
4. **Phân tích OLAP, time-series?** → ClickHouse
5. **Session, rate limit, lock?** → Valkey
6. **File binary lớn (video, audio)?** → MinIO/S3
7. **Full-text search?** → Meilisearch
8. **Vector similarity?** → Qdrant (scale) hoặc pgvector (<10M vectors)

---

## 7. Bốn phân hệ CRM (4 Phần Chính)

### 7.1. Phần 1 – Super Admin Portal (Chi tiết: `docs/01-super-admin/`)

**Mục đích:** Admin tối cao giám sát toàn bộ hệ thống, tạo/khóa tenant, theo dõi mọi chỉ số.

**Chức năng chính:**
- Tạo/Khóa tenant (công ty đối tác)
- Quản lý tài khoản Super Admin (Dark Admin – không có DNS record công khai)
- Realtime Dashboard: Tổng tenant, tổng user, doanh thu, traffic, lỗi
- Resource Allocation: Chỉ định VPS cho tenant
- Billing & Subscription Management
- Feature Flags (bật/tắt tính năng theo tenant)
- Tenant Impersonation (Admin có thể đăng nhập hộ tenant khi được phép)
- Multi-Region Control

### 7.2. Phần 2 – Tenant Company Site + Isolated VPS (Chi tiết: `docs/02-tenant-site/`)

**Mục đích:** Mỗi công ty đối tác có trang web riêng (`apex.hanghoaphaisinh.net` hoặc custom domain).

**Kiểu triển khai:**
- **Shared Cluster:** Chạy chung cụm, tiết kiệm chi phí.
- **Isolated VPS:** VPS riêng do khách mua, kết nối qua WireGuard.

**Cơ chế Mesh:**
- Tất cả VPS kết nối qua eBPF Mesh / WireGuard Tunnel.
- Distributed Resource Sharing Scheduler cho phép VPS rảnh hỗ trợ render/AI cho tenant khác.

### 7.3. Phần 3 – CRM cho nhân viên (Chi tiết: `docs/03-crm-tree/`)

**Mục đích:** Nhân viên theo sơ đồ cây: Giám đốc → Quản lý → Trưởng nhóm → Nhân viên.

**Tính năng:**
- Tree Org với PostgreSQL LTREE
- Tokenized Invitation Link (PASETO v4)
- Thăng/Hạ chức, di chuyển nhánh (move subtree)
- Phân quyền RBAC theo vai trò
- Dashboard Lead, KPI, hoa hồng

### 7.4. Phần 4 – Trình sinh mô hình doanh nghiệp động (Chi tiết: `docs/04-dynamic-model/`)

**Mục đích:** Mỗi ngành nghề (BĐS, Tài chính, Bán lẻ, B2B SaaS) có mô hình CRM khác nhau.

**Cơ chế:**
- Meta-Schema Engine lưu JSON Schema
- Field Customizer: Thêm/sửa/xóa trường động
- Workflow Builder: Tạo pipeline trạng thái
- Validation Engine: Xác thực runtime

---

## 8. Landing Page & Facebook CAPI

### 8.1. Nguyên tắc kế thừa (chi tiết: `docs/05-landing-capi/`)
- **Giữ 100% nội dung** từ `chiase_cu/`.
- **Thay thư viện cũ** (jQuery, Bootstrap cũ) bằng Tailwind CSS, Lucide Icons, Native Web Components.
- **Tracking chi tiết hơn:** UTM, fbclid, fbp, fbc, IP, UA, referrer, device fingerprint, scroll depth, time on page, heatmap.
- **Meta Pixel + Conversions API** chạy song song (Hybrid Dual-Tracking).

### 8.2. Luồng Facebook CAPI chống giả mạo
```
[User] → [Landing Form] → [Go Ingestion API]
                              ↓
                       [SHA-256 Normalize User Data]
                              ↓
                       [HMAC-SHA256 Signature]
                              ↓
                       [NATS Queue]
                              ↓
                       [FB CAPI Worker]
                              ↓
                       [Meta Graph API]
```
- Mỗi event có `event_id` UUIDv7.
- Payload bắt buộc có HMAC: `HMAC-SHA256(lead_id + fbclid + timestamp, Secret_Key)`.
- CRM chuyển trạng thái → bắn ngược Purchase/Custom_SQL_Event về Facebook.

---

## 9. Chat Real-time Engine

### 9.1. Kiến trúc (chi tiết: `docs/06-chat-engine/`)
```
[Web/App Clients]
        │ WebTransport / Zero-Copy WebSocket
        ▼
[Kernel-Bypass Gateway (Rust + io_uring + eBPF/XDP)]
        │
        ├─► FlatBuffers Binary Protocol (Zero-Parsing)
        ├─► Singleflight Coalescing Layer (Merge duplicate DB queries)
        ├─► Valkey 9.x Cluster (Session & Presence)
        └─► ScyllaDB (Shard-per-core Chat Logs)
```

### 9.2. Tính năng:
- **Tin nhắn 1-1, nhóm, channel công ty.**
- **Gửi file** qua Presigned S3 Direct Upload (không qua backend).
- **Reactions, threads, mentions, replies.**
- **Emoji tùy chỉnh, sticker.**
- **Typing indicator, read receipt.**
- **Search lịch sử** qua Meilisearch.
- **Voice message** ghi âm từ trình duyệt.
- **Push notification** khi offline.

---

## 10. WebRTC SFU & Recording

### 10.1. SFU Node (chi tiết: `docs/07-webrtc-sfu/`)
- **Rust/C++ SFU** với Zero-Transcoding.
- AV1/VP9 SVC (Scalable Video Coding).
- eBPF/XDP routing gói tin UDP/SRTP.
- BBR-WebRTC Congestion Control.

### 10.2. Recording
- **Egress Worker (C++ + CUDA)** render composite trực tiếp trên GPU VRAM.
- NVIDIA NVENC encode trực tiếp → MinIO.
- Whisper.cpp STT + Llama-3 Summarize.

### 10.3. Tính năng Meeting:
- Voice call, Video call 1-1, Group call.
- Screen share 4K@60fps.
- Background blur/virtual.
- Recording (cloud + local).
- Live caption (real-time STT).
- Action Items AI-extracted.
- Calendar integration.
- Waiting room, Knock to join.

---

## 11. Observability 4 tầng + AI SRE

### 11.1. Tầng 1 – Application Level (chi tiết: `docs/08-observability/`)
- Trace ID UUIDv7 cho mọi request.
- Structured JSON Log (Zap / tracing).
- Error Wrapping với context.

### 11.2. Tầng 2 – Telemetry Triad
- **Logs:** Vector (Rust) → ClickHouse/Loki.
- **Metrics:** Prometheus / VictoriaMetrics.
- **Traces:** OpenTelemetry → Jaeger/Tempo.
- **Dashboard:** Grafana unified view.

### 11.3. Tầng 3 – Realtime Alerting
- Sentry/GlitchTip aggregate errors.
- AI SRE (Code-LLM) phân tích RCA.
- Multi-channel alert: Telegram, Slack, PagerDuty, Twilio.

### 11.4. Tầng 4 – Self-Healing
- Circuit Breaker (gobreaker).
- K3s Liveness Probe (Restart < 2s).
- Fallback: Valkey/SQLite local khi ClickHouse sập.

---

## 12. Bảo mật Multi-Tenant Zero-Trust

### 12.1. Tầng Network (chi tiết: `docs/09-security/`)
- eBPF/XDP Anti-DDoS, JA4+ Fingerprint, Rate Limit Map.

### 12.2. Tầng Frontend & Ingestion
- Wasm Hardware Attestation (chống Headless).
- Argon2 Proof-of-Work.

### 12.3. Tầng Database
- PostgreSQL Row-Level Security (RLS).
- SET LOCAL `app.current_tenant_id`.
- Valkey cache phân quyền cây.

### 12.4. Tầng Admin
- Dark Admin: không DNS, không IP public.
- Single Packet Authorization (SPA) qua WireGuard.
- FIDO2/YubiKey + 2-of-3 Quorum cho thao tác nguy hiểm.

### 12.5. Tầng Kernel
- eBPF RASP Security: chặn syscall lạ.
- Auto BGP Blackhole cho IP tấn công.

---

## 13. Tích hợp AI toàn hệ thống

### 13.1. 3 phân vùng AI (chi tiết: `docs/11-ai-integration/`)

| Phân vùng | Mô hình | Chức năng | Độ trễ |
|----------|---------|-----------|--------|
| **AI SRE & Code Guard** | DeepSeek-Coder / Llama-3-Code | Auto RCA, Hotfix Proposal | < 3s |
| **Predictive Analytics** | XGBoost / LightGBM | Lead Scoring, pLTV, FB Optimizer | < 5ms |
| **Conversational & Media** | Whisper.cpp + Llama-3-70B + vLLM | Chatbot RAG, STT, Summarize | Sub-200ms |

### 13.2. Multi-Tenant AI Isolation
- **Tenant-Scoped Cache Key:** `SHA256(tenant_id + prompt_prefix)`.
- **GPU Token Pool Admission Control:** Token bucket per tenant.
- **Zero-Trust AI Pipeline:** Mỗi agent có quyền giới hạn.

---

## 14. Cơ sở hạ tầng đề xuất

### 14.1. Local Dev (Docker Compose)
- Tất cả service chạy trong Docker với profile: `core`, `ai`, `media`, `observability`.
- Xem chi tiết: `infra/docker/docker-compose.yml`.

### 14.2. Production (K3s Cluster)
- **Control Plane:** 3 nodes (HA).
- **Edge Nodes:** Auto-scale theo traffic.
- **VPS Mesh:** WireGuard overlay.
- **GPU Pool:** 1+ node trang bị GPU cho Recorder + AI.

### 14.3. Phân vùng mạng
| VLAN/Subnet | Chức năng |
|-------------|-----------|
| Public | Edge Gateway + Anti-DDoS |
| Private Service | Internal microservices |
| Data | Database, cache, storage |
| Admin Mesh | WireGuard chỉ Super Admin |

---

## 15. Lộ trình triển khai

### Phase 1 – MVP (Tháng 1-3)
- [x] Thiết kế tổng thể (file này).
- [ ] Core microservices: api-gateway, auth, tenant-manager.
- [ ] Landing Page Renderer (kế thừa chiase_cu).
- [ ] Landing Ingest + ScyllaDB.
- [ ] CRM Core (Lead, Contact, Deal).
- [ ] Super Admin Portal cơ bản.
- [ ] Tree-Org với PASETO Invitation Link.
- [ ] Docker Compose local.

### Phase 2 – Tính năng chính (Tháng 4-6)
- [ ] Chat Engine (Rust + ScyllaDB).
- [ ] WebRTC SFU + Recorder.
- [ ] Facebook CAPI Hybrid Dual-Tracking.
- [ ] AI Scoring (XGBoost).
- [ ] Notification Service.
- [ ] Analytics Dashboard (ClickHouse).

### Phase 3 – Multi-Tenant đầy đủ (Tháng 7-9)
- [ ] Dynamic Schema Engine.
- [ ] Isolated VPS + WireGuard Mesh.
- [ ] Dark Admin + FIDO2.
- [ ] PostgreSQL RLS hoàn chỉnh.
- [ ] Billing & Subscription.

### Phase 4 – AI & Optimization (Tháng 10-12)
- [ ] AI Conversation (vLLM + RAG).
- [ ] AI SRE (Code-LLM RCA).
- [ ] Whisper.cpp STT cho Meeting.
- [ ] GPU Composite Recording.
- [ ] Advanced eBPF optimizations.

---

## 16. Danh sách tài liệu chi tiết

| # | Tài liệu | Đường dẫn |
|---|---------|-----------|
| 01 | Super Admin Portal | `docs/01-super-admin/README.md` |
| 02 | Tenant Company Site + VPS | `docs/02-tenant-site/README.md` |
| 03 | CRM Tree (Phân cấp) | `docs/03-crm-tree/README.md` |
| 04 | Dynamic Model Engine | `docs/04-dynamic-model/README.md` |
| 05 | Landing Page + CAPI | `docs/05-landing-capi/README.md` |
| 06 | Chat Real-time Engine | `docs/06-chat-engine/README.md` |
| 07 | WebRTC SFU + Recording | `docs/07-webrtc-sfu/README.md` |
| 08 | Observability + AI SRE | `docs/08-observability/README.md` |
| 09 | Multi-Tenant Security | `docs/09-security/README.md` |
| 10 | Database Schema | `docs/10-database/README.md` |
| 11 | AI Integration | `docs/11-ai-integration/README.md` |

---

## Phụ lục: Quy ước đặt tên & Coding Standard

### Naming Convention
- **Service name:** `kebab-case` (vd: `crm-core`, `landing-ingest`).
- **Database name:** `snake_case` (vd: `rinco_crm`, `rinco_chat`).
- **Topic NATS:** `domain.action` (vd: `lead.created`, `deal.updated`).
- **Trace ID:** `UUIDv7`.
- **Tenant ID:** `kebab-case` slug (vd: `apex-fintech`).

### Git Workflow
- `main` – production-ready.
- `develop` – integration.
- `feature/<scope>-<short-desc>` – feature.
- `hotfix/<scope>-<short-desc>` – urgent fix.
- **Quy tắc:** Mỗi thay đổi phải commit + push. Các phiên quan trọng push toàn bộ.

### Code Style
- Go: `gofumpt`, `golangci-lint`.
- Rust: `rustfmt`, `clippy`.
- TS: `eslint`, `prettier`, `tsc --strict`.

---

# PHẦN MỞ RỘNG (v1.1) – AUDIT, CODE EXAMPLES, EDGE CASES

## 17. Audit – Đánh giá nội dung hiện tại

### 17.1. Phần đã đủ chi tiết ✓

| Mục | Nội dung | Mức đủ |
|-----|---------|--------|
| 4 – Tech Matrix | Liệt kê thư viện đầy đủ cho Go/Rust/TS/Python | ✓ |
| 6 – Polyglot Persistence | Ma trận 9 database kèm quy tắc chọn | ✓ |
| 9 – Chat Engine | Kiến trúc + tính năng chat | ✓ |
| 10 – WebRTC SFU | Zero-Transcoding, AV1 SVC, GPU composite | ✓ |
| 11 – Observability 4 tầng | Code-Level → Telemetry → Alerting → Self-Healing | ✓ |
| 12 – Bảo mật Zero-Trust | 5 tầng (Network → Kernel) | ✓ |

### 17.2. Phần còn thiếu ⚠

| Mục | Vấn đề | Hướng bổ sung |
|-----|--------|---------------|
| 1.2 – Bài toán | Chưa liệt kê trade-off khi scale (vd: ScyllaDB không JOIN → cần denormalize) | Bổ sung bảng trade-off |
| 2.3 – Văn hóa | CG1–CG4 chỉ liệt kê, chưa có cơ chế đo lường | Thêm OKR cho từng CG |
| 3.1 – Kiến trúc | Chưa nói rõ disaster recovery giữa các tầng | Bổ sung DR section (§22) |
| 5 – Microservices | Chưa có ownership matrix (team nào phụ trách service nào) | Bổ sung Team-Ownership Matrix |
| 8 – Landing Page | Chưa có fallback khi FB CAPI fail | Bổ sung fallback flow |
| 13 – AI | Thiếu guardrail chống prompt injection | Bổ sung §13.3 Guardrails |
| 15 – Lộ trình | Chưa có Gantt chart chi tiết từng tuần | Bổ sung roadmap tuần |
| – Tổng thể | Thiếu **khả năng chịu lỗi cụ thể** (vd: nếu ClickHouse chết 30 phút thì sao?) | Bổ sung Failure Mode Analysis |

### 17.3. Mâu thuẫn nội bộ ✗

| Vị trí | Mâu thuẫn |
|--------|-----------|
| §4.2.1 Go WebSocket | Đề cập `nhooyr/websocket` nhưng `nhooyr` đã deprecated → thay bằng `coder/websocket` |
| §10 SFU Zero-Transcoding vs §13 AI STT | Nếu không transcode thì Whisper.cpp xử lý raw RTP như thế nào? → Cần middleware decode |
| §3.2 AP2 (Zero-Copy) vs Go Gateway | Go GC xung đột với io_uring zero-copy đã được cảnh báo trong yeucauthietke.md – chưa phản ánh trong doc |
| §12.4 Dark Admin + §14.3 Network | Subnet "Admin Mesh" chỉ có ở §14.3 nhưng §12 chưa gọi tên subnet này |

### 17.4. Phần cần code example cụ thể 💡

| Mục | Cần code cho |
|-----|--------------|
| 5 – Microservices | `service.go` skeleton, `Dockerfile`, K8s manifest |
| 6 – Polyglot | sqlc.yaml + Ent schema definition |
| 8 – FB CAPI | HMAC generation function + Worker pool |
| 9 – Chat | FlatBuffers schema + Rust handler |
| 11 – Observability | Zap logger wrapper + OTEL middleware |
| 12 – Security | PASETO middleware + RLS policy template |

## 18. Edge Cases & Error Scenarios chi tiết

### 18.1. Edge Cases – Landing Ingest

| # | Edge case | Phát hiện | Xử lý |
|---|----------|-----------|-------|
| E1 | Form submit 2 lần trong 100ms (double-click) | idempotency_key + Scylla LWT | Reject lần 2, trả về cùng `lead_id` |
| E2 | Email là "user@" (không hợp lệ) | protovalidate + Zod | 422 ValidationError, không ghi DB |
| E3 | UTM rỗng nhưng fbclid có | Conditional validation | Lưu UTM=NULL, fbclid bắt buộc hash |
| E4 | IP nằm trong blacklist (bot) | Valkey lookup + eBPF Map | 403 Forbidden, log alert |
| E5 | User submit nhưng không có JS (noscript) | Fallback `<noscript>` form | POST qua standard form endpoint, server vẫn validate |
| E6 | Khách hàng nhập emoji vào tên | UTF-8 normalize | OK, lưu raw + normalized |
| E7 | Form có 50 fields custom (Dynamic Schema) | Schema lookup Valkey | Validate theo JSON Schema đã cached |
| E8 | Tenant quota Lead/day đã hết | Counter Valkey | 429 Too Many Requests + Retry-After |
| E9 | FB CAPI bị rate-limit | gobreaker Open | Đẩy vào NATS retry queue, không block ingest |
| E10 | ScyllaDB node fail giữa write | Retry với Token Aware | Re-route tới node khác trong 5s |

### 18.2. Edge Cases – CRM Multi-Tenant

| # | Edge case | Xử lý |
|---|----------|-------|
| E11 | User A thuộc tenant X, thử query Lead của tenant Y | RLS + SET LOCAL → return 0 row |
| E12 | Move subtree 1000 user xuống branch khác | Wrap transaction + ltree nlevel |
| E13 | 2 admin cùng merge user cùng lúc | SELECT FOR UPDATE |
| E14 | Tenant xóa mềm, user vẫn login | Token blacklist (Valkey) |
| E15 | Cycle trong cây (A là parent của B, B là parent của A) | Pre-check trước UPDATE |
| E16 | PASETO token hết hạn giữa session | Refresh token rotation |
| E17 | User đổi role từ GD xuống NV trong khi đang đăng nhập | Force logout next request |
| E18 | Department leader bị xóa | Auto-reassign leader hoặc báo lỗi |
| E19 | Bulk move 10K user mà 1 user conflict | All-or-nothing transaction → rollback |
| E20 | Lead được assign cho user không tồn tại | ON DELETE SET NULL |

### 18.3. Edge Cases – Chat Real-time

| # | Edge case | Xử lý |
|---|----------|-------|
| E21 | User gửi 1000 msg/giây (spam) | Token bucket per user: 10 msg/s |
| E22 | WebSocket disconnect giữa tin nhắn | Client resync từ last_event_id |
| E23 | Tin nhắn đến khi user offline | Push notification + persist in ScyllaDB |
| E24 | File 10GB upload | Client phải dùng multipart, server reject >100MB |
| E25 | Typing indicator flood | Server ignore nếu >1 indicator/giây/user |
| E26 | Mention không tồn tại | Server bỏ qua, không lỗi |
| E27 | Channel bị xóa khi đang chat | 410 Gone, client refresh |
| E28 | Encryption key rotate | Re-handshake + re-encrypt old messages |
| E29 | 50K users mở cùng 1 channel | Singleflight coalescing → 1 query |
| E30 | Server restart mid-message | Client retry với idempotency_key |

### 18.4. Edge Cases – WebRTC Meeting

| # | Edge case | Xử lý |
|---|----------|-------|
| E31 | User mất mạng 5s rồi reconnect | ICE restart + RTCP PLI/NACK |
| E32 | GPU crash khi đang record | Fallback sang CPU encode (chậm hơn) |
| E33 | Egress Worker không đủ NVENC session | Queue meeting, scale worker |
| E34 | 1 user trong call drop liên tục (network bad) | Auto-leave sau 3 lần fail |
| E35 | AV1 không support trên trình duyệt cũ | Fallback VP9 → H.264 |
| E36 | Recording file corrupt do MinIO network | Retry với S3 multipart, checksum |
| E37 | Whisper STT quá chậm | Batch STT mỗi 30s thay vì real-time |
| E38 | Background noise quá lớn | Bật noise suppression trong getUserMedia |
| E39 | User share screen với resolution 4K | BBR giảm xuống 1080p |
| E40 | Meeting vượt quá 4h (giới hạn) | Auto-end + notify user |

### 18.5. Edge Cases – Multi-VPS Mesh

| # | Edge case | Xử lý |
|---|----------|-------|
| E41 | VPS của tenant A mất kết nối WireGuard | Auto-failover sang VPS B của tenant |
| E42 | Resource sharing job chiếm 100% CPU VPS A | Cgroup limit + revert job |
| E43 | 2 tenant cùng request share GPU | Token bucket per tenant |
| E44 | Tenant A bị admin khóa nhưng VPS A vẫn share resource | Block ngay lập tức, revoke shares |
| E45 | WireGuard key bị leak | Force rekey toàn mesh |
| E46 | Backup replication lag > 1h | Alert SRE + manual run |
| E47 | Mesh coordinator chết | Ten cluster có local control plane fallback |
| E48 | Cross-region latency > 200ms | Block share job cross-region |

### 18.6. Edge Cases – Security

| # | Edge case | Xử lý |
|---|----------|-------|
| E49 | SQL injection trong tenant_id | Parameterized queries + RLS (không trust input) |
| E50 | CSRF trên form admin | SameSite=Strict + CSRF token |
| E51 | Replay attack HMAC | Nonce + timestamp window 5min |
| E52 | Prompt injection vào AI Chatbot | Strip system prompt + guardrail layer |
| E53 | Admin key bị compromise | 2-of-3 Quorum rollback + key rotation |
| E54 | DDoS 10Gbps | eBPF/XDP drop ở NIC |
| E55 | Botnet vượt qua Wasm attestation | Argon2 PoW thêm layer 2 |
| E56 | Insider attack (admin lạm quyền) | Audit log + anomaly detection AI |
| E57 | ClickHouse data leak | RLS + column encryption |
| E58 | Tenant A đọc file trong MinIO của tenant B | Presigned URL chỉ chứa tenant prefix |

### 18.7. Edge Cases – Observability

| # | Edge case | Xử lý |
|---|----------|-------|
| E59 | Trace_id bị trùng (collision) | UUIDv7 entropy 74 bits → ~0% |
| E60 | Log volume > 1TB/giờ | Sampling 10% cho INFO, 100% ERROR |
| E61 | Vector Agent crash | Systemd auto-restart, queue local buffer |
| E62 | Sentry rate limit hit | Local GlitchTip fallback |
| E63 | AI SRE hallucinate fix sai | Sandbox test PR trước khi merge |
| E64 | Alert storm (1000 alert/phút) | Dedup by fingerprint |
| E65 | Grafana query timeout | Cache materialized view |

## 19. Implementation Roadmap chi tiết

> **Lưu ý:** Roadmap này đã có ở §15 nhưng thiếu chi tiết từng tuần + acceptance gate. Phần này bổ sung.

### 19.1. Phase 1 (Tuần 1–12): MVP Foundation

#### Tuần 1–2: Bootstrap & Dev Environment
```bash
# 1. Khởi tạo monorepo
git clone https://github.com/itdoanh/rinco.git
cd rinco
mkdir -p services/{api-gateway,landing-ingest,crm-core,tree-org,auth-service} \
         apps/{landing,admin,crm} \
         infra/{docker,k8s,terraform} \
         docs/{00-master,01-super-admin,...}

# 2. Setup Docker Compose
cp .env.example .env
docker compose --profile core up -d

# 3. Verify services
curl http://localhost:8080/health  # api-gateway
psql -h localhost -U rinco -d rinco_crm  # postgres
```

**Acceptance Gate:** Tất cả container lên, health check pass.

#### Tuần 3–4: API Gateway + Auth
- [ ] Tạo `services/api-gateway/` (Echo + Huma).
- [ ] Tạo `services/auth-service/` (PASETO + FIDO2 stub).
- [ ] WireGuard container cho admin.
- [ ] Valkey container với domain map.
- [ ] Route `/api/v1/auth/*` qua api-gateway.

**Acceptance Gate:** Login admin thành công, nhận PASETO token.

#### Tuần 5–6: Landing Page Renderer
- [ ] Tạo `apps/landing/` Next.js + Bun.
- [ ] Migrate 100% content từ `chiase_cu/index.html`.
- [ ] Replace jQuery/Bootstrap bằng Tailwind + Lucide.
- [ ] Component `<LeadForm>` + Meta Pixel init.
- [ ] Component `<TrackingProvider>` gửi UTM/fbclid.

**Acceptance Gate:** Lighthouse FCP < 0.4s, LCP < 0.8s.

#### Tuần 7–8: Landing Ingest
- [ ] Tạo `services/landing-ingest/` (Huma + sqlc).
- [ ] ScyllaDB schema: `CREATE TABLE rinco_chat.leads_raw (...)`.
- [ ] HMAC middleware verify payload.
- [ ] Circuit breaker quanh FB CAPI call.
- [ ] NATS producer `apex-fintech.lead.created`.

**Acceptance Gate:** Submit form → nhận Lead ID < 50ms.

#### Tuần 9–10: CRM Core
- [ ] Tạo `services/crm-core/` (Ent).
- [ ] Tables: `leads`, `contacts`, `deals`, `activities`.
- [ ] RLS policies trên PostgreSQL.
- [ ] REST API `/api/crm/v1/leads` (CRUD).
- [ ] Valkey cache hot reads.

**Acceptance Gate:** Create Lead → list Lead → update Lead trong < 100ms.

#### Tuần 11–12: Tree-Org + Admin Portal skeleton
- [ ] Tạo `services/tree-org/` (sqlc + LTREE).
- [ ] PASETO invitation service.
- [ ] `apps/admin/` Next.js dashboard.
- [ ] Login flow + YubiKey placeholder.

**Acceptance Gate:** Tạo tenant mới → user root → mời 2 manager qua link.

### 19.2. Phase 2 (Tuần 13–24): Tính năng chính

#### Tuần 13–14: Chat Engine
- [ ] Rust service `chat-engine` với tokio-uring.
- [ ] ScyllaDB `messages` table với partition key `(channel_id, tenant_id)`.
- [ ] FlatBuffers schema định nghĩa `Message`.
- [ ] WebSocket endpoint `/ws/chat`.

#### Tuần 15–17: WebRTC SFU + Recorder
- [ ] Rust SFU service `media-sfu` (str0m).
- [ ] AV1 codec integration.
- [ ] eBPF program cho UDP routing (basic).
- [ ] C++ Egress Worker với NVENC.
- [ ] MinIO multipart upload.

#### Tuần 18–19: Facebook CAPI Worker
- [ ] Service `meta-capi` (Go).
- [ ] SHA-256 normalize.
- [ ] HMAC verify.
- [ ] Worker pool xử lý NATS message.
- [ ] Retry queue khi circuit open.

#### Tuần 20–22: AI Scoring
- [ ] Python service `ai-scoring` (XGBoost).
- [ ] NATS consumer `lead.created`.
- [ ] ONNX runtime < 5ms inference.
- [ ] Score → PostgreSQL `leads.ai_score`.

#### Tuần 23–24: Notification + Analytics
- [ ] Service `notification` (Email/Push/Telegram).
- [ ] ClickHouse schema `events_raw`.
- [ ] Grafana dashboard "Realtime Funnel".

### 19.3. Phase 3 (Tuần 25–36): Multi-Tenant đầy đủ

#### Tuần 25–27: Dynamic Schema
- [ ] Service `dynamic-schema` (Huma).
- [ ] JSON Schema editor trong Admin.
- [ ] Validator runtime compile sang Go.

#### Tuần 28–30: Isolated VPS + Mesh
- [ ] Service `mesh-controller` (Headscale wrapper).
- [ ] Bootstrap script `bootstrap-vps.sh` production-ready.
- [ ] Resource scheduler với scoring algorithm.

#### Tuần 31–33: Dark Admin + FIDO2
- [ ] SPA via WireGuard.
- [ ] WebAuthn flow hoàn chỉnh.
- [ ] 2-of-3 Quorum UI.

#### Tuần 34–36: Billing
- [ ] Service `billing` (Stripe + VNPay).
- [ ] Usage tracking (API calls, storage, AI calls).
- [ ] Invoice generator PDF.

### 19.4. Phase 4 (Tuần 37–48): AI & Optimization

#### Tuần 37–40: AI Conversation (vLLM)
- [ ] vLLM cluster 3 node (A100).
- [ ] Qdrant + pgvector setup.
- [ ] Tenant-scoped cache key.
- [ ] RAG pipeline cho CRM knowledge.

#### Tuần 41–43: AI SRE (Code-LLM)
- [ ] DeepSeek-Coder fine-tune trên codebase RINCO.
- [ ] Sentry webhook integration.
- [ ] Auto-PR generator.

#### Tuần 44–46: Whisper STT cho Meeting
- [ ] Whisper.cpp + TensorRT trên GPU node.
- [ ] Real-time caption.
- [ ] Action items extraction.

#### Tuần 47–48: GPU Composite Recording
- [ ] CUDA composite layout.
- [ ] 4K@60fps recording.
- [ ] Direct chunked upload to MinIO.

### 19.5. Acceptance Gate chung cho mỗi Phase
- [ ] Tất cả service pass integration test.
- [ ] Lighthouse FCP/LCP đạt target.
- [ ] Load test 10K RPS không lỗi.
- [ ] Security scan không có critical CVE.
- [ ] Audit log coverage 100%.
- [ ] Documentation cập nhật.

## 20. Testing Strategy End-to-End

### 20.1. Test Pyramid

```
                    ┌─────────┐
                    │   E2E   │  10% (Playwright + Cypress)
                    ├─────────┤
                  ┌─┴─────────┴─┐
                  │ Integration │  20% (Testcontainers + dockertest)
                  ├─────────────┤
                ┌─┴─────────────┴─┐
                │   Unit Test     │  70% (Go testify, Rust cargo test, Vitest)
                └─────────────────┘
```

### 20.2. Unit Test Targets

| Service | Coverage target | Tools |
|---------|----------------|-------|
| api-gateway | ≥ 85% | testify + gomock |
| landing-ingest | ≥ 90% | testify + dockertest Scylla |
| crm-core | ≥ 85% | testify + dockertest Postgres |
| tree-org | ≥ 85% | testify + dockertest Postgres |
| auth-service | ≥ 90% | testify + WebAuthn mock |
| chat-engine | ≥ 80% | cargo test + scylla mock |
| media-sfu | ≥ 70% | cargo test + str0m mock |
| recorder | ≥ 70% | gtest + NVENC mock |
| ai-scoring | ≥ 75% | pytest + ONNX mock |
| ai-conversation | ≥ 70% | pytest + vLLM mock |
| meta-capi | ≥ 85% | testify + httptest |
| tenant-manager | ≥ 85% | testify + K3s fake client |

### 20.3. Integration Tests (Test Pattern)

Ví dụ test cho landing-ingest:
```go
// services/landing-ingest/internal/service/ingest_test.go
package service_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go/modules/scylladb"
    "github.com/testcontainers/testcontainers-go/modules/nats"

    "rinco/landing-ingest/internal/service"
)

func TestIngestLead_EndToEnd(t *testing.T) {
    ctx := context.Background()

    // 1. Spin up ScyllaDB container
    scylla, _ := scylladb.RunContainer(ctx, testcontainers.WithImage("scylladb/scylla:6"))
    defer scylla.Terminate(ctx)

    // 2. Spin up NATS container
    natsC, _ := nats.RunContainer(ctx)
    defer natsC.Terminate(ctx)

    // 3. Init service
    svc := service.New(service.Config{
        ScyllaEndpoints: []string{scylla.ConnectionString()},
        NATSEndpoint:    natsC.ConnectionString(),
    })

    // 4. Submit lead
    leadID, err := svc.Ingest(ctx, service.IngestInput{
        TenantID:  "test-tenant",
        Email:     "user@example.com",
        Phone:     "+84909123456",
        FullName:  "Nguyen Van A",
        UTMSource: "facebook",
        FBCLID:    "fb.1.123456789.987654321",
    })
    require.NoError(t, err)
    require.NotEmpty(t, leadID)

    // 5. Verify event published to NATS
    sub, _ := natsC.Subscribe("test-tenant.lead.created")
    msg, err := sub.NextMsgWithTimeout(ctx, 5*time.Second)
    require.NoError(t, err)
    require.Contains(t, string(msg.Data), leadID)

    // 6. Verify persisted to ScyllaDB
    var stored service.IngestInput
    err = scylla.Query("SELECT * FROM rinco_chat.leads_raw WHERE lead_id = ?", leadID).Scan(&stored)
    require.NoError(t, err)
    require.Equal(t, "user@example.com", stored.Email)
}

func TestIngestLead_DuplicateSubmission(t *testing.T) {
    // Setup
    ctx := context.Background()
    scylla, _ := scylladb.RunContainer(ctx)
    defer scylla.Terminate(ctx)
    natsC, _ := nats.RunContainer(ctx)
    defer natsC.Terminate(ctx)
    svc := service.New(service.Config{
        ScyllaEndpoints: []string{scylla.ConnectionString()},
        NATSEndpoint:    natsC.ConnectionString(),
    })

    idempotencyKey := "idem-key-123"
    input := service.IngestInput{
        TenantID:       "test-tenant",
        Email:          "user@example.com",
        IdempotencyKey: idempotencyKey,
    }

    // First submission
    leadID1, err := svc.Ingest(ctx, input)
    require.NoError(t, err)

    // Second submission (same idempotency key)
    leadID2, err := svc.Ingest(ctx, input)
    require.NoError(t, err)

    // Should return same lead ID
    require.Equal(t, leadID1, leadID2)
}

func TestIngestLead_InvalidEmail(t *testing.T) {
    ctx := context.Background()
    svc := setupTestService(t, ctx)
    defer teardownTestService(t, svc)

    _, err := svc.Ingest(ctx, service.IngestInput{
        TenantID: "test-tenant",
        Email:    "invalid-email",
    })

    require.Error(t, err)
    require.Equal(t, service.ErrValidation, err)
}
```

### 20.4. E2E Tests (Playwright Pattern)

```typescript
// apps/landing/e2e/lead-submission.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Landing Page Lead Submission', () => {
  test('happy path: submit form → lead appears in admin dashboard', async ({ page, request }) => {
    // 1. Visit landing page
    await page.goto('/apexfintech');
    await expect(page.locator('h1')).toContainText('Apex Fintech');

    // 2. Fill form
    await page.fill('input[name="full_name"]', 'Nguyen Van Test');
    await page.fill('input[name="phone"]', '0909123456');
    await page.fill('input[name="email"]', 'test@example.com');

    // 3. Intercept Meta Pixel call
    const pixelCalls: any[] = [];
    await page.exposeFunction('captureFbq', (event: any) => pixelCalls.push(event));
    await page.addInitScript(() => {
      (window as any).fbq = ((...args: any[]) => {
        (window as any).captureFbq(args);
      });
    });

    // 4. Submit
    const [response] = await Promise.all([
      page.waitForResponse('**/api/ingest/v1/leads'),
      page.click('button[type="submit"]'),
    ]);
    expect(response.status()).toBe(200);

    // 5. Verify Meta Pixel fired
    expect(pixelCalls.some(c => c[1] === 'track' && c[2] === 'Lead')).toBeTruthy();

    // 6. Login admin and verify lead appears
    await page.goto('/admin/login');
    await page.fill('input[name="email"]', 'admin@rinco.app');
    await page.click('button:has-text("Login with YubiKey")');
    // ... YubiKey mock ...

    await page.goto('/admin/tenants/apexfintech/leads');
    await expect(page.locator('table tbody tr').first()).toContainText('Nguyen Van Test');
  });

  test('rejects bot submission without Wasm attestation', async ({ page, request }) => {
    // 1. Block Wasm
    await page.route('**/attestation.wasm', route => route.abort());

    // 2. Submit form
    await page.goto('/apexfintech');
    await page.fill('input[name="full_name"]', 'Bot User');
    await page.fill('input[name="phone"]', '0000');
    await page.click('button[type="submit"]');

    // 3. Should get rejected
    await expect(page.locator('text=Browser verification failed')).toBeVisible();
  });
});
```

### 20.5. Load Testing (k6 Pattern)

```javascript
// tests/load/landing-ingest.js
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 1000 },
    { duration: '1m', target: 10000 },
    { duration: '2m', target: 50000 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(99)<50'],
    http_req_failed: ['rate<0.001'],
  },
};

export default function () {
  const payload = JSON.stringify({
    tenant_id: `load-test-${__VU % 100}`,
    full_name: 'Load Test User',
    phone: `0909${String(__VU).padStart(6, '0')}`,
    email: `load${__VU}@example.com`,
    event_id: crypto.randomUUID(),
    idempotency_key: crypto.randomUUID(),
  });

  const res = http.post('http://api.rinco.local/api/ingest/v1/leads', payload, {
    headers: {
      'Content-Type': 'application/json',
      'X-HMAC-Signature': __ENV.HMAC_SECRET, // Generated by setup script
    },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'latency < 50ms': (r) => r.timings.duration < 50,
  });
}
```

### 20.6. Security Testing

```bash
# OWASP ZAP baseline scan
docker run -v $(pwd):/zap/wrk/:rw \
  owasp/zap2docker-stable \
  zap-baseline.py -t http://api.rinco.local

# SQL injection fuzzing
sqlmap -u "http://api.rinco.local/api/crm/v1/leads?id=1" \
       --batch --level=5 --risk=3 \
       --dbms=PostgreSQL

# eBPF security scan
bpftool prog show | grep rinco

# Container security scan
trivy image ghcr.io/itdoanh/rinco/crm-core:latest
```

## 21. Migration Plan cho Schema & Service

### 21.1. Nguyên tắc Migration

1. **Backward compatible:** Mọi migration phải tương thích ngược với code version cũ.
2. **Zero-downtime:** Không được downtime > 30s.
3. **Reversible:** Mỗi migration phải có file `down.sql` tương ứng.
4. **Test trên staging:** Bắt buộc chạy trên staging trước.
5. **Monitoring:** Theo dõi metric trong 24h sau deploy.

### 21.2. Workflow Migration

```
[Dev] Tạo migration file
   ↓
[CI] Lint SQL + dry-run trên staging snapshot
   ↓
[Review] 2 reviewers approve
   ↓
[Merge] Vào main branch
   ↓
[ArgoCD] Auto-sync lên staging cluster
   ↓
[Staging Smoke] Run smoke test 30 phút
   ↓
[Canary 10%] Deploy lên 10% production nodes
   ↓
[Monitor 1h] Check error rate, latency
   ↓
[Canary 50%] Nếu OK → 50% nodes
   ↓
[Monitor 1h]
   ↓
[Full rollout] 100% production
   ↓
[Monitor 24h]
```

### 21.3. Pattern: Thêm cột mới (Zero-Downtime)

```sql
-- +goose Up
-- +goose StatementBegin

-- Step 1: Thêm nullable column
ALTER TABLE leads ADD COLUMN ai_score DECIMAL(5,2);

-- Step 2: Tạo partial index cho performance
CREATE INDEX CONCURRENTLY idx_leads_ai_score 
ON leads(tenant_id, ai_score DESC) 
WHERE ai_score IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_leads_ai_score;
ALTER TABLE leads DROP COLUMN IF EXISTS ai_score;
-- +goose StatementEnd
```

Sau khi deploy migration, code sẽ bắt đầu ghi vào `ai_score`. Sau 7 ngày backfill xong, đánh `NOT NULL`:

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE leads ALTER COLUMN ai_score SET NOT NULL;
ALTER TABLE leads ALTER COLUMN ai_score SET DEFAULT 0;
-- +goose StatementEnd
```

### 21.4. Pattern: Đổi kiểu cột

```sql
-- +goose Up
-- +goose StatementBegin
-- Bước 1: Tạo cột mới
ALTER TABLE leads ADD COLUMN phone_new TEXT;

-- Bước 2: Backfill (chạy trong batch)
DO $$
DECLARE
  last_id UUID;
BEGIN
  LOOP
    UPDATE leads SET phone_new = phone
    WHERE id > COALESCE(last_id, '00000000-0000-0000-0000-000000000000'::UUID)
      AND phone_new IS NULL
    LIMIT 1000
    RETURNING id INTO last_id;
    EXIT WHEN last_id IS NULL;
    COMMIT;
  END LOOP;
END $$;

-- Bước 3: Tạo trigger cho write đồng thời
CREATE OR REPLACE FUNCTION sync_phone_new() RETURNS TRIGGER AS $$
BEGIN
  NEW.phone_new := NEW.phone;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER leads_phone_sync
BEFORE INSERT OR UPDATE OF phone ON leads
FOR EACH ROW EXECUTE FUNCTION sync_phone_new();
-- +goose StatementEnd
```

Sau khi ứng dụng đọc từ `phone_new`, drop `phone`:

```sql
ALTER TABLE leads DROP COLUMN phone;
ALTER TABLE leads RENAME COLUMN phone_new TO phone;
DROP TRIGGER leads_phone_sync;
DROP FUNCTION sync_phone_new();
```

### 21.5. Pattern: Microservice Decomposition

Khi cần tách 1 service lớn thành nhiều service nhỏ:

```
Trước:                          Sau:
[CRM-Core (monolith)]          [CRM-Lead Service]
                               [CRM-Deal Service]
                               [CRM-Activity Service]
```

Quy trình Strangler Fig Pattern:
1. **Tạo service mới** với API tương đương.
2. **Route 1% traffic** sang service mới (canary).
3. **Compare** kết quả giữa 2 version.
4. **Tăng dần** 10% → 50% → 100%.
5. **Deprecate** service cũ.

### 21.6. Migration Tracking Sheet

| Date | Service | Type | Risk | Status |
|------|---------|------|------|--------|
| 2026-09-15 | crm-core | Add column ai_score | Low | Pending |
| 2026-10-01 | crm-core | Change phone column | Medium | Pending |
| 2026-10-15 | chat-engine | Strangler → chat-engine-v2 | High | Pending |
| 2026-11-01 | tree-org | Refactor ltree path | Medium | Pending |

## 22. Disaster Recovery

### 22.1. RPO & RTO Targets

| Service | RPO (max data loss) | RTO (recovery time) | Backup frequency |
|---------|---------------------|---------------------|------------------|
| **api-gateway** | 0 (stateless) | 30s | N/A |
| **crm-core (Postgres)** | 5 min | 30 min | WAL continuous + daily |
| **tree-org (Postgres)** | 5 min | 30 min | WAL continuous + daily |
| **chat-engine (ScyllaDB)** | 1 hour | 2 hours | Daily snapshot |
| **landing-ingest (ScyllaDB)** | 1 hour | 2 hours | Daily snapshot |
| **analytics (ClickHouse)** | 1 hour | 4 hours | Daily |
| **MinIO/S3 (Media)** | 0 (replicated) | 1 hour | Cross-region replication |
| **Valkey (Cache)** | 0 (acceptable loss) | 5 min | AOF |
| **Tenant config (Postgres)** | 1 min | 15 min | WAL continuous |

### 22.2. Disaster Scenarios & Response

#### Scenario A: 1 Service Crash
**Detection:** K3s liveness probe fail sau 3 lần.
**Response:**
1. K3s restart pod (< 2s).
2. Nếu vẫn fail → restart toàn deployment (rolling restart).
3. Alert P2 tới SRE.
4. Tự phục hồi, không cần can thiệp.

#### Scenario B: 1 Node Fail
**Detection:** Heartbeat miss sau 30s.
**Response:**
1. K3s reschedule pods sang node khác.
2. WireGuard mesh route lại.
3. Alert P1, on-call kiểm tra log.
4. Recovery tự động trong 5-10 phút.

#### Scenario C: Database Primary Down
**Detection:** Health check fail.
**Response:**
1. PgBouncer route read sang replica.
2. Manual promote replica thành primary (< 5 phút).
3. WAL archive dùng để fill gap.
4. Alert P0, SRE manual.

#### Scenario D: Region Fail (Multi-Region)
**Detection:** Traffic drop > 80% trong 1 region.
**Response:**
1. DNS failover sang region khác (< 30s).
2. Replica region take over.
3. RPO có thể lên tới 5 phút.
4. Alert P0, war room.

#### Scenario E: Cyber Attack (Ransomware)
**Detection:** Anomaly AI score > threshold.
**Response:**
1. Auto-isolate infected tenant.
2. Block IP via eBPF XDP_DROP.
3. Restore từ backup ngày hôm qua.
4. Forensic analysis.
5. Alert P0, incident response team.

#### Scenario F: Data Corruption (Bad Migration)
**Detection:** Smoke test fail hoặc user report.
**Response:**
1. Rollback migration (`goose down`).
2. Rollback code version.
3. Restore data từ backup snapshot trước deploy.
4. RTO: 30 phút – 1 giờ.

### 22.3. Backup Strategy chi tiết

```bash
#!/bin/bash
# scripts/backup-postgres.sh
set -e

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/postgres/$TIMESTAMP"

# 1. Full backup
pg_basebackup -h postgres-primary -D $BACKUP_DIR/base \
  --checkpoint=fast --wal-method=stream

# 2. Compress
tar czf $BACKUP_DIR.tar.gz $BACKUP_DIR
rm -rf $BACKUP_DIR

# 3. Upload to MinIO
mc cp $BACKUP_DIR.tar.gz minio/backups/postgres/

# 4. Verify checksum
sha256sum $BACKUP_DIR.tar.gz > $BACKUP_DIR.tar.gz.sha256

# 5. Retain 30 days
mc rm --recursive --force --older-than 30d minio/backups/postgres/

echo "Backup completed: $BACKUP_DIR.tar.gz"
```

### 22.4. DR Drill (Hàng quý)

```yaml
# tests/dr/drill-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: dr-drill-q3-2026
spec:
  template:
    spec:
      containers:
      - name: dr-drill
        image: rinco/dr-drill:latest
        command: ["/bin/sh", "-c"]
        args:
        - |
          # 1. Tạo dummy tenant với 1000 Lead
          # 2. Snapshot Postgres
          # 3. Kill primary Postgres
          # 4. Promote replica
          # 5. Verify: tenant còn đọc/ghi được Lead không?
          # 6. Verify: không mất data (so sánh checksum)
          # 7. Pass/Fail report
        env:
        - name: SLACK_WEBHOOK
          valueFrom:
            secretKeyRef:
              name: dr-secrets
              key: slack-webhook
      restartPolicy: Never
```

## 23. Cost Estimation (rough numbers)

### 23.1. Infrastructure Cost (Cloud Provider – AWS/equivalent)

| Component | Spec | Qty | Unit price | Monthly |
|-----------|------|-----|------------|---------|
| **K3s Control Plane** | 4 vCPU, 16GB RAM | 3 | $120 | $360 |
| **Edge Gateway (Rust)** | 8 vCPU, 16GB, 10Gbps net | 4 | $200 | $800 |
| **API Gateway (Go)** | 4 vCPU, 8GB | 4 | $120 | $480 |
| **App Services** | 4 vCPU, 8GB | 12 | $120 | $1,440 |
| **PostgreSQL Primary** | 8 vCPU, 32GB, 500GB NVMe | 1 | $400 | $400 |
| **PostgreSQL Replica** | 8 vCPU, 32GB | 2 | $300 | $600 |
| **ScyllaDB** | 8 vCPU, 32GB, 1TB NVMe | 3 | $400 | $1,200 |
| **ClickHouse** | 16 vCPU, 64GB, 2TB NVMe | 3 | $600 | $1,800 |
| **Valkey Cluster** | 4 vCPU, 16GB | 3 | $150 | $450 |
| **MinIO Nodes** | 8 vCPU, 16GB, 4TB HDD | 4 | $300 | $1,200 |
| **GPU Recorder Node** | 16 vCPU, 32GB, NVIDIA L4 | 2 | $1,500 | $3,000 |
| **GPU AI Node (A100)** | 32 vCPU, 128GB, 1xA100 | 2 | $4,000 | $8,000 |
| **MongoDB** | 4 vCPU, 16GB, 200GB | 2 | $200 | $400 |
| **Meilisearch** | 4 vCPU, 8GB | 2 | $100 | $200 |
| **Object Storage (Backups)** | - | - | $0.023/GB | $500 |
| **Bandwidth** | 10TB egress | - | $0.05/GB | $500 |
| **Load Balancer** | - | 2 | $30 | $60 |
| **EIP** | - | 10 | $5 | $50 |
| **Total Infrastructure** | - | - | - | **~$21,440/month** |

### 23.2. AI Cost (GPU Inference)

| Service | GPU | Cost/hour | Hours/month | Monthly |
|---------|-----|-----------|-------------|---------|
| vLLM (chatbot) | 2xA100 | $4 | 720 | $2,880 |
| Whisper STT | 1xL4 | $0.7 | 720 | $504 |
| Recording | 2xL4 | $1.4 | 720 | $1,008 |
| AI Scoring | CPU only | - | - | $50 |
| AI SRE | 1xA100 (burst) | $2 | 100 | $200 |
| **Total AI** | - | - | - | **~$4,642/month** |

### 23.3. External Service Cost

| Service | Cost model | Monthly |
|---------|-----------|---------|
| Sentry (Team) | per seat | $300 |
| Datadog (or VictoriaMetrics self-host) | - | $0 (self-host) |
| Let's Encrypt | free | $0 |
| Cloudflare (DNS + DDoS) | Pro plan | $200 |
| GitHub (Team) | per seat | $400 |
| **Total External** | - | **~$900/month** |

### 23.4. Tổng chi phí hàng tháng

```
Infrastructure:  $21,440
AI:               $4,642
External:           $900
─────────────────────────
Tổng:            $26,982/month
```

Cho 10,000 tenants với 100K MAU mỗi tenant (~1M MAU tổng):
- Chi phí trên mỗi MAU: **~$27 / 1000 MAU = $0.027/MAU**
- Chi phí trên mỗi tenant: **$26,982 / 10,000 = ~$2.7/tenant/month**

### 23.5. So sánh với SaaS truyền thống

| SaaS | Chi phí/tenant/month (100 user) |
|------|---------------------------------|
| HubSpot Marketing Hub | $890 |
| Salesforce Sales Cloud | $1,500 |
| Intercom + Zoom + Calendly (bundle) | $500 |
| **RINCO** | **~$2.7** (chưa tính giá bán) |

→ **Tiết kiệm 90%+** so với SaaS truyền thống như cam kết trong OG2.

### 23.6. Break-even Analysis

| Giá bán/tenant/month | Số tenant để hòa vốn | Lợi nhuận ở 10K tenants |
|----------------------|----------------------|--------------------------|
| $50 | 540 | $473,018 |
| $100 | 270 | $973,018 |
| $200 | 135 | $1,973,018 |
| $500 | 54 | $4,973,018 |

## 24. Performance Budget & SLO Matrix

### 24.1. Service-Level Objectives (SLO)

| Service | Metric | Target | Measurement |
|---------|--------|--------|-------------|
| **api-gateway** | p99 latency | < 20ms | Prometheus histogram |
| | availability | ≥ 99.95% | Uptime check |
| | error rate | < 0.1% | 5xx / total |
| **landing-ingest** | p99 latency | < 50ms | Prometheus |
| | availability | ≥ 99.9% | Uptime |
| | write throughput | > 100K/s | Scylla load test |
| **crm-core** | p99 read latency | < 30ms | ClickHouse/Prom |
| | p99 write latency | < 50ms | Prometheus |
| | availability | ≥ 99.9% | Uptime |
| **chat-engine** | p99 message latency | < 50ms | Custom metric |
| | concurrent connections | > 100K per node | Load test |
| | availability | ≥ 99.9% | Uptime |
| **media-sfu** | p99 first frame | < 200ms | Custom metric |
| | concurrent streams | > 500 per node | Load test |
| | availability | ≥ 99.95% | Uptime |
| **meta-capi** | success rate | > 95% | Worker metric |
| | p99 latency to FB | < 500ms | Custom |
| **ai-scoring** | p99 inference | < 5ms | Custom |
| **ai-conversation** | first token | < 200ms | vLLM metric |
| **analytics dashboard** | p99 query | < 1s | ClickHouse |

### 24.2. Error Budget

Monthly error budget = (1 - SLO) × Time × Request volume

Ví dụ cho api-gateway:
- SLO: 99.95% → error budget = 0.05%
- 1 tháng = 2,592,000s
- Nếu 100K req/s → ~7.776 × 10¹¹ requests/tháng
- Error budget: 3.888 × 10⁸ errors

Nếu vượt error budget → freeze deploy, chỉ hotfix.

### 24.3. Performance Budget cho Frontend

| Asset | Budget |
|---|---|
| HTML | < 50KB |
| CSS | < 30KB (gzipped) |
| JS (critical) | < 100KB (gzipped) |
| Image (above fold) | < 200KB total |
| Font | < 50KB |
| **Total above fold** | **< 500KB** |
| FCP | < 0.4s |
| LCP | < 0.8s |
| TTI | < 1.5s |
| TBT | < 100ms |
| CLS | < 0.05 |

### 24.4. Capacity Planning

| Phase | DAU | Peak CCU | Storage/mo | Bandwidth/mo |
|-------|-----|----------|------------|--------------|
| MVP (Q1) | 10K | 1K | 100GB | 1TB |
| Phase 2 (Q2) | 100K | 10K | 1TB | 10TB |
| Phase 3 (Q3) | 500K | 50K | 5TB | 50TB |
| Phase 4 (Q4) | 1M | 100K | 10TB | 100TB |
| Year 2 | 10M | 1M | 100TB | 1PB |

## 25. Concrete Open Questions / TBD

### 25.1. Cần user xác nhận ngay ✋

| # | Câu hỏi | Options | Recommendation |
|---|---------|---------|----------------|
| Q1 | **Ngân sách ban đầu cho Phase 1?** | (a) Chỉ local Docker Compose, (b) + 1 VPS test, (c) Full K3s cluster | (b) – balance cost & reality |
| Q2 | **Có cần Dedicated Server cho ScyllaDB không?** | (a) Yêu cầu NVMe bare-metal, (b) Cloud instance OK | (a) cho performance |
| Q3 | **Phiên bản Go cụ thể?** | 1.22, 1.23, 1.24, 1.25, 1.26 | 1.26 (mới nhất 2026) |
| Q4 | **Có hỗ trợ on-premise hoàn toàn không (no cloud)?** | (a) Cloud-first, (b) Hybrid, (c) Full on-prem | (b) cho flexibility |
| Q5 | **Whisper model nào?** | tiny, base, small, medium, large | medium (best balance) |
| Q6 | **LLM cho chatbot: open-source hay commercial API?** | (a) Llama-3 self-host, (b) GPT-4 API, (c) Both | (a) – cost & privacy |
| Q7 | **Backup retention bao lâu?** | 7, 30, 90, 365 ngày | 90 ngày hot + 1 năm archive |
| Q8 | **Có cần hỗ trợ LDAP/SSO cho Enterprise?** | (a) Phase 1, (b) Phase 2, (c) Phase 3 | (b) |
| Q9 | **Mobile App native hay React Native?** | (a) Native (Swift/Kotlin), (b) RN, (c) Flutter | (c) Flutter – single codebase |
| Q10 | **Có cần hỗ trợ Email Marketing tích hợp không?** | (a) Yes full, (b) Yes basic, (c) No | (b) |

### 25.2. Cần quyết định trong Phase tiếp theo 📋

| # | Câu hỏi | Impact | Owner |
|---|---------|--------|-------|
| Q11 | Tenant isolation: shared DB + RLS hay DB-per-tenant? | Performance vs security | Architecture team |
| Q12 | Rate limit per tenant nên là bao nhiêu? | Quota, billing | Product |
| Q13 | Có cần hỗ trợ tenant custom-branding sâu? | Effort, cost | Design |
| Q14 | Có nên dùng CockroachDB thay Postgres để scale? | Operational complexity | Architecture |
| Q15 | Có cần hỗ trợ multi-language UI (i18n)? | Effort | Frontend |
| Q16 | GDPR/PDPA compliance level? | Legal, features | Legal |
| Q17 | Notification channels ưu tiên? | Effort | Product |
| Q18 | Cách tính usage-based billing? | Complexity | Finance |
| Q19 | Có cần Edge locations (Cloudflare Workers)? | Cost, perf | Architecture |
| Q20 | Data residency cho VN khách hàng? | Compliance | Legal |

### 25.3. TBD kỹ thuật ⏳

| # | Item | Status | Next step |
|---|------|--------|-----------|
| T1 | Chọn HAProxy vs Envoy cho API Gateway | TBD | Benchmark Q4 2026 |
| T2 | ClickHouse cluster size ban đầu | TBD | Test với 1 node first |
| T3 | NATS JetStream storage backend (file vs memory) | TBD | Test performance |
| T4 | Valkey vs KeyDB so sánh | TBD | Valkey 9.x chosen (theo yeucauthietke) |
| T5 | Meilisearch vs Typesense | TBD | Meilisearch chosen (Tiếng Việt tốt) |
| T6 | Cargo workspace structure | TBD | Decide after Rust services > 5 |
| T7 | WireGuard vs Tailscale vs Nebula | TBD | Headscale chosen (open source) |
| T8 | Prometheus vs VictoriaMetrics vs Mimir | TBD | VictoriaMetrics chosen (scale) |
| T9 | ClickHouse Grafana datasource vs custom BFF | TBD | Use direct Grafana plugin |
| T10 | Frontend monorepo (Turborepo/Nx) | TBD | Decide after apps > 3 |

### 25.4. Risk Register ⚠️

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| io_uring + Go GC conflict | High | Critical | Use Rust gateway only |
| AI vendor lock-in | Medium | High | Self-host Llama-3 fallback |
| ScyllaDB operational complexity | Medium | High | Hire DBA, automation |
| Multi-region latency | Medium | Medium | Single-region Phase 1 |
| WireGuard debugging | Medium | Medium | Mesh observability dashboard |
| Customer data leak | Low | Critical | Multi-layer defense (RLS, encryption, audit) |
| Vendor outage (Stripe, FB) | Medium | Medium | Multiple providers, offline mode |
| Open-source dependency CVE | High | Medium | Dependabot + Renovate weekly |

---

**Tài liệu này là bản thiết kế chính (master) phiên bản 1.1 đã được mở rộng. Mọi phần chi tiết phải đọc kèm theo các file trong `docs/01-*` đến `docs/11-*`.**