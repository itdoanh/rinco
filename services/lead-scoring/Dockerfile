FROM python:3.12-slim AS builder

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /build
COPY services/lead-scoring/requirements.txt .
RUN pip install --prefix=/install --no-cache-dir -r requirements.txt

FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PATH=/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin

WORKDIR /app

# System deps for runtime.
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential curl ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy installed dependencies from the build image.
COPY --from=builder /install /usr/local

# Copy service code.
COPY services/lead-scoring/ .

# Non-root user.
RUN useradd -u 1001 rinco && chown -R rinco:rinco /app
USER rinco

EXPOSE 8092
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8092", "--workers", "2", "--log-level", "info"]