//! WebSocket protocol handler.
//!
//! Envelope format (JSON):
//! ```json
//! { "type": "...", ... }
//! ```
//! Where `type` is one of `message`, `typing`, `read`, `react`, `presence`.

use std::sync::Arc;

use axum::extract::ws::{Message, WebSocket};
use futures::{SinkExt, StreamExt};
use serde::{Deserialize, Serialize};
use tracing::{info, warn};
use uuid::Uuid;

use crate::api;
use crate::error::ChatResult;
use crate::AppContext;

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum WsClientMessage {
    Message {
        channel_id: Uuid,
        sender_device_id: u64,
        ciphertext: String,
        ratchet_pub: String,
        msg_number: u64,
        reply_to: Option<Uuid>,
    },
    Typing {
        channel_id: Uuid,
        is_typing: bool,
    },
    Read {
        channel_id: Uuid,
        msg_id: Uuid,
    },
    React {
        channel_id: Uuid,
        msg_id: Uuid,
        reaction: String,
    },
    Ping,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum WsServerMessage {
    Ready {
        user_id: Uuid,
    },
    Message {
        message: api::types::Message,
    },
    Typing {
        channel_id: Uuid,
        user_id: Uuid,
        is_typing: bool,
    },
    Read {
        channel_id: Uuid,
        user_id: Uuid,
        msg_id: Uuid,
    },
    Reaction {
        channel_id: Uuid,
        msg_id: Uuid,
        user_id: Uuid,
        reaction: String,
    },
    Presence {
        user_id: Uuid,
        online: bool,
    },
    Error {
        code: u32,
        message: String,
    },
    Pong,
}

/// Drive a single WebSocket connection.
pub async fn handle_socket(socket: WebSocket, ctx: Arc<AppContext>, user_id: Uuid) {
    let (mut sender, mut receiver) = socket.split();

    // Initial presence heartbeat.
    if let Err(e) = crate::presence::heartbeat(&ctx.redis, user_id, 0).await {
        warn!(error = %e, "presence heartbeat failed");
    }

    // Send `ready` envelope.
    let ready = WsServerMessage::Ready { user_id };
    if let Err(e) = send_envelope(&mut sender, &ready).await {
        warn!(error = %e, "failed to send ready");
        return;
    }

    // Subscribe to per-user events.
    let sub_user = format!("user:{}", user_id);
    let mut user_sub = match ctx.redis.subscribe(&[sub_user.as_str()]).await {
        Ok(s) => s.into_on(),
        Err(e) => {
            warn!(error = %e, "redis subscribe failed");
            return;
        }
    };

    info!(user_id = %user_id, "websocket connected");

    loop {
        tokio::select! {
            incoming = receiver.next() => {
                match incoming {
                    Some(Ok(Message::Text(text))) => {
                        if let Err(e) = handle_text(&ctx, user_id, &text, &mut sender).await {
                            warn!(error = %e, "ws handler error");
                        }
                    }
                    Some(Ok(Message::Binary(_))) => {
                        warn!("binary frames are reserved for FlatBuffers – not yet implemented");
                    }
                    Some(Ok(Message::Close(_))) | None => break,
                    Some(Ok(Message::Ping(p))) => {
                        if sender.send(Message::Pong(p)).await.is_err() { break; }
                    }
                    _ => {}
                }
            }
            // Forward events from Redis pub/sub to the WebSocket.
            _ = futures::future::pending::<()>() => {}
        }

        // Non-blocking peek of the pub/sub stream.
        if let Ok(Some(msg)) = tokio::time::timeout(
            std::time::Duration::from_millis(1),
            user_sub.next(),
        )
        .await
        {
            let payload = String::from_utf8_lossy(&msg.payload).to_string();
            if sender
                .send(Message::Text(payload))
                .await
                .is_err()
            {
                break;
            }
        }
    }

    let _ = crate::presence::mark_offline(&ctx.redis, user_id, 0).await;
    info!(user_id = %user_id, "websocket disconnected");
}

async fn send_envelope(
    sender: &mut futures::stream::SplitSink<WebSocket, Message>,
    env: &WsServerMessage,
) -> ChatResult<()> {
    let text = serde_json::to_string(env)?;
    sender
        .send(Message::Text(text))
        .await
        .map_err(|e| crate::error::ChatError::WebSocket(e.to_string()))
}

async fn handle_text(
    ctx: &Arc<AppContext>,
    user_id: Uuid,
    text: &str,
    sender: &mut futures::stream::SplitSink<WebSocket, Message>,
) -> ChatResult<()> {
    let parsed: WsClientMessage = serde_json::from_str(text)?;
    match parsed {
        WsClientMessage::Message { channel_id, sender_device_id, ciphertext, ratchet_pub, msg_number, reply_to } => {
            let msg = api::types::Message {
                id: Uuid::new_v4(),
                tenant_id: ctx.tenant_id,
                channel_id,
                sender_id: user_id,
                sender_device_id,
                recipient_device_id: None,
                ciphertext: base64::decode(ciphertext).unwrap_or_default(),
                ratchet_pub: base64::decode(ratchet_pub).unwrap_or_default(),
                msg_number,
                attachments: vec![],
                reply_to,
                server_ts: chrono::Utc::now(),
                client_ts: None,
                edited: false,
                deleted: false,
            };
            let stored = api::messages::send_message(&ctx.db, &ctx.redis, msg).await?;
            let env = WsServerMessage::Message { message: stored };
            send_envelope(sender, &env).await?;
        }
        WsClientMessage::Typing { channel_id, is_typing } => {
            ctx.redis.mark_typing(channel_id, user_id, is_typing).await?;
        }
        WsClientMessage::Read { channel_id, msg_id } => {
            api::messages::mark_read(&ctx.db, &ctx.redis, user_id, 0, channel_id, msg_id).await?;
        }
        WsClientMessage::React { channel_id, msg_id, reaction } => {
            let r = api::types::Reaction {
                channel_id,
                msg_id,
                user_id,
                reaction_value: reaction,
                ts: chrono::Utc::now(),
            };
            ctx.db.insert_reaction(&r).await?;
        }
        WsClientMessage::Ping => {
            send_envelope(sender, &WsServerMessage::Pong).await?;
        }
    }
    Ok(())
}
