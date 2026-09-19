# Chat Real-time Engine — Summary

> Phân hệ nhắn tin thời gian thực của RINCO, phục vụ chat 1-1, nhóm, channel công ty với độ trễ p99 < 50ms, scale tới **1 triệu CCU** (concurrent connections). Stack chính: **Rust + tokio-uring + FlatBuffers + ScyllaDB + eBPF/XDP + Valkey + NATS JetStream**, được tách thành nhiều microservices chạy stateless phía sau load balancer.

Nguồn: `docs/06-chat-engine/README.md` (≈ 5.400 dòng, 51 sections bao gồm hai bản mở rộng v1.x và v2.0 với code Rust/Go/TypeScript đầy đủ).

---

## 1. Triết lý kiến trúc & mục tiêu

| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| CH-1 | 1.000.000 CCU | Concurrent WebSocket |
| CH-2 | p99 latency < 50ms | Send → Receive |
| CH-3 | Persistence không block realtime | ScyllaDB async |
| CH-4 | Multi-tenant isolation | Per-tenant channels |
| CH-5 | Zero-copy packet | io_uring + FlatBuffers |

**Nguyên tắc cốt lõi:**
- **Async-first** — không bao giờ block event loop; mọi I/O phải qua kernel-bypass.
- **Batching** — gộp nhiều message thành một frame FlatBuffers để giảm overhead.
- **State placement** — Presence ở Valkey (RAM, TTL), History ở ScyllaDB (SSD, async).
- **Channel routing qua consistent hash** để phân tán đều gateway nodes.

### Tổng quan 4 tầng
```
[Web/App Clients]
       │ WebTransport / Zero-Copy WebSocket
       ▼
[Kernel-Bypass Gateway: Rust + io_uring + eBPF/XDP]
       │
       ├─► FlatBuffers Parser (Zero-Allocation)
       ├─► Auth + Tenant Resolver
       ├─► Singleflight Coalescing Layer
       ├─► Channel Router (Consistent Hash)
       ├─► Presence State (Valkey Cluster)
       └─► ScyllaDB Shard-per-core Chat Logs
                  │
                  └─► NATS JetStream (replication + cross-tenant sync)
```

---

## 2. Microservices trong hệ thống

| Service | Ngôn ngữ | Vai trò | Đặc tính |
|---------|----------|--------|----------|
| `chat-gateway` | Rust (tokio-uring) | Main WebSocket server | Zero-copy receive FlatBuffers, kết nối với eBPF/XDP |
| `chat-router` | Rust | Phân phối message giữa các gateway nodes | Consistent hash, NATS subscriber |
| `chat-presence` | Rust + Valkey | Online status, typing, last-seen | TTL-based keys 60s |
| `chat-history` | Rust + ScyllaDB | Message persistence với shard-per-core | TWCS compaction, async write |
| `chat-search` | Go + Meilisearch | Full-text search, hỗ trợ Tiếng Việt (charabia tokenizer) | Subscribe NATS `chat.index` |
| `chat-notify` | Go | Push notification offline (FCM, APNs, Web Push, Email digest, Telegram) | Multi-channel, 5 phút batching |

Deployment: **100+ stateless gateway nodes**, sticky session theo `connection_id`, auto-scale theo CCU, **Valkey + ScyllaDB + NATS JetStream** chạy shared cluster.

---

## 3. Protocol: FlatBuffers Binary

### Lý do chọn FlatBuffers so với JSON/Protobuf
- **Zero-copy read** — client đọc thẳng từ byte buffer, không unmarshal.
- **Zero-allocation** — không cần parse → giảm 90% RAM và GC overhead.
- **Schema evolution** — thêm field mới OK backward compatible, enum chỉ append, không reorder.

### `ChatFrame` (root_type) — 21 frame types
| Frame | Mục đích |
|-------|---------|
| AUTH / AUTH_OK / AUTH_FAIL | Đăng nhập (PASETO) + session |
| SUBSCRIBE / UNSUBSCRIBE | Subscribe channel |
| SEND_MESSAGE / ACK | Gửi và xác nhận |
| MESSAGE_BATCH | Server broadcast tin nhắn |
| HISTORY / RESUME_BATCH | Lịch sử + resume sau reconnect |
| PRESENCE / TYPING_UPDATE / READ_RECEIPT | Real-time states |
| REACTION_UPDATE / CHANNEL_UPDATE | Reaction + metadata |
| PING / PONG / ERROR | Heartbeat + lỗi |

