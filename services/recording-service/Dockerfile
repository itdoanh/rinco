FROM rust:1.81-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev pkgconfig openssl-dev
COPY services/recording-service/Cargo.toml services/recording-service/Cargo.lock* ./
COPY services/recording-service/src ./src
RUN cargo build --release

FROM alpine:3.19
RUN apk add --no-cache ca-certificates libgcc
WORKDIR /app
COPY --from=builder /app/target/release/recording-service /app/recording-service
EXPOSE 8096
ENTRYPOINT ["/app/recording-service"]
