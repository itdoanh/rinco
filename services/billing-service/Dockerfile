FROM --platform=linux/amd64 golang:1.26-alpine AS builder

WORKDIR /app

# Install build deps
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o billing-service ./cmd/main.go

# Final stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/billing-service .
COPY --from=builder /app/migrations ./migrations

ENV PORT=8093
ENV ENVIRONMENT=production
ENV ENVIRONMENT=production

EXPOSE 8093

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8093/health/live || exit 1

ENTRYPOINT ["./billing-service"]
