# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy Go modules
COPY services/landing-service/go.mod services/landing-service/go.sum* ./
RUN go mod download

# Copy source
COPY services/landing-service/ ./services/landing-service/
COPY packages/go/ ./packages/go/

# Build
WORKDIR /app/services/landing-service
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/landing-service ./cmd

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary
COPY --from=builder /out/landing-service /app/landing-service

# Copy migrations (for reference, embedded in binary but useful for debugging)
COPY services/landing-service/migrations/ /app/migrations/

# Copy web assets placeholder
COPY services/landing-service/web/ /app/web/

EXPOSE 8082

ENTRYPOINT ["/app/landing-service"]
