//! Chat Engine - Real-time chat sử dụng Rust + Axum + ScyllaDB + Valkey.
//!
//! Architecture:
//! - WebSocket gateway cho client connections
//! - ScyllaDB cho message persistence
//! - Valkey cho presence + read receipts + typing
//! - FlatBuffers cho binary protocol (planned)
//! - NATS cho fan-out giữa các instances

use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;

use anyhow::Context;
use axum::{
    extract::{ws::Message, State, WebSocketUpgrade},
    response::IntoResponse,
    routing::get,
    Router,
};
use dashmap::DashMap;
use serde::{Deserialize, Serialize};
use tokio::sync::broadcast;
use tracing::{error, info, warn};
use uuid::Uuid;

mod db;
mod handlers;
mod presence;

use presence::PresenceManager;

#[derive(Clone)]
pub struct AppState {
    pub scylla: Arc<db::ScyllaClient>,
    pub redis: redis::Client,
    pub presence: Arc<PresenceManager>,
    pub tx: broadcast::Sender<ChatEvent>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum ChatEvent {
    #[serde(rename = "message")]
    Message {
        tenant_id: String,
        conversation_id: String,
        message_id: Uuid,
        sender_id: String,
        content: String,
        message_type: String,
        sent_at: i64,
    },
    #[serde(rename = "typing")]
    Typing {
        tenant_id: String,
        conversation_id: String,
        user_id: String,
        is_typing: bool,
    },
    #[serde(rename = "presence")]
    Presence {
        tenant_id: String,
        user_id: String,
        status: String, // online, away, offline, dnd
    },
    #[serde(rename = "read")]
    Read {
        tenant_id: String,
        conversation_id: String,
        user_id: String,
        message_id: Uuid,
    },
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Init tracing
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info")),
        )
        .json()
        .init();

    info!("starting chat engine");

    // Init ScyllaDB
    let scylla = Arc::new(
        db::ScyllaClient::new(&std::env::var("SCYLLA_HOST").unwrap_or_else(|_| "localhost".into()))
            .await
            .context("connect scylla")?,
    );

    // Init Valkey
    let redis_client = redis::Client::open(
        std::env::var("VALKEY_URL").unwrap_or_else(|_| "redis://localhost:6379".into()),
    )
    .context("open redis client")?;

    // Init presence
    let presence = Arc::new(PresenceManager::new(redis_client.clone()));

    // Broadcast channel for fan-out
    let (tx, _) = broadcast::channel::<ChatEvent>(1024);

    let state = AppState {
        scylla,
        redis: redis_client,
        presence,
        tx,
    };

    // Routes
    let app = Router::new()
        .route("/health", get(handlers::health))
        .route("/metrics", get(handlers::metrics))
        .route("/ws", get(ws_handler))
        .route("/v1/messages", axum::routing::post(handlers::send_message))
        .route(
            "/v1/conversations/:conversation_id/messages",
            axum::routing::get(handlers::get_messages),
        )
        .route(
            "/v1/presence/:user_id",
            axum::routing::post(handlers::update_presence),
        )
        .route(
            "/v1/conversations/:conversation_id/read",
            axum::routing::post(handlers::mark_read),
        )
        .with_state(state);

    let listener = tokio::net::TcpListener::bind(&format!(
        "0.0.0.0:{}",
        std::env::var("PORT").unwrap_or_else(|_| "8094".into())
    ))
    .await?;

    info!("chat engine listening on {}", listener.local_addr()?);

    axum::serve(listener, app).await?;

    Ok(())
}

async fn ws_handler(
    ws: WebSocketUpgrade,
    State(state): State<AppState>,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| handlers::handle_socket(socket, state))
}
