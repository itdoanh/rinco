use anyhow::Result;
use async_graphql::{EmptySubscription, Object, Schema, SimpleObject, InputObject};
use async_graphql_axum::GraphQL;
use axum::{extract::ws::{Message, WebSocket, WebSocketUpgrade}, response::IntoResponse, routing::get, Router};
use chrono::{DateTime, Utc};
use dashmap::DashMap;
use futures::{SinkExt, StreamExt};
use parking_lot::RwLock;
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, net::SocketAddr, sync::Arc};
use tokio::sync::{broadcast, RwLock as TokioRwLock};
use tower_http::trace::TraceLayer;
use tracing::{error, info, warn, Level};
use tracing_subscriber::FmtSubscriber;
use uuid::Uuid;

// ============================================
// WebRTC Room Management
// ============================================

#[derive(Debug, Clone)]
pub struct Participant {
    pub id: Uuid,
    pub user_id: Uuid,
    pub tenant_id: Uuid,
    pub display_name: String,
    pub is_audio_enabled: bool,
    pub is_video_enabled: bool,
    pub is_screen_sharing: bool,
    pub joined_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SDP Offer {
    pub sdp: String,
    pub video_codec: String,
    pub audio_codec: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ICE Candidate {
    pub candidate: String,
    pub sdp_mid: Option<String>,
    pub sdp_mline_index: Option<u16>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum SignalingMessage {
    Join { room_id: Uuid, user_id: Uuid, display_name: String },
    Leave { room_id: Uuid, user_id: Uuid },
    SDP { room_id: Uuid, from: Uuid, to: Uuid, sdp: SDP Offer },
    ICE { room_id: Uuid, from: Uuid, to: Uuid, candidate: ICE Candidate },
    Mute { room_id: Uuid, user_id: Uuid, media: String },
    Unmute { room_id: Uuid, user_id: Uuid, media: String },
    VideoOn { room_id: Uuid, user_id: Uuid },
    VideoOff { room_id: Uuid, user_id: Uuid },
    ScreenShareStart { room_id: Uuid, user_id: Uuid },
    ScreenShareStop { room_id: Uuid, user_id: Uuid },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum SignalingResponse {
    RoomJoined { room_id: Uuid, participants: Vec<Participant> },
    ParticipantJoined { participant: Participant },
    ParticipantLeft { user_id: Uuid },
    SDP { from: Uuid, sdp: SDP Offer },
    ICE { from: Uuid, candidate: ICE Candidate },
    MuteChanged { user_id: Uuid, media: String },
    VideoChanged { user_id: Uuid, is_enabled: bool },
    ScreenShareChanged { user_id: Uuid, is_sharing: bool },
    Error { code: u32, message: String },
}

#[derive(Debug, Clone)]
pub struct Room {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub name: String,
    pub created_at: DateTime<Utc>,
    pub participants: Arc<TokioRwLock<HashMap<Uuid, Participant>>>,
    pub events: broadcast::Sender<String>,
}

impl Room {
    pub fn new(id: Uuid, tenant_id: Uuid, name: String) -> Self {
        let (tx, _) = broadcast::channel(1000);
        Self {
            id,
            tenant_id,
            name,
            created_at: Utc::now(),
            participants: Arc::new(TokioRwLock::new(HashMap::new())),
            events: tx,
        }
    }

    pub async fn add_participant(&self, participant: Participant) {
        let mut participants = self.participants.write().await;
        participants.insert(participant.user_id, participant);
    }

    pub async fn remove_participant(&self, user_id: Uuid) -> Option<Participant> {
        let mut participants = self.participants.write().await;
        participants.remove(&user_id)
    }

    pub async fn get_participants(&self) -> Vec<Participant> {
        let participants = self.participants.read().await;
        participants.values().cloned().collect()
    }

    pub async fn participant_count(&self) -> usize {
        let participants = self.participants.read().await;
        participants.len()
    }
}

// ============================================
// Room Manager
// ============================================

#[derive(Clone)]
pub struct RoomManager {
    rooms: Arc<DashMap<Uuid, Arc<Room>>>,
}

impl RoomManager {
    pub fn new() -> Self {
        Self {
            rooms: Arc::new(DashMap::new()),
        }
    }

    pub async fn create_room(&self, id: Uuid, tenant_id: Uuid, name: String) -> Arc<Room> {
        let room = Arc::new(Room::new(id, tenant_id, name));
        self.rooms.insert(id, room.clone());
        info!("Room {} created for tenant {}", id, tenant_id);
        room
    }

    pub fn get_room(&self, id: Uuid) -> Option<Arc<Room>> {
        self.rooms.get(&id).map(|r| r.clone())
    }

    pub async fn delete_room(&self, id: Uuid) -> bool {
        if let Some(room) = self.rooms.remove(&id) {
            let participants = room.participants.read().await;
            info!("Room {} deleted with {} participants", id, participants.len());
            true
        } else {
            false
        }
    }

    pub fn list_rooms(&self) -> Vec<Uuid> {
        self.rooms.iter().map(|r| *r.key()).collect()
    }

    pub fn room_count(&self) -> usize {
        self.rooms.len()
    }

    pub async fn cleanup_empty_rooms(&self) {
        let empty: Vec<Uuid> = self.rooms.iter()
            .filter(|r| r.participants.try_read().map(|p| p.is_empty()).unwrap_or(true))
            .map(|r| *r.key())
            .collect();
        
        for room_id in empty {
            let _ = self.delete_room(room_id).await;
        }
    }
}

impl Default for RoomManager {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================
// WebRTC Peer Connection Manager
// ============================================

#[derive(Clone)]
pub struct PeerManager {
    peers: Arc<DashMap<(Uuid, Uuid), PeerConnection>>,
}

pub struct PeerConnection {
    pub user_id: Uuid,
    pub room_id: Uuid,
    pub has_audio: bool,
    pub has_video: bool,
    pub has_screen_share: bool,
}

impl PeerManager {
    pub fn new() -> Self {
        Self {
            peers: Arc::new(DashMap::new()),
        }
    }

    pub fn add_peer(&self, room_id: Uuid, user_id: Uuid, peer: PeerConnection) {
        self.peers.insert((room_id, user_id), peer);
    }

    pub fn remove_peer(&self, room_id: Uuid, user_id: Uuid) {
        self.peers.remove(&(room_id, user_id));
    }

    pub fn get_peer(&self, room_id: Uuid, user_id: Uuid) -> Option<PeerConnection> {
        self.peers.get(&(room_id, user_id)).map(|p| p.clone())
    }

    pub fn room_peers(&self, room_id: Uuid) -> Vec<PeerConnection> {
        self.peers.iter()
            .filter(|p| p.key().0 == room_id)
            .map(|p| p.clone())
            .collect()
    }
}

impl Default for PeerManager {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================
// GraphQL Schema
// ============================================

pub struct QueryRoot;

#[Object]
impl QueryRoot {
    async fn room_info(&self, room_id: Uuid, ctx: &async_graphql::Context<'_>) -> Option<RoomInfo> {
        let manager = ctx.data::<RoomManager>().ok()?;
        let room = manager.get_room(room_id)?;
        let participants = room.get_participants().await;
        
        Some(RoomInfo {
            id: room.id,
            name: room.name,
            tenant_id: room.tenant_id,
            participant_count: participants.len() as i32,
            created_at: room.created_at,
        })
    }

    async fn active_rooms(&self, ctx: &async_graphql::Context<'_>) -> Vec<RoomInfo> {
        let manager = ctx.data::<RoomManager>().ok().cloned().unwrap_or_default();
        let room_ids = manager.list_rooms();
        
        let mut rooms = Vec::new();
        for room_id in room_ids {
            if let Some(room) = manager.get_room(room_id) {
                let participants = room.get_participants().await;
                rooms.push(RoomInfo {
                    id: room.id,
                    name: room.name.clone(),
                    tenant_id: room.tenant_id,
                    participant_count: participants.len() as i32,
                    created_at: room.created_at,
                });
            }
        }
        rooms
    }
}

pub struct MutationRoot;

#[MutationRoot]
impl MutationRoot {
    async fn create_room(&self, ctx: &async_graphql::Context<'_>, input: CreateRoomInput) -> RoomInfo {
        let manager = ctx.data::<RoomManager>().cloned().unwrap_or_default();
        let room = manager.create_room(Uuid::new_v4(), input.tenant_id, input.name).await;
        
        RoomInfo {
            id: room.id,
            name: room.name.clone(),
            tenant_id: room.tenant_id,
            participant_count: 0,
            created_at: room.created_at,
        }
    }

    async fn join_room(&self, ctx: &async_graphql::Context<'_>, input: JoinRoomInput) -> JoinResult {
        let manager = ctx.data::<RoomManager>().cloned().unwrap_or_default();
        
        let room = match manager.get_room(input.room_id) {
            Some(r) => r,
            None => {
                return JoinResult {
                    success: false,
                    error: Some("Room not found".to_string()),
                    participants: vec![],
                };
            }
        };
        
        let participant = Participant {
            id: Uuid::new_v4(),
            user_id: input.user_id,
            tenant_id: input.tenant_id,
            display_name: input.display_name,
            is_audio_enabled: true,
            is_video_enabled: true,
            is_screen_sharing: false,
            joined_at: Utc::now(),
        };
        
        room.add_participant(participant.clone()).await;
        
        JoinResult {
            success: true,
            error: None,
            participants: room.get_participants().await,
        }
    }

    async fn leave_room(&self, ctx: &async_graphql::Context<'_>, room_id: Uuid, user_id: Uuid) -> bool {
        let manager = ctx.data::<RoomManager>().cloned().unwrap_or_default();
        
        if let Some(room) = manager.get_room(room_id) {
            room.remove_participant(user_id).await;
            
            // If room is empty, mark for cleanup
            if room.participant_count().await == 0 {
                warn!("Room {} is now empty", room_id);
            }
            
            true
        } else {
            false
        }
    }
}

#[derive(SimpleObject)]
pub struct RoomInfo {
    pub id: Uuid,
    pub name: String,
    pub tenant_id: Uuid,
    pub participant_count: i32,
    pub created_at: DateTime<Utc>,
}

#[derive(InputObject)]
pub struct CreateRoomInput {
    pub tenant_id: Uuid,
    pub name: String,
}

#[derive(InputObject)]
pub struct JoinRoomInput {
    pub room_id: Uuid,
    pub user_id: Uuid,
    pub tenant_id: Uuid,
    pub display_name: String,
}

#[derive(SimpleObject)]
pub struct JoinResult {
    pub success: bool,
    pub error: Option<String>,
    pub participants: Vec<Participant>,
}

pub type WebRTCSchema = Schema<QueryRoot, MutationRoot, EmptySubscription>;

// ============================================
// HTTP Handlers
// ============================================

async fn graphql_handler(schema: GraphQL<WebRTCSchema>) -> impl IntoResponse {
    schema
}

async fn websocket_handler(
    ws: WebSocketUpgrade,
    ctx: axum::extract::State<Arc<AppState>>,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| handle_signaling(socket, ctx))
}

async fn handle_signaling(socket: WebSocket, ctx: Arc<AppState>) {
    let (mut sender, mut receiver) = socket.split();
    let mut rx = ctx.room_manager.rooms.iter()
        .next()
        .map(|r| r.1.events.subscribe());
    
    // Subscribe to all room events
    let mut subscriptions: Vec<broadcast::Receiver<String>> = Vec::new();
    for room in ctx.room_manager.rooms.iter() {
        subscriptions.push(room.1.events.subscribe());
    }
    
    info!("New WebSocket connection for signaling");
    
    loop {
        tokio::select! {
            // Handle incoming messages
            msg = receiver.next() => {
                match msg {
                    Some(Ok(Message::Text(text))) => {
                        if let Ok(signal) = serde_json::from_str::<SignalingMessage>(&text) {
                            handle_signaling_message(&ctx, &mut sender, signal).await;
                        }
                    }
                    Some(Ok(Message::Close(_))) | None => {
                        info!("WebSocket disconnected");
                        break;
                    }
                    _ => {}
                }
            }
            // Handle room events
            _ = tokio::time::sleep(tokio::time::Duration::from_millis(100)) => {
                // This is a simplified event handling
            }
        }
    }
}

async fn handle_signaling_message(
    ctx: &Arc<AppState>,
    sender: &mut futures::channel::mpsc::Sender<Message>,
    msg: SignalingMessage,
) {
    match msg {
        SignalingMessage::Join { room_id, user_id, display_name } => {
            if let Some(room) = ctx.room_manager.get_room(room_id) {
                let participant = Participant {
                    id: Uuid::new_v4(),
                    user_id,
                    tenant_id: Uuid::nil(), // Should be from auth
                    display_name,
                    is_audio_enabled: true,
                    is_video_enabled: true,
                    is_screen_sharing: false,
                    joined_at: Utc::now(),
                };
                
                room.add_participant(participant.clone()).await;
                
                // Notify others
                let response = SignalingResponse::RoomJoined {
                    room_id,
                    participants: room.get_participants().await,
                };
                let _ = sender.send(Message::Text(serde_json::to_string(&response).unwrap())).await;
                
                // Broadcast to room
                let notification = SignalingResponse::ParticipantJoined { participant };
                let _ = room.events.send(serde_json::to_string(&notification).unwrap());
            }
        }
        
        SignalingMessage::Leave { room_id, user_id } => {
            if let Some(room) = ctx.room_manager.get_room(room_id) {
                if room.remove_participant(user_id).await.is_some() {
                    let notification = SignalingResponse::ParticipantLeft { user_id };
                    let _ = room.events.send(serde_json::to_string(&notification).unwrap());
                }
            }
        }
        
        SignalingMessage::SDP { room_id, from, to, sdp } => {
            if let Some(room) = ctx.room_manager.get_room(room_id) {
                let response = SignalingResponse::SDP { from, sdp };
                let _ = room.events.send(serde_json::to_string(&response).unwrap());
            }
        }
        
        SignalingMessage::ICE { room_id, from, to, candidate } => {
            if let Some(room) = ctx.room_manager.get_room(room_id) {
                let response = SignalingResponse::ICE { from, candidate };
                let _ = room.events.send(serde_json::to_string(&response).unwrap());
            }
        }
        
        SignalingMessage::Mute { room_id, user_id, media } => {
            if let Some(room) = ctx.room_manager.get_room(room_id) {
                let response = SignalingResponse::MuteChanged { user_id, media };
                let _ = room.events.send(serde_json::to_string(&response).unwrap());
            }
        }
        
        SignalingMessage::VideoOn { room_id, user_id } | SignalingMessage::VideoOff { room_id, user_id } => {
            if let Some(room) = ctx.room_manager.get_room(room_id) {
                let is_enabled = matches!(msg, SignalingMessage::VideoOn { .. });
                let response = SignalingResponse::VideoChanged { user_id, is_enabled };
                let _ = room.events.send(serde_json::to_string(&response).unwrap());
            }
        }
        
        SignalingMessage::ScreenShareStart { room_id, user_id } | SignalingMessage::ScreenShareStop { room_id, user_id } => {
            if let Some(room) = ctx.room_manager.get_room(room_id) {
                let is_sharing = matches!(msg, SignalingMessage::ScreenShareStart { .. });
                let response = SignalingResponse::ScreenShareChanged { user_id, is_sharing };
                let _ = room.events.send(serde_json::to_string(&response).unwrap());
            }
        }
    }
}

async fn health() -> impl IntoResponse {
    axum::Json(serde_json::json!({
        "status": "healthy",
        "service": "webrtc-sfu",
        "timestamp": Utc::now().to_rfc3339(),
        "active_rooms": 0
    }))
}

// ============================================
// Application State
// ============================================

pub struct AppState {
    pub room_manager: RoomManager,
    pub peer_manager: PeerManager,
}

impl Default for AppState {
    fn default() -> Self {
        Self {
            room_manager: RoomManager::new(),
            peer_manager: PeerManager::new(),
        }
    }
}

// ============================================
// Main Entry Point
// ============================================

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging
    let subscriber = FmtSubscriber::builder()
        .with_max_level(Level::INFO)
        .with_target(false)
        .init();
    
    info!("Starting WebRTC SFU service...");
    
    let state = Arc::new(AppState::default());
    
    // Build GraphQL schema
    let schema = Schema::build(QueryRoot, MutationRoot, EmptySubscription)
        .data(state.clone())
        .finish();
    
    // Build router
    let app = Router::new()
        .route("/graphql", get(graphql_handler).post(graphql_handler))
        .route("/ws/signaling", get(websocket_handler))
        .route("/health", get(health))
        .layer(TraceLayer::new_for_http())
        .with_state(state);
    
    let addr: SocketAddr = "0.0.0.0:8082".parse()?;
    info!("WebRTC SFU listening on {}", addr);
    
    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;
    
    Ok(())
}
