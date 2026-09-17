//! NATS consumer for cross-service events (notification fan-out, email).
//!
//! Subjects the chat engine subscribes to:
//! - `rinco.notifications.requested` – request push notification delivery
//! - `rinco.email.requested`         – request transactional email
//!
//! Subjects the chat engine publishes to:
//! - `rinco.chat.message.sent`      – new encrypted message
//! - `rinco.chat.message.read`      – read receipt
//! - `rinco.chat.presence.changed`  – user online/offline

use async_nats::Client;
use bytes::Bytes;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::ChatResult;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationRequest {
    pub tenant_id: Uuid,
    pub user_id: Uuid,
    pub channel_id: Uuid,
    pub msg_id: Uuid,
    pub preview: String,
    pub sender: Uuid,
}

/// Thin wrapper around the NATS client.
#[derive(Clone)]
pub struct NatsClient {
    client: Client,
}

impl NatsClient {
    /// Connect to NATS and start a JetStream context.
    pub async fn connect(url: &str) -> ChatResult<Self> {
        let client = async_nats::connect(url)
            .await
            .map_err(|e| crate::error::ChatError::Nats(e.to_string()))?;
        Ok(Self { client })
    }

    fn encode<T: Serialize>(payload: &T) -> ChatResult<Bytes> {
        let bytes = serde_json::to_vec(payload)
            .map_err(|e| crate::error::ChatError::InvalidRequest(e.to_string()))?;
        Ok(Bytes::from(bytes))
    }

    /// Publish a chat-event subject.
    pub async fn publish_message_sent(&self, payload: &serde_json::Value) -> ChatResult<()> {
        let body = Self::encode(payload)?;
        self.client
            .publish("rinco.chat.message.sent", body)
            .await
            .map_err(|e| crate::error::ChatError::Nats(e.to_string()))?;
        Ok(())
    }

    /// Publish a presence-changed subject.
    pub async fn publish_presence(&self, payload: &serde_json::Value) -> ChatResult<()> {
        let body = Self::encode(payload)?;
        self.client
            .publish("rinco.chat.presence.changed", body)
            .await
            .map_err(|e| crate::error::ChatError::Nats(e.to_string()))?;
        Ok(())
    }

    /// Subscribe to a subject; returns a stream of decoded JSON messages.
    pub async fn subscribe<T: for<'de> Deserialize<'de>>(
        &self,
        subject: &str,
    ) -> ChatResult<async_nats::Subscriber> {
        let sub = self
            .client
            .subscribe(subject.to_string())
            .await
            .map_err(|e| crate::error::ChatError::Nats(e.to_string()))?;
        Ok(sub)
    }

    /// Push a notification request to the notification service.
    pub async fn request_notification(&self, req: &NotificationRequest) -> ChatResult<()> {
        let body = Self::encode(req)?;
        self.client
            .publish("rinco.notifications.requested".to_string(), body)
            .await
            .map_err(|e| crate::error::ChatError::Nats(e.to_string()))?;
        Ok(())
    }
}
