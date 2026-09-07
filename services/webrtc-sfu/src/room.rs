//! WebRTC room management: create, join, leave, list participants, broadcast.

use chrono::{DateTime, Utc};
use dashmap::DashMap;
use parking_lot::RwLock;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::broadcast;
use uuid::Uuid;

use crate::error::{SfuError, SfuResult};

/// A room participant.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Participant {
    pub id: Uuid,
    pub user_id: Uuid,
    pub tenant_id: Uuid,
    pub display_name: String,
    pub is_audio_enabled: bool,
    pub is_video_enabled: bool,
    pub is_screen_sharing: bool,
    pub joined_at: DateTime<Utc>,
    /// Last known connection quality (0.0..=1.0).
    pub connection_quality: f32,
}

/// A logical meeting room.
#[derive(Debug)]
pub struct Room {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub name: String,
    pub created_at: DateTime<Utc>,
    pub max_peers: u32,
    /// Active participants keyed by user_id.
    pub participants: RwLock<Vec<Participant>>,
    /// Room-wide event bus.
    pub events: broadcast::Sender<RoomEvent>,
}

impl Room {
    /// Create a new room with the given capacity.
    pub fn new(id: Uuid, tenant_id: Uuid, name: String, max_peers: u32) -> Self {
        let (tx, _) = broadcast::channel(1024);
        Self {
            id,
            tenant_id,
            name,
            created_at: Utc::now(),
            max_peers,
            participants: RwLock::new(Vec::new()),
            events: tx,
        }
    }

    /// Add a participant, returning an error if the room is full.
    pub fn add_participant(&self, p: Participant) -> SfuResult<()> {
        let mut guard = self.participants.write();
        if guard.len() as u32 >= self.max_peers {
            return Err(SfuError::Forbidden("room full".into()));
        }
        if let Some(existing) = guard.iter_mut().find(|x| x.user_id == p.user_id) {
            *existing = p.clone();
        } else {
            guard.push(p.clone());
        }
        let _ = self.events.send(RoomEvent::ParticipantJoined { participant: p });
        Ok(())
    }

    /// Remove a participant.
    pub fn remove_participant(&self, user_id: Uuid) -> Option<Participant> {
        let mut guard = self.participants.write();
        let idx = guard.iter().position(|p| p.user_id == user_id)?;
        let removed = guard.remove(idx);
        let _ = self.events.send(RoomEvent::ParticipantLeft { user_id });
        Some(removed)
    }

    /// Snapshot the participant list.
    pub fn list_participants(&self) -> Vec<Participant> {
        self.participants.read().clone()
    }

    /// Subscribe to the room event stream.
    pub fn subscribe(&self) -> broadcast::Receiver<RoomEvent> {
        self.events.subscribe()
    }
}

/// Events broadcast to subscribers of a room.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum RoomEvent {
    ParticipantJoined { participant: Participant },
    ParticipantLeft { user_id: Uuid },
    AudioMute { user_id: Uuid, muted: bool },
    VideoToggle { user_id: Uuid, on: bool },
    ScreenShare { user_id: Uuid, on: bool },
    RecordingStarted,
    RecordingStopped,
    RoomClosed,
}

/// Manages all rooms in this SFU node.
#[derive(Clone, Default)]
pub struct RoomManager {
    rooms: Arc<DashMap<Uuid, Arc<Room>>>,
}

impl RoomManager {
    pub fn new() -> Self {
        Self::default()
    }

    /// Create a new room.
    pub fn create(&self, tenant_id: Uuid, name: String, max_peers: u32) -> Arc<Room> {
        let id = Uuid::new_v4();
        let room = Arc::new(Room::new(id, tenant_id, name, max_peers));
        self.rooms.insert(id, room.clone());
        room
    }

    /// Look up a room.
    pub fn get(&self, id: Uuid) -> Option<Arc<Room>> {
        self.rooms.get(&id).map(|r| r.value().clone())
    }

    /// List all room ids.
    pub fn list(&self) -> Vec<Uuid> {
        self.rooms.iter().map(|r| *r.key()).collect()
    }

    /// Delete a room and notify its participants.
    pub fn delete(&self, id: Uuid) -> bool {
        if let Some((_, room)) = self.rooms.remove(&id) {
            let _ = room.events.send(RoomEvent::RoomClosed);
            true
        } else {
            false
        }
    }

    /// Number of rooms.
    pub fn count(&self) -> usize {
        self.rooms.len()
    }

    /// Periodically remove empty rooms.
    pub fn cleanup_empty(&self) {
        let empty: Vec<Uuid> = self
            .rooms
            .iter()
            .filter(|r| r.value().list_participants().is_empty())
            .map(|r| *r.key())
            .collect();
        for id in empty {
            self.delete(id);
        }
    }
}
