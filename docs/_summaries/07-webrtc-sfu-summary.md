# WebRTC SFU + Recording + Whisper — Summary

> Phân hệ hội nghị truyền thông của RINCO: voice call, video call, screen share, recording GPU-accelerated, và AI transcription/summary (Whisper + Llama-3). Mục tiêu **sub-50ms latency**, **zero-transcoding SFU** (không decode video), **200+ concurrent recordings per GPU**, **AI summary < 30s sau meeting**. Stack: **Rust/C++ SFU với str0m + AV1/VP9 SVC + eBPF/XDP routing + GPU NVENC composite recording + Whisper.cpp STT + Llama-3 summarization**.

Nguồn: `docs/07-webrtc-sfu/README.md` (≈ 5.800 dòng, 49 sections bao gồm hai phần mở rộng với code Rust/C++/TypeScript/Go/Python đầy đủ).

---

## 1. Triết lý kiến trúc & mục tiêu

| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| SFU-1 | Sub-50ms voice/video latency | p99 |
| SFU-2 | Zero-transcoding | SFU không decode video |
| SFU-3 | 200+ concurrent meetings per GPU node | Load test |
| SFU-4 | AI auto-summary sau meeting | < 30s |
| SFU-5 | Screen share 4K@60fps | Bandwidth adaptive |

**Nguyên tắc cốt lõi:**
- **SFU thuần routing** — không mã hóa lại; subscriber yếu mạng nhận SVC layer thấp, mạnh nhận full layer.
- **GPU composite** — record hàng trăm meeting đồng thời mà không cần Headless Chromium (tiết kiệm 95% resource).
- **AI chạy local** — Whisper.cpp + Llama-3 hoàn toàn on-prem, không gọi cloud → privacy + zero per-minute cost.

### Tổng quan
```
[Publisher (WebRTC)] ──AV1/VP9 SVC──► [Rust/C++ SFU Server]
                                          │
                          (eBPF/XDP Accelerated Routing ~100ns)
                                          │
              ┌───────────────────────────┼───────────────────────────┐
              ▼                           ▼                           ▼
      [Subscriber High]           [Subscriber Med]            [Subscriber Low]
      (Full resolution)          (Temporal Layer 1)        (Spatial Layer 0)
                                          │
                                          └─► [C++ Egress Worker + NVIDIA NVENC]
                                                       │ (composite + encode)
                                                       ▼
                                          [MinIO/S3] → [HLS playback] + [Whisper STT] + [Llama-3 summary]
```

---

## 2. Microservices

| Service | Ngôn ngữ | Vai trò | Đặc tính |
|---------|----------|--------|----------|
| `sfu-node` | Rust (str0m-based) | Stateless SFU worker, pure routing | Zero-decode, layer filter |
| `meeting-orchestrator` | Go | Meeting lifecycle, SFU picker, TURN creds, webhooks | Quản lý 1M meetings state |
| `recorder` | C++ + CUDA | GPU composite recording + NVENC encode | 200 concurrent/Node |
| `stt-worker` | Python + Whisper.cpp | Speech-to-text + speaker diarization | Real-time + post-meeting |
| `summary-worker` | Python + Llama-3 70B | Meeting summary + action items | Auto CRM task creation |

ICE servers: **Coturn self-hosted** (`stun:stun.rinco.app:3478`, `turn:turn.rinco.app:3478` TCP/TLS, TURN auth bằng **PASETO time-limited token** — HMAC-SHA1 username = `expiration_timestamp:tenant_id:user_id`, TTL 10 phút).

---

## 3. AV1/VP9 SVC Scalable Coding

### So sánh SVC vs Simulcast
| | Simulcast | SVC |
|---|-----------|-----|
| **Băng thông** | Gửi N luồng (full + med + low) | 1 luồng có N layer |
| **CPU** | Encode N lần | Encode 1 lần |
| **Bitrate** | Tổng = full + med + low | ~ bằng full |
| **Latency switch** | Phải chuyển luồng | Trong cùng 1 frame |

### Layer Structure (Spatial + Temporal)
- **Spatial**: L0 = 320×240, L1 = 640×480 (15fps), L2 = 1280×720 (30fps), L3 = 1920×1080 (60fps).
- **Temporal** within each spatial: T0, T1, T2, T3 frames @ 60fps; subscriber chọn frame nào trở đi (cho phép rewind / catch-up frame).

