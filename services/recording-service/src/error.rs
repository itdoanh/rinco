//! Error type for the recording service.

use thiserror::Error;

pub type RecordingResult<T> = std::result::Result<T, RecordingError>;

#[derive(Debug, Error)]
pub enum RecordingError {
    #[error("storage error: {0}")]
    Storage(String),

    #[error("database error: {0}")]
    Database(String),

    #[error("ffmpeg error: {0}")]
    Ffmpeg(String),

    #[error("gpu error: {0}")]
    Gpu(String),

    #[error("upload error: {0}")]
    Upload(String),

    #[error("not found: {0}")]
    NotFound(String),

    #[error("invalid request: {0}")]
    InvalidRequest(String),

    #[error(transparent)]
    Other(#[from] anyhow::Error),
}
