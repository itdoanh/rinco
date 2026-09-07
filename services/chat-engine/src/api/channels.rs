//! Channel operations: create, list, add/remove members, distribute sender keys.

use chrono::Utc;
use uuid::Uuid;

use crate::api::types::{Channel, ChannelKind};
use crate::db::scylla::ScyllaStore;
use crate::error::ChatResult;

/// Create a channel (direct, group or public).
pub async fn create_channel(
    db: &ScyllaStore,
    tenant_id: Uuid,
    kind: ChannelKind,
    name: Option<String>,
    topic: Option<String>,
    members: Vec<Uuid>,
) -> ChatResult<Channel> {
    let now = Utc::now();
    let channel = Channel {
        id: Uuid::new_v4(),
        tenant_id,
        name,
        topic,
        kind,
        members,
        created_at: now,
        updated_at: now,
        archived: false,
    };
    db.insert_channel(&channel).await?;
    Ok(channel)
}

/// Add a member to a group channel and re-distribute the group's sender keys.
pub async fn add_group_member(
    db: &ScyllaStore,
    channel_id: Uuid,
    new_member: Uuid,
) -> ChatResult<()> {
    db.append_channel_member(channel_id, new_member).await?;
    // Re-distribute sender keys: read current group state and publish a fresh
    // chain key to all members. The actual key material is delivered via
    // pairwise Signal sessions; we just signal the change here.
    db.touch_group_sender_key(channel_id, Utc::now()).await?;
    Ok(())
}

/// Remove a member from a group channel.
pub async fn remove_group_member(
    db: &ScyllaStore,
    channel_id: Uuid,
    member: Uuid,
) -> ChatResult<()> {
    db.remove_channel_member(channel_id, member).await?;
    Ok(())
}

/// Fetch a channel.
pub async fn get_channel(db: &ScyllaStore, channel_id: Uuid) -> ChatResult<Option<Channel>> {
    db.get_channel(channel_id).await
}

/// List channels the user participates in.
pub async fn list_user_channels(db: &ScyllaStore, user_id: Uuid) -> ChatResult<Vec<Channel>> {
    db.list_user_channels(user_id).await
}