### `ChatMessage` — 18 message types
TEXT, IMAGE, VIDEO, FILE, VOICE, STICKER, REACTION, REPLY, FORWARD, SYSTEM, DELETED, EDITED, POLL, LOCATION, CONTACT, SCHEDULED, PINNED, THREAD_REPLY.

### Schema quan trọng (đầy đủ ở §16.1)
Mỗi message mang cả `event_id` (server UUIDv7), `client_event_id` (client UUIDv7 cho dedup), `tenant_id`, `channel_id`, `thread_parent_id`, `reply_to`, `forwarded_from`, attachments (image/video/audio/voice/file với `blurhash`, `width`, `height`, `duration`), location, contact, poll (question/options/expires_at), reactions, flags (urgent/silent/announcement), `encrypted: bool`, `encryption_key_id`. ID dạng `schema_version: uint16` cho phép parse cả client cũ.

Generated Rust binding qua `flatc --rust -o src/generated rinco_chat.fbs`; verification trước khi dispatch.

---

## 4. Connection Layer: io_uring + eBPF/XDP

### Rust Gateway với tokio-uring
Mỗi connection sử dụng `RecvZc` (zero-copy receive trên io_uring), parse FlatBuffers trực tiếp từ slice byte nhận được (qua `unsafe { root_as_chat_frame(&buf[..n])? }`) — không copy, không Unmarshal. Worker pool pinned theo CPU qua `cpuset`, mỗi node xử lý 100K CCU.

### eBPF/XDP Load Balancer + Bot Filter (full §21.1)
Lập trình bằng C chạy trong kernel, gắn vào NIC interface. Chức năng:
- **Blacklist map** (BPF_MAP_TYPE_LRU_HASH, 100K entries): IP bị cấm với TTL — drop ngay ở XDP_DROP.
- **Rate-limit map** (token bucket 100 SYN/s per IP) — vượt ngưỡng → auto-blacklist 5 phút.
- **Stats map** (PERCPU_ARRAY) — pass / drop_blacklist / drop_rate_limit / drop_bot.
- **Allowed UA hash map** — whitelist cho testing infra.

Hoạt động ở layer 2 nên latency < 100ns, decision trước khi gói tin vào TCP/IP stack → giảm 5-15ms so với user-space.

### Connection Lifecycle
```
Client TCP SYN
   → eBPF/XDP filter (drop bots, rate-limit)
   → TLS handshake (Rustls)
   → Frame: AUTH (PASETO token + device_id + client_version + last_event_id)
   → Gateway verify + tenant resolve
   → Session create (UUIDv7)
   → Presence set_online (Valkey 60s TTL)
   → Subscribe default channels
   → Bidirectional frame flow
```

Exponential backoff reconnect 1s/2s/4s/8s/16s/32s/max 60s với jitter; khi reconnect gửi kèm `last_event_id` để resume.

---

## 5. Singleflight Coalescing — pattern hotspot killer

### Vấn đề
Channel lớn (50.000 thành viên), cùng mở app → 50.000 request đồng thời query history của channel đó → ScyllaDB overwhelmed.

### Giải pháp
Một `DashMap<String, Arc<OnceCell<Vec<ChatMessage>>>>` key là `channel_id:cursor`:
- Request đầu tiên (initiator) thực hiện DB query, các request sau (follower) chờ kết quả.
- Window 10ms gom các request đến gần nhau.
- Hard timeout 50ms/query.
- 50.000 request → 1 DB query duy nhất → broadcast kết quả (full code Rust ở §38).
- Cleanup map entry 50ms sau khi xong để follower cuối cùng kịp join.

### Test đã chạy
Test load 50.000 concurrent history request với mock Scylla delay 100ms → assert `call_count == 1` và elapsed < 2s (full code ở §32.1).

---

## 6. Persistence Layer: ScyllaDB + Valkey

### ScyllaDB Schema — 3 tables chính (keyspace `rinco_chat`, RF=3)

**`messages`** — partition key `(channel_id, tenant_id)`, clustering `event_time DESC, event_id`:
- Hỗ trợ tất cả trường text, attachments (list<frozen<attachment>>), reactions (map), reply_to, mentions, `deleted`, `edited_at`.
- TWCS compaction cho time-series.

**`user_channels`** — `(user_id, tenant_id)` partition → list channels của user với `joined_at`, `last_read_event_time`, `unread_count`, `muted`.

