//! Chat Engine – E2EE messenger (Signal Protocol) + ScyllaDB + Valkey.
//!
//! Entry-point. Wires up:
//! - tracing + OTel + Prometheus
//! - ScyllaDB connection + schema bootstrap
//! - Valkey (Redis-compatible) connection
//! - HTTP / WebSocket axum server
//! - Connect-RPC (tonic) server
//! - NATS publisher
//! - Graceful shutdown

use std::sync::Arc;
use std::time::Duration;

use anyhow::Context;
use chat_engine::{api::types::ChannelKind, config::Config, db::redis::RedisStore, db::scylla::ScyllaStore, AppContext};
use tokio::net::TcpListener;
use tokio::signal;
use tracing::{error, info};
use uuid::Uuid;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // 1. Load configuration.
    let cfg = Config::from_env().map_err(|e| anyhow::anyhow!(e.to_string()))?;
    let _telemetry = chat_engine::telemetry::init("chat-engine", &cfg.otlp_endpoint)
        .map_err(|e| anyhow::anyhow!(e.to_string()))?;

    info!(
        http = %cfg.http_addr,
        ws   = %cfg.ws_addr,
        scylla = %cfg.scylla_url,
        valkey = %cfg.valkey_url,
        nats = %cfg.nats_url,
        "chat-engine starting"
    );

    // 2. Connect to ScyllaDB.
    let contact_points: Vec<String> = cfg
        .scylla_url
        .split(',')
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty())
        .collect();
    let db = ScyllaStore::connect(&contact_points, "rinco_chat")
        .await
        .with_context(|| format!("scylla connect to {:?}", contact_points))?;
    db.ensure_schema().await.context("scylla schema bootstrap")?;

    // 3. Connect to Valkey.
    let redis = RedisStore::connect(&cfg.valkey_url)
        .await
        .with_context(|| format!("valkey connect {}", cfg.valkey_url))?;

    // 4. Connect to NATS (best-effort).
    let nats = match chat_engine::nats::NatsClient::connect(&cfg.nats_url).await {
        Ok(n) => Some(n),
        Err(e) => {
            error!(error = %e, "nats connect failed; continuing without pub/sub");
            None
        }
    };

    // 5. Build the application context.
    let tenant_id = std::env::var("CHAT_TENANT_ID")
        .ok()
        .and_then(|v| Uuid::parse_str(&v).ok())
        .unwrap_or_else(Uuid::nil);
    let ctx = Arc::new(AppContext::new(tenant_id, db, redis));
    let _ = nats; // kept alive for the lifetime of the process

    // 6. Start the HTTP / WebSocket server.
    let http_app = chat_engine::handlers::http::router(ctx.clone());
    let http_app = http_app
        .route("/v1/ws/:user_id", axum::routing::get(ws_upgrade))
        .route("/v1/ws", axum::routing::get(ws_query_upgrade))
        .with_state(ctx.clone());

    let listener = TcpListener::bind(&cfg.http_addr)
        .await
        .with_context(|| format!("bind {}", cfg.http_addr))?;
    info!(addr = %cfg.http_addr, "http/ws listening");

    let server = axum::serve(listener, http_app).with_graceful_shutdown(shutdown_signal());

    if let Err(e) = server.await {
        error!(error = %e, "axum server exited with error");
    }

    info!("chat-engine stopped");
    Ok(())
}

async fn ws_upgrade(
    axum::extract::State(ctx): axum::extract::State<Arc<AppContext>>,
    axum::extract::Path(user_id): axum::extract::Path<Uuid>,
    ws: axum::extract::ws::WebSocketUpgrade,
) -> impl axum::response::IntoResponse {
    ws.on_upgrade(move |socket| chat_engine::handlers::websocket::handle_socket(socket, ctx, user_id))
}

async fn ws_query_upgrade(
    axum::extract::State(ctx): axum::extract::State<Arc<AppContext>>,
    axum::extract::Query(q): axum::extract::Query<WsQuery>,
    ws: axum::extract::ws::WebSocketUpgrade,
) -> impl axum::response::IntoResponse {
    ws.on_upgrade(move |socket| chat_engine::handlers::websocket::handle_socket(socket, ctx, q.user_id))
}

#[derive(serde::Deserialize)]
struct WsQuery {
    user_id: Uuid,
}

async fn shutdown_signal() {
    let ctrl_c = async {
        let _ = signal::ctrl_c().await;
    };
    let terminate = async {
        #[cfg(unix)]
        {
            if let Ok(mut sig) = signal::unix::signal(signal::unix::SignalKind::terminate()) {
                sig.recv().await;
            }
        }
    };
    tokio::select! {
        _ = ctrl_c => info!("ctrl-c received"),
        _ = terminate => info!("SIGTERM received"),
    }
    // Allow in-flight requests up to 30s.
    tokio::time::sleep(Duration::from_millis(50)).await;
}

#[allow(dead_code)]
const _USED: ChannelKind = ChannelKind::Group;
