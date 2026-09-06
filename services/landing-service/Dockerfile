FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY packages/ packages/
COPY services/landing-service/ services/landing-service/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/landing-service ./services/landing-service/cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/landing-service /app/landing-service
EXPOSE 8086
ENTRYPOINT ["/app/landing-service"]
