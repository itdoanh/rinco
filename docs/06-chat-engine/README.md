# Phần 6 – Chat Real-time Engine (Hyper-Scale Chat)

> **Phân hệ:** Nhắn tin thời gian thực cho mọi tenant, hỗ trợ chat 1-1, nhóm, channel công ty.  
> **Mục tiêu:** Triệu user đồng thời với độ trễ < 50ms, kernel bypass, FlatBuffers, singleflight coalescing.  
> **Đặc thù:** Rust + io_uring + FlatBuffers + ScyllaDB Shard-per-core + eBPF/XDP.

---

## Mục lục
1. [Mục tiêu & Nguyên tắc](#1-mục-tiêu--nguyên-tắc)
2. [Kiến trúc](#2-kiến-trúc)
3. [Protocol: FlatBuffers Binary](#3-protocol-flatbuffers-binary)
4. [Connection Layer: io_uring + eBPF](#4-connection-layer-io_uring--ebpf)
5. [Singleflight Coalescing](#5-singleflight-coalescing)
6. [Persistence Layer](#6-persistence-layer)
7. [Presence & State](#7-presence--state)
8. [Channel & Permission Model](#8-channel--permission-model)
9. [File Upload](#9-file-upload)
10. [Danh sách tính năng (≥ 100)](#10-danh-sách-tính-năng)
11. [Database Schema](#11-database-schema)
12. [API Surface](#12-api-surface)
13. [UI/UX](#13-uiux)

---

## 1. Mục tiêu & Nguyên tắc

### 1.1. Mục tiêu
| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| CH-1 | 1 triệu CCU | Concurrent WebSocket |
| CH-2 | p99 latency < 50ms | Send → Receive |
| CH-3 | Persistence không block realtime | ScyllaDB async |
| CH-4 | Multi-tenant isolation | Per-tenant channels |
| CH-5 | Zero-copy packet | io_uring + FlatBuffers |

### 1.2. Nguyên tắc
- **Async-first:** Không bao giờ block event loop.
- **Batching:** Gộp nhiều message thành 1 frame.
- **Presence ở Valkey, History ở ScyllaDB.**
- **Channel routing qua consistent hash.**

---

## 2. Kiến trúc

### 2.1. Sơ đồ
```
[Web/App Clients]
        │ WebTransport / Zero-Copy WebSocket
        ▼
[Kernel-Bypass Gateway (Rust + io_uring + eBPF/XDP)]
        │
        ├─► FlatBuffers Parser (Zero-Allocation)
        ├─► Auth + Tenant Resolver
        ├─► Singleflight Coalescing Layer
        ├─► Channel Router (Consistent Hash)
        │
        ├─► Presence State (Valkey Cluster)
        └─► ScyllaDB Shard-per-core Chat Logs
                │
                └─► NATS JetStream (replication + sync)
```

### 2.2. Services
- **`chat-gateway`** (Rust + tokio-uring): Main WS server.
- **`chat-router`** (Rust): Distribute messages across gateway nodes.
- **`chat-presence`** (Rust + Valkey): Online status, typing.
- **`chat-history`** (Rust + ScyllaDB): Message persistence.
- **`chat-search`** (Go + Meilisearch): Full-text search.
- **`chat-notify`** (Go): Push notifications khi offline.

### 2.3. Deployment
- 100+ gateway nodes (chạy song song, stateless).
- Sticky session theo `connection_id`.
- Auto-scale theo CCU.

---

## 3. Protocol: FlatBuffers Binary

### 3.1. Lý do chọn FlatBuffers
- **Zero-copy read:** Client đọc trực tiếp từ byte buffer.
- **Zero-allocation:** Không cần Unmarshal → giảm 90% RAM.
- **Schema evolution:** Backward compatible.

### 3.2. Message Schema
```fbs
namespace Rinco.Chat;

enum MessageType : byte {
  TEXT = 0,
  IMAGE = 1,
  VIDEO = 2,
  FILE = 3,
  VOICE = 4,
  STICKER = 5,
  REACTION = 6,
  REPLY = 7,
  FORWARD = 8,
  SYSTEM = 9,
  DELETED = 10,
  EDITED = 11,
  POLL = 12,
  LOCATION = 13,
  CONTACT = 14,
}

table ChatMessage {
  event_id: string;        // UUIDv7
  tenant_id: string;
  channel_id: string;
  sender_id: string;
  message_type: MessageType;
  text: string;
  metadata: [MetadataEntry];  // Map
  mentions: [string];
  reply_to: string;
  forwarded_from: string;
  attachments: [Attachment];
  reactions: [Reaction];
  created_at: long;
  edited_at: long;
  deleted: bool;
}

table ChatFrame {
  frame_type: FrameType;
  messages: [ChatMessage];
  presence_updates: [PresenceUpdate];
  typing_indicators: [TypingIndicator];
  read_receipts: [ReadReceipt];
  server_timestamp: long;
}

enum FrameType : byte {
  MESSAGE_BATCH = 0,
  PRESENCE = 1,
  TYPING = 2,
  READ = 3,
  ACK = 4,
  PING = 5,
  ERROR = 6,
}

root_type ChatFrame;
```

### 3.3. Client SDK (TypeScript)
```typescript
import { ChatFrame, ChatMessage } from '@rinco/chat-proto';

const frame = ChatFrame.create({
  frame_type: 0,
  messages: [{
    event_id: uuidv7(),
    tenant_id: 'apex',
    channel_id: 'c-sales',
    sender_id: 'u-123',
    message_type: 0,
    text: 'Hello!',
    created_at: Date.now(),
  }]
});
const bytes = frame.pack();  // Binary buffer
socket.send(bytes);
```

---

## 4. Connection Layer: io_uring + eBPF

### 4.1. Rust Gateway với tokio-uring
```rust
use tokio_uring::net::TcpStream;
use io_uring::types::RecvZc;

async fn handle_connection(stream: TcpStream) {
    let mut buf = vec![0u8; 4096];
    loop {
        // Zero-copy receive
        let (res, buf_) = stream.recv_zc(&mut buf).await;
        let n = res?;
        // Process FlatBuffers in-place (no copy)
        let frame = unsafe { root_as_chat_frame(&buf_[..n])? };
        // Forward to router
        router.dispatch(frame).await;
    }
}
```

### 4.2. eBPF/XDP Drop Bot
```c
// eBPF program drop known bot User-Agent
SEC("xdp")
int xdp_filter_bot(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;
    
    // Parse TLS ClientHello (simplified)
    struct tls_hello *hello = parse_tls_hello(data, data_end);
    if (hello) {
        if (is_known_bot_ua(hello->user_agent)) {
            return XDP_DROP;
        }
    }
    return XDP_PASS;
}
```

### 4.3. Connection Lifecycle
```
1. Client connect TCP → eBPF filter
2. TLS handshake (Rustls)
3. Client send JWT (PASETO) in first frame
4. Gateway verify + tenant resolve
5. Channel subscribe
6. Bidirectional frame flow
```

---

## 5. Singleflight Coalescing

### 5.1. Vấn đề Hotspot
- Channel có 50.000 thành viên, cùng mở app.
- 50.000 request cùng query history của channel.
- → ScyllaDB overwhelmed.

### 5.2. Giải pháp
```rust
// rust: singleflight coalescing
use std::sync::Arc;
use tokio::sync::OnceCell;

struct Coalescer {
    inflight: Arc<DashMap<String, Arc<OnceCell<Vec<ChatMessage>>>>>,
}

async fn fetch_history_coalesced(
    &self,
    channel_id: &str,
    cursor: &str,
) -> Result<Vec<ChatMessage>> {
    let key = format!("{}:{}", channel_id, cursor);
    
    // Group by 10ms window
    let cell = self.inflight.entry(key).or_insert_with(|| {
        Arc::new(OnceCell::new())
    }).clone();
    
    cell.get_or_init(|| async {
        // Only 1 actual DB query per 10ms window
        let result = self.scylla.fetch_history(channel_id, cursor).await?;
        Ok(result)
    }).await.clone()
}
```

### 5.3. Kết quả
- 50.000 request → 1 DB query.
- Broadcast kết quả tới 50.000 clients.

---

## 6. Persistence Layer

### 6.1. ScyllaDB Schema
```cql
CREATE KEYSPACE rinco_chat WITH replication = {
  'class': 'NetworkTopologyStrategy',
  'datacenter1': 3
};

-- Messages per channel
CREATE TABLE rinco_chat.messages (
  channel_id text,
  tenant_id text,
  event_time bigint,
  event_id uuid,
  sender_id text,
  message_type tinyint,
  text text,
  metadata map<text, text>,
  attachments list<frozen<attachment>>,
  reactions map<text, frozen<reaction>>,
  reply_to text,
  mentions list<text>,
  deleted boolean,
  edited_at bigint,
  PRIMARY KEY ((channel_id, tenant_id), event_time, event_id)
) WITH CLUSTERING ORDER BY (event_time DESC);

-- User channels
CREATE TABLE rinco_chat.user_channels (
  user_id text,
  tenant_id text,
  channel_id text,
  joined_at bigint,
  last_read_event_time bigint,
  unread_count int,
  muted boolean,
  PRIMARY KEY ((user_id, tenant_id), channel_id)
);

-- Channel metadata
CREATE TABLE rinco_chat.channels (
  channel_id text,
  tenant_id text,
  name text,
  type tinyint,             -- 1:DM, 2:Group, 3:Public, 4:Company
  members list<text>,
  admins list<text>,
  created_at bigint,
  avatar_url text,
  description text,
  PRIMARY KEY ((channel_id, tenant_id))
);
```

### 6.2. Shard-per-core
- ScyllaDB tự phân shard theo core.
- Mỗi gateway node pinned vào 1 Scylla shard.
- `cpuset` Linux để constrain CPU.

### 6.3. Write Path
```
Gateway nhận message
    │
    ▼
Validate + Normalize
    │
    ▼
Write ScyllaDB (async, fire-and-forget)
    │
    ▼
Emit NATS event cho cross-tenant sync
    │
    ▼
Push qua WebSocket tới subscribers
```

### 6.4. Read Path
```
Client request history (cursor pagination)
    │
    ▼
Gateway → Singleflight Coalescer
    │
    ▼
Query ScyllaDB (shard by channel_id)
    │
    ▼
Return binary FlatBuffers
```

---

## 7. Presence & State

### 7.1. Valkey Schema
```
Key: presence:user:{user_id}
Value: { status: "online", device: "web", last_seen: 1234567890, current_channel: "c-123" }
TTL: 60s (refresh mỗi 30s)

Key: presence:channel:{channel_id}
Value: Set of user_id currently online
TTL: 5 minutes

Key: typing:channel:{channel_id}
Value: List of user_id currently typing
TTL: 5s
```

### 7.2. Online Status
- `online`: Active trong 5 phút.
- `away`: Active 5-30 phút.
- `offline`: > 30 phút hoặc disconnect.

### 7.3. Push Notification (Offline)
- Khi user offline > 30s nhận message → push notification.
- FCM (Android), APNs (iOS), Web Push (Browser).
- Telegram Bot nếu user link.

---

## 8. Channel & Permission Model

### 8.1. Channel Types
| Type | Code | Use case |
|------|------|----------|
| **DM** | 1 | 1-1 chat |
| **Group** | 2 | Small group chat |
| **Public Channel** | 3 | Company-wide announcement |
| **Private Channel** | 4 | Restricted access |
| **Thread** | 5 | Reply dưới 1 message |

### 8.2. Permission
- `member`: Read + Write + Reaction.
- `admin`: + Add/Remove members, Edit channel.
- `owner`: + Delete channel, Transfer ownership.

### 8.3. Visibility
- **Private:** Chỉ members thấy.
- **Internal:** Members + tenant admin.
- **Public:** Anyone in tenant (searchable).
- **External:** Có thể mời user ngoài tenant (audit required).

---

## 9. File Upload

### 9.1. Presigned S3 Direct Upload
```
1. Client GET /api/chat/v1/upload-url?type=image&size=5242880
2. Server trả { url, fields, file_id }
3. Client POST trực tiếp lên MinIO/S3
4. MinIO webhook → Server update message
5. Client send message với attachment_id
```

### 9.2. Supported File Types
- Image: jpg, png, gif, webp, avif (max 10MB).
- Video: mp4, webm (max 200MB, transcode to 720p).
- Audio: mp3, ogg, wav (max 20MB).
- File: pdf, doc, xlsx, zip (max 100MB).
- Voice message: opus/webm (max 5 phút).

### 9.3. Image Processing
- Server-side resize (thumb, medium, full).
- Strip EXIF (privacy).
- WebP/AVIF conversion.
- Blurhash placeholder.

### 9.4. Virus Scan
- ClamAV integration.
- Async scan, nếu nhiễm → file bị block.

---

## 10. Danh sách tính năng (≥ 100)

### 10.1. Tin nhắn cơ bản (1-30)
1. Gửi text message.
2. Gửi emoji + custom emoji.
3. Gửi ảnh (paste/drag/upload).
4. Gửi video.
5. Gửi file (any type).
6. Gửi voice message (record từ mic).
7. Gửi sticker.
8. Gửi GIF (Giphy integration).
9. Gửi location.
10. Gửi contact card.
11. Reply to message.
12. Forward message.
13. Edit message (trong 24h).
14. Delete message (for me / for everyone).
15. Pin message.
16. Bookmark message.
17. Search message.
18. Quote message.
19. Mention user (@user).
20. Mention channel (#channel).
21. Mention everyone (@all) – restricted.
22. Hashtag.
23. Link preview.
24. Code block formatting.
25. Markdown formatting.
26. Rich text formatting (bold/italic/underline).
27. Code syntax highlight.
28. Mention user group.
29. Schedule message (gửi sau).
30. Draft (auto-save).

### 10.2. Reactions & Engagement (31-50)
31. React với emoji.
32. Custom reactions (tenant-specific).
33. Multiple reactions per message.
34. Top reactions display.
35. Reply in thread.
36. Thread view.
37. Follow thread.
38. Unfollow thread.
39. Pin thread.
40. Mark unread.
41. Mark read.
42. Read receipts (visible to user toggle).
43. Last seen.
44. Typing indicator.
45. Voice activity indicator (in call).
46. Online status.
47. Custom status message.
48. Away message.
49. Do not disturb mode.
50. Mute channel.

### 10.3. Channels & Groups (51-70)
51. Tạo DM (1-1).
52. Tạo group chat.
53. Tạo public channel.
54. Tạo private channel.
55. Tạo broadcast channel (announcement).
56. Add member.
57. Remove member.
58. Leave channel.
59. Join public channel.
60. Invite via link.
61. Invite via email.
62. Channel settings.
63. Channel description.
64. Channel avatar.
65. Channel topic.
66. Channel pinned messages.
67. Channel permissions.
69. Channel moderation.
70. Archive channel.

### 10.4. Search & Discovery (71-85)
71. Search messages (full-text).
72. Search filter (sender, date, type).
73. Search trong 1 channel.
74. Global search.
75. Search by file type.
76. Search by media.
77. Saved searches.
78. Recent searches.
79. Trending messages.
80. Pinned across channels.
81. Jump to date.
82. Jump to message (link).
83. Deep link (notification → message).
84. Search highlight.
85. Search suggestion.

### 10.5. Notifications (86-100)
86. Push notification (browser/mobile).
87. Email digest.
88. Notification sound.
89. Notification per channel.
90. Notification keywords alert.
91. Notification mute schedule.
92. Notification priority.
93. Group notifications.
94. Notification preview (privacy).
95. Hide notification content (lock screen).
96. Action in notification (reply).
97. Notification history.
98. Telegram bot integration.
99. Slack bridge.
100. Discord bridge.

### 10.6. Admin & Moderation (101-120)
101. Delete any message.
102. Ban user from channel.
103. Mute user in channel.
104. Warn user.
105. Audit log per channel.
106. Report message.
107. Report user.
108. Content moderation (AI filter).
109. Profanity filter.
110. Spam detection.
111. Rate limit per user.
112. Slow mode (channel).
113. Message retention policy.
114. Auto-delete after N days.
115. Legal hold (compliance).
116. Data export (GDPR).
117. Right to be forgotten.
118. E2E encryption (optional).
119. Channel archival.
120. Channel analytics.

---

## 11. Database Schema (Đã trình bày ở mục 6.1)

---

## 12. API Surface

### 12.1. REST
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

### 12.2. WebSocket
```
WS /ws/chat?token=...
   Frame types:
   - auth
   - subscribe_channel
   - unsubscribe_channel
   - send_message
   - typing
   - read_receipt
   - presence
   - ping
```

---

## 13. UI/UX

### 13.1. Layout
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

### 13.2. Components
- ChannelList (search, filter, sort).
- MessageList (virtualized, infinite scroll, jump-to).
- MessageBubble (markdown render, attachments).
- InputBar (mention picker, emoji picker, file upload, voice record).
- ThreadView (replies).
- ReactionPicker.
- FilePreview (image gallery, video player).
- SearchPanel (advanced filters).
- NotificationSettings.
- ChannelSettings.

### 13.3. Tech Frontend
- React 18 + TypeScript.
- Tailwind CSS + Lucide Icons.
- shadcn/ui components.
- Zustand (state) + TanStack Query (server state).
- flatbuffers (binary protocol).
- mediasoup-client (WebRTC cho voice).
- react-virtuoso (virtualized list).
- framer-motion (animations).

### 13.4. Mobile
- React Native (Expo) cho iOS/Android.
- Native gesture.
- Push notification qua FCM/APNs.
- Offline queue + sync khi online.

---

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-CH-01 | 100K concurrent connections per node | Load test |
| AC-CH-02 | Send → Receive p99 < 50ms | Benchmark |
| AC-CH-03 | 50K requests → 1 DB query (singleflight) | Trace |
| AC-CH-04 | Message persisted < 100ms | p95 |
| AC-CH-05 | Reconnect < 1s | Network test |

---

**Tiếp theo:** [`docs/07-webrtc-sfu/README.md`](../07-webrtc-sfu/README.md) – WebRTC SFU + Recording.