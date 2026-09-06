//! Presence Manager - tracks user online/offline status qua Valkey.

use anyhow::Result;
use chrono::Utc;
use redis::AsyncCommands;
use uuid::Uuid;

pub struct PresenceManager {
    redis: redis::Client,
}

impl PresenceManager {
    pub fn new(redis: redis::Client) -> Self {
        Self { redis }
    }

    pub async fn update(
        &self,
        tenant_id: Uuid,
        user_id: Uuid,
        status: &str,
    ) -> Result<()> {
        let mut conn = self.redis.get_async_connection().await?;
        let key = format!("presence:{}:{}", tenant_id, user_id);
        let now = Utc::now().timestamp();

        // Set status with TTL based on status
        let ttl = match status {
            "online" => 60,     // 60s heartbeat
            "away" => 300,      // 5 minutes
            "dnd" => 86400,     // 24 hours
            _ => 0,
        };

        let _: () = conn
            .hset_multiple(
                &key,
                &[
                    ("status", status),
                    ("last_seen", &now.to_string()),
                ],
            )
            .await?;

        if ttl > 0 {
            let _: () = conn.expire(&key, ttl as i64).await?;
        }

        Ok(())
    }

    pub async fn get(
        &self,
        tenant_id: Uuid,
        user_id: Uuid,
    ) -> Result<Option<PresenceInfo>> {
        let mut conn = self.redis.get_async_connection().await?;
        let key = format!("presence:{}:{}", tenant_id, user_id);

        let status: Option<String> = conn.hget(&key, "status").await?;
        let last_seen: Option<i64> = conn.hget(&key, "last_seen").await?;

        Ok(status.map(|s| PresenceInfo {
            user_id: user_id.to_string(),
            status: s,
            last_seen: last_seen.unwrap_or(0),
        }))
    }
}

#[derive(Debug, Clone, serde::Serialize)]
pub struct PresenceInfo {
    pub user_id: String,
    pub status: String,
    pub last_seen: i64,
}
