# RINCO – Bản Thiết Kế Hệ Thống Tổng Thể (Master Design)

> **Phiên bản:** v1.0  
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

**Kiếu triển khai:**
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

**Tài liệu này là bản thiết kế chính (master). Mọi phần chi tiết phải đọc kèm theo các file trong `docs/01-*` đến `docs/11-*`.**