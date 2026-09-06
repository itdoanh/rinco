FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY packages/ packages/
COPY services/crm-service/ services/crm-service/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/crm-service ./services/crm-service/cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/crm-service /app/crm-service
EXPOSE 8083
ENTRYPOINT ["/app/crm-service"]
