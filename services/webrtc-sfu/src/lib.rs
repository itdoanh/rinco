//! WebRTC SFU library.

pub mod api;
pub mod config;
pub mod error;
pub mod observability;
pub mod peer;
pub mod room;
pub mod safety;
pub mod sfu;
pub mod signaling;

pub use config::Config;
pub use error::{SfuError, SfuResult};

use std::sync::Arc;

use crate::peer::PeerManager;
use crate::room::RoomManager;
use crate::sfu::egress::EgressSink;
use crate::sfu::forwarder::Forwarder;

/// Shared SFU state injected into every handler.
pub struct SfuContext {
    pub config: Config,
    pub rooms: RoomManager,
    pub peers: PeerManager,
    pub forwarder: Forwarder,
    pub egress: Arc<dyn EgressSink>,
}
