//! Chat-engine configuration loaded from environment variables.
//!
//! All keys are prefixed with `CHAT_` to avoid clashing with other services
//! in the same process. Values are deserialized using the `config` crate so a
//! TOML file (`config/chat-engine.toml`) may override environment variables.

use serde::{Deserialize, Serialize};
use std::time::Duration;

/// Top-level configuration object.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    /// HTTP / Connect-RPC bind address.
    pub http_addr: String,
    /// WebSocket bind address (may equal `http_addr` if multiplexed).
    pub ws_addr: String,
    /// ScyllaDB contact points (CSV).
    pub scylla_url: String,
    /// Valkey (Redis-compatible) URL.
    pub valkey_url: String,
    /// NATS server URL.
    pub nats_url: String,
    /// OpenTelemetry OTLP endpoint.
    pub otlp_endpoint: String,
    /// Passphrase-derived master key for at-rest encryption of Signal private keys.
    /// NEVER set this to a static production value at runtime – the secret manager
    /// injects it.
    pub signal_db_key: String,
    /// Use hardware RNG when generating Signal Protocol keys.
    pub signal_use_hardware_rng: bool,
    /// Connect-RPC service bind (used for health checks + internal RPC).
    pub rpc_addr: String,
    /// Graceful shutdown timeout.
    #[serde(with = "humantime_serde_compat", default = "default_shutdown")]
    pub shutdown_timeout: Duration,
}

fn default_shutdown() -> Duration {
    Duration::from_secs(30)
}

/// Helper module so we don't pull in the `humantime_serde` crate.
mod humantime_serde_compat {
    use serde::{Deserialize, Deserializer, Serializer};
    use std::time::Duration;

    pub fn serialize<S: Serializer>(d: &Duration, s: S) -> Result<S::Ok, S::Error> {
        s.serialize_some(&d.as_secs())
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<Duration, D::Error> {
        let secs = u64::deserialize(d)?;
        Ok(Duration::from_secs(secs))
    }
}

impl Config {
    /// Build configuration from process environment.
    pub fn from_env() -> Result<Self, crate::error::ChatError> {
        let _ = dotenv_lite::load();
        Ok(Self {
            http_addr: env_or("CHAT_HTTP_ADDR", "0.0.0.0:8080"),
            ws_addr: env_or("CHAT_WS_ADDR", "0.0.0.0:8081"),
            scylla_url: env_or(
                "CHAT_SCYLLA_URL",
                "scylla-node1,scylla-node2,scylla-node3",
            ),
            valkey_url: env_or("CHAT_VALKEY_URL", "redis://valkey:6379"),
            nats_url: env_or("CHAT_NATS_URL", "nats://nats:4222"),
            otlp_endpoint: env_or("CHAT_OTLP_ENDPOINT", "http://otel-collector:4317"),
            signal_db_key: env_or("CHAT_SIGNAL_DB_KEY", "dev-only-do-not-use-in-prod"),
            signal_use_hardware_rng: env_bool("CHAT_SIGNAL_USE_HARDWARE_RNG", true),
            rpc_addr: env_or("CHAT_RPC_ADDR", "0.0.0.0:8082"),
            shutdown_timeout: Duration::from_secs(env_or("CHAT_SHUTDOWN_TIMEOUT_SECS", "30").parse().unwrap_or(30)),
        })
    }
}

fn env_or(key: &str, default: &str) -> String {
    std::env::var(key).unwrap_or_else(|_| default.to_string())
}

fn env_bool(key: &str, default: bool) -> bool {
    std::env::var(key)
        .ok()
        .map(|v| matches!(v.to_ascii_lowercase().as_str(), "1" | "true" | "yes" | "on"))
        .unwrap_or(default)
}

/// Tiny `.env` reader that avoids pulling the `dotenvy` crate. It only loads a
/// file if one exists and silently ignores errors – production code uses real
/// secret managers.
mod dotenv_lite {
    use std::fs;

    pub fn load() -> std::io::Result<()> {
        let path = std::path::Path::new(".env");
        if !path.exists() {
            return Ok(());
        }
        let content = fs::read_to_string(path)?;
        for line in content.lines() {
            let line = line.trim();
            if line.is_empty() || line.starts_with('#') {
                continue;
            }
            if let Some((k, v)) = line.split_once('=') {
                if std::env::var(k).is_err() {
                    std::env::set_var(k.trim(), v.trim().trim_matches('"'));
                }
            }
        }
        Ok(())
    }
}
