FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY packages/ packages/
COPY services/dynamic-model-service/ services/dynamic-model-service/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/dynamic-model-service ./services/dynamic-model-service/cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/dynamic-model-service /app/dynamic-model-service
EXPOSE 8084
ENTRYPOINT ["/app/dynamic-model-service"]
