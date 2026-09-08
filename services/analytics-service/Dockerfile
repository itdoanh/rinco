# syntax=docker/dockerfile:1.6
FROM golang:1.26-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/analytics-service ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/analytics-service /app/analytics-service
ENV ENVIRONMENT=production
EXPOSE 8099
USER nonroot:nonroot
ENTRYPOINT ["/app/analytics-service"]
