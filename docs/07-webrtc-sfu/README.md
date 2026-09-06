# Phần 7 – WebRTC SFU + Recording (Meeting Engine)

> **Phân hệ:** Voice call, Video call, Screen share, Recording, AI tóm tắt.  
> **Mục tiêu:** Sub-50ms latency, Zero-Transcoding, GPU-accelerated recording, AI transcription.  
> **Đặc thù:** Rust/C++ SFU + AV1/VP9 SVC + eBPF/XDP routing + GPU composite recording.

---

## Mục lục
1. [Mục tiêu & Nguyên tắc](#1-mục-tiêu--nguyên-tắc)
2. [Kiến trúc SFU](#2-kiến-trúc-sfu)
3. [AV1/VP9 SVC Scalable Coding](#3-av1vp9-svc-scalable-coding)
4. [eBPF/XDP Routing](#4-ebpfxdp-routing)
5. [Meeting Orchestrator](#5-meeting-orchestrator)
6. [Recording Engine (GPU Accelerated)](#6-recording-engine-gpu-accelerated)
7. [AI Transcription & Summary](#7-ai-transcription--summary)
8. [Database Schema](#8-database-schema)
9. [Danh sách tính năng (≥ 100)](#9-danh-sách-tính-năng)
10. [API Surface](#10-api-surface)
11. [UI/UX](#11-uiux)

---

## 1. Mục tiêu & Nguyên tắc

### 1.1. Mục tiêu
| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| SFU-1 | Sub-50ms voice/video latency | p99 |
| SFU-2 | Zero-transcoding | SFU không decode video |
| SFU-3 | 200+ concurrent meetings per GPU node | Load test |
| SFU-4 | AI auto-summary sau meeting | < 30s |
| SFU-5 | Screen share 4K@60fps | Bandwidth adaptive |

### 1.2. Nguyên tắc
- **SFU thuần routing** – không mã hóa lại.
- **SVC chia layer** – client yếu mạng chỉ nhận layer thấp.
- **GPU composite** – record hàng trăm meeting không Headless Chromium.
- **AI chạy local** – Whisper.cpp + Llama-3, không gọi cloud.

---

## 2. Kiến trúc SFU

### 2.1. Sơ đồ
```
[Publisher (WebRTC)] ──AV1/VP9 SVC──► [Rust/C++ SFU Server Node]
                                          │
                          (eBPF/XDP Accelerated Routing)
                                          │
              ┌───────────────────────────┼───────────────────────────┐
              ▼                           ▼                           ▼
      [Subscriber High]           [Subscriber Med]            [Subscriber Low]
      (Full resolution)          (Temporal Layer 1)        (Spatial Layer 0)
```

### 2.2. Rust SFU Implementation (str0m-based)
```rust
use str0m::media::{Media, Stream};
use str0m::Rtc;

async fn handle_publisher(rtp_packet: RtpPacket) {
    // No decode – pure forwarding
    let subscribers = routing_table.get(rtp_packet.ssrc);
    for sub in subscribers {
        if sub.network_quality < 0.3 {
            // Strip spatial layer 2,3 (high res)
            send_filtered(rtp_packet, sub, &[0, 1]).await;
        } else {
            send_full(rtp_packet, sub).await;
        }
    }
}
```

### 2.3. Components
- **`sfu-node`** (Rust): Stateless SFU worker.
- **`meeting-orchestrator`** (Go): Quản lý meeting lifecycle.
- **`recorder`** (C++ + CUDA): GPU composite recording.
- **`stt-worker`** (Python + Whisper.cpp): Speech-to-text.
- **`meeting-summary`** (Python + Llama-3): Tóm tắt meeting.

### 2.4. ICE / TURN
- **ICE servers:** Coturn self-hosted.
- **STUN:** `stun:stun.rinco.app:3478`.
- **TURN:** `turn:turn.rinco.app:3478` (TCP + TLS).
- **TURN auth:** Short-lived PASETO token.

---

## 3. AV1/VP9 SVC Scalable Coding

### 3.1. Tại sao SVC thay Simulcast?
| | Simulcast | SVC |
|---|-----------|-----|
| **Băng thông** | Gửi N luồng (full + med + low) | 1 luồng có N layer |
| **CPU** | Encode N lần | Encode 1 lần |
| **Bitrate** | Tổng bitrate = full + med + low | ~ bằng full |
| **Latency** | Phải switch | Trong 1 frame |

### 3.2. Layer Structure
```
Layer 3 (Spatial): 1920x1080, 60fps
Layer 2 (Spatial): 1280x720, 30fps
Layer 1 (Spatial): 640x480, 15fps
Layer 0 (Spatial): 320x240, 15fps (cho audio-only client)

Temporal within each spatial:
  T0, T1, T2, T3 (frames at 60fps)
  Subscriber chọn từ frame nào trở đi
```

### 3.3. Client Config (Chrome)
```javascript
const transceiver = pc.addTransceiver(track, {
  direction: 'sendonly',
  sendEncodings: [
    { rid: 'h', maxBitrate: 4_000_000, scalabilityMode: 'L3T3' },
    { rid: 'm', maxBitrate: 1_000_000, scalabilityMode: 'L2T2', scaleResolutionDownBy: 2 },
    { rid: 'l', maxBitrate: 250_000, scalabilityMode: 'L1T1', scaleResolutionDownBy: 4 },
  ]
});
```

### 3.4. SFU Layer Filtering
```rust
fn should_forward_layer(rtp_packet: &RtpPacket, subscriber: &Subscriber) -> bool {
    let subscriber_max_layer = match subscriber.network {
        NetworkQuality::Excellent => 3,
        NetworkQuality::Good => 2,
        NetworkQuality::Fair => 1,
        NetworkQuality::Poor => 0,
    };
    rtp_packet.spatial_layer <= subscriber_max_layer
}
```

### 3.5. Codec Support
- **AV1:** Ưu tiên (tiết kiệm 30% bitrate so với VP9).
- **VP9 SVC:** Fallback.
- **VP8:** Fallback cũ.
- **H.264:** Nếu client không support AV1/VP9.

---

## 4. eBPF/XDP Routing

### 4.1. Vấn đề
- Mỗi SFU xử lý hàng trăm ngàn gói tin/s.
- User space routing chậm do syscall overhead.

### 4.2. Giải pháp: XDP/eBPF
```c
// eBPF/XDP program trong kernel
SEC("xdp")
int sfu_route(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;
    
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end) return XDP_PASS;
    
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end) return XDP_PASS;
    
    // Lookup routing table (BPF_MAP_TYPE_LRU_HASH)
    u32 key = ip->saddr + ip->daddr;
    u32 *ifindex = bpf_map_lookup_elem(&route_map, &key);
    if (ifindex) {
        return bpf_redirect(*ifindex, 0);
    }
    return XDP_PASS;
}
```

### 4.3. Routing Table
- Maintain trong BPF_MAP_TYPE_LRU_HASH.
- Update từ userspace khi subscriber join/leave.
- Auto-scale: ~100K entries.

### 4.4. Performance
- Routing decision: < 100ns.
- Bypass TCP/IP stack cho SRTP packets.
- Latency giảm ~5-15ms so với userspace.

---

## 5. Meeting Orchestrator

### 5.1. Meeting Lifecycle
```
[Scheduled] ─► [Starting] ─► [Active] ─► [Ending] ─► [Recording Processing] ─► [Completed]
                     │             │              │
                     │             │              └─► AI Summary, Save to MinIO
                     │             └─► Recording starts (if enabled)
                     └─► Wait for first participant
```

### 5.2. Room Types
| Type | Code | Use case |
|------|------|----------|
| **Instant** | 1 | Tạo nhanh, không schedule |
| **Scheduled** | 2 | Có thời gian + calendar invite |
| **Recurring** | 3 | Họp định kỳ (daily/weekly) |
| **Webinar** | 4 | 1-nhiều, host + panelists + attendees |
| **Breakout** | 5 | Chia nhỏ từ meeting lớn |

### 5.3. Orchestrator API (Go)
```go
type Meeting struct {
  ID            uuid.UUID
  TenantID      uuid.UUID
  RoomCode      string        // Short code: "abc-defg-hij"
  Type          MeetingType
  HostID        uuid.UUID
  Title         string
  StartsAt      time.Time
  EndsAt        time.Time
  MaxParticipants int
  Recording     RecordingConfig
  Settings      MeetingSettings
  Status        MeetingStatus
}

func CreateMeeting(m Meeting) (Meeting, error) {
    m.RoomCode = generateRoomCode()  // Easy to share
    m.ID = uuid.New()
    // Allocate SFU node
    sfuNode := sfuPicker.Pick(m.TenantID)
    m.SFUNodeID = sfuNode.ID
    db.Insert(m)
    return m, nil
}
```

### 5.4. SFU Picker
```python
def pick_sfu(tenant_id, expected_size):
    sfu_nodes = get_active_sfu_nodes()
    
    # Filter by tenant region
    candidates = [n for n in sfu_nodes if n.region == tenant.region]
    
    # Filter by capacity
    candidates = [n for n in candidates 
                  if n.current_load + expected_size <= n.max_capacity]
    
    # Score: prefer less loaded, prefer same datacenter
    return min(candidates, key=lambda n: n.load_ratio)
```

### 5.5. Recording Control
- `auto_record`: Record từ đầu.
- `on_demand`: Click "Record" trong meeting.
- `cloud_only`: Chỉ cloud, không lưu local.
- `cloud_and_local`: Cả hai.

---

## 6. Recording Engine (GPU Accelerated)

### 6.1. Kiến trúc
```
[WebRTC SFU Media Stream] ──(Direct RTP Pipe)──► [C++/Rust Egress Worker]
                                                       │
                                            (GPU Memory Mapping via CUDA)
                                                       │
                                              [NVIDIA NVENC Encoder]
                                                       │
                                            (Direct Chunked Stream)
                                                       │
                                                       ▼
                                          [MinIO / S3 Object Storage]
```

### 6.2. Egress Worker (C++)
```cpp
// Receive RTP packets from SFU
// Decode to GPU VRAM
cudaMemcpy(gpu_buffer, rtp_payload, size, cudaMemcpyHostToDevice);

// Composite layout on GPU
composite_kernel<<<>>>(gpu_buffer, layout, participant_count);

// Encode via NVENC
nv.Encode(gpu_buffer, &output);
upload_chunk_to_s3(output);
```

### 6.3. Layouts
- **Grid:** N x N ô vuông cho webcam (auto).
- **Speaker view:** 1 lớn + N nhỏ.
- **Presentation:** Screen share + thumbnails.
- **Custom layout:** Designer.

### 6.4. Performance
- 1 GPU NVIDIA L4 (60 TFLOPS, 24GB VRAM): 200+ concurrent recordings.
- So với Headless Chromium (1.5GB RAM + 100% 1 core mỗi meeting): 95% tiết kiệm.

### 6.5. Audio Mixing
- Server-side audio mixing bằng FFmpeg + GPU Audio API.
- Echo cancellation từ client side.

### 6.6. Storage
- MinIO/S3 chunked upload.
- Format: mp4 (H.264) hoặc webm (VP9) – chọn khi record.
- Resolution: 720p default, 1080p nếu GPU đủ.
- Retention: configurable (30/90/365 ngày).

### 6.7. Playback
- HLS streaming (adaptive bitrate).
- DASH optional.
- Download as MP4.
- Trim/cut (basic editor).

---

## 7. AI Transcription & Summary

### 7.1. Pipeline
```
[Meeting End] → [Audio Extract] → [Whisper.cpp STT] → [Llama-3 Summary] → [Save to CRM]
                                       │                     │
                                       ▼                     ▼
                                transcript.json        summary.md + action_items.md
```

### 7.2. Whisper.cpp Configuration
```cpp
whisper_context_params cparams = whisper_context_default_params();
cparams.use_gpu = true;
cparams.gpu_device = 0;  // CUDA device

whisper_full_params wparams = whisper_full_default_params(WHISPER_SAMPLING_GREEDY);
wparams.language = "auto";
wparams.translate = false;
wparams.print_realtime = true;  // For live caption
wparams.print_timestamps = true;
```

### 7.3. Live Caption (Real-time)
- STT chạy song song trong khi meeting.
- Sub-200ms latency (GPU).
- Gửi qua WebSocket tới participants.

### 7.4. Post-meeting Summary
```python
# Llama-3 70B Dynamic Quantization
prompt = f"""
Bạn là trợ lý AI tóm tắt cuộc họp. Dựa trên transcript sau, hãy:
1. Tóm tắt các điểm chính (5-7 bullet points).
2. Trích xuất Action Items (ai làm gì, deadline).
3. Highlight quyết định quan trọng.
4. Đề xuất follow-up.

Transcript:
{transcript_text}
"""

summary = llama3.generate(prompt, max_tokens=2000)
```

### 7.5. Storage trong CRM
- Lưu transcript vào `meeting_transcripts` table.
- Lưu summary + action items vào CRM Activity.
- Liên kết với Deal/Lead liên quan.

---

## 8. Database Schema

### 8.1. PostgreSQL
```sql
-- Meeting metadata
CREATE TABLE meetings (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  room_code TEXT UNIQUE NOT NULL,
  type SMALLINT NOT NULL,    -- 1:Instant, 2:Scheduled, ...
  host_id UUID NOT NULL,
  title TEXT NOT NULL,
  description TEXT,
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ,
  actual_started_at TIMESTAMPTZ,
  actual_ended_at TIMESTAMPTZ,
  max_participants INT DEFAULT 50,
  recording_enabled BOOLEAN DEFAULT false,
  recording_storage_url TEXT,
  transcript_url TEXT,
  summary TEXT,
  action_items JSONB,
  status TEXT CHECK (status IN ('SCHEDULED','ACTIVE','ENDED','CANCELLED')),
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Participants
CREATE TABLE meeting_participants (
  meeting_id UUID REFERENCES meetings(id),
  user_id UUID NOT NULL,
  joined_at TIMESTAMPTZ,
  left_at TIMESTAMPTZ,
  role TEXT,        -- 'host','co-host','panelist','attendee'
  PRIMARY KEY (meeting_id, user_id)
);

-- Chat in meeting
CREATE TABLE meeting_chats (
  id UUID PRIMARY KEY,
  meeting_id UUID REFERENCES meetings(id),
  sender_id UUID,
  text TEXT,
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Recordings
CREATE TABLE recordings (
  id UUID PRIMARY KEY,
  meeting_id UUID REFERENCES meetings(id),
  storage_url TEXT NOT NULL,
  format TEXT,                  -- 'mp4','webm'
  duration_seconds INT,
  size_bytes BIGINT,
  resolution TEXT,              -- '1280x720'
  status TEXT,                  -- 'recording','processing','ready','failed'
  started_at TIMESTAMPTZ,
  ended_at TIMESTAMPTZ
);

-- AI Transcripts
CREATE TABLE meeting_transcripts (
  id UUID PRIMARY KEY,
  meeting_id UUID REFERENCES meetings(id),
  language TEXT,
  segments JSONB,               -- [{start, end, speaker, text}, ...]
  full_text TEXT,
  model_version TEXT,
  created_at TIMESTAMPTZ DEFAULT now()
);
```

### 8.2. ScyllaDB (Real-time state)
```cql
-- Active meeting state
CREATE TABLE rinco_meet.active_meetings (
  meeting_id text PRIMARY KEY,
  tenant_id text,
  host_id text,
  participants list<frozen<participant>>,
  started_at bigint,
  layout text,
  recording boolean
);

-- SFU node registry
CREATE TABLE rinco_meet.sfu_nodes (
  node_id text PRIMARY KEY,
  region text,
  endpoint text,
  capacity int,
  current_load int,
  last_heartbeat bigint
);
```

### 8.3. Valkey (Hot state)
```
meeting:room:{room_code} → JSON{meeting_id, password?, host_id}
meeting:participants:{meeting_id} → Set of user_ids
meeting:screen_share:{meeting_id} → user_id (current sharer)
meeting:breakout:{meeting_id} → JSON{parent_id, rooms: [...]}
```

---

## 9. Danh sách tính tính năng (≥ 100)

### 9.1. Voice/Video Call cơ bản (1-30)
1. Voice call 1-1.
2. Video call 1-1.
3. Group voice call.
4. Group video call.
5. Auto-join từ chat (click avatar).
6. Mute/unmute mic.
7. Camera on/off.
8. Switch camera (front/back mobile).
9. Switch mic (mobile).
10. Speaker view.
11. Grid view.
12. Pin participant.
13. Spotlight (chỉ 1 người lớn).
14. Side-by-side.
15. Picture-in-picture.
16. Mini player khi rời tab.
17. Background blur.
18. Background virtual image.
19. Background virtual video.
20. Beauty filter.
21. Noise suppression.
22. Echo cancellation.
23. Auto gain control.
24. Audio device picker.
25. Video device picker.
26. Audio quality settings.
27. Video quality settings.
28. Low bandwidth mode.
29. Bandwidth indicator.
30. Network quality indicator.

### 9.2. Screen Share (31-50)
31. Share full screen.
32. Share window.
33. Share browser tab.
34. Share audio (with screen).
35. Annotation tools (draw, text, arrow).
36. Laser pointer.
37. Zoom in/out.
38. Pause/resume share.
39. Replace share.
40. Multi-presenter mode.
41. Remote control (request).
42. Privacy mode (blur khi share sensitive).
43. Highlight cursor.
44. Follow cursor.
45. Record screen share.
46. Stream screen to recording.
47. Chat side panel during share.
48. Q&A panel during share.
49. Reactions (raise hand, applause).
50. Pause share notification.

### 9.3. Meeting Management (51-75)
51. Schedule meeting.
52. Recurring meeting.
53. Calendar invite (.ics).
54. Google Calendar sync.
55. Outlook Calendar sync.
56. Join by link.
57. Join by room code.
58. Join by phone (PSTN).
59. Lobby/waiting room.
60. Auto admit from same tenant.
61. Knock to join.
62. Pre-join screen.
63. Test mic/camera pre-join.
64. Choose virtual background pre-join.
65. Rename before join.
66. Host controls (mute others).
67. Mute on entry.
68. Co-host assignment.
69. Transfer host.
70. End meeting for all.
71. Leave meeting.
72. Rejoin.
73. Lock meeting.
74. Password protection.
75. SSO enforcement.

### 9.4. Recording (76-95)
76. Auto record.
77. Manual record (host only).
78. Permission to record (config).
79. Record to cloud.
80. Record to local (download).
81. Pause/resume recording.
82. Stop recording.
83. Recording notification (visible).
84. Recording notification (hidden).
85. Recording transcript.
86. Live caption during meeting.
87. Caption translation.
88. Highlight transcription (in meeting).
89. Speaker labels in transcript.
90. Search transcript.
91. Play recording in-app.
92. Download recording.
93. Share recording (link).
94. Recording expiry.
95. Recording access control.

### 9.5. AI Features (96-115)
96. AI summary (auto).
97. AI action items extraction.
98. AI sentiment analysis.
99. AI topic detection.
100. AI speaker identification.
101. AI noise removal (Krisp).
102. AI translation live.
103. AI meeting notes (shared doc).
104. AI follow-up email.
105. AI task creation (CRM).
106. AI meeting score (engagement).
107. AI attendee focus score.
108. AI compliance check.
109. AI custom vocabulary.
110. AI profanity filter.
111. AI attendance log.
112. AI next meeting suggestion.
113. AI availability finder.
114. AI reschedule optimizer.
115. AI insights dashboard.

### 9.6. Advanced (116-130)
116. Webinar mode (1-nhiều).
117. Webinar registration.
118. Webinar Q&A.
119. Webinar poll.
120. Webinar raise hand.
121. Breakout rooms (random).
122. Breakout rooms (manual).
123. Breakout timer.
124. Broadcast to all rooms.
125. Whiteboard (collaborative).
126. Polls (live).
127. Quiz mode.
128. Live streaming (RTMP out).
129. External integration (Zoom, Meet).
130. Cross-tenant meeting (guest).

---

## 10. API Surface

### 10.1. REST
```
POST   /api/meet/v1/meetings
GET    /api/meet/v1/meetings
GET    /api/meet/v1/meetings/:id
PATCH  /api/meet/v1/meetings/:id
DELETE /api/meet/v1/meetings/:id
POST   /api/meet/v1/meetings/:id/start
POST   /api/meet/v1/meetings/:id/end
POST   /api/meet/v1/meetings/:id/record/start
POST   /api/meet/v1/meetings/:id/record/stop

GET    /api/meet/v1/meetings/:id/participants
POST   /api/meet/v1/meetings/:id/participants/:user_id/admit
DELETE /api/meet/v1/meetings/:id/participants/:user_id

POST   /api/meet/v1/join                       # Get token + SFU endpoint
GET    /api/meet/v1/meetings/:id/transcript
GET    /api/meet/v1/meetings/:id/summary
GET    /api/meet/v1/meetings/:id/action-items

GET    /api/meet/v1/recordings
GET    /api/meet/v1/recordings/:id
GET    /api/meet/v1/recordings/:id/playback     # HLS playlist
```

### 10.2. WebSocket (Signaling)
```
WS /ws/meet?token=...
   Frames:
   - join
   - offer
   - answer
   - ice_candidate
   - renegotiate
   - screen_share_start
   - screen_share_stop
   - mute/unmute
   - camera on/off
   - layout_change
   - recording_status
   - chat
   - raise_hand
   - reaction
```

---

## 11. UI/UX

### 11.1. Pre-join Screen
```
┌────────────────────────────────────────┐
│        [Camera Preview]                │
│                                        │
│  Name: [_______________]               │
│  Mic:  [Default ▼]                     │
│  Cam:  [Default ▼]                     │
│  Background: [None | Blur | Image]     │
│                                        │
│            [Join Meeting]              │
└────────────────────────────────────────┘
```

### 11.2. In-meeting Layout
```
┌────────────────────────────────────────┐
│  Top Bar: Title │ Time │ Participants  │
├────────────────────────────────────────┤
│                                        │
│   [Grid of participants]               │
│                                        │
│                       [Screen Share]   │
│                                        │
├────────────────────────────────────────┤
│  Bottom Bar: Mic | Cam | Share | Chat  │
│             Reactions | More | Leave    │
└────────────────────────────────────────┘
```

### 11.3. Components
- VideoTile (avatar fallback, audio level indicator).
- GridLayout (auto-arrange).
- SpeakerView.
- ControlBar.
- ChatPanel (with reactions).
- ParticipantsList (with status).
- Whiteboard.
- ScreenShareViewer.
- RecordingIndicator (visible).
- LiveCaptionOverlay.
- ReactionBar (raise hand, applause).
- PollsPanel.

### 11.4. Mobile (React Native)
- Portrait + Landscape.
- Native gesture (swipe to switch layout).
- Background mode (audio).
- Lock screen controls.

### 11.5. AI Summary View
```
┌────────────────────────────────────────┐
│  📝 Meeting Summary                    │
│  ─────────────────────────────────     │
│  Date: 06/09/2026                      │
│  Duration: 45 min                      │
│  Participants: 8                       │
│                                        │
│  Key Points:                           │
│  • Sales target Q3 increased 20%       │
│  • New pricing tier launched           │
│  • Marketing budget reallocated        │
│                                        │
│  ✅ Action Items:                      │
│  • [John] Update pricing page          │
│  • [Mary] Send email to customers      │
│  • [Team] Test new checkout            │
│                                        │
│  [Watch Recording] [Download]          │
└────────────────────────────────────────┘
```

---

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-SFU-01 | Voice p99 < 50ms | RTT measure |
| AC-SFU-02 | Video p99 < 100ms | RTT measure |
| AC-SFU-03 | 200 meetings per GPU node | Load test |
| AC-SFU-04 | AI summary < 30s sau meeting | E2E test |
| AC-SFU-05 | Screen share 4K60 stable | Bandwidth test |
| AC-SFU-06 | Reconnect on network drop < 3s | TURN test |

---

**Tiếp theo:** [`docs/08-observability/README.md`](../08-observability/README.md) – Observability 4 tầng + AI SRE.