# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY services/landing-service/go.mod services/landing-service/go.sum* ./

# Copy source
COPY ./services/landing-service/ .

# Build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/landing-service ./cmd

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary
COPY --from=builder /out/landing-service /app/landing-service

EXPOSE 8082

ENTRYPOINT ["/app/landing-service"]
