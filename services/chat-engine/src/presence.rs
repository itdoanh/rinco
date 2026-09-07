//! Presence + typing + read-receipt orchestration.

use chrono::Utc;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::api::types::Presence;
use crate::db::redis::RedisStore;
use crate::error::ChatResult;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PresenceEvent {
    pub user_id: Uuid,
    pub device_id: u64,
    pub online: bool,
    pub last_seen: chrono::DateTime<Utc>,
}

/// Periodic heartbeat task. Refreshes the `presence:{user_id}` key and emits
/// `presence_changed` events to `user:{user_id}` so subscribers learn that the
/// user is still online.
pub async fn heartbeat(
    redis: &RedisStore,
    user_id: Uuid,
    device_id: u64,
) -> ChatResult<()> {
    let p = Presence {
        user_id,
        device_id,
        online: true,
        last_seen: Utc::now(),
    };
    redis.set_presence(&p).await?;
    let event = PresenceEvent {
        user_id,
        device_id,
        online: true,
        last_seen: p.last_seen,
    };
    redis
        .publish_json(
            &format!("user:{}", user_id),
            &serde_json::to_value(&event)?,
        )
        .await?;
    Ok(())
}

/// Mark a user as offline (used by graceful WebSocket close).
pub async fn mark_offline(
    redis: &RedisStore,
    user_id: Uuid,
    device_id: u64,
) -> ChatResult<()> {
    let event = PresenceEvent {
        user_id,
        device_id,
        online: false,
        last_seen: Utc::now(),
    };
    redis
        .publish_json(
            &format!("user:{}", user_id),
            &serde_json::to_value(&event)?,
        )
        .await?;
    Ok(())
}

/// Fetch presence for a list of user ids in one round-trip.
pub async fn presence_for(
    redis: &RedisStore,
    user_ids: &[Uuid],
) -> ChatResult<Vec<Presence>> {
    redis.get_presences(user_ids).await
}
