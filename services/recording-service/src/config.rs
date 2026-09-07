//! Recording-service configuration.

use serde::{Deserialize, Serialize};

/// Recording-service configuration.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub http_addr: String,
    pub database_url: String,
    pub valkey_url: String,
    pub nats_url: String,
    pub s3_endpoint: String,
    pub s3_region: String,
    pub s3_access_key: String,
    pub s3_secret_key: String,
    pub s3_hot_bucket: String,
    pub s3_cold_bucket: String,
    pub s3_deep_bucket: Option<String>,
    pub gpu_available: bool,
    pub stt_rpc_url: String,
    pub otlp_endpoint: String,
    pub hot_tier_days: i64,
    pub cold_tier_days: i64,
    pub deep_tier_days: i64,
}

impl Config {
    /// Load from environment variables.
    pub fn from_env() -> anyhow::Result<Self> {
        fn env(key: &str, default: &str) -> String {
            std::env::var(key).unwrap_or_else(|_| default.to_string())
        }
        Ok(Self {
            http_addr: env("RECORDING_HTTP_ADDR", "0.0.0.0:8085"),
            database_url: env(
                "RECORDING_DATABASE_URL",
                "postgres://rinco:rinco@postgres:5432/rinco_recordings",
            ),
            valkey_url: env("RECORDING_VALKEY_URL", "redis://valkey:6379"),
            nats_url: env("RECORDING_NATS_URL", "nats://nats:4222"),
            s3_endpoint: env("RECORDING_S3_ENDPOINT", "http://minio:9000"),
            s3_region: env("RECORDING_S3_REGION", "us-east-1"),
            s3_access_key: env("RECORDING_S3_ACCESS_KEY", "minioadmin"),
            s3_secret_key: env("RECORDING_S3_SECRET_KEY", "minioadmin"),
            s3_hot_bucket: env("RECORDING_S3_HOT_BUCKET", "rinco-recordings-hot"),
            s3_cold_bucket: env("RECORDING_S3_COLD_BUCKET", "rinco-recordings-cold"),
            s3_deep_bucket: std::env::var("RECORDING_S3_DEEP_BUCKET").ok(),
            gpu_available: std::env::var("RECORDING_GPU_AVAILABLE")
                .map(|v| matches!(v.to_ascii_lowercase().as_str(), "1" | "true" | "yes"))
                .unwrap_or(false),
            stt_rpc_url: env("RECORDING_STT_RPC_URL", "http://stt-service:8086"),
            otlp_endpoint: env("RECORDING_OTLP_ENDPOINT", "http://otel-collector:4317"),
            hot_tier_days: 30,
            cold_tier_days: 365,
            deep_tier_days: 365 * 3,
        })
    }
}
