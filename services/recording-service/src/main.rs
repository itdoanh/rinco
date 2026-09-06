use anyhow::Result;
use async_graphql::{EmptySubscription, Object, Schema, SimpleObject, InputObject};
use async_graphql_axum::GraphQL;
use axum::{
    body::Body,
    extract::{DefaultBodyLimit, Multipart, Path, State},
    response::IntoResponse,
    routing::{get, post},
    Router,
};
use aws_sdk_s3::presigning::presigned_get;
use aws_sdk_s3::{primitives::ByteStream, Client as S3Client};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::fs::File;
use tokio::io::AsyncWriteExt;
use tower_http::trace::TraceLayer;
use tracing::{error, info, Level};
use tracing_subscriber::FmtSubscriber;
use uuid::Uuid;

// ============================================
// S3/MinIO Client
// ============================================

#[derive(Clone)]
pub struct StorageClient {
    s3: S3Client,
    bucket: String,
}

impl StorageClient {
    pub async fn new(
        endpoint: Option<String>,
        region: String,
        access_key: String,
        secret_key: String,
        bucket: String,
    ) -> Result<Self> {
        let config = aws_config::defaults(aws_config::BehaviorVersion::latest())
            .region(aws_config::Region::new(region.clone()))
            .credentials_provider(aws_config::Credentials::new(
                access_key,
                secret_key,
                None,
                None,
                "env",
            ))
            .endpoint_resolver(aws_config::endpoint::IdentityResolver::new());

        let mut cfg = aws_sdk_s3::Config::builder()
            .behavior_version(aws_sdk_s3::config::BehaviorVersion::latest())
            .region(aws_config::Region::new(region));

        if let Some(ep) = endpoint {
            cfg = cfg.endpoint_url(ep);
        }

        let s3 = Client::from_conf(cfg.build());
        
        Ok(Self { s3, bucket })
    }

    pub async fn upload_file(&self, key: &str, data: Vec<u8>, content_type: &str) -> Result<String> {
        let body = ByteStream::from(data);
        
        self.s3
            .put_object()
            .bucket(&self.bucket)
            .key(key)
            .content_type(content_type)
            .body(body)
            .send()
            .await?;

        Ok(format!("s3://{}/{}", self.bucket, key))
    }

    pub async fn get_presigned_url(&self, key: &str, expires_in_secs: i64) -> Result<String> {
        let presigning_config = aws_sdk_s3::presigning::PresigningConfig::builder()
            .expires_in(std::time::Duration::from_secs(expires_in_secs as u64))
            .build()?;

        let presigned = self.s3
            .get_object()
            .bucket(&self.bucket)
            .key(key)
            .presigned(presigning_config)
            .await?;

        Ok(presigned.uri().to_string())
    }

    pub async fn delete_file(&self, key: &str) -> Result<()> {
        self.s3
            .delete_object()
            .bucket(&self.bucket)
            .key(key)
            .send()
            .await?;
        Ok(())
    }

    pub async fn list_recordings(&self, prefix: &str) -> Result<Vec<RecordingMetadata>> {
        let output = self.s3
            .list_objects_v2()
            .bucket(&self.bucket)
            .prefix(prefix)
            .send()
            .await?;

        let mut recordings = Vec::new();
        if let Some(contents) = output.contents() {
            for obj in contents {
                if let Some(key) = obj.key() {
                    recordings.push(RecordingMetadata {
                        key: key.to_string(),
                        size: obj.size() as u64,
                        last_modified: obj.last_modified().cloned(),
                    });
                }
            }
        }
        Ok(recordings)
    }
}

// ============================================
// Models
// ============================================

#[derive(Debug, Clone, Serialize, Deserialize, SimpleObject)]
pub struct Recording {
    pub id: Uuid,
    pub room_id: Uuid,
    pub tenant_id: Uuid,
    pub user_id: Uuid,
    pub filename: String,
    pub storage_key: String,
    pub content_type: String,
    pub size_bytes: u64,
    pub duration_seconds: u32,
    pub status: RecordingStatus,
    pub created_at: DateTime<Utc>,
    pub completed_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, async_graphql::Enum)]
pub enum RecordingStatus {
    Pending,
    Recording,
    Processing,
    Completed,
    Failed,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecordingMetadata {
    pub key: String,
    pub size: u64,
    pub last_modified: Option<chrono::DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, SimpleObject)]
pub struct UploadResult {
    pub success: bool,
    pub recording_id: Option<Uuid>,
    pub error: Option<String>,
    pub download_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, SimpleObject)]
pub struct RecordingInfo {
    pub id: Uuid,
    pub room_id: Uuid,
    pub filename: String,
    pub size_bytes: u64,
    pub duration_seconds: u32,
    pub status: RecordingStatus,
    pub created_at: DateTime<Utc>,
    pub download_url: Option<String>,
    pub expires_at: Option<DateTime<Utc>>,
}

// ============================================
// In-Memory Store (Replace with PostgreSQL/MongoDB in production)
// ============================================

use parking_lot::RwLock;
use std::collections::HashMap;

