FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY services/observability-service/ services/observability-service/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/observability-service ./services/observability-service/cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/observability-service /app/observability-service
EXPOSE 8099
ENTRYPOINT ["/app/observability-service"]