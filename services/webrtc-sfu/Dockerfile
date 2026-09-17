# =============================================================================
# WebRTC SFU (Go) Dockerfile
# =============================================================================
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for caching
COPY services/webrtc-sfu/go.mod services/webrtc-sfu/go.sum ./
RUN go mod download

# Copy source and build
COPY services/webrtc-sfu/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" \
    -o webrtc-sfu ./cmd/webrtc-sfu

# Final stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/webrtc-sfu /app/webrtc-sfu

ENV SFU_HTTP_ADDR=:8105
ENV SFU_RTC_ADDR=:8106

EXPOSE 8105 8106
ENTRYPOINT ["/app/webrtc-sfu"]