### Client Config Chrome (`sendEncodings` với `scalabilityMode`)
```javascript
const transceiver = pc.addTransceiver(track, {
  direction: 'sendonly',
  sendEncodings: [
    { rid: 'h', maxBitrate: 4_000_000, scalabilityMode: 'L3T3' },
    { rid: 'm', maxBitrate: 1_000_000, scalabilityMode: 'L2T2', scaleResolutionDownBy: 2 },
    { rid: 'l', maxBitrate: 250_000,   scalabilityMode: 'L1T1', scaleResolutionDownBy: 4 },
  ],
});
```

### SFU Layer Filtering (§14.5)
Subscriber có `network_quality ∈ {Poor, Fair, Good, Excellent}` map sang `max_layer ∈ {0, 1, 2, 3}`. SFU kiểm tra `rtp_packet.spatial_layer <= subscriber.max_layer` trước khi forward — pure bit decision, không decode. Có thể explicit-set `explicit_layer` hoặc runtime adjust qua `receiver.parameters.encodings[i].maxBitrate = bandwidthLimit`.

### Browser compatibility
| Browser | SVC | Max Layers |
|---------|-----|------------|
| Chrome 120+, Edge 120+ | ✅ Full AV1/VP9 | L3T3 |
| Firefox 120+ | ⚠️ Partial VP9 | L2T2 |
| Safari 17+ | ❌ None → Simulcast/H.264 fallback |

SFU detect qua User-Agent, tự route sang Simulcast cho Safari.

### Codec priority
**AV1** (ưu tiên, tiết kiệm 30% bitrate vs VP9) → VP9 SVC fallback → VP8 fallback cũ → H.264 nếu client không support AV1/VP9.

---

## 4. eBPF/XDP Routing & Load Balancing

### Vấn đề
Mỗi SFU xử lý hàng trăm ngàn RTP/SRTP packets/giây; user-space routing chậm vì syscall overhead.

### Giải pháp XDP/eBPF (§4.2)
Lập trình bằng C, attach vào NIC interface, routing decision trong kernel:
- Parse Ethernet + IPv4/UDP.
- Lookup `route_map` (BPF_MAP_TYPE_LRU_HASH, ~100K entries).
- `bpf_redirect(*ifindex, 0)` nếu match, ngược lại `XDP_PASS`.
- Bypass TCP/IP stack cho SRTP → giảm 5-15ms.

### Routing table
Maintain trong kernel map, update từ userspace khi subscriber join/leave. Auto-evict qua LRU.

### Kết hợp với DDoS protection
Cùng pattern XDP như Chat: blacklist map + token-bucket rate-limit map + stats PERCPU_ARRAY. Decision < 100ns.

---

## 5. Meeting Orchestrator (full §18)

### Meeting Lifecycle
```
[Scheduled] ─► [Starting] ─► [Active] ─► [Ending] ─► [Recording Processing] ─► [Completed]
                   │             │              │
                   │             │              └─► AI Summary, Save to MinIO
                   │             └─► Recording starts (if enabled)
                   └─► Wait for first participant
```

### Room Types (5)
| Code | Loại | Use case |
|------|------|----------|
| 1 | Instant | Tạo nhanh, không schedule |
| 2 | Scheduled | Có thời gian + calendar invite |
| 3 | Recurring | Họp định kỳ (daily/weekly) |
| 4 | Webinar | 1-nhiều, host + panelists + attendees |
| 5 | Breakout | Chia nhỏ từ meeting lớn |

### Room code
Generate `abc-defg-hij` (9 chars, alphabet + digits, dễ share) bằng `crypto/rand`.

### SFU Picker algorithm (Python full §18.2)
1. List active SFU nodes từ Valkey (heartbeat < 60s).
2. Filter by region (cùng region với tenant).
3. Filter by capacity: `n.current_load + expected_size <= n.capacity`.
4. Score: `score = (1 - current_load/capacity)*50 + (100 - cpu_percent)*0.3 + (10 nếu has_gpu)`.
5. Pick highest score.
6. `allocate(meeting_id, participants)` reserve capacity qua `HINCRBY current_load`. `release(meeting_id)` khi meeting kết thúc.

### Recording Control
- `auto_record`: Record từ đầu.
- `on_demand`: Click "Record" trong meeting.
- `cloud_only`: Chỉ cloud, không local.
- `cloud_and_local`: Cả hai.

### Webhooks (full §18.4)
HMAC-SHA256 signed (`X-Rinco-Signature`) cho tenants subscribe events `meeting.created`, `meeting.started`, `meeting.ended`, `recording.ready`, `transcript.ready`, `summary.ready`. Retry với exponential backoff, log + DLQ cho 5xx.

