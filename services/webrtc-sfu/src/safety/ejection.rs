//! Safety actions: kick / ban / admin override.

use chrono::{DateTime, Utc};
use parking_lot::RwLock;
use std::collections::HashSet;
use uuid::Uuid;

use crate::error::SfuResult;
use crate::room::RoomEvent;
use crate::SfuContext;

/// In-memory ban list (production replaces this with Redis/Postgres).
#[derive(Default)]
pub struct BanList {
    /// Banned user IDs.
    pub users: RwLock<HashSet<Uuid>>,
    /// Banned IP addresses (strings).
    pub ips: RwLock<HashSet<String>>,
    /// When the ban expires (for time-limited bans).
    pub until: RwLock<std::collections::HashMap<Uuid, DateTime<Utc>>>,
}

impl BanList {
    pub fn ban_user(&self, user_id: Uuid, until: Option<DateTime<Utc>>) {
        self.users.write().insert(user_id);
        if let Some(t) = until {
            self.until.write().insert(user_id, t);
        }
    }
    pub fn ban_ip(&self, ip: String) {
        self.ips.write().insert(ip);
    }
    pub fn is_user_banned(&self, user_id: Uuid) -> bool {
        if !self.users.read().contains(&user_id) {
            return false;
        }
        if let Some(until) = self.until.read().get(&user_id).copied() {
            until > Utc::now()
        } else {
            true
        }
    }
    pub fn is_ip_banned(&self, ip: &str) -> bool {
        self.ips.read().contains(ip)
    }
}

/// Kick a user from a room.
pub async fn kick(ctx: &SfuContext, room_id: Uuid, user_id: Uuid) -> SfuResult<()> {
    let room = ctx
        .rooms
        .get(room_id)
        .ok_or_else(|| crate::error::SfuError::RoomNotFound(room_id.to_string()))?;
    room.remove_participant(user_id);
    ctx.peers.remove(room_id, user_id);
    ctx.forwarder.remove_user(room_id, user_id);
    Ok(())
}

/// Force-mute audio/video for a participant.
pub async fn mute(ctx: &SfuContext, room_id: Uuid, user_id: Uuid) -> SfuResult<()> {
    let room = ctx
        .rooms
        .get(room_id)
        .ok_or_else(|| crate::error::SfuError::RoomNotFound(room_id.to_string()))?;
    let _ = room.events.send(RoomEvent::AudioMute { user_id, muted: true });
    let _ = room.events.send(RoomEvent::VideoToggle { user_id, on: false });
    Ok(())
}

/// Close a room and notify all participants.
pub async fn force_end(ctx: &SfuContext, room_id: Uuid) -> SfuResult<()> {
    ctx.rooms.delete(room_id);
    Ok(())
}
