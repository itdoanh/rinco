# Recording Service

Service thu và lưu trữ recordings cho WebRTC meetings, sử dụng MinIO/S3 storage.

## Tính năng

- **Multipart Upload**: Hỗ trợ upload file lớn (lên đến 1GB+)
- **MinIO/S3 Storage**: Lưu trữ object storage với presigned URLs
- **Recording Metadata**: Lưu thông tin (room_id, user_id, duration, size)
- **Presigned Download**: Tạo URL download có thời hạn
- **GraphQL API**: Query recordings theo room/tenant
- **Storage Lifecycle**: TTL và archival policies
- **Multi-tenant**: Tenant isolation trong storage paths

## Công nghệ

- **Language**: Rust 1.75+
- **Framework**: Axum
- **S3 Client**: aws-sdk-s3 (compatible với MinIO)
- **Storage**: MinIO (local) hoặc AWS S3 (production)

## API Endpoints

```
POST   /upload                                - Upload recording (multipart)
GET    /download/:recording_id                - Get presigned download URL
GET    /graphql                               - GraphQL API
GET    /health                                - Health check
```

### GraphQL

```graphql
type RecordingInfo {
  id: UUID!
  room_id: UUID!
  filename: String!
  size_bytes: BigInt!
  duration_seconds: Int!
  status: RecordingStatus!
  created_at: DateTime!
  download_url: String
  expires_at: DateTime
}

enum RecordingStatus {
  PENDING
  RECORDING
  PROCESSING
  COMPLETED
  FAILED
}

type Query {
  recording(id: UUID!): RecordingInfo
  roomRecordings(roomId: UUID!): [RecordingInfo!]!
}

type Mutation {
  createRecording(input: CreateRecordingInput!): RecordingInfo!
  updateRecording(
    id: UUID!,
    size_bytes: BigInt!,
    duration_seconds: Int!
  ): Boolean!
}
```

## Upload Format

Multipart form-data:

```
POST /upload
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary

------WebKitFormBoundary
Content-Disposition: form-data; name="recording_id"

uuid
------WebKitFormBoundary
Content-Disposition: form-data; name="filename"

meeting_2026_09_07.webm
------WebKitFormBoundary
Content-Disposition: form-data; name="room_id"

uuid
------WebKitFormBoundary
Content-Disposition: form-data; name="tenant_id"

uuid
------WebKitFormBoundary
Content-Disposition: form-data; name="file"; filename="recording.webm"
Content-Type: video/webm

[binary data]
------WebKitFormBoundary--
```

## Storage Structure

```
s3://recordings/
├── recordings/
│   ├── {tenant_id}/
│   │   ├── {room_id}/
│   │   │   ├── {recording_id}/
│   │   │   │   ├── meeting_001.webm
│   │   │   │   ├── meeting_001.meta.json
```

## Environment Variables

```bash
RECORDING_SERVICE_PORT=8083
MINIO_ENDPOINT=http://minio:9000
AWS_REGION=us-east-1
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=recordings
MAX_UPLOAD_SIZE=1073741824  # 1GB
PRESIGNED_URL_TTL=3600
```

## Performance

- **Upload throughput**: 200MB/s/instance
- **Concurrent uploads**: 50+
- **Storage cost**: ~$0.023/GB/month (S3 Standard)
- **Retrieval latency**: < 100ms (presigned URL)

## Development

```bash
cargo build --release
cargo run --release
```

## Cost Optimization

1. **S3 Intelligent-Tiering**: Auto-move infrequent recordings
2. **Glacier**: Archive recordings > 90 days
3. **Compression**: Use VP9/AV1 for 50% size reduction
4. **CDN**: CloudFront for distribution
