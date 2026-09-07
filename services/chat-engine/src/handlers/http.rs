//! Plain HTTP handlers: health, readiness, metrics, REST history, channel CRUD.

use std::sync::Arc;

use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::{delete, get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::api;
use crate::error::ChatError;
use crate::AppContext;

pub fn router(ctx: Arc<AppContext>) -> Router {
    Router::new()
        .route("/healthz", get(healthz))
        .route("/readyz", get(readyz))
        .route("/metrics", get(metrics))
        .route("/v1/channels", post(create_channel))
        .route("/v1/channels/:channel_id", get(get_channel))
        .route("/v1/channels/:channel_id/members/:user_id", delete(remove_member))
        .route("/v1/channels/:channel_id/messages", get(list_messages))
        .route("/v1/presence/:user_id", get(get_presence))
        .with_state(ctx)
}

async fn healthz() -> impl IntoResponse {
    Json(serde_json::json!({ "status": "ok" }))
}

async fn readyz(State(ctx): State<Arc<AppContext>>) -> impl IntoResponse {
    let scylla_ok = ctx.ping_scylla().await.is_ok();
    let redis_ok = ctx.ping_redis().await;
    let status = if scylla_ok && redis_ok.is_ok() { "ok" } else { "degraded" };
    let body = serde_json::json!({
        "status": status,
        "scylla": scylla_ok,
        "valkey": redis_ok.is_ok(),
    });
    (StatusCode::if_true(scylla_ok && redis_ok.is_ok()), Json(body))
}

async fn metrics() -> impl IntoResponse {
    (
        StatusCode::OK,
        [("content-type", "text/plain; version=0.0.4")],
        crate::telemetry::render_metrics(),
    )
}

#[derive(Deserialize)]
struct CreateChannelBody {
    tenant_id: Uuid,
    kind: crate::api::types::ChannelKind,
    name: Option<String>,
    topic: Option<String>,
    members: Vec<Uuid>,
}

async fn create_channel(
    State(ctx): State<Arc<AppContext>>,
    Json(body): Json<CreateChannelBody>,
) -> Result<impl IntoResponse, ChatError> {
    let channel = api::channels::create_channel(
        &ctx.db,
        body.tenant_id,
        body.kind,
        body.name,
        body.topic,
        body.members,
    )
    .await?;
    Ok((StatusCode::CREATED, Json(channel)))
}

async fn get_channel(
    State(ctx): State<Arc<AppContext>>,
    Path(channel_id): Path<Uuid>,
) -> Result<impl IntoResponse, ChatError> {
    let channel = api::channels::get_channel(&ctx.db, channel_id).await?;
    match channel {
        Some(c) => Ok(Json(c).into_response()),
        None => Err(ChatError::NotFound("channel not found".into())),
    }
}

async fn remove_member(
    State(ctx): State<Arc<AppContext>>,
    Path((channel_id, user_id)): Path<(Uuid, Uuid)>,
) -> Result<impl IntoResponse, ChatError> {
    api::channels::remove_group_member(&ctx.db, channel_id, user_id).await?;
    Ok(StatusCode::NO_CONTENT)
}

#[derive(Deserialize)]
struct ListMessagesQuery {
    cursor: Option<Uuid>,
    #[serde(default = "default_limit")]
    limit: i32,
}

fn default_limit() -> i32 {
    50
}

#[derive(Serialize)]
struct MessageEnvelope {
    messages: Vec<api::types::Message>,
    next_cursor: Option<Uuid>,
}

async fn list_messages(
    State(ctx): State<Arc<AppContext>>,
    Path(channel_id): Path<Uuid>,
    Query(q): Query<ListMessagesQuery>,
) -> Result<impl IntoResponse, ChatError> {
    let messages = api::messages::list_messages(&ctx.db, channel_id, q.cursor, q.limit).await?;
    let next_cursor = messages.last().map(|m| m.id);
    Ok(Json(MessageEnvelope { messages, next_cursor }))
}

async fn get_presence(
    State(ctx): State<Arc<AppContext>>,
    Path(user_id): Path<Uuid>,
) -> Result<impl IntoResponse, ChatError> {
    let p = ctx.redis.get_presence(user_id).await?;
    Ok(Json(p))
}

trait StatusCodeExt {
    fn if_true(b: bool) -> StatusCode;
}
impl StatusCodeExt for StatusCode {
    fn if_true(b: bool) -> StatusCode {
        if b { StatusCode::OK } else { StatusCode::SERVICE_UNAVAILABLE }
    }
}
