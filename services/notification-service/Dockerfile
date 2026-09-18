FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY ./services/notification-service/ .
RUN CGO_ENABLED=0 go build -o /out/notification-service ./cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/notification-service /app/notification-service
EXPOSE 8088
ENTRYPOINT ["/app/notification-service"]