#[derive(Clone)]
pub struct RecordingStore {
    recordings: Arc<RwLock<HashMap<Uuid, Recording>>>,
}

impl RecordingStore {
    pub fn new() -> Self {
        Self {
            recordings: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    pub fn save(&self, recording: Recording) {
        let mut recordings = self.recordings.write();
        recordings.insert(recording.id, recording);
    }

    pub fn get(&self, id: Uuid) -> Option<Recording> {
        let recordings = self.recordings.read();
        recordings.get(&id).cloned()
    }

    pub fn get_by_room(&self, room_id: Uuid) -> Vec<Recording> {
        let recordings = self.recordings.read();
        recordings.values()
            .filter(|r| r.room_id == room_id)
            .cloned()
            .collect()
    }

    pub fn update_status(&self, id: Uuid, status: RecordingStatus) -> bool {
        let mut recordings = self.recordings.write();
        if let Some(recording) = recordings.get_mut(&id) {
            recording.status = status;
            if status == RecordingStatus::Completed {
                recording.completed_at = Some(Utc::now());
            }
            true
        } else {
            false
        }
    }
}

impl Default for RecordingStore {
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
    async fn recording(&self, id: Uuid, ctx: &async_graphql::Context<'_>) -> Option<RecordingInfo> {
        let store = ctx.data::<RecordingStore>().ok()?;
        let storage = ctx.data::<StorageClient>().ok()?;
        
        let recording = store.get(id)?;
        
        let download_url = if recording.status == RecordingStatus::Completed {
            storage.get_presigned_url(&recording.storage_key, 3600).await.ok()
        } else {
            None
        };
        
        Some(RecordingInfo {
            id: recording.id,
            room_id: recording.room_id,
            filename: recording.filename,
            size_bytes: recording.size_bytes,
            duration_seconds: recording.duration_seconds,
            status: recording.status,
            created_at: recording.created_at,
            download_url,
            expires_at: None,
        })
    }

    async fn room_recordings(&self, room_id: Uuid, ctx: &async_graphql::Context<'_>) -> Vec<RecordingInfo> {
        let store = ctx.data::<RecordingStore>().ok().cloned().unwrap_or_default();
        let storage = ctx.data::<StorageClient>().ok().cloned().unwrap_or_default();
        
        store.get_by_room(room_id)
            .into_iter()
            .map(|r| {
                let download_url = if r.status == RecordingStatus::Completed {
                    futures::executor::block_on(storage.get_presigned_url(&r.storage_key, 3600)).ok()
                } else {
                    None
                };
                
                RecordingInfo {
                    id: r.id,
                    room_id: r.room_id,
                    filename: r.filename,
                    size_bytes: r.size_bytes,
                    duration_seconds: r.duration_seconds,
                    status: r.status,
                    created_at: r.created_at,
                    download_url,
                    expires_at: None,
                }
            })
            .collect()
    }
}

pub struct MutationRoot;

#[MutationRoot]
impl MutationRoot {
    async fn create_recording(&self, ctx: &async_graphql::Context<'_>, input: CreateRecordingInput) -> RecordingInfo {
        let store = ctx.data::<RecordingStore>().cloned().unwrap_or_default();
        
        let id = Uuid::new_v4();
        let storage_key = format!("recordings/{}/{}/{}/{}", input.tenant_id, input.room_id, id, input.filename);
        
        let recording = Recording {
            id,
            room_id: input.room_id,
            tenant_id: input.tenant_id,
            user_id: input.user_id,
            filename: input.filename,
            storage_key,
            content_type: input.content_type,
            size_bytes: 0,
            duration_seconds: 0,
            status: RecordingStatus::Pending,
            created_at: Utc::now(),
            completed_at: None,
        };
        
        store.save(recording.clone());
        
        RecordingInfo {
            id: recording.id,
            room_id: recording.room_id,
            filename: recording.filename,
            size_bytes: 0,
            duration_seconds: 0,
            status: recording.status,
            created_at: recording.created_at,
            download_url: None,
            expires_at: None,
        }
    }

    async fn update_recording(&self, ctx: &async_graphql::Context<'_>, id: Uuid, size_bytes: u64, duration_seconds: u32) -> bool {
        let store = ctx.data::<RecordingStore>().ok().cloned().unwrap_or_default();
        
        let mut recordings = store.recordings.write();
        if let Some(recording) = recordings.get_mut(&id) {
            recording.size_bytes = size_bytes;
            recording.duration_seconds = duration_seconds;
            recording.status = RecordingStatus::Completed;
            recording.completed_at = Some(Utc::now());
            true
        } else {
            false
        }
    }
}

#[derive(InputObject)]
pub struct CreateRecordingInput {
    pub room_id: Uuid,
    pub tenant_id: Uuid,
    pub user_id: Uuid,
    pub filename: String,
    pub content_type: String,
}

pub type RecordingSchema = Schema<QueryRoot, MutationRoot, EmptySubscription>;

// ============================================
// HTTP Handlers
// ============================================

#[derive(Clone)]
pub struct AppState {
    pub storage: StorageClient,
    pub store: RecordingStore,
}

async fn graphql_handler(schema: GraphQL<RecordingSchema>) -> impl IntoResponse {
    schema
}

async fn upload_handler(
    State(state): State<Arc<AppState>>,
    mut multipart: Multipart,
) -> impl IntoResponse {
    let mut recording_id: Option<Uuid> = None;
    let mut filename: Option<String> = None;
    let mut room_id: Option<Uuid> = None;
    let mut tenant_id: Option<Uuid> = None;
    let mut data: Vec<u8> = Vec::new();

    while let Some(field) = multipart.next_field().await.unwrap_or(None) {
        let name = field.name().unwrap_or("").to_string();
        
        match name.as_str() {
            "recording_id" => {
                if let Ok(text) = field.text().await {
                    recording_id = Uuid::parse_str(&text).ok();
                }
            }
            "filename" => {
                if let Ok(text) = field.text().await {
                    filename = Some(text);
                }
            }
            "room_id" => {
                if let Ok(text) = field.text().await {
                    room_id = Uuid::parse_str(&text).ok();
                }
            }
            "tenant_id" => {
                if let Ok(text) = field.text().await {
                    tenant_id = Uuid::parse_str(&text).ok();
                }
            }
            "file" => {
                let bytes = field.bytes().await.unwrap_or_default();
                data.extend_from_slice(&bytes);
            }
            _ => {}
        }
    }

    if data.is_empty() {
        return axum::Json(serde_json::json!({
            "success": false,
            "error": "No file data received"
        }));
    }

    let rec_id = recording_id.unwrap_or_else(Uuid::new_v4);
    let fname = filename.unwrap_or_else(|| format!("{}.webm", rec_id));
    let r_id = room_id.unwrap_or_else(Uuid::nil);
    let t_id = tenant_id.unwrap_or_else(Uuid::nil);

    let storage_key = format!("recordings/{}/{}/{}/{}", t_id, r_id, rec_id, fname);
    
    // Upload to S3/MinIO
    match state.storage.upload_file(&storage_key, data.clone(), "video/webm").await {
        Ok(_) => {
            info!("Recording {} uploaded successfully", rec_id);
            
            // Update recording status
            state.store.update_status(rec_id, RecordingStatus::Completed);
            
            // Get presigned download URL
            let download_url = state.storage.get_presigned_url(&storage_key, 3600).await.ok();
            
            axum::Json(UploadResult {
                success: true,
                recording_id: Some(rec_id),
                error: None,
                download_url,
            })
        }
        Err(e) => {
            error!("Failed to upload recording: {}", e);
            axum::Json(UploadResult {
                success: false,
                recording_id: Some(rec_id),
                error: Some(e.to_string()),
                download_url: None,
            })
        }
    }
}

async fn download_handler(
    State(state): State<Arc<AppState>>,
    Path(recording_id): Path<Uuid>,
) -> impl IntoResponse {
    let recording = state.store.get(recording_id);
    
    match recording {
        Some(rec) => {
            match state.storage.get_presigned_url(&rec.storage_key, 3600).await {
                Ok(url) => {
                    axum::Json(serde_json::json!({
                        "download_url": url,
                        "expires_in": 3600
                    }))
                }
                Err(e) => {
                    axum::Json(serde_json::json!({
                        "error": e.to_string()
                    }))
                }
            }
        }
        None => {
            axum::Json(serde_json::json!({
                "error": "Recording not found"
            }))
        }
    }
}

async fn health() -> impl IntoResponse {
    axum::Json(serde_json::json!({
        "status": "healthy",
        "service": "recording-service",
        "timestamp": Utc::now().to_rfc3339()
    }))
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
    
    info!("Starting Recording Service...");
    
    // Initialize S3/MinIO client
    let endpoint = std::env::var("MINIO_ENDPOINT").ok();
    let region = std::env::var("AWS_REGION").unwrap_or_else(|_| "us-east-1".to_string());
    let access_key = std::env::var("MINIO_ACCESS_KEY").unwrap_or_else(|_| "minioadmin".to_string());
    let secret_key = std::env::var("MINIO_SECRET_KEY").unwrap_or_else(|_| "minioadmin".to_string());
    let bucket = std::env::var("MINIO_BUCKET").unwrap_or_else(|_| "recordings".to_string());
    
    let storage = StorageClient::new(endpoint, region, access_key, secret_key, bucket).await?;
    let store = RecordingStore::new();
    
    let state = Arc::new(AppState { storage, store });
    
    // Build GraphQL schema
    let schema = Schema::build(QueryRoot, MutationRoot, EmptySubscription)
        .data(state.clone())
        .finish();
    
    // Build router
    let app = Router::new()
        .route("/graphql", get(graphql_handler).post(graphql_handler))
        .route("/upload", post(upload_handler))
        .route("/download/:recording_id", get(download_handler))
        .route("/health", get(health))
        .layer(TraceLayer::new_for_http())
        .layer(DefaultBodyLimit::max(1024 * 1024 * 1024)) // 1GB limit
        .with_state(state);
    
    let addr: SocketAddr = "0.0.0.0:8083".parse()?;
    info!("Recording Service listening on {}", addr);
    
    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;
    
    Ok(())
}
