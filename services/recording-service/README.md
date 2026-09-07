# Recording Service – GPU-accelerated egress + tiered S3 storage + STT

Egress pipeline for the WebRTC SFU. Receives RTP streams from one or more
meetings, composites them, muxes audio, encodes the result to MP4, and
uploads the finished recording to a multi-bucket S3-compatible store.
Optionally transcribes the audio via the `stt-service`.

## Highlights

- **Egress worker** (`egress::worker`): spawns per room, accepts video
  frames + audio frames, finalises when the meeting ends.
- **Compositor** (`egress::compositor`): CPU `image`-based blit with
  `Gallery` and `Focused` layouts. GPU backend behind `gpu` feature
  flag (falls back to CPU when disabled or when no CUDA device is
  available).
- **Audio mixer** (`egress::audio_mixer`): N input streams → 1 stereo
  track at the target sample rate, with simple linear resampling.
- **Uploader** (`egress::uploader`): encodes the final MP4 via
  `ffmpeg-next` (optional `ffmpeg` feature). When ffmpeg is disabled
  the encoder emits a placeholder buffer so the rest of the pipeline
  remains testable.
- **AI pipeline** (`egress::ai_pipeline`): posts audio to the STT
  service and persists the returned transcript.
- **Tiered storage** (`storage::tiered`): hot (SSD) → cold (HDD) →
  deep (Glacier) transitions. Each tier has its own bucket.
- **Background jobs** (`jobs`):
  - `transition` – daily sweep that moves objects between tiers based
    on age.
  - `transcript_indexer` – pushes transcript segments into Meilisearch
    / OpenSearch.
- **PostgreSQL metadata** (`storage::metadata`): durable metadata store
  via `sqlx`. An `InMemoryMetadata` shim is included for local dev.

## Module layout

```
src/
├── main.rs                       # actix-web server
├── lib.rs                        # Re-exports
├── config.rs                     # Env-driven configuration
├── error.rs                      # RecordingError + RecordingResult
├── observability.rs              # OTel + Prometheus
├── storage/
│   ├── tiered.rs                 # Multi-bucket S3 client
│   └── metadata.rs               # RecordingRow + InMemoryMetadata
├── egress/
│   ├── worker.rs                 # EgressWorker lifecycle
│   ├── compositor.rs             # Video frame composite
│   ├── audio_mixer.rs            # N→1 audio mix
│   ├── uploader.rs               # MP4 mux + S3 upload
│   └── ai_pipeline.rs            # STT integration
├── api/recording_service.rs      # Connect-RPC RecordingService
└── jobs/
    ├── transition.rs             # Tier lifecycle
    └── transcript_indexer.rs     # Search indexing
migrations/0001_recordings.sql    # PostgreSQL schema
```

## Endpoints

| Path                                       | Method | Purpose                            |
|--------------------------------------------|--------|------------------------------------|
| `/healthz`                                 | GET    | Liveness                           |
| `/readyz`                                  | GET    | Readiness                          |
| `/metrics`                                 | GET    | Prometheus exposition              |
| `/v1/recordings/start`                     | POST   | Start recording (Connect-RPC)      |
| `/v1/recordings/stop`                      | POST   | Stop recording                     |
| `/v1/recordings`                           | GET    | List recordings                    |
| `/v1/recordings/{id}`                      | GET    | Recording detail                   |
| `/v1/recordings/{id}/delete`               | POST   | Delete recording                   |
| `/v1/recordings/{id}/transcript`           | GET    | Transcript URL                     |
| `/v1/recordings/{id}/tier`                 | POST   | Update tier (hot/cold/deep)        |

## Connect-RPC `RecordingService`

| RPC              | Request                  | Response                |
|------------------|--------------------------|-------------------------|
| StartRecording   | StartRecordingRequest    | StartRecordingResponse  |
| StopRecording    | StopRecordingRequest     | Empty                   |
| GetRecording     | GetRecordingRequest      | GetRecordingResponse    |
| ListRecordings   | ListRecordingsRequest    | ListRecordingsResponse  |
| DeleteRecording  | DeleteRecordingRequest   | Empty                   |
| GetTranscript    | GetTranscriptRequest     | GetTranscriptResponse   |
| UpdateTier       | UpdateTierRequest        | Empty                   |

## Environment variables

| Key                            | Default                                                |
|--------------------------------|--------------------------------------------------------|
| `RECORDING_HTTP_ADDR`          | `0.0.0.0:8085`                                         |
| `RECORDING_DATABASE_URL`       | `postgres://rinco:rinco@postgres:5432/rinco_recordings`|
| `RECORDING_VALKEY_URL`         | `redis://valkey:6379`                                  |
| `RECORDING_NATS_URL`           | `nats://nats:4222`                                     |
| `RECORDING_S3_ENDPOINT`        | `http://minio:9000`                                    |
| `RECORDING_S3_REGION`          | `us-east-1`                                            |
| `RECORDING_S3_ACCESS_KEY`      | `minioadmin`                                           |
| `RECORDING_S3_SECRET_KEY`      | `minioadmin`                                           |
| `RECORDING_S3_HOT_BUCKET`      | `rinco-recordings-hot`                                 |
| `RECORDING_S3_COLD_BUCKET`     | `rinco-recordings-cold`                                |
| `RECORDING_S3_DEEP_BUCKET`     | (none)                                                 |
| `RECORDING_GPU_AVAILABLE`      | `false`                                                |
| `RECORDING_STT_RPC_URL`        | `http://stt-service:8086`                              |
| `RECORDING_OTLP_ENDPOINT`      | `http://otel-collector:4317`                           |

## Tier policy

| Age        | Tier  | Bucket                       |
|------------|-------|------------------------------|
| 0 – 30 d   | Hot   | `rinco-recordings-hot`       |
| 30 – 365 d | Cold  | `rinco-recordings-cold`      |
| > 365 d    | Deep  | `rinco-recordings-deep` (or cold fallback) |

## Build

```bash
cargo build --release
cargo test
```

## Commit

`feat(recording-service): GPU-accelerated egress (NVENC), tiered S3 storage, STT pipeline`
