//! Error and Result aliases used across chat-engine.

use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;
use thiserror::Error;

/// Crate-wide result alias.
pub type ChatResult<T> = std::result::Result<T, ChatError>;

/// All errors surfaced by the chat-engine crate.
#[derive(Debug, Error)]
pub enum ChatError {
    /// Configuration loading failed.
    #[error("configuration error: {0}")]
    Config(String),

    /// Database (ScyllaDB) failure.
    #[error("scylla error: {0}")]
    Scylla(String),

    /// Cache (Valkey/Redis) failure.
    #[error("valkey error: {0}")]
    Valkey(String),

    /// NATS messaging failure.
    #[error("nats error: {0}")]
    Nats(String),

    /// WebSocket protocol error.
    #[error("websocket error: {0}")]
    WebSocket(String),

    /// Connect-RPC serialization / transport error.
    #[error("rpc error: {0}")]
    Rpc(String),

    /// Cryptographic operation failed (decryption, signature mismatch, …).
    #[error("crypto error: {0}")]
    Crypto(String),

    /// Caller supplied invalid input.
    #[error("invalid request: {0}")]
    InvalidRequest(String),

    /// Caller is not authenticated for the requested operation.
    #[error("unauthorized: {0}")]
    Unauthorized(String),

    /// Resource not found.
    #[error("not found: {0}")]
    NotFound(String),

    /// Underlying `anyhow` error.
    #[error(transparent)]
    Other(#[from] anyhow::Error),
}

impl ChatError {
    /// Map the error to an HTTP status code.
    pub fn status_code(&self) -> StatusCode {
        match self {
            ChatError::Config(_)
            | ChatError::Scylla(_)
            | ChatError::Valkey(_)
            | ChatError::Nats(_)
            | ChatError::WebSocket(_)
            | ChatError::Rpc(_)
            | ChatError::Crypto(_)
            | ChatError::Other(_) => StatusCode::INTERNAL_SERVER_ERROR,
            ChatError::InvalidRequest(_) => StatusCode::BAD_REQUEST,
            ChatError::Unauthorized(_) => StatusCode::UNAUTHORIZED,
            ChatError::NotFound(_) => StatusCode::NOT_FOUND,
        }
    }
}

impl IntoResponse for ChatError {
    fn into_response(self) -> Response {
        let status = self.status_code();
        let body = Json(json!({
            "error": self.to_string(),
            "code": status.as_u16(),
        }));
        (status, body).into_response()
    }
}

impl From<redis::RedisError> for ChatError {
    fn from(e: redis::RedisError) -> Self {
        ChatError::Valkey(e.to_string())
    }
}

impl From<serde_json::Error> for ChatError {
    fn from(e: serde_json::Error) -> Self {
        ChatError::InvalidRequest(e.to_string())
    }
}
