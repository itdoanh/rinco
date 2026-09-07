# STT Service

Speech-to-text transcription service powered by **faster-whisper** (Whisper large-v3) with optional speaker diarization via **pyannote.audio**.

## Quick Start

```bash
pip install -r requirements.txt
python main.py
```

The service runs on `http://0.0.0.0:8093`.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `WHISPER_MODEL` | `large-v3` | Whisper model size |
| `WHISPER_DEVICE` | `cuda` | Device (`cuda` or `cpu`) |
| `WHISPER_COMPUTE` | `float16` | Compute type (`float16`, `int8`, etc.) |
| `AUDIO_SAMPLE_RATE` | `16000` | Target audio sample rate |

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/health` | Health check |
| `GET` | `/v1/metrics` | Prometheus metrics |
| `GET` | `/v1/languages` | List supported languages |
| `POST` | `/v1/transcribe` | Transcribe audio file |
| `POST` | `/v1/transcribe/stream` | Stream transcription via SSE |
| `POST` | `/v1/detect-language` | Detect spoken language |
| `POST` | `/v1/align` | Forced word-level alignment |

## API Examples

### Transcribe audio

```bash
curl -X POST http://localhost:8093/v1/transcribe \
  -F "file=@audio.wav" \
  -F "language=vi"
```

### Stream transcription

```bash
curl -X POST http://localhost:8093/v1/transcribe/stream \
  -F "file=@audio.wav" \
  -F "language=vi"
```

### Detect language

```bash
curl -X POST http://localhost:8093/v1/detect-language \
  -F "file=@audio.wav"
```

### Word-level alignment

```bash
curl -X POST http://localhost:8093/v1/align \
  -F "file=@audio.wav" \
  -F "reference_text=Hello world" \
  -F "language=en"
```

## Supported Languages

Vietnamese (vi), English (en), Chinese (zh), Japanese (ja), Korean (ko), French (fr), German (de), Spanish (es), Portuguese (pt), Russian (ru), Arabic (ar), Hindi (hi), Thai (th), Indonesian (id), Malay (ms), Tagalog (tl), Burmese (my), Khmer (km), Lao (lo).

## Development

```bash
# Run tests
pytest tests/

# Run with hot reload
uvicorn app.main:app --reload --port 8093
```

## Optional Dependencies

- **pyannote.audio** — speaker diarization (`pip install pyannote-audio`)
- **whisperx** — forced word-level alignment (`pip install whisperx`)
- **ffmpeg** — audio preprocessing (install separately)
