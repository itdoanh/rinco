FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY services/email-service/ services/email-service/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/email-service ./services/email-service/cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/email-service /app/email-service
EXPOSE 8087
ENTRYPOINT ["/app/email-service"]