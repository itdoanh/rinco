//! Peer connection wrapper for WebRTC.
//!
//! Trong production sẽ dùng webrtc-rs để handle ICE/DTLS/SRTP.

use anyhow::Result;
use serde::{Deserialize, Serialize};
use tracing::info;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PeerConnection {
    pub peer_id: String,
    pub user_id: String,
    pub room_id: String,
    pub state: PeerState,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum PeerState {
    New,
    Connecting,
    Connected,
    Disconnected,
    Failed,
}

impl PeerConnection {
    pub async fn new(user_id: String, room_id: String) -> Result<Self> {
        info!("creating peer connection for user {}", user_id);
        Ok(Self {
            peer_id: Uuid::now_v7().to_string(),
            user_id,
            room_id,
            state: PeerState::New,
        })
    }

    pub async fn set_remote_description(&mut self, _sdp: String) -> Result<()> {
        self.state = PeerState::Connecting;
        Ok(())
    }

    pub async fn create_answer(&mut self) -> Result<String> {
        // Stub: in production use webrtc-rs
        Ok("v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n".to_string())
    }

    pub async fn add_ice_candidate(&mut self, _candidate: String) -> Result<()> {
        Ok(())
    }

    pub async fn close(&mut self) -> Result<()> {
        self.state = PeerState::Disconnected;
        Ok(())
    }
}
