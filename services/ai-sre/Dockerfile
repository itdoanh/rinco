FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential curl ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY services/ai-sre/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY services/ai-sre/ .

RUN useradd -u 1001 rinco && chown -R rinco:rinco /app
USER rinco

EXPOSE 8090
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8090", "--workers", "1", "--log-level", "info"]