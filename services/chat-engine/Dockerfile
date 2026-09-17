# =============================================================================
# Chat Engine (Go) Dockerfile
# =============================================================================
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for caching
COPY services/chat-engine/go.mod services/chat-engine/go.sum ./
RUN go mod download

# Copy source and build
COPY services/chat-engine/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" \
    -o chat-engine ./cmd/chat-engine

# Final stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/chat-engine /app/chat-engine
COPY --from=builder /app/services/chat-engine/migrations /app/migrations

ENV CHAT_HTTP_ADDR=:8101
ENV CHAT_RPC_ADDR=:8102
ENV CHAT_IN_MEMORY_MODE=true

EXPOSE 8101 8102
ENTRYPOINT ["/app/chat-engine"]
