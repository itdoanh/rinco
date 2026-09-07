//! WebSocket signaling server.
//!
//! Protocol (JSON):
//! ```json
//! {"type": "join",  "room_id": "...", "user_id": "...", "display_name": "..."}
//! {"type": "offer", "sdp": "..."}
//! {"type": "answer","sdp": "..."}
//! {"type": "ice",   "candidate": "...", "sdp_mid": null, "sdp_mline_index": 0}
//! {"type": "leave"}
//! ```

use axum::extract::ws::{Message, WebSocket};
use futures::{SinkExt, StreamExt};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::{info, warn};
use uuid::Uuid;

use crate::error::SfuResult;
use crate::peer::{IceCandidate, Peer};
use crate::room::{Participant, RoomEvent};
use crate::SfuContext;

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum ClientMessage {
    Join {
        room_id: Uuid,
        user_id: Uuid,
        tenant_id: Uuid,
        display_name: String,
    },
    Leave,
    Offer { sdp: String },
    Answer { sdp: String },
    Ice {
        candidate: String,
        sdp_mid: Option<String>,
        sdp_mline_index: Option<u16>,
    },
    Mute { audio: bool, video: bool, screen: bool },
    Ping,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum ServerMessage {
    Ready {
        room_id: Uuid,
        participants: Vec<Participant>,
        ice_servers: Vec<crate::config::IceServer>,
    },
    Offer { sdp: String },
    Answer { sdp: String },
    Ice {
        candidate: String,
        sdp_mid: Option<String>,
        sdp_mline_index: Option<u16>,
    },
    PeerJoined { participant: Participant },
    PeerLeft { user_id: Uuid },
    AudioMute { user_id: Uuid, muted: bool },
    VideoToggle { user_id: Uuid, on: bool },
    ScreenShare { user_id: Uuid, on: bool },
    Recording { on: bool },
    RoomClosed,
    Error { code: u32, message: String },
    Pong,
}

/// Drive one WebSocket connection.
pub async fn handle_socket(socket: WebSocket, ctx: Arc<SfuContext>, initial_room: Option<Uuid>, initial_user: Option<Uuid>) {
    let (mut sink, mut stream) = socket.split();
    let mut state = ConnectionState::default();
    if let (Some(r), Some(u)) = (initial_room, initial_user) {
        state.room_id = Some(r);
        state.user_id = Some(u);
    }

    info!(room = ?state.room_id, user = ?state.user_id, "ws connected");

    loop {
        tokio::select! {
            incoming = stream.next() => {
                match incoming {
                    Some(Ok(Message::Text(text))) => {
                        if let Err(e) = handle_text(&ctx, &mut state, &text, &mut sink).await {
                            warn!(error = %e, "ws handler error");
                        }
                    }
                    Some(Ok(Message::Close(_))) | None => break,
                    Some(Ok(Message::Ping(p))) => {
                        if sink.send(Message::Pong(p)).await.is_err() { break; }
                    }
                    _ => {}
                }
            }
            else => break,
        }
    }

    if let (Some(room_id), Some(user_id)) = (state.room_id, state.user_id) {
        if let Some(room) = ctx.rooms.get(room_id) {
            room.remove_participant(user_id);
        }
        ctx.peers.remove(room_id, user_id);
        ctx.forwarder.remove_user(room_id, user_id);
    }
    info!(room = ?state.room_id, user = ?state.user_id, "ws disconnected");
}

#[derive(Default)]
struct ConnectionState {
    room_id: Option<Uuid>,
    user_id: Option<Uuid>,
    event_rx: Option<tokio::sync::broadcast::Receiver<RoomEvent>>,
}

async fn handle_text(
    ctx: &Arc<SfuContext>,
    state: &mut ConnectionState,
    text: &str,
    sink: &mut futures::stream::SplitSink<WebSocket, Message>,
) -> SfuResult<()> {
    let msg: ClientMessage = serde_json::from_str(text)?;
    match msg {
        ClientMessage::Join { room_id, user_id, tenant_id, display_name } => {
            let room = ctx
                .rooms
                .get(room_id)
                .ok_or_else(|| crate::error::SfuError::RoomNotFound(room_id.to_string()))?;
            let participant = Participant {
                id: Uuid::new_v4(),
                user_id,
                tenant_id,
                display_name,
                is_audio_enabled: true,
                is_video_enabled: true,
                is_screen_sharing: false,
                joined_at: chrono::Utc::now(),
                connection_quality: 1.0,
            };
            room.add_participant(participant.clone())?;
            let event_rx = room.subscribe();
            state.room_id = Some(room_id);
            state.user_id = Some(user_id);
            state.event_rx = Some(event_rx);

            let peer = Arc::new(Peer::new(user_id, room_id));
            ctx.peers.insert(peer);

            let ready = ServerMessage::Ready {
                room_id,
                participants: room.list_participants(),
                ice_servers: ctx.config.ice_servers.clone(),
            };
            send(sink, &ready).await?;
        }
        ClientMessage::Leave => {
            if let (Some(room_id), Some(user_id)) = (state.room_id, state.user_id) {
                if let Some(room) = ctx.rooms.get(room_id) {
                    room.remove_participant(user_id);
                }
                ctx.peers.remove(room_id, user_id);
            }
        }
        ClientMessage::Offer { sdp } => {
            if let (Some(room_id), Some(user_id)) = (state.room_id, state.user_id) {
                if let Some(peer) = ctx.peers.get(room_id, user_id) {
                    peer.validate_sdp(&sdp)?;
                    peer.set_remote_offer(sdp.clone()).await?;
                    // Real impl would push through the webrtc PeerConnection.
                    // Here we echo back a deterministic answer for testing.
                    let answer = String::from("v=0\r\no=- 0 0 IN IP4 0.0.0.0\r\ns=-\r\nt=0 0\r\n");
                    peer.set_local_answer(answer.clone()).await?;
                    send(sink, &ServerMessage::Answer { sdp: answer }).await?;
                }
            }
        }
        ClientMessage::Answer { sdp } => {
            if let (Some(room_id), Some(user_id)) = (state.room_id, state.user_id) {
                if let Some(peer) = ctx.peers.get(room_id, user_id) {
                    peer.set_local_answer(sdp).await?;
                }
            }
        }
        ClientMessage::Ice { candidate, sdp_mid, sdp_mline_index } => {
            if let (Some(room_id), Some(user_id)) = (state.room_id, state.user_id) {
                if let Some(peer) = ctx.peers.get(room_id, user_id) {
                    peer.add_ice_candidate(IceCandidate { candidate, sdp_mid, sdp_mline_index }).await?;
                }
            }
        }
        ClientMessage::Mute { audio, video, screen } => {
            if let (Some(room_id), Some(user_id)) = (state.room_id, state.user_id) {
                if let Some(room) = ctx.rooms.get(room_id) {
                    let _ = room.events.send(RoomEvent::AudioMute { user_id, muted: audio });
                    let _ = room.events.send(RoomEvent::VideoToggle { user_id, on: !video });
                    let _ = room.events.send(RoomEvent::ScreenShare { user_id, on: screen });
                }
            }
        }
        ClientMessage::Ping => send(sink, &ServerMessage::Pong).await?,
    }
    Ok(())
}

async fn send(sink: &mut futures::stream::SplitSink<WebSocket, Message>, msg: &ServerMessage) -> SfuResult<()> {
    let s = serde_json::to_string(msg)?;
    sink.send(Message::Text(s))
        .await
        .map_err(|e| crate::error::SfuError::Signaling(e.to_string()))
}
