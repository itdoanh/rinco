FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY packages/ packages/
COPY services/lead-service/ services/lead-service/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/lead-service ./services/lead-service/cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/lead-service /app/lead-service
EXPOSE 8085
ENTRYPOINT ["/app/lead-service"]
