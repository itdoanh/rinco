# WebRTC SFU Service

Selective Forwarding Unit cho WebRTC video calls, viết bằng Rust với thư viện `webrtc`.

## Tính năng

- **WebRTC Media Routing**: Forward video/audio streams giữa participants
- **Room Management**: Tạo, join, leave rooms
- **Participant Tracking**: Quản lý audio/video states
- **SDP/ICE Signaling**: Standard WebRTC signaling
- **Screen Sharing**: Hỗ trợ chia sẻ màn hình
- **Mute/Unmute**: Audio/video mute control
- **Multi-party**: Nhiều participants trong cùng room
- **Scalable Video Coding (SVC)**: AV1/VP9 với temporal/spatial scalability
- **GPU Acceleration**: NVIDIA NVENC cho recording (optional)

## Công nghệ

- **Language**: Rust 1.75+
- **Framework**: Axum
- **WebRTC**: webrtc-rs crate
- **Signaling**: WebSocket
- **State**: DashMap (concurrent hashmap)

## API Endpoints

### HTTP/GraphQL
```
POST   /graphql                                - GraphQL API
```

### WebSocket
```
WS     /ws/signaling                           - WebRTC signaling
```

### GraphQL Schema

```graphql
type RoomInfo {
  id: UUID!
  name: String!
  tenant_id: UUID!
  participant_count: Int!
  created_at: DateTime!
}

type JoinResult {
  success: Boolean!
  error: String
  participants: [Participant!]!
}

type Query {
  roomInfo(roomId: UUID!): RoomInfo
  activeRooms: [RoomInfo!]!
}

type Mutation {
  createRoom(input: CreateRoomInput!): RoomInfo!
  joinRoom(input: JoinRoomInput!): JoinResult!
  leaveRoom(roomId: UUID!, userId: UUID!): Boolean!
}
```

## Signaling Protocol

### WebSocket Messages

```json
{
  "type": "Join",
  "room_id": "uuid",
  "user_id": "uuid",
  "display_name": "Alice"
}
```

```json
{
  "type": "SDP",
  "room_id": "uuid",
  "from": "uuid",
  "to": "uuid",
  "sdp": {
    "sdp": "v=0\r\no=- ...",
    "video_codec": "VP9",
    "audio_codec": "opus"
  }
}
```

```json
{
  "type": "ICE",
  "room_id": "uuid",
  "from": "uuid",
  "to": "uuid",
  "candidate": {
    "candidate": "candidate:...",
    "sdp_mid": "0",
    "sdp_mline_index": 0
  }
}
```

## Architecture

```
Client A ←→ SFU ←→ Client B
              ↓
           Client C
              ↓
           Client D
```

- Each client establishes 1 peer connection with SFU
- SFU forwards media streams between all participants
- Selective forwarding based on subscription (who wants whose video)

## Performance

- **Concurrent participants**: 100+ per instance
- **Bandwidth**: ~2.5Mbps/participant (720p VP9)
- **Latency**: < 100ms media path
- **CPU**: ~30% per 100 participants @ 720p

## Environment Variables

```bash
WEBRTC_SFU_PORT=8082
RUST_LOG=info
STUN_SERVER=stun:stun.l.google.com:19302
TURN_SERVER=turn:turn.example.com:3478
TURN_USERNAME=username
TURN_PASSWORD=password
MAX_PARTICIPANTS_PER_ROOM=100
```

## Development

```bash
cargo build --release
cargo run --release
```
