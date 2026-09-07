//! Error type for the SFU service.

use thiserror::Error;

/// SFU result alias.
pub type SfuResult<T> = std::result::Result<T, SfuError>;

/// All errors surfaced by the SFU crate.
#[derive(Debug, Error)]
pub enum SfuError {
    #[error("room not found: {0}")]
    RoomNotFound(String),

    #[error("participant not found: {0}")]
    ParticipantNotFound(String),

    #[error("signaling error: {0}")]
    Signaling(String),

    #[error("sdp error: {0}")]
    Sdp(String),

    #[error("ice error: {0}")]
    Ice(String),

    #[error("forwarder error: {0}")]
    Forwarder(String),

    #[error("egress error: {0}")]
    Egress(String),

    #[error("invalid request: {0}")]
    InvalidRequest(String),

    #[error("forbidden: {0}")]
    Forbidden(String),

    #[error(transparent)]
    Other(#[from] anyhow::Error),
}
