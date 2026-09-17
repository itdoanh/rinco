# ADR-0007: Chat Engine in Rust

> **Status**: Accepted · **Date**: 2026-09-18 · **Authors**: RINCO Platform Team

## Context

RINCO cần **real-time chat** với yêu cầu khắt khe:

1. **End-to-end encryption (E2EE)** — server không thấy plaintext.
2. **Signal Protocol** — X3DH key agreement + Double Ratchet algorithm.
3. **Sub-millisecond message delivery** — fan-out tới 100+ connected devices.
4. **High concurrency** — 100k+ concurrent WebSocket connections per pod.
5. **Presence** — online/offline + typing indicators.
6. **Devices** — multi-device per user (mobile + desktop + web).
7. **Channels + DMs** — group + 1-1.
8. **Low memory per connection** — để scale.
9. **Crash safety** — không mất message khi pod restart.

### So sánh các lựa chọn ngôn ngữ

| Aspect | Go | Rust | Python | Node.js (TypeScript) |
|--------|----|----|--------|---------------------|
| **WebSocket concurrency** | 🟢 Tốt (goroutine) | 🟢 Xuất sắc (tokio) | 🟡 GIL bottleneck | 🟡 Event loop OK |
| **Memory per conn** | 🟡 2-4 KB | 🟢 0.5-1 KB | 🔴 10+ KB | 🟡 5-8 KB |
| **CPU efficiency** | 🟡 Tốt | 🟢 Xuất sắc | 🔴 Chậm | 🟡 Khá |
| **Crypto lib (Signal)** | 🟡 `libsignal-protocol-c` (CGO) | 🟢 `libsignal-client` (native binding) hoặc pure Rust | 🔴 Yếu | 🟡 Cần native module |
| **Async runtime mature** | 🟢 | 🟢 | 🟢 (asyncio) | 🟢 |
| **Static binary deploy** | 🟢 | 🟢 | ❌ | ❌ |
| **Cold start** | 🟢 < 50ms | 🟢 < 30ms | 🔴 1-2s | 🟡 200ms |
| **Developer ecosystem** | 🟢 | 🟢 (growing) | 🟢 | 🟢 |
| **Compile-time safety** | 🟡 | 🟢 | 🔴 Runtime | 🟡 |

## Decision

Viết **chat-engine bằng Rust** với:

- **axum** cho HTTP/WS server.
- **tokio** async runtime.
- **scylla-rust-driver** cho ScyllaDB.
- **redis-rs** cho Valkey (presence).
- **libsignal-client** (Rust binding) hoặc pure-Rust Signal impl (xem bên dưới).
- **serde / prost** cho serialization.
- **tracing** + OpenTelemetry cho observability.

### Signal Protocol implementation

Có 2 lựa chọn:

#### Option A: `libsignal-client` (Signal Foundation official)

- **Pro**: chính thức, audited, compatible với Signal app.
- **Con**: dependency lớn, build time dài, một số feature chưa expose.
- **Code size**: ~50MB binary.

#### Option B: Pure Rust impl (`x3dh-rs`, custom Double Ratchet)

- **Pro**: dependency nhỏ, build nhanh, full control.
- **Con**: phải tự audit, dễ sai cryptographic detail.
- **Code size**: ~20MB binary.

**Decision**: Option A (`libsignal-client`) cho production — security không thể compromise.

### Kiến trúc

```
┌──────────────────────────────────────────────────────────┐
│                    chat-engine (Rust)                    │
│                                                          │
│  ┌────────────────────┐  ┌─────────────────────────┐    │
│  │  WebSocket Server  │  │  Connect-RPC / HTTP     │    │
│  │  (axum + tokio)    │  │  (key bundle, profile)  │    │
│  └──────────┬─────────┘  └──────────┬──────────────┘    │
│             │                       │                    │
│             ▼                       ▼                    │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Message Router                                   │    │
│  │  - Validate PASETO                               │    │
│  │  - Lookup device pubkey                          │    │
│  │  - Decrypt E2EE header (server can see metadata) │    │
│  │  - Route ciphertext to recipient                 │    │
│  └──────────────────┬──────────────────────────────┘    │
│                     │                                   │
│  ┌──────────────────┼──────────────────────────────┐    │
│  │                  │                               │    │
│  ▼                  ▼                               ▼    │
│ ┌────────┐    ┌──────────┐                  ┌──────────┐│
│ │ScyllaDB│    │ Valkey   │                  │ NATS     ││
│ │messages│    │presence  │                  │chat.*    ││
│ │devices │    │typing    │                  │          ││
│ │channels│    │fcm-push  │                  │          ││
│ └────────┘    └──────────┘                  └──────────┘│
└──────────────────────────────────────────────────────────┘
```

### ScyllaDB schema

