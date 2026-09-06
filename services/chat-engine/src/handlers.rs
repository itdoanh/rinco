//! HTTP + WebSocket handlers.

use std::sync::Arc;

use axum::{
    extract::{
        ws::{Message, WebSocket},
        Path, State, WebSocketUpgrade,
    },
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use chrono::Utc;
use dashmap::DashMap;
use serde::{Deserialize, Serialize};
use tracing::{error, info, warn};
use uuid::Uuid;

use crate::{AppState, ChatEvent};

pub async fn health() -> &'static str {
    "ok"
}

pub async fn metrics() -> impl IntoResponse {
    // Return Prometheus metrics
    let metrics = prometheus::gather();
    let encoder = prometheus::TextEncoder::new();
    match encoder.encode_to_string(&metrics) {
        Ok(s) => (StatusCode::OK, s).into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()).into_response(),
    }
}

#[derive(Debug, Deserialize)]
pub struct SendMessageRequest {
    pub tenant_id: Uuid,
    pub conversation_id: Uuid,
    pub sender_id: Uuid,
    pub content: String,
    pub message_type: String,
}

pub async fn send_message(
    State(state): State<AppState>,
    Json(req): Json<SendMessageRequest>,
) -> impl IntoResponse {
    let message_id = Uuid::now_v7();
    let sent_at = Utc::now().timestamp_millis();

    match state
        .scylla
        .insert_message(
            req.tenant_id,
            req.conversation_id,
            message_id,
            req.sender_id,
            &req.content,
            &req.message_type,
        )
        .await
    {
        Ok(_) => {
            let event = ChatEvent::Message {
                tenant_id: req.tenant_id.to_string(),
                conversation_id: req.conversation_id.to_string(),
                message_id,
                sender_id: req.sender_id.to_string(),
                content: req.content,
                message_type: req.message_type,
                sent_at,
            };
            // Broadcast to subscribers
            let _ = state.tx.send(event);

            (
                StatusCode::CREATED,
                Json(serde_json::json!({
                    "status": "sent",
                    "message_id": message_id,
                    "sent_at": sent_at,
                })),
            )
        }
        Err(e) => {
            error!("insert message failed: {:?}", e);
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        }
    }
}

#[derive(Debug, Serialize)]
pub struct MessageDto {
    pub message_id: String,
    pub sender_id: String,
    pub content: String,
    pub message_type: String,
}

pub async fn get_messages(
    State(state): State<AppState>,
    Path(conversation_id): Path<Uuid>,
    Json(body): Json<serde_json::Value>,
) -> impl IntoResponse {
    let tenant_id: Uuid = match body.get("tenant_id").and_then(|v| v.as_str()).and_then(|s| Uuid::parse_str(s).ok()) {
        Some(id) => id,
        None => return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "tenant_id required"}))).into_response(),
    };
    let limit: i32 = body.get("limit").and_then(|v| v.as_i64()).unwrap_or(50) as i32;

    match state.scylla.get_messages(tenant_id, conversation_id, limit).await {
        Ok(rows) => {
            let messages: Vec<MessageDto> = rows
                .into_iter()
                .map(|r| MessageDto {
                    message_id: r.message_id.to_string(),
                    sender_id: r.sender_id.to_string(),
                    content: r.content,
                    message_type: r.message_type,
                })
                .collect();

            (
                StatusCode::OK,
                Json(serde_json::json!({
                    "conversation_id": conversation_id,
                    "messages": messages,
                    "count": messages.len(),
                })),
            )
        }
        Err(e) => {
            error!("get messages failed: {:?}", e);
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        }
    }
}

#[derive(Debug, Deserialize)]
pub struct PresenceUpdateRequest {
    pub tenant_id: Uuid,
    pub status: String, // online, away, offline, dnd
}

pub async fn update_presence(
    State(state): State<AppState>,
    Path(user_id): Path<Uuid>,
    Json(req): Json<PresenceUpdateRequest>,
) -> impl IntoResponse {
    match state.presence.update(req.tenant_id, user_id, &req.status).await {
        Ok(_) => {
            // Broadcast presence change
            let event = ChatEvent::Presence {
                tenant_id: req.tenant_id.to_string(),
                user_id: user_id.to_string(),
                status: req.status,
            };
            let _ = state.tx.send(event);
            (StatusCode::OK, Json(serde_json::json!({"status": "updated"})))
        }
        Err(e) => {
            error!("update presence failed: {:?}", e);
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        }
    }
}

#[derive(Debug, Deserialize)]
pub struct MarkReadRequest {
    pub tenant_id: Uuid,
    pub user_id: Uuid,
    pub message_id: Uuid,
}

pub async fn mark_read(
    State(state): State<AppState>,
    Path(conversation_id): Path<Uuid>,
    Json(req): Json<MarkReadRequest>,
) -> impl IntoResponse {
    let mut conn = match state.redis.get_async_connection().await {
        Ok(c) => c,
        Err(e) => {
            error!("redis connect failed: {:?}", e);
            return (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
                .into_response();
        }
    };

    let key = format!(
        "read_receipts:{}:{}:{}",
        req.tenant_id, conversation_id, req.user_id
    );
    let value = req.message_id.to_string();

    if let Err(e) = redis::cmd("SET")
        .arg(&key)
        .arg(&value)
        .query_async::<_, ()>(&mut conn)
        .await
    {
        error!("redis set failed: {:?}", e);
        return (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(serde_json::json!({"error": e.to_string()})),
        )
            .into_response();
    }

    let event = ChatEvent::Read {
        tenant_id: req.tenant_id.to_string(),
        conversation_id: conversation_id.to_string(),
        user_id: req.user_id.to_string(),
        message_id: req.message_id,
    };
    let _ = state.tx.send(event);

    (StatusCode::OK, Json(serde_json::json!({"status": "marked"}))).into_response()
}

pub async fn handle_socket(socket: WebSocket, state: AppState) {
    info!("client connected");
    let mut rx = state.tx.subscribe();
    let (mut sender, mut receiver) = socket.split();

    let send_task = tokio::spawn(async move {
        while let Ok(event) = rx.recv().await {
            if let Ok(json) = serde_json::to_string(&event) {
                if sender.send(Message::Text(json)).await.is_err() {
                    break;
                }
            }
        }
    });

    while let Some(msg) = receiver.next().await {
        match msg {
            Ok(Message::Text(text)) => {
                info!("received: {}", text);
                // Echo for now; in production: parse incoming event
            }
            Ok(Message::Close(_)) => {
                info!("client disconnected");
                break;
            }
            Err(e) => {
                warn!("websocket error: {:?}", e);
                break;
            }
            _ => {}
        }
    }

    send_task.abort();
}
