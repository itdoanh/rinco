//! WebRTC SFU – main binary.

use std::sync::Arc;

use anyhow::Context;
use axum::{
    extract::{ws::WebSocketUpgrade, Path, Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::{delete, get, post},
    Json, Router,
};
use serde::Deserialize;
use tokio::net::TcpListener;
use tokio::signal;
use tracing::{error, info};
use uuid::Uuid;
use webrtc_sfu::{
    config::Config,
    observability,
    peer::PeerManager,
    room::RoomManager,
    sfu::{egress::NullEgress, forwarder::Forwarder},
    signaling,
    SfuContext,
};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let cfg = Config::from_env().context("load sfu config")?;
    let _telemetry = observability::init("webrtc-sfu", &cfg.otlp_endpoint).context("init telemetry")?;
    info!(addr = %cfg.http_addr, "webrtc-sfu starting");

    let ctx = Arc::new(SfuContext {
        config: cfg.clone(),
        rooms: RoomManager::new(),
        peers: PeerManager::new(),
        forwarder: Forwarder::new(),
        egress: Arc::new(NullEgress::default()),
    });

    // Spawn periodic cleanup.
    {
        let ctx = ctx.clone();
        tokio::spawn(async move {
            let mut ticker = tokio::time::interval(std::time::Duration::from_secs(30));
            loop {
                ticker.tick().await;
                ctx.rooms.cleanup_empty();
            }
        });
    }

    let app = Router::new()
        .route("/healthz", get(healthz))
        .route("/readyz", get(readyz))
        .route("/metrics", get(metrics))
        .route("/v1/ws", get(ws_query))
        .route("/v1/ws/:room_id/:user_id", get(ws_path))
        .route("/v1/rooms", post(create_room).get(list_rooms))
        .route("/v1/rooms/:room_id", get(get_room).delete(delete_room))
        .route("/v1/rooms/:room_id/eject/:user_id", post(eject))
        .with_state(ctx.clone());

    let listener = TcpListener::bind(&cfg.http_addr)
        .await
        .with_context(|| format!("bind {}", cfg.http_addr))?;
    info!(addr = %cfg.http_addr, "sfu http+ws listening");
    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await?;
    Ok(())
}

async fn healthz() -> impl IntoResponse {
    Json(serde_json::json!({"status": "ok"}))
}

async fn readyz() -> impl IntoResponse {
    Json(serde_json::json!({"status": "ok"}))
}

async fn metrics() -> impl IntoResponse {
    (
        StatusCode::OK,
        [("content-type", "text/plain; version=0.0.4")],
        observability::render_metrics(),
    )
}

#[derive(Deserialize)]
struct WsQueryParams {
    room_id: Option<Uuid>,
    user_id: Option<Uuid>,
}

async fn ws_query(
    State(ctx): State<Arc<SfuContext>>,
    Query(q): Query<WsQueryParams>,
    ws: WebSocketUpgrade,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| {
        signaling::handle_socket(socket, ctx, q.room_id, q.user_id)
    })
}

async fn ws_path(
    State(ctx): State<Arc<SfuContext>>,
    Path((room_id, user_id)): Path<(Uuid, Uuid)>,
    ws: WebSocketUpgrade,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| {
        signaling::handle_socket(socket, ctx, Some(room_id), Some(user_id))
    })
}

#[derive(Deserialize)]
struct CreateRoomBody {
    tenant_id: Uuid,
    name: String,
    #[serde(default = "default_max_peers")]
    max_peers: u32,
}

fn default_max_peers() -> u32 {
    100
}

async fn create_room(
    State(ctx): State<Arc<SfuContext>>,
    Json(body): Json<CreateRoomBody>,
) -> impl IntoResponse {
    let room = ctx.rooms.create(body.tenant_id, body.name, body.max_peers);
    (StatusCode::CREATED, Json(serde_json::json!({"room_id": room.id})))
}

async fn list_rooms(State(ctx): State<Arc<SfuContext>>) -> impl IntoResponse {
    Json(serde_json::json!({"rooms": ctx.rooms.list()}))
}

async fn get_room(
    State(ctx): State<Arc<SfuContext>>,
    Path(room_id): Path<Uuid>,
) -> impl IntoResponse {
    match ctx.rooms.get(room_id) {
        Some(r) => Json(serde_json::json!({
            "id": r.id,
            "tenant_id": r.tenant_id,
            "name": r.name,
            "participants": r.list_participants(),
        }))
        .into_response(),
        None => (StatusCode::NOT_FOUND, Json(serde_json::json!({"error": "not found"}))).into_response(),
    }
}

async fn delete_room(
    State(ctx): State<Arc<SfuContext>>,
    Path(room_id): Path<Uuid>,
) -> impl IntoResponse {
    if ctx.rooms.delete(room_id) {
        StatusCode::NO_CONTENT
    } else {
        StatusCode::NOT_FOUND
    }
}

async fn eject(
    State(ctx): State<Arc<SfuContext>>,
    Path((room_id, user_id)): Path<(Uuid, Uuid)>,
) -> impl IntoResponse {
    match webrtc_sfu::safety::ejection::kick(&ctx, room_id, user_id).await {
        Ok(()) => StatusCode::NO_CONTENT,
        Err(_) => StatusCode::NOT_FOUND,
    }
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
        _ = ctrl_c => info!("ctrl-c"),
        _ = terminate => info!("SIGTERM"),
    }
}

#[allow(dead_code)]
fn _silence_unused_error() {
    error!("");
}