```cql
CREATE TABLE chat.messages (
    channel_id    UUID,
    tenant_id     UUID,
    message_id    TIMEUUID,
    sender_id     UUID,
    sender_device UUID,
    ciphertext    BLOB,         -- E2EE payload
    header        BLOB,         -- DH ratchet header
    created_at    TIMESTAMP,
    PRIMARY KEY ((channel_id), message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);

CREATE TABLE chat.devices (
    user_id       UUID,
    device_id     UUID,
    pubkey        BLOB,         -- identity key
    spk           BLOB,         -- signed prekey
    spk_sig       BLOB,
    one_time_prekeys LIST<BLOB>,
    last_seen     TIMESTAMP,
    PRIMARY KEY (user_id, device_id)
);

CREATE TABLE chat.channels (
    channel_id    UUID PRIMARY KEY,
    tenant_id     UUID,
    type          TEXT,         -- 'dm' | 'group' | 'channel'
    members       SET<UUID>,
    created_at    TIMESTAMP
);
```

### Valkey usage

- `presence:<user_id>` → JSON `{device_id, online, last_heartbeat}`
- `typing:<channel_id>` → set of user_ids (TTL 10s)
- `fcm:<user_id>:<device_id>` → FCM token
- Pub/sub: `chat:room:<channel_id>` cho multi-pod fan-out

### Performance targets

| Metric | Target |
|--------|--------|
| WebSocket concurrent conn / pod | 100,000+ |
| Memory / connection | < 2 KB |
| Message delivery p50 | < 5ms (intra-pod) |
| Message delivery p99 | < 50ms (cross-pod + ScyllaDB write) |
| CPU per 1k conn (idle) | < 5% |
| Cold start (K8s pod ready) | < 10s |
| Binary size | < 80MB |

### Crash safety

- **ScyllaDB write-ahead log** đảm bảo message persistence.
- **Outbox pattern** cho NATS events.
- **Heartbeat** reconnect: client reconnect với last `message_id`, server replay missed messages.
- **Graceful shutdown**: drain connections trong 30s trước khi SIGTERM.

### Tại sao không Go?

- **Memory per connection**: Go ~2-4 KB, Rust ~0.5-1 KB → Rust tốt hơn 2-4×.
- **Crypto (Signal)**: Rust có `libsignal-client` native binding; Go phải dùng CGO qua `libsignal-protocol-c` (cũ, maintain không tốt).
- **Async**: Go goroutine OK nhưng tốn hơn tokio về memory overhead.

### Tại sao không Python?

- **GIL** — bottleneck cho IO-bound concurrent connections.
- **Memory** — Python 10+ KB per connection (vs Rust 1 KB).
- **Cold start** — 1-2s so với < 30ms Rust.

### Trade-offs

| Aspect | Trade-off |
|--------|-----------|
| **Compile time** | Rust build chậm hơn Go (~3-5 phút so với ~30 giây) |
| **Onboarding** | Ownership/borrow checker — dev phải học Rust |
| **Library ecosystem** | Một số niche library chưa có (vd: Signal library alternative) |
| **Hiring** | Ít Rust dev hơn Go dev |

→ Mitigate bằng:
- **CI cache** (sccache).
- **Internal training** (3-day Rust course cho team).
- **Pre-built base image** (cargo chef) để cache deps.

## Consequences

### Positive

- **Performance**: 100k+ connections per pod, < 2 KB memory each.
- **Security**: official Signal Protocol library, audited.
- **Resource efficiency**: ít CPU/RAM → cost thấp.
- **Static binary**: deploy đơn giản (chỉ 1 file).
- **Cold start**: < 10s cho K8s pod ready.

### Negative

- **Build time** lâu (~3-5 phút cho full release).
- **Onboarding cost** — dev phải biết Rust.
- **Library ecosystem** hẹp hơn Go.
- **Hiring** khó hơn.
- **Cryptographic library** (libsignal-client) dependency lớn.

### Mitigations

- **cargo chef** trong Dockerfile để cache dependencies.
- **Rust learning path** cho team (1 sprint training).
- **Trivy** weekly scan cho libsignal-client CVEs.

## Alternatives Considered

### A. Go chat-engine

- **Pro**: dev quen thuộc, build nhanh.
- **Con**: memory per conn cao hơn, Signal lib qua CGO.
- **Verdict**: 🟡 Cân nhắc nếu perf không phải bottleneck.

### B. Elixir (Phoenix Channels)

- **Pro**: BEAM VM xử lý concurrent tốt, "let it crash" model.
- **Con**: ít dev Elixir, không có Signal lib mature.
- **Verdict**: ❌ Rejected — ecosystem không phù hợp.

### C. C++ (WebSocket++ + mbedTLS)

- **Pro**: performance tốt nhất, full control.
- **Con**: không có borrow checker, dev cycle chậm, ít concurrent lib.
- **Verdict**: ❌ Rejected — modern Rust thay thế tốt.

### D. Erlang/OTP

- **Pro**: concurrency tuyệt vời.
- **Con**: ít dev, không có Signal lib, syntax lạ.
- **Verdict**: ❌ Rejected.

## References

- [Signal Protocol spec](https://signal.org/docs/)
- [libsignal-client repository](https://github.com/signalapp/libsignal)
- [axum documentation](https://docs.rs/axum/)
- [services/chat-engine](../../services/chat-engine) — implementation
- [ARCHITECTURE.md §3.2](../ARCHITECTURE.md#32-read-path-chat-messages)
- [ADR-0003 NATS Event Bus](0003-nats-event-bus.md)