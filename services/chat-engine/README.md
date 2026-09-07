# Chat Engine – E2EE Messenger (Signal Protocol) + ScyllaDB + Valkey

Real-time chat engine with end-to-end encryption. Implements the **Signal
Protocol** (X3DH + Double Ratchet) in pure Rust, persists ciphertext-only
messages to ScyllaDB, and uses Valkey/Redis for presence, typing
indicators, and read receipts.

## Highlights

- **End-to-end encryption** – the server only sees ciphertext. Plaintext
  bodies are produced and consumed exclusively by clients using the
  `chat_engine::crypto::signal` and `chat_engine::crypto::group` modules.
- **Signal Protocol primitives**:
  - `IdentityKeyPair` (X25519) with stable `registration_id` and SHA-256
    fingerprint.
  - `SignedPreKey` (rotates weekly) signed with the identity.
  - `OneTimePreKey` pool – replenished when below 20 keys.
  - `x3dh_initiator` / `x3dh_responder` (X3DH key agreement).
  - `RatchetState` (Double Ratchet) – forward secrecy + post-compromise
    security.
  - `ChaCha20-Poly1305` AEAD (mobile) and `AES-256-GCM` (server fallback).
- **Group Sender Keys** – symmetric chain key per member, re-distributed
  via pairwise Signal sessions on membership change.
- **At-rest crypto** – Argon2id-derived KEK wraps per-device DEK which
  seals private Signal material (`crypto::storage`).
- **Media attachment encryption** – per-file DEK (XChaCha20-Poly1305) is
  wrapped with the session message key.
- **Cypher-only persistence** – `messages_by_channel` stores ciphertext,
  `ratchet_pub`, and `msg_number`; the server cannot decrypt.
- **Replay protection** – constant-time monotonic message number checks
  (`monotonic_within_window`).

## Module layout

```
src/
├── main.rs                  # Bootstrap + axum + Connect-RPC
├── lib.rs                   # Re-exports + AppContext
├── config.rs                # Env-driven configuration
├── error.rs                 # ChatError + ChatResult
├── telemetry.rs             # OTel + Prometheus
├── api/
│   ├── types.rs             # Channel, Message, Device, Reaction, …
│   ├── channels.rs          # Channel CRUD + sender-key rotation
│   ├── messages.rs          # send / list / mark_read
│   └── devices.rs           # PreKey upload + fetch
├── crypto/
│   ├── signal.rs            # X3DH + Double Ratchet + AEAD
│   ├── group.rs             # Group sender keys
│   └── storage.rs           # KEK/DEK sealed storage
├── db/
│   ├── scylla.rs            # ScyllaDB CQL adapter (DML + DDL)
│   └── redis.rs             # Valkey/Redis adapter
├── media/encrypt.rs         # Attachment encryption
├── presence.rs              # Heartbeat + presence events
├── nats.rs                  # NATS publisher for cross-service events
└── handlers/
    ├── http.rs              # axum REST endpoints
    ├── websocket.rs         # WS upgrade + JSON protocol
    └── connect_rpc.rs       # Connect-RPC trait + default impl
```

## Connect-RPC service

Hand-written trait `handlers::connect_rpc::ChatRpc` with the following
methods (mounted on a tonic server in the binary):

| RPC                   | Request                | Response               |
|-----------------------|------------------------|------------------------|
| SendMessage           | SendMessageRequest     | Message                |
| ListMessages          | ListMessagesRequest    | Vec\<Message\>         |
| MarkRead              | MarkReadRequest        | Empty                  |
| StreamMessages        | server-streaming       | Message stream         |
| RegisterDevice        | RegisterDeviceRequest  | Device                 |
| UploadPreKeyBundle    | PreKeyBundle           | Empty                  |
| GetPreKeyBundle       | PreKeyBundleRequest    | PreKeyBundle           |
| CreateChannel         | CreateChannelRequest   | Channel                |
| AddGroupMember        | AddGroupMemberRequest  | Empty                  |
| GetPresence           | GetPresenceRequest     | Vec\<Presence\>        |
| SetTyping             | SetTypingRequest       | Empty                  |

## HTTP / WebSocket endpoints

| Path                                  | Method | Purpose                                   |
|---------------------------------------|--------|-------------------------------------------|
| `/healthz`                            | GET    | Liveness                                  |
| `/readyz`                             | GET    | Readiness (checks Scylla + Valkey)        |
| `/metrics`                            | GET    | Prometheus exposition                     |
| `/v1/ws?user_id=…`                    | GET    | WebSocket upgrade (per-user)              |
| `/v1/ws/{user_id}`                    | GET    | WebSocket upgrade (path)                  |
| `/v1/channels`                        | POST   | Create channel                            |
| `/v1/channels/{id}`                   | GET    | Channel detail                            |
| `/v1/channels/{id}/members/{user_id}` | DELETE | Remove member                             |
| `/v1/channels/{id}/messages`          | GET    | List messages (cursor pagination)         |
| `/v1/presence/{user_id}`              | GET    | Current presence                          |

## WebSocket protocol

```jsonc
// Client → Server
{"type": "message", "channel_id": "...", "sender_device_id": 1,
 "ciphertext": "base64", "ratchet_pub": "base64", "msg_number": 7,
 "reply_to": null}
{"type": "typing", "channel_id": "...", "is_typing": true}
{"type": "read", "channel_id": "...", "msg_id": "..."}
{"type": "react", "channel_id": "...", "msg_id": "...", "reaction": "👍"}
{"type": "ping"}

// Server → Client
{"type": "ready", "user_id": "..."}
{"type": "message", "message": {...}}
{"type": "pong"}
```

## Environment variables

| Key                             | Default                                 |
|---------------------------------|-----------------------------------------|
| `CHAT_HTTP_ADDR`                | `0.0.0.0:8080`                          |
| `CHAT_WS_ADDR`                  | `0.0.0.0:8081`                          |
| `CHAT_RPC_ADDR`                 | `0.0.0.0:8082`                          |
| `CHAT_SCYLLA_URL`               | `scylla-node1,scylla-node2,scylla-node3`|
| `CHAT_VALKEY_URL`               | `redis://valkey:6379`                   |
| `CHAT_NATS_URL`                 | `nats://nats:4222`                      |
| `CHAT_OTLP_ENDPOINT`            | `http://otel-collector:4317`            |
| `CHAT_SIGNAL_DB_KEY`            | (dev-only)                              |
| `CHAT_SIGNAL_USE_HARDWARE_RNG`  | `true`                                  |
| `CHAT_SHUTDOWN_TIMEOUT_SECS`    | `30`                                    |

## Database schema (ScyllaDB)

`messages_by_channel`, `messages_by_user`, `channels`, `channels_by_member`,
`devices`, `groups`, `reactions`, `read_receipts`, `attachments_meta`.
DDL is embedded in `db::scylla::ScyllaStore::ensure_schema()` and runs on
startup.

## Threat model

- **Compromised server**: cannot decrypt message bodies, file attachments,
  or sender-key material. Server only ever sees `ciphertext`, `ratchet_pub`,
  `msg_number`.
- **Compromised device A**: forward secrecy – past sessions remain
  confidential if a later message is decrypted. Post-compromise security –
  new DH ratchet steps heal after a compromise window.
- **Replay**: monotonic `msg_number` checked with constant-time comparison;
  stale messages are dropped.
- **MITM**: identity public keys + signed prekey signatures prevent
  malicious prekey bundle injection.

## Build

```bash
cargo build --release
cargo test
```

## Commit

`feat(chat-engine): implement E2EE messenger (Signal Protocol) + ScyllaDB chat engine + presence`