**`channels`** — `(channel_id, tenant_id)` partition → metadata (`name`, `type` 1-7, `members`, `admins`, `owner`, `avatar_url`, `description`).

### Shard-per-core
ScyllaDB tự phân shard theo core CPU; mỗi gateway node pinned vào 1 Scylla shard qua `cpuset`. Config `shard_count: 8`, `shard_aware_transport: true`.

### Write Path (async, fire-and-forget từ góc nhìn client)
Gateway nhận → Validate + Normalize → Write ScyllaDB async (target < 50ms ACK) → Emit NATS event `chat.message.{channel_id}` cho cross-node sync → Push qua WebSocket tới subscribers.

### Read Path
Client request history (cursor pagination) → Singleflight Coalescer → Query ScyllaDB (shard by `(channel_id, tenant_id)`) → Return binary FlatBuffers.

### Valkey Keys
```
presence:user:{tenant_id}:{user_id}   → JSON{status, device, last_seen, current_channel}, TTL 60s (refresh 30s)
presence:online:{tenant_id}           → Set of user_id, TTL 5 phút
typing:channel:{channel_id}           → Set of user_id, TTL 5 giây
meeting:room:{room_code}              → meeting metadata
member:channel:{channel_id}           → Set of user_id để check membership
rate:user:{user_id}                   → token bucket rate limit
subscriber:c-sales:{user_id}          → set subscribers cho broadcast
```

### Valkey-backed Presence Service (full §19.1)
- `set_online` / `set_offline` / `refresh` mỗi 30s.
- `set_current_channel` cập nhật last_seen.
- `start_typing` 5s TTL — broadcast qua NATS `chat.presence.{tenant}.{channel}`.

---

## 7. Channels, DMs, Group Chat, Permissions

### Channel Types (7 loại)
| Code | Loại | Use case |
|------|------|----------|
| 1 | DM | 1-1 chat |
| 2 | Group | Small group |
| 3 | Public channel | Company-wide, searchable |
| 4 | Private channel | Restricted |
| 5 | Company | Announcement |
| 6 | Broadcast | One-to-many |
| 7 | Thread | Reply dưới 1 message |

### Permission tiers
- **member**: Read + Write + Reaction.
- **admin**: + Add/Remove members, Edit channel.
- **owner**: + Delete channel, Transfer ownership.

### Visibility
Private (chỉ members), Internal (members + tenant admin), Public (anyone in tenant, searchable), External (invite user ngoài tenant, audit required).

### Channel Router Consistent Hash (§15.4, §20)
`BTreeMap<u64, gateway_id>` với 100-150 virtual nodes per gateway, hash bằng XxHash64. Method `route(channel_id)` → next clockwise trên ring hoặc wrap-around; `add_gateway` / `remove_gateway` chỉ remap ~1/N vnodes, không toàn bộ. Per-tenant pin qua `tenant_id` key.

---

## 8. File Upload Pipeline

### Presigned MinIO/S3 Direct Upload (full §22)
1. Client `GET /api/chat/v1/upload-url?type=image&size=...` → trả `{url, fields, file_id}`.
2. Client `POST` trực tiếp lên MinIO (multipart, từng chunk 5MB có ETag).
3. Client `POST /upload-complete` → server `complete multipart`.
4. Song song: **ClamAV virus scan** + **Image processing** (resize thumb/medium/full, strip EXIF, AVIF/WebP, blurhash).
5. Update message với `attachment_id`.

### Supported types & giới hạn
- Image jpg/png/gif/webp/avif — 10MB.
- Video mp4/webm — 200MB, transcode 720p.
- Audio mp3/ogg/wav — 20MB.
- Voice opus/webm — 5 phút max.
- File pdf/doc/xlsx/zip — 100MB.

Khi infected → server delete file + reject upload với `VIRUS_DETECTED`; abort multipart, refund quota.

---

## 9. End-to-End Encryption (E2EE) — Signal-style Double Ratchet (§23)

### Cryptographic primitives
- **Key exchange**: X3DH (Extended Triple Diffie-Hellman).
- **Symmetric**: AES-256-GCM / NaCl `secretbox` (XSalsa20-Poly1305).
- **Forward secrecy**: Double Ratchet Algorithm.

