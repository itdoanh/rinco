use anyhow::Result;
use async_graphql::{Context, EmptySubscription, Object, Schema, SimpleObject};
use async_graphql_axum::GraphQL;
use axum::{extract::ws::{Message, WebSocket, WebSocketUpgrade}, response::IntoResponse, routing::get, Router};
use chrono::{DateTime, Utc};
use dashmap::DashMap;
use futures::{SinkExt, StreamExt};
use gocql::Session;
use parking_lot::RwLock;
use redis::AsyncCommands;
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, net::SocketAddr, sync::Arc};
use tokio::sync::broadcast;
use tower_http::trace::TraceLayer;
use tracing::{error, info, Level};
use tracing_subscriber::FmtSubscriber;
use uuid::Uuid;

// ============================================
// Database Layer
// ============================================

#[derive(Clone)]
pub struct ScyllaDB {
    session: Arc<Session>,
}

impl ScyllaDB {
    pub async fn new(seeds: &[String]) -> Result<Self> {
        let cluster = gocql::Cluster::from(seeds)
            .timeout(std::time::Duration::from_secs(10))
            .build();
        let session = cluster.connect().await?;
        Ok(Self { session: Arc::new(session) })
    }

    pub async fn init_keyspace(&self) -> Result<()> {
        let query = r#"
            CREATE KEYSPACE IF NOT EXISTS rinco_chat
            WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}
        "#;
        self.session.query(query, &[]).await?;
        Ok(())
    }

    pub async fn save_message(&self, msg: &ChatMessage) -> Result<()> {
        let query = r#"
            INSERT INTO rinco_chat.messages_by_conversation (conversation_id, message_id, sender_id, content, created_at, updated_at, is_deleted)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        "#;
        self.session.query(query, &[
            &msg.conversation_id.to_string(),
            &msg.id.to_string(),
            &msg.sender_id.to_string(),
            &msg.content,
            gocql::Timestamp::new(msg.created_at.timestamp()),
            gocql::Timestamp::new(msg.updated_at.timestamp()),
            &(msg.is_deleted as i8),
        ]).await?;
        Ok(())
    }

    pub async fn get_messages(&self, conversation_id: Uuid, limit: i64) -> Result<Vec<ChatMessage>> {
        let query = r#"
            SELECT message_id, sender_id, content, created_at, updated_at, is_deleted
            FROM rinco_chat.messages_by_conversation
            WHERE conversation_id = ?
            ORDER BY created_at DESC
            LIMIT ?
        "#;
        let mut results = self.session.query(query, &[
            &conversation_id.to_string(),
            &limit,
        ]).await?;
        
        let mut messages = Vec::new();
        if let Some(rows) = results.rows() {
            for row in rows {
                let msg_id: String = row.get(0)?;
                let sender_id: String = row.get(1)?;
                let content: String = row.get(2)?;
                let created_ts: i64 = row.get(3)?;
                let updated_ts: i64 = row.get(4)?;
                let is_deleted: i8 = row.get(5)?;
                
                messages.push(ChatMessage {
                    id: Uuid::parse_str(&msg_id).unwrap_or_default(),
                    conversation_id,
                    sender_id: Uuid::parse_str(&sender_id).unwrap_or_default(),
                    content,
                    created_at: DateTime::from_timestamp(created_ts, 0).unwrap_or_else(Utc::now),
                    updated_at: DateTime::from_timestamp(updated_ts, 0).unwrap_or_else(Utc::now),
                    is_deleted: is_deleted != 0,
                });
            }
        }
        messages.reverse();
        Ok(messages)
    }
}

// ============================================
// Models
// ============================================

#[derive(Debug, Clone, Serialize, Deserialize, SimpleObject)]
pub struct ChatMessage {
    pub id: Uuid,
    pub conversation_id: Uuid,
    pub sender_id: Uuid,
    pub content: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub is_deleted: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize, SimpleObject)]
