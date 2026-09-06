//! WebRTC SFU (Selective Forwarding Unit).
//!
//! Sử dụng `webrtc` crate (Pion port) để làm SFU.
//! Hỗ trợ AV1/VP9 SVC + simulcast cho efficient multi-party calls.
//!
//! Architecture:
//! - 1 room = N peer connections
//! - SFU forwards media tracks between peers (no transcoding)
//! - Optional recording (sử dụng recording-service)

use std::collections::HashMap;
use std::sync::Arc;

use anyhow::{Context, Result};
use axum::{
    extract::{Path, State, WebSocketUpgrade},
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use dashmap::DashMap;
use serde::{Deserialize, Serialize};
use tracing::{error, info};
use uuid::Uuid;

mod peer;
mod room;
mod signaling;

use peer::PeerConnection;
use room::Room;

#[derive(Clone)]
pub struct AppState {
    pub rooms: Arc<DashMap<String, Arc<Room>>>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CreateRoomRequest {
    pub tenant_id: String,
    pub host_user_id: String,
    pub title: String,
    pub max_participants: u32,
}

#[derive(Debug, Serialize)]
pub struct RoomResponse {
    pub room_id: String,
    pub host_user_id: String,
    pub title: String,
    pub max_participants: u32,
    pub created_at: i64,
}

#[derive(Debug, Serialize)]
pub struct JoinRoomResponse {
    pub room_id: String,
    pub ice_servers: Vec<IceServer>,
    pub participants: Vec<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct IceServer {
    pub urls: Vec<String>,
    pub username: Option<String>,
    pub credential: Option<String>,
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info")),
        )
        .json()
        .init();

    info!("starting WebRTC SFU");

    let state = AppState {
        rooms: Arc::new(DashMap::new()),
    };

    let app = Router::new()
        .route("/health", get(health))
        .route("/metrics", get(metrics_handler))
        .route("/v1/rooms", post(create_room))
        .route("/v1/rooms/:room_id", get(get_room))
        .route("/v1/rooms/:room_id/join", post(join_room))
        .route("/v1/rooms/:room_id/leave", post(leave_room))
        .route("/v1/rooms/:room_id/participants", get(list_participants))
        .route("/ws/:room_id", get(ws_handler))
        .with_state(state);

    let listener = tokio::net::TcpListener::bind(&format!(
        "0.0.0.0:{}",
        std::env::var("PORT").unwrap_or_else(|_| "8095".into())
    ))
    .await?;
    info!("SFU listening on {}", listener.local_addr()?);

    axum::serve(listener, app).await?;
    Ok(())
}

async fn health() -> &'static str {
    "ok"
}

async fn metrics_handler() -> impl IntoResponse {
    let metrics = prometheus::gather();
    let encoder = prometheus::TextEncoder::new();
    match encoder.encode_to_string(&metrics) {
        Ok(s) => (axum::http::StatusCode::OK, s).into_response(),
        Err(e) => (
            axum::http::StatusCode::INTERNAL_SERVER_ERROR,
            e.to_string(),
        )
            .into_response(),
    }
}

async fn create_room(
    State(state): State<AppState>,
    Json(req): Json<CreateRoomRequest>,
) -> impl IntoResponse {
    let room_id = Uuid::now_v7().to_string();
    let room = Arc::new(Room::new(
        room_id.clone(),
        req.tenant_id,
        req.host_user_id.clone(),
        req.title.clone(),
        req.max_participants,
    ));

    state.rooms.insert(room_id.clone(), room.clone());

    (
        axum::http::StatusCode::CREATED,
        Json(RoomResponse {
            room_id: room.id.clone(),
            host_user_id: room.host_user_id.clone(),
            title: room.title.clone(),
            max_participants: room.max_participants,
            created_at: room.created_at,
        }),
    )
}

async fn get_room(
    State(state): State<AppState>,
    Path(room_id): Path<String>,
) -> impl IntoResponse {
    match state.rooms.get(&room_id) {
        Some(room) => (
            axum::http::StatusCode::OK,
            Json(RoomResponse {
                room_id: room.id.clone(),
                host_user_id: room.host_user_id.clone(),
                title: room.title.clone(),
                max_participants: room.max_participants,
                created_at: room.created_at,
            }),
        ),
        None => (
            axum::http::StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "room not found"})),
        ),
    }
}

#[derive(Debug, Deserialize)]
pub struct JoinRequest {
    pub user_id: String,
}

async fn join_room(
    State(state): State<AppState>,
    Path(room_id): Path<String>,
    Json(req): Json<JoinRequest>,
) -> impl IntoResponse {
    let room = match state.rooms.get(&room_id) {
        Some(r) => r.clone(),
        None => {
            return (
                axum::http::StatusCode::NOT_FOUND,
                Json(serde_json::json!({"error": "room not found"})),
            )
        }
    };

    if !room.add_participant(req.user_id.clone()).await {
        return (
            axum::http::StatusCode::FORBIDDEN,
            Json(serde_json::json!({"error": "room is full"})),
        );
    }

    let ice_servers = vec![IceServer {
        urls: vec![
            "stun:stun.l.google.com:19302".into(),
            format!(
                "turn:{}:3478",
                std::env::var("TURN_HOST").unwrap_or_else(|_| "turn.rinco.app".into())
            ),
        ],
        username: Some("rinco".into()),
        credential: Some(std::env::var("TURN_SECRET").unwrap_or_else(|_| "dev".into())),
    }];

    let participants = room.list_participants().await;

    (
        axum::http::StatusCode::OK,
        Json(JoinRoomResponse {
            room_id: room.id.clone(),
            ice_servers,
            participants,
        }),
    )
}

async fn leave_room(
    State(state): State<AppState>,
    Path(room_id): Path<String>,
    Json(req): Json<JoinRequest>,
) -> impl IntoResponse {
    if let Some(room) = state.rooms.get(&room_id) {
        room.remove_participant(&req.user_id).await;
        if room.is_empty().await {
            drop(room);
            state.rooms.remove(&room_id);
        }
    }
    (axum::http::StatusCode::OK, Json(serde_json::json!({"status": "left"})))
}

async fn list_participants(
    State(state): State<AppState>,
    Path(room_id): Path<String>,
) -> impl IntoResponse {
    match state.rooms.get(&room_id) {
        Some(room) => (
            axum::http::StatusCode::OK,
            Json(serde_json::json!({
                "room_id": room_id,
                "participants": room.list_participants().await,
                "count": room.participant_count().await,
            })),
        ),
        None => (
            axum::http::StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "room not found"})),
        ),
    }
}

async fn ws_handler(
    ws: WebSocketUpgrade,
    Path(room_id): Path<String>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| signaling::handle_socket(socket, room_id, state))
}