---

## 6. GPU Recording Pipeline — full implementation

### Kiến trúc
```
[WebRTC SFU Media Stream]
   │ (Direct RTP Pipe — không qua signaling)
   ▼
[C++/Rust Egress Worker] ──(CUDA stream)──► [GPU VRAM buffers]
       │                                       │
       │ composite_kernel<<<>>>               │
       │ (CUDA grid, 16×16 blocks)              │
       ▼                                       ▼
[NVIDIA NVENC H.264 Encoder] ──zero-copy──► [Encoded NAL units]
       │
       ▼ (multipart upload)
[MinIO/S3 chunked .m4s] → [HLS packaging] → [Playback URL]
```

### Layouts
- **Grid** NxN ô vuông auto (composite_kernel layout cho 16 streams × 1920×1080).
- **Speaker view**: 1 lớn + N nhỏ.
- **Presentation**: Screen share + thumbnails nhỏ.
- **Custom** layout qua designer.

### CUDA Composite Kernel (§19.2)
Mỗi thread kiểm tra pixel thuộc stream nào, copy từ `input_streams[i * 1920*1080*4]` → `output[y*width+x]`; background = dark gray `(30, 30, 30, 255)`. `calculate_layout` host-side tính cell width/height dựa trên `num_streams`, dùng `cols = ceil(sqrt(N))`, `cell_w = output_w / cols`.

### Egress Worker (C++ full §19.1)
Flow mỗi frame:
1. `ReceiveRTPPacket` từ SFU.
2. Decode (H.264/VP9/AV1) — recorder **phải decode** để composite (khác SFU).
3. `cudaMemcpyAsync(... cudaMemcpyHostToDevice)`.
4. Đợi đủ tất cả streams → `CompositeFrame()` lập trình GPU.
5. `EncodeWithNVENC()` (zero-copy vì GPU memory).
6. `UploadToS3()` multipart `.m4s` chunk.

Buffer GPU: 16 streams × 1920×1080×4 bytes = ~132 MB input buffer, 1920×1080×4 = ~8MB composite + 8MB output.

### Performance
**1 GPU NVIDIA L4** (60 TFLOPS, 24GB VRAM): **200+ concurrent recordings**. So với Headless Chromium (1.5GB RAM + 100% 1 core / meeting): tiết kiệm 95%.

---

## 7. NVENC Encoder chi tiết (§20)

### Session Limits & Workaround
- Consumer GPU (RTX): 3-5 concurrent NVENC sessions.
- Datacenter GPU (L4, A10): **12-16 concurrent**.
- Quadro: unlimited.
- Workaround: multiple GPUs, software encoder x264 fallback, hoặc pool NVENC sessions.

### Cấu hình chuẩn
- Codec: **H.264 High Profile** (NV_ENC_H264_PROFILE_HIGH_GUID).
- Preset: **LOW_LATENCY_HQ** (NV_ENC_PRESET_LOW_LATENCY_HQ_GUID).
- RC mode: **CBR** với `averageBitRate = bitrate, maxBitRate = bitrate * 1.2`.
- QP range: 20-36.
- GOP: `fps * 2` (2-second GOP).
- Encode width/height: 1280×720 (default) hoặc 1920×1080 nếu GPU đủ.
- Frame rate: 30fps.

### Encode Frame Flow
1. `nvEncRegisterResource` cho CUDA input (NV_ENC_INPUT_RESOURCE_TYPE_CUDAARRAY).
2. `nvEncCreateInputBuffer` ARGB format.
3. `nvEncMapInputResource`.
4. `nvEncEncodePicture` với frameIdx.
5. `nvEncCreateBitstreamBuffer` 4MB heap.
6. `nvEncLockBitstream` → `cudaMemcpy(DeviceToHost)`.
7. Cleanup destroy buffers + unregister resource.

---

## 8. Audio Mixing (§21)

### FFmpeg-based Mixer
- Per-stream `SwrContext` resampler (48000 Hz stereo FLTP).
- Mixed frame 10ms @ 48kHz = 480 samples.
- Normalize + clip gain `1.0 / num_streams`, hard limit [-1, 1].
- Encode AAC.
- Mux video thành MP4 bằng FFmpeg.
- Echo cancellation client-side, gain control.

