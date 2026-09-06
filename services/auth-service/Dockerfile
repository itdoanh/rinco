FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY packages/ packages/
COPY services/auth-service/ services/auth-service/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/auth-service ./services/auth-service/cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/auth-service /app/auth-service
EXPOSE 8081
ENTRYPOINT ["/app/auth-service"]
