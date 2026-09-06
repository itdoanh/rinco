#!/usr/bin/env bash
# Generate Dockerfile cho các service chưa có
set -euo pipefail

cd "$(dirname "$0")/.."

SERVICES=(
  "tenant-service:8082"
  "crm-service:8083"
  "dynamic-model-service:8084"
  "lead-service:8085"
  "landing-service:8086"
  "email-service:8087"
  "notification-service:8088"
  "observability-service:8089"
)

for entry in "${SERVICES[@]}"; do
  IFS=':' read -r name port <<< "$entry"
  svc_dir="services/$name"
  if [ -d "$svc_dir" ] && [ ! -f "$svc_dir/Dockerfile" ]; then
    echo "Generating Dockerfile for $name (port $port)..."
    cat > "$svc_dir/Dockerfile" <<EOF
FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY packages/ packages/
COPY $svc_dir/ $svc_dir/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/$name ./$svc_dir/cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/$name /app/$name
EXPOSE $port
ENTRYPOINT ["/app/$name"]
EOF
    echo "  ✓ $svc_dir/Dockerfile"
  fi
done

echo "Done."