### `ChannelKeyManager` (TypeScript)
- Per-channel 32-byte symmetric `channelKey` được generate ngẫu nhiên (NaCl randomBytes).
- Mỗi member nhận `channelKey` được wrap bằng NaCl `box` (curve25519) với public key của họ.
- Server chỉ lưu `EncryptedMessage { ciphertext, nonce, key_id }` — không thể decrypt.
- Client encrypt bằng `secretbox` với channelKey trước khi gửi.

### Limitations khi E2EE bật
- ❌ Server không search messages.
- ❌ Server không filter spam/profanity.
- ❌ Push notification chỉ hiển thị "[Encrypted message]".
- ❌ File attachments phải encrypt URL only — server không xem trước được.

---

## 10. Search, Notification, AI Smart Reply (planned)

### Search (Meilisearch) — full §24
Index riêng `chat_messages` với filterable `[tenant_id, channel_id, sender_id, message_type, created_at]`, searchable `[text, sender_name]`, sortable `[created_at]`. Vietnamese tokenizer qua **charabia** + custom dictionary. NATS consumer subscribe `chat.index` tự động re-index sau khi message persisted.

### Notification Service — full §25
- **Multi-channel**: FCM (Android), APNs (iOS), Web Push, Email, Telegram Bot, SMS.
- **Priority 3 levels**:
  - High → gửi ngay (mention, DM).
  - Normal → batch 5 phút (group chat).
  - Low → daily digest.
- Filter theo notification settings per-channel (mute, keyword alert, priority).

### Per-channel notification settings (E2E2 Nitification)
Mute schedule, keyword alert, preview trên lock screen ẩn nội dung (privacy), action trong notification (reply inline), notification history. Channel silence: `Mute channel`, `Mute schedule`, `Notification per channel`, `Notification keywords alert`, `Notification priority`, `Group notifications`, `Notification preview (privacy)`, `Hide notification content (lock screen)`, `Action in notification (reply)`, `Notification history`.

---

## 11. Security Hardening (§29)

### Auth + Identity
- **PASETO token** verified locally (public key cached ở Valkey, TTL 60s) — không cần roundtrip đến Auth service.
- Channel permission check enforced cả ở gateway lẫn orchestrator.
- 24h edit window cho own message; admin hoặc owner được xóa mọi message.

### Content Moderation (§29.3)
Python service với:
- Toxic words dictionary (`toxic_words_vi.txt` cho Tiếng Việt).
- Spam regex (URL > 200 chars, lặp ký tự > 20 lần, viagra/casino/loan).
- PII pattern (phone VN, email, credit card, CMND/CCCD 9 hoặc 12 chữ số).
- Action: **block** / **warn** / **ok**.

### Network Security
- eBPF/XDP chống DDoS L7 + token bucket rate limit.
- TLS 1.3 với Rustls; không TLS 1.0/1.1.
- PASETO thay JWT (chống alg=none attack).

---

## 12. Scaling Strategy (§26)

### Sharding dimensions
| Shard Key | Use Case | Strategy |
|-----------|----------|----------|
| `channel_id + tenant_id` | Hot channel | Consistent hash, gateway pinning |
| `user_id` | User channels list | Hash % N |
| `tenant_id` | Tenant-wide ops | Single shard per tenant nếu nhỏ |
| `event_time` | Time-series | TWCS compaction |

### Capacity
Target 1M CCU / 100K CCU per gateway → 10 nodes (+2x redundancy = 20). Multi-region deployment dùng GeoDNS + per-region gateway pools, NATS JetStream replicate cross-region. Federation cho cross-region chat (VN ↔ US) là open question.

---

## 13. 30+ Edge Cases & Error Scenarios (§36)

Network: reconnect storm (10K disconnect cùng lúc — exponential backoff với jitter, gateway buffer 30s), half-open connection (heartbeat 60s), TLS handshake fail, eBPF false-positive, DDoS L7, captive portal injection.

Protocol: FlatBuffers parse error, schema version mismatch, frame > 64KB, unknown enum (default TEXT), UTF-8 invalid (replace U+FFFD), thread depth > 10 (flatten), duplicate `client_event_id` (idempotent ACK), replay attack (> 5 min old rejected).

Persistence: ScyllaDB timeout retry 3x → buffer Valkey Stream, ScyllaDB node down token-aware reroute, LWT timeout conflict → eventual consistency, Valkey down → LRU cache 512MB, Meilisearch lag → batch reindex.

