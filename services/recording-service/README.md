# Recording Service — RINCO

GPU/CPU-agnostic egress pipeline that captures WebRTC meetings, stores the
recordings in MinIO, and exposes them through a FastAPI surface.

## Stack

- **FastAPI** – HTTP and REST surface.
- **MinIO** – S3-compatible object storage for the recordings and transcripts.
- **FFmpeg** – encoder for the captured streams (libx264 + AAC by default).
- **httpx** – async client used to talk to the STT service for transcription.

## Endpoints

| Method | Path                                    | Purpose                                      |
|--------|-----------------------------------------|----------------------------------------------|
| GET    | /health                                 | Liveness probe                               |
| POST   | /v1/recordings/start                    | Start a recording for a room                 |
| POST   | /v1/recordings/{id}/stop                | Stop a recording                             |
| GET    | /v1/recordings/{id}                     | Recording metadata                           |
| GET    | /v1/recordings                          | List recordings (filter by room/tenant)      |
| DELETE | /v1/recordings/{id}                     | Delete a recording                           |
| POST   | /v1/recordings/{id}/transcribe          | Trigger STT for a finished recording         |
| POST   | /v1/recordings/{id}/transcript          | Upload a transcript manually                 |
| GET    | /v1/recordings/{id}/transcript          | Get the transcript URL                       |
| GET    | /v1/recordings/{id}/download            | Get a signed download URL                    |
| GET    | /v1/recordings/{id}/playback            | HLS streaming endpoint                       |

## Environment variables

| Variable              | Default                  |
|-----------------------|--------------------------|
| MINIO_HOST            | localhost:9000           |
| MINIO_USER            | minio                    |
| MINIO_PASSWORD        | changeme                 |
| MINIO_SECURE          | false                    |
| RECORDINGS_BUCKET     | rinco-recordings         |
| STT_URL               | http://localhost:8097/v1/transcribe |
| FFMPEG_BIN            | ffmpeg                   |
| MAX_RECORDING_SECONDS | 14400                    |

## Running locally

```bash
pip install -r requirements.txt
python -m uvicorn app.main:app --host 0.0.0.0 --port 8090
```
