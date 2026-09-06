FROM rust:1.81-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev pkgconfig openssl-dev
COPY services/webrtc-sfu/Cargo.toml services/webrtc-sfu/Cargo.lock* ./
COPY services/webrtc-sfu/src ./src
RUN cargo build --release

FROM alpine:3.19
RUN apk add --no-cache ca-certificates libgcc
WORKDIR /app
COPY --from=builder /app/target/release/webrtc-sfu /app/webrtc-sfu
EXPOSE 8095
ENTRYPOINT ["/app/webrtc-sfu"]
