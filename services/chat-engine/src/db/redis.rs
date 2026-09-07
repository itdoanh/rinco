//! Redis / Valkey adapter for the chat engine.
//!
//! Stores:
//! - `presence:{user_id}` → device id + last seen (TTL 90s)
//! - `typing:{channel_id}` → SET of user_ids (TTL 5s)
//! - `read:{channel_id}:{user_id}` → last read message id
//! - `unread:{user_id}:{channel_id}` → counter
//!
//! Also exposes pub/sub helpers for cross-instance fan-out.

use std::sync::Arc;
use uuid::Uuid;

use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use serde_json::Value;

use crate::api::types::Presence;
use crate::error::ChatResult;

#[derive(Clone)]
pub struct RedisStore {
    conn: Arc<tokio::sync::Mutex<ConnectionManager>>,
}

impl RedisStore {
    /// Connect to Valkey and ping the server.
    pub async fn connect(url: &str) -> ChatResult<Self> {
        let client = redis::Client::open(url)?;
        let manager = client.get_connection_manager().await?;
        Ok(Self {
            conn: Arc::new(tokio::sync::Mutex::new(manager)),
        })
    }

    async fn conn(&self) -> ConnectionManager {
        self.conn.lock().await.clone()
    }

    // ============================================================
    // Presence
    // ============================================================

    pub async fn set_presence(&self, p: &Presence) -> ChatResult<()> {
        let mut c = self.conn().await;
        let key = format!("presence:{}", p.user_id);
        let value = serde_json::to_string(p)?;
        // 90 second heartbeat: a missed heartbeat = offline.
        let _: () = c.set_ex(&key, value, 90).await?;
        Ok(())
    }

    pub async fn get_presence(&self, user_id: Uuid) -> ChatResult<Option<Presence>> {
        let mut c = self.conn().await;
        let key = format!("presence:{}", user_id);
        let raw: Option<String> = c.get(&key).await?;
        Ok(raw.and_then(|s| serde_json::from_str(&s).ok()))
    }

    pub async fn get_presences(&self, user_ids: &[Uuid]) -> ChatResult<Vec<Presence>> {
        let mut out = Vec::with_capacity(user_ids.len());
        for u in user_ids {
            if let Some(p) = self.get_presence(*u).await? {
                out.push(p);
            }
        }
        Ok(out)
    }

    // ============================================================
    // Typing
    // ============================================================

    pub async fn mark_typing(
        &self,
        channel_id: Uuid,
        user_id: Uuid,
        is_typing: bool,
    ) -> ChatResult<()> {
        let mut c = self.conn().await;
        let key = format!("typing:{}", channel_id);
        if is_typing {
            let _: () = c.sadd(&key, user_id.to_string()).await?;
            let _: () = c.expire(&key, 5).await?;
        } else {
            let _: () = c.srem(&key, user_id.to_string()).await?;
        }
        Ok(())
    }

    pub async fn get_typing(&self, channel_id: Uuid) -> ChatResult<Vec<Uuid>> {
        let mut c = self.conn().await;
        let key = format!("typing:{}", channel_id);
        let members: Vec<String> = c.smembers(&key).await?;
        Ok(members
            .into_iter()
            .filter_map(|s| Uuid::parse_str(&s).ok())
            .collect())
    }

    // ============================================================
    // Read receipts & unread counters
    // ============================================================

    pub async fn set_last_read(
        &self,
        user_id: Uuid,
        channel_id: Uuid,
        msg_id: Uuid,
    ) -> ChatResult<()> {
        let mut c = self.conn().await;
        let key = format!("read:{}:{}", channel_id, user_id);
        let _: () = c.set(&key, msg_id.to_string()).await?;
        Ok(())
    }

    pub async fn last_read(
        &self,
        user_id: Uuid,
        channel_id: Uuid,
    ) -> ChatResult<Option<Uuid>> {
        let mut c = self.conn().await;
        let key = format!("read:{}:{}", channel_id, user_id);
        let raw: Option<String> = c.get(&key).await?;
        Ok(raw.and_then(|s| Uuid::parse_str(&s).ok()))
    }

    pub async fn bump_unread(&self, user_id: Uuid, channel_id: Uuid) -> ChatResult<()> {
        let mut c = self.conn().await;
        let key = format!("unread:{}:{}", user_id, channel_id);
        let _: i64 = c.incr(&key, 1).await?;
        Ok(())
    }

    pub async fn clear_unread(&self, user_id: Uuid, channel_id: Uuid) -> ChatResult<()> {
        let mut c = self.conn().await;
        let key = format!("unread:{}:{}", user_id, channel_id);
        let _: () = c.del(&key).await?;
        Ok(())
    }

    pub async fn unread(&self, user_id: Uuid, channel_id: Uuid) -> ChatResult<i64> {
        let mut c = self.conn().await;
        let key = format!("unread:{}:{}", user_id, channel_id);
        let val: Option<i64> = c.get(&key).await?;
        Ok(val.unwrap_or(0))
    }

    // ============================================================
    // Pub/Sub
    // ============================================================

    /// Publish a JSON envelope to a channel. Used by every chat-engine node to
    /// broadcast newly sent messages to its locally connected WebSocket clients.
    pub async fn publish_json(&self, channel: &str, payload: &Value) -> ChatResult<()> {
        let mut c = self.conn().await;
        let text = serde_json::to_string(payload)?;
        let _: i64 = c.publish(channel, text).await?;
        Ok(())
    }

    /// Open a subscription stream. The caller is responsible for selecting over
    /// the returned receiver.
    pub async fn subscribe(
        &self,
        channels: &[&str],
    ) -> ChatResult<redis::streams::Stream> {
        let mut c = self.conn().await;
        let stream = c.subscribe(channels).await?;
        Ok(stream)
    }
}
