FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

# System deps for audio / ffmpeg (kept here even though STT is separate —
# lead-scoring may ingest audio-derived features in future).
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential curl ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY services/lead-scoring/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY services/lead-scoring/ .

# Non-root user.
RUN useradd -u 1001 rinco && chown -R rinco:rinco /app
USER rinco

EXPOSE 8092
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8092", "--workers", "2", "--log-level", "info"]