### Storage
- Default 720p H.264 8 Mbps, AAC 192 kbps.
- Optional 1080p 5 Mbps.
- webm VP9 nếu tenant prefer.
- MinIO/S3 chunked upload, multi-region replication.
- Retention config (30/90/365 ngày) theo tenant setting.

---

## 9. AI Transcription & Summary — Whisper + Llama-3

### Pipeline
```
[Meeting End] → [Audio Extract] → [Whisper.cpp STT] → [Llama-3 Summary] → [Save to CRM]
                                     │                     │
                                     ▼                     ▼
                              transcript.json        summary.md + action_items.md
```

### Whisper.cpp Integration (§22)
- Model **large-v3** GPU (fallback CPU int8).
- `compute_type = float16` GPU.
- `beam_size = 5`.
- `vad_filter = true`, min_silence 500ms.
- `print_realtime = true` cho live caption.
- Language `auto` detection.
- Speaker diarization qua **pyannote-audio 3.1** (requires HuggingFace token).
- Sub-200ms latency real-time (GPU).

### Live Caption
Real-time, gửi qua WebSocket tới participants mỗi segment khi Whisper commit.

### Post-meeting Llama-3 Prompt Engineering (§23)
Prompt Tiếng Việt structured **JSON output** với các field:
- `summary` (2-3 câu tổng quan).
- `key_points` (5-7 bullets).
- `decisions[]` (decision, context, impact).
- `action_items[]` (task, owner, deadline, priority high/medium/low).
- `follow_ups[]`.
- `topics[]`.
- `sentiment` (positive/neutral/negative).
- `engagement_score` (0-1).
- `next_meeting_suggestion`.

Endpoint: **vLLM** serving **meta-llama/Llama-3-70b-chat-hf**, `temperature=0.3`, `max_tokens=2000`, `response_format={"type":"json_object"}`, timeout 120s.

### CRM Auto-task Creation
Sau khi summary xong, worker tự tạo task trong CRM từ `action_items` qua `POST /api/crm/v1/task` với `owner_id` lookup qua name matching, `source = "ai_meeting_summary"` để audit.

### Lưu vào CRM
- `meeting_transcripts` table — segments JSONB + full_text + model_version.
- CRM Activity — summary + action_items.
- Link với Deal/Lead liên quan.

---

## 10. Database Schema (§8)

### PostgreSQL — 4 tables
```sql
meetings (id UUID PK, tenant_id, room_code UNIQUE, type SMALLINT,
          host_id, title, description, starts_at, ends_at,
          actual_started_at, actual_ended_at,
          max_participants, recording_enabled,
          recording_storage_url, transcript_url, summary TEXT,
          action_items JSONB, status CHECK IN ('SCHEDULED','ACTIVE','ENDED','CANCELLED'),
          created_at)

meeting_participants (meeting_id FK, user_id, joined_at, left_at, role
                      ['host','co-host','panelist','attendee'])

meeting_chats (id UUID PK, meeting_id FK, sender_id, text, created_at)

recordings (id UUID PK, meeting_id FK, storage_url, format ['mp4','webm'],
            duration_seconds, size_bytes, resolution, status ['recording','processing','ready','failed'],
            started_at, ended_at)

meeting_transcripts (id UUID PK, meeting_id FK, language, segments JSONB,
                     full_text, model_version, created_at)
```

### ScyllaDB — Real-time state
```cql
rinco_meet.active_meetings (meeting_id PK, tenant_id, host_id,
                             participants list<frozen<participant>>,
                             started_at, layout, recording)
rinco_meet.sfu_nodes (node_id PK, region, endpoint, capacity,
                       current_load, last_heartbeat)
```

### Valkey — Hot state
```
meeting:room:{room_code}            → JSON{meeting_id, password?, host_id}
meeting:participants:{meeting_id}   → Set of user_ids
meeting:screen_share:{meeting_id}   → user_id (current sharer)
meeting:breakout:{meeting_id}       → JSON{parent_id, rooms: [...]}
sfu:node:{node_id}                  → capacity + heartbeat
sfu:meeting:{meeting_id}            → node_id, participants
```

---

## 11. TURN Server Setup — Coturn chi tiết (§13)

### Docker Compose
Multi-region replicas (AP-SOUTH-1 / US-EAST-1 / EU-WEST-1) với dedicated `turnserver.conf` cho mỗi region. Ports: 3478 UDP/TCP (STUN/TURN), 5349 UDP/TCP (TLS/DTLS), relay ports 49152-49200.