Logic: message ordering across shards (clustering `event_time DESC` guarantee), large file upload fail → abort multipart + refund, bot spam 1K msg/s auto-ban 3 lần, mention @all 50K users rate-limit 5/s/user, channel deleted mid-conv → `CHANNEL_DELETED` frame, user banned → silent skip, E2EE key rotation conflict resolved by `key_version`, time skew > 5 min resync server_time, scheduled message TZ-aware, read receipt for E2EE chỉ gửi `message_hash`.

Presence & State: presence drift force re-fetch, typing spam > 1/s drop, multi-device read merge max(), revoke all sessions on logout, stale presence TTL expires tự nhiên.

Cross-cutting: NATS backlog full scale consumer, ClickHouse ingest lag, Meilisearch corrupted fallback ScyllaDB LIKE, MinIO disk full cleanup multipart, ScyllaDB partition > 100MB → split channel hoặc sub-partition theo week.

---

## 14. Disaster Recovery & Cost (§30, §31)

### RPO/RTO
- ScyllaDB messages: RPO 1h, RTO 2h — daily snapshot + WAL.
- Valkey presence: RPO 5min (AOF), RTO 30s — AOF replay.
- NATS JetStream: RPO 0 (replicated), RTO 10s — 3-node quorum.
- Session state: 0 in-memory + replicate, 30s.

### Cost (per-tenant, 1000 active users)
| Resource | Cost |
|----------|------|
| Gateway nodes (Rust) shared | $0.20 / 100K CCU shared |
| ScyllaDB cluster shared | $1.50 |
| Valkey cluster shared | $0.30 |
| NATS JetStream shared | $0.20 |
| Object storage (10GB/yr) | $0.50 |
| Push notifications | $0.10 / 1K users |
| Meilisearch shared | $0.20 |
| **Total** | **~$3/tenant/mo** |

10K tenants → ~$30K/mo. 100K tenants → ~$200K/mo (volume discount).

---

## 15. Testing Strategy (§32, §47)

### Test Pyramid
- **75% Unit (Rust `cargo test`, Go testify)** — pure functions, parser, coalescer.
- **20% Integration** — testcontainers (Scylla, Valkey, NATS, MinIO).
- **5% E2E** — Playwright (browser WS), Detox (mobile).

### Benchmarks targets
| Operation | p50 | p95 | p99 |
|-----------|-----|-----|-----|
| Send → ACK | 5ms | 20ms | **50ms** |
| Send → Receive (subscriber) | 10ms | 30ms | **50ms** |
| History query (50 msg) | 20ms | 50ms | 100ms |
| Singleflight 50K coalesced | 30ms | 100ms | 500ms |
| Reconnect + resume | 200ms | 500ms | 1s |
| 1M CCU per node | - | - | 100K CCU/node |

---

## 16. Implementation Roadmap

**16 tuần = 4 phase:**
1. **Core (1-4)** — Setup workspace, FlatBuffers + ScyllaDB schema, tokio-uring + eBPF, PASETO auth, send-message pipeline, singleflight, history + resume.
2. **Hardening (5-8)** — Valkey presence, Meilisearch, push notification, file upload pipeline, React UI.
3. **Production (9-12)** — Load test 1M CCU, E2EE, content moderation, OTel tracing, DR drill, React Native SDK.
4. **Scale (13-16)** — Multi-region, cross-region replication, federation, AI smart reply + summarization.

Acceptance gates mỗi tuần (ví dụ tuần 4: singleflight 50K → ≤10 queries, eBPF 100% SYN flood drop).

---

## 17. Danh sách ≥ 100 tính năng chat

