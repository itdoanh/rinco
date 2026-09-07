//! Domain types for the chat engine.
//!
//! All types are `Serialize + Deserialize` so they can flow through the
//! Connect-RPC layer as well as the WebSocket JSON envelope.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

// =============================================================
// Channel
// =============================================================

/// Channel kind. Direct messages use `Direct`; everything else is a multi-party
/// channel whose membership is stored alongside the channel.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ChannelKind {
    /// 1:1 conversation (two devices, one human on each side).
    Direct,
    /// Small group, sender-key encrypted.
    Group,
    /// Open company channel; can be joined via invite link.
    Public,
    /// Threaded reply to another message in a parent channel.
    Thread,
}

/// Channel metadata.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Channel {
    /// Stable identifier.
    pub id: Uuid,
    /// Owning tenant.
    pub tenant_id: Uuid,
    /// Display name (`None` for direct channels which derive a name).
    pub name: Option<String>,
    /// Topic / description.
    pub topic: Option<String>,
    /// Channel kind.
    pub kind: ChannelKind,
    /// Members (user IDs).
    pub members: Vec<Uuid>,
    /// Created at.
    pub created_at: DateTime<Utc>,
    /// Updated at.
    pub updated_at: DateTime<Utc>,
    /// Whether the channel is archived.
    pub archived: bool,
}

// =============================================================
// Message
// =============================================================

/// A chat message. The server never sees the plaintext body – the `body` field
/// is always a ciphertext blob produced by the Signal Protocol session.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Message {
    /// Stable identifier.
    pub id: Uuid,
    /// Owning tenant.
    pub tenant_id: Uuid,
    /// Channel id.
    pub channel_id: Uuid,
    /// Sender user.
    pub sender_id: Uuid,
    /// Sender device (multiple devices per user can participate).
    pub sender_device_id: u64,
    /// Recipient device (None for group sends where the SFU/router handles fan-out).
    pub recipient_device_id: Option<u64>,
    /// Encrypted body.
    pub ciphertext: Vec<u8>,
    /// Public ratchet key of the sender at this message.
    pub ratchet_pub: Vec<u8>,
    /// Monotonic message number inside the session.
    pub msg_number: u64,
    /// Optional attachments.
    pub attachments: Vec<Attachment>,
    /// Optional reactions attached to a previous message.
    pub reply_to: Option<Uuid>,
    /// Server-assigned timestamp. Note: clients should treat the value as
    /// informational only and prefer their local `received_at` for ordering.
    pub server_ts: DateTime<Utc>,
    /// When the sender actually emitted the message.
    pub client_ts: Option<DateTime<Utc>>,
    /// True if the message was edited.
    pub edited: bool,
    /// True if the message was deleted (soft tombstone).
    pub deleted: bool,
}

// =============================================================
// Device / PreKey
// =============================================================

/// A user device registered with the server.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Device {
    /// User owning this device.
    pub user_id: Uuid,
    /// Per-user device id (chosen by the client, opaque to the server).
    pub device_id: u64,
    /// Long-term identity public key (X25519).
    pub identity_pub: Vec<u8>,
    /// Public part of the signed prekey (rotates weekly).
    pub signed_prekey: Vec<u8>,
    /// Ed25519 signature over the signed prekey.
    pub signed_prekey_sig: Vec<u8>,
    /// Pool of remaining one-time prekeys.
    pub one_time_prekeys: Vec<Vec<u8>>,
    /// Signal `registration_id` – allows multi-device support.
    pub registration_id: u32,
    /// Last seen timestamp.
    pub last_seen: DateTime<Utc>,
    /// Optional display name.
    pub name: Option<String>,
}

/// A bundle of prekeys that a peer can use to initiate a session.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PreKeyBundle {
    /// Owning user.
    pub user_id: Uuid,
    /// Owning device.
    pub device_id: u64,
    /// Identity public key.
    pub identity_pub: Vec<u8>,
    /// Signed prekey (public part).
    pub signed_prekey: Vec<u8>,
    /// Signature over the signed prekey.
    pub signed_prekey_sig: Vec<u8>,
    /// Single one-time prekey (consumed on first X3DH).
    pub one_time_prekey: Option<Vec<u8>>,
}

// =============================================================
// Reaction / Attachment / Presence
// =============================================================

/// A reaction (emoji) attached to a message.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Reaction {
    pub channel_id: Uuid,
    pub msg_id: Uuid,
    pub user_id: Uuid,
    pub reaction_value: String,
    pub ts: DateTime<Utc>,
}

/// An encrypted media attachment.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Attachment {
    pub msg_id: Uuid,
    pub s3_key: String,
    pub mime: String,
    pub size: u64,
    /// DEK encrypted with the session's message key.
    pub encrypted_dek: Vec<u8>,
    pub encrypted_size: u64,
    pub thumbnail_url: Option<String>,
}

/// Per-user presence state.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Presence {
    pub user_id: Uuid,
    pub device_id: u64,
    pub online: bool,
    pub last_seen: DateTime<Utc>,
}