pub struct Conversation {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub name: Option<String>,
    pub is_group: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PresenceState {
    pub user_id: Uuid,
    pub status: String, // "online", "away", "busy", "offline"
    pub last_seen: DateTime<Utc>,
    pub device_info: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TypingIndicator {
    pub user_id: Uuid,
    pub conversation_id: Uuid,
    pub is_typing: bool,
}

// ============================================
// Redis Presence Store
// ============================================

#[derive(Clone)]
pub struct PresenceStore {
    redis: redis::aio::ConnectionManager,
}

impl PresenceStore {
    pub async fn new(redis_url: &str) -> Result<Self> {
        let client = redis::Client::open(redis_url)?;
        let manager = client.get_connection_manager().await?;
        Ok(Self { redis: manager })
    }

    pub async fn set_presence(&self, user_id: Uuid, status: &str) -> Result<()> {
        let mut conn = self.redis.clone();
        let key = format!("presence:{}", user_id);
        conn.set_ex(&key, status, 300).await?;
        Ok(())
    }

    pub async fn get_presence(&self, user_id: Uuid) -> Result<Option<String>> {
        let mut conn = self.redis.clone();
        let key = format!("presence:{}", user_id);
        let status: Option<String> = conn.get(&key).await?;
        Ok(status)
    }

    pub async fn set_typing(&self, user_id: Uuid, conversation_id: Uuid, is_typing: bool) -> Result<()> {
        let mut conn = self.redis.clone();
        let key = format!("typing:{}:{}", conversation_id, user_id);
        if is_typing {
            conn.set_ex(&key, "1", 5).await?; // 5 second TTL
        } else {
            conn.del(&key).await?;
        }
        Ok(())
    }

    pub async fn get_typing_users(&self, conversation_id: Uuid) -> Result<Vec<Uuid>> {
        let mut conn = self.redis.clone();
        let pattern = format!("typing:{}:*", conversation_id);
        let keys: Vec<String> = redis::cmd("KEYS")
            .arg(&pattern)
            .query_async(&mut conn)
            .await?;
        
        let mut user_ids = Vec::new();
        for key in keys {
            if let Some(user_id_str) = key.split(':').last() {
                if let Ok(uuid) = Uuid::parse_str(user_id_str) {
                    user_ids.push(uuid);
                }
            }
        }
        Ok(user_ids)
    }
}

// ============================================
// WebSocket Session Manager
// ============================================

#[derive(Clone)]
pub struct SessionManager {
    connections: Arc<DashMap<Uuid, broadcast::Sender<String>>>,
}

impl SessionManager {
    pub fn new() -> Self {
        Self {
            connections: Arc::new(DashMap::new()),
        }
    }

    pub fn subscribe(&self, user_id: Uuid) -> broadcast::Receiver<String> {
        let (tx, rx) = broadcast::channel(100);
        self.connections.insert(user_id, tx);
        rx
    }

    pub fn unsubscribe(&self, user_id: Uuid) {
        self.connections.remove(&user_id);
    }

    pub fn broadcast_to_user(&self, user_id: Uuid, message: &str) {
        if let Some(sender) = self.connections.get(&user_id) {
            let _ = sender.send(message.to_string());
        }
    }

    pub fn broadcast_to_conversation(&self, user_ids: &[Uuid], message: &str) {
        for user_id in user_ids {
            self.broadcast_to_user(*user_id, message);
        }
    }
}

impl Default for SessionManager {
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
    async fn messages(&self, ctx: &Context<'_>, conversation_id: Uuid, limit: Option<i64>) -> Vec<ChatMessage> {
        let db = ctx.data::<ScyllaDB>().unwrap();
        let limit = limit.unwrap_or(50);
        db.get_messages(conversation_id, limit).await.unwrap_or_default()
    }
}

pub struct MutationRoot;

#[MutationRoot]
impl MutationRoot {
    async fn send_message(&self, ctx: &Context<'_>, input: SendMessageInput) -> ChatMessage {
        let db = ctx.data::<ScyllaDB>().unwrap();
        let presence = ctx.data::<PresenceStore>().unwrap();
        let sessions = ctx.data::<SessionManager>().unwrap();
        
        let now = Utc::now();
        let msg = ChatMessage {
            id: Uuid::new_v4(),
            conversation_id: input.conversation_id,
            sender_id: input.sender_id,
            content: input.content,
            created_at: now,
            updated_at: now,
            is_deleted: false,
        };
        
        // Save to ScyllaDB
        let _ = db.save_message(&msg).await;
        
        // Broadcast to conversation participants
        let json = serde_json::to_string(&msg).unwrap();
        // In real implementation, we'd look up participants from conversation
        sessions.broadcast_to_user(msg.sender_id, &json);
        
        // Clear typing indicator
        let _ = presence.set_typing(msg.sender_id, msg.conversation_id, false).await;
        
        msg
    }

    async fn update_presence(&self, ctx: &Context<'_>, user_id: Uuid, status: String) -> bool {
        let presence = ctx.data::<PresenceStore>().unwrap();
        presence.set_presence(user_id, &status).await.is_ok()
    }

    async fn set_typing(&self, ctx: &Context<'_>, user_id: Uuid, conversation_id: Uuid, is_typing: bool) -> bool {
        let presence = ctx.data::<PresenceStore>().unwrap();
        presence.set_typing(user_id, conversation_id, is_typing).await.is_ok()
    }
}

#[derive(async_graphql::InputObject)]
pub struct SendMessageInput {
    pub conversation_id: Uuid,
    pub sender_id: Uuid,
    pub content: String,
}

pub type ChatSchema = Schema<QueryRoot, MutationRoot, EmptySubscription>;

// ============================================
// HTTP Handlers
// ============================================

async fn graphql_handler(schema: GraphQL<ChatSchema>) -> impl IntoResponse {
    schema
}

async fn websocket_handler(
    ws: WebSocketUpgrade,
    ctx: axum::extract::State<Arc<AppState>>,
    user_id: axum::extract::Query<Uuid>,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| handle_socket(socket, ctx, user_id.0))
}

async fn handle_socket(socket: WebSocket, ctx: Arc<AppState>, user_id: Uuid) {
    let (mut sender, mut receiver) = socket.split();
    let mut rx = ctx.sessions.subscribe(user_id);
    
    info!("WebSocket connected for user: {}", user_id);
    
    // Send initial presence
    let _ = sender.send(Message::Text(serde_json::json!({
        "type": "presence",
        "user_id": user_id.to_string(),
        "status": "online"
    }).to_string())).await;
    
    loop {
        tokio::select! {
            // Handle incoming messages
            msg = receiver.next() => {
                match msg {
                    Some(Ok(Message::Text(text))) => {
                        if let Ok(event) = serde_json::from_str::<serde_json::Value>(&text) {
                            handle_ws_event(&ctx, &mut sender, user_id, event).await;
                        }
                    }
                    Some(Ok(Message::Close(_))) | None => {
                        info!("WebSocket disconnected for user: {}", user_id);
                        break;
                    }
                    _ => {}
                }
            }
            // Handle broadcast messages
            msg = rx.recv() => {
                if let Ok(text) = msg {
                    let _ = sender.send(Message::Text(text)).await;
                }
            }
        }
    }
    
    ctx.sessions.unsubscribe(user_id);
    let _ = ctx.presence.set_presence(user_id, "offline").await;
}

async fn handle_ws_event(
    ctx: &Arc<AppState>,
    sender: &mut futures::channel::mpsc::Sender<Message>,
    user_id: Uuid,
    event: serde_json::Value,
) {
    match event.get("type").and_then(|v| v.as_str()) {
        Some("typing") => {
            let conversation_id = Uuid::parse_str(event["conversation_id"].as_str().unwrap_or("")).unwrap_or_default();
            let is_typing = event["is_typing"].as_bool().unwrap_or(false);
            let _ = ctx.presence.set_typing(user_id, conversation_id, is_typing).await;
        }
        Some("message") => {
            let conversation_id = Uuid::parse_str(event["conversation_id"].as_str().unwrap_or("")).unwrap_or_default();
            let content = event["content"].as_str().unwrap_or("");
            
            let msg = ChatMessage {
                id: Uuid::new_v4(),
                conversation_id,
                sender_id: user_id,
                content: content.to_string(),
                created_at: Utc::now(),
                updated_at: Utc::now(),
                is_deleted: false,
            };
            
            let _ = ctx.db.save_message(&msg).await;
            let json = serde_json::to_string(&msg).unwrap();
            
            // Broadcast to sender for confirmation
            let _ = sender.send(Message::Text(json.clone())).await;
        }
        Some("read_receipt") => {
            // Handle read receipts
            let message_id = Uuid::parse_str(event["message_id"].as_str().unwrap_or("")).unwrap_or_default();
            info!("Message {} read by user {}", message_id, user_id);
        }
        _ => {}
    }
}

// ============================================
// Health Check
// ============================================

async fn health() -> impl IntoResponse {
    axum::Json(serde_json::json!({
        "status": "healthy",
        "service": "chat-engine",
        "timestamp": Utc::now().to_rfc3339()
    }))
}

// ============================================
// Application State
// ============================================

pub struct AppState {
    pub db: ScyllaDB,
    pub presence: PresenceStore,
    pub sessions: SessionManager,
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
    
