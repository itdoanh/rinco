//! Recording Service - thu recording từ SFU, upload lên MinIO/S3.

use std::sync::Arc;

use axum::{
    extract::{Multipart, State},
    http::StatusCode,
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use chrono::Utc;
use dashmap::DashMap;
use serde::{Deserialize, Serialize};
use tracing::{error, info};
use uuid::Uuid;

#[derive(Clone)]
pub struct AppState {
    pub recordings: Arc<DashMap<String, Recording>>,
    pub s3_client: aws_sdk_s3::Client,
    pub bucket: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Recording {
    pub recording_id: String,
    pub session_id: String,
    pub tenant_id: String,
    pub started_at: i64,
    pub ended_at: Option<i64>,
    pub status: String,
    pub size_bytes: u64,
    pub s3_key: String,
    pub download_url: Option<String>,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter("info")
        .json()
        .init();

    info!("starting recording service");

    // Init S3 (MinIO)
    let endpoint = std::env::var("S3_ENDPOINT").unwrap_or_else(|_| "http://localhost:9000".into());
    let access_key = std::env::var("S3_ACCESS_KEY").unwrap_or_else(|_| "rinco".into());
    let secret_key = std::env::var("S3_SECRET_KEY").unwrap_or_else(|_| "rinco_dev_password".into());
    let bucket = std::env::var("S3_BUCKET").unwrap_or_else(|_| "recordings".into());

    let creds = aws_credential_types::Credentials::new(access_key, secret_key, None, None, "static");
    let config = aws_config::defaults(aws_config::BehaviorVersion::latest())
        .endpoint_url(endpoint)
        .region(aws_config::Region::new("us-east-1"))
        .credentials_provider(creds)
        .load()
        .await;
    let s3_client = aws_sdk_s3::Client::new(&config);

    let state = AppState {
        recordings: Arc::new(DashMap::new()),
        s3_client,
        bucket,
    };

    let app = Router::new()
        .route("/health", get(health))
        .route("/metrics", get(metrics_handler))
        .route("/v1/recordings", post(upload_recording))
        .route("/v1/recordings/:id", get(get_recording))
        .route("/v1/recordings/:id/url", get(get_download_url))
        .with_state(state);

    let listener = tokio::net::TcpListener::bind(&format!(
        "0.0.0.0:{}",
        std::env::var("PORT").unwrap_or_else(|_| "8096".into())
    ))
    .await?;

    info!("recording service listening on {}", listener.local_addr()?);
    axum::serve(listener, app).await?;
    Ok(())
}

async fn health() -> &'static str {
    "ok"
}

async fn metrics_handler() -> impl IntoResponse {
    let metrics = prometheus::gather();
    let encoder = prometheus::TextEncoder::new();
    match encoder.encode_to_string(&metrics) {
        Ok(s) => (StatusCode::OK, s).into_response(),
        Err(e) => (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()).into_response(),
    }
}

#[derive(Debug, Deserialize)]
pub struct UploadMetadata {
    pub session_id: String,
    pub tenant_id: String,
    pub started_at: i64,
    pub ended_at: i64,
}

async fn upload_recording(
    State(state): State<AppState>,
    mut multipart: Multipart,
) -> impl IntoResponse {
    let mut metadata: Option<UploadMetadata> = None;
    let mut file_data: Option<Vec<u8>> = None;
    let mut filename = String::new();

    while let Some(field) = multipart.next_field().await.unwrap_or(None) {
        let name = field.name().unwrap_or("").to_string();
        match name.as_str() {
            "metadata" => {
                let text = field.text().await.unwrap_or_default();
                metadata = serde_json::from_str(&text).ok();
            }
            "file" => {
                filename = field.file_name().unwrap_or("recording.webm").to_string();
                let bytes = field.bytes().await.unwrap_or_default();
                file_data = Some(bytes.to_vec());
            }
            _ => {}
        }
    }

    let meta = match metadata {
        Some(m) => m,
        None => {
            return (
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({"error": "metadata required"})),
            )
        }
    };

    let data = match file_data {
        Some(d) => d,
        None => {
            return (
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({"error": "file required"})),
            )
        }
    };

    let recording_id = Uuid::now_v7().to_string();
    let s3_key = format!(
        "{}/{}/{}-{}",
        meta.tenant_id, meta.session_id, recording_id, filename
    );

    // Upload to S3
    use aws_sdk_s3::types::ByteStream;
    let body = ByteStream::from(data.clone());

    match state
        .s3_client
        .put_object()
        .bucket(&state.bucket)
        .key(&s3_key)
        .body(body)
        .content_type("video/webm")
        .send()
        .await
    {
        Ok(_) => {
            let recording = Recording {
                recording_id: recording_id.clone(),
                session_id: meta.session_id.clone(),
                tenant_id: meta.tenant_id.clone(),
                started_at: meta.started_at,
                ended_at: Some(meta.ended_at),
                status: "uploaded".into(),
                size_bytes: data.len() as u64,
                s3_key: s3_key.clone(),
                download_url: None,
            };
            state.recordings.insert(recording_id.clone(), recording.clone());

            info!("recording uploaded: {} ({} bytes)", recording_id, data.len());
            (StatusCode::CREATED, Json(recording))
        }
        Err(e) => {
            error!("S3 upload failed: {}", e);
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e.to_string()})),
            )
        }
    }
}

async fn get_recording(
    State(state): State<AppState>,
    axum::extract::Path(id): axum::extract::Path<String>,
) -> impl IntoResponse {
    match state.recordings.get(&id) {
        Some(r) => (StatusCode::OK, Json(r.clone())),
        None => (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "not found"})),
        ),
    }
}

async fn get_download_url(
    State(state): State<AppState>,
    axum::extract::Path(id): axum::extract::Path<String>,
) -> impl IntoResponse {
    let recording = match state.recordings.get(&id) {
        Some(r) => r.clone(),
        None => {
            return (
                StatusCode::NOT_FOUND,
                Json(serde_json::json!({"error": "not found"})),
            )
        }
    };

    // Generate presigned URL (valid 1 hour)
    use aws_sdk_s3::presigning::PresigningConfig;
    let presign_config = PresigningConfig::expires_in(std::time::Duration::from_secs(3600));

    match state
        .s3_client
        .get_object()
        .bucket(&state.bucket)
        .key(&recording.s3_key)
        .presigned(presign_config)
        .await
    {
        Ok(presigned) => (
            StatusCode::OK,
            Json(serde_json::json!({
                "url": presigned.uri().to_string(),
                "expires_in": 3600,
            })),
        ),
        Err(e) => (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(serde_json::json!({"error": e.to_string()})),
        ),
    }
}