### Production Config (§13.2)
- `listening-port=3478`, `tls-listening-port=5349`.
- `relay-ip` + `external-ip` public.
- `lt-cred-mech` + `static-auth-secret=...` (32-byte hex).
- `realm=turn.rinco.app`.
- `no-loopback-peers`, `no-multicast-peers`, `no-cli`.
- `no-tlsv1`, `no-tlsv1_1`, `cipher-list="ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20"`.
- TLS cert + DH 2048.
- Quotas: `user-quota=12`, `total-quota=1200`, `max-allocate-timeout=60`.

### Time-limited credentials (Go §13.4)
Username = `expiration_timestamp:tenant_id:user_id`, password = `base64(HMAC-SHA1(shared_secret, username))`, TTL 10 phút. Verify lại expiry khi nhận.

### Multi-region via GeoDNS (§13.6)
```
turn.rinco.app → GeoDNS → AP-SOUTH-1 / US-EAST-1 / EU-WEST-1
                        latency routing
```

### Health Monitoring
Custom exporter parse `turn.log` → Prometheus metrics: `turn_active_sessions` (gauge), `turn_total_bytes_relayed`, `turn_auth_failures_total`, `turn_allocations_total`.

---

## 12. RTM + ICE / DTLS / SRTP Flow (§16)

```
Publisher (WebRTC Peer)
   │
   │ 1. STUN Binding Request
   ├─► STUN Server (returns public IP + port)
   │
   │ 2. ICE Candidate Exchange (host, srflx, relay)
   ├─► Signaling (WebSocket /ws/meet)
   │   via SFU orchestrator
   │
   │ 3. SDP Offer/Answer (unified-plan)
   ├─► DTLS Handshake (fingerprint verified)
   │
   │ 4. SRTP Encrypted Media (AES-128-GCM, AES-256-GCM)
   └─► Media packets (RTP) → SFU → Subscribers
```

DTLS-SRTP sử dụng fingerprint trong SDP — verify trước khi accept. SRTP không fallback plain RTP.

---

## 13. Congestion Control (§17)

### BBR cho bulk, GCC cho media
- **BBR (Bottleneck Bandwidth and Round-trip propagation time)**: dùng cho control plane traffic.
- **GCC (Google Congestion Control)**: chuyên cho WebRTC media, dựa trên delay-based detection.
- REMB, TMMBR feedback RTCP từ receiver.
- Sender adjusts bitrate theo `bwe.target_bps` mỗi 1 giây.
- Receiver rate adaptation: NACK, PLI, FIR.

---

## 14. SVC Layer Routing Logic chi tiết (§36)

```rust
pub enum NetworkQuality { Poor = 0, Fair = 1, Good = 2, Excellent = 3 }

pub fn should_forward(packet: &RtpPacket, subscriber: &Subscriber) -> bool {
    // 1. Check explicit layer override
    if let Some(explicit) = subscriber.explicit_layer {
        return packet.spatial_layer <= explicit;
    }
    // 2. Map network quality → max layer
    let max_layer = match subscriber.network_quality {
        Poor => 0,
        Fair => 1,
        Good => 2,
        Excellent => 3,
    };
    // 3. Bandwidth check
    if subscriber.max_bitrate < BITRATE_THRESHOLDS[packet.spatial_layer] {
        return false;
    }
    packet.spatial_layer <= max_layer
}
```

Bitrate thresholds: L0=250Kbps, L1=500Kbps, L2=1Mbps, L3=4Mbps.

---

## 15. Recording Playback HLS (§24)

### FFmpeg multi-bitrate packaging
3 variants: 1080p 5Mbps / 720p 2.5Mbps / 480p 1Mbps. AAC stereo 96-192kbps. Segment length 4s. VOD playlist + Master playlist chứa danh sách variants cho adaptive bitrate.

### Player React (§24.2)
- `hls.js` cho Chrome/Firefox/Edge với buffer 30s back, 60-120s forward.
- Safari native HLS qua `<video src=...>`.
- Error recovery: `hls.startLoad()` cho network error, `hls.recoverMediaError()` cho media error.
- Lazy destroy khi unmount.

### Download & trim
Download as MP4 single file. Basic editor: trim start/end, cut.

---

## 16. API Surface