### A. Tin nhắn cơ bản (30)
1. Gửi text message
2. Gửi emoji + custom emoji (tenant-specific)
3. Gửi ảnh (paste/drag/upload)
4. Gửi video
5. Gửi file (any type)
6. Gửi voice message (record từ mic)
7. Gửi sticker
8. Gửi GIF (Giphy integration)
9. Gửi location (lat/lng/address)
10. Gửi contact card
11. Reply to message (thread)
12. Forward message (cross-channel)
13. Edit message (trong 24h)
14. Delete message (for me / for everyone)
15. Pin message (channel)
16. Bookmark message (saved)
17. Search message (Meilisearch)
18. Quote message
19. Mention user (@user)
20. Mention channel (#channel)
21. Mention everyone (@all) — admin-only
22. Hashtag
23. Link preview (auto fetch metadata)
24. Code block formatting
25. Markdown formatting (GFM)
26. Rich text formatting (bold/italic/underline)
27. Code syntax highlight
28. Mention user group
29. Schedule message (gửi sau, TZ-aware)
30. Draft (auto-save mỗi keystroke)

### B. Reactions & Engagement (20)
31. React với emoji
32. Custom reactions (tenant upload)
33. Multiple reactions per message
34. Top reactions display
35. Reply in thread
36. Thread view (nested 1 level)
37. Follow thread (notifications)
38. Unfollow thread
39. Pin thread
40. Mark unread
41. Mark read
42. Read receipts (per-user toggle visible)
43. Last seen timestamp
44. Typing indicator (TTL 5s)
45. Voice activity indicator (in-call)
46. Online status (online/away/offline)
47. Custom status message
48. Away message
49. Do not disturb mode
50. Mute channel (permanent / schedule)

### C. Channels & Groups (20)
51. Tạo DM (1-1)
52. Tạo group chat
53. Tạo public channel
54. Tạo private channel
55. Tạo broadcast channel (announcement)
56. Add member
57. Remove member
58. Leave channel
59. Join public channel
60. Invite via link (expiry)
61. Invite via email
62. Channel settings
63. Channel description
64. Channel avatar (upload)
65. Channel topic
66. Channel pinned messages
67. Channel permissions (RBAC matrix)
68. Channel moderation (admin tools)
69. Channel archival
70. Cross-tenant channel (external guests, audit required)

### D. Search & Discovery (15)
71. Search messages (full-text Tiếng Việt)
72. Filter search (sender, date, type)
73. Search trong 1 channel
74. Global search (tenant-wide)
75. Filter by file type
76. Search media (image/video)
77. Saved searches
78. Recent searches
79. Trending messages
80. Pinned across channels
81. Jump to date
82. Jump to message (deep link)
83. Deep link notification → message
84. Search highlight (matched term)
85. Search suggestion (autocomplete)

### E. Notifications (15)
86. Push notification (browser/mobile FCM/APNs/Web Push)
87. Email digest (daily/weekly)
88. Notification sound
89. Notification per-channel
90. Notification keywords alert
91. Notification mute schedule (work hours)
92. Notification priority (high/normal/low)
93. Group notifications (collapse)
94. Notification preview (privacy lock-screen)
95. Hide notification content
96. Action in notification (inline reply)
97. Notification history
98. Telegram bot integration
99. Slack bridge
100. Discord bridge
101. SMS fallback cho keyword critical

### F. Admin & Moderation (20)
102. Delete any message (admin)
103. Ban user from channel (permanent)
104. Mute user in channel (timeout)
105. Warn user (visible)
106. Audit log per channel
107. Report message
108. Report user
109. Content moderation (AI Vietnamese)
110. Profanity filter
111. Spam detection (auto-ban)
112. Rate limit per user (30 msg/min)
113. Slow mode (channel 5s/msg)
114. Message retention policy (GDPR)
115. Auto-delete after N days
116. Legal hold (compliance)
117. Data export (GDPR Article 15)
118. Right to be forgotten (delete all user data)
119. End-to-end encryption (optional per channel, Signal Protocol)
120. Channel archival (read-only)
121. Channel analytics (msg/day, active members)
122. Multi-device session (mobile + web sync)
123. Multi-region chat (cross-region federation)
124. Custom emoji per tenant (upload)
125. Matrix federation (interop)
126. Bot API (REST/WebSocket)
127. Reactions limit (max 50/message)
128. Thread depth (flat 1 level)
129. Voice transcript (Whisper auto STT)
130. AI smart reply (GPT/Llama-3)

### G. File & Attachment (8)
131. Presigned S3/MinIO upload
132. Image resize (thumb/medium/full)
133. Strip EXIF privacy
134. WebP/AVIF conversion
135. Blurhash placeholder
136. Virus scan ClamAV
137. Voice message waveform
138. Multipart chunked upload (resumable)

### H. E2EE & Security (6)
139. X3DH key exchange
140. Double Ratchet forward secrecy
141. Per-channel symmetric key
142. Server only stores ciphertext
143. Member key rotation (multi-device)
144. Replay protection (nonce + timestamp)

### I. Realtime Infra (8)
145. WebSocket zero-copy
146. FlatBuffers binary protocol
147. io_uring kernel bypass
148. eBPF/XDP DDoS filter
149. Singleflight coalescing 50K→1
150. Consistent hash routing
151. Shard-per-core ScyllaDB
152. NATS JetStream cross-region replication

---

## 18. Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-CH-01 | 100K concurrent connections per node | Load test k6 |
| AC-CH-02 | Send → Receive p99 < 50ms | Benchmark |
| AC-CH-03 | 50K requests → 1 DB query (singleflight) | Trace |
| AC-CH-04 | Message persisted < 100ms | p95 |
| AC-CH-05 | Reconnect < 1s | Network test |

---

## 19. API Surface

### REST
```
POST   /api/chat/v1/channels
GET    /api/chat/v1/channels
GET    /api/chat/v1/channels/:id
PATCH  /api/chat/v1/channels/:id
DELETE /api/chat/v1/channels/:id
POST   /api/chat/v1/channels/:id/members
DELETE /api/chat/v1/channels/:id/members/:user_id
GET    /api/chat/v1/channels/:id/messages?cursor=...
POST   /api/chat/v1/channels/:id/messages
PATCH  /api/chat/v1/messages/:id
DELETE /api/chat/v1/messages/:id
POST   /api/chat/v1/upload-url
POST   /api/chat/v1/upload-complete
GET    /api/chat/v1/search?q=...
```

### WebSocket `wss://chat.rinco.app/ws/chat`
Frame types: `auth`, `subscribe_channel`, `unsubscribe_channel`, `send_message`, `typing`, `read_receipt`, `presence`, `ping` — all binary FlatBuffers.

---

## 20. UI/UX Stack

- **Web**: React 18 + TypeScript, Tailwind CSS + Lucide Icons, shadcn/ui, Zustand (state) + TanStack Query (server state), `flatbuffers` (binary protocol client SDK), mediasoup-client (voice), react-virtuoso (virtualized list, scroll 60fps với 10K messages), framer-motion.
- **Mobile**: React Native (Expo) iOS/Android, native gesture, FCM/APNs push, offline queue + sync.

### Layout
```
┌────────────────────────────────────────────────┐
│ Top Bar: Tenant | Search | Profile | Status    │
├──────────┬─────────────────────────────────────┤
│ Sidebar  │  Channel Header                     │
│ ▸ Channels│  ├─────────────────────────────┤    │
│ ▸ DMs    │  │ Pinned Messages              │    │
│ ▸ Search │  ├─────────────────────────────┤    │
│ ▸ Saved  │  │ Message List (infinite scroll)│    │
│ ▸ Files  │  │ ...                          │    │
│          │  ├─────────────────────────────┤    │
│          │  │ Input Bar                    │    │
│          │  └─────────────────────────────┘    │
│          │  Member List (right panel)           │
└──────────┴─────────────────────────────────────┘
```

Components: ChannelList (search/filter/sort), MessageList virtualized, MessageBubble markdown + attachments, InputBar (mention/emoji/file/voice), ThreadView, ReactionPicker, FilePreview (gallery/video), SearchPanel.

---

## 21. Open Questions cần user xác nhận (§34)

Có **20 câu hỏi mở**, trong đó P0-blocker:
1. Storage tier — ScyllaDB + Valkey đã đủ chưa hay cần PostgreSQL metadata?
2. E2EE priority — Phase 3 optional hay skip để rely on TLS?
3. Cross-region chat (VN ↔ US federation)?
4. Voice/Video message trong SFU riêng hay chỉ record + upload?
5. Read receipts default ON/OFF?
6. Message retention default (1 / 5 year / forever)?
7. Group chat max members (100 / 500 / 5000)?
8. DM history self-delete?
9. Cross-tenant chat (external channel)?
10. Custom emoji per tenant?

Khác: Matrix federation? Bot API? Reactions limit? Thread depth? Voice transcript Whisper? AI smart reply? Notification batching max? Backup retention? Schema versioning policy? Multi-device sync mechanism?

---

**Tổng kết:** Chat Engine là phân hệ phức tạp nhất của RINCO, scale tới 1M CCU với p99 < 50ms nhờ kết hợp Rust + io_uring zero-copy, FlatBuffers binary protocol, eBPF/XDP DDoS protection, singleflight coalescing chống hotspot, ScyllaDB shard-per-core. E2EE theo Signal Protocol cho phép opt-in per channel (tradeoff: search/notification). 152 tính năng chat tổng cộng qua 9 categories. Implementation 16 tuần chia 4 phase với acceptance gate rõ ràng. Chi phí ~$3/tenant/tháng.
