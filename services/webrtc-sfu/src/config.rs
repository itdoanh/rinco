//! WebRTC SFU configuration.

use serde::{Deserialize, Serialize};
use std::time::Duration;

/// SFU configuration.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    /// HTTP / Connect-RPC bind address.
    pub http_addr: String,
    /// Public IP advertised in ICE candidates. Used when behind 1:1 NAT.
    pub public_ip: Option<String>,
    /// STUN/TURN servers.
    pub ice_servers: Vec<IceServer>,
    /// Maximum peers per room.
    pub max_peers_per_room: u32,
    /// Per-peer bandwidth cap in kbps.
    pub max_bandwidth_kbps: u32,
    /// OTLP endpoint for tracing.
    pub otlp_endpoint: String,
    /// When true the SFU enables AV1/VP9 SVC routing.
    pub enable_svc: bool,
    /// When true, simulcast is enabled (default off).
    pub enable_simulcast: bool,
    /// Recording egress Connect-RPC endpoint.
    pub egress_endpoint: Option<String>,
    /// Graceful shutdown timeout.
    #[serde(default = "default_shutdown")]
    pub shutdown_timeout: Duration,
}

fn default_shutdown() -> Duration {
    Duration::from_secs(30)
}

/// STUN/TURN server config.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IceServer {
    pub urls: Vec<String>,
    pub username: Option<String>,
    pub credential: Option<String>,
}

impl Config {
    /// Build from process environment.
    pub fn from_env() -> anyhow::Result<Self> {
        let _ = std::env::var("SFU_ICE_SERVERS").ok();
        Ok(Self {
            http_addr: std::env::var("SFU_HTTP_ADDR")
                .unwrap_or_else(|_| "0.0.0.0:8084".to_string()),
            public_ip: std::env::var("SFU_PUBLIC_IP").ok(),
            ice_servers: vec![IceServer {
                urls: vec!["stun:stun.l.google.com:19302".into()],
                username: None,
                credential: None,
            }],
            max_peers_per_room: std::env::var("SFU_MAX_PEERS")
                .ok()
                .and_then(|v| v.parse().ok())
                .unwrap_or(500),
            max_bandwidth_kbps: std::env::var("SFU_MAX_BW_KBPS")
                .ok()
                .and_then(|v| v.parse().ok())
                .unwrap_or(4_000),
            otlp_endpoint: std::env::var("SFU_OTLP_ENDPOINT")
                .unwrap_or_else(|_| "http://otel-collector:4317".into()),
            enable_svc: std::env::var("SFU_ENABLE_SVC")
                .map(|v| matches!(v.to_ascii_lowercase().as_str(), "1" | "true" | "yes"))
                .unwrap_or(true),
            enable_simulcast: std::env::var("SFU_ENABLE_SIMULCAST")
                .map(|v| matches!(v.to_ascii_lowercase().as_str(), "1" | "true" | "yes"))
                .unwrap_or(false),
            egress_endpoint: std::env::var("SFU_EGRESS_ENDPOINT").ok(),
            shutdown_timeout: Duration::from_secs(
                std::env::var("SFU_SHUTDOWN_TIMEOUT_SECS")
                    .ok()
                    .and_then(|v| v.parse().ok())
                    .unwrap_or(30),
            ),
        })
    }
}