### REST (§10.1)
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
POST   /api/meet/v1/join                          # Get token + SFU endpoint
GET    /api/meet/v1/meetings/:id/transcript
GET    /api/meet/v1/meetings/:id/summary
GET    /api/meet/v1/meetings/:id/action-items
GET    /api/meet/v1/recordings
GET    /api/meet/v1/recordings/:id
GET    /api/meet/v1/recordings/:id/playback        # HLS playlist
```

### WebSocket Signaling `wss://sfu.rinco.app:8089/ws/meet`
Frames: `join`, `offer`, `answer`, `ice_candidate`, `renegotiate`, `screen_share_start/stop`, `mute/unmute`, `camera on/off`, `layout_change`, `recording_status`, `chat`, `raise_hand`, `reaction`.

---

## 17. UI/UX (§11)

### Pre-join Screen
Camera preview + name input + mic/cam picker + background picker (None / Blur / Image) + **Join** button.

### In-meeting Layout
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
│             Reactions | More | Leave   │
└────────────────────────────────────────┘
```

### Components
VideoTile (avatar fallback, audio level indicator), GridLayout, SpeakerView, ControlBar, ChatPanel, ParticipantsList, Whiteboard, ScreenShareViewer, RecordingIndicator (visible), LiveCaptionOverlay, ReactionBar (raise hand, applause), PollsPanel.

### AI Summary View (§11.5)
Card hiển thị Key Points (5 bullets), Action Items (ai/deadline), Watch Recording + Download buttons.

---

## 18. Performance Benchmarks (§26)

| Scenario | Target | Hardware | Result |
|----------|--------|----------|--------|
| 100 participants / 1 meeting | Voice p99 < 50ms | 4 vCPU SFU | 38ms ✓ |
| 1000 participants / 1 meeting | Voice p99 < 100ms | 8 vCPU SFU | 65ms ✓ |
| 100 concurrent meetings × 10 participants | Voice p99 < 50ms | SFU cluster (10 nodes) | 42ms ✓ |
| 200 recordings on 1 GPU node | Encoding < 2s lag | NVIDIA L4 | 1.4s ✓ |
| 4K60 screen share | Stable 60fps | 8 vCPU + HW encode | ✓ |
| AI summary (1-hour meeting) | < 30s | GPU A10 + Whisper large | 22s ✓ |

---

## 19. Security Hardening (§27)

### Permissions trong SFU
```rust
can_join_meeting: visibility == "public" → true
                   same tenant → true
                   private → phải có trong participants
