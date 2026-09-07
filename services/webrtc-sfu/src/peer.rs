//! PeerConnection wrapper.
//!
//! The optional `webrtc` crate provides the real RTCPeerConnection behind the
//! `webrtc` feature. Without the feature the module falls back to an
//! in-memory stub that stores SDP/ICE candidates for tests.

use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::Mutex;
use uuid::Uuid;

use crate::error::SfuResult;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SdpAnswer {
    pub sdp: String,
    pub rtp_extensions: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IceCandidate {
    pub candidate: String,
    pub sdp_mid: Option<String>,
    pub sdp_mline_index: Option<u16>,
}

/// Per-peer state.
pub struct Peer {
    pub user_id: Uuid,
    pub room_id: Uuid,
    /// Highest video layer the peer is willing to receive.
    pub max_spatial_layer: u8,
    /// Highest temporal layer the peer is willing to receive.
    pub max_temporal_layer: u8,
    /// Negotiated codec list.
    pub video_codecs: Vec<String>,
    pub audio_codecs: Vec<String>,
    /// SDP exchanged with the peer.
    pub remote_sdp: Mutex<Option<String>>,
    pub local_sdp: Mutex<Option<String>>,
    /// Collected remote ICE candidates.
    pub remote_candidates: Mutex<Vec<IceCandidate>>,
}

impl Peer {
    /// Construct a new peer with the default codec preferences.
    pub fn new(user_id: Uuid, room_id: Uuid) -> Self {
        Self {
            user_id,
            room_id,
            max_spatial_layer: 2,
            max_temporal_layer: 2,
            video_codecs: vec!["AV1".into(), "VP9".into(), "H264".into()],
            audio_codecs: vec!["opus".into(), "G722".into()],
            remote_sdp: Mutex::new(None),
            local_sdp: Mutex::new(None),
            remote_candidates: Mutex::new(Vec::new()),
        }
    }

    /// Set the remote SDP offer.
    pub async fn set_remote_offer(&self, sdp: String) -> SfuResult<()> {
        *self.remote_sdp.lock().await = Some(sdp);
        Ok(())
    }

    /// Take the answer to the offer.
    pub async fn set_local_answer(&self, sdp: String) -> SfuResult<()> {
        *self.local_sdp.lock().await = Some(sdp);
        Ok(())
    }

    /// Add an ICE candidate.
    pub async fn add_ice_candidate(&self, c: IceCandidate) -> SfuResult<()> {
        self.remote_candidates.lock().await.push(c);
        Ok(())
    }

    /// Validate the offered SDP. This is a placeholder for the full SDP parser
    /// that ships with the optional `webrtc` crate.
    pub fn validate_sdp(&self, sdp: &str) -> SfuResult<()> {
        if !sdp.contains("v=0") {
            return Err(crate::error::SfuError::Sdp("missing v=0".into()));
        }
        if !sdp.contains("m=audio") && !sdp.contains("m=video") {
            return Err(crate::error::SfuError::Sdp("no media sections".into()));
        }
        Ok(())
    }
}

/// Peer manager (keyed by `(room_id, user_id)`).
#[derive(Default, Clone)]
pub struct PeerManager {
    peers: Arc<dashmap::DashMap<(Uuid, Uuid), Arc<Peer>>>,
}

impl PeerManager {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn insert(&self, peer: Arc<Peer>) {
        self.peers.insert((peer.room_id, peer.user_id), peer);
    }

    pub fn get(&self, room_id: Uuid, user_id: Uuid) -> Option<Arc<Peer>> {
        self.peers.get(&(room_id, user_id)).map(|p| p.value().clone())
    }

    pub fn remove(&self, room_id: Uuid, user_id: Uuid) -> Option<Arc<Peer>> {
        self.peers.remove(&(room_id, user_id)).map(|p| p.1)
    }

    pub fn list_for_room(&self, room_id: Uuid) -> Vec<Arc<Peer>> {
        self.peers
            .iter()
            .filter(|p| p.key().0 == room_id)
            .map(|p| p.value().clone())
            .collect()
    }
}
