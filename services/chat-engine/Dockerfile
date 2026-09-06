FROM rust:1.81-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev pkgconfig openssl-dev
COPY services/chat-engine/Cargo.toml services/chat-engine/Cargo.lock* ./
COPY services/chat-engine/src ./src
RUN cargo build --release

FROM alpine:3.19
RUN apk add --no-cache ca-certificates libgcc
WORKDIR /app
COPY --from=builder /app/target/release/chat-engine /app/chat-engine
EXPOSE 8094
ENTRYPOINT ["/app/chat-engine"]