    info!("Starting Chat Engine service...");
    
    // Initialize ScyllaDB
    let scylla_seeds = std::env::var("SCYLLA_SEEDS").unwrap_or_else(|_| "localhost".to_string());
    let scylla: Vec<String> = scylla_seeds.split(',').map(|s| s.trim().to_string()).collect();
    let db = ScyllaDB::new(&scylla).await?;
    db.init_keyspace().await?;
    
    // Initialize Redis
    let redis_url = std::env::var("REDIS_URL").unwrap_or_else(|_| "redis://localhost:6379".to_string());
    let presence = PresenceStore::new(&redis_url).await?;
    
    // Initialize session manager
    let sessions = SessionManager::new();
    
    let state = Arc::new(AppState {
        db,
        presence,
        sessions,
    });
    
    // Build GraphQL schema
    let schema = Schema::build(QueryRoot, MutationRoot, EmptySubscription)
        .data(state.clone())
        .finish();
    
    // Build router
    let app = Router::new()
        .route("/graphql", get(graphql_handler).post(graphql_handler))
        .route("/ws", get(websocket_handler))
        .route("/health", get(health))
        .layer(TraceLayer::new_for_http())
        .with_state(state);
    
    let addr: SocketAddr = "0.0.0.0:8080".parse()?;
    info!("Chat Engine listening on {}", addr);
    
    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;
    
    Ok(())
}