can_record_meeting: host hoặc allow_co_host_record
can_mute_other: only host
can_remove_participant: host bất kỳ ai / user chính mình
```

### SRTP + DTLS Verify
WebRTC mặc định SRTP với DTLS key exchange, force không fallback plain RTP; verify DTLS fingerprint trong SDP trước khi accept.

### eBPF/XDP DDoS
Cùng pattern Chat: blacklist IP + token bucket rate-limit + allow UA hash.

### TURN Auth
Time-limited HMAC-SHA1 credentials (TTL 10 min).

---

## 20. Disaster Recovery + Cost

### RPO/RTO (§28.1)
- Meeting metadata (PostgreSQL): RPO 5min, RTO 15min — WAL streaming.
- Active meeting state (ScyllaDB): RPO 0 in-memory, RTO 30s — re-create on reconnect.
- Recordings (MinIO): RPO 0 versioned, RTO 5min — cross-region replication qua CronJob `mc mirror` hourly.
- Transcripts / Summaries (PostgreSQL): RPO 5min, RTO 15min.

### Recovery
`RecoverActiveMeetings(ctx)` query `status = 'ACTIVE'` → ping SFU node `IsHealthy` → nếu fail thì `pick` lại node mới, update DB, re-register trong Valkey.

### Cost per Concurrent Recording (§29)
| Component | Spec | Cost/month |
|-----------|------|------------|
| GPU Node (AWS g5.2xlarge) | 8 vCPU, 32GB RAM, 1× A10G | $1,200 |
| TURN server (10K concurrent) | 4 vCPU, 8GB RAM | $200 |
| SFU node (stateless) | 8 vCPU, 16GB RAM | $400 |
| Recording storage (S3) | 10TB/mo | $230 |
| Whisper.cpp inference (GPU) | 1× A10 | $1,000 |
| Llama-3 inference (GPU) | 4× A100 | $4,000 |
| NATS JetStream shared | - | $200 |
| PostgreSQL (RDS) shared | - | $300 |
| **Total for 200 recordings** | - | **~$7,530/mo** ($37.65/concurrent) |

### Optimization
1. **Spot instances** cho SFU stateless → 70% saving.
2. **Reserved GPU** 1-year commit → 40% saving.
3. **Auto-scale recorder**: tắt GPU khi < 50 concurrent.
4. **Multi-tenant GPU pool** share.
5. **Llama-3 4-bit quantize** → 50% GPU cost.
6. **Whisper.cpp small on CPU** cho Tiếng Việt rẻ hơn.

---

## 21. Testing Strategy (§30, §45)

- **Unit (Rust cargo test)**: SVC layer routing logic, network quality decision.
- **Integration (Playwright)**: 2 browsers join meeting, verify each sees the other + screen share.
- **Load (k6)**: 100 → 1000 → 5000 VU ramp; `ws_connecting p95 < 500ms`.
- **Constant-arrival**: 100 connect/s sustained 5 phút.
- **Recording Quality (Python ffprobe)**: assert duration, resolution 1280×720, AAC audio, PSNR > 30dB, ebur128 loudness.

---

## 22. Implementation Roadmap (§31, §44)

**20 tuần = 5 phase:**

### Phase 1: Core SFU (Tuần 1-5)
1. Setup Rust workspace + str0m + Coturn Docker + DB schemas.
2. Basic SFU với str0m + DTLS/SRTP + audio/video routing.
3. SVC layer detection + filtering + adaptive bitrate.
4. Meeting Orchestrator Go + PostgreSQL + SFU picker + TURN creds.
5. Coturn config + TURN credentials REST API + client SDK.

### Phase 2: Recording + GPU (Tuần 6-10)
6. Egress Worker C++ skeleton + GStreamer.
7. NVENC + CUDA composite kernel + layout calc.
8. Recording pipeline + MinIO + metadata.
9. Whisper.cpp + pyannote diarization + transcripts.
10. Llama-3 worker + prompt engineering + CRM task creation.

### Phase 3: UX & Polish (Tuần 11-15)
11. Web UI pre-join + grid + control bar + reactions.
12. Screen share + annotation + chat + Q&A + polls.
13. Webinar mode 1-n + registration + attendee view.
14. Recording playback HLS + trim editor + share.
15. Mobile React Native + offline support.

### Phase 4: AI & Integrations (Tuần 16-18)
16. Sentiment analysis + engagement score + compliance check.
17. Calendar sync (Google/Outlook/ics) + Zoom/Meet bridge.
18. Live streaming RTMP out + cross-tenant meeting guest.

### Phase 5: Scale & Multi-region (Tuần 19-20)
19. Multi-region SFU deployment + GeoDNS.
20. Cost optimization + DR drill + GA launch.

---

## 23. 130 tính năng SFU/Meeting

### A. Voice/Video Call cơ bản (30)
1. Voice call 1-1
2. Video call 1-1
3. Group voice call
4. Group video call
5. Auto-join từ chat (click avatar)
6. Mute/unmute mic
7. Camera on/off
8. Switch camera (front/back mobile)
9. Switch mic (mobile)
10. Speaker view
11. Grid view
12. Pin participant
13. Spotlight (chỉ 1 người lớn)
14. Side-by-side
15. Picture-in-picture
16. Mini player khi rời tab
17. Background blur
18. Background virtual image
19. Background virtual video (loop mp4)
20. Beauty filter
21. Noise suppression
22. Echo cancellation
23. Auto gain control
24. Audio device picker (input/output)
25. Video device picker
26. Audio quality settings (HD/Voice)
27. Video quality settings (360p/720p/1080p)
28. Low bandwidth mode (audio-only fallback)
29. Bandwidth indicator
30. Network quality indicator (color-coded)

### B. Screen Share (20)
31. Share full screen
32. Share window
33. Share browser tab
34. Share audio (with screen)
35. Annotation tools (draw, text, arrow)
36. Laser pointer
37. Zoom in/out
38. Pause/resume share
39. Replace share (chuyển tab)
40. Multi-presenter mode (nhiều người share)
41. Remote control (request)
42. Privacy mode (blur share sensitive)
43. Highlight cursor
44. Follow cursor (remote)
45. Record screen share
46. Stream screen to recording
47. Chat side panel during share
48. Q&A panel during share
49. Reactions (raise hand, applause, emoji)
50. Pause share notification

### C. Meeting Management (25)
51. Schedule meeting
52. Recurring meeting (daily/weekly/custom)
53. Calendar invite (.ics)
54. Google Calendar sync (2-way)
55. Outlook Calendar sync (2-way)
56. Join by link
57. Join by room code
58. Join by phone (PSTN via TURN)
59. Lobby/waiting room (host admit)
60. Auto admit from same tenant
61. Knock to join (external)
62. Pre-join screen
63. Test mic/cam pre-join
64. Choose virtual background pre-join
65. Rename before join
66. Host controls (mute others)
67. Mute on entry
68. Co-host assignment
69. Transfer host
70. End meeting for all
71. Leave meeting
72. Rejoin (same session)
73. Lock meeting (no new join)
74. Password protection
75. SSO enforcement (tenant SAML)

### D. Recording (20)
76. Auto record
77. Manual record (host only)
78. Permission config to record
79. Record to cloud
80. Record to local download
81. Pause/resume recording
82. Stop recording
83. Recording notification visible
84. Recording notification hidden
85. Recording transcript
86. Live caption during meeting
87. Caption translation (live)
88. Highlight transcription (click to jump)
89. Speaker labels in transcript
90. Search transcript
91. Play recording in-app (HLS)
92. Download recording (MP4)
93. Share recording (link)
94. Recording expiry (30/90/365 days)
95. Recording access control (RBAC)
96. Trim recording (start/end)
97. Multiple recording per meeting
98. Recording reuse as video message (chat)
99. Recording analytics (views, completion rate)
100. Recording webhook (CRM auto-attach)

### E. AI Features (20)
101. AI summary (auto sau meeting)
102. AI action items extraction
103. AI sentiment analysis (positive/neutral/negative)
104. AI topic detection
105. AI speaker identification (mapping)
106. AI noise removal (Krisp integration)
107. AI live translation (caption)
108. AI meeting notes (shared Google Doc)
109. AI follow-up email auto
110. AI task creation (CRM)
111. AI meeting score (engagement 0-1)
112. AI attendee focus score
113. AI compliance check (PII/toxic detection)
114. AI custom vocabulary
115. AI profanity filter
116. AI attendance log
117. AI next meeting suggestion
118. AI availability finder (calendar bot)
119. AI reschedule optimizer
120. AI insights dashboard (analytics)

### F. Advanced (10)
121. Webinar mode 1-n
122. Webinar registration (form)
123. Webinar Q&A panel
124. Webinar poll
125. Webinar raise hand
126. Breakout rooms random
127. Breakout rooms manual
128. Breakout timer
129. Broadcast to all rooms
130. Whiteboard collaborative (canvas + cursor sync)
131. Polls live (multiple choice, rating)
132. Quiz mode (correct answer)
133. Live streaming RTMP out (YouTube/Facebook)
134. External integration Zoom/Meet bridge
135. Cross-tenant meeting guest (audit)

---

## 24. Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-SFU-01 | Voice p99 < 50ms | RTT measure |
| AC-SFU-02 | Video p99 < 100ms | RTT measure |
| AC-SFU-03 | 200 meetings per GPU node | Load test |
| AC-SFU-04 | AI summary < 30s sau meeting | E2E test |
| AC-SFU-05 | Screen share 4K60 stable | Bandwidth test |
| AC-SFU-06 | Reconnect on network drop < 3s | TURN test |

---

## 25. Open Questions cần user xác nhận (§32, §49)

1. SFU cluster pinning (per region per tenant?)
2. Recording retention mặc định (30/90/365 ngày?)
3. AI summary language mặc định (Tiếng Việt / EN / multi?)
4. Cross-tenant meeting guest cho phép hay không (audit)
5. PSTN dial-in cần integrate Twilio hay không
6. Llama-3 on-prem GPU hay vLLM cloud hosted
7. Whisper.cpp large-v3 vs faster-whisper (Python) trade-off
8. Webinar max attendees (500/1000/5000)
9. Recording storage S3 standard vs IA vs Glacier
10. Multi-region failover strategy (active-active vs active-passive)
11. Live streaming RTMP out tích hợp sẵn hay add-on
12. External bridge Zoom/Meet (có cần cho enterprise?)

---

**Tổng kết:** WebRTC SFU là phân hệ media của RINCO với kiến trúc pure-routing (zero-decode AV1/VP9 SVC), kernel-bypass SRTP routing qua eBPF/XDP, GPU-accelerated recording (200 concurrent/L4) bằng NVENC + CUDA composite kernel. AI pipeline Whisper.cpp + Llama-3 70B chạy fully on-prem cho privacy. Coturn self-hosted với time-limited HMAC-SHA1 TURN credentials, multi-region qua GeoDNS. 135 tính năng qua 6 categories. Implementation 20 tuần chia 5 phase. Chi phí ~$7.5K/tháng cho 200 concurrent recordings — tối ưu được nhờ Spot instances, reserved GPU, quantized models.
