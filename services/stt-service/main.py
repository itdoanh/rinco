"""
STT Service - Speech-to-Text sử dụng faster-whisper.

Hỗ trợ Vietnamese với Whisper-large-v3.
"""
import os
import tempfile
from typing import Any

from fastapi import FastAPI, File, HTTPException, UploadFile
from faster_whisper import WhisperModel
import structlog

logger = structlog.get_logger()
app = FastAPI(title="STT Service", version="1.0.0")

model_size = os.getenv("WHISPER_MODEL", "large-v3")
device = os.getenv("WHISPER_DEVICE", "cuda")
compute_type = os.getenv("WHISPER_COMPUTE", "float16")

logger.info("loading_whisper", model=model_size, device=device)
model = WhisperModel(model_size, device=device, compute_type=compute_type)


@app.post("/transcribe")
async def transcribe(
    file: UploadFile = File(...),
    language: str = "vi",
    beam_size: int = 5,
):
    """Transcribe audio file → text."""
    if file.content_type not in (
        "audio/wav", "audio/mpeg", "audio/ogg", "audio/webm", "audio/mp4", "audio/x-wav"
    ):
        raise HTTPException(400, f"Unsupported audio format: {file.content_type}")

    audio_bytes = await file.read()

    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        tmp.write(audio_bytes)
        tmp_path = tmp.name

    try:
        segments, info = model.transcribe(
            tmp_path,
            language=language,
            beam_size=beam_size,
            vad_filter=True,
            vad_parameters={"min_silence_duration_ms": 200},
        )
        text_parts = []
        segment_list = []
        for segment in segments:
            text_parts.append(segment.text.strip())
            segment_list.append({
                "start": float(segment.start),
                "end": float(segment.end),
                "text": segment.text.strip(),
                "confidence": float(segment.avg_logprob),
            })

        text = " ".join(text_parts)
        return {
            "text": text,
            "language": info.language,
            "language_probability": float(info.language_probability),
            "duration": float(info.duration),
            "segments": segment_list,
        }
    except Exception as e:
        logger.error("transcribe_failed", error=str(e))
        raise HTTPException(500, f"Transcription failed: {str(e)}")
    finally:
        os.unlink(tmp_path)


@app.post("/transcribe-url")
async def transcribe_url(url: str, language: str = "vi"):
    """Transcribe audio from URL."""
    import httpx
    try:
        async with httpx.AsyncClient(timeout=30.0) as client:
            r = await client.get(url)
            r.raise_for_status()
            audio_bytes = r.content
    except Exception as e:
        raise HTTPException(400, f"Failed to fetch audio: {str(e)}")

    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        tmp.write(audio_bytes)
        tmp_path = tmp.name

    try:
        segments, info = model.transcribe(tmp_path, language=language, vad_filter=True)
        text = " ".join(s.text.strip() for s in segments)
        return {"text": text, "language": info.language, "duration": float(info.duration)}
    finally:
        os.unlink(tmp_path)


@app.get("/health")
async def health():
    return {
        "status": "ok",
        "service": "stt",
        "model": model_size,
        "device": device,
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8093, workers=1)
