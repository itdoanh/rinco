# WebRTC SFU – AV1/VP9 SVC, SRTP forwarder, bandwidth adaptation

Selective Forwarding Unit for real-time audio/video meetings. Routes RTP
packets between participants **without decoding media** (zero-transcoding),
applies **SVC layer filtering** to match each subscriber's bandwidth
budget, and exposes a **Connect-RPC `RoomService`** for internal use by
orchestrators and the meeting UI.

## Highlights

- **Zero-transcoding forwarder** (`sfu::forwarder`): packets are forwarded
  to subscribers in their original encoding. SVC layer switching is
  applied by inspecting the RTP header (spatial + temporal layer).
- **SVC routing**: per-subscription ceiling (`max_spatial_layer`,
  `max_temporal_layer`); higher layers are dropped instead of forwarded.
- **Congestion control** (`sfu::congestion`): delay-based estimator
  inspired by Google Congestion Control (GCC) – additive increase when
  delays drop, multiplicative decrease when delays rise.
- **Egress** (`sfu::egress`): pluggable `EgressSink` trait. Default
  implementations include a no-op `NullEgress` for tests and a
  `BufferEgress` for unit tests. Production wires a Connect-RPC client
  to `recording-service`.
- **Signaling** (`signaling`): JSON-over-WebSocket protocol with explicit
  SDP offer/answer + ICE candidate exchange.
- **Room management** (`room`): create/join/leave, broadcast room events
  to subscribers, periodic cleanup of empty rooms.
- **Safety** (`safety`): kick, mute, force-end, in-memory ban list
  (`BanList`).

## Module layout

```
src/
├── main.rs                  # Bootstrap + axum + Connect-RPC mount
├── lib.rs                   # Re-exports + SfuContext
├── config.rs                # Env-driven configuration
├── error.rs                 # SfuError + SfuResult
├── observability.rs         # OTel + Prometheus
├── signaling.rs             # WebSocket signaling protocol
├── room.rs                  # Room / Participant / RoomManager
├── peer.rs                  # PeerConnection wrapper (codec prefs)
├── sfu/
│   ├── forwarder.rs         # Zero-transcoding SRTP forwarder
│   ├── congestion.rs        # Bandwidth estimator
│   └── egress.rs            # EgressSink trait + impls
├── api/room_service.rs      # Connect-RPC RoomService
└── safety/ejection.rs       # Kick / mute / ban
```

## Endpoints

| Path                                | Method | Purpose                                       |
|-------------------------------------|--------|-----------------------------------------------|
| `/healthz`                          | GET    | Liveness                                      |
| `/readyz`                           | GET    | Readiness                                     |
| `/metrics`                          | GET    | Prometheus exposition                         |
| `/v1/ws?room_id=…&user_id=…`        | GET    | WebSocket signaling                           |
| `/v1/ws/{room_id}/{user_id}`        | GET    | WebSocket signaling (path)                    |
| `/v1/rooms`                         | POST   | Create room (Connect-RPC)                     |
| `/v1/rooms`                         | GET    | List rooms                                    |
| `/v1/rooms/{id}`                    | GET    | Room detail                                   |
| `/v1/rooms/{id}`                    | DELETE | Force end room                                |
| `/v1/rooms/{id}/eject/{user_id}`    | POST   | Kick participant                              |

## Connect-RPC `RoomService`

| RPC                | Request                | Response             |
|--------------------|------------------------|----------------------|
| CreateRoom         | CreateRoomRequest      | CreateRoomResponse   |
| GetRoom            | GetRoomRequest         | GetRoomResponse      |
| JoinRoom           | JoinRoomRequest        | JoinRoomResponse     |
| ListRooms          | ListRoomsRequest       | ListRoomsResponse    |
| UpdateRoomState    | UpdateRoomStateRequest | Empty                |
| StartRecording     | StartRecordingRequest  | Empty                |
| StopRecording      | Uuid                   | Empty                |

## WebSocket protocol

```jsonc
// Client → Server
{"type": "join",  "room_id": "...", "user_id": "...", "tenant_id": "...", "display_name": "Alice"}
{"type": "offer", "sdp": "v=0..."}
{"type": "answer","sdp": "v=0..."}
{"type": "ice",   "candidate": "...", "sdp_mid": "0", "sdp_mline_index": 0}
{"type": "mute",  "audio": false, "video": true, "screen": false}
{"type": "leave"}
{"type": "ping"}

// Server → Client
{"type": "ready", "room_id": "...", "participants": [...], "ice_servers": [...]}
{"type": "answer","sdp": "..."}
{"type": "peer_joined", "participant": {...}}
{"type": "peer_left",   "user_id": "..."}
{"type": "recording",   "on": true}
{"type": "room_closed"}
{"type": "pong"}
```

## Environment variables

| Key                       | Default                  |
|---------------------------|--------------------------|
| `SFU_HTTP_ADDR`           | `0.0.0.0:8084`           |
| `SFU_PUBLIC_IP`           | (none)                   |
| `SFU_ICE_SERVERS`         | (one Google STUN server) |
| `SFU_MAX_PEERS`           | `500`                    |
| `SFU_MAX_BW_KBPS`         | `4000`                   |
| `SFU_OTLP_ENDPOINT`       | `http://otel-collector:4317` |
| `SFU_ENABLE_SVC`          | `true`                   |
| `SFU_ENABLE_SIMULCAST`    | `false`                  |
| `SFU_EGRESS_ENDPOINT`     | (recording-service)      |

## Capacity

- 1 SFU node handles **~500 simultaneous peers** (limited by outbound
  bandwidth and the configured `max_peers_per_room`).
- Per-room SVC layer routing lets a room mix HD and SD participants
  without forcing a single global bitrate.
- ICE trickling supported (candidates sent individually over the WS).

## Build

```bash
cargo build --release
cargo test
```

## Commit

`feat(webrtc-sfu): SFU với AV1/VP9 SVC, SRTP forwarder, bandwidth adaptation`
