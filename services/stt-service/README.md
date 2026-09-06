# STT Service (Speech-to-Text)

Service chuyển đổi giọng nói thành text sử dụng faster-whisper.

## Tính năng

- **High Accuracy**: Whisper large-v3 model
- **Vietnamese Support**: Hỗ trợ tốt tiếng Việt
- **GPU Acceleration**: CUDA support cho inference nhanh
- **Multi-language**: 99+ languages
- **Streaming**: Real-time transcription
- **Speaker Diarization**: Phân biệt người nói
- **Timestamps**: Word-level và segment-level timestamps
- **Format Support**: MP3, WAV, M4A, FLAC, OGG

## Công nghệ

- **Language**: Python 3.11+
- **Framework**: FastAPI
- **STT Model**: faster-whisper (CTranslate2-based)
- **Models**: Whisper tiny/base/small/medium/large-v3
- **GPU**: CUDA (NVIDIA)

## API Endpoints

```
POST   /transcribe             - Upload file và transcribe
POST   /transcribe-url         - Transcribe từ URL
POST   /transcribe/stream      - Streaming với WebSocket
GET    /models                 - List available models
POST   /models/download        - Download model
GET    /health                 - Health check
```

## Transcribe Request

```
POST /transcribe
Content-Type: multipart/form-data

{
  "file": <audio binary>,
  "language": "vi",        # Auto-detect if not specified
  "model": "large-v3",     # tiny/base/small/medium/large-v3
  "task": "transcribe",    # transcribe or translate
  "diarize": false,
  "word_timestamps": true
}
```

## Transcribe Response

```json
{
  "task": "transcribe",
  "language": "vi",
  "duration": 120.5,
  "segments": [
    {
      "id": 0,
      "start": 0.0,
      "end": 3.5,
      "text": "Xin chào, tôi là nhân viên tư vấn.",
      "words": [
        {"word": "Xin", "start": 0.0, "end": 0.3, "probability": 0.98},
        {"word": "chào", "start": 0.3, "end": 0.7, "probability": 0.99},
        ...
      ]
    }
  ],
  "text": "Xin chào, tôi là nhân viên tư vấn...",
  "model": "large-v3",
  "compute_time": 8.2
}
```

## Model Performance

| Model | VRAM | Speed | Accuracy (WER) |
|-------|------|-------|----------------|
| tiny | 1GB | 30x realtime | 12% |
| base | 1GB | 15x realtime | 8% |
| small | 2GB | 8x realtime | 6% |
| medium | 5GB | 4x realtime | 4% |
| large-v3 | 10GB | 1.5x realtime | 2.5% |

(với GPU NVIDIA A100, audio tiếng Việt)

## Environment Variables

```bash
STT_SERVICE_PORT=8089
MODEL_SIZE=large-v3
DEVICE=cuda                  # cuda or cpu
COMPUTE_TYPE=float16         # float16, int8, float32
BEAM_SIZE=5
VAD_FILTER=true              # Voice Activity Detection
DOWNLOAD_ROOT=/models/whisper
MAX_FILE_SIZE=524288000      # 500MB
```

## Development

```bash
pip install -r requirements.txt

# CPU mode
uvicorn main:app --host 0.0.0.0 --port 8089

# GPU mode
CUDA_VISIBLE_DEVICES=0 uvicorn main:app --host 0.0.0.0 --port 8089
```

## Use Cases

1. **Meeting Transcription**: Tự động tạo transcript cho cuộc họp
2. **Call Center Analytics**: Phân tích cuộc gọi
3. **Voice Notes**: Chuyển voice message thành text
4. **Subtitle Generation**: Tạo phụ đề cho video
5. **Voice Commands**: Speech-to-text cho AI assistant

## Performance Optimization

- **VAD Filter**: Skip silent segments (30% speedup)
- **Beam Search**: Tune beam_size vs speed
- **Batch Processing**: Process multiple files in parallel
- **Caching**: Cache model in memory
- **Quantization**: Use int8 for 2x speedup
