# Chat Engine

Real-time chat engine với WebSocket, viết bằng Rust cho hiệu năng cao nhất.

## Tính năng

- **WebSocket Real-time**: Persistent connections cho chat
- **ScyllaDB**: Lưu trữ tin nhắn với write latency < 10ms
- **Valkey/Redis Presence**: User online/offline tracking với TTL
- **Typing Indicators**: Real-time "đang nhập..." indicator
- **Read Receipts**: Tracking ai đã đọc tin nhắn
- **Conversation History**: Lazy load với cursor-based pagination
- **GraphQL API**: Modern query API
- **Multi-tenant**: Tenant isolation

## Công nghệ

- **Language**: Rust 1.75+
- **Framework**: Axum (Tokio)
- **Database**: ScyllaDB (gocql)
- **Cache/Presence**: Valkey (Redis-compatible)
- **WebSocket**: tokio-tungstenite
- **GraphQL**: async-graphql

## Cấu trúc dữ liệu

### ScyllaDB Schema

```cql
CREATE KEYSPACE rinco_chat WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 3};

CREATE TABLE messages_by_conversation (
    conversation_id UUID,
    message_id UUID,
    sender_id UUID,
    content TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    is_deleted BOOLEAN,
    PRIMARY KEY (conversation_id, created_at, message_id)
) WITH CLUSTERING ORDER BY (created_at DESC);

CREATE TABLE presence_by_user (
    user_id UUID PRIMARY KEY,
    status TEXT,
    last_seen TIMESTAMP,
    device_info TEXT
);

CREATE TABLE read_receipts (
    message_id UUID,
    user_id UUID,
    read_at TIMESTAMP,
    PRIMARY KEY (message_id, user_id)
);
```

## WebSocket Protocol

### Client → Server

```json
{
  "type": "message",
  "conversation_id": "uuid",
  "content": "Hello!"
}
```

```json
{
  "type": "typing",
  "conversation_id": "uuid",
  "is_typing": true
}
```

### Server → Client

```json
{
  "type": "message",
  "id": "uuid",
  "conversation_id": "uuid",
  "sender_id": "uuid",
  "content": "Hello!",
  "created_at": "2026-09-07T..."
}
```

## API Endpoints

### HTTP
```
GET  /graphql              - GraphQL endpoint
GET  /ws                   - WebSocket upgrade
GET  /health               - Health check
```

### GraphQL
```graphql
query {
  messages(conversation_id: "uuid", limit: 50) {
    id
    content
    sender_id
    created_at
  }
}

mutation {
  sendMessage(input: {
    conversation_id: "uuid",
    sender_id: "uuid",
    content: "Hello!"
  }) {
    id
    content
  }
}
```

## Performance

- **Latency**: < 50ms p99 cho message round-trip
- **Throughput**: 100K messages/sec/instance
- **Memory**: < 100MB baseline với 10K concurrent connections
- **CPU**: < 1 core cho 10K concurrent connections

## Environment Variables

```bash
CHAT_ENGINE_PORT=8080
SCYLLA_SEEDS=scylla1,scylla2,scylla3
REDIS_URL=redis://valkey:6379
RUST_LOG=info
```

## Development

```bash
cargo build --release
cargo run --release

# Test
cargo test

# Benchmark
cargo bench
```
