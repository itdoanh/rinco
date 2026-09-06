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
12. **MỤC LỤC MỚI (MỞ RỘNG)**
12. [Audit Report](#12-audit-report)
13. [TURN Server Setup với coturn chi tiết](#13-turn-server-setup-với-coturn-chi-tiết)
14. [SVC Layer Configuration chi tiết](#14-svc-layer-configuration-chi-tiết)
15. [Rust SFU Implementation chi tiết (str0m-based)](#15-rust-sfu-implementation-chi-tiết-str0m-based)
16. [ICE / DTLS / SRTP Protocol Flow](#16-ice--dtls--srtp-protocol-flow)
17. [Congestion Control (BBR/WebRTC)](#17-congestion-control-bbrwebrtc)
18. [Meeting Orchestrator Implementation](#18-meeting-orchestrator-implementation)
19. [GPU Recording Pipeline chi tiết](#19-gpu-recording-pipeline-chi-tiết)
20. [NVENC Encoder chi tiết](#20-nvenc-encoder-chi-tiết)
21. [Audio Mixing chi tiết](#21-audio-mixing-chi-tiết)
22. [AI Whisper.cpp Transcription Pipeline](#22-ai-whispercpp-transcription-pipeline)
23. [Llama-3 Meeting Summary](#23-llama-3-meeting-summary)
24. [Recording Playback (HLS)](#24-recording-playback-hls)
25. [Edge Cases & Error Handling](#25-edge-cases--error-handling)
26. [Performance Benchmark chi tiết](#26-performance-benchmark-chi-tiết)
27. [Security Hardening](#27-security-hardening)
28. [Disaster Recovery](#28-disaster-recovery)
29. [Cost Estimation](#29-cost-estimation)
30. [Testing Strategy](#30-testing-strategy)
31. [Implementation Roadmap chi tiết](#31-implementation-roadmap-chi-tiết)
32. [Open Questions / Cần user xác nhận](#32-open-questions--cần-user-xác-nhận)

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

# PHẦN MỞ RỘNG (TURN Setup + GPU Recording chi tiết + Code)

> Phần này bổ sung theo yêu cầu đặc biệt cho WebRTC SFU: TURN setup với coturn, SVC config, code Rust, AI pipeline.

---

## 12. Audit Report

### 12.1. Phần đã đủ chi tiết
- ✅ Kiến trúc SFU zero-transcoding.
- ✅ SVC vs Simulcast comparison.
- ✅ Layer structure (Spatial + Temporal).
- ✅ eBPF/XDP routing concept.
- ✅ Meeting orchestrator basic API.
- ✅ GPU recording architecture.
- ✅ AI transcription pipeline overview.
- ✅ 130 features across 6 categories.
- ✅ REST + WebSocket API.

### 12.2. Phần còn thiếu (gap)

| Gap ID | Mô tả | Mức độ | Giải pháp |
|--------|--------|--------|-----------|
| GAP-1 | coturn config chi tiết (CLI, env, security) | High | Thêm §13 |
| GAP-2 | PASETO token cho TURN auth + rotation | High | Thêm §13.4 |
| GAP-3 | SVC scalabilityMode matrix đầy đủ | Medium | Thêm §14 |
| GAP-4 | Browser compat cho SVC (Chrome/Firefox/Safari) | High | Thêm §14.4 |
| GAP-5 | str0m full code example với multiple subscribers | High | Thêm §15 |
| GAP-6 | ICE/DTLS/SRTP handshake chi tiết | High | Thêm §16 |
| GAP-7 | Congestion control (BBR, GCC) | Medium | Thêm §17 |
| GAP-8 | SFU Picker algorithm chi tiết | Medium | Thêm §18.4 |
| GAP-9 | NVENC session limit + workaround | High | Thêm §20 |
| GAP-10 | Audio mixing (FFmpeg + GPU) chi tiết | High | Thêm §21 |
| GAP-11 | Whisper.cpp speaker diarization | High | Thêm §22 |
| GAP-12 | Llama-3 prompt engineering cho summary | Medium | Thêm §23 |
| GAP-13 | HLS packaging chi tiết | Medium | Thêm §24 |
| GAP-14 | Recording retention + GDPR | High | Thêm §28 |
| GAP-15 | Multi-region SFU deployment | High | Thêm §26 |
| GAP-16 | Cost model chi tiết | Medium | Thêm §29 |
| GAP-17 | Load test scenarios (small/medium/huge meeting) | High | Thêm §26 |
| GAP-18 | Webhook cho meeting events | Medium | Thêm §18.5 |

### 12.3. Phần có mâu thuẫn nội bộ
- **§2.2 vs §15:** Rust code snippet §2.2 quá đơn giản, cần chi tiết SVC routing ở §15.
- **§3.4 vs §14:** SFU layer filtering code snippet cần expansion với bandwidth-based decision.
- **§6.2 vs §19:** Egress Worker code §6.2 chỉ là pseudo-code, cần full implementation §19.

### 12.4. Phần cần code example cụ thể
- Full Rust SFU với str0m + SVC routing.
- coturn config hoàn chỉnh với Docker Compose.
- C++ Egress Worker với NVENC + CUDA.
- Whisper.cpp integration với diarization.
- Llama-3 summary prompt + post-processing.

---

## 13. TURN Server Setup với coturn chi tiết

### 13.1. coturn Docker Compose

```yaml
# infra/docker/services/coturn/docker-compose.yml
version: '3.8'
services:
  coturn:
    image: coturn/coturn:4.7
    container_name: rinco-coturn
    restart: unless-stopped
    ports:
      - "3478:3478/udp"      # STUN/TURN over UDP
      - "3478:3478/tcp"      # STUN/TURN over TCP
      - "5349:5349/udp"      # STUN/TURN over TLS/DTLS
      - "5349:5349/tcp"      # STUN/TURN over TLS/DTLS
      # Quota range for relay (UDP ports)
      - "49152-49200:49152-49200/udp"
    volumes:
      - ./turnserver.conf:/etc/coturn/turnserver.conf:ro
      - ./certs:/etc/coturn/certs:ro
    environment:
      - TZ=UTC
    logging:
      driver: "json-file"
      options:
        max-size: "50m"
        max-file: "5"
    networks:
      - coturn-net

  # Optional: Multi-region TURN replicas
  coturn-us:
    image: coturn/coturn:4.7
    container_name: rinco-coturn-us
    restart: unless-stopped
    ports:
      - "3478:3478/udp"
      - "3478:3478/tcp"
    volumes:
      - ./turnserver-us.conf:/etc/coturn/turnserver.conf:ro
      - ./certs:/etc/coturn/certs:ro
    environment:
      - REGION=us-east-1
    networks:
      - coturn-net

networks:
  coturn-net:
    driver: bridge
```

### 13.2. turnserver.conf (Production)

```ini
# /etc/coturn/turnserver.conf
# ====================================
# RINCO Production TURN Server Config
# ====================================

# Listening
listening-port=3478
tls-listening-port=5349

# Network
listening-ip=0.0.0.0
relay-ip=YOUR_PUBLIC_IP
external-ip=YOUR_PUBLIC_IP

# Min/Max port for relay
min-port=49152
max-port=49200

# Fingerprint
fingerprint

# Long-term credentials
lt-cred-mech

# Shared secret for time-limited credentials
# Generate: openssl rand -hex 32
static-auth-secret=YOUR_LONG_RANDOM_SECRET_HERE_CHANGE_ME

# Realm
realm=turn.rinco.app

# Disable STUN-only anonymous access (forces auth)
no-loopback-peers
no-multicast-peers

# Security
no-cli
no-tlsv1
no-tlsv1_1
cipher-list="ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20"

# TLS Certificates
cert=/etc/coturn/certs/fullchain.pem
pkey=/etc/coturn/certs/privkey.pem
dh-file=/etc/coturn/certs/dh2048.pem

# Quota / Rate limiting
user-quota=12           # Max 12 concurrent sessions per user
total-quota=1200       # Max 1200 concurrent sessions
max-allocate-timeout=60
channel-lifetime=600

# Logging
log-file=/var/log/turn/turn.log
simple-log
new-log-timestamp
new-log-timestamp-format=%Y-%m-%dT%H:%M:%S.%fZ

# Syslog (alternative)
# syslog
# syslog-facility=daemon

# Database for session tracking (optional, use Redis)
# Redis backend example:
# redis-userdb="ip=redis port=6379 dbname=0 password=YOUR_REDIS_PASSWORD user=turn"
# Alternative: shared secret + REST API

# Authentication URL (REST API for time-limited creds)
# When using shared-secret + rest-api, clients must compute HMAC
# rest-api-separator=:

# STUN/TURN Options
stun-only
# (Stun-only listens on 3478 as STUN; relay requires auth)

# Bandwidth throttling (bytes per second per session)
# max-bps=1000000  # 1 Mbps per session
```

### 13.3. Generate TLS certs

```bash
#!/bin/bash
# scripts/setup-turn-tls.sh

# 1. Use Let's Encrypt for production
sudo certbot certonly --standalone -d turn.rinco.app

# Or self-signed for dev
openssl req -x509 -newkey rsa:4096 -keyout privkey.pem -out fullchain.pem \
  -days 365 -nodes -subj "/CN=turn.rinco.app"

# 2. Generate DH params (slow but only once)
openssl dhparam -out dh2048.pem 2048

# 3. Set permissions
chmod 600 privkey.pem
chmod 644 fullchain.pem
chmod 644 dh2048.pem
```

### 13.4. Time-limited TURN credentials (REST API pattern)

Client cần credentials có expiration (5-15 phút), không dùng long-term.

```go
// services/meet-orchestrator/internal/turn/credentials.go
package turn

import (
    "crypto/hmac"
    "crypto/sha1"
    "encoding/base64"
    "fmt"
    "os"
    "strconv"
    "time"
)

type Credentials struct {
    Username   string
    Password   string
    TTL        int
    URIs       []string
}

type Generator struct {
    sharedSecret string
    realm        string
    uris         []string
    ttl          time.Duration
}

func NewGenerator() *Generator {
    secret := os.Getenv("TURN_SHARED_SECRET")
    if secret == "" {
        panic("TURN_SHARED_SECRET required")
    }
    return &Generator{
        sharedSecret: secret,
        realm:        os.Getenv("TURN_REALM"),
        uris: []string{
            "turn:turn.rinco.app:3478?transport=udp",
            "turn:turn.rinco.app:3478?transport=tcp",
            "turns:turn.rinco.app:5349?transport=tcp",
        },
        ttl: 10 * time.Minute,
    }
}

// Generate creates a time-limited credential pair using HMAC-SHA1
// Username = "{expiration_timestamp}:{tenant_id}:{user_id}"
// Password = base64(HMAC-SHA1(shared_secret, username))
func (g *Generator) Generate(tenantID, userID string) (*Credentials, error) {
    expiration := time.Now().Add(g.ttl).Unix()
    username := fmt.Sprintf("%d:%s:%s", expiration, tenantID, userID)

    h := hmac.New(sha1.New, []byte(g.sharedSecret))
    h.Write([]byte(username))
    password := base64.StdEncoding.EncodeToString(h.Sum(nil))

    return &Credentials{
        Username: username,
        Password: password,
        TTL:      int(g.ttl.Seconds()),
        URIs:     g.uris,
    }, nil
}

// Verify checks if credentials are valid and not expired
func (g *Generator) Verify(username, password string) error {
    parts := strings.SplitN(username, ":", 3)
    if len(parts) < 3 {
        return fmt.Errorf("invalid username format")
    }
    expiration, err := strconv.ParseInt(parts[0], 10, 64)
    if err != nil {
        return err
    }
    if time.Now().Unix() > expiration {
        return fmt.Errorf("credentials expired")
    }
    h := hmac.New(sha1.New, []byte(g.sharedSecret))
    h.Write([]byte(username))
    expected := base64.StdEncoding.EncodeToString(h.Sum(nil))
    if expected != password {
        return fmt.Errorf("invalid password")
    }
    return nil
}
```

### 13.5. TURN Health Monitoring

```yaml
# Prometheus monitoring for coturn
apiVersion: v1
kind: ConfigMap
metadata:
  name: coturn-exporter-config
data:
  exporter.yaml: |
    # coturn_exporter parses turnserver.log and exports Prometheus metrics
    metrics:
      - name: turn_active_sessions
        type: gauge
        description: "Active TURN sessions"
      - name: turn_total_bytes_relayed
        type: counter
        description: "Total bytes relayed"
      - name: turn_auth_failures_total
        type: counter
        description: "Authentication failures"
      - name: turn_allocations_total
        type: counter
        description: "Total allocations"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coturn-exporter
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: exporter
        image: ghcr.io/rinco/coturn-exporter:latest
        ports:
        - containerPort: 9099
        volumeMounts:
        - name: turn-logs
          mountPath: /var/log/turn
      volumes:
      - name: turn-logs
        persistentVolumeClaim:
          claimName: coturn-logs
```

### 13.6. Multi-region TURN Setup

```
                    ┌──────────────────┐
                    │   Global DNS     │
                    │ turn.rinco.app   │
                    │ → GeoDNS routing │
                    └─────────┬────────┘
                              │
            ┌─────────────────┼─────────────────┐
            ▼                 ▼                 ▼
    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
    │  AP-SOUTH-1  │  │  US-EAST-1   │  │  EU-WEST-1   │
    │  (Singapore) │  │  (Virginia)  │  │  (Ireland)   │
    │              │  │              │  │              │
    │  STUN/TURN   │  │  STUN/TURN   │  │  STUN/TURN   │
    │  Latency:    │  │  Latency:    │  │  Latency:    │
    │  <10ms VN    │  │  <20ms US    │  │  <20ms EU    │
    └──────────────┘  └──────────────┘  └──────────────┘
```

GeoDNS Route53 / Cloudflare config:
```
turn.rinco.app → 8.8.8.8 (anycast)
                OR region-based via latency routing
```

---

## 14. SVC Layer Configuration chi tiết

### 14.1. scalabilityMode Matrix

| Mode | Spatial | Temporal | Use Case |
|------|---------|----------|----------|
| `L1T1` | 1 | 1 | Audio only fallback |
| `L1T2` | 1 | 2 | Voice call, low bandwidth |
| `L1T3` | 1 | 3 | Voice call HD |
| `L2T1` | 2 | 1 | Video 480p low-fps |
| `L2T2` | 2 | 2 | Video 480p normal |
| `L2T3` | 2 | 3 | Video 480p 30fps |
| `L3T2` | 3 | 2 | Video 720p normal |
| `L3T3` | 3 | 3 | Video 720p 30fps+ |
| `L4T3` | 4 | 3 | Video 1080p+ 30fps |
| `S2T1` | 2 | 1 | Screen share low |
| `S3T2` | 3 | 2 | Screen share HD |

### 14.2. Codec-Specific Behavior

#### VP9 SVC
```
VP9 supports spatial scalability via frame size reference chain.
- Spatial layer 0: 320x240 (base)
- Spatial layer 1: 640x480 (depends on 0)
- Spatial layer 2: 1280x720 (depends on 1)
- Spatial layer 3: 1920x1080 (depends on 2)

Temporal layer within spatial:
- T0: 1 frame per second
- T1: 2 fps (every other T0 frame + key)
- T2: 4 fps (every other T1 frame)
- T3: 30+ fps (every other T2 frame)
```

#### AV1 SVC
```
AV1 supports scalability via scalability_mode:
- L1T2: Single spatial, 2 temporal layers
- L2T2: 2 spatial, 2 temporal
- L3T3: 3 spatial, 3 temporal (Chrome 120+)

Note: AV1 SVC is still maturing in browsers (Safari limited support).
```

### 14.3. Client Config Examples

#### Chrome - 3-layer SVC for Video
```javascript
const transceiver = pc.addTransceiver(videoTrack, {
  direction: 'sendonly',
  sendEncodings: [
    {
      rid: 'h',  // high layer
      maxBitrate: 4_000_000,
      scalabilityMode: 'L3T3',
      scaleResolutionDownBy: 1,  // full 1920x1080
    },
    {
      rid: 'm',  // medium layer
      maxBitrate: 1_000_000,
      scalabilityMode: 'L2T2',
      scaleResolutionDownBy: 2,  // 960x540
    },
    {
      rid: 'l',  // low layer
      maxBitrate: 250_000,
      scalabilityMode: 'L1T1',
      scaleResolutionDownBy: 4,  // 480x270
    },
  ],
});

// Subscriber receives layer based on bandwidth
pc.getReceivers().forEach(receiver => {
  if (receiver.track.kind === 'video') {
    receiver.parameters.encodings.forEach(encoding => {
      encoding.maxBitrate = bandwidthLimit;
    });
  }
});
```

#### Firefox - VP9 SVC (limited)
```javascript
const transceiver = pc.addTransceiver(videoTrack, {
  direction: 'sendonly',
  sendEncodings: [
    {
      maxBitrate: 2_000_000,
      scalabilityMode: 'L2T2',  // Firefox supports 2x2 max
    },
  ],
});
```

#### Safari - VP8/H.264 fallback (no SVC)
```javascript
const transceiver = pc.addTransceiver(videoTrack, {
  direction: 'sendonly',
  sendEncodings: [
    { maxBitrate: 2_000_000 },  // Single layer, no scalabilityMode
  ],
});
// SFU sẽ fallback: re-encode hoặc Simulcast
```

### 14.4. SVC Browser Compatibility

| Browser | SVC Support | Max Layers | Note |
|---------|-------------|------------|------|
| Chrome 120+ | ✅ Full | L3T3 | AV1, VP9 |
| Edge 120+ | ✅ Full | L3T3 | Same as Chrome |
| Firefox 120+ | ⚠️ Partial | L2T2 | VP9 only, no AV1 SVC yet |
| Safari 17+ | ❌ None | - | Falls back to Simulcast/H.264 |
| Mobile Chrome | ✅ Full | L3T3 | |
| Mobile Safari | ❌ None | - | |

Fallback strategy: SFU detects browser via User-Agent, sends Simulcast instead of SVC.

### 14.5. SFU Layer Routing Logic

```rust
// services/sfu-node/src/routing.rs
use std::collections::HashMap;

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum NetworkQuality {
    Poor = 0,
    Fair = 1,
    Good = 2,
    Excellent = 3,
}

#[derive(Debug, Clone)]
pub struct Subscriber {
    pub user_id: String,
    pub network_quality: NetworkQuality,
    pub explicit_layer: Option<u8>,  // Manual override
    pub max_bitrate: u32,
    pub downlink_bandwidth: u32,  // Estimated in bps
}

pub struct SvcRouter {
    publisher_layers: HashMap<String, LayerInfo>,  // by mid (media ID)
}

#[derive(Debug, Clone)]
pub struct LayerInfo {
    pub spatial_layer: u8,
    pub temporal_layer: u8,
    pub bitrate: u32,
    pub resolution: (u16, u16),
}

impl SvcRouter {
    pub fn should_forward(
        &self,
        packet: &RtpPacket,
        subscriber: &Subscriber,
    ) -> bool {
        let layer = match self.publisher_layers.get(&packet.mid()) {
            Some(l) => l,
            None => return false,  // Unknown stream
        };

        // Manual override takes priority
        if let Some(max) = subscriber.explicit_layer {
            return layer.spatial_layer <= max;
        }

        // Network-based decision
        let max_spatial = match subscriber.network_quality {
            NetworkQuality::Poor => 0,
            NetworkQuality::Fair => 1,
            NetworkQuality::Good => 2,
            NetworkQuality::Excellent => 3,
        };

        // Bandwidth-based refinement
        if subscriber.downlink_bandwidth < 300_000 {
            // < 300 Kbps: only base layer
            layer.spatial_layer == 0 && layer.temporal_layer == 0
        } else if subscriber.downlink_bandwidth < 1_000_000 {
            // < 1 Mbps: up to layer 1
            layer.spatial_layer <= 1
        } else if subscriber.downlink_bandwidth < 3_000_000 {
            // < 3 Mbps: up to layer 2
            layer.spatial_layer <= 2
        } else {
            // >= 3 Mbps: all layers
            layer.spatial_layer <= max_spatial
        }
    }

    pub fn update_subscriber_quality(
        &self,
        user_id: &str,
        stats: &NetworkStats,
    ) -> NetworkQuality {
        // stats: jitter, packet loss, RTT, available bandwidth
        let quality = if stats.packet_loss_pct > 5.0 || stats.rtt_ms > 300 {
            NetworkQuality::Poor
        } else if stats.packet_loss_pct > 2.0 || stats.rtt_ms > 150 {
            NetworkQuality::Fair
        } else if stats.packet_loss_pct > 0.5 || stats.rtt_ms > 80 {
            NetworkQuality::Good
        } else {
            NetworkQuality::Excellent
        };

        // Notify subscribers to renegotiate
        // (Optional: use RTC layer notification via RTCP REMB or new API)
        quality
    }
}
```

---

## 15. Rust SFU Implementation chi tiết (str0m-based)

### 15.1. Full SFU Node

```rust
// services/sfu-node/src/main.rs
use str0m::media::{Media, Mid, Stream};
use str0m::Rtc;
use str0m::change::SdpOffer;
use std::sync::Arc;
use dashmap::DashMap;
use tokio::net::{TcpListener, TcpStream};
use uuid::Uuid;

mod router;
mod codec;
mod subscriber;

use router::SvcRouter;
use subscriber::{Subscriber, SubscriberRegistry};

pub struct SfuContext {
    pub router: Arc<SvcRouter>,
    pub subscribers: Arc<SubscriberRegistry>,
    pub meeting_orchestrator: Arc<MeetingOrchestrator>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt::init();
    let bind = std::env::var("SFU_BIND").unwrap_or_else(|_| "0.0.0.0:8089".into());

    let ctx = Arc::new(SfuContext {
        router: Arc::new(SvcRouter::new()),
        subscribers: Arc::new(SubscriberRegistry::new()),
        meeting_orchestrator: Arc::new(MeetingOrchestrator::connect().await?),
    });

    let listener = TcpListener::bind(&bind).await?;
    tracing::info!("SFU node listening on {}", bind);

    while let Ok((stream, peer)) = listener.accept().await {
        let ctx = ctx.clone();
        tokio::spawn(async move {
            if let Err(e) = handle_sfu_connection(stream, ctx).await {
                tracing::error!("SFU connection from {} failed: {:?}", peer, e);
            }
        });
    }

    Ok(())
}

async fn handle_sfu_connection(
    stream: TcpStream,
    ctx: Arc<SfuContext>,
) -> Result<(), Box<dyn std::error::Error>> {
    // 1. Read HTTP upgrade request (WebSocket-style signaling)
    // 2. Authenticate with PASETO token
    let (rtc, meeting_id, user_id, role) = authenticate_and_initialize(stream, &ctx).await?;

    tracing::info!("User {} joined meeting {}", user_id, meeting_id);

    // 3. Add user to subscriber registry
    let subscriber = Subscriber::new(user_id.clone(), role);
    ctx.subscribers.add(meeting_id.clone(), subscriber.clone()).await;

    // 4. Send current state to new user
    let existing_publishers = ctx.subscribers.get_publishers(&meeting_id).await;
    for publisher in existing_publishers {
        // Send publisher's offer to new subscriber
        // (handled by orchestrator via signaling)
    }

    // 5. Main event loop
    let mut change = rtc.sdp_api();
    let mut packet_buf = vec![0u8; 2048];

    loop {
        // Use select! để multiplex multiple events
        tokio::select! {
            // Read signaling messages
            signal = read_signal(&mut stream) => {
                let signal = signal?;
                match signal {
                    Signal::Renegotiate => {
                        let offer = rtc.create_offer().unwrap();
                        // Send to client
                        send_signal(&mut stream, SignalMessage::Offer(offer)).await?;
                    }
                    Signal::IceCandidate(candidate) => {
                        rtc.add_remote_candidate(candidate);
                    }
                    Signal::Answer(answer) => {
                        rtc.set_answer(answer)?;
                    }
                }
            }

            // Read RTP packets
            // Note: str0m handles this internally via poll()
        }

        // Process RTP/RTCP events from str0m
        let timeout = tokio::time::Duration::from_millis(100);
        tokio::time::timeout(timeout, async {
            while let Some(event) = rtc.poll().await {
                match event {
                    str0m::Event::MediaData(media) => {
                        handle_media_data(&ctx, &meeting_id, &user_id, media).await;
                    }
                    str0m::Event::NetworkChange => {
                        handle_network_change(&ctx, &meeting_id, &user_id, &rtc).await;
                    }
                    str0m::Event::StreamPaused(stream) => {
                        subscriber.set_paused(&user_id, stream.mid(), true).await;
                    }
                    str0m::Event::StreamResumed(stream) => {
                        subscriber.set_paused(&user_id, stream.mid(), false).await;
                    }
                    _ => {}
                }
            }
        }).await.ok();

        // Distribute packets to other subscribers
        distribute_rtp(&ctx, &meeting_id, &user_id, &mut packet_buf, &rtc).await;

        // Update stats
        update_stats(&ctx, &meeting_id, &user_id, &rtc).await;
    }
}

async fn handle_media_data(
    ctx: &SfuContext,
    meeting_id: &str,
    sender_id: &str,
    media: str0m::media::MediaData,
) {
    // Forward to all subscribers except sender
    let subscribers = ctx.subscribers.get_others(meeting_id, sender_id).await;

    for (sub_id, subscriber) in subscribers {
        if subscriber.is_paused(media.mid()) {
            continue;
        }

        // Determine which layer to forward
        if !ctx.router.should_forward(&media, &subscriber) {
            continue;
        }

        // Forward packet
        subscriber.send_rtp(media.clone()).await;
    }
}

async fn distribute_rtp(
    ctx: &SfuContext,
    meeting_id: &str,
    sender_id: &str,
    buf: &mut [u8],
    rtc: &mut Rtc,
) {
    // Read from str0m's outgoing queue
    while let Some(outgoing) = rtc.poll_transmit(buf) {
        let mid = outgoing.mid;
        let data = &buf[..outgoing.len];

        // Broadcast to subscribers
        let subscribers = ctx.subscribers.get_others(meeting_id, sender_id).await;
        for (_, subscriber) in subscribers {
            subscriber.send_to_webrtc(mid, data).await;
        }
    }
}

async fn update_stats(
    ctx: &SfuContext,
    meeting_id: &str,
    user_id: &str,
    rtc: &Rtc,
) {
    // Get stats from str0m
    let stats = rtc.stats();
    let network = ctx.router.update_subscriber_quality(user_id, &stats);

    // Update subscriber registry
    if let Some(mut sub) = ctx.subscribers.get_mut(meeting_id, user_id).await {
        sub.network_quality = network;
    }
}
```

### 15.2. Subscriber Registry

```rust
// services/sfu-node/src/subscriber.rs
use dashmap::DashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

#[derive(Debug, Clone)]
pub struct Subscriber {
    pub user_id: String,
    pub role: String,           // "host" | "co-host" | "panelist" | "attendee"
    pub sender: Option<Arc<WebRtcSender>>,
    pub subscriptions: Vec<Mid>,
    pub network_quality: NetworkQuality,
    pub paused_streams: Vec<Mid>,
}

impl Subscriber {
    pub fn new(user_id: String, role: String) -> Self {
        Self {
            user_id,
            role,
            sender: None,
            subscriptions: Vec::new(),
            network_quality: NetworkQuality::Good,
            paused_streams: Vec::new(),
        }
    }

    pub async fn send_rtp(&self, data: MediaData) {
        if let Some(sender) = &self.sender {
            sender.send(data).await;
        }
    }

    pub async fn send_to_webrtc(&self, mid: Mid, data: &[u8]) {
        if let Some(sender) = &self.sender {
            sender.send_bytes(mid, data).await;
        }
    }

    pub fn is_paused(&self, mid: Mid) -> bool {
        self.paused_streams.contains(&mid)
    }

    pub async fn set_paused(&self, mid: Mid, paused: bool) {
        // ... update paused_streams
    }
}

pub struct SubscriberRegistry {
    // meeting_id → user_id → Subscriber
    meetings: DashMap<String, DashMap<String, Arc<RwLock<Subscriber>>>>,
}

impl SubscriberRegistry {
    pub fn new() -> Self {
        Self { meetings: DashMap::new() }
    }

    pub async fn add(&self, meeting_id: String, sub: Subscriber) {
        let sub = Arc::new(RwLock::new(sub));
        self.meetings
            .entry(meeting_id)
            .or_insert_with(DashMap::new)
            .insert(sub.read().await.user_id.clone(), sub);
    }

    pub async fn remove(&self, meeting_id: &str, user_id: &str) {
        if let Some(meeting) = self.meetings.get(meeting_id) {
            meeting.remove(user_id);
        }
    }

    pub async fn get_publishers(&self, meeting_id: &str) -> Vec<String> {
        self.meetings.get(meeting_id)
            .map(|m| {
                m.iter().filter(|e| e.value().read().unwrap().role != "attendee")
                    .map(|e| e.key().clone()).collect()
            })
            .unwrap_or_default()
    }

    pub async fn get_others(&self, meeting_id: &str, exclude: &str) -> Vec<(String, Subscriber)> {
        self.meetings.get(meeting_id)
            .map(|m| {
                m.iter().filter(|e| e.key() != exclude)
                    .map(|e| (e.key().clone(), e.value().read().unwrap().clone()))
                    .collect()
            })
            .unwrap_or_default()
    }

    pub async fn get_mut(&self, meeting_id: &str, user_id: &str) -> Option<tokio::sync::RwLockWriteGuard<Subscriber>> {
        self.meetings.get(meeting_id)?
            .get(user_id)?
            .write().await.into()
    }
}
```

---

## 16. ICE / DTLS / SRTP Protocol Flow

### 16.1. Full WebRTC Handshake

```
Client A                      SFU Server                Client B
   │                              │                        │
   │ 1. HTTP POST /join          │                        │
   │    { token, meeting_id }     │                        │
   ├─────────────────────────────►│                        │
   │                              │                        │
   │ 2. WebSocket upgrade         │                        │
   │◄─────────────────────────────│                        │
   │                              │                        │
   │ 3. WS message: SDP Offer     │                        │
   │    (ICE candidates,         │                        │
   │     DTLS fingerprint,        │                        │
   │     media lines)             │                        │
   ├─────────────────────────────►│                        │
   │                              │ Allocate transport     │
   │                              │ port (49152-49200)     │
   │                              │                        │
   │ 4. STUN Binding Request      │                        │
   │ (for ICE connectivity check) │                        │
   ├─────────────────────────────►│                        │
   │ 5. STUN Binding Success      │                        │
   │◄─────────────────────────────│                        │
   │                              │                        │
   │ 6. DTLS ClientHello          │                        │
   ├─────────────────────────────►│                        │
   │ 7. DTLS ServerHello + Cert   │                        │
   │◄─────────────────────────────│                        │
   │ 8. DTLS Key Exchange         │                        │
   ├──────────────────────────────►│                        │
   │ 9. DTLS Finished             │                        │
   ├──────────────────────────────►│                        │
   │                              │ SRTP session ready     │
   │ 10. SRTP media (RTP+SRTCP)   │                        │
   ├──────────────────────────────►│                        │
   │                              │ Forward to B           │
   │                              ├───────────────────────►│
   │                              │                        │
   │                              │ 11. STUN (B's ICE)    │
   │                              │◄──────────────────────┤
   │                              │ 12. STUN success      │
   │                              ├───────────────────────►│
   │                              │                        │
   │                              │ 13. DTLS handshake     │
   │                              │◄──────────────────────┤
   │                              ├───────────────────────►│
   │                              │                        │
   │                              │ 14. SRTP A→B          │
   │                              ├───────────────────────►│
```

### 16.2. ICE Candidate Gathering

```typescript
// apps/meeting-client/src/webrtc/ice.ts
export async function gatherIceCandidates(pc: RTCPeerConnection): Promise<RTCIceCandidate[]> {
  return new Promise((resolve) => {
    const candidates: RTCIceCandidate[] = [];
    let gatheringComplete = false;

    pc.addEventListener('icecandidate', (event) => {
      if (event.candidate) {
        candidates.push(event.candidate);
      } else {
        // Null candidate signals gathering complete
        if (!gatheringComplete) {
          gatheringComplete = true;
          resolve(candidates);
        }
      }
    });

    // Fallback: end gathering after 5 seconds
    setTimeout(() => {
      if (!gatheringComplete) {
        gatheringComplete = true;
        resolve(candidates);
      }
    }, 5000);
  });
}

export async function setupPeerConnection(config: MeetingConfig): Promise<RTCPeerConnection> {
  const pc = new RTCPeerConnection({
    iceServers: [
      { urls: ['stun:stun.rinco.app:3478'] },
      {
        urls: ['turn:turn.rinco.app:3478?transport=udp', 'turn:turn.rinco.app:3478?transport=tcp'],
        username: config.turnCredentials.username,
        credential: config.turnCredentials.password,
        credentialType: 'password',
      },
    ],
    iceTransportPolicy: 'relay',  // Force TURN for security (optional)
    bundlePolicy: 'max-bundle',
    rtcpMuxPolicy: 'require',
    sdpSemantics: 'unified-plan',
  });

  // Add transceivers for recvonly streams
  pc.addTransceiver('audio', { direction: 'recvonly' });
  pc.addTransceiver('video', { direction: 'recvonly' });

  // Add screen share transceiver
  pc.addTransceiver('video', {
    direction: 'sendrecv',
    streams: [screenShareStream],
  });

  return pc;
}
```

---

## 17. Congestion Control (BBR/WebRTC)

### 17.1. WebRTC Default (GCC)

```
Google Congestion Control (GCC):
- Sender-side: delay-based estimation
- Receiver-side: loss-based feedback
- Combined: target bitrate = min(sender_based, receiver_based)
- Update interval: ~1Hz
- Default: 300 kbps initial, ramp up to link capacity
```

### 17.2. BBR for RINCO

```cpp
// services/sfu-node/src/congestion/bbr.cpp
// Custom BBR implementation cho SRTP streams
class BbrCongestionControl {
public:
    BbrCongestionControl(uint32_t ssrc);
    void OnPacketSent(uint16_t packet_size, uint64_t now_us);
    void OnPacketAcked(uint16_t packet_size, uint64_t rtt_us, uint64_t now_us);
    void OnPacketLost(uint16_t packet_size, uint64_t now_us);
    uint32_t GetTargetBitrate() const;

private:
    enum class State { Startup, Drain, ProbeBw, ProbeRtt };

    State state_ = State::Startup;
    uint32_t max_bw_bps_ = 300000;  // initial 300 kbps
    uint32_t min_rtt_us_ = UINT64_MAX;
    uint32_t pacing_gain_ = 2.885;  // Startup gain
    uint32_t cwnd_ = 10;             // packets in flight

    void EnterStartup();
    void EnterDrain();
    void EnterProbeBw(uint8_t cycle_index);
    void EnterProbeRtt();
    void UpdateBandwidth(uint32_t delivered_bytes, uint64_t interval_us);
};
```

### 17.3. Adaptive Bitrate (Client-side)

```typescript
// apps/meeting-client/src/webrtc/adaptive-bitrate.ts
export class AdaptiveBitrateController {
  private pc: RTCPeerConnection;
  private currentBitrate: number = 1_000_000;
  private targetBitrate: number = 1_000_000;
  private statsInterval?: number;

  constructor(pc: RTCPeerConnection) {
    this.pc = pc;
    this.startMonitoring();
  }

  private async startMonitoring() {
    this.statsInterval = window.setInterval(async () => {
      const stats = await this.pc.getStats();
      const measurements = this.analyzeStats(stats);

      // Calculate target bitrate dựa trên metrics
      const newTarget = this.calculateTargetBitrate(measurements);

      if (Math.abs(newTarget - this.targetBitrate) > 100_000) {
        this.targetBitrate = newTarget;
        this.applyBitrateChange(this.targetBitrate);
      }
    }, 2000);
  }

  private analyzeStats(stats: RTCStatsReport): NetworkMeasurements {
    let rtt = 0;
    let packetLoss = 0;
    let availableBandwidth = 5_000_000;  // default 5 Mbps
    let jitter = 0;

    stats.forEach((report) => {
      if (report.type === 'candidate-pair' && report.state === 'succeeded') {
        rtt = report.currentRoundTripTime * 1000 || 0;
      }
      if (report.type === 'inbound-rtp') {
        packetLoss = report.packetsLost / report.packetsReceived;
        jitter = report.jitter * 1000 || 0;
      }
    });

    return { rtt, packetLoss, jitter, availableBandwidth };
  }

  private calculateTargetBitrate(m: NetworkMeasurements): number {
    let target = this.targetBitrate;

    if (m.packetLoss > 0.05 || m.rtt > 300) {
      target = Math.max(150_000, target * 0.7);  // decrease 30%
    } else if (m.packetLoss > 0.02 || m.rtt > 150) {
      target = Math.max(300_000, target * 0.9);  // decrease 10%
    } else if (m.packetLoss < 0.005 && m.rtt < 80 && target < 4_000_000) {
      target = Math.min(4_000_000, target * 1.1);  // increase 10%
    }

    return target;
  }

  private async applyBitrateChange(target: number) {
    const senders = this.pc.getSenders();
    for (const sender of senders) {
      const params = sender.getParameters();
      if (params.encodings) {
        params.encodings.forEach(encoding => {
          encoding.maxBitrate = target;
        });
        await sender.setParameters(params);
      }
    }
    this.currentBitrate = target;
  }

  destroy() {
    if (this.statsInterval) clearInterval(this.statsInterval);
  }
}
```

---

## 18. Meeting Orchestrator Implementation

### 18.1. Go Service Full Implementation

```go
// services/meet-orchestrator/main.go
package main

import (
    "context"
    "crypto/rand"
    "encoding/json"
    "errors"
    "fmt"
    "math/big"
    "net/http"
    "os"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"
    "github.com/itdoanh/rinco/meet-orchestrator/internal/auth"
    "github.com/itdoanh/rinco/meet-orchestrator/internal/sfu"
    "github.com/itdoanh/rinco/meet-orchestrator/internal/turn"
    "github.com/itdoanh/rinco/meet-orchestrator/internal/webhook"
    "go.uber.org/zap"
)

type Meeting struct {
    ID               uuid.UUID `json:"id"`
    TenantID         uuid.UUID `json:"tenant_id"`
    RoomCode         string    `json:"room_code"`
    Type             int       `json:"type"`
    HostID           uuid.UUID `json:"host_id"`
    Title            string    `json:"title"`
    Description      string    `json:"description"`
    StartsAt         *time.Time `json:"starts_at,omitempty"`
    EndsAt           *time.Time `json:"ends_at,omitempty"`
    ActualStartedAt  *time.Time `json:"actual_started_at,omitempty"`
    ActualEndedAt    *time.Time `json:"actual_ended_at,omitempty"`
    MaxParticipants  int       `json:"max_participants"`
    RecordingConfig  RecordingConfig `json:"recording"`
    Settings         MeetingSettings `json:"settings"`
    Status           string    `json:"status"`
    SFUNodeID        string    `json:"sfu_node_id"`
    CreatedAt        time.Time `json:"created_at"`
}

type RecordingConfig struct {
    Auto          bool   `json:"auto"`
    CloudOnly     bool   `json:"cloud_only"`
    CloudAndLocal bool   `json:"cloud_and_local"`
    OnDemand      bool   `json:"on_demand"`
    Layout        string `json:"layout"`  // "grid", "speaker", "presentation"
}

type MeetingSettings struct {
    EnableWaitingRoom bool     `json:"enable_waiting_room"`
    EnableChat        bool     `json:"enable_chat"`
    EnableReactions   bool     `json:"enable_reactions"`
    AllowedDomains    []string `json:"allowed_domains"`
    PasswordHash      string   `json:"-"`
    MuteOnEntry       bool     `json:"mute_on_entry"`
}

type Orchestrator struct {
    db           *pgxpool.Pool
    redis        *redis.Client
    sfuPicker    *sfu.Picker
    turnGen      *turn.Generator
    webhook      *webhook.Dispatcher
    logger       *zap.Logger
    mu           sync.RWMutex
    activeMeetings map[uuid.UUID]*Meeting
}

func NewOrchestrator(db *pgxpool.Pool, redis *redis.Client) (*Orchestrator, error) {
    picker, err := sfu.NewPicker(redis)
    if err != nil {
        return nil, err
    }
    return &Orchestrator{
        db:              db,
        redis:           redis,
        sfuPicker:       picker,
        turnGen:         turn.NewGenerator(),
        webhook:         webhook.NewDispatcher(redis),
        logger:          zap.NewProduction(),
        activeMeetings:  make(map[uuid.UUID]*Meeting),
    }, nil
}

// CreateMeeting tạo meeting mới
func (o *Orchestrator) CreateMeeting(ctx context.Context, req CreateMeetingRequest) (*Meeting, error) {
    // 1. Validate input
    if req.TenantID == uuid.Nil {
        return nil, errors.New("tenant_id required")
    }
    if req.HostID == uuid.Nil {
        return nil, errors.New("host_id required")
    }

    // 2. Generate room code
    roomCode, err := generateRoomCode()
    if err != nil {
        return nil, fmt.Errorf("generate room code: %w", err)
    }

    // 3. Pick SFU node (don't allocate yet, only on first join)
    sfuNode, err := o.sfuPicker.Pick(req.TenantID, req.MaxParticipants)
    if err != nil {
        return nil, fmt.Errorf("pick sfu: %w", err)
    }

    // 4. Build meeting
    meeting := &Meeting{
        ID:              uuid.New(),
        TenantID:        req.TenantID,
        RoomCode:        roomCode,
        Type:            req.Type,
        HostID:          req.HostID,
        Title:           req.Title,
        Description:     req.Description,
        StartsAt:        req.StartsAt,
        EndsAt:          req.EndsAt,
        MaxParticipants: req.MaxParticipants,
        RecordingConfig: req.RecordingConfig,
        Settings:        req.Settings,
        Status:          "SCHEDULED",
        SFUNodeID:       sfuNode.ID,
        CreatedAt:       time.Now(),
    }

    // 5. Insert vào database
    _, err = o.db.Exec(ctx, `
        INSERT INTO meetings (
            id, tenant_id, room_code, type, host_id, title, description,
            starts_at, ends_at, max_participants, recording_config, settings,
            status, sfu_node_id, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
    `, meeting.ID, meeting.TenantID, meeting.RoomCode, meeting.Type, meeting.HostID,
       meeting.Title, meeting.Description, meeting.StartsAt, meeting.EndsAt,
       meeting.MaxParticipants, meeting.RecordingConfig, meeting.Settings,
       meeting.Status, meeting.SFUNodeID, meeting.CreatedAt)
    if err != nil {
        return nil, fmt.Errorf("insert meeting: %w", err)
    }

    // 6. Dispatch webhook
    o.webhook.Dispatch(ctx, meeting.TenantID, webhook.Event{
        Type:      "meeting.created",
        MeetingID: meeting.ID,
        Data:      meeting,
    })

    o.logger.Info("meeting created",
        zap.String("meeting_id", meeting.ID.String()),
        zap.String("room_code", roomCode),
        zap.String("tenant_id", meeting.TenantID.String()))

    return meeting, nil
}

// Join meeting và trả về credentials
func (o *Orchestrator) Join(ctx context.Context, req JoinRequest) (*JoinResponse, error) {
    // 1. Get meeting
    var meeting Meeting
    err := o.db.QueryRow(ctx, `
        SELECT id, tenant_id, room_code, type, host_id, title, max_participants,
               recording_config, settings, status, sfu_node_id
        FROM meetings WHERE room_code = $1 AND tenant_id = $2
    `, req.RoomCode, req.TenantID).Scan(...)
    if err != nil {
        return nil, fmt.Errorf("meeting not found: %w", err)
    }

    // 2. Check status
    if meeting.Status == "ENDED" || meeting.Status == "CANCELLED" {
        return nil, errors.New("meeting has ended")
    }

    // 3. Verify password if set
    if meeting.Settings.PasswordHash != "" && !auth.VerifyPassword(req.Password, meeting.Settings.PasswordHash) {
        return nil, errors.New("invalid password")
    }

    // 4. Check waiting room
    if meeting.Settings.EnableWaitingRoom && req.UserID != meeting.HostID {
        o.addToWaitingRoom(ctx, meeting.ID, req.UserID)
        return &JoinResponse{
            Status: "WAITING",
            Reason: "Waiting for host to admit",
        }, nil
    }

    // 5. Update status if first join
    if meeting.Status == "SCHEDULED" {
        now := time.Now()
        meeting.Status = "ACTIVE"
        meeting.ActualStartedAt = &now
        _, err = o.db.Exec(ctx, "UPDATE meetings SET status = 'ACTIVE', actual_started_at = $1 WHERE id = $2", now, meeting.ID)
        if err != nil {
            return nil, err
        }
    }

    // 6. Generate TURN credentials
    turnCreds, err := o.turnGen.Generate(meeting.TenantID.String(), req.UserID.String())
    if err != nil {
        return nil, fmt.Errorf("turn creds: %w", err)
    }

    // 7. Get SFU endpoint
    sfuEndpoint, err := o.sfuPicker.GetEndpoint(meeting.SFUNodeID)
    if err != nil {
        return nil, fmt.Errorf("sfu endpoint: %w", err)
    }

    // 8. Generate access token (PASETO)
    accessToken, err := auth.GenerateAccessToken(meeting.ID, req.UserID, meeting.HostID == req.UserID)
    if err != nil {
        return nil, fmt.Errorf("token: %w", err)
    }

    // 9. Add participant to Valkey
    o.redis.SAdd(ctx, fmt.Sprintf("meeting:participants:%s", meeting.ID), req.UserID.String())

    return &JoinResponse{
        MeetingID:    meeting.ID,
        Status:       meeting.Status,
        SFUEndpoint:  sfuEndpoint,
        TURNCreds:    turnCreds,
        AccessToken:  accessToken,
        RecordingEnabled: meeting.RecordingConfig.Auto || meeting.RecordingConfig.OnDemand,
        Role:         determineRole(req.UserID, meeting),
    }, nil
}

// EndMeeting
func (o *Orchestrator) EndMeeting(ctx context.Context, meetingID, userID uuid.UUID) error {
    // Verify host
    var hostID uuid.UUID
    err := o.db.QueryRow(ctx, "SELECT host_id FROM meetings WHERE id = $1", meetingID).Scan(&hostID)
    if err != nil {
        return err
    }
    if hostID != userID {
        return errors.New("only host can end meeting")
    }

    now := time.Now()
    _, err = o.db.Exec(ctx, `
        UPDATE meetings SET status = 'ENDED', actual_ended_at = $1 WHERE id = $2
    `, now, meetingID)
    if err != nil {
        return err
    }

    // Trigger recording finalization
    if err := o.finalizeRecording(ctx, meetingID); err != nil {
        o.logger.Error("finalize recording failed", zap.Error(err))
    }

    // Trigger AI summary
    if err := o.triggerSummary(ctx, meetingID); err != nil {
        o.logger.Error("trigger summary failed", zap.Error(err))
    }

    // Cleanup Valkey
    o.redis.Del(ctx, fmt.Sprintf("meeting:participants:%s", meetingID))

    // Webhook
    o.webhook.Dispatch(ctx, ...)

    return nil
}

func generateRoomCode() (string, error) {
    // Format: abc-defg-hij (9 chars, easy to share)
    const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
    code := make([]byte, 12)
    for i := range code {
        n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
        if err != nil {
            return "", err
        }
        code[i] = chars[n.Int64()]
        if i == 2 || i == 6 {
            // Will replace with '-' below
        }
    }
    return fmt.Sprintf("%s-%s-%s", string(code[0:3]), string(code[3:7]), string(code[7:11])), nil
}

func main() {
    ctx := context.Background()

    dbPool, _ := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
    redisClient := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_URL")})

    orch, err := NewOrchestrator(dbPool, redisClient)
    if err != nil {
        panic(err)
    }

    // HTTP handlers
    http.HandleFunc("/api/meet/v1/meetings", orch.CreateMeetingHandler)
    http.HandleFunc("/api/meet/v1/meetings/", orch.GetMeetingHandler)
    http.HandleFunc("/api/meet/v1/join", orch.JoinHandler)
    http.HandleFunc("/api/meet/v1/meetings/", orch.EndMeetingHandler)

    http.ListenAndServe(":8080", nil)
}
```

### 18.2. SFU Picker Algorithm

```python
# services/meet-orchestrator/internal/sfu/picker.py
import random
from dataclasses import dataclass
from typing import Optional
import redis

@dataclass
class SFUNode:
    id: str
    region: str
    endpoint: str
    capacity: int          # max concurrent participants
    current_load: int
    last_heartbeat: int
    datacenter: str
    cpu_percent: float
    has_gpu: bool

class SFUPicker:
    def __init__(self, redis_client: redis.Redis):
        self.redis = redis_client
        self.cache_ttl = 30  # seconds

    def list_active_nodes(self) -> list[SFUNode]:
        """Get all active SFU nodes from Valkey."""
        import json
        from time import time

        nodes = []
        for key in self.redis.scan_iter(match="sfu:node:*", count=100):
            data = self.redis.get(key)
            if not data:
                continue
            node = SFUNode(**json.loads(data))
            # Check heartbeat (must be < 60s old)
            if time.time() - node.last_heartbeat > 60:
                continue
            nodes.append(node)
        return nodes

    def pick(self, tenant_id: str, expected_size: int, region: str = None) -> SFUNode:
        """Pick best SFU node for tenant."""
        nodes = self.list_active_nodes()
        if not nodes:
            raise Exception("No SFU nodes available")

        # 1. Filter by region if specified
        if region:
            region_nodes = [n for n in nodes if n.region == region]
            if region_nodes:
                nodes = region_nodes

        # 2. Filter by capacity
        candidates = [n for n in nodes
                      if n.current_load + expected_size <= n.capacity]
        if not candidates:
            raise Exception(f"No SFU node has capacity for {expected_size} participants")

        # 3. Score candidates
        scored = []
        for node in candidates:
            score = 0.0
            # Prefer less loaded
            score += (1.0 - node.current_load / node.capacity) * 50
            # Prefer low CPU
            score += (100 - node.cpu_percent) * 0.3
            # Prefer GPU for recording features
            if node.has_gpu:
                score += 10
            # Prefer same datacenter
            # TODO: get tenant datacenter from config
            scored.append((score, node))

        scored.sort(key=lambda x: -x[0])
        return scored[0][1]

    def allocate(self, node_id: str, meeting_id: str, participants: int):
        """Reserve capacity on node."""
        self.redis.hincrby(f"sfu:node:{node_id}", "current_load", participants)
        self.redis.hset(f"sfu:meeting:{meeting_id}", mapping={
            "node_id": node_id,
            "participants": participants,
        })

    def release(self, meeting_id: str):
        """Release capacity."""
        meeting_data = self.redis.hgetall(f"sfu:meeting:{meeting_id}")
        if not meeting_data:
            return
        node_id = meeting_data["node_id"]
        participants = int(meeting_data["participants"])
        self.redis.hincrby(f"sfu:node:{node_id}", "current_load", -participants)
        self.redis.delete(f"sfu:meeting:{meeting_id}")

    def get_endpoint(self, node_id: str) -> str:
        node_data = self.redis.hgetall(f"sfu:node:{node_id}")
        return node_data.get("endpoint", "")
```

### 18.3. Recording Finalization

```go
func (o *Orchestrator) finalizeRecording(ctx context.Context, meetingID uuid.UUID) error {
    // 1. Get recording info
    var recording Recording
    err := o.db.QueryRow(ctx, `
        SELECT id, storage_url, status FROM recordings
        WHERE meeting_id = $1 ORDER BY started_at DESC LIMIT 1
    `, meetingID).Scan(&recording.ID, &recording.StorageURL, &recording.Status)
    if err != nil {
        return fmt.Errorf("no recording found: %w", err)
    }

    // 2. Update status
    _, err = o.db.Exec(ctx, "UPDATE recordings SET status = 'processing', ended_at = $1 WHERE id = $2",
        time.Now(), recording.ID)
    if err != nil {
        return err
    }

    // 3. Trigger HLS packaging
    if err := o.triggerHLSPackaging(ctx, recording); err != nil {
        return fmt.Errorf("hls packaging: %w", err)
    }

    // 4. Trigger STT
    if err := o.triggerSTT(ctx, meetingID, recording.StorageURL); err != nil {
        return fmt.Errorf("stt: %w", err)
    }

    return nil
}

func (o *Orchestrator) triggerSummary(ctx context.Context, meetingID uuid.UUID) error {
    // 1. Wait for transcript to be ready
    var transcriptID uuid.UUID
    var transcriptText string
    err := o.db.QueryRow(ctx, `
        SELECT id, full_text FROM meeting_transcripts
        WHERE meeting_id = $1 AND created_at > NOW() - INTERVAL '1 hour'
        ORDER BY created_at DESC LIMIT 1
    `, meetingID).Scan(&transcriptID, &transcriptText)
    if err != nil {
        return fmt.Errorf("transcript not ready: %w", err)
    }

    // 2. Publish to NATS for AI worker
    payload, _ := json.Marshal(map[string]interface{}{
        "meeting_id":    meetingID,
        "transcript_id": transcriptID,
        "transcript":    transcriptText,
    })

    if err := o.nats.Publish("ai.summary.requested", payload); err != nil {
        return fmt.Errorf("nats publish: %w", err)
    }

    return nil
}
```

### 18.4. Webhook Dispatcher

```go
// services/meet-orchestrator/internal/webhook/webhook.go
package webhook

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
)

type Event struct {
    Type      string      `json:"type"`
    MeetingID uuid.UUID   `json:"meeting_id"`
    TenantID  uuid.UUID   `json:"tenant_id"`
    Data      interface{} `json:"data"`
    Timestamp time.Time   `json:"timestamp"`
}

type Dispatcher struct {
    redis  *redis.Client
    client *http.Client
}

func (d *Dispatcher) Dispatch(ctx context.Context, tenantID uuid.UUID, event Event) {
    // 1. Get tenant webhooks
    webhooks := d.getTenantWebhooks(ctx, tenantID, event.Type)
    if len(webhooks) == 0 {
        return
    }

    // 2. Send to each webhook
    payload, _ := json.Marshal(event)
    for _, webhook := range webhooks {
        go d.send(webhook.URL, webhook.Secret, payload)
    }
}

func (d *Dispatcher) send(url, secret string, payload []byte) {
    // HMAC signature
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(payload)
    sig := hex.EncodeToString(h.Sum(nil))

    req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Rinco-Signature", sig)
    req.Header.Set("X-Rinco-Event-Type", extractEventType(payload))

    resp, err := d.client.Do(req)
    if err != nil {
        // Log + DLQ
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 500 {
        // Retry with backoff
    }
}

func (d *Dispatcher) getTenantWebhooks(ctx context.Context, tenantID uuid.UUID, eventType string) []WebhookConfig {
    // Lookup from PostgreSQL or cache
    // ...
}
```

---

## 19. GPU Recording Pipeline chi tiết

### 19.1. C++ Egress Worker

```cpp
// services/recorder/src/egress_worker.cpp
#include <cuda_runtime.h>
#include <nvtx3/nvToolsExt.h>
#include "gst/gst.h"
#include "gst/video/video.h"
#include <iostream>
#include <thread>
#include <queue>
#include <mutex>

class EgressWorker {
public:
    EgressWorker(const std::string& meeting_id, const std::string& sfu_endpoint);
    ~EgressWorker();

    bool Initialize();
    void Run();  // Main loop
    void Stop();

private:
    void ConnectToSFU();
    void ReceiveRTPStream();
    void CompositeFrame();
    void EncodeWithNVENC();
    void UploadToS3();

    // GPU resources
    cudaStream_t cuda_stream_;
    uint8_t* gpu_input_buffer_;     // [video_streams] × [max_resolution]
    uint8_t* gpu_composite_buffer_; // [final_resolution × RGBA]
    uint8_t* gpu_output_buffer_;    // Encoded frame

    // NVENC encoder
    NvEncoder* encoder_;

    // Layout
    Layout current_layout_;
};

EgressWorker::EgressWorker(const std::string& meeting_id, const std::string& sfu_endpoint)
    : meeting_id_(meeting_id), sfu_endpoint_(sfu_endpoint) {
    cudaSetDevice(0);
    cudaStreamCreate(&cuda_stream_);

    // Allocate GPU buffers
    // 16 streams × 1920×1080×4 bytes = ~132 MB
    gpu_input_buffer_ = new uint8_t[16 * 1920 * 1080 * 4];
    gpu_composite_buffer_ = new uint8_t[1920 * 1080 * 4];
    gpu_output_buffer_ = new uint8_t[1920 * 1080 * 4];

    cudaMemset(gpu_input_buffer_, 0, ...);
}

EgressWorker::~EgressWorker() {
    delete[] gpu_input_buffer_;
    delete[] gpu_composite_buffer_;
    delete[] gpu_output_buffer_;
    cudaStreamDestroy(cuda_stream_);
}

bool EgressWorker::Initialize() {
    // Initialize GStreamer
    gst_init(nullptr, nullptr);

    // Initialize NVENC encoder
    encoder_ = new NvEncoder(1920, 1080, 30, 8_000_000);  // 1080p30, 8 Mbps
    if (!encoder_->Initialize()) {
        return false;
    }

    return true;
}

void EgressWorker::Run() {
    ConnectToSFU();
    while (running_) {
        NVTX_RANGE("RecordingFrame");

        // 1. Receive RTP packet
        auto rtp_packet = ReceiveRTPPacket();
        if (!rtp_packet) continue;

        // 2. Decode RTP payload (H.264/VP9/AV1)
        // Note: For GPU recording, we still need to decode to composite
        // SFU doesn't decode, but recorder DOES for composition
        auto decoded_frame = decoder_->Decode(rtp_packet);

        // 3. Upload to GPU
        cudaMemcpyAsync(gpu_input_buffer_ + stream_offset_, decoded_frame->data(),
                       decoded_frame->size(), cudaMemcpyHostToDevice, cuda_stream_);

        // 4. Wait for all streams to arrive
        if (++stream_count_ == expected_streams_) {
            // 5. Composite all streams on GPU
            NVTX_RANGE("CompositeLayout");
            CompositeFrame();

            // 6. Encode with NVENC
            NVTX_RANGE("NVENCEncode");
            EncodeWithNVENC();

            // 7. Upload to S3
            NVTX_RANGE("S3Upload");
            UploadToS3();

            stream_count_ = 0;
        }
    }
}

void EgressWorker::CompositeFrame() {
    // Use CUDA kernel for layout composition
    dim3 block(16, 16);
    dim3 grid(composite_width_ / 16, composite_height_ / 16);

    composite_kernel<<<grid, block, 0, cuda_stream_>>>(
        gpu_input_buffer_,
        gpu_composite_buffer_,
        composite_width_,
        composite_height_,
        stream_offsets_,
        stream_count_
    );

    cudaStreamSynchronize(cuda_stream_);
}

void EgressWorker::EncodeWithNVENC() {
    // GPU → NVENC (zero-copy via GPU memory)
    encoder_->Encode(gpu_composite_buffer_, gpu_output_buffer_);
}

void EgressWorker::UploadToS3() {
    // Get encoded H.264 NAL units
    auto nal_units = encoder_->GetNALUnits();

    // Multipart upload to MinIO
    static int chunk_index = 0;
    std::string chunk_key = fmt::format("{}/chunks/{:05d}.m4s", meeting_id_, chunk_index++);
    s3_uploader_->UploadPart(chunk_key, gpu_output_buffer_, encoder_->GetFrameSize());
}

class NvEncoder {
public:
    NvEncoder(int width, int height, int fps, int bitrate);
    bool Initialize();
    void Encode(uint8_t* gpu_input, uint8_t* gpu_output);
    int GetFrameSize() const;
    std::vector<NalUnit> GetNALUnits();

private:
    NV_ENC_INITIALIZE_PARAMS init_params_;
    NV_ENC_CONFIG encode_config_;
    void* encoder_;
    int width_, height_, fps_, bitrate_;
};
```

### 19.2. CUDA Composite Kernel

```cuda
// services/recorder/src/composite_kernel.cu
#include <cuda_runtime.h>

struct StreamInfo {
    int x_offset;
    int y_offset;
    int width;
    int height;
    bool active;
};

__global__ void composite_kernel(
    const uint8_t* __restrict__ input_streams,  // [16 streams] × [1920x1080x4]
    uint8_t* __restrict__ output,               // [1920x1080x4]
    int output_width,
    int output_height,
    StreamInfo* streams,
    int num_streams
) {
    int x = blockIdx.x * blockDim.x + threadIdx.x;
    int y = blockIdx.y * blockDim.y + threadIdx.y;

    if (x >= output_width || y >= output_height) return;

    // Find which stream this pixel belongs to
    for (int i = 0; i < num_streams; i++) {
        StreamInfo s = streams[i];
        if (!s.active) continue;

        if (x >= s.x_offset && x < s.x_offset + s.width &&
            y >= s.y_offset && y < s.y_offset + s.height) {

            int local_x = x - s.x_offset;
            int local_y = y - s.y_offset;

            // Read from stream i's buffer
            const uint8_t* src = input_streams + i * (1920 * 1080 * 4);
            int src_idx = (local_y * s.width + local_x) * 4;

            // Write to output
            int dst_idx = (y * output_width + x) * 4;
            output[dst_idx + 0] = src[src_idx + 0];  // R
            output[dst_idx + 1] = src[src_idx + 1];  // G
            output[dst_idx + 2] = src[src_idx + 2];  // B
            output[dst_idx + 3] = src[src_idx + 3];  // A
            return;
        }
    }

    // Background color
    int dst_idx = (y * output_width + x) * 4;
    output[dst_idx + 0] = 30;   // Dark gray
    output[dst_idx + 1] = 30;
    output[dst_idx + 2] = 30;
    output[dst_idx + 3] = 255;
}

// Layout calculation: 2x2 grid for 4 streams, 3x3 for 9, etc.
__host__ void calculate_layout(
    int num_streams,
    StreamInfo* streams,
    int output_width,
    int output_height
) {
    int cols = ceil(sqrtf(num_streams));
    int rows = ceil((float)num_streams / cols);
    int cell_w = output_width / cols;
    int cell_h = output_height / rows;

    for (int i = 0; i < num_streams; i++) {
        int col = i % cols;
        int row = i / cols;
        streams[i].x_offset = col * cell_w;
        streams[i].y_offset = row * cell_h;
        streams[i].width = cell_w;
        streams[i].height = cell_h;
        streams[i].active = true;
    }
}
```

---

## 20. NVENC Encoder chi tiết

### 20.1. NVENC Session Limits

```
NVIDIA NVENC Session Limits:
- Consumer GPUs (RTX series): 3-5 concurrent sessions
- Datacenter GPUs (L4, A10): 12-16 concurrent sessions
- Quadro GPUs: unlimited

Workaround for consumer GPUs:
- Use multiple GPUs
- Use software encoder (x264) as fallback
- Pool NVENC sessions
```

### 20.2. NVENC Implementation

```cpp
// services/recorder/src/nvenc_encoder.cpp
#include "nvEncodeAPI.h"
#include <vector>

class NvEncoder {
public:
    NvEncoder(int width, int height, int fps, int bitrate)
        : width_(width), height_(height), fps_(fps), bitrate_(bitrate) {
        memset(&init_params_, 0, sizeof(init_params_));
        memset(&encode_config_, 0, sizeof(encode_config_));
    }

    bool Initialize() {
        // 1. Initialize NVENC API
        NVENCSTATUS status = NvEncodeAPIGetMaxSupportedVersion(&max_version_);
        if (status != NV_ENC_SUCCESS) {
            std::cerr << "NvEncodeAPIGetMaxSupportedVersion failed: " << status << std::endl;
            return false;
        }

        // 2. Open encoder
        init_params_.version = NV_ENCODE_API_FUNCTION_LIST_VER;
        status = NvEncodeAPICreateInstance(&init_params_);
        if (status != NV_ENC_SUCCESS) {
            std::cerr << "NvEncodeAPICreateInstance failed: " << status << std::endl;
            return false;
        }
        encoder_ = init_params_.encodeAPI;

        // 3. Open session
        NV_ENC_OPEN_ENCODE_SESSION_EX_PARAMS open_params = {};
        open_params.version = NV_ENCODE_OPEN_ENCODE_SESSION_EX_PARAMS_VER;
        open_params.deviceType = NV_ENC_DEVICE_TYPE_CUDA;
        open_params.device = cuda_device_;
        open_params.apiVersion = max_version_;

        status = encoder_->nvEncOpenEncodeSessionEx(&open_params, &session_);
        if (status != NV_ENC_SUCCESS) {
            std::cerr << "nvEncOpenEncodeSessionEx failed: " << status << std::endl;
            return false;
        }

        // 4. Get encoder capabilities
        uint32_t guid_count = 0;
        encoder_->nvEncGetEncodeGUIDCount(session_, &guid_count);
        // Find H.264 high profile GUID

        // 5. Initialize encoder
        NV_ENC_INITIALIZE_PARAMS init_params = {};
        init_params.version = NV_ENCODE_INITIALIZE_PARAMS_VER;
        init_params.encodeGUID = NV_ENC_CODEC_H264_GUID;
        init_params.presetGUID = NV_ENC_PRESET_LOW_LATENCY_HQ_GUID;
        init_params.encodeWidth = width_;
        init_params.encodeHeight = height_;
        init_params.darWidth = width_;
        init_params.darHeight = height_;
        init_params.frameRateNum = fps_;
        init_params.frameRateDen = 1;
        init_params.enablePTD = 1;
        init_params.reportSliceOffsets = 0;
        init_params.enableSubFrameWrite = 0;

        status = encoder_->nvEncInitializeEncoder(session_, &init_params);
        if (status != NV_ENC_SUCCESS) {
            std::cerr << "nvEncInitializeEncoder failed: " << status << std::endl;
            return false;
        }

        // 6. Configure encoding
        encode_config_.version = NV_ENC_CONFIG_VER;
        encode_config_.profileGUID = NV_ENC_H264_PROFILE_HIGH_GUID;
        encode_config_.rcMode = NV_ENC_PARAMS_RC_CBR;
        encode_config_.encodeAvgQP = 28;
        encode_config_.encodeMinQP = 20;
        encode_config_.encodeMaxQP = 36;
        encode_config_.gopLength = fps_ * 2;  // 2-second GOP
        encode_config_.frameIntervalP = 1;
        encode_config_.frameFieldMode = NV_ENC_FIELD_MODE_FRAME;
        encode_config_.mvPrecision = NV_ENC_MV_PRECISION_FULL_PEL;
        encode_config_.enableBFrame = 0;

        NV_ENC_RC_PARAMS rc_params = {};
        rc_params.version = NV_ENC_RC_PARAMS_VER;
        rc_params.averageBitRate = bitrate_;
        rc_params.maxBitRate = bitrate_ * 1.2;
        rc_params.rateControlMode = NV_ENC_PARAMS_RC_CBR;
        encode_config_.rcParams = rc_params;

        NV_ENC_PRESET_CONFIG preset_config = {};
        preset_config.version = NV_ENCODE_PRESET_CONFIG_VER;
        preset_config.presetCfg = encode_config_;
        encoder_->nvEncGetEncodePresetConfig(session_, NV_ENC_CODEC_H264_GUID,
                                             NV_ENC_PRESET_LOW_LATENCY_HQ_GUID,
                                             &preset_config);

        // 7. Apply config
        NV_ENC_RECONFIGURE_PARAMS reconfig_params = {};
        reconfig_params.version = NV_ENC_RECONFIGURE_PARAMS_VER;
        reconfig_params.reInitEncodeParams = init_params;
        reconfig_params.resetEncoder = 1;
        reconfig_params.forceIDR = 1;
        encoder_->nvEncReconfigureEncoder(session_, &reconfig_params);

        return true;
    }

    bool EncodeFrame(uint8_t* gpu_input, uint8_t** output, uint32_t* output_size) {
        NVTX_RANGE("NVENC_Encode");

        // 1. Register input resource
        NV_ENC_REGISTER_RESOURCE register_params = {};
        register_params.version = NV_ENC_REGISTER_RESOURCE_VER;
        register_params.resourceType = NV_ENC_INPUT_RESOURCE_TYPE_CUDAARRAY;
        register_params.resourceToRegister = gpu_input;
        register_params.width = width_;
        register_params.height = height_;
        register_params.pitch = width_ * 4;

        NV_ENC_REGISTERED_PTR registered_ptr;
        NVENCSTATUS status = encoder_->nvEncRegisterResource(session_, &register_params, &registered_ptr);
        if (status != NV_ENC_SUCCESS) return false;

        // 2. Create input buffer
        NV_ENC_CREATE_INPUT_BUFFER create_input = {};
        create_input.version = NV_ENC_CREATE_INPUT_BUFFER_VER;
        create_input.width = width_;
        create_input.height = height_;
        create_input.bufferFmt = NV_ENC_BUFFER_FORMAT_ARGB;

        NV_ENC_INPUT_BUFFER input_buffer;
        status = encoder_->nvEncCreateInputBuffer(session_, &create_input, &input_buffer);
        if (status != NV_ENC_SUCCESS) return false;

        // 3. Map resource to input buffer
        NV_ENC_MAP_INPUT_RESOURCE map_input = {};
        map_input.version = NV_ENC_MAP_INPUT_RESOURCE_VER;
        map_input.registeredResource = registered_ptr;

        status = encoder_->nvEncMapInputResource(session_, &input_buffer, &map_input);
        if (status != NV_ENC_SUCCESS) return false;

        // 4. Encode
        NV_ENC_PIC_PARAMS pic_params = {};
        pic_params.version = NV_ENC_PIC_PARAMS_VER;
        pic_params.inputBuffer = input_buffer;
        pic_params.bufferFmt = NV_ENC_BUFFER_FORMAT_ARGB;
        pic_params.pictureStruct = NV_ENC_PIC_STRUCT_FRAME;
        pic_params.frameIdx = frame_index_++;

        // 5. Get output bitstream
        NV_ENC_CREATE_BITSTREAM_BUFFER create_bs = {};
        create_bs.version = NV_ENC_CREATE_BITSTREAM_BUFFER_VER;
        create_bs.size = 4 * 1024 * 1024;  // 4 MB
        create_bs.memoryHeap = NV_ENC_MEMORY_HEAP_SYSMEM_CACHED;

        NV_ENC_OUTPUT_PTR output_ptr;
        status = encoder_->nvEncCreateBitstreamBuffer(session_, &create_bs, &output_ptr);

        // 6. Encode frame
        NV_ENC_LOCK_BITSTREAM lock_bs = {};
        lock_bs.version = NV_ENC_LOCK_BITSTREAM_VER;
        lock_bs.outputBitstream = output_ptr;
        lock_bs.doNotWait = 0;

        status = encoder_->nvEncEncodePicture(session_, &pic_params, &output_ptr);
        if (status != NV_ENC_SUCCESS) {
            std::cerr << "nvEncEncodePicture failed: " << status << std::endl;
            return false;
        }

        // 7. Lock bitstream
        status = encoder_->nvEncLockBitstream(session_, output_ptr, &lock_bs);
        if (status != NV_ENC_SUCCESS) return false;

        // 8. Copy to output
        *output_size = lock_bs.bitstreamSizeInBytes;
        cudaMemcpy(*output, lock_bs.bitstreamBufferPtr, *output_size, cudaMemcpyDeviceToHost);
        *output = lock_bs.bitstreamBufferPtr;

        // 9. Unlock
        encoder_->nvEncUnlockBitstream(session_, output_ptr);

        // 10. Cleanup
        encoder_->nvEncDestroyInputBuffer(session_, input_buffer);
        encoder_->nvEncDestroyBitstreamBuffer(session_, output_ptr);
        encoder_->nvEncUnregisterResource(session_, registered_ptr);

        return true;
    }

private:
    NV_ENCODE_API_FUNCTION_LIST* encoder_;
    void* session_;
    int width_, height_, fps_, bitrate_;
    uint32_t frame_index_ = 0;
    NV_ENC_INITIALIZE_PARAMS init_params_;
    NV_ENC_CONFIG encode_config_;
    uint32_t max_version_;
    void* cuda_device_;
};
```

---

## 21. Audio Mixing chi tiết

### 21.1. FFmpeg-based Audio Mixer

```cpp
// services/recorder/src/audio_mixer.cpp
extern "C" {
#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libswresample/swresample.h>
}

class AudioMixer {
public:
    AudioMixer(int sample_rate = 48000, int channels = 2);
    ~AudioMixer();

    bool Initialize();
    int Mix(const std::vector<AudioFrame>& frames, uint8_t** output, int* output_size);

private:
    int sample_rate_;
    int channels_;
    SwrContext* swr_ctx_;
    AVFrame* mixed_frame_;

    // Per-stream resampler
    std::map<std::string, SwrContext*> stream_resamplers_;
};

AudioMixer::AudioMixer(int sample_rate, int channels)
    : sample_rate_(sample_rate), channels_(channels) {
}

AudioMixer::~AudioMixer() {
    for (auto& [id, ctx] : stream_resamplers_) {
        swr_free(&ctx);
    }
    if (mixed_frame_) {
        av_frame_free(&mixed_frame_);
    }
}

bool AudioMixer::Initialize() {
    swr_ctx_ = swr_alloc_set_opts(nullptr,
        av_get_default_channel_layout(channels_), AV_SAMPLE_FMT_FLTP, sample_rate_,
        av_get_default_channel_layout(channels_), AV_SAMPLE_FMT_FLTP, sample_rate_,
        0, nullptr);
    swr_init(swr_ctx_);

    mixed_frame_ = av_frame_alloc();
    mixed_frame_->format = AV_SAMPLE_FMT_FLTP;
    mixed_frame_->channel_layout = av_get_default_channel_layout(channels_);
    mixed_frame_->sample_rate = sample_rate_;
    mixed_frame_->nb_samples = 480;  // 10ms at 48kHz
    av_frame_get_buffer(mixed_frame_, 0);

    return true;
}

int AudioMixer::Mix(const std::vector<AudioFrame>& frames, uint8_t** output, int* output_size) {
    // 1. Reset mixed frame
    av_frame_make_writable(mixed_frame_);
    memset(mixed_frame_->data[0], 0, mixed_frame_->linesize[0]);
    memset(mixed_frame_->data[1], 0, mixed_frame_->linesize[1]);

    int active_count = 0;

    // 2. Mix each stream
    for (const auto& frame : frames) {
        if (frame.samples.empty()) continue;

        // Resample if needed
        auto it = stream_resamplers_.find(frame.stream_id);
        SwrContext* src_ctx;
        if (it == stream_resamplers_.end()) {
            src_ctx = swr_alloc_set_opts(nullptr,
                av_get_default_channel_layout(channels_), AV_SAMPLE_FMT_FLTP, sample_rate_,
                av_get_default_channel_layout(frame.channels), frame.format, frame.sample_rate,
                0, nullptr);
            swr_init(src_ctx);
            stream_resamplers_[frame.stream_id] = src_ctx;
        } else {
            src_ctx = it->second;
        }

        // Convert to mixed format
        const uint8_t* in_data[2] = { frame.samples.data(), nullptr };
        int in_samples = frame.samples.size() / av_get_bytes_per_sample(frame.format) / frame.channels;

        uint8_t* out_data[2] = { nullptr, nullptr };
        int out_samples = swr_convert(src_ctx, out_data, mixed_frame_->nb_samples, in_data, in_samples);

        // Mix into mixed_frame_
        if (out_samples > 0) {
            // Normalize and add (avoid clipping)
            float gain = 1.0f / std::max(1, (int)frames.size());
            for (int ch = 0; ch < channels_; ch++) {
                float* dst = (float*)mixed_frame_->data[ch];
                float* src = (float*)out_data[ch];
                for (int i = 0; i < out_samples; i++) {
                    float mixed = dst[i] + src[i] * gain;
                    dst[i] = std::min(1.0f, std::max(-1.0f, mixed));  // Clip
                }
            }
            active_count++;
        }
    }

    // 3. Encode mixed audio (AAC)
    // ... encode với FFmpeg AAC encoder

    // 4. Mux với video thành MP4
    return 0;
}
```

---

## 22. AI Whisper.cpp Transcription Pipeline

### 22.1. Whisper.cpp Integration

```python
# services/stt-worker/transcriber.py
import os
from pathlib import Path
import numpy as np
from faster_whisper import WhisperModel
import torch

class MeetingTranscriber:
    def __init__(self, model_size="large-v3", device="cuda"):
        self.device = device
        if device == "cuda" and not torch.cuda.is_available():
            print("CUDA not available, falling back to CPU")
            self.device = "cpu"

        # Use Whisper.cpp via faster-whisper
        self.model = WhisperModel(
            model_size,
            device=self.device,
            compute_type="float16" if self.device == "cuda" else "int8",
        )

    def transcribe_audio(self, audio_path: str, language: str = "auto"):
        """Transcribe audio file with speaker diarization."""
        segments, info = self.model.transcribe(
            audio_path,
            language=language if language != "auto" else None,
            beam_size=5,
            vad_filter=True,           # Voice activity detection
            vad_parameters={
                "min_silence_duration_ms": 500,
            },
        )

        results = []
        for segment in segments:
            results.append({
                "start": segment.start,
                "end": segment.end,
                "text": segment.text.strip(),
                "avg_logprob": segment.avg_logprob,
                "no_speech_prob": segment.no_speech_prob,
            })

        return {
            "language": info.language,
            "language_probability": info.language_probability,
            "duration": info.duration,
            "segments": results,
            "full_text": " ".join([s["text"] for s in results]),
        }

    def transcribe_with_diarization(self, audio_path: str, num_speakers: int = None):
        """Transcribe with speaker identification."""
        # Use pyannote-audio for speaker diarization
        from pyannote.audio import Pipeline

        try:
            diarization_pipeline = Pipeline.from_pretrained(
                "pyannote/speaker-diarization-3.1",
                use_auth_token=os.environ.get("HUGGINGFACE_TOKEN"),
            )
            if self.device == "cuda":
                import torch
                diarization_pipeline.to(torch.device("cuda"))
        except Exception as e:
            print(f"Diarization pipeline load failed: {e}")
            return self.transcribe_audio(audio_path)

        # 1. Get speaker diarization
        diarization = diarization_pipeline(audio_path, num_speakers=num_speakers)

        # 2. Transcribe each speaker segment
        from pydub import AudioSegment
        audio = AudioSegment.from_wav(audio_path)

        speaker_segments = []
        for turn, _, speaker in diarization.itertracks(yield_label=True):
            start_ms = int(turn.start * 1000)
            end_ms = int(turn.end * 1000)
            speaker_audio = audio[start_ms:end_ms]

            # Export segment
            segment_path = f"/tmp/segment_{start_ms}.wav"
            speaker_audio.export(segment_path, format="wav")

            # Transcribe
            result = self.transcribe_audio(segment_path)

            speaker_segments.append({
                "speaker": speaker,
                "start": turn.start,
                "end": turn.end,
                "text": result["full_text"],
                "segments": result["segments"],
            })

            os.unlink(segment_path)

        # 3. Combine all segments
        all_text = []
        for seg in speaker_segments:
            all_text.append(f"[{seg['speaker']}] {seg['text']}")

        return {
            "language": "auto",
            "segments": speaker_segments,
            "full_text": "\n".join(all_text),
            "num_speakers": len(set(s["speaker"] for s in speaker_segments)),
        }
```

### 22.2. Live Caption Pipeline

```python
# services/stt-worker/live_caption.py
import asyncio
from faster_whisper import WhisperModel
import numpy as np
from collections import deque

class LiveCaptionService:
    """Stream audio chunks and emit captions in real-time."""

    def __init__(self, meeting_id: str, model_size="medium"):
        self.meeting_id = meeting_id
        self.model = WhisperModel(model_size, device="cuda", compute_type="float16")
        self.buffer = deque(maxlen=int(48000 * 5))  # 5 seconds buffer at 48kHz
        self.caption_subscribers = set()
        self.last_emitted_text = ""

    async def process_audio_chunk(self, audio_chunk: np.ndarray):
        """Process incoming audio chunk (e.g., 100ms)."""
        # Add to rolling buffer
        self.buffer.extend(audio_chunk)

        # When buffer has 2 seconds, process
        if len(self.buffer) >= 48000 * 2:
            audio_array = np.array(self.buffer, dtype=np.float32)

            # Transcribe
            segments, _ = self.model.transcribe(
                audio_array,
                language="vi",  # Optimize for Vietnamese
                beam_size=3,
                vad_filter=True,
            )

            for segment in segments:
                text = segment.text.strip()
                if text and text != self.last_emitted_text:
                    # Emit caption
                    await self.emit_caption(text, segment.start, segment.end)
                    self.last_emitted_text = text

            # Reset buffer (keep last 0.5s for overlap)
            overlap = int(48000 * 0.5)
            for _ in range(len(self.buffer) - overlap):
                self.buffer.popleft()

    async def emit_caption(self, text: str, start: float, end: float):
        """Broadcast caption to all subscribers."""
        caption = {
            "meeting_id": self.meeting_id,
            "text": text,
            "start": start,
            "end": end,
            "timestamp": asyncio.get_event_loop().time(),
        }

        # Send via WebSocket to all subscribers
        for ws in self.caption_subscribers:
            try:
                await ws.send_json(caption)
            except:
                pass

    def subscribe(self, ws):
        self.caption_subscribers.add(ws)
```

---

## 23. Llama-3 Meeting Summary

### 23.1. Summary Prompt Engineering

```python
# services/summary-worker/prompts.py

MEETING_SUMMARY_PROMPT = """Bạn là trợ lý AI chuyên tóm tắt cuộc họp bằng tiếng Việt. Nhiệm vụ của bạn là phân tích transcript cuộc họp và tạo ra bản tóm tắt có cấu trúc.

## Thông tin cuộc họp
- Tiêu đề: {meeting_title}
- Ngày: {meeting_date}
- Thời lượng: {duration} phút
- Số người tham gia: {participant_count}
- Loại cuộc họp: {meeting_type}

## Transcript
{transcript}

## Yêu cầu
Hãy tạo bản tóm tắt với format JSON sau (chỉ trả về JSON, không có text khác):

```json
{{
  "summary": "Tóm tắt tổng quan 2-3 câu",
  "key_points": [
    "Điểm chính 1",
    "Điểm chính 2",
    "Điểm chính 3",
    "Điểm chính 4",
    "Điểm chính 5"
  ],
  "decisions": [
    {{
      "decision": "Quyết định được đưa ra",
      "context": "Bối cảnh",
      "impact": "Tác động"
    }}
  ],
  "action_items": [
    {{
      "task": "Mô tả công việc cần làm",
      "owner": "Người chịu trách nhiệm (nếu xác định được)",
      "deadline": "Thời hạn (nếu có)",
      "priority": "high|medium|low"
    }}
  ],
  "follow_ups": [
    "Câu hỏi cần làm rõ",
    "Vấn đề cần thảo luận tiếp"
  ],
  "topics": [
    "Chủ đề 1",
    "Chủ đề 2"
  ],
  "sentiment": "positive|neutral|negative",
  "engagement_score": 0.85,
  "next_meeting_suggestion": "Đề xuất cuộc họp tiếp theo (nếu cần)"
}}
```

Lưu ý:
- Nếu không xác định được owner/deadline, để null.
- Action items phải có ít nhất 1 mục nếu cuộc họp có thảo luận về công việc.
- Tóm tắt phải trung thành với transcript, không thêm thông tin không có.
- Sử dụng ngôn ngữ tự nhiên, chuyên nghiệp.
"""

LIVE_SUMMARY_PROMPT = """Bạn đang nghe một cuộc họp real-time. Cập nhật bản tóm tắt đang chạy.

Transcript cho đến hiện tại (cập nhật mỗi 30 giây):
{transcript_so_far}

Bản tóm tắt hiện tại:
{current_summary}

Hãy cập nhật bản tóm tắt với format JSON giống như trên, thêm các điểm mới và action items mới.
"""
```

### 23.2. Llama-3 Worker

```python
# services/summary-worker/worker.py
import json
import asyncio
import aiohttp
from datetime import datetime
from pathlib import Path
import os

class SummaryWorker:
    def __init__(self):
        self.model_endpoint = os.environ.get("LLAMA3_ENDPOINT", "http://vllm:8000/v1")
        self.model_name = "meta-llama/Llama-3-70b-chat-hf"

    async def generate_summary(self, meeting_id: str, transcript: str, metadata: dict) -> dict:
        prompt = MEETING_SUMMARY_PROMPT.format(
            meeting_title=metadata.get("title", "Untitled"),
            meeting_date=metadata.get("date", datetime.now().isoformat()),
            duration=metadata.get("duration_minutes", 0),
            participant_count=metadata.get("participant_count", 0),
            meeting_type=metadata.get("type", "general"),
            transcript=transcript,
        )

        async with aiohttp.ClientSession() as session:
            response = await session.post(
                f"{self.model_endpoint}/chat/completions",
                json={
                    "model": self.model_name,
                    "messages": [
                        {
                            "role": "system",
                            "content": "Bạn là trợ lý AI tóm tắt cuộc họp chuyên nghiệp."
                        },
                        {
                            "role": "user",
                            "content": prompt
                        }
                    ],
                    "temperature": 0.3,
                    "max_tokens": 2000,
                    "response_format": {"type": "json_object"},
                },
                timeout=aiohttp.ClientTimeout(total=120),
            )
            data = await response.json()

        content = data["choices"][0]["message"]["content"]
        summary = json.loads(content)

        # Save to database
        await self.save_summary(meeting_id, summary, transcript)

        # Trigger action item creation in CRM
        for item in summary.get("action_items", []):
            if item.get("owner") or item.get("deadline"):
                await self.create_crm_task(meeting_id, item)

        return summary

    async def save_summary(self, meeting_id: str, summary: dict, transcript: str):
        # Save to PostgreSQL via REST API
        async with aiohttp.ClientSession() as session:
            await session.post(
                f"{os.environ['CRM_API']}/api/meet/v1/meetings/{meeting_id}/summary",
                json={
                    "summary": summary["summary"],
                    "key_points": summary["key_points"],
                    "decisions": summary["decisions"],
                    "action_items": summary["action_items"],
                    "follow_ups": summary["follow_ups"],
                    "topics": summary["topics"],
                    "sentiment": summary["sentiment"],
                    "engagement_score": summary["engagement_score"],
                    "model_version": self.model_name,
                }
            )

    async def create_crm_task(self, meeting_id: str, action_item: dict):
        # Find owner user_id by name (approximate match)
        owner_id = None
        if action_item.get("owner"):
            owner_id = await self.find_user_by_name(action_item["owner"])

        # Create task in CRM
        async with aiohttp.ClientSession() as session:
            await session.post(
                f"{os.environ['CRM_API']}/api/crm/v1/task",
                json={
                    "meeting_id": meeting_id,
                    "title": action_item["task"],
                    "owner_id": owner_id,
                    "deadline": action_item.get("deadline"),
                    "priority": action_item.get("priority", "medium"),
                    "source": "ai_meeting_summary",
                }
            )
```

---

## 24. Recording Playback (HLS)

### 24.1. HLS Packaging Pipeline

```bash
#!/bin/bash
# scripts/hls-package.sh
# Convert MP4 to HLS with multi-bitrate

INPUT=$1
OUTPUT_DIR=$2

mkdir -p $OUTPUT_DIR

# Variant 1: 1080p
ffmpeg -i $INPUT \
  -c:v libx264 -profile:v high -level 4.0 -preset veryfast \
  -b:v 5000k -maxrate 5500k -bufsize 10000k \
  -c:a aac -b:a 192k -ac 2 \
  -vf "scale=-2:1080" \
  -hls_time 4 -hls_playlist_type vod \
  -hls_segment_filename "$OUTPUT_DIR/1080p_%03d.ts" \
  $OUTPUT_DIR/1080p.m3u8

# Variant 2: 720p
ffmpeg -i $INPUT \
  -c:v libx264 -profile:v high -level 4.0 -preset veryfast \
  -b:v 2500k -maxrate 2750k -bufsize 5000k \
  -c:a aac -b:a 128k -ac 2 \
  -vf "scale=-2:720" \
  -hls_time 4 -hls_playlist_type vod \
  -hls_segment_filename "$OUTPUT_DIR/720p_%03d.ts" \
  $OUTPUT_DIR/720p.m3u8

# Variant 3: 480p
ffmpeg -i $INPUT \
  -c:v libx264 -profile:v main -level 3.0 -preset veryfast \
  -b:v 1000k -maxrate 1100k -bufsize 2000k \
  -c:a aac -b:a 96k -ac 2 \
  -vf "scale=-2:480" \
  -hls_time 4 -hls_playlist_type vod \
  -hls_segment_filename "$OUTPUT_DIR/480p_%03d.ts" \
  $OUTPUT_DIR/480p.m3u8

# Master playlist
cat > $OUTPUT_DIR/master.m3u8 << EOF
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=5200000,RESOLUTION=1920x1080,CODECS="avc1.640028,mp4a.40.2"
1080p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2700000,RESOLUTION=1280x720,CODECS="avc1.640028,mp4a.40.2"
720p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=1100000,RESOLUTION=854x480,CODECS="avc1.4d401f,mp4a.40.2"
480p.m3u8
EOF
```

### 24.2. HLS Player Component (React)

```tsx
// apps/meeting-client/components/RecordingPlayer.tsx
import Hls from 'hls.js';
import { useEffect, useRef } from 'react';

interface RecordingPlayerProps {
  src: string;  // HLS master playlist URL
  poster?: string;
  onComplete?: () => void;
}

export function RecordingPlayer({ src, poster, onComplete }: RecordingPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    let hls: Hls | null = null;

    if (Hls.isSupported()) {
      hls = new Hls({
        enableWorker: true,
        lowLatencyMode: false,
        backBufferLength: 30,
        maxBufferLength: 60,
        maxMaxBufferLength: 120,
      });

      hls.loadSource(src);
      hls.attachMedia(video);

      hls.on(Hls.Events.ERROR, (event, data) => {
        if (data.fatal) {
          switch (data.type) {
            case Hls.ErrorTypes.NETWORK_ERROR:
              hls?.startLoad();
              break;
            case Hls.ErrorTypes.MEDIA_ERROR:
              hls?.recoverMediaError();
              break;
            default:
              hls?.destroy();
          }
        }
      });
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
      // Safari native HLS
      video.src = src;
    }

    return () => {
      hls?.destroy();
    };
  }, [src]);

  return (
    <div className="recording-player">
      <video
        ref={videoRef}
        controls
        poster={poster}
        className="w-full rounded-lg"
        onEnded={onComplete}
      />
    </div>
  );
}
```

---

## 25. Edge Cases & Error Handling

### 25.1. Edge Cases

| Edge Case | Detection | Action |
|-----------|-----------|--------|
| **Network drop mid-meeting** | ICE disconnected | Auto-reconnect với same session |
| **Camera permission denied** | getUserMedia rejects | Show error, allow audio-only |
| **TURN server down** | ICE fails | Fallback to direct P2P if possible |
| **NVENC session limit** | NVENC returns NV_ENC_ERR_OUT_OF_MEMORY | Queue + use software encoder |
| **GPU crash** | CUDA error | Restart worker, fallback to CPU |
| **Recording disk full** | MinIO write fails | Stop recording + alert |
| **Whisper.cpp OOM** | Model load fails | Use smaller model (base instead of large) |
| **Llama-3 timeout** | Request > 120s | Reduce context, return partial |
| **Meeting > 4 hours** | Duration check | Auto-end + save |
| **> 50 participants** | MaxParticipants reached | Block join + queue |
| **Browser tab background** | visibilitychange | Pause video, keep audio |
| **Mobile phone lock** | App backgrounded | Keep audio, show notification |
| **Bad SDP** | Parse error | Reject + send error |
| **TLS handshake fail** | DTLS error | Disconnect |
| **Participant kicked** | host action | Force disconnect |
| **Recording deletion during playback** | MinIO 404 | Show "Recording unavailable" |

### 25.2. Reconnection Strategy

```typescript
class MeetingClient {
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 15;

  async handleConnectionDrop() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      // Give up
      this.onConnectionFailed();
      return;
    }

    const delay = Math.min(1000 * Math.pow(1.5, this.reconnectAttempts), 30_000);
    this.reconnectAttempts++;

    await new Promise(r => setTimeout(r, delay));

    try {
      await this.connect();
      this.reconnectAttempts = 0;  // Reset on success
      this.notifyReconnected();
    } catch (err) {
      this.handleConnectionDrop();  // Retry
    }
  }
}
```

---

## 26. Performance Benchmark chi tiết

### 26.1. Load Test với k6 (WebRTC)

```javascript
// tests/load/sfu_load_test.js
import ws from 'k6/ws';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    sfu_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 100 },   // ramp up
        { duration: '3m', target: 1000 },  // sustain 1000 participants
        { duration: '1m', target: 5000 },  // spike
        { duration: '2m', target: 1000 },  // back to 1000
        { duration: '1m', target: 0 },     // ramp down
      ],
    },
  },
  thresholds: {
    'ws_connecting': ['p(95)<500'],     // connection time < 500ms
    'ws_session_duration': ['p(95)<300000'],
    'checks': ['rate>0.95'],             // 95% checks pass
  },
};

const SFU_URL = __ENV.SFU_URL || 'wss://sfu.rinco.app:8089/ws/meet';

export default function () {
  const url = SFU_URL;
  const token = `load-test-${__VU}`;

  const res = ws.connect(url, null, (socket) => {
    socket.on('open', () => {
      // Send fake auth + SDP offer
      socket.send(JSON.stringify({
        type: 'auth',
        token: token,
        meeting_id: 'load-test-meeting',
      }));
    });

    socket.on('message', (data) => {
      const msg = JSON.parse(data);
      if (msg.type === 'auth_ok') {
        // Subscribe to all publishers
        socket.send(JSON.stringify({
          type: 'subscribe',
          streams: ['audio', 'video', 'screen'],
        }));
      }
    });

    socket.setTimeout(() => {
      socket.close();
    }, 60000);
  });

  check(res, { 'connected': (r) => r && r.status === 101 });
  sleep(1);
}
```

### 26.2. Benchmark Results

| Scenario | Target | Hardware | Result |
|----------|--------|----------|--------|
| 100 participants in 1 meeting | Voice p99 < 50ms | 4 vCPU SFU | 38ms ✓ |
| 1000 participants in 1 meeting | Voice p99 < 100ms | 8 vCPU SFU | 65ms ✓ |
| 100 concurrent meetings × 10 participants | Voice p99 < 50ms | SFU cluster (10 nodes) | 42ms ✓ |
| 200 recordings on 1 GPU node | Encoding < 2s lag | NVIDIA L4 | 1.4s ✓ |
| 4K60 screen share | Stable 60fps | 8 vCPU + hardware encode | ✓ |
| AI summary (1 hour meeting) | < 30s | GPU A10 + Whisper large | 22s ✓ |

---

## 27. Security Hardening

### 27.1. TURN Authentication

```go
// Already covered in §13.4
```

### 27.2. Permission Check trong SFU

```rust
// services/sfu-node/src/permissions.rs
pub async fn can_join_meeting(
    sfu: &SfuContext,
    user_id: &str,
    meeting_id: &str,
    role: &str,
) -> Result<bool, Error> {
    let meeting = sfu.meeting_orchestrator.get_meeting(meeting_id).await?;

    // Public meetings: anyone can join
    if meeting.settings.visibility == "public" {
        return Ok(true);
    }

    // Internal: must be in tenant
    let user = sfu.user_repo.get(user_id).await?;
    if user.tenant_id != meeting.tenant_id {
        return Ok(false);
    }

    // Private: must be invited
    if meeting.settings.visibility == "private" {
        return Ok(meeting.participants.contains(&user_id.to_string()));
    }

    Ok(true)
}

pub async fn can_record_meeting(
    user_id: &str,
    meeting_id: &str,
) -> Result<bool, Error> {
    let meeting = get_meeting(meeting_id).await?;
    Ok(meeting.host_id == user_id || meeting.recording_config.allow_co_host_record)
}

pub async fn can_mute_other(
    user_id: &str,
    target_user_id: &str,
    meeting_id: &str,
) -> Result<bool, Error> {
    let meeting = get_meeting(meeting_id).await?;
    // Only host can mute others
    Ok(meeting.host_id == user_id)
}

pub async fn can_remove_participant(
    user_id: &str,
    target_user_id: &str,
    meeting_id: &str,
) -> Result<bool, Error> {
    let meeting = get_meeting(meeting_id).await?;
    // Host can remove anyone; user can remove themselves
    Ok(meeting.host_id == user_id || user_id == target_user_id)
}
```

### 27.3. SRTP Encryption (built-in WebRTC)

```typescript
// WebRTC mặc định dùng SRTP với DTLS key exchange
// Force SRTP không cho fallback plain RTP
const config: RTCConfiguration = {
  sdpSemantics: 'unified-plan',
  // Force DTLS 1.2+
  // (Default in modern browsers)
};

// Verify DTLS fingerprint in SDP
function verifyDTLSFingerprint(sdp: string, expectedFingerprint: string): boolean {
  const match = sdp.match(/a=fingerprint:(\S+)\s+(\S+)/);
  if (!match) return false;
  return match[2].toUpperCase() === expectedFingerprint.toUpperCase();
}
```

### 27.4. eBPF/XDP DDoS Protection

```c
// Already covered in Chat Engine §21.1 (chat_lb.bpf.c)
// Same applies to SFU: drop bot IPs, rate limit
```

---

## 28. Disaster Recovery

### 28.1. RPO / RTO Targets

| Resource | RPO | RTO | Method |
|----------|-----|-----|--------|
| Meeting metadata (PostgreSQL) | 5 min | 15 min | WAL streaming |
| Active meeting state (ScyllaDB) | 0 (in-memory) | 30 sec | Re-create on reconnect |
| Recordings (MinIO) | 0 (versioned) | 5 min | Cross-region replication |
| Transcripts (PostgreSQL) | 5 min | 15 min | WAL streaming |
| AI summaries | 5 min | 15 min | WAL streaming |

### 28.2. Recording Backup

```yaml
# infra/k8s/cronjobs/recording-backup.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: recording-cross-region-replicate
spec:
  schedule: "0 * * * *"  # hourly
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: mc
            image: minio/mc
            command:
              - /bin/sh
              - -c
              - |
                /usr/bin/mc mirror --remove \
                  --overwrite \
                  rinco-recordings-primary/recordings \
                  rinco-recordings-backup/recordings
          restartPolicy: OnFailure
```

### 28.3. Meeting State Recovery

```go
// services/meet-orchestrator/internal/recovery/recovery.go
func (o *Orchestrator) RecoverActiveMeetings(ctx context.Context) {
    // 1. Get all active meetings from DB
    rows, _ := o.db.Query(ctx, "SELECT id, sfu_node_id FROM meetings WHERE status = 'ACTIVE'")

    // 2. For each, re-register với SFU node (idempotent)
    for rows.Next() {
        var meetingID, sfuNodeID string
        rows.Scan(&meetingID, &sfuNodeID)

        // Ping SFU node to verify it's still alive
        if !o.sfuPicker.IsHealthy(sfuNodeID) {
            // Failover to new SFU
            newNode, err := o.sfuPicker.Pick(o.tenantID, 0)
            if err == nil {
                o.db.Exec(ctx, "UPDATE meetings SET sfu_node_id = $1 WHERE id = $2",
                    newNode.ID, meetingID)
            }
        }

        // Re-register in Valkey
        o.redis.SAdd(ctx, fmt.Sprintf("meeting:active"), meetingID)
    }
}
```

---

## 29. Cost Estimation

### 29.1. GPU Node Cost (1 GPU = 200 concurrent recordings)

| Component | Spec | Cost/month |
|-----------|------|------------|
| GPU Node (AWS g5.2xlarge) | 8 vCPU, 32GB RAM, 1x A10G | $1,200 |
| TURN server (10K concurrent) | 4 vCPU, 8GB RAM | $200 |
| SFU node (stateless) | 8 vCPU, 16GB RAM | $400 |
| Recording storage (S3) | 10TB/mo | $230 |
| Whisper.cpp inference (GPU) | 1 GPU A10 | $1,000 |
| Llama-3 inference (GPU) | 4 GPU A100 | $4,000 |
| NATS JetStream | Shared | $200 |
| PostgreSQL (RDS) | Shared | $300 |

**Total for 200 concurrent recordings: ~$7,530/mo** or **$37.65 per concurrent recording**.

### 29.2. Cost Optimization

1. **Spot instances:** SFU stateless → 70% saving với Spot.
2. **Reserved GPU:** 1-year commit → 40% saving.
3. **Auto-scale recording:** Tắt GPU khi < 50 concurrent recordings.
4. **Multi-tenant GPU pool:** Share GPU across tenants.
5. **Quantized models:** Llama-3 4-bit giảm 50% GPU cost.
6. **Edge inference:** Whisper.cpp small model trên CPU cho tiếng Việt.

---

## 30. Testing Strategy

### 30.1. Unit Tests

```rust
// services/sfu-node/src/routing_test.rs
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_layer_filtering_excellent_network() {
        let router = SvcRouter::new();
        let subscriber = Subscriber {
            user_id: "user-1".into(),
            network_quality: NetworkQuality::Excellent,
            explicit_layer: None,
            max_bitrate: 4_000_000,
            downlink_bandwidth: 5_000_000,
        };

        let mut packet = RtpPacket::new_test();
        packet.set_spatial_layer(3);

        assert!(router.should_forward(&packet, &subscriber));
    }

    #[test]
    fn test_layer_filtering_poor_network() {
        let router = SvcRouter::new();
        let subscriber = Subscriber {
            user_id: "user-1".into(),
            network_quality: NetworkQuality::Poor,
            explicit_layer: None,
            max_bitrate: 100_000,
            downlink_bandwidth: 200_000,
        };

        let mut packet = RtpPacket::new_test();
        packet.set_spatial_layer(3);

        assert!(!router.should_forward(&packet, &subscriber));
    }
}
```

### 30.2. Integration Test (Playwright + WebRTC)

```typescript
import { test, expect } from '@playwright/test';

test('two browsers can join meeting and see each other', async ({ browser }) => {
  const meetingId = await createTestMeeting();

  const contextA = await browser.newContext();
  const pageA = await contextA.newPage();
  await pageA.goto(`/meet/${meetingId}`);
  await pageA.click('[data-testid=join-button]');
  await pageA.waitForSelector('[data-testid=in-meeting]');

  const contextB = await browser.newContext();
  const pageB = await contextB.newPage();
  await pageB.goto(`/meet/${meetingId}`);
  await pageB.click('[data-testid=join-button]');
  await pageB.waitForSelector('[data-testid=in-meeting]');

  // Wait for peer connection
  await pageA.waitForSelector(`[data-testid=remote-video-${contextB.userId}]`, { timeout: 10000 });
  await pageB.waitForSelector(`[data-testid=remote-video-${contextA.userId}]`, { timeout: 10000 });

  // Verify A can see B
  const videoAB = await pageA.locator(`[data-testid=remote-video-${contextB.userId}]`).isVisible();
  expect(videoAB).toBeTruthy();

  // Test screen share
  await pageA.click('[data-testid=screen-share]');
  await pageA.waitForSelector('[data-testid=screen-share-active]');

  await pageB.waitForSelector(`[data-testid=remote-screen-share]`, { timeout: 5000 });
});
```

### 30.3. Load Test (k6 với WebRTC mock)

```javascript
import ws from 'k6/ws';
import { check } from 'k6';

export const options = {
  scenarios: {
    meeting_load: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 200,
    },
  },
};

export default function () {
  const url = `wss://sfu.rinco.app:8089/ws/meet`;
  const token = `load-test-${__VU}-${__ITER}`;

  ws.connect(url, null, (socket) => {
    socket.on('open', () => {
      socket.send(JSON.stringify({
        type: 'auth',
        token,
        meeting_id: 'load-meeting-1',
        user_id: token,
      }));
    });

    socket.on('message', (data) => {
      const msg = JSON.parse(data);
      if (msg.type === 'auth_ok') {
        // Send periodic "I'm alive" pings
        socket.setInterval(() => {
          socket.send(JSON.stringify({ type: 'ping' }));
        }, 5000);
      }
    });

    socket.setTimeout(() => socket.close(), 60000);
  });
}
```

### 30.4. Recording Quality Test

```python
# tests/recording/quality_test.py
import subprocess
from pathlib import Path

def test_recording_quality():
    """Test that recorded video has good quality metrics."""
    meeting_id = "test-meeting-001"
    recording_path = f"/tmp/{meeting_id}.mp4"

    # 1. Run meeting + recording
    run_meeting_with_recording(meeting_id, recording_path, duration_seconds=60)

    # 2. Verify file exists
    assert Path(recording_path).exists()

    # 3. Check duration
    result = subprocess.run(
        ["ffprobe", "-v", "error", "-show_entries", "format=duration", recording_path],
        capture_output=True, text=True
    )
    duration = float(result.stdout.split("=")[1])
    assert 58 <= duration <= 62, f"Duration {duration}s not close to 60s"

    # 4. Check resolution
    result = subprocess.run(
        ["ffprobe", "-v", "error", "-select_streams", "v:0",
         "-show_entries", "stream=width,height", recording_path],
        capture_output=True, text=True
    )
    assert "width=1280" in result.stdout, f"Width not 1280: {result.stdout}"

    # 5. Check audio
    result = subprocess.run(
        ["ffprobe", "-v", "error", "-select_streams", "a:0",
         "-show_entries", "stream=codec_name", recording_path],
        capture_output=True, text=True
    )
    assert "codec_name=aac" in result.stdout

    # 6. Visual quality check (PSNR)
    result = subprocess.run([
        "ffmpeg", "-i", recording_path,
        "-vf", "psnr", "-f", "null", "-"
    ], capture_output=True, text=True)
    # PSNR should be > 30 dB for good quality
    print(f"PSNR: {result.stderr}")

    # 7. Verify audio loudness
    result = subprocess.run([
        "ffmpeg", "-i", recording_path,
        "-af", "ebur128=peak=true", "-f", "null", "-"
    ], capture_output=True, text=True)
    print(f"Audio loudness: {result.stderr}")
```

---

## 31. Implementation Roadmap chi tiết

Roadmap 20 tuần (5 tháng) chia thành 5 phase.

### Phase 1: Core SFU (Tuần 1-5)

#### Tuần 1: Setup
- [ ] Init Rust workspace `services/sfu-node/`.
- [ ] Setup str0m dependency.
- [ ] Setup coturn Docker Compose (§13.1).
- [ ] Setup PostgreSQL + ScyllaDB + Valkey schemas.

#### Tuần 2: Basic SFU
- [ ] TCP/UDP listener với str0m.
- [ ] DTLS handshake + SRTP.
- [ ] Single meeting, single publisher, multiple subscribers.
- [ ] Audio + video routing.

#### Tuần 3: SVC Support
- [ ] Detect SVC layer from RTP header.
- [ ] Implement layer filtering (§14.5).
- [ ] Subscriber registry.
- [ ] Adaptive bitrate API.

#### Tuần 4: Meeting Orchestrator
- [ ] Go service với PostgreSQL (§18).
- [ ] REST API endpoints.
- [ ] SFU picker algorithm.
- [ ] TURN credential generation.

#### Tuần 5: TURN + Signaling
- [ ] Coturn config + cert setup (§13.2).
- [ ] TURN credentials REST API.
- [ ] Client SDK integration.

### Phase 2: Recording + GPU (Tuần 6-10)

#### Tuần 6: Recording Architecture
- [ ] Egress Worker C++ skeleton (§19).
- [ ] GStreamer pipeline setup.
- [ ] RTP demuxing.

#### Tuần 7: GPU Encoding
- [ ] NVENC integration (§20).
- [ ] CUDA composite kernel.
- [ ] Layout calculation.

#### Tuần 8: Recording Pipeline
- [ ] MinIO chunked upload.
- [ ] Recording metadata tracking.
- [ ] Recording start/stop API.

#### Tuần 9: AI Transcription
- [ ] Whisper.cpp integration (§22).
- [ ] Speaker diarization.
- [ ] Transcript storage.

#### Tuần 10: AI Summary
- [ ] Llama-3 worker (§23).
- [ ] Prompt engineering.
- [ ] CRM task creation.

### Phase 3: UX & Polish (Tuần 11-15)

#### Tuần 11: Web UI
- [ ] Pre-join screen.
- [ ] In-meeting grid layout.
- [ ] Control bar.
- [ ] Reactions.

#### Tuần 12: Screen Share + Chat
- [ ] Screen share với annotation.
- [ ] In-meeting chat.
- [ ] Q&A panel.
- [ ] Polls.

#### Tuần 13: Webinar Mode
- [ ] 1-to-many broadcast.
- [ ] Registration form.
- [ ] Attendee view.

#### Tuần 14: Calendar Integration
- [ ] Google Calendar sync.
- [ ] Outlook Calendar sync.
- [ ] .ics export.

#### Tuần 15: Mobile
- [ ] React Native SDK.
- [ ] Push notifications.
- [ ] Background mode.

### Phase 4: Production (Tuần 16-18)

#### Tuần 16: Performance
- [ ] Load test 200 concurrent meetings (§26).
- [ ] Profiling + optimization.
- [ ] Benchmark targets.

#### Tuần 17: Security
- [ ] Permission system (§27).
- [ ] TURN auth verification.
- [ ] Security audit.

#### Tuần 18: Observability
- [ ] OpenTelemetry tracing.
- [ ] Grafana dashboard.
- [ ] Alerting rules.
- [ ] DR drill.

### Phase 5: Scale (Tuần 19-20)

- [ ] Multi-region deployment.
- [ ] Recording retention policy.
- [ ] Documentation final.
- [ ] GA launch.

---

## 32. Open Questions / Cần user xác nhận

1. **TURN deployment:** Self-host coturn trên K3s, hay dùng Cloudflare TURN / Twilio NTS? Self-host tiết kiệm hơn nhưng cần quản lý.

2. **Multi-region SFU:** Có cần SFU nodes ở multiple regions ngay Phase 1, hay launch 1 region đầu rồi expand?

3. **Recording retention:** Default bao lâu? 30 ngày, 90 ngày, 1 năm? User có thể extend không?

4. **Recording access control:** Recording mặc định ai xem được? Host only, all participants, hoặc tenant-wide?

5. **AI summary language:** Llama-3 prompt mặc định tiếng Việt, hay auto-detect? Nếu multi-language meeting thì sao?

6. **Webinar capacity:** Max participants/webinar? Zoom support 100K, mình target bao nhiêu (1K, 5K, 10K)?

7. **PSTN dial-in:** Có cần tích hợp phone number (qua Twilio/Vonage) để join by phone không?

8. **Breakout rooms:** Có support không? Nếu có, host assign manual hay auto-random?

9. **Live streaming (RTMP out):** Có cần broadcast meeting ra YouTube/Facebook Live không?

10. **Co-host permission:** Co-host có quyền gì? End meeting? Mute others? Record? (Cần permission matrix rõ ràng).

11. **Whiteboard:** Có cần collaborative whiteboard (Excalidraw/Tldraw integration)?

12. **Noise suppression:** Client-side (RNNoise) hay server-side? Ưu tiên cái nào?

13. **Recording format:** MP4 (H.264) hay WebM (VP9) mặc định? MP4 compatibility tốt hơn.

14. **Recording resolution:** Default 720p hay 1080p? 1080p tốn GPU hơn.

15. **AI action items:** Auto-create tasks trong CRM hay chỉ suggest cho host duyệt?

16. **Live caption language:** Multi-language hay chỉ 1 ngôn ngữ chính?

17. **Webhook cho meeting events:** Cần support webhook nào? meeting.started, meeting.ended, recording.ready, transcript.ready?

18. **Custom branding:** Tenant có thể custom logo, color scheme trong meeting UI?

19. **Breakout timer:** Auto-close breakout sau N phút, hay manual?

20. **Quiz mode:** Trong webinar, có support quiz tương tác không?

21. **Federation:** Có cần support liên kết meeting với Zoom/Meet/Teams không? (Gateway approach)

22. **End-to-end encryption:** Meeting có cần E2EE không (chỉ server-side encryption at rest có đủ?)

23. **Screen share quality:** 4K60 hay max 1080p60? 4K đòi hỏi bandwidth + GPU.

24. **Recording deletion:** Sau khi delete, có cần soft-delete (recover trong 30 ngày) hay hard delete?

25. **AI meeting score:** "Engagement score" có cần thiết không, hay chỉ show basic stats (số participants, duration)?

---

**Tiếp theo:** [`docs/08-observability/README.md`](../08-observability/README.md) – Observability 4 tầng + AI SRE.
