FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY ./services/tenant-service/ .
RUN CGO_ENABLED=0 go build -o /out/tenant-service ./cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/tenant-service /app/tenant-service
EXPOSE 8082
ENTRYPOINT ["/app/tenant-service"]
