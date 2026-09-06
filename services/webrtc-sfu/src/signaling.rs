//! WebSocket signaling handler cho WebRTC.
//!
//! Protocol: JSON messages over WebSocket
//! - {type: "offer", sdp: "..."}
//! - {type: "answer", sdp: "..."}
//! - {type: "ice-candidate", candidate: "..."}
//! - {type: "join", user_id: "..."}

use axum::extract::ws::{Message, WebSocket};
use futures::{SinkExt, StreamExt};
use serde::{Deserialize, Serialize};
use tracing::{error, info, warn};

use crate::AppState;

#[derive(Debug, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum SignalingMessage {
    #[serde(rename = "join")]
    Join {
        user_id: String,
        sdp: Option<String>,
    },
    #[serde(rename = "offer")]
    Offer { from: String, to: String, sdp: String },
    #[serde(rename = "answer")]
    Answer { from: String, to: String, sdp: String },
    #[serde(rename = "ice")]
    Ice {
        from: String,
        to: String,
        candidate: String,
    },
    #[serde(rename = "leave")]
    Leave { user_id: String },
    #[serde(rename = "ping")]
    Ping {},
    #[serde(rename = "pong")]
    Pong {},
}

pub async fn handle_socket(socket: WebSocket, room_id: String, _state: AppState) {
    info!("client connected to room {}", room_id);
    let (mut sender, mut receiver) = socket.split();

    while let Some(msg) = receiver.next().await {
        match msg {
            Ok(Message::Text(text)) => {
                match serde_json::from_str::<SignalingMessage>(&text) {
                    Ok(SignalingMessage::Ping {}) => {
                        if let Ok(s) = serde_json::to_string(&SignalingMessage::Pong {}) {
                            let _ = sender.send(Message::Text(s)).await;
                        }
                    }
                    Ok(_) => {
                        // In production: forward message to other peers in room
                        info!("received signaling message in room {}", room_id);
                    }
                    Err(e) => {
                        warn!("invalid signaling message: {}", e);
                    }
                }
            }
            Ok(Message::Close(_)) => {
                info!("client disconnected from room {}", room_id);
                break;
            }
            Err(e) => {
                error!("websocket error: {}", e);
                break;
            }
            _ => {}
        }
    }
}
